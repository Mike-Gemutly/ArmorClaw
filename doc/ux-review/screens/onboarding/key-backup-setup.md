# Key Backup Setup Screen

> **Route:** `key_backup_setup`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/onboarding/KeyBackupSetupScreen.kt`
> **Category:** Onboarding

## Screenshot

![Key Backup Setup Screen](../../screenshots/onboarding/key-backup-setup.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←  Secure Your Keys                 │  ← TopAppBar
├─────────────────────────────────────┤
│                                     │
│              🔐                     │  ← Icon
│                                     │
│     Backup Your Encryption Keys     │  ← Title
│                                     │
│  Your encryption keys are used to   │
│  secure your messages. Save this    │
│  recovery phrase in a safe place.   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  1. crab     2. taco       │   │
│  │  3. rocket   4. guitar     │   │  ← Recovery phrase
│  │  5. rainbow  6. butterfly  │   │     (12 words)
│  │  7. ocean    8. mountain   │   │
│  │  9. forest   10. desert    │   │
│  │  11. river   12. cloud     │   │
│  └─────────────────────────────┘   │
│                                     │
│         [📋 Copy to Clipboard]      │
│                                     │
│  ⚠️ WARNING                         │
│  • Never share this phrase          │
│  • Store it securely offline        │
│  • It cannot be recovered if lost   │
│                                     │
│  ☐ I have saved my recovery phrase  │
│                                     │
│  ┌─────────────────────────────┐   │
│  │       Continue              │   │  ← Disabled until checked
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Initial (Phrase Shown)

```
┌─────────────────────────────────────┐
│     Backup Your Encryption Keys     │
│                                     │
│  [12-word recovery phrase]          │
│                                     │
│  [Copy to Clipboard]                │
│                                     │
│  ⚠️ [Warning message]               │
│                                     │
│  ☐ I have saved my recovery phrase  │
│  [Continue - Disabled]              │
└─────────────────────────────────────┘
```

### Confirmed

```
┌─────────────────────────────────────┐
│     Backup Your Encryption Keys     │
│                                     │
│  [12-word recovery phrase]          │
│                                     │
│  ✅ Copied to clipboard             │
│                                     │
│  ☑ I have saved my recovery phrase  │
│  [Continue - Enabled]               │
└─────────────────────────────────────┘
```

### Verification Step (Optional)

```
┌─────────────────────────────────────┐
│     Verify Your Recovery Phrase     │
│                                     │
│  Tap the words in the correct order:│
│                                     │
│  Word #3: _____                     │
│                                     │
│  [crab] [taco] [rocket] [guitar]    │  ← Word options
│  [rainbow] [butterfly] ...          │     (shuffled)
│                                     │
│         [Continue]                  │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────────┐
            │  Show       │
            │  Phrase     │
            └──────┬──────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │  Copy   │ │ Confirm │ │  Back   │
   │ Phrase  │ │ Checkbox│ │         │
   └─────────┘ └────┬────┘ └─────────┘
                   │
                   ▼
            ┌─────────────┐
            │  Continue   │
            │  (enabled)  │
            └──────┬──────┘
                   │
                   ▼
            ┌─────────────┐
            │  Verify     │
            │  (optional) │
            └──────┬──────┘
                   │
                   ▼
            ┌─────────────┐
            │   Next      │
            │   Step      │
            └─────────────┘
```

## User Flow

1. **User arrives from:** Security explanation screen
2. **User can:**
   - View recovery phrase
   - Copy to clipboard
   - Confirm they saved it
   - Optionally verify phrase
3. **User navigates to:**
   - Permissions screen (continue)
   - Previous screen (back)

## Security Warnings

| Warning | Purpose |
|---------|---------|
| Never share | Prevent social engineering |
| Store offline | Prevent digital theft |
| Cannot recover | Emphasize importance |

## Accessibility

- Phrase can be read aloud
- Large text for readability
- Clear checkbox confirmation
- Warning emphasized with icon

## Design Tokens

| Token | Value |
|-------|-------|
| Phrase card | surfaceVariant |
| Word text | bodyLarge |
| Warning color | error |
| Checkbox size | 24.dp |

## Notes

- Critical step for account recovery
- Cannot be skipped
- Verification recommended
- Clear security warnings
- Phrase only shown once
