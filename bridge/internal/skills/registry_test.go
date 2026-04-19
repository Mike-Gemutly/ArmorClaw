package skills

import "testing"

func TestRegistry_NewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	_, exists := r.GetSkill("anything")
	if exists {
		t.Error("expected empty registry to have no skills")
	}
	enabled := r.GetEnabled()
	if len(enabled) != 0 {
		t.Errorf("expected 0 enabled skills, got %d", len(enabled))
	}
}

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	RegisterWebDAV(r)

	skill, found := r.GetSkill("webdav")
	if !found {
		t.Fatal("expected to find webdav skill after registration")
	}
	if skill.Name != "webdav" {
		t.Errorf("expected skill name=webdav, got %s", skill.Name)
	}
	if !skill.Enabled {
		t.Error("expected registered skill to be enabled")
	}

	_, notFound := r.GetSkill("nonexistent")
	if notFound {
		t.Error("expected nonexistent skill lookup to return false")
	}
}

func TestRegistry_DomainExtraction(t *testing.T) {
	cases := []struct {
		input  string
		domain string
	}{
		{"github.repos", "github"},
		{"weather.forecast", "weather"},
		{"web.browse", "web"},
		{"search.query", "search"},
		{"email.send", "email"},
	}
	for _, tc := range cases {
		got := extractDomainFromName(tc.input)
		if got != tc.domain {
			t.Errorf("extractDomainFromName(%q) = %q, want %q", tc.input, got, tc.domain)
		}
	}
}

func TestRegistry_RiskAssignment(t *testing.T) {
	cases := []struct {
		domain string
		risk   string
	}{
		{"email", "high"},
		{"web", "medium"},
		{"weather", "low"},
		{"github", "medium"},
		{"search", "medium"},
	}
	for _, tc := range cases {
		got := getRiskForDomain(tc.domain)
		if got != tc.risk {
			t.Errorf("getRiskForDomain(%q) = %q, want %q", tc.domain, got, tc.risk)
		}
	}
}
