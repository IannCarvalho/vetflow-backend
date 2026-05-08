package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Anamnesis struct {
	MainComplaint    string `bson:"main_complaint" json:"main_complaint"`
	EvolutionTime    string `bson:"evolution_time" json:"evolution_time"`
	SymptomsWorsened bool   `bson:"symptoms_worsened" json:"symptoms_worsened"`
	SymptomFrequency string `bson:"symptom_frequency" json:"symptom_frequency"`
	BehaviorChange   bool   `bson:"behavior_change" json:"behavior_change"`
	Pain             bool   `bson:"pain" json:"pain"`
	HappenedBefore   bool   `bson:"happened_before" json:"happened_before"`
	PriorVetVisit    bool   `bson:"prior_vet_visit" json:"prior_vet_visit"`
}

type Diet struct {
	FoodType            string `bson:"food_type" json:"food_type"`
	AmountAndFrequency  string `bson:"amount_and_frequency" json:"amount_and_frequency"`
	DietChangedRecently bool   `bson:"diet_changed_recently" json:"diet_changed_recently"`
	NormalWaterIntake   bool   `bson:"normal_water_intake" json:"normal_water_intake"`
}

type Physiological struct {
	NormalUrineFeces  bool `bson:"normal_urine_feces" json:"normal_urine_feces"`
	Vomiting          bool `bson:"vomiting" json:"vomiting"`
	RespiratoryIssues bool `bson:"respiratory_issues" json:"respiratory_issues"`
}

type Summary struct {
	EvolutionTime          string `bson:"evolution_time" json:"evolution_time"`
	FecesUrine             string `bson:"feces_urine" json:"feces_urine"`
	VaccinesAndDeworming   string `bson:"vaccines_and_deworming" json:"vaccines_and_deworming"`
	Medications            string `bson:"medications" json:"medications"`
	EnvironmentAndContacts string `bson:"environment_and_contacts" json:"environment_and_contacts"`
}

type Consultation struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	VetID         primitive.ObjectID `bson:"vet_id" json:"vet_id"`
	PetID         primitive.ObjectID `bson:"pet_id" json:"pet_id"`
	Date          time.Time          `bson:"date" json:"date"`
	ReturnDate    *time.Time         `bson:"return_date,omitempty" json:"return_date,omitempty"`
	Anamnesis     Anamnesis          `bson:"anamnesis" json:"anamnesis"`
	Diet          Diet               `bson:"diet" json:"diet"`
	Physiological Physiological      `bson:"physiological" json:"physiological"`
	Summary       Summary            `bson:"summary" json:"summary"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
}
