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

type InvoiceController struct {
	repo repositories.InvoiceRepository
}

// NewInvoiceController creates a new invoice controller with dependency injection
func NewInvoiceController(repo repositories.InvoiceRepository) *InvoiceController {
	return &InvoiceController{
		repo: repo,
	}
}

// Create godoc
// @Summary Create a new invoice
// @Description Create an invoice (financeiro) for a client
// @Tags Invoices
// @Accept json
// @Produce json
// @Param invoice body models.Invoice true "Invoice object"
// @Success 201 {object} models.Invoice
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /invoices [post]
func (c *InvoiceController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var invoice models.Invoice
	if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateInvoice(&invoice); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	now := time.Now()
	invoice.VetID = userID
	if invoice.IssueDate.IsZero() {
		invoice.IssueDate = now
	}
	invoice.Status = normalizePaymentStatus(invoice.Status)
	recomputeInvoiceTotals(&invoice)
	syncPaidAt(&invoice, now)
	invoice.CreatedAt = now
	invoice.UpdatedAt = now

	if err := c.repo.Create(r.Context(), &invoice); err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao criar fatura no banco de dados", err.Error(), "DB_CREATE_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(invoice)
}

// GetAll godoc
// @Summary List invoices for the authenticated veterinarian
// @Description List invoices, optionally filtered by client_id
// @Tags Invoices
// @Accept json
// @Produce json
// @Param client_id query string false "Client ID to filter invoices"
// @Success 200 {array} models.Invoice
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /invoices [get]
func (c *InvoiceController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if clientID := r.URL.Query().Get("client_id"); clientID != "" {
		objectID, err := primitive.ObjectIDFromHex(clientID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusBadRequest, "Formato de client_id inválido", "Esperado um ObjectID válido do MongoDB", "INVALID_CLIENT_ID_FORMAT")
			return
		}

		invoices, err := c.repo.FindByClientID(r.Context(), objectID)
		if err != nil {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar faturas", err.Error(), "DB_QUERY_ERROR")
			return
		}

		invoices = filterOwnedInvoices(invoices, userID)
		writeInvoices(w, invoices)
		return
	}

	invoices, err := c.repo.FindByVetID(r.Context(), userID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar faturas", err.Error(), "DB_QUERY_ERROR")
		return
	}
	writeInvoices(w, invoices)
}

// GetByID godoc
// @Summary Get invoice by ID
// @Description Get a specific invoice by ID (must belong to the authenticated vet)
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} models.Invoice
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Invoice not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /invoices/{id} [get]
func (c *InvoiceController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID de fatura", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	invoice, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar fatura", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if invoice == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Fatura não encontrada", "Nenhuma fatura com este ID existe no banco de dados", "INVOICE_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(invoice.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(invoice)
}

// Update godoc
// @Summary Update an invoice
// @Description Update an existing invoice (must belong to the authenticated vet)
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param invoice body models.Invoice true "Updated invoice object"
// @Success 200 {object} models.Invoice
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 404 {object} map[string]string "Invoice not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /invoices/{id} [put]
func (c *InvoiceController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID de fatura", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar fatura", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Fatura não encontrada", "Nenhuma fatura com este ID existe no banco de dados", "INVOICE_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var invoice models.Invoice
	if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Falha ao decodificar corpo da requisição", err.Error(), "INVALID_JSON")
		return
	}

	if err := validateInvoice(&invoice); err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Validação falhou", err.Error(), "VALIDATION_ERROR")
		return
	}

	now := time.Now()
	invoice.ID = objectID
	invoice.VetID = userID
	if invoice.IssueDate.IsZero() {
		invoice.IssueDate = existing.IssueDate
	}
	invoice.Status = normalizePaymentStatus(invoice.Status)
	recomputeInvoiceTotals(&invoice)
	// Preserve original paid timestamp when already paid, else derive from status
	invoice.PaidAt = existing.PaidAt
	syncPaidAt(&invoice, now)
	invoice.CreatedAt = existing.CreatedAt
	invoice.UpdatedAt = now

	if err := c.repo.Update(r.Context(), objectID, &invoice); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Fatura não encontrada", "Nenhuma fatura com este ID existe no banco de dados", "INVOICE_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao atualizar fatura", err.Error(), "DB_UPDATE_ERROR")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(invoice)
}

// Delete godoc
// @Summary Delete an invoice
// @Description Delete an invoice by ID (must belong to the authenticated vet)
// @Tags Invoices
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Invoice not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /invoices/{id} [delete]
func (c *InvoiceController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	objectID, ok := parseObjectIDParam(w, r, "id", "ID de fatura", "INVALID_ID_FORMAT")
	if !ok {
		return
	}

	existing, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao recuperar fatura", err.Error(), "DB_QUERY_ERROR")
		return
	}
	if existing == nil {
		writeErrorWithDetails(w, http.StatusNotFound, "Fatura não encontrada", "Nenhuma fatura com este ID existe no banco de dados", "INVOICE_NOT_FOUND")
		return
	}
	if err := validateResourceOwnership(existing.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if err := c.repo.Delete(r.Context(), objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeErrorWithDetails(w, http.StatusNotFound, "Fatura não encontrada", "Nenhuma fatura com este ID existe no banco de dados", "INVOICE_NOT_FOUND")
		} else {
			writeErrorWithDetails(w, http.StatusInternalServerError, "Falha ao deletar fatura", err.Error(), "DB_DELETE_ERROR")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func writeInvoices(w http.ResponseWriter, invoices []*models.Invoice) {
	if invoices == nil {
		invoices = make([]*models.Invoice, 0)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(invoices)
}

func filterOwnedInvoices(invoices []*models.Invoice, vetID primitive.ObjectID) []*models.Invoice {
	owned := make([]*models.Invoice, 0, len(invoices))
	for _, inv := range invoices {
		if inv.VetID == vetID {
			owned = append(owned, inv)
		}
	}
	return owned
}

func normalizePaymentStatus(status models.PaymentStatus) models.PaymentStatus {
	if status == "" {
		return models.PaymentPending
	}
	return status
}

// recomputeInvoiceTotals derives each item total, the subtotal and the final
// total on the server so amounts cannot be tampered with by the client.
func recomputeInvoiceTotals(invoice *models.Invoice) {
	var subtotal float64
	for i := range invoice.Items {
		item := &invoice.Items[i]
		item.Total = item.Quantity * item.UnitPrice
		subtotal += item.Total
	}
	invoice.Subtotal = subtotal

	discount := invoice.Discount
	if discount < 0 {
		discount = 0
	}
	if discount > subtotal {
		discount = subtotal
	}
	invoice.Discount = discount
	invoice.Total = subtotal - discount
}

// syncPaidAt keeps the paid timestamp consistent with the status.
func syncPaidAt(invoice *models.Invoice, now time.Time) {
	if invoice.Status == models.PaymentPaid {
		if invoice.PaidAt == nil {
			invoice.PaidAt = &now
		}
	} else {
		invoice.PaidAt = nil
	}
}

// validateInvoice validates required fields
func validateInvoice(invoice *models.Invoice) error {
	if invoice.ClientID.IsZero() {
		return errValidation("client_id is required")
	}
	if len(invoice.Items) == 0 {
		return errValidation("at least one invoice item is required")
	}
	for i := range invoice.Items {
		item := invoice.Items[i]
		if item.Description == "" {
			return errValidationWithDetails("items", "description is required")
		}
		if item.Quantity <= 0 {
			return errValidationWithDetails("items", "quantity must be greater than zero")
		}
		if item.UnitPrice < 0 {
			return errValidationWithDetails("items", "unit_price must not be negative")
		}
		_ = i
	}
	return nil
}
