# Registration Screen

> **Route:** `registration`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/RegistrationScreen.kt`
> **Category:** Authentication

## Screenshot

![Registration Screen](../../screenshots/auth/registration.png)

## Layout

```
┌─────────────────────────────────────┐
│                                     │  ← Transparent TopAppBar
├─────────────────────────────────────┤
│                                     │
│              🛡️                     │  ← Logo
│                                     │
│          Create Account             │  ← Title
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 👤 Username                 │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 📧 Email                    │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 🔒 Password             👁️ │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 🔒 Confirm Password     👁️ │   │
│  └─────────────────────────────┘   │
│                                     │
│  ☐ I agree to Terms of Service     │
│  ☐ I agree to Privacy Policy       │
│                                     │
│  ┌─────────────────────────────┐   │
│  │       Create Account        │   │  ← Primary button
│  └─────────────────────────────┘   │
│                                     │
│    Already have an account? Login   │
│                                     │
│           Version 1.0.0             │
└─────────────────────────────────────┘
```

## UI States

### Default / Empty

```
┌─────────────────────────────────────┐
│          Create Account             │
│                                     │
│  [Empty fields]                     │
│  [Unchecked boxes]                  │
│  [Disabled Create button]           │
│                                     │
└─────────────────────────────────────┘
```

### Validation Error

```
┌─────────────────────────────────────┐
│  ┌─────────────────────────────┐   │
│  │ user@invalid        ✕      │   │
│  │ ⚠️ Invalid email format     │   │  ← Error message
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ ••••••                      │   │
│  │ ⚠️ Password too short (min 8)│  │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Loading / Submitting

```
┌─────────────────────────────────────┐
│                                     │
│  ┌─────────────────────────────┐   │
│  │     Creating account...     │   │  ← Progress indicator
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

### Success

```
                    ┌──────────────┐
                    │   Success!   │
                    │ Account      │
                    │ Created      │
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Key Backup  │
                    │  Setup       │
                    └──────────────┘
```

## State Flow

```
                    ┌──────────────┐
                    │    Idle      │
                    └──────┬───────┘
                           │
                          Fill
                           │
                           ▼
                    ┌──────────────┐
                    │   Validating │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌──────────┐ ┌──────────┐ ┌──────────┐
       │  Valid   │ │ Invalid  │ │ Terms    │
       │          │ │ Fields   │ │ Not      │
       │          │ │          │ │ Agreed   │
       └────┬─────┘ └──────────┘ └──────────┘
            │
            ▼
     ┌──────────────┐
     │   Submit     │
     │   Register   │
     └──────┬───────┘
            │
     ┌──────┴──────┐
     ▼             ▼
┌─────────┐  ┌──────────┐
│ Success │  │  Error   │
│ → Setup │  │  Show    │
└─────────┘  │  Message │
             └──────────┘
```

## User Flow

1. **User arrives from:** Login screen (Register link)
2. **User can:**
   - Enter username
   - Enter email
   - Enter password
   - Confirm password
   - Toggle password visibility
   - Agree to terms
   - Submit registration
3. **User navigates to:**
   - Key backup setup (success)
   - Login screen (has account link)

## Validation Rules

| Field | Rule |
|-------|------|
| Username | 3-20 chars, alphanumeric |
| Email | Valid email format |
| Password | 8+ chars, mixed case, number |
| Confirm | Must match password |
| Terms | Must be checked |

## Notes

- Multi-step validation with inline errors
- Terms acceptance required
- Password strength indicator
- Creates encryption keys on success
