package repositories

import (
	"context"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ClientRepository defines operations for client data
type ClientRepository interface {
	Create(ctx context.Context, client *models.Client) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Client, error)
	FindAll(ctx context.Context) ([]*models.Client, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Client, error)
	Update(ctx context.Context, id primitive.ObjectID, client *models.Client) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// PetRepository defines operations for pet data
type PetRepository interface {
	Create(ctx context.Context, pet *models.Pet) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Pet, error)
	FindAll(ctx context.Context) ([]*models.Pet, error)
	FindByOwnerID(ctx context.Context, ownerID primitive.ObjectID) ([]*models.Pet, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Pet, error)
	Update(ctx context.Context, id primitive.ObjectID, pet *models.Pet) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByClientID(ctx context.Context, clientID primitive.ObjectID) error
}

// ConsultationRepository defines operations for consultation data
type ConsultationRepository interface {
	Create(ctx context.Context, consultation *models.Consultation) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Consultation, error)
	FindAll(ctx context.Context) ([]*models.Consultation, error)
	FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Consultation, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Consultation, error)
	Update(ctx context.Context, id primitive.ObjectID, consultation *models.Consultation) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error
}

// VaccineRepository defines operations for vaccine data
type VaccineRepository interface {
	Create(ctx context.Context, vaccine *models.Vaccine) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Vaccine, error)
	FindAll(ctx context.Context) ([]*models.Vaccine, error)
	FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Vaccine, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Vaccine, error)
	Update(ctx context.Context, id primitive.ObjectID, vaccine *models.Vaccine) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error
}

// PrescriptionRepository defines operations for prescription (receituário) data
type PrescriptionRepository interface {
	Create(ctx context.Context, prescription *models.Prescription) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Prescription, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Prescription, error)
	FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Prescription, error)
	Update(ctx context.Context, id primitive.ObjectID, prescription *models.Prescription) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error
}

// CatalogItemRepository defines operations for catalog (services/products) data
type CatalogItemRepository interface {
	Create(ctx context.Context, item *models.CatalogItem) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.CatalogItem, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.CatalogItem, error)
	Update(ctx context.Context, id primitive.ObjectID, item *models.CatalogItem) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// InvoiceRepository defines operations for invoice (financeiro) data
type InvoiceRepository interface {
	Create(ctx context.Context, invoice *models.Invoice) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Invoice, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Invoice, error)
	FindByClientID(ctx context.Context, clientID primitive.ObjectID) ([]*models.Invoice, error)
	Update(ctx context.Context, id primitive.ObjectID, invoice *models.Invoice) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByClientID(ctx context.Context, clientID primitive.ObjectID) error
}

// AppointmentRepository defines operations for appointment (agenda) data
type AppointmentRepository interface {
	Create(ctx context.Context, appointment *models.Appointment) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Appointment, error)
	FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Appointment, error)
	FindByVetIDInRange(ctx context.Context, vetID primitive.ObjectID, start, end time.Time) ([]*models.Appointment, error)
	FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Appointment, error)
	Update(ctx context.Context, id primitive.ObjectID, appointment *models.Appointment) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error
	DeleteByClientID(ctx context.Context, clientID primitive.ObjectID) error
}

// NotificationRepository defines operations for notification data
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByVetID(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error)
	GetPending(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteOld(ctx context.Context, vetID primitive.ObjectID) error
}
