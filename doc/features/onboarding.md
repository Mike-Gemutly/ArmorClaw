# Onboarding Feature

> First-time user experience for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/onboarding/`

## Overview

The onboarding feature guides new users through the initial setup process, introducing key features and configuring the app for first use.

## Onboarding Flow

```
WelcomeScreen → SecurityExplanationScreen → ConnectServerScreen → PermissionsScreen → CompletionScreen → KeyBackupSetupScreen → HomeScreen

(If legacy Bridge session detected at splash: SplashScreen → MigrationScreen → HomeScreen)
```

## Feature Components

### WelcomeScreen
**Location:** `onboarding/WelcomeScreen.kt`

Initial welcome screen introducing ArmorClaw features.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `WelcomeScreen()` | Main welcome screen composable | `onGetStarted`, `onSkip` |
| `FeatureList()` | Animated feature list | `features` |
| `FeatureItem()` | Individual feature card | `icon`, `title`, `description` |
| `AnimatedLogo()` | Animated ArmorClaw logo | - |

#### Features Highlighted
- End-to-end encryption
- Secure messaging
- Biometric authentication
- Open source

#### User Actions
- **Get Started** → Proceed to security explanation
- **Skip** → Skip onboarding (not recommended)

---

### SecurityExplanationScreen
**Location:** `onboarding/SecurityExplanationScreen.kt`

Visual explanation of security features in 4 steps.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `SecurityExplanationScreen()` | Main security screen | `onNext`, `onBack` |
| `SecurityStep()` | Security step data class | `id`, `title`, `description`, `icon` |
| `StepIndicator()` | Progress dots indicator | `currentStep`, `totalSteps` |
| `AnimatedSecurityDiagram()` | Visual security animation | `currentStep` |
| `StepNode()` | Individual step circle | `step`, `isSelected`, `onClick` |
| `StepDetails()` | Step description card | `step` |

#### Security Steps
1. **Encryption** - Messages are encrypted on your device
2. **Keys** - Only you have the decryption keys
3. **Transport** - Secure transmission to recipients
4. **Verification** - Verify recipient identity

#### Animations
- Step indicator progress animation
- Security diagram transitions
- Confetti on completion

---

### ConnectServerScreen
**Location:** `onboarding/ConnectServerScreen.kt`

Configure Matrix homeserver connection.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `ConnectServerScreen()` | Main server connection screen | `onConnected`, `onBack` |
| `ServerInputForm()` | Homeserver URL input | `onConnect` |
| `ConnectionStatus()` | Connection progress indicator | `status` |
| `ServerInfoCard()` | Display server information | `serverInfo` |

#### Server Configuration
- **Homeserver URL** - Matrix server address
- **Username** - Matrix username
- **Demo Mode** - Use demo server for testing

#### Connection Flow
1. User enters homeserver URL
2. System validates URL
3. System fetches server info
4. User enters credentials
5. System authenticates
6. On success: Navigate to permissions

---

### PermissionsScreen
**Location:** `onboarding/PermissionsScreen.kt`

Request required and optional permissions.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `PermissionsScreen()` | Main permissions screen | `onComplete`, `onBack` |
| `PermissionItem()` | Individual permission card | `permission`, `onRequest` |
| `PermissionInfoCard()` | Why permissions are needed | - |
| `PermissionProgressIndicator()` | Overall progress | `granted`, `total` |

#### Permission Model

| Permission | Type | Purpose |
|------------|------|---------|
| Notifications | Required | Message alerts |
| Camera | Optional | Profile photos, attachments |
| Microphone | Optional | Voice messages |
| Storage | Optional | File attachments |
| Location | Optional | Location sharing |

#### Permission Card Features
- Permission icon
- Title and description
- Required/optional badge
- Grant status indicator
- Animated card on grant

---

### CompletionScreen
**Location:** `onboarding/CompletionScreen.kt`

Celebrate setup completion and show next steps.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `CompletionScreen()` | Main completion screen | `onStartChatting`, `onTakeTutorial` |
| `ConfettiParticles()` | Animated confetti effect | `showAnimation` |
| `WhatsNextCard()` | "What's Next" suggestions | - |
| `QuickTipCard()` | Quick tips for new users | - |
| `FeatureHighlightCard()` | Feature highlight carousel | `features` |

#### Next Steps Suggested
1. Start a chat
2. Invite friends
3. Customize settings
4. Enable biometric

#### Animations
- Confetti particle system
- Success checkmark animation
- Card entrance animations
- Feature carousel

---

### MigrationScreen (NEW 2026-02-24)
**Location:** `onboarding/MigrationScreen.kt`

Handles v2.5 → v3.0 migration for users with legacy Bridge-only sessions.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `MigrationScreen()` | Main migration screen | `onComplete`, `onSkip` |
| `LegacySessionDetected()` | Auto-detected banner | `session` |
| `RecoveryPhraseEntry()` | Manual entry fallback | `onSubmit` |
| `MigrationProgress()` | Progress indicator | `state` |
| `MigrationResult()` | Success/failure result | `success`, `error` |

#### Detection Logic
- `SplashViewModel` checks `AppPreferences.hasLegacyBridgeSession()`
- If true → navigates to `SplashTarget.Migration` → `MigrationScreen`
- On completion → `clearLegacyBridgeSession()` → Home

---

### KeyBackupSetupScreen (NEW 2026-02-24)
**Location:** `onboarding/KeyBackupSetupScreen.kt`

6-step guided key backup setup, mandatory during onboarding.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `KeyBackupSetupScreen()` | Main key backup screen | `onComplete`, `onBack` |
| `ExplainStep()` | Why backup matters | - |
| `GenerateStep()` | Generate recovery phrase | - |
| `DisplayStep()` | Show 12-word phrase | `words` |
| `VerifyStep()` | User confirms words | `expectedWords` |
| `StoreStep()` | Upload encrypted backup | - |
| `SuccessStep()` | Confirmation | - |

#### Flow
```
Explain → Generate → Display → Verify → Store → Success
```

- **Mandatory in onboarding**: CompletionScreen → KeyBackupSetupScreen → HomeScreen
- **Optional re-entry**: SecuritySettingsScreen → KeyBackupSetupScreen
- 12-word BIP39-compatible recovery phrase
- Encrypted backup uploaded to Matrix homeserver (SSSS)

---

## State Persistence

### OnboardingState
**Location:** `androidApp/src/main/kotlin/com/armorclaw/app/data/persistence/OnboardingState.kt`

```kotlin
data class OnboardingState(
    val hasCompletedOnboarding: Boolean,
    val currentStep: Int,
    val serverUrl: String?,
    val permissionsGranted: List<String>
)
```

### DataStore Integration
- Onboarding progress saved automatically
- Persists across app restarts
- Cleared on logout

---

## Navigation Flow

```
WelcomeScreen (Step 0)
    │
    ↓ Get Started / Skip
SecurityExplanationScreen (Step 1)
    │
    ↓ Next
ConnectServerScreen (Step 2)
    │
    ↓ Connected
PermissionsScreen (Step 3)
    │
    ↓ Continue
CompletionScreen (Step 4)
    │
    ↓ Continue
KeyBackupSetupScreen (Step 5)   ← NEW
    │
    ↓ Backup Complete
HomeScreen

(Legacy migration path):
SplashScreen
    │
    ↓ Legacy session detected
MigrationScreen                 ← NEW
    │
    ↓ Migration Complete
HomeScreen
```

---

## Related Documentation

- [Authentication](authentication.md) - Login/Registration
- [Encryption](encryption.md) - Security implementation
- [Settings](settings.md) - App configuration
