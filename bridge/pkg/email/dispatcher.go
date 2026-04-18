package email

import (
	"context"
	"fmt"
	"sync"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/secretary"
)

// TeamMatcher resolves an email address to a team ID.
// Return ("", false, nil) when no team matches the address.
type TeamMatcher func(ctx context.Context, address string) (teamID string, found bool, err error)

// TeamAgentLookup returns the agent IDs holding a specific role within a team.
// Used to find the email_clerk after a team is matched.
type TeamAgentLookup func(ctx context.Context, teamID, role string) ([]string, error)

type EmailDispatcher struct {
	store        secretary.Store
	scheduler    *secretary.TaskScheduler
	log          *logger.Logger
	emailStorage EmailStorage
	mu           sync.Mutex
	handlers     []func(*EmailReceivedEvent)

	// TeamMatcher is optional. When set, the dispatcher tries team-based
	// routing before falling back to template-based routing.
	teamMatcher   TeamMatcher
	teamAgentLookup TeamAgentLookup
}

type EmailDispatcherConfig struct {
	Store        secretary.Store
	Scheduler    *secretary.TaskScheduler
	Log          *logger.Logger
	EmailStorage EmailStorage

	// TeamMatcher enables team-based email routing. Nil disables it.
	TeamMatcher   TeamMatcher
	TeamAgentLookup TeamAgentLookup
}

func NewEmailDispatcher(cfg EmailDispatcherConfig) *EmailDispatcher {
	return &EmailDispatcher{
		store:           cfg.Store,
		scheduler:       cfg.Scheduler,
		log:             cfg.Log,
		emailStorage:    cfg.EmailStorage,
		teamMatcher:     cfg.TeamMatcher,
		teamAgentLookup: cfg.TeamAgentLookup,
	}
}

func (d *EmailDispatcher) RegisterHandler(handler func(*EmailReceivedEvent)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers = append(d.handlers, handler)
}

func (d *EmailDispatcher) OnEmailReceived(evt *EmailReceivedEvent) {
	ctx := context.Background()

	if d.tryTeamRouting(ctx, evt) {
		return
	}

	d.dispatchViaTemplate(ctx, evt)
}

func (d *EmailDispatcher) tryTeamRouting(ctx context.Context, evt *EmailReceivedEvent) bool {
	if d.teamMatcher == nil {
		return false
	}

	teamID, found, err := d.teamMatcher(ctx, evt.To)
	if err != nil {
		d.log.Error("dispatcher_team_match_error", "to", evt.To, "error", err)
		return false
	}
	if !found {
		return false
	}

	agentIDs, err := d.teamAgentLookup(ctx, teamID, "email_clerk")
	if err != nil {
		d.log.Error("dispatcher_team_agent_lookup_error", "team_id", teamID, "error", err)
		return false
	}
	if len(agentIDs) == 0 {
		d.log.Info("dispatcher_team_no_email_clerk", "team_id", teamID)
		return false
	}

	d.log.Info("dispatcher_team_routed", "email_id", evt.EmailID, "team_id", teamID, "agents", agentIDs, "to", evt.To)

	teamEvt := &TeamRoutedEmailEvent{
		EmailReceivedEvent: evt,
		TeamID:             teamID,
		AgentIDs:           agentIDs,
	}

	d.mu.Lock()
	handlers := make([]func(*EmailReceivedEvent), len(d.handlers))
	copy(handlers, d.handlers)
	d.mu.Unlock()

	for _, h := range handlers {
		h(&teamEvt.EmailReceivedEvent)
	}

	return true
}

func (d *EmailDispatcher) dispatchViaTemplate(ctx context.Context, evt *EmailReceivedEvent) {
	template, err := d.store.GetTemplateByTrigger(ctx, "email:"+evt.To)
	if err != nil {
		d.log.Error("dispatcher_template_lookup_error", "to", evt.To, "error", err)
		return
	}
	if template == nil {
		d.log.Info("dispatcher_no_template_for_recipient", "to", evt.To)
		return
	}

	task := &secretary.ScheduledTask{
		TemplateID: template.ID,
		IsActive:   true,
		CreatedBy:  "email-pipeline",
		OneShot:    true,
	}

	if d.scheduler != nil {
		if err := d.scheduler.DispatchNow(ctx, task); err != nil {
			d.log.Error("dispatcher_dispatch_failed", "template_id", template.ID, "email_id", evt.EmailID, "error", err)
			return
		}
	}

	d.log.Info("dispatcher_email_dispatched", "email_id", evt.EmailID, "template_id", template.ID, "to", evt.To)

	d.mu.Lock()
	handlers := make([]func(*EmailReceivedEvent), len(d.handlers))
	copy(handlers, d.handlers)
	d.mu.Unlock()

	for _, h := range handlers {
		h(evt)
	}
}

var _ = fmt.Sprintf
