package api

import (
	"context"
	"fmt"

	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"github.com/snowmerak/open-librarian/lib/client/ollama"
	"github.com/snowmerak/open-librarian/lib/client/opensearch"
	"github.com/snowmerak/open-librarian/lib/client/qdrant"
	"github.com/snowmerak/open-librarian/lib/util/language"
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
	qdrantClient, err := qdrant.NewClient(qdrantHost, qdrantPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	// Initialize Qdrant collection with 768-dimension vectors
	ctx := context.Background()
	if err := qdrantClient.CreateCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize Qdrant collection: %w", err)
	}

	// Create MongoDB client
	mongoClient, err := mongo.New(mongoURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Test MongoDB connection and initialize database
	if err := mongoClient.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := mongoClient.InitializeDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize MongoDB database: %w", err)
	}

	// Create JWT service
	jwtService := mongo.NewJWTService(jwtSecret, "open-librarian")

	return &Server{
		ollamaClient:     ollama.NewClient(ollamaBaseURL),
		opensearchClient: opensearch.NewClient(opensearchBaseURL),
		qdrantClient:     qdrantClient,
		mongoClient:      mongoClient,
		jwtService:       jwtService,
		languageDetector: language.NewDetector(),
	}, nil
}

// HealthCheck checks the health of all services
func (s *Server) HealthCheck(ctx context.Context) error {
	// Check Ollama
	if err := s.ollamaClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("ollama health check failed: %w", err)
	}

	// Check OpenSearch
	if err := s.opensearchClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("opensearch health check failed: %w", err)
	}

	// Check Qdrant
	if err := s.qdrantClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("qdrant health check failed: %w", err)
	}

	// Check MongoDB
	if err := s.mongoClient.Connect(ctx); err != nil {
		return fmt.Errorf("mongodb health check failed: %w", err)
	}

	return nil
}

// GetArticle retrieves a specific article by ID
func (s *Server) GetArticle(ctx context.Context, id string) (*opensearch.Article, error) {
	return s.opensearchClient.GetArticle(ctx, id)
}

// GetSupportedLanguages returns the list of supported languages
func (s *Server) GetSupportedLanguages() []string {
	return s.languageDetector.GetSupportedLanguages()
}
