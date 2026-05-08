package db

import (
	"context"
	"crypto/tls"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

// NewMongoConnection creates a new MongoDB connection
func NewMongoConnection(mongoURI string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse connection string to validate
	cs, err := connstring.ParseAndValidate(mongoURI)
	if err != nil {
		return nil, err
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:        tls.VersionTLS12,
	}

	opts := options.Client().
		SetTLSConfig(tlsConfig).
		ApplyURI(mongoURI).
		SetMaxConnIdleTime(30 * time.Second).
		SetServerSelectionTimeout(30 * time.Second)

	_ = cs // Use connection string to avoid unused variable

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
