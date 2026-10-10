package services

import (
	"context"
	"testing"
	"time"

	"github.com/vetflow/veterinary-api/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- fakes -----------------------------------------------------------------

type fakeConsultationRepo struct {
	byVet map[primitive.ObjectID][]*models.Consultation
}

func (f *fakeConsultationRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Consultation, error) {
	return f.byVet[vetID], nil
}

// unused interface methods
func (f *fakeConsultationRepo) Create(context.Context, *models.Consultation) error { return nil }
func (f *fakeConsultationRepo) FindByID(context.Context, primitive.ObjectID) (*models.Consultation, error) {
	return nil, nil
}
func (f *fakeConsultationRepo) FindAll(context.Context) ([]*models.Consultation, error) {
	return nil, nil
}
func (f *fakeConsultationRepo) FindByPetID(context.Context, primitive.ObjectID) ([]*models.Consultation, error) {
	return nil, nil
}
func (f *fakeConsultationRepo) Update(context.Context, primitive.ObjectID, *models.Consultation) error {
	return nil
}
func (f *fakeConsultationRepo) Delete(context.Context, primitive.ObjectID) error        { return nil }
func (f *fakeConsultationRepo) DeleteByPetID(context.Context, primitive.ObjectID) error { return nil }
func (f *fakeConsultationRepo) CountByVetIDInRange(context.Context, primitive.ObjectID, time.Time, time.Time) (int64, error) {
	return 0, nil
}

type fakeVaccineRepo struct {
	byVet map[primitive.ObjectID][]*models.Vaccine
}

func (f *fakeVaccineRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Vaccine, error) {
	return f.byVet[vetID], nil
}

func (f *fakeVaccineRepo) Create(context.Context, *models.Vaccine) error { return nil }
func (f *fakeVaccineRepo) FindByID(context.Context, primitive.ObjectID) (*models.Vaccine, error) {
	return nil, nil
}
func (f *fakeVaccineRepo) FindAll(context.Context) ([]*models.Vaccine, error) { return nil, nil }
func (f *fakeVaccineRepo) FindByPetID(context.Context, primitive.ObjectID) ([]*models.Vaccine, error) {
	return nil, nil
}
func (f *fakeVaccineRepo) Update(context.Context, primitive.ObjectID, *models.Vaccine) error {
	return nil
}
func (f *fakeVaccineRepo) Delete(context.Context, primitive.ObjectID) error        { return nil }
func (f *fakeVaccineRepo) DeleteByPetID(context.Context, primitive.ObjectID) error { return nil }
func (f *fakeVaccineRepo) CountByVetIDNextDoseInRange(context.Context, primitive.ObjectID, time.Time, time.Time) (int64, error) {
	return 0, nil
}

type fakePetRepo struct {
	pets           map[primitive.ObjectID]*models.Pet
	findByIDCalls  int
	findByIDsCalls int
}

func (f *fakePetRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Pet, error) {
	f.findByIDCalls++
	return f.pets[id], nil
}

func (f *fakePetRepo) FindByIDs(_ context.Context, ids []primitive.ObjectID) ([]*models.Pet, error) {
	f.findByIDsCalls++
	out := make([]*models.Pet, 0, len(ids))
	for _, id := range ids {
		if p, ok := f.pets[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakePetRepo) Create(context.Context, *models.Pet) error      { return nil }
func (f *fakePetRepo) FindAll(context.Context) ([]*models.Pet, error) { return nil, nil }
func (f *fakePetRepo) FindByOwnerID(context.Context, primitive.ObjectID) ([]*models.Pet, error) {
	return nil, nil
}
func (f *fakePetRepo) FindByVetID(context.Context, primitive.ObjectID) ([]*models.Pet, error) {
	return nil, nil
}
func (f *fakePetRepo) Update(context.Context, primitive.ObjectID, *models.Pet) error { return nil }
func (f *fakePetRepo) Delete(context.Context, primitive.ObjectID) error              { return nil }
func (f *fakePetRepo) DeleteByClientID(context.Context, primitive.ObjectID) error    { return nil }
func (f *fakePetRepo) CountByVetID(context.Context, primitive.ObjectID) (int64, error) {
	return 0, nil
}

type fakeNotificationRepo struct {
	created []*models.Notification
	// existing simulates already-persisted pending notifications keyed by
	// vet|resource|type|day.
	existing map[string]bool
}

func dedupeKey(vetID primitive.ObjectID, resourceID string, nType models.NotificationType, targetDate time.Time) string {
	t := targetDate.UTC()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return vetID.Hex() + "|" + resourceID + "|" + string(nType) + "|" + day.Format("2006-01-02")
}

func (f *fakeNotificationRepo) Create(_ context.Context, n *models.Notification) error {
	f.created = append(f.created, n)
	if f.existing == nil {
		f.existing = map[string]bool{}
	}
	f.existing[dedupeKey(n.VetID, n.ResourceID, n.Type, n.TargetDate)] = true
	return nil
}

func (f *fakeNotificationRepo) ExistsPending(_ context.Context, vetID primitive.ObjectID, resourceID string, nType models.NotificationType, targetDate time.Time) (bool, error) {
	return f.existing[dedupeKey(vetID, resourceID, nType, targetDate)], nil
}

func (f *fakeNotificationRepo) GetByVetID(context.Context, primitive.ObjectID) ([]models.Notification, error) {
	return nil, nil
}
func (f *fakeNotificationRepo) GetPending(context.Context, primitive.ObjectID) ([]models.Notification, error) {
	return nil, nil
}
func (f *fakeNotificationRepo) UpdateStatus(context.Context, primitive.ObjectID, models.NotificationStatus) error {
	return nil
}
func (f *fakeNotificationRepo) Delete(context.Context, primitive.ObjectID) error    { return nil }
func (f *fakeNotificationRepo) DeleteOld(context.Context, primitive.ObjectID) error { return nil }

// --- helpers ---------------------------------------------------------------

func newTestService(cons *fakeConsultationRepo, vac *fakeVaccineRepo, pet *fakePetRepo, notif *fakeNotificationRepo) *notificationService {
	return &notificationService{
		notificationRepo: notif,
		consultationRepo: cons,
		vaccineRepo:      vac,
		petRepo:          pet,
	}
}

// --- tests -----------------------------------------------------------------

// TestGenerateNotifications_BatchesPetLookups asserts the service loads pets in
// one FindByIDs call instead of one FindByID per due record (the old N+1).
func TestGenerateNotifications_BatchesPetLookups(t *testing.T) {
	vetID := primitive.NewObjectID()
	in7 := time.Now().AddDate(0, 0, 7)

	// Two pets, two due consultations each -> 4 due records referencing 2 pets.
	petA := &models.Pet{ID: primitive.NewObjectID(), Name: "Rex"}
	petB := &models.Pet{ID: primitive.NewObjectID(), Name: "Mia"}

	cons := &fakeConsultationRepo{byVet: map[primitive.ObjectID][]*models.Consultation{
		vetID: {
			{ID: primitive.NewObjectID(), VetID: vetID, PetID: petA.ID, ReturnDate: &in7},
			{ID: primitive.NewObjectID(), VetID: vetID, PetID: petA.ID, ReturnDate: &in7},
			{ID: primitive.NewObjectID(), VetID: vetID, PetID: petB.ID, ReturnDate: &in7},
		},
	}}
	vac := &fakeVaccineRepo{byVet: map[primitive.ObjectID][]*models.Vaccine{
		vetID: {{ID: primitive.NewObjectID(), VetID: vetID, PetID: petB.ID, NextDoseDate: in7}},
	}}
	pet := &fakePetRepo{pets: map[primitive.ObjectID]*models.Pet{petA.ID: petA, petB.ID: petB}}
	notif := &fakeNotificationRepo{}

	svc := newTestService(cons, vac, pet, notif)
	if err := svc.GenerateNotifications(context.Background(), vetID); err != nil {
		t.Fatalf("GenerateNotifications: %v", err)
	}

	if pet.findByIDCalls != 0 {
		t.Errorf("expected no per-record FindByID calls, got %d", pet.findByIDCalls)
	}
	if pet.findByIDsCalls != 1 {
		t.Errorf("expected exactly 1 batch FindByIDs call, got %d", pet.findByIDsCalls)
	}
	if len(notif.created) != 4 {
		t.Errorf("expected 4 notifications created, got %d", len(notif.created))
	}
}

// TestGenerateNotifications_SkipsDuplicates asserts a second sweep does not
// re-create a notification that already exists pending (the duplication bug).
func TestGenerateNotifications_SkipsDuplicates(t *testing.T) {
	vetID := primitive.NewObjectID()
	in1 := time.Now().AddDate(0, 0, 1)

	pet := &models.Pet{ID: primitive.NewObjectID(), Name: "Rex"}
	consID := primitive.NewObjectID()

	cons := &fakeConsultationRepo{byVet: map[primitive.ObjectID][]*models.Consultation{
		vetID: {{ID: consID, VetID: vetID, PetID: pet.ID, ReturnDate: &in1}},
	}}
	vac := &fakeVaccineRepo{byVet: map[primitive.ObjectID][]*models.Vaccine{}}
	petRepo := &fakePetRepo{pets: map[primitive.ObjectID]*models.Pet{pet.ID: pet}}
	notif := &fakeNotificationRepo{}

	svc := newTestService(cons, vac, petRepo, notif)
	ctx := context.Background()

	// First sweep creates one notification.
	if err := svc.GenerateNotifications(ctx, vetID); err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	// Second sweep (same day) must create nothing new.
	if err := svc.GenerateNotifications(ctx, vetID); err != nil {
		t.Fatalf("second sweep: %v", err)
	}

	if len(notif.created) != 1 {
		t.Errorf("expected 1 notification after two sweeps (deduped), got %d", len(notif.created))
	}
	if got := notif.created[0].ResourceID; got != consID.Hex() {
		t.Errorf("expected ResourceID %q persisted for dedupe, got %q", consID.Hex(), got)
	}
}

// TestGenerateNotifications_IgnoresNonDueDates asserts records outside the
// 7/1/0-day windows produce nothing and trigger no pet load.
func TestGenerateNotifications_IgnoresNonDueDates(t *testing.T) {
	vetID := primitive.NewObjectID()
	in3 := time.Now().AddDate(0, 0, 3) // not a notify window

	pet := &models.Pet{ID: primitive.NewObjectID(), Name: "Rex"}
	cons := &fakeConsultationRepo{byVet: map[primitive.ObjectID][]*models.Consultation{
		vetID: {{ID: primitive.NewObjectID(), VetID: vetID, PetID: pet.ID, ReturnDate: &in3}},
	}}
	vac := &fakeVaccineRepo{byVet: map[primitive.ObjectID][]*models.Vaccine{}}
	petRepo := &fakePetRepo{pets: map[primitive.ObjectID]*models.Pet{pet.ID: pet}}
	notif := &fakeNotificationRepo{}

	svc := newTestService(cons, vac, petRepo, notif)
	if err := svc.GenerateNotifications(context.Background(), vetID); err != nil {
		t.Fatalf("GenerateNotifications: %v", err)
	}

	if len(notif.created) != 0 {
		t.Errorf("expected 0 notifications for a non-due date, got %d", len(notif.created))
	}
	if petRepo.findByIDsCalls != 0 {
		t.Errorf("expected no pet load when nothing is due, got %d FindByIDs calls", petRepo.findByIDsCalls)
	}
}
