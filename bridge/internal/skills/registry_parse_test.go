package skills

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeSkillMD(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return p
}

func TestParseSkillFile_AllFields(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: weather.forecast
description: Get weather forecast
homepage: https://example.com
timeout: 30
version: "1.0"
enabled: true
parameters:
  - name: city
    type: string
    required: true
    description: City name
  - name: days
    type: integer
    required: false
    description: Number of days
---
# Weather Forecast Skill
`
	path := writeSkillMD(t, dir, content)

	skill, err := parseSkillFile(path)
	if err != nil {
		t.Fatalf("parseSkillFile() error: %v", err)
	}

	if skill.Name != "weather.forecast" {
		t.Errorf("Name = %q, want %q", skill.Name, "weather.forecast")
	}
	if skill.Description != "Get weather forecast" {
		t.Errorf("Description = %q, want %q", skill.Description, "Get weather forecast")
	}
	if skill.Homepage != "https://example.com" {
		t.Errorf("Homepage = %q, want %q", skill.Homepage, "https://example.com")
	}
	if skill.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", skill.Timeout, 30*time.Second)
	}
	if skill.Version != "1.0" {
		t.Errorf("Version = %q, want %q", skill.Version, "1.0")
	}
	if skill.Enabled != true {
		t.Errorf("Enabled = %v, want true", skill.Enabled)
	}
	if len(skill.Parameters) != 2 {
		t.Fatalf("Parameters count = %d, want 2", len(skill.Parameters))
	}
	city, ok := skill.Parameters["city"]
	if !ok {
		t.Fatal("missing parameter 'city'")
	}
	if city.Type != "string" {
		t.Errorf("city.Type = %q, want %q", city.Type, "string")
	}
	if !city.Required {
		t.Errorf("city.Required = false, want true")
	}
	if city.Description != "City name" {
		t.Errorf("city.Description = %q, want %q", city.Description, "City name")
	}
	days, ok := skill.Parameters["days"]
	if !ok {
		t.Fatal("missing parameter 'days'")
	}
	if days.Type != "integer" {
		t.Errorf("days.Type = %q, want %q", days.Type, "integer")
	}
	if days.Required {
		t.Errorf("days.Required = true, want false")
	}
}

func TestParseSkillFile_TimeoutExtraction(t *testing.T) {
	dir := t.TempDir()

	t.Run("int_timeout", func(t *testing.T) {
		p := writeSkillMD(t, dir, "---\nname: test.a\ntimeout: 30\n---\n")
		skill, err := parseSkillFile(p)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if skill.Timeout != 30*time.Second {
			t.Errorf("Timeout = %v, want 30s", skill.Timeout)
		}
	})

	t.Run("string_timeout", func(t *testing.T) {
		p := writeSkillMD(t, dir, "---\nname: test.b\ntimeout: \"45s\"\n---\n")
		skill, err := parseSkillFile(p)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if skill.Timeout != 45*time.Second {
			t.Errorf("Timeout = %v, want 45s", skill.Timeout)
		}
	})

	t.Run("missing_timeout", func(t *testing.T) {
		p := writeSkillMD(t, dir, "---\nname: test.c\n---\n")
		skill, err := parseSkillFile(p)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if skill.Timeout != 0 {
			t.Errorf("Timeout = %v, want 0", skill.Timeout)
		}
	})
}

func TestParseSkillFile_EnabledFalse(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: test.disabled\nenabled: false\n---\n"
	path := writeSkillMD(t, dir, content)

	skill, err := parseSkillFile(path)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if skill.Enabled {
		t.Error("Enabled = true, want false")
	}
}

func TestParseSkillFile_MissingOptionalFields(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: minimal.skill\n---\n"
	path := writeSkillMD(t, dir, content)

	skill, err := parseSkillFile(path)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if skill.Timeout != 0 {
		t.Errorf("Timeout = %v, want 0 (zero means executor applies default)", skill.Timeout)
	}
	if skill.Version != "" {
		t.Errorf("Version = %q, want empty", skill.Version)
	}
	if !skill.Enabled {
		t.Error("Enabled = false, want true (default)")
	}
	if skill.Parameters == nil {
		t.Fatal("Parameters = nil, want non-nil map")
	}
	if len(skill.Parameters) != 0 {
		t.Errorf("Parameters count = %d, want 0", len(skill.Parameters))
	}
}
