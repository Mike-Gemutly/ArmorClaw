package team

import (
	"fmt"
	"sort"
	"strings"

	"github.com/armorclaw/bridge/pkg/capability"
)

// builtins is the immutable registry of built-in team roles.
// Populated once at package init; never mutated after that.
var builtins map[string]TeamRole

func init() {
	browserCaps := capability.CapabilitySet{
		"browser.browse":   true,
		"browser.extract":  true,
		"browser.screenshot": true,
	}

	formCaps := capability.CapabilitySet{
		"browser.fill":   true,
		"secret.request": true,
	}

	docCaps := capability.CapabilitySet{
		"doc.ingest":    true,
		"doc.summarize": true,
		"doc.reference": true,
	}

	emailCaps := capability.CapabilitySet{
		"email.read":  true,
		"email.draft": true,
		"email.send":  true,
	}

	supervisorCaps := capability.CapabilitySet{
		"team.synthesize":   true,
		"team.request_hitl": true,
		"team.review":       true,
	}

	// team_lead gets the superset of all other roles' capabilities.
	leadCaps := make(capability.CapabilitySet)
	for k := range browserCaps {
		leadCaps[k] = true
	}
	for k := range formCaps {
		leadCaps[k] = true
	}
	for k := range docCaps {
		leadCaps[k] = true
	}
	for k := range emailCaps {
		leadCaps[k] = true
	}
	for k := range supervisorCaps {
		leadCaps[k] = true
	}

	builtins = map[string]TeamRole{
		"team_lead": {
			Name:         "team_lead",
			Capabilities: leadCaps,
			Description:  "Coordinates the team and holds all capabilities.",
		},
		"browser_specialist": {
			Name:         "browser_specialist",
			Capabilities: browserCaps,
			Description:  "Browses the web, extracts data, and captures screenshots.",
		},
		"form_filler": {
			Name:         "form_filler",
			Capabilities: formCaps,
			Description:  "Fills browser forms using BlindFill secret injection.",
		},
		"doc_analyst": {
			Name:         "doc_analyst",
			Capabilities: docCaps,
			Description:  "Ingests, summarizes, and cross-references documents.",
		},
		"email_clerk": {
			Name:         "email_clerk",
			Capabilities: emailCaps,
			Description:  "Reads, drafts, and sends emails through human approval.",
		},
		"supervisor": {
			Name:         "supervisor",
			Capabilities: supervisorCaps,
			Description:  "Synthesizes results, requests human review, and reviews team output.",
		},
	}
}

// GetRole returns the built-in role identified by name.
// Returns an error if no role matches.
func GetRole(name string) (TeamRole, error) {
	r, ok := builtins[name]
	if !ok {
		return TeamRole{}, fmt.Errorf("team: unknown role %q", name)
	}
	return r, nil
}

// ListRoles returns all built-in roles sorted alphabetically by name.
func ListRoles() []TeamRole {
	roles := make([]TeamRole, 0, len(builtins))
	for _, r := range builtins {
		roles = append(roles, r)
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Name < roles[j].Name
	})
	return roles
}

// ValidateRoleAssignment checks whether assigning roleName is valid given
// the set of existingRoles already present in the team.
//
// Rules:
//   - roleName must exist in the registry
//   - a team may have at most one team_lead
func ValidateRoleAssignment(roleName string, existingRoles []string) error {
	if _, err := GetRole(roleName); err != nil {
		return err
	}
	if roleName == "team_lead" {
		for _, er := range existingRoles {
			if er == "team_lead" {
				return fmt.Errorf("team: duplicate team_lead assignment")
			}
		}
	}
	return nil
}

// capsKeys returns the sorted keys of a CapabilitySet for deterministic
// comparison in tests.
func capsKeys(cs capability.CapabilitySet) []string {
	keys := make([]string, 0, len(cs))
	for k := range cs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// capsString returns a sorted, comma-joined representation of a CapabilitySet.
func capsString(cs capability.CapabilitySet) string {
	return strings.Join(capsKeys(cs), ", ")
}
