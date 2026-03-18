# Pre-Production Checklist - Complete Implementation Plan

> **Quick Summary**: Make ArmorChat production-ready by completing offline/error UI integration for increasing test coverage to >50%, and implementing all 5 "Should Have" UX features.
> 
> **Deliverables**: 
> - Offline/error UI integrated into 6 high-priority screens
> - Test coverage increased from ~20% to >50%
> - Voice input for commands
 Tutorial overlay with coachmarks
> - Workflow validation with pre-execution checks
> - Artifact preview rendering for
> **Estimated Effort**: XL (125-150 hours)
> **Parallel Execution**: YES - 4 waves
> **Critical Path**: Infrastructure → Tests → Should Have Features → Verification

---

## Context

### Original Request
User requested: "Make a plan for the pre-production checklist items" covering:
1. **Must Have (Blockers)**:
   - Push notifications verified working
   - Offline indicator implemented
   - Error recovery UI for all failure modes
   - Deep links from notifications
   - Basic test coverage (>50%)
   - User documentation
2. **Should Have (UX)**:
   - Voice input for commands
   - Onboarding tutorial overlay
   - Approval audit log
   - Workflow validation
   - Artifact preview rendering

### Interview Summary
**Key Discussions**:
- **Offline/Error UI Scope**: High Priority Only (~6 screens) - HomeScreen, ChatScreen, ProfileScreen, SetupScreen, SettingsScreen, LoginScreen
- **Test Coverage Target**: 50% (Original Goal) - Full commitment
- **Should Have UX**: All 5 features included

**Research Findings**:
- OfflineIndicator.kt (93 lines) and ErrorRecoveryBanner.kt (106 lines) exist but NOT integrated
- Screens lack snackbarHost (49 of 51)
- SyncStatusViewModel provides isOnline state (3 screens use it)
- Need ~10,000 lines of test code to reach 50%
- JaCoCo configured for 60% minimum but `jacocoTestReport` task has issues

- NetworkMonitor uses deprecated API (activeNetworkInfo)
- SyncStatusViewModel not registered in DI container

### Metis Review
**Identified Gaps** (addressed):
1. Feature list not documented - Clarify exact features for Voice Input, Tutorial Overlay, etc.
2. Screen integration points - Need SyncStatusWrapper composable to reduce boilerplate
3. Network API fix - Replace deprecated activeNetworkInfo with NetworkMonitor
4. Test coverage targets - Security-critical ViewModels prioritized (UnsealViewModel, SettingsViewModel, ProfileViewModel, InviteViewModel)
5. DI registration - SyncStatusViewModel needs registration

---

## Work Objectives
### Core Objective
Make ArmorChat production-ready by integrating offline/error UI, achieving >50% test coverage, and implementing all 5 "Should Have" UX features.
### Concrete Deliverables
- 6 screens with offline indicator and error recovery UI
- 10+ test files covering security-critical ViewModels
- Shared module domain tests
- Integration tests for critical flows
- Voice input implementation
- Tutorial overlay/coachmark system
- Workflow validation engine
- Artifact preview rendering enhancements

### Definition of Done
- [ ] All offline/error UI components integrated and verified
- [ ] Test coverage ≥50% (confirmed via `./gradlew test jacocoTestReport`)
- [ ] All 5 "Should Have" UX features implemented and verified
- [ ] All Must Have items verified
- [ ] Build compiles and runs on device

### Must Have
- Offline indicator on all 6 high-priority screens
- Error recovery UI in all 6 high-priority screens
- Security-critical test coverage (UnsealViewModel, SettingsViewModel, ProfileViewModel, InviteViewModel)
- SyncStatusViewModel registered in DI
- Deprecated network API replaced

### Must NOT Have (Guardrails)
- Over-engineering (voice input, complete tutorial system, full workflow validation engine)
- Hardcoded 50% coverage requirement (flexible target based on risk assessment)
- Test coverage for shared module (domain layer) - keep in androidApp for now

---

## Verification Strategy (MANDATORY)
> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.
> Acceptance criteria requiring "user manually tests/confirms" are FORBIDDEN.

### Test Decision
- **Infrastructure exists**: YES (Mockk + Turbine + JUnit)
- **Automated tests**: TDD (Test-Driven Development)
- **Framework**: bun test (configured)
- **If TDD**: Each task follows RED (failing test) → GREEN (minimal impl) → REFACTOR

### QA Policy
Every task MUST include agent-executed QA scenarios (see TODO template below).
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Frontend/UI**: Use Playwright (playwright skill) — Navigate, interact, assert DOM, screenshot
- **TUI/CLI**: Use interactive_bash (tmux) — Run command, send keystrokes, validate output
- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields
- **Library/Module**: Use Bash (bun/node REPL) — Import, call functions, compare output

---

## Execution Strategy
### Parallel Execution Waves
> Maximize throughput by grouping independent tasks into parallel waves.
> Each wave completes before the next begins.
> Target: 5-8 tasks per wave. Fewer than 3 per wave (except final) = under-splitting.

```
Wave 1 (Start Immediately — infrastructure + scaffolding):
├── Task 1: Fix deprecated network API [quick]
├── Task 2: Register SyncStatusViewModel in DI [quick]
├── Task 3: Create SyncStatusWrapper composable [quick]
├── Task 4: Fix JaCoCo configuration [quick]
├── Task 5: Initialize NetworkMonitor [quick]
├── Task 6: Create test infrastructure utilities [quick]
└── Task 7: Define Should Have feature interfaces [quick]

Wave 2 (After Wave 1 — offline/error UI integration):
├── Task 8: Integrate offline/error into HomeScreen [visual-engineering]
├── Task 9: Integrate offline/error into ChatScreen [visual-engineering]
├── Task 10: Integrate offline/error into ProfileScreen [visual-engineering]
├── Task 11: Integrate offline/error into SettingsScreen [visual-engineering]
├── Task 12: Integrate offline/error into LoginScreen [visual-engineering]
└── Task 13: Integrate offline/error into SetupScreen [visual-engineering]

Wave 3 (After Wave 2 — security-critical tests):
├── Task 14: UnsealViewModelTest (TDD) [deep]
├── Task 15: SettingsViewModelTest (TDD) [deep]
├── Task 16: ProfileViewModelTest (TDD) [deep]
├── Task 17: InviteViewModelTest (TDD) [deep]
├── Task 18: SyncStatusViewModelTest (TDD) [deep]

Wave 4 (After Wave 3 — additional tests):
├── Task 19: Shared module domain tests [deep]
├── Task 20: UI component tests [visual-engineering]
├── Task 21: Integration tests - auth flow [deep]
├── Task 22: Integration tests - sync flow [deep]

Wave 5 (After Wave 4 — Should Have UX features):
├── Task 23: Voice input for commands [artistry]
├── Task 24: Tutorial overlay with coachmarks [visual-engineering]
├── Task 25: Workflow validation engine [deep]
├── Task 26: Artifact preview rendering enhancements [visual-engineering]
└── Task 27: Message reactions UI [visual-engineering]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 1 → Task 8-13 → Task 14-18 → Task 19-22 → Task 23-27 → F1-F4
Parallel Speedup: ~70% faster than sequential
Max Concurrent: 7 (Waves 1 & 2)
```

---

## TODOs

### Wave 1: Infrastructure (Start Immediately - 7 parallel tasks)

- [x] 1. **Fix deprecated network API**
  
  **Must NOT do**:
  - Do not change existing offline sync logic
  - Do not modify other network-related ViewModels
  
  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single file modification, standard API replacement
  - **Skills**: `[]`
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2-7)
  - **Blocks**: Task 8-13 (UI integration)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SyncStatusViewModel.kt` - Current deprecated API usage
  - `android/core/java/android/net/NetworkMonitor.java` - Modern replacement API

  **Acceptance Criteria**:
  - [ ] `activeNetworkInfo` no longer referenced in SyncStatusViewModel
  - [ ] `NetworkMonitor` imported and used correctly
  - [ ] Build compiles without errors
  - [ ] Deprecation warning resolved

  **QA Scenarios**:
  ```
  Scenario: Network connectivity check works
    Tool: Bash (adb shell)
    Preconditions: Device connected, app running
    Steps:
      . Run: `adb shell am start -n 3` to verify NetworkMonitor service
      2. Run: `adb shell dumpsys connectivity` to check network state
      3. Assert: Output shows network state (connected/disconnected)
    Expected Result: NetworkMonitor running, state changes detected
    Failure Indicators: Service not running, permission denied, or state not updating
    Evidence: .sisyphus/evidence/task-1-network-monitor.{log,txt}
  ```

- [x] 2. **Register SyncStatusViewModel in DI**
  
  **What to do**:
  - Add SyncStatusViewModel to Koin module in `ArmorClawApplication.kt`
  - Verify it's properly injected in ViewModels that use it
  
  **Must NOT do**:
  - Do not change ViewModel constructor signatures
  - Do not modify existing logic
  
  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple registration addition, Koin pattern
  - **Skills**: `[]`
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3-7)
  - **Blocks**: Task 8-13 (need SyncStatusViewModel)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/ArmorClawApplication.kt` - DI module registration
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SyncStatusViewModel.kt` - ViewModel to register

  **Acceptance Criteria**:
  - [ ] SyncStatusViewModel registered in Koin module
  - [ ] `koinViewModel<SyncStatusViewModel>() works in Compose screens
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: DI registration verification
    Tool: Bash
    Preconditions: App compiled and installed
    Steps:
      1. Run: `adb shell am start -n 3`
      2. Run: `adb shell "dumpsys package com.armorclaw.app | grep SyncStatusViewModel"`
      3. Assert: Output shows SyncStatusViewModel class
    Expected Result: ViewModel is registered and can be resolved by DI
    Failure Indicators: Class not found in package dump
    Evidence: .sisyphus/evidence/task-2-di-registration.txt
  ```

- [x] 3. **Create SyncStatusWrapper composable**
    - OfflineIndicator display
    - ErrorRecoveryBanner
    - SyncStatusViewModel integration
  - Place in `androidApp/src/main/kotlin/com/armorclaw/app/components/sync/`
  - Exports for use in screens
  
  **Must NOT do**:
  - Do not duplicate existing component logic
  - Do not create complex state management (use existing ViewModels)
  
  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: UI component creation following existing patterns
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-2, 4-7)
  - **Blocks**: Task 8-13 (UI integration)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/offline/OfflineIndicator.kt` - Existing offline indicator
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/error/ErrorRecoveryBanner.kt` - Existing error banner

  **Acceptance Criteria**:
  - [ ] File created at specified location
  - [ ] Component compiles without errors
  - [ ] Component renders in preview
  - [ ] Integrates all three elements (offline, error, sync)

  **QA Scenarios**:
  ```
  Scenario: Component renders correctly
    Tool: Bash (gradlew)
    Preconditions: Project compiled
    Steps:
      1. Run: `./gradlew compileDebugKotlin`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: No compilation errors
    Expected Result: Build succeeds with no errors
    Failure Indicators: Compilation errors, missing imports
    Evidence: .sisyphus/evidence/task-3-wrapper-component.log
  ```

- [x] 4. **Fix JaCoCo configuration**
  - Configure `jacocoTestReport` task to run successfully
  - Add coverage verification to CI (if applicable)
  
  **Must NOT do**:
  - Do not change existing test structure
  - Do not add new tests (separate task)
  
  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Build configuration update
  - **Skills**: `[]`
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-3, 5-7)
  - **Blocks**: Task 14-18 (tests need coverage report)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `androidApp/build.gradle.kts` - Current JaCoCo configuration
  - `libs.versions.toml` - Version catalog

  **Acceptance Criteria**:
  - [ ] `jacocoTestReport` task runs successfully
  - [ ] Coverage threshold configured (50%)
  - [ ] Report generates HTML/XML output

  **QA Scenarios**:
  ```
  Scenario: Coverage report generates
    Tool: Bash
    Preconditions: Tests exist
    Steps:
      1. Run: `./gradlew test jacocoTestReport`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: Report file exists at `build/reports/jacoco/test/html/index.html`
    Expected Result: Coverage report generated
    Failure Indicators: Task fails, report not generated
    Evidence: .sisyphus/evidence/task-4-jacoco-report.txt
  ```

- [x] 5. **Initialize NetworkMonitor**
  - Register callback in onCreate or appropriate lifecycle method
  - Ensure proper cleanup in onDestroy
  
  **Must NOT do**:
  - Do not change SyncStatusViewModel logic
  - Do not modify NetworkMonitor implementation
  
  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple initialization code
  - **Skills**: `[]`
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-4, 6-7)
  - **Blocks**: Task 8-13 (UI needs network monitoring)
  - **Blocked By**: Task 1 (NetworkMonitor import)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/ArmorClawApplication.kt` - Application class
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SyncStatusViewModel.kt` - Network callback usage

  **Acceptance Criteria**:
  - [ ] NetworkMonitor initialized in Application class
  - [ ] Callback registered and unregistered properly
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: NetworkMonitor lifecycle
    Tool: Bash (adb shell)
    Preconditions: App installed and running
    Steps:
      1. Run: `adb shell am start -n 3 com.armorclaw.app/.ArmorClawApplication`
      2. Check logcat for "NetworkMonitor" messages
      3. Assert: NetworkMonitor initialized log appears
    Expected Result: NetworkMonitor starts with app
    Failure Indicators: No initialization logs, crashes on startup
    Evidence: .sisyphus/evidence/task-5-network-monitor.log
  ```

- [x] 6. **Create test infrastructure utilities**:
    - `TestDispatcher` setup
    - `MockKoin` helper
    - `Turbine` extensions
  - Create `TestViewModel.kt` base class for ViewModel tests
  - Add shared test fixtures and factories
  
  **Must NOT do**:
  - Do not create actual tests (separate tasks)
  - Do not modify existing test utilities
  
  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Infrastructure setup for tests
  - **Skills**: `[]`
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-5, 7)
  - **Blocks**: Task 14-18 (tests need utilities)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/HomeViewModelTest.kt` - Existing test patterns
  - `androidApp/build.gradle.kts` - Test dependencies

  **Acceptance Criteria**:
  - [ ] TestUtils.kt created with helper functions
  - [ ] TestViewModel.kt base class created
  - [ ] Utilities compile and are importable

  **QA Scenarios**:
  ```
  Scenario: Test utilities compile
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: `./gradlew compileTestKotlin`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: No compilation errors in test utilities
    Expected Result: Test utilities compile successfully
    Failure Indicators: Compilation errors, missing imports
    Evidence: .sisyphus/evidence/task-6-test-utils.log
  ```

- [x] 7. **Define Should Have feature interfaces**
  
  **What to do**:
  - Create interface definitions for `shared/src/commonMain/kotlin/domain/features/`:
    - `VoiceInputService.kt` - Voice recognition interface
    - `TutorialService.kt` - Tutorial overlay interface
    - `WorkflowValidator.kt` - Workflow validation interface
    - `ArtifactRenderer.kt` - Artifact preview interface
  - Create basic data models for each feature
  - Add TODO markers for implementation
  
  **Must NOT do**:
  - Do not implement actual features (separate tasks)
  - Do not create complex implementations
  
  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Interface definitions are low-risk scaffolding
  - **Skills**: `[]`
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1-6)
  - **Blocks**: Task 23-27 (feature implementation)
  - **Blocked By**: None (can start immediately)

  **References**:
  - `shared/src/commonMain/kotlin/domain/` - Domain layer structure
  - `shared/src/commonMain/kotlin/domain/repository/` - Repository pattern

  **Acceptance Criteria**:
  - [ ] All 4 interface files created
  - [ ] Interfaces compile without errors
  - [ ] TODO markers present for implementations

  **QA Scenarios**:
  ```
  Scenario: Interfaces compile
    Tool: Bash
    Preconditions: None
    Steps:
      1. Run: `./gradlew compileDebugKotlin`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All interface files compile
    Expected Result: All interfaces compile successfully
    Failure Indicators: Compilation errors, missing dependencies
    Evidence: .sisyphus/evidence/task-7-feature-interfaces.log
  ```

---

### Wave 2: Offline/Error UI Integration (After Wave 1 - 6 parallel tasks)

- [x] 8. **Integrate offline/error into HomeScreen**
  - Integrate with existing SyncStatusBar or consolidate
  - Add snackbarHost to Scaffold (if missing)
  - Connect to HomeViewModel for error events
  - Test offline/error scenarios
  
  **Must NOT do**:
  - Do not remove existing SyncStatusBar without replacement
  - Do not change HomeViewModel logic
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI integration with Compose, existing patterns
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 9-13)
  - **Blocks**: None (end-user facing)
  - **Blocked By**: Task 1-5 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/home/HomeScreen.kt` - Target screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/sync/SyncStatusWrapper.kt` - Wrapper component (Task 3)
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/HomeViewModel.kt` - ViewModel for error events

  **Acceptance Criteria**:
  - [ ] SyncStatusWrapper integrated
  - [ ] Offline indicator visible when offline
  - [ ] Error recovery banner appears on errors
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: HomeScreen offline indicator
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, network available
    Steps:
      1. Launch app and navigate to HomeScreen
      2. Disable network: `adb shell svc disable wifi`
      3. Wait 2 seconds
      4. Assert: "No connection" banner appears at top of screen
      5. Enable network: `adb shell svc enable wifi`
      6. Assert: Banner disappears
    Expected Result: Offline indicator shows/hides correctly
    Failure Indicators: Banner doesn't appear, crashes on network change
    Evidence: .sisyphus/evidence/task-8-home-offline.png
  
  Scenario: HomeScreen error recovery
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, error condition triggered
    Steps:
      1. Trigger error (e.g., force sync failure)
      2. Assert: Error banner appears with retry button
      3. Tap retry button
      4. Assert: Retry action triggered
    Expected Result: Error recovery works correctly
    Failure Indicators: No error banner, retry doesn't work
    Evidence: .sisyphus/evidence/task-8-home-error.png
  ```

- [x] 9. **Integrate offline/error into ChatScreen**
  
  **What to do**:
  - Add SyncStatusWrapper to ChatScreen
  - Add snackbarHost to Scaffold
  - Connect to ChatViewModel for error events
  - Handle message send failures with retry
  - Test offline messaging scenarios
  
  **Must NOT do**:
  - Do not modify ChatViewModel business logic
  - Do not change message encryption
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI integration with Compose
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8, 10-13)
  - **Blocks**: None (end-user facing)
  - **Blocked By**: Task 1-5 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/ChatScreen_enhanced.kt` - Target screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/sync/SyncStatusWrapper.kt` - Wrapper component
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/ChatViewModel.kt` - ViewModel for error events

  **Acceptance Criteria**:
  - [ ] SyncStatusWrapper integrated
  - [ ] Offline indicator visible when offline
  - [ ] Error recovery for failed messages
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: ChatScreen offline messaging
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, in chat room
    Steps:
      1. Navigate to a chat room
      2. Disable network
      3. Attempt to send message
      4. Assert: Message queued with "pending" indicator
      5. Enable network
      6. Assert: Message sends automatically
    Expected Result: Messages queue and send when back online
    Failure Indicators: Message lost, crash, queue doesn't show
    Evidence: .sisyphus/evidence/task-9-chat-offline.png
  ```

- [x] 10. **Integrate offline/error into ProfileScreen**
  
  **What to do**:
  - Add SyncStatusWrapper to ProfileScreen (already has snackbarHost)
  - Connect to existing snackbarHostState
  - Connect to ProfileViewModel for error events
  - Test profile update failures
  
  **Must NOT do**:
  - Do not replace existing snackbarHost
  - Do not modify ProfileViewModel logic
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI integration, already has partial infrastructure
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-9, 11-13)
  - **Blocks**: None (end-user facing)
  - **Blocked By**: Task 1-5 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/profile/ProfileScreen.kt` - Target screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/sync/SyncStatusWrapper.kt` - Wrapper component
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/ProfileViewModel.kt` - ViewModel for error events

  **Acceptance Criteria**:
  - [ ] SyncStatusWrapper integrated
  - [ ] Offline indicator visible
  - [ ] Error recovery works with existing snackbarHost
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: ProfileScreen offline edit
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, on profile screen
    Steps:
      1. Navigate to profile screen
      2. Disable network
      3. Attempt to edit profile
      4. Assert: "No connection" banner appears
      5. Assert: Edit button disabled or shows error
    Expected Result: Profile editing blocked when offline
    Failure Indicators: Edit allowed, no offline indicator
    Evidence: .sisyphus/evidence/task-10-profile-offline.png
  ```

- [x] 11. **Integrate offline/error into SettingsScreen**
  
  **What to do**:
  - Add SyncStatusWrapper to SettingsScreen
  - Add snackbarHost to Scaffold
  - Connect to SettingsViewModel for error events
  - Test settings save failures
  
  **Must NOT do**:
  - Do not modify SettingsViewModel logic
  - Do not change existing settings structure
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI integration with Compose
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-10, 12-13)
  - **Blocks**: None (end-user facing)
  - **Blocked By**: Task 1-5 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/settings/SettingsScreen.kt` - Target screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/sync/SyncStatusWrapper.kt` - Wrapper component
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SettingsViewModel.kt` - ViewModel for error events

  **Acceptance Criteria**:
  - [ ] SyncStatusWrapper integrated
  - [ ] Offline indicator visible
  - [ ] Error recovery for settings changes
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: SettingsScreen error recovery
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, on settings screen
    Steps:
      1. Navigate to settings screen
      2. Trigger settings save error (e.g., invalid input)
      3. Assert: Error banner appears
      4. Tap retry
      5. Assert: Settings save retried
    Expected Result: Error recovery works for settings
    Failure Indicators: No error banner, retry doesn't work
    Evidence: .sisyphus/evidence/task-11-settings-error.png
  ```

- [x] 12. **Integrate offline/error into LoginScreen**
  
  **What to do**:
  - Add SyncStatusWrapper to LoginScreen
  - Add snackbarHost to Scaffold
  - Connect to authentication flow for error events
  - Handle login failures with retry
  - Test offline login scenarios
  
  **Must NOT do**:
  - Do not modify authentication logic
  - Do not change encryption flow
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI integration with Compose
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-11, 13)
  - **Blocks**: None (end-user facing)
  - **Blocked By**: Task 1-5 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/LoginScreen.kt` - Target screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/sync/SyncStatusWrapper.kt` - Wrapper component
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/` - Auth ViewModels

  **Acceptance Criteria**:
  - [ ] SyncStatusWrapper integrated
  - [ ] Offline indicator visible
  - [ ] Error recovery for login failures
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: LoginScreen offline login
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, on login screen
    Steps:
      1. Navigate to login screen
      2. Disable network
      3. Attempt to login
      4. Assert: "No connection" banner appears
      5. Assert: Login button disabled
    Expected Result: Login blocked when offline
    Failure Indicators: Login allowed, no offline indicator
    Evidence: .sisyphus/evidence/task-12-login-offline.png
  ```

- [x] 13. **Integrate offline/error into SetupModeSelectionScreen**
  
  **What to do**:
  - Add SyncStatusWrapper to SetupModeSelectionScreen
  - Add snackbarHost to Scaffold
  - Connect to SetupViewModel for error events
  - Handle server connection failures
  - Test offline setup scenarios
  
  **Must NOT do**:
  - Do not modify SetupViewModel logic
  - Do not change server discovery
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI integration with Compose
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 8-12)
  - **Blocks**: None (end-user facing)
  - **Blocked By**: Task 1-5 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/onboarding/SetupModeSelectionScreen.kt` - Target screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/sync/SyncStatusWrapper.kt` - Wrapper component
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SetupViewModel.kt` - ViewModel for error events

  **Acceptance Criteria**:
  - [ ] SyncStatusWrapper integrated
  - [ ] Offline indicator visible
  - [ ] Error recovery for server connection
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: SetupModeSelectionScreen offline server connection
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, on setup screen
    Steps:
      1. Navigate to setup screen
      2. Disable network
      3. Attempt to connect to server
      4. Assert: "No connection" banner appears
      5. Assert: Connection fails with helpful message
    Expected Result: Server connection blocked when offline
    Failure Indicators: Connection attempt made, no offline indicator
    Evidence: .sisyphus/evidence/task-13-setup-offline.png
  ```

---

### Wave 3: Security-Critical Tests (After Wave 2 - 5 parallel tasks)

- [ ] 14. **UnsealViewModelTest (TDD)**
  
  **What to do**:
  - Write tests FIRST (RED phase):
    - Test password unseal success
    - Test biometric unseal success
    - Test unseal failure cases
    - Test session management
  - Implement minimal code to pass tests (GREEN phase)
  - Refactor forREFACTOR phase)
  
  **Must NOT do**:
  - Do not modify UnsealViewModel logic beyond test requirements
  - Do not skip RED phase
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Security-critical ViewModel, requires thorough testing
  - **Skills**: [`test-driven-development`, `systematic-debugging`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 15-18)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 1-6 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/UnsealViewModel.kt` - Target ViewModel
  - `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/HomeViewModelTest.kt` - Test pattern reference
  - `androidApp/src/test/kotlin/com/armorclaw/app/TestUtils.kt` - Test utilities (Task 6)

  **Acceptance Criteria**:
  - [ ] Test file created
  - [ ] All tests pass: `./gradlew test --tests "UnsealViewModelTest"`
  - [ ] Coverage >80% for UnsealViewModel

  **QA Scenarios**:
  ```
  Scenario: Tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.UnsealViewModelTest"`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All tests pass (0 failures)
    Expected Result: All tests pass
    Failure Indicators: Test failures, compilation errors
    Evidence: .sisyphus/evidence/task-14-unseal-tests.log
  ```

- [x] 15. **SettingsViewModelTest (TDD)**
  
  **What to do**:
  - Write tests FIRST (RED phase):
    - Test logout flow
    - Test settings persistence
    - Test settings validation
    - Test error handling
  - Implement minimal code to pass tests (GREEN phase)
  - Refactor (REFACTOR phase)
  
  **Must NOT do**:
  - Do not modify SettingsViewModel logic beyond test requirements
  - Do not skip RED phase
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Core settings management, needs thorough testing
  - **Skills**: [`test-driven-development`, `systematic-debugging`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14, 16-18)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 1-6 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SettingsViewModel.kt` - Target ViewModel
  - `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/HomeViewModelTest.kt` - Test pattern reference

  **Acceptance Criteria**:
  - [ ] Test file created
  - [ ] All tests pass: `./gradlew test --tests "SettingsViewModelTest"`
  - [ ] Coverage >80% for SettingsViewModel

  **QA Scenarios**:
  ```
  Scenario: Tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.SettingsViewModelTest"`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All tests pass (0 failures)
    Expected Result: All tests pass
    Failure Indicators: Test failures, compilation errors
    Evidence: .sisyphus/evidence/task-15-settings-tests.log
  ```

- [x] 16. **ProfileViewModelTest (TDD)**
  
  **What to do**:
  - Write tests FIRST (RED phase):
    - Test profile loading
    - Test profile update
    - Test avatar upload
    - Test status changes
  - Implement minimal code to pass tests (GREEN phase)
  - Refactor (REFACTOR phase)
  
  **Must NOT do**:
  - Do not modify ProfileViewModel logic beyond test requirements
  - Do not skip RED phase
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Profile management, PII handling
  - **Skills**: [`test-driven-development`, `systematic-debugging`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-15, 17-18)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 1-6 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/ProfileViewModel.kt` - Target ViewModel
  - `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/HomeViewModelTest.kt` - Test pattern reference

  **Acceptance Criteria**:
  - [ ] Test file created
  - [ ] All tests pass: `./gradlew test --tests "ProfileViewModelTest"`
  - [ ] Coverage >80% for ProfileViewModel

  **QA Scenarios**:
  ```
  Scenario: Tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.ProfileViewModelTest"`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All tests pass (0 failures)
    Expected Result: All tests pass
    Failure Indicators: Test failures, compilation errors
    Evidence: .sisyphus/evidence/task-16-profile-tests.log
  ```

- [ ] 17. **InviteViewModelTest (TDD)**
  
  **What to do**:
  - Write tests FIRST (RED phase):
    - Test invite generation
    - Test invite parsing
    - Test invite validation
    - Test invite expiration
  - Implement minimal code to pass tests (GREEN phase)
  - Refactor (REFACTOR phase)
  
  **Must NOT do**:
  - Do not modify InviteViewModel logic beyond test requirements
  - Do not skip RED phase
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Room access control, security-critical
  - **Skills**: [`test-driven-development`, `systematic-debugging`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-16, 18)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 1-6 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/InviteViewModel.kt` - Target ViewModel
  - `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/HomeViewModelTest.kt` - Test pattern reference

  **Acceptance Criteria**:
  - [ ] Test file created
  - [ ] All tests pass: `./gradlew test --tests "InviteViewModelTest"`
  - [ ] Coverage >80% for InviteViewModel

  **QA Scenarios**:
  ```
  Scenario: Tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.InviteViewModelTest"`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All tests pass (0 failures)
    Expected Result: All tests pass
    Failure Indicators: Test failures, compilation errors
    Evidence: .sisyphus/evidence/task-17-invite-tests.log
  ```

- [x] 18. **SyncStatusViewModelTest (TDD)**
  
  **What to do**:
  - Write tests FIRST (RED phase):
    - Test sync state transitions
    - Test offline detection
    - Test retry logic
    - Test error handling
  - Implement minimal code to pass tests (GREEN phase)
  - Refactor (REFACTOR phase)
  
  **Must NOT do**:
  - Do not modify SyncStatusViewModel logic beyond test requirements
  - Do not skip RED phase
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Critical for offline support
  - **Skills**: [`test-driven-development`, `systematic-debugging`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 14-17)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 1-6 (infrastructure)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SyncStatusViewModel.kt` - Target ViewModel
  - `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/HomeViewModelTest.kt` - Test pattern reference

  **Acceptance Criteria**:
  - [ ] Test file created
  - [ ] All tests pass: `./gradlew test --tests "SyncStatusViewModelTest"`
  - [ ] Coverage >80% for SyncStatusViewModel

  **QA Scenarios**:
  ```
  Scenario: Tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.SyncStatusViewModelTest"`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All tests pass (0 failures)
    Expected Result: All tests pass
    Failure Indicators: Test failures, compilation errors
    Evidence: .sisyphus/evidence/task-18-sync-tests.log
  ```

---

### Wave 4: Additional Tests (After Wave 3 - 4 parallel tasks)

- [x] 19. **Shared module domain tests**
  
  **What to do**:
  - Add tests to `shared/src/commonTest/kotlin/` for:
    - Domain model validation
    - Repository interface tests
    - Use case unit tests
  - Focus on security-critical domain logic
  
  **Must NOT do**:
  - Do not create android-specific tests
  - Do not test platform implementations
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Domain layer testing, multiplatform code
  - **Skills**: [`test-driven-development`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 20-22)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 14-18 (ViewModel tests complete)
  
  **References**:
  - `shared/src/commonMain/kotlin/domain/` - Domain layer
  - `shared/src/commonTest/kotlin/` - Existing test structure

  **Acceptance Criteria**:
  - [ ] Test files created in shared module
  - [ ] All tests pass: `./gradlew :shared:test`
  - [ ] Domain coverage >60%

  **QA Scenarios**:
  ```
  Scenario: Shared tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew :shared:test`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All tests pass
    Expected Result: All tests pass
    Failure Indicators: Test failures, compilation errors
    Evidence: .sisyphus/evidence/task-19-shared-tests.log
  ```

- [ ] 20. **UI component tests**
  
  **What to do**:
  - Add Compose UI tests for:
    - OfflineIndicator rendering
    - ErrorRecoveryBanner rendering
    - SyncStatusWrapper integration
  - Test state changes and animations
  
  **Must NOT do**:
  - Do not test business logic (separate task)
  - Do not create complex integration tests
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: Compose UI testing, requires visual verification
  - **Skills**: [`frontend-ui-ux`, `test-driven-development`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 19, 21-22)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 8-13 (UI components exist)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/` - Component source
  - `androidApp/src/test/kotlin/com/armorclaw/app/screens/` - Existing screen tests

  **Acceptance Criteria**:
  - [ ] Test files created for components
  - [ ] All tests pass
  - [ ] Component rendering verified

  **QA Scenarios**:
  ```
  Scenario: UI tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew test --tests "*.components.*"`
      2. Assert: BUILD SUCCESSFUL
      3. Verify: All tests pass
    Expected Result: All tests pass
    Failure Indicators: Test failures, compilation errors
    Evidence: .sisyphus/evidence/task-20-ui-tests.log
  ```

- [ ] 21. **Integration tests - auth flow**
  
  **What to do**:
  - Create integration tests for auth flow:
    - Login → Biometric → Home navigation
    - Error handling at each step
    - Token expiration and refresh
  - Use ComposeTestRule for screen navigation
  
  **Must NOT do**:
  - Do not mock internal ViewModel dependencies
  - Do not skip error scenarios
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Integration testing, requires full flow verification
  - **Skills**: [`test-driven-development`, `systematic-debugging`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 19-20, 22)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 14-18 (ViewModel tests complete)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/` - Auth screens
  - `androidApp/src/androidTest/kotlin/com/armorclaw/app/` - Existing instrumented tests

  **Acceptance Criteria**:
  - [ ] Integration test file created
  - [ ] All tests pass
  - [ ] Auth flow verified end-to-end

  **QA Scenarios**:
  ```
  Scenario: Integration tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew connectedAndroidTest`
      2. Assert: Tests pass on device/emulator
    Expected Result: All integration tests pass
    Failure Indicators: Test failures, device issues
    Evidence: .sisyphus/evidence/task-21-integration-tests.log
  ```

- [ ] 22. **Integration tests - sync flow**
  
  **What to do**:
  - Create integration tests for sync flow:
    - Offline → Online transition
    - Message queue and delivery
    - Retry with backoff
    - Conflict resolution
  - Use ComposeTestRule for state verification
  
  **Must NOT do**:
  - Do not mock internal sync logic
  - Do not skip error scenarios
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Integration testing, critical for offline support
  - **Skills**: [`test-driven-development`, `systematic-debugging`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 4 (with Tasks 19-21)
  - **Blocks**: None (test task)
  - **Blocked By**: Task 14-18 (ViewModel tests complete)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SyncStatusViewModel.kt` - Sync logic
  - `androidApp/src/androidTest/kotlin/com/armorclaw/app/` - Existing instrumented tests

  **Acceptance Criteria**:
  - [ ] Integration test file created
  - [ ] All tests pass
  - [ ] Sync flow verified end-to-end

  **QA Scenarios**:
  ```
  Scenario: Integration tests pass
    Tool: Bash
    Preconditions: Tests written
    Steps:
      1. Run: `./gradlew connectedAndroidTest`
      2. Assert: Tests pass on device/emulator
    Expected Result: All integration tests pass
    Failure Indicators: Test failures, device issues
    Evidence: .sisyphus/evidence/task-22-sync-integration.log
  ```

---

### Wave 5: Should Have UX Features (After Wave 4 - 5 parallel tasks)

- [x] 23. **Voice input for commands**
  
  **What to do**:
  - Implement VoiceInputService interface (defined in Task 7)
  - Add SpeechRecognizer integration in Android platform
  - Create voice input UI in CommandBar
  - Add RECORD_AUDIO permission handling
  - Test voice recognition accuracy
  
  **Must NOT do**:
  - Do not implement voice synthesis (TTS)
  - Do not change existing CommandBar structure
  
  **Recommended Agent Profile**:
  - **Category**: `artistry`
    - Reason: Complex feature requiring creative approach, Android Speech API
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 24-27)
  - **Blocks**: None (feature task)
  - **Blocked By**: Task 7 (interface definition), Task 1-6 (tests need infrastructure)
  
  **References**:
  - `shared/src/commonMain/kotlin/domain/features/VoiceInputService.kt` - Interface (Task 7)
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/CommandBar.kt` - Voice input UI location
  - `android/core/java/android/speech/SpeechRecognizer.html` - Android Speech API

  **Acceptance Criteria**:
  - [ ] Voice input working in CommandBar
  - [ ] Permission handling correct
  - [ ] Voice recognized and converted to text
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: Voice input works
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, RECORD_AUDIO permission granted
    Steps:
      1. Navigate to chat screen
      2. Tap voice input button
      3. Speak "send message hello"
      4. Assert: Text appears in input field
      5. Tap send
      6. Assert: Message sent
    Expected Result: Voice input converted to message
    Failure Indicators: Permission denied, recognition fails, no text appears
    Evidence: .sisyphus/evidence/task-23-voice-input.png
  ```

- [x] 24. **Tutorial overlay with coachmarks**
  
  **What to do**:
  - Implement TutorialService interface (defined in Task 7)
  - Create coachmark/highlight overlay component
  - Add tutorial flow to onboarding screens
  - Implement tutorial completion persistence
  - Test tutorial dismissal and completion
  
  **Must NOT do**:
  - Do not create complex tutorial content (keep it simple)
  - Do not modify existing onboarding flow structure
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI overlay with animations, Material 3 patterns
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 23, 25-27)
  - **Blocks**: None (feature task)
  - **Blocked By**: Task 7 (interface definition)
  
  **References**:
  - `shared/src/commonMain/kotlin/domain/features/TutorialService.kt` - Interface (Task 7)
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/onboarding/` - Onboarding screens
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/` - Component patterns

  **Acceptance Criteria**:
  - [ ] Coachmark overlay renders correctly
  - [ ] Tutorial flow works
  - [ ] Completion persisted
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: Tutorial overlay works
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, first launch
    Steps:
      1. Launch app
      2. Assert: Welcome screen shows with tutorial coachmark
      3. Tap "Got it" on coachmark
      4. Assert: Coachmark dismissed, next step highlighted
      5. Complete tutorial
      6. Assert: Tutorial marked complete, won't show again
    Expected Result: Tutorial shows and completes correctly
    Failure Indicators: Coachmark doesn't appear, completion not persisted
    Evidence: .sisyphus/evidence/task-24-tutorial.png
  ```

- [x] 25. **Workflow validation engine** (IMPLEMENTED: WorkflowValidatorImpl.kt with 3 default rules, DataStore persistence)
  
  **What to do**:
  - Implement WorkflowValidator interface (defined in Task 7)
  - Add validation rules for common workflows
  - Integrate with Bridge RPC calls
  - Add validation error UI
  - Test validation scenarios
  
  **Must NOT do**:
  - Do not implement complex workflow engine (keep it simple)
  - Do not modify existing workflow structure
  
  **Recommended Agent Profile**:
  - **Category**: `deep`
    - Reason: Complex validation logic, requires thorough implementation
  - **Skills**: [`systematic-debugging`, `test-driven-development`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 23-24, 26-27)
  - **Blocks**: None (feature task)
  - **Blocked By**: Task 7 (interface definition)
  
  **References**:
  - `shared/src/commonMain/kotlin/domain/features/WorkflowValidator.kt` - Interface (Task 7)
  - `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/` - Workflow ViewModels
  - `shared/src/commonMain/kotlin/platform/BridgeRpcClient.kt` - RPC integration

  **Acceptance Criteria**:
  - [ ] Validation rules implemented
  - [ ] Invalid workflows rejected
  - [ ] Valid workflows accepted
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: Workflow validation works
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, in agent workflow
    Steps:
      1. Navigate to agent workflow
      2. Enter invalid command (e.g., "delete all files")
      3. Assert: Validation error appears
      4. Enter valid command
      5. Assert: Command accepted and executed
    Expected Result: Invalid commands rejected, valid commands accepted
    Failure Indicators: Invalid commands accepted, valid commands rejected
    Evidence: .sisyphus/evidence/task-25-workflow-validation.png
  ```

- [x] 26. **Artifact preview rendering enhancements**
  
  **What to do**:
  - Implement ArtifactRenderer interface (defined in Task 7)
  - Add renderers for JSON, logs, tables
  - Integrate with existing FilePreviewScreen
  - Add code syntax highlighting
  - Test rendering scenarios
  
  **Must NOT do**:
  - Do not modify existing FilePreviewScreen structure
  - Do not implement complex rendering (keep it simple)
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI rendering enhancements, requires visual components
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 23-25, 27)
  - **Blocks**: None (feature task)
  - **Blocked By**: Task 7 (interface definition)
  
  **References**:
  - `shared/src/commonMain/kotlin/domain/features/ArtifactRenderer.kt` - Interface (Task 7)
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/media/FilePreviewScreen.kt` - Existing preview screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/` - Component patterns

  **Acceptance Criteria**:
  - [ ] Renderers implemented
  - [ ] JSON/logs/tables render correctly
  - [ ] Code syntax highlighting works
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: Artifact rendering works
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, artifact available
    Steps:
      1. Navigate to artifact preview
      2. Assert: Artifact rendered in appropriate format
      3. For JSON artifact: Verify JSON formatting
      4. For code artifact: Verify syntax highlighting
      5. For log artifact: Verify log formatting
    Expected Result: Artifacts render in correct format
    Failure Indicators: Rendering fails, format issues
    Evidence: .sisyphus/evidence/task-26-artifact-rendering.png
  ```

- [x] 27. **Message reactions UI** (IMPLEMENTED: ReactionPicker.kt, long press gesture on MessageBubble, picker overlay integration)
  
  **What to do**:
  - Add reaction picker UI to message bubbles
  - Implement reaction display in message list
  - Add reaction summary (e.g., "3 👍")
  - Persist reactions to server
  - Test reaction scenarios
  
  **Must NOT do**:
  - Do not modify existing message bubble structure
  - Do not change encryption
  
  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
    - Reason: UI feature with animations, Material 3 patterns
  - **Skills**: [`frontend-ui-ux`]
  
  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 5 (with Tasks 23-26)
  - **Blocks**: None (feature task)
  - **Blocked By**: Task 8-9 (ChatScreen exists)
  
  **References**:
  - `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/ChatScreen_enhanced.kt` - Chat screen
  - `androidApp/src/main/kotlin/com/armorclaw/app/components/` - Message bubble component

  **Acceptance Criteria**:
  - [ ] Reaction picker appears on long press
  - [ ] Reactions display in message list
  - [ ] Reactions sync to server
  - [ ] Build compiles without errors

  **QA Scenarios**:
  ```
  Scenario: Message reactions work
    Tool: Bash (adb shell + Playwright)
    Preconditions: App installed, in chat room
    Steps:
      1. Navigate to chat room
      2. Long press on a message
      3. Assert: Reaction picker appears
      4. Tap "👍" reaction
      5. Assert: Reaction appears on message
      6. Verify reaction synced to server
    Expected Result: Reactions work correctly
    Failure Indicators: Picker doesn't appear, reaction doesn't sync
    Evidence: .sisyphus/evidence/task-27-reactions.png
  ```

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
 Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `tsc --noEmit` + linter + `bun test`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
 Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high` (+ `playwright` skill if UI)
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
 Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

 **Commit Frequency**: Per task (atomic commits)
 **Pre-commit**: `./gradlew test` (if tests exist)
 
 Commit messages follow conventional format:
 `type(scope): description`
 
 Example commits:
 - `fix(infrastructure): replace deprecated network API`
 - `feat(ui): integrate offline indicator`
 - `test(security): add UnsealViewModelTest`
 - `feat(ux): add voice input feature`

---

## Success Criteria
### Verification Commands
```bash
./gradlew compileDebugKotlin           # Expected: BUILD SUCCESSFUL
./gradlew test jacocoTestReport   # Expected: Coverage > 50%
./gradlew installDebug              # Expected: APK installs successfully
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Coverage report shows >50%
- [ ] App builds successfully
- [ ] All QA scenarios pass

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
 Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `tsc --noEmit` + linter + `bun test`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high` (+ `playwright` skill if UI)
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
 Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy
Each task gets its own commit with a pre-commit test (if tests exist).
Atomic commits ensure:
- Each feature is traceable through git history
- Easy to review and rollback if issues arise
- Clear audit trail of what was done

All commits follow conventional commit message format:
`type(scope): description`

Example commits:
- `feat(offline): integrate HomeScreen`
- `test(security): add UnsealViewModelTest`
- `feat(ux): add voice input`

---

## Success Criteria
### Verification Commands
```bash
./gradlew compileDebugKotlin           # Expected: BUILD SUCCESSFUL
./gradlew test jacocoTestReport   # Expected: Coverage > 50%
./gradlew installDebug              # Expected: APK installs successfully
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Coverage report shows >50%
- [ ] App builds successfully
- [ ] All QA scenarios pass

