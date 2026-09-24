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

// NoteService handles tasting notes.
type NoteService struct {
	repo      *repository.TastingNoteRepository
	recipeSvc *RecipeService
	logger    *slog.Logger
}

// NewNoteService creates a NoteService.
func NewNoteService(repo *repository.TastingNoteRepository, recipeSvc *RecipeService, logger *slog.Logger) *NoteService {
	return &NoteService{repo: repo, recipeSvc: recipeSvc, logger: logger}
}

// Create adds a note for a user. When a recipe is referenced, the note
// freezes the recipe's current version (id, name, water temp, steps) so the
// record stays accurate even after the author publishes newer versions.
func (s *NoteService) Create(userID uint, n *model.TastingNote) (*model.TastingNote, error) {
	if !constants.IsValidRoastLevel(n.RoastLevel) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("TastingNote[roast_level=%s] create failed: invalid roast level", n.RoastLevel))
	}
	n.UserID = userID
	if n.FlavorTags == "" {
		n.FlavorTags = "[]"
	}
	if n.BrewRecipeID != 0 {
		v, err := s.recipeSvc.CurrentVersion(n.BrewRecipeID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, util.NewAppError(422, constants.CodeValidationError,
					fmt.Sprintf("TastingNote create failed: BrewRecipe[id=%d] not found", n.BrewRecipeID))
			}
			return nil, fmt.Errorf("note recipe snapshot: %w", err)
		}
		n.BrewRecipeVersionID = v.ID
		n.BrewRecipeName = v.Name
		n.BrewRecipeWaterTemp = v.WaterTemp
		n.BrewRecipeSteps = v.Steps
	}
	if err := s.repo.Create(n); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogNoteCreateFailed, n.CoffeeName), "error", err)
		return nil, fmt.Errorf("note create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogNoteCreateSuccess, n.CoffeeName), "id", n.ID)
	return n, nil
}

// Get returns a note by id.
func (s *NoteService) Get(id uint) (*model.TastingNote, error) {
	n, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("TastingNote[id=%d] not found", id))
		}
		return nil, fmt.Errorf("note get: %w", err)
	}
	return n, nil
}

// RecipeSnapshot resolves the recipe data a note was brewed with. The frozen
// snapshot columns drive name/water/steps; the version row supplies device /
// grind / ratio and the live status (deprecated versions are flagged) and is
// still returned so old notes can always be cross-checked.
func (s *NoteService) RecipeSnapshot(n *model.TastingNote) *dto.RecipeSnapshotResponse {
	if n.BrewRecipeID == 0 {
		return nil
	}
	snap := &dto.RecipeSnapshotResponse{
		RecipeID:      n.BrewRecipeID,
		VersionID:     n.BrewRecipeVersionID,
		VersionNumber: 0,
		Name:          n.BrewRecipeName,
		WaterTemp:     n.BrewRecipeWaterTemp,
		Steps:         n.BrewRecipeSteps,
	}
	if n.BrewRecipeVersionID != 0 {
		if v, err := s.recipeSvc.FindVersion(n.BrewRecipeID, n.BrewRecipeVersionID); err == nil {
			snap.VersionNumber = v.VersionNumber
			snap.Device = v.Device
			snap.GrindSize = v.GrindSize
			snap.Ratio = v.Ratio
			snap.Status = v.Status
			snap.Deprecated = v.Status == constants.VersionStatusDeprecated
			// Keep the frozen snapshot authoritative even if the version row
			// somehow diverges.
			if n.BrewRecipeName != "" {
				snap.Name = n.BrewRecipeName
			} else {
				snap.Name = v.Name
			}
			if n.BrewRecipeSteps != "" {
				snap.Steps = n.BrewRecipeSteps
			} else {
				snap.Steps = v.Steps
			}
			if n.BrewRecipeWaterTemp != 0 {
				snap.WaterTemp = n.BrewRecipeWaterTemp
			} else {
				snap.WaterTemp = v.WaterTemp
			}
			return snap
		}
		// Version row gone/never seeded: the snapshot columns alone still
		// carry the name/water/steps used that day.
		snap.Status = "unknown"
		return snap
	}
	// Legacy notes predating versions: fall back to the current recipe row.
	if rec, err := s.recipeSvc.Get(n.BrewRecipeID); err == nil {
		snap.Name = rec.Name
		snap.WaterTemp = rec.WaterTemp
		snap.Steps = rec.Steps
		snap.Device = rec.Device
		snap.GrindSize = rec.GrindSize
		snap.Ratio = rec.Ratio
		snap.Status = "legacy"
	}
	return snap
}

// Update edits a note owned by the user. The recipe snapshot is immutable and
// intentionally not touched here.
func (s *NoteService) Update(userID, id uint, n *model.TastingNote) (*model.TastingNote, error) {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("note update find: %w", err)
	}
	if exist.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("TastingNote[id=%d] update failed: user_id=%d not owner", id, userID))
	}
	if n.CoffeeName != "" {
		exist.CoffeeName = n.CoffeeName
	}
	if n.Origin != "" {
		exist.Origin = n.Origin
	}
	if n.RoastLevel != "" {
		if !constants.IsValidRoastLevel(n.RoastLevel) {
			return nil, util.NewAppError(422, constants.CodeValidationError, "invalid roast level")
		}
		exist.RoastLevel = n.RoastLevel
	}
	if n.FlavorTags != "" {
		exist.FlavorTags = n.FlavorTags
	}
	if n.NotesText != "" {
		exist.NotesText = n.NotesText
	}
	if n.OverallScore > 0 {
		exist.AromaScore = n.AromaScore
		exist.AcidityScore = n.AcidityScore
		exist.BodyScore = n.BodyScore
		exist.OverallScore = n.OverallScore
	}
	if err := s.repo.Update(exist); err != nil {
		return nil, fmt.Errorf("note update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogNoteUpdateSuccess, id), "id", id)
	return exist, nil
}

// Delete removes a note owned by the user.
func (s *NoteService) Delete(userID, id uint) error {
	n, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("note delete find: %w", err)
	}
	if n.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("TastingNote[id=%d] delete failed: not owner", id))
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("note delete: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogNoteDeleteSuccess, id), "id", id)
	return nil
}

// List filters notes.
func (s *NoteService) List(roast, origin, keyword string, hot bool, page, pageSize int) ([]model.TastingNote, int64, error) {
	items, total, err := s.repo.List(roast, origin, keyword, hot, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("note list: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogNoteListSuccess, roast, page), "total", total)
	return items, total, nil
}

// ListByUser returns notes of a user.
func (s *NoteService) ListByUser(userID uint) ([]model.TastingNote, error) {
	items, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("note list by user: %w", err)
	}
	return items, nil
}

// AvgScore returns the average overall score of a user's notes.
func (s *NoteService) AvgScore(userID uint) (float64, error) {
	avg, err := s.repo.AvgScore(userID)
	if err != nil {
		return 0, fmt.Errorf("note avg score: %w", err)
	}
	return avg, nil
}

// TopOrigins returns the top 3 origins by note count.
func (s *NoteService) TopOrigins(userID uint) ([]string, error) {
	origins, err := s.repo.TopOrigins(userID)
	if err != nil {
		return nil, fmt.Errorf("note top origins: %w", err)
	}
	return origins, nil
}
