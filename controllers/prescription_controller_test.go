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

// validPrescriptionBody builds a prescription payload that passes
// validatePrescription (requires a pet and at least one item with Medication).
func validPrescriptionBody(petID primitive.ObjectID) models.Prescription {
	return models.Prescription{
		PetID: petID,
		Items: []models.PrescriptionItem{{Medication: "Amoxicilina", Dosage: "250mg"}},
	}
}

func TestPrescriptionCreateSetsVetScope(t *testing.T) {
	repo := newFakePrescriptionRepo()
	ctrl := NewPrescriptionController(repo)

	vetID := primitive.NewObjectID()
	body := validPrescriptionBody(primitive.NewObjectID())

	req := httptest.NewRequest(http.MethodPost, "/prescriptions", mustJSON(t, body))
	req = withUserID(req, vetID)
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 stored prescription, got %d", len(repo.items))
	}
	for _, p := range repo.items {
		if p.VetID != vetID {
			t.Errorf("stored vet_id = %s, want %s", p.VetID.Hex(), vetID.Hex())
		}
	}
}

func TestPrescriptionGetAllIsolatesByVet(t *testing.T) {
	repo := newFakePrescriptionRepo()
	ctrl := NewPrescriptionController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.Prescription{VetID: vetA, PetID: primitive.NewObjectID()})
	_ = repo.Create(context.Background(), &models.Prescription{VetID: vetA, PetID: primitive.NewObjectID()})
	_ = repo.Create(context.Background(), &models.Prescription{VetID: vetB, PetID: primitive.NewObjectID()})

	req := httptest.NewRequest(http.MethodGet, "/prescriptions", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []models.Prescription
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 prescriptions for vetA, got %d", len(got))
	}
	for _, p := range got {
		if p.VetID != vetA {
			t.Errorf("got prescription owned by %s, want %s", p.VetID.Hex(), vetA.Hex())
		}
	}
}

func TestPrescriptionGetByIDOtherVetForbidden(t *testing.T) {
	repo := newFakePrescriptionRepo()
	ctrl := NewPrescriptionController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Prescription{VetID: vetA, PetID: primitive.NewObjectID()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodGet, "/prescriptions/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestPrescriptionUpdateOtherVetForbidden(t *testing.T) {
	repo := newFakePrescriptionRepo()
	ctrl := NewPrescriptionController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Prescription{VetID: vetA, PetID: primitive.NewObjectID(), GeneralNotes: "Original"}
	_ = repo.Create(context.Background(), original)

	body := validPrescriptionBody(original.PetID)
	body.GeneralNotes = "Hijacked"
	req := httptest.NewRequest(http.MethodPut, "/prescriptions/"+original.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	stored := repo.items[original.ID]
	if stored.GeneralNotes != "Original" {
		t.Errorf("prescription was mutated: general_notes = %q, want %q", stored.GeneralNotes, "Original")
	}
	if stored.VetID != vetA {
		t.Errorf("vet_id was mutated: %s, want %s", stored.VetID.Hex(), vetA.Hex())
	}
}

func TestPrescriptionDeleteOtherVetForbidden(t *testing.T) {
	repo := newFakePrescriptionRepo()
	ctrl := NewPrescriptionController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Prescription{VetID: vetA, PetID: primitive.NewObjectID()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/prescriptions/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if _, ok := repo.items[original.ID]; !ok {
		t.Error("prescription was deleted despite ownership mismatch")
	}
}

func TestPrescriptionUnauthenticated(t *testing.T) {
	repo := newFakePrescriptionRepo()
	ctrl := NewPrescriptionController(repo)
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
				body := validPrescriptionBody(primitive.NewObjectID())
				req = httptest.NewRequest(tt.method, "/prescriptions", mustJSON(t, body))
			} else {
				req = httptest.NewRequest(tt.method, "/prescriptions", nil)
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
