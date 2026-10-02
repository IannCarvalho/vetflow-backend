package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vetflow/veterinary-api/internal/ctxkeys"
	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, http.StatusBadRequest, "something went wrong")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.Error != "something went wrong" {
		t.Errorf("Error = %q, want %q", body.Error, "something went wrong")
	}
	if body.Details != "" || body.Code != "" {
		t.Errorf("expected empty Details/Code, got Details=%q Code=%q", body.Details, body.Code)
	}
}

func TestWriteErrorWithDetails(t *testing.T) {
	rec := httptest.NewRecorder()

	writeErrorWithDetails(rec, http.StatusUnprocessableEntity, "invalid", "field x is empty", "ERR_X")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.Error != "invalid" {
		t.Errorf("Error = %q, want %q", body.Error, "invalid")
	}
	if body.Details != "field x is empty" {
		t.Errorf("Details = %q, want %q", body.Details, "field x is empty")
	}
	if body.Code != "ERR_X" {
		t.Errorf("Code = %q, want %q", body.Code, "ERR_X")
	}
}

func TestErrValidation(t *testing.T) {
	err := errValidation("bad input")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err.Error() != "bad input" {
		t.Errorf("Error() = %q, want %q", err.Error(), "bad input")
	}
}

func TestErrValidationWithDetails(t *testing.T) {
	err := errValidationWithDetails("email", "formato inválido")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	want := "Campo 'email': formato inválido"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	validHex := primitive.NewObjectID().Hex()

	tests := []struct {
		name      string
		ctxValue  interface{}
		setKey    bool
		wantErr   string
		wantIsSet bool
	}{
		{
			name:      "valid object id string",
			ctxValue:  validHex,
			setKey:    true,
			wantIsSet: true,
		},
		{
			name:    "missing user id",
			setKey:  false,
			wantErr: "usuário não autenticado",
		},
		{
			name:     "wrong type in context",
			ctxValue: 12345,
			setKey:   true,
			wantErr:  "ID de usuário inválido",
		},
		{
			name:     "invalid hex string",
			ctxValue: "not-a-valid-hex",
			setKey:   true,
			wantErr:  "formato de ID de usuário inválido",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.setKey {
				ctx = context.WithValue(ctx, ctxkeys.UserID, tt.ctxValue)
			}

			id, err := getUserIDFromContext(ctx)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantIsSet && id.Hex() != validHex {
				t.Fatalf("id = %q, want %q", id.Hex(), validHex)
			}
		})
	}
}

func TestValidateResourceOwnership(t *testing.T) {
	owner := primitive.NewObjectID()
	other := primitive.NewObjectID()

	if err := validateResourceOwnership(owner, owner); err != nil {
		t.Errorf("expected no error for matching IDs, got %v", err)
	}

	if err := validateResourceOwnership(owner, other); err == nil {
		t.Error("expected an error for mismatched IDs, got nil")
	}
}

func TestValidateClient(t *testing.T) {
	tests := []struct {
		name    string
		client  models.Client
		wantErr bool
	}{
		{
			name:    "valid client",
			client:  models.Client{FullName: "John Doe", CPF: "12345678900", Phone: "5511999999999"},
			wantErr: false,
		},
		{name: "missing full name", client: models.Client{CPF: "123", Phone: "456"}, wantErr: true},
		{name: "missing cpf", client: models.Client{FullName: "John", Phone: "456"}, wantErr: true},
		{name: "missing phone", client: models.Client{FullName: "John", CPF: "123"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateClient(&tt.client)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePet(t *testing.T) {
	ownerID := primitive.NewObjectID()

	tests := []struct {
		name    string
		pet     models.Pet
		wantErr bool
	}{
		{
			name:    "valid pet",
			pet:     models.Pet{Name: "Rex", Species: "dog", OwnerID: ownerID},
			wantErr: false,
		},
		{name: "missing name", pet: models.Pet{Species: "dog", OwnerID: ownerID}, wantErr: true},
		{name: "missing species", pet: models.Pet{Name: "Rex", OwnerID: ownerID}, wantErr: true},
		{name: "missing owner id", pet: models.Pet{Name: "Rex", Species: "dog"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePet(&tt.pet)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validatePet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConsultation(t *testing.T) {
	petID := primitive.NewObjectID()

	if err := validateConsultation(&models.Consultation{PetID: petID}); err != nil {
		t.Errorf("expected no error for valid consultation, got %v", err)
	}
	if err := validateConsultation(&models.Consultation{}); err == nil {
		t.Error("expected error for consultation without pet_id, got nil")
	}
}

func TestValidateVaccine(t *testing.T) {
	petID := primitive.NewObjectID()

	tests := []struct {
		name    string
		vaccine models.Vaccine
		wantErr bool
	}{
		{
			name:    "valid vaccine",
			vaccine: models.Vaccine{PetID: petID, VaccineName: "Rabies"},
			wantErr: false,
		},
		{name: "missing pet id", vaccine: models.Vaccine{VaccineName: "Rabies"}, wantErr: true},
		{name: "missing vaccine name", vaccine: models.Vaccine{PetID: petID}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVaccine(&tt.vaccine)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateVaccine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// healthHandler mirrors the /health endpoint registered in routes.NewRouter,
// allowing it to be exercised in isolation without a Mongo connection.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", body, `{"status":"ok"}`)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if parsed["status"] != "ok" {
		t.Errorf("status field = %q, want %q", parsed["status"], "ok")
	}
}
