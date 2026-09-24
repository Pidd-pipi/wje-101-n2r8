package service

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/wjecoffeetaste/wjecoffeetaste/internal/constants"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/dto"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/model"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/repository"
	"github.com/wjecoffeetaste/wjecoffeetaste/internal/util"
)

func errorsIsDuplicate(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func newVersionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a shared in-memory cache so all connections in the pool see the schema.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.BrewRecipe{}, &model.RecipeVersion{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM recipe_versions").Error
		_ = db.Exec("DELETE FROM brew_recipes").Error
	})
	return db
}

func newRecipeSvc(db *gorm.DB) *RecipeService {
	return NewRecipeService(
		repository.NewBrewRecipeRepository(db),
		repository.NewRecipeVersionRepository(db),
		newTestLogger(),
	)
}

func TestRecipeCreateMakesVersion1(t *testing.T) {
	svc := newRecipeSvc(newVersionTestDB(t))
	rec := &model.BrewRecipe{Name: "V60 基础", Device: "手冲壶", WaterTemp: 92, Steps: "[]"}
	created, err := svc.Create(7, rec)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.CurrentVersionID == 0 {
		t.Fatal("current_version_id should point at v1")
	}
	detail, err := svc.GetDetail(created.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if len(detail.Versions) != 1 || detail.Versions[0].VersionNumber != 1 {
		t.Fatalf("expected exactly v1, got %+v", detail.Versions)
	}
	if !detail.Versions[0].IsCurrent || detail.Versions[0].Status != constants.VersionStatusActive {
		t.Fatal("v1 should be current and active")
	}
}

func TestAddVersionAllocatesConsecutiveNumbersAndSwitchesCurrent(t *testing.T) {
	svc := newRecipeSvc(newVersionTestDB(t))
	created, _ := svc.Create(7, &model.BrewRecipe{Name: "配方", WaterTemp: 90})

	for _, temp := range []int{88, 94, 93} {
		if _, err := svc.AddVersion(7, created.ID, &dto.RecipeVersionCreateRequest{
			Name: "配方", WaterTemp: temp,
		}); err != nil {
			t.Fatalf("add version @%d: %v", temp, err)
		}
	}
	detail, _ := svc.GetDetail(created.ID)
	if len(detail.Versions) != 4 {
		t.Fatalf("expected 4 versions, got %d", len(detail.Versions))
	}
	// Desc order: newest first.
	for i, want := range []int{4, 3, 2, 1} {
		if detail.Versions[i].VersionNumber != want {
			t.Fatalf("position %d: want v%d got v%d", i, want, detail.Versions[i].VersionNumber)
		}
	}
	var current *model.RecipeVersion
	for i := range detail.Versions {
		if detail.Versions[i].IsCurrent {
			current = &detail.Versions[i]
		}
	}
	if current == nil || current.VersionNumber != 4 || current.WaterTemp != 93 {
		t.Fatalf("newest v4 with temp 93 should be current, got %+v", current)
	}
	// Recipe mirror follows current.
	if created, _ = svc.Get(created.ID); created.WaterTemp != 93 {
		t.Fatalf("recipe mirror should be 93, got %d", created.WaterTemp)
	}
}

func TestSetCurrentVersion(t *testing.T) {
	svc := newRecipeSvc(newVersionTestDB(t))
	created, _ := svc.Create(7, &model.BrewRecipe{Name: "配方", WaterTemp: 90})
	if _, err := svc.AddVersion(7, created.ID, &dto.RecipeVersionCreateRequest{Name: "配方", WaterTemp: 96}); err != nil {
		t.Fatalf("add v2: %v", err)
	}
	if err := svc.SetCurrentVersion(7, created.ID, 1); err != nil {
		t.Fatalf("switch back to v1: %v", err)
	}
	rec, _ := svc.Get(created.ID)
	detail, _ := svc.GetDetail(created.ID)
	v1ID := uint(0)
	for i := range detail.Versions {
		if detail.Versions[i].VersionNumber == 1 {
			v1ID = detail.Versions[i].ID
		}
	}
	if rec.WaterTemp != 90 || rec.CurrentVersionID != v1ID {
		t.Fatalf("current should be v1 (temp 90), got temp=%d ver=%d want ver=%d", rec.WaterTemp, rec.CurrentVersionID, v1ID)
	}
}

func TestOnlyOwnerCanMutate(t *testing.T) {
	svc := newRecipeSvc(newVersionTestDB(t))
	created, _ := svc.Create(7, &model.BrewRecipe{Name: "配方"})

	_, err := svc.AddVersion(8, created.ID, &dto.RecipeVersionCreateRequest{Name: "配方"})
	if err == nil || util.ErrorCode(err) != constants.CodeForbidden {
		t.Fatalf("non-owner add version should be 403, got %v", err)
	}
	if err := svc.SetCurrentVersion(8, created.ID, 1); err == nil || util.ErrorCode(err) != constants.CodeForbidden {
		t.Fatalf("non-owner set current should be 403, got %v", err)
	}
	if err := svc.SetVersionStatus(8, created.ID, 1, constants.VersionStatusDeprecated); err == nil || util.ErrorCode(err) != constants.CodeForbidden {
		t.Fatalf("non-owner deprecate should be 403, got %v", err)
	}
}

func TestDeprecatedVersionCannotBeCurrentButStaysReadable(t *testing.T) {
	svc := newRecipeSvc(newVersionTestDB(t))
	created, _ := svc.Create(7, &model.BrewRecipe{Name: "配方", WaterTemp: 90})
	v2, verr := svc.AddVersion(7, created.ID, &dto.RecipeVersionCreateRequest{Name: "配方", WaterTemp: 96})
	if verr != nil {
		t.Fatalf("add v2: %v", verr)
	}
	// Current is v2: cannot deprecate it directly.
	if err := svc.SetVersionStatus(7, created.ID, 2, constants.VersionStatusDeprecated); err == nil ||
		util.ErrorCode(err) != constants.CodeValidationError {
		t.Fatalf("deprecating current version should be rejected, got %v", err)
	}
	// Switch back to v1, then deprecate v2.
	if err := svc.SetCurrentVersion(7, created.ID, 1); err != nil {
		t.Fatalf("set v1 current: %v", err)
	}
	if err := svc.SetVersionStatus(7, created.ID, 2, constants.VersionStatusDeprecated); err != nil {
		t.Fatalf("deprecate v2: %v", err)
	}
	// Deprecated v2 can no longer be selected as current.
	if err := svc.SetCurrentVersion(7, created.ID, 2); err == nil ||
		util.ErrorCode(err) != constants.CodeValidationError {
		t.Fatalf("selecting deprecated v2 as current should be rejected, got %v", err)
	}
	// ... but the deprecated row is still readable for cross-checking.
	got, err := svc.FindVersion(created.ID, v2.ID)
	if err != nil {
		t.Fatalf("deprecated version must remain readable: %v", err)
	}
	if got.Status != constants.VersionStatusDeprecated {
		t.Fatalf("v2 status = %s, want deprecated", got.Status)
	}
	// Re-activating makes it selectable again.
	if err := svc.SetVersionStatus(7, created.ID, 2, constants.VersionStatusActive); err != nil {
		t.Fatalf("restore v2: %v", err)
	}
	if err := svc.SetCurrentVersion(7, created.ID, 2); err != nil {
		t.Fatalf("re-activated v2 should be selectable: %v", err)
	}
}

func TestVersionNumberUniqueConstraint(t *testing.T) {
	db := newVersionTestDB(t)
	rec := model.BrewRecipe{UserID: 1, Name: "配方"}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatal(err)
	}
	v := model.RecipeVersion{RecipeID: rec.ID, VersionNumber: 1, Status: "active", Name: "配方", Steps: "[]"}
	if err := db.Create(&v).Error; err != nil {
		t.Fatal(err)
	}
	dup := model.RecipeVersion{RecipeID: rec.ID, VersionNumber: 1, Status: "active", Name: "配方", Steps: "[]"}
	err := db.Create(&dup).Error
	if err == nil || !errorsIsDuplicate(err) {
		t.Fatalf("duplicate (recipe_id, version_number) must be rejected, got %v", err)
	}
}
