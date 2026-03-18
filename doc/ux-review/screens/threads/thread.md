# Thread View Screen

> **Route:** `thread/{roomId}/{rootMessageId}`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/ThreadViewScreen.kt`
> **Category:** Threads

## Screenshot

![Thread View Screen](../../screenshots/threads/thread.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←  Thread                      ⋮   │  ← TopAppBar
├─────────────────────────────────────┤
│  ┌─────────────────────────────┐   │
│  │ 👤 Alice                    │   │  ← Original message
│  │    What's the security      │   │     (pinned header)
│  │    architecture?            │   │
│  │    2:30 PM                  │   │
│  └─────────────────────────────┘   │
│                                     │
│  3 replies                          │  ← Reply count
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 👤 Bob                      │   │
│  │    We use ECDH for key      │   │
│  │    exchange...              │   │
│  │    2:35 PM            💬 1  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 👤 Carol                    │   │
│  │    And AES-256-GCM for      │   │
│  │    message encryption       │   │
│  │    2:40 PM                  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 👤 Dave                     │   │
│  │    Don't forget SQLCipher!  │   │
│  │    2:45 PM                  │   │
│  └─────────────────────────────┘   │
│                                     │
├─────────────────────────────────────┤
│ 📎 │ Reply to thread... │ 🎤  ➤   │  ← Input bar
└─────────────────────────────────────┘
```

## UI States

### Loading

```
┌─────────────────────────────────────┐
│ ←  Thread                           │
├─────────────────────────────────────┤
│                                     │
│           ◠ ◠ ◠                     │
│        Loading thread...            │
│                                     │
└─────────────────────────────────────┘
```

### Loaded

```
┌─────────────────────────────────────┐
│ ←  Thread                           │
├─────────────────────────────────────┤
│  [Original message]                 │
│                                     │
│  [Reply count]                      │
│                                     │
│  [Threaded replies]                 │
│                                     │
│  [Input bar]                        │
└─────────────────────────────────────┘
```

### Empty (No Replies)

```
┌─────────────────────────────────────┐
│  [Original message]                 │
│                                     │
│              💬                     │
│        No replies yet               │
│                                     │
│    Be the first to reply!           │
│                                     │
│  [Input bar]                        │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────────┐
            │   Loading   │
            └──────┬──────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │  Empty  │ │ Loaded  │ │  Error  │
   └─────────┘ └────┬────┘ └─────────┘
                   │
                   ▼
            ┌─────────────┐
            │ Reply to    │
            │ Thread      │
            └──────┬──────┘
                   │
                   ▼
            ┌─────────────┐
            │ Optimistic  │
            │ Update      │
            └─────────────┘
```

## User Flow

1. **User arrives from:** Chat screen (tap thread indicator)
2. **User can:**
   - View original message
   - View all replies
   - Reply to thread
   - React to messages
3. **User navigates to:**
   - Chat screen (back)
   - User profile (tap avatar)

## Accessibility

- **Content descriptions:**
  - Original message: "Original message by [name], [content], [time]"
  - Replies: "Reply by [name], [content], [time]"

- **Touch targets:**
  - Messages: 48.dp minimum height
  - Input bar: 48.dp

## Design Tokens

| Token | Value |
|-------|-------|
| Original message | surfaceVariant |
| Reply count | labelMedium |
| Thread connector | vertical line |

## Notes

- Keeps conversations organized
- Original message always visible
- Reply count indicator
- Same input bar as chat
- Supports reactions
