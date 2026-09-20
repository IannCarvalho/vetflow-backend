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
	var vaccine models.Vaccine
	if err := json.NewDecoder(r.Body).Decode(&vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateVaccine(&vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	vaccine.CreatedAt = time.Now()

	if err := c.repo.Create(r.Context(), &vaccine); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar vacina no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vaccine)
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
		json.NewEncoder(w).Encode(vaccines)
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
	json.NewEncoder(w).Encode(vaccines)
}

// GetAllPublic godoc (PUBLIC - NO AUTH REQUIRED)
// @Summary Get all vaccines (public)
// @Description Get all vaccines (public endpoint)
// @Tags Vaccines
// @Accept json
// @Produce json
// @Success 200 {array} models.Vaccine
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines [get]
func (c *VaccineController) GetAllPublic(w http.ResponseWriter, r *http.Request) {
	// When a pet_id filter is provided, return only that pet's vaccines.
	if petID := r.URL.Query().Get("pet_id"); petID != "" {
		objectID, err := primitive.ObjectIDFromHex(petID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "Formato de pet_id inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_PET_ID_FORMAT")
			return
		}

		vaccines, err := c.repo.FindByPetID(r.Context(), objectID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacinas", err.Error(), "DB_QUERY_ERROR")
			return
		}

		if vaccines == nil {
			vaccines = make([]*models.Vaccine, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vaccines)
		return
	}

	vaccines, err := c.repo.FindAll(r.Context())
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar todas as vacinas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if vaccines == nil {
		vaccines = make([]*models.Vaccine, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vaccines)
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
	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de vacina inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vaccine)
}

// GetByIDPublic godoc (PUBLIC - NO AUTH REQUIRED)
// @Summary Get vaccine by ID (public)
// @Description Get a specific vaccine by ID (public endpoint)
// @Tags Vaccines
// @Accept json
// @Produce json
// @Param id path string true "Vaccine ID"
// @Success 200 {object} models.Vaccine
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Vaccine not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines/{id} [get]
func (c *VaccineController) GetByIDPublic(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de vacina inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vaccine)
}

// GetByPetID godoc
// @Summary List vaccines by pet
// @Description Get all vaccines for a specific pet
// @Tags Vaccines
// @Accept json
// @Produce json
// @Param pet_id query string true "Pet ID"
// @Success 200 {array} models.Vaccine
// @Failure 400 {object} map[string]string "Invalid pet_id"
// @Failure 500 {object} map[string]string "Server error"
// @Router /vaccines [get]
func (c *VaccineController) GetByPetID(w http.ResponseWriter, r *http.Request) {
	petID := r.URL.Query().Get("pet_id")
	if petID == "" {
		writeErrorWithDetails(w, http.StatusBadRequest, "Parâmetro pet_id ausente", "O parâmetro de query 'pet_id' é obrigatório", "MISSING_PARAM")
		return
	}

	objectID, err := primitive.ObjectIDFromHex(petID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de pet_id inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_PET_ID_FORMAT")
		return
	}

	vaccines, err := c.repo.FindByPetID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar vacinas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if vaccines == nil {
		vaccines = make([]*models.Vaccine, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vaccines)
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
	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de vacina inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
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

	if err := c.repo.Update(r.Context(), objectID, &vaccine); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Vacina não encontrada", "Nenhuma vacina com este ID existe no banco de dados", "VACCINE_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar vacina", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vaccine)
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
	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de vacina inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
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

// validateVaccine validates required fields
func validateVaccine(vaccine *models.Vaccine) error {
	if vaccine.PetID.IsZero() {
		return errValidation("pet_id is required")
	}
	if vaccine.VaccineName == "" {
		return errValidation("vaccine_name is required")
	}
	return nil
}
