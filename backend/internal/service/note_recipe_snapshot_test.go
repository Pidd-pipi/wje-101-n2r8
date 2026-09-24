package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/dto"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

// TestNoteSnapshotStaysFrozenAcrossRecipeEdits proves the core requirement:
// after a note is published against a recipe's current version, the author
// changing water temp/steps (new versions) must not alter the old note.
func TestNoteSnapshotStaysFrozenAcrossRecipeEdits(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.BrewRecipe{}, &model.RecipeVersion{}, &model.TastingNote{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_ = db.Exec("DELETE FROM tasting_notes").Error
	_ = db.Exec("DELETE FROM recipe_versions").Error
	_ = db.Exec("DELETE FROM brew_recipes").Error

	recipeSvc := NewRecipeService(
		repository.NewBrewRecipeRepository(db),
		repository.NewRecipeVersionRepository(db),
		newTestLogger(),
	)
	noteSvc := NewNoteService(repository.NewTastingNoteRepository(db), recipeSvc, newTestLogger())

	// Author shares the recipe at 92°C.
	rec, err := recipeSvc.Create(1, &model.BrewRecipe{
		Name: "手冲三段式", Device: "手冲壶", WaterTemp: 92,
		Steps: `[{"step_number":1,"description":"闷蒸30秒","duration_seconds":30}]`,
	})
	if err != nil {
		t.Fatalf("recipe create: %v", err)
	}

	// A drinker publishes a note: it should freeze v1 data.
	note, err := noteSvc.Create(2, &model.TastingNote{
		CoffeeName: "耶加雪菲", RoastLevel: "light", BrewRecipeID: rec.ID,
	})
	if err != nil {
		t.Fatalf("note create: %v", err)
	}
	if note.BrewRecipeVersionID == 0 {
		t.Fatal("note must record the recipe version id")
	}
	if note.BrewRecipeName != "手冲三段式" || note.BrewRecipeWaterTemp != 92 {
		t.Fatalf("snapshot mismatch: name=%s temp=%d", note.BrewRecipeName, note.BrewRecipeWaterTemp)
	}

	// Author later raises the water temperature and reworks the steps (v2).
	if _, err := recipeSvc.AddVersion(1, rec.ID, &dto.RecipeVersionCreateRequest{
		Name: "手冲三段式", Device: "手冲壶", WaterTemp: 96,
		Steps: `[{"step_number":1,"description":"高温快冲","duration_seconds":20}]`,
	}); err != nil {
		t.Fatalf("add v2: %v", err)
	}

	// Re-read the note: snapshot fields must still show the original brew.
	oldNote, _ := noteSvc.Get(note.ID)
	if oldNote.BrewRecipeName != "手冲三段式" || oldNote.BrewRecipeWaterTemp != 92 {
		t.Fatalf("old note snapshot changed: name=%s temp=%d", oldNote.BrewRecipeName, oldNote.BrewRecipeWaterTemp)
	}
	if oldNote.BrewRecipeSteps != `[{"step_number":1,"description":"闷蒸30秒","duration_seconds":30}]` {
		t.Fatalf("old note steps changed: %s", oldNote.BrewRecipeSteps)
	}

	snap := noteSvc.RecipeSnapshot(oldNote)
	if snap == nil {
		t.Fatal("snapshot must be returned")
	}
	if snap.VersionNumber != 1 || snap.WaterTemp != 92 {
		t.Fatalf("snapshot should reference v1 @92°C, got v%d @%d°C", snap.VersionNumber, snap.WaterTemp)
	}
	if snap.Deprecated {
		t.Fatal("v1 active -> snapshot must not be flagged deprecated")
	}

	// Deprecating v1 must flag the snapshot but keep its data readable.
	if err := recipeSvc.SetCurrentVersion(1, rec.ID, 2); err != nil {
		t.Fatalf("switch current to v2: %v", err)
	}
	if err := recipeSvc.SetVersionStatus(1, rec.ID, 1, constants.VersionStatusDeprecated); err != nil {
		t.Fatalf("deprecate v1: %v", err)
	}
	snap = noteSvc.RecipeSnapshot(oldNote)
	if !snap.Deprecated || snap.Status != constants.VersionStatusDeprecated {
		t.Fatalf("snapshot must be flagged deprecated, got status=%s deprecated=%v", snap.Status, snap.Deprecated)
	}
	if snap.WaterTemp != 92 || snap.Name != "手冲三段式" {
		t.Fatal("deprecated snapshot must still expose the original brew data")
	}

	// A brand-new note now freezes the current v2.
	newNote, err := noteSvc.Create(3, &model.TastingNote{
		CoffeeName: "曼特宁", RoastLevel: "dark", BrewRecipeID: rec.ID,
	})
	if err != nil {
		t.Fatalf("new note create: %v", err)
	}
	if newNote.BrewRecipeWaterTemp != 96 {
		t.Fatalf("new note should freeze v2 @96°C, got %d", newNote.BrewRecipeWaterTemp)
	}
}

// TestNoteCreateRejectsUnknownRecipe guards the FK-like validation at publish.
func TestNoteCreateRejectsUnknownRecipe(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&model.BrewRecipe{}, &model.RecipeVersion{}, &model.TastingNote{})
	svc := NewNoteService(
		repository.NewTastingNoteRepository(db),
		NewRecipeService(repository.NewBrewRecipeRepository(db), repository.NewRecipeVersionRepository(db), newTestLogger()),
		newTestLogger(),
	)
	_, err := svc.Create(1, &model.TastingNote{CoffeeName: "x", RoastLevel: "light", BrewRecipeID: 9999})
	if err == nil || util.ErrorCode(err) != constants.CodeValidationError {
		t.Fatalf("unknown recipe should be 422, got %v", err)
	}
}
