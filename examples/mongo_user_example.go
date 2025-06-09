package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/snowmerak/open-librarian/lib/client/mongo"
)

func main() {
	// MongoDB 연결
	client, err := mongo.New("mongodb://localhost:27017")
	if err != nil {
		log.Fatal("Failed to create MongoDB client:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 연결 테스트
	if err := client.Connect(ctx); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer client.Disconnect(ctx)

	// 데이터베이스 초기화 (인덱스 생성)
	if err := client.InitializeDatabase(ctx); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	fmt.Println("MongoDB connected and initialized successfully!")

	// 사용자 생성 예제
	userReq := mongo.CreateUserRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "securepassword123",
	}

	user, err := client.CreateUser(ctx, userReq)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
	} else {
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
		log.Printf("Authentication failed: %v", err)
	} else {
		fmt.Printf("Authentication successful: %s\n", authenticatedUser.Username)
	}

	// 사용자명으로 조회 예제
	foundUser, err := client.GetUserByUsername(ctx, "testuser")
	if err != nil {
		log.Printf("Failed to find user: %v", err)
	} else {
		fmt.Printf("Found user: %s (%s)\n", foundUser.Username, foundUser.Email)
	}
}
