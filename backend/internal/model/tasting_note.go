package model

import "time"

// TastingNote is a user's coffee tasting record.
//
// BrewRecipeID + BrewRecipeVersionID identify which recipe version this cup
// was brewed with. BrewRecipeName/WaterTemp/Steps are a snapshot taken at
// publish time, so later recipe edits (new versions) never rewrite what an
// old note displays.
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
	BrewRecipeName      string    `gorm:"size:128" json:"brew_recipe_name"`
	BrewRecipeWaterTemp int       `json:"brew_recipe_water_temp"`
	BrewRecipeSteps     string    `gorm:"type:json" json:"brew_recipe_steps"`
	NotesText           string    `gorm:"type:text" json:"notes_text"`
	ImageURL            string    `gorm:"size:255" json:"image_url"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
