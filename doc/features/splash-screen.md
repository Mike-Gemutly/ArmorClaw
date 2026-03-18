# Splash Screen Feature

> App initialization and routing screen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/splash/`

## Overview

The splash screen is the first screen users see when opening ArmorClaw. It displays branding, performs initialization checks, and routes users to the appropriate screen based on their authentication and onboarding status.

## Feature Components

### SplashScreen
**Location:** `splash/SplashScreen.kt`

Main splash screen composable with branding and routing logic.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `SplashScreen()` | Main splash screen | `onNavigateToOnboarding`, `onNavigateToLogin`, `onNavigateToHome` |
| `Logo()` | App logo display | `modifier` |
| `PlaceholderLogo()` | Shield icon placeholder | - |
| `AppName()` | App name text | `modifier` |
| `Tagline()` | Tagline text | `modifier` |
| `LoadingIndicator()` | Circular loading indicator | `modifier` |

#### Screen Layout
```
┌────────────────────────────────────┐
│                                    │
│                                    │
│          ┌─────────┐               │
│          │   🛡️    │               │
│          │  Logo   │               │
│          └─────────┘               │
│                                    │
│          ArmorClaw                 │
│                                    │
│   Secure. Private. Encrypted.     │
│                                    │
│              ◌                     │  ← Loading indicator
│                                    │
│                                    │
└────────────────────────────────────┘
```

---

## Navigation Logic

### Routing Flow
```kotlin
when {
    !hasCompletedOnboarding -> onNavigateToOnboarding()
    !isLoggedIn -> onNavigateToLogin()
    else -> onNavigateToHome()
}
```

### Routes

| Condition | Destination | Description |
|-----------|-------------|-------------|
| First launch | Onboarding | User needs to complete setup |
| Not logged in | Login | User needs to authenticate |
| Logged in | Home | User proceeds to main app |

---

## Animations

### Logo Animation
```kotlin
val alphaSpec = tween<Float>(durationMillis = 800, easing = FastOutSlowInEasing)
val scaleSpec = spring<Float>(dampingRatio = Spring.DampingRatioMediumBouncy)
```

**Animation Properties:**
- **Alpha:** Fade in over 800ms with FastOutSlowInEasing
- **Scale:** Bouncy spring animation from 0.8f to 1f

### Animation Sequence
1. Logo fades in and scales up (0-800ms)
2. App name and tagline fade in
3. Loading indicator appears (50% opacity)
4. Initialization check (1500ms delay)
5. Navigate to appropriate screen

---

## Implementation Details

### State Management
```kotlin
var alpha by remember { mutableFloatStateOf(0f) }
var scale by remember { mutableFloatStateOf(0.8f) }
```

### Initialization
```kotlin
LaunchedEffect(Unit) {
    alpha = 1f
    scale = 1f
    delay(1500)
    // Check onboarding status from DataStore
    // Route accordingly
}
```

---

## TODO Items

- Replace `PlaceholderLogo()` with actual app logo asset
- Implement DataStore check for onboarding completion
- Implement actual authentication state check
- Add error handling for initialization failures

---

## Related Documentation

- [Authentication](authentication.md) - Login flow
- [Onboarding](onboarding.md) - First-time user setup
- [Home Screen](home-screen.md) - Main app destination
