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

	// Get consultations by vet
	consultations, err := s.consultationRepo.FindByVetID(ctx, vetID)
	if err != nil {
		return err
	}

	// Process consultations with return dates
	for _, consultation := range consultations {
		if consultation.ReturnDate != nil {
			s.checkAndCreateNotification(ctx, vetID, consultation.PetID, consultation.ID.Hex(),
				models.NotificationConsultation, *consultation.ReturnDate, now)
		}
	}

	// Get vaccines by vet
	vaccines, err := s.vaccineRepo.FindByVetID(ctx, vetID)
	if err != nil {
		return err
	}

	// Process vaccines with next dose dates
	for _, vaccine := range vaccines {
		if !vaccine.NextDoseDate.IsZero() {
			s.checkAndCreateNotification(ctx, vetID, vaccine.PetID, vaccine.ID.Hex(),
				models.NotificationVaccine, vaccine.NextDoseDate, now)
		}
	}

	return nil
}

func (s *notificationService) checkAndCreateNotification(ctx context.Context, vetID primitive.ObjectID,
	petID primitive.ObjectID, resourceID string, notificationType models.NotificationType,
	targetDate time.Time, now time.Time) error {

	// Normalize both times to UTC for consistent date comparison
	targetDateUTC := targetDate.UTC()
	nowUTC := now.UTC()

	// Truncate to midnight for date comparison
	targetDateMidnight := time.Date(targetDateUTC.Year(), targetDateUTC.Month(), targetDateUTC.Day(), 0, 0, 0, 0, time.UTC)
	nowMidnight := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)

	// Calculate days until target date (using midnight times)
	daysUntil := int(targetDateMidnight.Sub(nowMidnight).Hours() / 24)

	// Check if notification should be created (7 days before, 1 day before, or on the day)
	shouldNotify := daysUntil == 7 || daysUntil == 1 || daysUntil == 0

	if !shouldNotify || daysUntil < 0 {
		return nil
	}

	// Get pet name for notification
	pet, err := s.petRepo.FindByID(ctx, petID)
	if err != nil {
		return err
	}

	// Create notification
	title := ""
	message := ""

	if notificationType == models.NotificationConsultation {
		title = fmt.Sprintf("Retorno de %s", pet.Name)
		switch daysUntil {
		case 7:
			message = fmt.Sprintf("Consulta de retorno de %s está agendada para daqui a 7 dias", pet.Name)
		case 1:
			message = fmt.Sprintf("Consulta de retorno de %s está agendada para amanhã", pet.Name)
		case 0:
			message = fmt.Sprintf("Hoje é o dia da consulta de retorno de %s", pet.Name)
		}
	} else if notificationType == models.NotificationVaccine {
		title = fmt.Sprintf("Vacina de %s", pet.Name)
		switch daysUntil {
		case 7:
			message = fmt.Sprintf("Próxima dose de vacina de %s está agendada para daqui a 7 dias", pet.Name)
		case 1:
			message = fmt.Sprintf("Próxima dose de vacina de %s está agendada para amanhã", pet.Name)
		case 0:
			message = fmt.Sprintf("Hoje é o dia da próxima dose de vacina de %s", pet.Name)
		}
	}

	notification := &models.Notification{
		VetID:      vetID,
		PetID:      petID,
		Type:       notificationType,
		Title:      title,
		Message:    message,
		DaysUntil:  daysUntil,
		TargetDate: targetDate,
		Status:     models.NotificationStatusPending,
	}

	err = s.notificationRepo.Create(ctx, notification)
	if err != nil {
		return err
	}
	return nil
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
