package repository

import (
	"gorm.io/gorm"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
)

// BrewRecipeRepository handles recipe and recipe version persistence.
type BrewRecipeRepository struct{ db *gorm.DB }

// NewBrewRecipeRepository creates the repository.
func NewBrewRecipeRepository(db *gorm.DB) *BrewRecipeRepository { return &BrewRecipeRepository{db: db} }

// Create inserts a recipe together with its version 1 inside a transaction.
// The unique index on (recipe_id, version_number) guarantees version
// numbers can never repeat.
func (r *BrewRecipeRepository) Create(rec *model.BrewRecipe, ver *model.BrewRecipeVersion) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := translate(tx.Create(rec).Error); err != nil {
			return err
		}
		ver.RecipeID = rec.ID
		ver.VersionNumber = 1
		ver.Status = model.RecipeVersionActive
		if ver.Steps == "" {
			ver.Steps = "[]"
		}
		if err := translate(tx.Create(ver).Error); err != nil {
			return err
		}
		rec.CurrentVersion = 1
		rec.ActiveVersionID = ver.ID
		return translate(tx.Model(rec).Updates(map[string]any{
			"current_version":   1,
			"active_version_id": ver.ID,
		}).Error)
	})
}

// FindByID locates a recipe by id.
func (r *BrewRecipeRepository) FindByID(id uint) (*model.BrewRecipe, error) {
	var rec model.BrewRecipe
	if err := translate(r.db.First(&rec, id).Error); err != nil {
		return nil, err
	}
	return &rec, nil
}

// FindVersionByID locates a recipe version by id.
func (r *BrewRecipeRepository) FindVersionByID(id uint) (*model.BrewRecipeVersion, error) {
	var ver model.BrewRecipeVersion
	if err := translate(r.db.First(&ver, id).Error); err != nil {
		return nil, err
	}
	return &ver, nil
}

// FindVersion locates a specific (recipe, version number) pair.
func (r *BrewRecipeRepository) FindVersion(recipeID uint, versionNumber int) (*model.BrewRecipeVersion, error) {
	var ver model.BrewRecipeVersion
	err := translate(r.db.Where("recipe_id = ? AND version_number = ?", recipeID, versionNumber).First(&ver).Error)
	if err != nil {
		return nil, err
	}
	return &ver, nil
}

// AddVersion appends a new revision: the next consecutive version number is
// assigned, all previously active versions are deprecated, and the recipe's
// active pointer moves to the new version. Row locking serialises
// concurrent publishes so version numbers never repeat or skip.
func (r *BrewRecipeRepository) AddVersion(rec *model.BrewRecipe, ver *model.BrewRecipeVersion) (*model.BrewRecipeVersion, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var locked model.BrewRecipe
		if err := translate(tx.Clauses(clauseForUpdate()).First(&locked, rec.ID).Error); err != nil {
			return err
		}
		next := locked.CurrentVersion + 1
		ver.RecipeID = rec.ID
		ver.VersionNumber = next
		ver.Status = model.RecipeVersionActive
		if ver.Steps == "" {
			ver.Steps = "[]"
		}
		if err := translate(tx.Create(ver).Error); err != nil {
			return err
		}
		if err := tx.Model(&model.BrewRecipeVersion{}).
			Where("recipe_id = ? AND status = ?", rec.ID, model.RecipeVersionActive).
			Update("status", model.RecipeVersionDeprecated).Error; err != nil {
			return err
		}
		updates := map[string]any{"current_version": next, "active_version_id": ver.ID}
		if rec.Name != "" {
			updates["name"] = rec.Name
		}
		if rec.Device != "" {
			updates["device"] = rec.Device
		}
		if err := tx.Model(&model.BrewRecipe{}).Where("id = ?", rec.ID).Updates(updates).Error; err != nil {
			return err
		}
		rec.CurrentVersion = next
		rec.ActiveVersionID = ver.ID
		if rec.Name == "" {
			rec.Name = locked.Name
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ver, nil
}

// SetActiveVersion marks the chosen version active for the next brew and
// deprecates every other version of the recipe. Deprecated versions stay
// readable for old notes.
func (r *BrewRecipeRepository) SetActiveVersion(recipeID, versionID uint) (*model.BrewRecipeVersion, error) {
	var target model.BrewRecipeVersion
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var locked model.BrewRecipe
		if err := translate(tx.Clauses(clauseForUpdate()).First(&locked, recipeID).Error); err != nil {
			return err
		}
		if err := translate(tx.Clauses(clauseForUpdate()).
			Where("id = ? AND recipe_id = ?", versionID, recipeID).
			First(&target).Error); err != nil {
			return err
		}
		if err := tx.Model(&model.BrewRecipeVersion{}).
			Where("recipe_id = ?", recipeID).
			Update("status", model.RecipeVersionDeprecated).Error; err != nil {
			return err
		}
		if err := tx.Model(&target).Update("status", model.RecipeVersionActive).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.BrewRecipe{}).Where("id = ?", recipeID).
			Update("active_version_id", versionID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	target.Status = model.RecipeVersionActive
	return &target, nil
}

// ListVersions returns all versions of a recipe newest first.
func (r *BrewRecipeRepository) ListVersions(recipeID uint) ([]model.BrewRecipeVersion, error) {
	var versions []model.BrewRecipeVersion
	if err := r.db.Where("recipe_id = ?", recipeID).
		Order("version_number DESC").Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// List filters recipes by device/keyword.
func (r *BrewRecipeRepository) List(device, keyword string, page, pageSize int) ([]model.BrewRecipe, int64, error) {
	var items []model.BrewRecipe
	var total int64
	q := r.db.Model(&model.BrewRecipe{})
	if device != "" {
		q = q.Where("device = ?", device)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR device LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// FindActiveVersions returns the active version for each of the given
// recipes in one query, keyed by recipe id.
func (r *BrewRecipeRepository) FindActiveVersions(recipeIDs []uint) (map[uint]model.BrewRecipeVersion, error) {
	out := make(map[uint]model.BrewRecipeVersion)
	if len(recipeIDs) == 0 {
		return out, nil
	}
	var versions []model.BrewRecipeVersion
	if err := r.db.Where("recipe_id IN ? AND status = ?", recipeIDs, model.RecipeVersionActive).
		Find(&versions).Error; err != nil {
		return nil, err
	}
	for _, v := range versions {
		out[v.RecipeID] = v
	}
	return out, nil
}
