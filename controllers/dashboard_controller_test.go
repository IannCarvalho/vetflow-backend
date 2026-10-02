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

	// Appointments: 2 today, 1 tomorrow.
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: now})
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: now.Add(2 * time.Hour)})
	_ = appointmentRepo.Create(ctx, &models.Appointment{VetID: vetA, StartTime: now.AddDate(0, 0, 1)})

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

func TestDashboardStatsUnauthenticated(t *testing.T) {
	ctrl, _, _, _, _, _, _ := newDashboardFixture()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	rec := httptest.NewRecorder()

	ctrl.Stats(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
