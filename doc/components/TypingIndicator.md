# TypingIndicator Component

> Typing animation indicator
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/TypingIndicator.kt`

## Overview

TypingIndicator displays an animated visual indication when other users are typing in the current chat room.

## Functions

### TypingIndicator
```kotlin
@Composable
fun TypingIndicator(
    users: List<String>,
    modifier: Modifier = Modifier
)
```

**Description:** Animated typing indicator showing which users are typing.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `users` | `List<String>` | List of typing user names |
| `modifier` | `Modifier` | Optional styling modifier |

---

### TypingDots
```kotlin
@Composable
private fun TypingDots(
    modifier: Modifier = Modifier
)
```

**Description:** Animated bouncing dots animation.

---

## Visual Layout

### Single User
```
┌────────────────────────────────────┐
│  Alice is typing ● ● ●             │
└────────────────────────────────────┘
```

### Multiple Users
```
┌────────────────────────────────────┐
│  Alice and Bob are typing ● ● ●    │
└────────────────────────────────────┘
```

### Many Users
```
┌────────────────────────────────────┐
│  Several people are typing ● ● ●   │
└────────────────────────────────────┘
```

---

## Animation

### Dot Animation
```kotlin
@Composable
private fun AnimatedDot(
    delay: Int
) {
    var visible by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        delay(delay)
        visible = true
    }

    val alpha by animateFloatAsState(
        targetValue = if (visible) 1f else 0.3f,
        animationSpec = infiniteRepeatable(
            animation = tween(600, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "dot_$delay"
    )

    Box(
        modifier = Modifier
            .size(8.dp)
            .alpha(alpha)
            .background(
                MaterialTheme.colorScheme.primary,
                CircleShape
            )
    )
}
```

### Animation Timing
| Dot | Delay | Duration |
|-----|-------|----------|
| 1 | 0ms | 600ms |
| 2 | 200ms | 600ms |
| 3 | 400ms | 600ms |

---

## Text Formatting

### User Name Display
```kotlin
fun formatTypingText(users: List<String>): String {
    return when {
        users.isEmpty() -> ""
        users.size == 1 -> "${users[0]} is typing"
        users.size == 2 -> "${users[0]} and ${users[1]} are typing"
        users.size <= 5 -> {
            val names = users.dropLast(1).joinToString(", ")
            "$names, and ${users.last()} are typing"
        }
        else -> "Several people are typing"
    }
}
```

---

## Full Implementation

```kotlin
@Composable
fun TypingIndicator(
    users: List<String>,
    modifier: Modifier = Modifier
) {
    if (users.isEmpty()) return

    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // User names
        Text(
            text = formatTypingText(users),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        // Animated dots
        Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            repeat(3) { index ->
                AnimatedDot(delay = index * 200)
            }
        }
    }
}
```

---

## State Management

### TypingIndicator Model
```kotlin
data class TypingIndicator(
    val users: List<String> = emptyList(),
    val isActive: Boolean = false
)
```

### ViewModel Integration
```kotlin
// In ChatViewModel
private val _typingIndicator = MutableStateFlow(TypingIndicator())
val typingIndicator: StateFlow<TypingIndicator> = _typingIndicator.asStateFlow()

fun updateTypingUsers(users: List<String>) {
    _typingIndicator.value = TypingIndicator(
        users = users,
        isActive = users.isNotEmpty()
    )
}
```

---

## Usage Example

```kotlin
@Composable
fun ChatContent(
    typingIndicator: TypingIndicator
) {
    Column {
        // Message list
        MessageList(...)

        // Typing indicator at bottom
        if (typingIndicator.isActive) {
            TypingIndicator(users = typingIndicator.users)
        }

        // Input area
        ChatInput(...)
    }
}
```

---

## Accessibility

### Content Description
```kotlin
Modifier.semantics {
    contentDescription = formatTypingText(users)
}
```

### Live Region
```kotlin
Modifier.semantics {
    liveRegion = LiveRegionMode.Polite
}
```

---

## Performance

### Optimization
- Only recompose when users list changes
- Memoize formatted text
- Efficient animation cleanup

---

## Related Documentation

- [ChatScreen](../screens/ChatScreen.md) - Chat screen
- [ChatViewModel](../viewmodels/ChatViewModel.md) - State management
- [MessageList](MessageList.md) - Message list
