package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/dto"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// RecipeService handles brew recipes and their version history.
type RecipeService struct {
	repo        *repository.BrewRecipeRepository
	versionRepo *repository.RecipeVersionRepository
	logger      *slog.Logger
}

// NewRecipeService creates a RecipeService.
func NewRecipeService(repo *repository.BrewRecipeRepository, versionRepo *repository.RecipeVersionRepository, logger *slog.Logger) *RecipeService {
	return &RecipeService{repo: repo, versionRepo: versionRepo, logger: logger}
}

// Create shares a recipe together with its first (v1) version.
func (s *RecipeService) Create(userID uint, rec *model.BrewRecipe) (*model.BrewRecipe, error) {
	rec.UserID = userID
	if rec.Steps == "" {
		rec.Steps = "[]"
	}
	v := &model.RecipeVersion{
		Name: rec.Name, Device: rec.Device, WaterTemp: rec.WaterTemp,
		GrindSize: rec.GrindSize, Ratio: rec.Ratio, Steps: rec.Steps,
		Status: constants.VersionStatusActive,
	}
	if err := s.repo.CreateWithFirstVersion(rec, v); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogRecipeCreateFailed, rec.Name), "error", err)
		return nil, fmt.Errorf("recipe create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeCreateSuccess, rec.Name), "id", rec.ID, "version_id", v.ID)
	return rec, nil
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

// GetDetail returns a recipe and its full version history, marking which
// version is currently selected.
func (s *RecipeService) GetDetail(id uint) (*dto.RecipeDetailResponse, error) {
	rec, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	versions, err := s.versionRepo.ListByRecipe(id)
	if err != nil {
		return nil, fmt.Errorf("recipe versions list: %w", err)
	}
	for i := range versions {
		versions[i].IsCurrent = versions[i].ID == rec.CurrentVersionID
	}
	return &dto.RecipeDetailResponse{BrewRecipe: *rec, Versions: versions}, nil
}

// List filters recipes.
func (s *RecipeService) List(device, keyword string, page, pageSize int) ([]model.BrewRecipe, int64, error) {
	items, total, err := s.repo.List(device, keyword, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("recipe list: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeListSuccess, device), "total", total)
	return items, total, nil
}

// AddVersion publishes a new immutable version of the author's recipe and
// makes it the current version. Version numbers are allocated consecutively.
func (s *RecipeService) AddVersion(userID, recipeID uint, req *dto.RecipeVersionCreateRequest) (*model.RecipeVersion, error) {
	rec, err := s.requireOwner(userID, recipeID)
	if err != nil {
		return nil, err
	}
	steps := req.Steps
	if steps == "" {
		steps = "[]"
	}
	v := &model.RecipeVersion{
		Name: req.Name, Device: req.Device, WaterTemp: req.WaterTemp,
		GrindSize: req.GrindSize, Ratio: req.Ratio, Steps: steps,
		Status: constants.VersionStatusActive,
	}
	updated, err := s.repo.AddVersion(rec.ID, v)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("BrewRecipe[id=%d] version number conflict, please retry", recipeID))
		}
		return nil, fmt.Errorf("recipe add version: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeVersionCreated, rec.ID, v.VersionNumber), "version_id", v.ID)
	_ = updated
	return v, nil
}

// SetCurrentVersion selects the version used next time. Only the author may
// switch, and a deprecated version cannot be selected.
func (s *RecipeService) SetCurrentVersion(userID, recipeID uint, versionNumber int) error {
	rec, err := s.requireOwner(userID, recipeID)
	if err != nil {
		return err
	}
	v, err := s.versionRepo.FindByNumber(rec.ID, versionNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("BrewRecipe[id=%d] v%d not found", recipeID, versionNumber))
		}
		return fmt.Errorf("recipe version find: %w", err)
	}
	if v.Status == constants.VersionStatusDeprecated {
		return util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("BrewRecipe[id=%d] v%d is deprecated and cannot be current", recipeID, versionNumber))
	}
	if err := s.repo.SetCurrentVersion(rec.ID, v); err != nil {
		return fmt.Errorf("recipe set current version: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeCurrentSet, rec.ID, versionNumber))
	return nil
}

// SetVersionStatus marks a version active or deprecated. Deprecated versions
// remain stored and readable by old notes; they just cannot be chosen as the
// current version. Deprecating the currently selected version is rejected.
func (s *RecipeService) SetVersionStatus(userID, recipeID uint, versionNumber int, status string) error {
	rec, err := s.requireOwner(userID, recipeID)
	if err != nil {
		return err
	}
	if !constants.IsValidRecipeVersionStatus(status) {
		return util.NewAppError(400, constants.CodeBadRequest, "invalid version status")
	}
	v, err := s.versionRepo.FindByNumber(rec.ID, versionNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("BrewRecipe[id=%d] v%d not found", recipeID, versionNumber))
		}
		return fmt.Errorf("recipe version find: %w", err)
	}
	if status == constants.VersionStatusDeprecated && v.ID == rec.CurrentVersionID {
		return util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("BrewRecipe[id=%d] v%d is the current version, switch current before deprecating", recipeID, versionNumber))
	}
	if err := s.versionRepo.UpdateStatus(v.ID, status); err != nil {
		return fmt.Errorf("recipe version status update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogRecipeVersionStatus, rec.ID, versionNumber, status))
	return nil
}

// CurrentVersion returns the currently selected version of a recipe.
func (s *RecipeService) CurrentVersion(recipeID uint) (*model.RecipeVersion, error) {
	rec, err := s.repo.FindByID(recipeID)
	if err != nil {
		return nil, err
	}
	return s.versionRepo.FindByID(rec.CurrentVersionID)
}

// FindVersion returns any version of a recipe (including deprecated ones).
func (s *RecipeService) FindVersion(recipeID uint, versionID uint) (*model.RecipeVersion, error) {
	v, err := s.versionRepo.FindByID(versionID)
	if err != nil {
		return nil, err
	}
	if v.RecipeID != recipeID {
		return nil, repository.ErrNotFound
	}
	return v, nil
}

// requireOwner loads a recipe and verifies userID owns it.
func (s *RecipeService) requireOwner(userID, recipeID uint) (*model.BrewRecipe, error) {
	rec, err := s.repo.FindByID(recipeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("BrewRecipe[id=%d] not found", recipeID))
		}
		return nil, fmt.Errorf("recipe find: %w", err)
	}
	if rec.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("BrewRecipe[id=%d] modify failed: user_id=%d not owner", recipeID, userID))
	}
	return rec, nil
}
