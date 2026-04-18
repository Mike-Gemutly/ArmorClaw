package team

import (
	"testing"

	"github.com/armorclaw/bridge/pkg/capability"
)

func TestGetRole_ValidRoles(t *testing.T) {
	t.Parallel()
	expected := map[string]capability.CapabilitySet{
		"team_lead": {
			"browser.browse": true, "browser.extract": true, "browser.screenshot": true,
			"browser.fill": true, "secret.request": true,
			"doc.ingest": true, "doc.summarize": true, "doc.reference": true,
			"email.read": true, "email.draft": true, "email.send": true,
			"team.synthesize": true, "team.request_hitl": true, "team.review": true,
		},
		"browser_specialist": {
			"browser.browse": true, "browser.extract": true, "browser.screenshot": true,
		},
		"form_filler": {
			"browser.fill": true, "secret.request": true,
		},
		"doc_analyst": {
			"doc.ingest": true, "doc.summarize": true, "doc.reference": true,
		},
		"email_clerk": {
			"email.read": true, "email.draft": true, "email.send": true,
		},
		"supervisor": {
			"team.synthesize": true, "team.request_hitl": true, "team.review": true,
		},
	}

	for name, wantCaps := range expected {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			role, err := GetRole(name)
			if err != nil {
				t.Fatalf("GetRole(%q) returned error: %v", name, err)
			}
			if role.Name != name {
				t.Errorf("Name = %q, want %q", role.Name, name)
			}
			if len(role.Capabilities) != len(wantCaps) {
				t.Errorf("Capabilities count = %d, want %d (got: %s)",
					len(role.Capabilities), len(wantCaps), capsString(role.Capabilities))
			}
			for k := range wantCaps {
				if !role.Capabilities[k] {
					t.Errorf("missing capability %q in role %q", k, name)
				}
			}
		})
	}
}

func TestGetRole_UnknownRole(t *testing.T) {
	t.Parallel()
	_, err := GetRole("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown role, got nil")
	}
}

func TestListRoles(t *testing.T) {
	t.Parallel()
	roles := ListRoles()
	if len(roles) != 6 {
		t.Fatalf("ListRoles() returned %d roles, want 6", len(roles))
	}
	for i := 1; i < len(roles); i++ {
		if roles[i].Name <= roles[i-1].Name {
			t.Errorf("roles not sorted: %q >= %q at index %d", roles[i-1].Name, roles[i].Name, i)
		}
	}
	names := make([]string, len(roles))
	for i, r := range roles {
		names[i] = r.Name
	}
	want := []string{"browser_specialist", "doc_analyst", "email_clerk", "form_filler", "supervisor", "team_lead"}
	for i, n := range want {
		if names[i] != n {
			t.Errorf("role[%d] = %q, want %q", i, names[i], n)
		}
	}
}

func TestTeamLead_HasSuperset(t *testing.T) {
	t.Parallel()
	lead, err := GetRole("team_lead")
	if err != nil {
		t.Fatalf("GetRole(team_lead) error: %v", err)
	}
	others := []string{"browser_specialist", "form_filler", "doc_analyst", "email_clerk", "supervisor"}
	for _, name := range others {
		role, err := GetRole(name)
		if err != nil {
			t.Fatalf("GetRole(%q) error: %v", name, err)
		}
		for k := range role.Capabilities {
			if !lead.Capabilities[k] {
				t.Errorf("team_lead missing capability %q (from %q)", k, name)
			}
		}
	}
}

func TestBrowserSpecialist_CannotSendEmail(t *testing.T) {
	t.Parallel()
	role, err := GetRole("browser_specialist")
	if err != nil {
		t.Fatalf("GetRole(browser_specialist) error: %v", err)
	}
	if role.Capabilities["email.send"] {
		t.Error("browser_specialist should not have email.send capability")
	}
}

func TestValidateRoleAssignment_DuplicateLead(t *testing.T) {
	t.Parallel()
	err := ValidateRoleAssignment("team_lead", []string{"team_lead"})
	if err == nil {
		t.Fatal("expected error for duplicate team_lead, got nil")
	}
}

func TestValidateRoleAssignment_ValidAssignment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		role    string
		existing []string
	}{
		{"first team_lead", "team_lead", nil},
		{"team_lead with other roles", "team_lead", []string{"browser_specialist"}},
		{"non-lead role", "browser_specialist", []string{"team_lead"}},
		{"non-lead with existing non-lead", "doc_analyst", []string{"browser_specialist"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateRoleAssignment(tt.role, tt.existing); err != nil {
				t.Errorf("ValidateRoleAssignment(%q, %v) = %v, want nil", tt.role, tt.existing, err)
			}
		})
	}
}
