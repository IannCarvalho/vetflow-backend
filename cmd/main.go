// @title VetFlow - Veterinary Management API
// @version 1.0
// @description Complete REST API for managing veterinary clinics, clients, pets, consultations and vaccines.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@vetflow.local

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath /
// @schemes http https

// @externalDocs.description OpenAPI
// @externalDocs.url https://swagger.io

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vetflow/veterinary-api/config"
	"github.com/vetflow/veterinary-api/db"
	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/repositories"
	"github.com/vetflow/veterinary-api/routes"
	"github.com/vetflow/veterinary-api/services"
)

func main() {
	// Direct stdout output - should always appear
	fmt.Println("========== VETFLOW APPLICATION STARTING ==========")

	// Root context cancelled on SIGINT/SIGTERM so shutdown can propagate.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load configuration
	log.Println("[MAIN] Starting application...")
	cfg := config.LoadConfig()
	log.Printf("[MAIN] Configuration loaded: %v\n", cfg)

	// Fail fast on invalid/unsafe configuration (e.g. missing JWT_SECRET in prod).
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[MAIN] FATAL: Invalid configuration: %v", err)
	}

	// Connect to MongoDB
	log.Println("[MAIN] Attempting MongoDB connection...")
	client, err := db.NewMongoConnection(cfg.MongoURI)
	if err != nil {
		log.Fatalf("[MAIN] FATAL: Failed to connect to MongoDB: %v", err)
	}
	log.Println("[MAIN] MongoDB connection successful")
	defer func() {
		if err := db.CloseConnection(client); err != nil {
			log.Printf("[MAIN] WARNING: Failed to close MongoDB connection: %v", err)
		}
	}()

	// Get database instance
	log.Println("[MAIN] Getting database instance...")
	database := client.Database(cfg.MongoDB)

	// Seed database (creates indexes and initial data). NEVER seed in
	// production: the seed inserts fixed demo accounts with a well-known
	// password, which would be a credential backdoor in a live environment.
	if cfg.IsProduction() {
		log.Println("[MAIN] Production environment detected - skipping database seed")
	} else {
		log.Println("[MAIN] Starting database seed...")
		if err := db.SeedDatabase(database); err != nil {
			log.Printf("[MAIN] WARNING: Database seed encountered an issue: %v\n", err)
		}
		log.Println("[MAIN] Database seed completed")
	}

	// Ensure non-unique indexes for vet-scoped and lookup queries. Index
	// creation must never block boot, so failures are only logged.
	idxCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.CreateIndexes(idxCtx, database); err != nil {
		log.Printf("[MAIN] WARNING: index creation issue: %v", err)
	}

	// Create router
	log.Println("[MAIN] Creating router...")
	router := routes.NewRouter(database)
	log.Println("[MAIN] Router created successfully")

	// Start the daily e-mail notification scheduler (non-blocking).
	if cfg.EmailNotificationsEnabled {
		log.Println("[MAIN] Starting notification scheduler...")
		notificationRepo := repositories.NewNotificationRepository(database)
		consultationRepo := repositories.NewConsultationRepository(database.Collection("consultations"))
		vaccineRepo := repositories.NewVaccineRepository(database.Collection("vaccines"))
		petRepo := repositories.NewPetRepository(database.Collection("pets"))
		userRepo := repositories.NewUserRepository(database.Collection("users"))

		notificationService := services.NewNotificationService(notificationRepo, consultationRepo, vaccineRepo, petRepo)
		mailer := helpers.NewMailer(helpers.EmailConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			User:     cfg.SMTPUser,
			Pass:     cfg.SMTPPass,
			From:     cfg.SMTPFrom,
			FromName: cfg.FromName,
		})

		scheduler := services.NewNotificationScheduler(
			notificationService, notificationRepo, userRepo, mailer, 24*time.Hour,
		)
		scheduler.Start(ctx)
	} else {
		log.Println("[MAIN] Email notifications disabled (EMAIL_NOTIFICATIONS_ENABLED=false)")
	}

	// Start server
	addr := ":" + cfg.Port
	// Timeouts bound how long the server waits on slow clients, mitigating
	// slowloris / slow-client attacks that hold connections open indefinitely.
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[MAIN] FATAL: Server error: %v", err)
		}
	}()
	log.Printf("[MAIN] Server listening on %s\n", addr)

	// Block until a shutdown signal cancels the root context.
	<-ctx.Done()
	log.Println("[MAIN] Shutdown signal received, shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[MAIN] Server shutdown error: %v\n", err)
	} else {
		log.Println("[MAIN] Server shut down gracefully")
	}
}
