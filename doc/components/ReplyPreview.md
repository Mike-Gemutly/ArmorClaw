# ReplyPreview Component

> Reply quote preview display
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/ReplyPreview.kt`

## Overview

ReplyPreview displays a quoted message preview when composing a reply, showing the original message context above the input field.

## Functions

### ReplyPreview
```kotlin
@Composable
fun ReplyPreview(
    message: Message,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Reply preview card showing original message being replied to.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `message` | `Message` | Original message being replied to |
| `onCancel` | `() -> Unit` | Cancel reply callback |
| `modifier` | `Modifier` | Optional styling modifier |

---

## Visual Layout

### Reply Preview Above Input
```
┌────────────────────────────────────┐
│ ┌────────────────────────────────┐ │
│ │ ▌👤 Alice                      │ │
│ │ ▌Original message content...   │ │
│ │ └──────────────────────────── ✕│ │  ← Cancel button
│ └────────────────────────────────┘ │
│ ┌────────────────────────────────┐ │
│ │ Type your reply...          📤│ │  ← Input field
│ └────────────────────────────────┘ │
└────────────────────────────────────┘
```

### With Image
```
┌────────────────────────────────────┐
│ ┌────────────────────────────────┐ │
│ │ ▌┌────┐ 👤 Alice               │ │
│ │ ▌│ 🖼 │ Photo                  │ │
│ │ ▌└────┘                        │ │
│ │ └────────────────────────── ✕ │ │
│ └────────────────────────────────┘ │
└────────────────────────────────────┘
```

---

## Component Structure

### Implementation
```kotlin
@Composable
fun ReplyPreview(
    message: Message,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Accent bar
            Box(
                modifier = Modifier
                    .width(4.dp)
                    .height(IntrinsicSize.Max)
                    .background(
                        AccentColor,
                        RoundedCornerShape(2.dp)
                    )
            )

            // Content
            Column(modifier = Modifier.weight(1f)) {
                // Sender name
                Text(
                    text = message.senderName,
                    style = MaterialTheme.typography.labelMedium,
                    color = AccentColor,
                    fontWeight = FontWeight.SemiBold
                )

                Spacer(Modifier.height(4.dp))

                // Message preview
                MessagePreviewContent(message = message)
            }

            // Cancel button
            IconButton(
                onClick = onCancel,
                modifier = Modifier.size(24.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = "Cancel reply",
                    modifier = Modifier.size(16.dp)
                )
            }
        }
    }
}
```

---

### MessagePreviewContent
```kotlin
@Composable
private fun MessagePreviewContent(
    message: Message
) {
    when (message.content.type) {
        MessageType.TEXT -> {
            Text(
                text = message.content.body.take(100).let {
                    if (message.content.body.length > 100) "$it..."
                    else it
                },
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis
            )
        }
        MessageType.IMAGE -> {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Default.Image, null, Modifier.size(16.dp))
                Spacer(Modifier.width(4.dp))
                Text("Photo", style = MaterialTheme.typography.bodySmall)
            }
        }
        MessageType.VIDEO -> {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Default.VideoFile, null, Modifier.size(16.dp))
                Spacer(Modifier.width(4.dp))
                Text("Video", style = MaterialTheme.typography.bodySmall)
            }
        }
        else -> {
            Text("Attachment", style = MaterialTheme.typography.bodySmall)
        }
    }
}
```

---

## Styling

### Colors
| Element | Light Theme | Dark Theme |
|---------|-------------|------------|
| Background | surfaceVariant | surfaceVariant |
| Accent bar | primary | primary |
| Sender name | primary | primary |
| Preview text | onSurfaceVariant | onSurfaceVariant |

### Dimensions
| Element | Size |
|---------|------|
| Accent bar width | 4.dp |
| Corner radius | 12.dp |
| Horizontal padding | 12.dp |
| Vertical padding | 12.dp |
| Cancel icon | 16.dp |

---

## State Management

### Reply State
```kotlin
// In ChatViewModel
private val _replyTo = MutableStateFlow<Message?>(null)
val replyTo: StateFlow<Message?> = _replyTo.asStateFlow()

fun setReplyTarget(message: Message) {
    _replyTo.value = message
}

fun cancelReply() {
    _replyTo.value = null
}
```

### Usage
```kotlin
@Composable
fun ChatInputArea(
    replyTo: Message?,
    onCancelReply: () -> Unit
) {
    Column {
        // Show reply preview if replying
        replyTo?.let { message ->
            ReplyPreview(
                message = message,
                onCancel = onCancelReply
            )
        }

        // Input field
        OutlinedTextField(...)
    }
}
```

---

## Interactions

### Actions
| Action | Trigger | Result |
|--------|---------|--------|
| Cancel | Tap X button | Clear reply target |
| Tap | Tap preview | Scroll to original message |

### Cancel Behavior
- Clears `replyTo` state
- Allows user to cancel without sending
- Keyboard remains visible

---

## Accessibility

### Content Descriptions
- Preview card: "Replying to [sender]: [preview]"
- Cancel button: "Cancel reply"

### Semantic Properties
- Role: Button (for tap to scroll)
- State: None

---

## Related Documentation

- [MessageBubble](MessageBubble.md) - Message display
- [ChatScreen](../screens/ChatScreen.md) - Chat screen
- [ChatViewModel](../viewmodels/ChatViewModel.md) - State management
