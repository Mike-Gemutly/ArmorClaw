# Voice Calls Feature

> Audio calling functionality for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/call/`

## Overview

The voice calls feature provides secure audio calling between ArmorClaw users with end-to-end encryption.

## Feature Components

### ActiveCallScreen
**Location:** `call/ActiveCallScreen.kt`

Main screen during an active voice call.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `ActiveCallScreen()` | Main call screen | `callId`, `onEndCall`, `onMinimize` |
| `CallHeader()` | Call duration and status | `duration`, `callState` |
| `CallerAvatar()` | Large avatar display | `avatar`, `isAnimated` |
| `CallStatusText()` | Status indicator | `state`, `callerName` |
| `AudioLevelIndicator()` | Microphone level | `audioLevel` |

#### Call States
| State | Display | Description |
|-------|---------|-------------|
| CONNECTING | "Connecting..." | Establishing connection |
| RINGING | "Ringing" | Waiting for answer |
| ACTIVE | Duration timer | Call in progress |
| ON_HOLD | "On Hold" | Call paused |
| ENDING | "Ending call..." | Call terminating |

#### User Actions
- **Mute** - Toggle microphone
- **Speaker** - Toggle speakerphone
- **Hold** - Put call on hold
- **End** - Terminate call

---

### IncomingCallDialog
**Location:** `call/IncomingCallDialog.kt`

Dialog for incoming call notification.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `IncomingCallDialog()` | Main dialog composable | `caller`, `onAccept`, `onDecline` |
| `CallerInfo()` | Caller identity display | `name`, `avatar` |
| `CallTypeIndicator()` | Voice/video indicator | `isVideo` |
| `ActionButtons()` | Accept/Decline buttons | `onAccept`, `onDecline` |

#### Incoming Call Flow
1. Receive call signal
2. Display dialog with caller info
3. User accepts or declines
4. On accept: Navigate to ActiveCallScreen
5. On decline: Send rejection signal

---

### CallControls
**Location:** `call/CallControls.kt`

In-call control buttons.

#### Functions

| Function | Description |
|----------|-------------|
| `CallControls()` | Control button row |
| `MuteButton()` | Microphone toggle |
| `SpeakerButton()` | Speaker toggle |
| `HoldButton()` | Hold toggle |
| `EndCallButton()` | End call button |
| `ControlButton()` | Generic control button |

#### Control States

| Control | Active State | Inactive State |
|---------|--------------|----------------|
| Mute | Mic off, red | Mic on, white |
| Speaker | Speaker on, green | Speaker off, white |
| Hold | Held, yellow | Active, white |
| End | Always red | - |

---

### AudioVisualizer
**Location:** `call/AudioVisualizer.kt`

Visual representation of audio levels.

#### Functions

| Function | Description |
|----------|-------------|
| `AudioVisualizer()` | Main visualizer composable |
| `AudioBar()` | Individual frequency bar |
| `AnimatedAudioLevel()` | Level animation |
| `WaveformVisualizer()` | Waveform display |

#### Visualization Types
- **Bar** - Vertical bars representing frequency
- **Wave** - Continuous waveform line
- **Circle** - Circular pulse animation

#### Animation Parameters
- `barCount` - Number of bars (default: 5)
- `animationDuration` - Animation speed
- `amplitudeRange` - Min/max amplitude
- `colorGradient` - Color range

---

## Call State Management

### CallState
```kotlin
sealed class CallState {
    object Idle : CallState()
    object Connecting : CallState()
    object Ringing : CallState()
    data class Active(val duration: Long) : CallState()
    object OnHold : CallState()
    object Ending : CallState()
    object Ended : CallState()
}
```

### CallInfo
```kotlin
data class CallInfo(
    val callId: String,
    val callerId: String,
    val callerName: String,
    val callerAvatar: String?,
    val isOutgoing: Boolean,
    val state: CallState,
    val startTime: Instant?,
    val audioState: AudioState
)
```

---

## Audio Management

### AudioState
```kotlin
data class AudioState(
    val isMuted: Boolean,
    val isSpeakerOn: Boolean,
    val isOnHold: Boolean,
    val audioLevel: Float
)
```

### Audio Actions
| Action | Function | Description |
|--------|----------|-------------|
| Toggle Mute | `toggleMute()` | Enable/disable microphone |
| Toggle Speaker | `toggleSpeaker()` | Switch between earpiece/speaker |
| Toggle Hold | `toggleHold()` | Pause/resume call |
| Set Audio Level | `setAudioLevel(level)` | Update visual level |

---

## Call Signaling

### WebRTC Integration
- Uses WebRTC for audio transport
- Encrypted media streams
- ICE candidate negotiation
- STUN/TURN server support

### Signaling Flow
```
Caller                          Server                        Callee
  │                               │                             │
  │──── Call Offer ──────────────→│                             │
  │                               │──── Call Offer ────────────→│
  │                               │                             │
  │                               │←── Call Answer ─────────────│
  │←── Call Answer ───────────────│                             │
  │                               │                             │
  │←────── ICE Candidates ────────│──── ICE Candidates ────────→│
  │                               │                             │
  │═══════════ Encrypted Audio ═════════════════════════════════│
```

---

## Permissions Required

| Permission | Purpose |
|------------|---------|
| RECORD_AUDIO | Capture microphone input |
| MODIFY_AUDIO_SETTINGS | Route audio |
| ACCESS_NETWORK_STATE | WebRTC connectivity |

---

## Related Documentation

- [Chat](chat.md) - Messaging functionality
- [Encryption](encryption.md) - Security implementation
- [Notifications](notifications.md) - Call notifications
