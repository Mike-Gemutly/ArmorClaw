# LoginScreen

> User authentication login screen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/LoginScreen.kt`

## Overview

LoginScreen provides the user authentication interface for existing users to log in with their credentials or biometric authentication.

## Functions

### LoginScreen
```kotlin
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoginScreen(
    onLogin: (username: String, password: String) -> Unit,
    onRegister: () -> Unit,
    onForgotPassword: () -> Unit,
    onBiometricLogin: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Main login screen with form fields and authentication options.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `onLogin` | `(String, String) -> Unit` | Callback with credentials |
| `onRegister` | `() -> Unit` | Navigate to registration |
| `onForgotPassword` | `() -> Unit` | Navigate to password recovery |
| `onBiometricLogin` | `() -> Unit` | Trigger biometric auth |
| `modifier` | `Modifier` | Optional styling modifier |

---

### Logo
```kposable
@Composable
private fun Logo()
```

**Description:** Displays the app logo (shield emoji placeholder).

---

## Screen Layout

```
┌────────────────────────────────────┐
│                                    │
│          ┌─────────┐               │
│          │   🛡️    │               │
│          └─────────┘               │
│          ArmorClaw                 │
│   Secure. Private. Encrypted.     │
│                                    │
│  ┌──────────────────────────────┐  │
│  │ 👤 Username or Email         │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ Enter your username...   │ │  │
│  │ └──────────────────────────┘ │  │
│  │                              │  │
│  │ 🔒 Password                  │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ ••••••••••    👁️ ✕      │ │  │
│  │ └──────────────────────────┘ │  │
│  │                    Forgot?   │  │
│  │                              │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │        Log In            │ │  │
│  │ └──────────────────────────┘ │  │
│  │                              │  │
│  │ ─────── OR ───────          │  │
│  │                              │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ 🔐 Log in with Biometrics│ │  │
│  │ └──────────────────────────┘ │  │
│  └──────────────────────────────┘  │
│                                    │
│  Don't have an account? Register  │
│                                    │
│  By continuing, you agree to our  │
│  Terms of Service and Privacy     │
│                                    │
│         Version 1.0.0              │
└────────────────────────────────────┘
```

---

## Form Fields

### Username/Email Field
```kotlin
OutlinedTextField(
    value = username,
    onValueChange = { username = it },
    label = { Text("Username or Email") },
    placeholder = { Text("Enter your username or email") },
    leadingIcon = { Icon(Icons.Default.Person, null) },
    trailingIcon = {
        if (username.isNotEmpty()) {
            IconButton(onClick = { username = "" }) {
                Icon(Icons.Default.Clear, "Clear")
            }
        }
    },
    keyboardOptions = KeyboardOptions(
        keyboardType = KeyboardType.Email,
        imeAction = ImeAction.Next
    )
)
```

### Password Field
```kotlin
OutlinedTextField(
    value = password,
    onValueChange = { password = it },
    label = { Text("Password") },
    visualTransformation = if (isPasswordVisible)
        VisualTransformation.None
    else
        PasswordVisualTransformation(),
    leadingIcon = { Icon(Icons.Default.Lock, null) },
    trailingIcon = {
        Row {
            IconButton(onClick = { isPasswordVisible = !isPasswordVisible }) {
                Icon(
                    if (isPasswordVisible) Icons.Default.Visibility
                    else Icons.Default.VisibilityOff,
                    null
                )
            }
        }
    },
    keyboardOptions = KeyboardOptions(
        keyboardType = KeyboardType.Password,
        imeAction = ImeAction.Done
    )
)
```

---

## Validation

### Form Validation
```kotlin
val isValid = username.isNotBlank() && password.isNotBlank()
```

### Button States
| Condition | Button State |
|-----------|--------------|
| Both fields empty | Disabled |
| One field empty | Disabled |
| Both fields filled | Enabled |

---

## State Management

### Form State
```kotlin
var username by remember { mutableStateOf("") }
var password by remember { mutableStateOf("") }
var isPasswordVisible by remember { mutableStateOf(false) }
var isBiometricAvailable by remember { mutableStateOf(true) }
```

### Focus Management
```kotlin
val passwordFocusRequester = FocusRequester()

// In username field:
keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
keyboardActions = KeyboardActions(
    onNext = { passwordFocusRequester.requestFocus() }
)
```

---

## Actions

### Login Actions
| Action | Trigger | Result |
|--------|---------|--------|
| Login | Tap button | `onLogin(username, password)` |
| Register | Tap link | Navigate to RegistrationScreen |
| Forgot Password | Tap link | Navigate to ForgotPasswordScreen |
| Biometric | Tap button | `onBiometricLogin()` |
| Submit | Keyboard Done | Trigger login if valid |

---

## Accessibility

### Content Descriptions
- Clear button: "Clear"
- Visibility toggle: "Show/Hide password"
- Biometric button: "Log in with biometrics"

### Semantic Properties
- Username field: TextInput with email type
- Password field: TextInput with password type
- Login button: Button with enabled state

---

## Preview

```kotlin
@Preview(showBackground = true)
@Composable
private fun LoginScreenPreview() {
    ArmorClawTheme {
        LoginScreen(
            onLogin = { _, _ -> },
            onRegister = {},
            onForgotPassword = {},
            onBiometricLogin = {}
        )
    }
}
```

---

## Related Documentation

- [Authentication](../features/authentication.md) - Auth feature
- [RegistrationScreen](RegistrationScreen.md) - Registration screen
- [ForgotPasswordScreen](ForgotPasswordScreen.md) - Password recovery
- [Biometric Auth](../features/biometric-auth.md) - Biometric login
