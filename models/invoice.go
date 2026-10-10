package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PaymentStatus represents the payment lifecycle of an invoice
type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "pending"  // Pendente
	PaymentPaid     PaymentStatus = "paid"     // Pago
	PaymentPartial  PaymentStatus = "partial"  // Parcial
	PaymentOverdue  PaymentStatus = "overdue"  // Vencido
	PaymentCanceled PaymentStatus = "canceled" // Cancelado
)

// PaymentMethod represents how an invoice was paid
type PaymentMethod string

const (
	MethodCash       PaymentMethod = "cash"     // Dinheiro
	MethodCard       PaymentMethod = "card"     // Cartão
	MethodPix        PaymentMethod = "pix"      // Pix
	MethodTransfer   PaymentMethod = "transfer" // Transferência
	MethodBoleto     PaymentMethod = "boleto"   // Boleto
	MethodUnassigned PaymentMethod = ""         // Não informado
)

// InvoiceItem represents a single billable line in an invoice
type InvoiceItem struct {
	Description string  `bson:"description" json:"description"`
	Quantity    float64 `bson:"quantity" json:"quantity"`
	UnitPrice   float64 `bson:"unit_price" json:"unit_price"`
	Total       float64 `bson:"total" json:"total"`
}

// Invoice represents a billing document for a service rendered to a client
type Invoice struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID          primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	ClientID       primitive.ObjectID `bson:"client_id" json:"client_id"`
	PetID          primitive.ObjectID `bson:"pet_id,omitempty" json:"pet_id,omitempty"`
	ConsultationID primitive.ObjectID `bson:"consultation_id,omitempty" json:"consultation_id,omitempty"`
	Items          []InvoiceItem      `bson:"items" json:"items"`
	Subtotal       float64            `bson:"subtotal" json:"subtotal"`
	Discount       float64            `bson:"discount" json:"discount"`
	Total          float64            `bson:"total" json:"total"`
	Status         PaymentStatus      `bson:"status" json:"status"`
	Method         PaymentMethod      `bson:"method,omitempty" json:"method,omitempty"`
	IssueDate      time.Time          `bson:"issue_date" json:"issue_date"`
	DueDate        *time.Time         `bson:"due_date,omitempty" json:"due_date,omitempty"`
	PaidAt         *time.Time         `bson:"paid_at,omitempty" json:"paid_at,omitempty"`
	Notes          string             `bson:"notes" json:"notes"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}
