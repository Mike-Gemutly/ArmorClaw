package email

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/secretary"
)

type dispatcherTestStore struct {
	mu        sync.RWMutex
	templates map[string]*secretary.TaskTemplate
}

func newDispatcherTestStore() *dispatcherTestStore {
	return &dispatcherTestStore{templates: make(map[string]*secretary.TaskTemplate)}
}

func (s *dispatcherTestStore) GetTemplateByTrigger(ctx context.Context, trigger string) (*secretary.TaskTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.templates {
		if "email:"+t.ID == trigger {
			return t, nil
		}
	}
	return nil, nil
}

func (s *dispatcherTestStore) addTemplate(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[id] = &secretary.TaskTemplate{ID: id, IsActive: true}
}

func (s *dispatcherTestStore) CreateTemplate(ctx context.Context, t *secretary.TaskTemplate) error {
	return nil
}
func (s *dispatcherTestStore) GetTemplate(ctx context.Context, id string) (*secretary.TaskTemplate, error) {
	return nil, nil
}
func (s *dispatcherTestStore) ListTemplates(ctx context.Context, f secretary.TemplateFilter) ([]secretary.TaskTemplate, error) {
	return nil, nil
}
func (s *dispatcherTestStore) UpdateTemplate(ctx context.Context, t *secretary.TaskTemplate) error {
	return nil
}
func (s *dispatcherTestStore) DeleteTemplate(ctx context.Context, id string) error { return nil }
func (s *dispatcherTestStore) CreateWorkflow(ctx context.Context, w *secretary.Workflow) error {
	return nil
}
func (s *dispatcherTestStore) GetWorkflow(ctx context.Context, id string) (*secretary.Workflow, error) {
	return nil, nil
}
func (s *dispatcherTestStore) ListWorkflows(ctx context.Context, f secretary.WorkflowFilter) ([]secretary.Workflow, error) {
	return nil, nil
}
func (s *dispatcherTestStore) UpdateWorkflow(ctx context.Context, w *secretary.Workflow) error {
	return nil
}
func (s *dispatcherTestStore) DeleteWorkflow(ctx context.Context, id string) error { return nil }
func (s *dispatcherTestStore) CreatePolicy(ctx context.Context, p *secretary.ApprovalPolicy) error {
	return nil
}
func (s *dispatcherTestStore) GetPolicy(ctx context.Context, id string) (*secretary.ApprovalPolicy, error) {
	return nil, nil
}
func (s *dispatcherTestStore) ListPolicies(ctx context.Context) ([]secretary.ApprovalPolicy, error) {
	return nil, nil
}
func (s *dispatcherTestStore) UpdatePolicy(ctx context.Context, p *secretary.ApprovalPolicy) error {
	return nil
}
func (s *dispatcherTestStore) DeletePolicy(ctx context.Context, id string) error { return nil }
func (s *dispatcherTestStore) CreateScheduledTask(ctx context.Context, t *secretary.ScheduledTask) error {
	return nil
}
func (s *dispatcherTestStore) GetScheduledTask(ctx context.Context, id string) (*secretary.ScheduledTask, error) {
	return nil, nil
}
func (s *dispatcherTestStore) ListScheduledTasks(ctx context.Context) ([]secretary.ScheduledTask, error) {
	return nil, nil
}
func (s *dispatcherTestStore) UpdateScheduledTask(ctx context.Context, t *secretary.ScheduledTask) error {
	return nil
}
func (s *dispatcherTestStore) DeleteScheduledTask(ctx context.Context, id string) error { return nil }
func (s *dispatcherTestStore) ListPendingScheduledTasks(ctx context.Context) ([]secretary.ScheduledTask, error) {
	return nil, nil
}
func (s *dispatcherTestStore) ListDueTasks(ctx context.Context) ([]secretary.ScheduledTask, error) {
	return nil, nil
}
func (s *dispatcherTestStore) MarkDispatched(ctx context.Context, id string, next time.Time) error {
	return nil
}
func (s *dispatcherTestStore) CreateNotificationChannel(ctx context.Context, c *secretary.NotificationChannel) error {
	return nil
}
func (s *dispatcherTestStore) GetNotificationChannel(ctx context.Context, id string) (*secretary.NotificationChannel, error) {
	return nil, nil
}
func (s *dispatcherTestStore) ListNotificationChannels(ctx context.Context, u string) ([]secretary.NotificationChannel, error) {
	return nil, nil
}
func (s *dispatcherTestStore) UpdateNotificationChannel(ctx context.Context, c *secretary.NotificationChannel) error {
	return nil
}
func (s *dispatcherTestStore) DeleteNotificationChannel(ctx context.Context, id string) error {
	return nil
}
func (s *dispatcherTestStore) CreateContact(ctx context.Context, c *secretary.Contact) error {
	return nil
}
func (s *dispatcherTestStore) GetContact(ctx context.Context, id string) (*secretary.Contact, error) {
	return nil, nil
}
func (s *dispatcherTestStore) ListContacts(ctx context.Context, f secretary.ContactFilter) ([]secretary.Contact, error) {
	return nil, nil
}
func (s *dispatcherTestStore) UpdateContact(ctx context.Context, c *secretary.Contact) error {
	return nil
}
func (s *dispatcherTestStore) DeleteContact(ctx context.Context, id string) error { return nil }
func (s *dispatcherTestStore) Close() error                                       { return nil }

var _ = fmt.Sprintf

func newTestDispatcher(store secretary.Store) *EmailDispatcher {
	log, _ := logger.New(logger.Config{Output: "stdout"})
	return NewEmailDispatcher(EmailDispatcherConfig{
		Store: store,
		Log:   log,
	})
}

func TestDispatcher_NoTemplateForRecipient(t *testing.T) {
	store := newDispatcherTestStore()
	d := newTestDispatcher(store)

	evt := NewEmailReceivedEvent("from@test.com", "unknown@test.com", "Hi", "body", "e1", nil, nil)
	d.OnEmailReceived(evt)
}

func TestDispatcher_TemplateFound(t *testing.T) {
	store := newDispatcherTestStore()
	store.addTemplate("handler@test.com")
	d := newTestDispatcher(store)

	evt := NewEmailReceivedEvent("from@test.com", "handler@test.com", "Hi", "body", "e2", nil, nil)
	d.OnEmailReceived(evt)
}

func TestDispatcher_RegisterHandler(t *testing.T) {
	store := newDispatcherTestStore()
	d := newTestDispatcher(store)

	called := false
	d.RegisterHandler(func(evt *EmailReceivedEvent) {
		called = true
	})

	if len(d.handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(d.handlers))
	}

	handler := d.handlers[0]
	evt := NewEmailReceivedEvent("a@b.com", "c@d.com", "sub", "body", "e3", nil, nil)
	handler(evt)

	if !called {
		t.Error("handler was not called")
	}
}

func TestDispatcher_MultipleHandlers(t *testing.T) {
	store := newDispatcherTestStore()
	d := newTestDispatcher(store)

	count := 0
	for i := 0; i < 3; i++ {
		d.RegisterHandler(func(evt *EmailReceivedEvent) {
			count++
		})
	}

	if len(d.handlers) != 3 {
		t.Fatalf("expected 3 handlers, got %d", len(d.handlers))
	}
}

func TestDispatcher_TeamRouting_Matched(t *testing.T) {
	store := newDispatcherTestStore()
	log, _ := logger.New(logger.Config{Output: "stdout"})

	matcherCalled := false
	lookupCalled := false

	matcher := func(ctx context.Context, address string) (string, bool, error) {
		matcherCalled = true
		if address == "sales@team.com" {
			return "team-abc", true, nil
		}
		return "", false, nil
	}

	lookup := func(ctx context.Context, teamID, role string) ([]string, error) {
		lookupCalled = true
		if teamID == "team-abc" && role == "email_clerk" {
			return []string{"agent-email-1"}, nil
		}
		return nil, nil
	}

	d := NewEmailDispatcher(EmailDispatcherConfig{
		Store:           store,
		Log:             log,
		TeamMatcher:     matcher,
		TeamAgentLookup: lookup,
	})

	handlerCalled := false
	d.RegisterHandler(func(evt *EmailReceivedEvent) {
		handlerCalled = true
	})

	evt := NewEmailReceivedEvent("client@external.com", "sales@team.com", "Inquiry", "body", "e-team-1", nil, nil)
	d.OnEmailReceived(evt)

	if !matcherCalled {
		t.Error("expected TeamMatcher to be called")
	}
	if !lookupCalled {
		t.Error("expected TeamAgentLookup to be called")
	}
	if !handlerCalled {
		t.Error("expected handler to be called for team-routed email")
	}
}

func TestDispatcher_TeamRouting_Unmatched(t *testing.T) {
	store := newDispatcherTestStore()
	store.addTemplate("fallback@test.com")
	log, _ := logger.New(logger.Config{Output: "stdout"})

	matcher := func(ctx context.Context, address string) (string, bool, error) {
		return "", false, nil
	}

	lookup := func(ctx context.Context, teamID, role string) ([]string, error) {
		return nil, nil
	}

	d := NewEmailDispatcher(EmailDispatcherConfig{
		Store:           store,
		Log:             log,
		TeamMatcher:     matcher,
		TeamAgentLookup: lookup,
	})

	handlerCalled := false
	d.RegisterHandler(func(evt *EmailReceivedEvent) {
		handlerCalled = true
	})

	evt := NewEmailReceivedEvent("sender@test.com", "fallback@test.com", "Hi", "body", "e-fallback", nil, nil)
	d.OnEmailReceived(evt)

	if !handlerCalled {
		t.Error("expected handler to be called via template fallback")
	}
}

func TestDispatcher_TeamRouting_NilMatcher_FallsBackToTemplate(t *testing.T) {
	store := newDispatcherTestStore()
	store.addTemplate("plain@test.com")

	d := newTestDispatcher(store)

	handlerCalled := false
	d.RegisterHandler(func(evt *EmailReceivedEvent) {
		handlerCalled = true
	})

	evt := NewEmailReceivedEvent("from@test.com", "plain@test.com", "Subject", "body", "e-nil", nil, nil)
	d.OnEmailReceived(evt)

	if handlerCalled {
		t.Error("handlers should not fire when template dispatch path has no scheduler")
	}
}

func TestDispatcher_TeamRouting_NoEmailClerk_FallsBack(t *testing.T) {
	store := newDispatcherTestStore()
	store.addTemplate("noclerk@test.com")
	log, _ := logger.New(logger.Config{Output: "stdout"})

	matcher := func(ctx context.Context, address string) (string, bool, error) {
		return "team-no-clerk", true, nil
	}

	lookup := func(ctx context.Context, teamID, role string) ([]string, error) {
		return []string{}, nil
	}

	d := NewEmailDispatcher(EmailDispatcherConfig{
		Store:           store,
		Log:             log,
		TeamMatcher:     matcher,
		TeamAgentLookup: lookup,
	})

	handlerCalled := false
	d.RegisterHandler(func(evt *EmailReceivedEvent) {
		handlerCalled = true
	})

	evt := NewEmailReceivedEvent("from@test.com", "noclerk@test.com", "Hi", "body", "e-noclerk", nil, nil)
	d.OnEmailReceived(evt)

	if handlerCalled {
		t.Error("handlers should not fire when no email_clerk and no scheduler for template path")
	}
}

func TestDispatcher_TeamRouting_MatcherError_FallsBack(t *testing.T) {
	store := newDispatcherTestStore()
	store.addTemplate("errmatch@test.com")
	log, _ := logger.New(logger.Config{Output: "stdout"})

	matcher := func(ctx context.Context, address string) (string, bool, error) {
		return "", false, errors.New("db connection lost")
	}

	lookup := func(ctx context.Context, teamID, role string) ([]string, error) {
		return nil, nil
	}

	d := NewEmailDispatcher(EmailDispatcherConfig{
		Store:           store,
		Log:             log,
		TeamMatcher:     matcher,
		TeamAgentLookup: lookup,
	})

	handlerCalled := false
	d.RegisterHandler(func(evt *EmailReceivedEvent) {
		handlerCalled = true
	})

	evt := NewEmailReceivedEvent("from@test.com", "errmatch@test.com", "Hi", "body", "e-errmatch", nil, nil)
	d.OnEmailReceived(evt)

	if handlerCalled {
		t.Error("handlers should not fire when matcher errors and no scheduler for template path")
	}
}
