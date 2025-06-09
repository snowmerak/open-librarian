package mongo

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/argon2"
)

// User represents a user in the system
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string        `bson:"email" json:"email"`
	Username     string        `bson:"username" json:"username"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	Salt         string        `bson:"salt" json:"-"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// UserCredentials represents login credentials
type UserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateUserRequest represents user creation request
type CreateUserRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Argon2 parameters
const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

// HashPassword hashes a password using Argon2
func HashPassword(password string) (string, string, error) {
	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Encode to base64
	encodedHash := base64.StdEncoding.EncodeToString(hash)
	encodedSalt := base64.StdEncoding.EncodeToString(salt)

	return encodedHash, encodedSalt, nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, encodedHash, encodedSalt string) (bool, error) {
	// Decode the hash and salt
	hash, err := base64.StdEncoding.DecodeString(encodedHash)
	if err != nil {
		return false, err
	}

	salt, err := base64.StdEncoding.DecodeString(encodedSalt)
	if err != nil {
		return false, err
	}

	// Hash the password with the same salt
	otherHash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Compare hashes
	if len(hash) != len(otherHash) {
		return false, nil
	}

	for i := 0; i < len(hash); i++ {
		if hash[i] != otherHash[i] {
			return false, nil
		}
	}

	return true, nil
}

// CreateUser creates a new user in the database
func (c *Client) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	collection := c.client.Database("open_librarian").Collection("users")

	// Check if user already exists
	var existingUser User
	err := collection.FindOne(ctx, bson.M{"$or": []bson.M{
		{"email": req.Email},
		{"username": req.Username},
	}}).Decode(&existingUser)

	if err == nil {
		return nil, errors.New("user with this email or username already exists")
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	// Hash password
	passwordHash, salt, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		Salt:         salt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	user.ID = result.InsertedID.(bson.ObjectID)
	return &user, nil
}

// AuthenticateUser authenticates a user with email and password
func (c *Client) AuthenticateUser(ctx context.Context, credentials UserCredentials) (*User, error) {
	collection := c.client.Database("open_librarian").Collection("users")

	var user User
	err := collection.FindOne(ctx, bson.M{"email": credentials.Email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("invalid email or password")
		}
		return nil, err
	}

	// Verify password
	isValid, err := VerifyPassword(credentials.Password, user.PasswordHash, user.Salt)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, errors.New("invalid email or password")
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (c *Client) GetUserByID(ctx context.Context, id bson.ObjectID) (*User, error) {
	collection := c.client.Database("open_librarian").Collection("users")

	var user User
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (c *Client) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	collection := c.client.Database("open_librarian").Collection("users")

	var user User
	err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (c *Client) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	collection := c.client.Database("open_librarian").Collection("users")

	var user User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// UpdateUser updates user information (except password)
func (c *Client) UpdateUser(ctx context.Context, id bson.ObjectID, updates bson.M) error {
	collection := c.client.Database("open_librarian").Collection("users")

	// Add updated_at field
	updates["updated_at"] = time.Now()

	// Remove sensitive fields that shouldn't be updated directly
	delete(updates, "password_hash")
	delete(updates, "salt")
	delete(updates, "_id")

	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// ChangePassword changes a user's password
func (c *Client) ChangePassword(ctx context.Context, id bson.ObjectID, oldPassword, newPassword string) error {
	collection := c.client.Database("open_librarian").Collection("users")

	// Get user to verify old password
	user, err := c.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// Verify old password
	isValid, err := VerifyPassword(oldPassword, user.PasswordHash, user.Salt)
	if err != nil {
		return err
	}

	if !isValid {
		return errors.New("invalid old password")
	}

	// Hash new password
	newPasswordHash, newSalt, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"password_hash": newPasswordHash,
			"salt":          newSalt,
			"updated_at":    time.Now(),
		},
	})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// DeleteUser deletes a user from the database
func (c *Client) DeleteUser(ctx context.Context, id bson.ObjectID) error {
	collection := c.client.Database("open_librarian").Collection("users")

	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}

// CreateUserIndexes creates necessary indexes for the users collection
func (c *Client) CreateUserIndexes(ctx context.Context) error {
	collection := c.client.Database("open_librarian").Collection("users")

	// Create unique index on email
	emailIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	// Create unique index on username
	usernameIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{emailIndex, usernameIndex})
	return err
}
