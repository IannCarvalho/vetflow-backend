package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
)

func TestAuthRegisterSuccess(t *testing.T) {
	repo := newFakeUserRepo()
	ctrl := NewAuthController(repo, openPolicy(repo))

	body := models.RegisterRequest{
		Name:            "Ada Lovelace",
		Email:           "ada@example.com",
		Password:        "secret123",
		ConfirmPassword: "secret123",
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/register", mustJSON(t, body))
	rec := httptest.NewRecorder()

	ctrl.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if _, ok := repo.byEmail[body.Email]; !ok {
		t.Fatalf("user %q was not stored", body.Email)
	}
}

func TestAuthRegisterDuplicateReturns409(t *testing.T) {
	repo := newFakeUserRepo()
	ctrl := NewAuthController(repo, openPolicy(repo))

	body := models.RegisterRequest{
		Name:            "Ada Lovelace",
		Email:           "ada@example.com",
		Password:        "secret123",
		ConfirmPassword: "secret123",
	}

	first := httptest.NewRequest(http.MethodPost, "/auth/register", mustJSON(t, body))
	firstRec := httptest.NewRecorder()
	ctrl.Register(firstRec, first)
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("first register status = %d, want %d (body: %s)", firstRec.Code, http.StatusCreated, firstRec.Body.String())
	}

	second := httptest.NewRequest(http.MethodPost, "/auth/register", mustJSON(t, body))
	secondRec := httptest.NewRecorder()
	ctrl.Register(secondRec, second)
	if secondRec.Code != http.StatusConflict {
		t.Fatalf("duplicate register status = %d, want %d (body: %s)", secondRec.Code, http.StatusConflict, secondRec.Body.String())
	}
}

func TestAuthLoginSuccess(t *testing.T) {
	repo := newFakeUserRepo()
	ctrl := NewAuthController(repo, openPolicy(repo))

	if err := repo.Create(context.Background(), &models.User{
		Name:     "Ada Lovelace",
		Email:    "ada@example.com",
		Password: helpers.HashPassword("secret123"),
	}); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	body := models.LoginRequest{Email: "ada@example.com", Password: "secret123"}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", mustJSON(t, body))
	rec := httptest.NewRecorder()

	ctrl.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp models.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode AuthResponse: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("expected non-empty token, got empty")
	}
}

func TestAuthLoginWrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	ctrl := NewAuthController(repo, openPolicy(repo))

	if err := repo.Create(context.Background(), &models.User{
		Name:     "Ada Lovelace",
		Email:    "ada@example.com",
		Password: helpers.HashPassword("secret123"),
	}); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	body := models.LoginRequest{Email: "ada@example.com", Password: "wrongpass"}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", mustJSON(t, body))
	rec := httptest.NewRecorder()

	ctrl.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestAuthLoginUnknownEmail(t *testing.T) {
	repo := newFakeUserRepo()
	ctrl := NewAuthController(repo, openPolicy(repo))

	body := models.LoginRequest{Email: "nobody@example.com", Password: "secret123"}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", mustJSON(t, body))
	rec := httptest.NewRecorder()

	ctrl.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}
