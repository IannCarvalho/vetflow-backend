package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PrescriptionController struct {
	repo repositories.PrescriptionRepository
	// The following repositories are only used to resolve the human-readable
	// header of a printable prescription (pet -> tutor -> vet). They are
	// optional: when nil, Print still renders the document with placeholders.
	petRepo    repositories.PetRepository
	clientRepo repositories.ClientRepository
	userRepo   repositories.UserRepository
}

// NewPrescriptionController creates a new prescription controller with dependency injection
func NewPrescriptionController(repo repositories.PrescriptionRepository) *PrescriptionController {
	return &PrescriptionController{
		repo: repo,
	}
}

// WithPrintContext injects the repositories needed to resolve the printable
// prescription header (pet, tutor and veterinarian). It returns the same
// controller for fluent configuration and keeps the base constructor's
// signature stable for the many tests that build a controller with just the
// prescription repository.
func (c *PrescriptionController) WithPrintContext(
	petRepo repositories.PetRepository,
	clientRepo repositories.ClientRepository,
	userRepo repositories.UserRepository,
) *PrescriptionController {
	c.petRepo = petRepo
	c.clientRepo = clientRepo
	c.userRepo = userRepo
	return c
}

// Create godoc
// @Summary Create a new prescription
// @Description Create a prescription (receituário) for a pet
// @Tags Prescriptions
// @Accept json
// @Produce json
// @Param prescription body models.Prescription true "Prescription object"
// @Success 201 {object} models.Prescription
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /prescriptions [post]
func (c *PrescriptionController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var prescription models.Prescription
	if err := json.NewDecoder(r.Body).Decode(&prescription); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validatePrescription(&prescription); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	now := time.Now()
	prescription.VetID = userID
	if prescription.Date.IsZero() {
		prescription.Date = now
	}
	prescription.CreatedAt = now
	prescription.UpdatedAt = now

	if err := c.repo.Create(r.Context(), &prescription); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar receituário no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(prescription)
}

// GetAll godoc
// @Summary List prescriptions for the authenticated veterinarian
// @Description List prescriptions, optionally filtered by pet_id
// @Tags Prescriptions
// @Accept json
// @Produce json
// @Param pet_id query string false "Pet ID to filter prescriptions"
// @Success 200 {array} models.Prescription
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /prescriptions [get]
func (c *PrescriptionController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if petID := r.URL.Query().Get("pet_id"); petID != "" {
		objectID, err := primitive.ObjectIDFromHex(petID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "Formato de pet_id inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_PET_ID_FORMAT")
			return
		}

		prescriptions, err := c.repo.FindByPetID(r.Context(), objectID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar receituários", err.Error(), "DB_QUERY_ERROR")
			return
		}

		prescriptions = filterOwnedPrescriptions(prescriptions, userID)
		writePrescriptions(w, prescriptions)
		return
	}

	prescriptions, err := c.repo.FindByVetID(r.Context(), userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar receituários", err.Error(), "DB_QUERY_ERROR")
		return
	}
	writePrescriptions(w, prescriptions)
}

// GetByID godoc
// @Summary Get prescription by ID
// @Description Get a specific prescription by ID (must belong to the authenticated vet)
// @Tags Prescriptions
// @Accept json
// @Produce json
// @Param id path string true "Prescription ID"
// @Success 200 {object} models.Prescription
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Prescription not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /prescriptions/{id} [get]
func (c *PrescriptionController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parsePrescriptionIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de receituário inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	prescription, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar receituário", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if prescription == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Receituário não encontrado", "Nenhum receituário com este ID existe no banco de dados", "PRESCRIPTION_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(prescription.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(prescription)
}

// Print godoc
// @Summary Render a printable prescription
// @Description Returns the prescription (receituário) as a print-ready HTML document with the pet, tutor and veterinarian header resolved. Pass ?format=json to receive the aggregated payload instead of HTML. Must belong to the authenticated vet.
// @Tags Prescriptions
// @Accept json
// @Produce html
// @Produce json
// @Param id path string true "Prescription ID"
// @Param format query string false "Response format: html (default) or json"
// @Success 200 {object} PrintablePrescription
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Prescription not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /prescriptions/{id}/print [get]
func (c *PrescriptionController) Print(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parsePrescriptionIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de receituário inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	prescription, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar receituário", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if prescription == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Receituário não encontrado", "Nenhum receituário com este ID existe no banco de dados", "PRESCRIPTION_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(prescription.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	doc := c.buildPrintable(r.Context(), prescription)

	// ?format=json returns the aggregated payload so the frontend can render its
	// own layout; the default is a self-contained, print-ready HTML document.
	if r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := renderPrescriptionHTML(w, doc); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao renderizar receituário", err.Error(), "RENDER_ERROR")
		return
	}
}

// buildPrintable resolves the pet, tutor and veterinarian so the printable
// document carries human-readable names rather than raw ObjectIDs. Lookups are
// best-effort: when a repository is unavailable or a record is missing, the
// corresponding fields are simply left blank.
func (c *PrescriptionController) buildPrintable(ctx context.Context, p *models.Prescription) *PrintablePrescription {
	doc := &PrintablePrescription{Prescription: p}

	var ownerID primitive.ObjectID
	if c.petRepo != nil && !p.PetID.IsZero() {
		if pet, err := c.petRepo.FindByID(ctx, p.PetID); err == nil && pet != nil {
			doc.PetName = pet.Name
			doc.PetSpecies = pet.Species
			doc.PetBreed = pet.Breed
			ownerID = pet.OwnerID
		}
	}

	if c.clientRepo != nil && !ownerID.IsZero() {
		if client, err := c.clientRepo.FindByID(ctx, ownerID); err == nil && client != nil {
			doc.OwnerName = client.FullName
		}
	}

	if c.userRepo != nil && !p.VetID.IsZero() {
		if vet, err := c.userRepo.FindByObjectID(ctx, p.VetID); err == nil && vet != nil {
			doc.VetName = vet.Name
		}
	}

	return doc
}

// Update godoc
// @Summary Update a prescription
// @Description Update an existing prescription (must belong to the authenticated vet)
// @Tags Prescriptions
// @Accept json
// @Produce json
// @Param id path string true "Prescription ID"
// @Param prescription body models.Prescription true "Updated prescription object"
// @Success 200 {object} models.Prescription
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Prescription not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /prescriptions/{id} [put]
func (c *PrescriptionController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parsePrescriptionIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de receituário inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar receituário", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Receituário não encontrado", "Nenhum receituário com este ID existe no banco de dados", "PRESCRIPTION_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var prescription models.Prescription
	if err := json.NewDecoder(r.Body).Decode(&prescription); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validatePrescription(&prescription); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	prescription.ID = objectID
	prescription.VetID = userID
	if prescription.Date.IsZero() {
		prescription.Date = existing.Date
	}
	prescription.CreatedAt = existing.CreatedAt
	prescription.UpdatedAt = time.Now()

	if err := c.repo.Update(r.Context(), objectID, &prescription); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Receituário não encontrado", "Nenhum receituário com este ID existe no banco de dados", "PRESCRIPTION_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar receituário", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(prescription)
}

// Delete godoc
// @Summary Delete a prescription
// @Description Delete a prescription by ID (must belong to the authenticated vet)
// @Tags Prescriptions
// @Accept json
// @Produce json
// @Param id path string true "Prescription ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Prescription not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /prescriptions/{id} [delete]
func (c *PrescriptionController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parsePrescriptionIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de receituário inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar receituário", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Receituário não encontrado", "Nenhum receituário com este ID existe no banco de dados", "PRESCRIPTION_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Receituário não encontrado", "Nenhum receituário com este ID existe no banco de dados", "PRESCRIPTION_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar receituário", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func parsePrescriptionIDParam(r *http.Request, name string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(chi.URLParam(r, name))
}

func writePrescriptions(w http.ResponseWriter, prescriptions []*models.Prescription) {
	if prescriptions == nil {
		prescriptions = make([]*models.Prescription, 0)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(prescriptions)
}

func filterOwnedPrescriptions(prescriptions []*models.Prescription, vetID primitive.ObjectID) []*models.Prescription {
	owned := make([]*models.Prescription, 0, len(prescriptions))
	for _, p := range prescriptions {
		if p.VetID == vetID {
			owned = append(owned, p)
		}
	}
	return owned
}

// validatePrescription validates required fields
func validatePrescription(prescription *models.Prescription) error {
	if prescription.PetID.IsZero() {
		return errValidation("pet_id is required")
	}
	if len(prescription.Items) == 0 {
		return errValidation("at least one prescription item is required")
	}
	for i, item := range prescription.Items {
		if item.Medication == "" {
			return errValidationWithDetails("items", fmt.Sprintf("medication is required for item %d", i+1))
		}
	}
	return nil
}
