package repositories

import (
	"context"

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

// NotificationRepository defines operations for notification data
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByVetID(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error)
	GetPending(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteOld(ctx context.Context, vetID primitive.ObjectID) error
}
