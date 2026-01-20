package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/snowmerak/open-librarian/lib/util/logger"
	"github.com/snowmerak/open-librarian/lib/util/parser"
)

// UploadArticleHandler handles file upload and article indexing
func (h *HTTPServer) UploadArticleHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.NewLoggerWithContext(ctx, "upload_article").Start()
	defer log.End()

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Error().Err(err).Msg("Failed to parse multipart form")
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get file from form")
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := header.Filename
	log.Info().Str("filename", filename).Msg("Processing uploaded file")

	doc, err := parser.Parse(file, filename)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse file")
		http.Error(w, fmt.Sprintf("Failed to parse file: %v", err), http.StatusBadRequest)
		return
	}

	if doc.Content == "" {
		log.Warn().Msg("Parsed content is empty")
		http.Error(w, "File content is empty or could not be valid text", http.StatusBadRequest)
		return
	}

	// Create ArticleRequest
	req := &ArticleRequest{
		Title:   doc.Title,
		Content: doc.Content,
	}

	// Override/Set metadata from form values
	if title := r.FormValue("title"); title != "" {
		req.Title = title
	}
	if author := r.FormValue("author"); author != "" {
		req.Author = author
	}
	if originalURL := r.FormValue("original_url"); originalURL != "" {
		req.OriginalURL = originalURL
	}
	if createdDate := r.FormValue("created_date"); createdDate != "" {
		req.CreatedDate = createdDate
	}

	// Call AddArticle
	resp, err := h.server.AddArticle(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add article")
		http.Error(w, fmt.Sprintf("Failed to index article: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
