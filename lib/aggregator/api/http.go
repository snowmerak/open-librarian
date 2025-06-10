package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
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

// SetupRoutes configures the HTTP routes
func (h *HTTPServer) SetupRoutes() *chi.Mux {
	router := chi.NewRouter()

	// API routes
	router.Route("/api/v1", func(r chi.Router) {
		// WebSocket routes (handle authentication internally)
		r.Get("/articles/ws", h.WebSocketAddArticleHandler)
		r.Get("/articles/bulk/ws", h.WebSocketBulkAddArticleHandler)
		r.Get("/search/ws", h.WebSocketSearchHandler)

		// Articles (protected routes)
		r.Group(func(r chi.Router) {
			r.Use(h.server.JWTMiddleware(h.server.jwtService))
			r.Post("/articles", h.AddArticleHandler)
			r.Delete("/articles/{id}", h.DeleteArticleHandler)
			r.Post("/articles/user", h.GetUserArticlesHandler) // New route for user articles by date range
		})

		// Articles (public routes)
		r.Get("/articles/{id}", h.GetArticleHandler)

		// Search
		r.Post("/search", h.SearchHandler)
		r.Get("/search/keyword", h.KeywordSearchHandler)
		r.Get("/search/ws", h.WebSocketSearchHandler)

		// Users
		h.server.RegisterUserRoutes(r)

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
