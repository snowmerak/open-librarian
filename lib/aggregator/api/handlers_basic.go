package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
		addLogger.Error().Err(err).Msg("Error adding article")
		addLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusInternalServerError, "processing_error", "Failed to process article")
		return
	}

	addLogger.Info().Str("article_id", resp.ID).Msg("Article added successfully")
	addLogger.EndWithMsg("Add article request completed")
	writeJSONResponse(w, http.StatusCreated, resp)
}

// SearchHandler handles search requests
func (h *HTTPServer) SearchHandler(w http.ResponseWriter, r *http.Request) {
	searchLogger := logger.NewLogger("search-handler")
	searchLogger.StartWithMsg("Processing search request")

	ctx := r.Context()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		searchLogger.Error().Err(err).Msg("Invalid JSON format")
		searchLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	searchLogger.Info().Str("query", req.Query).Msg("Search request details")

	// Basic validation
	if req.Query == "" {
		searchLogger.Error().Msg("Missing query in request")
		searchLogger.EndWithError(fmt.Errorf("query is required"))
		writeErrorResponse(w, http.StatusBadRequest, "missing_query", "Query is required")
		return
	}

	resp, err := h.server.Search(ctx, &req)
	if err != nil {
		searchLogger.Error().Err(err).Msg("Error performing search")
		searchLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to perform search")
		return
	}

	searchLogger.Info().Int("result_count", len(resp.Sources)).Msg("Search completed successfully")
	searchLogger.EndWithMsg("Search request completed")
	writeJSONResponse(w, http.StatusOK, resp)
}

// GetArticleHandler handles getting a specific article
func (h *HTTPServer) GetArticleHandler(w http.ResponseWriter, r *http.Request) {
	getLogger := logger.NewLogger("get-article-handler")
	getLogger.StartWithMsg("Processing get article request")

	ctx := r.Context()

	id := chi.URLParam(r, "id")
	if id == "" {
		getLogger.Error().Msg("Missing article ID in request")
		getLogger.EndWithError(fmt.Errorf("article ID is required"))
		writeErrorResponse(w, http.StatusBadRequest, "missing_id", "Article ID is required")
		return
	}

	getLogger.Info().Str("article_id", id).Msg("Getting article by ID")

	article, err := h.server.GetArticle(ctx, id)
	if err != nil {
		getLogger.Error().Err(err).Str("article_id", id).Msg("Error getting article")
		getLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusNotFound, "article_not_found", "Article not found")
		return
	}

	getLogger.Info().Str("article_id", id).Str("title", article.Title).Msg("Article retrieved successfully")
	getLogger.EndWithMsg("Get article request completed")
	writeJSONResponse(w, http.StatusOK, article)
}

// KeywordSearchHandler handles keyword-only search requests
func (h *HTTPServer) KeywordSearchHandler(w http.ResponseWriter, r *http.Request) {
	keywordLogger := logger.NewLogger("keyword-search-handler")
	keywordLogger.StartWithMsg("Processing keyword search request")

	ctx := r.Context()

	query := r.URL.Query().Get("q")
	if query == "" {
		keywordLogger.Error().Msg("Missing query parameter 'q'")
		keywordLogger.EndWithError(fmt.Errorf("query parameter 'q' is required"))
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

	keywordLogger.Info().Str("query", query).Str("lang", lang).Int("size", size).Int("from", from).Msg("Keyword search request details")

	resp, err := h.server.opensearchClient.KeywordSearch(ctx, query, lang, size, from)
	if err != nil {
		keywordLogger.Error().Err(err).Msg("Error performing keyword search")
		keywordLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to perform search")
		return
	}

	keywordLogger.Info().Int("result_count", len(resp.Results)).Msg("Keyword search completed successfully")
	keywordLogger.EndWithMsg("Keyword search request completed")
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
		listLogger := logger.NewLogger("external-article-list")
		listLogger.Error().Err(err).Msg("Error listing articles")
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
	extDetailLogger := logger.NewLogger("external-article-detail")
	extDetailLogger.StartWithMsg("Processing external article detail request")

	ctx := r.Context()

	id := chi.URLParam(r, "id")
	if id == "" {
		extDetailLogger.Error().Msg("Missing article ID in request")
		extDetailLogger.EndWithError(fmt.Errorf("article ID is required"))
		writeErrorResponse(w, http.StatusBadRequest, "missing_id", "Article ID is required")
		return
	}

	extDetailLogger.Info().Str("article_id", id).Msg("Getting article for external request")

	article, err := h.server.GetArticle(ctx, id)
	if err != nil {
		extDetailLogger.Error().Err(err).Str("article_id", id).Msg("Error getting article")
		extDetailLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusNotFound, "article_not_found", "Article not found")
		return
	}

	extDetailLogger.Info().Str("article_id", id).Str("title", article.Title).Msg("Article retrieved for external request")
	extDetailLogger.EndWithMsg("External article detail request completed")
	writeJSONResponse(w, http.StatusOK, article)
}

// ExternalSearchHandler handles external search requests (read-only, simplified)
func (h *HTTPServer) ExternalSearchHandler(w http.ResponseWriter, r *http.Request) {
	extSearchLogger := logger.NewLogger("external-search-handler")
	extSearchLogger.StartWithMsg("Processing external search request")

	ctx := r.Context()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		extSearchLogger.Error().Err(err).Msg("Invalid JSON format")
		extSearchLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	// Basic validation
	if req.Query == "" {
		extSearchLogger.Error().Msg("Missing query in request")
		extSearchLogger.EndWithError(fmt.Errorf("query is required"))
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

	extSearchLogger.Info().Str("query", req.Query).Int("size", req.Size).Msg("External search request details")

	resp, err := h.server.Search(ctx, &req)
	if err != nil {
		extSearchLogger.Error().Err(err).Msg("Error performing external search")
		extSearchLogger.EndWithError(err)
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

	extSearchLogger.Info().Int("result_count", len(resp.Sources)).Msg("External search completed successfully")
	extSearchLogger.EndWithMsg("External search request completed")
	writeJSONResponse(w, http.StatusOK, simplifiedResponse)
}

// ExternalKeywordSearchHandler handles external keyword search requests (read-only)
func (h *HTTPServer) ExternalKeywordSearchHandler(w http.ResponseWriter, r *http.Request) {
	extKeywordLogger := logger.NewLogger("external-keyword-search")
	extKeywordLogger.StartWithMsg("Processing external keyword search request")

	ctx := r.Context()

	query := r.URL.Query().Get("q")
	if query == "" {
		extKeywordLogger.Error().Msg("Missing query parameter 'q'")
		extKeywordLogger.EndWithError(fmt.Errorf("query parameter 'q' is required"))
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

	extKeywordLogger.Info().Str("query", query).Str("lang", lang).Int("size", size).Int("from", from).Msg("External keyword search request details")

	resp, err := h.server.opensearchClient.KeywordSearch(ctx, query, lang, size, from)
	if err != nil {
		extKeywordLogger.Error().Err(err).Msg("Error performing external keyword search")
		extKeywordLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusInternalServerError, "search_error", "Failed to perform search")
		return
	}

	extKeywordLogger.Info().Int("result_count", len(resp.Results)).Msg("External keyword search completed successfully")
	extKeywordLogger.EndWithMsg("External keyword search request completed")
	writeJSONResponse(w, http.StatusOK, resp)
}

// DeleteArticleHandler handles article deletion requests
func (h *HTTPServer) DeleteArticleHandler(w http.ResponseWriter, r *http.Request) {
	deleteHandlerLogger := logger.NewLogger("delete-article-handler")
	deleteHandlerLogger.StartWithMsg("Processing delete article request")

	ctx := r.Context()

	id := chi.URLParam(r, "id")
	if id == "" {
		deleteHandlerLogger.Error().Msg("Missing article ID in request")
		deleteHandlerLogger.EndWithError(fmt.Errorf("article ID is required"))
		writeErrorResponse(w, http.StatusBadRequest, "missing_id", "Article ID is required")
		return
	}

	deleteHandlerLogger.Info().Str("article_id", id).Msg("Deleting article by ID")

	err := h.server.DeleteArticle(ctx, id)
	if err != nil {
		deleteHandlerLogger.Error().Err(err).Str("article_id", id).Msg("Error deleting article")
		deleteHandlerLogger.EndWithError(err)
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

	deleteHandlerLogger.Info().Str("article_id", id).Msg("Article deleted successfully")
	deleteHandlerLogger.EndWithMsg("Delete article request completed")
	w.WriteHeader(http.StatusNoContent)
}

// GetUserArticlesHandler handles user articles retrieval requests with date range
func (h *HTTPServer) GetUserArticlesHandler(w http.ResponseWriter, r *http.Request) {
	userArticlesLogger := logger.NewLogger("get-user-articles-handler")
	userArticlesLogger.StartWithMsg("Processing get user articles request")

	ctx := r.Context()

	var req UserArticlesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		userArticlesLogger.Error().Err(err).Msg("Invalid JSON format")
		userArticlesLogger.EndWithError(err)
		writeErrorResponse(w, http.StatusBadRequest, "invalid_json", "Invalid JSON format")
		return
	}

	userArticlesLogger.Info().
		Str("date_from", req.DateFrom).
		Str("date_to", req.DateTo).
		Int("size", req.Size).
		Int("from", req.From).
		Msg("User articles request details")

	// Validate date formats if provided
	if req.DateFrom != "" {
		if _, err := time.Parse(time.RFC3339, req.DateFrom); err != nil {
			userArticlesLogger.Error().Err(err).Str("date_from", req.DateFrom).Msg("Invalid date_from format")
			userArticlesLogger.EndWithError(err)
			writeErrorResponse(w, http.StatusBadRequest, "invalid_date_format", "date_from must be in RFC3339 format (e.g., 2023-12-25T15:30:00Z)")
			return
		}
	}

	if req.DateTo != "" {
		if _, err := time.Parse(time.RFC3339, req.DateTo); err != nil {
			userArticlesLogger.Error().Err(err).Str("date_to", req.DateTo).Msg("Invalid date_to format")
			userArticlesLogger.EndWithError(err)
			writeErrorResponse(w, http.StatusBadRequest, "invalid_date_format", "date_to must be in RFC3339 format (e.g., 2023-12-25T15:30:00Z)")
			return
		}
	}

	resp, err := h.server.GetUserArticles(ctx, &req)
	if err != nil {
		userArticlesLogger.Error().Err(err).Msg("Error getting user articles")
		userArticlesLogger.EndWithError(err)
		if strings.Contains(err.Error(), "authentication required") {
			writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
			return
		}
		writeErrorResponse(w, http.StatusInternalServerError, "query_error", "Failed to get user articles")
		return
	}

	userArticlesLogger.Info().Int("total", resp.Total).Int("returned", len(resp.Articles)).Msg("User articles retrieved successfully")
	userArticlesLogger.EndWithMsg("Get user articles request completed")
	writeJSONResponse(w, http.StatusOK, resp)
}
