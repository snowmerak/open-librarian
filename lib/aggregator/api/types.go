package api

import (
	"github.com/snowmerak/open-librarian/lib/client/opensearch"
)

// ArticleRequest represents the request to add an article
type ArticleRequest struct {
	Title       string `json:"title" validate:"required"`
	Content     string `json:"content" validate:"required"`
	OriginalURL string `json:"original_url,omitempty"`
	Author      string `json:"author,omitempty"`
	CreatedDate string `json:"created_date,omitempty"` // RFC3339 format (e.g., "2023-12-25T15:30:00Z")
}

// SearchRequest represents the search request
type SearchRequest struct {
	Query     string `json:"query" validate:"required"`
	Size      int    `json:"size,omitempty"`
	From      int    `json:"from,omitempty"`
	DateFrom  string `json:"date_from,omitempty"`  // RFC3339 format for filtering articles created after this date
	DateTo    string `json:"date_to,omitempty"`    // RFC3339 format for filtering articles created before this date
	SessionID string `json:"session_id,omitempty"` // For chat history
}

// ArticleResponse represents the response after adding an article
type ArticleResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// SearchResultWithScore represents a search result with score
type SearchResultWithScore struct {
	Article opensearch.Article `json:"article"`
	Score   float64            `json:"score"`
	Source  string             `json:"source"` // "keyword" or "vector"
}

// SearchResponse represents the search response
type SearchResponse struct {
	Answer  string                  `json:"answer"`
	Sources []SearchResultWithScore `json:"sources"`
	Took    int                     `json:"took"`
}

// BulkArticleRequest represents a bulk upload request
type BulkArticleRequest struct {
	Articles []ArticleRequest `json:"articles" validate:"required"`
}

// BulkArticleResponse represents the response for bulk upload
type BulkArticleResponse struct {
	SuccessCount int                 `json:"success_count"`
	ErrorCount   int                 `json:"error_count"`
	Results      []BulkArticleResult `json:"results"`
}

// BulkArticleResult represents the result of a single article in bulk upload
type BulkArticleResult struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WSMessage represents WebSocket message type
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ProgressCallback represents a function that can be called to report progress
type ProgressCallback func(step string, progress int, total int) error

// BulkProgressCallback represents a function that can be called to report bulk upload progress
type BulkProgressCallback func(articleIndex int, totalArticles int, currentStep string, stepProgress int, stepTotal int, result *BulkArticleResult) error

// UserArticlesRequest represents the request to get user's articles by date range
type UserArticlesRequest struct {
	DateFrom string `json:"date_from,omitempty"` // RFC3339 format for filtering articles created after this date
	DateTo   string `json:"date_to,omitempty"`   // RFC3339 format for filtering articles created before this date
	Size     int    `json:"size,omitempty"`      // Number of articles to return (default: 20, max: 100)
	From     int    `json:"from,omitempty"`      // Offset for pagination (default: 0)
}

// UserArticlesResponse represents the response for user articles query
type UserArticlesResponse struct {
	Articles []opensearch.Article `json:"articles"`
	Total    int                  `json:"total"`
	From     int                  `json:"from"`
	Size     int                  `json:"size"`
	Took     int                  `json:"took"`
}
