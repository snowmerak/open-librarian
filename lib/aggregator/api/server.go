package api

import (
	"context"
	"fmt"

	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"github.com/snowmerak/open-librarian/lib/client/ollama"
	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/client/qdrant"
	"github.com/snowmerak/open-librarian/lib/util/language"
	"github.com/snowmerak/open-librarian/lib/util/logger"
)

// Server represents the main API server
type Server struct {
	ollamaClient     *ollama.Client
	opensearchClient *opensearch.Client
	qdrantClient     *qdrant.Client
	mongoClient      *mongo.Client
	jwtService       *mongo.JWTService
	languageDetector *language.Detector
}

// NewServer creates a new API server instance
func NewServer(ollamaBaseURL, opensearchBaseURL, qdrantHost, mongoURI, jwtSecret string, qdrantPort int) (*Server, error) {
	serverLogger := logger.NewLogger("server_init").StartWithMsg("Initializing server components")

	// Initialize Qdrant client
	qdrantLogger := logger.NewLogger("qdrant_init").StartWithMsg("Initializing Qdrant client")
	qdrantClient, err := qdrant.NewClient(qdrantHost, qdrantPort)
	if err != nil {
		qdrantLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}
	qdrantLogger.Info().Str("host", qdrantHost).Int("port", qdrantPort).Msg("Qdrant client created")

	// Initialize Qdrant collection with 768-dimension vectors
	ctx := context.Background()
	if err := qdrantClient.CreateCollection(ctx); err != nil {
		qdrantLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to initialize Qdrant collection: %w", err)
	}
	qdrantLogger.EndWithMsg("Qdrant client initialization complete")

	// Create MongoDB client
	mongoLogger := logger.NewLogger("mongo_init").StartWithMsg("Initializing MongoDB client")
	mongoClient, err := mongo.New(mongoURI)
	if err != nil {
		mongoLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}
	mongoLogger.Info().Str("uri", mongoURI).Msg("MongoDB client created")

	// Test MongoDB connection and initialize database
	if err := mongoClient.Connect(ctx); err != nil {
		mongoLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	mongoLogger.Info().Msg("MongoDB connection established")

	if err := mongoClient.InitializeDatabase(ctx); err != nil {
		mongoLogger.EndWithError(err)
		return nil, fmt.Errorf("failed to initialize MongoDB database: %w", err)
	}
	mongoLogger.EndWithMsg("MongoDB client initialization complete")

	// Create JWT service
	jwtLogger := logger.NewLogger("jwt_init").StartWithMsg("Initializing JWT service")
	jwtService := mongo.NewJWTService(jwtSecret, "open-librarian")
	jwtLogger.EndWithMsg("JWT service initialization complete")

	// Create other clients
	ollamaClient := ollama.NewClient(ollamaBaseURL)
	opensearchClient := opensearch.NewClient(opensearchBaseURL)
	languageDetector := language.NewDetector()

	serverLogger.Info().
		Str("ollama_url", ollamaBaseURL).
		Str("opensearch_url", opensearchBaseURL).
		Str("qdrant_host", qdrantHost).
		Int("qdrant_port", qdrantPort).
		Str("mongo_uri", mongoURI).
		Msg("All server components initialized")

	serverLogger.EndWithMsg("Server initialization complete")

	return &Server{
		ollamaClient:     ollamaClient,
		opensearchClient: opensearchClient,
		qdrantClient:     qdrantClient,
		mongoClient:      mongoClient,
		jwtService:       jwtService,
		languageDetector: languageDetector,
	}, nil
}

// HealthCheck checks the health of all services
func (s *Server) HealthCheck(ctx context.Context) error {
	healthLogger := logger.NewLogger("health_check").StartWithMsg("Running health checks")

	// Check Ollama
	if err := s.ollamaClient.HealthCheck(ctx); err != nil {
		healthLogger.Error().Err(err).Msg("Ollama health check failed")
		healthLogger.EndWithError(err)
		return fmt.Errorf("ollama health check failed: %w", err)
	}
	healthLogger.Info().Msg("Ollama health check passed")

	// Check OpenSearch
	if err := s.opensearchClient.HealthCheck(ctx); err != nil {
		healthLogger.Error().Err(err).Msg("OpenSearch health check failed")
		healthLogger.EndWithError(err)
		return fmt.Errorf("opensearch health check failed: %w", err)
	}
	healthLogger.Info().Msg("OpenSearch health check passed")

	// Check Qdrant
	if err := s.qdrantClient.HealthCheck(ctx); err != nil {
		healthLogger.Error().Err(err).Msg("Qdrant health check failed")
		healthLogger.EndWithError(err)
		return fmt.Errorf("qdrant health check failed: %w", err)
	}
	healthLogger.Info().Msg("Qdrant health check passed")

	// Check MongoDB
	if err := s.mongoClient.Connect(ctx); err != nil {
		healthLogger.Error().Err(err).Msg("MongoDB health check failed")
		healthLogger.EndWithError(err)
		return fmt.Errorf("mongodb health check failed: %w", err)
	}
	healthLogger.Info().Msg("MongoDB health check passed")

	healthLogger.EndWithMsg("All health checks passed")
	return nil
}

// GetArticle retrieves a specific article by ID
func (s *Server) GetArticle(ctx context.Context, id string) (*opensearch.Article, error) {
	articleLogger := logger.NewLogger("get_article").StartWithMsg("Retrieving article")
	articleLogger.Info().Str("article_id", id).Msg("Fetching article from OpenSearch")

	article, err := s.opensearchClient.GetArticle(ctx, id)
	if err != nil {
		articleLogger.EndWithError(err)
		return nil, err
	}

	articleLogger.Info().Str("article_id", id).Str("title", article.Title).Msg("Article retrieved successfully")
	articleLogger.EndWithMsg("Article retrieval complete")
	return article, nil
}

// GetSupportedLanguages returns the list of supported languages
func (s *Server) GetSupportedLanguages() []string {
	return s.languageDetector.GetSupportedLanguages()
}
