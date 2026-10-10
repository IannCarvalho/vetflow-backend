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

// validInvoiceBody builds an invoice payload that passes validateInvoice
// (requires a client and at least one item with description, qty>0, price>=0).
func validInvoiceBody(clientID primitive.ObjectID) models.Invoice {
	return models.Invoice{
		ClientID: clientID,
		Items: []models.InvoiceItem{
			{Description: "Consulta", Quantity: 2, UnitPrice: 50},
		},
	}
}

func TestInvoiceCreateSetsVetScopeAndTotals(t *testing.T) {
	repo := newFakeInvoiceRepo()
	ctrl := NewInvoiceController(repo)

	vetID := primitive.NewObjectID()
	body := validInvoiceBody(primitive.NewObjectID())
	// Client-sent totals must be ignored and recomputed on the server.
	body.Subtotal = 999
	body.Total = 999

	req := httptest.NewRequest(http.MethodPost, "/invoices", mustJSON(t, body))
	req = withUserID(req, vetID)
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 stored invoice, got %d", len(repo.items))
	}
	for _, inv := range repo.items {
		if inv.VetID != vetID {
			t.Errorf("stored vet_id = %s, want %s", inv.VetID.Hex(), vetID.Hex())
		}
		if inv.Subtotal != 100 || inv.Total != 100 {
			t.Errorf("totals not recomputed: subtotal=%v total=%v, want 100/100", inv.Subtotal, inv.Total)
		}
	}
}

func TestInvoiceGetAllIsolatesByVet(t *testing.T) {
	repo := newFakeInvoiceRepo()
	ctrl := NewInvoiceController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.Invoice{VetID: vetA, ClientID: primitive.NewObjectID()})
	_ = repo.Create(context.Background(), &models.Invoice{VetID: vetA, ClientID: primitive.NewObjectID()})
	_ = repo.Create(context.Background(), &models.Invoice{VetID: vetB, ClientID: primitive.NewObjectID()})

	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []models.Invoice
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 invoices for vetA, got %d", len(got))
	}
	for _, inv := range got {
		if inv.VetID != vetA {
			t.Errorf("got invoice owned by %s, want %s", inv.VetID.Hex(), vetA.Hex())
		}
	}
}

func TestInvoiceGetByIDOtherVetForbidden(t *testing.T) {
	repo := newFakeInvoiceRepo()
	ctrl := NewInvoiceController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Invoice{VetID: vetA, ClientID: primitive.NewObjectID()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodGet, "/invoices/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestInvoiceUpdateOtherVetForbidden(t *testing.T) {
	repo := newFakeInvoiceRepo()
	ctrl := NewInvoiceController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Invoice{VetID: vetA, ClientID: primitive.NewObjectID(), Notes: "Original"}
	_ = repo.Create(context.Background(), original)

	body := validInvoiceBody(original.ClientID)
	body.Notes = "Hijacked"
	req := httptest.NewRequest(http.MethodPut, "/invoices/"+original.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	stored := repo.items[original.ID]
	if stored.Notes != "Original" {
		t.Errorf("invoice was mutated: notes = %q, want %q", stored.Notes, "Original")
	}
	if stored.VetID != vetA {
		t.Errorf("vet_id was mutated: %s, want %s", stored.VetID.Hex(), vetA.Hex())
	}
}

func TestInvoiceDeleteOtherVetForbidden(t *testing.T) {
	repo := newFakeInvoiceRepo()
	ctrl := NewInvoiceController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.Invoice{VetID: vetA, ClientID: primitive.NewObjectID()}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/invoices/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if _, ok := repo.items[original.ID]; !ok {
		t.Error("invoice was deleted despite ownership mismatch")
	}
}

func TestInvoiceUnauthenticated(t *testing.T) {
	repo := newFakeInvoiceRepo()
	ctrl := NewInvoiceController(repo)
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
				body := validInvoiceBody(primitive.NewObjectID())
				req = httptest.NewRequest(tt.method, "/invoices", mustJSON(t, body))
			} else {
				req = httptest.NewRequest(tt.method, "/invoices", nil)
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
