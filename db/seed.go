package db

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SeedDatabase ensures indices exist and populates the database with test data
// only when it is empty. Existing data is never dropped, so restarting the
// server does not wipe records created during use. Set SEED_FORCE=true to force
// a clean re-seed (drops all collections first).
func SeedDatabase(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create indices (idempotent - safe to run on every startup)
	if err := createUserEmailIndex(db, ctx); err != nil {
		log.Printf("Warning: Failed to create email index: %v", err)
	}

	forceSeed := strings.EqualFold(os.Getenv("SEED_FORCE"), "true")

	if forceSeed {
		if err := clearCollections(db, ctx); err != nil {
			return fmt.Errorf("failed to clear collections: %w", err)
		}
		log.Println("SEED_FORCE=true -> database cleared for a clean re-seed")
	} else {
		// Skip seeding if the database already has data.
		count, err := db.Collection("users").EstimatedDocumentCount(ctx)
		if err != nil {
			return fmt.Errorf("failed to check existing data: %w", err)
		}
		if count > 0 {
			log.Printf("Database already has data (%d users) - skipping seed", count)
			return nil
		}
		log.Println("Database is empty - seeding test data")
	}

	// Seed test data
	if err := seedTestData(db, ctx); err != nil {
		return fmt.Errorf("failed to seed test data: %w", err)
	}

	log.Println("Database seeded with test data successfully")
	return nil
}

// clearCollections drops all data from the database
func clearCollections(db *mongo.Database, ctx context.Context) error {
	collections := []string{"users", "clients", "pets", "consultations", "vaccines"}
	for _, colName := range collections {
		col := db.Collection(colName)
		if err := col.Drop(ctx); err != nil {
			log.Printf("Warning: Failed to drop collection %s: %v", colName, err)
		}
	}
	return nil
}

// seedTestData populates the database with realistic test data
func seedTestData(db *mongo.Database, ctx context.Context) error {
	rand.Seed(time.Now().UnixNano())

	// Create test passwords (senha: 123456)
	hashedPassword := helpers.HashPassword("123456")

	// Create first test user (veterinarian)
	vetUser1 := models.User{
		ID:             primitive.NewObjectID(),
		Name:           "Dr. João Silva",
		Email:          "dr.joao@vetflow.com",
		Password:       hashedPassword,
		ProfilePicture: "https://api.dicebear.com/7.x/avataaars/svg?seed=joao",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create second test user (veterinarian)
	vetUser2 := models.User{
		ID:             primitive.NewObjectID(),
		Name:           "Dra. Maria Oliveira",
		Email:          "dra.maria@vetflow.com",
		Password:       hashedPassword,
		ProfilePicture: "https://api.dicebear.com/7.x/avataaars/svg?seed=maria",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if _, err := db.Collection("users").InsertOne(ctx, vetUser1); err != nil {
		return err
	}
	log.Printf("Created test user: %s (%s) - Password: 123456", vetUser1.Name, vetUser1.Email)

	if _, err := db.Collection("users").InsertOne(ctx, vetUser2); err != nil {
		return err
	}
	log.Printf("Created test user: %s (%s) - Password: 123456", vetUser2.Name, vetUser2.Email)

	// Create clients - split between two vets
	clientNames := []string{
		"Maria Santos", "Carlos Oliveira", "Ana Costa",
		"Pedro Gomes", "Fernanda Lima", "Ricardo Souza",
	}

	phoneNumbers := []string{
		"(11) 98765-4321", "(11) 97654-3210", "(11) 96543-2109",
		"(11) 95432-1098", "(11) 94321-0987", "(11) 93210-9876",
	}

	streets := []string{
		"Rua das Flores", "Avenida Principal", "Rua da Paz",
		"Avenida Brasil", "Rua do Comércio", "Avenida Paulista",
	}

	var clients []*models.Client
	for i, name := range clientNames {
		// Alternate between the two vets
		vetID := vetUser1.ID
		if i%2 == 1 {
			vetID = vetUser2.ID
		}

		client := models.Client{
			ID:       primitive.NewObjectID(),
			VetID:    vetID,
			FullName: name,
			CPF:      fmt.Sprintf("%011d", rand.Intn(99999999999)),
			Phone:    phoneNumbers[i],
			Address: models.Address{
				Street:     streets[i],
				Number:     fmt.Sprintf("%d", 100+rand.Intn(900)),
				Complement: fmt.Sprintf("Apt %d", 100+rand.Intn(900)),
				ZipCode:    fmt.Sprintf("%05d-%03d", 1000+rand.Intn(9000), rand.Intn(1000)),
			},
			CreatedAt: time.Now().AddDate(0, -rand.Intn(6), -rand.Intn(30)),
			UpdatedAt: time.Now(),
		}
		clients = append(clients, &client)
	}

	clientObjs := make([]interface{}, len(clients))
	for i, c := range clients {
		clientObjs[i] = c
	}

	if _, err := db.Collection("clients").InsertMany(ctx, clientObjs); err != nil {
		return err
	}
	log.Printf("Created %d test clients", len(clients))

	// Create pets
	petSpecies := []string{"Cachorro", "Gato", "Coelho", "Pássaro"}
	petBreeds := map[string][]string{
		"Cachorro": {"Labrador", "Pastor Alemão", "Poodle", "Bulldog", "Cocker Spaniel"},
		"Gato":     {"Siamês", "Persa", "Maine Coon", "Ragdoll"},
		"Coelho":   {"Holandês", "Angorá", "Lop"},
		"Pássaro":  {"Canário", "Periquito", "Papagaio"},
	}
	petNames := []string{
		"Max", "Bella", "Luna", "Charlie", "Rocky",
		"Mimi", "Whiskers", "Félix", "Mittens", "Simba",
	}

	var pets []*models.Pet
	for i := 0; i < 15; i++ {
		species := petSpecies[rand.Intn(len(petSpecies))]
		breeds := petBreeds[species]
		breed := breeds[rand.Intn(len(breeds))]
		client := clients[rand.Intn(len(clients))]

		pet := models.Pet{
			ID:        primitive.NewObjectID(),
			Name:      petNames[rand.Intn(len(petNames))],
			Species:   species,
			Breed:     breed,
			Age:       rand.Intn(15) + 1,
			AgeUnit:   "anos",
			Sex:       []string{"M", "F"}[rand.Intn(2)],
			Neutered:  rand.Float32() > 0.3,
			Weight:    math.Round(float64(rand.Intn(50)+5)*10) / 10,
			OwnerID:   client.ID,
			CreatedAt: time.Now().AddDate(0, -rand.Intn(12), -rand.Intn(30)),
			UpdatedAt: time.Now(),
		}
		pets = append(pets, &pet)
	}

	petObjs := make([]interface{}, len(pets))
	for i, p := range pets {
		petObjs[i] = p
	}

	if _, err := db.Collection("pets").InsertMany(ctx, petObjs); err != nil {
		return err
	}
	log.Printf("Created %d test pets", len(pets))

	// Create consultations
	complaints := []string{
		"Tosse contínua", "Vômito frequente", "Diarreia",
		"Perda de apetite", "Coceira excessiva", "Mancando",
		"Comportamento agressivo", "Apatia",
	}
	evolutionTimes := []string{"1 dia", "2 dias", "1 semana", "2 semanas", "1 mês"}
	frequencies := []string{"Constante", "Intermitente", "Uma vez ao dia", "Várias vezes ao dia"}
	summaryTexts := []string{
		"Animal apresenta bom estado geral",
		"Recomendado repouso e medicação",
		"Encaminhar para especialista se persistir",
		"Realizar exames complementares",
		"Prescrever dieta especial",
	}

	var consultations []*models.Consultation
	for i := 0; i < 20; i++ {
		pet := pets[rand.Intn(len(pets))]
		consultationDate := time.Now().AddDate(0, -rand.Intn(6), -rand.Intn(30))

		// Alternate between the two vets
		vetID := vetUser1.ID
		if i%2 == 1 {
			vetID = vetUser2.ID
		}

		// Some consultations have return dates scheduled for the future (7 days, 1 day, or today from now)
		var returnDate *time.Time
		if i%3 == 0 {
			// 7 days from now
			rd := time.Now().AddDate(0, 0, 7)
			returnDate = &rd
		} else if i%3 == 1 {
			// 1 day from now
			rd := time.Now().AddDate(0, 0, 1)
			returnDate = &rd
		}

		consultation := models.Consultation{
			ID:         primitive.NewObjectID(),
			VetID:      vetID,
			PetID:      pet.ID,
			Date:       consultationDate,
			ReturnDate: returnDate,
			Anamnesis: models.Anamnesis{
				MainComplaint:    complaints[rand.Intn(len(complaints))],
				EvolutionTime:    evolutionTimes[rand.Intn(len(evolutionTimes))],
				SymptomsWorsened: rand.Float32() > 0.5,
				SymptomFrequency: frequencies[rand.Intn(len(frequencies))],
				BehaviorChange:   rand.Float32() > 0.4,
				Pain:             rand.Float32() > 0.6,
				HappenedBefore:   rand.Float32() > 0.5,
				PriorVetVisit:    rand.Float32() > 0.3,
			},
			Diet: models.Diet{
				FoodType:            "Ração premium",
				AmountAndFrequency:  fmt.Sprintf("%d gramas 2x ao dia", 100+rand.Intn(300)),
				DietChangedRecently: rand.Float32() > 0.7,
				NormalWaterIntake:   rand.Float32() > 0.3,
			},
			Physiological: models.Physiological{
				NormalUrineFeces:  rand.Float32() > 0.3,
				Vomiting:          rand.Float32() > 0.7,
				RespiratoryIssues: rand.Float32() > 0.8,
			},
			Summary: models.Summary{
				EvolutionTime:          evolutionTimes[rand.Intn(len(evolutionTimes))],
				FecesUrine:             "Normal",
				VaccinesAndDeworming:   "Vacinação em dia, última vermifugação faz 6 meses",
				Medications:            "Prescrever antibiótico por 10 dias",
				EnvironmentAndContacts: summaryTexts[rand.Intn(len(summaryTexts))],
			},
			CreatedAt: consultationDate,
		}
		consultations = append(consultations, &consultation)
	}

	consultationObjs := make([]interface{}, len(consultations))
	for i, c := range consultations {
		consultationObjs[i] = c
	}

	if _, err := db.Collection("consultations").InsertMany(ctx, consultationObjs); err != nil {
		return err
	}
	log.Printf("Created %d test consultations", len(consultations))

	// Create vaccines
	vaccineNames := []string{
		"Vacina Múltipla (V10/V8)", "Raiva", "Gripe Felina",
		"Leucemia Felina", "Bordetelose", "Leptospirose",
	}
	manufacturers := []string{
		"Merial", "Pfizer", "Boehringer Ingelheim", "MSD",
	}

	var vaccines []*models.Vaccine
	for i := 0; i < 30; i++ {
		pet := pets[rand.Intn(len(pets))]
		appliedDate := time.Now().AddDate(0, -rand.Intn(12), -rand.Intn(30))

		// Alternate between the two vets
		vetID := vetUser1.ID
		if i%2 == 1 {
			vetID = vetUser2.ID
		}

		// Some vaccines have next dose dates that trigger notifications (7 days, 1 day, or today)
		var nextDoseDate time.Time
		if i%3 == 0 {
			// 7 days from now
			nextDoseDate = time.Now().AddDate(0, 0, 7)
		} else if i%3 == 1 {
			// 1 day from now
			nextDoseDate = time.Now().AddDate(0, 0, 1)
		} else {
			// Default: 12 months from now (no notification)
			nextDoseDate = appliedDate.AddDate(0, 12, 0)
		}

		vaccine := models.Vaccine{
			ID:           primitive.NewObjectID(),
			VetID:        vetID,
			PetID:        pet.ID,
			VaccineName:  vaccineNames[rand.Intn(len(vaccineNames))],
			Batch:        fmt.Sprintf("BATCH%d", 1000000+rand.Intn(9000000)),
			Manufacturer: manufacturers[rand.Intn(len(manufacturers))],
			DateApplied:  appliedDate,
			NextDoseDate: nextDoseDate,
			Notes:        "Aplicada com sucesso, sem reações adversas",
			CreatedAt:    appliedDate,
		}
		vaccines = append(vaccines, &vaccine)
	}

	vaccineObjs := make([]interface{}, len(vaccines))
	for i, v := range vaccines {
		vaccineObjs[i] = v
	}

	if _, err := db.Collection("vaccines").InsertMany(ctx, vaccineObjs); err != nil {
		return err
	}
	log.Printf("Created %d test vaccines", len(vaccines))

	return nil
}

// createUserEmailIndex creates a unique index on the email field
func createUserEmailIndex(db *mongo.Database, ctx context.Context) error {
	collection := db.Collection("users")
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "email", Value: 1},
		},
		Options: &options.IndexOptions{
			Unique: &[]bool{true}[0],
		},
	}

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	return err
}
