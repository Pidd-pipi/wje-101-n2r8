package dto

import "github.com/wjecoffeetaste/wjecoffeetaste/internal/model"

// RecipeCreateRequest creates a brew recipe together with its first version.
type RecipeCreateRequest struct {
	Name      string `json:"name" binding:"required,max=128"`
	Device    string `json:"device" binding:"omitempty,max=64"`
	WaterTemp int    `json:"water_temp"`
	GrindSize string `json:"grind_size" binding:"omitempty,max=32"`
	Ratio     string `json:"ratio" binding:"omitempty,max=32"`
	Steps     string `json:"steps"`
}

// RecipeVersionCreateRequest appends a new immutable version to a recipe.
type RecipeVersionCreateRequest struct {
	Name      string `json:"name" binding:"required,max=128"`
	Device    string `json:"device" binding:"omitempty,max=64"`
	WaterTemp int    `json:"water_temp"`
	GrindSize string `json:"grind_size" binding:"omitempty,max=32"`
	Ratio     string `json:"ratio" binding:"omitempty,max=32"`
	Steps     string `json:"steps"`
}

// RecipeVersionStatusRequest flips a version between active and deprecated.
type RecipeVersionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active deprecated"`
}

// RecipeCurrentVersionRequest selects which version is used next time.
type RecipeCurrentVersionRequest struct {
	VersionNumber int `json:"version_number" binding:"required,min=1"`
}

// RecipeDetailResponse is a recipe plus its full version history.
type RecipeDetailResponse struct {
	model.BrewRecipe
	Versions []model.RecipeVersion `json:"versions"`
}
