# Forgot Password Screen

> **Route:** `forgot_password`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/ForgotPasswordScreen.kt`
> **Category:** Authentication

## Screenshot

![Forgot Password Screen](../../screenshots/auth/forgot-password.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←                                   │  ← TopAppBar
├─────────────────────────────────────┤
│                                     │
│              🔑                     │  ← Icon
│                                     │
│        Forgot Password?             │  ← Title
│                                     │
│  Enter your email address and we'll │
│  send you instructions to reset     │
│  your password.                     │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 📧 Email                    │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │       Send Reset Link       │   │  ← Primary button
│  └─────────────────────────────┘   │
│                                     │
│  Remember your password? Login      │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Default

```
┌─────────────────────────────────────┐
│        Forgot Password?             │
│                                     │
│  [Empty email field]                │
│  [Disabled Send button]             │
│                                     │
└─────────────────────────────────────┘
```

### Sending

```
┌─────────────────────────────────────┐
│                                     │
│  ┌─────────────────────────────┐   │
│  │       Sending...            │   │  ← Progress
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

### Success

```
┌─────────────────────────────────────┐
│              ✅                     │
│        Email Sent!                  │
│                                     │
│  Check your inbox for reset         │
│  instructions.                      │
│                                     │
│  ┌─────────────────────────────┐   │
│  │     Back to Login           │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Error

```
┌─────────────────────────────────────┐
│  ⚠️ Error                           │
│                                     │
│  No account found with this email.  │
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
          │ Email    │
          └────┬─────┘
               │
               ▼
          ┌──────────┐
          │ Sending  │
          └────┬─────┘
               │
        ┌──────┴──────┐
        ▼             ▼
   ┌─────────┐   ┌─────────┐
   │ Success │   │  Error  │
   │ → Login │   │  Retry  │
   └─────────┘   └─────────┘
```

## Notes

- Simple email-based recovery
- Rate limited to prevent abuse
- Clear success/error feedback
