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

// resolveFound collapses the FindByID -> 500 -> nil-check -> 404 preamble that
// precedes the ownership guard in every by-ID read/update/delete handler. The
// caller passes the (*T, error) pair a repository FindByID already returned, so
// no model or repository abstraction is touched and Go generics let one helper
// serve *Invoice, *Appointment, *Vaccine, ... alike:
//
//	res, err := c.repo.FindByID(r.Context(), objectID)
//	existing, ok := resolveFound(w, res, err,
//	    "Falha ao recuperar fatura",
//	    "Fatura não encontrada",
//	    "Nenhuma fatura com este ID existe no banco de dados",
//	    "INVOICE_NOT_FOUND")
//	if !ok {
//	    return
//	}
//
// Byte-for-byte preservation is the whole point, so the three per-resource texts
// and the not-found code are parameters: queryErrorMessage is the exact 500
// message that site used ("Falha ao recuperar <recurso>"), and notFoundMessage /
// notFoundDetails / notFoundCode are the exact 404 message, details and code.
// The two invariants of this preamble stay fixed inside the helper and are NOT
// parameters, because they never varied across the ~26 converted sites: the 500
// always carries the underlying err.Error() as its details under the
// "DB_QUERY_ERROR" code. On err != nil it writes that standard 500 and returns
// (nil, false); on resource == nil it writes the 404 and returns (nil, false);
// otherwise it returns (resource, true) and the caller proceeds to its own
// ownership check (ensureOwnershipStrict / validateResourceOwnership),
// which is deliberately left untouched here.
func resolveFound[T any](w http.ResponseWriter, resource *T, err error, queryErrorMessage, notFoundMessage, notFoundDetails, notFoundCode string) (*T, bool) {
	if err != nil {
		writeErrorWithDetails(w, http.StatusInternalServerError, queryErrorMessage, err.Error(), "DB_QUERY_ERROR")
		return nil, false
	}
	if resource == nil {
		writeErrorWithDetails(w, http.StatusNotFound, notFoundMessage, notFoundDetails, notFoundCode)
		return nil, false
	}
	return resource, true
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

// ensureOwnershipStrict enforces that a resource already loaded by the caller
// belongs to the authenticated veterinarian, writing the standard 403 response
// and returning ok=false when it does not. It is the single ownership helper
// used across every VetID-scoped handler (appointment, invoice, prescription,
// attachment, catalog_item, consultation), deduplicating the block they all
// carry verbatim:
//
//	if err := validateResourceOwnership(resource.VetID, userID); err != nil {
//	    writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
//	    return
//	}
//
// Being the single ownership-403 site, it emits the uniform rich error shape —
// error "Acesso negado", details carrying validateResourceOwnership's message
// ("acesso negado: você não tem permissão para acessar este recurso") and code
// "ACCESS_DENIED" — so every VetID-scoped controller returns the same 403 body.
//
// It has NO IsZero() early return: a zero resourceVetID is NOT treated as
// owned. Legacy records with a zero VetID predate vet-scoping, belong to no
// vet, and must not be accessible — a zero resourceVetID fails
// validateResourceOwnership against a non-zero userID and yields a 403,
// uniformly for every resource.
func ensureOwnershipStrict(w http.ResponseWriter, resourceVetID, userID primitive.ObjectID) bool {
	if err := validateResourceOwnership(resourceVetID, userID); err != nil {
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", err.Error(), "ACCESS_DENIED")
		return false
	}
	return true
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
// found", a 403 in the uniform ownership shape (error "Acesso negado",
// details "acesso negado: você não tem permissão para acessar este recurso",
// code "ACCESS_DENIED"), and 500 "Failed to verify client ownership" for an
// underlying repository failure. Handlers whose not-found case differs
// (e.g. pet Create returns 400) should inspect the sentinels directly instead.
func writeClientOwnershipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errClientNotFound):
		writeError(w, http.StatusNotFound, "Client not found")
	case errors.Is(err, errClientAccessDenied):
		writeErrorWithDetails(w, http.StatusForbidden, "Acesso negado", "acesso negado: você não tem permissão para acessar este recurso", "ACCESS_DENIED")
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
