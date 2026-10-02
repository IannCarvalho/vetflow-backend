package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// AuthController handles authentication operations
type AuthController struct {
	userRepo repositories.UserRepository
}

// NewAuthController creates a new auth controller
func NewAuthController(userRepo repositories.UserRepository) *AuthController {
	return &AuthController{userRepo: userRepo}
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Register Request"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /auth/register [post]
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create user model
	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: helpers.HashPassword(req.Password),
	}

	// Save to database
	ctx := r.Context()
	if err := c.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrEmailAlreadyRegistered) {
			writeError(w, http.StatusConflict, "Email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Issue a signed JWT for the newly created user.
	token, err := helpers.GenerateToken(user.ID.Hex())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Create auth response (password should not be exposed)
	response := models.AuthResponse{
		User:  user,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// Login godoc
// @Summary Login user
// @Description Authenticate user with email and password
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login Request"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Find user by email
	ctx := r.Context()
	user, err := c.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to find user")
		return
	}

	if user == nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Check if user has password (social login users won't have one)
	if user.Password == "" {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Verify password
	if !helpers.VerifyPassword(user.Password, req.Password) {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Issue a signed JWT for the authenticated user.
	token, err := helpers.GenerateToken(user.ID.Hex())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Create auth response
	response := models.AuthResponse{
		User:  user,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Verify godoc
// @Summary Verify user token
// @Description Verify if the provided token is valid
// @Tags Auth
// @Accept json
// @Produce json
// @Param token path string true "User Token"
// @Success 200 {object} models.User
// @Failure 401 {object} map[string]string
// @Router /auth/verify/{token} [get]
func (c *AuthController) Verify(w http.ResponseWriter, r *http.Request) {
	tokenString := chi.URLParam(r, "token")
	if tokenString == "" {
		writeError(w, http.StatusUnauthorized, "Token is required")
		return
	}

	// Actually validate the JWT signature and expiry.
	claims, err := helpers.ParseToken(tokenString)
	if err != nil || claims.UserID == "" {
		writeError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// Confirm the user still exists.
	user, err := c.userRepo.FindByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify user")
		return
	}
	if user == nil {
		writeError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
