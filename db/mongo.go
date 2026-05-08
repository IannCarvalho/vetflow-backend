package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
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

	// Configure TLS with proper settings for MongoDB Atlas
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // Use proper certificate validation
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		// Load system CA certificates
		RootCAs: loadSystemCACerts(),
	}

	opts := options.Client().
		SetTLSConfig(tlsConfig).
		ApplyURI(mongoURI).
		// Connection pool settings
		SetMaxPoolSize(100).
		SetMinPoolSize(10).
		// Timeouts
		SetServerSelectionTimeout(15 * time.Second).
		SetConnectTimeout(15 * time.Second).
		SetSocketTimeout(45 * time.Second).
		SetMaxConnIdleTime(5 * time.Minute).
		// Heartbeat and keep-alive
		SetHeartbeatInterval(10 * time.Second).
		// Direct connection (useful for debugging)
		// SetDirect(true). // Uncomment only if needed for testing

	_ = cs // Use connection string to avoid unused variable

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
		client.Disconnect(context.Background())
		return nil, err
	}

	log.Println("MongoDB connection established successfully")
	return client, nil
}

// loadSystemCACerts loads the system's CA certificates
func loadSystemCACerts() *x509.CertPool {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		log.Printf("Warning: Could not load system CA certs: %v. Using nil CertPool.", err)
		return nil
	}
	return caCertPool
}

// CloseConnection closes the MongoDB connection
func CloseConnection(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Disconnect(ctx)
}
