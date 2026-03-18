# CallControls Component

> Voice call action buttons
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/call/CallControls.kt`

## Overview

CallControls provides the user interface for controlling active voice calls including mute, speaker, and end call actions.

## Functions

### CallControls
```kotlin
@Composable
fun CallControls(
    isMuted: Boolean,
    isSpeakerOn: Boolean,
    onMuteToggle: () -> Unit,
    onSpeakerToggle: () -> Unit,
    onEndCall: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Row of call control buttons for active calls.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `isMuted` | `Boolean` | Current mute state |
| `isSpeakerOn` | `Boolean` | Current speaker state |
| `onMuteToggle` | `() -> Unit` | Toggle mute callback |
| `onSpeakerToggle` | `() -> Unit` | Toggle speaker callback |
| `onEndCall` | `() -> Unit` | End call callback |
| `modifier` | `Modifier` | Optional styling |

---

### CallButton
```kotlin
@Composable
fun CallButton(
    icon: ImageVector,
    contentDescription: String,
    isActive: Boolean,
    backgroundColor: Color,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Individual call control button.

---

## Visual Layout

### Call Controls
```
┌────────────────────────────────────┐
│                                    │
│        ┌─────┐                     │
│        │ 🎤  │                     │  ← Mute (active when red)
│        │Mute │                     │
│        └─────┘                     │
│                                    │
│  ┌─────┐         ┌─────┐          │
│  │ 🔇  │         │ 🔊  │          │  ← Speaker
│  │Mute │         │Speak│          │
│  └─────┘         └─────┘          │
│                                    │
│        ┌─────┐                     │
│        │ 📞  │                     │  ← End call (red)
│        │ End │                     │
│        └─────┘                     │
│                                    │
└────────────────────────────────────┘
```

---

## Button States

### Mute Button
| State | Icon | Background |
|-------|------|------------|
| Not muted | 🎤 Microphone | surfaceVariant |
| Muted | 🎤 Microphone (strikethrough) | error |

### Speaker Button
| State | Icon | Background |
|-------|------|------------|
| Off | 🔊 Volume up | surfaceVariant |
| On | 🔊 Volume up (highlighted) | primary |

### End Call Button
| State | Icon | Background |
|-------|------|------------|
| Always | 📞 Call end | error |

---

## Implementation

### CallButton
```kotlin
@Composable
fun CallButton(
    icon: ImageVector,
    contentDescription: String,
    isActive: Boolean,
    backgroundColor: Color,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    FilledIconButton(
        onClick = onClick,
        modifier = modifier.size(64.dp),
        colors = IconButtonDefaults.filledIconButtonColors(
            containerColor = backgroundColor
        )
    ) {
        Icon(
            imageVector = icon,
            contentDescription = contentDescription,
            modifier = Modifier.size(28.dp),
            tint = if (isActive) Color.White else MaterialTheme.colorScheme.onSurface
        )
    }
}
```

### CallControls
```kotlin
@Composable
fun CallControls(
    isMuted: Boolean,
    isSpeakerOn: Boolean,
    onMuteToggle: () -> Unit,
    onSpeakerToggle: () -> Unit,
    onEndCall: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(32.dp),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Mute button
        CallButton(
            icon = if (isMuted) Icons.Default.MicOff else Icons.Default.Mic,
            contentDescription = if (isMuted) "Unmute" else "Mute",
            isActive = isMuted,
            backgroundColor = if (isMuted)
                MaterialTheme.colorScheme.error
            else
                MaterialTheme.colorScheme.surfaceVariant,
            onClick = onMuteToggle
        )

        // End call button (larger)
        CallButton(
            icon = Icons.Default.CallEnd,
            contentDescription = "End call",
            isActive = true,
            backgroundColor = MaterialTheme.colorScheme.error,
            onClick = onEndCall,
            modifier = Modifier.size(72.dp)
        )

        // Speaker button
        CallButton(
            icon = if (isSpeakerOn) Icons.Default.VolumeUp else Icons.Default.VolumeDown,
            contentDescription = if (isSpeakerOn) "Turn off speaker" else "Turn on speaker",
            isActive = isSpeakerOn,
            backgroundColor = if (isSpeakerOn)
                MaterialTheme.colorScheme.primary
            else
                MaterialTheme.colorScheme.surfaceVariant,
            onClick = onSpeakerToggle
        )
    }
}
```

---

## Animation

### State Transitions
```kotlin
val backgroundColor by animateColorAsState(
    targetValue = if (isActive) activeColor else inactiveColor,
    animationSpec = tween(200),
    label = "backgroundColor"
)
```

### Press Feedback
- Scale down on press
- Ripple effect
- 100ms transition

---

## Accessibility

### Content Descriptions
| Button | Description |
|--------|-------------|
| Mute | "Mute" or "Unmute" |
| Speaker | "Turn on speaker" or "Turn off speaker" |
| End | "End call" |

### Semantic Properties
- Role: Button
- State: pressed/active

---

## State Management

### Call State
```kotlin
data class CallState(
    val callId: String?,
    val callerId: String?,
    val callerName: String?,
    val status: CallStatus,
    val duration: Long,
    val isMuted: Boolean,
    val isSpeakerOn: Boolean
)

enum class CallStatus {
    CONNECTING,
    ACTIVE,
    HOLD,
    ENDING,
    ENDED
}
```

### ViewModel Actions
```kotlin
fun toggleMute() {
    _callState.update { it.copy(isMuted = !it.isMuted) }
    audioManager.isMicrophoneMute = callState.value.isMuted
}

fun toggleSpeaker() {
    _callState.update { it.copy(isSpeakerOn = !it.isSpeakerOn) }
    audioManager.mode = if (callState.value.isSpeakerOn)
        AudioManager.MODE_NORMAL
    else
        AudioManager.MODE_IN_CALL
}

fun endCall() {
    _callState.update { it.copy(status = CallStatus.ENDING) }
    webRTCClient.endCall()
}
```

---

## Related Documentation

- [ActiveCallScreen](../screens/ActiveCallScreen.md) - Call screen
- [AudioVisualizer](AudioVisualizer.md) - Audio visualization
- [Voice Calls](../features/voice-calls.md) - Call feature
