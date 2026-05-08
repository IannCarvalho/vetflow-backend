package db

import (
	"context"
	"crypto/tls"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewMongoConnection creates a new MongoDB connection
func NewMongoConnection(mongoURI string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	opts := options.Client().
		ApplyURI(mongoURI).
		SetTLSConfig(tlsConfig)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Verify connection with longer timeout
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer pingCancel()
	err = client.Ping(pingCtx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CloseConnection closes the MongoDB connection
func CloseConnection(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Disconnect(ctx)
}
