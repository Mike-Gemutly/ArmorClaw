# Secretary Logic Implementation Plan

## TL;DR

> **Quick Summary**: Implement proactive automation system with template engine, cron-based scheduling, and Bridge backend execution for ArmorChat Android app.
> 
> **Deliverables**:
> - Template engine with 5 operators (eq, neq, in, nin, contains)
> - IANA timezone-aware scheduling (Bridge-side)
> - Secretary CRUD RPCs on Bridge
> - Blockly-based template editor UI (Android)
> - Template list/management screen
> - Execution history/logs
> 
> **Estimated Effort**: Large (2-3 weeks)
> **Parallel Execution**: YES - 4 waves
> **Critical Path**: Domain Models → RPC Interface → Android UI → Integration Tests

---

## Context

### Original Request
Implement "Secretary Logic" - a proactive automation system for ArmorChat with template engine, scheduling, and CRUD RPCs.

### Interview Summary

**Key Discussions**:
- **Operators**: 5 core (eq, neq, in, nin, contains)
- **Data Sources**: Message metadata, User/contact state, Time-based conditions, System/external data
- **Actions**: Send message, Push notification, API/webhook call, Execute workflow
- **Execution**: Bridge backend required (Android UI, Bridge handles scheduling/execution)
- **Persistence**: Yes - survive app restarts
- **RPC Interface**: CRUD + List/filter + Toggle + Execution history
- **UI**: Blockly-based editor + Both visual schedule builder AND cron expression input
- **Test Strategy**: TDD (RED-GREEN-REFACTOR)
- **Scope**: AI/ML, Template sharing, Multi-user collab EXCLUDED

### Metis Review

**Identified Gaps** (addressed):
- **Bridge backend scope**: CONFIRMED - Bridge has scheduling/action execution capabilities
- **Timezone handling**: CONFIRMED - UTC storage, device timezone display, IANA support
- **Failure handling**: CONFIRMED - Queue + Retry (3x, exponential backoff) + User notification
- **Resource limits**: CONFIRMED - 100 templates/user, 10 concurrent, 30s timeout, 1000 history entries

---

## Work Objectives

### Core Objective
Implement a complete proactive automation system enabling users to create templates with conditions and scheduled actions, executed by the Bridge backend.

### Concrete Deliverables
- `shared/domain/model/SecretaryTemplate.kt` - Template domain model
- `shared/domain/model/TemplateCondition.kt` - Condition model with 5 operators
- `shared/domain/model/TemplateAction.kt` - Action model (4 action types)
- `shared/domain/model/ExecutionHistory.kt` - Execution log model
- `shared/domain/repository/SecretaryRepository.kt` - Repository interface
- `shared/platform/bridge/SecretaryRpcClient.kt` - RPC client interface
- `androidApp/.../platform/SecretaryRpcClientImpl.kt` - RPC implementation
- `androidApp/.../data/SecretaryRepositoryImpl.kt` - Repository implementation
- `androidApp/.../viewmodels/SecretaryViewModel.kt` - ViewModel
- `androidApp/.../screens/secretary/SecretaryScreen.kt` - Template list screen
- `androidApp/.../screens/secretary/TemplateEditorScreen.kt` - Blockly editor screen
- `androidApp/.../screens/secretary/SchedulePickerScreen.kt` - Schedule picker UI
- `androidApp/.../components/secretary/TemplateCard.kt` - Template card component
- `androidApp/.../components/secretary/ConditionBlock.kt` - Blockly condition block
- `androidApp/.../components/secretary/ActionBlock.kt` - Blockly action block
- `androidApp/.../navigation/AppNavigation.kt` - Add secretary routes

### Definition of Done
- [ ] All 5 operators work correctly (eq, neq, in, nin, contains)
- [ ] Cron expressions parse and execute at correct times
- [ ] IANA timezone handling works correctly
- [ ] Template CRUD operations work via RPC
- [ ] Execution history logged and viewable
- [ ] Blockly editor allows creating/editing templates
- [ ] Visual schedule builder works alongside cron input
- [ ] TDD: All tests pass (unit + integration)
- [ ] Build succeeds: `./gradlew assembleDebug`
- [ ] Tests pass: `./gradlew test`

### Must Have
- Template engine with 5 operators: eq, neq, in, nin, contains
- Bridge-side scheduling with IANA timezone support
- Secretary CRUD RPCs on Bridge
- Blockly-based template editor UI
- Visual schedule builder + cron expression input
- Execution history logging
- TDD workflow

### Must NOT Have (Guardrails)
- NO AI/ML-based template suggestions
- NO template marketplace/sharing features
- NO multi-user collaboration on templates
- NO template versioning system (edit-and-save only)
- NO custom scripting/code evaluation in templates
- NO complex condition operators beyond the 5 core
- NO analytics/dashboards beyond basic execution history
- NO modifications to existing BridgeRpcClient interface (add new RPCs only)
- NO changes to existing AgentBlocks (extend, don't modify)
- NO SQLite schema changes in shared module (Bridge handles storage)

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: YES (TDD)
- **Framework**: Kotlin Test 1.9.20 + JUnit 4.13.2 + Mockk 1.13.8 + Turbine 1.0.0
- **If TDD**: Each task follows RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Frontend/UI**: Use Playwright (playwright skill) — Navigate, interact, assert DOM, screenshot
- **TUI/CLI**: Use interactive_bash (tmux) — Run command, send keystrokes, validate output
- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields
- **Library/Module**: Use Bash (gradlew test) — Import, call functions, compare output

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — Foundation + Models):
├── Task 1: Domain models + data classes [quick]
├── Task 2: Repository interface [quick]
├── Task 3: RPC client interface [quick]
├── Task 4: Blockly block definitions [quick]
├── Task 5: Navigation routes [quick]
└── Task 6: Test file scaffolding [quick]

Wave 2 (After Wave 1 — Core Logic, MAX PARALLEL):
├── Task 7: Template condition evaluator (depends: 1) [deep]
├── Task 8: Cron expression parser (depends: 1) [deep]
├── Task 9: RPC client implementation (depends: 2, 3) [unspecified-high]
├── Task 10: Repository implementation (depends: 1, 2) [unspecified-high]
├── Task 11: ViewModel implementation (depends: 9, 10) [unspecified-high]
└── Task 12: Failure handling logic (depends: 1, 7) [unspecified-high]

└── Task 13: Execution history logger (depends: 1) [unspecified-high]

Wave 3 (After Wave 2 — UI Components):
├── Task 14: TemplateCard component (depends: 11) [visual-engineering]
├── Task 15: ConditionBlock component (depends: 4) [visual-engineering]
├── Task 16: ActionBlock component (depends: 4) [visual-engineering]
├── Task 17: SchedulePickerDialog component (depends: 8) [visual-engineering]
└── Task 18: TemplateEditorLayout component (depends: 15, 16) [visual-engineering]

Wave 4 (After Wave 3 — Screens + Integration):
├── Task 19: SecretaryScreen (depends: 14, 11) [visual-engineering]
├── Task 20: TemplateEditorScreen (depends: 18, 11) [visual-engineering]
├── Task 21: SchedulePickerScreen (depends: 17, 8) [visual-engineering]
├── Task 22: Integration tests (depends: 19, 20, 21) [deep]
└── Task 23: E2E QA scenarios (depends: 22) [unspecified-high]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 1 → Task 7 → Task 9 → Task 11 → Task 19 → Task 22
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 6 (Waves 1 & 2)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|------------|--------|
| 1 | — | 7, 8, 12, 13 |
| 2 | — | 9, 10 |
| 3 | — | 9 |
| 4 | — | 15, 16 |
| 5 | — | 19, 20, 21 |
| 6 | — | ALL (test scaffolding) |
| 7 | 1 | 12 |
| 8 | 1 | 17, 21 |
| 9 | 2, 3 | 11 |
| 10 | 1, 2 | 11 |
| 11 | 9, 10 | 14, 19, 20, 21 |
| 12 | 1, 7 | 13 |
| 13 | 1 | — | 14 | 19, 20, 21 |
| 14 | 11 | 19 |
| 15 | 4 | 18 |
| 16 | 4 | 18 |
| 17 | 8 | 21 |
| 18 | 15, 16 | 20 |
| 19 | 11, 14 | 22 |
| 20 | 11, 18 | 22 |
| 21 | 8, 17 | 22 |
| 22 | 19, 20, 21 | 23 |
| 23 | 22 | F1-F4 |

### Agent Dispatch Summary

- **Wave 1**: **6** — T1-T4 → `quick`, T5 → `quick`, T6 → `quick`
- **Wave 2**: **7** — T7-T8 → `deep`, T9-T13 → `unspecified-high`
- **Wave 3**: **5** — T14-T18 → `visual-engineering`
- **Wave 4**: **5** — T19-T21 → `visual-engineering`, T22 → `deep`, T23 → `unspecified-high`
- **FINAL**: **4** — F1 → `oracle`, F2-F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> > EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.
> > **A task WITHOUT QA Scenarios is INCOMPLETE. No exceptions.**

- [ ] 1. **Create domain models for SecretaryTemplate and TemplateCondition, TemplateAction**

  **What to do**:
  - Create `SecretaryTemplate` data class in `shared/domain/model/SecretaryTemplate.kt`
  - Define fields: `id`, `name`, `description`, `conditions: List<TemplateCondition>`, `actions: List<TemplateAction>`, `schedule: String` (cron expression), `timezone: String` (IANA timezone ID), `active: Boolean`, `createdAt`, `updatedAt`
  - Create `TemplateCondition` sealed class in `shared/domain/model/TemplateCondition.kt`
  - Define operators enum: `EQ`, `NEQ`, `IN`, `NIN`, `CONTAINS`
  - Define fields: `field`, `operator`, `value`
  - Create `TemplateAction` sealed class in `shared/domain/model/TemplateAction.kt`
  - Define action type enum: `SEND_MESSAGE`, `PUSH_NOTIFICATION`, `WEBHOOK`, `EXECUTE_WORKFLOW`
  - Define data class for each action type with appropriate fields

  **Must NOT do**:
  - No PII in domain models (no user emails, message content)
  - No implementation logic (data classes only)
  - No null/undefined values (use sensible defaults)

  **Recommended Agent Profile**:
  > Select category + skills based on task domain. Justify each choice.
  - **Category**: `quick` — Domain models are simple data classes
      - Reason: Foundation for all subsequent work
    - **Skills**: [] — No special skills needed for data classes
    - **Skills Evaluated but Omitted**:
      - `frontend-ui-ux`: Not UI work
      - `visual-engineering`: Not visual work
      - `ultrabrain`: No complex logic needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2-6)
  - **Blocks**: Tasks 7-22 (all downstream tasks)
  - **Blocked By**: None (can start immediately)

  **References** (CRITICAL - Be Exhaustive):

  > The executor has NO context from your interview. References are their ONLY guide.
  > Each reference must answer: "What should I look at and WHY?"

  **Pattern References** (existing code to follow):
  - `shared/domain/model/Message.kt:1-50` - Message structure with id, content, timestamp pattern
  - `shared/domain/model/BrowserEvents.kt:1-100` - Event structure with type, data pattern

  **WHY Each Reference Matters**:
  - `Message.kt`: Use this structure for template model - id, timestamps, serialization pattern
  - `BrowserEvents.kt`: Use the pattern for action data classes - type field, data structure

  **Acceptance Criteria**:

  > **AGENT-EXECUTABLE VERIFICATION ONLY** — No human action permitted.
  > Every criterion must be verifiable by running a command or using a tool.

      **If TDD (tests enabled):**
      - [ ] Test file created: `shared/domain/model/SecretaryTemplate.test.kt`
      - [ ] Test file created: `shared/domain/model/TemplateCondition.test.kt`
      - [ ] Test file created: `shared/domain/model/TemplateAction.test.kt`
      - [ ] `./gradlew test --tests "com.armorclaw.shared.domain.model.*"` → PASS (3 test files, 0 failures)

      **QA Scenarios**:

      ```
      Scenario: Create template with valid data
        Tool: Bash (./gradlew test)
        Preconditions: SecretaryTemplate, TemplateCondition, TemplateAction classes exist
        Steps:
          1. Run `./gradlew test --tests "com.armorclaw.shared.domain.model.SecretaryTemplateTest"`
          2. Assert tests pass
        Expected Result: All 3 tests pass (template, condition, action)
        Failure Indicators: Any test fails
    Evidence: .sisyphus/evidence/task-1-model.test

      ```
      Scenario: Template condition operators work correctly
        Tool: Bash (./gradlew test)
        Preconditions: TemplateCondition class exists
        Steps:
          1. Run `./gradlew test --tests "com.armorclaw.shared.domain.model.TemplateConditionTest"`
          2. Test EQ operator with "hello" and "world"
          3. Test NEQ operator with "hello" and "world"
          4. Test IN operator with "foo" in ["foo", "bar"]
          5. Test NIN operator with "foo" in ["bar", "baz"]
          6. Test CONTAINS operator with "hello" and "hello world"
        Expected Result: All operator tests pass
        Failure Indicators: Any operator test fails
    Evidence: .sisyphus/evidence/task-1-operators.test
      ```

  **Evidence to Capture:**
  - [ ] Test files created (3 files)
  - [ ] Test output: `.sisyphus/evidence/task-1-*.test` files

---

- [ ] 2. **Create repository interface**

  **What to do**:
  - Create `SecretaryRepository` interface in `shared/domain/repository/SecretaryRepository.kt`
  - Define CRUD methods: `createTemplate`, `getTemplate`, `updateTemplate`, `deleteTemplate`
  - Define `listTemplates` method with filtering (status, search)
  - Define `getExecutionHistory` method with pagination
  - Define `toggleTemplate` method

  **Must NOT do**:
  - No implementation (interface only)
  - No Android-specific code (shared module)

  **Recommended Agent Profile**:
  - **Category**: `quick` — Interface definition, minimal logic
    - Reason: Simple interface with clear method signatures
  - **Skills**: [] — No special skills needed
  - **Skills Evaluated but Omitted**:
    - `frontend-ui-ux`: Not UI work
    - `visual-engineering`: Not visual work

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-6)
  - **Blocks**: Tasks 9, 10, 12, 13
  - **Blocked By**: None (can start immediately)

  **References**:
  - Pattern: `shared/domain/repository/MessageRepository.kt` - Repository interface pattern
  - Pattern: `shared/domain/repository/BrowserWorkflowRepository.kt` - Repository with filtering

  **WHY Each Reference Matters**:
  - `MessageRepository.kt`: Follow method signature patterns, return types
  - `BrowserWorkflowRepository.kt`: Follow filtering pattern with status, search parameters

  **Acceptance Criteria**:

  > **AGENT-EXECUTABLE VERIFICATION ONLY** — No human action permitted.

      **If TDD (tests enabled):**
      - [ ] Test file created: `shared/domain/repository/SecretaryRepository.test.kt`
      - [ ] `./gradlew test --tests "com.armorclaw.shared.domain.repository.*"` → PASS (compiles)

      **QA Scenarios**:
      - [ ] N/A (interface only, no runtime QA needed)

  **Evidence to Capture:**
  - [ ] Interface file created

---

- [ ] 3. **Create RPC client interface**

  **What to do**:
  - Create `SecretaryRpcClient` interface in `shared/platform/bridge/SecretaryRpcClient.kt`
  - Define RPC methods matching backend endpoints:
    - `createTemplate(template: SecretaryTemplate): SecretaryTemplate`
    - `getTemplate(id: String): SecretaryTemplate?`
    - `updateTemplate(template: SecretaryTemplate): SecretaryTemplate`
    - `deleteTemplate(id: String): Boolean`
    - `listTemplates(filter: TemplateFilter): List<SecretaryTemplate>`
    - `toggleTemplate(id: String, active: Boolean): Boolean`
    - `getExecutionHistory(templateId: String?, page: Int, limit: Int): PaginatedResult<ExecutionHistory>`
  - Define `TemplateFilter` data class with status, search, timezone fields
  - Define `PaginatedResult<T>` data class with items, totalCount, hasMore fields

  **Must NOT do**:
  - No implementation (interface only)
  - No Android-specific code (shared module)
  - No modifications to existing BridgeRpcClient interface (add new RPCs only)

  **Recommended Agent Profile**:
  - **Category**: `quick` — Interface definition, minimal logic
    - Reason: Interface with clear method signatures, following existing RPC patterns
  - **Skills**: [] — No special skills needed
  - **Skills Evaluated but Omitted**:
    - `frontend-ui-ux`: Not UI work
    - `visual-engineering`: Not visual work

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-6)
  - **Blocks**: Tasks 9, 10, 11, 12, 13
  - **Blocked By**: None (can start immediately)

  **References**:
  - Pattern: `shared/platform/bridge/BridgeRpcClient.kt:10-50` - RPC method patterns
  - Pattern: `shared/platform/bridge/RpcModels.kt` - RPC model patterns

  **WHY Each Reference Matters**:
  - `BridgeRpcClient.kt`: Follow method naming patterns, parameter types, return types
  - `RpcModels.kt`: Follow data class structure for filter/result types

  **Acceptance Criteria**:

  > **AGENT-EXECUTABLE VERIFICATION ONLY** — No human action permitted.

      **If TDD (tests enabled):**
      - [ ] Test file created: `shared/platform/bridge/SecretaryRpcClient.test.kt`
      - [ ] `./gradlew test --tests "com.armorclaw.shared.platform.bridge.*"` → PASS (compiles)

      **QA Scenarios**:
      - [ ] N/A (interface only, no runtime QA needed)

  **Evidence to Capture:**
  - [ ] Interface file created
  - [ ] Filter/Result data classes created

---

- [ ] 4. **Add Blockly block definitions**

  **What to do**:
  - Add condition blocks to `shared/ui/components/secretary/ConditionBlocks.kt`:
    - `MessageContainsBlock` - Check if message contains text
    - `MessageFromBlock` - Check if message is from specific sender
    - `TimeOfDayBlock` - Check current time against value
    - `ContactStatusBlock` - Check contact online/offline status
    - `CustomConditionBlock` - Generic condition with operator dropdown
  - Add action blocks in `shared/ui/components/secretary/ActionBlocks.kt`:
    - `SendMessageBlock` - Configure message recipient and content
    - `PushNotificationBlock` - Configure notification title and body
    - `WebhookBlock` - Configure webhook URL and payload
    - `ExecuteWorkflowBlock` - Select workflow to execute

  **Must NOT do**:
  - Modify existing AgentBlocks.kt (extend only)
  - Add complex condition logic (keep simple)
  - Add UI rendering logic (data classes only)

  **Recommended Agent Profile**:
  - **Category**: `quick` — Block definitions, minimal logic
    - Reason: Simple data classes following existing block patterns
  - **Skills**: [`frontend-ui-ux`] — UI component patterns
    - `frontend-ui-ux`: Block definitions follow Compose patterns
  - **Skills Evaluated but Omitted**:
    - `visual-engineering`: Blocks are simple, no complex visual design
    - `ultrabrain`: No complex logic needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-6)
  - **Blocks**: Tasks 15, 16, 18
  - **Blocked By**: None (can start immediately)

  **References**:
  - Pattern: `shared/src/androidMain/kotlin/com/armorclaw/app/studio/AgentBlocks.kt:1-500` - Existing Blockly block definitions
  - Pattern: `shared/ui/components/` - UI component patterns

  **WHY Each Reference Matters**:
  - `AgentBlocks.kt`: Follow existing block type structure, serialization format
  - `shared/ui/components/`: Follow component organization pattern

  **Acceptance Criteria**:

  > **AGENT-EXECUTABLE VERIFICATION ONLY** — No human action permitted.

      **If TDD (tests enabled):**
      - [ ] Test files created for condition and action blocks
      - [ ] `./gradlew test --tests "com.armorclaw.shared.ui.components.secretary.*"` → PASS

      **QA Scenarios**:
      - [ ] N/A (block definitions compile, no runtime QA)

  **Evidence to Capture:**
  - [ ] Block files created (2 files)

---

- [ ] 5. **Add navigation routes**

  **What to do**:
  - Add secretary routes to `androidApp/.../navigation/AppNavigation.kt`:
    - `secretary` - Template list screen
    - `secretary/edit` - Template editor screen (with templateId argument)
    - `secretary/schedule` - Schedule picker screen (with templateId argument)
    - `secretary/history` - Execution history screen (with templateId argument)
  - Register routes in navigation graph

  **Must NOT do**:
  - Modify existing route logic
  - Add bottom navigation items (separate task)
  - Break existing navigation patterns

  **Recommended Agent Profile**:
  - **Category**: `quick` — Route definitions, minimal logic
    - Reason: Simple route additions following existing patterns
  - **Skills**: [] — No special skills needed
  - **Skills Evaluated but Omitted**:
    - `frontend-ui-ux`: Not UI work
    - `visual-engineering`: Not visual work

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-6)
  - **Blocks**: Tasks 19, 20, 21
  - **Blocked By**: None (can start immediately)

  **References**:
  - Pattern: `androidApp/.../navigation/AppNavigation.kt:1-100` - Existing route definitions
  - Pattern: `androidApp/.../navigation/AppNavigation.kt:200-300` - Navigation graph setup

  **WHY Each Reference Matters**:
  - `AppNavigation.kt`: Follow route definition pattern, composable signature, navigation graph registration

  **Acceptance Criteria**:

  > **AGENT-EXECUTABLE VERIFICATION ONLY** — No human action permitted.

      **If TDD (tests enabled):**
      - [ ] Test file created: `androidApp/.../navigation/SecretaryNavigation.test.kt`
      - [ ] `./gradlew test --tests "com.armorclaw.app.navigation.*"` → PASS

      **QA Scenarios**:
      - [ ] N/A (route definitions compile, no runtime QA)

  **Evidence to Capture:**
  - [ ] Navigation file updated

---

- [ ] 6. **Create test file scaffolding**

  **What to do**:
  - Create test utilities in `shared/domain/model/TestUtils.kt`:
    - `createTestTemplate()` - Factory for test templates
    - `createTestCondition()` - Factory for test conditions
    - `createTestAction()` - Factory for test actions
    - `createTestExecutionHistory()` - Factory for test history
  - Create test fixtures file in `shared/domain/model/TestFixtures.kt`
  - Add common test data sets for templates, conditions, actions

  **Must NOT do**:
  - Add production code to test utilities
  - Add flaky test logic
  - Create integration tests (unit tests only)

  **Recommended Agent Profile**:
  - **Category**: `quick` — Test utilities, minimal logic
    - Reason: Simple factory functions for test data
  - **Skills**: [] — No special skills needed
  - **Skills Evaluated but Omitted**:
    - `frontend-ui-ux`: Not UI work
    - `visual-engineering`: Not visual work

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-6)
  - **Blocks**: All test tasks
  - **Blocked By**: None (can start immediately)

  **References**:
  - Pattern: `shared/domain/model/MessageTest.kt` - Test file pattern
  - Pattern: `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/SyncStatusViewModelTest.kt` - ViewModel test pattern

  **WHY Each Reference Matters**:
  - `MessageTest.kt`: Follow test file structure, assertions, mocking patterns
  - `SyncStatusViewModelTest.kt`: Follow Turbine patterns for Flow testing

  **Acceptance Criteria**:

  > **AGENT-EXECUTABLE VERIFICATION ONLY** — No human action permitted.

      **If TDD (tests enabled):**
      - [ ] Test files created: `TestUtils.kt`, `TestFixtures.kt`
      - [ ] `./gradlew test --tests "com.armorclaw.shared.domain.model.*Test*"` → PASS

      **QA Scenarios**:
      - [ ] N/A (test utilities compile, no runtime QA)

  **Evidence to Capture:**
  - [ ] Test files created (2 files)

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPRO. Rejection → fix → re-run.

---

- [ ] 7. **Implement template condition evaluator**

  **What to do**:
  - Create `TemplateConditionEvaluator` class in `shared/domain/logic/TemplateConditionEvaluator.kt`
  - Implement `evaluate(condition: TemplateCondition, context: EvaluationContext): Boolean`
  - Implement all 5 operators: `EQ`, `NEQ`, `IN`, `NIN`, `CONTAINS`
  - Add `EvaluationContext` data class with messageData, userData, timeData, externalData

  **Must NOT do**:
  - No slow operations (keep evaluation <100ms)
  - No logging of sensitive data

  **Recommended Agent Profile**:
  - **Category**: `deep` — Core logic with algorithmic complexity
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 2, depends on Task 1)

  **References**: 
  - `shared/domain/model/TemplateCondition.kt` - Condition structure

  **Acceptance Criteria**:
  - [ ] Test file created: `shared/domain/logic/TemplateConditionEvaluator.test.kt`
  - [ ] `./gradlew test --tests "*.TemplateConditionEvaluatorTest"` → PASS

  **QA Scenarios**:
  - Test EQ operator with matching/non-matching values
  - Test IN/NIN operators with arrays
  - Test CONTAINS operator case-insensitive

  **Evidence**: `.sisyphus/evidence/task-7-*.test`

---

- [ ] 8. **Implement cron expression parser**

  **What to do**:
  - Add `krontab` library dependency to `shared/build.gradle.kts`
  - Create `CronExpressionParser` class in `shared/domain/logic/CronExpressionParser.kt`
  - Implement `parse(expression: String): CronSchedule`
  - Implement `getNextRunTime(schedule: CronSchedule, timezone: String): Instant`
  - Handle IANA timezone conversion using kotlinx-datetime

  **Must NOT do**:
  - No custom cron parser (use krontab library)

  **Recommended Agent Profile**:
  - **Category**: `deep` — Core logic with datetime complexity
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 2, depends on Task 1)

  **References**:
  - External: `krontab` library documentation
  - External: `kotlinx-datetime` documentation

  **Acceptance Criteria**:
  - [ ] Test file created: `shared/domain/logic/CronExpressionParser.test.kt`
  - [ ] `./gradlew test --tests "*.CronExpressionParserTest"` → PASS

  **QA Scenarios**:
  - Parse valid cron expressions
  - Get next run time with timezone
  - Handle DST transitions

  **Evidence**: `.sisyphus/evidence/task-8-*.test`

---

- [ ] 9. **Implement RPC client**

  **What to do**:
  - Create `SecretaryRpcClientImpl` class in `androidApp/.../platform/SecretaryRpcClientImpl.kt`
  - Implement all RPC methods using existing BridgeWebSocketClientImpl patterns
  - Add JSON serialization with kotlinx.serialization
  - Add error handling with retry logic (3 retries, exponential backoff)
  - Add koinModule for DI registration

  **Must NOT do**:
  - Modify existing BridgeRpcClient interface

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — Implementation with networking complexity
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 2, depends on Tasks 2, 3)

  **References**:
  - `androidApp/.../platform/BridgeRpcClientImpl.kt` - RPC implementation pattern
  - `androidApp/.../di/AppModules.kt` - DI module pattern

  **Acceptance Criteria**:
  - [ ] Test file created: `androidApp/.../platform/SecretaryRpcClientImpl.test.kt`
  - [ ] `./gradlew test --tests "*.SecretaryRpcClientImplTest"` → PASS

  **QA Scenarios**:
  - Create template via RPC
  - Handle RPC failure with retry

  **Evidence**: `.sisyphus/evidence/task-9-*.test`

---

- [ ] 10. **Implement repository**

  **What to do**:
  - Create `SecretaryRepositoryImpl` class in `androidApp/.../data/SecretaryRepositoryImpl.kt`
  - Implement all CRUD methods using RPC client
  - Add in-memory caching with LRU (max 100 templates)
  - Add execution history storage (max 1000 entries)

  **Must NOT do**:
  - No database storage (use RPC client + cache)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — Implementation with state management
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 2, depends on Tasks 1, 2)

  **References**:
  - `androidApp/.../data/MessageRepositoryImpl.kt` - Repository pattern

  **Acceptance Criteria**:
  - [ ] Test file created: `androidApp/.../data/SecretaryRepositoryImpl.test.kt`
  - [ ] `./gradlew test --tests "*.SecretaryRepositoryImplTest"` → PASS

  **QA Scenarios**:
  - List templates with pagination

  **Evidence**: `.sisyphus/evidence/task-10-*.test`

---

- [ ] 11. **Implement ViewModel**

  **What to do**:
  - Create `SecretaryViewModel` class in `androidApp/.../viewmodels/SecretaryViewModel.kt`
  - Define UI state: `SecretaryUiState` with templates, isLoading, error
  - Define events: `SecretaryEvent` sealed class
  - Implement state management with StateFlow
  - Implement event handling with Channel<SecretaryEvent>

  **Must NOT do**:
  - No business logic (delegate to repository)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — Implementation with state management
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 2, depends on Tasks 9, 10)

  **References**:
  - `androidApp/.../viewmodels/ChatViewModel.kt` - ViewModel pattern
  - `androidApp/.../viewmodels/SyncStatusViewModelTest.kt` - Turbine patterns

  **Acceptance Criteria**:
  - [ ] Test file created: `androidApp/.../viewmodels/SecretaryViewModel.test.kt`
  - [ ] `./gradlew test --tests "*.SecretaryViewModelTest"` → PASS

  **QA Scenarios**:
  - Load templates on ViewModel

  **Evidence**: `.sisyphus/evidence/task-11-*.test`

---

- [ ] 12. **Implement failure handling**

  **What to do**:
  - Create `FailureHandler` class in `shared/domain/logic/FailureHandler.kt`
  - Implement `handleNetworkError(error: Exception): FailureAction`
  - Implement `handleExecutionError(error: Exception): FailureAction`
  - Return RETRY, QUEUE, or FAIL based on error type

  **Must NOT do**:
  - No excessive notifications (max 3 per day)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — Implementation with error handling
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 2, depends on Tasks 1, 7)

  **References**:
  - Existing error handling in `androidApp/.../platform/`

  **Acceptance Criteria**:
  - [ ] Test file created: `shared/domain/logic/FailureHandler.test.kt`
  - [ ] `./gradlew test --tests "*.FailureHandlerTest"` → PASS

  **QA Scenarios**:
  - Handle network error with retry

  **Evidence**: `.sisyphus/evidence/task-12-*.test`

---

- [ ] 13. **Implement execution history logger**

  **What to do**:
  - Create `ExecutionHistoryLogger` class in `shared/domain/logic/ExecutionHistoryLogger.kt`
  - Implement `logExecution(templateId: String, status: ExecutionStatus, error: String?)`
  - Implement `getHistory(templateId: String, page: Int, limit: Int): List<ExecutionHistory>`
  - Add retention policy (max 1000 entries, 30 days)
  - Add cleanup logic for old entries

  **Must NOT do**:
  - No sensitive data in logs

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — Implementation with logging
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 2, depends on Task 1)

  **References**:
  - Existing logging patterns in `androidApp/.../`

  **Acceptance Criteria**:
  - [ ] Test file created: `shared/domain/logic/ExecutionHistoryLogger.test.kt`
  - [ ] `./gradlew test --tests "*.ExecutionHistoryLoggerTest"` → PASS

  **QA Scenarios**:
  - Log execution and retrieve

  **Evidence**: `.sisyphus/evidence/task-13-*.test`

---

## Wave 3: UI Components

---

- [ ] 14. **Create TemplateCard component**

  **What to do**:
  - Create `TemplateCard` composable in `androidApp/.../components/secretary/TemplateCard.kt`
  - Display template name, description, status (active/inactive), last run time
  - Add toggle switch for enable/disable
  - Add edit button
  - Add delete button

  **Must NOT do**:
  - No complex animations (use simple transitions)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — UI component with Material 3
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 3, depends on Task 11)

  **References**:
  - `shared/ui/components/` - UI component patterns
  - `androidApp/.../components/` - Component patterns

  **Acceptance Criteria**:
  - [ ] Component renders template info correctly
  - [ ] Toggle switch works
  - [ ] Edit/Delete buttons visible

  **QA Scenarios**:
  - Display active template
  - Display inactive template
  - Toggle template active/inactive

  **Evidence**: Screenshots

---

- [ ] 15. **Create ConditionBlock component**

  **What to do**:
  - Create `ConditionBlock` composable in `androidApp/.../components/secretary/ConditionBlock.kt`
  - Display condition field, operator dropdown, value input
  - Add delete button
  - Use Material 3 components

  **Must NOT do**:
  - No custom dropdown (use ExposedDropdownMenu)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — UI component
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 3, depends on Task 4)

  **References**:
  - `androidApp/.../studio/` - Blockly patterns

  **Acceptance Criteria**:
  - [ ] Condition block renders fields correctly
  - [ ] Operator dropdown has 5 options
  - [ ] Delete button removes block

  **QA Scenarios**:
  - Add condition block
  - Select operator
  - Delete condition block

  **Evidence**: Screenshots

---

- [ ] 16. **Create ActionBlock component**

  **What to do**:
  - Create `ActionBlock` composable in `androidApp/.../components/secretary/ActionBlock.kt`
  - Display action type dropdown, action-specific fields
  - Add delete button
  - Use Material 3 components

  **Must NOT do**:
  - No custom dropdown (use ExposedDropdownMenu)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — UI component
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 3, depends on Task 4)

  **References**:
  - `androidApp/.../studio/` - Blockly patterns

  **Acceptance Criteria**:
  - [ ] Action block renders fields correctly
  - [ ] Action type dropdown has 4 options
  - [ ] Action-specific fields appear based on type

  **QA Scenarios**:
  - Add action block
  - Select action type
  - Configure action fields

  **Evidence**: Screenshots

---

- [ ] 17. **Create SchedulePickerDialog component**

  **What to do**:
  - Create `SchedulePickerDialog` composable in `androidApp/.../components/secretary/SchedulePickerDialog.kt`
  - Implement visual schedule builder (hour, minute, day of week)
  - Implement cron expression input mode
  - Add toggle between visual and cron modes
  - Add timezone selector
  - Use Material 3 components

  **Must NOT do**:
  - No complex cron validation UI (use backend validation)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — UI component with complexity
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 3, depends on Task 8)

  **References**:
  - `androidApp/.../components/` - Dialog patterns

  **Acceptance Criteria**:
  - [ ] Visual builder works
  - [ ] Cron input mode works
  - [ ] Toggle between modes works
  - [ ] Timezone selector displays IANA timezones

  **QA Scenarios**:
  - Build schedule visually
  - Enter cron expression
  - Switch between modes

  **Evidence**: Screenshots

---

- [ ] 18. **Create TemplateEditorLayout component**

  **What to do**:
  - Create `TemplateEditorLayout` composable in `androidApp/.../components/secretary/TemplateEditorLayout.kt`
  - Add condition block container
  - Add action block container
  - Add Add Condition button
  - Add Add Action button
  - Add Save/Cancel buttons
  - Use Material 3 components

  **Must NOT do**:
  - No complex layout (use simple Column/Row)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — UI component
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 3, depends on Tasks 15, 16)

  **References**:
  - `androidApp/.../screens/` - Screen layout patterns

  **Acceptance Criteria**:
  - [ ] Condition blocks display in container
  - [ ] Action blocks display in container
  - [ ] Add Condition button works
  - [ ] Add Action button works
  - [ ] Save/Cancel buttons work

  **QA Scenarios**:
  - Add multiple conditions
  - Add multiple actions
  - Save template

  **Evidence**: Screenshots

---

## Wave 4: Screens + Integration

---

- [ ] 19. **Create SecretaryScreen**

  **What to do**:
  - Create `SecretaryScreen` composable in `androidApp/.../screens/secretary/SecretaryScreen.kt`
  - Display template list using TemplateCard
  - Add search/filter controls
  - Add Add Template button (FAB)
  - Implement pull-to-refresh
  - Handle empty state
  - Use Material 3 components

  **Must NOT do**:
  - No complex list animations (use LazyColumn)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — Screen with list
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 4, depends on Tasks 11, 14)

  **References**:
  - `androidApp/.../screens/HomeScreen.kt` - Screen pattern

  **Acceptance Criteria**:
  - [ ] Templates display in list
  - [ ] Search works
  - [ ] Add Template button opens editor
  - [ ] Pull-to-refresh works
  - [ ] Empty state displays message

  **QA Scenarios**:
  - Open secretary screen
  - Search for template
  - Add new template

  **Evidence**: Screenshots

---

- [ ] 20. **Create TemplateEditorScreen**

  **What to do**:
  - Create `TemplateEditorScreen` composable in `androidApp/.../screens/secretary/TemplateEditorScreen.kt`
  - Integrate TemplateEditorLayout
  - Add template name/description inputs
  - Add condition block workspace
  - Add action block workspace
  - Add schedule picker button
  - Implement save/cancel
  - Handle back navigation

  **Must NOT do**:
  - No complex Blockly integration (use existing patterns)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — Screen with editor
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 4, depends on Tasks 11, 18)

  **References**:
  - `androidApp/.../screens/studio/` - Editor patterns

  **Acceptance Criteria**:
  - [ ] Name/description inputs work
  - [ ] Condition blocks can be added/removed
  - [ ] Action blocks can be added/removed
  - [ ] Schedule picker opens
  - [ ] Save saves template
  - [ ] Cancel discards changes

  **QA Scenarios**:
  - Create new template
  - Edit existing template
  - Save template with conditions and actions

  **Evidence**: Screenshots

---

- [ ] 21. **Create SchedulePickerScreen**

  **What to do**:
  - Create `SchedulePickerScreen` composable in `androidApp/.../screens/secretary/SchedulePickerScreen.kt`
  - Integrate SchedulePickerDialog
  - Add confirm/cancel buttons
  - Handle back navigation

  **Must NOT do**:
  - No complex UI (use dialog pattern)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering` — Screen with dialog
  - **Skills**: [`frontend-ui-ux`]
  - **Parallelization**: YES (Wave 4, depends on Tasks 8, 17)

  **References**:
  - `androidApp/.../screens/` - Screen patterns

  **Acceptance Criteria**:
  - [ ] Schedule picker dialog displays
  - [ ] Visual builder works
  - [ ] Cron input works
  - [ ] Confirm saves schedule
  - [ ] Cancel closes screen

  **QA Scenarios**:
  - Open schedule picker
  - Build schedule visually
  - Enter cron expression
  - Save schedule

  **Evidence**: Screenshots

---

- [ ] 22. **Integration tests**

  **What to do**:
  - Create integration tests in `shared/.../integration/SecretaryIntegrationTest.kt`
  - Test full workflows: create template, enable, trigger, log history
  - Test CRUD operations end-to-end
  - Test scheduling behavior
  - Test failure handling

  **Must NOT do**:
  - No flaky tests

  **Recommended Agent Profile**:
  - **Category**: `deep` — Integration test complexity
  - **Skills**: [`superpowers/systematic-debugging`]
  - **Parallelization**: YES (Wave 4, depends on Tasks 19, 20, 21)

  **References**:
  - `shared/.../viewmodel/ChatViewModelUnifiedTest.kt` - Integration test patterns

  **Acceptance Criteria**:
  - [ ] Test file created: `shared/.../integration/SecretaryIntegrationTest.kt`
  - [ ] `./gradlew connectedAndroidTest` → PASS

  **QA Scenarios**:
  - Test complete workflow

  **Evidence**: Test output

---

- [ ] 23. **E2E QA scenarios**

  **What to do**:
  - Create QA scenarios in `.sisyphus/evidence/final-qa/`
  - Test cross-task integration
  - Test edge cases: empty state, invalid input, rapid actions
  - Save evidence: screenshots, logs

  **Must NOT do**:
  - No missing scenarios

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high` — QA scenarios
  - **Skills**: []
  - **Parallelization**: YES (Wave 4, depends on Task 22)

  **Acceptance Criteria**:
  - [ ] All scenarios documented
  - [ ] Evidence captured

  **QA Scenarios**:
  - Full workflow test
  - Edge case tests

  **Evidence**: Screenshots and logs

---

## Final Verification Tasks

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read plan end-to-end. For each "Must Have": verify implementation exists. For each "Must NOT Have": search codebase for forbidden patterns.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `tsc --noEmit` + linter + `./gradlew test`. Review all changed files.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Execute EVERY QA scenario. Test cross-task integration. Test edge cases.
  Output: `Scenarios [N/N pass] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff. Verify 1:1 compliance.
  Output: `Tasks [N/N compliant] | VERDICT`

