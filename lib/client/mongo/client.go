package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client struct {
	client *mongo.Client
}

// New creates a new MongoDB client
func New(uri string) (*Client, error) {
	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, nil
}

// Connect establishes connection to MongoDB
func (c *Client) Connect(ctx context.Context) error {
	return c.client.Ping(ctx, nil)
}

// Disconnect closes the MongoDB connection
func (c *Client) Disconnect(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// GetClient returns the underlying MongoDB client
func (c *Client) GetClient() *mongo.Client {
	return c.client
}

// InitializeDatabase creates necessary indexes for all collections
func (c *Client) InitializeDatabase(ctx context.Context) error {
	// Create user indexes
	return c.CreateUserIndexes(ctx)
}
