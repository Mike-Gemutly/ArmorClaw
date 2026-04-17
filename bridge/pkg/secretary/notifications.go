package secretary

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
)

//=============================================================================
// Notification Types
//=============================================================================

type NotificationType string

const (
	NotificationWorkflowStarted   NotificationType = "workflow.started"
	NotificationWorkflowProgress  NotificationType = "workflow.progress"
	NotificationWorkflowCompleted NotificationType = "workflow.completed"
	NotificationWorkflowFailed    NotificationType = "workflow.failed"
	NotificationWorkflowCancelled NotificationType = "workflow.cancelled"
	NotificationApprovalRequired  NotificationType = "approval.required"
	NotificationApprovalApproved  NotificationType = "approval.approved"
	NotificationApprovalDenied    NotificationType = "approval.denied"
)

//=============================================================================
// Notification Payload
//=============================================================================

type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	TemplateID  string                 `json:"template_id,omitempty"`
	StepID      string                 `json:"step_id,omitempty"`
	StepName    string                 `json:"step_name,omitempty"`
	Progress    float64                `json:"progress,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	PolicyID    string                 `json:"policy_id,omitempty"`
	Initiator   string                 `json:"initiator,omitempty"`
	DecidedBy   string                 `json:"decided_by,omitempty"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	Result      string                 `json:"result,omitempty"`
	PIIFields   []string               `json:"pii_fields,omitempty"`
	Duration    int64                  `json:"duration_ms,omitempty"`
	Recoverable bool                   `json:"recoverable,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Recipient   string                 `json:"recipient,omitempty"`
	Delivered   bool                   `json:"delivered"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
}

//=============================================================================
// Notification Subscriber Interface
//=============================================================================

type NotificationSubscriber interface {
	Notify(notification *Notification) error
}

//=============================================================================
// Notification Service
//=============================================================================

type NotificationService struct {
	mu            sync.RWMutex
	store         Store
	subscribers   []NotificationSubscriber
	log           *logger.Logger
	notifications map[string]*Notification
}

type NotificationServiceConfig struct {
	Store  Store
	Logger *logger.Logger
}

func NewNotificationService(cfg NotificationServiceConfig) *NotificationService {
	log := cfg.Logger
	if log == nil {
		log = logger.Global().WithComponent("notification_service")
	}

	return &NotificationService{
		store:         cfg.Store,
		subscribers:   make([]NotificationSubscriber, 0),
		notifications: make(map[string]*Notification),
		log:           log,
	}
}

//=============================================================================
// Subscriber Management
//=============================================================================

func (s *NotificationService) Subscribe(subscriber NotificationSubscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers = append(s.subscribers, subscriber)
}

func (s *NotificationService) Unsubscribe(subscriber NotificationSubscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sub := range s.subscribers {
		if sub == subscriber {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			break
		}
	}
}

//=============================================================================
// Notification Dispatch
//=============================================================================

func (s *NotificationService) Dispatch(notification *Notification) error {
	if notification.ID == "" {
		notification.ID = generateNotificationID()
	}
	if notification.Timestamp == 0 {
		notification.Timestamp = time.Now().UnixMilli()
	}

	s.mu.Lock()
	s.notifications[notification.ID] = notification
	subscribers := make([]NotificationSubscriber, len(s.subscribers))
	copy(subscribers, s.subscribers)
	s.mu.Unlock()

	s.log.Info("notification_dispatching",
		"notification_id", notification.ID,
		"type", notification.Type,
		"workflow_id", notification.WorkflowID,
		"recipient", notification.Recipient)

	var dispatchErr error
	for _, subscriber := range subscribers {
		if err := subscriber.Notify(notification); err != nil {
			dispatchErr = err
			s.log.Warn("notification_subscriber_error",
				"notification_id", notification.ID,
				"error", err.Error())
		}
	}

	now := time.Now()
	notification.Delivered = true
	notification.DeliveredAt = &now

	return dispatchErr
}

//=============================================================================
// Workflow Notification Helpers
//=============================================================================

func (s *NotificationService) NotifyWorkflowStarted(workflow *Workflow, template *TaskTemplate) error {
	notification := &Notification{
		Type:       NotificationWorkflowStarted,
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Initiator:  workflow.CreatedBy,
		Message:    fmt.Sprintf("Workflow '%s' has started", workflow.Name),
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"workflow_name": workflow.Name,
			"status":        string(workflow.Status),
		},
		Recipient: workflow.CreatedBy,
	}

	if template != nil {
		notification.Metadata["template_name"] = template.Name
		notification.Metadata["total_steps"] = len(template.Steps)
	}

	return s.Dispatch(notification)
}

func (s *NotificationService) NotifyWorkflowProgress(workflow *Workflow, stepID, stepName string, progress float64, template *TaskTemplate) error {
	notification := &Notification{
		Type:       NotificationWorkflowProgress,
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		StepID:     stepID,
		StepName:   stepName,
		Progress:   progress,
		Message:    fmt.Sprintf("Workflow '%s' progress: %s (%.0f%% complete)", workflow.Name, stepName, progress*100),
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"workflow_name": workflow.Name,
			"status":        string(workflow.Status),
		},
		Recipient: workflow.CreatedBy,
	}

	if template != nil {
		notification.Metadata["total_steps"] = len(template.Steps)
	}

	return s.Dispatch(notification)
}

func (s *NotificationService) NotifyWorkflowCompleted(workflow *Workflow, result string, template *TaskTemplate) error {
	var duration int64
	if workflow.CompletedAt != nil {
		duration = workflow.CompletedAt.Sub(workflow.StartedAt).Milliseconds()
	}

	notification := &Notification{
		Type:       NotificationWorkflowCompleted,
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Result:     result,
		Duration:   duration,
		Message:    fmt.Sprintf("Workflow '%s' completed successfully", workflow.Name),
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"workflow_name": workflow.Name,
			"duration_ms":   duration,
		},
		Recipient: workflow.CreatedBy,
	}

	if template != nil {
		notification.Metadata["template_name"] = template.Name
	}

	return s.Dispatch(notification)
}

func (s *NotificationService) NotifyWorkflowFailed(workflow *Workflow, stepID string, err error, recoverable bool, template *TaskTemplate) error {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	notification := &Notification{
		Type:        NotificationWorkflowFailed,
		WorkflowID:  workflow.ID,
		TemplateID:  workflow.TemplateID,
		StepID:      stepID,
		Error:       errorMsg,
		Recoverable: recoverable,
		Message:     fmt.Sprintf("Workflow '%s' failed: %s", workflow.Name, errorMsg),
		Timestamp:   time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"workflow_name": workflow.Name,
			"recoverable":   recoverable,
		},
		Recipient: workflow.CreatedBy,
	}

	if template != nil {
		for _, step := range template.Steps {
			if step.StepID == stepID {
				notification.StepName = step.Name
				break
			}
		}
	}

	return s.Dispatch(notification)
}

func (s *NotificationService) NotifyWorkflowCancelled(workflow *Workflow, reason string, template *TaskTemplate) error {
	var duration int64
	if workflow.CompletedAt != nil {
		duration = workflow.CompletedAt.Sub(workflow.StartedAt).Milliseconds()
	}

	notification := &Notification{
		Type:       NotificationWorkflowCancelled,
		WorkflowID: workflow.ID,
		TemplateID: workflow.TemplateID,
		Reason:     reason,
		Duration:   duration,
		Message:    fmt.Sprintf("Workflow '%s' cancelled: %s", workflow.Name, reason),
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"workflow_name": workflow.Name,
			"reason":        reason,
		},
		Recipient: workflow.CreatedBy,
	}

	return s.Dispatch(notification)
}

//=============================================================================
// Approval Notification Helpers
//=============================================================================

func (s *NotificationService) NotifyApprovalRequired(request *ApprovalRequest, workflow *Workflow) error {
	notification := &Notification{
		Type:       NotificationApprovalRequired,
		RequestID:  request.ID,
		PolicyID:   request.PolicyID,
		WorkflowID: request.WorkflowID,
		TemplateID: request.TemplateID,
		StepID:     request.StepID,
		Initiator:  request.Initiator,
		PIIFields:  request.PIIFields,
		Message:    fmt.Sprintf("Approval required for workflow '%s' - fields: %v", request.WorkflowID, request.PIIFields),
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"pii_fields": request.PIIFields,
			"subject":    request.Subject,
		},
		Recipient: request.Initiator,
	}

	if request.Context != nil {
		if delegateTo, ok := request.Context["delegate_to"].(string); ok && delegateTo != "" {
			notification.Recipient = delegateTo
		}
	}

	return s.Dispatch(notification)
}

func (s *NotificationService) NotifyApprovalApproved(request *ApprovalRequest, decidedBy string, reason string) error {
	notification := &Notification{
		Type:       NotificationApprovalApproved,
		RequestID:  request.ID,
		PolicyID:   request.PolicyID,
		WorkflowID: request.WorkflowID,
		TemplateID: request.TemplateID,
		StepID:     request.StepID,
		Initiator:  request.Initiator,
		DecidedBy:  decidedBy,
		Reason:     reason,
		PIIFields:  request.PIIFields,
		Message:    fmt.Sprintf("Approval granted for request %s by %s", request.ID, decidedBy),
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"decided_by": decidedBy,
			"decision":   string(DecisionAllow),
			"pii_fields": request.PIIFields,
		},
		Recipient: request.Initiator,
	}

	return s.Dispatch(notification)
}

func (s *NotificationService) NotifyApprovalDenied(request *ApprovalRequest, decidedBy string, reason string) error {
	notification := &Notification{
		Type:       NotificationApprovalDenied,
		RequestID:  request.ID,
		PolicyID:   request.PolicyID,
		WorkflowID: request.WorkflowID,
		TemplateID: request.TemplateID,
		StepID:     request.StepID,
		Initiator:  request.Initiator,
		DecidedBy:  decidedBy,
		Reason:     reason,
		PIIFields:  request.PIIFields,
		Message:    fmt.Sprintf("Approval denied for request %s by %s: %s", request.ID, decidedBy, reason),
		Timestamp:  time.Now().UnixMilli(),
		Metadata: map[string]interface{}{
			"decided_by": decidedBy,
			"decision":   string(DecisionDeny),
			"reason":     reason,
			"pii_fields": request.PIIFields,
		},
		Recipient: request.Initiator,
	}

	return s.Dispatch(notification)
}

//=============================================================================
// Notification Retrieval
//=============================================================================

func (s *NotificationService) GetNotification(id string) (*Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notification, exists := s.notifications[id]
	if !exists {
		return nil, fmt.Errorf("notification not found: %s", id)
	}

	copy := *notification
	return &copy, nil
}

func (s *NotificationService) ListNotifications(workflowID string) []*Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Notification
	for _, notification := range s.notifications {
		if workflowID == "" || notification.WorkflowID == workflowID {
			copy := *notification
			result = append(result, &copy)
		}
	}
	return result
}

func (s *NotificationService) ListPendingNotifications(recipient string) []*Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Notification
	for _, notification := range s.notifications {
		if !notification.Delivered && (recipient == "" || notification.Recipient == recipient) {
			copy := *notification
			result = append(result, &copy)
		}
	}
	return result
}

func (s *NotificationService) GetNotificationCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.notifications)
}

//=============================================================================
// Matrix Subscriber Adapter
//=============================================================================

type MatrixNotificationAdapter struct {
	sendFunc func(ctx context.Context, roomID, message string) error
}

func NewMatrixNotificationAdapter(sendFunc func(ctx context.Context, roomID, message string) error) *MatrixNotificationAdapter {
	return &MatrixNotificationAdapter{sendFunc: sendFunc}
}

func (a *MatrixNotificationAdapter) Notify(notification *Notification) error {
	if a.sendFunc == nil {
		return nil
	}

	recipient := notification.Recipient
	if recipient == "" {
		return nil
	}

	return a.sendFunc(context.Background(), recipient, notification.Message)
}

//=============================================================================
// Notification Builder
//=============================================================================

type NotificationBuilder struct {
	notification *Notification
}

func NewNotificationBuilder(notifType NotificationType) *NotificationBuilder {
	return &NotificationBuilder{
		notification: &Notification{
			Type:      notifType,
			Timestamp: time.Now().UnixMilli(),
			Metadata:  make(map[string]interface{}),
			PIIFields: make([]string, 0),
		},
	}
}

func (b *NotificationBuilder) WithWorkflowID(workflowID string) *NotificationBuilder {
	b.notification.WorkflowID = workflowID
	return b
}

func (b *NotificationBuilder) WithTemplateID(templateID string) *NotificationBuilder {
	b.notification.TemplateID = templateID
	return b
}

func (b *NotificationBuilder) WithStep(stepID, stepName string) *NotificationBuilder {
	b.notification.StepID = stepID
	b.notification.StepName = stepName
	return b
}

func (b *NotificationBuilder) WithProgress(progress float64) *NotificationBuilder {
	b.notification.Progress = progress
	return b
}

func (b *NotificationBuilder) WithRequestID(requestID string) *NotificationBuilder {
	b.notification.RequestID = requestID
	return b
}

func (b *NotificationBuilder) WithPolicyID(policyID string) *NotificationBuilder {
	b.notification.PolicyID = policyID
	return b
}

func (b *NotificationBuilder) WithInitiator(initiator string) *NotificationBuilder {
	b.notification.Initiator = initiator
	return b
}

func (b *NotificationBuilder) WithDecidedBy(decidedBy string) *NotificationBuilder {
	b.notification.DecidedBy = decidedBy
	return b
}

func (b *NotificationBuilder) WithMessage(message string) *NotificationBuilder {
	b.notification.Message = message
	return b
}

func (b *NotificationBuilder) WithError(err string) *NotificationBuilder {
	b.notification.Error = err
	return b
}

func (b *NotificationBuilder) WithReason(reason string) *NotificationBuilder {
	b.notification.Reason = reason
	return b
}

func (b *NotificationBuilder) WithResult(result string) *NotificationBuilder {
	b.notification.Result = result
	return b
}

func (b *NotificationBuilder) WithPIIFields(fields []string) *NotificationBuilder {
	b.notification.PIIFields = fields
	return b
}

func (b *NotificationBuilder) WithDuration(duration time.Duration) *NotificationBuilder {
	b.notification.Duration = duration.Milliseconds()
	return b
}

func (b *NotificationBuilder) WithRecoverable(recoverable bool) *NotificationBuilder {
	b.notification.Recoverable = recoverable
	return b
}

func (b *NotificationBuilder) WithRecipient(recipient string) *NotificationBuilder {
	b.notification.Recipient = recipient
	return b
}

func (b *NotificationBuilder) WithMetadata(key string, value interface{}) *NotificationBuilder {
	b.notification.Metadata[key] = value
	return b
}

func (b *NotificationBuilder) Build() *Notification {
	if b.notification.ID == "" {
		b.notification.ID = generateNotificationID()
	}
	return b.notification
}

//=============================================================================
// Timeline Formatting
//=============================================================================

// stepIcon returns an emoji for the given event type.
func stepIcon(eventType string) string {
	switch eventType {
	case "step":
		return "🔹"
	case "file_read":
		return "📄"
	case "file_write":
		return "✏️"
	case "file_delete":
		return "🗑️"
	case "command_run":
		return "⌨️"
	case "observation":
		return "💭"
	case "blocker":
		return "🚧"
	case "error":
		return "❌"
	case "artifact":
		return "📦"
	case "checkpoint":
		return "🏁"
	default:
		return "•"
	}
}

// FormatTimelineMessage formats an ExtendedStepResult into a human-readable
// timeline string. If no parsed events are available, it falls back to the
// plain output text.
func FormatTimelineMessage(result *ExtendedStepResult) string {
	if result == nil || result.Events == nil || len(result.Events) == 0 {
		if result != nil {
			return result.Output
		}
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📋 %s\n", result.Output))
	b.WriteString("───────────\n")

	totalDuration := int64(0)
	if result.ContainerStepResult != nil {
		totalDuration = result.DurationMS
	}

	stepCount := 0
	for _, evt := range result.Events {
		if evt.Type == "_summary" || evt.Type == "progress" {
			continue
		}

		icon := stepIcon(evt.Type)
		b.WriteString(fmt.Sprintf("%s %s", icon, evt.Name))

		switch evt.Type {
		case "file_read":
			if lines, ok := evt.Detail["lines"]; ok {
				b.WriteString(fmt.Sprintf(" (%v lines)", lines))
			}
		case "file_write":
			if changes, ok := evt.Detail["changes"]; ok {
				b.WriteString(fmt.Sprintf(" (%v changes)", changes))
			}
		case "command_run":
			if code, ok := evt.Detail["exit_code"]; ok {
				if codeNum, ok := toInt(code); ok && codeNum == 0 {
					b.WriteString(" ✓")
				} else if ok {
					b.WriteString(fmt.Sprintf(" ✗ exit %d", codeNum))
				}
			}
		case "blocker":
			if msg, ok := evt.Detail["message"].(string); ok && msg != "" {
				b.WriteString(fmt.Sprintf(": %s", msg))
			}
		case "artifact":
			if size, ok := evt.Detail["size_bytes"]; ok {
				b.WriteString(fmt.Sprintf(" (%v bytes)", size))
			}
		}

		if evt.Detail != nil {
			if truncated, ok := evt.Detail["_truncated"].(bool); ok && truncated {
				b.WriteString(" [truncated]")
			}
		}

		if evt.DurationMs != nil {
			b.WriteString(fmt.Sprintf(" (%dms)", *evt.DurationMs))
		}

		b.WriteString("\n")
		stepCount++
	}

	summaryTotal := 0
	if result.EventsSummary != nil {
		summaryTotal = result.EventsSummary.Total
	}
	if summaryTotal == 0 {
		summaryTotal = stepCount
	}

	totalSec := float64(totalDuration) / 1000.0
	b.WriteString(fmt.Sprintf("\n⏱ %.1fs · %d steps", totalSec, summaryTotal))

	return b.String()
}

// FormatBlockerMessage formats a list of blockers into a human-readable Matrix
// notification. Returns an empty string if blockers is nil or empty.
func FormatBlockerMessage(blockers []Blocker, timeout time.Duration) string {
	if len(blockers) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("🚧 **Agent blocked — action required**\n\n")

	for i, bl := range blockers {
		b.WriteString(fmt.Sprintf("**Blocker %d:** %s\n", i+1, bl.Message))
		if bl.Suggestion != "" {
			b.WriteString(fmt.Sprintf("💡 %s\n", bl.Suggestion))
		}
		if bl.Field != "" {
			b.WriteString(fmt.Sprintf("🔑 Field: %s\n", bl.Field))
		}
		if bl.BlockerType != "" {
			b.WriteString(fmt.Sprintf("📋 Type: %s\n", bl.BlockerType))
		}
		if i < len(blockers)-1 {
			b.WriteString("───────────\n")
		}
	}

	b.WriteString(fmt.Sprintf("\n⏱ Expires in %s", timeout.Round(time.Second)))

	return b.String()
}

// toInt attempts to convert an interface{} to int.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

//=============================================================================
// ID Generation
//=============================================================================

var notificationIDCounter int64

func generateNotificationID() string {
	notificationIDCounter++
	return fmt.Sprintf("notif_%d_%d", time.Now().UnixMilli(), notificationIDCounter)
}
