package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/snowmerak/open-librarian/lib/aggregator/api"
)

func main() {
	// Configuration from environment variables with defaults
	port := getEnv("PORT", "8080")
	opensearchURL := getEnv("OPENSEARCH_URL", "http://localhost:9200")
	ollamaURL := getEnv("OLLAMA_URL", "http://localhost:11434")
	qdrantHost := getEnv("QDRANT_HOST", "localhost")
	qdrantPortStr := getEnv("QDRANT_PORT", "6334")
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017/open_librarian")
	jwtSecret := getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production")

	// Convert qdrant port to integer
	qdrantPort := 6334 // default
	if portNum, err := parsePort(qdrantPortStr); err == nil {
		qdrantPort = portNum
	} else {
		log.Printf("Invalid QDRANT_PORT '%s', using default 6334", qdrantPortStr)
	}

	// Initialize API server
	apiServer, err := api.NewServer(ollamaURL, opensearchURL, qdrantHost, mongoURI, jwtSecret, qdrantPort)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}
	httpServer := api.NewHTTPServer(apiServer)

	// Setup router with middleware
	router := setupRouter(httpServer)

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Minute, // Increased to match longest operation timeout
		WriteTimeout: 15 * time.Minute, // Increased to match longest operation timeout
		IdleTimeout:  5 * time.Minute,  // Increased but kept reasonable
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", port)
		log.Printf("Swagger UI available at: http://localhost:%s/swagger/", port)
		log.Printf("Health check at: http://localhost:%s/health", port)
		log.Printf("Public files served from: http://localhost:%s/public/", port)
		log.Printf("API available at: http://localhost:%s/api/", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(httpServer *api.HTTPServer) *chi.Mux {
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(30 * time.Minute)) // Increased to be less than HTTP server timeout

	// CORS configuration
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Setup API routes
	apiRoutes := httpServer.SetupRoutes()
	router.Mount("/", apiRoutes)

	// Health check endpoint
	router.Get("/health", httpServer.HealthCheckHandler)

	// Serve static files from public directory
	publicFS := http.FileServer(http.Dir("./cmd/server/public/"))
	router.Handle("/public/*", http.StripPrefix("/public/", publicFS))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/public/index.html", http.StatusFound)
	})

	// Serve Swagger UI
	swaggerFS := http.FileServer(http.Dir("./cmd/server/swagger/"))
	router.Handle("/swagger/*", http.StripPrefix("/swagger/", swaggerFS))

	return router
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parsePort(portStr string) (int, error) {
	if portStr == "" {
		return 0, fmt.Errorf("empty port string")
	}
	port := 0
	for _, char := range portStr {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid port format")
		}
		port = port*10 + int(char-'0')
	}
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("port out of range")
	}
	return port, nil
}
