package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"github.com/snowmerak/open-librarian/lib/util/logger"
)

// GetChatHistoryHandler returns the chat history for a user
func (h *HTTPServer) GetChatHistoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.NewLoggerWithContext(ctx, "get_chat_history").Start()
	defer log.End()

	// TODO: Replace with real user ID from context once auth is enforced
	userID := ""

	sessions, err := h.server.mongoClient.GetChatSessions(ctx, userID, 50)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get chat sessions")
		http.Error(w, "Failed to get history", http.StatusInternalServerError)
		return
	}

	if sessions == nil {
		sessions = []mongo.ChatSession{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// DeleteChatSessionHandler deletes a chat session
func (h *HTTPServer) DeleteChatSessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	log := logger.NewLoggerWithContext(ctx, "delete_chat_session").WithField("id", id).Start()
	defer log.End()

	err := h.server.mongoClient.DeleteChatSession(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to delete session")
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetChatSessionHandler returns a single chat session
func (h *HTTPServer) GetChatSessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	session, err := h.server.mongoClient.GetChatSession(ctx, id)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}
