package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Vaccine struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID        primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	PetID        primitive.ObjectID `bson:"pet_id" json:"pet_id"`
	VaccineName  string             `bson:"vaccine_name" json:"vaccine_name"`
	Batch        string             `bson:"batch" json:"batch"`
	Manufacturer string             `bson:"manufacturer" json:"manufacturer"`
	DateApplied  time.Time          `bson:"date_applied" json:"date_applied"`
	NextDoseDate time.Time          `bson:"next_dose_date" json:"next_dose_date"`
	Notes        string             `bson:"notes" json:"notes"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}
