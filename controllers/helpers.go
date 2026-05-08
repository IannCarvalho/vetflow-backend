package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}

// writeErrorWithDetails writes a JSON error response with additional details
func writeErrorWithDetails(w http.ResponseWriter, statusCode int, message, details, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{Error: message, Details: details, Code: code})
}

// errValidation returns a validation error
func errValidation(message string) error {
	return errors.New(message)
}

// errValidationWithDetails returns a validation error with formatted message
func errValidationWithDetails(field, reason string) error {
	return errors.New(fmt.Sprintf("Campo '%s': %s", field, reason))
}

// getUserIDFromContext extracts and validates user ID from request context
func getUserIDFromContext(ctx context.Context) (primitive.ObjectID, error) {
	userIDStr := ctx.Value("userID")
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
