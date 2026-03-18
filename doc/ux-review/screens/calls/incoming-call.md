# Incoming Call Screen

> **Route:** `incoming_call/{callId}/{callerId}/{callerName}/{callType}`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/call/IncomingCallDialog.kt`
> **Category:** Calls

## Screenshot

![Incoming Call Screen](../../screenshots/calls/incoming-call.png)

## Layout

```
┌─────────────────────────────────────┐
│                                     │
│                                     │
│                                     │
│              👤                     │  ← Caller avatar
│           John Doe                  │  ← Caller name
│                                     │
│        📞 Incoming Call             │  ← Call type indicator
│                                     │
│                                     │
│                                     │
│                                     │
│    ┌─────────┐         ┌─────────┐ │
│    │   📵    │         │   📞    │ │  ← Action buttons
│    │ Decline │         │ Accept  │ │
│    │ (red)   │         │ (green) │ │
│    └─────────┘         └─────────┘ │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Voice Call - Ringing

```
┌─────────────────────────────────────┐
│              👤                     │
│           John Doe                  │
│                                     │
│        📞 Incoming Voice Call       │
│           Ringing...                │
│                                     │
│    [   📵   ]     [   📞   ]       │
│    [ Decline]     [ Accept ]       │
└─────────────────────────────────────┘
```

### Video Call - Ringing

```
┌─────────────────────────────────────┐
│              👤                     │
│           John Doe                  │
│                                     │
│        📹 Incoming Video Call       │
│           Ringing...                │
│                                     │
│    [   📵   ]     [   📹   ]       │
│    [ Decline]     [ Accept ]       │
└─────────────────────────────────────┘
```

### Call Timeout

```
┌─────────────────────────────────────┐
│              👤                     │
│           John Doe                  │
│                                     │
│        Missed Call                  │
│                                     │
│         [   Dismiss   ]             │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────────┐
            │  Ringing    │
            └──────┬──────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │ Accept  │ │ Decline │ │ Timeout │
   └────┬────┘ └────┬────┘ └────┬────┘
        │           │           │
        ▼           ▼           ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │ Active  │ │  Call   │ │ Missed  │
   │  Call   │ │ Ended   │ │  Call   │
   │ Screen  │ │         │ │ Screen  │
   └─────────┘ └─────────┘ └─────────┘
```

## User Flow

1. **User arrives from:**
   - Push notification (call invite)
   - App in foreground (incoming event)

2. **User can:**
   - Accept call (green button)
   - Decline call (red button)
   - Swipe to decline (optional)

3. **User navigates to:**
   - Active call screen (accept)
   - Previous screen (decline)
   - Chat screen (after missed)

## Accessibility

- **Content descriptions:**
  - Avatar: "Incoming call from [name]"
  - Accept: "Accept call"
  - Decline: "Decline call"

- **Touch targets:**
  - Accept/Decline buttons: 64.dp minimum

- **Screen reader considerations:**
  - Call type announced (voice/video)
  - Caller name announced
  - Ringing state announced

## Design Tokens

| Token | Value |
|-------|-------|
| Avatar size | 100.dp |
| Accept button | Green (#4CAF50) |
| Decline button | Red (#F44336) |
| Button size | 72.dp |

## Notes

- Full-screen overlay
- Prominent action buttons
- Call type clearly indicated
- Auto-timeout after 60 seconds
- Vibration/ringtone feedback
