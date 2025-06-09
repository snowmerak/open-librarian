package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"github.com/snowmerak/open-librarian/lib/util/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// RegisterUserRoutes registers user-related routes
func (s *Server) RegisterUserRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		// Public routes (no authentication required)
		r.Post("/", s.createUserHandler)
		r.Post("/auth", s.authenticateUserHandler)
		r.Post("/refresh", s.refreshTokenHandler)

		// Protected routes (authentication required)
		r.Group(func(r chi.Router) {
			r.Use(s.JWTMiddleware(s.jwtService))

			// Routes that require ownership verification
			r.Group(func(r chi.Router) {
				r.Use(RequireOwnership())
				r.Get("/{id}", s.getUserByIDHandler)
				r.Put("/{id}", s.updateUserHandler)
				r.Put("/{id}/password", s.changePasswordHandler)
				r.Delete("/{id}", s.deleteUserHandler)
			})

			// Routes accessible to any authenticated user
			r.Get("/username/{username}", s.getUserByUsernameHandler)
			r.Get("/me", s.getCurrentUserHandler)
		})
	})
}

// createUserHandler handles user creation
func (s *Server) createUserHandler(w http.ResponseWriter, r *http.Request) {
	userLogger := logger.NewLogger("create_user").StartWithMsg("Creating new user")

	var req mongo.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		userLogger.Error().Err(err).Msg("Invalid request body")
		userLogger.EndWithError(err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Email == "" || req.Username == "" || req.Password == "" {
		userLogger.Error().
			Str("email", req.Email).
			Str("username", req.Username).
			Bool("password_empty", req.Password == "").
			Msg("Missing required fields")
		userLogger.EndWithError(nil)
		http.Error(w, "Email, username, and password are required", http.StatusBadRequest)
		return
	}

	userLogger.Info().
		Str("email", req.Email).
		Str("username", req.Username).
		Msg("Creating user with valid request")

	user, err := s.mongoClient.CreateUser(r.Context(), req)
	if err != nil {
		if err.Error() == "user with this email or username already exists" {
			userLogger.Warn().
				Str("email", req.Email).
				Str("username", req.Username).
				Msg("User already exists")
			userLogger.EndWithError(err)
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		userLogger.Error().Err(err).Msg("Failed to create user")
		userLogger.EndWithError(err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	userLogger.DataCreated("user", user.ID.Hex(), map[string]interface{}{
		"email":    user.Email,
		"username": user.Username,
	})
	userLogger.EndWithMsg("User created successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// authenticateUserHandler handles user authentication and returns JWT token
func (s *Server) authenticateUserHandler(w http.ResponseWriter, r *http.Request) {
	authLogger := logger.NewLogger("authenticate_user").StartWithMsg("Authenticating user")

	var credentials mongo.UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		authLogger.Error().Err(err).Msg("Invalid request body")
		authLogger.EndWithError(err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if credentials.Email == "" || credentials.Password == "" {
		authLogger.Error().
			Str("email", credentials.Email).
			Bool("password_empty", credentials.Password == "").
			Msg("Missing required credentials")
		authLogger.EndWithError(nil)
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	authLogger.Info().Str("email", credentials.Email).Msg("Attempting authentication")

	authResponse, err := s.mongoClient.AuthenticateUserWithToken(r.Context(), credentials, s.jwtService)
	if err != nil {
		if err.Error() == "invalid email or password" {
			authLogger.Warn().Str("email", credentials.Email).Msg("Authentication failed - invalid credentials")
			authLogger.EndWithError(err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		authLogger.Error().Err(err).Str("email", credentials.Email).Msg("Authentication failed")
		authLogger.EndWithError(err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	authLogger.Info().
		Str("email", credentials.Email).
		Str("user_id", authResponse.User.ID.Hex()).
		Msg("User authenticated successfully")
	authLogger.EndWithMsg("Authentication complete")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse)
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	Token string `json:"token"`
}

// refreshTokenHandler handles JWT token refresh
func (s *Server) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	newToken, err := s.jwtService.RefreshToken(req.Token)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}

	response := map[string]string{
		"token": newToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

// getCurrentUserHandler returns the current authenticated user's information
func (s *Server) getCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUserFromContext(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
