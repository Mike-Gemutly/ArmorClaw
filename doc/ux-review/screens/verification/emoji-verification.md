# Emoji Verification Screen

> **Route:** `verification/{deviceId}`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/settings/EmojiVerificationScreen.kt`
> **Category:** Verification

## Screenshot

![Emoji Verification Screen](../../screenshots/verification/emoji-verification.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←  Verify Device                    │  ← TopAppBar
├─────────────────────────────────────┤
│                                     │
│  Compare these emojis with the      │
│  other device to verify it's really │
│  you.                               │
│                                     │
│  ┌─────────────────────────────┐   │
│  │                             │   │
│  │    🦀  🌮  🚀  🎸  🌈  🦋   │   │  ← Emoji comparison
│  │                             │   │
│  └─────────────────────────────┘   │
│                                     │
│  Do these emojis match?             │
│                                     │
│  ┌─────────────────────────────┐   │
│  │                             │   │
│  │     They don't match        │   │  ← Reject button
│  │          ❌                 │   │
│  │                             │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │                             │   │
│  │      They match!            │   │  ← Confirm button
│  │          ✅                 │   │
│  │                             │   │
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Comparing

```
┌─────────────────────────────────────┐
│                                     │
│  Compare these emojis with the      │
│  other device...                    │
│                                     │
│    🦀  🌮  🚀  🎸  🌈  🦋           │
│                                     │
│  Do these emojis match?             │
│                                     │
│  [They don't match ❌]              │
│  [They match! ✅]                   │
│                                     │
└─────────────────────────────────────┘
```

### Verifying (Loading)

```
┌─────────────────────────────────────┐
│                                     │
│    🦀  🌮  🚀  🎸  🌈  🦋           │
│                                     │
│         Verifying...                │
│           ◠ ◠ ◠                     │
│                                     │
│                                     │
└─────────────────────────────────────┘
```

### Success

```
┌─────────────────────────────────────┐
│                                     │
│              ✅                     │
│                                     │
│        Device Verified!             │
│                                     │
│  Your device is now trusted and     │
│  can send/receive encrypted         │
│  messages.                           │
│                                     │
│      [ Continue ]                   │
│                                     │
└─────────────────────────────────────┘
```

### Mismatch / Failed

```
┌─────────────────────────────────────┐
│                                     │
│              ❌                     │
│                                     │
│        Verification Failed          │
│                                     │
│  The emojis did not match. This     │
│  could indicate a security issue.   │
│                                     │
│  [ Try Again ]  [ Contact Support ] │
│                                     │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────────┐
            │  Comparing  │
            └──────┬──────┘
                   │
        ┌──────────┼──────────┐
        ▼                     ▼
   ┌──────────┐         ┌──────────┐
   │  Match   │         │ No Match │
   │  (Yes)   │         │   (No)   │
   └────┬─────┘         └────┬─────┘
        │                    │
        ▼                    ▼
   ┌──────────┐         ┌──────────┐
   │Verifying │         │ Failed   │
   └────┬─────┘         └──────────┘
        │
   ┌────┴────┐
   ▼         ▼
┌──────┐ ┌────────┐
│Success│ │ Error  │
└───┬───┘ └────────┘
    │
    ▼
┌──────────┐
│ Devices  │
│ Screen   │
└──────────┘
```

## User Flow

1. **User arrives from:**
   - Device list screen (verify button)
   - Notification (new device login)

2. **User can:**
   - Compare emojis on both devices
   - Confirm match (verify)
   - Reject match (security concern)

3. **User navigates to:**
   - Device list screen (success)
   - Support (security concern)

## Security Notes

- Emojis generated from shared secret
- Both devices must show same sequence
- Mismatch indicates potential MITM attack
- Time-limited verification window

## Accessibility

- Emoji sequence read aloud
- Clear button labels
- Security implications explained
- Large touch targets for actions

## Design Tokens

| Token | Value |
|-------|-------|
| Emoji size | 40.sp |
| Button height | 56.dp |
| Success color | Green |
| Error color | Red |

## Notes

- Critical security verification step
- Simple visual comparison
- Clear success/failure feedback
- Security education opportunity
