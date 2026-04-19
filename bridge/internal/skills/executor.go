package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/armorclaw/bridge/pkg/governor"
	"github.com/armorclaw/bridge/pkg/interfaces"
)

// CommandContext manages execution context for skills
type CommandContext struct {
	Timeout    time.Duration
	Args       []string
	Env        map[string]string
	WorkingDir string
	User       string
}

// SkillResult represents the result of skill execution
type SkillResult struct {
	Success   bool                   `json:"success"`
	Output    interface{}            `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Type      string                 `json:"type"` // "data", "error", "timeout"
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SkillExecutor handles skill execution with full PETG security pipeline
type SkillExecutor struct {
	registry       *Registry
	router         *Router
	ssrfValidator  *SSRFValidator
	allowlist      *AllowlistManager
	policyEnforcer *PolicyEnforcer
	skillGate      interfaces.SkillGate
	authorizer     interfaces.Authorizer
}

type SkillExecutorConfig struct {
	SkillGate  interfaces.SkillGate
	Authorizer interfaces.Authorizer
}

func NewSkillExecutor() *SkillExecutor {
	return NewSkillExecutorWithConfig(SkillExecutorConfig{})
}

func NewSkillExecutorWithConfig(cfg SkillExecutorConfig) *SkillExecutor {
	if cfg.SkillGate == nil {
		cfg.SkillGate = governor.NewGovernor(nil, nil)
	}
	return &SkillExecutor{
		registry:       NewRegistry(),
		router:         NewRouter(),
		ssrfValidator:  NewSSRFValidator(),
		allowlist:      NewAllowlistManager(),
		policyEnforcer: NewPolicyEnforcer(),
		skillGate:      cfg.SkillGate,
		authorizer:     cfg.Authorizer,
	}
}

// LoadSkills loads skills from a directory into the executor's registry
func (se *SkillExecutor) LoadSkills(skillsDir string) error {
	return se.registry.ScanSkills(skillsDir)
}

// ExecuteSkill executes a skill with full PETG validation
func (se *SkillExecutor) ExecuteSkill(ctx context.Context, skillName string, params map[string]interface{}) (*SkillResult, error) {
	startTime := time.Now()

	skill, exists := se.registry.GetSkill(skillName)
	if !exists {
		return &SkillResult{
			Success:   false,
			Error:     fmt.Sprintf("Skill not found: %s", skillName),
			Type:      "error",
			Timestamp: startTime,
		}, fmt.Errorf("skill not found: %s", skillName)
	}

	if se.authorizer != nil {
		if err := se.authorizer.AuthorizeAction(ctx, "", skillName, params); err != nil {
			return &SkillResult{
				Success:   false,
				Error:     err.Error(),
				Type:      "error",
				Timestamp: startTime,
			}, err
		}
	}

	call := &interfaces.ToolCall{ToolName: skillName, Arguments: params}
	_, err := se.skillGate.InterceptToolCall(ctx, call)
	if err != nil {
		return &SkillResult{
			Success:   false,
			Error:     fmt.Sprintf("PII interception failed: %s", err.Error()),
			Type:      "error",
			Timestamp: startTime,
		}, err
	}

	if !se.policyEnforcer.IsAllowed(skillName) {
		return &SkillResult{
			Success:   false,
			Error:     fmt.Sprintf("Skill '%s' is not allowed by policy", skillName),
			Type:      "error",
			Timestamp: startTime,
		}, fmt.Errorf("policy violation: skill %s not allowed", skillName)
	}

	// Step 3: Pre-execution security checks
	if err := se.preExecutionChecks(skill, params); err != nil {
		return &SkillResult{
			Success:   false,
			Error:     fmt.Sprintf("Security check failed: %s", err.Error()),
			Type:      "error",
			Timestamp: startTime,
		}, err
	}

	// Step 4: Get appropriate handler for skill's domain
	handler := se.getHandlerForDomain(skill.Domain)

	// Step 5: Execute skill with timeout
	executionCtx, cancel := context.WithTimeout(ctx, skill.Timeout)
	defer cancel()

	result, err := handler(executionCtx, params)
	if err != nil {
		errorType := "error"
		if ctx.Err() == context.DeadlineExceeded {
			errorType = "timeout"
		}

		return &SkillResult{
			Success:   false,
			Error:     err.Error(),
			Type:      errorType,
			Timestamp: startTime,
		}, err
	}

	// Step 7: Post-execution filtering
	if err := se.postExecutionFiltering(skill, result); err != nil {
		return &SkillResult{
			Success:   false,
			Error:     fmt.Sprintf("Result filtering failed: %s", err.Error()),
			Type:      "error",
			Timestamp: startTime,
		}, err
	}

	// Step 8: Return successful result
	return &SkillResult{
		Success:   true,
		Output:    result,
		Type:      "data",
		Timestamp: startTime,
		Metadata: map[string]interface{}{
			"skill":    skillName,
			"version":  skill.Version,
			"duration": time.Since(startTime).String(),
			"timeout":  skill.Timeout.String(),
		},
	}, nil
}

// preExecutionChecks performs security checks before execution
func (se *SkillExecutor) preExecutionChecks(skill *Skill, params map[string]interface{}) error {
	// Check for URLs in parameters and validate against SSRF
	urls := se.extractURLsFromParams(params)
	for _, urlStr := range urls {
		if err := se.ssrfValidator.ValidateURL(urlStr); err != nil {
			return fmt.Errorf("URL validation failed: %w", err)
		}
	}

	// Check for dangerous patterns
	if err := se.checkDangerousPatterns(params); err != nil {
		return fmt.Errorf("dangerous pattern detected: %w", err)
	}

	// Skill-specific security checks
	if skill.SecurityCheck != nil {
		if err := skill.SecurityCheck(params); err != nil {
			return fmt.Errorf("skill security check failed: %w", err)
		}
	}

	return nil
}

// postExecutionFiltering applies filters to execution results
func (se *SkillExecutor) postExecutionFiltering(skill *Skill, result interface{}) error {
	// Filter out sensitive information from results
	switch v := result.(type) {
	case map[string]interface{}:
		se.filterSensitiveFields(v)
	case []map[string]interface{}:
		for _, item := range v {
			se.filterSensitiveFields(item)
		}
	}

	// Apply result size limits
	if err := se.checkResultSize(result); err != nil {
		return fmt.Errorf("result size limit exceeded: %w", err)
	}

	return nil
}

// extractURLsFromParams extracts URL strings from parameters
func (se *SkillExecutor) extractURLsFromParams(params map[string]interface{}) []string {
	var urls []string

	for _, value := range params {
		switch v := value.(type) {
		case string:
			if se.isURL(v) {
				urls = append(urls, v)
			}
		case []string:
			for _, s := range v {
				if se.isURL(s) {
					urls = append(urls, s)
				}
			}
		}
	}

	return urls
}

// isURL checks if a string is a URL
func (se *SkillExecutor) isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// checkDangerousPatterns checks for potentially dangerous input patterns
func (se *SkillExecutor) checkDangerousPatterns(params map[string]interface{}) error {
	for _, value := range params {
		if str, ok := value.(string); ok {
			// Check for file:// scheme
			if len(str) > 7 && str[:7] == "file://" {
				return fmt.Errorf("file:// URLs are not allowed")
			}

			// Check for ftp:// scheme
			if len(str) > 6 && str[:6] == "ftp://" {
				return fmt.Errorf("ftp:// URLs are not allowed")
			}

		}
	}
	return nil
}

// contains checks if a string contains a substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr ||
		(len(str) > len(substr) &&
			(str[:len(substr)] == substr ||
				str[len(str)-len(substr):] == substr ||
				findSubstring(str, substr))))
}

// findSubstring helper for contains
func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// filterSensitiveFields removes sensitive fields from result data
func (se *SkillExecutor) filterSensitiveFields(data map[string]interface{}) {
	sensitiveFields := []string{
		"password", "secret", "token", "key", "api_key",
		"private_key", "auth", "credential", "session",
	}

	for _, field := range sensitiveFields {
		if _, exists := data[field]; exists {
			data[field] = "***REDACTED***"
		}
	}
}

// checkResultSize validates result size limits
func (se *SkillExecutor) checkResultSize(result interface{}) error {
	// Convert to JSON to check size
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil // If we can't marshal, we can't check size
	}

	// 10MB limit for results
	const maxSize = 10 * 1024 * 1024
	if len(jsonData) > maxSize {
		return fmt.Errorf("result too large: %d bytes (max %d)", len(jsonData), maxSize)
	}

	return nil
}

// AllowIP adds an IP to the allowlist for admin overrides
func (se *SkillExecutor) AllowIP(ip string) error {
	return se.allowlist.AllowIP(ip)
}

// AllowCIDR adds a CIDR range to the allowlist
func (se *SkillExecutor) AllowCIDR(cidr string) error {
	return se.allowlist.AllowCIDR(cidr)
}

// PolicyEnforcer handles skill policy validation
type PolicyEnforcer struct {
	allowedSkills map[string]bool
	blockedSkills map[string]bool
}

// NewPolicyEnforcer creates a new policy enforcer
func NewPolicyEnforcer() *PolicyEnforcer {
	pe := &PolicyEnforcer{
		allowedSkills: make(map[string]bool),
		blockedSkills: make(map[string]bool),
	}
	// Auto-allow skills that have policy entries
	for toolName := range Policy {
		pe.allowedSkills[toolName] = true
	}
	return pe
}

// IsAllowed checks if a skill is allowed by policy
func (pe *PolicyEnforcer) IsAllowed(skillName string) bool {
	// If explicitly blocked, deny
	if pe.blockedSkills[skillName] {
		return false
	}

	// If explicitly allowed, allow
	if pe.allowedSkills[skillName] {
		return true
	}

	// Deny by default - only explicitly allowed skills pass
	return false
}

// AllowSkill adds a skill to the allowed list
func (pe *PolicyEnforcer) AllowSkill(skillName string) {
	pe.allowedSkills[skillName] = true
	delete(pe.blockedSkills, skillName)
}

// BlockSkill adds a skill to the blocked list
func (pe *PolicyEnforcer) BlockSkill(skillName string) {
	pe.blockedSkills[skillName] = true
	delete(pe.allowedSkills, skillName)
}

// HandlerFunc represents a skill handler function
type HandlerFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// getHandlerForDomain returns the appropriate handler for a domain
func (se *SkillExecutor) getHandlerForDomain(domain string) HandlerFunc {
	switch domain {
	case "weather":
		return se.handleWeatherSkill
	case "github":
		return se.handleGithubSkill
	case "web":
		return se.handleWebSkill
	case "search":
		return se.handleWebSearchSkill
	case "extract":
		return se.handleWebExtractSkill
	case "email":
		return se.handleEmailSkill
	case "slack":
		return se.handleSlackSkill
	case "webdav":
		return se.handleWebDAVSkill
	default:
		return se.handleGeneralSkill
	}
}

// handleWeatherSkill handles weather-related skills
func (se *SkillExecutor) handleWeatherSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// For Phase 1, return mock weather data
	return map[string]interface{}{
		"temperature": 22.5,
		"condition":   "Partly Cloudy",
		"humidity":    65,
		"location":    "Unknown",
	}, nil
}

// handleGithubSkill handles GitHub-related skills
func (se *SkillExecutor) handleGithubSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// For Phase 1, return mock GitHub data
	return map[string]interface{}{
		"repository": "test/repo",
		"stars":      42,
		"language":   "Go",
		"updated_at": "2024-01-01T00:00:00Z",
	}, nil
}

// handleWebSkill handles web-related skills
func (se *SkillExecutor) handleWebSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// For Phase 1, return mock web data
	return map[string]interface{}{
		"title":       "Example Page",
		"url":         "https://example.com",
		"status_code": 200,
		"content":     "Mock content for testing",
	}, nil
}

// handleWebSearchSkill handles web search skills
func (se *SkillExecutor) handleWebSearchSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Validate parameters
	if err := ValidateWebSearchParams(params); err != nil {
		return nil, fmt.Errorf("web search validation failed: %w", err)
	}

	// Execute web search
	return ExecuteWebSearch(ctx, params)
}

// handleWebExtractSkill handles web extraction skills
func (se *SkillExecutor) handleWebExtractSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Validate parameters
	if err := ValidateWebExtractParams(params); err != nil {
		return nil, fmt.Errorf("web extract validation failed: %w", err)
	}

	// Execute web extraction
	return ExecuteWebExtract(ctx, params)
}

// handleEmailSkill handles email-related skills
func (se *SkillExecutor) handleEmailSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Validate parameters
	if err := ValidateEmailSendParams(params); err != nil {
		return nil, fmt.Errorf("email validation failed: %w", err)
	}

	// Execute email sending
	return ExecuteEmailSend(ctx, params)
}

// handleSlackSkill handles Slack-related skills
func (se *SkillExecutor) handleSlackSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Validate parameters
	if err := ValidateSlackMessageParams(params); err != nil {
		return nil, fmt.Errorf("slack message validation failed: %w", err)
	}

	// Execute Slack message sending
	return ExecuteSlackMessage(ctx, params)
}

// handleWebDAVSkill handles WebDAV operations
func (se *SkillExecutor) handleWebDAVSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return ExecuteWebDAV(ctx, params)
}

// handleGeneralSkill handles general/miscellaneous skills
func (se *SkillExecutor) handleGeneralSkill(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// For Phase 1, return general response
	return map[string]interface{}{
		"message": "General skill executed",
		"params":  params,
	}, nil
}

// Wrapper methods for RPC integration

// ListEnabled returns all enabled skills
func (se *SkillExecutor) ListEnabled() []*Skill {
	return se.registry.GetEnabled()
}

// GetSkill returns a specific skill by name
func (se *SkillExecutor) GetSkill(name string) (*Skill, bool) {
	return se.registry.GetSkill(name)
}

// AllowSkill adds a skill to the allowed list
func (se *SkillExecutor) AllowSkill(skillName string) error {
	se.policyEnforcer.AllowSkill(skillName)
	return nil
}

// BlockSkill adds a skill to the blocked list
func (se *SkillExecutor) BlockSkill(skillName string) error {
	se.policyEnforcer.BlockSkill(skillName)
	return nil
}

// GetAllowlist returns the current allowlist
func (se *SkillExecutor) GetAllowlist() ([]string, []string) {
	return se.allowlist.ListAllowed()
}

// GenerateSchema generates an OpenAI-compatible schema for a skill
func (se *SkillExecutor) GenerateSchema(skill *Skill) interface{} {
	// For Phase 1, return a simple schema
	// In production, this would use the schema.go generator
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"skill_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the skill to execute",
			},
			"params": map[string]interface{}{
				"type":        "object",
				"description": "Parameters for the skill",
				"properties":  se.getParameterSchemas(skill.Parameters),
			},
		},
		"required": []string{"skill_name"},
	}
}

// getParameterSchemas generates JSON schemas for skill parameters
func (se *SkillExecutor) getParameterSchemas(params map[string]Param) map[string]interface{} {
	schemas := make(map[string]interface{})

	for name, param := range params {
		schemas[name] = map[string]interface{}{
			"type":        se.mapTypeToJSONType(param.Type),
			"description": param.Description,
		}
		if param.Required {
			// This would be handled in the main schema's required array
		}
	}

	return schemas
}

// mapTypeToJSONType maps skill parameter types to JSON schema types
func (se *SkillExecutor) mapTypeToJSONType(skillType string) string {
	switch skillType {
	case "string":
		return "string"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "array"
	case "object":
		return "object"
	default:
		return "string" // Default to string for unknown types
	}
}
