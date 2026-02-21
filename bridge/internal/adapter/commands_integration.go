// Package adapter provides Matrix client functionality for ArmorClaw bridge.
// This file integrates command handling for admin operations via Matrix.
package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/armorclaw/bridge/pkg/admin"
	"github.com/armorclaw/bridge/pkg/lockdown"
)

// CommandHandler handles Matrix commands for ArmorClaw admin operations
type CommandHandler struct {
	claimMgr    *admin.ClaimManager
	lockdownMgr *lockdown.Manager
	adapter     *MatrixAdapter
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(
	claimMgr *admin.ClaimManager,
	lockdownMgr *lockdown.Manager,
	adapter *MatrixAdapter,
) *CommandHandler {
	return &CommandHandler{
		claimMgr:    claimMgr,
		lockdownMgr: lockdownMgr,
		adapter:     adapter,
	}
}

// HandleCommand processes a Matrix message as a potential command
// Returns true if the message was a command (handled), false otherwise
func (h *CommandHandler) HandleCommand(ctx context.Context, roomID, userID, message string) bool {
	message = strings.TrimSpace(message)

	// Check if it starts with /
	if !strings.HasPrefix(message, "/") {
		return false
	}

	// Parse command
	parts := strings.Fields(message)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	var response string
	var err error

	switch cmd {
	case "/claim_admin":
		response, err = h.handleClaimAdmin(ctx, userID, args)
	case "/status":
		response, err = h.handleStatus(ctx)
	case "/verify":
		response, err = h.handleVerify(ctx, userID, args)
	case "/approve":
		response, err = h.handleApprove(ctx, userID, args)
	case "/reject":
		response, err = h.handleReject(ctx, userID, args)
	case "/help":
		response = h.handleHelp()
	default:
		// Unknown command, don't consume it
		return false
	}

	// Send response
	if response != "" {
		if err != nil {
			response = fmt.Sprintf("❌ **Error:** %s\n\n%s", err.Error(), response)
		}
		h.adapter.SendMessageWithRetry(roomID, response, "m.notice")
	}

	return true
}

// handleClaimAdmin handles the /claim_admin command
func (h *CommandHandler) handleClaimAdmin(ctx context.Context, userID string, args []string) (string, error) {
	// Check current lockdown state
	state := h.lockdownMgr.GetState()

	// Only allow claiming in lockdown or bonding mode
	if state.Mode != lockdown.ModeLockdown && state.Mode != lockdown.ModeBonding {
		return "", fmt.Errorf("admin claiming is only allowed during initial setup (current mode: %s)", state.Mode)
	}

	// Check if admin is already bonded
	if state.AdminEstablished {
		return "An admin has already been bonded to this instance. Use `/help` for available commands.", nil
	}

	// Extract and validate device name from args
	deviceName := "Element X"
	if len(args) > 0 {
		deviceName = sanitizeDeviceName(strings.Join(args, " "))
	}

	// Initiate claim
	claim, err := h.claimMgr.InitiateClaim(userID, "element-x", "Element X", "matrix", deviceName)
	if err != nil {
		return "", err
	}

	// Return the challenge code
	return fmt.Sprintf(
		"🔐 **Admin Claim Initiated**\n\n"+
			"To complete admin claiming, verify this code on the host terminal:\n\n"+
			"```\n%s\n```\n\n"+
			"This code expires in 10 minutes.\n"+
			"Claim ID: `%s`",
		claim.Challenge,
		claim.ID,
	), nil
}

// handleStatus handles the /status command
func (h *CommandHandler) handleStatus(ctx context.Context) (string, error) {
	state := h.lockdownMgr.GetState()

	var statusEmoji string
	switch state.Mode {
	case lockdown.ModeLockdown:
		statusEmoji = "🔒"
	case lockdown.ModeBonding:
		statusEmoji = "🔗"
	case lockdown.ModeConfiguring:
		statusEmoji = "⚙️"
	case lockdown.ModeHardening:
		statusEmoji = "🛡️"
	case lockdown.ModeOperational:
		statusEmoji = "✅"
	}

	var adminStatus string
	if state.AdminEstablished {
		adminStatus = fmt.Sprintf("Admin: `%s`", state.AdminID)
	} else {
		adminStatus = "Admin: Not claimed"
	}

	stats := h.claimMgr.ClaimsStats()
	statusStats, ok := stats["status"].(map[string]int)
	if !ok {
		statusStats = make(map[string]int)
	}

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
func (h *CommandHandler) handleVerify(ctx context.Context, userID string, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: /verify <code>")
	}

	code := args[0]

	// Try to verify as claim challenge response
	err := h.claimMgr.RespondChallenge(code, code)
	if err == nil {
		return "✅ Challenge verified successfully. Waiting for terminal confirmation.", nil
	}

	// Check if it's a claim token
	claim, err := h.claimMgr.GetClaimByToken(code)
	if err == nil {
		return fmt.Sprintf(
			"📋 **Claim Found**\n\n"+
				"Status: `%s`\n"+
				"User: `%s`\n"+
				"Created: %s",
			claim.Status,
			claim.MatrixUserID,
			claim.CreatedAt.Format("2006-01-02 15:04:05"),
		), nil
	}

	return "", fmt.Errorf("invalid verification code")
}

// handleApprove handles the /approve command
func (h *CommandHandler) handleApprove(ctx context.Context, userID string, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: /approve <claim_id>")
	}

	claimID := args[0]

	// Check if user is admin
	state := h.lockdownMgr.GetState()
	if !state.AdminEstablished {
		return "", fmt.Errorf("no admin has been established yet")
	}

	err := h.claimMgr.ApproveClaim(claimID, userID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("✅ Claim `%s` approved successfully.", claimID), nil
}

// handleReject handles the /reject command
func (h *CommandHandler) handleReject(ctx context.Context, userID string, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: /reject <claim_id> [reason]")
	}

	claimID := args[0]
	reason := "No reason provided"
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}

	// Check if user is admin
	state := h.lockdownMgr.GetState()
	if !state.AdminEstablished {
		return "", fmt.Errorf("no admin has been established yet")
	}

	err := h.claimMgr.RejectClaim(claimID, userID, reason)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("❌ Claim `%s` rejected: %s", claimID, reason), nil
}

// handleHelp returns available commands
func (h *CommandHandler) handleHelp() string {
	return "📖 **ArmorClaw Commands**\n\n" +
		"- `/claim_admin [device_name]` - Claim admin rights (only during setup)\n" +
		"- `/status` - Show ArmorClaw status\n" +
		"- `/verify <code>` - Verify a device or claim\n" +
		"- `/approve <claim_id>` - Approve a pending claim (admin only)\n" +
		"- `/reject <claim_id> [reason]` - Reject a pending claim (admin only)\n" +
		"- `/help` - Show this help message\n\n" +
		"💡 Commands only work in rooms with the ArmorClaw agent."
}

// Update processEvents in matrix.go to call this
// Add to MatrixAdapter struct:
//   commandHandler *CommandHandler
//
// Add to New():
//   commandHandler: nil, // Set via SetCommandHandler
//
// Add method:
//   func (m *MatrixAdapter) SetCommandHandler(h *CommandHandler) {
//       m.mu.Lock()
//       defer m.mu.Unlock()
//       m.commandHandler = h
//   }

// ProcessMessageWithCommands should be called from processEvents
// before queuing the event
func (m *MatrixAdapter) ProcessMessageWithCommands(ctx context.Context, roomID, sender, message string) bool {
	m.mu.RLock()
	handler := m.commandHandler
	m.mu.RUnlock()

	if handler == nil {
		return false
	}

	return handler.HandleCommand(ctx, roomID, sender, message)
}

// sanitizeDeviceName sanitizes a device name to prevent injection attacks
// Removes control characters and limits length
func sanitizeDeviceName(name string) string {
	// Limit length to prevent abuse
	const maxLen = 64
	if len(name) > maxLen {
		name = name[:maxLen]
	}

	// Remove control characters and normalize whitespace
	var result strings.Builder
	result.Grow(len(name))

	for _, r := range name {
		// Allow printable ASCII and common Unicode, reject control characters
		if r >= 0x20 && r != 0x7F {
			// Normalize various whitespace to regular space
			if r == '\t' || r == '\n' || r == '\r' {
				result.WriteRune(' ')
			} else {
				result.WriteRune(r)
			}
		}
	}

	sanitized := result.String()

	// Trim leading/trailing whitespace and collapse multiple spaces
	sanitized = strings.TrimSpace(sanitized)
	for strings.Contains(sanitized, "  ") {
		sanitized = strings.ReplaceAll(sanitized, "  ", " ")
	}

	// If empty after sanitization, return default
	if sanitized == "" {
		return "Element X"
	}

	return sanitized
}
