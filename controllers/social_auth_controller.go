package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// SocialAuthController handles social authentication operations
type SocialAuthController struct {
	userRepo repositories.UserRepository
}

// NewSocialAuthController creates a new social auth controller
func NewSocialAuthController(userRepo repositories.UserRepository) *SocialAuthController {
	return &SocialAuthController{
		userRepo: userRepo,
	}
}

// SocialLogin godoc
// @Summary Login with social provider
// @Description Authenticate user with Google, Facebook, or Discord
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body models.SocialLoginRequest true "Social Login Request"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/social-login [post]
func (c *SocialAuthController) SocialLogin(w http.ResponseWriter, r *http.Request) {
	var req models.SocialLoginRequest

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("Social login request: provider=%s, userData=%+v", req.Provider, req.UserData)

	// Extract user data from request
	userData := req.UserData
	if userData == nil {
		writeError(w, http.StatusBadRequest, "User data not provided")
		return
	}

	// Extract email from provider data
	email := extractString(userData, "email")
	if email == "" {
		log.Printf("Email is empty! Full userData: %+v", userData)
		writeError(w, http.StatusBadRequest, "Email not available from provider")
		return
	}

	name := extractString(userData, "name")
	picture := extractString(userData, "picture")

	log.Printf("Extracted from provider: email=%s, name=%s, picture_present=%v", email, name, picture != "")

	ctx := r.Context()

	// Check if user exists by email
	user, err := c.userRepo.FindByEmail(ctx, email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check user")
		return
	}

	// If user doesn't exist, create a new one
	if user == nil {
		log.Printf("Creating new user from social provider: email=%s, name=%s", email, name)

		// Download and convert profile picture to base64 if available
		var profilePicture string
		if picture != "" {
			profilePicture, err = helpers.DownloadImageAsBase64(picture)
			if err != nil {
				log.Printf("Warning: Failed to download profile picture: %v", err)
				// Continue anyway - picture is optional
			}
		}

		// Create new user with data from social provider
		user = &models.User{
			Name:           name,
			Email:          email,
			Password:       "", // Social users have no password
			ProfilePicture: profilePicture,
		}

		if err := c.userRepo.Create(ctx, user); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		log.Printf("User created successfully: id=%s, email=%s", user.ID.Hex(), email)
	} else {
		log.Printf("User already exists: id=%s, email=%s", user.ID.Hex(), email)
	}

	// Create auth response
	response := models.AuthResponse{
		User:  user,
		Token: "dummy_token_123", // TODO: Replace with actual JWT token
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper functions
func extractString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
