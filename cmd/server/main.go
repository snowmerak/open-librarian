package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/snowmerak/open-librarian/lib/aggregator/api"
	"github.com/snowmerak/open-librarian/lib/util/logger"

	_ "github.com/snowmerak/open-librarian/lib/util/logger"
)

func main() {
	// Initialize main scope logger
	mainLogger := logger.NewLogger("main").StartWithMsg("Starting Open Librarian server")
	defer mainLogger.EndWithMsg("Open Librarian server shutdown complete")

	// Configuration from environment variables with defaults
	configLogger := logger.NewLogger("config").StartWithMsg("Loading configuration")
	port := getEnv("PORT", "8080")
	opensearchURL := getEnv("OPENSEARCH_URL", "http://localhost:9200")
	ollamaURL := getEnv("OLLAMA_URL", "http://localhost:11434")
	qdrantHost := getEnv("QDRANT_HOST", "localhost")
	qdrantPortStr := getEnv("QDRANT_PORT", "6334")
	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017/open_librarian")
	jwtSecret := getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production")

	configLogger.Info().
		Str("port", port).
		Str("opensearch_url", opensearchURL).
		Str("ollama_url", ollamaURL).
		Str("qdrant_host", qdrantHost).
		Str("qdrant_port", qdrantPortStr).
		Str("mongo_uri", mongoURI).
		Bool("jwt_secret_default", jwtSecret == "your-super-secret-jwt-key-change-this-in-production").
		Msg("Configuration loaded")
	configLogger.EndWithMsg("Configuration loading complete")

	// Convert qdrant port to integer
	qdrantPort := 6334 // default
	if portNum, err := parsePort(qdrantPortStr); err == nil {
		qdrantPort = portNum
		mainLogger.Info().Int("qdrant_port", qdrantPort).Msg("Qdrant port parsed successfully")
	} else {
		mainLogger.Warn().Err(err).Str("qdrant_port_str", qdrantPortStr).Int("default_port", 6334).Msg("Invalid QDRANT_PORT, using default")
	}

	// Initialize API server
	apiInitLogger := logger.NewLogger("api_init").StartWithMsg("Initializing API server")
	apiServer, err := api.NewServer(ollamaURL, opensearchURL, qdrantHost, mongoURI, jwtSecret, qdrantPort)
	if err != nil {
		apiInitLogger.EndWithError(err)
		mainLogger.Error().Err(err).Msg("Failed to create API server")
		os.Exit(1)
	}
	httpServer := api.NewHTTPServer(apiServer)
	apiInitLogger.EndWithMsg("API server initialization complete")

	// Setup router with middleware
	routerLogger := logger.NewLogger("router_setup").StartWithMsg("Setting up router and middleware")
	router := setupRouter(httpServer)
	routerLogger.EndWithMsg("Router setup complete")

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Minute, // Increased to match longest operation timeout
		WriteTimeout: 15 * time.Minute, // Increased to match longest operation timeout
		IdleTimeout:  5 * time.Minute,  // Increased but kept reasonable
	}

	// Start server in a goroutine
	serverLogger := logger.NewLogger("http_server").StartWithMsg("Starting HTTP server")
	go func() {
		serverLogger.Info().
			Str("port", port).
			Str("swagger_url", fmt.Sprintf("http://localhost:%s/swagger/", port)).
			Str("health_url", fmt.Sprintf("http://localhost:%s/health", port)).
			Str("public_url", fmt.Sprintf("http://localhost:%s/public/", port)).
			Str("api_url", fmt.Sprintf("http://localhost:%s/api/", port)).
			Msg("Server endpoints configured")

		serverLogger.Info().Str("port", port).Msg("Starting server")
		serverLogger.Info().Str("swagger_url", fmt.Sprintf("http://localhost:%s/swagger/", port)).Msg("Swagger UI available")
		serverLogger.Info().Str("health_url", fmt.Sprintf("http://localhost:%s/health", port)).Msg("Health check endpoint available")
		serverLogger.Info().Str("public_url", fmt.Sprintf("http://localhost:%s/public/", port)).Msg("Public files endpoint available")
		serverLogger.Info().Str("api_url", fmt.Sprintf("http://localhost:%s/api/", port)).Msg("API endpoint available")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverLogger.EndWithError(err)
			mainLogger.Error().Err(err).Msg("Server failed to start")
			os.Exit(1)
		}
	}()
	serverLogger.EndWithMsg("HTTP server started successfully")

	// Wait for interrupt signal to gracefully shutdown
	shutdownLogger := logger.NewLogger("shutdown").StartWithMsg("Waiting for shutdown signal")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	shutdownLogger.Info().Str("signal", sig.String()).Msg("Shutdown signal received")
	shutdownLogger.Info().Msg("Shutting down server")

	// Graceful shutdown with timeout
	shutdownLogger.Info().Msg("Starting graceful shutdown")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		shutdownLogger.EndWithError(err)
		mainLogger.Error().Err(err).Msg("Server forced to shutdown")
		os.Exit(1)
	}

	shutdownLogger.EndWithMsg("Graceful shutdown completed")
	shutdownLogger.Info().Msg("Server exited")
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
