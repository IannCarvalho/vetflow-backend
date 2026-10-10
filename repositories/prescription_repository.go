package repositories

import (
	"context"
	"errors"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoPrescriptionRepository struct {
	collection *mongo.Collection
}

// NewPrescriptionRepository creates a new prescription repository
func NewPrescriptionRepository(collection *mongo.Collection) PrescriptionRepository {
	return &mongoPrescriptionRepository{
		collection: collection,
	}
}

// Create inserts a new prescription
func (r *mongoPrescriptionRepository) Create(ctx context.Context, prescription *models.Prescription) error {
	prescription.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, prescription)
	return err
}

// FindByID retrieves a prescription by ID
func (r *mongoPrescriptionRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Prescription, error) {
	prescription := &models.Prescription{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(prescription)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return prescription, nil
}

// FindByVetID retrieves all prescriptions for a vet, most recent first
func (r *mongoPrescriptionRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Prescription, error) {
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"vet_id": vetID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var prescriptions []*models.Prescription
	if err = cursor.All(ctx, &prescriptions); err != nil {
		return nil, err
	}
	return prescriptions, nil
}

// FindByPetID retrieves all prescriptions for a given pet, most recent first
func (r *mongoPrescriptionRepository) FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Prescription, error) {
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"pet_id": petID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var prescriptions []*models.Prescription
	if err = cursor.All(ctx, &prescriptions); err != nil {
		return nil, err
	}
	return prescriptions, nil
}

// Update updates an existing prescription
func (r *mongoPrescriptionRepository) Update(ctx context.Context, id primitive.ObjectID, prescription *models.Prescription) error {
	update := bson.M{
		"$set": prescription,
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

// Delete removes a prescription
func (r *mongoPrescriptionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByPetID removes all prescriptions for a given pet
func (r *mongoPrescriptionRepository) DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"pet_id": petID})
	return err
}
