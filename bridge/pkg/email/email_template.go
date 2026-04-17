package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/armorclaw/bridge/pkg/secretary"
)

const (
	emailWorkflowTemplateName = "Email Analysis and Response"
	emailWorkflowTrigger      = "email:received"
)

func CreateEmailWorkflowTemplate(ctx context.Context, store secretary.Store, createdBy string) (string, error) {
	template := &secretary.TaskTemplate{
		ID:          "tmpl_email_analysis_response",
		Name:        emailWorkflowTemplateName,
		Description: "Analyzes incoming email and drafts or sends a response via bridge-local execution",
		IsActive:    true,
		CreatedBy:   createdBy,
		PIIRefs:     []string{"email_body", "email_subject", "sender_address"},
	}

	step1Config := map[string]interface{}{
		"execution_mode": "container",
		"prompt":         "Analyze the following email and determine the appropriate response. From: {{from}}, Subject: {{subject}}, Body: {{body_masked}}. Consider: Is this spam? Does it require a response? What tone should the response use? Are there PII fields that need approval before including in a response?",
		"max_tokens":     2000,
	}
	step1ConfigJSON, _ := json.Marshal(step1Config)

	step2Config := map[string]interface{}{
		"execution_mode": "bridge_local",
		"handler":        "email_send",
		"timeout":        300,
	}
	step2ConfigJSON, _ := json.Marshal(step2Config)

	template.Steps = []secretary.WorkflowStep{
		{
			StepID:     "step_1_analyze",
			Order:      0,
			Type:       secretary.StepAction,
			Name:       "Analyze Email Content",
			Config:     json.RawMessage(step1ConfigJSON),
			NextStepID: "step_2_send",
		},
		{
			StepID: "step_2_send",
			Order:  1,
			Type:   secretary.StepAction,
			Name:   "Send Email Response (Bridge-Local)",
			Config: json.RawMessage(step2ConfigJSON),
		},
	}

	template.Variables = json.RawMessage(`{
		"from": {"type": "string", "description": "Email sender address"},
		"to": {"type": "string", "description": "Email recipient address"},
		"subject": {"type": "string", "description": "Email subject line"},
		"body_masked": {"type": "string", "description": "Email body with PII masked"},
		"email_id": {"type": "string", "description": "Internal email tracking ID"}
	}`)

	if err := store.CreateTemplate(ctx, template); err != nil {
		return "", fmt.Errorf("create email workflow template: %w", err)
	}

	return template.ID, nil
}
