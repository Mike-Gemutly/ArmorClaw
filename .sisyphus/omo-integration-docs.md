# OMO Integration Documentation

> **Version**: 1.0
> **Last Updated**: 2026-03-15
> **Status**: Complete

---

## Overview

This document describes the OMO (OhMyOpenagent) Integration features added to ArmorChat, transforming it from a Matrix chat client into an Agent Command Center.

---

## Architecture

### Module Structure

```
ArmorChat/
├── shared/                              # KMP shared module
│   ├── src/commonMain/kotlin/
│   │   ├── domain/
│   │   │   ├── model/                  # AgentEvent, OMOIdentityData
│   │   │   ├── repository/             # VaultKey, VaultKeyCategory, VaultKeySensitivity
│   │   │   └── security/               # PiiRegistry (extended)
│   │   └── ui/components/              # SplitViewLayout, ActivityLog, CommandBar
│   └── src/androidMain/kotlin/
│       └── com/armorclaw/app/studio/   # AgentBlocks
│
└── androidApp/                         # Android application
    └── src/main/kotlin/com/armorclaw/app/
        ├── screens/
        │   ├── chat/                   # ChatScreen with SplitView integration
        │   ├── vault/                  # VaultScreen for PII management
        │   └── studio/                 # AgentStudioScreen (4-step wizard)
        ├── studio/                     # BlocklyWebView
        ├── security/                   # VaultRepository (extended)
        └── navigation/                 # AppNavigation with VAULT, AGENT_STUDIO routes
```

---

## Features

### 1. Agent Workspace

A split-view layout that combines chat functionality with real-time agent monitoring.

#### Components

| Component | Description | Location |
|-----------|-------------|----------|
| SplitViewLayout | Responsive two-pane layout | `shared/ui/components/SplitViewLayout.kt` |
| ActivityLog | Agent activity timeline | `shared/ui/components/ActivityLog.kt` |
| CommandBar | Quick command chips | `shared/ui/components/CommandBar.kt` |

#### Behavior

- **Large screens**: Split view with chat on left, ActivityLog on right
- **Small screens**: ActivityLog collapses to BottomSheet
- **Drag handle**: Users can resize panes by dragging

#### Command Chips

| Chip | Command | Description |
|------|---------|-------------|
| Status | `!status` | Check agent status |
| Screenshot | `!screenshot` | Capture screen |
| Stop | `!stop` | Emergency stop |
| Pause | `!pause` | Pause all agents |
| Logs | `!logs` | View recent logs |

---

### 2. Vault (PII Management)

Secure storage for Personally Identifiable Information with biometric protection.

#### OMO Categories

| Category | Description | Sensitivity |
|----------|-------------|-------------|
| OMO_CREDENTIALS | API keys, tokens | OMO_HIGH |
| OMO_IDENTITY | Agent identity data | OMO_HIGH |
| OMO_SETTINGS | Agent configuration | OMO_MEDIUM |
| OMO_TOKENS | Access tokens | OMO_CRITICAL |
| OMO_WORKSPACE | Workspace settings | OMO_LOW |
| OMO_TASKS | Task configurations | OMO_MEDIUM |

#### Operations

```kotlin
// Store OMO data
vaultRepository.storeOMOIdentity(identity: OMOIdentityData)
vaultRepository.storeOMOCredentials(credentials: OMOCredentials)
vaultRepository.storeOMOSettings(settings: OMOSettings)
vaultRepository.storeOMOTokens(tokens: List<OMOToken>)
vaultRepository.storeOMOWorkspace(workspace: OMOWorkspace)
vaultRepository.storeOMOTasks(tasks: List<OMOTask>)

// Retrieve OMO data
vaultRepository.retrieveOMOIdentity(): OMOIdentityData?
vaultRepository.retrieveOMOCredentials(): OMOCredentials?
vaultRepository.retrieveOMOSettings(): OMOSettings?
vaultRepository.retrieveOMOTokens(): List<OMOToken>
vaultRepository.retrieveOMOWorkspace(): OMOWorkspace?
vaultRepository.retrieveOMOTasks(): List<OMOTask>
```

#### Security Features

- Biometric authentication required for sensitive operations
- AES-256-GCM encryption
- AndroidKeyStore for key storage
- Auto-clear for sensitive data in memory

---

### 3. Agent Studio

A 4-step wizard for creating AI agents with visual workflow builder.

#### Steps

| Step | Name | Purpose |
|------|------|---------|
| 1 | Role Definition | Set agent name, type, description |
| 2 | Skill Selection | Choose agent capabilities |
| 3 | Workflow Builder | Visual Blockly workflow editor |
| 4 | Permissions | Set access permissions and sensitivity |

#### Block Categories

| Category | Color | Blocks |
|----------|-------|--------|
| Triggers | Blue | Message received, User joins, Timer, Schedule |
| Actions | Green | Send message, Send email, API call, Run command |
| Logic | Yellow | If/then/else, Compare, Conditional response |
| Control Flow | Red | Break, Continue, Return, Stop agent |
| Variables | Purple | Set/Get/Increment variable |
| API Calls | Cyan | HTTP request, Database query |

#### Available Blocks (22 total)

**Triggers:**
- `message_received` - Fires when message received
- `user_joins` - Fires when user joins channel
- `timer_expired` - Fires after time interval
- `schedule_triggered` - Fires at scheduled time (cron)

**Actions:**
- `send_message` - Send text message to channel
- `send_email` - Send email notification
- `api_call` - Call external API
- `run_command` - Execute shell command

**Logic:**
- `if_then_else` - Conditional branching
- `compare` - Compare values
- `conditional_response` - Response based on condition

**Control Flow:**
- `break` - Exit loop
- `continue` - Skip to next iteration
- `return` - Return from workflow
- `stop_agent` - Stop agent execution
- `pause_agent` - Pause agent execution

**Variables:**
- `set_variable` - Store value
- `get_variable` - Retrieve value
- `increment_variable` - Increment number

**API Calls:**
- `http_request` - Make HTTP request
- `database_query` - Query database

---

## Test Coverage

### Unit Tests Created

| File | Tests | Lines |
|------|-------|-------|
| AgentBlocksTest.kt | 30+ | 106 |
| BlocklyWebViewTest.kt | 20+ | 169 |
| AgentStudioScreenTest.kt | 20+ | 118 |
| PiiRegistryTest.kt | 15+ | 91 |
| CommandBarTest.kt | 31 | 379 |
| SplitViewLayoutTest.kt | 22 | - |
| ActivityLogTest.kt | 41 | - |
| **Total** | **180+** | **863+** |

### Test Patterns

- JUnit4 with kotlin.test assertions
- Mockk for mocking
- Pure logic function testing
- No Compose UI rendering tests

---

## Navigation

### New Routes

```kotlin
// In AppNavigation.kt
composable(route = "vault") { VaultScreen() }
composable(route = "agent_studio") { AgentStudioScreen() }
```

### Deep Links

- `armorclaw://room/{roomId}` - Opens specific chat room

---

## Dependencies

No new external dependencies were added. The implementation uses:

- Existing Kotlin Multiplatform infrastructure
- Existing Jetpack Compose UI framework
- Existing Koin dependency injection
- Existing SQLDelight database
- Blockly loaded via WebView (no native dependency)

---

## Configuration

### ProGuard/R8

No additional rules required. All new classes follow existing patterns.

### Permissions

No new permissions required. Biometric auth uses existing permission.

---

## Known Limitations

1. **Blockly WebView**: Requires internet connection for initial Blockly load (can be cached locally)
2. **Vault Sync**: PII data does not sync across devices (by design - local only)
3. **Agent Execution**: Visual workflow created but not yet executable (Phase 2)

---

## Future Enhancements

1. **Workflow Execution Engine**: Execute created workflows
2. **Agent Marketplace**: Share/export agent configurations
3. **Multi-device PII Sync**: Optional encrypted sync
4. **Blockly Offline**: Bundle Blockly assets locally
5. **Agent Templates**: Pre-built agent configurations

---

## References

- [Omo Integration Plan](/.sisyphus/plans/omo-integration.md)
- [UAT Checklist](/.sisyphus/uat-checklist.md)
- [Architecture Documentation](/doc/ARCHITECTURE.md)

---

## Changelog

### v1.0 (2026-03-15)
- Initial OMO integration
- SplitViewLayout component
- ActivityLog component
- CommandBar component
- VaultScreen with OMO categories
- AgentStudioScreen 4-step wizard
- BlocklyWebView integration
- 22 AgentBlocks definitions
- PiiRegistry extensions
- VaultRepository OMO methods
- 180+ unit tests
