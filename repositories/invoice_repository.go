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

type mongoInvoiceRepository struct {
	collection *mongo.Collection
}

// NewInvoiceRepository creates a new invoice repository
func NewInvoiceRepository(collection *mongo.Collection) InvoiceRepository {
	return &mongoInvoiceRepository{
		collection: collection,
	}
}

// Create inserts a new invoice
func (r *mongoInvoiceRepository) Create(ctx context.Context, invoice *models.Invoice) error {
	invoice.ID = primitive.NewObjectID()
	_, err := r.collection.InsertOne(ctx, invoice)
	return err
}

// FindByID retrieves an invoice by ID
func (r *mongoInvoiceRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Invoice, error) {
	invoice := &models.Invoice{}
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(invoice)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return invoice, nil
}

// FindByVetID retrieves all invoices for a vet, most recent first
func (r *mongoInvoiceRepository) FindByVetID(ctx context.Context, vetID primitive.ObjectID) ([]*models.Invoice, error) {
	opts := options.Find().SetSort(bson.D{{Key: "issue_date", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"vet_id": vetID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var invoices []*models.Invoice
	if err = cursor.All(ctx, &invoices); err != nil {
		return nil, err
	}
	return invoices, nil
}

// FindByClientID retrieves all invoices for a given client, most recent first
func (r *mongoInvoiceRepository) FindByClientID(ctx context.Context, clientID primitive.ObjectID) ([]*models.Invoice, error) {
	opts := options.Find().SetSort(bson.D{{Key: "issue_date", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"client_id": clientID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var invoices []*models.Invoice
	if err = cursor.All(ctx, &invoices); err != nil {
		return nil, err
	}
	return invoices, nil
}

// CountByVetIDAndStatus counts invoices for a vet with the given payment status,
// without loading them.
func (r *mongoInvoiceRepository) CountByVetIDAndStatus(ctx context.Context, vetID primitive.ObjectID, status models.PaymentStatus) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"vet_id": vetID, "status": status})
}

// Update updates an existing invoice
func (r *mongoInvoiceRepository) Update(ctx context.Context, id primitive.ObjectID, invoice *models.Invoice) error {
	update := bson.M{
		"$set": invoice,
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

// Delete removes an invoice
func (r *mongoInvoiceRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteByClientID removes all invoices for a given client
func (r *mongoInvoiceRepository) DeleteByClientID(ctx context.Context, clientID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"client_id": clientID})
	return err
}
