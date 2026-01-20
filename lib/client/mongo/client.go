package mongo

import (
	"context"

	"github.com/snowmerak/open-librarian/lib/util/logger"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client struct {
	client *mongo.Client
}

// New creates a new MongoDB client
func New(uri string) (*Client, error) {
	mongoLogger := logger.NewLogger("mongo_client").StartWithMsg("Creating MongoDB client")

	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		mongoLogger.EndWithError(err)
		return nil, err
	}

	mongoLogger.Info().Str("uri", uri).Msg("MongoDB client created successfully")
	mongoLogger.EndWithMsg("MongoDB client creation complete")

	return &Client{
		client: client,
	}, nil
}

// Connect establishes connection to MongoDB
func (c *Client) Connect(ctx context.Context) error {
	connLogger := logger.NewLogger("mongo_connect").StartWithMsg("Establishing MongoDB connection")

	err := c.client.Ping(ctx, nil)
	if err != nil {
		connLogger.EndWithError(err)
		return err
	}

	connLogger.EndWithMsg("MongoDB connection established")
	return nil
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
	initLogger := logger.NewLogger("mongo_init_db").StartWithMsg("Initializing MongoDB database")

	// Create user indexes
	err := c.CreateUserIndexes(ctx)
	if err != nil {
		initLogger.EndWithError(err)
		return err
	}

	// Create chat indexes
	err = c.CreateChatIndexes(ctx)
	if err != nil {
		initLogger.EndWithError(err)
		return err
	}

	initLogger.EndWithMsg("MongoDB database initialization complete")
	return nil
}
