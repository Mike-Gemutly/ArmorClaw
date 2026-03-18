# WelcomeScreen

> Initial onboarding screen for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/onboarding/WelcomeScreen.kt`

## Overview

WelcomeScreen is the first screen users see when launching ArmorClaw for the first time. It introduces the app's key features and provides options to begin the onboarding process.

## Screen Layout

```
┌────────────────────────────────────┐
│                                    │
│          🔒 ARMORCLAW              │
│         [Animated Logo]            │
│                                    │
├────────────────────────────────────┤
│                                    │
│  Welcome to ArmorClaw              │
│                                    │
│  ✓ End-to-End Encryption           │
│  ✓ Secure Messaging                │
│  ✓ Biometric Authentication        │
│  ✓ Open Source                     │
│                                    │
├────────────────────────────────────┤
│                                    │
│  [      GET STARTED      ]         │
│                                    │
│  Already have an account? Login    │
│                                    │
└────────────────────────────────────┘
```

## Functions

### WelcomeScreen
```kotlin
@Composable
fun WelcomeScreen(
    onGetStarted: () -> Unit,
    onSkip: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Main composable for the welcome screen.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `onGetStarted` | `() -> Unit` | Callback when user taps "Get Started" |
| `onSkip` | `() -> Unit` | Callback when user skips onboarding |
| `modifier` | `Modifier` | Optional modifier for styling |

---

### FeatureList
```kotlin
@Composable
private fun FeatureList(
    features: List<FeatureItem>,
    modifier: Modifier = Modifier
)
```

**Description:** Animated list of app features.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `features` | `List<FeatureItem>` | List of features to display |
| `modifier` | `Modifier` | Optional modifier |

**Features Displayed:**
1. End-to-End Encryption (Lock icon)
2. Secure Messaging (Chat icon)
3. Biometric Authentication (Fingerprint icon)
4. Open Source (Code icon)

---

### FeatureItem
```kotlin
@Composable
private fun FeatureItem(
    icon: ImageVector,
    title: String,
    description: String,
    modifier: Modifier = Modifier
)
```

**Description:** Individual feature card with icon and text.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `icon` | `ImageVector` | Feature icon |
| `title` | `String` | Feature title |
| `description` | `String` | Feature description |
| `modifier` | `Modifier` | Optional modifier |

---

### AnimatedLogo
```kotlin
@Composable
private fun AnimatedLogo(
    modifier: Modifier = Modifier
)
```

**Description:** Animated ArmorClaw logo with fade-in and scale effects.

**Animations:**
- Initial fade-in (300ms)
- Scale from 0.8 to 1.0 (300ms)
- Subtle pulse effect

---

## Data Models

### FeatureItem
```kotlin
data class FeatureItem(
    val icon: ImageVector,
    val title: String,
    val description: String
)
```

### Sample Features
```kotlin
val defaultFeatures = listOf(
    FeatureItem(
        icon = Icons.Default.Lock,
        title = "End-to-End Encryption",
        description = "Your messages are secured with military-grade encryption"
    ),
    FeatureItem(
        icon = Icons.Default.Chat,
        title = "Secure Messaging",
        description = "Private conversations that stay private"
    ),
    FeatureItem(
        icon = Icons.Default.Fingerprint,
        title = "Biometric Authentication",
        description = "Unlock with your fingerprint or face"
    ),
    FeatureItem(
        icon = Icons.Default.Code,
        title = "Open Source",
        description = "Fully auditable security you can trust"
    )
)
```

---

## State Management

The screen is stateless and relies on parent navigation for state.

---

## Navigation

### Entry Points
- First app launch (onboarding not complete)
- Manual navigation to welcome

### Exit Points
| Action | Destination |
|--------|-------------|
| Get Started | SecurityExplanationScreen |
| Skip | HomeScreen (if logged in) or LoginScreen |
| Login | LoginScreen |

---

## Styling

### Colors
| Element | Light Theme | Dark Theme |
|---------|-------------|------------|
| Background | Surface | Surface |
| Title | OnBackground | OnBackground |
| Feature Icon | BrandPurple | BrandPurple |
| Button | BrandPurple | BrandPurple |

### Typography
| Element | Style |
|---------|-------|
| Title | headlineMedium |
| Feature Title | titleMedium |
| Description | bodyMedium |
| Button | labelLarge |

---

## Accessibility

### Content Descriptions
- Logo: "ArmorClaw secure messaging app logo"
- Get Started button: "Start the setup process"
- Skip link: "Skip the introduction"

### Semantic Properties
- Features grouped as list
- Button has role = Role.Button
- Proper heading hierarchy

---

## Testing

### Test Scenarios
1. **Display**: Verify all features are shown
2. **Get Started**: Navigation to security screen
3. **Skip**: Navigation to login/home
4. **Animations**: Verify smooth transitions
5. **Accessibility**: Screen reader compatibility

### Compose Testing
```kotlin
@Test
fun welcomeScreen_displaysFeatures() {
    composeTestRule.setContent {
        WelcomeScreen(
            onGetStarted = {},
            onSkip = {}
        )
    }

    composeTestRule
        .onNodeWithText("End-to-End Encryption")
        .assertIsDisplayed()
}
```

---

## Related Documentation

- [Onboarding](../features/onboarding.md) - Full onboarding flow
- [SecurityExplanationScreen](SecurityExplanationScreen.md) - Next screen
