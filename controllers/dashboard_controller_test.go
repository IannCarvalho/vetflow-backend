package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newDashboardFixture wires a DashboardController over map-backed fakes and
// returns both so tests can seed data and assert the aggregated response.
func newDashboardFixture() (*DashboardController, *fakeClientRepo, *fakePetRepo, *fakeConsultationRepo, *fakeVaccineRepo, *fakeInvoiceRepo, *fakeAppointmentRepo) {
	clientRepo := newFakeClientRepo()
	petRepo := newFakePetRepo(clientRepo)
	consultationRepo := newFakeConsultationRepo()
	vaccineRepo := newFakeVaccineRepo()
	invoiceRepo := newFakeInvoiceRepo()
	appointmentRepo := newFakeAppointmentRepo()

	ctrl := NewDashboardController(clientRepo, petRepo, consultationRepo, vaccineRepo, invoiceRepo, appointmentRepo)
	return ctrl, clientRepo, petRepo, consultationRepo, vaccineRepo, invoiceRepo, appointmentRepo
}

func TestDashboardStatsCountsSeededData(t *testing.T) {
	ctrl, clientRepo, petRepo, consultationRepo, vaccineRepo, invoiceRepo, appointmentRepo := newDashboardFixture()
	ctx := context.Background()
	now := time.Now()

	vetA := primitive.NewObjectID()

	// Two clients for vetA.
	clientA1 := &models.Client{VetID: vetA, FullName: "Client A1"}
	clientA2 := &models.Client{VetID: vetA, FullName: "Client A2"}
	_ = clientRepo.Create(ctx, clientA1)
	_ = clientRepo.Create(ctx, clientA2)

	// Three pets owned by vetA's clients.
	_ = petRepo.Create(ctx, &models.Pet{OwnerID: clientA1.ID, Name: "Rex"})
	_ = petRepo.Create(ctx, &models.Pet{OwnerID: clientA1.ID, Name: "Bela"})
	_ = petRepo.Create(ctx, &models.Pet{OwnerID: clientA2.ID, Name: "Thor"})

	// Consultations: 2 this month, 1 well in the past.
	_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: now})
	_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: now.AddDate(0, 0, -1)})
	_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: now.AddDate(0, -2, 0)})

	// Vaccines: next dose in 10 days (counts), in 40 days (out of window), yesterday (past, out).
	_ = vaccineRepo.Create(ctx, &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "V1", NextDoseDate: now.AddDate(0, 0, 10)})
	_ = vaccineRepo.Create(ctx, &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "V2", NextDoseDate: now.AddDate(0, 0, 40)})
	_ = vaccineRepo.Create(ctx, &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "V3", NextDoseDate: now.AddDate(0, 0, -1)})

	// Invoices: 2 pending, 1 paid.
	_ = invoiceRepo.Create(ctx, &models.Invoice{VetID: vetA, Status: models.PaymentPending})
	_ = invoiceRepo.Create(ctx, &models.Invoice{VetID: vetA, Status: models.PaymentPending})
	_ = invoiceRepo.Create(ctx, &models.Invoice{VetID: vetA, Status: models.PaymentPaid})

	// Appointments: 2 today, 1 tomorrow. Anchor to fixed intra-day times so the
	// "today" window stays stable regardless of what time the suite runs (a
	// now.Add(2h) seed rolls past midnight when run late in the day, which made
	// this test flaky). The dashboard buckets by calendar day via sameDay().
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayMorning := startOfToday.Add(9 * time.Hour)
	todayAfternoon := startOfToday.Add(15 * time.Hour)
	tomorrowNoon := startOfToday.AddDate(0, 0, 1).Add(12 * time.Hour)
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: todayMorning})
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: todayAfternoon})
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: tomorrowNoon})

	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.Stats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got dashboardStats
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	want := dashboardStats{
		TotalClients:           2,
		TotalPets:              3,
		ConsultationsThisMonth: 2,
		UpcomingVaccinations:   1,
		PendingInvoices:        2,
		AppointmentsToday:      2,
	}
	if got != want {
		t.Fatalf("stats = %+v, want %+v", got, want)
	}
}

func TestDashboardStatsIsolatesByVet(t *testing.T) {
	ctrl, clientRepo, petRepo, consultationRepo, vaccineRepo, invoiceRepo, appointmentRepo := newDashboardFixture()
	ctx := context.Background()
	now := time.Now()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()

	// vetA data.
	clientA := &models.Client{VetID: vetA, FullName: "Owned"}
	_ = clientRepo.Create(ctx, clientA)
	_ = petRepo.Create(ctx, &models.Pet{OwnerID: clientA.ID, Name: "Rex"})
	_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: now})
	_ = vaccineRepo.Create(ctx, &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "V", NextDoseDate: now.AddDate(0, 0, 5)})
	_ = invoiceRepo.Create(ctx, &models.Invoice{VetID: vetA, Status: models.PaymentPending})
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: now})

	// vetB data that must NOT be counted for vetA.
	clientB := &models.Client{VetID: vetB, FullName: "Other"}
	_ = clientRepo.Create(ctx, clientB)
	_ = petRepo.Create(ctx, &models.Pet{OwnerID: clientB.ID, Name: "NotYours"})
	_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetB, Date: now})
	_ = vaccineRepo.Create(ctx, &models.Vaccine{VetID: vetB, PetID: primitive.NewObjectID(), VaccineName: "VB", NextDoseDate: now.AddDate(0, 0, 5)})
	_ = invoiceRepo.Create(ctx, &models.Invoice{VetID: vetB, Status: models.PaymentPending})
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetB, StartTime: now})

	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.Stats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got dashboardStats
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	want := dashboardStats{
		TotalClients:           1,
		TotalPets:              1,
		ConsultationsThisMonth: 1,
		UpcomingVaccinations:   1,
		PendingInvoices:        1,
		AppointmentsToday:      1,
	}
	if got != want {
		t.Fatalf("stats = %+v, want %+v (vetB data leaked into vetA counts)", got, want)
	}
}

// TestDashboardStatsWindowBoundaryInclusiveness locks in the half-open vs
// inclusive window contract that the mongo filters implement and the fakes
// mirror: consultation and appointment windows are half-open [start, end)
// ($gte/$lt), while the vaccine window is inclusive [start, end] ($gte/$lte).
//
// The existing dashboard tests only seed interior points, so the $lt-vs-$lte
// distinction on the upper bound is never asserted. This test seeds documents
// exactly on the window boundaries so a future regression (e.g. flipping the
// vaccine query to $lt, or a consultation/appointment query to $lte) fails.
func TestDashboardStatsWindowBoundaryInclusiveness(t *testing.T) {
	// Part 1: drive the controller end-to-end for the two half-open windows.
	// The controller derives monthStart/monthEnd and dayStart/dayEnd purely
	// from the calendar, so we can reproduce the exact same bounds here and the
	// assertion stays deterministic regardless of the current date.
	t.Run("controller half-open consultation and appointment windows", func(t *testing.T) {
		ctrl, _, _, consultationRepo, _, _, appointmentRepo := newDashboardFixture()
		ctx := context.Background()
		now := time.Now()

		vetA := primitive.NewObjectID()

		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		monthEnd := monthStart.AddDate(0, 1, 0)
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		dayEnd := dayStart.AddDate(0, 0, 1)

		// Consultations: one exactly at monthStart (included, $gte) and one
		// exactly at monthEnd (excluded, $lt). Expected count = 1.
		_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: monthStart})
		_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: monthEnd})

		// Appointments: one exactly at dayStart (included) and one exactly at
		// dayEnd (excluded). Expected count = 1.
		_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: dayStart})
		_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: dayEnd})

		req := withUserID(httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil), vetA)
		rec := httptest.NewRecorder()
		ctrl.Stats(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
		}

		var got dashboardStats
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if got.ConsultationsThisMonth != 1 {
			t.Errorf("consultations_this_month = %d, want 1 (monthStart must count, monthEnd must not)", got.ConsultationsThisMonth)
		}
		if got.AppointmentsToday != 1 {
			t.Errorf("appointments_today = %d, want 1 (dayStart must count, dayEnd must not)", got.AppointmentsToday)
		}
	})

	// Part 2: assert the inclusiveness contract directly against the fake count
	// methods with explicit start/end bounds. This is deterministic for the
	// vaccine upper bound too, which the controller otherwise computes from an
	// internal time.Now() we cannot pin without adding a seam to production
	// code. The fakes are the exact semantics the controller depends on.
	t.Run("fake count methods inclusive vs exclusive end", func(t *testing.T) {
		ctx := context.Background()
		vetA := primitive.NewObjectID()

		start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)

		// Vaccine: inclusive [start, end]. Seed exactly at both bounds; both count.
		vaccineRepo := newFakeVaccineRepo()
		_ = vaccineRepo.Create(ctx, &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "atStart", NextDoseDate: start})
		_ = vaccineRepo.Create(ctx, &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "atEnd", NextDoseDate: end})
		vaccineCount, err := vaccineRepo.CountByVetIDNextDoseInRange(ctx, vetA, start, end)
		if err != nil {
			t.Fatalf("vaccine count: %v", err)
		}
		if vaccineCount != 2 {
			t.Errorf("vaccine count = %d, want 2 (both start and end are inclusive, $gte/$lte)", vaccineCount)
		}

		// Consultation: half-open [start, end). start counts, end does not.
		consultationRepo := newFakeConsultationRepo()
		_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: start})
		_ = consultationRepo.Create(ctx, &models.Consultation{VetID: vetA, Date: end})
		consultationCount, err := consultationRepo.CountByVetIDInRange(ctx, vetA, start, end)
		if err != nil {
			t.Fatalf("consultation count: %v", err)
		}
		if consultationCount != 1 {
			t.Errorf("consultation count = %d, want 1 (start inclusive, end exclusive, $gte/$lt)", consultationCount)
		}

		// Appointment: half-open [start, end). start counts, end does not.
		appointmentRepo := newFakeAppointmentRepo()
		_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: start})
		_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: end})
		appointmentCount, err := appointmentRepo.CountByVetIDInRange(ctx, vetA, start, end)
		if err != nil {
			t.Fatalf("appointment count: %v", err)
		}
		if appointmentCount != 1 {
			t.Errorf("appointment count = %d, want 1 (start inclusive, end exclusive, $gte/$lt)", appointmentCount)
		}
	})
}

func TestDashboardStatsUnauthenticated(t *testing.T) {
	ctrl, _, _, _, _, _, _ := newDashboardFixture()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	rec := httptest.NewRecorder()

	ctrl.Stats(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
