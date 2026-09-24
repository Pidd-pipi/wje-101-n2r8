package main

import "gorm.io/gorm"

// backfillLegacyRecipes upgrades databases created before recipe
// versioning: every recipe without a version gets an immutable version 1
// copied from its legacy columns, the active pointer is set, and existing
// notes are pinned to that version with full parameter snapshots. It is
// idempotent and safe to run on every startup.
func backfillLegacyRecipes(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Skip on fresh databases (brew_recipes has no legacy columns).
		if !tx.Migrator().HasColumn("brew_recipes", "water_temp") {
			return nil
		}
		if err := tx.Exec(`
			INSERT INTO brew_recipe_versions (recipe_id, version_number, water_temp, grind_size, ratio, steps, status, created_at)
			SELECT id, 1, water_temp, COALESCE(grind_size, ''), COALESCE(ratio, ''), COALESCE(steps, '[]'), 'active', created_at
			FROM brew_recipes
			WHERE current_version = 0`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE brew_recipes
			SET current_version = 1,
			    active_version_id = (
			        SELECT id FROM brew_recipe_versions
			        WHERE recipe_id = brew_recipes.id AND version_number = 1
			    )
			WHERE current_version = 0`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE tasting_notes n
			SET brew_recipe_version_id = v.id,
			    recipe_name_snapshot = r.name,
			    recipe_temp_snapshot = v.water_temp,
			    recipe_steps_snapshot = v.steps,
			    recipe_version_snapshot = v.version_number
			FROM brew_recipes r, brew_recipe_versions v
			WHERE n.brew_recipe_id <> 0
			  AND n.brew_recipe_version_id = 0
			  AND r.id = n.brew_recipe_id
			  AND v.recipe_id = r.id AND v.version_number = 1`).Error; err != nil {
			return err
		}
		return nil
	})
}
