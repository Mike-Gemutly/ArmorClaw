# MessageList Component

> Message list with loading and error states
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/MessageList.kt`

## Overview

MessageList displays a scrollable list of chat messages with support for loading states, pagination, pull-to-refresh, and error handling.

## Functions

### MessageList
```kotlin
@Composable
fun MessageList(
    messages: List<Message>,
    onLoadMore: () -> Unit,
    onRefresh: () -> Unit,
    onReplyClick: (Message) -> Unit,
    onReactionClick: (Message) -> Unit,
    modifier: Modifier = Modifier,
    isLoading: Boolean = false,
    isLoadingMore: Boolean = false,
    isRefreshing: Boolean = false,
    hasMore: Boolean = true,
    error: String? = null
)
```

**Description:** Main message list with all state handling.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `messages` | `List<Message>` | List of messages to display |
| `onLoadMore` | `() -> Unit` | Load older messages |
| `onRefresh` | `() -> Unit` | Pull-to-refresh callback |
| `onReplyClick` | `(Message) -> Unit` | Reply action |
| `onReactionClick` | `(Message) -> Unit` | Reaction action |
| `modifier` | `Modifier` | Optional styling |
| `isLoading` | `Boolean` | Initial loading state |
| `isLoadingMore` | `Boolean` | Pagination loading |
| `isRefreshing` | `Boolean` | Refresh in progress |
| `hasMore` | `Boolean` | More messages available |
| `error` | `String?` | Error message if any |

---

## Component Structure

### List Layout
```
┌────────────────────────────────────┐
│  ┌──────────────────────────────┐  │
│  │ Loading older...             │  │  ← Loading more indicator
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ 👤 Alice                     │  │
│  │    Oldest message            │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ 👤 Bob                       │  │
│  │    Middle message            │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │         Newer message    👤  │  │
│  │              2:30 PM  ✓✓✓   │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │         Latest message   👤  │  │
│  │              2:35 PM  ✓✓✓   │  │
│  └──────────────────────────────┘  │
└────────────────────────────────────┘
```

---

## State Handling

### Loading States
```kotlin
when {
    isLoading -> {
        Box(Modifier.fillMaxSize(), contentAlignment = Center) {
            CircularProgressIndicator()
        }
    }
    error != null -> {
        ErrorState(error = error, onRetry = onRefresh)
    }
    messages.isEmpty() -> {
        EmptyState()
    }
    else -> {
        MessageListContent(...)
    }
}
```

### Empty State
```kotlin
@Composable
fun EmptyState() {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Icon(Icons.Default.ChatBubbleOutline, null, Modifier.size(64.dp))
        Spacer(Modifier.height(16.dp))
        Text("No messages yet", style = MaterialTheme.typography.titleMedium)
        Text("Start the conversation!", style = MaterialTheme.typography.bodyMedium)
    }
}
```

### Error State
```kotlin
@Composable
fun ErrorState(error: String, onRetry: () -> Unit) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Icon(Icons.Default.Error, null, Modifier.size(64.dp), tint = error)
        Spacer(Modifier.height(16.dp))
        Text("Failed to load messages", style = MaterialTheme.typography.titleMedium)
        Text(error, style = MaterialTheme.typography.bodyMedium)
        Spacer(Modifier.height(16.dp))
        Button(onClick = onRetry) {
            Text("Retry")
        }
    }
}
```

---

## Pagination

### Load More Trigger
```kotlin
LazyColumn(
    state = listState,
    reverseLayout = true
) {
    // Load more indicator at top
    if (isLoadingMore) {
        item {
            Box(Modifier.fillMaxWidth().padding(16.dp)) {
                CircularProgressIndicator(Modifier.size(24.dp).align(Center))
            }
        }
    }

    // Message items
    items(messages, key = { it.id }) { message ->
        MessageBubble(message = message, ...)
    }

    // Trigger load more when scrolled to top
    if (listState.firstVisibleItemIndex == 0 && hasMore && !isLoadingMore) {
        LaunchedEffect(Unit) {
            onLoadMore()
        }
    }
}
```

---

## Pull to Refresh

### Implementation
```kotlin
val refreshState = rememberPullRefreshState(
    refreshing = isRefreshing,
    onRefresh = onRefresh
)

Box(Modifier.pullRefresh(refreshState)) {
    LazyColumn(...) {
        // Message items
    }

    PullRefreshIndicator(
        refreshing = isRefreshing,
        state = refreshState,
        modifier = Modifier.align(TopCenter)
    )
}
```

---

## Scroll Behavior

### Auto-Scroll to Bottom
```kotlin
LaunchedEffect(messages.size) {
    if (messages.isNotEmpty() && !isLoadingMore) {
        listState.animateScrollToItem(0)
    }
}
```

### Scroll Position Preservation
- Maintain scroll position on configuration changes
- Preserve position when new messages arrive
- Smooth scroll to new outgoing messages

---

## Performance Optimizations

### Key Generation
```kotlin
items(
    items = messages,
    key = { message -> message.id }
) { message ->
    MessageBubble(message = message, ...)
}
```

### Diffing Strategy
- Use stable message IDs for keys
- Minimize recomposition on updates
- Efficient list diffing with keys

---

## Date Separators

### Date Header
```kotlin
@Composable
fun DateSeparator(date: LocalDate) {
    Box(
        Modifier
            .fillMaxWidth()
            .padding(vertical = 16.dp),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = date.format(DateTimeFormatter.ofPattern("MMMM d, yyyy")),
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}
```

### Date Grouping
```kotlin
// Group messages by date
val groupedMessages = messages.groupBy {
    it.timestamp.toLocalDateTime(TimeZone.currentSystemDefault()).date
}
```

---

## Typing Indicator Integration

### At Bottom of List
```kotlin
LazyColumn {
    items(messages) { ... }

    // Typing indicator
    if (typingIndicator.isActive) {
        item {
            TypingIndicator(users = typingIndicator.users)
        }
    }
}
```

---

## Accessibility

### Content Descriptions
- List: "Messages, [count] items"
- Loading: "Loading messages"
- Error: "Failed to load: [error]"

### Semantic Properties
- Role: Role.List
- Collection info: itemCount

---

## Related Documentation

- [MessageBubble](MessageBubble.md) - Individual message display
- [ChatScreen](../screens/ChatScreen.md) - Chat screen
- [ChatViewModel](../viewmodels/ChatViewModel.md) - State management
