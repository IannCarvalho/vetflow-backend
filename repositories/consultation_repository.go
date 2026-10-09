package repositories

import (
	"context"
	"errors"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoConsultationRepository struct {
	collection *mongo.Collection
}

// NewConsultationRepository creates a new consultation repository
func NewConsultationRepository(collection *mongo.Collection) ConsultationRepository {
	return &mongoConsultationRepository{
		collection: collection,
	}
}

// Create inserts a new consultation
func (r *mongoConsultationRepository) Create(ctx context.Context, consultation *models.Consultation) error {
	consultation.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, consultation)
	return err
}

// FindByID retrieves a consultation by ID
func (r *mongoConsultationRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Consultation, error) {
	consultation := &models.Consultation{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(consultation)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return consultation, nil
}

// FindAll retrieves all consultations
func (r *mongoConsultationRepository) FindAll(ctx context.Context) ([]*models.Consultation, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var consultations []*models.Consultation
	err = cursor.All(ctx, &consultations)
	if err != nil {
		return nil, err
	}
	return consultations, nil
}

// FindByPetID retrieves all consultations for a given pet
func (r *mongoConsultationRepository) FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Consultation, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"pet_id": petID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var consultations []*models.Consultation
	err = cursor.All(ctx, &consultations)
	if err != nil {
		return nil, err
	}
	return consultations, nil
}

// FindByVetID retrieves all consultations for a specific veterinarian
func (r *mongoConsultationRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Consultation, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"vet_id": vetID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var consultations []*models.Consultation
	if err = cursor.All(ctx, &consultations); err != nil {
		return nil, err
	}

	return consultations, nil
}

// Update updates an existing consultation
func (r *mongoConsultationRepository) Update(ctx context.Context, id primitive.ObjectID, consultation *models.Consultation) error {
	update := bson.M{
		"$set": consultation,
	}
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Delete removes a consultation
func (r *mongoConsultationRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByPetID removes all consultations for a given pet
func (r *mongoConsultationRepository) DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"pet_id": petID})
	return err
}
