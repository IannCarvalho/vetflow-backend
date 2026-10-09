package db

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// collectionIndexSpec declares the indexes a single collection should have.
// SingleFields each get a plain ascending, non-unique single-field index.
// Compound holds ordered key-sets for multi-field ascending indexes, used by
// vet-scoped count queries that filter on vet_id plus one more field.
type collectionIndexSpec struct {
	SingleFields []string
	Compound     [][]string
}

// CreateIndexes creates the non-unique indexes used by the vet-scoped and
// lookup queries across the application. It is idempotent: MongoDB ignores a
// CreateMany call for indexes that already exist with the same definition, so
// it is safe to run on every startup. Field names match the models' BSON tags
// and the repository count/query filters exactly, so the indexes are usable by
// those queries.
//
// NOTE: pets are intentionally indexed on owner_id, not vet_id. The Pet model
// has no vet_id field; pets are scoped through their owning client's vet_id.
func CreateIndexes(ctx context.Context, db *mongo.Database) error {
	// collectionIndexes maps a collection name to its index specification.
	// Compound {vet_id, <field>} indexes back the dashboard count queries
	// (e.g. CountByVetIDInRange / CountByVetIDAndStatus) so those filters stay
	// index-friendly at scale; the single-field indexes still cover the other
	// lookups (pet_id history, client_id invoices, owner_id pets, etc.).
	collectionIndexes := map[string]collectionIndexSpec{
		"clients": {SingleFields: []string{"vet_id"}},
		"pets":    {SingleFields: []string{"owner_id"}},
		"consultations": {
			SingleFields: []string{"vet_id", "pet_id"},
			// CountByVetIDInRange filters vet_id + date.
			Compound: [][]string{{"vet_id", "date"}},
		},
		"vaccines": {
			SingleFields: []string{"vet_id", "pet_id"},
			// CountByVetIDNextDoseInRange filters vet_id + next_dose_date.
			Compound: [][]string{{"vet_id", "next_dose_date"}},
		},
		"prescriptions": {SingleFields: []string{"vet_id", "pet_id"}},
		"catalog_items": {SingleFields: []string{"vet_id"}},
		"invoices": {
			SingleFields: []string{"vet_id", "client_id"},
			// CountByVetIDAndStatus filters vet_id + status.
			Compound: [][]string{{"vet_id", "status"}},
		},
		"appointments": {
			SingleFields: []string{"vet_id", "start_time"},
			// CountByVetIDInRange filters vet_id + start_time.
			Compound: [][]string{{"vet_id", "start_time"}},
		},
		"attachments":   {SingleFields: []string{"pet_id"}},
		"notifications": {SingleFields: []string{"vet_id"}},
	}

	for name, spec := range collectionIndexes {
		models := make([]mongo.IndexModel, 0, len(spec.SingleFields)+len(spec.Compound))

		for _, field := range spec.SingleFields {
			models = append(models, mongo.IndexModel{
				Keys: bson.D{{Key: field, Value: 1}},
			})
		}

		for _, keys := range spec.Compound {
			compoundKeys := make(bson.D, 0, len(keys))
			for _, field := range keys {
				compoundKeys = append(compoundKeys, bson.E{Key: field, Value: 1})
			}
			models = append(models, mongo.IndexModel{Keys: compoundKeys})
		}

		if _, err := db.Collection(name).Indexes().CreateMany(ctx, models); err != nil {
			return fmt.Errorf("failed to create indexes on %s: %w", name, err)
		}
		log.Printf("Ensured %d index(es) on %s", len(models), name)
	}

	return nil
}
