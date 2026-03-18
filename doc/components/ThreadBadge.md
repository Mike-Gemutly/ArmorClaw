# ThreadBadge Component

> Unread thread indicator badge
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/ThreadBadge.kt`

## Overview

ThreadBadge displays an unread count indicator for threaded conversations, appearing on room list items and message bubbles.

## Functions

### ThreadBadge
```kotlin
@Composable
fun ThreadBadge(
    count: Int,
    modifier: Modifier = Modifier
)
```

**Description:** Badge showing unread thread message count.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `count` | `Int` | Unread message count |
| `modifier` | `Modifier` | Optional styling |

---

### UnreadThreadDot
```kotlin
@Composable
fun UnreadThreadDot(
    modifier: Modifier = Modifier
)
```

**Description:** Small indicator dot for unread thread presence.

---

## Visual Layout

### Badge on Room Item
```
┌────────────────────────────────────┐
│  👤 Alice              (3)        │
│      Hey, check this out!          │
└────────────────────────────────────┘
```

### Badge Variations
```
(3)     ← Small count
(99+)   ← Large count capped
●       ← Dot indicator only
```

---

## Implementation

### ThreadBadge
```kotlin
@Composable
fun ThreadBadge(
    count: Int,
    modifier: Modifier = Modifier
) {
    if (count <= 0) return

    val displayText = when {
        count > 99 -> "99+"
        else -> count.toString()
    }

    Surface(
        modifier = modifier,
        shape = CircleShape,
        color = MaterialTheme.colorScheme.primary,
        contentColor = MaterialTheme.colorScheme.onPrimary
    ) {
        Text(
            text = displayText,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(
                horizontal = if (count > 9) 6.dp else 8.dp,
                vertical = 2.dp
            )
        )
    }
}
```

### UnreadThreadDot
```kotlin
@Composable
fun UnreadThreadDot(
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier
            .size(8.dp)
            .background(
                color = MaterialTheme.colorScheme.primary,
                shape = CircleShape
            )
    )
}
```

---

## Color Scheme

### Badge Colors
| Theme | Background | Text |
|-------|------------|------|
| Light | primary | onPrimary |
| Dark | primary | onPrimary |

### Dot Colors
| State | Color |
|-------|-------|
| Unread | primary |
| Mention | error |

---

## Size Variations

| Count | Width | Padding |
|-------|-------|---------|
| 1-9 | Auto | 8dp horizontal |
| 10-99 | Auto | 6dp horizontal |
| 99+ | Auto | 4dp horizontal |

---

## Animation

### Count Change Animation
```kotlin
@Composable
fun AnimatedThreadBadge(
    count: Int,
    modifier: Modifier = Modifier
) {
    AnimatedContent(
        targetState = count,
        transitionSpec = {
            slideInVertically { it } with
                slideOutVertically { -it }
        },
        label = "badge_count"
    ) { targetCount ->
        ThreadBadge(count = targetCount, modifier = modifier)
    }
}
```

---

## Usage Examples

### In RoomListItem
```kotlin
@Composable
fun RoomListItem(
    room: Room,
    onClick: () -> Unit
) {
    ListItem(
        headlineContent = { Text(room.name) },
        supportingContent = { Text(room.lastMessage) },
        trailingContent = {
            if (room.unreadThreadCount > 0) {
                ThreadBadge(count = room.unreadThreadCount)
            }
        },
        onClick = onClick
    )
}
```

### On MessageBubble
```kotlin
@Composable
fun MessageBubbleWithThread(
    message: Message,
    threadInfo: ThreadInfo?
) {
    Column {
        MessageBubble(message = message)

        if (threadInfo != null && threadInfo.unreadCount > 0) {
            Row(
                modifier = Modifier.padding(start = 8.dp, top = 4.dp)
            ) {
                UnreadThreadDot()
                Spacer(Modifier.width(4.dp))
                Text(
                    text = "${threadInfo.unreadCount} new replies",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.primary
                )
            }
        }
    }
}
```

---

## Accessibility

### Content Descriptions
```kotlin
Modifier.semantics {
    contentDescription = "$count unread thread messages"
}
```

---

## Related Documentation

- [Threads](../features/threads.md) - Thread feature
- [ThreadIndicator](ThreadIndicator.md) - Thread indicator
- [ThreadView](ThreadView.md) - Thread view
