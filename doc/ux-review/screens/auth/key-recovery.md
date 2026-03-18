# Key Recovery Screen

> **Route:** `key_recovery`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/KeyRecoveryScreen.kt`
> **Category:** Authentication

## Screenshot

![Key Recovery Screen](../../screenshots/auth/key-recovery.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←                                   │  ← TopAppBar
├─────────────────────────────────────┤
│                                     │
│              🔐                     │  ← Icon
│                                     │
│      Recover Encryption Keys        │  ← Title
│                                     │
│  Enter your recovery phrase to      │
│  restore your encryption keys.      │
│                                     │
│  ┌─────────────────────────────┐   │
│  │                             │   │
│  │  Enter recovery phrase...   │   │  ← Multi-line input
│  │                             │   │
│  │                             │   │
│  └─────────────────────────────┘   │
│                                     │
│  💡 This is the 12-word phrase     │
│     you saved during setup          │
│                                     │
│  ┌─────────────────────────────┐   │
│  │     Recover Keys            │   │  ← Primary button
│  └─────────────────────────────┘   │
│                                     │
│  Don't have your recovery phrase?  │
│  Contact support                    │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Default

```
┌─────────────────────────────────────┐
│      Recover Encryption Keys        │
│                                     │
│  [Empty phrase field]               │
│  [Disabled Recover button]          │
│                                     │
└─────────────────────────────────────┘
```

### Validating

```
┌─────────────────────────────────────┐
│                                     │
│  ┌─────────────────────────────┐   │
│  │    Validating phrase...     │   │
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

### Success

```
┌─────────────────────────────────────┐
│              ✅                     │
│        Keys Recovered!              │
│                                     │
│  Your encryption keys have been     │
│  restored. You can now access       │
│  your encrypted messages.           │
│                                     │
│  ┌─────────────────────────────┐   │
│  │     Continue to App         │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Error

```
┌─────────────────────────────────────┐
│  ⚠️ Invalid Recovery Phrase         │
│                                     │
│  The phrase you entered doesn't     │
│  match any known keys. Check for    │
│  typos and try again.               │
│                                     │
│  [Try Again]                        │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────┐
            │  Idle   │
            └────┬────┘
                 │
                 ▼
          ┌──────────┐
          │ Enter    │
          │ Phrase   │
          └────┬─────┘
               │
               ▼
          ┌──────────┐
          │Validate  │
          │ Phrase   │
          └────┬─────┘
               │
        ┌──────┴──────┐
        ▼             ▼
   ┌─────────┐   ┌─────────┐
   │ Success │   │  Error  │
   │ → Home  │   │  Retry  │
   └─────────┘   └─────────┘
```

## Notes

- Critical for key restoration after device loss
- 12-word BIP39 mnemonic phrase
- Case-insensitive word matching
- Supports paste from clipboard
- Security-focused UX with warnings
