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

type mongoCatalogItemRepository struct {
	collection *mongo.Collection
}

// NewCatalogItemRepository creates a new catalog item repository
func NewCatalogItemRepository(collection *mongo.Collection) CatalogItemRepository {
	return &mongoCatalogItemRepository{
		collection: collection,
	}
}

// Create inserts a new catalog item
func (r *mongoCatalogItemRepository) Create(ctx context.Context, item *models.CatalogItem) error {
	item.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, item)
	return err
}

// FindByID retrieves a catalog item by ID
func (r *mongoCatalogItemRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.CatalogItem, error) {
	item := &models.CatalogItem{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(item)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

// FindByVetID retrieves all catalog items for a vet, sorted by name
func (r *mongoCatalogItemRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.CatalogItem, error) {
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"vet_id": vetID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var items []*models.CatalogItem
	if err = cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Update updates an existing catalog item
func (r *mongoCatalogItemRepository) Update(ctx context.Context, id primitive.ObjectID, item *models.CatalogItem) error {
	update := bson.M{
		"$set": item,
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

// Delete removes a catalog item
func (r *mongoCatalogItemRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
