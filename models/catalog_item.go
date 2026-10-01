package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CatalogItemType distinguishes services from stockable products
type CatalogItemType string

const (
	CatalogService CatalogItemType = "service" // Serviço (consulta, banho, cirurgia...)
	CatalogProduct CatalogItemType = "product" // Produto (vacina, medicamento...)
)

// CatalogItem represents a billable service or a stockable product offered by the clinic
type CatalogItem struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID       primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	Type        CatalogItemType    `bson:"type" json:"type"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Price       float64            `bson:"price" json:"price"`
	Unit        string             `bson:"unit" json:"unit"` // ex: "un", "ml", "cx"
	// Stock fields are only meaningful for products.
	StockQuantity float64   `bson:"stock_quantity" json:"stock_quantity"`
	MinStock      float64   `bson:"min_stock" json:"min_stock"`
	Active        bool      `bson:"active" json:"active"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
}
