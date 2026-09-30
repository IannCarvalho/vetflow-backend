package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vetflow/veterinary-api/helpers"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
)

// NotificationScheduler periodically generates reminder notifications and
// e-mails each veterinarian a digest of the pending ones.
type NotificationScheduler struct {
	notificationService NotificationService
	notificationRepo    repositories.NotificationRepository
	userRepo            repositories.UserRepository
	mailer              *helpers.Mailer
	interval            time.Duration
}

// NewNotificationScheduler wires the scheduler dependencies. The interval
// controls how often the sweep runs (24h in production).
func NewNotificationScheduler(
	notificationService NotificationService,
	notificationRepo repositories.NotificationRepository,
	userRepo repositories.UserRepository,
	mailer *helpers.Mailer,
	interval time.Duration,
) *NotificationScheduler {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &NotificationScheduler{
		notificationService: notificationService,
		notificationRepo:    notificationRepo,
		userRepo:            userRepo,
		mailer:              mailer,
		interval:            interval,
	}
}

// Start launches the scheduler in a background goroutine. It runs one sweep
// shortly after startup and then every interval. It stops when ctx is done.
func (s *NotificationScheduler) Start(ctx context.Context) {
	go func() {
		// Small initial delay so startup isn't blocked by the first sweep.
		timer := time.NewTimer(1 * time.Minute)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[SCHEDULER] Notification scheduler stopped")
				return
			case <-timer.C:
				s.runSweep(ctx)
				timer.Reset(s.interval)
			}
		}
	}()
	log.Printf("[SCHEDULER] Notification scheduler started (interval=%s)", s.interval)
}

// runSweep processes every vet once.
func (s *NotificationScheduler) runSweep(ctx context.Context) {
	log.Println("[SCHEDULER] Running notification sweep...")

	users, err := s.userRepo.FindAll(ctx)
	if err != nil {
		log.Printf("[SCHEDULER] Failed to list users: %v", err)
		return
	}

	for _, user := range users {
		if err := s.processVet(ctx, user); err != nil {
			log.Printf("[SCHEDULER] Error processing vet %s: %v", user.ID.Hex(), err)
		}
	}

	log.Printf("[SCHEDULER] Sweep complete (%d vets)", len(users))
}

// processVet generates notifications for a vet and e-mails a digest of the
// pending ones, marking them as sent afterwards.
func (s *NotificationScheduler) processVet(ctx context.Context, user *models.User) error {
	if err := s.notificationService.GenerateNotifications(ctx, user.ID); err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	pending, err := s.notificationRepo.GetPending(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get pending: %w", err)
	}
	if len(pending) == 0 {
		return nil
	}

	// Only e-mail if we have an address for the vet.
	if user.Email != "" {
		subject := fmt.Sprintf("VetFlow: %d lembrete(s) para hoje", len(pending))
		body := buildDigestBody(user.Name, pending)
		if err := s.mailer.Send(user.Email, subject, body); err != nil {
			// Do not mark as sent if the e-mail failed; try again next sweep.
			return fmt.Errorf("send email: %w", err)
		}
	}

	// Mark the pending notifications as sent so they aren't re-e-mailed.
	for _, n := range pending {
		if err := s.notificationRepo.UpdateStatus(ctx, n.ID, models.NotificationStatusSent); err != nil {
			log.Printf("[SCHEDULER] Failed to mark notification %s as sent: %v", n.ID.Hex(), err)
		}
	}

	return nil
}

// buildDigestBody composes a plain-text digest of pending reminders.
func buildDigestBody(vetName string, notifications []models.Notification) string {
	var b strings.Builder
	greeting := "Olá"
	if vetName != "" {
		greeting = "Olá, " + vetName
	}
	b.WriteString(fmt.Sprintf("%s!\r\n\r\n", greeting))
	b.WriteString("Você tem os seguintes lembretes no VetFlow:\r\n\r\n")

	for _, n := range notifications {
		b.WriteString(fmt.Sprintf("- %s: %s\r\n", n.Title, n.Message))
	}

	b.WriteString("\r\nAcesse o VetFlow para ver os detalhes.\r\n")
	return b.String()
}
