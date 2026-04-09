package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//=============================================================================
// Matrix Command Interface
//=============================================================================

// MatrixAdapter defines the interface for Matrix operations
type MatrixAdapter interface {
	SendMessage(ctx context.Context, roomID, message string) error
	SendFormattedMessage(ctx context.Context, roomID, plainBody, formattedBody string) error
	ReplyToEvent(ctx context.Context, roomID, eventID, message string) error
}

// CommandHandler handles Matrix commands for the Agent Studio
type CommandHandler struct {
	store          Store
	factory        *AgentFactory
	skillRegistry  *SkillRegistry
	piiRegistry    *PIIRegistry
	profileManager *ProfileManager
	matrix         MatrixAdapter

	// Wizard state tracking
	wizardMu sync.RWMutex
	wizards  map[string]*WizardState // keyed by userID

	// Configuration
	prefix  string // Command prefix, default "!"
	timeout time.Duration
}

// CommandHandlerConfig holds configuration
type CommandHandlerConfig struct {
	Store         Store
	Factory       *AgentFactory
	Matrix        MatrixAdapter
	CommandPrefix string
	WizardTimeout time.Duration
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(cfg CommandHandlerConfig) *CommandHandler {
	prefix := cfg.CommandPrefix
	if prefix == "" {
		prefix = "!"
	}

	timeout := cfg.WizardTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	return &CommandHandler{
		store:          cfg.Store,
		factory:        cfg.Factory,
		skillRegistry:  NewSkillRegistry(cfg.Store),
		piiRegistry:    NewPIIRegistry(cfg.Store),
		profileManager: NewProfileManager(cfg.Store),
		matrix:         cfg.Matrix,
		wizards:        make(map[string]*WizardState),
		prefix:         prefix,
		timeout:        timeout,
	}
}

//=============================================================================
// Command Parsing
//=============================================================================

// Command represents a parsed Matrix command
type Command struct {
	Prefix  string
	Name    string
	Args    []string
	KVArgs  map[string]string
	RawText string
}

// ParseCommand parses a Matrix message into a Command
func ParseCommand(text, prefix string) (*Command, bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, prefix) {
		return nil, false
	}

	// Remove prefix
	text = strings.TrimPrefix(text, prefix)

	// Split into parts
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil, false
	}

	cmd := &Command{
		Prefix:  prefix,
		Name:    strings.ToLower(parts[0]),
		Args:    []string{},
		KVArgs:  make(map[string]string),
		RawText: text,
	}

	// Parse remaining args
	for _, part := range parts[1:] {
		// Check for key=value format
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				// Remove quotes if present
				value := strings.Trim(kv[1], `"`)
				cmd.KVArgs[kv[0]] = value
				continue
			}
		}
		cmd.Args = append(cmd.Args, part)
	}

	return cmd, true
}

//=============================================================================
// Command Handling
//=============================================================================

// HandleMessage processes a Matrix message for commands
func (h *CommandHandler) HandleMessage(ctx context.Context, roomID, userID, eventID, text string) (bool, error) {
	// Check for ongoing wizard
	h.wizardMu.RLock()
	wizard, hasWizard := h.wizards[userID]
	h.wizardMu.RUnlock()

	if hasWizard && !wizard.IsExpired() {
		return h.handleWizardInput(ctx, roomID, userID, text)
	}

	// Parse command
	cmd, ok := ParseCommand(text, h.prefix)
	if !ok {
		return false, nil
	}

	// Check if it's an agent command
	if cmd.Name != "agent" && cmd.Name != "agents" {
		return false, nil
	}

	// Route to subcommand
	subCmd := ""
	if len(cmd.Args) > 0 {
		subCmd = strings.ToLower(cmd.Args[0])
	}

	switch subCmd {
	case "help":
		return true, h.handleHelp(ctx, roomID)
	case "list-skills", "skills":
		return true, h.handleListSkills(ctx, roomID, cmd)
	case "list-pii", "pii":
		return true, h.handleListPII(ctx, roomID, cmd)
	case "list-profiles", "profiles":
		return true, h.handleListProfiles(ctx, roomID)
	case "create":
		return true, h.handleCreateStart(ctx, roomID, userID, cmd)
	case "list", "ls":
		return true, h.handleListAgents(ctx, roomID, cmd)
	case "show", "get":
		return true, h.handleShowAgent(ctx, roomID, cmd)
	case "spawn", "run":
		return true, h.handleSpawnAgent(ctx, roomID, userID, cmd)
	case "stop":
		return true, h.handleStopInstance(ctx, roomID, userID, cmd)
	case "delete", "rm":
		return true, h.handleDeleteAgent(ctx, roomID, userID, cmd)
	case "stats":
		return true, h.handleStats(ctx, roomID)
	default:
		return true, h.handleHelp(ctx, roomID)
	}
}

//=============================================================================
// Help Command
//=============================================================================

func (h *CommandHandler) handleHelp(ctx context.Context, roomID string) error {
	help := fmt.Sprintf(`🤖 Agent Studio Commands

%sagent help              - Show this help
%sagent list-skills       - List available skills
%sagent list-pii          - List PII fields
%sagent list-profiles     - List resource profiles
%sagent create name="X"   - Start creation wizard
%sagent list              - List agent definitions
%sagent show <id>         - Show agent details
%sagent spawn <id>        - Spawn agent instance
%sagent stop <instance>   - Stop running instance
%sagent delete <id>       - Delete agent definition
%sagent stats             - Show studio statistics

Examples:
  !agent create name="Contracts Agent"
  !agent spawn agent_123
  !agent list-skills category=document`, h.prefix, h.prefix, h.prefix, h.prefix, h.prefix, h.prefix, h.prefix, h.prefix, h.prefix, h.prefix, h.prefix)

	return h.matrix.SendMessage(ctx, roomID, help)
}

//=============================================================================
// List Skills Command
//=============================================================================

func (h *CommandHandler) handleListSkills(ctx context.Context, roomID string, cmd *Command) error {
	category := cmd.KVArgs["category"]

	skills, err := h.skillRegistry.List(category)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Failed to list skills: "+err.Error())
	}

	if len(skills) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "No skills found.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 Available Skills (%d)\n\n", len(skills)))

	// Group by category
	byCategory := make(map[string][]*Skill)
	for _, skill := range skills {
		byCategory[skill.Category] = append(byCategory[skill.Category], skill)
	}

	for cat, catSkills := range byCategory {
		sb.WriteString(fmt.Sprintf("**%s**\n", strings.Title(cat)))
		for _, s := range catSkills {
			sb.WriteString(fmt.Sprintf("  %d. `%s` - %s\n", len(sb.String())/50+1, s.ID, s.Name))
		}
		sb.WriteString("\n")
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

//=============================================================================
// List PII Command
//=============================================================================

func (h *CommandHandler) handleListPII(ctx context.Context, roomID string, cmd *Command) error {
	sensitivity := cmd.KVArgs["sensitivity"]

	fields, err := h.piiRegistry.List(sensitivity)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Failed to list PII fields: "+err.Error())
	}

	if len(fields) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "No PII fields found.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔒 PII Fields (%d)\n\n", len(fields)))

	// Group by sensitivity
	bySensitivity := map[string][]*PIIField{
		"low":      {},
		"medium":   {},
		"high":     {},
		"critical": {},
	}

	for _, field := range fields {
		bySensitivity[field.Sensitivity] = append(bySensitivity[field.Sensitivity], field)
	}

	sensitivityOrder := []string{"low", "medium", "high", "critical"}
	icons := map[string]string{"low": "🟢", "medium": "🟡", "high": "🟠", "critical": "🔴"}

	for _, sens := range sensitivityOrder {
		sensFields := bySensitivity[sens]
		if len(sensFields) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s **%s**\n", icons[sens], strings.Title(sens)))
		for _, f := range sensFields {
			approval := ""
			if f.RequiresApproval {
				approval = " (requires approval)"
			}
			sb.WriteString(fmt.Sprintf("  • `%s` - %s%s\n", f.ID, f.Name, approval))
		}
		sb.WriteString("\n")
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

//=============================================================================
// List Profiles Command
//=============================================================================

func (h *CommandHandler) handleListProfiles(ctx context.Context, roomID string) error {
	profiles := h.profileManager.List()

	var sb strings.Builder
	sb.WriteString("⚙️ Resource Profiles\n\n")

	for tier, profile := range profiles {
		sb.WriteString(fmt.Sprintf("**%s** - %s\n", strings.Title(tier), profile.Description))
		sb.WriteString(fmt.Sprintf("  Memory: %dMB | CPU: %d | Timeout: %dm | Max: %d\n\n",
			profile.MemoryMB, profile.CPUShares, profile.TimeoutSeconds/60, profile.MaxConcurrency))
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

//=============================================================================
// Create Agent Wizard
//=============================================================================

func (h *CommandHandler) handleCreateStart(ctx context.Context, roomID, userID string, cmd *Command) error {
	name := cmd.KVArgs["name"]
	if name == "" {
		return h.matrix.SendMessage(ctx, roomID, "❌ Please provide a name: `!agent create name=\"My Agent\"`")
	}

	// Get skills for wizard
	skills, err := h.skillRegistry.List("")
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Failed to get skills: "+err.Error())
	}

	// Create wizard state
	wizard := &WizardState{
		UserID:    userID,
		RoomID:    roomID,
		Step:      WizardStepSkills,
		Name:      name,
		Tier:      "medium",
		ExpiresAt: time.Now().Add(h.timeout),
	}

	h.wizardMu.Lock()
	h.wizards[userID] = wizard
	h.wizardMu.Unlock()

	// Send first step
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🤖 Agent Creation Wizard: \"%s\"\n\n", name))
	sb.WriteString("Step 1: Select Skills\n")
	sb.WriteString("Reply with skill numbers (comma-separated):\n\n")

	for i, skill := range skills {
		sb.WriteString(fmt.Sprintf("%d. `%s` - %s\n", i+1, skill.ID, skill.Name))
	}

	sb.WriteString("\n_Reply 'cancel' to abort_")

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

// handleWizardInput processes wizard step responses
func (h *CommandHandler) handleWizardInput(ctx context.Context, roomID, userID, input string) (bool, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	// Check for cancel
	if input == "cancel" || input == "abort" {
		h.wizardMu.Lock()
		delete(h.wizards, userID)
		h.wizardMu.Unlock()
		return true, h.matrix.SendMessage(ctx, roomID, "❌ Agent creation cancelled.")
	}

	h.wizardMu.RLock()
	wizard, ok := h.wizards[userID]
	h.wizardMu.RUnlock()

	if !ok || wizard.IsExpired() {
		return true, h.matrix.SendMessage(ctx, roomID, "❌ Wizard session expired. Start again with `!agent create`")
	}

	switch wizard.Step {
	case WizardStepSkills:
		return h.handleWizardSkills(ctx, roomID, wizard, input)
	case WizardStepPII:
		return h.handleWizardPII(ctx, roomID, wizard, input)
	case WizardStepResources:
		return h.handleWizardResources(ctx, roomID, wizard, input)
	case WizardStepConfirm:
		return h.handleWizardConfirm(ctx, roomID, userID, wizard, input)
	}

	return true, nil
}

func (h *CommandHandler) handleWizardSkills(ctx context.Context, roomID string, wizard *WizardState, input string) (bool, error) {
	// Parse skill numbers
	numbers, err := parseNumbers(input)
	if err != nil {
		return true, h.matrix.SendMessage(ctx, roomID, "❌ Invalid input. Please enter numbers like: 1,3,5")
	}

	// Get all skills
	skills, err := h.skillRegistry.List("")
	if err != nil {
		return true, h.matrix.SendMessage(ctx, roomID, "❌ Failed to get skills: "+err.Error())
	}

	// Map numbers to skill IDs
	var selectedSkills []string
	for _, num := range numbers {
		if num >= 1 && num <= len(skills) {
			selectedSkills = append(selectedSkills, skills[num-1].ID)
		}
	}

	if len(selectedSkills) == 0 {
		return true, h.matrix.SendMessage(ctx, roomID, "❌ Please select at least one valid skill.")
	}

	wizard.Skills = selectedSkills
	wizard.Step = WizardStepPII

	// Get PII fields for next step
	fields, _ := h.piiRegistry.List("")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ Selected skills: %s\n\n", strings.Join(selectedSkills, ", ")))
	sb.WriteString("Step 2: Select PII Access\n")
	sb.WriteString("What data can this agent access?\n\n")

	for i, field := range fields {
		icon := "🟢"
		if field.RequiresApproval {
			icon = "🔒"
		}
		sb.WriteString(fmt.Sprintf("%d. %s `%s` - %s\n", i+1, icon, field.ID, field.Name))
	}

	sb.WriteString("\n_Reply with numbers, or 'none' for no PII access_")

	return true, h.matrix.SendMessage(ctx, roomID, sb.String())
}

func (h *CommandHandler) handleWizardPII(ctx context.Context, roomID string, wizard *WizardState, input string) (bool, error) {
	if input == "none" {
		wizard.PIIAccess = []string{}
	} else {
		numbers, err := parseNumbers(input)
		if err != nil {
			return true, h.matrix.SendMessage(ctx, roomID, "❌ Invalid input. Use numbers or 'none'.")
		}

		fields, _ := h.piiRegistry.List("")
		for _, num := range numbers {
			if num >= 1 && num <= len(fields) {
				wizard.PIIAccess = append(wizard.PIIAccess, fields[num-1].ID)
			}
		}
	}

	wizard.Step = WizardStepResources

	return true, h.matrix.SendMessage(ctx, roomID, fmt.Sprintf(`✅ PII access: %s

Step 3: Resource Tier

1. low - 256MB, 5min timeout (quick tasks)
2. medium - 512MB, 10min timeout (standard)
3. high - 2GB, 30min timeout (heavy processing)

Reply with 1, 2, or 3 (default: medium)`, formatList(wizard.PIIAccess)))
}

func (h *CommandHandler) handleWizardResources(ctx context.Context, roomID string, wizard *WizardState, input string) (bool, error) {
	switch input {
	case "1", "low":
		wizard.Tier = "low"
	case "2", "medium", "":
		wizard.Tier = "medium"
	case "3", "high":
		wizard.Tier = "high"
	default:
		wizard.Tier = "medium"
	}

	wizard.Step = WizardStepConfirm

	return true, h.matrix.SendMessage(ctx, roomID, fmt.Sprintf(`✅ Resource tier: %s

Step 4: Confirm Creation

Name: %s
Skills: %s
PII Access: %s
Resources: %s

Reply 'yes' to create, or 'cancel' to abort.`,
		wizard.Tier,
		wizard.Name,
		strings.Join(wizard.Skills, ", "),
		formatList(wizard.PIIAccess),
		wizard.Tier))
}

func (h *CommandHandler) handleWizardConfirm(ctx context.Context, roomID, userID string, wizard *WizardState, input string) (bool, error) {
	if input != "yes" && input != "confirm" {
		h.wizardMu.Lock()
		delete(h.wizards, userID)
		h.wizardMu.Unlock()
		return true, h.matrix.SendMessage(ctx, roomID, "❌ Agent creation cancelled.")
	}

	// Create the agent
	req := &RPCRequest{
		Params: mustMarshal(CreateAgentParams{
			Name:         wizard.Name,
			Skills:       wizard.Skills,
			PIIAccess:    wizard.PIIAccess,
			ResourceTier: wizard.Tier,
		}),
		UserID: userID,
	}

	// Use the RPC handler
	handler := NewRPCHandler(RPCHandlerConfig{Store: h.store, Factory: h.factory})
	resp := handler.Handle(req)

	h.wizardMu.Lock()
	delete(h.wizards, userID)
	h.wizardMu.Unlock()

	if resp.Error != nil {
		return true, h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ Failed to create agent: %s", resp.Error.Message))
	}

	result := resp.Result.(map[string]interface{})
	agent := result["agent"].(*AgentDefinition)

	return true, h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("✅ Agent \"%s\" created!\n\nID: `%s`\nSpawn with: `!agent spawn %s`",
		agent.Name, agent.ID, agent.ID))
}

//=============================================================================
// List Agents Command
//=============================================================================

func (h *CommandHandler) handleListAgents(ctx context.Context, roomID string, cmd *Command) error {
	activeOnly := cmd.KVArgs["active"] != "false"

	definitions, err := h.store.ListDefinitions(activeOnly)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Failed to list agents: "+err.Error())
	}

	if len(definitions) == 0 {
		return h.matrix.SendMessage(ctx, roomID, "No agents defined. Create one with `!agent create name=\"...\"`")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🤖 Agent Definitions (%d)\n\n", len(definitions)))

	for _, def := range definitions {
		status := "✅"
		if !def.IsActive {
			status = "⏸️"
		}
		sb.WriteString(fmt.Sprintf("%s **%s** `%s`\n", status, def.Name, def.ID))
		sb.WriteString(fmt.Sprintf("   Skills: %s | Tier: %s\n\n",
			strings.Join(def.Skills, ", "), def.ResourceTier))
	}

	return h.matrix.SendMessage(ctx, roomID, sb.String())
}

//=============================================================================
// Show Agent Command
//=============================================================================

func (h *CommandHandler) handleShowAgent(ctx context.Context, roomID string, cmd *Command) error {
	if len(cmd.Args) < 2 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: `!agent show <agent-id>`")
	}

	agentID := cmd.Args[1]
	def, err := h.store.GetDefinition(agentID)
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Agent not found: "+agentID)
	}

	status := "Active"
	if !def.IsActive {
		status = "Inactive"
	}

	msg := fmt.Sprintf(`🤖 Agent: %s

ID: %s
Status: %s
Created: %s
By: %s

Skills: %s
PII Access: %s
Resources: %s

Spawn: !agent spawn %s`,
		def.Name,
		def.ID,
		status,
		def.CreatedAt.Format("2006-01-02"),
		def.CreatedBy,
		strings.Join(def.Skills, ", "),
		strings.Join(def.PIIAccess, ", "),
		def.ResourceTier,
		def.ID,
	)

	return h.matrix.SendMessage(ctx, roomID, msg)
}

//=============================================================================
// Spawn Agent Command
//=============================================================================

func (h *CommandHandler) handleSpawnAgent(ctx context.Context, roomID, userID string, cmd *Command) error {
	if len(cmd.Args) < 2 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: `!agent spawn <agent-id>`")
	}

	agentID := cmd.Args[1]

	req := &RPCRequest{
		Params: mustMarshal(SpawnAgentParams{
			ID: agentID,
		}),
		UserID: userID,
	}

	handler := NewRPCHandler(RPCHandlerConfig{Store: h.store, Factory: h.factory})
	resp := handler.Handle(req)

	if resp.Error != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ %s", resp.Error.Message))
	}

	result := resp.Result.(map[string]interface{})

	// Check if approval required
	if status, ok := result["status"].(string); ok && status == "approval_required" {
		fields := result["requires_approval"].([]string)
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf(
			"🔒 This agent requires approval for sensitive PII: %s\n\nUse the PII approval workflow to grant access.",
			strings.Join(fields, ", ")))
	}

	instance := result["instance"].(*AgentInstance)

	return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf(
		"✅ Agent spawned!\n\nInstance: `%s`\nStatus: %s\n\nStop with: `!agent stop %s`",
		instance.ID, instance.Status, instance.ID))
}

//=============================================================================
// Stop Instance Command
//=============================================================================

func (h *CommandHandler) handleStopInstance(ctx context.Context, roomID, userID string, cmd *Command) error {
	if len(cmd.Args) < 2 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: `!agent stop <instance-id>`")
	}

	instanceID := cmd.Args[1]

	req := &RPCRequest{
		Params: mustMarshal(StopInstanceParams{
			ID: instanceID,
		}),
		UserID: userID,
	}

	handler := NewRPCHandler(RPCHandlerConfig{Store: h.store, Factory: h.factory})
	resp := handler.Handle(req)

	if resp.Error != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ %s", resp.Error.Message))
	}

	return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("✅ Instance `%s` stopped.", instanceID))
}

//=============================================================================
// Delete Agent Command
//=============================================================================

func (h *CommandHandler) handleDeleteAgent(ctx context.Context, roomID, userID string, cmd *Command) error {
	if len(cmd.Args) < 2 {
		return h.matrix.SendMessage(ctx, roomID, "❌ Usage: `!agent delete <agent-id>`")
	}

	agentID := cmd.Args[1]

	req := &RPCRequest{
		Params: mustMarshal(DeleteAgentParams{
			ID: agentID,
		}),
		UserID: userID,
	}

	handler := NewRPCHandler(RPCHandlerConfig{Store: h.store, Factory: h.factory})
	resp := handler.Handle(req)

	if resp.Error != nil {
		return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("❌ %s", resp.Error.Message))
	}

	return h.matrix.SendMessage(ctx, roomID, fmt.Sprintf("✅ Agent `%s` deleted.", agentID))
}

//=============================================================================
// Stats Command
//=============================================================================

func (h *CommandHandler) handleStats(ctx context.Context, roomID string) error {
	stats, err := h.store.GetStats()
	if err != nil {
		return h.matrix.SendMessage(ctx, roomID, "❌ Failed to get stats: "+err.Error())
	}

	msg := fmt.Sprintf(`📊 Agent Studio Statistics

Agents: %d (%d active)
Instances: %d (%d running)
Skills: %d available
PII Fields: %d available

By Tier:
  Low: %d
  Medium: %d
  High: %d`,
		stats.TotalDefinitions, stats.ActiveDefinitions,
		stats.TotalInstances, stats.RunningInstances,
		stats.SkillsAvailable,
		stats.PIIFieldsAvailable,
		stats.ByTier["low"],
		stats.ByTier["medium"],
		stats.ByTier["high"],
	)

	return h.matrix.SendMessage(ctx, roomID, msg)
}

//=============================================================================
// Helper Functions
//=============================================================================

func parseNumbers(input string) ([]int, error) {
	var numbers []int
	parts := strings.Split(input, ",")

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, n)
	}

	return numbers, nil
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

// CleanExpiredWizards removes expired wizard sessions
func (h *CommandHandler) CleanExpiredWizards() int {
	h.wizardMu.Lock()
	defer h.wizardMu.Unlock()

	removed := 0
	for userID, wizard := range h.wizards {
		if wizard.IsExpired() {
			delete(h.wizards, userID)
			removed++
		}
	}
	return removed
}

// isValidAgentID checks if an ID matches expected format
var agentIDRegex = regexp.MustCompile(`^agent_\d+_[a-z0-9]+$`)

func isValidAgentID(id string) bool {
	return agentIDRegex.MatchString(id)
}
