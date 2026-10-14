package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Demo/visitor credentials. The seed creates these accounts in every
// non-production environment, so a public demo can offer a one-click
// "Entrar como visitante" login. They are intentionally well-known — which is
// exactly why SeedDatabase refuses to run in production (see below) and the
// config guard forbids a non-production app from using the production
// database. NEVER rely on these for a real deployment.
const (
	DemoVisitorEmail    = "dr.joao@vetflow.com"
	DemoVisitorEmailAlt = "dra.maria@vetflow.com"
	DemoVisitorPassword = "123456"
)

// SeedDatabase ensures indices exist and populates the database with test data
// only when it is empty. Existing data is never dropped, so restarting the
// server does not wipe records created during use. Set SEED_FORCE=true to force
// a clean re-seed (drops all collections first).
func SeedDatabase(db *mongo.Database) error {
	// Defense in depth: refuse to seed in production even if a caller invokes
	// this directly. The seed inserts fixed demo accounts with a well-known
	// password, so running it against a live database is a credential backdoor.
	appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	if appEnv == "production" || appEnv == "prod" {
		log.Println("Refusing to seed database in production environment (APP_ENV=" + appEnv + ")")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create indices (idempotent - safe to run on every startup)
	if err := createUserEmailIndex(ctx, db); err != nil {
		log.Printf("Warning: Failed to create email index: %v", err)
	}

	forceSeed := strings.EqualFold(os.Getenv("SEED_FORCE"), "true")

	if forceSeed {
		if err := clearCollections(ctx, db); err != nil {
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

	if err := seedTestData(ctx, db); err != nil {
		return fmt.Errorf("failed to seed test data: %w", err)
	}

	recordSeedTimestamp(ctx, db)
	log.Println("Database seeded with test data successfully")
	return nil
}

// ResetDemoDatabase wipes the demo database and re-seeds it from scratch,
// reusing the exact same seed data as a fresh boot. It exists so a public demo
// can be reset on a schedule, discarding anything visitors created and
// restoring a clean, coherent dataset — without a redeploy.
//
// It applies the SAME production guard as SeedDatabase: it refuses to run when
// APP_ENV is production/prod, because dropping all collections and inserting
// fixed demo accounts against a live database would destroy real data and open
// a credential backdoor. The HTTP route that calls this is additionally gated
// by APP_ENV and an admin token, so there are two independent layers.
func ResetDemoDatabase(db *mongo.Database) error {
	appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	if appEnv == "production" || appEnv == "prod" {
		log.Println("Refusing to reset database in production environment (APP_ENV=" + appEnv + ")")
		return fmt.Errorf("reset refused: APP_ENV=%s is a production environment", appEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Drop everything, then re-insert the canonical seed data. Index creation
	// is idempotent, so re-ensure the unique email index after the wipe.
	if err := clearCollections(ctx, db); err != nil {
		return fmt.Errorf("failed to clear collections: %w", err)
	}
	log.Println("Demo reset -> all collections dropped, re-seeding")

	if err := createUserEmailIndex(ctx, db); err != nil {
		log.Printf("Warning: Failed to re-create email index after reset: %v", err)
	}

	if err := seedTestData(ctx, db); err != nil {
		return fmt.Errorf("failed to re-seed test data: %w", err)
	}

	recordSeedTimestamp(ctx, db)
	log.Println("Demo database reset and re-seeded successfully")
	return nil
}

// demoMetaCollection stores a single document tracking when the demo data was
// last (re)seeded, so reset-on-boot can throttle itself. It is intentionally
// NOT in clearCollections' drop list, so a reset never erases its own clock.
const demoMetaCollection = "_demo_meta"

// defaultDemoResetIntervalHours is how long demo data lives before boot
// re-seeds it, when DEMO_RESET_INTERVAL_HOURS is unset/invalid.
const defaultDemoResetIntervalHours = 6

// MaybeResetDemoOnBoot re-seeds the demo database on startup, but only when the
// last seed is older than the configured window (DEMO_RESET_INTERVAL_HOURS,
// default 6h). This gives a public demo a hands-off self-cleaning loop — the
// Render free tier sleeps when idle and re-boots on the next visit, so an idle
// gap longer than the window yields a clean demo — WITHOUT any external
// scheduler, token, or dependency.
//
// The window matters: without it, every boot would wipe the database, so a
// redeploy or a restart in the middle of someone's session would destroy their
// data. With it, a boot shortly after a recent seed is a no-op.
//
// It is a no-op outside demo mode (APP_ENV must be exactly "demo"), so neither
// production nor local development is ever touched. Errors are logged, never
// fatal: a failed reset must not stop the server from booting.
func MaybeResetDemoOnBoot(db *mongo.Database) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) != "demo" {
		return
	}

	hours := defaultDemoResetIntervalHours
	if v := strings.TrimSpace(os.Getenv("DEMO_RESET_INTERVAL_HOURS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			hours = n
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	meta := db.Collection(demoMetaCollection)
	var doc struct {
		LastSeededAt time.Time `bson:"last_seeded_at"`
	}
	err := meta.FindOne(ctx, bson.M{"_id": "seed"}).Decode(&doc)

	switch {
	case err == mongo.ErrNoDocuments:
		log.Println("Demo reset-on-boot: no prior seed timestamp -> reseeding")
	case err != nil:
		log.Printf("Demo reset-on-boot: failed to read seed timestamp (%v) -> skipping to be safe", err)
		return
	default:
		age := time.Since(doc.LastSeededAt)
		if age < time.Duration(hours)*time.Hour {
			log.Printf("Demo reset-on-boot: last seed %s ago (< %dh window) -> skipping", age.Round(time.Minute), hours)
			return
		}
		log.Printf("Demo reset-on-boot: last seed %s ago (>= %dh window) -> reseeding", age.Round(time.Minute), hours)
	}

	if err := ResetDemoDatabase(db); err != nil {
		log.Printf("Demo reset-on-boot: reset failed: %v", err)
		return
	}
	// Note: ResetDemoDatabase already records the seed timestamp.
}

// recordSeedTimestamp upserts the "last seeded" clock used by reset-on-boot.
// Only meaningful in demo mode; harmless elsewhere. Failures are logged, not
// returned — a missing timestamp just means the next boot reseeds.
func recordSeedTimestamp(ctx context.Context, db *mongo.Database) {
	if _, err := db.Collection(demoMetaCollection).UpdateOne(
		ctx,
		bson.M{"_id": "seed"},
		bson.M{"$set": bson.M{"last_seeded_at": time.Now()}},
		options.Update().SetUpsert(true),
	); err != nil {
		log.Printf("Demo meta: failed to record seed timestamp: %v", err)
	}
}

// clearCollections drops all seeded collections.
func clearCollections(ctx context.Context, db *mongo.Database) error {
	collections := []string{
		"users", "clients", "pets", "consultations", "vaccines",
		"prescriptions", "catalog_items", "invoices", "appointments",
		"attachments", "notifications",
	}
	for _, colName := range collections {
		if err := db.Collection(colName).Drop(ctx); err != nil {
			log.Printf("Warning: Failed to drop collection %s: %v", colName, err)
		}
	}
	return nil
}

// dayAt returns `days` from midnight today (UTC) at 09:00, useful for building
// predictable due/return/next-dose dates that drive notifications.
func dayAt(days int) time.Time {
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC)
	return midnight.AddDate(0, 0, days)
}

// seedTestData populates the database with coherent, realistic sample data.
func seedTestData(ctx context.Context, db *mongo.Database) error {
	hashedPassword := helpers.HashPassword(DemoVisitorPassword)
	now := time.Now()

	// --- Veterinarians -----------------------------------------------------
	vet1 := models.User{
		ID:             primitive.NewObjectID(),
		Name:           "Dr. João Silva",
		Email:          DemoVisitorEmail,
		Password:       hashedPassword,
		ProfilePicture: "https://api.dicebear.com/7.x/avataaars/svg?seed=joao",
		CreatedAt:      now, UpdatedAt: now,
	}
	vet2 := models.User{
		ID:             primitive.NewObjectID(),
		Name:           "Dra. Maria Oliveira",
		Email:          DemoVisitorEmailAlt,
		Password:       hashedPassword,
		ProfilePicture: "https://api.dicebear.com/7.x/avataaars/svg?seed=maria",
		CreatedAt:      now, UpdatedAt: now,
	}
	if _, err := db.Collection("users").InsertMany(ctx, []interface{}{vet1, vet2}); err != nil {
		return err
	}
	log.Printf("Created 2 test users")

	// --- Clients -----------------------------------------------------------
	type clientSpec struct {
		name, cpf, phone, street, number, zip string
		vet                                   primitive.ObjectID
	}
	clientSpecs := []clientSpec{
		{"Maria Santos", "529.982.247-25", "(11) 98765-4321", "Rua das Flores", "123", "01234-567", vet1.ID},
		{"Carlos Oliveira", "168.995.350-09", "(11) 97654-3210", "Avenida Paulista", "1500", "01310-100", vet1.ID},
		{"Ana Costa", "111.444.777-35", "(11) 96543-2109", "Rua da Paz", "88", "02345-678", vet1.ID},
		{"Pedro Gomes", "390.533.447-05", "(21) 95432-1098", "Avenida Brasil", "742", "20010-000", vet2.ID},
		{"Fernanda Lima", "457.845.818-05", "(21) 94321-0987", "Rua do Comércio", "56", "20021-130", vet2.ID},
		{"Ricardo Souza", "764.831.158-57", "(31) 93210-9876", "Rua dos Ipês", "320", "30140-071", vet2.ID},
	}
	clients := make([]*models.Client, len(clientSpecs))
	clientObjs := make([]interface{}, len(clientSpecs))
	for i, s := range clientSpecs {
		c := &models.Client{
			ID:       primitive.NewObjectID(),
			VetID:    s.vet,
			FullName: s.name,
			CPF:      s.cpf,
			Phone:    s.phone,
			Address: models.Address{
				Street: s.street, Number: s.number, Complement: "", ZipCode: s.zip,
			},
			CreatedAt: now.AddDate(0, -(i + 2), 0),
			UpdatedAt: now,
		}
		clients[i] = c
		clientObjs[i] = c
	}
	if _, err := db.Collection("clients").InsertMany(ctx, clientObjs); err != nil {
		return err
	}
	log.Printf("Created %d test clients", len(clients))

	// --- Pets (coherent name/species/breed/sex per owner) ------------------
	type petSpec struct {
		name, species, breed, sex string
		age                       int
		ageUnit                   string
		neutered                  bool
		weight                    float64
		ownerIdx                  int
	}
	petSpecs := []petSpec{
		{"Thor", "Cachorro", "Labrador", "M", 4, "anos", true, 32.5, 0},
		{"Nina", "Gato", "Siamês", "F", 3, "anos", true, 4.2, 0},
		{"Bidu", "Cachorro", "Poodle", "M", 7, "anos", true, 8.1, 1},
		{"Amora", "Cachorro", "Golden Retriever", "F", 2, "anos", false, 28.0, 2},
		{"Lola", "Gato", "Persa", "F", 5, "anos", true, 3.8, 2},
		{"Rex", "Cachorro", "Pastor Alemão", "M", 6, "anos", false, 35.4, 3},
		{"Mel", "Cachorro", "Shih Tzu", "F", 1, "anos", false, 5.5, 4},
		{"Pingo", "Gato", "SRD", "M", 2, "anos", true, 4.6, 4},
		{"Fiona", "Coelho", "Mini Lop", "F", 1, "anos", false, 1.6, 5},
		{"Zeca", "Cachorro", "Bulldog Francês", "M", 3, "anos", true, 11.2, 5},
	}
	pets := make([]*models.Pet, len(petSpecs))
	petObjs := make([]interface{}, len(petSpecs))
	for i, s := range petSpecs {
		p := &models.Pet{
			ID:        primitive.NewObjectID(),
			Name:      s.name,
			Species:   s.species,
			Breed:     s.breed,
			Age:       s.age,
			AgeUnit:   s.ageUnit,
			Sex:       s.sex,
			Neutered:  s.neutered,
			Weight:    s.weight,
			OwnerID:   clients[s.ownerIdx].ID,
			CreatedAt: now.AddDate(0, -s.age*2, 0),
			UpdatedAt: now,
		}
		pets[i] = p
		petObjs[i] = p
	}
	if _, err := db.Collection("pets").InsertMany(ctx, petObjs); err != nil {
		return err
	}
	log.Printf("Created %d test pets", len(pets))

	// vetForPet returns the vet that owns the pet (via its client).
	vetForPet := func(p *models.Pet) primitive.ObjectID {
		for _, c := range clients {
			if c.ID == p.OwnerID {
				return c.VetID
			}
		}
		return vet1.ID
	}
	// clientForPet returns the owning client id.
	clientForPet := func(p *models.Pet) primitive.ObjectID { return p.OwnerID }

	// --- Consultations -----------------------------------------------------
	type consultSpec struct {
		petIdx       int
		daysAgo      int
		complaint    string
		returnInDays int // -1 = no return scheduled
		summary      string
		meds         string
	}
	consultSpecs := []consultSpec{
		{0, 20, "Mancando da pata traseira direita", 7, "Suspeita de distensão muscular; repouso e anti-inflamatório", "Carprofeno 25mg, 1x ao dia por 7 dias"},
		{1, 15, "Vômitos ocasionais e menos apetite", 1, "Gastrite leve; dieta branda e protetor gástrico", "Omeprazol 10mg em jejum por 5 dias"},
		{3, 10, "Coceira intensa e vermelhidão na pele", 0, "Dermatite alérgica; banho medicamentoso e antialérgico", "Prednisolona 20mg em dose decrescente"},
		{5, 30, "Check-up anual de rotina", -1, "Animal hígido; vacinação e vermifugação em dia", "Nenhuma medicação necessária"},
		{6, 5, "Tosse seca há 3 dias", 14, "Traqueobronquite; repouso e xarope", "Bromexina conforme peso, 2x ao dia"},
		{2, 45, "Avaliação pós-cirúrgica (castração)", -1, "Cicatrização adequada; alta clínica", "Finalizar antibiótico já prescrito"},
		{7, 3, "Dificuldade para urinar", 1, "Suspeita de cistite; coletar urina para exame", "Antibiótico após resultado do exame"},
		{9, 60, "Spider check de pele e orelhas", -1, "Otite externa leve; limpeza auricular", "Solução otológica 2x ao dia por 10 dias"},
	}
	consultations := make([]*models.Consultation, 0, len(consultSpecs))
	consultObjs := make([]interface{}, 0, len(consultSpecs))
	for _, s := range consultSpecs {
		pet := pets[s.petIdx]
		date := now.AddDate(0, 0, -s.daysAgo)
		var returnDate *time.Time
		if s.returnInDays >= 0 {
			rd := dayAt(s.returnInDays)
			returnDate = &rd
		}
		c := &models.Consultation{
			ID:         primitive.NewObjectID(),
			VetID:      vetForPet(pet),
			PetID:      pet.ID,
			Date:       date,
			ReturnDate: returnDate,
			Anamnesis: models.Anamnesis{
				MainComplaint:    s.complaint,
				EvolutionTime:    "3 dias",
				SymptomsWorsened: false,
				SymptomFrequency: "Intermitente",
				BehaviorChange:   false,
				Pain:             s.returnInDays == 0,
				HappenedBefore:   false,
				PriorVetVisit:    true,
			},
			Diet: models.Diet{
				FoodType:            "Ração seca premium",
				AmountAndFrequency:  "2 refeições ao dia conforme porção da embalagem",
				DietChangedRecently: false,
				NormalWaterIntake:   true,
			},
			Physiological: models.Physiological{
				NormalUrineFeces:  s.petIdx != 7,
				Vomiting:          s.petIdx == 1,
				RespiratoryIssues: s.petIdx == 6,
			},
			Summary: models.Summary{
				EvolutionTime:          "3 dias",
				FecesUrine:             "Normais",
				VaccinesAndDeworming:   "Em dia",
				Medications:            s.meds,
				EnvironmentAndContacts: s.summary,
			},
			CreatedAt: date,
		}
		consultations = append(consultations, c)
		consultObjs = append(consultObjs, c)
	}
	if _, err := db.Collection("consultations").InsertMany(ctx, consultObjs); err != nil {
		return err
	}
	log.Printf("Created %d test consultations", len(consultations))

	// --- Vaccines (species-appropriate) ------------------------------------
	dogVaccines := []string{"V10 (Déctupla)", "Antirrábica", "Giárdia", "Tosse dos Canis (Bordetella)"}
	catVaccines := []string{"V4 (Quádrupla Felina)", "Antirrábica", "Leucemia Felina (FeLV)"}
	rabbitVaccines := []string{"Mixomatose", "Doença Hemorrágica Viral"}
	manufacturers := []string{"Zoetis", "MSD Saúde Animal", "Boehringer Ingelheim", "Ceva"}

	vaccinesFor := func(species string) []string {
		switch species {
		case "Gato":
			return catVaccines
		case "Coelho":
			return rabbitVaccines
		default:
			return dogVaccines
		}
	}

	var vaccines []interface{}
	vaccineCount := 0
	for i, pet := range pets {
		names := vaccinesFor(pet.Species)
		// Each pet gets 2 vaccine records: one applied in the past, and one with
		// an upcoming next dose. A couple are tuned to fire notifications.
		for j := 0; j < 2; j++ {
			applied := now.AddDate(0, -(3 + j*2), -(i % 20))
			var next time.Time
			switch {
			case i == 0 && j == 0:
				next = dayAt(1) // fires "amanhã" notification
			case i == 1 && j == 0:
				next = dayAt(7) // fires "7 dias" notification
			case i == 2 && j == 0:
				next = dayAt(0) // fires "hoje" notification
			default:
				next = now.AddDate(0, 4+((i+j)%8), 0) // 4..11 months out
			}
			v := &models.Vaccine{
				ID:           primitive.NewObjectID(),
				VetID:        vetForPet(pet),
				PetID:        pet.ID,
				VaccineName:  names[(i+j)%len(names)],
				Batch:        fmt.Sprintf("LTE-%d%02d", 2024+j, (i*7+j)%99),
				Manufacturer: manufacturers[(i+j)%len(manufacturers)],
				DateApplied:  applied,
				NextDoseDate: next,
				Notes:        "Aplicada sem intercorrências",
				CreatedAt:    applied,
			}
			vaccines = append(vaccines, v)
			vaccineCount++
		}
	}
	if _, err := db.Collection("vaccines").InsertMany(ctx, vaccines); err != nil {
		return err
	}
	log.Printf("Created %d test vaccines", vaccineCount)

	// --- Catalog (services + products) per vet -----------------------------
	type catSpec struct {
		typ         models.CatalogItemType
		name, desc  string
		price       float64
		unit        string
		stock, minS float64
	}
	catSpecs := []catSpec{
		{models.CatalogService, "Consulta clínica", "Consulta veterinária de rotina", 120.00, "un", 0, 0},
		{models.CatalogService, "Consulta de retorno", "Retorno em até 15 dias", 0.00, "un", 0, 0},
		{models.CatalogService, "Banho e tosa", "Higiene completa", 90.00, "un", 0, 0},
		{models.CatalogService, "Castração", "Procedimento cirúrgico de castração", 450.00, "un", 0, 0},
		{models.CatalogService, "Aplicação de vacina", "Taxa de aplicação", 35.00, "un", 0, 0},
		{models.CatalogProduct, "Vacina V10", "Déctupla canina", 95.00, "un", 40, 10},
		{models.CatalogProduct, "Vacina Antirrábica", "Dose antirrábica", 60.00, "un", 25, 8},
		{models.CatalogProduct, "Vermífugo (comprimido)", "Vermífugo de amplo espectro", 28.00, "cx", 6, 10},
		{models.CatalogProduct, "Ração terapêutica 2kg", "Dieta renal/gastrointestinal", 180.00, "un", 12, 5},
		{models.CatalogProduct, "Antipulgas (pipeta)", "Proteção mensal", 55.00, "un", 3, 6},
	}
	var catalog []interface{}
	catalogByVet := map[primitive.ObjectID][]*models.CatalogItem{}
	for _, vetID := range []primitive.ObjectID{vet1.ID, vet2.ID} {
		for _, s := range catSpecs {
			item := &models.CatalogItem{
				ID:            primitive.NewObjectID(),
				VetID:         vetID,
				Type:          s.typ,
				Name:          s.name,
				Description:   s.desc,
				Price:         s.price,
				Unit:          s.unit,
				StockQuantity: s.stock,
				MinStock:      s.minS,
				Active:        true,
				CreatedAt:     now.AddDate(0, -6, 0),
				UpdatedAt:     now,
			}
			catalog = append(catalog, item)
			catalogByVet[vetID] = append(catalogByVet[vetID], item)
		}
	}
	if _, err := db.Collection("catalog_items").InsertMany(ctx, catalog); err != nil {
		return err
	}
	log.Printf("Created %d catalog items", len(catalog))

	// --- Invoices ----------------------------------------------------------
	// A few invoices across statuses so the financial dashboard has data.
	type invSpec struct {
		petIdx    int
		daysAgo   int
		status    models.PaymentStatus
		method    models.PaymentMethod
		items     []models.InvoiceItem
		discount  float64
		dueInDays int // relative to issue; used for pending/overdue
	}
	invSpecs := []invSpec{
		{0, 20, models.PaymentPaid, models.MethodPix, []models.InvoiceItem{{Description: "Consulta clínica", Quantity: 1, UnitPrice: 120, Total: 120}, {Description: "Aplicação de vacina", Quantity: 1, UnitPrice: 35, Total: 35}}, 0, 0},
		{1, 15, models.PaymentPaid, models.MethodCard, []models.InvoiceItem{{Description: "Consulta clínica", Quantity: 1, UnitPrice: 120, Total: 120}}, 0, 0},
		{2, 45, models.PaymentPaid, models.MethodCash, []models.InvoiceItem{{Description: "Castração", Quantity: 1, UnitPrice: 450, Total: 450}}, 50, 0},
		{3, 5, models.PaymentPending, models.MethodUnassigned, []models.InvoiceItem{{Description: "Banho e tosa", Quantity: 1, UnitPrice: 90, Total: 90}}, 0, 10},
		{6, 2, models.PaymentPending, models.MethodUnassigned, []models.InvoiceItem{{Description: "Consulta clínica", Quantity: 1, UnitPrice: 120, Total: 120}, {Description: "Vermífugo (comprimido)", Quantity: 1, UnitPrice: 28, Total: 28}}, 0, 15},
		{5, 40, models.PaymentOverdue, models.MethodUnassigned, []models.InvoiceItem{{Description: "Consulta clínica", Quantity: 1, UnitPrice: 120, Total: 120}}, 0, -10},
	}
	var invoices []interface{}
	for _, s := range invSpecs {
		pet := pets[s.petIdx]
		issue := now.AddDate(0, 0, -s.daysAgo)
		subtotal := 0.0
		for _, it := range s.items {
			subtotal += it.Total
		}
		total := subtotal - s.discount
		inv := &models.Invoice{
			ID:        primitive.NewObjectID(),
			VetID:     vetForPet(pet),
			ClientID:  clientForPet(pet),
			PetID:     pet.ID,
			Items:     s.items,
			Subtotal:  subtotal,
			Discount:  s.discount,
			Total:     total,
			Status:    s.status,
			Method:    s.method,
			IssueDate: issue,
			Notes:     "",
			CreatedAt: issue,
			UpdatedAt: now,
		}
		if s.status == models.PaymentPaid {
			paid := issue.AddDate(0, 0, 1)
			inv.PaidAt = &paid
		} else {
			due := issue.AddDate(0, 0, s.dueInDays)
			inv.DueDate = &due
		}
		invoices = append(invoices, inv)
	}
	if _, err := db.Collection("invoices").InsertMany(ctx, invoices); err != nil {
		return err
	}
	log.Printf("Created %d invoices", len(invoices))

	// --- Appointments (upcoming agenda) ------------------------------------
	type apptSpec struct {
		petIdx       int
		inDays       int
		hour         int
		durationMin  int
		typ          models.AppointmentType
		status       models.AppointmentStatus
		title, notes string
	}
	apptSpecs := []apptSpec{
		{0, 7, 9, 30, models.AppointmentReturn, models.AppointmentScheduled, "Retorno - Thor", "Reavaliar pata traseira"},
		{1, 1, 10, 30, models.AppointmentReturn, models.AppointmentConfirmed, "Retorno - Nina", "Verificar quadro gástrico"},
		{3, 2, 14, 60, models.AppointmentGrooming, models.AppointmentScheduled, "Banho e tosa - Amora", ""},
		{4, 3, 11, 30, models.AppointmentVaccine, models.AppointmentScheduled, "Vacinação - Mel", "V10 2ª dose"},
		{5, 5, 15, 90, models.AppointmentSurgery, models.AppointmentScheduled, "Castração - Rex", "Jejum de 8h antes"},
		{6, 0, 16, 30, models.AppointmentConsultation, models.AppointmentConfirmed, "Consulta - Pingo", "Dificuldade urinária"},
		{2, -3, 9, 30, models.AppointmentConsultation, models.AppointmentCompleted, "Consulta - Bidu", "Check-up concluído"},
	}
	var appts []interface{}
	for _, s := range apptSpecs {
		pet := pets[s.petIdx]
		start := dayAt(s.inDays).Add(time.Duration(s.hour-9) * time.Hour)
		a := &models.Appointment{
			ID:        primitive.NewObjectID(),
			VetID:     vetForPet(pet),
			PetID:     pet.ID,
			ClientID:  clientForPet(pet),
			Title:     s.title,
			Type:      s.typ,
			Status:    s.status,
			StartTime: start,
			EndTime:   start.Add(time.Duration(s.durationMin) * time.Minute),
			Notes:     s.notes,
			CreatedAt: now,
			UpdatedAt: now,
		}
		appts = append(appts, a)
	}
	if _, err := db.Collection("appointments").InsertMany(ctx, appts); err != nil {
		return err
	}
	log.Printf("Created %d appointments", len(appts))

	// --- Prescriptions (linked to the consultations above) -----------------
	prescriptions := []interface{}{
		&models.Prescription{
			ID:    primitive.NewObjectID(),
			VetID: vetForPet(pets[0]),
			PetID: pets[0].ID,
			Date:  now.AddDate(0, 0, -20),
			Items: []models.PrescriptionItem{
				{Medication: "Carprofeno 25mg", Dosage: "1 comprimido", Route: "Oral", Frequency: "1x ao dia", Duration: "7 dias", Instructions: "Administrar junto à alimentação"},
			},
			GeneralNotes: "Repouso e evitar esforço físico.",
			CreatedAt:    now.AddDate(0, 0, -20), UpdatedAt: now,
		},
		&models.Prescription{
			ID:    primitive.NewObjectID(),
			VetID: vetForPet(pets[1]),
			PetID: pets[1].ID,
			Date:  now.AddDate(0, 0, -15),
			Items: []models.PrescriptionItem{
				{Medication: "Omeprazol 10mg", Dosage: "1 comprimido", Route: "Oral", Frequency: "1x ao dia em jejum", Duration: "5 dias", Instructions: "30 min antes da refeição"},
				{Medication: "Dieta branda", Dosage: "-", Route: "Oral", Frequency: "3x ao dia", Duration: "3 dias", Instructions: "Frango cozido sem tempero com arroz"},
			},
			GeneralNotes: "Retornar se os vômitos persistirem.",
			CreatedAt:    now.AddDate(0, 0, -15), UpdatedAt: now,
		},
		&models.Prescription{
			ID:    primitive.NewObjectID(),
			VetID: vetForPet(pets[3]),
			PetID: pets[3].ID,
			Date:  now.AddDate(0, 0, -10),
			Items: []models.PrescriptionItem{
				{Medication: "Prednisolona 20mg", Dosage: "dose decrescente", Route: "Oral", Frequency: "1x ao dia", Duration: "10 dias", Instructions: "Reduzir pela metade a cada 3 dias"},
			},
			GeneralNotes: "Banho com shampoo medicamentoso 2x por semana.",
			CreatedAt:    now.AddDate(0, 0, -10), UpdatedAt: now,
		},
	}
	if _, err := db.Collection("prescriptions").InsertMany(ctx, prescriptions); err != nil {
		return err
	}
	log.Printf("Created %d prescriptions", len(prescriptions))

	log.Printf("Demo visitor login -> email=%s password=%s (non-production only)",
		DemoVisitorEmail, DemoVisitorPassword)

	return nil
}

// createUserEmailIndex creates a unique index on the email field.
func createUserEmailIndex(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("users")
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: &options.IndexOptions{Unique: &[]bool{true}[0]},
	}
	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	return err
}
