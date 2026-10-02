package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// validConsultationBody builds a consultation payload that passes
// validateConsultation (requires a pet and a non-empty main complaint).
func validConsultationBody(petID primitive.ObjectID) models.Consultation {
	c := models.Consultation{PetID: petID}
	c.Anamnesis.MainComplaint = "Vômito há 2 dias"
	return c
}

func TestConsultationCreateSetsVetScope(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)

	vetID := primitive.NewObjectID()
	body := validConsultationBody(primitive.NewObjectID())

	req := httptest.NewRequest(http.MethodPost, "/consultations", mustJSON(t, body))
	req = withUserID(req, vetID)
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 stored consultation, got %d", len(repo.items))
	}
	for _, c := range repo.items {
		if c.VetID != vetID {
			t.Errorf("stored vet_id = %s, want %s", c.VetID.Hex(), vetID.Hex())
		}
	}
}

func TestConsultationGetAllIsolatesByVet(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.Consultation{VetID: vetA, PetID: primitive.NewObjectID()})
	_ = repo.Create(context.Background(), &models.Consultation{VetID: vetA, PetID: primitive.NewObjectID()})
	_ = repo.Create(context.Background(), &models.Consultation{VetID: vetB, PetID: primitive.NewObjectID()})

	req := httptest.NewRequest(http.MethodGet, "/consultations", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []models.Consultation
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 consultations for vetA, got %d", len(got))
	}
	for _, c := range got {
		if c.VetID != vetA {
			t.Errorf("got consultation owned by %s, want %s", c.VetID.Hex(), vetA.Hex())
		}
	}
}

func TestConsultationGetByIDOtherVetForbidden(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Consultation{VetID: vetA, PetID: primitive.NewObjectID()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodGet, "/consultations/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestConsultationUpdateOtherVetForbidden(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Consultation{VetID: vetA, PetID: primitive.NewObjectID()}
	original.Anamnesis.MainComplaint = "Original"
	_ = repo.Create(context.Background(), original)

	body := validConsultationBody(original.PetID)
	body.Anamnesis.MainComplaint = "Hijacked"
	req := httptest.NewRequest(http.MethodPut, "/consultations/"+original.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	stored := repo.items[original.ID]
	if stored.Anamnesis.MainComplaint != "Original" {
		t.Errorf("consultation was mutated: main_complaint = %q, want %q", stored.Anamnesis.MainComplaint, "Original")
	}
	if stored.VetID != vetA {
		t.Errorf("vet_id was mutated: %s, want %s", stored.VetID.Hex(), vetA.Hex())
	}
}

func TestConsultationDeleteOtherVetForbidden(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Consultation{VetID: vetA, PetID: primitive.NewObjectID()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/consultations/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if _, ok := repo.items[original.ID]; !ok {
		t.Error("consultation was deleted despite ownership mismatch")
	}
}

func TestConsultationDeleteOwnSucceeds(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)

	vetA := primitive.NewObjectID()
	original := &models.Consultation{VetID: vetA, PetID: primitive.NewObjectID()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/consultations/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetA)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if _, ok := repo.items[original.ID]; ok {
		t.Error("consultation was not deleted for its owner")
	}
}

func TestConsultationDeleteMissingNotFound(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)

	vetA := primitive.NewObjectID()
	missing := primitive.NewObjectID()

	req := httptest.NewRequest(http.MethodDelete, "/consultations/"+missing.Hex(), nil)
	req = withIDParam(req, missing, vetA)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestConsultationUnauthenticated(t *testing.T) {
	repo := newFakeConsultationRepo()
	ctrl := NewConsultationController(repo)
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
				body := validConsultationBody(primitive.NewObjectID())
				req = httptest.NewRequest(tt.method, "/consultations", mustJSON(t, body))
			} else {
				req = httptest.NewRequest(tt.method, "/consultations", nil)
			}

			// Attach the id path param but NO userID.
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
