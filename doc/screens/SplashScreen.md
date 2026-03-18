# SplashScreen

> App initialization and routing screen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/splash/SplashScreen.kt`

## Overview

SplashScreen is the entry point of the application, displaying branding while performing initialization checks and routing users to the appropriate screen based on their authentication state.

## Functions

### SplashScreen
```kotlin
@Composable
fun SplashScreen(
    onNavigateToOnboarding: () -> Unit,
    onNavigateToLogin: () -> Unit,
    onNavigateToHome: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Main splash screen composable with branding animations and navigation logic.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `onNavigateToOnboarding` | `() -> Unit` | Navigate to onboarding flow |
| `onNavigateToLogin` | `() -> Unit` | Navigate to login screen |
| `onNavigateToHome` | `() -> Unit` | Navigate to home screen |
| `modifier` | `Modifier` | Optional styling modifier |

---

### Logo
```kotlin
@Composable
private fun Logo(
    modifier: Modifier = Modifier
)
```

**Description:** Displays the app logo with scaling animation.

---

### PlaceholderLogo
```kotlin
@Composable
private fun PlaceholderLogo()
```

**Description:** Temporary shield emoji logo until final asset is available.

---

### AppName
```kotlin
@Composable
private fun AppName(
    modifier: Modifier = Modifier
)
```

**Description:** Displays "ArmorClaw" app name text.

---

### Tagline
```kotlin
@Composable
private fun Tagline(
    modifier: Modifier = Modifier
)
```

**Description:** Displays "Secure. Private. Encrypted." tagline.

---

### LoadingIndicator
```kotlin
@Composable
private fun LoadingIndicator(
    modifier: Modifier = Modifier
)
```

**Description:** Circular progress indicator showing initialization progress.

---

## Screen Layout

```
┌────────────────────────────────────┐
│                                    │
│                                    │
│          ┌─────────┐               │
│          │   🛡️    │               │  ← Logo (animated)
│          │  Logo   │               │
│          └─────────┘               │
│                                    │
│          ArmorClaw                 │  ← App name
│                                    │
│   Secure. Private. Encrypted.     │  ← Tagline
│                                    │
│                                    │
│              ◌                     │  ← Loading indicator
│                                    │
│                                    │
└────────────────────────────────────┘
```

---

## Navigation Logic

### Routing Decision
```kotlin
LaunchedEffect(Unit) {
    // Fade in and scale up logo
    alpha = 1f
    scale = 1f

    // Simulate initialization
    delay(1500)

    // Check onboarding status from DataStore
    val hasCompletedOnboarding = false
    val isLoggedIn = false

    when {
        !hasCompletedOnboarding -> onNavigateToOnboarding()
        !isLoggedIn -> onNavigateToLogin()
        else -> onNavigateToHome()
    }
}
```

### Routes

| Condition | Destination | Description |
|-----------|-------------|-------------|
| First launch | Onboarding | User needs setup |
| Not logged in | Login | User needs auth |
| Logged in | Home | Proceed to app |

---

## Animations

### Animation Specs
```kotlin
val alphaSpec = tween<Float>(
    durationMillis = 800,
    easing = FastOutSlowInEasing
)

val scaleSpec = spring<Float>(
    dampingRatio = Spring.DampingRatioMediumBouncy
)
```

### Animation Properties
| Element | Property | Duration | Easing |
|---------|----------|----------|--------|
| Logo | Alpha | 800ms | FastOutSlowIn |
| Logo | Scale | Spring | MediumBouncy |
| Name | Alpha | 800ms | FastOutSlowIn |
| Tagline | Alpha | 800ms | FastOutSlowIn |
| Indicator | Alpha | 800ms | FastOutSlowIn |

---

## State Management

### Animation States
```kotlin
var alpha by remember { mutableFloatStateOf(0f) }
var scale by remember { mutableFloatStateOf(0.8f) }

val animatedAlpha by animateFloatAsState(
    targetValue = alpha,
    animationSpec = alphaSpec,
    label = "alpha"
)

val animatedScale by animateFloatAsState(
    targetValue = scale,
    animationSpec = scaleSpec,
    label = "scale"
)
```

---

## TODO Items

- [ ] Replace PlaceholderLogo with actual app logo asset
- [ ] Implement DataStore check for onboarding completion
- [ ] Implement actual authentication state check
- [ ] Add error handling for initialization failures
- [ ] Add version display at bottom

---

## Preview

```kotlin
@Preview(showBackground = true)
@Composable
private fun SplashScreenPreview() {
    ArmorClawTheme {
        SplashScreen(
            onNavigateToOnboarding = {},
            onNavigateToLogin = {},
            onNavigateToHome = {}
        )
    }
}
```

---

## Related Documentation

- [Splash Screen Feature](../features/splash-screen.md) - Feature overview
- [Onboarding](../features/onboarding.md) - Onboarding flow
- [Authentication](../features/authentication.md) - Login flow
