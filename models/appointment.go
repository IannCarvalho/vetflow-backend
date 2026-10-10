package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AppointmentType represents the kind of scheduled appointment
type AppointmentType string

const (
	AppointmentConsultation AppointmentType = "consultation" // Consulta
	AppointmentVaccine      AppointmentType = "vaccine"      // Vacinação
	AppointmentReturn       AppointmentType = "return"       // Retorno
	AppointmentSurgery      AppointmentType = "surgery"      // Cirurgia
	AppointmentExam         AppointmentType = "exam"         // Exame
	AppointmentGrooming     AppointmentType = "grooming"     // Banho e tosa
	AppointmentOther        AppointmentType = "other"        // Outro
)

// AppointmentStatus represents the lifecycle status of an appointment
type AppointmentStatus string

const (
	AppointmentScheduled AppointmentStatus = "scheduled" // Agendado
	AppointmentConfirmed AppointmentStatus = "confirmed" // Confirmado
	AppointmentCompleted AppointmentStatus = "completed" // Concluído
	AppointmentCancelled AppointmentStatus = "cancelled" // Cancelado
	AppointmentNoShow    AppointmentStatus = "no_show"   // Não compareceu
)

// Appointment represents a scheduled visit in the agenda/calendar
type Appointment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID     primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	PetID     primitive.ObjectID `bson:"pet_id" json:"pet_id"`
	ClientID  primitive.ObjectID `bson:"client_id,omitempty" json:"client_id,omitempty"`
	Title     string             `bson:"title" json:"title"`
	Type      AppointmentType    `bson:"type" json:"type"`
	Status    AppointmentStatus  `bson:"status" json:"status"`
	StartTime time.Time          `bson:"start_time" json:"start_time"`
	EndTime   time.Time          `bson:"end_time" json:"end_time"`
	Notes     string             `bson:"notes" json:"notes"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
