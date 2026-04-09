package models

import (
	"testing"
)

func TestValidateVersionValid(t *testing.T) {
	validVersions := []string{
		"1.0.0",
		"1.0.4",
		"2.3.1",
		"10.20.30",
		"1.0.0-alpha",
		"1.0.0-beta.1",
		"1.0.0-alpha.1",
	}

	for _, version := range validVersions {
		err := ValidateVersion(version)
		if err != nil {
			t.Errorf("ValidateVersion(%q) returned unexpected error: %v", version, err)
		}
	}
}

func TestValidateVersionInvalid(t *testing.T) {
	invalidVersions := []string{
		"invalid",
		"1",
		"1.0",
		"v1.0.0",
		"1.0.0.",
		".1.0.0",
		"1.0.0-",
		"",
		"a.b.c",
		"1.0.0-abc!",
	}

	for _, version := range invalidVersions {
		err := ValidateVersion(version)
		if err == nil {
			t.Errorf("ValidateVersion(%q) expected error but got nil", version)
		}
	}
}
