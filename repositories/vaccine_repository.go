package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoVaccineRepository struct {
	collection *mongo.Collection
}

// NewVaccineRepository creates a new vaccine repository
func NewVaccineRepository(collection *mongo.Collection) VaccineRepository {
	return &mongoVaccineRepository{
		collection: collection,
	}
}

// Create inserts a new vaccine
func (r *mongoVaccineRepository) Create(ctx context.Context, vaccine *models.Vaccine) error {
	vaccine.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, vaccine)
	return err
}

// FindByID retrieves a vaccine by ID
func (r *mongoVaccineRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Vaccine, error) {
	vaccine := &models.Vaccine{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(vaccine)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return vaccine, nil
}

// FindAll retrieves all vaccines
func (r *mongoVaccineRepository) FindAll(ctx context.Context) ([]*models.Vaccine, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var vaccines []*models.Vaccine
	err = cursor.All(ctx, &vaccines)
	if err != nil {
		return nil, err
	}
	return vaccines, nil
}

// FindByPetID retrieves all vaccines for a given pet
func (r *mongoVaccineRepository) FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Vaccine, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"pet_id": petID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var vaccines []*models.Vaccine
	err = cursor.All(ctx, &vaccines)
	if err != nil {
		return nil, err
	}
	return vaccines, nil
}

// FindByVetID retrieves all vaccines for a specific veterinarian
func (r *mongoVaccineRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Vaccine, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"vet_id": vetID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var vaccines []*models.Vaccine
	if err = cursor.All(ctx, &vaccines); err != nil {
		return nil, err
	}

	return vaccines, nil
}

// CountByVetIDNextDoseInRange counts vaccines for a vet whose next dose date falls
// in the window [start, end]. Unlike the consultation/appointment ranges (which
// are half-open [start, end) via $lt to match FindByVetIDInRange), the vaccine
// window is INCLUSIVE on both ends ($gte start, $lte end). This preserves the
// original dashboard semantics (!Before(now) && !After(windowEnd)) byte-for-byte.
func (r *mongoVaccineRepository) CountByVetIDNextDoseInRange(ctx context.Context, vetID primitive.ObjectID, start, end time.Time) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"vet_id": vetID,
		"next_dose_date": bson.M{
			"$gte": start,
			"$lte": end,
		},
	})
}

// Update updates an existing vaccine
func (r *mongoVaccineRepository) Update(ctx context.Context, id primitive.ObjectID, vaccine *models.Vaccine) error {
	update := bson.M{
		"$set": vaccine,
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

// Delete removes a vaccine
func (r *mongoVaccineRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByPetID removes all vaccines for a given pet
func (r *mongoVaccineRepository) DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"pet_id": petID})
	return err
}
