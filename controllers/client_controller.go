package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ClientController struct {
	repo             repositories.ClientRepository
	petRepo          repositories.PetRepository
	consultationRepo repositories.ConsultationRepository
	vaccineRepo      repositories.VaccineRepository
	prescriptionRepo repositories.PrescriptionRepository
	invoiceRepo      repositories.InvoiceRepository
	appointmentRepo  repositories.AppointmentRepository
	attachmentRepo   repositories.AttachmentRepository
}

// NewClientController creates a new client controller with dependency injection
func NewClientController(
	repo repositories.ClientRepository,
	petRepo repositories.PetRepository,
	consultationRepo repositories.ConsultationRepository,
	vaccineRepo repositories.VaccineRepository,
	prescriptionRepo repositories.PrescriptionRepository,
	invoiceRepo repositories.InvoiceRepository,
	appointmentRepo repositories.AppointmentRepository,
	attachmentRepo repositories.AttachmentRepository,
) *ClientController {
	return &ClientController{
		repo:             repo,
		petRepo:          petRepo,
		consultationRepo: consultationRepo,
		vaccineRepo:      vaccineRepo,
		prescriptionRepo: prescriptionRepo,
		invoiceRepo:      invoiceRepo,
		appointmentRepo:  appointmentRepo,
		attachmentRepo:   attachmentRepo,
	}
}

// Create godoc
// @Summary Create a new client
// @Description Create a new veterinary clinic client
// @Tags Clients
// @Accept json
// @Produce json
// @Param client body models.Client true "Client object"
// @Success 201 {object} models.Client
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Server error"
// @Router /clients [post]
func (c *ClientController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeErrorWithDetails(w, http.StatusUnauthorized, "Falha na autenticação", err.Error(), "AUTH_ERROR")
		return
	}

	var client models.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateClient(&client); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	client.VetID = userID
	client.CreatedAt = time.Now()
	client.UpdatedAt = time.Now()

	if err := c.repo.Create(r.Context(), &client); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar cliente no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(client)
}

// GetByID godoc
// @Summary Get client by ID
// @Description Get a specific client by their ID
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} models.Client
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Client not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /clients/{id} [get]
func (c *ClientController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeErrorWithDetails(w, http.StatusUnauthorized, "Falha na autenticação", err.Error(), "AUTH_ERROR")
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de cliente inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	client, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar cliente", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if client == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Cliente não encontrado", "Nenhum cliente com este ID existe no banco de dados", "CLIENT_NOT_FOUND")
		return
	}

	if err := validateResourceOwnership(client.VetID, userID); err != nil {
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(client)
}

// GetAll godoc
// @Summary List all clients
// @Description Retrieve all registered clients for the authenticated veterinarian
// @Tags Clients
// @Accept json
// @Produce json
// @Success 200 {array} models.Client
// @Failure 500 {object} map[string]string "Server error"
// @Router /clients [get]
func (c *ClientController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeErrorWithDetails(w, http.StatusUnauthorized, "Falha na autenticação", err.Error(), "AUTH_ERROR")
		return
	}

	clients, err := c.repo.FindByVetID(r.Context(), userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar todos os clientes", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if clients == nil {
		clients = make([]*models.Client, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(clients)
}

// Update godoc
// @Summary Update client
// @Description Update an existing client
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param client body models.Client true "Updated client object"
// @Success 200 {object} models.Client
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 403 {object} map[string]string "Access denied"
// @Failure 404 {object} map[string]string "Client not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /clients/{id} [put]
func (c *ClientController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeErrorWithDetails(w, http.StatusUnauthorized, "Falha na autenticação", err.Error(), "AUTH_ERROR")
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de cliente inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	existingClient, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar cliente", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if existingClient == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Cliente não encontrado", "Nenhum cliente com este ID existe no banco de dados", "CLIENT_NOT_FOUND")
		return
	}

	if err := validateResourceOwnership(existingClient.VetID, userID); err != nil {
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
		return
	}

	var client models.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateClient(&client); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	client.ID = objectID
	client.VetID = userID
	client.UpdatedAt = time.Now()

	if err := c.repo.Update(r.Context(), objectID, &client); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Cliente não encontrado", "Nenhum cliente com este ID existe no banco de dados", "CLIENT_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar cliente", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(client)
}

// Delete godoc
// @Summary Delete client
// @Description Delete a client by ID (cascades to delete all associated pets, consultations, and vaccines)
// @Tags Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 403 {object} map[string]string "Access denied"
// @Failure 404 {object} map[string]string "Client not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /clients/{id} [delete]
func (c *ClientController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeErrorWithDetails(w, http.StatusUnauthorized, "Falha na autenticação", err.Error(), "AUTH_ERROR")
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de ID de cliente inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_ID_FORMAT")
		return
	}

	existingClient, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar cliente", err.Error(), "DB_QUERY_ERROR")
		return
	}

	if existingClient == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Cliente não encontrado", "Nenhum cliente com este ID existe no banco de dados", "CLIENT_NOT_FOUND")
		return
	}

	if err := validateResourceOwnership(existingClient.VetID, userID); err != nil {
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
		return
	}

	// Get all pets for this client
	pets, err := c.petRepo.FindByOwnerID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar pets associados ao cliente", err.Error(), "DB_QUERY_ERROR")
		return
	}

	// Delete all consultations and vaccines for each pet
	for _, pet := range pets {
		if err := c.consultationRepo.DeleteByPetID(r.Context(), pet.ID); err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar consultas do pet", err.Error(), "DB_DELETE_ERROR")
			return
		}
		if err := c.vaccineRepo.DeleteByPetID(r.Context(), pet.ID); err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar vacinas do pet", err.Error(), "DB_DELETE_ERROR")
			return
		}
		if err := c.prescriptionRepo.DeleteByPetID(r.Context(), pet.ID); err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar receituários do pet", err.Error(), "DB_DELETE_ERROR")
			return
		}
		if err := c.appointmentRepo.DeleteByPetID(r.Context(), pet.ID); err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar agendamentos do pet", err.Error(), "DB_DELETE_ERROR")
			return
		}
		if err := c.attachmentRepo.DeleteByPetID(r.Context(), pet.ID); err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar anexos do pet", err.Error(), "DB_DELETE_ERROR")
			return
		}
	}

	// Delete all invoices for this client
	if err := c.invoiceRepo.DeleteByClientID(r.Context(), objectID); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar faturas do cliente", err.Error(), "DB_DELETE_ERROR")
		return
	}

	// Delete all appointments directly linked to this client
	if err := c.appointmentRepo.DeleteByClientID(r.Context(), objectID); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar agendamentos do cliente", err.Error(), "DB_DELETE_ERROR")
		return
	}

	// Delete all pets for this client
	if err := c.petRepo.DeleteByClientID(r.Context(), objectID); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar pets do cliente", err.Error(), "DB_DELETE_ERROR")
		return
	}

	// Delete the client
	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Cliente não encontrado", "Nenhum cliente com este ID existe no banco de dados", "CLIENT_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar cliente", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateClient validates client fields. FullName is required (at least two
// characters after trimming); CPF, Phone and Address.ZipCode are optional, but
// when informed they must be well-formed Brazilian values.
func validateClient(client *models.Client) error {
	if len(strings.TrimSpace(client.FullName)) < 2 {
		return errValidationWithDetails("full_name", "nome completo é obrigatório e deve ter ao menos 2 caracteres")
	}
	if client.CPF != "" && !helpers.IsValidCPF(client.CPF) {
		return errValidationWithDetails("cpf", "CPF inválido")
	}
	if client.Phone != "" && !helpers.IsValidBRPhone(client.Phone) {
		return errValidationWithDetails("phone", "telefone inválido (use DDD + número, 10 ou 11 dígitos)")
	}
	if client.Address.ZipCode != "" && !helpers.IsValidCEP(client.Address.ZipCode) {
		return errValidationWithDetails("zip_code", "CEP inválido (deve conter 8 dígitos)")
	}
	return nil
}
