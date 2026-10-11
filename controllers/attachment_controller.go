package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// maxAttachmentContentBytes caps the base64 payload at roughly 10MB encoded.
const maxAttachmentContentBytes = 10 * 1024 * 1024

type AttachmentController struct {
	repo       repositories.AttachmentRepository
	petRepo    repositories.PetRepository
	clientRepo repositories.ClientRepository
}

// NewAttachmentController creates a new attachment controller with dependency injection
func NewAttachmentController(
	repo repositories.AttachmentRepository,
	petRepo repositories.PetRepository,
	clientRepo repositories.ClientRepository,
) *AttachmentController {
	return &AttachmentController{
		repo:       repo,
		petRepo:    petRepo,
		clientRepo: clientRepo,
	}
}

// Create godoc
// @Summary Upload a new attachment
// @Description Attach an exam/image/report to a pet's medical record
// @Tags Attachments
// @Accept json
// @Produce json
// @Param attachment body models.Attachment true "Attachment object"
// @Success 201 {object} models.Attachment
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /attachments [post]
func (c *AttachmentController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var attachment models.Attachment
	if err := json.NewDecoder(r.Body).Decode(&attachment); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateAttachment(&attachment); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	// Verify the pet exists and belongs to the authenticated vet (via its owner).
	if err := c.assertPetOwnership(r, attachment.PetID, userID); err != nil {
		writeAttachmentOwnershipError(w, err)
		return
	}

	now := time.Now()
	attachment.VetID = userID
	if attachment.Category == "" {
		attachment.Category = models.AttachmentDocument
	}
	if attachment.Date.IsZero() {
		attachment.Date = now
	}
	attachment.CreatedAt = now
	attachment.UpdatedAt = now

	if err := c.repo.Create(r.Context(), &attachment); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao salvar anexo no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(attachment)
}

// GetAll godoc
// @Summary List attachments (summaries) for a pet
// @Description List attachment metadata for a pet, without the base64 content
// @Tags Attachments
// @Accept json
// @Produce json
// @Param pet_id query string true "Pet ID"
// @Success 200 {array} models.AttachmentSummary
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /attachments [get]
func (c *AttachmentController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	petIDStr := r.URL.Query().Get("pet_id")
	if petIDStr == "" {
		writeErrorWithDetails(w, http.StatusBadRequest, "Parâmetro pet_id ausente", "O parâmetro de query 'pet_id' é obrigatório", "MISSING_PARAM")
		return
	}

	petID, err := primitive.ObjectIDFromHex(petIDStr)
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de pet_id inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_PET_ID_FORMAT")
		return
	}

	if err := c.assertPetOwnership(r, petID, userID); err != nil {
		writeAttachmentOwnershipError(w, err)
		return
	}

	summaries, err := c.repo.FindSummariesByPetID(r.Context(), petID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar anexos", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if summaries == nil {
		summaries = make([]*models.AttachmentSummary, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summaries)
}

// GetContent godoc
// @Summary Get an attachment's full content
// @Description Get a specific attachment including its base64 content (for download/preview)
// @Tags Attachments
// @Accept json
// @Produce json
// @Param id path string true "Attachment ID"
// @Success 200 {object} models.Attachment
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Attachment not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /attachments/{id}/content [get]
func (c *AttachmentController) GetContent(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	attachment, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar anexo", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if attachment == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Anexo não encontrado", "Nenhum anexo com este ID existe no banco de dados", "ATTACHMENT_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(attachment.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(attachment)
}

// Update godoc
// @Summary Update attachment metadata
// @Description Update name/category/notes of an attachment (content is not changed here)
// @Tags Attachments
// @Accept json
// @Produce json
// @Param id path string true "Attachment ID"
// @Param attachment body models.Attachment true "Updated metadata"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Attachment not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /attachments/{id} [put]
func (c *AttachmentController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar anexo", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Anexo não encontrado", "Nenhum anexo com este ID existe no banco de dados", "ATTACHMENT_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var body struct {
		Name     string                    `json:"name"`
		Category models.AttachmentCategory `json:"category"`
		Notes    string                    `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}
	if body.Name == "" {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", "name is required", "VALIDATION_ERROR")
		return
	}

	updates := map[string]interface{}{
		"name":       body.Name,
		"notes":      body.Notes,
		"updated_at": time.Now(),
	}
	if body.Category != "" {
		updates["category"] = body.Category
	}

	if err := c.repo.Update(r.Context(), objectID, updates); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Anexo não encontrado", "Nenhum anexo com este ID existe no banco de dados", "ATTACHMENT_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar anexo", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Anexo atualizado com sucesso"})
}

// Delete godoc
// @Summary Delete an attachment
// @Description Delete an attachment by ID (must belong to the authenticated vet)
// @Tags Attachments
// @Accept json
// @Produce json
// @Param id path string true "Attachment ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Attachment not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /attachments/{id} [delete]
func (c *AttachmentController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar anexo", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Anexo não encontrado", "Nenhum anexo com este ID existe no banco de dados", "ATTACHMENT_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Anexo não encontrado", "Nenhum anexo com este ID existe no banco de dados", "ATTACHMENT_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar anexo", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

// assertPetOwnership confirms the pet exists and its owning client belongs to
// the authenticated vet. Pet ownership is derived through the owning client's
// VetID. Returns a sentinel error describing what failed.
func (c *AttachmentController) assertPetOwnership(r *http.Request, petID, userID primitive.ObjectID) error {
	pet, err := c.petRepo.FindByID(r.Context(), petID)
	if err != nil {
		return errAttachmentDB
	}
	if pet == nil {
		return errAttachmentPetNotFound
	}

	client, err := c.clientRepo.FindByID(r.Context(), pet.OwnerID)
	if err != nil {
		return errAttachmentDB
	}
	if client == nil || client.VetID != userID {
		return errAttachmentForbidden
	}
	return nil
}

var (
	errAttachmentDB          = errValidation("db")
	errAttachmentPetNotFound = errValidation("pet_not_found")
	errAttachmentForbidden   = errValidation("forbidden")
)

func writeAttachmentOwnershipError(w http.ResponseWriter, err error) {
	switch err {
	case errAttachmentPetNotFound:
		writeErrorWithDetails(w, http.StatusNotFound, "Pet não encontrado", "Nenhum pet com este ID existe no banco de dados", "PET_NOT_FOUND")
	case errAttachmentForbidden:
		writeError(w, http.StatusForbidden, "acesso negado: você não tem permissão para acessar este recurso")
	default:
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao verificar propriedade do pet", err.Error(), "DB_QUERY_ERROR")
	}
}

// validateAttachment validates required fields
func validateAttachment(attachment *models.Attachment) error {
	if attachment.PetID.IsZero() {
		return errValidation("pet_id is required")
	}
	if attachment.Name == "" {
		return errValidation("name is required")
	}
	if attachment.Content == "" {
		return errValidation("content is required")
	}
	if len(attachment.Content) > maxAttachmentContentBytes {
		return errValidation("content too large (max ~10MB)")
	}
	return nil
}
