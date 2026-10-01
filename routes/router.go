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
	prescriptionRepo := repositories.NewPrescriptionRepository(db.Collection("prescriptions"))
	catalogRepo := repositories.NewCatalogItemRepository(db.Collection("catalog_items"))
	invoiceRepo := repositories.NewInvoiceRepository(db.Collection("invoices"))
	appointmentRepo := repositories.NewAppointmentRepository(db.Collection("appointments"))
	notificationRepo := repositories.NewNotificationRepository(db)

	// Initialize services
	notificationService := services.NewNotificationService(notificationRepo, consultationRepo, vaccineRepo, petRepo)

	// Initialize controllers
	clientController := controllers.NewClientController(clientRepo, petRepo, consultationRepo, vaccineRepo, prescriptionRepo, invoiceRepo, appointmentRepo)
	petController := controllers.NewPetController(petRepo, clientRepo, consultationRepo, vaccineRepo, prescriptionRepo, appointmentRepo)
	consultationController := controllers.NewConsultationController(consultationRepo)
	vaccineController := controllers.NewVaccineController(vaccineRepo)
	authController := controllers.NewAuthController(userRepo)
	userController := controllers.NewUserController(userRepo)
	socialAuthController := controllers.NewSocialAuthController(userRepo)
	oauthCallbackController := controllers.NewOAuthCallbackController(userRepo)
	notificationController := controllers.NewNotificationController(notificationService)
	prescriptionController := controllers.NewPrescriptionController(prescriptionRepo)
	catalogController := controllers.NewCatalogItemController(catalogRepo)
	invoiceController := controllers.NewInvoiceController(invoiceRepo)
	appointmentController := controllers.NewAppointmentController(appointmentRepo)

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

	// Client routes (fully protected and scoped to the authenticated vet)
	r.Route("/clients", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", clientController.GetAll)
		r.Get("/{id}", clientController.GetByID)
		r.Post("/", clientController.Create)
		r.Put("/{id}", clientController.Update)
		r.Delete("/{id}", clientController.Delete)
	})

	// Pet routes (fully protected and scoped to the authenticated vet)
	r.Route("/pets", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", petController.GetAll)
		r.Get("/{id}", petController.GetByID)
		r.Post("/", petController.Create)
		r.Put("/{id}", petController.Update)
		r.Delete("/{id}", petController.Delete)
	})

	// Consultation routes (fully protected and scoped to the authenticated vet)
	r.Route("/consultations", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", consultationController.GetAll)
		r.Get("/{id}", consultationController.GetByID)
		r.Post("/", consultationController.Create)
		r.Put("/{id}", consultationController.Update)
		r.Delete("/{id}", consultationController.Delete)
	})

	// Vaccine routes (fully protected and scoped to the authenticated vet)
	r.Route("/vaccines", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", vaccineController.GetAll)
		r.Get("/{id}", vaccineController.GetByID)
		r.Post("/", vaccineController.Create)
		r.Put("/{id}", vaccineController.Update)
		r.Delete("/{id}", vaccineController.Delete)
	})

	// Prescription routes (fully protected - receituário is private to the vet)
	r.Route("/prescriptions", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", prescriptionController.GetAll)
		r.Post("/", prescriptionController.Create)
		r.Get("/{id}", prescriptionController.GetByID)
		r.Put("/{id}", prescriptionController.Update)
		r.Delete("/{id}", prescriptionController.Delete)
	})

	// Catalog routes (fully protected - o catálogo é privado do vet)
	r.Route("/catalog", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", catalogController.GetAll)
		r.Post("/", catalogController.Create)
		r.Get("/{id}", catalogController.GetByID)
		r.Put("/{id}", catalogController.Update)
		r.Delete("/{id}", catalogController.Delete)
	})

	// Invoice routes (fully protected - o financeiro é privado do vet)
	r.Route("/invoices", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", invoiceController.GetAll)
		r.Post("/", invoiceController.Create)
		r.Get("/{id}", invoiceController.GetByID)
		r.Put("/{id}", invoiceController.Update)
		r.Delete("/{id}", invoiceController.Delete)
	})

	// Appointment routes (fully protected - the agenda is private to the vet)
	r.Route("/appointments", func(r chi.Router) {
		r.Use(AuthMiddleware)
		r.Get("/", appointmentController.GetAll)
		r.Post("/", appointmentController.Create)
		r.Get("/{id}", appointmentController.GetByID)
		r.Put("/{id}", appointmentController.Update)
		r.Delete("/{id}", appointmentController.Delete)
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
