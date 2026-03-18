# Authentication Feature

> Secure user authentication for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/`

## Overview

The authentication feature provides secure login, registration, and biometric authentication for ArmorClaw users.

## Feature Components

### LoginScreen
**Location:** `auth/LoginScreen.kt`

Handles user authentication with username/email and password.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `LoginScreen()` | Main login screen composable | `onLogin`, `onBiometricLogin`, `onForgotPassword`, `onRegister` |
| `LoginForm()` | Login form with validation | `onLogin`, `isLoading` |
| `BiometricButton()` | Biometric login trigger | `onBiometricLogin`, `enabled` |
| `SocialLoginButtons()` | Social login options (placeholder) | `onGoogleLogin`, `onAppleLogin` |

#### State Management
- Username/email input state
- Password input state with visibility toggle
- Loading state during authentication
- Error state display

#### User Flow
1. User enters username/email
2. User enters password
3. Optional: User taps biometric button
4. System authenticates
5. On success: Navigate to home
6. On failure: Display error message

---

### RegistrationScreen
**Location:** `auth/RegistrationScreen.kt`

Handles new user registration.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `RegistrationScreen()` | Main registration screen | `onRegister`, `onNavigateBack` |
| `RegistrationForm()` | Registration form with validation | `onRegister`, `isLoading` |
| `TermsCheckbox()` | Terms acceptance checkbox | `checked`, `onCheckedChange` |

#### Registration Fields
- Display name
- Email address
- Password (with confirmation)
- Terms acceptance

---

### ForgotPasswordScreen
**Location:** `auth/ForgotPasswordScreen.kt`

Handles password recovery flow.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `ForgotPasswordScreen()` | Password recovery screen | `onNavigateBack` |
| `EmailInputForm()` | Email input for recovery | `onSubmit` |
| `SuccessMessage()` | Recovery email sent confirmation | - |

---

## Biometric Authentication

### BiometricAuth Interface
**Location:** `shared/src/commonMain/kotlin/platform/BiometricAuth.kt`

```kotlin
interface BiometricAuth {
    suspend fun isAvailable(): Boolean
    suspend fun authenticate(prompt: String): Result<Boolean>
    fun getBiometricType(): BiometricType
}
```

### BiometricAuthImpl
**Location:** `androidApp/src/main/kotlin/com/armorclaw/app/platform/BiometricAuthImpl.kt`

Android implementation using AndroidX Biometric API.

#### Functions

| Function | Description |
|----------|-------------|
| `isAvailable()` | Check if biometric hardware is available |
| `authenticate()` | Show biometric prompt and authenticate |
| `getBiometricType()` | Returns FINGERPRINT, FACE, or NONE |
| `createPromptInfo()` | Create BiometricPrompt.PromptInfo |

#### Supported Methods
- Fingerprint (all Android versions)
- Face recognition (Android 10+)
- Iris scan (Android 10+)

---

## Security Implementation

### Session Management
- Encrypted session tokens
- Automatic session refresh
- Secure token storage in DataStore

### Password Security
- Minimum 8 characters
- Requires uppercase, lowercase, number
- Secure transmission via TLS

### Biometric Security
- Uses Android Keystore
- No biometric data stored
- Fallback to password available

---

## Navigation Flow

```
SplashScreen
    │
    ├── Onboarding Not Complete ──→ WelcomeScreen
    │
    └── Onboarding Complete
            │
            ├── Not Logged In ──→ LoginScreen
            │                          │
            │                          ├── Register ──→ RegistrationScreen
            │                          │
            │                          └── Forgot Password ──→ ForgotPasswordScreen
            │
            └── Logged In ──→ HomeScreen
```

---

## Related Documentation

- [Onboarding](onboarding.md) - Initial setup flow
- [Encryption](encryption.md) - Security implementation
- [Biometric Auth](biometric-auth.md) - Detailed biometric documentation
