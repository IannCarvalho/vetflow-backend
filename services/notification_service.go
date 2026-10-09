package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationService interface {
	GenerateNotifications(ctx context.Context, vetID primitive.ObjectID) error
	GetNotifications(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error)
	MarkAsRead(ctx context.Context, notificationID primitive.ObjectID) error
	Dismiss(ctx context.Context, notificationID primitive.ObjectID) error
}

type notificationService struct {
	notificationRepo repositories.NotificationRepository
	consultationRepo repositories.ConsultationRepository
	vaccineRepo      repositories.VaccineRepository
	petRepo          repositories.PetRepository
}

func NewNotificationService(
	notificationRepo repositories.NotificationRepository,
	consultationRepo repositories.ConsultationRepository,
	vaccineRepo repositories.VaccineRepository,
	petRepo repositories.PetRepository,
) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
		consultationRepo: consultationRepo,
		vaccineRepo:      vaccineRepo,
		petRepo:          petRepo,
	}
}

func (s *notificationService) GenerateNotifications(ctx context.Context, vetID primitive.ObjectID) error {
	now := time.Now()

	// Collect the reminders that are actually due, so we only touch the pets
	// those reminders reference instead of one FindByID per record (N+1).
	type reminder struct {
		petID      primitive.ObjectID
		resourceID string
		nType      models.NotificationType
		targetDate time.Time
	}

	var reminders []reminder
	petIDSet := make(map[primitive.ObjectID]struct{})

	addReminder := func(petID primitive.ObjectID, resourceID string, nType models.NotificationType, targetDate time.Time) {
		if _, ok := daysUntilIfDue(targetDate, now); !ok {
			return
		}
		reminders = append(reminders, reminder{petID, resourceID, nType, targetDate})
		petIDSet[petID] = struct{}{}
	}

	// Get consultations by vet
	consultations, err := s.consultationRepo.FindByVetID(ctx, vetID)
	if err != nil {
		return err
	}
	for _, consultation := range consultations {
		if consultation.ReturnDate != nil {
			addReminder(consultation.PetID, consultation.ID.Hex(),
				models.NotificationConsultation, *consultation.ReturnDate)
		}
	}

	// Get vaccines by vet
	vaccines, err := s.vaccineRepo.FindByVetID(ctx, vetID)
	if err != nil {
		return err
	}
	for _, vaccine := range vaccines {
		if !vaccine.NextDoseDate.IsZero() {
			addReminder(vaccine.PetID, vaccine.ID.Hex(),
				models.NotificationVaccine, vaccine.NextDoseDate)
		}
	}

	if len(reminders) == 0 {
		return nil
	}

	// Batch-load every referenced pet in a single query and index by id.
	petIDs := make([]primitive.ObjectID, 0, len(petIDSet))
	for id := range petIDSet {
		petIDs = append(petIDs, id)
	}
	pets, err := s.petRepo.FindByIDs(ctx, petIDs)
	if err != nil {
		return err
	}
	petByID := make(map[primitive.ObjectID]*models.Pet, len(pets))
	for _, pet := range pets {
		petByID[pet.ID] = pet
	}

	for _, r := range reminders {
		pet, ok := petByID[r.petID]
		if !ok || pet == nil {
			// Pet was deleted after the consultation/vaccine was recorded; skip.
			continue
		}
		if err := s.createNotification(ctx, vetID, pet, r.resourceID, r.nType, r.targetDate, now); err != nil {
			return err
		}
	}

	return nil
}

// daysUntilIfDue returns the number of whole days (UTC, midnight-truncated)
// between now and targetDate, and whether a reminder is due on that offset
// (7 days before, 1 day before, or the day itself).
func daysUntilIfDue(targetDate, now time.Time) (int, bool) {
	targetDateUTC := targetDate.UTC()
	nowUTC := now.UTC()

	targetMidnight := time.Date(targetDateUTC.Year(), targetDateUTC.Month(), targetDateUTC.Day(), 0, 0, 0, 0, time.UTC)
	nowMidnight := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)

	daysUntil := int(targetMidnight.Sub(nowMidnight).Hours() / 24)
	due := daysUntil == 7 || daysUntil == 1 || daysUntil == 0
	return daysUntil, due
}

// createNotification builds and persists a single reminder, skipping it when a
// pending notification for the same vet/resource/type/target date already
// exists (idempotent across repeated sweeps).
func (s *notificationService) createNotification(ctx context.Context, vetID primitive.ObjectID,
	pet *models.Pet, resourceID string, notificationType models.NotificationType,
	targetDate time.Time, now time.Time) error {

	daysUntil, due := daysUntilIfDue(targetDate, now)
	if !due {
		return nil
	}

	exists, err := s.notificationRepo.ExistsPending(ctx, vetID, resourceID, notificationType, targetDate)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	title, message := notificationContent(notificationType, pet.Name, daysUntil)

	notification := &models.Notification{
		VetID:      vetID,
		PetID:      pet.ID,
		ResourceID: resourceID,
		Type:       notificationType,
		Title:      title,
		Message:    message,
		DaysUntil:  daysUntil,
		TargetDate: targetDate,
		Status:     models.NotificationStatusPending,
	}

	return s.notificationRepo.Create(ctx, notification)
}

// notificationContent returns the title and message for a reminder, keyed by
// type and how many days remain.
func notificationContent(nType models.NotificationType, petName string, daysUntil int) (title, message string) {
	switch nType {
	case models.NotificationConsultation:
		title = fmt.Sprintf("Retorno de %s", petName)
		switch daysUntil {
		case 7:
			message = fmt.Sprintf("Consulta de retorno de %s está agendada para daqui a 7 dias", petName)
		case 1:
			message = fmt.Sprintf("Consulta de retorno de %s está agendada para amanhã", petName)
		case 0:
			message = fmt.Sprintf("Hoje é o dia da consulta de retorno de %s", petName)
		}
	case models.NotificationVaccine:
		title = fmt.Sprintf("Vacina de %s", petName)
		switch daysUntil {
		case 7:
			message = fmt.Sprintf("Próxima dose de vacina de %s está agendada para daqui a 7 dias", petName)
		case 1:
			message = fmt.Sprintf("Próxima dose de vacina de %s está agendada para amanhã", petName)
		case 0:
			message = fmt.Sprintf("Hoje é o dia da próxima dose de vacina de %s", petName)
		}
	}
	return title, message
}

func (s *notificationService) GetNotifications(ctx context.Context, vetID primitive.ObjectID) ([]models.Notification, error) {
	return s.notificationRepo.GetByVetID(ctx, vetID)
}

func (s *notificationService) MarkAsRead(ctx context.Context, notificationID primitive.ObjectID) error {
	return s.notificationRepo.UpdateStatus(ctx, notificationID, models.NotificationStatusSent)
}

func (s *notificationService) Dismiss(ctx context.Context, notificationID primitive.ObjectID) error {
	return s.notificationRepo.UpdateStatus(ctx, notificationID, models.NotificationStatusDismissed)
}
