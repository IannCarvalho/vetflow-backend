package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vetflow/veterinary-api/models"
)

func TestAuthRegisterSuccess(t *testing.T) {
	repo := newFakeUserRepo()
	ctrl := NewAuthController(repo)

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
	ctrl := NewAuthController(repo)

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
