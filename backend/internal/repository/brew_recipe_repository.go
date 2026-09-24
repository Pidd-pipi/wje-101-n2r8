package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
)

// BrewRecipeRepository handles recipe persistence.
type BrewRecipeRepository struct{ db *gorm.DB }

// NewBrewRecipeRepository creates the repository.
func NewBrewRecipeRepository(db *gorm.DB) *BrewRecipeRepository { return &BrewRecipeRepository{db: db} }

// Create inserts a recipe.
func (r *BrewRecipeRepository) Create(rec *model.BrewRecipe) error {
	return translate(r.db.Create(rec).Error)
}

// FindByID locates a recipe by id.
func (r *BrewRecipeRepository) FindByID(id uint) (*model.BrewRecipe, error) {
	var rec model.BrewRecipe
	if err := translate(r.db.First(&rec, id).Error); err != nil {
		return nil, err
	}
	tmp := []model.BrewRecipe{rec}
	if err := r.fillCurrentVersionNumber(tmp); err != nil {
		return nil, err
	}
	rec.CurrentVersionNumber = tmp[0].CurrentVersionNumber
	return &rec, nil
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
		q = q.Where("name LIKE ? OR grind_size LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if err := r.fillCurrentVersionNumber(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// fillCurrentVersionNumber populates the transient current_version_number
// field from the versions pointed at by current_version_id.
func (r *BrewRecipeRepository) fillCurrentVersionNumber(items []model.BrewRecipe) error {
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		if it.CurrentVersionID != 0 {
			ids = append(ids, it.CurrentVersionID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var versions []model.RecipeVersion
	if err := r.db.Select("id", "version_number").Where("id IN ?", ids).Find(&versions).Error; err != nil {
		return err
	}
	numberByID := make(map[uint]int, len(versions))
	for _, v := range versions {
		numberByID[v.ID] = v.VersionNumber
	}
	for i := range items {
		items[i].CurrentVersionNumber = numberByID[items[i].CurrentVersionID]
	}
	return nil
}

// CreateWithFirstVersion inserts a recipe and its version 1 atomically.
// The recipe's current-version pointer and mirrored columns are set from
// that first version.
func (r *BrewRecipeRepository) CreateWithFirstVersion(rec *model.BrewRecipe, v *model.RecipeVersion) error {
	return translate(r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rec).Error; err != nil {
			return translate(err)
		}
		v.RecipeID = rec.ID
		v.VersionNumber = 1
		if v.Status == "" {
			v.Status = constants.VersionStatusActive
		}
		if err := tx.Create(v).Error; err != nil {
			return translate(err)
		}
		return applyVersionToRecipe(tx, rec, v)
	}))
}

// AddVersion appends the next consecutive version and makes it current.
// The recipe row is locked for the duration so concurrent publishes cannot
// allocate the same version number; the unique index is the backstop.
func (r *BrewRecipeRepository) AddVersion(recipeID uint, v *model.RecipeVersion) (*model.BrewRecipe, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var rec model.BrewRecipe
		// SELECT ... FOR UPDATE serializes version allocation per recipe.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&rec, recipeID).Error; err != nil {
			return translate(err)
		}
		var maxNumber int
		if err := tx.Model(&model.RecipeVersion{}).
			Where("recipe_id = ?", recipeID).
			Select("COALESCE(MAX(version_number), 0)").Scan(&maxNumber).Error; err != nil {
			return err
		}
		v.RecipeID = recipeID
		v.VersionNumber = maxNumber + 1
		if v.Status == "" {
			v.Status = constants.VersionStatusActive
		}
		if err := tx.Create(v).Error; err != nil {
			return translate(err)
		}
		return applyVersionToRecipe(tx, &rec, v)
	})
	if err != nil {
		return nil, translate(err)
	}
	return r.FindByID(recipeID)
}

// SetCurrentVersion points the recipe at an existing version and mirrors that
// version's data onto the recipe row ("use this version next time").
func (r *BrewRecipeRepository) SetCurrentVersion(recipeID uint, v *model.RecipeVersion) error {
	return translate(r.db.Transaction(func(tx *gorm.DB) error {
		var rec model.BrewRecipe
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&rec, recipeID).Error; err != nil {
			return translate(err)
		}
		return applyVersionToRecipe(tx, &rec, v)
	}))
}

// applyVersionToRecipe updates the current-version pointer and the mirrored
// columns of a recipe from a version row.
func applyVersionToRecipe(tx *gorm.DB, rec *model.BrewRecipe, v *model.RecipeVersion) error {
	rec.CurrentVersionID = v.ID
	rec.Name = v.Name
	rec.Device = v.Device
	rec.WaterTemp = v.WaterTemp
	rec.GrindSize = v.GrindSize
	rec.Ratio = v.Ratio
	rec.Steps = v.Steps
	return translate(tx.Save(rec).Error)
}
