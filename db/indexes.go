package db

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CreateIndexes creates the non-unique indexes used by the vet-scoped and
// lookup queries across the application. It is idempotent: MongoDB ignores a
// CreateMany call for indexes that already exist with the same definition, so
// it is safe to run on every startup. Field names match the models' BSON tags.
//
// NOTE: pets are intentionally indexed on owner_id, not vet_id. The Pet model
// has no vet_id field; pets are scoped through their owning client's vet_id.
func CreateIndexes(ctx context.Context, db *mongo.Database) error {
	// collectionIndexes maps a collection name to the fields that should each
	// get a plain ascending, non-unique single-field index.
	collectionIndexes := map[string][]string{
		"clients":       {"vet_id"},
		"pets":          {"owner_id"},
		"consultations": {"vet_id", "pet_id"},
		"vaccines":      {"vet_id", "pet_id"},
		"prescriptions": {"vet_id", "pet_id"},
		"catalog_items": {"vet_id"},
		"invoices":      {"vet_id", "client_id"},
		"appointments":  {"vet_id", "start_time"},
		"attachments":   {"pet_id"},
		"notifications": {"vet_id"},
	}

	// compositeIndexes maps a collection name to the composite (multi-field)
	// indexes that back vet-scoped, time/date-ordered list queries. Each key
	// is a bson.D so field order and sort direction are preserved. Field names
	// match the models' BSON tags:
	//   appointments.start_time, invoices.issue_date, consultations.date,
	//   vaccines.date_applied, prescriptions.date.
	compositeIndexes := map[string][]bson.D{
		"appointments":  {{{Key: "vet_id", Value: 1}, {Key: "start_time", Value: 1}}},
		"invoices":      {{{Key: "vet_id", Value: 1}, {Key: "issue_date", Value: -1}}},
		"consultations": {{{Key: "vet_id", Value: 1}, {Key: "date", Value: -1}}},
		"vaccines":      {{{Key: "vet_id", Value: 1}, {Key: "date_applied", Value: -1}}},
		"prescriptions": {{{Key: "vet_id", Value: 1}, {Key: "date", Value: -1}}},
	}

	for name, fields := range collectionIndexes {
		models := make([]mongo.IndexModel, 0, len(fields)+len(compositeIndexes[name]))
		for _, field := range fields {
			models = append(models, mongo.IndexModel{
				Keys: bson.D{{Key: field, Value: 1}},
			})
		}
		for _, keys := range compositeIndexes[name] {
			models = append(models, mongo.IndexModel{
				Keys: keys,
			})
		}

		if _, err := db.Collection(name).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("failed to create indexes on %s: %w", name, err)
		}
		log.Printf("Ensured %d index(es) on %s", len(models), name)
	}

	return nil
}
