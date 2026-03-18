# ChatScreen

> Main messaging screen for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/ChatScreen_enhanced.kt`

## Overview

ChatScreen is the primary messaging interface where users view and send messages, manage replies, add reactions, and interact with chat content.

## Screen Layout

```
┌────────────────────────────────────┐
│  ← Room Name         🔒 🔍 ⋮       │
├────────────────────────────────────┤
│                                    │
│  ┌──────────────────────────────┐  │
│  │  Replying to: Original msg   │  │  ← Reply preview
│  └──────────────────────────────┘  │
│                                    │
│  ┌──────────────────────────────┐  │
│  │ 👤 Alice                     │  │
│  │    Hello! How are you?       │  │
│  │    2:30 PM      ✓✓          │  │
│  │    ❤️ 3  👍 2                │  │
│  └──────────────────────────────┘  │
│                                    │
│  ┌──────────────────────────────┐  │
│  │         Hi Alice! Great!  👤 │  │
│  │         2:31 PM    ✓✓✓      │  │
│  └──────────────────────────────┘  │
│                                    │
├────────────────────────────────────┤
│  🎤 [    Type a message...    ] 📎 │  ← Input area
└────────────────────────────────────┘
```

## Functions

### ChatScreenEnhanced
```kotlin
@Composable
fun ChatScreenEnhanced(
    roomId: String,
    viewModel: ChatViewModel = viewModel(key = roomId) { ChatViewModel(roomId) },
    onNavigateBack: () -> Unit
)
```

**Description:** Main chat screen composable that orchestrates all chat components.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `roomId` | `String` | Unique identifier for the chat room |
| `viewModel` | `ChatViewModel` | ViewModel for state management |
| `onNavigateBack` | `() -> Unit` | Callback for back navigation |

---

### ChatTopBar
```kotlin
@Composable
private fun ChatTopBar(
    roomName: String,
    encryptionStatus: EncryptionStatus,
    typingIndicator: TypingIndicator,
    onNavigateBack: () -> Unit,
    onSearchClick: () -> Unit,
    onMenuClick: () -> Unit
)
```

**Description:** Top app bar with room info and actions.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `roomName` | `String` | Display name of the room |
| `encryptionStatus` | `EncryptionStatus` | Current encryption state |
| `typingIndicator` | `TypingIndicator` | Who is currently typing |
| `onNavigateBack` | `() -> Unit` | Back navigation callback |
| `onSearchClick` | `() -> Unit` | Open search callback |
| `onMenuClick` | `() -> Unit` | Open menu callback |

---

### ChatContent
```kotlin
@Composable
private fun ChatContent(
    messageListState: MessageListState,
    typingIndicator: TypingIndicator,
    onLoadMore: () -> Unit,
    onRefresh: () -> Unit,
    onReplyClick: (Message) -> Unit,
    onReactionClick: (Message) -> Unit
)
```

**Description:** Main content area containing the message list.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `messageListState` | `MessageListState` | State of message list |
| `typingIndicator` | `TypingIndicator` | Typing users |
| `onLoadMore` | `() -> Unit` | Load older messages |
| `onRefresh` | `() -> Unit` | Pull-to-refresh callback |
| `onReplyClick` | `(Message) -> Unit` | Reply to message |
| `onReactionClick` | `(Message) -> Unit` | Add reaction |

---

### ChatInputArea
```kotlin
@Composable
private fun ChatInputArea(
    replyTo: Message?,
    onSendMessage: (String) -> Unit,
    onReplyClick: (Message) -> Unit,
    onCancelReply: () -> Unit,
    onAttachmentClick: () -> Unit,
    onVoiceInputClick: () -> Unit
)
```

**Description:** Message input area with reply preview.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `replyTo` | `Message?` | Message being replied to |
| `onSendMessage` | `(String) -> Unit` | Send message callback |
| `onReplyClick` | `(Message) -> Unit` | Set reply target |
| `onCancelReply` | `() -> Unit` | Cancel reply |
| `onAttachmentClick` | `() -> Unit` | Add attachment |
| `onVoiceInputClick` | `() -> Unit` | Voice input |

---

### SearchOverlay
```kotlin
@Composable
private fun SearchOverlay(
    query: String,
    results: List<SearchResult>,
    onQueryChange: (String) -> Unit,
    onResultClick: (Message) -> Unit,
    onClose: () -> Unit
)
```

**Description:** Search overlay for finding messages.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `query` | `String` | Current search query |
| `results` | `List<SearchResult>` | Search results |
| `onQueryChange` | `(String) -> Unit` | Query update callback |
| `onResultClick` | `(Message) -> Unit` | Result selection |
| `onClose` | `() -> Unit` | Close search |

---

## State Management

### ChatUiState
```kotlin
sealed class ChatUiState {
    object Initial : ChatUiState()
    object Loading : ChatUiState()
    object MessagesLoaded : ChatUiState()
    object MessagesRefreshed : ChatUiState()
    data class Error(val message: String) : ChatUiState()
}
```

### MessageListState
```kotlin
data class MessageListState(
    val messages: List<Message> = emptyList(),
    val isLoading: Boolean = false,
    val isLoadingMore: Boolean = false,
    val isRefreshing: Boolean = false,
    val hasMore: Boolean = true,
    val error: String? = null
)
```

### TypingIndicator
```kotlin
data class TypingIndicator(
    val users: List<String> = emptyList(),
    val isActive: Boolean = false
)
```

---

## User Interactions

### Message Actions
| Action | Trigger | Result |
|--------|---------|--------|
| Reply | Long press → Reply | Set reply target |
| React | Long press → React | Show emoji picker |
| Forward | Long press → Forward | Select recipient |
| Copy | Long press → Copy | Copy to clipboard |
| Delete | Long press → Delete | Remove message |

### Input Actions
| Action | Trigger | Result |
|--------|---------|--------|
| Send | Tap send button | Transmit message |
| Attach | Tap paperclip | Open file picker |
| Voice | Tap microphone | Start recording |
| Cancel Reply | Tap X on preview | Clear reply target |

---

## Encryption Display

### Status Levels
| Status | Icon | Color | Description |
|--------|------|-------|-------------|
| VERIFIED | ✅ | Green | All devices verified |
| UNVERIFIED | ⚠️ | Yellow | Some devices unverified |
| UNENCRYPTED | ❌ | Red | No encryption |
| NONE | ℹ️ | Gray | Encryption unavailable |

---

## Animations

### Message Entry
- Fade in from appropriate side
- Slide animation (200ms)
- Status indicator pulse

### Typing Indicator
- Three dots with staggered bounce
- 1.5 second animation loop

### Send Animation
- Message slides up from input
- Status transitions: Sending → Sent → Delivered → Read

---

## Accessibility

### Content Descriptions
- Message: "[Sender] said: [content] at [time], [status]"
- Reaction: "[Emoji] reaction by [count] people"
- Status: "Message [status]"

### Actions
- Long press for message actions
- Swipe gestures announced

---

## Performance

### Optimizations
- Lazy column for message list
- Message diffing for updates
- Image loading with Coil
- Memory-efficient reactions

---

## Related Documentation

- [Chat Feature](../features/chat.md) - Full chat documentation
- [MessageBubble](../components/MessageBubble.md) - Message display
- [ChatViewModel](../viewmodels/ChatViewModel.md) - State management
