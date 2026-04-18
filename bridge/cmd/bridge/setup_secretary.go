// setup_secretary.go — Secretary service wiring (extracted from runBridgeServer)
package main

import (
	"log"

	"github.com/armorclaw/bridge/internal/adapter"
	"github.com/armorclaw/bridge/internal/events"
	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/secretary"
	"github.com/armorclaw/bridge/pkg/studio"
)

func setupSecretaryServices(ks *keystore.Keystore) (secretary.Store, *secretary.RolodexService, *secretary.WebDAVService, *secretary.CalendarService) {
	log.Println("Initializing Rolodex service...")
	rolodexStore, err := secretary.NewStore(secretary.StoreConfig{
		Path:   "/var/lib/armorclaw/rolodex.db",
		Logger: nil,
	})
	if err != nil {
		log.Printf("Warning: Failed to initialize Rolodex store: %v", err)
		rolodexStore = nil
	}

	var rolodexService *secretary.RolodexService
	if rolodexStore != nil {
		rolodexService, err = secretary.NewRolodexService(secretary.RolodexConfig{
			Store:    rolodexStore,
			Keystore: ks,
			Logger:   nil,
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize Rolodex service: %v", err)
			rolodexService = nil
		} else {
			log.Println("Rolodex service initialized")
		}
	}

	log.Println("Initializing WebDAV service...")
	webdavService := secretary.NewWebDAVService()
	log.Println("WebDAV service initialized")

	log.Println("Initializing Calendar service...")
	calendarService := secretary.NewCalendarService()
	log.Println("Calendar service initialized")

	return rolodexStore, rolodexService, webdavService, calendarService
}

func setupApprovalAndTrust(rolodexStore secretary.Store) (*secretary.ApprovalEngineImpl, *secretary.TrustedWorkflowEngine) {
	var approvalEngine *secretary.ApprovalEngineImpl
	if rolodexStore != nil {
		var err error
		approvalEngine, err = secretary.NewApprovalEngine(secretary.ApprovalEngineConfig{
			Store:  rolodexStore,
			Logger: nil,
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize Approval engine: %v", err)
			approvalEngine = nil
		} else {
			log.Println("Approval engine initialized")
		}
	}

	var trustEngine *secretary.TrustedWorkflowEngine
	if rolodexStore != nil {
		var err error
		trustEngine, err = secretary.NewTrustedWorkflowEngine(secretary.TrustedWorkflowConfig{
			Store:  rolodexStore,
			Logger: nil,
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize Trust engine: %v", err)
			trustEngine = nil
		} else {
			log.Println("Trust engine initialized")
		}
	}

	return approvalEngine, trustEngine
}

func setupWorkflowEngine(rolodexStore secretary.Store, matrixBus *events.MatrixEventBus, studioService *studio.StudioIntegration) (*secretary.WorkflowOrchestratorImpl, *secretary.OrchestratorIntegration) {
	var workflowOrchestrator *secretary.WorkflowOrchestratorImpl
	if rolodexStore != nil && matrixBus != nil {
		workflowEmitter := secretary.NewWorkflowEventEmitter(matrixBus)

		var orchestratorFactory secretary.Factory
		if studioService != nil {
			if fac := studioService.GetFactory(); fac != nil {
				orchestratorFactory = fac
			}
		}

		var orchErr error
		workflowOrchestrator, orchErr = secretary.NewWorkflowOrchestrator(secretary.OrchestratorConfig{
			Store:    rolodexStore,
			Factory:  orchestratorFactory,
			EventBus: workflowEmitter,
		})
		if orchErr != nil {
			log.Printf("Warning: failed to create workflow orchestrator: %v", orchErr)
			workflowOrchestrator = nil
		}
	}

	var orchestratorIntegration *secretary.OrchestratorIntegration
	if rolodexStore != nil && studioService != nil {
		dependencyValidator := secretary.NewDependencyValidator()

		var stepExecutor *secretary.StepExecutor
		if fac := studioService.GetFactory(); fac != nil {
			stepExecutor = secretary.NewStepExecutor(secretary.StepExecutorConfig{
				Factory:        fac,
				Validator:      dependencyValidator,
				ApprovalEngine: nil,
				EventBus:       matrixBus,
			})
		}

		notificationService := secretary.NewNotificationService(secretary.NotificationServiceConfig{
			Store: rolodexStore,
		})

		if workflowOrchestrator != nil && stepExecutor != nil {
			orchestratorIntegration = secretary.NewOrchestratorIntegration(secretary.IntegrationConfig{
				Orchestrator:        workflowOrchestrator,
				Executor:            stepExecutor,
				Store:               rolodexStore,
				ApprovalEngine:      nil,
				NotificationService: notificationService,
			})
			log.Println("Workflow execution engine initialized")
		}
	}

	return workflowOrchestrator, orchestratorIntegration
}

func setupSecretaryCommandHandler(
	rolodexStore secretary.Store,
	workflowOrchestrator *secretary.WorkflowOrchestratorImpl,
	orchestratorIntegration *secretary.OrchestratorIntegration,
	matrixAdapter *adapter.MatrixAdapter,
	studioService *studio.StudioIntegration,
	rolodexService *secretary.RolodexService,
	webdavService *secretary.WebDAVService,
	calendarService *secretary.CalendarService,
	approvalEngine *secretary.ApprovalEngineImpl,
	trustEngine *secretary.TrustedWorkflowEngine,
) *secretary.TaskScheduler {
	if matrixAdapter != nil {
		secretaryHandler := secretary.NewSecretaryCommandHandler(secretary.SecretaryCommandHandlerConfig{
			Store:          rolodexStore,
			Orchestrator:   workflowOrchestrator,
			Integration:    orchestratorIntegration,
			Studio:         nil,
			Matrix:         secretary.WrapMatrixAdapter(matrixAdapter),
			Prefix:         "!",
			Rolodex:        rolodexService,
			WebDAV:         webdavService,
			Calendar:       calendarService,
			ApprovalEngine: approvalEngine,
			TrustEngine:    trustEngine,
		})
		matrixAdapter.SetStudioCommandHandler(&compositeStudioHandler{
			studio:    studioService,
			secretary: secretaryHandler,
		})
		log.Println("Studio command handler wired to Matrix adapter")
	}

	var taskScheduler *secretary.TaskScheduler
	if rolodexStore != nil && studioService != nil {
		var schedulerFactory secretary.FactoryInterface
		if fac := studioService.GetFactory(); fac != nil {
			schedulerFactory = &studioFactoryAdapter{factory: fac}
		}

		var schedulerMatrix secretary.MatrixAdapter
		if matrixAdapter != nil {
			schedulerMatrix = &schedulerMatrixAdapter{adapter: matrixAdapter}
		}

		if schedulerFactory != nil {
			taskScheduler = secretary.NewTaskScheduler(rolodexStore, schedulerFactory, schedulerMatrix, nil, workflowOrchestrator, orchestratorIntegration)
			taskScheduler.Start()
			log.Println("Task scheduler started (15s tick interval)")
		}
	}

	return taskScheduler
}
