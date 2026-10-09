package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// DashboardController aggregates read-only stats scoped to the authenticated vet.
// It pushes each count down into MongoDB (CountDocuments / aggregation $count)
// instead of loading whole collections into memory, computing only the time
// bounds in Go.
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

	// Compute the time bounds the counts need. The consultation window is the
	// whole current calendar month and the appointment window is the current
	// calendar day; both are half-open [start, end) to match the
	// FindByVetIDInRange convention. The vaccine window is [now, now+30d]
	// INCLUSIVE on both ends, preserving the original !Before(now) &&
	// !After(windowEnd) semantics (the vaccine repo uses $lte for this reason).
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	vaccineWindowEnd := now.Add(30 * 24 * time.Hour)

	totalClients, err := c.clientRepo.CountByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar clientes", err.Error(), "DB_QUERY_ERROR")
		return
	}

	totalPets, err := c.petRepo.CountByVetID(ctx, userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar pets", err.Error(), "DB_QUERY_ERROR")
		return
	}

	consultationsThisMonth, err := c.consultationRepo.CountByVetIDInRange(ctx, userID, monthStart, monthEnd)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar consultas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	upcomingVaccinations, err := c.vaccineRepo.CountByVetIDNextDoseInRange(ctx, userID, now, vaccineWindowEnd)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacinas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	pendingInvoices, err := c.invoiceRepo.CountByVetIDAndStatus(ctx, userID, models.PaymentPending)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar faturas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	appointmentsToday, err := c.appointmentRepo.CountByVetIDInRange(ctx, userID, dayStart, dayEnd)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamentos", err.Error(), "DB_QUERY_ERROR")
		return
	}

	stats := dashboardStats{
		TotalClients:           int(totalClients),
		TotalPets:              int(totalPets),
		ConsultationsThisMonth: int(consultationsThisMonth),
		UpcomingVaccinations:   int(upcomingVaccinations),
		PendingInvoices:        int(pendingInvoices),
		AppointmentsToday:      int(appointmentsToday),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}
