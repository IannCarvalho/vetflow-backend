package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// DashboardController aggregates read-only stats scoped to the authenticated vet.
// It reuses the existing per-resource FindByVetID methods and computes the
// counts in memory, so it does not require any new repository interface methods.
type DashboardController struct {
	clientRepo       repositories.ClientRepository
	petRepo          repositories.PetRepository
	consultationRepo repositories.ConsultationRepository
	vaccineRepo      repositories.VaccineRepository
	invoiceRepo      repositories.InvoiceRepository
	appointmentRepo  repositories.AppointmentRepository
}

// NewDashboardController creates a new dashboard controller with dependency injection.
func NewDashboardController(
	clientRepo repositories.ClientRepository,
	petRepo repositories.PetRepository,
	consultationRepo repositories.ConsultationRepository,
	vaccineRepo repositories.VaccineRepository,
	invoiceRepo repositories.InvoiceRepository,
	appointmentRepo repositories.AppointmentRepository,
) *DashboardController {
	return &DashboardController{
		clientRepo:       clientRepo,
		petRepo:          petRepo,
		consultationRepo: consultationRepo,
		vaccineRepo:      vaccineRepo,
		invoiceRepo:      invoiceRepo,
		appointmentRepo:  appointmentRepo,
	}
}

// dashboardStats is the aggregated response payload for GET /dashboard/stats.
type dashboardStats struct {
	TotalClients           int `json:"total_clients"`
	TotalPets              int `json:"total_pets"`
	ConsultationsThisMonth int `json:"consultations_this_month"`
	UpcomingVaccinations   int `json:"upcoming_vaccinations"`
	PendingInvoices        int `json:"pending_invoices"`
	AppointmentsToday      int `json:"appointments_today"`
}

// Stats godoc
// @Summary Get dashboard stats for the authenticated veterinarian
// @Description Aggregated, vet-scoped counts for the dashboard home screen
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /dashboard/stats [get]
func (c *DashboardController) Stats(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	ctx := r.Context()

	clients, err := c.clientRepo.FindByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar clientes", err.Error(), "DB_QUERY_ERROR")
		return
	}

	pets, err := c.petRepo.FindByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar pets", err.Error(), "DB_QUERY_ERROR")
		return
	}

	consultations, err := c.consultationRepo.FindByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar consultas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	vaccines, err := c.vaccineRepo.FindByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacinas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	invoices, err := c.invoiceRepo.FindByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar faturas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	appointments, err := c.appointmentRepo.FindByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamentos", err.Error(), "DB_QUERY_ERROR")
		return
	}

	now := time.Now()
	windowEnd := now.Add(30 * 24 * time.Hour)

	stats := dashboardStats{
		TotalClients: len(clients),
		TotalPets:    len(pets),
	}

	for _, consultation := range consultations {
		if consultation.Date.Year() == now.Year() && consultation.Date.Month() == now.Month() {
			stats.ConsultationsThisMonth++
		}
	}

	for _, vaccine := range vaccines {
		next := vaccine.NextDoseDate
		if !next.Before(now) && !next.After(windowEnd) {
			stats.UpcomingVaccinations++
		}
	}

	for _, invoice := range invoices {
		if invoice.Status == models.PaymentPending {
			stats.PendingInvoices++
		}
	}

	for _, appointment := range appointments {
		if sameDay(appointment.StartTime, now) {
			stats.AppointmentsToday++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// sameDay reports whether a and b fall on the same calendar day.
func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
