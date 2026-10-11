package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a system user for authentication
type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name           string             `bson:"name" json:"name"`
	Email          string             `bson:"email" json:"email"`
	Password       string             `bson:"password" json:"-"`                      // Never expose password
	ProfilePicture string             `bson:"profile_picture" json:"profile_picture"` // Base64 encoded image
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// RegisterRequest is the request body for user registration
type RegisterRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// LoginRequest is the request body for user login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is the response for login and register
type AuthResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

// SocialLoginRequest is the request body for social authentication
type SocialLoginRequest struct {
	Provider string         `json:"provider"`
	Token    string         `json:"token,omitempty"`
	UserData map[string]any `json:"userData"`
}

// Validate performs validation on RegisterRequest
func (r *RegisterRequest) Validate() error {
	if r.Name == "" {
		return NewValidationError("name is required")
	}
	if r.Email == "" {
		return NewValidationError("email is required")
	}
	if r.Password == "" {
		return NewValidationError("password is required")
	}
	if len(r.Password) < 6 {
		return NewValidationError("password must be at least 6 characters")
	}
	if r.Password != r.ConfirmPassword {
		return NewValidationError("passwords do not match")
	}
	return nil
}

// Validate performs validation on LoginRequest
func (l *LoginRequest) Validate() error {
	if l.Email == "" {
		return NewValidationError("email is required")
	}
	if l.Password == "" {
		return NewValidationError("password is required")
	}
	return nil
}
