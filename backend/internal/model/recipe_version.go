package model

import "time"

// RecipeVersion is an immutable snapshot of a brew recipe at one point in time.
// Once created its brewing data never changes; only Status may flip between
// active and deprecated. Old tasting notes keep referencing these rows so the
// exact data a cup was brewed with is always verifiable.
type RecipeVersion struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RecipeID      uint      `gorm:"not null;uniqueIndex:uniq_recipe_version_number,priority:1" json:"recipe_id"`
	VersionNumber int       `gorm:"not null;uniqueIndex:uniq_recipe_version_number,priority:2" json:"version_number"`
	Status        string    `gorm:"size:16;not null;default:active" json:"status"`
	Name          string    `gorm:"size:128;not null" json:"name"`
	Device        string    `gorm:"size:64" json:"device"`
	WaterTemp     int       `json:"water_temp"`
	GrindSize     string    `gorm:"size:32" json:"grind_size"`
	Ratio         string    `gorm:"size:32" json:"ratio"`
	Steps         string    `gorm:"type:json" json:"steps"`
	CreatedAt     time.Time `json:"created_at"`

	// IsCurrent is populated by the service when listing versions; not persisted.
	IsCurrent bool `gorm:"-" json:"is_current"`
}
