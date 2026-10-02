package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// validAppointmentBody builds an appointment payload that passes
// validateAppointment (requires a pet and a start time).
func validAppointmentBody(petID primitive.ObjectID) models.Appointment {
	return models.Appointment{
		PetID:     petID,
		StartTime: time.Now().Add(24 * time.Hour),
	}
}

func TestAppointmentCreateSetsVetScopeAndDefaultStatus(t *testing.T) {
	repo := newFakeAppointmentRepo()
	ctrl := NewAppointmentController(repo)

	vetID := primitive.NewObjectID()
	body := validAppointmentBody(primitive.NewObjectID())

	req := httptest.NewRequest(http.MethodPost, "/appointments", mustJSON(t, body))
	req = withUserID(req, vetID)
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 stored appointment, got %d", len(repo.items))
	}
	for _, a := range repo.items {
		if a.VetID != vetID {
			t.Errorf("stored vet_id = %s, want %s", a.VetID.Hex(), vetID.Hex())
		}
		if a.Status != models.AppointmentScheduled {
			t.Errorf("default status = %q, want %q", a.Status, models.AppointmentScheduled)
		}
	}
}

func TestAppointmentGetAllIsolatesByVet(t *testing.T) {
	repo := newFakeAppointmentRepo()
	ctrl := NewAppointmentController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.Appointment{VetID: vetA, PetID: primitive.NewObjectID(), StartTime: time.Now()})
	_ = repo.Create(context.Background(), &models.Appointment{VetID: vetA, PetID: primitive.NewObjectID(), StartTime: time.Now()})
	_ = repo.Create(context.Background(), &models.Appointment{VetID: vetB, PetID: primitive.NewObjectID(), StartTime: time.Now()})

	req := httptest.NewRequest(http.MethodGet, "/appointments", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []models.Appointment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 appointments for vetA, got %d", len(got))
	}
	for _, a := range got {
		if a.VetID != vetA {
			t.Errorf("got appointment owned by %s, want %s", a.VetID.Hex(), vetA.Hex())
		}
	}
}

func TestAppointmentGetByIDOtherVetForbidden(t *testing.T) {
	repo := newFakeAppointmentRepo()
	ctrl := NewAppointmentController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Appointment{VetID: vetA, PetID: primitive.NewObjectID(), StartTime: time.Now()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodGet, "/appointments/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAppointmentUpdateOtherVetForbidden(t *testing.T) {
	repo := newFakeAppointmentRepo()
	ctrl := NewAppointmentController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Appointment{VetID: vetA, PetID: primitive.NewObjectID(), StartTime: time.Now(), Title: "Original"}
	_ = repo.Create(context.Background(), original)

	body := validAppointmentBody(original.PetID)
	body.Title = "Hijacked"
	req := httptest.NewRequest(http.MethodPut, "/appointments/"+original.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	stored := repo.items[original.ID]
	if stored.Title != "Original" {
		t.Errorf("appointment was mutated: title = %q, want %q", stored.Title, "Original")
	}
	if stored.VetID != vetA {
		t.Errorf("vet_id was mutated: %s, want %s", stored.VetID.Hex(), vetA.Hex())
	}
}

func TestAppointmentDeleteOtherVetForbidden(t *testing.T) {
	repo := newFakeAppointmentRepo()
	ctrl := NewAppointmentController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Appointment{VetID: vetA, PetID: primitive.NewObjectID(), StartTime: time.Now()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/appointments/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if _, ok := repo.items[original.ID]; !ok {
		t.Error("appointment was deleted despite ownership mismatch")
	}
}

func TestAppointmentUnauthenticated(t *testing.T) {
	repo := newFakeAppointmentRepo()
	ctrl := NewAppointmentController(repo)
	id := primitive.NewObjectID()

	tests := []struct {
		name    string
		call    func(w http.ResponseWriter, r *http.Request)
		method  string
		withID  bool
		hasBody bool
	}{
		{name: "Create", call: ctrl.Create, method: http.MethodPost, hasBody: true},
		{name: "GetAll", call: ctrl.GetAll, method: http.MethodGet},
		{name: "GetByID", call: ctrl.GetByID, method: http.MethodGet, withID: true},
		{name: "Update", call: ctrl.Update, method: http.MethodPut, withID: true, hasBody: true},
		{name: "Delete", call: ctrl.Delete, method: http.MethodDelete, withID: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.hasBody {
				body := validAppointmentBody(primitive.NewObjectID())
				req = httptest.NewRequest(tt.method, "/appointments", mustJSON(t, body))
			} else {
				req = httptest.NewRequest(tt.method, "/appointments", nil)
			}

			if tt.withID {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", id.Hex())
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			rec := httptest.NewRecorder()
			tt.call(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}
