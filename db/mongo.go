package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoConnection creates a new MongoDB connection
func NewMongoConnection(mongoURI string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a simple approach - let ApplyURI handle SSL/TLS
	opts := options.Client().
		ApplyURI(mongoURI).
		SetServerSelectionTimeout(30 * time.Second).
		SetConnectTimeout(15 * time.Second)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		log.Printf("MongoDB connection error: %v", err)
		return nil, err
	}

	// Verify connection with retry logic
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer pingCancel()

	err = client.Ping(pingCtx, nil)
	if err != nil {
		log.Printf("MongoDB ping failed: %v", err)
		// Best-effort cleanup; the ping error below is the one that matters.
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	log.Println("MongoDB connection established successfully")
	return client, nil
}

// CloseConnection closes the MongoDB connection
func CloseConnection(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Disconnect(ctx)
}
