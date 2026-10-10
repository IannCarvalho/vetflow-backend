package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AttachmentCategory classifies an attachment in the pet's medical record
type AttachmentCategory string

const (
	AttachmentExam     AttachmentCategory = "exam"     // Resultado de exame
	AttachmentImage    AttachmentCategory = "image"    // Imagem (raio-x, ultrassom, foto)
	AttachmentReport   AttachmentCategory = "report"   // Laudo
	AttachmentDocument AttachmentCategory = "document" // Documento geral
)

// Attachment represents a file (exam, image, report...) attached to a pet's record
type Attachment struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID          primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	PetID          primitive.ObjectID `bson:"pet_id" json:"pet_id"`
	ConsultationID primitive.ObjectID `bson:"consultation_id,omitempty" json:"consultation_id,omitempty"`
	Name           string             `bson:"name" json:"name"`
	Category       AttachmentCategory `bson:"category" json:"category"`
	MimeType       string             `bson:"mime_type" json:"mime_type"`
	// Content is the base64-encoded file payload (data URL body).
	Content   string    `bson:"content" json:"content"`
	Notes     string    `bson:"notes" json:"notes"`
	Date      time.Time `bson:"date" json:"date"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// AttachmentSummary is a lightweight view without the heavy base64 content,
// used for listing attachments without transferring every payload.
type AttachmentSummary struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID          primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	PetID          primitive.ObjectID `bson:"pet_id" json:"pet_id"`
	ConsultationID primitive.ObjectID `bson:"consultation_id,omitempty" json:"consultation_id,omitempty"`
	Name           string             `bson:"name" json:"name"`
	Category       AttachmentCategory `bson:"category" json:"category"`
	MimeType       string             `bson:"mime_type" json:"mime_type"`
	Notes          string             `bson:"notes" json:"notes"`
	Date           time.Time          `bson:"date" json:"date"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}
