package email

import (
	"context"
	"fmt"
	"sync"

	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/secretary"
)

type EmailDispatcher struct {
	store        secretary.Store
	scheduler    *secretary.TaskScheduler
	log          *logger.Logger
	emailStorage EmailStorage
	mu           sync.Mutex
	handlers     []func(*EmailReceivedEvent)
}

type EmailDispatcherConfig struct {
	Store        secretary.Store
	Scheduler    *secretary.TaskScheduler
	Log          *logger.Logger
	EmailStorage EmailStorage
}

func NewEmailDispatcher(cfg EmailDispatcherConfig) *EmailDispatcher {
	return &EmailDispatcher{
		store:        cfg.Store,
		scheduler:    cfg.Scheduler,
		log:          cfg.Log,
		emailStorage: cfg.EmailStorage,
	}
}

func (d *EmailDispatcher) RegisterHandler(handler func(*EmailReceivedEvent)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers = append(d.handlers, handler)
}

func (d *EmailDispatcher) OnEmailReceived(evt *EmailReceivedEvent) {
	ctx := context.Background()

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
