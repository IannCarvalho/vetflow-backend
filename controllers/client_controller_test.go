package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newClientControllerWithFakes wires a ClientController to fresh fakes, returning
// the controller plus the client repo the tests seed and assert against.
func newClientControllerWithFakes() (*ClientController, *fakeClientRepo) {
	clientRepo := newFakeClientRepo()
	petRepo := newFakePetRepo(clientRepo)
	ctrl := NewClientController(
		clientRepo,
		petRepo,
		newFakeConsultationRepo(),
		newFakeVaccineRepo(),
		newFakePrescriptionRepo(),
		newFakeInvoiceRepo(),
		newFakeAppointmentRepo(),
		newFakeAttachmentRepo(),
	)
	return ctrl, clientRepo
}

func TestClientCreateSetsVetScope(t *testing.T) {
	ctrl, repo := newClientControllerWithFakes()

	vetID := primitive.NewObjectID()
	body := models.Client{FullName: "Jane", CPF: "529.982.247-25", Phone: "(11) 98765-4321"}

	req := withUserID(httptest.NewRequest(http.MethodPost, "/clients", mustJSON(t, body)), vetID)
	rec := httptest.NewRecorder()
	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 stored client, got %d", len(repo.items))
	}
	for _, c := range repo.items {
		if c.VetID != vetID {
			t.Errorf("stored vet_id = %s, want %s", c.VetID.Hex(), vetID.Hex())
		}
	}
}

func TestClientGetAllIsolatesByVet(t *testing.T) {
	ctrl, repo := newClientControllerWithFakes()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.Client{VetID: vetA, FullName: "A1", CPF: "1", Phone: "1"})
	_ = repo.Create(context.Background(), &models.Client{VetID: vetA, FullName: "A2", CPF: "2", Phone: "2"})
	_ = repo.Create(context.Background(), &models.Client{VetID: vetB, FullName: "B1", CPF: "3", Phone: "3"})

	req := withUserID(httptest.NewRequest(http.MethodGet, "/clients", nil), vetA)
	rec := httptest.NewRecorder()
	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []models.Client
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 clients for vetA, got %d", len(got))
	}
	for _, c := range got {
		if c.VetID != vetA {
			t.Errorf("got client owned by %s, want %s", c.VetID.Hex(), vetA.Hex())
		}
	}
}

func TestClientUpdateOtherVetForbidden(t *testing.T) {
	ctrl, repo := newClientControllerWithFakes()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Client{VetID: vetA, FullName: "Original", CPF: "1", Phone: "1"}
	_ = repo.Create(context.Background(), original)

	body := models.Client{FullName: "Hijacked", CPF: "1", Phone: "1"}
	req := httptest.NewRequest(http.MethodPut, "/clients/"+original.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()
	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if repo.items[original.ID].FullName != "Original" {
		t.Errorf("client was mutated: full_name = %q, want %q", repo.items[original.ID].FullName, "Original")
	}
}

func TestClientDeleteOtherVetForbidden(t *testing.T) {
	ctrl, repo := newClientControllerWithFakes()

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Client{VetID: vetA, FullName: "Original", CPF: "1", Phone: "1"}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/clients/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()
	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if _, ok := repo.items[original.ID]; !ok {
		t.Error("client was deleted despite ownership mismatch")
	}
}

func TestClientUnauthenticated(t *testing.T) {
	ctrl, _ := newClientControllerWithFakes()
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
			req := httptest.NewRequest(tt.method, "/clients/"+id.Hex(), nil)
			rec := httptest.NewRecorder()
			tt.call(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}
