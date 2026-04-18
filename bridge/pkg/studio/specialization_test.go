package studio

import (
	"strings"
	"testing"
)

// stubLookup returns a RoleLookupFunc backed by a static map of role name
// to capability names. This avoids importing pkg/team in tests.
func stubLookup(roles map[string][]string) RoleLookupFunc {
	return func(roleName string) ([]string, error) {
		caps, ok := roles[roleName]
		if !ok {
			return nil, errUnknownRole(roleName)
		}
		return caps, nil
	}
}

type unknownRoleErr string

func (e unknownRoleErr) Error() string { return "studio: unknown role " + string(e) }

func errUnknownRole(name string) error { return unknownRoleErr(name) }

// TestGetSpecialization_BrowserSpecialist verifies that the browser_specialist
// role resolves to browser-related skills and its system prompt mentions browsing.
func TestGetSpecialization_BrowserSpecialist(t *testing.T) {
	orig := activeRoleLookup
	t.Cleanup(func() { activeRoleLookup = orig })

	SetRoleLookup(stubLookup(map[string][]string{
		"browser_specialist": {"browser.browse", "browser.extract", "browser.screenshot"},
	}))

	spec, err := GetSpecialization("browser_specialist")
	if err != nil {
		t.Fatalf("GetSpecialization: %v", err)
	}

	if spec.Role != "browser_specialist" {
		t.Errorf("Role = %q, want %q", spec.Role, "browser_specialist")
	}

	wantSkills := []string{"data_extraction", "screenshot", "web_browsing"}
	if len(spec.Skills) != len(wantSkills) {
		t.Fatalf("Skills = %v, want %v", spec.Skills, wantSkills)
	}
	for i, s := range spec.Skills {
		if s != wantSkills[i] {
			t.Errorf("Skills[%d] = %q, want %q", i, s, wantSkills[i])
		}
	}

	if !strings.Contains(strings.ToLower(spec.SystemPrompt), "brows") {
		t.Errorf("SystemPrompt should mention browsing, got: %q", spec.SystemPrompt)
	}
}

// TestGetSpecialization_FormFiller verifies that the form_filler role resolves
// to form-related skills.
func TestGetSpecialization_FormFiller(t *testing.T) {
	orig := activeRoleLookup
	t.Cleanup(func() { activeRoleLookup = orig })

	SetRoleLookup(stubLookup(map[string][]string{
		"form_filler": {"browser.fill", "secret.request"},
	}))

	spec, err := GetSpecialization("form_filler")
	if err != nil {
		t.Fatalf("GetSpecialization: %v", err)
	}

	wantSkills := []string{"form_filling", "secret_injection"}
	if len(spec.Skills) != len(wantSkills) {
		t.Fatalf("Skills = %v, want %v", spec.Skills, wantSkills)
	}
	for i, s := range spec.Skills {
		if s != wantSkills[i] {
			t.Errorf("Skills[%d] = %q, want %q", i, s, wantSkills[i])
		}
	}

	if !strings.Contains(strings.ToLower(spec.SystemPrompt), "form") {
		t.Errorf("SystemPrompt should mention forms, got: %q", spec.SystemPrompt)
	}
}

// TestGetSpecialization_UnknownRole verifies that an unknown role returns an error.
func TestGetSpecialization_UnknownRole(t *testing.T) {
	orig := activeRoleLookup
	t.Cleanup(func() { activeRoleLookup = orig })

	SetRoleLookup(stubLookup(map[string][]string{}))

	_, err := GetSpecialization("nonexistent_role")
	if err == nil {
		t.Fatal("expected error for unknown role, got nil")
	}
}

// TestRoleSystemPrompts_AllRolesHavePrompts verifies that every built-in role
// has an entry in RoleSystemPrompts.
func TestRoleSystemPrompts_AllRolesHavePrompts(t *testing.T) {
	builtinRoles := []string{
		"team_lead",
		"browser_specialist",
		"form_filler",
		"doc_analyst",
		"email_clerk",
		"supervisor",
	}

	for _, role := range builtinRoles {
		prompt, ok := RoleSystemPrompts[role]
		if !ok {
			t.Errorf("RoleSystemPrompts missing entry for %q", role)
			continue
		}
		if strings.TrimSpace(prompt) == "" {
			t.Errorf("RoleSystemPrompts[%q] is empty", role)
		}
	}
}
