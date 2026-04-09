package models

import (
	"regexp"
)

var semVerRegex = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

// ValidateVersion validates that a version string follows SemVer format
func ValidateVersion(version string) error {
	if !semVerRegex.MatchString(version) {
		return &InvalidVersionError{Version: version}
	}
	return nil
}

// InvalidVersionError represents an invalid version format error
type InvalidVersionError struct {
	Version string
}

func (e *InvalidVersionError) Error() string {
	return "invalid SemVer format: " + e.Version
}
