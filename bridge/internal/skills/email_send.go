package skills

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// EmailParams represents parameters for sending emails
type EmailParams struct {
	From     string            `json:"from"`
	To       []string          `json:"to"`
	CC       []string          `json:"cc,omitempty"`
	BCC      []string          `json:"bcc,omitempty"`
	Subject  string            `json:"subject"`
	Body     string            `json:"body"`
	HTML     bool              `json:"html,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
	ReplyTo  string            `json:"reply_to,omitempty"`
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
	MimeType string `json:"mime_type,omitempty"`
}

// EmailResult represents the result of sending an email
type EmailResult struct {
	MessageID   string            `json:"message_id"`
	From       string            `json:"from"`
	To         []string          `json:"to"`
	CC         []string          `json:"cc,omitempty"`
	BCC        []string          `json:"bcc,omitempty"`
	Subject    string            `json:"subject"`
	SentAt     time.Time         `json:"sent_at"`
	Size       int               `json:"size_bytes"`
	Provider   string            `json:"provider"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// SMTPConfig represents SMTP server configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	From     string `json:"from"`
	TLS      bool   `json:"tls,omitempty"`
	StartTLS bool   `json:"starttls,omitempty"`
}

// ExecuteEmailSend sends an email using the specified parameters
func ExecuteEmailSend(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	emailParams, err := parseEmailParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid email parameters: %w", err)
	}

	// Validate parameters
	if err := validateEmailParams(emailParams); err != nil {
		return nil, fmt.Errorf("email validation failed: %w", err)
	}

	// Get SMTP configuration (in production, this would come from secure storage)
	smtpConfig, err := getSMTPConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMTP configuration: %w", err)
	}

	// Send the email
	return sendEmailViaSMTP(ctx, emailParams, smtpConfig)
}

// parseEmailParams parses email parameters from input
func parseEmailParams(params map[string]interface{}) (*EmailParams, error) {
	emailParams := &EmailParams{}

	// Extract required parameters
	if from, ok := params["from"].(string); ok {
		emailParams.From = strings.TrimSpace(from)
	} else {
		return nil, fmt.Errorf("from parameter is required and must be a string")
	}

	if toList, ok := params["to"].([]interface{}); ok {
		emailParams.To = make([]string, len(toList))
		for i, to := range toList {
			if toStr, ok := to.(string); ok {
				emailParams.To[i] = strings.TrimSpace(toStr)
			} else {
				return nil, fmt.Errorf("to parameter must contain only strings")
			}
		}
	} else {
		return nil, fmt.Errorf("to parameter is required and must be an array")
	}

	if len(emailParams.To) == 0 {
		return nil, fmt.Errorf("to parameter cannot be empty")
	}

	if subject, ok := params["subject"].(string); ok {
		emailParams.Subject = strings.TrimSpace(subject)
	} else {
		return nil, fmt.Errorf("subject parameter is required and must be a string")
	}

	if body, ok := params["body"].(string); ok {
		emailParams.Body = body
	} else {
		return nil, fmt.Errorf("body parameter is required and must be a string")
	}

	// Extract optional parameters
	if ccList, ok := params["cc"].([]interface{}); ok {
		emailParams.CC = make([]string, len(ccList))
		for i, cc := range ccList {
			if ccStr, ok := cc.(string); ok {
				emailParams.CC[i] = strings.TrimSpace(ccStr)
			}
		}
	}

	if bccList, ok := params["bcc"].([]interface{}); ok {
		emailParams.BCC = make([]string, len(bccList))
		for i, bcc := range bccList {
			if bccStr, ok := bcc.(string); ok {
				emailParams.BCC[i] = strings.TrimSpace(bccStr)
			}
		}
	}

	if html, ok := params["html"].(bool); ok {
		emailParams.HTML = html
	}

	if replyTo, ok := params["reply_to"].(string); ok {
		emailParams.ReplyTo = strings.TrimSpace(replyTo)
	}

	return emailParams, nil
}

// validateEmailParams validates email parameters
func validateEmailParams(params *EmailParams) error {
	// Validate from address
	if params.From == "" {
		return fmt.Errorf("from address cannot be empty")
	}
	if err := validateEmailAddress(params.From); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}

	// Validate to addresses
	for _, to := range params.To {
		if err := validateEmailAddress(to); err != nil {
			return fmt.Errorf("invalid to address '%s': %w", to, err)
		}
	}

	// Validate cc addresses
	for _, cc := range params.CC {
		if err := validateEmailAddress(cc); err != nil {
			return fmt.Errorf("invalid cc address '%s': %w", cc, err)
		}
	}

	// Validate bcc addresses
	for _, bcc := range params.BCC {
		if err := validateEmailAddress(bcc); err != nil {
			return fmt.Errorf("invalid bcc address '%s': %w", bcc, err)
		}
	}

	// Validate reply-to if provided
	if params.ReplyTo != "" {
		if err := validateEmailAddress(params.ReplyTo); err != nil {
			return fmt.Errorf("invalid reply-to address: %w", err)
		}
	}

	// Validate subject and body
	if params.Subject == "" {
		return fmt.Errorf("subject cannot be empty")
	}
	if len(params.Subject) > 500 {
		return fmt.Errorf("subject too long (max 500 characters)")
	}

	if params.Body == "" {
		return fmt.Errorf("body cannot be empty")
	}
	if len(params.Body) > 1000000 { // 1MB limit
		return fmt.Errorf("body too long (max 1MB)")
	}

	return nil
}

// validateEmailAddress validates an email address format
func validateEmailAddress(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email address cannot be empty")
	}

	// Use Go's mail.ParseAddress for validation
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}

	// Check for dangerous characters
	dangerousChars := []string{"|", "&", ";", "`", "$", "(", ")", "{", "}", "<", ">"}
	for _, char := range dangerousChars {
		if strings.Contains(email, char) {
			return fmt.Errorf("email address contains dangerous character: '%s'", char)
		}
	}

	return nil
}

// getSMTPConfig gets SMTP configuration (mock for Phase 2)
func getSMTPConfig(ctx context.Context) (*SMTPConfig, error) {
	// In production, this would load from secure storage or environment variables
	// For Phase 2, we'll return a mock configuration
	return &SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "mock-password",
		From:     "noreply@example.com",
		TLS:      true,
		StartTLS: true,
	}, nil
}

// sendEmailViaSMTP sends an email using SMTP
func sendEmailViaSMTP(ctx context.Context, emailParams *EmailParams, smtpConfig *SMTPConfig) (*EmailResult, error) {
	// For Phase 2, we'll simulate email sending
	// In production, this would use net/smtp to actually send the email
	
	startTime := time.Now()
	
	// Generate a mock message ID
	messageID := fmt.Sprintf("<%s@%s>", generateRandomID(), smtpConfig.Host)
	
	// Calculate email size
	emailSize := len(emailParams.Subject) + len(emailParams.Body)
	emailSize += len(emailParams.From) * 2 // From and Reply-To
	for _, to := range emailParams.To {
		emailSize += len(to)
	}
	for _, cc := range emailParams.CC {
		emailSize += len(cc)
	}
	for _, bcc := range emailParams.BCC {
		emailSize += len(bcc)
	}

	// Create result
	result := &EmailResult{
		MessageID: messageID,
		From:      emailParams.From,
		To:        emailParams.To,
		CC:        emailParams.CC,
		BCC:       emailParams.BCC,
		Subject:   emailParams.Subject,
		SentAt:    startTime,
		Size:      emailSize,
		Provider:  "smtp",
		Metadata: map[string]string{
			"smtp_host":     smtpConfig.Host,
			"smtp_port":     fmt.Sprintf("%d", smtpConfig.Port),
			"content_type":  "text/plain",
			"message_size":  fmt.Sprintf("%d", emailSize),
		},
	}

	// In a real implementation, this would be:
	/*
	// Connect to SMTP server
	auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)
	
	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)
	
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()
	
	// Enable TLS if required
	if smtpConfig.TLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: smtpConfig.Host}
			if err := client.StartTLS(config); err != nil {
				return nil, fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}
	
	// Authenticate if credentials provided
	if smtpConfig.Username != "" && smtpConfig.Password != "" {
		if err := client.Auth(auth); err != nil {
			return nil, fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}
	
	// Set sender
	if err := client.Mail(smtpConfig.From); err != nil {
		return nil, fmt.Errorf("failed to set sender: %w", err)
	}
	
	// Add recipients (TO, CC, BCC)
	allRecipients := append(append(emailParams.To, emailParams.CC...), emailParams.BCC...)
	for _, recipient := range allRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return nil, fmt.Errorf("failed to add recipient '%s': %w", recipient, err)
		}
	}
	
	// Send email content
	wc, err := client.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare email content: %w", err)
	}
	defer wc.Close()
	
	// Write email headers and body
	emailContent := buildEmailContent(emailParams, smtpConfig.From, messageID)
	if _, err := fmt.Fprint(wc, emailContent); err != nil {
		return nil, fmt.Errorf("failed to send email content: %w", err)
	}
	*/

	return result, nil
}

// buildEmailContent builds the raw email content (for real SMTP implementation)
func buildEmailContent(params *EmailParams, from string, messageID string) string {
	var content strings.Builder
	
	// Headers
	content.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))
	content.WriteString(fmt.Sprintf("From: %s\r\n", from))
	content.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(params.To, ", ")))
	
	if len(params.CC) > 0 {
		content.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(params.CC, ", ")))
	}
	
	if params.ReplyTo != "" {
		content.WriteString(fmt.Sprintf("Reply-To: %s\r\n", params.ReplyTo))
	}
	
	content.WriteString(fmt.Sprintf("Subject: %s\r\n", params.Subject))
	
	contentType := "text/plain; charset=utf-8"
	if params.HTML {
		contentType = "text/html; charset=utf-8"
	}
	content.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	
	content.WriteString("MIME-Version: 1.0\r\n")
	content.WriteString("\r\n")
	
	// Body
	content.WriteString(params.Body)
	content.WriteString("\r\n")
	
	return content.String()
}

// generateRandomID generates a random ID for email message ID
func generateRandomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ValidateEmailSendParams validates email sending parameters
func ValidateEmailSendParams(params map[string]interface{}) error {
	// Check required parameters
	required := []string{"from", "to", "subject", "body"}
	for _, param := range required {
		if _, exists := params[param]; !exists {
			return fmt.Errorf("%s parameter is required", param)
		}
	}
	
	// Validate from address
	if from, ok := params["from"].(string); ok {
		if err := validateEmailAddress(from); err != nil {
			return fmt.Errorf("invalid from address: %w", err)
		}
	} else {
		return fmt.Errorf("from parameter must be a string")
	}
	
	// Validate to addresses
	if toList, ok := params["to"].([]interface{}); ok {
		if len(toList) == 0 {
			return fmt.Errorf("to parameter cannot be empty")
		}
		for i, to := range toList {
			if toStr, ok := to.(string); ok {
				if err := validateEmailAddress(toStr); err != nil {
					return fmt.Errorf("invalid to address at index %d: %w", i, err)
				}
			} else {
				return fmt.Errorf("to parameter must contain only strings")
			}
		}
	} else {
		return fmt.Errorf("to parameter must be an array")
	}
	
	// Validate subject
	if subject, ok := params["subject"].(string); ok {
		if strings.TrimSpace(subject) == "" {
			return fmt.Errorf("subject cannot be empty")
		}
		if len(subject) > 500 {
			return fmt.Errorf("subject too long (max 500 characters)")
		}
	} else {
		return fmt.Errorf("subject parameter must be a string")
	}
	
	// Validate body
	if body, ok := params["body"].(string); ok {
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("body cannot be empty")
		}
		if len(body) > 1000000 {
			return fmt.Errorf("body too long (max 1MB)")
		}
	} else {
		return fmt.Errorf("body parameter must be a string")
	}
	
	// Validate optional cc if provided
	if ccList, ok := params["cc"].([]interface{}); ok {
		for i, cc := range ccList {
			if ccStr, ok := cc.(string); ok {
				if err := validateEmailAddress(ccStr); err != nil {
					return fmt.Errorf("invalid cc address at index %d: %w", i, err)
				}
			} else {
				return fmt.Errorf("cc parameter must contain only strings")
			}
		}
	}
	
	// Validate optional bcc if provided
	if bccList, ok := params["bcc"].([]interface{}); ok {
		for i, bcc := range bccList {
			if bccStr, ok := bcc.(string); ok {
				if err := validateEmailAddress(bccStr); err != nil {
					return fmt.Errorf("invalid bcc address at index %d: %w", i, err)
				}
			} else {
				return fmt.Errorf("bcc parameter must contain only strings")
			}
		}
	}
	
	// Validate optional reply_to if provided
	if replyTo, ok := params["reply_to"].(string); ok && replyTo != "" {
		if err := validateEmailAddress(replyTo); err != nil {
			return fmt.Errorf("invalid reply_to address: %w", err)
		}
	}
	
	return nil
}