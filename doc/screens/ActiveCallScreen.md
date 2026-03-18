# ActiveCallScreen

> Active voice call screen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/call/ActiveCallScreen.kt`

## Overview

ActiveCallScreen displays the UI during an active voice call, showing caller information, call duration, and control buttons.

## Functions

### ActiveCallScreen
```kotlin
@Composable
fun ActiveCallScreen(
    callerName: String,
    callerAvatar: String?,
    callDuration: Long,
    isMuted: Boolean,
    isSpeakerOn: Boolean,
    callStatus: CallStatus,
    onMuteToggle: () -> Unit,
    onSpeakerToggle: () -> Unit,
    onEndCall: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Main screen for active voice calls.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `callerName` | `String` | Name of caller |
| `callerAvatar` | `String?` | Avatar URL |
| `callDuration` | `Long` | Duration in seconds |
| `isMuted` | `Boolean` | Mute state |
| `isSpeakerOn` | `Boolean` | Speaker state |
| `callStatus` | `CallStatus` | Current call status |
| `onMuteToggle` | `() -> Unit` | Toggle mute |
| `onSpeakerToggle` | `() -> Unit` | Toggle speaker |
| `onEndCall` | `() -> Unit` | End call |
| `modifier` | `Modifier` | Optional styling |

---

## Screen Layout

### Active Call
```
┌────────────────────────────────────┐
│                                    │
│         ┌─────────┐                │
│         │   👤    │                │
│         │ Avatar  │                │
│         └─────────┘                │
│                                    │
│         Alice Johnson              │
│         03:45                      │  ← Duration
│                                    │
│    ┌─────────────────────────┐     │
│    │     ▢  ▢  ▢  ▢  ▢      │     │  ← Audio visualizer
│    └─────────────────────────┘     │
│                                    │
│         🔒 End-to-end encrypted    │
│                                    │
│  ┌─────┐   ┌─────┐   ┌─────┐      │
│  │ 🎤  │   │ 📞  │   │ 🔊  │      │
│  │Mute │   │ End │   │Speak│      │
│  └─────┘   └─────┘   └─────┘      │
│                                    │
└────────────────────────────────────┘
```

### Connecting State
```
┌────────────────────────────────────┐
│                                    │
│         ┌─────────┐                │
│         │   👤    │                │
│         └─────────┘                │
│                                    │
│         Alice Johnson              │
│         Calling...                 │  ← Status
│                                    │
│              ◌                     │  ← Loading
│                                    │
│  ┌─────┐                           │
│  │ 📞  │                           │
│  │ End │                           │
│  └─────┘                           │
│                                    │
└────────────────────────────────────┘
```

---

## Components

### CallHeader
```kotlin
@Composable
private fun CallHeader(
    callerName: String,
    callerAvatar: String?,
    callStatus: CallStatus,
    duration: Long
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Avatar
        Box(
            modifier = Modifier.size(120.dp),
            contentAlignment = Alignment.Center
        ) {
            if (callerAvatar != null) {
                AsyncImage(
                    model = callerAvatar,
                    contentDescription = "Caller avatar",
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(CircleShape)
                )
            } else {
                PlaceholderAvatar(name = callerName)
            }
        }

        // Name
        Text(
            text = callerName,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold
        )

        // Status or duration
        when (callStatus) {
            CallStatus.CONNECTING -> {
                Text("Calling...", style = MaterialTheme.typography.bodyLarge)
            }
            CallStatus.ACTIVE -> {
                Text(
                    text = formatDuration(duration),
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.primary
                )
            }
            CallStatus.ENDING -> {
                Text("Ending call...", style = MaterialTheme.typography.bodyLarge)
            }
            else -> {}
        }
    }
}
```

### EncryptionIndicator
```kotlin
@Composable
private fun EncryptionIndicator() {
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = Icons.Default.Lock,
            contentDescription = null,
            modifier = Modifier.size(16.dp),
            tint = MaterialTheme.colorScheme.primary
        )
        Text(
            text = "End-to-end encrypted",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}
```

---

## Duration Formatting

```kotlin
fun formatDuration(seconds: Long): String {
    val minutes = seconds / 60
    val remainingSeconds = seconds % 60
    return "%02d:%02d".format(minutes, remainingSeconds)
}
```

---

## Call Status

```kotlin
enum class CallStatus {
    CONNECTING,   // Establishing connection
    ACTIVE,       // Call in progress
    ON_HOLD,      // Call on hold
    ENDING,       // Ending call
    ENDED         // Call ended
}
```

---

## Animations

### Avatar Pulse (During Ringing)
```kotlin
val infiniteTransition = rememberInfiniteTransition(label = "pulse")
val scale by infiniteTransition.animateFloat(
    initialValue = 1f,
    targetValue = 1.1f,
    animationSpec = infiniteRepeatable(
        animation = tween(1000, easing = LinearEasing),
        repeatMode = RepeatMode.Reverse
    ),
    label = "scale"
)

Box(modifier = Modifier.scale(scale)) {
    Avatar(...)
}
```

---

## State Management

### Call State
```kotlin
data class CallState(
    val callId: String?,
    val callerId: String?,
    val callerName: String?,
    val callerAvatar: String?,
    val status: CallStatus,
    val duration: Long,
    val isMuted: Boolean,
    val isSpeakerOn: Boolean,
    val audioLevel: Float
)
```

### Duration Timer
```kotlin
LaunchedEffect(callStatus) {
    if (callStatus == CallStatus.ACTIVE) {
        while (true) {
            delay(1000)
            _callState.update { it.copy(duration = it.duration + 1) }
        }
    }
}
```

---

## Related Documentation

- [CallControls](../components/CallControls.md) - Call control buttons
- [AudioVisualizer](../components/AudioVisualizer.md) - Audio visualization
- [IncomingCallDialog](IncomingCallDialog.md) - Incoming call dialog
- [Voice Calls](../features/voice-calls.md) - Call feature
