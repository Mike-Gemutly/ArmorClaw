package email

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

// EmailTeamTemplate is a reusable email template scoped to a team.
type EmailTeamTemplate struct {
	ID              string    `json:"id"`
	TeamID          string    `json:"team_id"`
	Name            string    `json:"name"`
	SubjectTemplate string    `json:"subject_template"`
	BodyTemplate    string    `json:"body_template"`
	CreatedAt       time.Time `json:"created_at"`
}

// RenderedEmail holds the result of rendering a template with variables.
type RenderedEmail struct {
	Subject  string `json:"subject"`
	BodyText string `json:"body_text"`
}

// TeamTemplateManager stores and renders per-team email templates in memory.
type TeamTemplateManager struct {
	templates map[string]*EmailTeamTemplate // keyed by "teamID:name"
	log       *logger.Logger
	mu        sync.RWMutex
}

// TeamTemplateManagerConfig holds the dependencies for constructing a TeamTemplateManager.
type TeamTemplateManagerConfig struct {
	Log *logger.Logger
}

// NewTeamTemplateManager creates a TeamTemplateManager ready to use.
func NewTeamTemplateManager(cfg TeamTemplateManagerConfig) *TeamTemplateManager {
	return &TeamTemplateManager{
		templates: make(map[string]*EmailTeamTemplate),
		log:       cfg.Log,
	}
}

// templateKey builds the composite map key from teamID and template name.
func templateKey(teamID, name string) string {
	return teamID + ":" + name
}

// CreateTemplate stores a new template. The caller should set ID and CreatedAt
// before calling; if they are zero they will be populated automatically.
func (tm *TeamTemplateManager) CreateTemplate(_ context.Context, tmpl EmailTeamTemplate) error {
	if tmpl.TeamID == "" {
		return fmt.Errorf("team_id is required")
	}
	if tmpl.Name == "" {
		return fmt.Errorf("template name is required")
	}

	key := templateKey(tmpl.TeamID, tmpl.Name)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.templates[key]; exists {
		return fmt.Errorf("template already exists: %s", key)
	}

	if tmpl.ID == "" {
		id, err := generateID()
		if err != nil {
			return fmt.Errorf("template id generation: %w", err)
		}
		tmpl.ID = id
	}
	if tmpl.CreatedAt.IsZero() {
		tmpl.CreatedAt = time.Now()
	}

	tm.templates[key] = &tmpl

	if tm.log != nil {
		tm.log.Info("template_created", "template_id", tmpl.ID, "team_id", tmpl.TeamID, "name", tmpl.Name)
	}
	return nil
}

// GetTemplate retrieves a template by teamID and name.
func (tm *TeamTemplateManager) GetTemplate(_ context.Context, teamID, templateName string) (*EmailTeamTemplate, error) {
	key := templateKey(teamID, templateName)

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tmpl, ok := tm.templates[key]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", key)
	}
	cp := *tmpl
	return &cp, nil
}

// ListTemplates returns all templates for a given team.
func (tm *TeamTemplateManager) ListTemplates(_ context.Context, teamID string) ([]EmailTeamTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var result []EmailTeamTemplate
	for k, t := range tm.templates {
		if strings.HasPrefix(k, teamID+":") {
			result = append(result, *t)
		}
	}
	return result, nil
}

// RenderTemplate applies variable substitution to the template identified by
// teamID and templateName. Variables use the {{key}} syntax. Missing variables
// are left as-is and a warning is logged.
func (tm *TeamTemplateManager) RenderTemplate(_ context.Context, teamID, templateName string, variables map[string]string) (*RenderedEmail, error) {
	tmpl, err := tm.GetTemplate(context.Background(), teamID, templateName)
	if err != nil {
		return nil, err
	}

	subject := substituteVars(tmpl.SubjectTemplate, variables, tm.log)
	body := substituteVars(tmpl.BodyTemplate, variables, tm.log)

	return &RenderedEmail{
		Subject:  subject,
		BodyText: body,
	}, nil
}

// substituteVars replaces all {{key}} occurrences with values from the map.
// Missing keys are left untouched and a warning is logged.
func substituteVars(tmpl string, variables map[string]string, log *logger.Logger) string {
	// Find all {{...}} tokens and replace matching ones.
	// Simple approach: iterate through variables and replace.
	result := tmpl
	for key, value := range variables {
		token := "{{" + key + "}}"
		result = strings.ReplaceAll(result, token, value)
	}

	// Log warning for any remaining unreplaced tokens.
	if log != nil && strings.Contains(result, "{{") {
		log.Warn("template_unresolved_variables", "template_fragment", result)
	}

	return result
}
