package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/internal/ctxkeys"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	Code    string `json:"code,omitempty"`
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

// writeErrorWithDetails writes a JSON error response with additional details
func writeErrorWithDetails(w http.ResponseWriter, statusCode int, message, details, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message, Details: details, Code: code})
}

// parseObjectIDParam reads a chi URL param, parses it as a Mongo ObjectID, and
// on failure writes the standard INVALID_ID_FORMAT 400 response and returns
// ok=false. The label is interpolated into the user-facing message ("Formato de
// "+label+" inválido") and code is the error code, so each call site keeps its
// exact existing message and code (the API contract is unchanged).
func parseObjectIDParam(w http.ResponseWriter, r *http.Request, param, label, code string) (primitive.ObjectID, bool) {
	objectID, err := primitive.ObjectIDFromHex(chi.URLParam(r, param))
	if err != nil {
		writeErrorWithDetails(w, http.StatusBadRequest, "Formato de "+label+" inválido", "Esperado um ObjectID válido do MongoDB", code)
		return primitive.ObjectID{}, false
	}
	return objectID, true
}

// errValidation returns a validation error
func errValidation(message string) error {
	return errors.New(message)
}

// errValidationWithDetails returns a validation error with formatted message.
// The user-facing message is intentionally capitalized ("Campo") for the API
// contract, so the ST1005 (uncapitalized error strings) check is suppressed here.
func errValidationWithDetails(field, reason string) error {
	//nolint:staticcheck // ST1005: user-facing message must stay "Campo '%s': %s"
	return fmt.Errorf("Campo '%s': %s", field, reason)
}

// getUserIDFromContext extracts and validates user ID from request context
func getUserIDFromContext(ctx context.Context) (primitive.ObjectID, error) {
	userIDStr := ctx.Value(ctxkeys.UserID)
	if userIDStr == nil {
		return primitive.ObjectID{}, errors.New("usuário não autenticado")
	}

	userIDHex, ok := userIDStr.(string)
	if !ok {
		return primitive.ObjectID{}, errors.New("ID de usuário inválido")
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return primitive.ObjectID{}, errors.New("formato de ID de usuário inválido")
	}

	return userID, nil
}

// validateResourceOwnership checks if the resource belongs to the authenticated user
func validateResourceOwnership(resourceOwnerID, userID primitive.ObjectID) error {
	if resourceOwnerID != userID {
		return errors.New("acesso negado: você não tem permissão para acessar este recurso")
	}
	return nil
}

// Sentinel errors returned by verifyClientOwnership so callers can map each
// failure mode to the exact HTTP status code and message they already use.
// (The not-found case, for example, is a 400 in pet Create but a 404 in the
// read/update/delete handlers, so the mapping is intentionally left to callers.)
var (
	// errClientNotFound indicates the owning client could not be located
	// (either FindByID returned no document, treated as (nil, nil)).
	errClientNotFound = errors.New("client not found")
	// errClientAccessDenied indicates the client exists but does not belong to
	// the authenticated veterinarian.
	errClientAccessDenied = errors.New("client access denied")
)

// writeClientOwnershipError maps a verifyClientOwnership error to the HTTP
// response used by the read/update/delete style handlers: 404 "Client not
// found", 403 "Access denied", and 500 "Failed to verify client ownership" for
// an underlying repository failure. Handlers whose not-found case differs
// (e.g. pet Create returns 400) should inspect the sentinels directly instead.
func writeClientOwnershipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errClientNotFound):
		writeError(w, http.StatusNotFound, "Client not found")
	case errors.Is(err, errClientAccessDenied):
		writeError(w, http.StatusForbidden, "Access denied")
	default:
		writeError(w, http.StatusInternalServerError, "Failed to verify client ownership")
	}
}

// verifyClientOwnership loads the owning client by ID and enforces that it
// belongs to the authenticated veterinarian. It encapsulates the sequence that
// was previously duplicated across handlers: FindByID -> treat (nil, nil) as
// not-found -> validateResourceOwnership(client.VetID, userID).
//
// The returned error is one of:
//   - a wrapped repository error (DB failure) -> callers should emit 500;
//   - errClientNotFound (via errors.Is) -> callers choose 400 or 404;
//   - errClientAccessDenied (via errors.Is) -> callers emit 403.
//
// On success it returns the loaded client and a nil error.
func verifyClientOwnership(ctx context.Context, clientRepo repositories.ClientRepository, clientID, userID primitive.ObjectID) (*models.Client, error) {
	client, err := clientRepo.FindByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify client ownership: %w", err)
	}
	if client == nil {
		return nil, errClientNotFound
	}
	if err := validateResourceOwnership(client.VetID, userID); err != nil {
		return nil, errClientAccessDenied
	}
	return client, nil
}
