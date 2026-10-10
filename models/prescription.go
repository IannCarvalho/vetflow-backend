package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PrescriptionItem represents a single medication line in a prescription
type PrescriptionItem struct {
	Medication   string `bson:"medication" json:"medication"`     // Nome do medicamento
	Dosage       string `bson:"dosage" json:"dosage"`             // Dose (ex: "10mg")
	Route        string `bson:"route" json:"route"`               // Via de administração (oral, tópica...)
	Frequency    string `bson:"frequency" json:"frequency"`       // Frequência (ex: "2x ao dia")
	Duration     string `bson:"duration" json:"duration"`         // Duração (ex: "7 dias")
	Instructions string `bson:"instructions" json:"instructions"` // Instruções adicionais
}

// Prescription represents a veterinary prescription (receituário) for a pet
type Prescription struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID          primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	PetID          primitive.ObjectID `bson:"pet_id" json:"pet_id"`
	ConsultationID primitive.ObjectID `bson:"consultation_id,omitempty" json:"consultation_id,omitempty"`
	Date           time.Time          `bson:"date" json:"date"`
	Items          []PrescriptionItem `bson:"items" json:"items"`
	GeneralNotes   string             `bson:"general_notes" json:"general_notes"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}
