package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationType string

const (
	NotificationConsultation NotificationType = "consultation"
	NotificationVaccine      NotificationType = "vaccine"
)

type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusDismissed NotificationStatus = "dismissed"
)

type Notification struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	PetID primitive.ObjectID `bson:"pet_id" json:"pet_id"`
	// ResourceID is the hex id of the consultation/vaccine that originated this
	// notification. Persisted so a sweep can detect an existing notification for
	// the same resource/type/target date and avoid creating duplicates.
	ResourceID string             `bson:"resource_id,omitempty" json:"resource_id,omitempty"`
	Type       NotificationType   `bson:"type" json:"type"`
	Title      string             `bson:"title" json:"title"`
	Message    string             `bson:"message" json:"message"`
	DaysUntil  int                `bson:"days_until" json:"days_until"`
	TargetDate time.Time          `bson:"target_date" json:"target_date"`
	Status     NotificationStatus `bson:"status" json:"status"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	SentAt     *time.Time         `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
}
