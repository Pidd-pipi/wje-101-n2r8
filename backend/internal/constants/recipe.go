package constants

// Recipe version lifecycle statuses. Versions are immutable content-wise;
// deprecated versions stay readable for old tasting notes but can't be
// selected as the current version.
const (
	VersionStatusActive     = "active"
	VersionStatusDeprecated = "deprecated"
)

// IsValidRecipeVersionStatus reports whether s is a supported version status.
func IsValidRecipeVersionStatus(s string) bool {
	return s == VersionStatusActive || s == VersionStatusDeprecated
}
