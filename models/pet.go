package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Pet struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	Species   string             `bson:"species" json:"species"`
	Breed     string             `bson:"breed" json:"breed"`
	Age       int                `bson:"age" json:"age"`
	AgeUnit   string             `bson:"age_unit" json:"age_unit"`
	Sex       string             `bson:"sex" json:"sex"`
	Neutered  bool               `bson:"neutered" json:"neutered"`
	Weight    float64            `bson:"weight" json:"weight"`
	OwnerID   primitive.ObjectID `bson:"owner_id" json:"owner_id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
