package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type VaccineController struct {
	repo repositories.VaccineRepository
}

// NewVaccineController creates a new vaccine controller with dependency injection
func NewVaccineController(repo repositories.VaccineRepository) *VaccineController {
	return &VaccineController{
		repo: repo,
	}
}

// Create godoc
// @Summary Create a new vaccine
// @Description Create a new vaccine record for a pet
// @Tags Vaccines
// @Accept json
// @Produce json
// @Param vaccine body models.Vaccine true "Vaccine object"
// @Success 201 {object} models.Vaccine
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines [post]
func (c *VaccineController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var vaccine models.Vaccine
	if err := json.NewDecoder(r.Body).Decode(&vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateVaccine(&vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	// Scope the vaccine to the authenticated vet so listing by vet and the
	// notification sweep can find it (and clients can't spoof another vet).
	vaccine.VetID = userID
	vaccine.CreatedAt = time.Now()

	if err := c.repo.Create(r.Context(), &vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar vacina no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(vaccine)
}

// GetAll godoc
// @Summary Get all vaccines for the authenticated veterinarian
// @Description Get vaccines for the authenticated veterinarian, optionally filtered by pet
// @Tags Vaccines
// @Accept json
// @Produce json
// @Param pet_id query string false "Pet ID to filter vaccines"
// @Success 200 {array} models.Vaccine
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines [get]
func (c *VaccineController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	petID := r.URL.Query().Get("pet_id")

	// If pet_id is provided, filter by pet
	if petID != "" {
		objectID, err := primitive.ObjectIDFromHex(petID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "Formato de pet_id inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_PET_ID_FORMAT")
			return
		}

		vaccines, err := c.repo.FindByPetID(r.Context(), objectID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacinas do banco de dados", err.Error(), "DB_QUERY_ERROR")
			return
		}

		if vaccines == nil {
			vaccines = make([]*models.Vaccine, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vaccines)
		return
	}

	// Otherwise, return all vaccines for the vet
	vaccines, err := c.repo.FindByVetID(r.Context(), userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar todas as vacinas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if vaccines == nil {
		vaccines = make([]*models.Vaccine, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(vaccines)
}

// GetByID godoc
// @Summary Get vaccine by ID
// @Description Get a specific vaccine by ID
// @Tags Vaccines
// @Accept json
// @Produce json
// @Param id path string true "Vaccine ID"
// @Success 200 {object} models.Vaccine
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Vaccine not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines/{id} [get]
func (c *VaccineController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID de vacina", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	vaccine, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacina", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if vaccine == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Vacina não encontrada", "Nenhuma vacina com este ID existe no banco de dados", "VACCINE_NOT_FOUND")
		return
	}

	if err := validateResourceOwnership(vaccine.VetID, userID); err != nil {
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(vaccine)
}

// Update godoc
// @Summary Update vaccine
// @Description Update an existing vaccine
// @Tags Vaccines
// @Accept json
// @Produce json
// @Param id path string true "Vaccine ID"
// @Param vaccine body models.Vaccine true "Updated vaccine object"
// @Success 200 {object} models.Vaccine
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Vaccine not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines/{id} [put]
func (c *VaccineController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID de vacina", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacina", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Vacina não encontrada", "Nenhuma vacina com este ID existe no banco de dados", "VACCINE_NOT_FOUND")
		return
	}

	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
		return
	}

	var vaccine models.Vaccine
	if err := json.NewDecoder(r.Body).Decode(&vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateVaccine(&vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	vaccine.ID = objectID
	// Preserve vet ownership so an update never blanks or reassigns vet_id.
	vaccine.VetID = userID

	if err := c.repo.Update(r.Context(), objectID, &vaccine); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Vacina não encontrada", "Nenhuma vacina com este ID existe no banco de dados", "VACCINE_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar vacina", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(vaccine)
}

// Delete godoc
// @Summary Delete vaccine
// @Description Delete a vaccine by ID
// @Tags Vaccines
// @Accept json
// @Produce json
// @Param id path string true "Vaccine ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Vaccine not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines/{id} [delete]
func (c *VaccineController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID de vacina", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacina", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Vacina não encontrada", "Nenhuma vacina com este ID existe no banco de dados", "VACCINE_NOT_FOUND")
		return
	}

	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
		return
	}

	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Vacina não encontrada", "Nenhuma vacina com este ID existe no banco de dados", "VACCINE_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar vacina", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateVaccine validates required fields: the pet and a non-empty vaccine name.
func validateVaccine(vaccine *models.Vaccine) error {
	if vaccine.PetID.IsZero() {
		return errValidationWithDetails("pet_id", "pet é obrigatório")
	}
	if strings.TrimSpace(vaccine.VaccineName) == "" {
		return errValidationWithDetails("vaccine_name", "nome da vacina é obrigatório")
	}
	return nil
}
