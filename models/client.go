package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Address struct {
	Street     string `bson:"street" json:"street"`
	Number     string `bson:"number" json:"number"`
	Complement string `bson:"complement" json:"complement"`
	ZipCode    string `bson:"zip_code" json:"zip_code"`
}

type Client struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID     primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	FullName  string             `bson:"full_name" json:"full_name"`
	CPF       string             `bson:"cpf" json:"cpf"`
	Phone     string             `bson:"phone" json:"phone"`
	Address   Address            `bson:"address" json:"address"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
