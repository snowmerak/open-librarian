package main

import (
	"context"
	"fmt"
	"time"

	"github.com/snowmerak/open-librarian/lib/client/mongo"
	"github.com/snowmerak/open-librarian/lib/util/logger"
)

func main() {
	// Initialize example logger
	exampleLogger := logger.NewLogger("mongo-user-example").StartWithMsg("Starting MongoDB user example")
	defer exampleLogger.EndWithMsg("MongoDB user example completed")

	// MongoDB 연결
	client, err := mongo.New("mongodb://localhost:27017")
	if err != nil {
		exampleLogger.Error().Err(err).Msg("Failed to create MongoDB client")
		exampleLogger.EndWithError(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 연결 테스트
	if err := client.Connect(ctx); err != nil {
		exampleLogger.Error().Err(err).Msg("Failed to connect to MongoDB")
		exampleLogger.EndWithError(err)
		return
	}
	defer client.Disconnect(ctx)

	// 데이터베이스 초기화 (인덱스 생성)
	if err := client.InitializeDatabase(ctx); err != nil {
		exampleLogger.Error().Err(err).Msg("Failed to initialize database")
		exampleLogger.EndWithError(err)
		return
	}

	exampleLogger.Info().Msg("MongoDB connected and initialized successfully")
	fmt.Println("MongoDB connected and initialized successfully!")

	// 사용자 생성 예제
	userReq := mongo.CreateUserRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "securepassword123",
	}

	user, err := client.CreateUser(ctx, userReq)
	if err != nil {
		exampleLogger.Error().Err(err).Msg("Failed to create user")
	} else {
		exampleLogger.Info().
			Str("user_id", user.ID.Hex()).
			Str("email", user.Email).
			Str("username", user.Username).
			Msg("User created successfully")
		fmt.Printf("User created successfully: ID=%s, Email=%s, Username=%s\n",
			user.ID.Hex(), user.Email, user.Username)
	}

	// 사용자 인증 예제
	credentials := mongo.UserCredentials{
		Email:    "test@example.com",
		Password: "securepassword123",
	}

	authenticatedUser, err := client.AuthenticateUser(ctx, credentials)
	if err != nil {
		exampleLogger.Error().Err(err).Msg("Authentication failed")
	} else {
		exampleLogger.Info().Str("username", authenticatedUser.Username).Msg("Authentication successful")
		fmt.Printf("Authentication successful: %s\n", authenticatedUser.Username)
	}

	// 사용자명으로 조회 예제
	foundUser, err := client.GetUserByUsername(ctx, "testuser")
	if err != nil {
		exampleLogger.Error().Err(err).Msg("Failed to find user")
	} else {
		exampleLogger.Info().
			Str("username", foundUser.Username).
			Str("email", foundUser.Email).
			Msg("User found successfully")
		fmt.Printf("Found user: %s (%s)\n", foundUser.Username, foundUser.Email)
	}
}
