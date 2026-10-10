package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// newPrintControllerWithData wires a prescription controller with its print
// context (pet/client/user repos) and seeds a coherent pet -> tutor -> vet
// graph owned by vetID. It returns the controller plus the stored prescription.
func newPrintControllerWithData(vetID primitive.ObjectID) (*PrescriptionController, *models.Prescription) {
	prescRepo := newFakePrescriptionRepo()
	clientRepo := newFakeClientRepo()
	petRepo := newFakePetRepo(clientRepo)
	userRepo := newFakeUserRepo()

	owner := &models.Client{VetID: vetID, FullName: "Maria Tutora"}
	_ = clientRepo.Create(context.Background(), owner)

	pet := &models.Pet{Name: "Rex", Species: "Cachorro", Breed: "Labrador", OwnerID: owner.ID}
	_ = petRepo.Create(context.Background(), pet)

	_ = userRepo.Create(context.Background(), &models.User{ID: vetID, Name: "Dr. João", Email: "dr.joao@vetflow.com"})

	presc := &models.Prescription{
		VetID:        vetID,
		PetID:        pet.ID,
		GeneralNotes: "Repouso por 3 dias",
		Items: []models.PrescriptionItem{
			{Medication: "Amoxicilina", Dosage: "250mg", Route: "Oral", Frequency: "2x ao dia", Duration: "7 dias"},
		},
	}
	_ = prescRepo.Create(context.Background(), presc)

	ctrl := NewPrescriptionController(prescRepo).WithPrintContext(petRepo, clientRepo, userRepo)
	return ctrl, presc
}

func TestPrescriptionPrintRendersHTMLWithResolvedHeader(t *testing.T) {
	vetID := primitive.NewObjectID()
	ctrl, presc := newPrintControllerWithData(vetID)

	req := httptest.NewRequest(http.MethodGet, "/prescriptions/"+presc.ID.Hex()+"/print", nil)
	req = withIDParam(req, presc.ID, vetID)
	rec := httptest.NewRecorder()

	ctrl.Print(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"Rex", "Maria Tutora", "Dr. João", "Amoxicilina", "Repouso por 3 dias"} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered HTML missing %q", want)
		}
	}
}

func TestPrescriptionPrintJSONFormat(t *testing.T) {
	vetID := primitive.NewObjectID()
	ctrl, presc := newPrintControllerWithData(vetID)

	req := httptest.NewRequest(http.MethodGet, "/prescriptions/"+presc.ID.Hex()+"/print?format=json", nil)
	req = withIDParam(req, presc.ID, vetID)
	rec := httptest.NewRecorder()

	ctrl.Print(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var doc PrintablePrescription
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if doc.PetName != "Rex" || doc.OwnerName != "Maria Tutora" || doc.VetName != "Dr. João" {
		t.Errorf("resolved header = {pet:%q owner:%q vet:%q}, want {Rex Maria Tutora Dr. João}", doc.PetName, doc.OwnerName, doc.VetName)
	}
	if len(doc.Items) != 1 || doc.Items[0].Medication != "Amoxicilina" {
		t.Errorf("items not carried through: %+v", doc.Items)
	}
}

func TestPrescriptionPrintOtherVetForbidden(t *testing.T) {
	vetA := primitive.NewObjectID()
	ctrl, presc := newPrintControllerWithData(vetA)

	req := httptest.NewRequest(http.MethodGet, "/prescriptions/"+presc.ID.Hex()+"/print", nil)
	req = withIDParam(req, presc.ID, primitive.NewObjectID())
	rec := httptest.NewRecorder()

	ctrl.Print(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestPrescriptionPrintNotFound(t *testing.T) {
	vetID := primitive.NewObjectID()
	ctrl, _ := newPrintControllerWithData(vetID)

	missing := primitive.NewObjectID()
	req := httptest.NewRequest(http.MethodGet, "/prescriptions/"+missing.Hex()+"/print", nil)
	req = withIDParam(req, missing, vetID)
	rec := httptest.NewRecorder()

	ctrl.Print(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPrescriptionPrintUnauthenticated(t *testing.T) {
	ctrl, presc := newPrintControllerWithData(primitive.NewObjectID())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", presc.ID.Hex())
	req := httptest.NewRequest(http.MethodGet, "/prescriptions/"+presc.ID.Hex()+"/print", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	ctrl.Print(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
