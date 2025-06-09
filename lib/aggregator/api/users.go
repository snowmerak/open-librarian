package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// RegisterUserRoutes registers user-related routes
func (s *Server) RegisterUserRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/", s.createUserHandler)
		r.Post("/auth", s.authenticateUserHandler)
		r.Get("/{id}", s.getUserByIDHandler)
		r.Get("/username/{username}", s.getUserByUsernameHandler)
		r.Put("/{id}", s.updateUserHandler)
		r.Put("/{id}/password", s.changePasswordHandler)
		r.Delete("/{id}", s.deleteUserHandler)
	})
}

// createUserHandler handles user creation
func (s *Server) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req mongo.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Email == "" || req.Username == "" || req.Password == "" {
		http.Error(w, "Email, username, and password are required", http.StatusBadRequest)
		return
	}

	user, err := s.mongoClient.CreateUser(r.Context(), req)
	if err != nil {
		if err.Error() == "user with this email or username already exists" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// authenticateUserHandler handles user authentication
func (s *Server) authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	var credentials mongo.UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if credentials.Email == "" || credentials.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	user, err := s.mongoClient.AuthenticateUser(r.Context(), credentials)
	if err != nil {
		if err.Error() == "invalid email or password" {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// getUserByIDHandler retrieves a user by ID
func (s *Server) getUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	user, err := s.mongoClient.GetUserByID(r.Context(), id)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// getUserByUsernameHandler retrieves a user by username
func (s *Server) getUserByUsernameHandler(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	user, err := s.mongoClient.GetUserByUsername(r.Context(), username)
	if err != nil {
		if err.Error() == "user not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to retrieve user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// updateUserHandler updates user information
func (s *Server) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	var updates bson.M
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.mongoClient.UpdateUser(r.Context(), id, updates); err != nil {
		if err.Error() == "user not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// changePasswordHandler handles password changes
func (s *Server) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.OldPassword == "" || req.NewPassword == "" {
		http.Error(w, "Old password and new password are required", http.StatusBadRequest)
		return
	}

	if err := s.mongoClient.ChangePassword(r.Context(), id, req.OldPassword, req.NewPassword); err != nil {
		if err.Error() == "user not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err.Error() == "invalid old password" {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to change password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteUserHandler deletes a user
func (s *Server) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	if err := s.mongoClient.DeleteUser(r.Context(), id); err != nil {
		if err.Error() == "user not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
