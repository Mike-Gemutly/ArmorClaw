// Package adapter provides Matrix-based consent request formatting for PII access.
// Users receive consent requests in Matrix rooms and can approve/reject them.
package adapter

import (
	"fmt"
	"strings"
	"time"

	"github.com/armorclaw/bridge/pkg/pii"
)

// PIIConsentFormatter formats PII consent requests for Matrix messages
type PIIConsentFormatter struct {
	// CommandPrefix is the prefix for consent commands (e.g., "!armorclaw")
	CommandPrefix string
}

// NewPIIConsentFormatter creates a new consent formatter
func NewPIIConsentFormatter(commandPrefix string) *PIIConsentFormatter {
	if commandPrefix == "" {
		commandPrefix = "!armorclaw"
	}
	return &PIIConsentFormatter{
		CommandPrefix: commandPrefix,
	}
}

// FormatConsentRequest formats a PII access request as a Matrix message
func (f *PIIConsentFormatter) FormatConsentRequest(request *pii.AccessRequest) string {
	var sb strings.Builder

	// Header
	sb.WriteString("## 🔐 PII Access Request\n\n")

	// Skill info
	sb.WriteString(fmt.Sprintf("**Skill:** %s (`%s`)\n", request.SkillName, request.SkillID))
	sb.WriteString(fmt.Sprintf("**Request ID:** `%s`\n", request.ID))
	sb.WriteString(fmt.Sprintf("**Profile:** `%s`\n\n", request.ProfileID))

	// Requested fields
	sb.WriteString("### Requested Fields\n\n")

	// Required fields
	if len(request.RequiredFields) > 0 {
		sb.WriteString("**Required:**\n")
		for _, field := range request.RequiredFields {
			sb.WriteString(fmt.Sprintf("- %s\n", field))
		}
		sb.WriteString("\n")
	}

	// Optional fields
	optionalFields := f.getOptionalFields(request)
	if len(optionalFields) > 0 {
		sb.WriteString("**Optional:**\n")
		for _, field := range optionalFields {
			sb.WriteString(fmt.Sprintf("- %s\n", field))
		}
		sb.WriteString("\n")
	}

	// Expiration notice
	expiresIn := time.Until(request.ExpiresAt).Round(time.Second)
	sb.WriteString(fmt.Sprintf("⏱️ Expires in: %s\n\n", expiresIn))

	// Commands
	sb.WriteString("### Actions\n\n")
	sb.WriteString(fmt.Sprintf("To approve all fields:\n```\n%s approve %s\n```\n\n", f.CommandPrefix, request.ID))
	sb.WriteString(fmt.Sprintf("To approve specific fields:\n```\n%s approve %s field1,field2\n```\n\n", f.CommandPrefix, request.ID))
	sb.WriteString(fmt.Sprintf("To reject:\n```\n%s reject %s [optional reason]\n```\n", f.CommandPrefix, request.ID))

	return sb.String()
}

// FormatConsentApproved formats an approval notification
func (f *PIIConsentFormatter) FormatConsentApproved(request *pii.AccessRequest) string {
	return fmt.Sprintf("✅ **PII Access Approved**\n\n"+
		"Request `%s` has been approved.\n"+
		"Approved fields: %s",
		request.ID,
		strings.Join(request.ApprovedFields, ", "))
}

// FormatConsentRejected formats a rejection notification
func (f *PIIConsentFormatter) FormatConsentRejected(request *pii.AccessRequest) string {
	reason := request.RejectionReason
	if reason == "" {
		reason = "No reason provided"
	}
	return fmt.Sprintf("❌ **PII Access Rejected**\n\n"+
		"Request `%s` has been rejected.\n"+
		"Reason: %s",
		request.ID,
		reason)
}

// FormatConsentExpired formats an expiration notification
func (f *PIIConsentFormatter) FormatConsentExpired(requestID string) string {
	return fmt.Sprintf("⏰ **PII Access Request Expired**\n\n"+
		"Request `%s` has expired without response.",
		requestID)
}

// ParseConsentCommand parses a consent command from a Matrix message
func (f *PIIConsentFormatter) ParseConsentCommand(message string) (*ConsentCommand, error) {
	message = strings.TrimSpace(message)

	// Check prefix
	if !strings.HasPrefix(message, f.CommandPrefix) {
		return nil, nil // Not a command
	}

	// Remove prefix and parse
	parts := strings.Fields(strings.TrimPrefix(message, f.CommandPrefix))
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid command format")
	}

	cmd := &ConsentCommand{
		Action: strings.ToLower(parts[0]),
		RequestID: parts[1],
	}

	// Parse additional arguments
	if len(parts) > 2 {
		switch cmd.Action {
		case "approve":
			// Fields are comma-separated
			cmd.Fields = strings.Split(parts[2], ",")
			for i, f := range cmd.Fields {
				cmd.Fields[i] = strings.TrimSpace(f)
			}
		case "reject":
			// Reason is the rest of the message
			cmd.Reason = strings.Join(parts[2:], " ")
		}
	}

	return cmd, nil
}

// ConsentCommand represents a parsed consent command
type ConsentCommand struct {
	Action    string   // "approve" or "reject"
	RequestID string   // The request ID
	Fields    []string // Fields to approve (for approve command)
	Reason    string   // Rejection reason (for reject command)
}

// getOptionalFields returns fields that are not required
func (f *PIIConsentFormatter) getOptionalFields(request *pii.AccessRequest) []string {
	requiredSet := make(map[string]bool)
	for _, f := range request.RequiredFields {
		requiredSet[f] = true
	}

	var optional []string
	for _, f := range request.RequestedFields {
		if !requiredSet[f] {
			optional = append(optional, f)
		}
	}

	return optional
}

// SendPIIConsentRequest sends a consent request to a Matrix room
// This is a helper function that uses the MatrixAdapter
func SendPIIConsentRequest(matrix *MatrixAdapter, roomID string, request *pii.AccessRequest) error {
	formatter := NewPIIConsentFormatter("")

	message := formatter.FormatConsentRequest(request)

	// Send via Matrix adapter (roomID, message, msgType)
	_, err := matrix.SendMessage(roomID, message, "m.text")
	if err != nil {
		return fmt.Errorf("failed to send consent request: %w", err)
	}

	return nil
}
