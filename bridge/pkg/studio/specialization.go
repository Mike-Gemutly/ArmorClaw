package studio

import (
	"fmt"
	"sort"
)

// SpecializationConfig configures role-based agent specialization.
// When attached to a SpawnRequest, the factory uses the role's system
// prompt and scoped skill list instead of the agent definition defaults.
type SpecializationConfig struct {
	Role          string   // e.g., "browser_specialist", "form_filler"
	Skills        []string // skill names available to this agent
	SystemPrompt  string   // role-specific system prompt
	ScopedSecrets []string // secret names this role can access
}

// RoleLookupFunc resolves a role name to a list of capability names.
// This is injected by the caller to avoid a direct dependency on pkg/team.
type RoleLookupFunc func(roleName string) ([]string, error)

// DefaultRoleLookup is the default implementation used when no custom
// lookup is provided. It returns an error for every role, because the
// studio package must not import pkg/team.
var DefaultRoleLookup RoleLookupFunc = func(roleName string) ([]string, error) {
	return nil, fmt.Errorf("studio: no role lookup configured for %q", roleName)
}

// activeRoleLookup holds the currently installed lookup function.
var activeRoleLookup = DefaultRoleLookup

// SetRoleLookup installs the role lookup function. Call this once during
// application startup with a function that delegates to team.GetRole and
// extracts capability keys.
func SetRoleLookup(fn RoleLookupFunc) {
	if fn != nil {
		activeRoleLookup = fn
	}
}

// RoleSystemPrompts maps role names to their default system prompts.
var RoleSystemPrompts = map[string]string{
	"team_lead":          "You are the team lead coordinator. You oversee task delegation, synthesize results from specialist agents, and ensure the team delivers complete, accurate output.",
	"browser_specialist": "You are a browser automation specialist. You browse the web, extract data from pages, capture screenshots, and report structured findings back to the team.",
	"form_filler":        "You are a form filling specialist. You fill browser forms using BlindFill secret injection, ensuring sensitive data never passes through the agent layer.",
	"doc_analyst":        "You are a document analysis specialist. You ingest documents, produce concise summaries, and cross-reference information across multiple sources.",
	"email_clerk":        "You are an email management specialist. You read, draft, and send emails through human approval, respecting confidentiality and communication policies.",
	"supervisor":         "You are a governance supervisor. You synthesize team results, request human review when needed, and enforce quality and compliance standards.",
}

// capabilityToSkill maps capability dot-names to skill identifiers used by
// the ENABLED_SKILLS environment variable inside agent containers.
var capabilityToSkill = map[string]string{
	"browser.browse":    "web_browsing",
	"browser.extract":   "data_extraction",
	"browser.screenshot": "screenshot",
	"browser.fill":      "form_filling",
	"secret.request":    "secret_injection",
	"doc.ingest":        "document_ingest",
	"doc.summarize":     "document_summary",
	"doc.reference":     "document_reference",
	"email.read":        "email_read",
	"email.draft":       "email_draft",
	"email.send":        "email_send",
	"team.synthesize":   "team_synthesize",
	"team.request_hitl": "hitl_review",
	"team.review":       "team_review",
}

// GetSpecialization returns a SpecializationConfig for the given role name.
// It uses the installed RoleLookupFunc to resolve capabilities, maps them to
// skill names, and selects the appropriate system prompt.
func GetSpecialization(roleName string) (SpecializationConfig, error) {
	capabilities, err := activeRoleLookup(roleName)
	if err != nil {
		return SpecializationConfig{}, fmt.Errorf("studio: specialization: %w", err)
	}

	prompt, ok := RoleSystemPrompts[roleName]
	if !ok {
		return SpecializationConfig{}, fmt.Errorf("studio: no system prompt for role %q", roleName)
	}

	skills := capabilitiesToSkills(capabilities)

	return SpecializationConfig{
		Role:         roleName,
		Skills:       skills,
		SystemPrompt: prompt,
	}, nil
}

// capabilitiesToSkills converts a list of capability names to deduplicated,
// sorted skill names. Unknown capabilities are silently skipped.
func capabilitiesToSkills(caps []string) []string {
	seen := make(map[string]bool, len(caps))
	var skills []string
	for _, cap := range caps {
		if skill, ok := capabilityToSkill[cap]; ok && !seen[skill] {
			seen[skill] = true
			skills = append(skills, skill)
		}
	}
	sort.Strings(skills)
	return skills
}
