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

func TestCatalogItemCreateSetsVetScope(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)

	vetID := primitive.NewObjectID()
	body := models.CatalogItem{Name: "Consulta geral", Type: models.CatalogService, Price: 120}

	req := httptest.NewRequest(http.MethodPost, "/catalog", mustJSON(t, body))
	req = withUserID(req, vetID)
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected 1 stored catalog item, got %d", len(repo.items))
	}
	for _, it := range repo.items {
		if it.VetID != vetID {
			t.Errorf("stored vet_id = %s, want %s", it.VetID.Hex(), vetID.Hex())
		}
	}
}

func TestCatalogItemCreateRequiresName(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)

	req := httptest.NewRequest(http.MethodPost, "/catalog", mustJSON(t, models.CatalogItem{Price: 10}))
	req = withUserID(req, primitive.NewObjectID())
	rec := httptest.NewRecorder()

	ctrl.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if len(repo.items) != 0 {
		t.Errorf("expected no stored item on validation failure, got %d", len(repo.items))
	}
}

func TestCatalogItemGetAllIsolatesByVet(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.CatalogItem{VetID: vetA, Name: "A1", Type: models.CatalogService})
	_ = repo.Create(context.Background(), &models.CatalogItem{VetID: vetA, Name: "A2", Type: models.CatalogProduct})
	_ = repo.Create(context.Background(), &models.CatalogItem{VetID: vetB, Name: "B1", Type: models.CatalogService})

	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []models.CatalogItem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items for vetA, got %d", len(got))
	}
	for _, it := range got {
		if it.VetID != vetA {
			t.Errorf("got item owned by %s, want %s", it.VetID.Hex(), vetA.Hex())
		}
	}
}

func TestCatalogItemGetAllFiltersByType(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)

	vetA := primitive.NewObjectID()
	_ = repo.Create(context.Background(), &models.CatalogItem{VetID: vetA, Name: "Serviço", Type: models.CatalogService})
	_ = repo.Create(context.Background(), &models.CatalogItem{VetID: vetA, Name: "Produto", Type: models.CatalogProduct})

	req := httptest.NewRequest(http.MethodGet, "/catalog?type=product", nil)
	req = withUserID(req, vetA)
	rec := httptest.NewRecorder()

	ctrl.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got []models.CatalogItem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(got) != 1 || got[0].Type != models.CatalogProduct {
		t.Fatalf("expected 1 product, got %d items", len(got))
	}
}

func TestCatalogItemGetByIDOtherVetForbidden(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.CatalogItem{VetID: vetA, Name: "Original", Type: models.CatalogService}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodGet, "/catalog/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.GetByID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCatalogItemUpdateOtherVetForbidden(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.CatalogItem{VetID: vetA, Name: "Original", Type: models.CatalogService}
	_ = repo.Create(context.Background(), original)

	body := models.CatalogItem{Name: "Hijacked", Type: models.CatalogService}
	req := httptest.NewRequest(http.MethodPut, "/catalog/"+original.ID.Hex(), mustJSON(t, body))
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	stored := repo.items[original.ID]
	if stored.Name != "Original" {
		t.Errorf("item was mutated: name = %q, want %q", stored.Name, "Original")
	}
	if stored.VetID != vetA {
		t.Errorf("vet_id was mutated: %s, want %s", stored.VetID.Hex(), vetA.Hex())
	}
}

func TestCatalogItemDeleteOtherVetForbidden(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)

	vetA := primitive.NewObjectID()
	vetB := primitive.NewObjectID()
	original := &models.CatalogItem{VetID: vetA, Name: "Original", Type: models.CatalogService}
	_ = repo.Create(context.Background(), original)

	req := httptest.NewRequest(http.MethodDelete, "/catalog/"+original.ID.Hex(), nil)
	req = withIDParam(req, original.ID, vetB)
	rec := httptest.NewRecorder()

	ctrl.Delete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if _, ok := repo.items[original.ID]; !ok {
		t.Error("item was deleted despite ownership mismatch")
	}
}

func TestCatalogItemUnauthenticated(t *testing.T) {
	repo := newFakeCatalogItemRepo()
	ctrl := NewCatalogItemController(repo)
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
				body := models.CatalogItem{Name: "X", Type: models.CatalogService}
				req = httptest.NewRequest(tt.method, "/catalog", mustJSON(t, body))
			} else {
				req = httptest.NewRequest(tt.method, "/catalog", nil)
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
