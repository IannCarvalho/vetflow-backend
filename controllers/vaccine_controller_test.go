package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/internal/ctxkeys"
	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// withUserID layers the authenticated user's ID onto a request context under the
// typed ctxkeys.UserID key, mirroring routes.AuthMiddleware.
func withUserID(req *http.Request, userID primitive.ObjectID) *http.Request {
	ctx := context.WithValue(req.Context(), ctxkeys.UserID, userID.Hex())
	return req.WithContext(ctx)
}

// withIDParam attaches a chi route context carrying the {id} path param, then
// layers the user ID on the same context.
func withIDParam(req *http.Request, id, userID primitive.ObjectID) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.Hex())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, ctxkeys.UserID, userID.Hex())
	return req.WithContext(ctx)
}

func mustJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	return bytes.NewBuffer(data)
}

func TestVaccineCreateSetsVetScope(t *testing.T) {
	repo := newFakeVaccineRepo()
	ctrl := NewVaccineController(repo)

	vetID := primitive.NewObjectID()
	body := models.Vaccine{PetID: primitive.NewObjectID(), VaccineName: "Rabies"}

	req := httptest.NewRequest(http.MethodPost, "/vaccines", mustJSON(t, body))
	req = withUserID(req, vetID)
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	if len(repo.items) != 1 {
		t.Fatalf("expected 1 stored vaccine, got %d", len(repo.items))
	}
	for _, v := range repo.items {
		if v.VetID != vetID {
			t.Errorf("stored vet_id = %s, want %s", v.VetID.Hex(), vetID.Hex())
		}
	}
}

func TestVaccineGetAllIsolatesByVet(t *testing.T) {
	repo := newFakeVaccineRepo()
	ctrl := NewVaccineController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "A1"})
	_ = repo.Create(context.Background(), &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "A2"})
	_ = repo.Create(context.Background(), &models.Vaccine{VetID: vetB, PetID: primitive.NewObjectID(), VaccineName: "B1"})

	req := httptest.NewRequest(http.MethodGet, "/vaccines", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []models.Vaccine
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 vaccines for vetA, got %d", len(got))
	}
	for _, v := range got {
		if v.VetID != vetA {
			t.Errorf("got vaccine owned by %s, want %s", v.VetID.Hex(), vetA.Hex())
		}
	}
}

func TestVaccineUpdateOtherVetForbidden(t *testing.T) {
	repo := newFakeVaccineRepo()
	ctrl := NewVaccineController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "Original"}
	_ = repo.Create(context.Background(), original)

	body := models.Vaccine{PetID: original.PetID, VaccineName: "Hijacked"}
	req := httptest.NewRequest(http.MethodPut, "/vaccines/"+original.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}

	stored := repo.items[original.ID]
	if stored.VaccineName != "Original" {
		t.Errorf("vaccine was mutated: name = %q, want %q", stored.VaccineName, "Original")
	}
	if stored.VetID != vetA {
		t.Errorf("vet_id was mutated: %s, want %s", stored.VetID.Hex(), vetA.Hex())
	}
}

func TestVaccineDeleteOtherVetForbidden(t *testing.T) {
	repo := newFakeVaccineRepo()
	ctrl := NewVaccineController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "Original"}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/vaccines/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}

	if _, ok := repo.items[original.ID]; !ok {
		t.Error("vaccine was deleted despite ownership mismatch")
	}
}

func TestVaccineGetByIDOtherVetForbidden(t *testing.T) {
	repo := newFakeVaccineRepo()
	ctrl := NewVaccineController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Vaccine{VetID: vetA, PetID: primitive.NewObjectID(), VaccineName: "Original"}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodGet, "/vaccines/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestVaccineGetByIDMissingNotFound(t *testing.T) {
	repo := newFakeVaccineRepo()
	ctrl := NewVaccineController(repo)

	vetA := primitive.NewObjectID()
	missing := primitive.NewObjectID()

	req := httptest.NewRequest(http.MethodGet, "/vaccines/"+missing.Hex(), nil)
	req = withIDParam(req, missing, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestVaccineUnauthenticated(t *testing.T) {
	repo := newFakeVaccineRepo()
	ctrl := NewVaccineController(repo)
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
				body := models.Vaccine{PetID: primitive.NewObjectID(), VaccineName: "X"}
				req = httptest.NewRequest(tt.method, "/vaccines", mustJSON(t, body))
			} else {
				req = httptest.NewRequest(tt.method, "/vaccines", nil)
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
