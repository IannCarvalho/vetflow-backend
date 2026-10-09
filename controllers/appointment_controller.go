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

type AppointmentController struct {
	repo repositories.AppointmentRepository
}

// NewAppointmentController creates a new appointment controller with dependency injection
func NewAppointmentController(repo repositories.AppointmentRepository) *AppointmentController {
	return &AppointmentController{
		repo: repo,
	}
}

// Create godoc
// @Summary Create a new appointment
// @Description Schedule a new appointment in the agenda for the authenticated veterinarian
// @Tags Appointments
// @Accept json
// @Produce json
// @Param appointment body models.Appointment true "Appointment object"
// @Success 201 {object} models.Appointment
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /appointments [post]
func (c *AppointmentController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var appointment models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&appointment); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateAppointment(&appointment); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	now := time.Now()
	appointment.VetID = userID
	appointment.Status = normalizeStatus(appointment.Status)
	appointment.CreatedAt = now
	appointment.UpdatedAt = now

	if err := c.repo.Create(r.Context(), &appointment); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar agendamento no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(appointment)
}

// GetAll godoc
// @Summary List appointments for the authenticated veterinarian
// @Description List appointments, optionally filtered by pet_id or by a date range (start/end RFC3339)
// @Tags Appointments
// @Accept json
// @Produce json
// @Param pet_id query string false "Pet ID to filter appointments"
// @Param start query string false "Range start (RFC3339)"
// @Param end query string false "Range end (RFC3339)"
// @Success 200 {array} models.Appointment
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /appointments [get]
func (c *AppointmentController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Filter by pet
	if petID := r.URL.Query().Get("pet_id"); petID != "" {
		objectID, err := primitive.ObjectIDFromHex(petID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "Formato de pet_id inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_PET_ID_FORMAT")
			return
		}

		appointments, err := c.repo.FindByPetID(r.Context(), objectID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamentos", err.Error(), "DB_QUERY_ERROR")
			return
		}

		// Keep only the vet's own appointments
		appointments = filterOwnedAppointments(appointments, userID)
		writeAppointments(w, appointments)
		return
	}

	// Filter by date range (used by the calendar view)
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	if startStr != "" && endStr != "" {
		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "Formato de 'start' inválido", "Esperado uma data no formato RFC3339", "INVALID_START_FORMAT")
			return
		}
		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "Formato de 'end' inválido", "Esperado uma data no formato RFC3339", "INVALID_END_FORMAT")
			return
		}

		appointments, err := c.repo.FindByVetIDInRange(r.Context(), userID, start, end)
		if err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamentos", err.Error(), "DB_QUERY_ERROR")
			return
		}
		writeAppointments(w, appointments)
		return
	}

	// Otherwise return all appointments for the vet
	appointments, err := c.repo.FindByVetID(r.Context(), userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamentos", err.Error(), "DB_QUERY_ERROR")
		return
	}
	writeAppointments(w, appointments)
}

// GetByID godoc
// @Summary Get appointment by ID
// @Description Get a specific appointment by ID (must belong to the authenticated vet)
// @Tags Appointments
// @Accept json
// @Produce json
// @Param id path string true "Appointment ID"
// @Success 200 {object} models.Appointment
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Appointment not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /appointments/{id} [get]
func (c *AppointmentController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parseObjectIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de agendamento inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	appointment, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamento", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if appointment == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Agendamento não encontrado", "Nenhum agendamento com este ID existe no banco de dados", "APPOINTMENT_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(appointment.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(appointment)
}

// Update godoc
// @Summary Update an appointment
// @Description Update an existing appointment (must belong to the authenticated vet)
// @Tags Appointments
// @Accept json
// @Produce json
// @Param id path string true "Appointment ID"
// @Param appointment body models.Appointment true "Updated appointment object"
// @Success 200 {object} models.Appointment
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Appointment not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /appointments/{id} [put]
func (c *AppointmentController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parseObjectIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de agendamento inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	// Ensure the appointment exists and belongs to the vet
	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamento", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Agendamento não encontrado", "Nenhum agendamento com este ID existe no banco de dados", "APPOINTMENT_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var appointment models.Appointment
	if err := json.NewDecoder(r.Body).Decode(&appointment); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateAppointment(&appointment); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	appointment.ID = objectID
	appointment.VetID = userID
	appointment.Status = normalizeStatus(appointment.Status)
	appointment.CreatedAt = existing.CreatedAt
	appointment.UpdatedAt = time.Now()

	if err := c.repo.Update(r.Context(), objectID, &appointment); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Agendamento não encontrado", "Nenhum agendamento com este ID existe no banco de dados", "APPOINTMENT_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar agendamento", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(appointment)
}

// Delete godoc
// @Summary Delete an appointment
// @Description Delete an appointment by ID (must belong to the authenticated vet)
// @Tags Appointments
// @Accept json
// @Produce json
// @Param id path string true "Appointment ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Appointment not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /appointments/{id} [delete]
func (c *AppointmentController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, err := parseObjectIDParam(r, "id")
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de agendamento inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar agendamento", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Agendamento não encontrado", "Nenhum agendamento com este ID existe no banco de dados", "APPOINTMENT_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Agendamento não encontrado", "Nenhum agendamento com este ID existe no banco de dados", "APPOINTMENT_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar agendamento", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func parseObjectIDParam(r *http.Request, name string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(chi.URLParam(r, name))
}

func writeAppointments(w http.ResponseWriter, appointments []*models.Appointment) {
	if appointments == nil {
		appointments = make([]*models.Appointment, 0)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(appointments)
}

func filterOwnedAppointments(appointments []*models.Appointment, vetID primitive.ObjectID) []*models.Appointment {
	owned := make([]*models.Appointment, 0, len(appointments))
	for _, a := range appointments {
		if a.VetID == vetID {
			owned = append(owned, a)
		}
	}
	return owned
}

func normalizeStatus(status models.AppointmentStatus) models.AppointmentStatus {
	if status == "" {
		return models.AppointmentScheduled
	}
	return status
}

// validateAppointment validates required fields
func validateAppointment(appointment *models.Appointment) error {
	if appointment.PetID.IsZero() {
		return errValidation("pet_id is required")
	}
	if appointment.StartTime.IsZero() {
		return errValidation("start_time is required")
	}
	if !appointment.EndTime.IsZero() && appointment.EndTime.Before(appointment.StartTime) {
		return errValidation("end_time must be after start_time")
	}
	return nil
}
