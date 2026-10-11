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

type mongoAttachmentRepository struct {
	collection *mongo.Collection
}

// NewAttachmentRepository creates a new attachment repository
func NewAttachmentRepository(collection *mongo.Collection) AttachmentRepository {
	return &mongoAttachmentRepository{
		collection: collection,
	}
}

// Create inserts a new attachment
func (r *mongoAttachmentRepository) Create(ctx context.Context, attachment *models.Attachment) error {
	attachment.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, attachment)
	return err
}

// FindByID retrieves a full attachment (including base64 content) by ID
func (r *mongoAttachmentRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Attachment, error) {
	attachment := &models.Attachment{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(attachment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return attachment, nil
}

// FindSummariesByPetID lists attachments for a pet, excluding the heavy content
// field via a projection, most recent first.
func (r *mongoAttachmentRepository) FindSummariesByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.AttachmentSummary, error) {
	opts := options.Find().
		SetProjection(bson.M{"content": 0}).
		SetSort(bson.D{{Key: "date", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"pet_id": petID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var summaries []*models.AttachmentSummary
	if err = cursor.All(ctx, &summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

// Update applies partial updates to an attachment (used for metadata edits).
func (r *mongoAttachmentRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]any) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Delete removes an attachment
func (r *mongoAttachmentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByPetID removes all attachments for a given pet
func (r *mongoAttachmentRepository) DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"pet_id": petID})
	return err
}
