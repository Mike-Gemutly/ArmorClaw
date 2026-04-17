package email

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const auditLogDir = "/var/log/armorclaw/email"

type EmailAuditLogger struct {
	logDir string
}

func NewEmailAuditLogger() *EmailAuditLogger {
	return &EmailAuditLogger{logDir: auditLogDir}
}

func (a *EmailAuditLogger) LogEmailReceived(emailID, from, to string, attachmentCount int, piiFieldCount int) {
	a.write("email_received", map[string]interface{}{
		"email_id":         emailID,
		"from_hash":        hashForAudit(from),
		"to_hash":          hashForAudit(to),
		"attachment_count": attachmentCount,
		"pii_field_count":  piiFieldCount,
	})
}

func (a *EmailAuditLogger) LogEmailSent(emailID, to, provider, messageID string, success bool) {
	a.write("email_sent", map[string]interface{}{
		"email_id":   emailID,
		"to_hash":    hashForAudit(to),
		"provider":   provider,
		"message_id": messageID,
		"success":    success,
	})
}

func (a *EmailAuditLogger) LogApprovalRequested(emailID, approvalID string, piiFieldCount int) {
	a.write("approval_requested", map[string]interface{}{
		"email_id":        emailID,
		"approval_id":     approvalID,
		"pii_field_count": piiFieldCount,
	})
}

func (a *EmailAuditLogger) LogApprovalDecision(emailID, approvalID, decision string) {
	a.write("approval_decision", map[string]interface{}{
		"email_id":    emailID,
		"approval_id": approvalID,
		"decision":    decision,
	})
}

func (a *EmailAuditLogger) LogPIIMasking(emailID string, fieldCount int, fieldTypes []string) {
	a.write("pii_masking", map[string]interface{}{
		"email_id":    emailID,
		"field_count": fieldCount,
		"field_types": fieldTypes,
	})
}

func (a *EmailAuditLogger) LogYARAScan(emailID, filename string, isClean bool) {
	a.write("yara_scan", map[string]interface{}{
		"email_id": emailID,
		"filename": filename,
		"is_clean": isClean,
	})
}

func (a *EmailAuditLogger) write(eventType string, fields map[string]interface{}) {
	fields["event_type"] = eventType
	fields["timestamp"] = time.Now().Format(time.RFC3339)

	os.MkdirAll(a.logDir, 0700)

	dateStr := time.Now().Format("2006-01-02")
	path := filepath.Join(a.logDir, dateStr+".audit.log")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		slog.Error("audit_log_write_failed", "path", path, "error", err)
		return
	}
	defer f.Close()

	var pairs []interface{}
	for k, v := range fields {
		pairs = append(pairs, k, v)
	}
	slog.Info("email_audit", pairs...)
}

func hashForAudit(value string) string {
	if len(value) > 4 {
		return value[:2] + "***" + value[len(value)-2:]
	}
	return "***"
}
