package secretary

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/armorclaw/bridge/pkg/studio"
)

//=============================================================================
// Interfaces for Testability
//=============================================================================

type LearnWebsiteServiceInterface interface {
	Learn(ctx context.Context, req *LearnWebsiteRequest) (*LearnWebsiteResult, error)
	ConfirmMapping(ctx context.Context, mapping *ConfirmedMapping) error
}

type BlindFillExecutorInterface interface {
	Execute(ctx context.Context, req *BlindFillRequest) (*BlindFillResult, error)
	Cancel(executionID string) error
	GetExecutionStatus(executionID string) (*BlindFillResult, bool)
	ListActiveExecutions() []string
	GetActiveExecutionCount() int
}

type TrustedWorkflowEngineInterface interface {
	Evaluate(ctx context.Context, req *TrustEvaluationRequest) (*TrustEvaluationResult, error)
	CreatePolicy(ctx context.Context, policy *TrustedWorkflowPolicy) error
	GetPolicy(ctx context.Context, id string) (*TrustedWorkflowPolicy, error)
	ListPolicies(ctx context.Context, activeOnly bool) ([]*TrustedWorkflowPolicy, error)
	DeletePolicy(ctx context.Context, id string) error
	RevokePolicy(ctx context.Context, id string, revokedBy string, reason string) error
}

//=============================================================================
// Secretary Command Handler
//=============================================================================

type SecretaryCommandHandler struct {
	store        SecretaryStore
	orchestrator *WorkflowOrchestratorImpl
	studio       *StudioIntegration
	matrix       studio.MatrixAdapter
	prefix       string

	learnWebsite   LearnWebsiteServiceInterface
	blindFill      BlindFillExecutorInterface
	trustEngine    TrustedWorkflowEngineInterface
	approvalEngine *ApprovalEngineImpl
}

type SecretaryCommandHandlerConfig struct {
	Store        SecretaryStore
	Orchestrator *WorkflowOrchestratorImpl
	Studio       *StudioIntegration
	Matrix       studio.MatrixAdapter
	Prefix       string

	LearnWebsite   LearnWebsiteServiceInterface
	BlindFill      BlindFillExecutorInterface
	TrustEngine    TrustedWorkflowEngineInterface
	ApprovalEngine *ApprovalEngineImpl
}

func NewSecretaryCommandHandler(cfg SecretaryCommandHandlerConfig) *SecretaryCommandHandler {
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "!"
	}

	return &SecretaryCommandHandler{
		store:          cfg.Store,
		orchestrator:   cfg.Orchestrator,
		studio:         cfg.Studio,
		matrix:         cfg.Matrix,
		prefix:         prefix,
		learnWebsite:   cfg.LearnWebsite,
		blindFill:      cfg.BlindFill,
		trustEngine:    cfg.TrustEngine,
		approvalEngine: cfg.ApprovalEngine,
	}
}

// HandleMessage processes Matrix messages for Secretary commands
func (h *SecretaryCommandHandler) HandleMessage(ctx context.Context, roomID, userID, eventID, text string) (bool, error) {
	text = strings.TrimSpace(text)

	if !strings.HasPrefix(text, h.prefix+"secretary ") {
		return false, nil
	}

	text = strings.TrimPrefix(text, h.prefix+"secretary ")
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return true, h.sendHelp(ctx, roomID)
	}

	subCmd := parts[0]
	args := parts[1:]

	switch subCmd {
	case "help":
		return true, h.sendHelp(ctx, roomID)
	case "create":
		if len(args) >= 2 && args[0] == "workflow" {
			return true, h.handleCreateWorkflow(ctx, roomID, userID, args[1:])
		}
	case "start":
		if len(args) >= 2 && args[0] == "workflow" {
			return true, h.handleStartWorkflow(ctx, roomID, args[1])
		}
	case "list":
		if len(args) >= 1 {
			switch args[0] {
			case "workflows":
				return true, h.handleListWorkflows(ctx, roomID)
			case "agents":
				return true, h.handleListAgents(ctx, roomID)
			case "templates":
				return true, h.handleListTemplates(ctx, roomID)
			case "trust", "trust-policies", "trusted":
				return true, h.handleListTrustPolicies(ctx, roomID)
			}
		}
	case "workflow":
		if len(args) >= 1 {
			switch args[0] {
			case "status":
				if len(args) >= 2 {
					return true, h.handleWorkflowStatus(ctx, roomID, args[1])
				}
			case "cancel":
				if len(args) >= 2 {
					return true, h.handleCancelWorkflow(ctx, roomID, args[1])
				}
			}
		}
	case "delete":
		if len(args) >= 2 && args[0] == "agent" {
			return true, h.handleDeleteAgent(ctx, roomID, args[1])
		}
	case "learn":
		if len(args) >= 2 && args[0] == "website" {
			return true, h.handleLearnWebsite(ctx, roomID, userID, args[1:])
		}
	case "review":
		if len(args) >= 2 && args[0] == "mapping" {
			return true, h.handleReviewMapping(ctx, roomID, args[1:])
		}
	case "confirm":
		if len(args) >= 2 && args[0] == "mapping" {
			return true, h.handleConfirmMapping(ctx, roomID, userID, args[1:])
		}
	case "run":
		if len(args) >= 2 && args[0] == "blindfill" {
			return true, h.handleRunBlindFill(ctx, roomID, userID, args[1:])
		}
	case "trust":
		if len(args) >= 1 {
			switch args[0] {
			case "list":
				return true, h.handleListTrustPolicies(ctx, roomID)
			case "create":
				return true, h.handleCreateTrustPolicy(ctx, roomID, userID, args[1:])
			case "revoke":
				if len(args) >= 2 {
					return true, h.handleRevokeTrustPolicy(ctx, roomID, userID, args[1:])
				}
			}
		}
	default:
		return true, h.sendHelp(ctx, roomID)
	}

	return false, nil
}

func (h *SecretaryCommandHandler) sendHelp(ctx context.Context, roomID string) error {
	help := `📋 Secretary Commands

**Workflow Management:**
` + h.prefix + `secretary help                    - Show this help
` + h.prefix + `secretary create workflow <id>    - Create workflow from template
` + h.prefix + `secretary start workflow <id>     - Start a pending workflow
` + h.prefix + `secretary list workflows          - List all workflows
` + h.prefix + `secretary workflow status <id>    - Show workflow status
` + h.prefix + `secretary workflow cancel <id>    - Cancel running workflow
` + h.prefix + `secretary list agents             - List secretary agents
` + h.prefix + `secretary delete agent <id>       - Delete secretary agent
` + h.prefix + `secretary list templates          - List available templates

**Learn Website (Form Discovery):**
` + h.prefix + `secretary learn website <url>     - Discover form fields on a website
` + h.prefix + `secretary review mapping <id>     - Review a mapping draft
` + h.prefix + `secretary confirm mapping <id> [fields...] - Confirm field mappings

**BlindFill Execution:**
` + h.prefix + `secretary run blindfill <template> [options] - Execute BlindFill

**Trusted Workflows:**
` + h.prefix + `secretary trust list              - List trust policies
` + h.prefix + `secretary trust create <name>     - Create a trust policy
` + h.prefix + `secretary trust revoke <id>       - Revoke a trust policy

**Examples:**
  ` + h.prefix + `secretary learn website https://example.com/form
  ` + h.prefix + `secretary review mapping draft_abc123
  ` + h.prefix + `secretary confirm mapping draft_abc123
  ` + h.prefix + `secretary run blindfill template_xyz
  ` + h.prefix + `secretary trust list
`
	return h.matrix.SendMessage(ctx, roomID, help)
}

func (h *SecretaryCommandHandler) handleCreateWorkflow(ctx context.Context, roomID, userID string, args []string) error {
	if len(args) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: !secretary create workflow <template_id>")
	}

	templateID := args[0]

	if h.studio == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Studio integration not configured")
	}

	workflow, err := h.studio.CreateWorkflowFromTemplate(ctx, templateID, nil, userID)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to create workflow: %v", err))
	}

	msg := fmt.Sprintf("✅ Workflow created\n\nID: `%s`\nStatus: %s", workflow.ID, workflow.Status)
	return h.matrix.SendMessage(ctx, roomID, msg)
}

func (h *SecretaryCommandHandler) handleStartWorkflow(ctx context.Context, roomID, workflowID string) error {
	if h.orchestrator == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Orchestrator not configured")
	}

	if err := h.orchestrator.StartWorkflow(workflowID); err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to start workflow: %v", err))
	}

	workflow, _ := h.orchestrator.GetWorkflow(workflowID)
	msg := fmt.Sprintf("✅ Workflow started\n\nID: `%s`\nStatus: %s", workflowID, workflow.Status)
	return h.matrix.SendMessage(ctx, roomID, msg)
}

func (h *SecretaryCommandHandler) handleListWorkflows(ctx context.Context, roomID string) error {
	if h.store == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Store not configured")
	}

	workflows, err := h.store.ListWorkflows(ctx, WorkflowFilter{})
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to list workflows: %v", err))
	}

	if len(workflows) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "No workflows found.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Workflows (%d)\n\n", len(workflows)))
	for _, wf := range workflows {
		sb.WriteString(fmt.Sprintf("**%s** `%s` - %s\n\n", wf.Name, wf.ID, wf.Status))
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *SecretaryCommandHandler) handleWorkflowStatus(ctx context.Context, roomID, workflowID string) error {
	if h.store == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Store not configured")
	}

	workflow, err := h.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Workflow not found: %s", workflowID))
	}

	msg := fmt.Sprintf(`📋 Workflow: %s

ID: %s
Status: %s
Started: %s
`, workflow.Name, workflow.ID, workflow.Status, workflow.StartedAt.Format("2006-01-02 15:04"))

	return h.matrix.SendMessage(ctx, roomID, msg)
}

func (h *SecretaryCommandHandler) handleCancelWorkflow(ctx context.Context, roomID, workflowID string) error {
	if h.orchestrator != nil {
		if err := h.orchestrator.CancelWorkflow(workflowID, "cancelled via command"); err != nil {
			return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to cancel workflow: %v", err))
		}
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("✅ Workflow `%s` cancelled", workflowID))
	}

	if h.store == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Store not configured")
	}

	workflow, err := h.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Workflow not found: %s", workflowID))
	}

	if workflow.Status != StatusRunning {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Workflow is not running: %s (status: %s)", workflowID, workflow.Status))
	}

	workflow.Status = StatusCancelled
	if err := h.store.UpdateWorkflow(ctx, workflow); err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to cancel workflow: %v", err))
	}

	return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("✅ Workflow `%s` cancelled", workflowID))
}

func (h *SecretaryCommandHandler) handleListAgents(ctx context.Context, roomID string) error {
	if h.studio == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Studio integration not configured")
	}

	agents, err := h.studio.ListSecretaryAgents(ctx)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to list agents: %v", err))
	}

	if len(agents) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "No secretary agents running.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🤖 Secretary Agents (%d)\n\n", len(agents)))
	for _, agent := range agents {
		sb.WriteString(fmt.Sprintf("**%s** `%s` - %s\n\n", agent.DefinitionID, agent.ID, agent.Status))
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *SecretaryCommandHandler) handleDeleteAgent(ctx context.Context, roomID, agentID string) error {
	if h.studio == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Studio integration not configured")
	}

	if err := h.studio.DeleteSecretaryAgent(ctx, agentID); err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to delete agent: %v", err))
	}

	return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("✅ Agent `%s` deleted", agentID))
}

func (h *SecretaryCommandHandler) handleListTemplates(ctx context.Context, roomID string) error {
	if h.store == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Store not configured")
	}

	templates, err := h.store.ListTemplates(ctx, TemplateFilter{})
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to list templates: %v", err))
	}

	if len(templates) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "No templates found.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Templates (%d)\n\n", len(templates)))
	for _, t := range templates {
		status := "inactive"
		if t.IsActive {
			status = "active"
		}
		sb.WriteString(fmt.Sprintf("**%s** `%s` - %s\n", t.Name, t.ID, status))
		if t.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", t.Description))
		}
		sb.WriteString("\n")
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *SecretaryCommandHandler) handleLearnWebsite(ctx context.Context, roomID, userID string, args []string) error {
	if h.learnWebsite == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Learn Website service not configured")
	}

	if len(args) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: "+h.prefix+"secretary learn website <url> [form_selector]")
	}

	targetURL := args[0]
	var formSelector string
	if len(args) > 1 {
		formSelector = args[1]
	}

	h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("🔍 Learning website: %s", targetURL))

	req := &LearnWebsiteRequest{
		TargetURL:    targetURL,
		Initiator:    userID,
		FormSelector: formSelector,
		Timeout:      30000,
		WaitUntil:    "networkidle",
	}

	result, err := h.learnWebsite.Learn(ctx, req)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to learn website: %v", err))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Website learned successfully\n\n"))
	sb.WriteString(fmt.Sprintf("**Request ID:** `%s`\n", result.RequestID))
	sb.WriteString(fmt.Sprintf("**URL:** %s\n", result.TargetURL))
	if result.PageTitle != "" {
		sb.WriteString(fmt.Sprintf("**Page Title:** %s\n", result.PageTitle))
	}
	sb.WriteString(fmt.Sprintf("**Fields Discovered:** %d\n\n", len(result.Fields)))

	if result.MappingDraft != nil {
		sb.WriteString(fmt.Sprintf("**Mapping Draft ID:** `%s`\n", result.MappingDraft.ID))
		sb.WriteString(fmt.Sprintf("**Template Name:** %s\n\n", result.MappingDraft.TemplateName))
	}

	sb.WriteString("**Discovered Fields:**\n")
	for i, field := range result.Fields {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("  ... and %d more fields\n", len(result.Fields)-10))
			break
		}
		sb.WriteString(fmt.Sprintf("  • %s (`%s`) - %s\n", field.LabelText, field.Selector, field.TagName))
	}

	if len(result.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("\n⚠️ **Warnings:** %d\n", len(result.Warnings)))
	}

	sb.WriteString(fmt.Sprintf("\nTo review and confirm mapping: `%ssecretary review mapping %s`", h.prefix, result.MappingDraft.ID))

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *SecretaryCommandHandler) handleReviewMapping(ctx context.Context, roomID string, args []string) error {
	if len(args) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: "+h.prefix+"secretary review mapping <draft_id>")
	}

	draftID := args[0]

	if h.store == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Store not configured")
	}

	templates, err := h.store.ListTemplates(ctx, TemplateFilter{})
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to search templates: %v", err))
	}

	var foundTemplate *TaskTemplate
	for i := range templates {
		if templates[i].ID == draftID || strings.Contains(templates[i].Name, draftID) {
			foundTemplate = &templates[i]
			break
		}
	}

	if foundTemplate == nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Mapping draft not found: %s", draftID))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 **Mapping Draft Review**\n\n"))
	sb.WriteString(fmt.Sprintf("**ID:** `%s`\n", foundTemplate.ID))
	sb.WriteString(fmt.Sprintf("**Name:** %s\n", foundTemplate.Name))
	if foundTemplate.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description:** %s\n", foundTemplate.Description))
	}
	sb.WriteString(fmt.Sprintf("**Created:** %s\n", foundTemplate.CreatedAt.Format("2006-01-02 15:04")))
	sb.WriteString(fmt.Sprintf("**Steps:** %d\n\n", len(foundTemplate.Steps)))

	sb.WriteString("**Field Mappings:**\n")
	for _, step := range foundTemplate.Steps {
		sb.WriteString(fmt.Sprintf("  %d. **%s** (`%s`)\n", step.Order+1, step.Name, step.StepID))
	}

	if len(foundTemplate.PIIRefs) > 0 {
		sb.WriteString(fmt.Sprintf("\n**PII Fields Required:** %d\n", len(foundTemplate.PIIRefs)))
		for _, ref := range foundTemplate.PIIRefs {
			sb.WriteString(fmt.Sprintf("  • %s\n", ref))
		}
	}

	sb.WriteString(fmt.Sprintf("\nTo confirm: `%ssecretary confirm mapping %s`", h.prefix, draftID))

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *SecretaryCommandHandler) handleConfirmMapping(ctx context.Context, roomID, userID string, args []string) error {
	if h.learnWebsite == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Learn Website service not configured")
	}

	if len(args) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: "+h.prefix+"secretary confirm mapping <draft_id> [field1=value1 field2=value2 ...]")
	}

	draftID := args[0]

	fieldOverrides := make(map[string]string)
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			fieldOverrides[parts[0]] = parts[1]
		}
	}

	h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("⏳ Confirming mapping `%s`...", draftID))

	mapping := &ConfirmedMapping{
		DraftID:       draftID,
		TemplateName:  draftID,
		ConfirmedBy:   userID,
		ConfirmedAt:   timeNow(),
		FieldMappings: []ConfirmedFieldMapping{},
	}

	if err := h.learnWebsite.ConfirmMapping(ctx, mapping); err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to confirm mapping: %v", err))
	}

	msg := fmt.Sprintf("✅ Mapping confirmed and template created\n\n**Template ID:** `%s`\n**Created by:** %s",
		draftID, userID)

	return h.matrix.SendMessage(ctx, roomID, msg)
}

func (h *SecretaryCommandHandler) handleRunBlindFill(ctx context.Context, roomID, userID string, args []string) error {
	if h.blindFill == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ BlindFill executor not configured")
	}

	if len(args) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: "+h.prefix+"secretary run blindfill <template_id|workflow_id> [url] [dryrun]")
	}

	templateID := args[0]
	var targetURL string
	dryRun := false

	for _, arg := range args[1:] {
		if arg == "dryrun" || arg == "--dry-run" {
			dryRun = true
		} else if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			targetURL = arg
		}
	}

	h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("⏳ Starting BlindFill execution for template `%s`...", templateID))

	req := &BlindFillRequest{
		TemplateID:      templateID,
		TargetURL:       targetURL,
		Initiator:       userID,
		DryRun:          dryRun,
		Timeout:         120000,
		ContinueOnError: false,
	}

	result, err := h.blindFill.Execute(ctx, req)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ BlindFill execution failed: %v", err))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 **BlindFill Result**\n\n"))
	sb.WriteString(fmt.Sprintf("**Execution ID:** `%s`\n", result.ExecutionID))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", result.Status))
	sb.WriteString(fmt.Sprintf("**Duration:** %dms\n\n", result.Duration))

	if len(result.StepResults) > 0 {
		sb.WriteString(fmt.Sprintf("**Steps Executed:** %d\n", len(result.StepResults)))
		for _, step := range result.StepResults {
			statusIcon := "✅"
			if step.Status == "error" {
				statusIcon = "❌"
			} else if step.Status == "skipped" {
				statusIcon = "⏭️"
			}
			sb.WriteString(fmt.Sprintf("  %s %s (%dms)\n", statusIcon, step.StepName, step.ExecutionTime))
		}
	}

	if len(result.FilledFields) > 0 {
		sb.WriteString(fmt.Sprintf("\n**Fields Filled:** %d\n", len(result.FilledFields)))
	}

	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("\n❌ **Error:** %s\n", result.Error))
	}

	if result.ApprovalRequired {
		sb.WriteString(fmt.Sprintf("\n⚠️ **Approval Required**\n"))
		sb.WriteString(fmt.Sprintf("**Request ID:** `%s`\n", result.ApprovalRequestID))
		if len(result.RequiredPIIFields) > 0 {
			sb.WriteString("**PII Fields:**\n")
			for _, field := range result.RequiredPIIFields {
				sb.WriteString(fmt.Sprintf("  • %s\n", field))
			}
		}
	}

	if len(result.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("\n⚠️ **Warnings:**\n"))
		for _, w := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  • %s\n", w))
		}
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *SecretaryCommandHandler) handleListTrustPolicies(ctx context.Context, roomID string) error {
	if h.trustEngine == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Trust engine not configured")
	}

	policies, err := h.trustEngine.ListPolicies(ctx, true)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to list trust policies: %v", err))
	}

	if len(policies) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "No trust policies found.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔐 **Trust Policies** (%d active)\n\n", len(policies)))

	for _, p := range policies {
		status := "✅ active"
		if p.RevokedAt != nil {
			status = "❌ revoked"
		} else if p.ExpiresAt != nil && timeNow().After(*p.ExpiresAt) {
			status = "⏰ expired"
		}

		sb.WriteString(fmt.Sprintf("**%s** `%s` - %s\n", p.Name, p.ID, status))
		if p.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", p.Description))
		}

		sb.WriteString("  Scope: ")
		if len(p.Scope.TemplateIDs) > 0 {
			sb.WriteString(fmt.Sprintf("%d templates, ", len(p.Scope.TemplateIDs)))
		}
		if len(p.Scope.Initiators) > 0 {
			sb.WriteString(fmt.Sprintf("%d initiators, ", len(p.Scope.Initiators)))
		}
		if len(p.AllowedPIIRefs) > 0 {
			sb.WriteString(fmt.Sprintf("%d PII refs allowed", len(p.AllowedPIIRefs)))
		}
		sb.WriteString("\n")

		if p.MaxExecutions > 0 {
			sb.WriteString(fmt.Sprintf("  Executions: %d/%d\n", p.ExecutionCount, p.MaxExecutions))
		}
		if p.ExpiresAt != nil {
			sb.WriteString(fmt.Sprintf("  Expires: %s\n", p.ExpiresAt.Format("2006-01-02 15:04")))
		}
		sb.WriteString("\n")
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *SecretaryCommandHandler) handleCreateTrustPolicy(ctx context.Context, roomID, userID string, args []string) error {
	if h.trustEngine == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Trust engine not configured")
	}

	if len(args) == 0 {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Usage: %ssecretary trust create <name> [template=...] [initiator=...] [expires=...] [max_executions=...]", h.prefix))
	}

	name := args[0]

	policy := &TrustedWorkflowPolicy{
		Name:      name,
		CreatedBy: userID,
		Scope:     TrustScope{},
	}

	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		switch key {
		case "template":
			policy.Scope.TemplateIDs = []string{value}
		case "initiator":
			policy.Scope.Initiators = []string{value}
		case "subject":
			policy.Scope.Subjects = []string{value}
		case "pii_ref", "pii":
			policy.AllowedPIIRefs = []string{value}
		case "pii_class":
			policy.AllowedPIIClasses = []string{value}
		case "domain":
			policy.Scope.AllowedDomains = []string{value}
		}
	}

	if err := h.trustEngine.CreatePolicy(ctx, policy); err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to create trust policy: %v", err))
	}

	msg := fmt.Sprintf("✅ Trust policy created\n\n**ID:** `%s`\n**Name:** %s\n**Created by:** %s",
		policy.ID, policy.Name, userID)

	return h.matrix.SendMessage(ctx, roomID, msg)
}

func (h *SecretaryCommandHandler) handleRevokeTrustPolicy(ctx context.Context, roomID, userID string, args []string) error {
	if h.trustEngine == nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Trust engine not configured")
	}

	if len(args) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: "+h.prefix+"secretary trust revoke <policy_id> [reason]")
	}

	policyID := args[0]
	reason := "Revoked via command"
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}

	if err := h.trustEngine.RevokePolicy(ctx, policyID, userID, reason); err != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to revoke trust policy: %v", err))
	}

	msg := fmt.Sprintf("✅ Trust policy revoked\n\n**ID:** `%s`\n**Revoked by:** %s\n**Reason:** %s",
		policyID, userID, reason)

	return h.matrix.SendMessage(ctx, roomID, msg)
}

func timeNow() time.Time {
	return time.Now()
}
