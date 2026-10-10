package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// decodeErrorBody decodes a JSON error response body and returns its "error"
// field, failing the test if the body is not valid JSON.
func decodeErrorBody(t *testing.T, body []byte) string {
	t.Helper()
	var parsed errorResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decode error body: %v (raw: %s)", err, body)
	}
	return parsed.Error
}

// newPetControllerWithFakes wires a PetController to fresh fakes, returning the
// controller plus the client and pet repos the tests seed and assert against.
func newPetControllerWithFakes() (*PetController, *fakeClientRepo, *fakePetRepo) {
	clientRepo := newFakeClientRepo()
	petRepo := newFakePetRepo(clientRepo)
	ctrl := NewPetController(
		petRepo,
		clientRepo,
		newFakeConsultationRepo(),
		newFakeVaccineRepo(),
		newFakePrescriptionRepo(),
		newFakeAppointmentRepo(),
		newFakeAttachmentRepo(),
	)
	return ctrl, clientRepo, petRepo
}

func seedClient(t *testing.T, repo *fakeClientRepo, vetID primitive.ObjectID) *models.Client {
	t.Helper()
	client := &models.Client{VetID: vetID, FullName: "Owner", CPF: "529.982.247-25", Phone: "(11) 98765-4321"}
	if err := repo.Create(context.Background(), client); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	return client
}

func seedPet(t *testing.T, repo *fakePetRepo, ownerID primitive.ObjectID) *models.Pet {
	t.Helper()
	pet := &models.Pet{Name: "Rex", Species: "dog", OwnerID: ownerID}
	if err := repo.Create(context.Background(), pet); err != nil {
		t.Fatalf("seed pet: %v", err)
	}
	return pet
}

func TestPetCreateOwnershipOfOwningClient(t *testing.T) {
	ctrl, clientRepo, petRepo := newPetControllerWithFakes()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	client := seedClient(t, clientRepo, vetA)

	// vetA creates a pet for its own client -> 201
	bodyOK := models.Pet{Name: "Rex", Species: "dog", OwnerID: client.ID}
	reqOK := withUserID(httptest.NewRequest(http.MethodPost, "/pets", mustJSON(t, bodyOK)), vetA)
	recOK := httptest.NewRecorder()
	ctrl.Create(recOK, reqOK)
	if recOK.Code != http.StatusCreated {
		t.Fatalf("vetA create: status = %d, want %d (body: %s)", recOK.Code, http.StatusCreated, recOK.Body.String())
	}
	if len(petRepo.items) != 1 {
		t.Fatalf("expected 1 stored pet, got %d", len(petRepo.items))
	}

	// vetB creates a pet for vetA's client -> 403
	bodyBad := models.Pet{Name: "Spy", Species: "cat", OwnerID: client.ID}
	reqBad := withUserID(httptest.NewRequest(http.MethodPost, "/pets", mustJSON(t, bodyBad)), vetB)
	recBad := httptest.NewRecorder()
	ctrl.Create(recBad, reqBad)
	if recBad.Code != http.StatusForbidden {
		t.Fatalf("vetB create: status = %d, want %d", recBad.Code, http.StatusForbidden)
	}
	if len(petRepo.items) != 1 {
		t.Errorf("vetB create mutated store: now %d pets", len(petRepo.items))
	}
}

func TestPetGetAllIsolatesByVet(t *testing.T) {
	ctrl, clientRepo, petRepo := newPetControllerWithFakes()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	clientA := seedClient(t, clientRepo, vetA)
	clientB := seedClient(t, clientRepo, vetB)
	seedPet(t, petRepo, clientA.ID)
	seedPet(t, petRepo, clientA.ID)
	seedPet(t, petRepo, clientB.ID)

	req := withUserID(httptest.NewRequest(http.MethodGet, "/pets", nil), vetA)
	rec := httptest.NewRecorder()
	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []models.Pet
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 pets for vetA, got %d", len(got))
	}
	for _, p := range got {
		if p.OwnerID != clientA.ID {
			t.Errorf("got pet owned by %s, want vetA's client %s", p.OwnerID.Hex(), clientA.ID.Hex())
		}
	}
}

func TestPetUpdateOtherVetForbidden(t *testing.T) {
	ctrl, clientRepo, petRepo := newPetControllerWithFakes()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	clientA := seedClient(t, clientRepo, vetA)
	pet := seedPet(t, petRepo, clientA.ID)

	body := models.Pet{Name: "Hijacked", Species: "dog", OwnerID: clientA.ID}
	req := httptest.NewRequest(http.MethodPut, "/pets/"+pet.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, pet.ID, vetB)
	rec := httptest.NewRecorder()
	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if petRepo.items[pet.ID].Name != "Rex" {
		t.Errorf("pet was mutated: name = %q, want %q", petRepo.items[pet.ID].Name, "Rex")
	}
}

func TestPetDeleteOtherVetForbidden(t *testing.T) {
	ctrl, clientRepo, petRepo := newPetControllerWithFakes()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	clientA := seedClient(t, clientRepo, vetA)
	pet := seedPet(t, petRepo, clientA.ID)

	req := httptest.NewRequest(http.MethodDelete, "/pets/"+pet.ID.Hex(), nil)
	req = withIDParam(req, pet.ID, vetB)
	rec := httptest.NewRecorder()
	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if _, ok := petRepo.items[pet.ID]; !ok {
		t.Error("pet was deleted despite ownership mismatch")
	}
}

func TestPetUnauthenticated(t *testing.T) {
	ctrl, _, _ := newPetControllerWithFakes()
	id := primitive.NewObjectID()

	tests := []struct {
		name   string
		call   func(w http.ResponseWriter, r *http.Request)
		method string
	}{
		{name: "Create", call: ctrl.Create, method: http.MethodPost},
		{name: "GetAll", call: ctrl.GetAll, method: http.MethodGet},
		{name: "GetByID", call: ctrl.GetByID, method: http.MethodGet},
		{name: "Update", call: ctrl.Update, method: http.MethodPut},
		{name: "Delete", call: ctrl.Delete, method: http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/pets/"+id.Hex(), nil)
			rec := httptest.NewRecorder()
			tt.call(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

// TestPetCreateClientNotFound locks in the pet Create not-found mapping: when
// the owning client does not exist, Create returns 400 with "Client not found".
func TestPetCreateClientNotFound(t *testing.T) {
	ctrl, _, petRepo := newPetControllerWithFakes()

	vet := primitive.NewObjectID()
	missingClient := primitive.NewObjectID()

	body := models.Pet{Name: "Rex", Species: "dog", OwnerID: missingClient}
	req := withUserID(httptest.NewRequest(http.MethodPost, "/pets", mustJSON(t, body)), vet)
	rec := httptest.NewRecorder()
	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if got := decodeErrorBody(t, rec.Body.Bytes()); got != "Client not found" {
		t.Fatalf("error = %q, want %q", got, "Client not found")
	}
	if len(petRepo.items) != 0 {
		t.Errorf("expected no stored pet, got %d", len(petRepo.items))
	}
}

// TestPetReadWriteClientNotFound locks in the shared read/update/delete not-found
// mapping: when a pet exists but its owning client does not, GetByID, Update and
// Delete each return 404 with "Client not found".
func TestPetReadWriteClientNotFound(t *testing.T) {
	vet := primitive.NewObjectID()
	missingClient := primitive.NewObjectID()

	tests := []struct {
		name   string
		method string
		call   func(ctrl *PetController) func(http.ResponseWriter, *http.Request)
	}{
		{name: "GetByID", method: http.MethodGet, call: func(ctrl *PetController) func(http.ResponseWriter, *http.Request) { return ctrl.GetByID }},
		{name: "Update", method: http.MethodPut, call: func(ctrl *PetController) func(http.ResponseWriter, *http.Request) { return ctrl.Update }},
		{name: "Delete", method: http.MethodDelete, call: func(ctrl *PetController) func(http.ResponseWriter, *http.Request) { return ctrl.Delete }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, _, petRepo := newPetControllerWithFakes()
			// Pet exists, but its owning client was never seeded.
			pet := seedPet(t, petRepo, missingClient)

			req := httptest.NewRequest(tt.method, "/pets/"+pet.ID.Hex(), nil)
			req = withIDParam(req, pet.ID, vet)
			rec := httptest.NewRecorder()
			tt.call(ctrl)(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusNotFound, rec.Body.String())
			}
			if got := decodeErrorBody(t, rec.Body.Bytes()); got != "Client not found" {
				t.Fatalf("error = %q, want %q", got, "Client not found")
			}
		})
	}
}

// TestPetClientRepoError locks in the DB-error mapping: when the client
// repository returns an error from FindByID, the handlers return 500 with
// "Failed to verify client ownership".
func TestPetClientRepoError(t *testing.T) {
	vet := primitive.NewObjectID()
	owner := primitive.NewObjectID()

	t.Run("Create", func(t *testing.T) {
		ctrl, clientRepo, _ := newPetControllerWithFakes()
		clientRepo.findByIDErr = errors.New("db down")

		body := models.Pet{Name: "Rex", Species: "dog", OwnerID: owner}
		req := withUserID(httptest.NewRequest(http.MethodPost, "/pets", mustJSON(t, body)), vet)
		rec := httptest.NewRecorder()
		ctrl.Create(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
		}
		if got := decodeErrorBody(t, rec.Body.Bytes()); got != "Failed to verify client ownership" {
			t.Fatalf("error = %q, want %q", got, "Failed to verify client ownership")
		}
	})

	readWrite := []struct {
		name   string
		method string
		call   func(ctrl *PetController) func(http.ResponseWriter, *http.Request)
	}{
		{name: "GetByID", method: http.MethodGet, call: func(ctrl *PetController) func(http.ResponseWriter, *http.Request) { return ctrl.GetByID }},
		{name: "Update", method: http.MethodPut, call: func(ctrl *PetController) func(http.ResponseWriter, *http.Request) { return ctrl.Update }},
		{name: "Delete", method: http.MethodDelete, call: func(ctrl *PetController) func(http.ResponseWriter, *http.Request) { return ctrl.Delete }},
	}

	for _, tt := range readWrite {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, clientRepo, petRepo := newPetControllerWithFakes()
			pet := seedPet(t, petRepo, owner)
			clientRepo.findByIDErr = errors.New("db down")

			req := httptest.NewRequest(tt.method, "/pets/"+pet.ID.Hex(), nil)
			req = withIDParam(req, pet.ID, vet)
			rec := httptest.NewRecorder()
			tt.call(ctrl)(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusInternalServerError, rec.Body.String())
			}
			if got := decodeErrorBody(t, rec.Body.Bytes()); got != "Failed to verify client ownership" {
				t.Fatalf("error = %q, want %q", got, "Failed to verify client ownership")
			}
		})
	}
}
