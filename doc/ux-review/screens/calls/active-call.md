# Active Call Screen

> **Route:** `call/{callId}`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/call/ActiveCallScreen.kt`
> **Category:** Calls

## Screenshot

![Active Call Screen](../../screenshots/calls/active-call.png)

## Layout

```
┌─────────────────────────────────────┐
│                                     │
│                                     │
│                                     │
│              👤                     │  ← Caller avatar (large)
│                                     │
│           John Doe                  │  ← Caller name
│                                     │
│           05:32                     │  ← Call duration
│                                     │
│                                     │
│    ╔═════════════════════════╗     │  ← Audio visualizer
│    ║ ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁ ▂▃▄▅▆ ║     │     (animated bars)
│    ╚═════════════════════════╝     │
│                                     │
│                                     │
│                                     │
│                                     │
│    ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐│
│    │ 🔇 │ │ 📹 │ │ 🔊 │ │ 📞 ││  ← Call controls
│    │Mute │ │Video│ │Spkr │ │End  ││
│    └─────┘ └─────┘ └─────┘ └─────┘│
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Voice Call - Active

```
┌─────────────────────────────────────┐
│                                     │
│              👤                     │
│           John Doe                  │
│           05:32                     │
│                                     │
│    [Audio visualizer animated]      │
│                                     │
│    [🔇] [📹] [🔊] [📞]             │
│    Mute Video Speaker End           │
└─────────────────────────────────────┘
```

### Video Call - Active

```
┌─────────────────────────────────────┐
│ ┌─────────────────────────────────┐ │  ← Remote video (full)
│ │                                 │ │
│ │      [Remote participant]       │ │
│ │                                 │ │
│ └─────────────────────────────────┘ │
│ ┌───────────┐                       │  ← Local video (pip)
│ │   👤      │                       │
│ │  (You)    │                       │
│ └───────────┘                       │
│    [🔇] [📹] [🔊] [📞]             │
└─────────────────────────────────────┘
```

### Voice Call - Muted

```
┌─────────────────────────────────────┐
│              👤                     │
│           John Doe                  │
│           05:32                     │
│           🔇 MUTED                  │  ← Muted indicator
│                                     │
│    [🔇✓] [📹] [🔊] [📞]            │  ← Mute button active
│                                     │
└─────────────────────────────────────┘
```

### Connecting

```
┌─────────────────────────────────────┐
│              👤                     │
│           John Doe                  │
│                                     │
│         Connecting...               │  ← Connecting state
│           ◠ ◠ ◠                     │
│                                     │
│    [🔇] [📹] [🔊] [📞]             │
│                                     │
└─────────────────────────────────────┘
```

### Reconnecting

```
┌─────────────────────────────────────┐
│              👤                     │
│           John Doe                  │
│           05:32                     │
│                                     │
│    ⚠️ Reconnecting...               │  ← Warning indicator
│                                     │
│    [🔇] [📹] [🔊] [📞]             │
│                                     │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────────┐
            │ Connecting  │
            └──────┬──────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │ Active  │ │ Failed  │ │ Declined│
   │  Call   │ │         │ │         │
   └────┬────┘ └─────────┘ └─────────┘
        │
        │ (network issue)
        ▼
   ┌───────────┐
   │Reconnecting│
   └─────┬─────┘
         │
         ▼
   ┌─────────┐
   │ Active  │
   └────┬────┘
        │
   ┌────┴────┐
   ▼         ▼
┌──────┐ ┌──────┐
│ End  │ │Remote│
│ Call │ │ End  │
└──┬───┘ └───┬──┘
   │         │
   ▼         ▼
┌──────────────┐
│ Call Ended   │
│ → Chat/Back  │
└──────────────┘
```

## User Flow

1. **User arrives from:**
   - Chat screen (call button tap)
   - Incoming call screen (accept)

2. **User can:**
   - Mute/unmute microphone
   - Enable/disable camera (video calls)
   - Switch speaker output
   - End call
   - View call duration

3. **User navigates to:**
   - Chat screen (end call)
   - Home screen (end call)

## Call Controls

| Button | States | Action |
|--------|--------|--------|
| Mute | Active/Inactive | Toggle microphone |
| Video | Active/Inactive | Toggle camera |
| Speaker | Earpiece/Speaker | Toggle output |
| End | Always active | End call |

## Accessibility

- **Content descriptions:**
  - Mute: "Mute microphone" / "Unmute microphone"
  - Video: "Turn off camera" / "Turn on camera"
  - Speaker: "Switch to speaker" / "Switch to earpiece"
  - End: "End call"

- **Touch targets:**
  - All buttons: 48.dp minimum

- **Screen reader considerations:**
  - Call duration announced periodically
  - Mute state announced
  - Reconnecting status announced

## Design Tokens

| Token | Value |
|-------|-------|
| Avatar size | 120.dp |
| Duration text | headlineMedium |
| End button | error color |
| Control button | 48.dp |
| Visualizer height | 40.dp |

## Notes

- Full-screen immersive experience
- Large touch targets for call controls
- Visual feedback for all actions
- Network status indicators
- Automatic screen wake lock
