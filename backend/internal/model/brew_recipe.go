package model

import "time"

// BrewRecipe is the identity of a shared coffee brewing recipe. All mutable
// brewing parameters live in BrewRecipeVersion; a recipe points at the
// version used for the next brew via ActiveVersionID.
type BrewRecipe struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index;not null" json:"user_id"`
	Name            string    `gorm:"size:128;not null" json:"name"`
	Device          string    `gorm:"size:64;index" json:"device"`
	CurrentVersion  int       `gorm:"not null;default:0" json:"current_version"`
	ActiveVersionID uint      `gorm:"index" json:"active_version_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BrewRecipeVersion is one immutable revision of a recipe. Deprecated
// versions are kept forever so old tasting notes can always be checked
// against the exact parameters used at brew time.
type BrewRecipeVersion struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RecipeID      uint      `gorm:"index:idx_recipe_version,unique;not null" json:"recipe_id"`
	VersionNumber int       `gorm:"index:idx_recipe_version,unique;not null" json:"version_number"`
	WaterTemp     int       `json:"water_temp"`
	GrindSize     string    `gorm:"size:32" json:"grind_size"`
	Ratio         string    `gorm:"size:32" json:"ratio"`
	Steps         string    `gorm:"type:json" json:"steps"`
	Status        string    `gorm:"size:16;index;not null;default:active" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// Recipe version statuses. A version is "active" when it is the one selected
// for the next brew; publishing a new revision or selecting another version
// marks the previously active one "deprecated" without deleting it.
const (
	RecipeVersionActive     = "active"
	RecipeVersionDeprecated = "deprecated"
)
