# Login Screen

> **Route:** `login`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/LoginScreen.kt`
> **Category:** Authentication

## Screenshot

![Login Screen](../../screenshots/auth/login.png)

## Layout

```
┌─────────────────────────────────────┐
│                                     │  ← Transparent TopAppBar
├─────────────────────────────────────┤
│                                     │
│              🛡️                     │  ← Logo (100.dp)
│                                     │
│           ArmorClaw                 │  ← App name
│   Secure. Private. Encrypted.       │  ← Tagline
│                                     │
│  ┌─────────────────────────────┐   │
│  │  ┌──────────────────────┐   │   │
│  │  │👤 Username or Email  │   │   │  ← Username field
│  │  └──────────────────────┘   │   │
│  │                             │   │
│  │  ┌──────────────────────┐   │   │
│  │  │🔒 Password       👁️✕│   │   │  ← Password field
│  │  └──────────────────────┘   │   │
│  │                             │   │
│  │            Forgot password? │   │  ← Forgot link
│  │                             │   │
│  │  ┌──────────────────────┐   │   │
│  │  │       Log In         │   │   │  ← Primary button
│  │  └──────────────────────┘   │   │
│  │                             │   │
│  │  ────────── OR ──────────   │   │
│  │                             │   │
│  │  ┌──────────────────────┐   │   │
│  │  │ingerprint Log in w/  │   │   │  ← Biometric button
│  │  │        Biometrics    │   │   │
│  │  └──────────────────────┘   │   │
│  │                             │   │
│  │  🔒 Recover encryption keys │   │  ← Recovery link
│  └─────────────────────────────┘   │
│                                     │
│    Don't have an account? Register  │  ← Register link
│                                     │
│    By continuing, you agree to our  │
│         Terms of Service            │  ← Legal links
│           and Privacy Policy        │
│                                     │
│           Version 1.0.0             │
└─────────────────────────────────────┘
```

## UI States

### Default / Empty

```
┌─────────────────────────────────────┐
│              🛡️                     │
│           ArmorClaw                 │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  [Empty username field]     │   │
│  │  [Empty password field]     │   │
│  │                             │   │
│  │  [Disabled Log In button]   │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Filled / Valid

```
┌─────────────────────────────────────┐
│              🛡️                     │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  user@example.com      ✕   │   │  ← Clear button visible
│  │  ••••••••          👁️ ✕   │   │  ← Toggle/Clear visible
│  │                             │   │
│  │  [Enabled Log In button]    │   │  ← Button enabled
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Loading / Submitting

```
┌─────────────────────────────────────┐
│  ┌─────────────────────────────┐   │
│  │                             │   │
│  │  [Log In...] (showing prog) │   │  ← Button shows progress
│  │                             │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Error

```
┌─────────────────────────────────────┐
│  ┌─────────────────────────────┐   │
│  │  user@example.com           │   │
│  │  ⚠️ Invalid credentials     │   │  ← Error message
│  │                             │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

## State Flow

```
                    ┌──────────────┐
                    │    Idle      │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌──────────┐ ┌──────────┐ ┌──────────┐
       │  Typing  │ │ Biometric│ │ Forgot   │
       │  Input   │ │  Login   │ │ Password │
       └────┬─────┘ └────┬─────┘ └────┬─────┘
            │            │            │
            ▼            ▼            ▼
       ┌──────────┐ ┌──────────┐ ┌──────────┐
       │ Validate │ │ System   │ │ Navigate │
       │ Form     │ │ Auth     │ │ to       │
       │          │ │ Prompt   │ │ Recovery │
       └────┬─────┘ └──────────┘ └──────────┘
            │
    ┌───────┴───────┐
    ▼               ▼
┌────────┐    ┌─────────┐
│ Invalid│    │ Submit  │
│ (error)│    │ Login   │
└────────┘    └────┬────┘
                   │
           ┌───────┴───────┐
           ▼               ▼
    ┌──────────┐    ┌──────────┐
    │  Success │    │  Error   │
    │  → Home  │    │  Show    │
    │          │    │  Message │
    └──────────┘    └──────────┘
```

## User Flow

1. **User arrives from:**
   - Splash screen (returning user without session)
   - Welcome screen (Skip)
   - Deep link (magic link expired)

2. **User can:**
   - Enter username/email and password
   - Toggle password visibility
   - Clear input fields
   - Tap "Log In" button
   - Use biometric authentication
   - Navigate to "Forgot password"
   - Navigate to "Recover encryption keys"
   - Navigate to "Register"

3. **User navigates to:**
   - Home (successful login)
   - ForgotPassword screen
   - KeyRecovery screen
   - Registration screen

## Components Used

| Component | Source | Purpose |
|-----------|--------|---------|
| Scaffold | Material3 | Screen layout |
| TopAppBar | Material3 | Navigation bar (transparent) |
| Card | Material3 | Form container |
| OutlinedTextField | Material3 | Input fields |
| IconButton | Material3 | Toggle/clear actions |
| Button | Material3 | Primary login action |
| OutlinedButton | Material3 | Biometric login |
| TextButton | Material3 | Navigation links |
| Divider | Material3 | Visual separator |

## Accessibility

- **Content descriptions:**
  - Username field: "Enter your username or email"
  - Password field: "Enter your password"
  - Toggle: "Show password" / "Hide password"
  - Clear: "Clear"
  - Fingerprint icon: decorative (null)

- **Touch targets:**
  - All buttons: 48.dp minimum
  - Icon buttons: 48.dp

- **Focus order:**
  1. Username field
  2. Password field
  3. Forgot password
  4. Log In button
  5. Biometric button
  6. Recovery link
  7. Register link

- **Screen reader considerations:**
  - Error messages announced
  - Button state changes announced
  - IME actions for keyboard flow

## Design Tokens

| Token | Value |
|-------|-------|
| Background | SurfaceColor |
| Card background | surfaceVariant |
| Primary button | AccentColor |
| Link color | AccentColor |
| Corner radius | 12.dp |
| Form padding | 24.dp |
| Logo size | 100.dp |

## Validation Rules

| Field | Rule |
|-------|------|
| Username | notBlank() |
| Password | notBlank() |
| Form valid | username.isNotBlank() && password.isNotBlank() |

## Keyboard Options

| Field | Keyboard Type | IME Action |
|-------|---------------|------------|
| Username | Email | Next |
| Password | Password | Done |

## Notes

- Form scrollable with imePadding() for keyboard
- Biometric availability checked at runtime
- Clear buttons only show when field has content
- Version number at bottom for support reference
- Legal links for compliance
- Recovery link for key restoration flow
