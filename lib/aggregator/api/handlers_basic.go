package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/snowmerak/open-librarian/lib/util/logger"
)

// HealthCheckHandler handles health check requests
func (h *HTTPServer) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	healthLogger := logger.NewLogger("health-check-handler")
	healthLogger.StartWithMsg("Processing health check request")

	ctx := r.Context()

	if err := h.server.HealthCheck(ctx); err != nil {
		healthLogger.Error().Err(err).Msg("Health check failed")
		healthLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusServiceUnavailable, "service_unavailable", err.Error())
		return
	}

	response := map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}

	healthLogger.Info().Msg("Health check passed")
	healthLogger.EndWithMsg("Health check completed successfully")
	writeJSONResponse(w, http.StatusOK, response)
}

// AddArticleHandler handles article addition requests
func (h *HTTPServer) AddArticleHandler(w http.ResponseWriter, r *http.Request) {
	addLogger := logger.NewLogger("add-article-handler")
	addLogger.StartWithMsg("Processing add article request")

	ctx := r.Context()

	var req ArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		addLogger.Error().Err(err).Msg("Invalid JSON format")
		addLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	addLogger.Info().Str("title", req.Title).Msg("Article addition request details")

	// Basic validation
	if req.Title == "" {
		addLogger.Error().Msg("Missing title in request")
		addLogger.EndWithError(fmt.Errorf("title is required"))
		writeErrorResponse(w, http.StatusBadRequest, "missing_title", "Title is required")
		return
	}
	if req.Content == "" {
		addLogger.Error().Msg("Missing content in request")
		addLogger.EndWithError(fmt.Errorf("content is required"))
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

// DeleteArticleHandler handles article deletion requests
func (h *HTTPServer) DeleteArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")
	if id == "" {
		writeErrorResponse(w, http.StatusBadRequest, "missing_id", "Article ID is required")
		return
	}

	err := h.server.DeleteArticle(ctx, id)
	if err != nil {
		log.Printf("Error deleting article: %v", err)
		if err.Error() == "article not found" {
			writeErrorResponse(w, http.StatusNotFound, "not_found", "Article not found")
			return
		}
		if err.Error() == "permission denied: only the registrar can delete this article" {
			writeErrorResponse(w, http.StatusForbidden, "permission_denied", "Only the registrar can delete this article")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "deletion_error", "Failed to delete article")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
