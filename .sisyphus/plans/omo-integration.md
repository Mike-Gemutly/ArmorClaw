# OMO Integration Plan: ArmorChat Evolution to Agent Command Center

> **Status**: READY FOR EXECUTION
> **Plan Version**: 1.0
> **Created**: 2025-03-14
> **Priority Order**: Core → UI → Studio

---

## Executive Summary

**Objective**: Transform ArmorChat from a Matrix chat client into a sophisticated "VPS Secretary" Agent Command Center through OMO (OhMyOpenagent) integration.

**Scope**: 6 Phases covering Matrix integration, Dashboard/Workspace UI, Agent Studio no-code builder, and Vault security enhancements.

**Key Decisions**:
- ✅ **Trixnity Integration**: Full replacement of current MatrixClient
- ✅ **Dashboard**: Replace HomeScreen with Mission Control Dashboard
- ✅ **Workflow Builder**: Blockly WebView integration
- ✅ **Split View**: Responsive adaptation (split pane / BottomSheet)
- ✅ **Test Strategy**: TDD with agent-executed QA
- ✅ **Priority**: Core → UI → Studio
- ✅ **Studio Scope**: Full 4-step wizard

**Estimated Effort**: 6-8 weeks across 3 engineers

---

## Phase 0: Architecture Decision Gates

**CRITICAL**: Complete these decision gates BEFORE starting Phase 1.

### Task 0.1: Trixnity Integration POC
**Goal**: Validate Trixnity vs current MatrixClientImpl before committing to migration.

**Approach**:
1. Build minimal POC: TrixnityMatrixClient implementing MatrixClient interface
2. Benchmark against current MatrixClientImpl:
   - Lines of Code
   - API completeness
   - E2EE capabilities
   - Maintenance burden
   - Performance
3. Document decision criteria: Keep current OR migrate to Trixnity

**Acceptance Criteria**:
- [ ] POC connects to Matrix homeserver successfully
- [ ] Login flow works end-to-end
- [ ] Room listing retrieves data
- [ ] Message sending/receiving works
- [ ] Comparison matrix created (Trixnity vs Current)
- [ ] Decision documented with rationale

**Estimate**: 3-5 days

---

### Task 0.2: HomeScreen Gap Analysis
**Goal**: Document exactly what's missing from current HomeScreen vs required Mission Control features.

**Approach**:
1. Read current HomeScreenFull.kt implementation
2. Document existing features:
   - Room list (Favorites, Chats, Archived)
   - Active workflows section
   - Needs attention queue
   - Emergency controls
3. Compare with Mission Control requirements:
   - Agent fleet card grid
   - Real-time status indicators
   - Quick action buttons
   - Metrics display
4. Define: Replace vs Extend vs Create separate screen

**Acceptance Criteria**:
- [ ] HomeScreen features documented
- [ ] Mission Control requirements mapped
- [ ] Gap matrix created (Current vs Required)
- [ ] Integration decision made (Replace/Extend/Create)
- [ ] Navigation impact documented

**Estimate**: 1-2 days

---

### Task 0.3: Vault Integration Plan
**Goal**: Define exactly where and how Vault components wire into existing codebase.

**Approach**:
1. Identify integration points:
   - Which ViewModels need VaultRepository injection?
   - Where does ShadowMap intercept requests?
   - Which screens need Vault access UI?
2. Define PII schema for OMO:
   - Extend PiiRegistry with OMO-specific fields
   - Define VaultKeyCategory extensions
   - Define VaultKeySensitivity extensions
3. Plan Vault UI:
   - VaultScreen for viewing/editing PII
   - Biometric prompt integration points
   - Keystore unseal flow

**Acceptance Criteria**:
- [ ] Integration points documented (5+ locations)
- [ ] OMO PII schema defined
- [ ] Vault UI requirements documented
- [ ] Biometric prompt flows defined
- [ ] Integration task list created

**Estimate**: 2-3 days

---

## Phase 1: OMO Core (Matrix Logic Layer)

**Objective**: Establish the secure, reactive foundation for agent messaging.

**Prerequisite**: Task 0.1 (Trixnity POC) MUST show migration path.

### Task 1.1: Add Trixnity Dependencies
**File**: `gradle/libs.versions.toml`, `shared/build.gradle.kts`

**Implementation**:
1. Add Trixnity library to version catalog:
   ```toml
   trixnity = "3.9.0"
   ```

2. Add to shared module dependencies:
   ```kotlin
   implementation("net.folivo:trixnity-client-core:${libs.versions.trixnity}")
   implementation("net.folivo:trixnity-client-repository:${libs.versions.trixnity}")
   ```

3. Update Android manifest (if required for Trixnity)

**Acceptance Criteria**:
- [ ] Gradle sync succeeds
- [ ] Trixnity artifacts resolved
- [ ] No version conflicts
- [ ] Version catalog updated

**Verification**:
```bash
./gradlew :shared:dependencies
```

**Estimate**: 0.5 day

---

### Task 1.2: Implement TrixnityMatrixClient
**File**: `shared/src/androidMain/kotlin/platform/matrix/TrixnityMatrixClient.kt` (NEW)

**Implementation**:
1. Implement MatrixClient interface using Trixnity SDK
2. Map Trixnity models to MatrixClient types:
   - Authentication (login, logout, session restoration)
   - Sync (startSync, stopSync, observeSync)
   - Rooms (getRoom, observeRoom, joinRoom, etc.)
   - Messages (sendTextMessage, observeMessages, reactions)
   - Events (observeEvents, observeArmorClawEvents)
   - Presence/Typing/Read Receipts
   - Encryption (verification, E2EE status)
   - User operations
   - Media upload/download
   - Push notifications

**Acceptance Criteria**:
- [ ] All MatrixClient interface methods implemented
- [ ] Trixnity initialization works
- [ ] Login flow succeeds
- [ ] Session storage integration works
- [ ] Room listing works
- [ ] Message sending/receiving works
- [ ] Sync events flow to existing event types
- [ ] No compilation errors
- [ ] Unit tests for core methods (login, send, receive)

**Verification**:
```bash
./gradlew test --tests "*TrixnityMatrixClient*"
```

**Estimate**: 5-7 days

---

### Task 1.3: Create Trixnity Koin Module
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/di/TrixnityModule.kt` (NEW)

**Implementation**:
1. Create new Koin module:
   ```kotlin
   val trixnityModule = module {
       single<MatrixClient> {
           TrixnityMatrixClient(
               httpClient = get(),
               sessionStorage = get(),
               syncManager = get(),
               config = get()
           )
       }
   }
   ```

2. Update AppModules.kt to include trixnityModule
3. Add feature flag for Trixnity vs Current implementation:
   ```kotlin
   val USE_TRIXTNITY = BuildConfig.ENABLE_TRIXTNITY
   ```

**Acceptance Criteria**:
- [ ] TrixnityModule created
- [ ] Added to appModules list
- [ ] Feature flag implemented
- [ ] MatrixClient resolves correctly from Koin
- [ ] No DI errors at runtime

**Verification**:
```bash
./gradlew assembleDebug
```

**Estimate**: 1 day

---

### Task 1.4: Remove Placeholder MatrixClient
**File**: `AppModules.kt`

**Implementation**:
1. Remove MatrixClientImpl from DI (keep as backup)
2. Ensure TrixnityMatrixClient is the active implementation
3. Clean up unused placeholder files

**Acceptance Criteria**:
- [ ] MatrixClientImpl removed from DI
- [ ] TrixnityMatrixClient active
- [ ] App starts without errors
- [ ] Login flow works end-to-end
- [ ] Chat message sending works

**Verification**:
```bash
# Manual testing:
./gradlew installDebug
# Test: Login → Chat → Send message
```

**Estimate**: 0.5 day

---

## Phase 2: Mission Control Dashboard (Home Screen)

**Prerequisite**: Task 0.2 (HomeScreen Gap Analysis) defines scope.

**Objective**: Visualize the "Fleet" of agents with real-time status and quick actions.

### Task 2.1: Design Dashboard Layout
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/screens/dashboard/MissionControlDashboard.kt` (NEW)

**Implementation**:
1. Replace HomeScreenFull with MissionControlDashboard
2. Layout sections:
   - Header: App status, sync state, quick actions
   - Agent Fleet: LazyVerticalGrid of AgentCard components
   - Active Tasks: LazyRow of ActiveTaskCard
   - Needs Attention: Queue of attention items
   - Recent Workflows: List of workflow cards

3. Use DesignTokens for consistent spacing/typography

**Acceptance Criteria**:
- [ ] MissionControlDashboard screen created
- [ ] Agent fleet grid displays agents
- [ ] Status indicators (🟢 Active / ⚪ Idle) work
- [ ] Active tasks section shows running tasks
- [ ] Needs attention queue displays prioritized items
- [ ] Quick action buttons (View Log, Stop, Pause) work
- [ ] Material 3 design system followed
- [ ] Edge-to-edge support with WindowInsets
- [ ] Dark mode support

**Verification**:
```bash
# Manual testing: Verify UI layout and design consistency
./gradlew installDebug
```

**Estimate**: 3-4 days

---

### Task 2.2: Create AgentCard Component
**File**: `shared/src/commonMain/kotlin/ui/components/AgentCard.kt` (NEW)

**Implementation**:
1. Atomic component following existing patterns
2. Card displays:
   - Agent avatar
   - Agent name and type
   - Status indicator (online/offline/processing)
   - Current task summary
   - Last activity timestamp
   - Quick action buttons

**Acceptance Criteria**:
- [ ] AgentCard component created
- [ ] Status indicator animated
- [ ] Avatar displays correctly
- [ ] Quick actions trigger callbacks
- [ ] Follows Material 3 design
- [ ] Uses DesignTokens for spacing/colors
- [ ] @Preview for light/dark mode

**Verification**:
```bash
./gradlew test --tests "*AgentCard*"
```

**Estimate**: 1-2 days

---

### Task 2.3: Implement Dashboard ViewModel
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/DashboardViewModel.kt` (NEW)

**Implementation**:
1. Expose StateFlow<DashboardUiState>
2. Load agents from AgentRepository
3. Observe active workflows
4. Observe needs attention queue from ControlPlaneStore
5. Handle quick actions (stop, pause, view log)

**Acceptance Criteria**:
- [ ] DashboardViewModel created
- [ ] Agents state flows correctly
- [ ] Active workflows observed
- [ ] Needs attention observed
- [ ] Quick actions implemented
- [ ] Error handling for failed actions
- [ ] Unit tests for state transitions

**Verification**:
```bash
./gradlew test --tests "*DashboardViewModel*"
```

**Estimate**: 2 days

---

### Task 2.4: Update Navigation
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/navigation/AppNavigation.kt`

**Implementation**:
1. Add route: `const val DASHBOARD = "dashboard"`
2. Update startDestination to DASHBOARD (or keep as HOME)
3. Update chat flow to use DASHBOARD as home

**Acceptance Criteria**:
- [ ] DASHBOARD route added
- [ ] Navigation updated to use new route
- [ ] Deep links work correctly
- [ ] Back stack management correct

**Verification**:
```bash
# Manual testing: Navigation flows correctly
./gradlew installDebug
```

**Estimate**: 0.5 day

---

## Phase 3: Agent Workspace (Split-View Chat)

**Objective**: Allow users to monitor agent "thought process" alongside conversation.

### Task 3.1: Create Split-View Layout Component
**File**: `shared/src/commonMain/kotlin/ui/components/SplitViewLayout.kt` (NEW)

**Implementation**:
1. Responsive layout component:
   - Tablet/Landscape: Two-pane (60% chat, 40% activity log)
   - Phone/Portrait: BottomSheet for activity log
2. Use Compose adaptive APIs:
   - `WindowSizeClass`
   - `LocalConfiguration`
   - `NavigationSuiteScaffold` (if available)

**Acceptance Criteria**:
- [ ] SplitViewLayout component created
- [ ] Two-pane layout works on tablets
- [ ] BottomSheet works on phones
- [ ] Smooth transition between layouts
- [ ] WindowInsets handled correctly
- [ ] Animation < 300ms

**Estimate**: 2-3 days

---

### Task 3.2: Create ActivityLog Component
**File**: `shared/src/commonMain/kotlin/ui/components/ActivityLog.kt` (NEW)

**Implementation**:
1. Vertical timeline of agent steps
2. Each item displays:
   - Step name (e.g., "Navigating...", "Filling Form...")
   - Status (running/completed/error)
   - Timestamp
   - Output preview
3. Auto-scroll to latest event

**Acceptance Criteria**:
- [ ] ActivityLog component created
- [ ] Steps display in timeline format
- [ ] Status indicators work
- [ ] Auto-scroll to latest
- [ ] Follows Material 3 design
- [ ] Uses DesignTokens

**Estimate**: 2 days

---

### Task 3.3: Update ChatScreenEnhanced with Split View
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/ChatScreenEnhanced.kt`

**Implementation**:
1. Integrate SplitViewLayout
2. Left pane: Existing chat timeline
3. Right pane/BottomSheet: ActivityLog
4. Bind ActivityLog to agent events from ControlPlaneStore

**Acceptance Criteria**:
- [ ] Split view integrated into ChatScreen
- [ ] Chat timeline displays correctly
- [ ] Activity log shows agent events
- [ ] Layout adapts to screen size
- [ ] Both panes scroll independently
- [ ] State preservation on configuration change
- [ ] Performance: No lag on rapid events

**Verification**:
```bash
# Manual testing: Test on phone and tablet
./gradlew installDebug
```

**Estimate**: 3-4 days

---

### Task 3.4: Implement Workspace ViewModel Updates
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/ChatViewModel.kt`

**Implementation**:
1. Expose agent activity events as StateFlow
2. Bind to ControlPlaneStore agent events
3. Optimize event batching (don't overwhelm UI)

**Acceptance Criteria**:
- [ ] Agent activity events exposed
- [ ] Events batched efficiently
- [ ] No memory leaks
- [ ] Unit tests for event flow

**Verification**:
```bash
./gradlew test --tests "*ChatViewModel*"
```

**Estimate**: 1-2 days

---

## Phase 4: Synthesis Command Bar

**Objective**: Bridge the gap between chatting and commanding with hybrid text/chip input.

### Task 4.1: Create CommandBar Component
**File**: `shared/src/commonMain/kotlin/ui/components/CommandBar.kt` (NEW)

**Implementation**:
1. Row with horizontal scroll for chips
2. Chips: Status, Screenshot, Stop, Pause, Logs
3. OutlinedTextField with:
   - Leading icon (mic for voice)
   - Trailing icon (send)
   - Placeholder: "Delegate a task..."
4. Chip click handlers inject commands into input

**Acceptance Criteria**:
- [ ] CommandBar component created
- [ ] Chips display and select correctly
- [ ] Input field works
- [ ] Chip commands inject text
- [ ] Voice icon ready (placeholder)
- [ ] Material 3 design
- [ ] Haptic feedback on chip tap

**Estimate**: 2 days

---

### Task 4.2: Integrate CommandBar into ChatScreen
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/ChatScreenEnhanced.kt`

**Implementation**:
1. Replace existing input bar with CommandBar
2. Handle chip command injection
3. Handle text + chip commands

**Acceptance Criteria**:
- [ ] CommandBar integrated
- [ ] Chip commands inject correctly
- [ ] Text input works
- [ ] Command sending works
- [ ] Keyboard handling correct

**Estimate**: 1 day

---

## Phase 5: Agent Studio (No-Code Builder)

**Objective**: Enable visual agent creation and configuration.

### Task 5.1: Create Blockly WebView Bridge
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/studio/BlocklyWebView.kt` (NEW)

**Implementation**:
1. AndroidView wrapping WebView
2. Load Blockly from local assets or CDN
3. Implement JavaScript bridge:
   ```kotlin
   class AndroidBridge(@JavascriptInterface) {
       fun onWorkspaceChanged(json: String)
       fun saveWorkspace(filename: String)
       fun loadWorkspace(filename: String)
   }
   ```
4. Enable JavaScript, DOM storage, database
5. Handle lifecycle (pause/resume/destroy)

**Acceptance Criteria**:
- [ ] BlocklyWebView component created
- [ ] Blockly loads in WebView
- [ ] JavaScript bridge works bidirectionally
- [ ] Workspace changes captured
- [ ] Save/load functions work
- [ ] No WebView memory leaks

**Estimate**: 3-4 days

---

### Task 5.2: Define Agent Block Library
**File**: `androidApp/src/main/assets/blocks/agent-blocks.json` (NEW)

**Implementation**:
1. Define trigger blocks:
   - `message_received`
   - `user_joins`
   - `timer_expired`
   - `schedule_triggered`

2. Define action blocks:
   - `send_message`
   - `send_email`
   - `api_call`
   - `run_command`

3. Define logic blocks:
   - `if_then_else`
   - `repeat`
   - `wait`
   - `parallel_execute`

4. Define toolbox JSON with categories

**Acceptance Criteria**:
- [ ] Block definitions created
- [ ] All triggers defined
- [ ] All actions defined
- [ ] Logic blocks defined
- [ ] Toolbox JSON valid
- [ ] Blocks render in Blockly

**Estimate**: 2-3 days

---

### Task 5.3: Create Agent Studio Wizard
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/screens/studio/AgentStudioScreen.kt` (NEW)

**Implementation**:
1. Use HorizontalPager for 4 steps
2. Step 0: RoleDefinitionScreen
   - Name, type, description
   - Avatar upload
3. Step 1: SkillSelectionScreen
   - SDUI dynamic form rendering
   - Checkboxes for skill selection
4. Step 2: WorkflowBuilderScreen
   - BlocklyWebView integration
   - Toolbox configuration
   - Save/load buttons
5. Step 3: PermissionsScreen
   - PII access checkboxes
   - Biometric approval prompt
   - Sensitivity badges

**Acceptance Criteria**:
- [ ] 4-step wizard created
- [ ] Pager navigation works
- [ ] Step 0 (Role) collects data
- [ ] Step 1 (Skills) renders SDUI
- [ ] Step 2 (Workflow) shows Blockly
- [ ] Step 3 (Permissions) handles PII
- [ ] Back/Next navigation works
- [ ] Progress indicator shows step

**Estimate**: 5-6 days

---

### Task 5.4: Implement Studio ViewModel
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/AgentStudioViewModel.kt` (NEW)

**Implementation**:
1. StateFlow<StudioUiState> with 4 steps
2. Handle:
   - Wizard navigation (next/back)
   - Data collection per step
   - Blockly workspace save/load
   - Agent creation submission
3. Integrate with AgentRepository

**Acceptance Criteria**:
- [ ] ViewModel created
- [ ] Wizard state managed correctly
- [ ] Blockly workspace saved/loaded
- [ ] Agent creation submits correctly
- [ ] Validation for each step
- [ ] Unit tests for all steps

**Verification**:
```bash
./gradlew test --tests "*AgentStudioViewModel*"
```

**Estimate**: 3 days

---

### Task 5.5: Add Studio Navigation Routes
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/navigation/AppNavigation.kt`

**Implementation**:
1. Add route: `const val AGENT_STUDIO = "agent_studio"`
2. Add navigation from Dashboard → Agent Studio
3. Add navigation from AgentManagement → Agent Studio

**Acceptance Criteria**:
- [ ] AGENT_STUDIO route added
- [ ] Navigation works from multiple entry points
- [ ] Back stack correct

**Estimate**: 0.5 day

---

## Phase 6: Vault Security Integration

**Prerequisite**: Task 0.3 (Vault Integration Plan) defines wire-up points.

**Objective**: Activate Cold Vault infrastructure for OMO PII protection.

### Task 6.1: Extend PiiRegistry for OMO
**File**: `shared/src/commonMain/kotlin/domain/security/PiiRegistry.kt`

**Implementation**:
1. Add OMO PII field definitions:
   - OMO_CREDENTIALS (API keys, tokens)
   - OMO_IDENTITY (user identity)
   - OMO_SETTINGS (configuration)
   - OMO_TOKENS (session tokens)

**Acceptance Criteria**:
- [ ] OMO PII fields registered
- [ ] PiiRegistry extension methods created
- [ ] Validation for new fields

**Estimate**: 1 day

---

### Task 6.2: Extend VaultRepository for OMO
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/security/VaultRepository.kt`

**Implementation**:
1. Add OMO-specific CRUD methods:
   - `storeOMOCredential()`
   - `retrieveOMOCredential()`
   - `storeOMOToken()`
   - `retrieveOMOToken()`
2. Add access logging for OMO fields
3. Update vault schema if needed

**Acceptance Criteria**:
- [ ] OMO storage methods implemented
- [ ] Access logging works
- [ ] Encryption uses existing keys
- [ ] Unit tests for OMO methods

**Verification**:
```bash
./gradlew test --tests "*VaultRepository*"
```

**Estimate**: 2 days

---

### Task 6.3: Wire Vault into Agent Flows
**File**: `shared/src/commonMain/kotlin/domain/repository/AgentRepository.kt`, `shared/src/commonMain/kotlin/domain/security/ShadowMap.kt`

**Implementation**:
1. AgentRepository: Call VaultRepository for PII access
2. ShadowMap: Create placeholders for OMO PII fields
3. BiometricAuthorizer: Add OMO-specific authorization prompts

**Acceptance Criteria**:
- [ ] AgentRepository uses VaultRepository
- [ ] ShadowMap masks OMO PII
- [ ] Biometric prompts for OMO access
- [ ] Access logged to vault_access_log

**Estimate**: 2-3 days

---

### Task 6.4: Create Vault UI Screen
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/screens/vault/VaultScreen.kt` (NEW)

**Implementation**:
1. Screen to view stored PII
2. Section headers by category (Personal, Financial, OMO)
3. Edit button per field with biometric prompt
4. Add new PII button

**Acceptance Criteria**:
- [ ] VaultScreen created
- [ ] PII displayed by category
- [ ] Edit requires biometric
- [ ] New PII form works
- [ ] Material 3 design
- [ ] Dark mode support

**Estimate**: 3-4 days

---

### Task 6.5: Add Vault Navigation
**File**: `androidApp/src/main/kotlin/com/armorclaw/app/navigation/AppNavigation.kt`

**Implementation**:
1. Add route: `const val VAULT = "vault"`
2. Add navigation from Settings → Vault
3. Add biometric gate before opening Vault

**Acceptance Criteria**:
- [ ] VAULT route added
- [ ] Biometric prompt before navigation
- [ ] Back stack correct

**Estimate**: 0.5 day

---

## Phase 7: Testing & QA

**Objective**: Comprehensive test coverage for all OMO features.

### Task 7.1: Write Unit Tests for Phase 1 (Matrix)
**Files**: Multiple test files under `shared/src/test/`

**Coverage**:
- TrixnityMatrixClient tests
- MatrixClient interface contract tests
- DI module tests
- Session storage tests

**Acceptance Criteria**:
- [ ] 20+ unit tests for TrixnityMatrixClient
- [ ] All MatrixClient interface methods covered
- [ ] Mock implementations for dependencies
- [ ] Test coverage > 70%

**Verification**:
```bash
./gradlew test --tests "*Trixnity*"
```

**Estimate**: 3 days

---

### Task 7.2: Write Unit Tests for Phase 2 (Dashboard)
**Files**: `androidApp/src/test/kotlin/com/armorclaw/app/screens/dashboard/`

**Coverage**:
- DashboardViewModel tests
- AgentCard tests
- Navigation tests

**Acceptance Criteria**:
- [ ] 15+ unit tests for DashboardViewModel
- [ ] Component snapshot tests
- [ ] State transition tests
- [ ] Test coverage > 60%

**Verification**:
```bash
./gradlew test --tests "*Dashboard*"
```

**Estimate**: 2 days

---

### Task 7.3: Write Unit Tests for Phase 3 (Workspace)
**Files**: `androidApp/src/test/kotlin/com/armorclaw/app/screens/chat/`

**Coverage**:
- SplitViewLayout tests
- ActivityLog tests
- ChatViewModel updates tests

**Acceptance Criteria**:
- [ ] 15+ unit tests for workspace components
- [ ] Layout adaptation tests
- [ ] Event flow tests
- [ ] Test coverage > 60%

**Verification**:
```bash
./gradlew test --tests "*Workspace*"
```

**Estimate**: 2 days

---

### Task 7.4: Write Unit Tests for Phase 5 (Studio)
**Files**: `androidApp/src/test/kotlin/com/armorclaw/app/screens/studio/`

**Coverage**:
- AgentStudioViewModel tests
- BlocklyWebView tests (mocked)
- Block definition tests

**Acceptance Criteria**:
- [ ] 20+ unit tests for AgentStudioViewModel
- [ ] Wizard navigation tests
- [ ] Blockly save/load tests
- [ ] Test coverage > 65%

**Verification**:
```bash
./gradlew test --tests "*AgentStudio*"
```

**Estimate**: 3 days

---

### Task 7.5: Write Unit Tests for Phase 6 (Vault)
**Files**: `androidApp/src/test/kotlin/com/armorclaw/app/security/`

**Coverage**:
- VaultRepository tests
- PiiRegistry tests
- ShadowMap tests
- BiometricAuthorizer tests

**Acceptance Criteria**:
- [ ] 20+ unit tests for VaultRepository
- [ ] Encryption tests
- [ ] Access logging tests
- [ ] Test coverage > 75%

**Verification**:
```bash
./gradlew test --tests "*Vault*"
```

**Estimate**: 2 days

---

### Task 7.6: Integration Tests
**Files**: `androidApp/src/androidTest/kotlin/com/armorclaw/app/`

**Coverage**:
- End-to-end flows: Dashboard → Chat → Studio
- Agent lifecycle tests
- Vault integration tests

**Acceptance Criteria**:
- [ ] 10+ integration tests
- [ ] Full flow tests (onboarding to agent execution)
- [ ] UI component tests with ComposeTestRule
- [ ] All critical paths covered

**Verification**:
```bash
./gradlew connectedAndroidTest
```

**Estimate**: 3 days

---

## Final Wave: Verification & Approval

### F1: Build Verification
**Goal**: Ensure code compiles and app builds successfully.

**Commands**:
```bash
# Clean build
./gradlew clean

# Build debug
./gradlew assembleDebug

# Build release
./gradlew assembleRelease

# Run all tests
./gradlew test

# Run LSP diagnostics (via IDE)
```

**Acceptance Criteria**:
- [ ] Zero compilation errors
- [ ] Zero LSP errors at project level
- [ ] All unit tests pass (> 80% pass rate)
- [ ] All integration tests pass
- [ ] APK builds successfully
- [ ] APK size < 50MB

**Estimate**: 1 day

---

### F2: Manual Code Review
**Goal**: Review every changed file for correctness and adherence to standards.

**Review Checklist**:
- [ ] All new files follow Material 3 design
- [ ] All ViewModels use StateFlow pattern
- [ ] All components use DesignTokens
- [ ] All navigation follows existing patterns
- [ ] All encryption uses VaultRepository
- [ ] All biometric prompts use BiometricAuthorizer
- [ ] No hardcoded values (use tokens/constants)
- [ ] Proper error handling in all ViewModels
- [ ] No memory leaks (coroutine scopes cleaned)
- [ ] Accessibility labels (contentDescription)

**Files to Review**:
1. TrixnityMatrixClient.kt
2. MissionControlDashboard.kt
3. AgentCard.kt, ActiveTaskCard.kt
4. SplitViewLayout.kt
5. ActivityLog.kt
6. CommandBar.kt
7. BlocklyWebView.kt, AgentStudioScreen.kt
8. VaultRepository.kt (extensions)
9. PiiRegistry.kt (extensions)
10. AppNavigation.kt (route additions)

**Estimate**: 2 days

---

### F3: User Acceptance Testing (UAT)
**Goal**: Validate features meet user expectations.

**Test Scenarios**:
1. **Dashboard Scenario**:
   - User opens app → sees agent fleet
   - Tap agent → sees details
   - Tap stop → agent stops
   - Observe status changes

2. **Workspace Scenario**:
   - User opens chat → sees split view
   - Send message → agent responds
   - Activity log shows agent steps
   - Rotate device → layout adapts

3. **Studio Scenario**:
   - User opens Agent Studio → wizard starts
   - Complete 4 steps → agent created
   - Navigate to dashboard → new agent appears
   - Test agent workflow execution

4. **Vault Scenario**:
   - User opens Vault → sees PII
   - Edit field → biometric prompt
   - Approve → field updated
   - Observe agent using masked value

5. **E2E Scenario**:
   - Create agent via Studio
   - Navigate to Dashboard
   - Chat with agent
   - Observe agent execution
   - Agent requests PII → Vault prompt
   - Approve → Agent completes task

**Acceptance Criteria**:
- [ ] All scenarios pass without crashes
- [ ] No console errors or exceptions
- [ ] All animations smooth (< 300ms)
- [ ] All biometric prompts work
- [ ] All split views adapt correctly
- [ ] All navigation flows work

**Estimate**: 2 days

---

### F4: Documentation & Handoff
**Goal**: Complete documentation and prepare for deployment.

**Deliverables**:
1. Update README.md with OMO features
2. Update ARCHITECTURE.md with new components
3. Update COMPONENTS.md with new atoms/molecules/organisms
4. Create OMO_DEPLOYMENT.md with deployment checklist
5. Update SCREENS.md with new screen documentation
6. Update CHANGELOG.md with version notes

**Acceptance Criteria**:
- [ ] README updated
- [ ] Architecture documented
- [ ] Components cataloged
- [ ] Deployment guide created
- [ ] Screens documented
- [ ] Changelog updated
- [ ] All diagrams up to date

**Estimate**: 1 day

---

## Scope Boundaries

### IN SCOPE
- ✅ Trixnity Matrix client integration
- ✅ Mission Control Dashboard (replacing HomeScreen)
- ✅ Agent Workspace with split-view layout
- ✅ Synthesis Command Bar with chips
- ✅ Agent Studio with 4-step wizard
- ✅ Blockly visual workflow builder
- ✅ Vault security activation for OMO
- ✅ Responsive design (phone/tablet/landscape)
- ✅ Material 3 design system
- ✅ TDD with comprehensive test coverage

### OUT OF SCOPE
- ❌ Multi-user support
- ❌ Agent marketplace
- ❌ Voice command input (placeholder only)
- ❌ Cloud sync for workflows
- ❌ Advanced workflow debugging tools
- ❌ Custom block creation by users
- ❌ Agent collaboration features
- ❌ Real-time agent video feed
- ❌ Offline workflow execution
- ❌ Workflow templates library

---

## Guardrails

### Non-Negotiable Constraints
1. **Technology Stack**:
   - Kotlin 1.9.20
   - Compose 1.5.0
   - Material 3
   - Koin 3.5.0
   - SQLCipher 2.0.0

2. **Architecture Patterns**:
   - MVVM with StateFlow
   - Clean Architecture (Domain → Data → Presentation)
   - Expect/Actual for platform services
   - Koin modular DI

3. **Security Requirements**:
   - All PII encrypted via VaultRepository
   - Biometric prompts for CRITICAL/HIGH sensitivity
   - No hardcoded secrets
   - AndroidKeyStore for key storage
   - SQLCipher for database encryption

4. **Design System**:
   - Material 3 tokens
   - DesignTokens for spacing/typography
   - Atomic design components
   - Edge-to-edge with WindowInsets
   - Dark mode support

5. **Testing Requirements**:
   - TDD approach (tests first)
   - Minimum 70% test coverage
   - Unit + integration tests
   - Compose snapshot tests

---

## Risk Assessment

| Risk | Level | Mitigation |
|------|--------|------------|
| **Trixnity migration complexity** | HIGH | Complete Task 0.1 POC before committing |
| **HomeScreen replacement impact** | MEDIUM | Ensure all existing features preserved in Dashboard |
| **Blockly WebView performance** | MEDIUM | Monitor memory usage; implement lazy loading |
| **Split view UX on phones** | MEDIUM | Test on multiple device sizes; use BottomSheet fallback |
| **Vault integration breaking existing flows** | MEDIUM | Thorough integration testing; feature flag for rollback |
| **Scope creep to marketplace** | LOW | Explicit OUT OF SCOPE guardrails set |

---

## Success Criteria

**Plan is complete when:**
- [ ] All 7 phases with 40+ tasks defined
- [ ] All tasks have clear acceptance criteria
- [ ] All tasks have verification commands
- [ ] Test strategy defined (TDD, unit, integration, UAT)
- [ ] Guardrails and scope boundaries set
- [ ] Risk assessment documented
- [ ] Effort estimates provided (6-8 weeks total)

**Implementation is successful when:**
- [ ] All Final Wave tasks (F1-F4) pass
- [ ] All unit tests pass (> 80%)
- [ ] All integration tests pass
- [ ] Build succeeds with zero errors
- [ ] UAT scenarios pass without blockers
- [ ] Documentation complete

---

## Next Steps

After plan approval, execute in order:

1. **Wave 0**: Complete decision gates (Tasks 0.1, 0.2, 0.3)
2. **Wave 1**: Phase 1 tasks (Matrix integration)
3. **Wave 2**: Phase 2 tasks (Dashboard)
4. **Wave 3**: Phase 3 tasks (Workspace)
5. **Wave 4**: Phase 4 tasks (Command Bar)
6. **Wave 5**: Phase 5 tasks (Agent Studio)
7. **Wave 6**: Phase 6 tasks (Vault)
8. **Wave 7**: Phase 7 tasks (Testing)
9. **Final Wave**: F1-F4 verification tasks

**Command to start execution:**
```bash
/start-work
```

---

**Plan Status**: ✅ READY FOR EXECUTION

**Note**: 3 critical gaps (C1, C2, C3) and 7 minor/ambiguous gaps identified. See `.sisyphus/reviews/omo-integration-gap-analysis.md` for details. Address critical gaps in Phase 0 before starting Phase 1.
