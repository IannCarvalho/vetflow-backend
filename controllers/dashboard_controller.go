package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// countError wraps the error returned by a single dashboard count with the
// user-facing message that count is responsible for. The six counts now run
// concurrently and errgroup.Wait() surfaces only the FIRST error, so each count
// must carry its own friendly message with it — otherwise the 500 body would
// lose the specificity it had when the counts ran sequentially (a vaccine
// failure must still read "Falha ao recuperar vacinas", not whichever count the
// scheduler happened to fail first). The handler recovers both the message and
// the original error with errors.As to rebuild the exact same
// writeErrorWithDetails call the sequential code produced.
type countError struct {
	message string
	err     error
}

func (e *countError) Error() string { return e.message + ": " + e.err.Error() }

func (e *countError) Unwrap() error { return e.err }

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

	// Run the six vet-scoped counts concurrently instead of as six serial Mongo
	// round-trips. Each goroutine writes ONLY its own dedicated int64 below, so
	// there is no shared mutable state between them and therefore no data race
	// (-race is clean). gctx is derived from errgroup.WithContext, so the first
	// failure cancels the remaining in-flight queries. Each goroutine wraps its
	// error in a *countError carrying that count's user-facing message, so the
	// single error g.Wait() returns still identifies which count failed.
	var (
		totalClients           int64
		totalPets              int64
		consultationsThisMonth int64
		upcomingVaccinations   int64
		pendingInvoices        int64
		appointmentsToday      int64
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		n, err := c.clientRepo.CountByVetID(gctx, userID)
		if err != nil {
			return &countError{message: "Falha ao recuperar clientes", err: err}
		}
		totalClients = n
		return nil
	})

	g.Go(func() error {
		n, err := c.petRepo.CountByVetID(gctx, userID)
		if err != nil {
			return &countError{message: "Falha ao recuperar pets", err: err}
		}
		totalPets = n
		return nil
	})

	g.Go(func() error {
		n, err := c.consultationRepo.CountByVetIDInRange(gctx, userID, monthStart, monthEnd)
		if err != nil {
			return &countError{message: "Falha ao recuperar consultas", err: err}
		}
		consultationsThisMonth = n
		return nil
	})

	g.Go(func() error {
		n, err := c.vaccineRepo.CountByVetIDNextDoseInRange(gctx, userID, now, vaccineWindowEnd)
		if err != nil {
			return &countError{message: "Falha ao recuperar vacinas", err: err}
		}
		upcomingVaccinations = n
		return nil
	})

	g.Go(func() error {
		n, err := c.invoiceRepo.CountByVetIDAndStatus(gctx, userID, models.PaymentPending)
		if err != nil {
			return &countError{message: "Falha ao recuperar faturas", err: err}
		}
		pendingInvoices = n
		return nil
	})

	g.Go(func() error {
		n, err := c.appointmentRepo.CountByVetIDInRange(gctx, userID, dayStart, dayEnd)
		if err != nil {
			return &countError{message: "Falha ao recuperar agendamentos", err: err}
		}
		appointmentsToday = n
		return nil
	})

	if err := g.Wait(); err != nil {
		// Recover the count-specific message and the underlying error so the 500
		// body is byte-for-byte identical to what the sequential handler emitted
		// for that failing count. Every goroutine wraps in *countError, so the
		// errors.As branch always matches; the fallback keeps the handler safe
		// if a future count returns a bare error.
		var ce *countError
		if errors.As(err, &ce) {
			writeErrorWithDetails(w, http.StatusInternalServerError, ce.message, ce.err.Error(), "DB_QUERY_ERROR")
			return
		}
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar estatísticas", err.Error(), "DB_QUERY_ERROR")
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
