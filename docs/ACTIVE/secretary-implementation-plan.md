# ArmorClaw Secretary Implementation Plan

> **Version:** 1.0
> **Created:** 2026-03-13
> **Status:** Draft for Review

---

## Executive Summary

This plan details the implementation of "Secretary" features for ArmorClaw - enabling users to delegate common tasks like booking appointments, managing calendars, filling forms, and handling email through AI agents with proper PII protection and approval flows.

**Key Constraints:**
- ✅ Browser automation remains Playwright-based (existing `browser-service`)
- ✅ No Rod dependency introduced
- ✅ Runtime orchestration stays in ArmorClaw Bridge + Matrix
- ✅ API keys remain environment-based (no persistence)
- ✅ All custodial PII requires policy enforcement, audit logging, and approval flow

---

## Phase 1: Secretary Skill Package (2-3 days)

### Goal
Create `bridge/pkg/secretary/` package with core secretary skills.

### Files to Create

```
bridge/pkg/secretary/
├── types.go              # Core types (Task, Booking, CalendarEvent, etc.)
├── registry.go           # Secretary skill registration
├── calendar.go           # Calendar operations (read, create, update)
├── booking.go            # Appointment booking workflows
├── email.go              # Email composition and sending
├── forms.go              # Form filling with PII protection
├── contacts.go           # Contact management
├── workflows.go          # Multi-step secretary workflows
├── audit.go              # Secretary-specific audit logging
└── secretary_test.go     # Unit tests
```

### New Skills to Register

| Skill ID | Category | Description | PII Required |
|----------|----------|-------------|--------------|
| `calendar.read` | productivity | Read calendar events | none |
| `calendar.create` | productivity | Create calendar events | `user_email` |
| `calendar.update` | productivity | Modify calendar events | `user_email` |
| `booking.appointment` | automation | Book appointments | `user_name`, `user_email`, `user_phone` |
| `booking.reschedule` | automation | Reschedule appointments | `user_name`, `user_email` |
| `booking.cancel` | automation | Cancel appointments | `user_email` |
| `email.compose` | communication | Draft emails | `user_email` |
| `email.send` | communication | Send emails (requires approval) | `user_email`, `email_recipients` |
| `forms.fill` | automation | Fill web forms | Varies by form |
| `contacts.read` | data | Read contact info | none |
| `contacts.update` | data | Update contacts | `user_name`, `user_email`, `user_phone` |

### Types Definition

```go
// types.go
package secretary

import "time"

// Task represents a secretary task
type Task struct {
    ID          string       `json:"id"`
    Type        TaskType     `json:"type"`
    Status      TaskStatus   `json:"status"`
    Priority    Priority     `json:"priority"`
    Description string       `json:"description"`
    DueDate     *time.Time   `json:"due_date,omitempty"`
    CreatedAt   time.Time    `json:"created_at"`
    CompletedAt *time.Time   `json:"completed_at,omitempty"`
    Result      *TaskResult  `json:"result,omitempty"`
    PIIUsed     []string     `json:"pii_used,omitempty"`
    AuditID     string       `json:"audit_id"`
}

type TaskType string
const (
    TaskTypeBooking     TaskType = "booking"
    TaskTypeCalendar    TaskType = "calendar"
    TaskTypeEmail       TaskType = "email"
    TaskTypeFormFill    TaskType = "form_fill"
    TaskTypeResearch    TaskType = "research"
)

type TaskStatus string
const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusApproved   TaskStatus = "approved"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusCompleted  TaskStatus = "completed"
    TaskStatusFailed     TaskStatus = "failed"
    TaskStatusCancelled  TaskStatus = "cancelled"
)

type Priority string
const (
    PriorityLow      Priority = "low"
    PriorityMedium   Priority = "medium"
    PriorityHigh     Priority = "high"
    PriorityCritical Priority = "critical"
)

// Booking represents an appointment booking
type Booking struct {
    ID           string     `json:"id"`
    Service      string     `json:"service"`      // e.g., "dentist", "haircut"
    Provider     string     `json:"provider"`     // Business or person name
    ProviderURL  string     `json:"provider_url"` // Booking website
    DateTime     time.Time  `json:"date_time"`
    Duration     int        `json:"duration"`     // Minutes
    Status       BookingStatus `json:"status"`
    Confirmation string     `json:"confirmation,omitempty"`
    Notes        string     `json:"notes,omitempty"`
}

type BookingStatus string
const (
    BookingStatusPending   BookingStatus = "pending"
    BookingStatusConfirmed BookingStatus = "confirmed"
    BookingStatusCancelled BookingStatus = "cancelled"
    BookingStatusCompleted BookingStatus = "completed"
)

// CalendarEvent represents a calendar entry
type CalendarEvent struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description,omitempty"`
    Start       time.Time `json:"start"`
    End         time.Time `json:"end"`
    Location    string    `json:"location,omitempty"`
    Attendees   []string  `json:"attendees,omitempty"`
    Recurrence  string    `json:"recurrence,omitempty"` // RFC 5545 RRULE
    Reminders   []int     `json:"reminders,omitempty"` // Minutes before
}

// EmailDraft represents an email composition
type EmailDraft struct {
    ID        string   `json:"id"`
    To        []string `json:"to"`
    Cc        []string `json:"cc,omitempty"`
    Bcc       []string `json:"bcc,omitempty"`
    Subject   string   `json:"subject"`
    Body      string   `json:"body"`
    Attachments []string `json:"attachments,omitempty"`
    Status    EmailStatus `json:"status"`
    SentAt    *time.Time `json:"sent_at,omitempty"`
}

type EmailStatus string
const (
    EmailStatusDraft   EmailStatus = "draft"
    EmailStatusPending EmailStatus = "pending_approval"
    EmailStatusSent    EmailStatus = "sent"
    EmailStatusFailed  EmailStatus = "failed"
)
```

### Acceptance Criteria Phase 1

- [ ] `bridge/pkg/secretary/types.go` exists with core types
- [ ] 11 secretary skills registered in skill registry
- [ ] Unit tests pass for all types
- [ ] No external dependencies added beyond existing

---

## Phase 2: Secretary PII Fields (1 day)

### Goal
Add secretary-specific PII fields to the existing PII registry.

### New PII Fields

| Field ID | Name | Sensitivity | Requires Approval | Keystore Key |
|----------|------|-------------|-------------------|--------------|
| `user_full_name` | Full Name | low | No | profile.full_name |
| `user_email` | Email Address | medium | No | profile.email |
| `user_phone` | Phone Number | medium | No | profile.phone |
| `user_address` | Street Address | medium | No | profile.address |
| `user_city` | City | low | No | profile.city |
| `user_postal_code` | Postal Code | low | No | profile.postal_code |
| `user_country` | Country | low | No | profile.country |
| `payment_card_number` | Credit Card Number | critical | Yes | payment.card_number |
| `payment_card_expiry` | Card Expiry | high | Yes | payment.card_expiry |
| `payment_card_cvv` | Card CVV | critical | Yes | payment.card_cvv |
| `payment_card_name` | Name on Card | medium | No | payment.card_name |
| `emergency_contact` | Emergency Contact | medium | No | profile.emergency_contact |
| `date_of_birth` | Date of Birth | high | Yes | profile.dob |
| `insurance_number` | Insurance/SSN | critical | Yes | profile.insurance_id |
| `passport_number` | Passport Number | critical | Yes | profile.passport |

### Files to Modify

```
bridge/pkg/studio/store.go          # Add default PII fields
bridge/pkg/pii/profile.go           # Extend profile with secretary fields
bridge/pkg/keystore/pii_request.go  # Add secretary field resolvers
```

### Acceptance Criteria Phase 2

- [ ] 15 secretary PII fields registered
- [ ] Critical/high sensitivity fields require approval
- [ ] Keystore resolvers work for all fields
- [ ] PII scrubbing covers new field patterns

---

## Phase 3: Booking Workflow Implementation (3-4 days)

### Goal
Implement appointment booking workflows using existing browser client.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     BOOKING WORKFLOW ARCHITECTURE                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌───────────┐ │
│  │ Matrix Cmd  │────►│ Secretary   │────►│ PII HITL    │────►│ Browser   │ │
│  │ !book       │     │ Package     │     │ Consent     │     │ Client    │ │
│  └─────────────┘     └─────────────┘     └─────────────┘     └───────────┘ │
│         │                   │                   │                   │       │
│         │                   │                   │                   │       │
│         ▼                   ▼                   ▼                   ▼       │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌───────────┐ │
│  │ Audit Log   │     │ Task Queue  │     │ Approval    │     │ Playwright│ │
│  │ Entry       │     │             │     │ Request     │     │ Service   │ │
│  └─────────────┘     └─────────────┘     └─────────────┘     └───────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Booking Flow

```
1. User sends: !book haircut at "Supercuts" tomorrow 2pm
2. Bridge creates Task with type=booking
3. Secretary package resolves "tomorrow 2pm" to specific datetime
4. System checks required PII fields for booking skill
5. If PII needed, create HITL consent request
6. User approves via Matrix reaction or command
7. Browser client navigates to provider URL
8. Fill form with PII (via BlindFill references)
9. Confirm booking
10. Extract confirmation details
11. Log audit entry
12. Notify user of result
```

### Files to Create/Modify

```
bridge/pkg/secretary/booking.go      # Booking workflow logic
bridge/pkg/secretary/workflows.go    # Multi-step workflow orchestration
bridge/pkg/rpc/secretary.go          # RPC handlers for secretary methods
bridge/pkg/matrixcmd/secretary.go    # Matrix command handlers
```

### Browser Workflow Integration

```go
// booking.go - BookingWorkflow executes a booking using browser client
func (b *BookingExecutor) Execute(ctx context.Context, booking *Booking, pii map[string]string) (*BookingResult, error) {
    // Step 1: Navigate to booking site
    navResp, err := b.browser.Navigate(ctx, browser.ServiceNavigateCommand{
        URL:       booking.ProviderURL,
        WaitUntil: browser.ServiceWaitUntilNetworkIdle,
        Timeout:   30000,
    })
    if err != nil {
        return nil, fmt.Errorf("navigation failed: %w", err)
    }
    
    // Step 2: Fill booking form using PII references (not plaintext)
    fillFields := []browser.ServiceFillField{
        {Selector: "#name", ValueRef: "user_full_name"},
        {Selector: "#email", ValueRef: "user_email"},
        {Selector: "#phone", ValueRef: "user_phone"},
        {Selector: "#date", Value: booking.DateTime.Format("2006-01-02")},
        {Selector: "#time", Value: booking.DateTime.Format("15:04")},
    }
    
    fillResp, err := b.browser.Fill(ctx, browser.ServiceFillCommand{
        Fields:     fillFields,
        AutoSubmit: false, // Require confirmation step
    })
    
    // Step 3: Wait for user to select specific time slot if needed
    // ... interactive steps ...
    
    // Step 4: Submit and confirm
    clickResp, err := b.browser.Click(ctx, browser.ServiceClickCommand{
        Selector: "#confirm-booking",
        WaitFor:  "navigation",
    })
    
    // Step 5: Extract confirmation
    extractResp, err := b.browser.Extract(ctx, browser.ServiceExtractCommand{
        Fields: []browser.ServiceExtractField{
            {Name: "confirmation_number", Selector: ".confirmation-number"},
            {Name: "confirmed_time", Selector: ".confirmed-time"},
        },
    })
    
    // Step 6: Audit log the booking
    b.audit.LogBooking(ctx, booking.ID, pii["user_email"], extractResp.Data)
    
    return &BookingResult{
        Confirmation: extractResp.Data["confirmation_number"],
        Status:       BookingStatusConfirmed,
    }, nil
}
```

### Acceptance Criteria Phase 3

- [ ] No Rod dependency introduced (verified via `go.mod` diff)
- [ ] Browser service remains Playwright-based (no changes to `browser-service/`)
- [ ] Policy checks precede fill execution (PII consent required before fill)
- [ ] Audit entries emitted for all booking actions
- [ ] No plaintext PII in logs (verify via log inspection)
- [ ] Screenshot on error for debugging

### Security Checkpoints Phase 3

| Checkpoint | Verification |
|------------|--------------|
| No Rod import | `grep -r "github.com/go-rod/rod" bridge/` returns nothing |
| Playwright only | Browser client only calls `browser-service` HTTP endpoints |
| PII consent first | Fill not called until `WaitForApproval` returns approved |
| Audit logged | Every `Execute()` call creates audit entry with task ID |
| No PII in logs | Logger uses `pii.Redact()` for all sensitive values |

---

## Phase 4: Calendar Integration (2 days)

### Goal
Add calendar read/write capabilities with Google Calendar and CalDAV support.

### Files to Create

```
bridge/pkg/secretary/calendar.go           # Calendar interface
bridge/pkg/secretary/calendar_google.go    # Google Calendar client
bridge/pkg/secretary/calendar_caldav.go    # CalDAV client
bridge/pkg/secretary/calendar_local.go     # Local SQLite calendar (fallback)
```

### Calendar Interface

```go
// calendar.go
type CalendarProvider interface {
    // ListEvents returns events in a time range
    ListEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error)
    
    // CreateEvent creates a new event
    CreateEvent(ctx context.Context, event *CalendarEvent) (*CalendarEvent, error)
    
    // UpdateEvent modifies an existing event
    UpdateEvent(ctx context.Context, event *CalendarEvent) error
    
    // DeleteEvent removes an event
    DeleteEvent(ctx context.Context, eventID string) error
    
    // GetFreeBusy returns busy time slots
    GetFreeBusy(ctx context.Context, start, end time.Time) ([]TimeSlot, error)
}
```

### Acceptance Criteria Phase 4

- [ ] Google Calendar OAuth flow works
- [ ] Events can be listed, created, updated, deleted
- [ ] Free/busy query works
- [ ] Audit logging for all calendar operations

---

## Phase 5: Email Integration (2 days)

### Goal
Enable email composition and sending with approval flow.

### Files to Create

```
bridge/pkg/secretary/email.go        # Email interface and types
bridge/pkg/secretary/email_smtp.go   # SMTP client
bridge/pkg/secretary/email_draft.go  # Draft management
```

### Email Approval Flow

```
1. User: !email draft to john@example.com "Meeting tomorrow"
2. Agent composes email body
3. System creates draft with status=pending_approval
4. User reviews via Matrix (shows subject, recipient, body preview)
5. User reacts with ✅ to approve or ❌ to reject
6. If approved, email sent via SMTP
7. Audit log entry created
8. User notified of send status
```

### Acceptance Criteria Phase 5

- [ ] Email drafts can be created
- [ ] Approval required before sending
- [ ] SMTP integration works
- [ ] Audit log includes recipient, subject, timestamp

---

## Phase 6: Matrix Commands (2 days)

### Goal
Add secretary-specific Matrix commands.

### Commands

| Command | Description | Example |
|---------|-------------|---------|
| `!book <service> at <provider> <when>` | Book appointment | `!book haircut at Supercuts tomorrow 2pm` |
| `!calendar [list\|today\|week]` | View calendar | `!calendar week` |
| `!schedule <what> <when>` | Create event | `!schedule "Team meeting" friday 3pm` |
| `!email <to> <subject>` | Draft email | `!email john@example.com "Project update"` |
| `!tasks [list\|done <id>]` | Manage tasks | `!tasks list` |
| `!approve <request_id>` | Approve PII access | `!approve req_abc123` |
| `!reject <request_id> <reason>` | Reject PII access | `!reject req_abc123 Not needed` |

### Files to Create/Modify

```
bridge/pkg/matrixcmd/secretary.go    # Secretary command handlers
bridge/pkg/matrixcmd/handler.go      # Register secretary commands
```

### Acceptance Criteria Phase 6

- [ ] All 7 commands work via Matrix
- [ ] Approval commands integrate with HITL consent
- [ ] Responses formatted for mobile (Element X)

---

## Phase 7: Migrations and Config (1 day)

### Goal
Add database migrations and configuration.

### Database Migrations

```sql
-- migrations/010_secretary_tasks.sql
CREATE TABLE secretary_tasks (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    priority TEXT NOT NULL,
    description TEXT,
    due_date INTEGER,
    created_at INTEGER NOT NULL,
    completed_at INTEGER,
    result TEXT,
    pii_used TEXT, -- JSON array
    audit_id TEXT,
    user_id TEXT NOT NULL,
    room_id TEXT NOT NULL
);

CREATE INDEX idx_tasks_user ON secretary_tasks(user_id);
CREATE INDEX idx_tasks_status ON secretary_tasks(status);

-- migrations/011_secretary_bookings.sql
CREATE TABLE secretary_bookings (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES secretary_tasks(id),
    service TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_url TEXT,
    date_time INTEGER NOT NULL,
    duration INTEGER,
    status TEXT NOT NULL,
    confirmation TEXT,
    notes TEXT,
    user_id TEXT NOT NULL
);

-- migrations/012_secretary_calendar_events.sql
CREATE TABLE secretary_calendar_cache (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    external_id TEXT NOT NULL,
    title TEXT,
    description TEXT,
    start_time INTEGER NOT NULL,
    end_time INTEGER NOT NULL,
    location TEXT,
    attendees TEXT, -- JSON array
    last_synced INTEGER NOT NULL,
    UNIQUE(provider, external_id)
);
```

### Configuration

```toml
# config.toml additions
[secretary]
enabled = true
default_calendar_provider = "google"  # google, caldav, local
timezone = "America/New_York"

[secretary.booking]
auto_confirm = false
default_duration_minutes = 60
reminder_minutes = [30, 60, 1440]  # 30min, 1hr, 1day before

[secretary.email]
default_from = ""
smtp_host = ""
smtp_port = 587
require_approval = true

[secretary.audit]
retention_days = 365
include_screenshots = true
```

### Acceptance Criteria Phase 7

- [ ] Migrations run successfully
- [ ] Config options documented
- [ ] Backward compatible with existing installs

---

## Summary: File/Package Map

### New Files

| Package | Files | Purpose |
|---------|-------|---------|
| `bridge/pkg/secretary/` | 12 files | Core secretary functionality |
| `bridge/pkg/matrixcmd/secretary.go` | 1 file | Matrix command handlers |
| `bridge/pkg/rpc/secretary.go` | 1 file | RPC method handlers |
| `migrations/` | 3 SQL files | Database schema |

### Modified Files

| File | Changes |
|------|---------|
| `bridge/pkg/studio/store.go` | Add default secretary skills |
| `bridge/pkg/studio/registry.go` | Register secretary skills |
| `bridge/pkg/pii/profile.go` | Add secretary PII fields |
| `bridge/pkg/keystore/pii_request.go` | Add field resolvers |
| `bridge/pkg/matrixcmd/handler.go` | Register secretary commands |
| `bridge/config.example.toml` | Add secretary config section |

---

## Security Checkpoints Summary

| Phase | Checkpoint | Verification Method |
|-------|------------|---------------------|
| 1 | No Rod dependency | `grep -r "go-rod" bridge/` |
| 2 | PII fields properly classified | Review sensitivity levels |
| 3 | Playwright only | Browser calls only via HTTP client |
| 3 | PII consent before fill | Code review of workflow order |
| 3 | Audit on all operations | Check audit log after test |
| 3 | No PII in logs | Run test, inspect log output |
| 4 | Calendar API tokens encrypted | Verify keystore storage |
| 5 | Email approval required | Test email send without approval |
| 6 | Commands properly authorized | Test with non-admin user |

---

## Estimated Timeline

| Phase | Duration | Dependencies |
|-------|----------|--------------|
| Phase 1: Secretary Skills | 2-3 days | None |
| Phase 2: PII Fields | 1 day | Phase 1 |
| Phase 3: Booking Workflow | 3-4 days | Phase 1, 2 |
| Phase 4: Calendar | 2 days | Phase 1 |
| Phase 5: Email | 2 days | Phase 1, 2 |
| Phase 6: Matrix Commands | 2 days | Phase 3, 4, 5 |
| Phase 7: Migrations | 1 day | All phases |
| **Total** | **13-17 days** | |

---

## Next Steps

1. **Review this plan** against actual requirements
2. **Prioritize phases** based on user value
3. **Create implementation branches** per phase
4. **Begin Phase 1** after approval
