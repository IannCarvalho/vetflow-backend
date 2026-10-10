package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConsultationController struct {
	repo repositories.ConsultationRepository
}

// NewConsultationController creates a new consultation controller with dependency injection
func NewConsultationController(repo repositories.ConsultationRepository) *ConsultationController {
	return &ConsultationController{
		repo: repo,
	}
}

// Create godoc
// @Summary Create a new consultation
// @Description Create a new veterinary consultation record
// @Tags Consultations
// @Accept json
// @Produce json
// @Param consultation body models.Consultation true "Consultation object"
// @Success 201 {object} models.Consultation
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Server error"
// @Router /consultations [post]
func (c *ConsultationController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var consultation models.Consultation
	if err := json.NewDecoder(r.Body).Decode(&consultation); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateConsultation(&consultation); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	// Scope the consultation to the authenticated vet so listing by vet works
	// and records cannot be attributed to another vet by the client.
	consultation.VetID = userID
	consultation.CreatedAt = time.Now()

	if err := c.repo.Create(r.Context(), &consultation); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar consulta no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(consultation)
}

// GetAll godoc
// @Summary Get all consultations for the authenticated veterinarian
// @Description Get consultations for the authenticated veterinarian, optionally filtered by pet
// @Tags Consultations
// @Accept json
// @Produce json
// @Param pet_id query string false "Pet ID to filter consultations"
// @Success 200 {array} models.Consultation
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Server error"
// @Router /consultations [get]
func (c *ConsultationController) GetAll(w http.ResponseWriter, r *http.Request) {
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

		consultations, err := c.repo.FindByPetID(r.Context(), objectID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar consultas do banco de dados", err.Error(), "DB_QUERY_ERROR")
			return
		}

		if consultations == nil {
			consultations = make([]*models.Consultation, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(consultations)
		return
	}

	// Otherwise, return all consultations for the vet
	consultations, err := c.repo.FindByVetID(r.Context(), userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar todas as consultas", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if consultations == nil {
		consultations = make([]*models.Consultation, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(consultations)
}

// GetByID godoc
// @Summary Get consultation by ID
// @Description Get a specific consultation by ID
// @Tags Consultations
// @Accept json
// @Produce json
// @Param id path string true "Consultation ID"
// @Success 200 {object} models.Consultation
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Consultation not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /consultations/{id} [get]
func (c *ConsultationController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de consulta inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	consultation, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar consulta", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if consultation == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Consulta não encontrada", "Nenhuma consulta com este ID existe no banco de dados", "CONSULTATION_NOT_FOUND")
		return
	}

	// Only expose consultations that belong to the authenticated vet. Legacy
	// records with a zero VetID remain visible until re-saved.
	if !consultation.VetID.IsZero() {
		if err := validateResourceOwnership(consultation.VetID, userID); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(consultation)
}

// PrescriptionDraft godoc
// @Summary Build a prescription draft from a consultation
// @Description Returns an unsaved prescription (receituário) pre-filled from the consultation: pet_id, consultation_id, today's date and general notes seeded from the main complaint and any medications noted in the summary. The client edits it and persists it with POST /prescriptions. Must belong to the authenticated vet.
// @Tags Consultations
// @Accept json
// @Produce json
// @Param id path string true "Consultation ID"
// @Success 200 {object} models.Prescription
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Consultation not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /consultations/{id}/prescription-draft [get]
func (c *ConsultationController) PrescriptionDraft(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de consulta inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	consultation, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar consulta", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if consultation == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Consulta não encontrada", "Nenhuma consulta com este ID existe no banco de dados", "CONSULTATION_NOT_FOUND")
		return
	}
	// Legacy records with a zero VetID remain accessible; otherwise block drafting
	// from another vet's consultation.
	if !consultation.VetID.IsZero() {
		if err := validateResourceOwnership(consultation.VetID, userID); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	draft := buildPrescriptionDraft(consultation)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(draft)
}

// buildPrescriptionDraft assembles an unsaved prescription seeded from a
// consultation. It intentionally leaves the ID and VetID zero: the draft is not
// persisted here, and POST /prescriptions sets VetID from the token on save. A
// single empty item is included so the client has a row to start editing.
func buildPrescriptionDraft(consultation *models.Consultation) models.Prescription {
	var notes strings.Builder
	if complaint := strings.TrimSpace(consultation.Anamnesis.MainComplaint); complaint != "" {
		notes.WriteString("Queixa principal: ")
		notes.WriteString(complaint)
	}
	if meds := strings.TrimSpace(consultation.Summary.Medications); meds != "" {
		if notes.Len() > 0 {
			notes.WriteString("\n")
		}
		notes.WriteString("Medicações em uso: ")
		notes.WriteString(meds)
	}

	return models.Prescription{
		PetID:          consultation.PetID,
		ConsultationID: consultation.ID,
		Date:           time.Now(),
		Items:          []models.PrescriptionItem{{}},
		GeneralNotes:   notes.String(),
	}
}

// Update godoc
// @Summary Update consultation
// @Description Update an existing consultation
// @Tags Consultations
// @Accept json
// @Produce json
// @Param id path string true "Consultation ID"
// @Param consultation body models.Consultation true "Updated consultation object"
// @Success 200 {object} models.Consultation
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Consultation not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /consultations/{id} [put]
func (c *ConsultationController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de consulta inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	// Ensure the consultation exists and belongs to the authenticated vet.
	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar consulta", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Consulta não encontrada", "Nenhuma consulta com este ID existe no banco de dados", "CONSULTATION_NOT_FOUND")
		return
	}
	// Records created before vet-scoping may have a zero VetID; allow those to
	// be claimed, but block editing another vet's consultation.
	if !existing.VetID.IsZero() {
		if err := validateResourceOwnership(existing.VetID, userID); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	var consultation models.Consultation
	if err := json.NewDecoder(r.Body).Decode(&consultation); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateConsultation(&consultation); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	consultation.ID = objectID
	consultation.VetID = userID

	if err := c.repo.Update(r.Context(), objectID, &consultation); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Consulta não encontrada", "Nenhuma consulta com este ID existe no banco de dados", "CONSULTATION_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar consulta", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(consultation)
}

// Delete godoc
// @Summary Delete consultation
// @Description Delete a consultation by ID
// @Tags Consultations
// @Accept json
// @Produce json
// @Param id path string true "Consultation ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Consultation not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /consultations/{id} [delete]
func (c *ConsultationController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de consulta inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	// Ensure the consultation exists and belongs to the authenticated vet before
	// deleting, mirroring GetByID/Update.
	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar consulta", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Consulta não encontrada", "Nenhuma consulta com este ID existe no banco de dados", "CONSULTATION_NOT_FOUND")
		return
	}
	// Legacy records with a zero VetID remain deletable; otherwise block removing
	// another vet's consultation.
	if !existing.VetID.IsZero() {
		if err := validateResourceOwnership(existing.VetID, userID); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Consulta não encontrada", "Nenhuma consulta com este ID existe no banco de dados", "CONSULTATION_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar consulta", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateConsultation validates required fields: the pet and a non-empty main
// complaint in the anamnesis.
func validateConsultation(consultation *models.Consultation) error {
	if consultation.PetID.IsZero() {
		return errValidationWithDetails("pet_id", "pet é obrigatório")
	}
	if strings.TrimSpace(consultation.Anamnesis.MainComplaint) == "" {
		return errValidationWithDetails("main_complaint", "queixa principal é obrigatória")
	}
	return nil
}
