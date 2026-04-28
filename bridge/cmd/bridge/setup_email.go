package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/armorclaw/bridge/pkg/email"
	"github.com/armorclaw/bridge/pkg/eventbus"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/secretary"
)

func setupEmailIngest(eventBus *eventbus.EventBus, storageBaseDir string) *email.IngestServer {
	if eventBus == nil {
		return nil
	}

	emailStorageDir := filepath.Join(storageBaseDir, "email-files")
	storage := email.NewLocalFSEmailStorage(emailStorageDir)

	socketPath := "/run/armorclaw/email-ingest.sock"

	server := email.NewIngestServer(email.IngestServerConfig{
		Storage:             storage,
		Bus:                 eventBus,
		Socket:              socketPath,
		Log:                 logger.Global(),
		SidecarOfficeClient: nil,
		SidecarRustClient:   nil,
		SidecarJavaClient:   nil,
	})

	if err := server.Start(); err != nil {
		log.Printf("Warning: Failed to start email ingest server: %v", err)
		return nil
	}

	log.Println("Email ingest server listening on", socketPath)
	return server
}

func setupEmailDispatcher(
	eventBus *eventbus.EventBus,
	taskScheduler *secretary.TaskScheduler,
	rolodexStore secretary.Store,
) *email.EmailDispatcher {
	if eventBus == nil || taskScheduler == nil || rolodexStore == nil {
		return nil
	}

	dispatcher := email.NewEmailDispatcher(email.EmailDispatcherConfig{
		Store:             rolodexStore,
		Scheduler:         taskScheduler,
		Log:               logger.Global(),
		TeamMatcher:       nil,
		TeamAgentLookup:   nil,
	})

	eventBus.RegisterBridgeHandler(eventbus.EventTypeEmailReceived, func(evt eventbus.BridgeEvent) {
		if emailEvt, ok := evt.(*email.EmailReceivedEvent); ok {
			dispatcher.OnEmailReceived(emailEvt)
		}
	})

	log.Println("Email dispatcher wired to EventBus")

	ctx := context.Background()
	if _, err := email.CreateEmailWorkflowTemplate(ctx, rolodexStore, "system"); err != nil {
		log.Printf("Warning: Failed to seed email workflow template: %v", err)
	} else {
		log.Println("Email workflow template seeded")
	}

	return dispatcher
}
