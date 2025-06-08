package qdrant

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

type Client struct {
	client         *qdrant.Client
	collectionName string
}

// VectorSearchResult represents a single search result with score
type VectorSearchResult struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

const (
	DefaultCollectionName = "articles"
	DefaultGrpcPort       = 6334
	VectorDimension       = 768 // Standard embedding dimension
)

// NewClient creates a new Qdrant client using the official gRPC client
func NewClient(host string, port int) (*Client, error) {
	if port == 0 {
		port = DefaultGrpcPort
	}

	// Connect to Qdrant using gRPC
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	return &Client{
		client:         client,
		collectionName: DefaultCollectionName,
	}, nil
}

// Close closes the Qdrant client connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// CreateCollection creates a collection for storing 768-dimension vectors
func (c *Client) CreateCollection(ctx context.Context) error {
	exists, err := c.client.CollectionExists(ctx, c.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil // Collection already exists
	}

	err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: c.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(VectorDimension),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// UpsertPoint inserts or updates a point in the collection
func (c *Client) UpsertPoint(ctx context.Context, pointID string, vector []float64, lang string) error {
	// Convert float64 to float32 for Qdrant
	vector32 := make([]float32, len(vector))
	for i, v := range vector {
		vector32[i] = float32(v)
	}

	// Create minimal payload with only language for filtering
	langValue, err := qdrant.NewValue(lang)
	if err != nil {
		return fmt.Errorf("failed to create language value: %w", err)
	}

	// Also store the original OpenSearch ID in payload for mapping
	idValue, err := qdrant.NewValue(pointID)
	if err != nil {
		return fmt.Errorf("failed to create id value: %w", err)
	}

	payload := map[string]*qdrant.Value{
		"lang":          langValue,
		"opensearch_id": idValue,
	}

	// Convert string ID to numeric ID using hash
	numericID := c.stringToNumericID(pointID)

	point := &qdrant.PointStruct{
		Id:      qdrant.NewIDNum(numericID),
		Vectors: qdrant.NewVectorsDense(vector32),
		Payload: payload,
	}

	_, err = c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: c.collectionName,
		Points:         []*qdrant.PointStruct{point},
	})

	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

// VectorSearch performs vector similarity search and returns IDs with scores
func (c *Client) VectorSearch(ctx context.Context, queryVector []float64, limit uint64, lang string) ([]VectorSearchResult, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector is required")
	}

	log.Printf("=== Qdrant VectorSearch START ===")
	log.Printf("Query vector dimension: %d, Limit: %d, Lang: '%s'", len(queryVector), limit, lang)

	// Convert float64 to float32 for Qdrant
	queryVector32 := make([]float32, len(queryVector))
	for i, v := range queryVector {
		queryVector32[i] = float32(v)
	}

	queryRequest := &qdrant.QueryPoints{
		CollectionName: c.collectionName,
		Query:          qdrant.NewQuery(queryVector32...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true), // Include payload to get original ID
	}

	// Add language filter if specified
	// Temporarily disable language filtering for debugging
	if false && lang != "" {
		queryRequest.Filter = &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("lang", lang),
			},
		}
		log.Printf("Added language filter: %s", lang)
	} else {
		log.Printf("No language filter applied (debugging: language filtering disabled)")
	}

	searchResult, err := c.client.Query(ctx, queryRequest)
	if err != nil {
		log.Printf("Qdrant query failed: %v", err)
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	log.Printf("Qdrant returned %d results", len(searchResult))

	results := make([]VectorSearchResult, 0, len(searchResult))
	for i, hit := range searchResult {
		// Try to get original OpenSearch ID from payload first
		var id string
		if hit.Payload != nil && hit.Payload["opensearch_id"] != nil {
			if stringVal := hit.Payload["opensearch_id"].GetStringValue(); stringVal != "" {
				id = stringVal
			}
		}

		// Fallback to numeric ID if original ID not found
		if id == "" {
			switch idType := hit.Id.PointIdOptions.(type) {
			case *qdrant.PointId_Uuid:
				id = idType.Uuid
			case *qdrant.PointId_Num:
				id = fmt.Sprintf("%d", idType.Num)
			}
		}

		result := VectorSearchResult{
			ID:    id,
			Score: float64(hit.Score),
		}
		results = append(results, result)

		log.Printf("Qdrant result #%d: ID=%s, Score=%.4f", i+1, id, hit.Score)
	}

	log.Printf("=== Qdrant VectorSearch END ===")
	return results, nil
}

// DeletePoint deletes a point from the collection
func (c *Client) DeletePoint(ctx context.Context, pointID string) error {
	// Convert string ID to numeric ID using hash
	numericID := c.stringToNumericID(pointID)

	_, err := c.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: c.collectionName,
		Points:         qdrant.NewPointsSelector(qdrant.NewIDNum(numericID)),
	})

	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}

	return nil
}

// HealthCheck checks if Qdrant is accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// GetCollectionInfo gets information about a collection
func (c *Client) GetCollectionInfo(ctx context.Context) (*qdrant.CollectionInfo, error) {
	info, err := c.client.GetCollectionInfo(ctx, c.collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	return info, nil
}

// stringToNumericID converts a string ID to a numeric ID using hash
func (c *Client) stringToNumericID(id string) uint64 {
	hash := sha256.Sum256([]byte(id))
	return binary.BigEndian.Uint64(hash[:8])
}
