package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/snowmerak/open-librarian/lib/client/mongo"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// UserContextKey is the key for storing user in context
	UserContextKey ContextKey = "user"
	// ClaimsContextKey is the key for storing JWT claims in context
	ClaimsContextKey ContextKey = "claims"
)

// JWTMiddleware creates a middleware for JWT authentication
func (s *Server) JWTMiddleware(jwtService *mongo.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check if header starts with "Bearer "
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Authorization header must start with 'Bearer '", http.StatusUnauthorized)
				return
			}

			// Extract token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == "" {
				http.Error(w, "Token required", http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Get user from database to ensure user still exists
			user, err := s.mongoClient.GetUserFromToken(r.Context(), tokenString, jwtService)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Add user and claims to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			ctx = context.WithValue(ctx, ClaimsContextKey, claims)

			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTMiddleware creates a middleware for optional JWT authentication
// If token is provided, it validates and adds user to context
// If no token is provided, it continues without user context
func (s *Server) OptionalJWTMiddleware(jwtService *mongo.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No token provided, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Check if header starts with "Bearer "
			if !strings.HasPrefix(authHeader, "Bearer ") {
				// Invalid format, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Extract token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == "" {
				// Empty token, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Validate token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				// Invalid token, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Get user from database
			user, err := s.mongoClient.GetUserFromToken(r.Context(), tokenString, jwtService)
			if err != nil {
				// User not found, continue without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Add user and claims to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			ctx = context.WithValue(ctx, ClaimsContextKey, claims)

			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext extracts the user from request context
func GetUserFromContext(r *http.Request) (*mongo.User, bool) {
	user, ok := r.Context().Value(UserContextKey).(*mongo.User)
	return user, ok
}

// GetClaimsFromContext extracts the JWT claims from request context
func GetClaimsFromContext(r *http.Request) (*mongo.JWTClaims, bool) {
	claims, ok := r.Context().Value(ClaimsContextKey).(*mongo.JWTClaims)
	return claims, ok
}

// RequireOwnership middleware ensures the authenticated user can only access their own data
func RequireOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUserFromContext(r)
			if !ok {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Get user ID from URL parameter
			userID := getUserIDFromURL(r)
			if userID == "" {
				http.Error(w, "User ID required", http.StatusBadRequest)
				return
			}

			// Check if the authenticated user is accessing their own data
			if user.ID.Hex() != userID {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getUserIDFromURL extracts user ID from URL path
func getUserIDFromURL(r *http.Request) string {
	// This is a simple implementation that assumes the user ID is in the URL
	// You might need to adjust this based on your routing structure
	path := r.URL.Path
	parts := strings.Split(path, "/")

	// Look for user ID in URL patterns like /api/v1/users/{id}
	for i, part := range parts {
		if part == "users" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}
