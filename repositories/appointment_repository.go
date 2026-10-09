package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoAppointmentRepository struct {
	collection *mongo.Collection
}

// NewAppointmentRepository creates a new appointment repository
func NewAppointmentRepository(collection *mongo.Collection) AppointmentRepository {
	return &mongoAppointmentRepository{
		collection: collection,
	}
}

// Create inserts a new appointment
func (r *mongoAppointmentRepository) Create(ctx context.Context, appointment *models.Appointment) error {
	appointment.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, appointment)
	return err
}

// FindByID retrieves an appointment by ID
func (r *mongoAppointmentRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Appointment, error) {
	appointment := &models.Appointment{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(appointment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return appointment, nil
}

// FindByVetID retrieves all appointments for a specific veterinarian, sorted by start time
func (r *mongoAppointmentRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Appointment, error) {
	opts := options.Find().SetSort(bson.D{{Key: "start_time", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"vet_id": vetID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var appointments []*models.Appointment
	if err = cursor.All(ctx, &appointments); err != nil {
		return nil, err
	}
	return appointments, nil
}

// FindByVetIDInRange retrieves appointments for a vet within a time window (e.g. a calendar month/week)
func (r *mongoAppointmentRepository) FindByVetIDInRange(ctx context.Context, vetID primitive.ObjectID, start, end time.Time) ([]*models.Appointment, error) {
	filter := bson.M{
		"vet_id": vetID,
		"start_time": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}
	opts := options.Find().SetSort(bson.D{{Key: "start_time", Value: 1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var appointments []*models.Appointment
	if err = cursor.All(ctx, &appointments); err != nil {
		return nil, err
	}
	return appointments, nil
}

// CountByVetIDInRange counts appointments for a vet whose start time falls in the
// half-open window [start, end) ($gte start, $lt end), without loading them.
func (r *mongoAppointmentRepository) CountByVetIDInRange(ctx context.Context, vetID primitive.ObjectID, start, end time.Time) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"vet_id": vetID,
		"start_time": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	})
}

// FindByPetID retrieves all appointments for a given pet, sorted by start time
func (r *mongoAppointmentRepository) FindByPetID(ctx context.Context, petID primitive.ObjectID) ([]*models.Appointment, error) {
	opts := options.Find().SetSort(bson.D{{Key: "start_time", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"pet_id": petID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var appointments []*models.Appointment
	if err = cursor.All(ctx, &appointments); err != nil {
		return nil, err
	}
	return appointments, nil
}

// Update updates an existing appointment
func (r *mongoAppointmentRepository) Update(ctx context.Context, id primitive.ObjectID, appointment *models.Appointment) error {
	update := bson.M{
		"$set": appointment,
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

// Delete removes an appointment
func (r *mongoAppointmentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByPetID removes all appointments for a given pet
func (r *mongoAppointmentRepository) DeleteByPetID(ctx context.Context, petID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"pet_id": petID})
	return err
}

// DeleteByClientID removes all appointments for a given client
func (r *mongoAppointmentRepository) DeleteByClientID(ctx context.Context, clientID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"client_id": clientID})
	return err
}
