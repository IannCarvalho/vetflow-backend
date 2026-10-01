package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CatalogItemController struct {
	repo repositories.CatalogItemRepository
}

// NewCatalogItemController creates a new catalog item controller with dependency injection
func NewCatalogItemController(repo repositories.CatalogItemRepository) *CatalogItemController {
	return &CatalogItemController{
		repo: repo,
	}
}

// Create godoc
// @Summary Create a new catalog item
// @Description Create a service or product in the clinic catalog
// @Tags Catalog
// @Accept json
// @Produce json
// @Param item body models.CatalogItem true "Catalog item object"
// @Success 201 {object} models.CatalogItem
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /catalog [post]
func (c *CatalogItemController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var item models.CatalogItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateCatalogItem(&item); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	now := time.Now()
	item.VetID = userID
	normalizeCatalogItem(&item)
	item.CreatedAt = now
	item.UpdatedAt = now

	if err := c.repo.Create(r.Context(), &item); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar item no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

// GetAll godoc
// @Summary List catalog items for the authenticated veterinarian
// @Description List services and products, optionally filtered by type
// @Tags Catalog
// @Accept json
// @Produce json
// @Param type query string false "Filter by type: service or product"
// @Success 200 {array} models.CatalogItem
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /catalog [get]
func (c *CatalogItemController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	items, err := c.repo.FindByVetID(r.Context(), userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar itens do catálogo", err.Error(), "DB_QUERY_ERROR")
		return
	}

	// Optional filter by type
	if typeFilter := r.URL.Query().Get("type"); typeFilter != "" {
		filtered := make([]*models.CatalogItem, 0, len(items))
		for _, it := range items {
			if string(it.Type) == typeFilter {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}

	if items == nil {
		items = make([]*models.CatalogItem, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GetByID godoc
// @Summary Get catalog item by ID
// @Description Get a specific catalog item by ID (must belong to the authenticated vet)
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Catalog item ID"
// @Success 200 {object} models.CatalogItem
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Item not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /catalog/{id} [get]
func (c *CatalogItemController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parseCatalogIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	item, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar item", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if item == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Item não encontrado", "Nenhum item com este ID existe no banco de dados", "CATALOG_ITEM_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(item.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// Update godoc
// @Summary Update a catalog item
// @Description Update an existing catalog item (must belong to the authenticated vet)
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Catalog item ID"
// @Param item body models.CatalogItem true "Updated catalog item object"
// @Success 200 {object} models.CatalogItem
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Item not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /catalog/{id} [put]
func (c *CatalogItemController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parseCatalogIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar item", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Item não encontrado", "Nenhum item com este ID existe no banco de dados", "CATALOG_ITEM_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var item models.CatalogItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateCatalogItem(&item); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	item.ID = objectID
	item.VetID = userID
	normalizeCatalogItem(&item)
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = time.Now()

	if err := c.repo.Update(r.Context(), objectID, &item); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Item não encontrado", "Nenhum item com este ID existe no banco de dados", "CATALOG_ITEM_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar item", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

// Delete godoc
// @Summary Delete a catalog item
// @Description Delete a catalog item by ID (must belong to the authenticated vet)
// @Tags Catalog
// @Accept json
// @Produce json
// @Param id path string true "Catalog item ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Item not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /catalog/{id} [delete]
func (c *CatalogItemController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parseCatalogIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar item", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Item não encontrado", "Nenhum item com este ID existe no banco de dados", "CATALOG_ITEM_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Item não encontrado", "Nenhum item com este ID existe no banco de dados", "CATALOG_ITEM_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar item", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func parseCatalogIDParam(r *http.Request, name string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(chi.URLParam(r, name))
}

// normalizeCatalogItem applies defaults and clears stock for services.
func normalizeCatalogItem(item *models.CatalogItem) {
	if item.Type == "" {
		item.Type = models.CatalogService
	}
	// Stock only makes sense for products.
	if item.Type == models.CatalogService {
		item.StockQuantity = 0
		item.MinStock = 0
	}
	if item.Price < 0 {
		item.Price = 0
	}
}

// validateCatalogItem validates required fields
func validateCatalogItem(item *models.CatalogItem) error {
	if item.Name == "" {
		return errValidation("name is required")
	}
	if item.Type != "" && item.Type != models.CatalogService && item.Type != models.CatalogProduct {
		return errValidation("type must be 'service' or 'product'")
	}
	if item.Price < 0 {
		return errValidation("price must not be negative")
	}
	return nil
}
