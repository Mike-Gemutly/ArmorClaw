# BiometricAuth Platform Service

> Biometric authentication integration
> Location: `shared/src/commonMain/kotlin/platform/BiometricAuth.kt` (expect)
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/platform/BiometricAuth.kt` (actual)

## Overview

BiometricAuth provides cross-platform biometric authentication using Android's BiometricPrompt API for secure user authentication.

## Expect Declaration (shared)

```kotlin
// shared/src/commonMain/kotlin/platform/BiometricAuth.kt
expect class BiometricAuth {
    suspend fun isAvailable(): Boolean
    suspend fun authenticate(
        title: String,
        subtitle: String? = null
    ): BiometricResult

    fun enableBiometrics(userId: String)
    fun disableBiometrics(userId: String)
    fun hasBiometricCredential(userId: String): Boolean
}

sealed class BiometricResult {
    object Success : BiometricResult()
    data class Error(val code: Int, val message: String) : BiometricResult()
    object Failed : BiometricResult()
    object Cancelled : BiometricResult()
}
```

---

## Android Actual Implementation

```kotlin
// androidApp/src/main/kotlin/com/armorclaw/app/platform/BiometricAuth.kt
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
    ): BiometricResult = suspendCancellableCoroutine { continuation ->
        val activity = activityProvider()

        val promptInfo = BiometricPrompt.PromptInfo.Builder()
            .setTitle(title)
            .setSubtitle(subtitle ?: "")
            .setNegativeButtonText("Use password")
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build()

        val biometricPrompt = BiometricPrompt(
            activity,
            ContextCompat.getMainExecutor(context),
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                    continuation.resume(BiometricResult.Success)
                }

                override fun onAuthenticationFailed() {
                    // Don't resume here - user can retry
                }

                override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                    when (errorCode) {
                        BiometricPrompt.ERROR_USER_CANCELED,
                        BiometricPrompt.ERROR_NEGATIVE_BUTTON -> {
                            continuation.resume(BiometricResult.Cancelled)
                        }
                        BiometricPrompt.ERROR_TOO_MANY_ATTEMPTS -> {
                            continuation.resume(
                                BiometricResult.Error(errorCode, "Too many attempts")
                            )
                        }
                        else -> {
                            continuation.resume(
                                BiometricResult.Error(errorCode, errString.toString())
                            )
                        }
                    }
                }
            }
        )

        biometricPrompt.authenticate(promptInfo)

        continuation.invokeOnCancellation {
            biometricPrompt.cancelAuthentication()
        }
    }

    actual fun enableBiometrics(userId: String) {
        // Store preference in DataStore/SharedPreferences
        biometricPreferences.edit { putBoolean("biometric_$userId", true) }
    }

    actual fun disableBiometrics(userId: String) {
        biometricPreferences.edit { remove("biometric_$userId") }
    }

    actual fun hasBiometricCredential(userId: String): Boolean {
        return biometricPreferences.getBoolean("biometric_$userId", false)
    }
}
```

---

## Functions

### isAvailable
```kotlin
suspend fun isAvailable(): Boolean
```

**Description:** Checks if biometric authentication is available on the device.

**Returns:** `true` if biometric hardware is present and biometrics are enrolled.

---

### authenticate
```kotlin
suspend fun authenticate(
    title: String,
    subtitle: String? = null
): BiometricResult
```

**Description:** Prompts the user for biometric authentication.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `title` | `String` | Dialog title |
| `subtitle` | `String?` | Optional subtitle |

**Returns:** `BiometricResult` indicating success, failure, or error.

---

### enableBiometrics / disableBiometrics
```kotlin
fun enableBiometrics(userId: String)
fun disableBiometrics(userId: String)
```

**Description:** Enables or disables biometric login for a specific user.

---

### hasBiometricCredential
```kotlin
fun hasBiometricCredential(userId: String): Boolean
```

**Description:** Checks if biometric login is enabled for a user.

---

## BiometricResult

### Success
```kotlin
object Success : BiometricResult()
```
Authentication succeeded.

### Error
```kotlin
data class Error(val code: Int, val message: String) : BiometricResult()
```
Authentication failed with an error.

**Error Codes:**
| Code | Constant | Description |
|------|----------|-------------|
| 1 | ERROR_HW_UNAVAILABLE | Hardware unavailable |
| 2 | ERROR_UNABLE_TO_PROCESS | Unable to process |
| 3 | ERROR_TIMEOUT | Operation timed out |
| 7 | ERROR_LOCKED_OUT | Too many attempts |
| 8 | ERROR_LOCKED_OUT_PERMANENT | Biometric locked |
| 11 | ERROR_NO_BIOMETRICS | No biometrics enrolled |

### Failed
```kotlin
object Failed : BiometricResult()
```
Authentication failed (user can retry).

### Cancelled
```kotlin
object Cancelled : BiometricResult()
```
User cancelled the operation.

---

## Usage Examples

### Login Screen
```kotlin
@Composable
fun LoginScreen(viewModel: LoginViewModel) {
    val biometricAuth = LocalBiometricAuth.current
    var biometricResult by remember { mutableStateOf<BiometricResult?>(null) }

    LaunchedEffect(Unit) {
        if (biometricAuth.isAvailable()) {
            biometricResult = biometricAuth.authenticate(
                title = "Log in to ArmorClaw",
                subtitle = "Use your biometrics to continue"
            )
            when (biometricResult) {
                is BiometricResult.Success -> {
                    viewModel.biometricLogin()
                }
                else -> { /* Show password login */ }
            }
        }
    }
}
```

### App Lock
```kotlin
suspend fun unlockApp(): Boolean {
    val result = biometricAuth.authenticate(
        title = "Unlock ArmorClaw",
        subtitle = "Verify your identity"
    )
    return result is BiometricResult.Success
}
```

---

## Dependency Injection

### Koin Module
```kotlin
val platformModule = module {
    single<BiometricAuth> {
        BiometricAuth(
            context = androidContext(),
            activityProvider = { get<FragmentActivity>() }
        )
    }
}
```

---

## Testing

### Mock Implementation
```kotlin
class MockBiometricAuth : BiometricAuth {
    var isAvailableResult = true
    var authenticateResult: BiometricResult = BiometricResult.Success

    override suspend fun isAvailable() = isAvailableResult
    override suspend fun authenticate(title: String, subtitle: String?) = authenticateResult
    override fun enableBiometrics(userId: String) {}
    override fun disableBiometrics(userId: String) {}
    override fun hasBiometricCredential(userId: String) = true
}
```

---

## Related Documentation

- [Biometric Auth Feature](../features/biometric-auth.md) - Feature overview
- [LoginScreen](../screens/LoginScreen.md) - Login screen
- [Security Settings](../features/settings.md#security-settings) - Security preferences
