package repository

import "gorm.io/gorm/clause"

// clauseForUpdate returns a row-level locking clause (SELECT ... FOR UPDATE)
// used to serialise transactions that allocate version numbers.
func clauseForUpdate() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}
