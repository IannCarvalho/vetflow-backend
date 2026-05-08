// @title VetFlow - Veterinary Management API
// @version 1.0
// @description Complete REST API for managing veterinary clinics, clients, pets, consultations and vaccines.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@vetflow.local

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http

// @externalDocs.description OpenAPI
// @externalDocs.url https://swagger.io

package main

import (
	"log"
	"net/http"

	"github.com/vetflow/veterinary-api/config"
	"github.com/vetflow/veterinary-api/db"
	"github.com/vetflow/veterinary-api/routes"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	log.Println("Configuration loaded:", cfg)

	// Connect to MongoDB
	client, err := db.NewMongoConnection(cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.CloseConnection(client)

	// Get database instance
	database := client.Database(cfg.MongoDB)

	// Seed database (creates indexes and initial data)
	if err := db.SeedDatabase(database); err != nil {
		log.Println("Warning: Database seed encountered an issue:", err)
	}

	// Create router
	router := routes.NewRouter(database)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
