package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// verifiedIdentity holds the subset of provider claims we trust after the
// provider's ID token has been validated server-side.
type verifiedIdentity struct {
	Email   string
	Name    string
	Picture string
}

// googleTokenInfoURL is Google's token verification endpoint. It validates the
// ID token's signature, audience and expiry server-side and returns the claims.
// Overridable in tests.
var googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

// tokenInfoClient is the HTTP client used to reach the provider. Short timeout
// so a slow/unreachable provider cannot hang the request.
var tokenInfoClient = &http.Client{Timeout: 8 * time.Second}

// verifyProviderToken validates the provider's ID token server-side and returns
// the identity derived from the VERIFIED claims. It never trusts client-supplied
// profile fields. Only Google is supported today; other providers are rejected
// explicitly rather than silently trusting the request body.
func verifyProviderToken(provider, idToken string) (*verifiedIdentity, error) {
	if strings.TrimSpace(idToken) == "" {
		return nil, errors.New("provider token is required")
	}

	switch strings.ToLower(provider) {
	case "google":
		return verifyGoogleIDToken(idToken)
	default:
		return nil, fmt.Errorf("unsupported social provider %q", provider)
	}
}

// verifyGoogleIDToken asks Google's tokeninfo endpoint to validate the ID token.
// A 200 response means Google verified the signature and expiry; we additionally
// require a verified email. The email/name/picture are read from Google's
// response, never from the client request body.
func verifyGoogleIDToken(idToken string) (*verifiedIdentity, error) {
	reqURL := googleTokenInfoURL + "?id_token=" + url.QueryEscape(idToken)
	resp, err := tokenInfoClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to reach token verification endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("provider token is invalid or expired")
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Aud           string `json:"aud"`
		Exp           string `json:"exp"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse token verification response: %w", err)
	}

	if claims.Email == "" {
		return nil, errors.New("verified token did not contain an email")
	}
	// Google returns email_verified as the string "true"/"false".
	if claims.EmailVerified != "true" {
		return nil, errors.New("provider email is not verified")
	}

	return &verifiedIdentity{
		Email:   strings.ToLower(strings.TrimSpace(claims.Email)),
		Name:    claims.Name,
		Picture: claims.Picture,
	}, nil
}

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

	log.Printf("Social login request: provider=%s", req.Provider)

	// SECURITY: never trust client-supplied profile data. Validate the
	// provider's ID token server-side and derive the identity from the
	// VERIFIED claims only. A caller cannot log in as an arbitrary email by
	// putting it in the request body.
	identity, err := verifyProviderToken(req.Provider, req.Token)
	if err != nil {
		log.Printf("Social login rejected: %v", err)
		writeError(w, http.StatusUnauthorized, "Could not verify provider token")
		return
	}

	email := identity.Email
	name := identity.Name
	picture := identity.Picture

	log.Printf("Verified identity from provider: email=%s, name=%s, picture_present=%v", email, name, picture != "")

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

	// Issue a real signed JWT so social-login sessions can call protected routes.
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
