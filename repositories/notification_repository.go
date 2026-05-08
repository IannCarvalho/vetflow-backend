package repositories

import (
	"context"
	"time"

	"github.com/vetflow/veterinary-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type notificationRepository struct {
	collection *mongo.Collection
}

func NewNotificationRepository(db *mongo.Database) NotificationRepository {
	return &notificationRepository{
		collection: db.Collection("notifications"),
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = primitive.NewObjectID()
	notification.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, notification)
	return err
}

func (r *notificationRepository) GetByVetID(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error) {
	filter := bson.M{"vet_id": vetID}
	opts := options.Find().SetSort(bson.M{"target_date": 1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *notificationRepository) GetPending(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error) {
	filter := bson.M{
		"vet_id": vetID,
		"status": models.NotificationStatusPending,
	}
	opts := options.Find().SetSort(bson.M{"target_date": 1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []models.Notification
	if err = cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	if status == models.NotificationStatusSent {
		update["$set"] = bson.M{
			"status":  status,
			"sent_at": time.Now(),
		}
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *notificationRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

// DeleteOld removes notifications older than 30 days
func (r *notificationRepository) DeleteOld(ctx context.Context, vetID primitive.ObjectID) error {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	filter := bson.M{
		"vet_id": vetID,
		"created_at": bson.M{
			"$lt": thirtyDaysAgo,
		},
	}

	_, err := r.collection.DeleteMany(ctx, filter)
	return err
}
