package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserController handles user-related operations
type UserController struct {
	userRepository repositories.UserRepository
}

// NewUserController creates a new user controller
func NewUserController(userRepository repositories.UserRepository) *UserController {
	return &UserController{
		userRepository: userRepository,
	}
}

// UpdateProfile updates user profile (name and email)
// @Summary Update User Profile
// @Description Update user name and email
// @Tags Users
// @Accept json
// @Produce json
// @Param body body UpdateProfileRequest true "Profile update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/profile [put]
func (c *UserController) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	log.Printf("UpdateProfile: userID=%s", userID)

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// Convert userID string to ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update user
	user, err := c.userRepository.UpdateUser(ctx, objID, map[string]interface{}{
		"name":       req.Name,
		"email":      req.Email,
		"updated_at": time.Now(),
	})

	if err != nil {
		log.Printf("Error updating user: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

// UpdateProfilePicture updates user profile picture
// @Summary Update Profile Picture
// @Description Update user profile picture (base64 encoded)
// @Tags Users
// @Accept json
// @Produce json
// @Param body body UpdateProfilePictureRequest true "Profile picture update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/profile-picture [put]
func (c *UserController) UpdateProfilePicture(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		ProfilePicture string `json:"profile_picture"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ProfilePicture == "" {
		writeError(w, http.StatusBadRequest, "Profile picture is required")
		return
	}

	// Validate base64 length (limit to ~5MB encoded)
	if len(req.ProfilePicture) > 6*1024*1024 {
		writeError(w, http.StatusBadRequest, "Profile picture too large")
		return
	}

	// Convert userID string to ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update user
	user, err := c.userRepository.UpdateUser(ctx, objID, map[string]interface{}{
		"profile_picture": req.ProfilePicture,
		"updated_at":      time.Now(),
	})

	if err != nil {
		log.Printf("Error updating profile picture: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update profile picture")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

// RemoveProfilePicture removes user profile picture
// @Summary Remove Profile Picture
// @Description Remove user profile picture
// @Tags Users
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /users/profile-picture [delete]
func (c *UserController) RemoveProfilePicture(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Convert userID string to ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update user
	user, err := c.userRepository.UpdateUser(ctx, objID, map[string]interface{}{
		"profile_picture": "",
		"updated_at":      time.Now(),
	})

	if err != nil {
		log.Printf("Error removing profile picture: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to remove profile picture")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}

// ChangePassword changes user password
// @Summary Change Password
// @Description Change user password - works for both regular users and social login users without password
// @Tags Users
// @Accept json
// @Produce json
// @Param body body ChangePasswordRequest true "Change password request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/password [put]
func (c *UserController) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "New password is required")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "New password must be at least 8 characters")
		return
	}

	// Convert userID string to ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user to verify current password
	user, err := c.userRepository.FindByObjectID(ctx, objID)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		writeError(w, http.StatusUnauthorized, "Invalid current password")
		return
	}

	if user == nil {
		writeError(w, http.StatusUnauthorized, "Invalid current password")
		return
	}

	// Hash new password
	hashedPassword := helpers.HashPassword(req.NewPassword)

	// Update user password
	_, err = c.userRepository.UpdateUser(ctx, objID, map[string]interface{}{
		"password":   hashedPassword,
		"updated_at": time.Now(),
	})

	if err != nil {
		log.Printf("Error updating password: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to change password")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password changed successfully",
	})
}

// SetPassword sets a password for users without one (e.g., social login users)
// @Summary Set Password
// @Description Set password for users without one - for social login users who want to enable email/password login
// @Tags Users
// @Accept json
// @Produce json
// @Param body body SetPasswordRequest true "Set password request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/password/set [put]
func (c *UserController) SetPassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "Password is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	// Convert userID string to ObjectID
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user
	user, err := c.userRepository.FindByObjectID(ctx, objID)
	if err != nil {
		log.Printf("Error finding user: %v", err)
		writeError(w, http.StatusUnauthorized, "User not found")
		return
	}

	if user == nil {
		writeError(w, http.StatusUnauthorized, "User not found")
		return
	}

	// Check if user already has a password
	if user.Password != "" {
		writeError(w, http.StatusBadRequest, "User already has a password. Use /users/password to change it")
		return
	}

	// Hash password
	hashedPassword := helpers.HashPassword(req.Password)

	// Update user password
	_, err = c.userRepository.UpdateUser(ctx, objID, map[string]interface{}{
		"password":   hashedPassword,
		"updated_at": time.Now(),
	})

	if err != nil {
		log.Printf("Error setting password: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to set password")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password set successfully",
	})
}

// UpdateProfileRequest is the request body for updating user profile
type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UpdateProfilePictureRequest is the request body for updating profile picture
type UpdateProfilePictureRequest struct {
	ProfilePicture string `json:"profile_picture"`
}
