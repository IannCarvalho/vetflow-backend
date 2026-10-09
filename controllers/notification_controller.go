package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/services"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationController struct {
	notificationService services.NotificationService
}

func NewNotificationController(notificationService services.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
	}
}

func (c *NotificationController) Generate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	err = c.notificationService.GenerateNotifications(ctx, userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to generate notifications"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Notifications generated"}`))
}

func (c *NotificationController) GetAll(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	notifications, err := c.notificationService.GetNotifications(ctx, userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch notifications"}`, http.StatusInternalServerError)
		return
	}

	if notifications == nil {
		notifications = []models.Notification{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(notifications)
}

func (c *NotificationController) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	notificationID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, `{"error":"Invalid notification ID"}`, http.StatusBadRequest)
		return
	}

	err = c.notificationService.MarkAsRead(ctx, notificationID)
	if err != nil {
		http.Error(w, `{"error":"Failed to update notification"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Notification marked as read"}`))
}

func (c *NotificationController) Dismiss(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	notificationID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, `{"error":"Invalid notification ID"}`, http.StatusBadRequest)
		return
	}

	err = c.notificationService.Dismiss(ctx, notificationID)
	if err != nil {
		http.Error(w, `{"error":"Failed to dismiss notification"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"Notification dismissed"}`))
}
