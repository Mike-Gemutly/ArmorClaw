# MessageBubble Component

> Individual message display component
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/MessageBubble.kt`

## Overview

MessageBubble renders individual chat messages with support for different message types, status indicators, reactions, and reply quotes.

## Functions

### MessageBubble
```kotlin
@Composable
fun MessageBubble(
    message: Message,
    onReplyClick: (Message) -> Unit,
    onReactionClick: (Message) -> Unit,
    onAttachmentClick: (String) -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Main message bubble composable.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `message` | `Message` | Message data to display |
| `onReplyClick` | `(Message) -> Unit` | Reply action callback |
| `onReactionClick` | `(Message) -> Unit` | Reaction action callback |
| `onAttachmentClick` | `(String) -> Unit` | Attachment tap callback |
| `modifier` | `Modifier` | Optional styling modifier |

---

### OutgoingMessageBubble
```kotlin
@Composable
private fun OutgoingMessageBubble(
    message: Message,
    onReplyClick: (Message) -> Unit,
    onReactionClick: (Message) -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Styled bubble for sent messages (right-aligned).

**Styling:**
- Background: BrandPurple
- Text: White
- Alignment: Right
- Rounded corners: top-right squared

---

### IncomingMessageBubble
```kotlin
@Composable
private fun IncomingMessageBubble(
    message: Message,
    onReplyClick: (Message) -> Unit,
    onReactionClick: (Message) -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Styled bubble for received messages (left-aligned).

**Styling:**
- Background: SurfaceVariant
- Text: OnSurface
- Alignment: Left
- Rounded corners: top-left squared

---

### MessageContent
```kotlin
@Composable
private fun MessageContent(
    message: Message,
    onAttachmentClick: (String) -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Renders message body based on type.

**Supported Types:**
- TEXT - Plain text message
- IMAGE - Image with loading/error states
- VIDEO - Video thumbnail with play button
- AUDIO - Audio player controls
- FILE - File attachment card
- LOCATION - Map preview

---

### MessageHeader
```kotlin
@Composable
private fun MessageHeader(
    message: Message,
    modifier: Modifier = Modifier
)
```

**Description:** Displays sender name and timestamp.

**Elements:**
- Sender name (for group chats)
- Timestamp (relative format)
- Encryption indicator

---

### MessageStatusIcon
```kotlin
@Composable
private fun MessageStatusIcon(
    status: MessageStatus,
    modifier: Modifier = Modifier
)
```

**Description:** Visual indicator for message delivery status.

**Status Icons:**
| Status | Icon | Color |
|--------|------|-------|
| SENDING | Clock | Gray |
| SENT | Check | Gray |
| DELIVERED | DoubleCheck | Gray |
| READ | DoubleCheck | BrandPurple |
| FAILED | Error | Red |

---

### MessageReactionsRow
```kotlin
@Composable
private fun MessageReactionsRow(
    reactions: List<Reaction>,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Horizontal row of reaction chips.

**Features:**
- Shows emoji with count
- Highlights if user reacted
- Click to add/change reaction

---

### ReplyQuote
```kotlin
@Composable
private fun ReplyQuote(
    originalMessage: Message,
    modifier: Modifier = Modifier
)
```

**Description:** Quoted message preview for replies.

**Elements:**
- Colored accent bar
- Original sender name
- Truncated content preview
- Click to scroll to original

---

## Data Models

### Message
```kotlin
data class Message(
    val id: String,
    val roomId: String,
    val senderId: String,
    val content: MessageContent,
    val timestamp: Instant,
    val isOutgoing: Boolean,
    val status: MessageStatus,
    val replyTo: String?,
    val reactions: List<Reaction>
)
```

### MessageStatus
```kotlin
enum class MessageStatus {
    SENDING,
    SENT,
    DELIVERED,
    READ,
    FAILED
}
```

### Reaction
```kotlin
data class Reaction(
    val emoji: String,
    val count: Int,
    val includesMe: Boolean,
    val reactedBy: List<String>
)
```

---

## Styling

### Outgoing Bubble
```kotlin
val outgoingBubbleShape = RoundedCornerShape(
    topStart = 16.dp,
    topEnd = 4.dp,
    bottomStart = 16.dp,
    bottomEnd = 16.dp
)
```

### Incoming Bubble
```kotlin
val incomingBubbleShape = RoundedCornerShape(
    topStart = 4.dp,
    topEnd = 16.dp,
    bottomStart = 16.dp,
    bottomEnd = 16.dp
)
```

### Colors
| Element | Outgoing | Incoming |
|---------|----------|----------|
| Background | BrandPurple | SurfaceVariant |
| Text | White | OnSurface |
| Time | White.copy(0.7) | OnSurface.copy(0.6) |

---

## Interactions

### Long Press Menu
Shows context menu with actions:
- Reply
- React
- Forward
- Copy
- Delete (own messages)

### Reaction Picker
Emoji grid overlay for adding reactions.

---

## Accessibility

### Content Description
```
"[Sender] said: [message content] at [time]. [Status]. [Reactions]"
```

### Semantic Properties
- Role: Role.Button
- State: Selected if reacted

---

## Performance

### Optimizations
- Stable message key for recomposition
- Reaction list memoization
- Image loading with placeholder

---

## Related Documentation

- [ChatScreen](../screens/ChatScreen.md) - Chat screen
- [Chat Feature](../features/chat.md) - Chat functionality
- [ReplyPreview](ReplyPreview.md) - Reply display
