# Biometric Authentication Feature

> Fingerprint and face authentication
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/platform/`

## Overview

Biometric authentication provides secure and convenient access to ArmorClaw using fingerprint or face recognition, replacing traditional password entry while maintaining security.

## Feature Components

### BiometricAuth Service
**Location:** `platform/BiometricAuth.kt`

Platform service for biometric authentication.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `isAvailable()` | Check if biometrics available | - |
| `authenticate()` | Trigger authentication | `title`, `subtitle`, `callback` |
| `enableBiometrics()` | Enable for user | `userId` |
| `disableBiometrics()` | Disable for user | `userId` |
| `hasBiometricCredential()` | Check if enrolled | `userId` |

#### Authentication Flow
```
┌────────────────────────────────────┐
│                                    │
│     ArmorClaw                      │
│                                    │
│     Use biometrics to unlock       │
│                                    │
│          ┌─────────┐               │
│          │   👆    │               │
│          │ Fingerprint             │
│          └─────────┘               │
│                                    │
│     [ Use Password Instead ]       │
│                                    │
└────────────────────────────────────┘
```

---

### BiometricPromptHandler
**Location:** `platform/BiometricPromptHandler.kt`

Android BiometricPrompt integration.

#### Functions

| Function | Description |
|----------|-------------|
| `showPrompt()` | Display biometric prompt |
| `handleResult()` | Process authentication result |
| `onSuccess()` | Authentication succeeded callback |
| `onError()` | Authentication failed callback |
| `onFailed()` | Biometric not recognized |

---

## Platform Expect/Actual

### Expect Declaration (shared)
```kotlin
// shared/src/commonMain/kotlin/platform/BiometricAuth.kt
expect class BiometricAuth {
    suspend fun isAvailable(): Boolean
    suspend fun authenticate(title: String, subtitle: String?): BiometricResult
    fun enableBiometrics(userId: String)
    fun disableBiometrics(userId: String)
}

sealed class BiometricResult {
    object Success : BiometricResult()
    data class Error(val message: String) : BiometricResult()
    object Failed : BiometricResult()
    object Cancelled : BiometricResult()
}
```

### Android Actual Implementation
```kotlin
// androidApp/src/androidMain/kotlin/platform/BiometricAuth.kt
actual class BiometricAuth(
    private val context: Context,
    private val activityProvider: () -> FragmentActivity
) {
    private val biometricManager = BiometricManager.from(context)

    actual suspend fun isAvailable(): Boolean {
        return biometricManager.canAuthenticate(
            BiometricManager.Authenticators.BIOMETRIC_STRONG
        ) == BiometricManager.BIOMETRIC_SUCCESS
    }

    actual suspend fun authenticate(
        title: String,
        subtitle: String?
    ): BiometricResult {
        // BiometricPrompt implementation
    }
}
```

---

## Authentication Types

### Supported Biometrics
| Type | API Level | Description |
|------|-----------|-------------|
| Fingerprint | 23+ | Fingerprint sensor |
| Face | 28+ | Face recognition |
| Iris | 28+ | Iris scanning |

### Authenticator Types
| Type | Constant | Security Level |
|------|----------|----------------|
| Strong | BIOMETRIC_STRONG | Class 3 biometric |
| Weak | BIOMETRIC_WEAK | Class 2 biometric |
| Device Credential | DEVICE_CREDENTIAL | PIN/Pattern/Password |

---

## Usage Examples

### In LoginScreen
```kotlin
@Composable
fun LoginScreen(onBiometricLogin: () -> Unit) {
    val biometricAuth = remember { BiometricAuth(context, activityProvider) }
    var isBiometricAvailable by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        isBiometricAvailable = biometricAuth.isAvailable()
    }

    if (isBiometricAvailable) {
        OutlinedButton(onClick = onBiometricLogin) {
            Icon(Icons.Default.Fingerprint, null)
            Spacer(Modifier.width(8.dp))
            Text("Log in with Biometrics")
        }
    }
}
```

### In App Lock
```kotlin
suspend fun authenticateUser(): Boolean {
    val result = biometricAuth.authenticate(
        title = "Unlock ArmorClaw",
        subtitle = "Use your biometrics to access the app"
    )
    return result is BiometricResult.Success
}
```

---

## Security Considerations

### Key Storage
- Biometric keys stored in AndroidKeyStore
- Keys require biometric authentication to use
- Keys invalidated on biometric enrollment change

### Fallback Options
- Always offer password fallback
- After 3 failed attempts, require password
- Biometric can be disabled in settings

### CryptoObject Usage
```kotlin
val cipher = cryptoManager.getDecryptionCipher()
val cryptoObject = BiometricPrompt.CryptoObject(cipher)

biometricPrompt.authenticate(
    promptInfo,
    cryptoObject
)
```

---

## Error Handling

### Error Types
| Error | Code | User Message |
|-------|------|--------------|
| HW_UNAVAILABLE | 1 | "Biometric hardware unavailable" |
| HW_UNAVAILABLE | 2 | "No biometric hardware" |
| NONE_ENROLLED | 11 | "No biometrics enrolled" |
| NO_SPACE | 4 | "Not enough storage" |
| TIMEOUT | 3 | "Authentication timeout" |
| LOCKED_OUT | 7 | "Too many attempts. Try later" |
| LOCKED_OUT_PERMANENT | 8 | "Biometric locked. Use password" |

### Error Recovery
```kotlin
when (result) {
    is BiometricResult.Error -> {
        if (result.code == ERROR_LOCKED_OUT) {
            // Show cooldown timer
        } else {
            // Offer password fallback
        }
    }
    is BiometricResult.Failed -> {
        // Retry or use password
    }
}
```

---

## Settings Integration

### SecuritySettingsScreen
```kotlin
// In SecuritySettingsScreen.kt
SettingsItem(
    title = "Biometric Login",
    subtitle = if (isBiometricEnabled) "Enabled" else "Disabled",
    onClick = { showBiometricToggle() }
)
```

### Enable/Disable Flow
1. User toggles setting
2. Authenticate to confirm
3. Store preference in DataStore
4. Update login behavior

---

## Testing

### Unit Tests
```kotlin
@Test
fun `isAvailable returns true when hardware present`() = runTest {
    // Mock BiometricManager
    whenever(biometricManager.canAuthenticate(any())).thenReturn(
        BiometricManager.BIOMETRIC_SUCCESS
    )

    assertTrue(biometricAuth.isAvailable())
}
```

### UI Tests
```kotlin
@Test
fun biometricButton_shows_when_available() {
    composeTestRule.setContent {
        LoginScreen(onBiometricLogin = {})
    }

    composeTestRule
        .onNodeWithText("Log in with Biometrics")
        .assertIsDisplayed()
}
```

---

## Related Documentation

- [Authentication](authentication.md) - Login flow
- [Security Settings](settings.md#security-settings) - Security preferences
- [Device Management](device-management.md) - Device verification
