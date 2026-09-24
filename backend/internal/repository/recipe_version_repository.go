package repository

import (
	"gorm.io/gorm"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
)

// RecipeVersionRepository handles read/write access to immutable recipe versions.
type RecipeVersionRepository struct{ db *gorm.DB }

// NewRecipeVersionRepository creates the repository.
func NewRecipeVersionRepository(db *gorm.DB) *RecipeVersionRepository {
	return &RecipeVersionRepository{db: db}
}

// Create inserts a version row.
func (r *RecipeVersionRepository) Create(v *model.RecipeVersion) error {
	return translate(r.db.Create(v).Error)
}

// FindByID locates a version by its own id.
func (r *RecipeVersionRepository) FindByID(id uint) (*model.RecipeVersion, error) {
	var v model.RecipeVersion
	if err := translate(r.db.First(&v, id).Error); err != nil {
		return nil, err
	}
	return &v, nil
}

// FindByNumber locates a version within one recipe by its sequence number.
func (r *RecipeVersionRepository) FindByNumber(recipeID uint, versionNumber int) (*model.RecipeVersion, error) {
	var v model.RecipeVersion
	if err := translate(r.db.Where("recipe_id = ? AND version_number = ?", recipeID, versionNumber).
		First(&v).Error); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListByRecipe returns all versions of a recipe, newest first.
func (r *RecipeVersionRepository) ListByRecipe(recipeID uint) ([]model.RecipeVersion, error) {
	var items []model.RecipeVersion
	if err := r.db.Where("recipe_id = ?", recipeID).
		Order("version_number DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// UpdateStatus flips a version between active and deprecated.
func (r *RecipeVersionRepository) UpdateStatus(id uint, status string) error {
	res := r.db.Model(&model.RecipeVersion{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
