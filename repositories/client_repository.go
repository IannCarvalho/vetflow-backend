package repositories

import (
	"context"
	"errors"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoClientRepository struct {
	collection *mongo.Collection
}

// NewClientRepository creates a new client repository
func NewClientRepository(collection *mongo.Collection) ClientRepository {
	return &mongoClientRepository{
		collection: collection,
	}
}

// Create inserts a new client
func (r *mongoClientRepository) Create(ctx context.Context, client *models.Client) error {
	client.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, client)
	return err
}

// FindByID retrieves a client by ID
func (r *mongoClientRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Client, error) {
	client := &models.Client{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(client)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return client, nil
}

// FindAll retrieves all clients
func (r *mongoClientRepository) FindAll(ctx context.Context) ([]*models.Client, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var clients []*models.Client
	err = cursor.All(ctx, &clients)
	if err != nil {
		return nil, err
	}
	return clients, nil
}

// FindByVetID retrieves all clients for a given veterinarian
func (r *mongoClientRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Client, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"vet_id": vetID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var clients []*models.Client
	err = cursor.All(ctx, &clients)
	if err != nil {
		return nil, err
	}
	return clients, nil
}

// Update updates an existing client
func (r *mongoClientRepository) Update(ctx context.Context, id primitive.ObjectID, client *models.Client) error {
	update := bson.M{
		"$set": client,
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

// Delete removes a client
func (r *mongoClientRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
