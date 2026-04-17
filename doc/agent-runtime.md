# Agent Runtime Internals

> Part of the [ArmorClaw System Documentation](armorclaw.md)

> **Bridge-Side Architecture**
>
> The agent runtime and state machine described in this document are **Bridge-side only**. They run inside the Go Bridge process, not inside agent containers.
>
> **Container-to-Bridge reporting.** Containers in step mode now emit structured events to `_events.jsonl` during execution (via `EventEmitter` in `events.py`). The Bridge tails this file via `EventReader` for real-time progress. The 11-state state machine (`IDLE` through `OFFLINE`) is a Bridge-internal library used for lifecycle tracking. Containers cannot report which high-level state they are in. The Bridge observes: `running` (container exists) -> `completed` (exit 0) -> `failed` (exit non-zero), plus structured `StepEvent` entries from the event stream.
>
> **Backward channel:** Containers in step mode (STEP_CONFIG present) write structured results to `result.json` in the bind-mounted state dir before exit, and emit `StepEvent` entries to `_events.jsonl` throughout execution. The Bridge reads results via `ParseContainerStepResult()` (or `ParseExtendedStepResult()` for enriched output) and tails events via `EventReader`. See [Step Mode](#step-mode-step_config) below.
>
> Agent containers execute with `NetworkMode: "none"` and have zero network access. Communication: environment variables in, exit code + `result.json` + `_events.jsonl` out (step mode) or exit code only out (agent mode).

## Overview

The agent runtime is the in-process Go engine inside the Bridge that manages task execution, conversation memory, tool dispatch, and result caching. It operates below the container lifecycle (which is handled by `pkg/studio` via Docker) and above the AI provider layer. When the Bridge receives a task through the Matrix control plane, the runtime takes over: it routes the request, executes tool calls through the executor, caches results, and tracks progress step by step.

The runtime is *not* the same thing as the agent state machine documented in [armorclaw.md Agent State Machine](armorclaw.md#agent-state-machine-go-bridge). That state machine tracks high-level agent phases like `BROWSING`, `FORM_FILLING`, and `AWAITING_APPROVAL`. This runtime handles the lower-level task loop: reasoning steps, tool invocations, speculative predictions, and memory persistence.

### Execution Modes

- **Mode A (Agent Studio)**: Containers spawned by `factory.Spawn()` with `NetworkMode: "none"`. Task delivered via `STEP_CONFIG` env var. Results via exit code + `result.json` in bind-mounted state dir (`/home/claw/.openclaw/`). When `STEP_CONFIG` is present, container runs in **step mode**: parses config, executes task, writes `result.json`, exits. When absent, runs in **agent mode** (Matrix polling loop). No network access. This is the default for secretary workflow steps.
- **Network-Isolated Execution**: Agent containers always run with `NetworkMode: "none"` and zero network access. Browser automation runs via the Jetski sidecar (separate container with network access, CDP proxy with PII scrubbing). LLM API calls are made by the Bridge process, not by agent containers. No proxy-out path exists for containers — all outbound network operations are handled by the Bridge or Jetski. See `jetski/` for the sidecar architecture.

## Architecture

```
                         ┌─────────────────────────────────────┐
                         │         Runtime                     │
                         │  (internal/agent/)                  │
                         │  Bridge-side only, not in container │
                         │                                     │
    Task config ────────▶│  Run(ctx, task)                     │
    (STEP_CONFIG env)    │    │                                │
                         │    ├─▶ Router.Route()               │
                         │    ├─▶ SpeculativeExecutor          │
                         │    │     (predictions)              │
                         │    ├─▶ ToolExecutor.Execute         │
                         │    │     (per tool call)            │
                         │    └─▶ build Result                 │
                         │                                     │
                         │  Wired at startup:                  │
                         │    Store  (memory)                  │
                         │    ToolCache (LRU + TTL)            │
                         └─────────────────────────────────────┘

    Bidirectional container communication (step mode):

    Task config (STEP_CONFIG env var) ──────▶ Container (NetworkMode: "none")
    Bridge polls Docker ContainerInspect ◀────── Container (exit code + result.json)
    Bridge tails _events.jsonl via EventReader ◀────── Container (StepEvent entries during execution)
```

## Step Mode (STEP_CONFIG)

When the Bridge sets `STEP_CONFIG` via `factory.go:Spawn()`, the container enters step mode instead of the default agent mode (Matrix polling loop). Step mode is the backward channel for Mode A containers.

**Flow:**
1. Bridge calls `factory.Spawn()` with `step.Config` → sets `STEP_CONFIG` env var
2. `entrypoint.py` detects `STEP_CONFIG` → imports `step_runner`
3. `step_runner.py` creates an `EventEmitter`, parses config via `step_config.py`, dispatches to a handler, emits events to `_events.jsonl`, writes `result.json`
4. Container exits (0 for success, 1 for failure)
5. Bridge's `waitForCompletion()` polls Docker (500ms interval) and tails `_events.jsonl` via `EventReader.ReadNew()`, then calls `ParseExtendedStepResult(stateDir)`

**Container-side modules** (in `container/openclaw/`):
- `step_config.py` — Parses `STEP_CONFIG` env var into `StepConfig` object. Provides `_blocker_response` and `relevant_skills` properties.
- `step_runner.py` — Executes step via handler (echo, transform, default), writes enriched result. Creates `EventEmitter` per step, merges blockers from config and events.
- `result_writer.py` — Atomic write of `result.json` (temp file + `os.rename()`)
- `events.py` — `EventEmitter` class. Writes `StepEvent` entries to `_events.jsonl`. Enforces `PIPE_BUF` (4096 bytes) atomic writes.

**result.json schema** (matches `ContainerStepResult` in `bridge/pkg/secretary/result.go`, enriched via `ExtendedStepResult`):
```json
{
  "status": "success",
  "output": "human-readable output",
  "data": {"key": "value"},
  "error": "error message if failed",
  "duration_ms": 1500,
  "_comments": ["optional annotations"],
  "_blockers": [{"blocker_type": "missing_input", "message": "...", "suggestion": "..."}],
  "_skill_candidates": [{"name": "...", "pattern_type": "...", "confidence": 0.7}],
  "_events_summary": {"total": 12, "types": {"step": 3, "command_run": 5}}
}
```

Base fields: `status` (string, required), `output` (string, required), `data` (map, omitempty), `error` (string, omitempty), `duration_ms` (int, required). Enriched underscore-prefixed fields: `_comments`, `_blockers`, `_skill_candidates`, `_events_summary`. Parsed by `ParseExtendedStepResult()` which also reads `_events.jsonl` for the full event list.

**Handlers:** Computation-only (no network). `echo` (for testing), `transform` (JSON-to-JSON), default (logs task received). Additional handlers require a separate plan with AI proxy socket.

### Observable Containers / Event Emission

Containers in step mode emit structured events to `_events.jsonl` throughout execution. This is implemented by the `EventEmitter` class in `events.py`.

**EventEmitter** (`container/openclaw/events.py`):

- Constructor takes a `state_dir` path and opens `_events.jsonl` for append.
- Each `emit()` call serializes a `StepEvent` dataclass as a single JSON line.
- `PIPE_BUF` (4096 bytes) enforcement: lines exceeding this limit are progressively truncated (detail replaced, then name shortened, then detail dropped). This guarantees atomic writes on Linux when the file is read concurrently by the Bridge.
- Convenience methods: `step()`, `file_read()`, `file_write()`, `file_delete()`, `command_run()`, `observation()`, `blocker()`, `error()`, `artifact()`, `progress()`, `checkpoint()`.
- `close()` writes a `_summary` event and closes the file handle.

**`_events.jsonl` format:** One JSON object per line. Schema matches Go `StepEvent` struct:

```json
{"seq": 1, "type": "step", "name": "processing data", "ts_ms": 500, "detail": {}, "duration_ms": null}
{"seq": 2, "type": "command_run", "name": "python transform.py", "ts_ms": 1200, "detail": {"exit_code": 0}, "duration_ms": 700}
```

**10 MB soft cap:** The Bridge's `EventReader` enforces a 10 MB limit. If `_events.jsonl` exceeds this, `ReadNew()` returns `ErrEventLogExceeded`. The calling code in `waitForCompletion()` logs a warning and stops tailing events, but does **not** kill the container. The container continues executing and finishes naturally via the normal Docker polling loop. After completion, `cleanupStateDir()` purges the oversized log. This soft cap preserves the container's output while preventing unbounded memory/disk growth from event tailing.

**Integration with StepRunner:** `StepRunner.run()` creates the `EventEmitter` as the first action and injects it into `step_config.config["_emitter_ref"]` so handlers can emit events without importing `events.py`. On completion, the runner reads `_events.jsonl` to extract blockers and event summaries for the enriched result.

### Blocker Protocol (Container Side)

Containers can signal that they need human input to proceed, distinct from the PII approval flow. Blockers handle missing input, ambiguous situations, or required decisions. PII approval gates access to sensitive data fields.

**Container signaling:**

1. The handler calls `emitter.blocker(blocker_type, message, suggestion, field)` to write a blocker event to `_events.jsonl`.
2. Alternatively, the handler appends to `step_config.config["_blockers"]` list.
3. On completion, `StepRunner` merges blockers from both sources into the enriched result.

**Bridge detection:**

1. `executeStepWithBlockerHandling()` in `orchestrator_integration.go` checks `ExtendedStepResult.Blockers` after the container exits.
2. If blockers are found, the workflow transitions to `StatusBlocked`.
3. The Bridge waits for a response via the `resolve_blocker` RPC or Matrix event.

**Container receives resolution:**

1. The Bridge calls `appendBlockerResponse()` to add `_blocker_response` to the step config.
2. `UnblockWorkflow()` transitions back to `StatusRunning`.
3. The container is re-spawned with the updated config.
4. The handler reads `step_config._blocker_response` property to get the response data (input, note, user_id, provided_at).

**PII safety distinction:** PII approval operates via `PendingApproval()` + `HandlePIIResponse()` with `app.armorclaw.pii_request`/`pii_response` Matrix events and a 120-second timeout. Blocker resolution operates via `waitForBlockerResponse()` + `DeliverBlockerResponse()` with a 10-minute timeout and up to 3 retries. Blocker input is never logged or written to disk, passed via environment variable only.

The `Runtime` struct in `internal/agent/runtime.go` holds all subsystems together:

| Field | Package | Role |
|-------|---------|------|
| `executor` | `internal/executor/` | Runs individual tool calls (shell, skills) |
| `cache` | `internal/cache/` | LRU cache for tool results with TTL eviction |
| `router` | `internal/router/` | Resolves which tools are available for a room/user |
| `memory` | `internal/memory/` | SQLite-backed message and context store |
| `speculative` | `internal/speculative/` | Pre-executes predicted tool calls |

The runtime connects to the container lifecycle like this:

1. **factory.Spawn** (in `pkg/studio`) creates a Docker container with the OpenClaw agent inside.
2. The Bridge sends a task to the agent via the Matrix room.
3. Inside the Bridge process, `Runtime.Run(ctx, task)` executes the task loop: route, predict, execute tools, collect results.
4. Steps accumulate on the `Task` object. Each step records its type (`reason`, `tool_call`, `tool_result`, `final`), duration, and output.
5. When the loop finishes (or hits `MaxSteps`), the runtime produces a `Result` with token usage, step count, and total duration.
6. **waitForCompletion** in the studio layer waits for the container to report back, then surfaces the `StepResult` to the Matrix room.

## Key Packages

### Runtime (internal/agent/)

**Files**: `runtime.go`, `types.go`

The runtime is the top-level coordinator. `NewRuntime(cfg)` wires together the executor, cache, router, memory store, and (optionally) the speculative executor. Defaults are applied for any zero-valued config fields.

**RuntimeConfig fields:**

| Field | Default | Purpose |
|-------|---------|---------|
| `MaxSteps` | 10 | Upper bound on reasoning steps per task |
| `MaxTokens` | 4096 | Token budget (checked against task usage) |
| `Timeout` | 30s | Per-task wall clock timeout |
| `EnableSpeculation` | true | Whether to pre-execute predicted tool calls |
| `MaxParallelTools` | 3 | Concurrency limit for tool execution |

**Task execution flow** (`Run` method):

1. Set task status to `running`, record start time, emit metrics.
2. Call `router.Route(roomID, userID)` to get available tools. If none, return immediately.
3. Loop up to `MaxSteps`:
   - Check for context cancellation.
   - Create a `reason` step and attach it to the task.
   - Generate predictions and feed them to the speculative executor.
   - Extract tool calls from the step.
   - Execute each tool call via `executor.Execute(ctx, call)`.
   - Record tool results as new steps on the task.
4. Build and return the final `Result`.

**Task model** (`types.go`):

A `Task` carries an ID, room ID, user ID, conversation history, step list, status, and metadata. Steps are typed: `reason`, `tool_call`, `tool_result`, `final`. Each step tracks its tool name, input, output, error, duration, and whether it was speculative.

Task statuses follow a simple lifecycle: `pending` -> `running` -> one of `completed`, `failed`, `cancelled`.

The `Result` struct captures the task ID, response text, tool call count, token usage breakdown (prompt / completion / total), wall clock duration, step count, and completion timestamp.

### Memory (internal/memory/)

**Files**: `store.go`, `checkpoint.go`, `batch.go`

The memory subsystem provides durable, per-room storage for conversation history and arbitrary key-value context. It backs onto SQLite (via the `modernc.org/sqlite` driver, no CGO required).

**Store** (`store.go`):

The `Store` wraps a `sql.DB` connection with a read-write mutex. It manages two tables:

- **`messages`**: Stores conversation turns keyed by room ID. Each message has an ID, role, content, timestamp, and serialized metadata. Queries are indexed on `room_id`. `GetMessages(roomID, limit)` returns messages in chronological order (the query fetches `DESC`, then the results are reversed in Go).
- **`contexts`**: A key-value store scoped to room ID. Uses `ON CONFLICT` upsert for `SetContext`. Supports get-by-key, get-all-for-room, delete, and full room clearing.

Both tables record metrics through `metrics.RecordMemoryOperation()` for every mutation and query.

`PruneMessages(olderThan)` deletes messages older than the given duration. `ClearRoom(roomID)` wipes all messages and context for a room.

**Checkpointer** (`checkpoint.go`):

Runs a background goroutine that issues `PRAGMA wal_checkpoint(TRUNCATE)` on a configurable interval (default: 5 minutes). On close, it performs one final checkpoint before exiting. This keeps the WAL file from growing unbounded under heavy write loads.

**BatchWriter** (`batch.go`):

Buffers messages in memory and flushes them to the store in batches. This reduces write amplification when the runtime is processing many messages in quick succession.

- **`maxBatch`**: Default 100. When the pending buffer reaches this size, a flush triggers immediately.
- **`interval`**: Default 1 second. A ticker also triggers periodic flushes.
- **`flushChan`**: A non-blocking signal channel. `Add(msg)` pushes to the buffer and signals the flush goroutine if the batch is full.

On close (`Close()`), the stop channel is closed and the goroutine performs a final flush before exiting.

### Cache (internal/cache/)

**Files**: `lru.go`, `ratelimit.go`

**LRU cache** (`lru.go`):

A generic least-recently-used cache with TTL-based expiration. Uses Go's `container/list` for eviction ordering and a map for O(1) lookups.

- **`maxSize`**: Default 1000 entries. When full, the oldest entry is evicted.
- **`defaultTTL`**: Default 5 minutes. Each entry tracks its `expiresAt` timestamp.
- **`onEvict`**: Optional callback fired when an entry is removed (by eviction, deletion, or clear).

`Get(key)` promotes the entry to the front of the eviction list. If the entry is expired, it is removed and returns `nil, false`.

`GetOrCompute(key, compute)` is a convenience that returns the cached value if present, otherwise calls the `compute` function and caches the result. This is the primary access pattern for tool results.

`PurgeExpired()` scans all entries and removes those past their TTL. Returns the count of purged entries.

**Rate limiter** (`ratelimit.go`):

A per-key rate limiter built on `golang.org/x/time/rate`. Each key gets its own token-bucket limiter, lazily created on first access.

- **`Rate`**: Default 10 tokens/second.
- **`Burst`**: Default 20 tokens.

`Allow(key)` is non-blocking: returns true if the request can proceed. `Wait(key)` blocks until a token is available. `WaitTimeout(key, timeout)` adds a deadline.

`SetRate` and `SetBurst` update the configuration for all existing limiters. `Remove(key)` cleans up a limiter that is no longer needed.

### Tool Executor (internal/executor/)

**File**: `engine.go`

The tool executor dispatches tool calls to their implementations. It wraps execution in a worker pool for concurrency control and routes calls through the security gateway.

**ToolExecutor** struct:

| Field | Purpose |
|-------|---------|
| `pool` | Worker pool that limits concurrent executions |
| `petg` | Security gateway that validates and filters tool calls |
| `skills` | Registry that resolves skill names to executable definitions |
| `timeout` | Per-call timeout (default 30s) |

**Execution flow:**

1. Look up the tool name in the skill registry. If unknown, reject immediately.
2. Pass the call through the security gateway (`petg.ValidateToolCall`) for policy checks.
3. Submit the call to the worker pool with a timeout context.
4. Record metrics (success or error) with timing.

**Worker pool** (`ToolPool`):

The pool starts with `MaxWorkers/2` goroutines (minimum 5 by default). Each worker reads from a buffered task channel. `ExecuteBatch` runs multiple calls concurrently and collects all results. The pool is closed by closing the task channel and waiting for workers to drain.

**Shell tool**: The only built-in tool is `shell`, which runs commands via `exec.CommandContext`. Output goes through `petg.FilterOutput` to strip any PII that might have leaked into tool output.

### Speculative Execution (internal/speculative/)

**File**: `executor.go`

The speculative executor pre-runs tool calls that the runtime predicts the task will need. This hides latency: when the actual tool call arrives, the result is already cached.

> **Note on Container Isolation**: Speculative execution pre-computes Go-side tool call results. However, the actual agent work happens inside containers with `NetworkMode: "none"` and no network access. The speculative cache is useful for Go-side operations (keystore lookups, approval checks) but cannot pre-compute results for container-internal LLM calls or browser operations, since those execute in isolation.

**SpeculativeExecutor** struct:

| Field | Purpose |
|-------|---------|
| `executor` | The underlying `ToolExecutor` for actual execution |
| `cache` | LRU cache for predicted results (default 1000 entries, 5 min TTL) |
| `predictions` | Map of call ID to prediction timestamp |
| `results` | Map of call ID to pre-computed `ToolResult` |
| `pendingCalls` | Queue of calls queued for prediction |

**Predict(ctx, call)**:

1. Check if a result is already cached for this call ID. If so, return it immediately (cache hit, metrics recorded).
2. If no cached result, execute the call through the underlying executor.
3. Store the result and record the prediction timestamp.
4. Return the result.

`AddPredictions(calls)` queues calls for prediction. The runtime calls this after generating predictions from the current step's route result.

`ExecuteBatch(ctx, calls)` runs predictions for multiple calls concurrently. Each call goes through `Predict`, and results are collected. If any prediction fails, `ErrPredictionFailed` is returned.

`ClearPredictions()` wipes all cached predictions, results, and pending calls. Called on close.

## Configuration

All configuration flows through `RuntimeConfig` in `internal/agent/runtime.go`. The runtime is created in the Bridge startup sequence and lives for the lifetime of the process.

| Parameter | Config Field | Default | Notes |
|-----------|-------------|---------|-------|
| Max steps per task | `RuntimeConfig.MaxSteps` | 10 | Prevents runaway reasoning loops |
| Max tokens per task | `RuntimeConfig.MaxTokens` | 4096 | Token budget for the full task |
| Task timeout | `RuntimeConfig.Timeout` | 30s | Wall clock limit per task |
| Speculative execution | `RuntimeConfig.EnableSpeculation` | true | Set false to disable prediction |
| Parallel tool limit | `RuntimeConfig.MaxParallelTools` | 3 | Worker pool size for tool calls |
| Tool cache size | Hardcoded in `NewRuntime` | 500 entries | ToolCache max size |
| Tool cache TTL | Hardcoded in `NewRuntime` | 10 min | ToolCache entry TTL |
| Memory DB path | `StoreConfig.Path` | `:memory:` | Set to file path for persistence |
| Checkpoint interval | `CheckpointerConfig.Interval` | 5 min | WAL checkpoint frequency |
| Batch write size | `BatchWriterConfig.MaxBatch` | 100 | Messages buffered before flush |
| Batch write interval | `BatchWriterConfig.Interval` | 1s | Periodic flush cadence |
| LRU cache size | `LRUConfig.MaxSize` | 1000 | Max cached entries |
| LRU cache TTL | `LRUConfig.DefaultTTL` | 5 min | Default entry lifetime |
| Rate limit (tokens/s) | `RateLimitConfig.Rate` | 10 | Per-key token bucket rate |
| Rate limit burst | `RateLimitConfig.Burst` | 20 | Per-key burst capacity |

## Integration Points

### Container Lifecycle

The runtime sits between the container factory and the Matrix control plane. The sequence:

1. `pkg/studio` calls `factory.Spawn()` to create an isolated Docker container running OpenClaw.
2. The Bridge creates a `Task` with the room ID, user ID, and conversation.
3. `Runtime.Run(ctx, task)` executes the task loop internally (routing, tool calls, speculation).
4. Steps accumulate on the task. Results flow back through the studio layer.
5. `waitForCompletion` blocks until the container reports its final `StepResult`.
6. The result is surfaced to the Matrix room as a message.

### Bridge Observable States vs State Machine States

The Bridge can observe four container states via Docker `ContainerInspect` and `_events.jsonl`:

| Bridge-Observable State | How Detected |
|------------------------|-------------|
| **Running** | Container exists and `State.Running == true` |
| **Completed** | Container exited with code 0 |
| **Failed** | Container exited with non-zero code, or container gone |
| **Events** | `StepEvent` entries in `_events.jsonl` (step, file ops, commands, observations, blockers, errors) |

The 11-state agent state machine (`IDLE`, `INITIALIZING`, `BROWSING`, `FORM_FILLING`, `AWAITING_CAPTCHA`, `AWAITING_2FA`, `AWAITING_APPROVAL`, `PROCESSING_PAYMENT`, `ERROR`, `COMPLETE`, `OFFLINE`) is defined in `bridge/pkg/agent/state.go` but transitions are **programmatic only**: triggered by Bridge-side code, not by agent-reported events. The `BroadcastStatus()` method that would relay states to clients is currently unimplemented and returns `fmt.Errorf("BroadcastStatus: agent status broadcasting not implemented — no container-to-Bridge state reporting channel exists")`.

For the agent state machine definition (states like `BROWSING`, `AWAITING_APPROVAL`, `PROCESSING_PAYMENT`), see the [Agent State Machine section in armorclaw.md](armorclaw.md#agent-state-machine-go-bridge).

### Security Gateway (PETG)

Tool calls pass through `petg.ValidateToolCall(ctx, toolName, args)` before execution. This checks policies, rate limits, and PII interception rules. Tool output is filtered through `petg.FilterOutput()` to strip any secrets that leaked into results.

### Metrics

All subsystems emit metrics via `internal/metrics`:
- `RecordTaskStart` / `RecordTaskComplete` for task lifecycle
- `RecordToolCall(name, status, duration)` for tool execution
- `RecordMemoryOperation(op)` for store operations (insert, select, upsert, delete)
- `RecordSpeculativeCall(status)` for prediction hits and misses
- `RecordCacheHit` / `RecordCacheMiss` for cache effectiveness

### Memory Store

The memory store is shared across the runtime. Conversations and per-room context persist between tasks. The `BatchWriter` handles high-throughput ingestion, while the `Checkpointer` keeps the WAL file compact. For long-running agents, `PruneMessages` can be called periodically to cap storage growth.

### Cache Layer

The `ToolCache` (an LRU instance) sits in front of the executor. Tool results that are expensive to compute but deterministic can be served from cache. The speculative executor maintains its own separate LRU for predicted results, preventing cache pollution from speculative calls that never materialize.
