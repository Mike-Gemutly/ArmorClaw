package studio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

//=============================================================================
// MCP Approval Workflow - Server-Side Implementation
//=============================================================================

// ApprovalStatus represents the state of an approval request
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "PENDING"
	ApprovalApproved ApprovalStatus = "APPROVED"
	ApprovalRejected ApprovalStatus = "REJECTED"
	ApprovalExpired  ApprovalStatus = "EXPIRED"
)

// ApprovalType categorizes different approval requests
type ApprovalType string

const (
	ApprovalTypeMCPAccess ApprovalType = "MCP_ACCESS"
	ApprovalTypePIIAccess ApprovalType = "PII_ACCESS"
	ApprovalTypeSkillAdd  ApprovalType = "SKILL_ADD"
)

// McpApprovalRequest represents a user request for MCP access
type McpApprovalRequest struct {
	ID          string         `json:"id"`
	Type        ApprovalType   `json:"type"`
	MCPId       string         `json:"mcp_id"`
	MCPName     string         `json:"mcp_name"`
	AgentName   string         `json:"agent_name"`
	Reason      string         `json:"reason,omitempty"`
	RequestedBy string         `json:"requested_by"`
	Status      ApprovalStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	ExpiresAt   time.Time      `json:"expires_at"`

	// Resolution fields (filled when approved/rejected)
	ReviewedBy  *string    `json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes string     `json:"review_notes,omitempty"`
}

// McpDefinition represents an MCP server definition
type McpDefinition struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Endpoint    string            `json:"endpoint"`
	Category    string            `json:"category"`
	RiskLevel   McpRiskLevel      `json:"risk_level"`
	RequiresPII []string          `json:"requires_pii,omitempty"`
	Capabilities []string         `json:"capabilities,omitempty"`
	IsActive    bool              `json:"is_active"`
}

// McpRiskLevel categorizes MCP security risk
type McpRiskLevel string

const (
	McpRiskLow      McpRiskLevel = "low"
	McpRiskMedium   McpRiskLevel = "medium"
	McpRiskHigh     McpRiskLevel = "high"
	McpRiskCritical McpRiskLevel = "critical"
)

// UserRole for permission checks
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
	RoleViewer UserRole = "viewer"
)

//=============================================================================
// MCP Registry
//=============================================================================

// McpRegistry manages available MCP servers
type McpRegistry interface {
	// GetMcp retrieves an MCP definition by ID
	GetMcp(id string) (*McpDefinition, error)

	// ListMcps returns all available MCPs
	ListMcps(filter *McpFilter) ([]*McpDefinition, error)

	// IsValid checks if an MCP ID exists
	IsValid(id string) bool

	// RegisterMcp adds a new MCP to the registry
	RegisterMcp(mcp *McpDefinition) error
}

// McpFilter for querying MCPs
type McpFilter struct {
	Category  string        `json:"category,omitempty"`
	RiskLevel McpRiskLevel  `json:"risk_level,omitempty"`
	ActiveOnly bool         `json:"active_only,omitempty"`
}

// DefaultMcpRegistry is an in-memory MCP registry
type DefaultMcpRegistry struct {
	mcps map[string]*McpDefinition
	mu   interface{} // sync.RWMutex in production
}

// NewMcpRegistry creates a new MCP registry with defaults
func NewMcpRegistry() *DefaultMcpRegistry {
	return &DefaultMcpRegistry{
		mcps: map[string]*McpDefinition{
			"filesystem": {
				ID:           "filesystem",
				Name:         "File System Access",
				Description:  "Read and write files on the host system",
				Endpoint:     "local://filesystem",
				Category:     "utility",
				RiskLevel:    McpRiskHigh,
				Capabilities: []string{"read", "write", "delete"},
				IsActive:     true,
			},
			"web-search": {
				ID:           "web-search",
				Name:         "Web Search",
				Description:  "Search the web for information",
				Endpoint:     "https://api.search.example.com/mcp",
				Category:     "research",
				RiskLevel:    McpRiskLow,
				Capabilities: []string{"search", "fetch"},
				IsActive:     true,
			},
			"database": {
				ID:           "database",
				Name:         "Database Connector",
				Description:  "Connect to SQL databases",
				Endpoint:     "local://database",
				Category:     "data",
				RiskLevel:    McpRiskCritical,
				RequiresPII:  []string{"database_credentials"},
				Capabilities: []string{"query", "insert", "update", "delete"},
				IsActive:     true,
			},
			"github": {
				ID:           "github",
				Name:         "GitHub Integration",
				Description:  "Access GitHub repositories and issues",
				Endpoint:     "https://api.github.com/mcp",
				Category:     "development",
				RiskLevel:    McpRiskMedium,
				RequiresPII:  []string{"github_token"},
				Capabilities: []string{"repo:read", "repo:write", "issues", "pr"},
				IsActive:     true,
			},
			"slack": {
				ID:           "slack",
				Name:         "Slack Integration",
				Description:  "Send and receive Slack messages",
				Endpoint:     "https://slack.com/api/mcp",
				Category:     "communication",
				RiskLevel:    McpRiskMedium,
				RequiresPII:  []string{"slack_token"},
				Capabilities: []string{"messages:read", "messages:write", "channels:read"},
				IsActive:     true,
			},
		},
	}
}

func (r *DefaultMcpRegistry) GetMcp(id string) (*McpDefinition, error) {
	if mcp, ok := r.mcps[id]; ok {
		return mcp, nil
	}
	return nil, fmt.Errorf("MCP not found: %s", id)
}

func (r *DefaultMcpRegistry) ListMcps(filter *McpFilter) ([]*McpDefinition, error) {
	var result []*McpDefinition
	for _, mcp := range r.mcps {
		if filter != nil {
			if filter.Category != "" && mcp.Category != filter.Category {
				continue
			}
			if filter.RiskLevel != "" && mcp.RiskLevel != filter.RiskLevel {
				continue
			}
			if filter.ActiveOnly && !mcp.IsActive {
				continue
			}
		}
		result = append(result, mcp)
	}
	return result, nil
}

func (r *DefaultMcpRegistry) IsValid(id string) bool {
	_, ok := r.mcps[id]
	return ok
}

func (r *DefaultMcpRegistry) RegisterMcp(mcp *McpDefinition) error {
	if mcp.ID == "" {
		return fmt.Errorf("MCP ID is required")
	}
	r.mcps[mcp.ID] = mcp
	return nil
}

//=============================================================================
// Approval Manager
//=============================================================================

// ApprovalManager handles approval workflow logic
type ApprovalManager struct {
	store   Store
	registry McpRegistry
	notifier AdminNotifier
}

// ApprovalManagerConfig for creating approval manager
type ApprovalManagerConfig struct {
	Store    Store
	Registry McpRegistry
	Notifier AdminNotifier
}

// NewApprovalManager creates a new approval manager
func NewApprovalManager(cfg ApprovalManagerConfig) *ApprovalManager {
	if cfg.Registry == nil {
		cfg.Registry = NewMcpRegistry()
	}
	return &ApprovalManager{
		store:    cfg.Store,
		registry: cfg.Registry,
		notifier: cfg.Notifier,
	}
}

// CreateApprovalRequest creates a new approval request from a non-admin user
func (m *ApprovalManager) CreateApprovalRequest(ctx context.Context, req *McpApprovalRequest) (*McpApprovalRequest, error) {
	// Validate MCP exists
	mcp, err := m.registry.GetMcp(req.MCPId)
	if err != nil {
		return nil, fmt.Errorf("invalid MCP: %w", err)
	}

	// Set MCP name from registry
	req.MCPName = mcp.Name

	// Validate request
	if req.RequestedBy == "" {
		return nil, fmt.Errorf("requested_by is required")
	}
	if req.AgentName == "" {
		return nil, fmt.Errorf("agent_name is required")
	}

	// Set defaults
	if req.ID == "" {
		req.ID = generateID("approval")
	}
	if req.Type == "" {
		req.Type = ApprovalTypeMCPAccess
	}
	req.Status = ApprovalPending
	req.CreatedAt = time.Now()
	req.ExpiresAt = req.CreatedAt.Add(7 * 24 * time.Hour) // 7 day expiry

	// Save to store
	if err := m.store.SaveApprovalRequest(req); err != nil {
		return nil, fmt.Errorf("failed to save approval request: %w", err)
	}

	// Notify admins
	if m.notifier != nil {
		notification := &AdminNotification{
			Type:        "APPROVAL_REQUEST",
			Title:       fmt.Sprintf("MCP Access Request: %s", mcp.Name),
			Message:     fmt.Sprintf("User %s requests access to %s for agent '%s'", req.RequestedBy, mcp.Name, req.AgentName),
			ApprovalID:  req.ID,
			RiskLevel:   string(mcp.RiskLevel),
			CreatedAt:   req.CreatedAt,
		}
		if req.Reason != "" {
			notification.Message += fmt.Sprintf("\n\nReason: %s", req.Reason)
		}
		go m.notifier.NotifyAdmins(ctx, notification)
	}

	return req, nil
}

// ApproveRequest approves a pending approval request
func (m *ApprovalManager) ApproveRequest(ctx context.Context, approvalID, reviewedBy, notes string) error {
	req, err := m.store.GetApprovalRequest(approvalID)
	if err != nil {
		return fmt.Errorf("approval request not found: %w", err)
	}

	if req.Status != ApprovalPending {
		return fmt.Errorf("approval request is not pending (status: %s)", req.Status)
	}

	// Check expiry
	if time.Now().After(req.ExpiresAt) {
		req.Status = ApprovalExpired
		m.store.SaveApprovalRequest(req)
		return fmt.Errorf("approval request has expired")
	}

	// Update status
	now := time.Now()
	req.Status = ApprovalApproved
	req.ReviewedBy = &reviewedBy
	req.ReviewedAt = &now
	req.ReviewNotes = notes

	if err := m.store.SaveApprovalRequest(req); err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	// Notify requester
	if m.notifier != nil {
		notification := &AdminNotification{
			Type:        "APPROVAL_GRANTED",
			Title:       fmt.Sprintf("MCP Access Approved: %s", req.MCPName),
			Message:     fmt.Sprintf("Your request to use %s for agent '%s' has been approved.", req.MCPName, req.AgentName),
			ApprovalID:  req.ID,
			TargetUser:  req.RequestedBy,
			CreatedAt:   now,
		}
		go m.notifier.NotifyUser(ctx, notification)
	}

	return nil
}

// RejectRequest rejects a pending approval request
func (m *ApprovalManager) RejectRequest(ctx context.Context, approvalID, reviewedBy, notes string) error {
	req, err := m.store.GetApprovalRequest(approvalID)
	if err != nil {
		return fmt.Errorf("approval request not found: %w", err)
	}

	if req.Status != ApprovalPending {
		return fmt.Errorf("approval request is not pending (status: %s)", req.Status)
	}

	// Update status
	now := time.Now()
	req.Status = ApprovalRejected
	req.ReviewedBy = &reviewedBy
	req.ReviewedAt = &now
	req.ReviewNotes = notes

	if err := m.store.SaveApprovalRequest(req); err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	// Notify requester
	if m.notifier != nil {
		message := fmt.Sprintf("Your request to use %s for agent '%s' has been rejected.", req.MCPName, req.AgentName)
		if notes != "" {
			message += fmt.Sprintf("\n\nReason: %s", notes)
		}
		notification := &AdminNotification{
			Type:        "APPROVAL_REJECTED",
			Title:       fmt.Sprintf("MCP Access Rejected: %s", req.MCPName),
			Message:     message,
			ApprovalID:  req.ID,
			TargetUser:  req.RequestedBy,
			CreatedAt:   now,
		}
		go m.notifier.NotifyUser(ctx, notification)
	}

	return nil
}

// ListPendingApprovals returns all pending approval requests
func (m *ApprovalManager) ListPendingApprovals(ctx context.Context) ([]*McpApprovalRequest, error) {
	return m.store.ListApprovalRequests(ApprovalPending)
}

// ListUserApprovals returns approval requests for a specific user
func (m *ApprovalManager) ListUserApprovals(ctx context.Context, userID string) ([]*McpApprovalRequest, error) {
	return m.store.ListUserApprovalRequests(userID)
}

// GetMcpRiskAssessment returns risk information for an MCP
func (m *ApprovalManager) GetMcpRiskAssessment(mcpID string) (*McpRiskAssessment, error) {
	mcp, err := m.registry.GetMcp(mcpID)
	if err != nil {
		return nil, err
	}

	assessment := &McpRiskAssessment{
		MCP:          mcp,
		ExternalData: mcp.Endpoint != "" && !isLocalEndpoint(mcp.Endpoint),
		Warnings:     []string{},
	}

	// Add risk warnings
	if mcp.RiskLevel == McpRiskHigh || mcp.RiskLevel == McpRiskCritical {
		assessment.Warnings = append(assessment.Warnings,
			fmt.Sprintf("This MCP has a %s risk level", mcp.RiskLevel))
	}

	if len(mcp.RequiresPII) > 0 {
		assessment.Warnings = append(assessment.Warnings,
			fmt.Sprintf("This MCP requires access to sensitive data: %v", mcp.RequiresPII))
	}

	if assessment.ExternalData {
		assessment.Warnings = append(assessment.Warnings,
			"ArmorClaw cannot guarantee the security of external servers")
		assessment.Warnings = append(assessment.Warnings,
			"Data sent may include sensitive information")
	}

	return assessment, nil
}

// McpRiskAssessment contains risk analysis for an MCP
type McpRiskAssessment struct {
	MCP          *McpDefinition `json:"mcp"`
	ExternalData bool           `json:"external_data"`
	Warnings     []string       `json:"warnings"`
}

//=============================================================================
// Admin Notifier Interface
//=============================================================================

// AdminNotifier sends notifications to admins and users
type AdminNotifier interface {
	NotifyAdmins(ctx context.Context, notification *AdminNotification) error
	NotifyUser(ctx context.Context, notification *AdminNotification) error
}

// AdminNotification represents a notification payload
type AdminNotification struct {
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	ApprovalID  string    `json:"approval_id,omitempty"`
	TargetUser  string    `json:"target_user,omitempty"`
	RiskLevel   string    `json:"risk_level,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

//=============================================================================
// Audit Logging for MCP Actions
//=============================================================================

// McpAuditLogEntry represents an audit log for MCP operations
type McpAuditLogEntry struct {
	Timestamp   time.Time      `json:"timestamp"`
	Action      string         `json:"action"`
	MCPId       string         `json:"mcp_id"`
	UserID      string         `json:"user_id"`
	UserRole    UserRole       `json:"user_role"`
	AgentName   string         `json:"agent_name,omitempty"`
	Status      string         `json:"status"`
	ApprovalID  string         `json:"approval_id,omitempty"`
	IPAddress   string         `json:"ip_address,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
}

// LogMcpAction creates an audit log entry for MCP operations
func LogMcpAction(action, mcpID, userID string, role UserRole) *McpAuditLogEntry {
	return &McpAuditLogEntry{
		Timestamp: time.Now(),
		Action:    action,
		MCPId:     mcpID,
		UserID:    userID,
		UserRole:  role,
		Status:    "completed",
	}
}

//=============================================================================
// RPC Response Types
//=============================================================================

// McpListResponse for studio.list_mcps RPC
type McpListResponse struct {
	McPs []*McpDefinition `json:"mcps"`
}

// McpApprovalResponse for approval request RPC
type McpApprovalResponse struct {
	ApprovalID string         `json:"approval_id"`
	Status     ApprovalStatus `json:"status"`
	MCPId      string         `json:"mcp_id"`
	MCPName    string         `json:"mcp_name"`
	CreatedAt  time.Time      `json:"created_at"`
	ExpiresAt  time.Time      `json:"expires_at"`
}

// McpWarningResponse for admin warning dialog
type McpWarningResponse struct {
	MCP            *McpDefinition `json:"mcp"`
	RiskAssessment *McpRiskAssessment `json:"risk_assessment"`
	AuditLogged    bool           `json:"audit_logged"`
}

//=============================================================================
// Helper Functions
//=============================================================================

func isLocalEndpoint(endpoint string) bool {
	return len(endpoint) >= 7 && endpoint[:7] == "local://"
}

// ParseMcpApprovalRequest parses JSON into McpApprovalRequest
func ParseMcpApprovalRequest(data json.RawMessage) (*McpApprovalRequest, error) {
	var req McpApprovalRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("invalid approval request: %w", err)
	}
	return &req, nil
}
