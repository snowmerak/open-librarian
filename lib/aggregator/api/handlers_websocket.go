package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/client/qdrant"
	"github.com/snowmerak/open-librarian/lib/util/logger"
)

// WebSocket 업그레이더
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 모든 도메인에서의 연결 허용 (개발용)
	},
}

// WebSocketSearchHandler handles WebSocket search requests
func (h *HTTPServer) WebSocketSearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.NewLoggerWithContext(ctx, "websocket_search").Start()
	defer log.End()

	// WebSocket 업그레이드
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	log.Info().Msg("WebSocket connection established")

	// 메시지 수신 대기
	for {
		var req SearchRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			log.Error().Err(err).Msg("Error reading WebSocket message")
			break
		}

		// 검색 요청 검증
		if req.Query == "" {
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: "Query is required",
			})
			continue
		}

		searchLog := logger.NewLoggerWithContext(ctx, "search_request").
			WithField("query", req.Query).
			Start()

		// 검색 시작 알림
		conn.WriteJSON(WSMessage{
			Type: "status",
			Data: "검색을 시작합니다...",
		})

		// 1. 언어 감지
		queryLang := h.server.languageDetector.DetectLanguage(req.Query)
		searchLog.Info().Str("detected_language", queryLang).Msg("Language detected")

		// 2. 쿼리 임베딩 생성
		queryEmbedding, err := h.server.llmClient.GenerateEmbedding(ctx, "query: "+req.Query)
		if err != nil {
			searchLog.Error().Err(err).Msg("Failed to generate query embedding")
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: fmt.Sprintf("Failed to generate query embedding: %v", err),
			})
			continue
		}

		conn.WriteJSON(WSMessage{
			Type: "status",
			Data: "검색 중...",
		})

		// 3. 기본 크기 설정
		size := req.Size
		if size == 0 {
			size = 5
		}

		// 4. 병렬 검색 수행
		// 4a. Qdrant 벡터 검색
		allVectorResults, err := h.server.qdrantClient.VectorSearch(ctx, queryEmbedding, uint64(size*4), queryLang)
		if err != nil {
			searchLog.Error().Err(err).Msg("Vector search failed")
			allVectorResults = []qdrant.VectorSearchResult{}
		}

		// 제목과 요약 결과 분리
		var titleVectorResults, summaryVectorResults []qdrant.VectorSearchResult
		for _, result := range allVectorResults {
			if len(result.ID) > 6 && result.ID[len(result.ID)-6:] == "_title" {
				titleVectorResults = append(titleVectorResults, result)
			} else if len(result.ID) > 8 && result.ID[len(result.ID)-8:] == "_summary" {
				summaryVectorResults = append(summaryVectorResults, result)
			}
		}

		// 벡터 결과 결합 및 중복 제거
		combinedVectorResults := h.server.combineVectorResults(titleVectorResults, summaryVectorResults, size*2)

		// 4b. OpenSearch 키워드 검색
		keywordResp, err := h.server.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, size*2, req.From)
		if err != nil {
			searchLog.Error().Err(err).Msg("Keyword search failed")
			keywordResp = &opensearch.SearchResponse{Results: []opensearch.SearchResult{}}
		}

		// 5. 벡터 검색 결과에서 아티클 ID 추출
		var vectorArticleIDs []string
		uniqueIDs := make(map[string]bool)
		for _, result := range combinedVectorResults {
			articleID := h.server.extractArticleID(result.ID)
			if !uniqueIDs[articleID] {
				vectorArticleIDs = append(vectorArticleIDs, articleID)
				uniqueIDs[articleID] = true
			}
		}

		var vectorArticles []opensearch.Article
		if len(vectorArticleIDs) > 0 {
			vectorArticles, err = h.server.opensearchClient.GetArticlesByIDs(ctx, vectorArticleIDs)
			if err != nil {
				searchLog.Error().Err(err).Msg("Failed to get articles by IDs")
				vectorArticles = []opensearch.Article{}
			}
		}

		// 6. 검색 결과 결합 및 중복 제거
		combinedResults := h.server.combineSearchResults(combinedVectorResults, vectorArticles, keywordResp.Results, size)

		// 6.5. LLM을 사용한 검색 관련성 검증
		filteredResults, err := h.server.validateSearchRelevance(ctx, req.Query, combinedResults)
		if err != nil {
			searchLog.Error().Err(err).Msg("Failed to validate search relevance")
			// 검증 실패 시 원본 결과 사용
			filteredResults = combinedResults
		}

		searchLog.Info().
			Int("total_results", len(filteredResults)).
			Int("vector_results", len(combinedVectorResults)).
			Int("keyword_results", len(keywordResp.Results)).
			Msg("Search results combined")

		// 참조 소스 전송
		conn.WriteJSON(WSMessage{
			Type: "sources",
			Data: filteredResults,
		})

		// 7. AI 답변 생성을 위한 아티클 추출
		articles := make([]opensearch.Article, len(filteredResults))
		for i, result := range filteredResults {
			articles[i] = result.Article
		}

		conn.WriteJSON(WSMessage{
			Type: "status",
			Data: "AI 답변을 생성하고 있습니다...",
		})

		// 8. 스트리밍 답변 생성
		err = h.server.generateAnswerStream(ctx, req.Query, articles, func(chunk string) error {
			return conn.WriteJSON(WSMessage{
				Type: "answer",
				Data: chunk,
			})
		})

		if err != nil {
			searchLog.Error().Err(err).Msg("Failed to generate answer")
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: fmt.Sprintf("Failed to generate answer: %v", err),
			})
			continue
		}

		// 완료 알림
		conn.WriteJSON(WSMessage{
			Type: "done",
			Data: "검색이 완료되었습니다.",
		})

		searchLog.EndWithMsg("Search request completed successfully")
	}
}

// WebSocketAddArticleHandler handles WebSocket article addition requests with progress updates
func (h *HTTPServer) WebSocketAddArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.NewLoggerWithContext(ctx, "websocket_add_article").Start()
	defer log.End()

	// Check for authentication - first try Authorization header, then query parameter
	var tokenString string
	authHeader := r.Header.Get("Authorization")

	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		// Try to get token from query parameter
		tokenString = r.URL.Query().Get("token")
	}

	if tokenString == "" {
		log.Error().Msg("Authorization required")
		http.Error(w, "Authorization required (header or query parameter)", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.server.jwtService.ValidateToken(tokenString)
	if err != nil {
		log.Error().Err(err).Msg("Invalid token")
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := h.server.mongoClient.GetUserFromToken(ctx, tokenString, h.server.jwtService)
	if err != nil {
		log.Error().Err(err).Msg("User not found")
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Add user and claims to context
	ctx = context.WithValue(ctx, UserContextKey, user)
	ctx = context.WithValue(ctx, ClaimsContextKey, claims)

	// WebSocket upgrade
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	log.Info().Str("username", user.Username).Msg("WebSocket connection established for article addition")

	// Wait for incoming messages
	for {
		var req ArticleRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			log.Error().Err(err).Msg("Error reading WebSocket message")
			break
		}

		// Validate request
		if req.Title == "" {
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: "Title is required",
			})
			continue
		}
		if req.Content == "" {
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: "Content is required",
			})
			continue
		}

		// Validate created_date format if provided
		if req.CreatedDate != "" {
			if _, err := time.Parse(time.RFC3339, req.CreatedDate); err != nil {
				conn.WriteJSON(WSMessage{
					Type: "error",
					Data: "Created date must be in RFC3339 format (e.g., 2023-12-25T15:30:00Z)",
				})
				continue
			}
		}

		articleLog := logger.NewLoggerWithContext(ctx, "add_article").
			WithFields(map[string]interface{}{
				"title":    req.Title,
				"username": user.Username,
			}).Start()

		// Define progress callback
		progressCallback := func(step string, progress int, total int) error {
			return conn.WriteJSON(WSMessage{
				Type: "progress",
				Data: map[string]interface{}{
					"step":     step,
					"progress": progress,
					"total":    total,
					"percent":  float64(progress) / float64(total) * 100,
				},
			})
		}

		// Send initial status
		conn.WriteJSON(WSMessage{
			Type: "status",
			Data: "Starting article processing...",
		})

		// Call AddArticleWithProgress with WebSocket progress updates
		resp, err := h.server.AddArticleWithProgress(ctx, &req, progressCallback)
		if err != nil {
			articleLog.Error().Err(err).Msg("Error adding article")
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: fmt.Sprintf("Failed to process article: %v", err),
			})
			continue
		}

		articleLog.DataCreated("article", resp.ID, map[string]interface{}{
			"title": req.Title,
		})

		// Send success response
		conn.WriteJSON(WSMessage{
			Type: "success",
			Data: resp,
		})

		// Send completion notification
		conn.WriteJSON(WSMessage{
			Type: "done",
			Data: "Article has been successfully added",
		})

		articleLog.EndWithMsg("Article addition completed successfully")
	}
}

// WebSocketBulkAddArticleHandler handles WebSocket bulk article addition requests with progress updates
func (h *HTTPServer) WebSocketBulkAddArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.NewLoggerWithContext(ctx, "websocket_bulk_add_article").Start()
	defer log.End()

	// Check for authentication - first try Authorization header, then query parameter
	var tokenString string
	authHeader := r.Header.Get("Authorization")

	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		// Try to get token from query parameter
		tokenString = r.URL.Query().Get("token")
	}

	if tokenString == "" {
		log.Error().Msg("Authorization required")
		http.Error(w, "Authorization required (header or query parameter)", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.server.jwtService.ValidateToken(tokenString)
	if err != nil {
		log.Error().Err(err).Msg("Invalid token")
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Get user from database
	user, err := h.server.mongoClient.GetUserFromToken(ctx, tokenString, h.server.jwtService)
	if err != nil {
		log.Error().Err(err).Msg("User not found")
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Add user and claims to context
	ctx = context.WithValue(ctx, UserContextKey, user)
	ctx = context.WithValue(ctx, ClaimsContextKey, claims)

	// WebSocket upgrade
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	log.Info().Str("username", user.Username).Msg("WebSocket connection established for bulk article addition")

	// Wait for incoming messages
	for {
		var req BulkArticleRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			log.Error().Err(err).Msg("Error reading WebSocket message")
			break
		}

		// Validate request
		if len(req.Articles) == 0 {
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: "No articles provided",
			})
			continue
		}

		bulkLog := logger.NewLoggerWithContext(ctx, "bulk_add_articles").
			WithFields(map[string]interface{}{
				"article_count": len(req.Articles),
				"username":      user.Username,
			}).Start()

		// Define bulk progress callback
		bulkProgressCallback := func(articleIndex int, totalArticles int, currentStep string, stepProgress int, stepTotal int, result *BulkArticleResult) error {
			data := map[string]interface{}{
				"article_index":   articleIndex,
				"total_articles":  totalArticles,
				"step":            currentStep,
				"step_progress":   stepProgress,
				"step_total":      stepTotal,
				"step_percent":    float64(stepProgress) / float64(stepTotal) * 100,
				"overall_percent": float64(articleIndex) / float64(totalArticles) * 100,
			}

			if result != nil {
				data["article_title"] = result.Title
				data["success"] = result.Success
				if result.Error != "" {
					data["error"] = result.Error
				}
			}

			return conn.WriteJSON(WSMessage{
				Type: "bulk_progress",
				Data: data,
			})
		}

		// Send initial status
		conn.WriteJSON(WSMessage{
			Type: "status",
			Data: fmt.Sprintf("Starting bulk processing of %d articles...", len(req.Articles)),
		})

		// Call AddArticlesBulkWithProgress with WebSocket progress updates
		resp, err := h.server.AddArticlesBulkWithProgress(ctx, &req, bulkProgressCallback)
		if err != nil {
			bulkLog.Error().Err(err).Msg("Error in bulk article addition")
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: fmt.Sprintf("Failed to process articles: %v", err),
			})
			continue
		}

		bulkLog.Info().
			Int("success_count", resp.SuccessCount).
			Int("error_count", resp.ErrorCount).
			Msg("Bulk article addition completed")

		// Send success response
		conn.WriteJSON(WSMessage{
			Type: "bulk_success",
			Data: resp,
		})

		// Send completion notification
		conn.WriteJSON(WSMessage{
			Type: "done",
			Data: fmt.Sprintf("Bulk upload completed: %d successful, %d failed", resp.SuccessCount, resp.ErrorCount),
		})

		bulkLog.EndWithMsg("Bulk article addition completed successfully")
	}
}
