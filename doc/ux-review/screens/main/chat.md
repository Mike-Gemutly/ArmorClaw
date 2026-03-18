# Chat Screen

> **Route:** `chat/{roomId}`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/ChatScreen_enhanced.kt`
> **Category:** Main

## Screenshot

![Chat Screen](../../screenshots/main/chat.png)

## Layout

```
┌─────────────────────────────────────┐
│ ← │ ArmorClaw Agent ● 🔒 🟡 │📞 📹🔍⋮│  ← TopAppBar
├─────────────────────────────────────┤
│ ┌─────────────────────────────────┐ │  ← Workflow banner (if active)
│ │ 🔄 Processing payment...   75%  │ │
│ └─────────────────────────────────┘ │
│ ┌─────────────────────────────────┐ │  ← Agent thinking (if active)
│ │ 🤖 Agent is thinking...         │ │
│ └─────────────────────────────────┘ │
│ ┌─────────────────────────────────┐ │  ← Search bar (if active)
│ │ 🔍 Search messages...      ✕   │ │
│ └─────────────────────────────────┘ │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ 👤 Alice                        │ │
│ │    Hey! How are you?     10:30  │ │  ← Message (other)
│ └─────────────────────────────────┘ │
│                                     │
│              ┌────────────────────┐ │
│              │ Hi! I'm good! 10:31│ │  ← Message (self)
│              │              ✓✓   │ │     (blue background)
│              └────────────────────┘ │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ 🤖 Assistant                    │ │
│ │    I can help with that! 10:32  │ │  ← Agent message
│ └─────────────────────────────────┘ │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ 👤 Bob is typing...             │ │  ← Typing indicator
│ └─────────────────────────────────┘ │
│                                     │
│ ┌─────────────────────────────────┐ │  ← Reply preview
│ │ ↩ Replying to Alice             │ │
│ │    "Hey! How are you?"     ✕   │ │
│ └─────────────────────────────────┘ │
├─────────────────────────────────────┤
│ 📎 │ Type a message...    │ 🎤  ➤ │  ← Input bar
└─────────────────────────────────────┘
```

## UI States

### Loading

```
┌─────────────────────────────────────┐
│ ← │ Loading...           │  ⋮      │
├─────────────────────────────────────┤
│                                     │
│                                     │
│           ◠ ◠ ◠ ◠                   │
│        Loading messages...          │
│                                     │
│                                     │
├─────────────────────────────────────┤
│ 📎 │                    │ 🎤       │
└─────────────────────────────────────┘
```

### Loaded (Default)

```
┌─────────────────────────────────────┐
│ ← │ Room Name ● 🔒       │📞 📹🔍⋮ │
├─────────────────────────────────────┤
│                                     │
│    [Message list displayed]         │
│                                     │
│                                     │
├─────────────────────────────────────┤
│ 📎 │ Type a message...  │ 🎤  ➤   │
└─────────────────────────────────────┘
```

### Empty (No Messages)

```
┌─────────────────────────────────────┐
│ ← │ Room Name ● 🔒       │📞 📹🔍⋮ │
├─────────────────────────────────────┤
│                                     │
│                                     │
│            💬                       │
│    No messages yet                  │
│                                     │
│    Say something to start!          │
│                                     │
│                                     │
├─────────────────────────────────────┤
│ 📎 │ Type a message...  │ 🎤  ➤   │
└─────────────────────────────────────┘
```

### Error

```
┌─────────────────────────────────────┐
│                                     │
│              ⚠️                     │
│    Failed to load messages          │
│                                     │
│         [Retry]                     │
│                                     │
└─────────────────────────────────────┘
```

### Unverified Bridge Warning

```
┌─────────────────────────────────────┐
│              🛡️                     │
│      Unverified Bridge              │
│                                     │
│  This room is bridged to an         │
│  external platform whose identity   │
│  has not been verified. Messages    │
│  may be relayed through an          │
│  untrusted server.                  │
│                                     │
│  [Leave Room]    [I Understand]     │
└─────────────────────────────────────┘
```

## State Flow

```
                    ┌──────────────┐
                    │   Loading    │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌──────────┐ ┌──────────┐ ┌──────────┐
       │  Empty   │ │  Loaded  │ │  Error   │
       └──────────┘ └────┬─────┘ └────┬─────┘
                         │            │
                         │            ▼
                         │     ┌──────────┐
                         │     │  Retry   │
                         │     └──────────┘
                         │
    ┌────────────────────┼────────────────────┐
    ▼                    ▼                    ▼
┌─────────┐       ┌──────────┐        ┌──────────┐
│ Send    │       │ Toggle   │        │ Reply    │
│ Message │       │ Search   │        │ To       │
└────┬────┘       └────┬─────┘        └────┬─────┘
     │                 │                   │
     ▼                 ▼                   ▼
┌─────────┐       ┌──────────┐        ┌──────────┐
│ Optimistic│      │ Filter   │        │ Show     │
│ Update   │       │ Messages │        │ Reply    │
└─────────┘       └──────────┘        │ Preview  │
                                      └──────────┘
```

## User Flow

1. **User arrives from:**
   - Home screen (room tap)
   - Notification tap
   - Deep link (`armorclaw://chat/{roomId}`)

2. **User can:**
   - Read messages (auto-scroll to latest)
   - Send text messages
   - Reply to specific messages
   - Add reactions to messages
   - Search messages
   - Start voice/video call
   - Attach files
   - Use voice input
   - View room details (tap title)
   - Navigate back

3. **User navigates to:**
   - Home screen (back)
   - Room details screen
   - Voice call screen
   - Video call screen
   - Thread view
   - Image viewer
   - File preview
   - User profile

## Components Used

| Component | Source | Purpose |
|-----------|--------|---------|
| Scaffold | Material3 | Screen layout |
| TopAppBar | Material3 | Navigation and actions |
| UnifiedMessageList | shared/ui | Message display |
| MessageInputBar | Local | Text input |
| ChatSearchBar | Local | Search functionality |
| TypingIndicatorComponent | Local | Typing status |
| ReplyPreviewBar | Local | Reply preview |
| WorkflowProgressBanner | shared/ui | Workflow status |
| AgentThinkingIndicator | shared/ui | Agent status |
| EncryptionStatusIndicator | Local | Encryption status |
| AlertDialog | Material3 | Bridge warning |

## Accessibility

- **Content descriptions:**
  - Back: "Back"
  - Voice call: "Voice call"
  - Video call: "Video call"
  - Search: "Search"
  - More: "More options"
  - Attach: "Attach file"
  - Mic: "Voice input" / "Stop recording"
  - Send: "Send"
  - Encryption status: Dynamic based on state

- **Touch targets:**
  - All icon buttons: 48.dp
  - Message input: Full width

- **Focus order:**
  1. Back button
  2. Title (room details)
  3. Call buttons
  4. Search
  5. More options
  6. Message list
  7. Attach button
  8. Message input
  9. Mic button
  10. Send button

- **Screen reader considerations:**
  - Messages announced with sender and time
  - Encryption status announced
  - Typing indicators announced

## Design Tokens

| Token | Value |
|-------|-------|
| TopAppBar color | Primary |
| Input bar elevation | 4.dp |
| Input field shape | CircleShape |
| Send button active | BrandPurple |
| Send button inactive | onSurface.copy(0.3f) |
| Mic active | BrandRed |
| Mic inactive | BrandPurple |
| Online indicator | BrandGreen |
| Warning indicator | #F57F17 |

## Encryption Status Indicators

| Status | Icon | Color |
|--------|------|-------|
| Verified | Shield check | Green |
| Unverified | Shield warning | Yellow |
| No Encryption | Shield off | Gray |

## Message Input States

| State | Description |
|-------|-------------|
| Empty | Send button disabled, placeholder visible |
| Has text | Send button enabled, BrandPurple |
| Voice active | Mic button BrandRed |
| Replying | Reply preview bar visible |

## Notes

- Real-time updates via WebSocket
- Optimistic UI updates for sent messages
- Pull-to-load-more for history
- Encryption status always visible
- Bridge verification warning for security
- Agent-specific UI when AI agent in room
- Workflow progress for multi-step operations
- Voice input with visual feedback
