# ChatViewModel

> State management for ChatScreen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/ChatViewModel.kt`

## Overview

ChatViewModel manages the state for the chat screen, handling message loading, sending, reactions, replies, and search functionality.

## Class Definition

```kotlin
class ChatViewModel(
    private val roomId: String
) : ViewModel()
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `roomId` | `String` | Unique identifier for the chat room |

---

## State Flows

### uiState
```kotlin
private val _uiState = MutableStateFlow<ChatUiState>(ChatUiState.Initial)
val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()
```

**Description:** Overall screen state.

**States:**
| State | Description |
|-------|-------------|
| `Initial` | Screen just created |
| `Loading` | Loading messages |
| `MessagesLoaded` | Messages loaded successfully |
| `MessagesRefreshed` | Refresh completed |
| `Error(message)` | Error occurred |

---

### messageListState
```kotlin
private val _messageListState = MutableStateFlow(MessageListState())
val messageListState: StateFlow<MessageListState> = _messageListState.asStateFlow()
```

**Description:** Message list state including messages and loading status.

**MessageListState:**
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

---

### typingIndicator
```kotlin
private val _typingIndicator = MutableStateFlow(TypingIndicator())
val typingIndicator: StateFlow<TypingIndicator> = _typingIndicator.asStateFlow()
```

**Description:** Shows which users are currently typing.

**TypingIndicator:**
```kotlin
data class TypingIndicator(
    val users: List<String> = emptyList(),
    val isActive: Boolean = false
)
```

---

### encryptionStatus
```kotlin
private val _encryptionStatus = MutableStateFlow(EncryptionStatus.VERIFIED)
val encryptionStatus: StateFlow<EncryptionStatus> = _encryptionStatus.asStateFlow()
```

**Description:** Current encryption state of the conversation.

---

### isSearchActive
```kotlin
private val _isSearchActive = MutableStateFlow(false)
val isSearchActive: StateFlow<Boolean> = _isSearchActive.asStateFlow()
```

**Description:** Whether search overlay is visible.

---

### searchQuery
```kotlin
private val _searchQuery = MutableStateFlow("")
val searchQuery: StateFlow<String> = _searchQuery.asStateFlow()
```

**Description:** Current search query text.

---

### replyTo
```kotlin
private val _replyTo = MutableStateFlow<Message?>(null)
val replyTo: StateFlow<Message?> = _replyTo.asStateFlow()
```

**Description:** Message being replied to (null if no reply).

---

## Actions

### loadMessages
```kotlin
fun loadMessages()
```

**Description:** Loads initial messages for the room.

**Flow:**
1. Set loading state
2. Fetch messages from repository
3. Update message list state
4. Set loaded state

---

### refreshMessages
```kotlin
fun refreshMessages()
```

**Description:** Pull-to-refresh handler.

**Flow:**
1. Set refreshing state
2. Fetch latest messages
3. Update list with new messages
4. Set refreshed state

---

### loadMoreMessages
```kotlin
fun loadMoreMessages()
```

**Description:** Pagination handler for loading older messages.

**Flow:**
1. Set loadingMore state
2. Fetch older messages
3. Prepend to existing list
4. Update hasMore flag

---

### sendMessage
```kotlin
fun sendMessage(content: String)
```

**Description:** Sends a new message.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `content` | `String` | Message text content |

**Flow:**
1. Validate content (not blank)
2. Create new message with SENDING status
3. Add to beginning of list
4. Clear reply target
5. Simulate send (would call repository in production)
6. Update status: SENT → DELIVERED → READ

**Message Creation:**
```kotlin
val newMessage = Message(
    id = "msg_${System.currentTimeMillis()}",
    roomId = roomId,
    senderId = "user",
    content = MessageContent(
        type = MessageType.TEXT,
        body = content
    ),
    timestamp = Clock.System.now(),
    isOutgoing = true,
    status = MessageStatus.SENDING,
    replyTo = _replyTo.value?.id
)
```

---

### replyToMessage
```kotlin
fun replyToMessage(message: Message)
```

**Description:** Sets a message as the reply target.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `message` | `Message` | Message to reply to |

---

### cancelReply
```kotlin
fun cancelReply()
```

**Description:** Clears the reply target.

---

### toggleReaction
```kotlin
fun toggleReaction(message: Message, emoji: String)
```

**Description:** Adds or removes a reaction from a message.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `message` | `Message` | Target message |
| `emoji` | `String` | Reaction emoji |

**Behavior:**
- If reaction exists: Remove it
- If reaction doesn't exist: Add it

---

### toggleSearch
```kotlin
fun toggleSearch()
```

**Description:** Toggles search overlay visibility.

**Behavior:**
- Toggles `isSearchActive`
- Clears `searchQuery` when closing

---

### onSearchQueryChange
```kotlin
fun onSearchQueryChange(query: String)
```

**Description:** Updates search query.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `query` | `String` | New search text |

---

### clearEvent
```kotlin
fun clearEvent()
```

**Description:** Clears any pending UI events.

---

## Initialization

```kotlin
init {
    loadMessages()
}
```

Messages are automatically loaded when the ViewModel is created.

---

## Sample Data

### Sample Messages
```kotlin
private val sampleMessages = listOf(
    Message(
        id = "msg_1",
        roomId = "room_1",
        senderId = "assistant",
        content = MessageContent(
            type = MessageType.TEXT,
            body = "Hello! I'm your AI assistant..."
        ),
        timestamp = Clock.System.now().minus(2.hours),
        isOutgoing = false,
        status = MessageStatus.SENT,
        reactions = listOf(Reaction(emoji = "👋", count = 3))
    ),
    // ... more messages
)
```

---

## Usage Example

```kotlin
@Composable
fun ChatScreen(roomId: String) {
    val viewModel: ChatViewModel = viewModel(key = roomId) {
        ChatViewModel(roomId)
    }

    val messageListState by viewModel.messageListState.collectAsState()
    val replyTo by viewModel.replyTo.collectAsState()
    val isSearchActive by viewModel.isSearchActive.collectAsState()

    // ... UI implementation
}
```

---

## Testing

### Unit Tests
```kotlin
@Test
fun sendMessage_addsMessageToList() = runTest {
    val viewModel = ChatViewModel("room_1")

    viewModel.sendMessage("Hello")

    val state = viewModel.messageListState.value
    assertEquals(1, state.messages.size)
    assertEquals("Hello", state.messages.first().content.body)
}

@Test
fun toggleReaction_addsAndRemovesReaction() = runTest {
    val viewModel = ChatViewModel("room_1")
    viewModel.loadMessages()

    val message = viewModel.messageListState.value.messages.first()
    viewModel.toggleReaction(message, "👍")

    // Verify reaction added
    // Toggle again to remove
}
```

---

## Related Documentation

- [ChatScreen](../screens/ChatScreen.md) - Chat screen UI
- [Chat Feature](../features/chat.md) - Chat functionality
- [MessageBubble](../components/MessageBubble.md) - Message display
