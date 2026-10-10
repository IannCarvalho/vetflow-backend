package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// OAuthCallbackController handles OAuth callbacks from providers
type OAuthCallbackController struct {
	userRepo repositories.UserRepository
}

// NewOAuthCallbackController creates a new OAuth callback controller
func NewOAuthCallbackController(userRepo repositories.UserRepository) *OAuthCallbackController {
	return &OAuthCallbackController{
		userRepo: userRepo,
	}
}

// OAuthCallback handles OAuth callbacks and exchanges code for token
// @Summary OAuth Callback Handler
// @Description Handle OAuth callback from providers - delegates to social login endpoint
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body models.SocialLoginRequest true "OAuth callback request (provider + userData)"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {object} map[string]string
// @Router /auth/oauth/callback [post]
func (c *OAuthCallbackController) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		Token    string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode OAuth callback request: %v", err)
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("OAuth callback request: provider=%s", req.Provider)

	// SECURITY: validate the provider's ID token server-side and derive the
	// identity from the VERIFIED claims only. Never trust client-supplied
	// profile data, and never issue a hardcoded/dummy token.
	identity, err := verifyProviderToken(req.Provider, req.Token)
	if err != nil {
		log.Printf("OAuth callback rejected: %v", err)
		writeError(w, http.StatusUnauthorized, "Could not verify provider token")
		return
	}

	email := identity.Email
	name := identity.Name
	picture := identity.Picture

	ctx := r.Context()

	// Check if user exists by email
	user, err := c.userRepo.FindByEmail(ctx, email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to check user")
		return
	}

	// If user doesn't exist, create a new one
	if user == nil {
		log.Printf("Creating new user from OAuth callback: email=%s, name=%s", email, name)

		// Download and convert profile picture to base64 if available
		var profilePicture string
		if picture != "" {
			profilePicture, err = helpers.DownloadImageAsBase64(picture)
			if err != nil {
				log.Printf("Warning: Failed to download profile picture: %v", err)
			}
		}

		// Create new user with data from the verified provider claims
		user = &models.User{
			Name:           name,
			Email:          email,
			Password:       "", // OAuth users have no password
			ProfilePicture: profilePicture,
		}

		if err := c.userRepo.Create(ctx, user); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		log.Printf("User created successfully: id=%s, email=%s", user.ID.Hex(), email)
	}

	// Issue a real signed JWT scoped to this user.
	token, err := helpers.GenerateToken(user.ID.Hex())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Return auth response
	response := models.AuthResponse{
		User:  user,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
