# Chat Feature

> Unified messaging supporting Regular, Agent, System, and Command messages
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/`

## Overview

The chat feature provides a unified messaging experience for all message types:
- **Regular**: User messages via Matrix
- **Agent**: AI assistant responses
- **System**: Workflow events, notifications
- **Command**: User commands to agents (!)

---

## Critical Functions

### ChatViewModel

```kotlin
// Message handling
fun sendMessage(content: String)              // Auto-detects commands (!)
fun handleRegularMessage(content: String)     // Send via Matrix
fun handleCommandMessage(content, cmd, args)  // Execute command

// Actions
fun handleAgentAction(message, action)        // Agent quick actions
fun handleSystemAction(message, action)       // System event actions
fun retryCommand(message)                     // Retry failed command

// State
fun loadMessages()                            // Initial load
fun loadMoreMessages()                        // Pagination
fun refreshMessages()                         // Pull-to-refresh

// Interactions
fun replyToMessage(message)                   // Set reply target
fun toggleReaction(message, emoji)            // Add/remove reaction
fun markAsRead(messageId)                     // Read receipt
```

### State Flows

| State | Type | Description |
|-------|------|-------------|
| `unifiedMessages` | `List<UnifiedMessage>` | All message types |
| `activeWorkflow` | `WorkflowState?` | Active workflow |
| `agentThinking` | `AgentThinkingState?` | Agent thinking |
| `isAgentRoom` | `Boolean` | Has agent |
| `activeAgent` | `AgentSender?` | Agent info |
| `currentUser` | `UserSender?` | Current user |
| `replyTo` | `UnifiedMessage?` | Reply target |

---

## Unified Message Types

### UnifiedMessage.Regular
User messages from Matrix.

| Property | Type | Description |
|----------|------|-------------|
| id | String | Message ID |
| content | MessageContent | Text/media content |
| status | MessageStatus | SENDING/SENT/DELIVERED/READ |
| reactions | List<Reaction> | Emoji reactions |
| replyTo | String? | Reply target ID |
| isEncrypted | Boolean | E2EE status |

### UnifiedMessage.Agent
AI assistant responses.

| Property | Type | Description |
|----------|------|-------------|
| agentType | AgentType | GENERAL/ANALYSIS/CODE_REVIEW/etc. |
| confidence | Float? | Confidence score |
| sources | List<SourceReference> | Source documents |
| actions | List<AgentAction> | Quick actions |

### UnifiedMessage.System
Workflow events and notifications.

| Property | Type | Description |
|----------|------|-------------|
| eventType | SystemEventType | WORKFLOW_*/USER_*/etc. |
| title | String | Event title |
| description | String? | Event details |
| actions | List<SystemAction> | Available actions |

### UnifiedMessage.Command
User commands to agents.

| Property | Type | Description |
|----------|------|-------------|
| command | String | Command name |
| args | List<String> | Arguments |
| status | CommandStatus | PENDING/EXECUTING/COMPLETED/FAILED |
| result | String? | Execution result |
| executionTime | Long? | Duration in ms |

---

## Command Detection

Commands are detected by the `!` prefix:

```kotlin
val isCommand = content.startsWith("!")
if (isCommand && isAgentRoom) {
    val commandText = content.removePrefix("!").trim()
    val parts = commandText.split(" ")
    val command = parts.first()      // e.g., "analyze"
    val args = parts.drop(1)         // e.g., ["--deep", "doc.pdf"]
    handleCommandMessage(content, command, args)
}
```

---

## Dependencies

### Platform
| Dependency | Purpose |
|------------|---------|
| `MatrixClient` | Send/receive Matrix messages |
| `ControlPlaneStore` | Workflow/agent events |

### Domain
| Dependency | Purpose |
|------------|---------|
| `MessageRepository` | Legacy fallback storage |
| `UnifiedMessage` | Message model |

### UI Components
| Component | Purpose |
|-----------|---------|
| `UnifiedMessageList` | Render all message types |
| `UnifiedChatInput` | Unified input with command mode |
| `WorkflowProgressBanner` | Active workflow display |
| `AgentThinkingIndicator` | Agent thinking animation |

---

## Agent Quick Actions

| AgentType | Actions |
|-----------|---------|
| GENERAL | Help, Analyze, Summarize |
| ANALYSIS | Analyze, Compare, Report |
| CODE_REVIEW | Review, Fix, Explain |
| RESEARCH | Search, Find, Sources |
| WRITING | Write, Edit, Improve |
| TRANSLATION | Translate, Detect |
| SCHEDULING | Schedule, Remind, Calendar |
| WORKFLOW | Start, Status, List |
| PLATFORM_BRIDGE | Connect, Sync, Status |

---

## System Event Actions

| ActionType | Purpose |
|------------|---------|
| CANCEL | Cancel running workflow |
| RETRY | Retry failed operation |
| VERIFY | Start verification |
| ACKNOWLEDGE | Dismiss notification |
| VIEW_DETAILS | View more details |

---

## Files

| File | Location |
|------|----------|
| ChatViewModel | `androidApp/.../viewmodels/ChatViewModel.kt` |
| UnifiedMessage | `shared/.../domain/model/UnifiedMessage.kt` |
| UnifiedMessageList | `shared/.../ui/components/UnifiedMessageList.kt` |
| UnifiedChatInput | `shared/.../ui/components/UnifiedChatInput.kt` |

---

## Legacy Components

### MessageList
**Location:** `chat/components/MessageList.kt`

Scrollable list of messages with loading states.

| Function | Description |
|----------|-------------|
| `MessageList()` | Main list composable |
| `LoadingState()` | Initial loading indicator |
| `LoadingMoreIndicator()` | Pagination loading |
| `ErrorState()` | Error with retry |
| `EmptyState()` | No messages placeholder |

### MessageBubble
**Location:** `chat/components/MessageBubble.kt`

Individual message display component.

| Function | Description |
|----------|-------------|
| `MessageBubble()` | Main bubble composable |
| `OutgoingMessageBubble()` | Sent message style |
| `IncomingMessageBubble()` | Received message style |
| `MessageStatusIcon()` | Status indicator |
| `MessageReactionsRow()` | Reaction display |

### MessageStatus
| Status | Icon | Description |
|--------|------|-------------|
| SENDING | Clock | Message being sent |
| SENT | Check | Message delivered to server |
| DELIVERED | Double check | Message delivered to device |
| READ | Double check (filled) | Message read by recipient |
| FAILED | Error | Send failed |

---

## Related Documentation

- [Home Screen](home-screen.md) - Room list and navigation
- [Threads](threads.md) - Threaded conversations
- [Encryption](encryption.md) - Security implementation
- [Matrix Migration](../MATRIX_MIGRATION.md) - Protocol details
