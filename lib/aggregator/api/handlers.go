package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/client/qdrant"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// HTTPServer wraps the API server with HTTP handlers
type HTTPServer struct {
	server *Server
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(server *Server) *HTTPServer {
	return &HTTPServer{
		server: server,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeErrorResponse writes an error response to the client
func writeErrorResponse(w http.ResponseWriter, statusCode int, err string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   err,
		Message: message,
	})
}

// writeJSONResponse writes a JSON response to the client
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// HealthCheckHandler handles health check requests
func (h *HTTPServer) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.server.HealthCheck(ctx); err != nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "service_unavailable", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// AddArticleHandler handles article addition requests
func (h *HTTPServer) AddArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	// Basic validation
	if req.Title == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_title", "Title is required")
		return
	}
	if req.Content == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_content", "Content is required")
		return
	}

	// Validate created_date format if provided
	if req.CreatedDate != "" {
		if _, err := time.Parse(time.RFC3339, req.CreatedDate); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "invalid_date_format", "Created date must be in RFC3339 format (e.g., 2023-12-25T15:30:00Z)")
			return
		}
	}

	resp, err := h.server.AddArticle(ctx, &req)
	if err != nil {
		log.Printf("Error adding article: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "processing_error", "Failed to process article")
		return
	}

	writeJSONResponse(w, http.StatusCreated, resp)
}

// SearchHandler handles search requests
func (h *HTTPServer) SearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	// Basic validation
	if req.Query == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_query", "Query is required")
		return
	}

	resp, err := h.server.Search(ctx, &req)
	if err != nil {
		log.Printf("Error performing search: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to perform search")
		return
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// GetArticleHandler handles getting a specific article
func (h *HTTPServer) GetArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")
	if id == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_id", "Article ID is required")
		return
	}

	article, err := h.server.GetArticle(ctx, id)
	if err != nil {
		log.Printf("Error getting article: %v", err)
		writeErrorResponse(w, http.StatusNotFound, "article_not_found", "Article not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, article)
}

// KeywordSearchHandler handles keyword-only search requests
func (h *HTTPServer) KeywordSearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := r.URL.Query().Get("q")
	if query == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_query", "Query parameter 'q' is required")
		return
	}

	lang := r.URL.Query().Get("lang")

	sizeStr := r.URL.Query().Get("size")
	size := 10
	if sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			size = s
		}
	}

	fromStr := r.URL.Query().Get("from")
	from := 0
	if fromStr != "" {
		if f, err := strconv.Atoi(fromStr); err == nil && f >= 0 {
			from = f
		}
	}

	resp, err := h.server.opensearchClient.KeywordSearch(ctx, query, lang, size, from)
	if err != nil {
		log.Printf("Error performing keyword search: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to perform search")
		return
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// GetSupportedLanguagesHandler returns supported languages
func (h *HTTPServer) GetSupportedLanguagesHandler(w http.ResponseWriter, r *http.Request) {
	languages := h.server.GetSupportedLanguages()
	writeJSONResponse(w, http.StatusOK, map[string][]string{
		"languages": languages,
	})
}

// SearchStreamHandler handles search requests with SSE streaming
func (h *HTTPServer) SearchStreamHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	// Basic validation
	if req.Query == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_query", "Query is required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Send initial message
	sendSSEMessage(w, "status", "Starting search...")
	w.(http.Flusher).Flush()

	// 1. Detect query language
	queryLang := h.server.languageDetector.DetectLanguage(req.Query)

	// 2. Generate query embedding for vector search
	queryEmbedding, err := h.server.ollamaClient.GenerateEmbedding(ctx, "query: "+req.Query)
	if err != nil {
		sendSSEMessage(w, "error", fmt.Sprintf("Failed to generate query embedding: %v", err))
		return
	}

	sendSSEMessage(w, "status", "Performing search...")
	w.(http.Flusher).Flush()

	// 3. Set default size if not provided
	size := req.Size
	if size == 0 {
		size = 5 // Default to top 5 results for AI answer generation
	}

	// 4. Perform parallel searches
	// 4a. Vector search with Qdrant
	allVectorResults, err := h.server.qdrantClient.VectorSearch(ctx, queryEmbedding, uint64(size*4), queryLang)
	if err != nil {
		log.Printf("Vector search failed: %v", err)
		allVectorResults = []qdrant.VectorSearchResult{}
	}

	// Separate title and summary results
	var titleVectorResults, summaryVectorResults []qdrant.VectorSearchResult
	for _, result := range allVectorResults {
		if len(result.ID) > 6 && result.ID[len(result.ID)-6:] == "_title" {
			titleVectorResults = append(titleVectorResults, result)
		} else if len(result.ID) > 8 && result.ID[len(result.ID)-8:] == "_summary" {
			summaryVectorResults = append(summaryVectorResults, result)
		}
	}

	// Combine and deduplicate vector results
	combinedVectorResults := h.server.combineVectorResults(titleVectorResults, summaryVectorResults)

	// 4b. Keyword search with OpenSearch
	keywordResp, err := h.server.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, size*2, req.From)
	if err != nil {
		log.Printf("Keyword search failed: %v", err)
		keywordResp = &opensearch.SearchResponse{Results: []opensearch.SearchResult{}}
	}

	// 5. Get articles by IDs from vector search results
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
			log.Printf("Failed to get articles by IDs: %v", err)
			vectorArticles = []opensearch.Article{}
		}
	}

	// 6. Combine and deduplicate results
	combinedResults := h.server.combineSearchResults(combinedVectorResults, vectorArticles, keywordResp.Results, size)

	// Send sources information
	sourcesData, _ := json.Marshal(combinedResults)
	sendSSEMessage(w, "sources", string(sourcesData))
	w.(http.Flusher).Flush()

	// 7. Extract articles for AI answer generation
	articles := make([]opensearch.Article, len(combinedResults))
	for i, result := range combinedResults {
		articles[i] = result.Article
	}

	sendSSEMessage(w, "status", "Generating AI answer...")
	w.(http.Flusher).Flush()

	// 8. Generate AI answer using search results with streaming
	err = h.server.generateAnswerStream(ctx, req.Query, articles, func(chunk string) error {
		sendSSEMessage(w, "answer", chunk)
		w.(http.Flusher).Flush()
		return nil
	})

	if err != nil {
		sendSSEMessage(w, "error", fmt.Sprintf("Failed to generate answer: %v", err))
		return
	}

	sendSSEMessage(w, "done", "")
	w.(http.Flusher).Flush()
}

// sendSSEMessage sends a Server-Sent Event message
func sendSSEMessage(w http.ResponseWriter, eventType, data string) {
	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// WebSocket 업그레이더
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 모든 도메인에서의 연결 허용 (개발용)
	},
}

// WebSocket 메시지 타입
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// WebSocketSearchHandler handles WebSocket search requests
func (h *HTTPServer) WebSocketSearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// WebSocket 업그레이드
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connection established")

	// 메시지 수신 대기
	for {
		var req SearchRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
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

		log.Printf("Received search query: %s", req.Query)

		// 검색 시작 알림
		conn.WriteJSON(WSMessage{
			Type: "status",
			Data: "검색을 시작합니다...",
		})

		// 1. 언어 감지
		queryLang := h.server.languageDetector.DetectLanguage(req.Query)

		// 2. 쿼리 임베딩 생성
		queryEmbedding, err := h.server.ollamaClient.GenerateEmbedding(ctx, "query: "+req.Query)
		if err != nil {
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
			log.Printf("Vector search failed: %v", err)
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
		combinedVectorResults := h.server.combineVectorResults(titleVectorResults, summaryVectorResults)

		// 4b. OpenSearch 키워드 검색
		keywordResp, err := h.server.opensearchClient.KeywordSearch(ctx, req.Query, queryLang, size*2, req.From)
		if err != nil {
			log.Printf("Keyword search failed: %v", err)
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
				log.Printf("Failed to get articles by IDs: %v", err)
				vectorArticles = []opensearch.Article{}
			}
		}

		// 6. 검색 결과 결합 및 중복 제거
		combinedResults := h.server.combineSearchResults(combinedVectorResults, vectorArticles, keywordResp.Results, size)

		// 참조 소스 전송
		conn.WriteJSON(WSMessage{
			Type: "sources",
			Data: combinedResults,
		})

		// 7. AI 답변 생성을 위한 아티클 추출
		articles := make([]opensearch.Article, len(combinedResults))
		for i, result := range combinedResults {
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
	}
}

// WebSocketAddArticleHandler handles WebSocket article addition requests with progress updates
func (h *HTTPServer) WebSocketAddArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// WebSocket upgrade
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connection established for article addition")

	// Wait for incoming messages
	for {
		var req ArticleRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
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

		log.Printf("Received article addition request: %s", req.Title)

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
			log.Printf("Error adding article: %v", err)
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: fmt.Sprintf("Failed to process article: %v", err),
			})
			continue
		}

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
	}
}

// WebSocketBulkAddArticleHandler handles WebSocket bulk article addition requests with progress updates
func (h *HTTPServer) WebSocketBulkAddArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// WebSocket upgrade
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Println("WebSocket connection established for bulk article addition")

	// Wait for incoming messages
	for {
		var req BulkArticleRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
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

		log.Printf("Received bulk article addition request: %d articles", len(req.Articles))

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
			log.Printf("Error in bulk article addition: %v", err)
			conn.WriteJSON(WSMessage{
				Type: "error",
				Data: fmt.Sprintf("Failed to process articles: %v", err),
			})
			continue
		}

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
	}
}

// ExternalArticleListHandler handles external article listing requests (read-only)
func (h *HTTPServer) ExternalArticleListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	lang := r.URL.Query().Get("lang")
	author := r.URL.Query().Get("author")

	sizeStr := r.URL.Query().Get("size")
	size := 20 // Default size
	if sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			size = s // Max 100 articles per request
		}
	}

	fromStr := r.URL.Query().Get("from")
	from := 0
	if fromStr != "" {
		if f, err := strconv.Atoi(fromStr); err == nil && f >= 0 {
			from = f
		}
	}

	// Build search query for listing
	query := "*" // Match all documents
	if author != "" {
		query = fmt.Sprintf("author:\"%s\"", author)
	}

	resp, err := h.server.opensearchClient.KeywordSearch(ctx, query, lang, size, from)
	if err != nil {
		log.Printf("Error listing articles: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to list articles")
		return
	}

	// Format response for external agents
	articlesResponse := map[string]interface{}{
		"articles": resp.Results,
		"total":    resp.Total,
		"took":     resp.Took,
		"from":     from,
		"size":     size,
	}

	writeJSONResponse(w, http.StatusOK, articlesResponse)
}

// ExternalArticleDetailHandler handles external article detail requests (read-only)
func (h *HTTPServer) ExternalArticleDetailHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")
	if id == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_id", "Article ID is required")
		return
	}

	article, err := h.server.GetArticle(ctx, id)
	if err != nil {
		log.Printf("Error getting article: %v", err)
		writeErrorResponse(w, http.StatusNotFound, "article_not_found", "Article not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, article)
}

// ExternalSearchHandler handles external search requests (read-only, simplified)
func (h *HTTPServer) ExternalSearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	// Basic validation
	if req.Query == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_query", "Query is required")
		return
	}

	// Set default size if not provided
	if req.Size == 0 {
		req.Size = 10
	}

	// Limit max size for external agents
	if req.Size > 50 {
		req.Size = 50
	}

	resp, err := h.server.Search(ctx, &req)
	if err != nil {
		log.Printf("Error performing external search: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to perform search")
		return
	}

	// Simplified response without AI-generated answer for external agents
	simplifiedResponse := map[string]interface{}{
		"query":   req.Query,
		"results": resp.Sources,
		"total":   len(resp.Sources),
		"took":    resp.Took,
	}

	writeJSONResponse(w, http.StatusOK, simplifiedResponse)
}

// ExternalKeywordSearchHandler handles external keyword search requests (read-only)
func (h *HTTPServer) ExternalKeywordSearchHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := r.URL.Query().Get("q")
	if query == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_query", "Query parameter 'q' is required")
		return
	}

	lang := r.URL.Query().Get("lang")

	sizeStr := r.URL.Query().Get("size")
	size := 10
	if sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 50 {
			size = s // Max 50 for external agents
		}
	}

	fromStr := r.URL.Query().Get("from")
	from := 0
	if fromStr != "" {
		if f, err := strconv.Atoi(fromStr); err == nil && f >= 0 {
			from = f
		}
	}

	resp, err := h.server.opensearchClient.KeywordSearch(ctx, query, lang, size, from)
	if err != nil {
		log.Printf("Error performing external keyword search: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to perform search")
		return
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// SetupRoutes configures the HTTP routes
func (h *HTTPServer) SetupRoutes() *chi.Mux {
	router := chi.NewRouter()

	// API routes
	router.Route("/api/v1", func(r chi.Router) {
		// Articles
		r.Post("/articles", h.AddArticleHandler)
		r.Get("/articles/{id}", h.GetArticleHandler)
		r.Get("/articles/ws", h.WebSocketAddArticleHandler)
		r.Get("/articles/bulk/ws", h.WebSocketBulkAddArticleHandler)

		// Search
		r.Post("/search", h.SearchHandler)
		r.Post("/search/stream", h.SearchStreamHandler)
		r.Get("/search/keyword", h.KeywordSearchHandler)
		r.Get("/search/ws", h.WebSocketSearchHandler)

		// Utilities
		r.Get("/languages", h.GetSupportedLanguagesHandler)

		// External agent APIs (read-only)
		r.Route("/external", func(r chi.Router) {
			r.Get("/articles", h.ExternalArticleListHandler)
			r.Get("/articles/{id}", h.ExternalArticleDetailHandler)
			r.Post("/search", h.ExternalSearchHandler)
			r.Get("/search/keyword", h.ExternalKeywordSearchHandler)
		})
	})

	return router
}
