# Secretary Workflow System

> Part of the [ArmorClaw System Documentation](armorclaw.md)

Deep dive into ArmorClaw's workflow engine: scheduled tasks, multi-step workflows, and PII approval gates.

> **Source root:** `bridge/pkg/secretary/`

---

## Overview

The secretary package is ArmorClaw's automation core. It turns task templates into runnable workflows, dispatches them on cron schedules or on demand, and enforces human approval gates whenever a step touches PII.

Three things happen inside the secretary:

1. **Scheduling.** A `TaskScheduler` polls the database every 15 seconds, picks up tasks whose `next_run` is due, and dispatches them.
2. **Workflow execution.** For tasks with a template, the scheduler creates a `Workflow` record, hands it to the `WorkflowOrchestratorImpl`, and the orchestrator walks through each `WorkflowStep` sequentially, spawning isolated containers for each one.
3. **PII approval.** Before any step that references PII fields, the `ApprovalEngineImpl` evaluates policies. If a policy requires manual approval, the step blocks until the user responds from ArmorChat (or a 120 second timeout expires).

```
ScheduledTask (cron)
       │
       ▼
  TaskScheduler.tick()
       │
       ├── template_id set? ──► templateDispatch()
       │        creates Workflow ──► Orchestrator.StartWorkflow()
       │        OrchestratorIntegration.StartWorkflowExecution()
       │               │
       │               ▼
       │        StepExecutor.ExecuteSteps()
       │          for each step:
       │            ApprovalEngine.EvaluateStep()  ──► PII gate
       │            factory.Spawn()                 ──► container
       │            waitForCompletion()             ──► 500ms poll
       │            AdvanceWorkflow()
       │
       └── no template ──► warmDispatch() or coldDispatch()
                running agent? ──► Matrix event injection
                no agent?      ──► container spawn
```

---

> ⚠️ **CRITICAL: Data Flow Limitation**
>
> Agent containers in the secretary workflow execute with `NetworkMode: "none"`, meaning they have **zero network access**. Communication is strictly unidirectional:
>
> - **Inbound to container**: Environment variables (`STEP_CONFIG`, `PII_*` fallback)
> - **Outbound from container**: Exit code only (0 = success, non-zero = failure)
>
> There is **no structured result passing**. The container cannot report partial output, intermediate data, rich results, or progress. The only signal the Bridge receives is binary: **success or failure**.
>
> This means:
> - Multi-step workflows cannot pass structured data between steps
> - Agent state transitions (BROWSING, FORM_FILLING, etc.) are **invisible** to the Bridge
> - Browser automation is **impossible** in this mode (no network to reach browser service)
> - `workflow.progress` events are **Bridge-inferred** (container still running), NOT agent-reported
>
> A backward communication channel is planned to address these limitations.

---

## Architecture

### Component map

| Component | Source file | Role |
|-----------|------------|------|
| `WorkflowOrchestratorImpl` | `orchestrator.go` | State machine: pending → running → completed/failed/cancelled. Holds active workflows in memory, emits events on every transition. |
| `DependencyValidator` | `orchestrator.go` (embedded) | Validates step ordering before execution. |
| `OrchestratorIntegration` | `orchestrator_integration.go` | Glues orchestrator + executor + approval engine + notifications together. Owns the goroutine that runs a workflow end to end. |
| `StepExecutor` | `orchestrator_integration.go` | Spawns containers, polls for completion, retries on recoverable errors. |
| `WorkflowEventEmitter` | `orchestrator_events.go` | Publishes `workflow.*` events to the `MatrixEventBus`. |
| `ApprovalEngineImpl` | `approvals.go` | Evaluates policies against PII fields. Returns allow/deny/require_approval per field. |
| `PendingApproval` / `HandlePIIResponse` | `pending_approval.go` | Blocking PII gate: publishes `app.armorclaw.pii_request` to Matrix, waits for `app.armorclaw.pii_response`. |
| `NotificationService` | `notifications.go` | Fan out workflow and approval notifications to subscribers (Matrix adapter, etc.). |
| `TaskScheduler` | `task_scheduler.go` | 15 second tick loop. Stateless dispatcher that reads due tasks from DB. |
| `MatrixEventBus` | `bridge/internal/events/matrix_event_bus.go` | Ring buffer (default 1024 slots). Delivers events to the Matrix conduit and to in process subscribers. |

### Key types (types.go)

```
TaskTemplate           Definition of a reusable workflow (steps, variables, PII refs)
Workflow               Runtime instance of a template
WorkflowStep           One step in a template (action, condition, parallel variants)
WorkflowStatus         pending | running | completed | failed | cancelled
ApprovalPolicy         Rules for auto approve vs. manual gate, per PII field
ApprovalResult         Outcome of evaluating policies: approved/denied/needs_approval
ScheduledTask          Cron entry that triggers a template dispatch or a direct agent spawn
```

Step types (`StepType` enum):

| StepType | Constant | Purpose |
|----------|----------|---------|
| `StepAction` | `"action"` | Execute an agent action |
| `StepCondition` | `"condition"` | Evaluate a condition |
| `StepParallel` | `"parallel"` | Execute steps in parallel |
| `StepParallelSplit` | `"parallel_split"` | Fork into parallel branches |
| `StepParallelMerge` | `"parallel_merge"` | Rejoin parallel branches |

---

## Two Dispatch Paths

The `TaskScheduler` has two completely separate dispatch paths.

### Path 1: Workflow engine (template dispatch)

Triggered when `ScheduledTask.TemplateID` is set.

1. `templateDispatch()` fetches the `TaskTemplate` from the store.
2. Creates a `Workflow` record in `pending` status.
3. Calls `Orchestrator.StartWorkflow(workflowID)` which transitions to `running` and launches a background goroutine.
4. Calls `OrchestratorIntegration.StartWorkflowExecution(workflowID)` which runs `StepExecutor.ExecuteSteps()` in a goroutine.
5. On completion, calls `Orchestrator.CompleteWorkflow()` or `FailWorkflow()`.
6. After dispatch, calculates the next run time from the cron expression and updates the scheduled task.

### Path 2: Warm / cold dispatch (no template)

Triggered when `ScheduledTask.DefinitionID` is set but `TemplateID` is empty.

The scheduler asks the factory if there is already a running instance for that definition:

- **❌ Warm dispatch (NON-FUNCTIONAL).** A running agent exists. The scheduler sends a `app.armorclaw.task_dispatch` Matrix event into the agent's room. However, the agent container has **no Matrix connection** (`NetworkMode: "none"`). The event is received only by ArmorChat clients and the Bridge's own Matrix sync. The container never sees it. This path **silently fails** with no error indication.
- **✅ Cold dispatch (FUNCTIONAL, limited).** No running agent. The scheduler calls `factory.Spawn()` to create a fresh container for the task. Functional but limited to exit-code-only results.

After either path, the task's `next_run` is updated (cron) or the task is deactivated (one shot).

---

## Workflow Lifecycle

### State machine

```
              ┌─────────────┐
              │   pending   │
              └──────┬──────┘
                     │ StartWorkflow()
                     ▼
              ┌─────────────┐
        ┌────▶│   running   │◀───┐
        │     └──┬───┬───┬──┘    │
        │        │   │   │       │
        │        │   │   │       │
   CancelWorkflow│   │   │ AdvanceWorkflow
        │        │   │   │ (last step)
        │        │   │   │       │
        │        ▼   │   ▼       │
  ┌──────────┐  │  ┌──────────┐  │
  │cancelled │  │  │completed │──┘
  └──────────┘  │  └──────────┘
           FailWorkflow()
                │
                ▼
         ┌──────────┐
         │  failed  │
         └──────────┘
```

Valid transitions (defined in `validateTransition`):

| From | To |
|------|----|
| pending | running, cancelled |
| running | completed, failed, cancelled |
| completed | (terminal) |
| failed | (terminal) |
| cancelled | (terminal) |

### Lifecycle walk through

1. **Template loaded.** `StartWorkflow()` loads the `TaskTemplate` from the store, sets `CurrentStep` to the first step's ID, stores the workflow in `activeWorkflows` map, emits `workflow.started`, and kicks off `executeWorkflow()` in a goroutine.

2. **Execution loop.** `OrchestratorIntegration.runWorkflow()` calls `StepExecutor.ExecuteSteps()`, which iterates through the validated step order.

3. **Step completion.** After each step, `AdvanceWorkflow()` updates `CurrentStep` and `currentIndex`. When the last step finishes, it calls `completeWorkflowLocked()` which sets status to `completed`, removes the workflow from `activeWorkflows`, and emits `workflow.completed`.

4. **Failure.** `FailWorkflow()` sets status to `failed`, stores the error message, removes from active map, emits `workflow.failed`.

5. **Cancellation.** `CancelWorkflow()` cancels the context, sets status to `cancelled`, removes from active map, emits `workflow.cancelled`. Also cancels any running step containers via `executor.CancelAllForWorkflow()`.

---

## Step Execution

Each step goes through `StepExecutor.executeStep()`:

```
StepExecutor                      Container (NetworkMode: "none")
    │                                      │
    ├─ No agentIDs? ──► ErrNoAgentForStep  │
    │                                      │
    ├─ Build SpawnRequest:                 │
    │     DefinitionID, TaskDescription,   │
    │     UserID, RoomID, Config           │
    │                                      │
    ├─ STEP_CONFIG env ──────────────────▶ │ (inbound: env vars only)
    │                                      │
    ├─ Register in runningSteps map        │
    │                                      │
    └─ waitForCompletion(ctx, instanceID)  │
         │                                 │
         └─ 500ms polling loop:            │
              GetStatus(instanceID)        │
                Completed  ──▶ nil (0) ◀───│ (outbound: exit code only)
                Failed     ──▶ Err  (non0)│
                Running    ──▶ continue    │
                ctx.Done() ──▶ Stop, error│
```

### Retry behavior

`executeStepWithRetry()` wraps `executeStep()` with configurable retry:

- Default retry count: 1 (one retry after initial failure)
- Default retry delay: 1 second
- Default timeout: 5 minutes
- Only recoverable errors are retried. Spawn failures are recoverable. Steps with no agent assigned are not.

### Config passthrough

Each `WorkflowStep` carries a `Config` field (`json.RawMessage`). This is passed directly to `SpawnRequest.Config` when the container is spawned. The container receives it as the `STEP_CONFIG` environment variable.

This is how template authors pass step specific configuration (API endpoints, parameters, flags) into the agent container without modifying the agent definition.

### Data flow limitation

Containers spawned by the step executor run with `NetworkMode: "none"`. The executor observes only the container's exit code:

- Exit 0 (status `Completed`): step succeeded.
- Non zero exit (status `Failed`): step failed.
- Container still running: keep polling.

There is no structured result passing. The container cannot report partial output, intermediate data, or rich results back to the orchestrator. The only signal is binary: success or failure.

---

## PII Approval Flow

This is the human in the loop gate that blocks workflow steps involving sensitive data.

### How it triggers

Inside `StepExecutor.ExecuteSteps()`, before executing each step:

```go
if e.approvalEngine != nil && len(template.PIIRefs) > 0 {
    approvalResult, err := e.approvalEngine.EvaluateStep(...)
    // If approval required and not yet approved:
    approvedFields, err := PendingApproval(ctx, eventBus, roomID, stepID, deniedFields)
}
```

Only runs when the template declares `PIIRefs` (PII field references). If there are no PII references, the approval check is skipped entirely.

### Policy evaluation (approvals.go)

`ApprovalEngineImpl.Evaluate()`:

1. If `PIIFields` is empty, immediately returns approved (no gate).
2. Loads all active policies from the store.
3. Filters policies whose `PIIFields` overlap with the requested fields.
4. For each matching policy, calls `evaluateSinglePolicy()`:
   - If `AutoApprove` is true and conditions pass: allow.
   - If `AutoApprove` is true but conditions fail: require_approval.
   - If conditions pass: allow.
   - Otherwise: require_approval.
5. Merges results across policies: approved fields minus denied fields, denied fields win over approved.

Condition evaluation supports these operators: `eq`/`==`/`=`, `neq`/`!=`, `in`, `nin`/`not_in`, `contains`. Fields that can be checked include `workflow.status`, `workflow.created_by`, `template.id`, `template.name`, `step.type`, `step.id`, `initiator`, `subject`, plus any key in the workflow's `Variables` map.

### The blocking gate (pending_approval.go)

When `ApprovalResult.NeedsApproval` is true:

1. `PendingApproval()` registers a channel in a global map (`pendingApps`), keyed by step ID.
2. Publishes an `app.armorclaw.pii_request` event to the Matrix room containing the step ID and required fields.
3. Blocks on one of three outcomes:
   - **Approved.** `HandlePIIResponse()` delivers a response via the channel. Returns approved field list.
   - **Denied.** Same channel, but `Approved: false`. Returns error.
   - **Timeout.** 120 seconds. Returns error.
   - **Cancellation.** Context cancelled (workflow cancelled). Returns error.

4. `HandlePIIResponse()` is the entry point called by the RPC handler when an `app.armorclaw.pii_response` Matrix event arrives from the ArmorChat client. It looks up the step ID in the pending map and sends the response down the channel.

```
StepExecutor                    MatrixEventBus              ArmorChat
    │                               │                         │
    ├─ PendingApproval()            │                         │
    │   register channel            │                         │
    │   publish pii_request ────────▶│──► Matrix room ────────▶│
    │   block on channel            │                         │
    │                               │                         │
    │                               │    user taps Approve     │
    │                               │◀── pii_response ────────│
    │                               │                         │
    │   HandlePIIResponse()         │                         │
    │   channel <- response         │                         │
    │   unblock                     │                         │
    │   continue step execution     │                         │
```

### Approval outcomes

| Scenario | What happens |
|----------|-------------|
| No PII fields in template | No approval check. Step runs immediately. |
| PII fields but no matching policies | Auto approved. Step runs. |
| Policy with auto_approve + conditions pass | Auto approved. Step runs. |
| Policy requires approval | Blocks. PII request sent to Matrix. Waits for response. |
| User approves | Step unblocks. Execution continues. |
| User denies | Step fails with "PII approval denied". Workflow fails. |
| 120s timeout | Step fails with timeout error. Workflow fails. |
| Workflow cancelled while waiting | Step unblocks with context cancellation. Workflow cleaned up. |

---

## Event System

### Workflow events (orchestrator_events.go)

`WorkflowEventEmitter` publishes five event types to the `MatrixEventBus`:

| Event type | Constant | Triggered by |
|------------|----------|-------------|
| `workflow.started` | `WorkflowEventStarted` | `StartWorkflow()` |
| `workflow.progress` | `WorkflowEventProgress` | `AdvanceWorkflow()`, `UpdateProgress()`, `executeWorkflow()` ticker |
| `workflow.completed` | `WorkflowEventCompleted` | `completeWorkflowLocked()` |
| `workflow.failed` | `WorkflowEventFailed` | `FailWorkflow()` |
| `workflow.cancelled` | `WorkflowEventCancelled` | `CancelWorkflow()` |

Each event carries: workflow ID, template ID, status, optional step info, progress percentage, error message, duration in milliseconds, and arbitrary metadata.

### PII events (pending_approval.go)

| Event type | Direction | Purpose |
|------------|-----------|---------|
| `app.armorclaw.pii_request` | Orchestrator → client | Asks user to approve PII field access |
| `app.armorclaw.pii_response` | Client → orchestrator | User's approve/deny decision |

### MatrixEventBus (bridge/internal/events/matrix_event_bus.go)

The bus is a fixed size ring buffer (default 1024 slots, max batch 128 events). It supports:

- `Publish()` — adds an event, broadcasts to waiters, notifies live subscribers.
- `GetEventsAfter(cursor)` — returns events newer than the given sequence number.
- `WaitForEvents(cursor)` — blocks with 25ms polling until new events arrive.
- `Subscribe()` — returns a buffered channel (cap 100) that receives every published event.

Subscribers that are too slow are silently skipped (non blocking send). The ring buffer wraps around, dropping the oldest events when full.

> **Note**: `workflow.progress` events are emitted by the Bridge when it polls the container's Docker status and finds it still running. This is **not** agent-reported progress. It merely indicates the container process has not exited yet. There is no mechanism for the container to report its actual execution phase.

---

## Notifications

`NotificationService` is a pub/sub layer separate from the raw event bus. It formats human readable messages and dispatches them to registered `NotificationSubscriber` implementations.

### Notification types

| Type | When |
|------|------|
| `workflow.started` | Workflow begins execution |
| `workflow.progress` | Step progress updates |
| `workflow.completed` | All steps finished |
| `workflow.failed` | A step failed |
| `workflow.cancelled` | Workflow was cancelled |
| `approval.required` | PII approval needed |
| `approval.approved` | User approved PII access |
| `approval.denied` | User denied PII access |

### Matrix adapter

`MatrixNotificationAdapter` implements `NotificationSubscriber` by calling a `sendFunc(ctx, roomID, message)`. This is how notifications reach the user's Matrix room as readable messages (not structured events).

---

## Execution Mode Capabilities

The secretary workflow engine operates in **Mode A (Agent Studio)**:

| Capability | Status | Notes |
|-----------|--------|-------|
| Scheduled task triggering | ✅ Works | Cron-based, 15s tick interval |
| Container lifecycle management | ✅ Works | Spawn, poll, stop |
| PII approval gating | ✅ Works | Matrix → user → approve/deny |
| Workflow state tracking | ✅ Works | Bridge-level: pending → running → completed/failed |
| Structured step results | ❌ Not available | Exit code only |
| Agent-reported progress | ❌ Not available | Bridge-inferred only |
| Browser automation | ❌ Not available | No network access |
| Warm dispatch | ❌ Non-functional | No Matrix connection in container |

**Mode B (OpenClaw Gateway)** provides network access via HTTP_PROXY but has its own limitations (integration incomplete). Mode A/B convergence is deferred until both modes have a working backward channel.

---

## Integration Points

### How the pieces connect

```
TaskScheduler
    │
    ├── store (SQLCipher) ── templates, workflows, scheduled tasks, policies
    ├── factory (studio.AgentFactory) ── container spawn/stop/status
    ├── matrix (MatrixAdapter) ── warm dispatch events
    │
    ├── orchestrator (WorkflowOrchestratorImpl)
    │       ├── store ── workflow CRUD
    │       ├── factory ── container lifecycle
    │       └── eventEmitter (WorkflowEventEmitter)
    │               └── bus (MatrixEventBus) ── ring buffer + subscribers
    │
    └── integration (OrchestratorIntegration)
            ├── orchestrator
            ├── executor (StepExecutor)
            │       ├── factory ── spawn containers
            │       ├── validator (DependencyValidator) ── step order validation
            │       ├── approvalEngine (ApprovalEngineImpl) ── PII policy eval
            │       └── eventBus (MatrixEventBus) ── PII request/response
            ├── store
            ├── approvalEngine
            └── notificationService (NotificationService)
                    └── subscribers [MatrixNotificationAdapter, ...]
```

### Shutdown behavior

`Orchestrator.Shutdown()` iterates all active workflows, cancels their contexts, sets each to `cancelled` status with reason "orchestrator shutdown", persists to store, emits `workflow.cancelled` events, and clears the active map.

`TaskScheduler.Stop()` closes the stop channel and waits for the goroutine to exit.

---

## Source File Reference

| File | Key types/functions |
|------|-------------------|
| `orchestrator.go` | `WorkflowOrchestratorImpl`, `NewWorkflowOrchestrator`, `StartWorkflow`, `AdvanceWorkflow`, `CancelWorkflow`, `CompleteWorkflow`, `FailWorkflow`, `validateTransition` |
| `orchestrator_integration.go` | `StepExecutor`, `NewStepExecutor`, `ExecuteSteps`, `executeStep`, `executeStepWithRetry`, `waitForCompletion`, `OrchestratorIntegration`, `StartWorkflowExecution`, `runWorkflow` |
| `orchestrator_events.go` | `EventEmitter` interface, `WorkflowEventEmitter`, `WorkflowEvent`, `WorkflowEventBuilder` |
| `approvals.go` | `ApprovalEngineImpl`, `Evaluate`, `EvaluateStep`, `EvaluateWorkflow`, `evaluatePolicies`, `ApprovalPolicy`, `ApprovalRequest` |
| `pending_approval.go` | `PendingApproval`, `HandlePIIResponse`, PII event constants |
| `notifications.go` | `NotificationService`, `Notification`, `NotificationSubscriber` interface, `MatrixNotificationAdapter` |
| `task_scheduler.go` | `TaskScheduler`, `NewTaskScheduler`, `Start`, `Stop`, `tick`, `dispatchTask`, `templateDispatch`, `warmDispatch`, `coldDispatch` |
| `types.go` | `TaskTemplate`, `Workflow`, `WorkflowStep`, `StepType`, `WorkflowStatus`, `ApprovalResult`, `ApprovalPolicy`, `ScheduledTask`, interface definitions |
| `bridge/internal/events/matrix_event_bus.go` | `MatrixEventBus`, `MatrixEvent`, `Publish`, `GetEventsAfter`, `Subscribe` |

---

## Prerequisites for Full Functionality

The secretary workflow engine's full potential requires a **backward communication channel** from agent containers to the Bridge. The planned approach:

1. **Shared state dir**: Container writes `result.json` to the bind-mounted state directory before exit
2. **Bridge reads result**: After container exit, Bridge reads and parses `result.json`
3. **Structured step results**: Multi-step workflows can pass data between steps
4. **PII socket wiring**: Secure PII delivery via Unix socket instead of environment variables

This is planned as a single atomic change. Both the backward channel and PII socket wiring must ship together.
