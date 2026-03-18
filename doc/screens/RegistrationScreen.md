# RegistrationScreen

> New user registration screen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/auth/RegistrationScreen.kt`

## Overview

RegistrationScreen allows new users to create an ArmorClaw account with username, email, and password credentials.

## Functions

### RegistrationScreen
```kotlin
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RegistrationScreen(
    onNavigateBack: () -> Unit,
    onRegister: (username: String, email: String, password: String) -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Main registration screen with form fields for account creation.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `onNavigateBack` | `() -> Unit` | Navigate back/close |
| `onRegister` | `(String, String, String) -> Unit` | Callback with credentials |
| `modifier` | `Modifier` | Optional styling modifier |

---

## Screen Layout

```
┌────────────────────────────────────┐
│  ✕ Create Account                 │
├────────────────────────────────────┤
│                                    │
│            👤 Add                  │
│       Create Account               │
│  Join ArmorClaw and chat securely │
│                                    │
│  ┌──────────────────────────────┐  │
│  │ 👤 Username                  │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ Choose a username...     │ │  │
│  │ └──────────────────────────┘ │  │
│  │                              │  │
│  │ 📧 Email Address             │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ Enter your email...      │ │  │
│  │ └──────────────────────────┘ │  │
│  │                              │  │
│  │ 🔒 Password                  │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ ••••••••    👁️ ✕        │ │  │
│  │ └──────────────────────────┘ │  │
│  │ ████░░░░ Medium             │  │  ← Password strength
│  │                              │  │
│  │ 🔒 Confirm Password          │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ ••••••••    👁️ ✕        │ │  │
│  │ └──────────────────────────┘ │  │
│  │ ⚠️ Passwords do not match   │  │  ← Error message
│  │                              │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │     Create Account       │ │  │
│  │ └──────────────────────────┘ │  │
│  └──────────────────────────────┘  │
│                                    │
│  Already have an account? Log In  │
│                                    │
│  By creating an account, you      │
│  agree to Terms and Privacy       │
└────────────────────────────────────┘
```

---

## Form Fields

### Username Field
```kotlin
OutlinedTextField(
    value = username,
    onValueChange = { username = it },
    label = { Text("Username") },
    placeholder = { Text("Choose a username") },
    leadingIcon = { Icon(Icons.Default.Person, null) },
    trailingIcon = {
        if (username.isNotEmpty()) {
            IconButton(onClick = { username = "" }) {
                Icon(Icons.Default.Clear, "Clear")
            }
        }
    },
    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next)
)
```

### Email Field
```kotlin
OutlinedTextField(
    value = email,
    onValueChange = { email = it },
    label = { Text("Email Address") },
    placeholder = { Text("Enter your email") },
    leadingIcon = { Icon(Icons.Default.Email, null) },
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
    placeholder = { Text("Create a password (min 8 chars)") },
    visualTransformation = if (isPasswordVisible)
        VisualTransformation.None
    else
        PasswordVisualTransformation(),
    leadingIcon = { Icon(Icons.Default.Lock, null) },
    keyboardOptions = KeyboardOptions(
        keyboardType = KeyboardType.Password,
        imeAction = ImeAction.Next
    )
)
```

### Confirm Password Field
```kotlin
OutlinedTextField(
    value = confirmPassword,
    onValueChange = { confirmPassword = it },
    label = { Text("Confirm Password") },
    placeholder = { Text("Enter password again") },
    isError = confirmPassword.isNotEmpty() && password != confirmPassword,
    supportingText = {
        if (confirmPassword.isNotEmpty() && password != confirmPassword) {
            Text("Passwords do not match", color = error)
        }
    },
    keyboardOptions = KeyboardOptions(
        keyboardType = KeyboardType.Password,
        imeAction = ImeAction.Done
    )
)
```

---

## Password Strength Indicator

### Strength Calculation
```kotlin
val passwordStrength = when {
    password.isEmpty() -> 0
    password.length < 6 -> 1      // Weak
    password.length < 8 -> 2      // Medium
    else -> 3                      // Strong
}
```

### Visual Display
```kotlin
Row {
    repeat(3) { index ->
        Box(
            modifier = Modifier
                .weight(1f)
                .height(4.dp)
                .background(
                    if (index < passwordStrength) AccentColor
                    else surfaceVariant
                )
        )
    }
    Text(
        when (passwordStrength) {
            1 -> "Weak"
            2 -> "Medium"
            3 -> "Strong"
            else -> ""
        }
    )
}
```

---

## Validation

### Form Validation
```kotlin
val isValid = username.isNotBlank() &&
    email.isNotBlank() &&
    email.contains("@") &&
    password.isNotBlank() &&
    password == confirmPassword &&
    password.length >= 8
```

### Validation Rules
| Field | Rules |
|-------|-------|
| Username | Required, non-blank |
| Email | Required, contains @ |
| Password | Required, min 8 chars |
| Confirm | Must match password |

---

## State Management

### Form State
```kotlin
var username by remember { mutableStateOf("") }
var email by remember { mutableStateOf("") }
var password by remember { mutableStateOf("") }
var confirmPassword by remember { mutableStateOf("") }
var isPasswordVisible by remember { mutableStateOf(false) }
var isConfirmPasswordVisible by remember { mutableStateOf(false) }
```

### Focus Management
```kotlin
val passwordFocusRequester = FocusRequester()
val confirmPasswordFocusRequester = FocusRequester()
```

---

## Actions

### Registration Actions
| Action | Trigger | Result |
|--------|---------|--------|
| Register | Tap button | `onRegister(username, email, password)` |
| Back | Tap X / back | `onNavigateBack()` |
| Login | Tap link | Navigate to LoginScreen |

---

## Preview

```kotlin
@Preview(showBackground = true)
@Composable
private fun RegistrationScreenPreview() {
    ArmorClawTheme {
        RegistrationScreen(
            onNavigateBack = {},
            onRegister = { _, _, _ -> }
        )
    }
}
```

---

## Related Documentation

- [Authentication](../features/authentication.md) - Auth feature
- [LoginScreen](LoginScreen.md) - Login screen
- [ForgotPasswordScreen](ForgotPasswordScreen.md) - Password recovery
