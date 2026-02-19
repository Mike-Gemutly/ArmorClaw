// Package matrixcmd provides Matrix command handlers for ArmorClaw.
// These commands can be used through Element X or other Matrix clients.
package matrixcmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// CommandHandler handles Matrix commands
type CommandHandler struct {
	mu sync.RWMutex

	// Lockdown state checker
	getLockdownState func() (LockdownState, error)

	// Claim manager interface
	claimManager ClaimManager

	// Response sender
	sendMessage func(roomID, userID, message string) error

	// Command patterns
	commands map[string]*command
}

// LockdownState represents the lockdown state interface
type LockdownState struct {
	Mode        string
	AdminBound  bool
	AdminUserID string
}

// ClaimManager represents the claim manager interface
type ClaimManager interface {
	InitiateClaim(matrixUserID, deviceID, userAgent, ipAddress, deviceName string) (interface{}, error)
	RespondChallenge(token, response string) error
	GetClaimByToken(token string) (interface{}, error)
	ApproveClaim(claimID, approvedBy string) error
	RejectClaim(claimID, rejectedBy, reason string) error
	ClaimsStats() map[string]interface{}
}

type command struct {
	name        string
	pattern     *regexp.Regexp
	description string
	handler     func(ctx context.Context, roomID, userID string, args []string) (string, error)
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(
	getLockdownState func() (LockdownState, error),
	claimManager ClaimManager,
	sendMessage func(roomID, userID, message string) error,
) *CommandHandler {
	h := &CommandHandler{
		getLockdownState: getLockdownState,
		claimManager:     claimManager,
		sendMessage:      sendMessage,
		commands:         make(map[string]*command),
	}

	// Register commands
	h.registerCommand("claim_admin", `^/claim_admin\s*(.*)$`,
		"Claim admin rights for this ArmorClaw instance",
		h.handleClaimAdmin)

	h.registerCommand("status", `^/status$`,
		"Show ArmorClaw status",
		h.handleStatus)

	h.registerCommand("verify", `^/verify\s+(\S+)$`,
		"Verify a device or claim",
		h.handleVerify)

	h.registerCommand("approve", `^/approve\s+(\S+)$`,
		"Approve a pending claim or device",
		h.handleApprove)

	h.registerCommand("reject", `^/reject\s+(\S+)(?:\s+(.+))?$`,
		"Reject a pending claim or device",
		h.handleReject)

	h.registerCommand("help", `^/help$`,
		"Show available commands",
		h.handleHelp)

	return h
}

func (h *CommandHandler) registerCommand(name, pattern, description string,
	handler func(ctx context.Context, roomID, userID string, args []string) (string, error)) {
	h.commands[name] = &command{
		name:        name,
		pattern:     regexp.MustCompile(pattern),
		description: description,
		handler:     handler,
	}
}

// HandleMessage processes a Matrix message for commands
func (h *CommandHandler) HandleMessage(ctx context.Context, roomID, userID, message string) (bool, error) {
	message = strings.TrimSpace(message)

	// Check each command pattern
	for _, cmd := range h.commands {
		if matches := cmd.pattern.FindStringSubmatch(message); matches != nil {
			args := matches[1:]
			response, err := cmd.handler(ctx, roomID, userID, args)
			if err != nil {
				response = fmt.Sprintf("Error: %s", err.Error())
			}

			if response != "" && h.sendMessage != nil {
				if err := h.sendMessage(roomID, userID, response); err != nil {
					return true, fmt.Errorf("failed to send response: %w", err)
				}
			}

			return true, nil
		}
	}

	return false, nil
}

// handleClaimAdmin handles the /claim_admin command
func (h *CommandHandler) handleClaimAdmin(ctx context.Context, roomID, userID string, args []string) (string, error) {
	// Check current lockdown state
	state, err := h.getLockdownState()
	if err != nil {
		return "", fmt.Errorf("failed to get state: %w", err)
	}

	// Only allow claiming in lockdown or bonding mode
	if state.Mode != "lockdown" && state.Mode != "bonding" {
		return "", fmt.Errorf("admin claiming is only allowed during initial setup")
	}

	// Check if admin is already bonded
	if state.AdminBound {
		return "An admin has already been bonded to this instance. Use /help for available commands.", nil
	}

	// Extract device info from args (if provided)
	deviceName := "Unknown Device"
	if len(args) > 0 && args[0] != "" {
		deviceName = strings.TrimSpace(args[0])
	}

	// Initiate claim
	claim, err := h.claimManager.InitiateClaim(userID, "element-x", "Element X", "matrix", deviceName)
	if err != nil {
		return "", err
	}

	// Extract challenge and ID from claim
	claimMap, ok := claim.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid claim response")
	}

	challenge, _ := claimMap["challenge"].(string)
	claimID, _ := claimMap["id"].(string)

	// Return the challenge code
	return fmt.Sprintf(
		"🔐 **Admin Claim Initiated**\n\n"+
			"To complete admin claiming, verify this code on the host terminal:\n\n"+
			"```\n%s\n```\n\n"+
			"This code expires in 10 minutes.\n"+
			"Claim ID: `%s`",
		challenge,
		claimID,
	), nil
}

// handleStatus handles the /status command
func (h *CommandHandler) handleStatus(ctx context.Context, roomID, userID string, args []string) (string, error) {
	state, err := h.getLockdownState()
	if err != nil {
		return "", err
	}

	var statusEmoji string
	switch state.Mode {
	case "lockdown":
		statusEmoji = "🔒"
	case "bonding":
		statusEmoji = "🔗"
	case "configuring":
		statusEmoji = "⚙️"
	case "hardening":
		statusEmoji = "🛡️"
	case "operational":
		statusEmoji = "✅"
	}

	var adminStatus string
	if state.AdminBound {
		adminStatus = fmt.Sprintf("Admin: `%s`", state.AdminUserID)
	} else {
		adminStatus = "Admin: Not claimed"
	}

	stats := h.claimManager.ClaimsStats()
	statusStats, _ := stats["status"].(map[string]int)

	return fmt.Sprintf(
		"%s **ArmorClaw Status**\n\n"+
			"Mode: `%s`\n"+
			"%s\n\n"+
			"Claims: %d pending, %d approved, %d rejected",
		statusEmoji,
		state.Mode,
		adminStatus,
		statusStats["pending"],
		statusStats["approved"],
		statusStats["rejected"],
	), nil
}

// handleVerify handles the /verify command
func (h *CommandHandler) handleVerify(ctx context.Context, roomID, userID string, args []string) (string, error) {
	code := args[0]

	// Try to verify as claim challenge response
	err := h.claimManager.RespondChallenge(code, code)
	if err == nil {
		return "✅ Challenge verified successfully. Waiting for terminal confirmation.", nil
	}

	// Check if it's a claim token
	claim, err := h.claimManager.GetClaimByToken(code)
	if err == nil {
		claimMap, ok := claim.(map[string]interface{})
		if ok {
			return fmt.Sprintf(
				"📋 **Claim Found**\n\n"+
					"Status: `%s`\n"+
					"User: `%s`\n"+
					"Created: %v",
				claimMap["status"],
				claimMap["matrix_user_id"],
				claimMap["created_at"],
			), nil
		}
	}

	return "", fmt.Errorf("invalid verification code")
}

// handleApprove handles the /approve command
func (h *CommandHandler) handleApprove(ctx context.Context, roomID, userID string, args []string) (string, error) {
	claimID := args[0]

	// Check if user is admin
	state, err := h.getLockdownState()
	if err != nil {
		return "", err
	}

	if !state.AdminBound || state.AdminUserID != userID {
		return "", fmt.Errorf("only the admin can approve claims")
	}

	err = h.claimManager.ApproveClaim(claimID, userID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("✅ Claim `%s` approved successfully.", claimID), nil
}

// handleReject handles the /reject command
func (h *CommandHandler) handleReject(ctx context.Context, roomID, userID string, args []string) (string, error) {
	claimID := args[0]
	reason := "No reason provided"
	if len(args) > 1 {
		reason = args[1]
	}

	// Check if user is admin
	state, err := h.getLockdownState()
	if err != nil {
		return "", err
	}

	if !state.AdminBound || state.AdminUserID != userID {
		return "", fmt.Errorf("only the admin can reject claims")
	}

	err = h.claimManager.RejectClaim(claimID, userID, reason)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("❌ Claim `%s` rejected: %s", claimID, reason), nil
}

// handleHelp handles the /help command
func (h *CommandHandler) handleHelp(ctx context.Context, roomID, userID string, args []string) (string, error) {
	var help strings.Builder
	help.WriteString("📖 **ArmorClaw Commands**\n\n")

	for name, cmd := range h.commands {
		help.WriteString(fmt.Sprintf("- `/%s` - %s\n", name, cmd.description))
	}

	help.WriteString("\n💡 Use `/command help` for detailed usage.")

	return help.String(), nil
}

// GetCommands returns available commands
func (h *CommandHandler) GetCommands() map[string]string {
	commands := make(map[string]string)
	for name, cmd := range h.commands {
		commands[name] = cmd.description
	}
	return commands
}
