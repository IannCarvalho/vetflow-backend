package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/vetflow/veterinary-api/controllers"
	_ "github.com/vetflow/veterinary-api/docs"
	"github.com/vetflow/veterinary-api/repositories"
	"github.com/vetflow/veterinary-api/services"
	"go.mongodb.org/mongo-driver/mongo"
)

// NewRouter creates and configures the Chi router
func NewRouter(db *mongo.Database) *chi.Mux {
	r := chi.NewRouter()

	// CORS Middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Initialize repositories
	clientRepo := repositories.NewClientRepository(db.Collection("clients"))
	petRepo := repositories.NewPetRepository(db.Collection("pets"))
	consultationRepo := repositories.NewConsultationRepository(db.Collection("consultations"))
	vaccineRepo := repositories.NewVaccineRepository(db.Collection("vaccines"))
	userRepo := repositories.NewUserRepository(db.Collection("users"))
	notificationRepo := repositories.NewNotificationRepository(db)

	// Initialize services
	notificationService := services.NewNotificationService(notificationRepo, consultationRepo, vaccineRepo, petRepo)

	// Initialize controllers
	clientController := controllers.NewClientController(clientRepo, petRepo, consultationRepo, vaccineRepo)
	petController := controllers.NewPetController(petRepo, clientRepo, consultationRepo, vaccineRepo)
	consultationController := controllers.NewConsultationController(consultationRepo)
	vaccineController := controllers.NewVaccineController(vaccineRepo)
	authController := controllers.NewAuthController(userRepo)
	userController := controllers.NewUserController(userRepo)
	socialAuthController := controllers.NewSocialAuthController(userRepo)
	oauthCallbackController := controllers.NewOAuthCallbackController(userRepo)
	notificationController := controllers.NewNotificationController(notificationService)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authController.Register)
		r.Post("/login", authController.Login)
		r.Post("/social-login", socialAuthController.SocialLogin)
		r.Post("/oauth/callback", oauthCallbackController.OAuthCallback)
		r.Get("/verify/{token}", authController.Verify)
	})

	// User routes (protected)
	r.Route("/users", func(r chi.Router) {
		r.Use(AuthMiddleware) // Add auth middleware for user routes
		r.Put("/profile", userController.UpdateProfile)
		r.Put("/profile-picture", userController.UpdateProfilePicture)
		r.Delete("/profile-picture", userController.RemoveProfilePicture)
		r.Put("/password", userController.ChangePassword)
		r.Put("/password/set", userController.SetPassword)
	})

	// Swagger documentation redirect
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Client routes (GET public, write protected)
	r.Route("/clients", func(r chi.Router) {
		// GET endpoints are public (read-only) - use Public versions
		r.Get("/", clientController.GetAllPublic)
		r.Get("/{id}", clientController.GetByIDPublic)

		// POST/PUT/DELETE endpoints are protected
		r.With(AuthMiddleware).Post("/", clientController.Create)
		r.With(AuthMiddleware).Put("/{id}", clientController.Update)
		r.With(AuthMiddleware).Delete("/{id}", clientController.Delete)
	})

	// Pet routes (GET public, write protected)
	r.Route("/pets", func(r chi.Router) {
		// GET endpoints are public (read-only) - use Public versions
		r.Get("/", petController.GetAllPublic)
		r.Get("/{id}", petController.GetByIDPublic)

		// POST/PUT/DELETE endpoints are protected
		r.With(AuthMiddleware).Post("/", petController.Create)
		r.With(AuthMiddleware).Put("/{id}", petController.Update)
		r.With(AuthMiddleware).Delete("/{id}", petController.Delete)
	})

	// Consultation routes (GET public, write protected)
	r.Route("/consultations", func(r chi.Router) {
		// GET endpoints are public (read-only)
		r.Get("/", consultationController.GetAllPublic)
		r.Get("/{id}", consultationController.GetByID)

		// POST/PUT/DELETE endpoints are protected
		r.With(AuthMiddleware).Post("/", consultationController.Create)
		r.With(AuthMiddleware).Put("/{id}", consultationController.Update)
		r.With(AuthMiddleware).Delete("/{id}", consultationController.Delete)
	})

	// Vaccine routes (GET public, write protected)
	r.Route("/vaccines", func(r chi.Router) {
		// GET endpoints are public (read-only) - use Public versions
		r.Get("/", vaccineController.GetAllPublic)
		r.Get("/{id}", vaccineController.GetByIDPublic)

		// POST/PUT/DELETE endpoints are protected
		r.With(AuthMiddleware).Post("/", vaccineController.Create)
		r.With(AuthMiddleware).Put("/{id}", vaccineController.Update)
		r.With(AuthMiddleware).Delete("/{id}", vaccineController.Delete)
	})

	// Notification routes (protected)
	r.Route("/notifications", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Post("/generate", notificationController.Generate)
		r.Get("/", notificationController.GetAll)
		r.Put("/{id}/read", notificationController.MarkAsRead)
		r.Put("/{id}/dismiss", notificationController.Dismiss)
	})

	return r
}
