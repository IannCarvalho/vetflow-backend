package controllers

import (
	"context"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// This file provides in-memory fake repositories backed by maps, satisfying the
// interfaces in repositories/interfaces.go. They mirror the mongo repos'
// observable behavior: FindByID returns (nil, nil) when absent, Create assigns a
// new ObjectID when zero and stores a copy, and Update/Delete return
// mongo.ErrNoDocuments when the target is missing. Only the methods exercised by
// the controller tests carry real logic; the rest exist to satisfy the
// interfaces and return zero values.

// --- User ---

type fakeUserRepo struct {
	byEmail map[string]*models.User
}

var _ repositories.UserRepository = (*fakeUserRepo)(nil)

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: make(map[string]*models.User)}
}

func (r *fakeUserRepo) Create(_ context.Context, user *models.User) error {
	if _, ok := r.byEmail[user.Email]; ok {
		return repositories.ErrEmailAlreadyRegistered
	}
	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}
	stored := *user
	r.byEmail[user.Email] = &stored
	return nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*models.User, error) {
	user, ok := r.byEmail[email]
	if !ok {
		return nil, nil
	}
	copied := *user
	return &copied, nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id string) (*models.User, error) {
	for _, u := range r.byEmail {
		if u.ID.Hex() == id {
			copied := *u
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepo) FindByObjectID(_ context.Context, id primitive.ObjectID) (*models.User, error) {
	for _, u := range r.byEmail {
		if u.ID == id {
			copied := *u
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepo) FindAll(_ context.Context) ([]*models.User, error) {
	out := make([]*models.User, 0, len(r.byEmail))
	for _, u := range r.byEmail {
		copied := *u
		out = append(out, &copied)
	}
	return out, nil
}

func (r *fakeUserRepo) UpdateUser(_ context.Context, _ primitive.ObjectID, _ map[string]interface{}) (*models.User, error) {
	return nil, nil
}

// --- Client ---

type fakeClientRepo struct {
	items map[primitive.ObjectID]*models.Client
}

var _ repositories.ClientRepository = (*fakeClientRepo)(nil)

func newFakeClientRepo() *fakeClientRepo {
	return &fakeClientRepo{items: make(map[primitive.ObjectID]*models.Client)}
}

func (r *fakeClientRepo) Create(_ context.Context, client *models.Client) error {
	if client.ID.IsZero() {
		client.ID = primitive.NewObjectID()
	}
	stored := *client
	r.items[client.ID] = &stored
	return nil
}

func (r *fakeClientRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Client, error) {
	client, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *client
	return &copied, nil
}

func (r *fakeClientRepo) FindAll(_ context.Context) ([]*models.Client, error) {
	out := make([]*models.Client, 0, len(r.items))
	for _, c := range r.items {
		copied := *c
		out = append(out, &copied)
	}
	return out, nil
}

func (r *fakeClientRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Client, error) {
	out := make([]*models.Client, 0)
	for _, c := range r.items {
		if c.VetID == vetID {
			copied := *c
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *fakeClientRepo) Update(_ context.Context, id primitive.ObjectID, client *models.Client) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *client
	stored.ID = id
	r.items[id] = &stored
	return nil
}

func (r *fakeClientRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}

// --- Pet ---

type fakePetRepo struct {
	items map[primitive.ObjectID]*models.Pet
	// clients lets FindByVetID resolve a pet's owning client to the vet scope,
	// mirroring the aggregation $lookup in the mongo pet repository.
	clients *fakeClientRepo
}

var _ repositories.PetRepository = (*fakePetRepo)(nil)

func newFakePetRepo(clients *fakeClientRepo) *fakePetRepo {
	return &fakePetRepo{items: make(map[primitive.ObjectID]*models.Pet), clients: clients}
}

func (r *fakePetRepo) Create(_ context.Context, pet *models.Pet) error {
	if pet.ID.IsZero() {
		pet.ID = primitive.NewObjectID()
	}
	stored := *pet
	r.items[pet.ID] = &stored
	return nil
}

func (r *fakePetRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Pet, error) {
	pet, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *pet
	return &copied, nil
}

func (r *fakePetRepo) FindByIDs(_ context.Context, ids []primitive.ObjectID) ([]*models.Pet, error) {
	out := make([]*models.Pet, 0, len(ids))
	for _, id := range ids {
		if p, ok := r.items[id]; ok {
			copied := *p
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *fakePetRepo) FindAll(_ context.Context) ([]*models.Pet, error) {
	out := make([]*models.Pet, 0, len(r.items))
	for _, p := range r.items {
		copied := *p
		out = append(out, &copied)
	}
	return out, nil
}

func (r *fakePetRepo) FindByOwnerID(_ context.Context, ownerID primitive.ObjectID) ([]*models.Pet, error) {
	out := make([]*models.Pet, 0)
	for _, p := range r.items {
		if p.OwnerID == ownerID {
			copied := *p
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *fakePetRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Pet, error) {
	out := make([]*models.Pet, 0)
	for _, p := range r.items {
		if r.clients == nil {
			continue
		}
		owner, ok := r.clients.items[p.OwnerID]
		if ok && owner.VetID == vetID {
			copied := *p
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *fakePetRepo) Update(_ context.Context, id primitive.ObjectID, pet *models.Pet) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *pet
	stored.ID = id
	r.items[id] = &stored
	return nil
}

func (r *fakePetRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}

func (r *fakePetRepo) DeleteByClientID(_ context.Context, clientID primitive.ObjectID) error {
	for id, p := range r.items {
		if p.OwnerID == clientID {
			delete(r.items, id)
		}
	}
	return nil
}

// --- Vaccine ---

type fakeVaccineRepo struct {
	items map[primitive.ObjectID]*models.Vaccine
}

var _ repositories.VaccineRepository = (*fakeVaccineRepo)(nil)

func newFakeVaccineRepo() *fakeVaccineRepo {
	return &fakeVaccineRepo{items: make(map[primitive.ObjectID]*models.Vaccine)}
}

func (r *fakeVaccineRepo) Create(_ context.Context, vaccine *models.Vaccine) error {
	if vaccine.ID.IsZero() {
		vaccine.ID = primitive.NewObjectID()
	}
	stored := *vaccine
	r.items[vaccine.ID] = &stored
	return nil
}

func (r *fakeVaccineRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Vaccine, error) {
	vaccine, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *vaccine
	return &copied, nil
}

func (r *fakeVaccineRepo) FindAll(_ context.Context) ([]*models.Vaccine, error) {
	out := make([]*models.Vaccine, 0, len(r.items))
	for _, v := range r.items {
		copied := *v
		out = append(out, &copied)
	}
	return out, nil
}

func (r *fakeVaccineRepo) FindByPetID(_ context.Context, petID primitive.ObjectID) ([]*models.Vaccine, error) {
	out := make([]*models.Vaccine, 0)
	for _, v := range r.items {
		if v.PetID == petID {
			copied := *v
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *fakeVaccineRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Vaccine, error) {
	out := make([]*models.Vaccine, 0)
	for _, v := range r.items {
		if v.VetID == vetID {
			copied := *v
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *fakeVaccineRepo) Update(_ context.Context, id primitive.ObjectID, vaccine *models.Vaccine) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *vaccine
	stored.ID = id
	r.items[id] = &stored
	return nil
}

func (r *fakeVaccineRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}

func (r *fakeVaccineRepo) DeleteByPetID(_ context.Context, petID primitive.ObjectID) error {
	for id, v := range r.items {
		if v.PetID == petID {
			delete(r.items, id)
		}
	}
	return nil
}

// --- Consultation (map-backed so FindByVetID returns real counts) ---

type fakeConsultationRepo struct {
	items map[primitive.ObjectID]*models.Consultation
}

var _ repositories.ConsultationRepository = (*fakeConsultationRepo)(nil)

func newFakeConsultationRepo() *fakeConsultationRepo {
	return &fakeConsultationRepo{items: make(map[primitive.ObjectID]*models.Consultation)}
}

func (r *fakeConsultationRepo) Create(_ context.Context, consultation *models.Consultation) error {
	if consultation.ID.IsZero() {
		consultation.ID = primitive.NewObjectID()
	}
	stored := *consultation
	r.items[consultation.ID] = &stored
	return nil
}
func (r *fakeConsultationRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Consultation, error) {
	consultation, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *consultation
	return &copied, nil
}
func (r *fakeConsultationRepo) FindAll(context.Context) ([]*models.Consultation, error) {
	out := make([]*models.Consultation, 0, len(r.items))
	for _, c := range r.items {
		copied := *c
		out = append(out, &copied)
	}
	return out, nil
}
func (r *fakeConsultationRepo) FindByPetID(_ context.Context, petID primitive.ObjectID) ([]*models.Consultation, error) {
	out := make([]*models.Consultation, 0)
	for _, c := range r.items {
		if c.PetID == petID {
			copied := *c
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeConsultationRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Consultation, error) {
	out := make([]*models.Consultation, 0)
	for _, c := range r.items {
		if c.VetID == vetID {
			copied := *c
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeConsultationRepo) Update(_ context.Context, id primitive.ObjectID, consultation *models.Consultation) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *consultation
	stored.ID = id
	r.items[id] = &stored
	return nil
}
func (r *fakeConsultationRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}
func (r *fakeConsultationRepo) DeleteByPetID(_ context.Context, petID primitive.ObjectID) error {
	for id, c := range r.items {
		if c.PetID == petID {
			delete(r.items, id)
		}
	}
	return nil
}

// --- Prescription (map-backed so FindByVetID returns real counts) ---

type fakePrescriptionRepo struct {
	items map[primitive.ObjectID]*models.Prescription
}

var _ repositories.PrescriptionRepository = (*fakePrescriptionRepo)(nil)

func newFakePrescriptionRepo() *fakePrescriptionRepo {
	return &fakePrescriptionRepo{items: make(map[primitive.ObjectID]*models.Prescription)}
}

func (r *fakePrescriptionRepo) Create(_ context.Context, prescription *models.Prescription) error {
	if prescription.ID.IsZero() {
		prescription.ID = primitive.NewObjectID()
	}
	stored := *prescription
	r.items[prescription.ID] = &stored
	return nil
}
func (r *fakePrescriptionRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Prescription, error) {
	prescription, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *prescription
	return &copied, nil
}
func (r *fakePrescriptionRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Prescription, error) {
	out := make([]*models.Prescription, 0)
	for _, p := range r.items {
		if p.VetID == vetID {
			copied := *p
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakePrescriptionRepo) FindByPetID(_ context.Context, petID primitive.ObjectID) ([]*models.Prescription, error) {
	out := make([]*models.Prescription, 0)
	for _, p := range r.items {
		if p.PetID == petID {
			copied := *p
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakePrescriptionRepo) Update(_ context.Context, id primitive.ObjectID, prescription *models.Prescription) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *prescription
	stored.ID = id
	r.items[id] = &stored
	return nil
}
func (r *fakePrescriptionRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}
func (r *fakePrescriptionRepo) DeleteByPetID(_ context.Context, petID primitive.ObjectID) error {
	for id, p := range r.items {
		if p.PetID == petID {
			delete(r.items, id)
		}
	}
	return nil
}

// --- CatalogItem (map-backed so FindByVetID returns real counts) ---

type fakeCatalogItemRepo struct {
	items map[primitive.ObjectID]*models.CatalogItem
}

var _ repositories.CatalogItemRepository = (*fakeCatalogItemRepo)(nil)

func newFakeCatalogItemRepo() *fakeCatalogItemRepo {
	return &fakeCatalogItemRepo{items: make(map[primitive.ObjectID]*models.CatalogItem)}
}

func (r *fakeCatalogItemRepo) Create(_ context.Context, item *models.CatalogItem) error {
	if item.ID.IsZero() {
		item.ID = primitive.NewObjectID()
	}
	stored := *item
	r.items[item.ID] = &stored
	return nil
}
func (r *fakeCatalogItemRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.CatalogItem, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *item
	return &copied, nil
}
func (r *fakeCatalogItemRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.CatalogItem, error) {
	out := make([]*models.CatalogItem, 0)
	for _, it := range r.items {
		if it.VetID == vetID {
			copied := *it
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeCatalogItemRepo) Update(_ context.Context, id primitive.ObjectID, item *models.CatalogItem) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *item
	stored.ID = id
	r.items[id] = &stored
	return nil
}
func (r *fakeCatalogItemRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}

// --- Invoice (map-backed so FindByVetID returns real counts) ---

type fakeInvoiceRepo struct {
	items map[primitive.ObjectID]*models.Invoice
}

var _ repositories.InvoiceRepository = (*fakeInvoiceRepo)(nil)

func newFakeInvoiceRepo() *fakeInvoiceRepo {
	return &fakeInvoiceRepo{items: make(map[primitive.ObjectID]*models.Invoice)}
}

func (r *fakeInvoiceRepo) Create(_ context.Context, invoice *models.Invoice) error {
	if invoice.ID.IsZero() {
		invoice.ID = primitive.NewObjectID()
	}
	stored := *invoice
	r.items[invoice.ID] = &stored
	return nil
}
func (r *fakeInvoiceRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Invoice, error) {
	invoice, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *invoice
	return &copied, nil
}
func (r *fakeInvoiceRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Invoice, error) {
	out := make([]*models.Invoice, 0)
	for _, inv := range r.items {
		if inv.VetID == vetID {
			copied := *inv
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeInvoiceRepo) FindByClientID(_ context.Context, clientID primitive.ObjectID) ([]*models.Invoice, error) {
	out := make([]*models.Invoice, 0)
	for _, inv := range r.items {
		if inv.ClientID == clientID {
			copied := *inv
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeInvoiceRepo) Update(_ context.Context, id primitive.ObjectID, invoice *models.Invoice) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *invoice
	stored.ID = id
	r.items[id] = &stored
	return nil
}
func (r *fakeInvoiceRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}
func (r *fakeInvoiceRepo) DeleteByClientID(_ context.Context, clientID primitive.ObjectID) error {
	for id, inv := range r.items {
		if inv.ClientID == clientID {
			delete(r.items, id)
		}
	}
	return nil
}

// --- Appointment (map-backed so FindByVetID returns real counts) ---

type fakeAppointmentRepo struct {
	items map[primitive.ObjectID]*models.Appointment
}

var _ repositories.AppointmentRepository = (*fakeAppointmentRepo)(nil)

func newFakeAppointmentRepo() *fakeAppointmentRepo {
	return &fakeAppointmentRepo{items: make(map[primitive.ObjectID]*models.Appointment)}
}

func (r *fakeAppointmentRepo) Create(_ context.Context, appointment *models.Appointment) error {
	if appointment.ID.IsZero() {
		appointment.ID = primitive.NewObjectID()
	}
	stored := *appointment
	r.items[appointment.ID] = &stored
	return nil
}
func (r *fakeAppointmentRepo) FindByID(_ context.Context, id primitive.ObjectID) (*models.Appointment, error) {
	appointment, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copied := *appointment
	return &copied, nil
}
func (r *fakeAppointmentRepo) FindByVetID(_ context.Context, vetID primitive.ObjectID) ([]*models.Appointment, error) {
	out := make([]*models.Appointment, 0)
	for _, a := range r.items {
		if a.VetID == vetID {
			copied := *a
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeAppointmentRepo) FindByVetIDInRange(_ context.Context, vetID primitive.ObjectID, start, end time.Time) ([]*models.Appointment, error) {
	out := make([]*models.Appointment, 0)
	for _, a := range r.items {
		if a.VetID == vetID && !a.StartTime.Before(start) && !a.StartTime.After(end) {
			copied := *a
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeAppointmentRepo) FindByPetID(_ context.Context, petID primitive.ObjectID) ([]*models.Appointment, error) {
	out := make([]*models.Appointment, 0)
	for _, a := range r.items {
		if a.PetID == petID {
			copied := *a
			out = append(out, &copied)
		}
	}
	return out, nil
}
func (r *fakeAppointmentRepo) Update(_ context.Context, id primitive.ObjectID, appointment *models.Appointment) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	stored := *appointment
	stored.ID = id
	r.items[id] = &stored
	return nil
}
func (r *fakeAppointmentRepo) Delete(_ context.Context, id primitive.ObjectID) error {
	if _, ok := r.items[id]; !ok {
		return mongo.ErrNoDocuments
	}
	delete(r.items, id)
	return nil
}
func (r *fakeAppointmentRepo) DeleteByPetID(_ context.Context, petID primitive.ObjectID) error {
	for id, a := range r.items {
		if a.PetID == petID {
			delete(r.items, id)
		}
	}
	return nil
}
func (r *fakeAppointmentRepo) DeleteByClientID(_ context.Context, clientID primitive.ObjectID) error {
	for id, a := range r.items {
		if a.ClientID == clientID {
			delete(r.items, id)
		}
	}
	return nil
}

// --- Attachment (satisfies the interface; no exercised logic) ---

type fakeAttachmentRepo struct{}

var _ repositories.AttachmentRepository = (*fakeAttachmentRepo)(nil)

func newFakeAttachmentRepo() *fakeAttachmentRepo { return &fakeAttachmentRepo{} }

func (r *fakeAttachmentRepo) Create(context.Context, *models.Attachment) error { return nil }
func (r *fakeAttachmentRepo) FindByID(context.Context, primitive.ObjectID) (*models.Attachment, error) {
	return nil, nil
}
func (r *fakeAttachmentRepo) FindSummariesByPetID(context.Context, primitive.ObjectID) ([]*models.AttachmentSummary, error) {
	return nil, nil
}
func (r *fakeAttachmentRepo) Update(context.Context, primitive.ObjectID, map[string]interface{}) error {
	return nil
}
func (r *fakeAttachmentRepo) Delete(context.Context, primitive.ObjectID) error        { return nil }
func (r *fakeAttachmentRepo) DeleteByPetID(context.Context, primitive.ObjectID) error { return nil }
