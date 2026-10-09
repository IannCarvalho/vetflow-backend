package repositories

import (
	"context"
	"errors"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoPetRepository struct {
	collection *mongo.Collection
}

// NewPetRepository creates a new pet repository
func NewPetRepository(collection *mongo.Collection) PetRepository {
	return &mongoPetRepository{
		collection: collection,
	}
}

// Create inserts a new pet
func (r *mongoPetRepository) Create(ctx context.Context, pet *models.Pet) error {
	pet.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, pet)
	return err
}

// FindByID retrieves a pet by ID
func (r *mongoPetRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Pet, error) {
	pet := &models.Pet{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(pet)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return pet, nil
}

// FindByIDs retrieves all pets whose id is in the given list, in a single
// round-trip ($in). Returns an empty slice when ids is empty (no query sent).
func (r *mongoPetRepository) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*models.Pet, error) {
	if len(ids) == 0 {
		return []*models.Pet{}, nil
	}

	cursor, err := r.collection.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var pets []*models.Pet
	if err = cursor.All(ctx, &pets); err != nil {
		return nil, err
	}
	return pets, nil
}

// FindAll retrieves all pets
func (r *mongoPetRepository) FindAll(ctx context.Context) ([]*models.Pet, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var pets []*models.Pet
	err = cursor.All(ctx, &pets)
	if err != nil {
		return nil, err
	}
	return pets, nil
}

// FindByOwnerID retrieves all pets for a given owner
func (r *mongoPetRepository) FindByOwnerID(ctx context.Context, ownerID primitive.ObjectID) ([]*models.Pet, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"owner_id": ownerID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var pets []*models.Pet
	err = cursor.All(ctx, &pets)
	if err != nil {
		return nil, err
	}
	return pets, nil
}

// FindByVetID retrieves all pets for clients belonging to a specific veterinarian
func (r *mongoPetRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Pet, error) {
	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "clients"},
				{Key: "localField", Value: "owner_id"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "client"},
			}},
		},
		bson.D{
			{Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$client"},
				{Key: "preserveNullAndEmptyArrays", Value: false},
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{"client.vet_id": vetID}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"_id":        1,
				"name":       1,
				"species":    1,
				"breed":      1,
				"age":        1,
				"age_unit":   1,
				"sex":        1,
				"neutered":   1,
				"weight":     1,
				"owner_id":   1,
				"created_at": 1,
				"updated_at": 1,
			}},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var pets []*models.Pet
	err = cursor.All(ctx, &pets)
	if err != nil {
		return nil, err
	}
	return pets, nil
}

// Update updates an existing pet
func (r *mongoPetRepository) Update(ctx context.Context, id primitive.ObjectID, pet *models.Pet) error {
	update := bson.M{
		"$set": pet,
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

// Delete removes a pet
func (r *mongoPetRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByClientID removes all pets for a given client
func (r *mongoPetRepository) DeleteByClientID(ctx context.Context, clientID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"owner_id": clientID})
	return err
}
