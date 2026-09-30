package controllers

import (
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
}

// NewPrescriptionController creates a new prescription controller with dependency injection
func NewPrescriptionController(repo repositories.PrescriptionRepository) *PrescriptionController {
	return &PrescriptionController{
		repo: repo,
	}
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
	json.NewEncoder(w).Encode(prescription)
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
	json.NewEncoder(w).Encode(prescription)
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
	json.NewEncoder(w).Encode(prescription)
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
	json.NewEncoder(w).Encode(prescriptions)
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
