package team

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/armorclaw/bridge/pkg/capability"
)

func TestTeamRoundTrip(t *testing.T) {
	original := Team{
		ID:             "team-1",
		Name:           "Research Team",
		TemplateID:     "tpl-alpha",
		SharedContext:  "shared context data",
		LifecycleState: LifecycleActive,
		Budgets: &TeamBudgets{
			MaxTokenUsage:   1000000,
			MaxCost:         50.0,
			MaxDuration:     "2h",
			MaxSecretAccess: 10,
		},
		Version: 3,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Team
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch:\n  got:  %+v\n  want: %+v", decoded, original)
	}
}

func TestTeamMemberRoundTrip(t *testing.T) {
	original := TeamMember{
		TeamID:                "team-1",
		AgentID:               "agent-42",
		RoleName:              "researcher",
		AllowedTools:          []string{"web_browse", "form_fill"},
		AllowedSecretPrefixes: []string{"payment/"},
		BrowserContextID:      "ctx-abc",
		Priority:              5,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TeamMember
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch:\n  got:  %+v\n  want: %+v", decoded, original)
	}
}

func TestTeamRoleRoundTrip(t *testing.T) {
	original := TeamRole{
		Name:         "lead",
		Capabilities: capability.CapabilitySet{"web_browse": true, "form_fill": true},
		Description:  "Team lead role",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TeamRole
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch:\n  got:  %+v\n  want: %+v", decoded, original)
	}
}

func TestTeamTemplateRoundTrip(t *testing.T) {
	original := TeamTemplate{
		ID:   "tpl-alpha",
		Name: "Research Template",
		Roles: []TeamRole{
			{Name: "lead", Capabilities: capability.CapabilitySet{"web_browse": true}},
			{Name: "assistant", Capabilities: capability.CapabilitySet{"web_browse": true, "doc_read": true}},
		},
		DefaultContext: "You are a research team.",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TeamTemplate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round-trip mismatch:\n  got:  %+v\n  want: %+v", decoded, original)
	}
}

func TestValidate_Team_InvalidCases(t *testing.T) {
	t.Run("empty ID", func(t *testing.T) {
		v := Team{Name: "x", LifecycleState: LifecycleActive}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty ID")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		v := Team{ID: "x", LifecycleState: LifecycleActive}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("invalid lifecycle", func(t *testing.T) {
		v := Team{ID: "x", Name: "x", LifecycleState: LifecycleState("unknown")}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for invalid lifecycle state")
		}
	})
}

func TestValidate_Team_ValidCases(t *testing.T) {
	for _, ls := range []LifecycleState{LifecycleActive, LifecycleSuspended, LifecycleDissolved} {
		t.Run(string(ls), func(t *testing.T) {
			v := Team{ID: "team-1", Name: "Test Team", LifecycleState: ls}
			if err := v.Validate(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidate_TeamMember_InvalidCases(t *testing.T) {
	t.Run("empty TeamID", func(t *testing.T) {
		v := TeamMember{AgentID: "a", RoleName: "r"}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty TeamID")
		}
	})

	t.Run("empty AgentID", func(t *testing.T) {
		v := TeamMember{TeamID: "t", RoleName: "r"}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty AgentID")
		}
	})

	t.Run("empty RoleName", func(t *testing.T) {
		v := TeamMember{TeamID: "t", AgentID: "a"}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty RoleName")
		}
	})
}

func TestValidate_TeamRole_InvalidCases(t *testing.T) {
	t.Run("empty Name", func(t *testing.T) {
		v := TeamRole{Capabilities: capability.CapabilitySet{"x": true}}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty Name")
		}
	})

	t.Run("empty capabilities map", func(t *testing.T) {
		v := TeamRole{Name: "lead", Capabilities: capability.CapabilitySet{}}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty capabilities map")
		}
	})

	t.Run("nil capabilities is valid", func(t *testing.T) {
		v := TeamRole{Name: "lead", Capabilities: nil}
		if err := v.Validate(); err != nil {
			t.Fatalf("nil capabilities should be valid, got: %v", err)
		}
	})
}

func TestValidate_TeamTemplate_InvalidCases(t *testing.T) {
	t.Run("empty ID", func(t *testing.T) {
		v := TeamTemplate{Name: "x", Roles: []TeamRole{{Name: "r"}}}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty ID")
		}
	})

	t.Run("empty Name", func(t *testing.T) {
		v := TeamTemplate{ID: "x", Roles: []TeamRole{{Name: "r"}}}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty Name")
		}
	})

	t.Run("empty Roles", func(t *testing.T) {
		v := TeamTemplate{ID: "x", Name: "x", Roles: nil}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for empty Roles")
		}
	})

	t.Run("zero-length Roles slice", func(t *testing.T) {
		v := TeamTemplate{ID: "x", Name: "x", Roles: []TeamRole{}}
		if err := v.Validate(); err == nil {
			t.Fatal("expected error for zero-length Roles")
		}
	})
}
