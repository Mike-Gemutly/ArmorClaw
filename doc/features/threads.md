# Threads Feature

> Threaded conversations for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/`

## Overview

The threads feature allows users to create and view threaded conversations within chat rooms. Threads help organize discussions by keeping replies to specific messages grouped together.

## Feature Components

### ThreadView
**Location:** `chat/components/ThreadView.kt`

Displays a threaded conversation with all replies to a parent message.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `ThreadView()` | Main thread display | `thread`, `onClose`, `onReply` |
| `ThreadHeader()` | Thread header with parent message | `parentMessage`, `replyCount` |
| `ThreadRepliesList()` | List of replies in thread | `replies`, `onMessageClick` |
| `ThreadReplyInput()` | Input field for thread replies | `onSend`, `onCancel` |

#### Thread Layout
```
┌────────────────────────────────────┐
│  ← Thread (3 replies)         ⋮   │  ← Thread header
├────────────────────────────────────┤
│  ┌──────────────────────────────┐  │
│  │ 👤 Alice                     │  │
│  │    Original message content  │  │  ← Parent message
│  │    2:30 PM                   │  │
│  └──────────────────────────────┘  │
├────────────────────────────────────┤
│  ┌──────────────────────────────┐  │
│  │    👤 Bob                    │  │
│  │    Reply to the message...   │  │  ← Thread replies
│  │    2:32 PM                   │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │    👤 Carol                  │  │
│  │    Another reply...          │  │
│  │    2:35 PM                   │  │
│  └──────────────────────────────┘  │
├────────────────────────────────────┤
│  [    Reply to thread...      ] 📤 │  ← Reply input
└────────────────────────────────────┘
```

---

### ThreadIndicator
**Location:** `chat/components/ThreadIndicator.kt`

Visual indicator showing that a message has a thread attached.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `ThreadIndicator()` | Thread indicator chip | `replyCount`, `onClick`, `isExpanded` |
| `ThreadPreviewText()` | Preview of last reply | `lastReply` |

#### Visual States

| State | Appearance |
|-------|------------|
| Collapsed | Small chip with reply count icon |
| Expanded | Shows preview of last reply |
| Hovered | Slight elevation change |

#### Indicator Layout
```
┌────────────────────────────────────┐
│  Message content here...           │
│                                    │
│  💬 3 replies  →                   │  ← Thread indicator
└────────────────────────────────────┘
```

---

### ThreadBadge
**Location:** `chat/components/ThreadBadge.kt`

Badge showing unread thread count on room list items.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `ThreadBadge()` | Unread thread badge | `count`, `modifier` |
| `UnreadThreadDot()` | Small indicator dot | `modifier` |

#### Badge Styles

| Count | Display |
|-------|---------|
| 0 | Hidden |
| 1-99 | Number in circle |
| 100+ | "99+" in circle |

#### Badge Layout
```
┌─────────────────────────────────┐
│  🛡️ General              (3)   │  ← Unread thread badge
│      Last message preview...    │
└─────────────────────────────────┘
```

---

## Data Models

### Thread
```kotlin
data class Thread(
    val id: String,
    val parentMessageId: String,
    val roomId: String,
    val replies: List<Message>,
    val replyCount: Int,
    val lastReplyAt: Instant?,
    val lastReplyBy: String?,
    val isResolved: Boolean
)
```

### ThreadPreview
```kotlin
data class ThreadPreview(
    val threadId: String,
    val parentMessageId: String,
    val replyCount: Int,
    val lastReplyPreview: String?,
    val lastReplyAt: Instant?,
    val hasUnread: Boolean
)
```

---

## State Management

### ThreadState
```kotlin
data class ThreadState(
    val thread: Thread?,
    val isLoading: Boolean,
    val isSending: Boolean,
    val error: String?,
    val replyText: String
)
```

### ThreadActions
| Action | Description |
|--------|-------------|
| `loadThread(messageId)` | Load thread for a message |
| `sendReply(content)` | Send a reply in the thread |
| `closeThread()` | Close thread view |
| `markAsRead()` | Mark thread as read |

---

## User Interactions

### Thread Actions
| Action | Trigger | Result |
|--------|---------|--------|
| Open Thread | Tap thread indicator | Expand thread view |
| Reply | Type and send | Add reply to thread |
| Close | Back button / X | Collapse thread view |
| Resolve | Long press → Resolve | Mark thread as resolved |

### Indicators
| Indicator | Meaning |
|-----------|---------|
| 💬 icon | Thread exists |
| Number | Reply count |
| Blue dot | Unread replies |
| Green check | Thread resolved |

---

## Performance

### Optimizations
- Lazy loading of thread replies
- Pagination for threads with many replies
- Cached thread previews
- Efficient diffing for reply updates

---

## Accessibility

### Content Descriptions
- Thread indicator: "Thread with X replies"
- Badge: "X unread thread messages"
- Reply button: "Reply to thread"

---

## Related Documentation

- [Chat Feature](chat.md) - Main chat functionality
- [MessageBubble](../components/MessageBubble.md) - Message display
- [ChatViewModel](../viewmodels/ChatViewModel.md) - State management
