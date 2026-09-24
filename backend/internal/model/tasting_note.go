package model

import "time"

// TastingNote is a user's coffee tasting record. When a note references a
// recipe version, the name/water temperature/steps in effect at brew time
// are snapshotted so later recipe revisions never alter the historical
// record.
type TastingNote struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	UserID              uint      `gorm:"index;not null" json:"user_id"`
	CoffeeName          string    `gorm:"size:128;not null" json:"coffee_name"`
	Origin              string    `gorm:"size:128" json:"origin"`
	RoastLevel          string    `gorm:"size:16;index;not null" json:"roast_level"`
	FlavorTags          string    `gorm:"type:json" json:"flavor_tags"`
	AromaScore          float64   `json:"aroma_score"`
	AcidityScore        float64   `json:"acidity_score"`
	BodyScore           float64   `json:"body_score"`
	OverallScore        float64   `json:"overall_score"`
	BrewMethod          string    `gorm:"size:64" json:"brew_method"`
	BrewRecipeID        uint      `json:"brew_recipe_id"`
	BrewRecipeVersionID uint      `json:"brew_recipe_version_id"`
	RecipeNameSnapshot  string    `gorm:"size:128" json:"recipe_name_snapshot"`
	RecipeTempSnapshot  int       `json:"recipe_temp_snapshot"`
	RecipeStepsSnapshot string    `gorm:"type:json" json:"recipe_steps_snapshot"`
	RecipeVersionSnap   int       `json:"recipe_version_snapshot"`
	NotesText           string    `gorm:"type:text" json:"notes_text"`
	ImageURL            string    `gorm:"size:255" json:"image_url"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
