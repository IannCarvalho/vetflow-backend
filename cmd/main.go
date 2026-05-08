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
	"fmt"
	"log"
	"net/http"

	"github.com/vetflow/veterinary-api/config"
	"github.com/vetflow/veterinary-api/db"
	"github.com/vetflow/veterinary-api/routes"
)

func main() {
	// Direct stdout output - should always appear
	fmt.Println("========== VETFLOW APPLICATION STARTING ==========")
	
	// Create a health check endpoint that works before everything else
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"vetflow"}`)
	})
	
	// Load configuration
	log.Println("[MAIN] Starting application...")
	cfg := config.LoadConfig()
	log.Printf("[MAIN] Configuration loaded: %v\n", cfg)

	// Connect to MongoDB
	log.Println("[MAIN] Attempting MongoDB connection...")
	client, err := db.NewMongoConnection(cfg.MongoURI)
	if err != nil {
		log.Fatalf("[MAIN] FATAL: Failed to connect to MongoDB: %v", err)
	}
	log.Println("[MAIN] MongoDB connection successful")
	defer db.CloseConnection(client)

	// Get database instance
	log.Println("[MAIN] Getting database instance...")
	database := client.Database(cfg.MongoDB)

	// Seed database (creates indexes and initial data)
	log.Println("[MAIN] Starting database seed...")
	if err := db.SeedDatabase(database); err != nil {
		log.Printf("[MAIN] WARNING: Database seed encountered an issue: %v\n", err)
	}
	log.Println("[MAIN] Database seed completed")

	// Create router
	log.Println("[MAIN] Creating router...")
	router := routes.NewRouter(database)
	log.Println("[MAIN] Router created successfully")

	// Start server
	addr := ":" + cfg.Port
	log.Printf("[MAIN] Starting server on %s\n", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("[MAIN] FATAL: Server error: %v", err)
	}
}
