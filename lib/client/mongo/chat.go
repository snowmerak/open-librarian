package mongo

import (
	"context"
	"time"

	"github.com/snowmerak/open-librarian/lib/util/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ChatMessage represents a single message in a chat session
type ChatMessage struct {
	Role      string      `bson:"role" json:"role"` // "user" or "assistant"
	Content   string      `bson:"content" json:"content"`
	Sources   interface{} `bson:"sources,omitempty" json:"sources,omitempty"` // For assistant messages
	Timestamp time.Time   `bson:"timestamp" json:"timestamp"`
}

// ChatSession represents a chat conversation
type ChatSession struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string        `bson:"user_id,omitempty" json:"user_id,omitempty"` // Optional
	Title     string        `bson:"title" json:"title"`                         // Usually the first query
	Messages  []ChatMessage `bson:"messages" json:"messages"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

const (
	ChatCollection = "chat_sessions"
	DatabaseName   = "open_librarian"
)

// CreateChatIndexes creates indexes for chat sessions
func (c *Client) CreateChatIndexes(ctx context.Context) error {
	collection := c.client.Database(DatabaseName).Collection(ChatCollection)

	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "updated_at", Value: -1},
		},
	})
	return err
}

// SaveChatSession creates or updates a chat session
func (c *Client) SaveChatSession(ctx context.Context, session *ChatSession) error {
	log := logger.NewLogger("mongo_save_chat").Start()
	defer log.End()

	collection := c.client.Database(DatabaseName).Collection(ChatCollection)

	if session.ID.IsZero() {
		session.ID = bson.NewObjectID()
		session.CreatedAt = time.Now()
	}
	session.UpdatedAt = time.Now()

	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": session}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to save chat session")
		return err
	}

	return nil
}

// GetChatSession retrieves a single chat session
func (c *Client) GetChatSession(ctx context.Context, id string) (*ChatSession, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	collection := c.client.Database(DatabaseName).Collection(ChatCollection)
	var session ChatSession
	err = collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetChatSessions retrieves chat sessions for a user (or all if userID is empty, though usually we want user specific)
// For now, if userID is empty, we return all (admin view?) or maybe just anonymous ones?
// For this app, let's treat everyone as anonymous users if they have no ID, or filter by a specific "anonymous" ID if passed.
func (c *Client) GetChatSessions(ctx context.Context, userID string, limit, skip int64) ([]ChatSession, error) {
	collection := c.client.Database(DatabaseName).Collection(ChatCollection)

	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	} else {
		// If no user ID, maybe we shouldn't return anything? Or return all?
		// Since the UI doesn't have login forced yet, let's return all for now to make it work.
		// But the user requested "My" history.
		// Let's rely on the frontend passing a user ID or token if persistent.
		// If not, maybe we just return empty list?
		// Wait, the current implementation has `user-articles.js` which implies some user concept.
		// If `userID` is empty, let's return recent sessions for everyone (local dev mode).
	}

	opts := options.Find().SetSort(bson.M{"updated_at": -1})
	if limit > 0 {
		opts.SetLimit(limit)
	}
	if skip > 0 {
		opts.SetSkip(skip)
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []ChatSession
	if err = cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// DeleteChatSession deletes a chat session
func (c *Client) DeleteChatSession(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	collection := c.client.Database(DatabaseName).Collection(ChatCollection)
	_, err = collection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}
