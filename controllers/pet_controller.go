package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vetflow/veterinary-api/models"
	"github.com/vetflow/veterinary-api/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PetController struct {
	repo             repositories.PetRepository
	clientRepo       repositories.ClientRepository
	consultationRepo repositories.ConsultationRepository
	vaccineRepo      repositories.VaccineRepository
}

// NewPetController creates a new pet controller with dependency injection
func NewPetController(
	repo repositories.PetRepository,
	clientRepo repositories.ClientRepository,
	consultationRepo repositories.ConsultationRepository,
	vaccineRepo repositories.VaccineRepository,
) *PetController {
	return &PetController{
		repo:             repo,
		clientRepo:       clientRepo,
		consultationRepo: consultationRepo,
		vaccineRepo:      vaccineRepo,
	}
}

// Create godoc
// @Summary Create a new pet
// @Description Create a new pet record for an existing client
// @Tags Pets
// @Accept json
// @Produce json
// @Param pet body models.Pet true "Pet object with owner_id"
// @Success 201 {object} models.Pet
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 403 {object} map[string]string "Access denied"
// @Failure 500 {object} map[string]string "Server error"
// @Router /pets [post]
func (c *PetController) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var pet models.Pet
	if err := json.NewDecoder(r.Body).Decode(&pet); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validatePet(&pet); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Verify that the client (owner) belongs to this veterinarian
	client, err := c.clientRepo.FindByID(r.Context(), pet.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify client ownership")
		return
	}

	if client == nil {
		writeError(w, http.StatusBadRequest, "Client not found")
		return
	}

	if err := validateResourceOwnership(client.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, "You do not have permission to add pets to this client")
		return
	}

	pet.CreatedAt = time.Now()
	pet.UpdatedAt = time.Now()

	if err := c.repo.Create(r.Context(), &pet); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create pet")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pet)
}

// GetByID godoc
// @Summary Get pet by ID
// @Description Get a specific pet by their ID
// @Tags Pets
// @Accept json
// @Produce json
// @Param id path string true "Pet ID"
// @Success 200 {object} models.Pet
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 403 {object} map[string]string "Access denied"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /pets/{id} [get]
func (c *PetController) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid pet ID")
		return
	}

	pet, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve pet")
		return
	}

	if pet == nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}

	// Verify the client owner belongs to this veterinarian
	client, err := c.clientRepo.FindByID(r.Context(), pet.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify client ownership")
		return
	}

	if client == nil {
		writeError(w, http.StatusNotFound, "Client not found")
		return
	}

	if err := validateResourceOwnership(client.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pet)
}

// GetAll godoc
// @Summary List all pets
// @Description Retrieve all pets for the authenticated veterinarian
// @Tags Pets
// @Accept json
// @Produce json
// @Success 200 {array} models.Pet
// @Failure 500 {object} map[string]string "Server error"
// @Router /pets [get]
func (c *PetController) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var pets []*models.Pet

	pets, err = c.repo.FindByVetID(r.Context(), userID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve pets")
		return
	}

	if pets == nil {
		pets = make([]*models.Pet, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pets)
}

// GetAllPublic godoc (PUBLIC - NO AUTH REQUIRED)
// @Summary List all pets (public)
// @Description Retrieve all pets (public endpoint)
// @Tags Pets
// @Accept json
// @Produce json
// @Success 200 {array} models.Pet
// @Failure 500 {object} map[string]string "Server error"
// @Router /pets [get]
func (c *PetController) GetAllPublic(w http.ResponseWriter, r *http.Request) {
	var pets []*models.Pet

	pets, err := c.repo.FindAll(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve pets")
		return
	}

	if pets == nil {
		pets = make([]*models.Pet, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pets)
}
// GetByIDPublic godoc (PUBLIC - NO AUTH REQUIRED)
// @Summary Get pet by ID (public)
// @Description Get a specific pet by their ID (public endpoint)
// @Tags Pets
// @Accept json
// @Produce json
// @Param id path string true "Pet ID"
// @Success 200 {object} models.Pet
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /pets/{id} [get]
func (c *PetController) GetByIDPublic(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid pet ID")
		return
	}

	pet, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve pet")
		return
	}

	if pet == nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pet)
}
// Update godoc
// @Summary Update pet
// @Description Update an existing pet
// @Tags Pets
// @Accept json
// @Produce json
// @Param id path string true "Pet ID"
// @Param pet body models.Pet true "Updated pet object"
// @Success 200 {object} models.Pet
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 403 {object} map[string]string "Access denied"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /pets/{id} [put]
func (c *PetController) Update(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid pet ID")
		return
	}

	existingPet, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve pet")
		return
	}

	if existingPet == nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}

	// Verify the client owner belongs to this veterinarian
	client, err := c.clientRepo.FindByID(r.Context(), existingPet.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify client ownership")
		return
	}

	if client == nil {
		writeError(w, http.StatusNotFound, "Client not found")
		return
	}

	if err := validateResourceOwnership(client.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	var pet models.Pet
	if err := json.NewDecoder(r.Body).Decode(&pet); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validatePet(&pet); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	pet.ID = objectID
	pet.OwnerID = existingPet.OwnerID // Preserve the original owner
	pet.UpdatedAt = time.Now()

	if err := c.repo.Update(r.Context(), objectID, &pet); err != nil {
		if err == mongo.ErrNoDocuments {
			writeError(w, http.StatusNotFound, "Pet not found")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to update pet")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pet)
}

// Delete godoc
// @Summary Delete pet
// @Description Delete a pet by ID
// @Tags Pets
// @Accept json
// @Produce json
// @Param id path string true "Pet ID"
// @Success 204 "No content"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 403 {object} map[string]string "Access denied"
// @Failure 404 {object} map[string]string "Pet not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /pets/{id} [delete]
func (c *PetController) Delete(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid pet ID")
		return
	}

	pet, err := c.repo.FindByID(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve pet")
		return
	}

	if pet == nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}

	// Verify the client owner belongs to this veterinarian
	client, err := c.clientRepo.FindByID(r.Context(), pet.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify client ownership")
		return
	}

	if client == nil {
		writeError(w, http.StatusNotFound, "Client not found")
		return
	}

	if err := validateResourceOwnership(client.VetID, userID); err != nil {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	ctx := r.Context()

	// Cascade delete: First delete all consultations for this pet
	if err := c.consultationRepo.DeleteByPetID(ctx, objectID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete pet consultations")
		return
	}

	// Cascade delete: Then delete all vaccines for this pet
	if err := c.vaccineRepo.DeleteByPetID(ctx, objectID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete pet vaccines")
		return
	}

	// Finally delete the pet itself
	if err := c.repo.Delete(ctx, objectID); err != nil {
		if err == mongo.ErrNoDocuments {
			writeError(w, http.StatusNotFound, "Pet not found")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to delete pet")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validatePet validates required fields
func validatePet(pet *models.Pet) error {
	if pet.Name == "" {
		return errValidation("name is required")
	}
	if pet.Species == "" {
		return errValidation("species is required")
	}
	if pet.OwnerID.IsZero() {
		return errValidation("owner_id is required")
	}
	return nil
}
