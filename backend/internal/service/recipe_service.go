package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// RecipeService handles brew recipes and their version history.
type RecipeService struct {
	repo   *repository.BrewRecipeRepository
	logger *slog.Logger
}

// NewRecipeService creates a RecipeService.
func NewRecipeService(repo *repository.BrewRecipeRepository, logger *slog.Logger) *RecipeService {
	return &RecipeService{repo: repo, logger: logger}
}

// Create shares a recipe with its initial version 1.
func (s *RecipeService) Create(userID uint, rec *model.BrewRecipe, ver *model.BrewRecipeVersion) (*model.BrewRecipe, *model.BrewRecipeVersion, error) {
	rec.UserID = userID
	if err := s.repo.Create(rec, ver); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogRecipeCreateFailed, rec.Name), "error", err)
		return nil, nil, fmt.Errorf("recipe create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeCreateSuccess, rec.Name), "id", rec.ID, "version", ver.VersionNumber)
	return rec, ver, nil
}

// Get returns a recipe by id.
func (s *RecipeService) Get(id uint) (*model.BrewRecipe, error) {
	rec, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("BrewRecipe[id=%d] not found", id))
		}
		return nil, fmt.Errorf("recipe get: %w", err)
	}
	return rec, nil
}

// GetVersion returns one version of a recipe. Deprecated versions remain
// readable so old tasting notes can always be cross-checked.
func (s *RecipeService) GetVersion(recipeID uint, versionNumber int) (*model.BrewRecipeVersion, error) {
	ver, err := s.repo.FindVersion(recipeID, versionNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("BrewRecipe[id=%d] version=%d not found", recipeID, versionNumber))
		}
		return nil, fmt.Errorf("recipe version get: %w", err)
	}
	return ver, nil
}

// GetVersionByID returns a version by its own id, optionally scoped to a recipe.
func (s *RecipeService) GetVersionByID(versionID uint, recipeID uint) (*model.BrewRecipeVersion, error) {
	ver, err := s.repo.FindVersionByID(versionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("BrewRecipeVersion[id=%d] not found", versionID))
		}
		return nil, fmt.Errorf("recipe version get: %w", err)
	}
	if recipeID != 0 && ver.RecipeID != recipeID {
		return nil, util.NewAppError(400, constants.CodeBadRequest,
			fmt.Sprintf("BrewRecipeVersion[id=%d] does not belong to recipe %d", versionID, recipeID))
	}
	return ver, nil
}

// GetDetail returns a recipe plus its active version and full history.
func (s *RecipeService) GetDetail(id uint) (*model.BrewRecipe, *model.BrewRecipeVersion, []model.BrewRecipeVersion, error) {
	rec, err := s.Get(id)
	if err != nil {
		return nil, nil, nil, err
	}
	versions, err := s.repo.ListVersions(id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("recipe versions list: %w", err)
	}
	var active *model.BrewRecipeVersion
	for i := range versions {
		if versions[i].ID == rec.ActiveVersionID {
			active = &versions[i]
			break
		}
	}
	return rec, active, versions, nil
}

// PublishVersion appends a new revision. Only the recipe owner may do this;
// the repository assigns the next consecutive version number atomically.
func (s *RecipeService) PublishVersion(userID, recipeID uint, patch *model.BrewRecipe, ver *model.BrewRecipeVersion) (*model.BrewRecipe, *model.BrewRecipeVersion, error) {
	rec, err := s.Get(recipeID)
	if err != nil {
		return nil, nil, err
	}
	if rec.UserID != userID {
		return nil, nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("BrewRecipe[id=%d] publish version failed: user_id=%d not owner", recipeID, userID))
	}
	patch.ID = recipeID
	created, err := s.repo.AddVersion(patch, ver)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, nil, util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("BrewRecipe[id=%d] version number already exists", recipeID))
		}
		return nil, nil, fmt.Errorf("recipe version create: %w", err)
	}
	// Reload so name/device/pointer reflect post-update state.
	rec, err = s.repo.FindByID(recipeID)
	if err != nil {
		return nil, nil, fmt.Errorf("recipe reload: %w", err)
	}
	s.logger.Info(constants.LogRecipeVersionPublished, "id", recipeID, "version", created.VersionNumber, "user_id", userID)
	return rec, created, nil
}

// SetActiveVersion chooses the version to use for the next brew. Only the
// owner may change it. The chosen version is usually deprecated beforehand.
func (s *RecipeService) SetActiveVersion(userID, recipeID, versionID uint) (*model.BrewRecipeVersion, error) {
	rec, err := s.Get(recipeID)
	if err != nil {
		return nil, err
	}
	if rec.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("BrewRecipe[id=%d] set active failed: user_id=%d not owner", recipeID, userID))
	}
	ver, err := s.repo.SetActiveVersion(recipeID, versionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("BrewRecipe[id=%d] version[id=%d] not found", recipeID, versionID))
		}
		return nil, fmt.Errorf("recipe set active: %w", err)
	}
	s.logger.Info(constants.LogRecipeActiveVersionSet, "id", recipeID, "version_id", versionID, "user_id", userID)
	return ver, nil
}

// ResolveVersionForNote returns the version a new tasting note should be
// pinned to: the explicitly requested one (validated against the recipe) or
// the recipe's active version.
func (s *RecipeService) ResolveVersionForNote(recipeID, versionID uint) (*model.BrewRecipeVersion, error) {
	rec, err := s.Get(recipeID)
	if err != nil {
		return nil, err
	}
	if versionID != 0 {
		return s.GetVersionByID(versionID, recipeID)
	}
	if rec.ActiveVersionID == 0 {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("BrewRecipe[id=%d] has no active version", recipeID))
	}
	return s.GetVersionByID(rec.ActiveVersionID, recipeID)
}

// RecipeListItem pairs a recipe with the parameters of its active version.
type RecipeListItem struct {
	model.BrewRecipe
	ActiveVersion *model.BrewRecipeVersion `json:"active_version"`
}

// List filters recipes and attaches each recipe's active version.
func (s *RecipeService) List(device, keyword string, page, pageSize int) ([]RecipeListItem, int64, error) {
	items, total, err := s.repo.List(device, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("recipe list: %w", err)
	}
	ids := make([]uint, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	active, err := s.repo.FindActiveVersions(ids)
	if err != nil {
		return nil, 0, fmt.Errorf("recipe list versions: %w", err)
	}
	result := make([]RecipeListItem, 0, len(items))
	for i := range items {
		item := RecipeListItem{BrewRecipe: items[i]}
		if v, ok := active[items[i].ID]; ok {
			vv := v
			item.ActiveVersion = &vv
		}
		result = append(result, item)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeListSuccess, device), "total", total)
	return result, total, nil
}
