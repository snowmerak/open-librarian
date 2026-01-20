package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/snowmerak/open-librarian/lib/client/llm"
	mongoClient "github.com/snowmerak/open-librarian/lib/client/mongo"
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
			Data: "AI 에이전트가 답변을 생성하고 있습니다...",
		})

		// Chat Session Management
		var session *mongoClient.ChatSession
		if req.SessionID != "" {
			s, err := h.server.mongoClient.GetChatSession(ctx, req.SessionID)
			if err == nil {
				session = s
			}
		}

		if session == nil {
			session = &mongoClient.ChatSession{
				Title:    req.Query,
				Messages: []mongoClient.ChatMessage{},
			}
		}

		// Load history into request
		// Convert existing Mongo messages to LLM messages
		for _, msg := range session.Messages {
			req.History = append(req.History, llm.ChatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// Add User Message
		session.Messages = append(session.Messages, mongoClient.ChatMessage{
			Role:      "user",
			Content:   req.Query,
			Timestamp: time.Now(),
		})

		// AI 검색 수행 (Agentic Search)
		resp, err := h.server.Search(ctx, &req)
		if err != nil {
			searchLog.Error().Err(err).Msg("Search failed")
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: fmt.Sprintf("Search failed: %v", err),
			})
			continue
		}

		// Add AI Message & Save Session
		session.Messages = append(session.Messages, mongoClient.ChatMessage{
			Role:      "assistant",
			Content:   resp.Answer,
			Sources:   resp.Sources,
			Timestamp: time.Now(),
		})

		if err := h.server.mongoClient.SaveChatSession(ctx, session); err != nil {
			searchLog.Error().Err(err).Msg("Failed to save chat session")
		}

		// 참조 소스 전송
		conn.WriteJSON(WSMessage{
			Type: "sources",
			Data: resp.Sources,
		})

		// 답변 전송 (전체 답변을 한번에 전송)
		conn.WriteJSON(WSMessage{
			Type: "answer",
			Data: resp.Answer,
		})

		// 완료 알림
		conn.WriteJSON(WSMessage{
			Type: "done",
			Data: map[string]interface{}{
				"message":    "검색이 완료되었습니다.",
				"session_id": session.ID.Hex(),
			},
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
