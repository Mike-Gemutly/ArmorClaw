# ArmorChat Project Review

> **Last Updated:** 2026-04-18
> **Version:** 1.0.0 (versionCode 10000)
> **Build Status:** ✅ ALL MODULES COMPILE
> **Deployment:** ✅ SUCCESS (Samsung SM-N986U — Android 13, SDK 33)
> **Architecture:** Kotlin Multiplatform (KMP) + Jetpack Compose
> **Test Coverage:** ✅ 368 tests (290 unit / 58 instrumented / 7 Maestro / 10 Appium / 2 auto-skip / 1 blocked)
> **Navigation:** 59 routes across 77 screens
> **Matrix Migration:** ✅ COMPLETE (See MATRIX_MIGRATION.md)
> **Unified Theme:** ✅ armorclaw-ui Module (See Section 3.3)
> **Modules:** shared, androidApp, armorclaw-ui

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Module Structure](#3-module-structure)
4. [Domain Layer](#4-domain-layer)
5. [Platform Layer](#5-platform-layer)
6. [Data Layer](#6-data-layer)
7. [Presentation Layer](#7-presentation-layer)
8. [Navigation System](#8-navigation-system)
9. [Security Architecture](#9-security-architecture)
10. [Communication Architecture](#10-communication-architecture)
11. [Control Plane & Agent System](#11-control-plane--agent-system)
12. [Feature Reference](#12-feature-reference)
13. [Build & Configuration](#13-build--configuration)
14. [Development Guidelines](#14-development-guidelines)
15. [ArmorChat ↔ ArmorClaw Communication](#15-armorchat--armorclaw-communication)
16. [Quick Reference](#16-quick-reference)
17. [Governor Strategy](#17-governor-strategy)
18. [Deployment Status & Issues Fixed](#18-deployment-status--issues-fixed)
19. [ArmorChat ↔ ArmorClaw Compatibility](#19-armorchat--armorclaw-compatibility)
20. [Deployment Order](#20-deployment-order)
21. [Server Discovery Implementation](#21-server-discovery-implementation)
22. [Signed Configuration System Review](#22-signed-configuration-system-review)
23. [OpenClaw Suggestions](#23-openclaw-suggestions)
24. [Onboarding Flow](#24-onboarding-flow)
25. [ArmorChat: Signed Config Integration](#25-armorchat--signed-config-integration)

---

## 1. Executive Summary

ArmorChat is a **secure end-to-end encrypted chat application** built on the Matrix protocol. The project uses **Kotlin Multiplatform (KMP)** for code sharing between platforms, with the Android app as the primary target using **Jetpack Compose** for the UI.

### Key Characteristics

| Aspect | Description |
|--------|-------------|
| **Language** | Kotlin 1.9.20 |
| **Architecture** | Clean Architecture (Domain → Data → Presentation) |
| **UI Framework** | Jetpack Compose 1.5.0, Material 3 |
| **DI Framework** | Koin 3.5.0 |
| **Database** | SQLDelight 2.0.0 with SQLCipher |
| **Network** | Ktor 2.3.5 (WebSocket, HTTP) |
| **Async** | Kotlin Coroutines, Flow |
| **Messaging Protocol** | Matrix (via `MatrixClientImpl` — Kotlin) |
| **Admin Backend** | ArmorClaw Bridge (Go) via JSON-RPC 2.0 |

### Project Status

| Component | Status | Notes |
|-----------|--------|-------|
| Shared Module (main) | ✅ Compiles | All compilation errors fixed |
| Shared Module (tests) | ✅ Compiles | Tests updated for Matrix SDK |
| AndroidApp Module | ✅ Compiles | 59 navigation routes, 77 screens |
| armorclaw-ui Module | ✅ Compiles | Unified theme module (teal/navy branding) |
| Bridge Client | ✅ Compiles | JSON-RPC client in shared module (no separate Go backend) |
| Matrix Integration | ✅ Complete | `MatrixClientImpl` + `MatrixSyncManager` |
| Unified Branding | ✅ Complete | `ArmorColors` / `ArmorClawTheme` in armorclaw-ui |
| Documentation | ✅ Complete | Comprehensive docs in `doc/` |
| **Device Deployment** | ✅ **Success** | Deployed to SM-N986U (Android 13, SDK 33) |
| **Test Suite** | ✅ **368 tests** | 290 unit / 58 instrumented / 7 Maestro / 10 Appium |

### Historical Bug Fixes

_Bugs #1–#7 fixed 2026-02-21 (connection hardcoding, well-known discovery, URLDecoder, session expiry, sync injection, encryption docs, URL handling)._

_Bugs #8–#12 fixed 2026-02-24 (RPC method names, WebSocket failure handling, enum casing, doc mismatches)._

_All 12 bugs resolved. See git history for details._

### Self-Hosted VPS Support

The app now supports connecting to any Matrix homeserver:

```kotlin
// Runtime configuration
BridgeConfig.setRuntimeConfig(
    BridgeConfig.createCustom(
        bridgeUrl = "https://your-vps.com:8080",
        homeserverUrl = "https://your-vps.com:8008",
        serverName = "My VPS"
    )
)
```

Or via Well-Known Discovery:
```json
// https://your-vps.com/.well-known/matrix/client
{
  "m.homeserver": { "base_url": "https://matrix.your-vps.com" },
  "com.armorclaw": {
    "bridge_url": "https://bridge.your-vps.com",
    "server_name": "My VPS"
  }
}
```

### Migration Status (Matrix Protocol)

ArmorChat has completed the migration from custom JSON-RPC to **proper Matrix Protocol**:

| Aspect | Status | Implementation |
|--------|--------|----------------|
| Messaging | ✅ Matrix SDK | `MatrixClient` interface (Kotlin impl via `MatrixClientImpl`) |
| Real-time Events | ✅ Matrix /sync | `MatrixSyncManager` long-poll |
| E2E Encryption | ✅ Client Keys | ECDH + AES-256-GCM via `VaultCryptoManager` |
| Admin Operations | ✅ Bridge RPC | `BridgeAdminClient` (31 methods) |
| Control Plane | ✅ Matrix Events | `ControlPlaneStore` processor |
| Session Storage | ✅ Encrypted | AES256-GCM + Android Keystore |
| Session Expiration | ✅ Proper | `MatrixSession.expiresAt` handling |

---

## 2. Architecture Overview

ArmorChat follows **Clean Architecture** with clear separation of concerns across multiple layers:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     PRESENTATION LAYER (Compose UI)                      │
│                                                                          │
│   Screens │ Components │ ViewModels │ Navigation (59 routes)            │
│   Location: androidApp/src/main/kotlin/com/armorclaw/app/               │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           DOMAIN LAYER                                   │
│                                                                          │
│   Models │ Repository Interfaces │ Use Cases                            │
│   Location: shared/src/commonMain/kotlin/domain/                        │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            DATA LAYER                                    │
│                                                                          │
│   Repository Implementations │ DAOs │ Stores │ Offline Queue            │
│   Location: shared/src/commonMain/kotlin/data/                          │
│            androidApp/src/main/kotlin/com/armorclaw/app/data/           │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       PLATFORM/SDK LAYER                                 │
│                                                                          │
│   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐     │
│   │  MatrixClient   │    │ BridgeRpcClient │    │  Platform SVCS  │     │
│   │  (Messaging)    │    │ (Admin Ops)     │    │  (Biometric,    │     │
│   │                 │    │                 │    │   Clipboard,    │     │
│   │  Kotlin impl    │    │  JSON-RPC 2.0   │    │   Network)      │     │
│   │  (MatrixClient  │    │  /api endpoint  │    │  expect/actual  │     │
│   │   Impl)         │    │  (NOT WebSocket)│    │                 │     │
│   └────────┬────────┘    └────────┬────────┘    └─────────────────┘     │
│            │                      │                                      │
│   Location: shared/src/commonMain/kotlin/platform/                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         EXTERNAL SERVICES                                │
│                                                                          │
│   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐     │
│   │ Matrix Server   │    │ ArmorClaw Server│    │ Firebase (FCM)  │     │
│   │ (Conduit)       │    │ (Bridge)        │    │ Push Notifs     │     │
│   │ :8008 (HTTPS)   │    │ JSON-RPC 2.0    │    │                 │     │
│   └─────────────────┘    └─────────────────┘    └─────────────────┘     │
└─────────────────────────────────────────────────────────────────────────┘
```

### Architecture Patterns

| Pattern | Implementation | Location |
|---------|---------------|----------|
| **Clean Architecture** | Domain → Data → Presentation layers | All modules |
| **MVVM** | ViewModels expose StateFlow, UI via collectAsState() | `viewmodels/` |
| **Repository Pattern** | Interfaces in shared/domain, impl in androidApp/data | `repository/` |
| **Expect/Actual** | Platform services declared in shared, impl in androidApp | `platform/` |
| **Atomic Design** | UI components: atom → molecule → organism | `shared/ui/components/` |
| **Use Case Pattern** | Single responsibility use case classes | `domain/usecase/` |

---

## 3. Module Structure

### Root Directory

```
ArmorChat/
├── shared/                    # Kotlin Multiplatform shared module
├── androidApp/                # Android application
├── armorclaw-ui/              # Shared UI theme module (unified branding)
├── doc/                       # Documentation (ARCHITECTURE.md, SECURITY.md, etc.)
├── docs/                      # Additional documentation
├── styling/                   # Branding assets (SVG, BrandingKit.docx)
├── gradle/                    # Gradle wrapper
├── build.gradle.kts           # Root build configuration
├── settings.gradle.kts        # Project settings
├── gradle.properties          # Gradle properties
├── gradlew / gradlew.bat      # Gradle wrapper scripts
├── CLAUDE.md                  # AI coding guidelines
├── ArmorChat.md               # This file (project review)
├── REVIEW.md                  # Alternate project review
├── BUILD_STATUS.md            # Current build status
```

### Shared Module (`shared/`)

The shared module contains platform-agnostic business logic:

```
shared/src/
├── commonMain/kotlin/
│   ├── domain/                # Domain Layer
│   │   ├── model/            # Domain models (20 files)
│   │   │   ├── ActivityEvent.kt        # Activity tracking
│   │   │   ├── AgentStatusEvent.kt     # Agent status
│   │   │   ├── AgentStatusHistory.kt   # Agent status history
│   │   │   ├── AppResult.kt            # Result wrapper
│   │   │   ├── ArmorClawErrorCode.kt   # Error codes (83 codes)
│   │   │   ├── AttentionItem.kt        # Attention items
│   │   │   ├── BrowserEvents.kt        # Browser extension events
│   │   │   ├── Call.kt                 # Call state models
│   │   │   ├── KeystoreStatus.kt       # Keystore status models
│   │   │   ├── KeystoreUnseal.kt       # Unseal request/response
│   │   │   ├── Message.kt              # Basic message model
│   │   │   ├── Notification.kt         # Notification model
│   │   │   ├── OperationContext.kt      # Operation context
│   │   │   ├── PiiAccessRequest.kt      # PII access requests
│   │   │   ├── Room.kt                 # Room model
│   │   │   ├── SyncState.kt            # Sync state enum
│   │   │   ├── SystemAlert.kt           # System alert types
│   │   │   ├── Trust.kt                # Trust level models
│   │   │   ├── UnifiedMessage.kt       # Unified message model
│   │   │   └── User.kt                 # User model
│   │   ├── repository/       # Repository interfaces (11 files)
│   │   │   ├── AgentRepository.kt
│   │   │   ├── AuthRepository.kt
│   │   │   ├── CallRepository.kt
│   │   │   ├── MessageRepository.kt
│   │   │   ├── NotificationRepository.kt
│   │   │   ├── RoomRepository.kt
│   │   │   ├── SyncRepository.kt
│   │   │   ├── ThreadRepository.kt
│   │   │   ├── UserRepository.kt
│   │   │   ├── VerificationRepository.kt
│   │   │   └── WorkflowRepository.kt
│   │   └── usecase/          # Use case classes (7 files)
│   │       ├── CheckFeatureUseCase.kt
│   │       ├── GetRoomsUseCase.kt
│   │       ├── LoadMessagesUseCase.kt
│   │       ├── LoginUseCase.kt
│   │       ├── LogoutUseCase.kt
│   │       ├── SendMessageUseCase.kt
│   │       └── SyncWhenOnlineUseCase.kt
│   │
│   ├── platform/             # Platform Layer
│   │   ├── biometric/        # Biometric auth interface
│   │   ├── bridge/           # Bridge client (13 files)
│   │   │   ├── BridgeAdminClient.kt      # Admin interface (31 methods)
│   │   │   ├── BridgeAdminClientImpl.kt  # Admin implementation
│   │   │   ├── BridgeClientFactory.kt    # Factory (expect/actual)
│   │   │   ├── BridgeEvent.kt            # Event types (25 events)
│   │   │   ├── BridgeRepository.kt       # Repository facade
│   │   │   ├── BridgeRpcClient.kt        # RPC interface (74 methods)
│   │   │   ├── BridgeRpcClientImpl.kt    # RPC implementation
│   │   │   ├── BridgeWebSocketClient.kt  # WebSocket interface
│   │   │   ├── BridgeWebSocketClientImpl.kt
│   │   │   ├── DevSettings.kt            # Developer settings
│   │   │   ├── InviteService.kt          # Room invite handling
│   │   │   ├── RpcModels.kt              # RPC data models
│   │   │   └── SetupService.kt           # Setup flow service
│   │   ├── clipboard/        # Secure clipboard interface
│   │   ├── encryption/       # Encryption interfaces
│   │   ├── error/            # Error handling
│   │   ├── logging/          # Logging (AppLogger, LoggerDelegate)
│   │   ├── matrix/           # Matrix SDK integration
│   │   │   ├── event/        # Matrix event types
│   │   │   ├── MatrixClient.kt          # Matrix client interface (53 methods)
│   │   │   ├── MatrixClientFactory.kt   # Factory (expect/actual)
│   │   │   ├── MatrixSyncManager.kt     # Sync long-poll manager
│   │   │   └── MatrixSessionStorage.kt  # Session persistence
│   │   ├── network/          # Network monitoring
│   │   ├── notification/     # Notification interface
│   │   ├── voice/            # Voice input/recording
│   │   ├── Analytics.kt      # Analytics interface
│   │   └── CrashReporting.kt # Crash reporting interface
│   │
│   ├── data/                 # Data Layer
│   │   ├── dao/              # Data access objects
│   │   ├── repository/       # Repository implementations
│   │   └── store/            # State stores
│   │       ├── ControlPlaneStore.kt     # Control plane event processor
│   │       └── RealTimeEventStore.kt    # Event distribution to UI
│   │
│   └── ui/                   # Shared UI
│       ├── theme/            # Design system (colors, typography, shapes)
│       ├── components/       # UI components
│       │   ├── atom/        # Atomic components (Button, Input, etc.)
│       │   └── molecule/    # Molecular components (MessageBubble, etc.)
│       └── base/             # Base classes
│
├── androidMain/kotlin/       # Android implementations
│   └── com/armorclaw/shared/
│       └── platform/         # Actual implementations
│
├── androidUnitTest/kotlin/   # Android unit tests
└── commonTest/kotlin/        # Common tests
```

### Android App Module (`androidApp/`)

The Android app module contains platform-specific code:

```
androidApp/src/main/kotlin/com/armorclaw/app/
├── accessibility/            # Accessibility features
│   ├── AccessibilityConfig.kt
│   └── AccessibilityExtensions.kt
│
├── components/               # Android-specific UI components
│   └── DeepLinkConfirmationDialog.kt
│
├── ui/                       # Shared UI composables (v3)
│   └── components/          # Reusable workflow UI components
│       ├── WorkflowTimeline.kt        # Scrollable vertical timeline (473 lines)
│       ├── BlockerResponseDialog.kt   # HITL blocker resolution modal (461 lines)
│       └── GovernanceBanner.kt        # Workflow status banner (317 lines)
│
├── data/                     # Data implementations
│   ├── database/            # SQLCipher database
│   ├── offline/             # Offline sync
│   └── persistence/         # DataStore persistence
│
├── di/                       # Dependency Injection (Koin modules)
│   └── AppModules.kt
│
├── navigation/               # Navigation
│   ├── AppNavigation.kt     # All 59 routes defined here
│   └── DeepLinkHandler.kt   # Deep link parsing and routing
│
├── notifications/            # Push notification handling
│
├── performance/              # Performance monitoring
│   ├── MemoryMonitor.kt
│   └── PerformanceProfiler.kt
│
├── platform/                 # Platform implementations
│   ├── Analytics.kt
│   ├── BiometricAuthImpl.kt
│   ├── CertificatePinner.kt
│   └── CrashReporter.kt
│
├── release/                  # Release configuration
│
├── screens/                  # Compose screens by feature
│   ├── call/                # Call UI components
│   │   ├── AudioVisualizer.kt
│   │   └── CallControls.kt
│   ├── chat/                # Chat screens & components
│   │   ├── ThreadViewScreen.kt
│   │   └── components/      # MessageBubble, EncryptionStatus, etc. (11 files)
│   ├── home/                # Home screen components
│   │   └── components/GovernanceBanner.kt
│   ├── keystore/            # Keystore screens
│   │   └── UnsealScreen.kt
│   ├── media/               # Media viewers
│   │   ├── ImageViewerScreen.kt
│   │   └── FilePreviewScreen.kt
│   ├── onboarding/          # Onboarding flow (7 screens)
│   │   ├── ExpressSetupCompleteScreen.kt
│   │   ├── KeyBackupSetupScreen.kt
│   │   ├── MigrationScreen.kt
│   │   ├── OnboardingConfigScreen.kt
│   │   ├── OnboardingInviteScreen.kt
│   │   ├── OnboardingSetupScreen.kt
│   │   └── QRScanScreen.kt
│   ├── profile/             # Profile screens (6 screens)
│   │   ├── ChangePasswordScreen.kt
│   │   ├── ChangePhoneNumberScreen.kt
│   │   ├── DeleteAccountScreen.kt
│   │   ├── EditBioScreen.kt
│   │   └── SharedRoomsScreen.kt
│   ├── room/                # Room management
│   │   ├── RoomManagementScreen.kt
│   │   └── RoomSettingsScreen.kt
│   ├── settings/            # Settings screens (17 screens)
│   │   ├── AboutScreen.kt
│   │   ├── AddDeviceScreen.kt
│   │   ├── AppearanceSettingsScreen.kt
│   │   ├── DataSafetyScreen.kt
│   │   ├── DevMenuScreen.kt
│   │   ├── DeviceListScreen.kt
│   │   ├── EmojiVerificationScreen.kt
│   │   ├── InviteManagementScreen.kt
│   │   ├── InviteScreen.kt
│   │   ├── MyDataScreen.kt
│   │   ├── NotificationSettingsScreen.kt
│   │   ├── OpenSourceLicensesScreen.kt
│   │   ├── PrivacyPolicyScreen.kt
│   │   ├── ReportBugScreen.kt
│   │   ├── SecuritySettingsScreen.kt
│   │   ├── ServerConnectionScreen.kt
│   │   ├── TermsOfServiceScreen.kt
│   │   └── components/      # DeviceListItem, TrustBadge
│   ├── vault/               # Vault screens
│   │   ├── VaultScreen.kt
│   │   └── AddSecretScreen.kt
│   └── v2/                  # V2 screens (route + screen pairs)
│       ├── auth/            # Login, Registration, ForgotPassword, KeyRecovery, ConnectServer
│       ├── call/            # ActiveCall, IncomingCall
│       ├── chat/            # ChatScreen
│       ├── home/            # HomeScreen
│       ├── onboarding/      # Welcome, Security, Permissions, Completion, Tutorial
│       ├── profile/         # Profile, UserProfile
│       ├── room/            # RoomDetails
│       ├── search/          # SearchScreen
│       ├── settings/        # SettingsScreen
│       ├── splash/          # SplashScreen
│       └── PreviewData.kt
│
├── security/                 # Security implementations
│   ├── KeystoreManager.kt
│   ├── SqlCipherProvider.kt
│   └── VaultRepository.kt
│
├── service/                  # Android services
│   └── FirebaseMessagingService.kt
│
├── util/                     # Utilities
│
├── viewmodels/               # Screen ViewModels (16 files)
│   ├── AccountDeletionViewModel.kt
│   ├── AppPreferences.kt    # App preferences (DataStore)
│   ├── ChangePasswordViewModel.kt
│   ├── ChangePhoneViewModel.kt
│   ├── ChatViewModel.kt     # Chat screen state
│   ├── EditBioViewModel.kt
│   ├── HomeViewModel.kt     # Home screen state
│   ├── InviteViewModel.kt   # Invite handling
│   ├── ProfileViewModel.kt  # Profile state
│   ├── SettingsViewModel.kt # Settings state
│   ├── SetupViewModel.kt    # Setup flow state
│   ├── SplashViewModel.kt   # Splash state
│   ├── SyncStatusViewModel.kt # Sync status
│   ├── UnsealViewModel.kt   # Keystore unseal
│   ├── UserProfileViewModel.kt
│   └── WelcomeViewModel.kt  # Welcome state
│
├── ArmorClawApplication.kt   # Application class (Koin init)
└── MainActivity.kt           # Main activity
```

### armorclaw-ui Module (`armorclaw-ui/`)

The armorclaw-ui module provides a **unified theme** for ArmorClaw applications (ArmorChat, ArmorTerminal, Component Catch browser extension):

```
armorclaw-ui/src/commonMain/kotlin/
├── theme/
│   ├── ArmorClawColor.kt       # Color palette (Teal #14F0C8, Navy #0A1428)
│   ├── ArmorClawTypography.kt  # Typography (Inter, JetBrains Mono)
│   ├── ArmorClawShapes.kt      # Shape definitions
│   ├── ArmorClawTheme.kt       # Theme wrapper composable
│   ├── GlowModifiers.kt        # Teal glow effect modifiers
│   └── StatusIcons.kt          # Status indicator icons
├── components/
│   ├── vault/
│   │   ├── VaultModels.kt      # Vault UI data models
│   │   ├── VaultPulseIndicator.kt  # Pulse animation
│   │   └── VaultKeyPanel.kt    # Sidebar key panel
│   ├── governor/
│   │   ├── GovernorModels.kt   # Governor data models
│   │   ├── CommandBlock.kt     # Action card
│   │   ├── CapabilityRibbon.kt # Capability display
│   │   ├── HITLAuthorization.kt # Hold-to-approve
│   │   └── GovernorComponents.kt # Barrel export
│   └── audit/
│       ├── AuditModels.kt      # Audit data models
│       ├── ArmorTerminal.kt    # Real-time activity log
│       └── RevocationControls.kt # One-click revocation
└── androidMain/                # Android manifest
```

**Brand Colors:**
| Color | Hex | Usage |
|-------|-----|-------|
| Teal (Primary) | `#14F0C8` | Primary actions, accents |
| Teal Glow | `#67F5D8` | Hover states, glows |
| Navy (Background) | `#0A1428` | Dark background |
| Precision Blue | `#0EA5E9` | Secondary actions |
| Success Green | `#22C55E` | Success states |
| Warning Amber | `#F59E0B` | Warning states |

**Design Principles:**
- Dark mode only (default experience)
- High contrast (≥4.5:1 ratio)
- Sparse mascot usage (splash, empty states, about screen)
- Unified typography across platforms

---

## 4. Domain Layer

The domain layer contains business logic that is platform-agnostic.

### Domain Models

#### UnifiedMessage

The `UnifiedMessage` is the primary message model that supports both regular Matrix messages and agent/system messages:

```kotlin
sealed class UnifiedMessage {
    abstract val id: String
    abstract val roomId: String
    abstract val timestamp: Instant
    abstract val sender: MessageSender

    data class Regular(...) : UnifiedMessage()   // User messages
    data class Agent(...) : UnifiedMessage()     // AI assistant messages
    data class System(...) : UnifiedMessage()    // System notifications
    data class Command(...) : UnifiedMessage()   // User commands to agent
}
```

Message senders are also typed:

```kotlin
sealed class MessageSender {
    data class UserSender(...) : MessageSender()    // Regular users
    data class AgentSender(...) : MessageSender()   // AI agents
    data class SystemSender(...) : MessageSender()  // System messages
}
```

#### Other Domain Models

| Model | File | Purpose |
|-------|------|---------|
| `User` | User.kt | User profile information |
| `Room` | Room.kt | Chat room with metadata |
| `Message` | Message.kt | Basic message structure |
| `Call` | Call.kt | Call session and state |
| `Trust` | Trust.kt | Trust level for users/devices |
| `SyncState` | SyncState.kt | Synchronization state |
| `SystemAlert` | SystemAlert.kt | System alert types |
| `AppResult` | AppResult.kt | Result wrapper for operations |
| `ArmorClawErrorCode` | ArmorClawErrorCode.kt | 83 error codes |

### Repository Interfaces

All repository interfaces are defined in `shared/domain/repository/`:

| Repository | Purpose | Key Methods |
|------------|---------|-------------|
| `AuthRepository` | Authentication | `login()`, `logout()`, `isLoggedIn()` |
| `MessageRepository` | Messages | `getMessages()`, `sendMessage()`, `sendReaction()` |
| `RoomRepository` | Rooms | `getRooms()`, `createRoom()`, `joinRoom()` |
| `UserRepository` | Users | `getUser()`, `updateProfile()` |
| `CallRepository` | Calls | `startCall()`, `endCall()`, `toggleMute()` |
| `ThreadRepository` | Threads | `getThread()`, `sendReply()` |
| `VerificationRepository` | Verification | `requestVerification()`, `confirmVerification()` |
| `WorkflowRepository` | Workflows | `startWorkflow()`, `updateStep()` |
| `AgentRepository` | Agents | `taskStarted()`, `taskCompleted()` |
| `SyncRepository` | Sync | `sync()`, `getSyncState()` |
| `NotificationRepository` | Notifications | `registerToken()`, `updateSettings()` |

---

## 5. Platform Layer

The platform layer provides abstractions for platform-specific functionality using the **expect/actual** pattern.

### MatrixClient Interface

The `MatrixClient` is the primary interface for all Matrix operations:

```kotlin
interface MatrixClient {
    // Connection State
    val syncState: StateFlow<SyncState>
    val isLoggedIn: StateFlow<Boolean>
    val currentUser: StateFlow<User?>
    val connectionState: StateFlow<ConnectionState>

    // Authentication
    suspend fun login(homeserver: String, username: String, password: String, deviceId: String?): Result<MatrixSession>
    suspend fun logout(): Result<Unit>
    suspend fun restoreSession(session: MatrixSession): Result<Unit>

    // Sync
    fun startSync()
    fun stopSync()
    suspend fun syncOnce(): Result<Unit>

    // Rooms
    val rooms: StateFlow<List<Room>>
    suspend fun createRoom(...): Result<Room>
    suspend fun joinRoom(roomIdOrAlias: String): Result<Room>
    suspend fun leaveRoom(roomId: String): Result<Unit>

    // Messages
    suspend fun getMessages(roomId: String, limit: Int, fromToken: String?): Result<MessageBatch>
    suspend fun sendTextMessage(roomId: String, text: String): Result<String>
    suspend fun sendReaction(roomId: String, eventId: String, key: String): Result<String>

    // Events
    fun observeEvents(): Flow<MatrixEvent>
    fun observeRoomEvents(roomId: String): Flow<MatrixEvent>
    fun observeArmorClawEvents(roomId: String?): Flow<MatrixEvent>

    // Presence, Typing, Read Receipts, Encryption, etc.
    // ... (53 methods total)
}
```

### BridgeRpcClient Interface

The `BridgeRpcClient` handles all RPC operations via JSON-RPC 2.0 (74 methods):

```kotlin
interface BridgeRpcClient {
    // Connection
    fun isConnected(): Boolean
    fun getSessionId(): String?
    suspend fun setAdminToken(token: String?)

    // Bridge Lifecycle (4)
    suspend fun startBridge(userId, deviceId, context): RpcResult<BridgeStartResponse>
    suspend fun getBridgeStatus(context): RpcResult<BridgeStatusResponse>
    suspend fun stopBridge(sessionId, context): RpcResult<BridgeStopResponse>
    suspend fun healthCheck(context): RpcResult<HealthCheckResponse>

    // Matrix Operations (9) — ⚠️ Deprecated, use MatrixClient instead
    suspend fun matrixLogin(homeserver, username, password, deviceId, context): RpcResult<MatrixLoginResponse>
    suspend fun matrixSync(context): RpcResult<MatrixSyncResponse>
    suspend fun matrixSend(roomId, eventType, content, txnId, context): RpcResult<MatrixSendResponse>
    suspend fun matrixRefreshToken(...): RpcResult<MatrixRefreshResponse>
    suspend fun matrixCreateRoom(...): RpcResult<Room>
    suspend fun matrixJoinRoom(roomIdOrAlias, context): RpcResult<Room>
    suspend fun matrixLeaveRoom(roomId, context): RpcResult<Unit>
    suspend fun matrixInviteUser(roomId, userId, context): RpcResult<Unit>
    suspend fun matrixSendTyping(roomId, typing, timeout, context): RpcResult<Unit>
    suspend fun matrixSendReadReceipt(roomId, eventId, context): RpcResult<Unit>

    // Provisioning (5)
    suspend fun provisioningStart(...): RpcResult<ProvisioningStartResponse>
    suspend fun provisioningStatus(...): RpcResult<ProvisioningStatusResponse>
    suspend fun provisioningClaim(claimToken, context): RpcResult<ProvisioningClaimResponse>
    suspend fun provisioningRotate(secretType, context): RpcResult<ProvisioningRotateResponse>
    suspend fun provisioningCancel(targetId, context): RpcResult<Unit>

    // WebRTC Signaling (7)
    suspend fun webrtcOffer(callId, sdpOffer, context): RpcResult<WebrtcAnswerResponse>
    suspend fun webrtcAnswer(callId, sdpAnswer, context): RpcResult<Unit>
    suspend fun webrtcIceCandidate(callId, candidate, context): RpcResult<Unit>
    suspend fun webrtcHangup(callId, context): RpcResult<Unit>
    suspend fun webrtcStart(...): RpcResult<WebrtcStartResponse>
    suspend fun webrtcSendIceCandidate(...): RpcResult<Unit>
    suspend fun webrtcEnd(...): RpcResult<Unit>
    suspend fun webrtcList(...): RpcResult<List<WebrtcSession>>

    // Recovery (6)
    suspend fun recoveryGeneratePhrase(context): RpcResult<RecoveryPhraseResponse>
    suspend fun recoveryStorePhrase(phrase, context): RpcResult<Unit>
    suspend fun recoveryVerify(phrase, context): RpcResult<VerificationResponse>
    suspend fun recoveryStatus(context): RpcResult<RecoveryStatusResponse>
    suspend fun recoveryComplete(phrase, context): RpcResult<Unit>
    suspend fun recoveryIsDeviceValid(deviceId, context): RpcResult<DeviceValidResponse>

    // Platform Integration (5)
    suspend fun platformConnect(platformType, config, context): RpcResult<PlatformConnectResponse>
    suspend fun platformDisconnect(platformId, context): RpcResult<Unit>
    suspend fun platformList(context): RpcResult<List<PlatformInfo>>
    suspend fun platformStatus(platformId, context): RpcResult<PlatformStatusResponse>
    suspend fun platformTest(platformId, context): RpcResult<PlatformTestResponse>

    // Push Notifications (3)
    suspend fun pushRegister(pushToken, platform, deviceId, context): RpcResult<Unit>
    suspend fun pushUnregister(deviceId, context): RpcResult<Unit>
    suspend fun pushUpdateSettings(settings, context): RpcResult<Unit>

    // License & Compliance (5)
    suspend fun licenseStatus(context): RpcResult<LicenseStatusResponse>
    suspend fun licenseFeatures(context): RpcResult<LicenseFeaturesResponse>
    suspend fun licenseCheckFeature(feature, context): RpcResult<FeatureCheckResponse>
    suspend fun complianceStatus(context): RpcResult<ComplianceStatusResponse>
    suspend fun platformLimits(context): RpcResult<PlatformLimitsResponse>

    // Error Management (2)
    suspend fun getErrors(context): RpcResult<List<ErrorInfo>>
    suspend fun resolveError(errorId, context): RpcResult<Unit>

    // Agent Management (5)
    suspend fun agentList(context): RpcResult<AgentListResponse>
    suspend fun agentStatus(agentId, context): RpcResult<AgentStatusResponse>
    suspend fun agentStop(agentId, context): RpcResult<Unit>
    suspend fun agentGetStatus(agentId, context): RpcResult<AgentStatusResponse>
    suspend fun agentStatusHistory(agentId, context): RpcResult<AgentStatusHistoryResponse>

    // Workflow (3)
    suspend fun workflowTemplates(context): RpcResult<WorkflowTemplatesResponse>
    suspend fun workflowStart(templateId, params, context): RpcResult<WorkflowStartResponse>
    suspend fun workflowStatus(workflowId, context): RpcResult<WorkflowStatusResponse>

    // HITL — Human-in-the-Loop (3)
    suspend fun hitlPending(context): RpcResult<HitlPendingResponse>
    suspend fun hitlApprove(gateId, context): RpcResult<Unit>
    suspend fun hitlReject(gateId, reason, context): RpcResult<Unit>

    // Budget (1)
    suspend fun budgetStatus(context): RpcResult<BudgetStatusResponse>

    // Browser Commands (6)
    suspend fun browserEnqueue(command, priority, context): RpcResult<BrowserEnqueueResponse>
    suspend fun browserGetJob(jobId, context): RpcResult<BrowserJobResponse>
    suspend fun browserCancelJob(jobId, context): RpcResult<BrowserCancelResponse>
    suspend fun browserRetryJob(jobId, context): RpcResult<BrowserRetryResponse>
    suspend fun browserListJobs(status, context): RpcResult<BrowserJobListResponse>
    suspend fun browserQueueStats(context): RpcResult<BrowserQueueStatsResponse>

    // Keystore (4)
    suspend fun keystoreSealed(context): RpcResult<KeystoreStatusResponse>
    suspend fun keystoreUnsealChallenge(request, context): RpcResult<UnsealChallenge>
    suspend fun keystoreUnsealRespond(result, context): RpcResult<UnsealResult>
    suspend fun keystoreExtendSession(sessionToken, context): RpcResult<SessionExtensionResult>

    // Generic RPC call
    suspend fun <T> call(method, params, context): RpcResult<T>
}
```

### Platform Services (Expect/Actual)

| Service | Expect Location | Actual Location | Purpose |
|---------|-----------------|-----------------|---------|
| `BiometricAuth` | `shared/.../platform/biometric/` | `shared/src/androidMain/.../biometric/` | Fingerprint/FaceID |
| `SecureClipboard` | `shared/.../platform/clipboard/` | `shared/src/androidMain/.../clipboard/` | Encrypted clipboard |
| `NotificationManager` | `shared/.../platform/notification/` | `shared/src/androidMain/.../notification/` | Push notifications |
| `NetworkMonitor` | `shared/.../platform/network/` | `shared/src/androidMain/.../network/` | Connectivity |
| `VoiceCallManager` | `shared/.../platform/voice/` | `shared/src/androidMain/.../voice/` | Voice calls |
| `MatrixClientFactory` | `shared/.../platform/matrix/` | `shared/src/androidMain/.../matrix/` | Client creation |
| `BridgeClientFactory` | `shared/.../platform/bridge/` | `shared/src/androidMain/.../bridge/` | RPC client creation |

---

## 6. Data Layer

The data layer handles persistence and data access.

### ControlPlaneStore

The `ControlPlaneStore` processes ArmorClaw-specific Matrix events:

```kotlin
class ControlPlaneStore(
    private val matrixClient: MatrixClient,
    private val workflowRepository: WorkflowRepository,
    private val agentRepository: AgentRepository
) {
    // State Flows
    val activeWorkflows: StateFlow<List<WorkflowState>>
    val agentTasks: StateFlow<List<AgentTaskState>>
    val thinkingAgents: StateFlow<Map<String, AgentThinkingState>>
    val budgetWarnings: StateFlow<List<BudgetWarningState>>
    val bridgeStatus: StateFlow<Map<String, BridgeStatusState>>

    // Event Processing
    fun subscribeToRoom(roomId: String)
    private suspend fun processEvent(event: MatrixEvent)
}
```

### Database

The app uses **SQLDelight** with **SQLCipher** for encrypted storage:

```
shared/src/commonMain/sqldelight/
└── com/armorclaw/shared/
    └── ArmorClawDatabase.sq   # Schema definitions
```

Key tables:
- Messages (with sync state, expiration)
- Rooms (with metadata, statistics)
- SyncQueue (pending operations)
- Users (profiles, presence)

### Offline Sync

The offline sync system includes:

| Component | Purpose |
|-----------|---------|
| `OfflineQueue` | Queue pending operations with priorities |
| `SyncEngine` | Execute operations, detect conflicts |
| `ConflictResolver` | Resolve sync conflicts |
| `BackgroundSyncWorker` | WorkManager for periodic sync |

---

## 7. Presentation Layer

The presentation layer uses **Jetpack Compose** with **MVVM** pattern.

### ViewModels

Key ViewModels and their responsibilities:

| ViewModel | Screen | Key State |
|-----------|--------|-----------|
| `ChatViewModel` | Chat | Messages, input, typing, call state |
| `HomeViewModel` | Home | Room list, filters, sync status |
| `ProfileViewModel` | Profile | User data, edit mode |
| `UserProfileViewModel` | User Profile | Other user's profile data |
| `SettingsViewModel` | Settings | Settings state, logout |
| `SetupViewModel` | Connect | Server connection, credentials |
| `SplashViewModel` | Splash | Auth state, navigation decision |
| `WelcomeViewModel` | Welcome | Onboarding navigation |
| `InviteViewModel` | Invite | Room invite handling |
| `SyncStatusViewModel` | (Various) | Sync progress, errors |
| `UnsealViewModel` | Keystore | Unseal challenge/response |
| `ChangePasswordViewModel` | Change Password | Password change flow |
| `ChangePhoneViewModel` | Change Phone | Phone change flow |
| `EditBioViewModel` | Edit Bio | Bio editing |
| `AccountDeletionViewModel` | Delete Account | Account deletion flow |
| `AppPreferences` | (Global) | DataStore preferences |

### State Management Pattern

```kotlin
// ViewModel exposes StateFlow
class ChatViewModel(...) : ViewModel() {
    private val _uiState = MutableStateFlow(ChatUiState())
    val uiState: StateFlow<ChatState> = _uiState.asStateFlow()

    private val _events = Channel<ChatEvent>()
    val events: Flow<ChatEvent> = _events.receiveAsFlow()

    fun onAction(action: ChatAction) { ... }
}

// Screen collects state
@Composable
fun ChatScreen(viewModel: ChatViewModel) {
    val state by viewModel.uiState.collectAsState()

    LaunchedEffect(Unit) {
        viewModel.events.collect { event ->
            when (event) { ... }
        }
    }

    // UI based on state
}
```

### UI Component Hierarchy

Components follow **Atomic Design**:

```
Screen
└── Organism Components (e.g., MessageList, RoomList)
    └── Molecule Components (e.g., MessageBubble, RoomItem)
        └── Atom Components (e.g., Button, InputField, Avatar)
```

### Workflow Composables (v3)

Three Jetpack Compose components handle agent workflow visualization and blocker resolution:

| Composable | Package | Purpose |
|-----------|---------|---------|
| `WorkflowTimeline` | `app.armorclaw.ui.components` | Scrollable vertical timeline showing agent step progress, icons, durations |
| `BlockerResponseDialog` | `app.armorclaw.ui.components` | Modal dialog for resolving HITL blockers with PII-safe input |
| `GovernanceBanner` | `app.armorclaw.ui.components` | Compact status banner (RUNNING/BLOCKED/COMPLETED/FAILED/CANCELLED) |

State flows from Matrix `/sync` events through the ViewModel/StateFlow pattern into these composables. They do not manage network connections directly.

---

## 8. Navigation System

### Route Definitions

All 59 routes are defined in `AppNavigation.kt`:

```kotlin
object AppNavigation {
    // Core
    const val SPLASH = "splash"
    const val HOME = "home"
    const val SEARCH = "search"

    // Onboarding
    const val WELCOME = "welcome"
    const val SECURITY = "security"
    const val CONNECT = "connect"
    const val PERMISSIONS = "permissions"
    const val COMPLETION = "complete"
    const val TUTORIAL = "tutorial"
    const val MIGRATION = "migration"
    const val KEY_BACKUP_SETUP = "key_backup_setup"
    const val EXPRESS_COMPLETE = "express_complete"
    const val QR_SCAN = "qr_scan"

    // Onboarding Extended
    const val ONBOARDING_CONFIG = "onboarding/config"
    const val ONBOARDING_SETUP = "onboarding/setup"
    const val ONBOARDING_INVITE = "onboarding/invite"

    // Auth
    const val LOGIN = "login"
    const val REGISTRATION = "registration"
    const val FORGOT_PASSWORD = "forgot_password"
    const val KEY_RECOVERY = "key_recovery"

    // Chat
    const val CHAT = "chat/{roomId}"
    const val THREAD = "thread/{roomId}/{rootMessageId}"

    // Profile
    const val PROFILE = "profile"
    const val CHANGE_PASSWORD = "change_password"
    const val CHANGE_PHONE = "change_phone"
    const val EDIT_BIO = "edit_bio"
    const val DELETE_ACCOUNT = "delete_account"
    const val USER_PROFILE = "user_profile/{userId}"
    const val SHARED_ROOMS = "shared_rooms/{userId}"

    // Settings
    const val SETTINGS = "settings"
    const val SECURITY_SETTINGS = "security_settings"
    const val NOTIFICATION_SETTINGS = "notification_settings"
    const val APPEARANCE = "appearance"
    const val PRIVACY_POLICY = "privacy_policy"
    const val MY_DATA = "my_data"
    const val DATA_SAFETY = "data_safety"
    const val ABOUT = "about"
    const val REPORT_BUG = "report_bug"
    const val TERMS_OF_SERVICE = "terms_of_service"
    const val LICENSES = "licenses"
    const val INVITE = "invite"
    const val SERVER_CONNECTION = "server_connection"
    const val DEV_MENU = "dev_menu"

    // Devices & Verification
    const val DEVICES = "devices"
    const val ADD_DEVICE = "add_device"
    const val EMOJI_VERIFICATION = "emoji_verification/{deviceId}"
    const val BRIDGE_VERIFICATION = "bridge_verification/{deviceId}"
    const val KEYSTORE = "keystore"

    // Room
    const val ROOM_MANAGEMENT = "room_management"
    const val ROOM_DETAILS = "room_details/{roomId}"
    const val ROOM_SETTINGS = "room_settings/{roomId}"

    // Calls
    const val ACTIVE_CALL = "active_call/{callId}"
    const val INCOMING_CALL = "incoming_call/{callId}/{callerId}/{callerName}/{callType}"

    // Media
    const val IMAGE_VIEWER = "image_viewer/{imageId}"
    const val FILE_PREVIEW = "file_preview/{fileId}"

    // Vault & Governor
    const val VAULT = "vault"
    const val ADD_SECRET = "add_secret"
    const val AGENT_MANAGEMENT = "agent_management"
    const val HITL_APPROVALS = "hitl_approvals"

    // Helper functions for parameterized routes
    fun createChatRoute(roomId: String): String = "chat/$roomId"
    fun createRoomDetailsRoute(roomId: String): String = "room_details/$roomId"
    fun createRoomSettingsRoute(roomId: String): String = "room_settings/$roomId"
    fun createUserProfileRoute(userId: String): String = "user_profile/$userId"
    fun createSharedRoomsRoute(userId: String): String = "shared_rooms/$userId"
    fun createCallRoute(callId: String): String = "active_call/$callId"
    fun createIncomingCallRoute(
        callId: String,
        callerId: String,
        callerName: String,
        callType: String = "voice"
    ): String = "incoming_call/$callId/$callerId/$callerName/$callType"
    fun createThreadRoute(roomId: String, rootMessageId: String): String = "thread/$roomId/$rootMessageId"
    fun createVerificationRoute(deviceId: String): String = "emoji_verification/$deviceId"
    fun createBridgeVerificationRoute(deviceId: String): String = "bridge_verification/$deviceId"
    fun createImageViewerRoute(imageId: String): String = "image_viewer/$imageId"
    fun createFilePreviewRoute(fileId: String): String = "file_preview/$fileId"
}
```

### Navigation Flows

#### Onboarding Flow
```
Splash → [Migration?] → [KeyBackup?] → Connect → Permissions → Completion → KeyBackup → Home
         │                  │              (QR-first onboarding)
         │                  └── Home
         └── Login (if session expired)
```

#### Authentication Flow
```
Connect → Login → [Forgot Password | Registration | Key Recovery] → Home
```

#### Chat Flow
```
Home → Chat → [Room Details → Room Settings]
           → [User Profile → DM Chat | Call | Shared Rooms]
           → [Thread]
           → [Call]
           → [Image Viewer | File Preview]
```

#### Settings Flow
```
Home → Settings → [Profile → Change Password | Change Phone | Edit Bio | Delete Account]
                → [Security Settings → Devices → Emoji Verification | Bridge Verification]
                → [Notification Settings]
                → [Appearance]
                → [Privacy Policy | My Data | Data Safety]
                → [About → Licenses | Terms | Report Bug]
                → [Server Connection]
                → [Invite]
                → [Dev Menu]
                → [Vault → Add Secret]
                → [Agent Management]
                → [HITL Approvals]
                → [Keystore → Unseal]
                → [Logout → Login]
```

---

## 9. Security Architecture

### Security Layers

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     SECURITY LAYERS                                      │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 1: Transport Security (TLS 1.3 + Certificate Pinning)            │
│  Layer 2: Message Encryption (AES-256-GCM via Matrix E2EE)              │
│  Layer 3: Database Encryption (SQLCipher 256-bit)                       │
│  Layer 4: Key Storage (AndroidKeyStore)                                 │
│  Layer 5: Biometric Authentication                                      │
│  Layer 6: Deep Link Validation                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Encryption Trust Model

> **Note:** The `EncryptionMode` enum is defined in `platform/encryption/EncryptionTypes.kt`.

| Model | Trust Assumption | Key Management | Status |
|-------|-----------------|----------------|--------|
| **CLIENT_SIDE** | Trust only self | Client manages keys via Matrix Rust SDK | ✅ Active (default) |
| **SERVER_SIDE** | Trust Bridge Server | Bridge handles E2EE via Megolm/Olm | ⚠️ Deprecated |
| **NONE** | No encryption | N/A | ⚠️ Dev only |

#### CLIENT_SIDE Mode (Active — Current Default)

**How it works:**
1. Client encrypts message directly using Matrix Rust SDK / vodozemac
2. Client sends encrypted message to Matrix homeserver
3. Only recipient devices can decrypt
4. No server can read message content

**Security Benefits:**
- ✅ True end-to-end encryption
- ✅ No server (including Bridge) can read messages
- ✅ Forward secrecy via Megolm ratchet
- ✅ Cross-signing verification
- ✅ Device verification workflow

**Room Encryption Status:**
```kotlin
sealed class RoomEncryptionStatus {
    object NotAvailable : RoomEncryptionStatus()
    data class Encrypted(val verified: Boolean) : RoomEncryptionStatus()
    data class EncryptedUnverified(val reason: String, val trustModel: String = "client-side") : RoomEncryptionStatus()
}

enum class DeviceVerificationStatus {
    VERIFIED, UNVERIFIED, VERIFYING
}
```

#### SERVER_SIDE Mode (Deprecated)

**How it worked:**
1. Client sends plaintext message to Bridge
2. Bridge encrypts message using Matrix E2EE (Megolm/Olm)
3. Bridge sends encrypted message to Matrix homeserver
4. Recipients' devices decrypt via their own Bridge servers

**Trust Implications:**
- ⚠️ Bridge server CAN read all message content
- ✅ Matrix homeserver CANNOT read messages (E2EE to Bridge)
- ✅ TLS protects client ↔ Bridge connection

**Historical Use Cases:**
- Self-hosted deployments where you control the Bridge
- Enterprise deployments with trusted Bridge operator
- Development and testing environments

### Session Management (NEW)

Sessions now properly handle expiration:

```kotlin
data class MatrixSession(
    val userId: String,
    val deviceId: String,
    val accessToken: String,
    val refreshToken: String? = null,
    val homeserver: String,
    val displayName: String? = null,
    val avatarUrl: String? = null,
    val expiresIn: Long? = null,          // Seconds until token expires
    val expiresAt: Long? = null,          // Absolute expiration timestamp
    val loginAt: Long? = null             // When session was created
) {
    fun isExpired(): Boolean
    fun isExpiringSoon(withinSeconds: Long = 300): Boolean
    fun remainingTimeSeconds(): Long?
}
```

**Session Lifecycle:**
1. On login: Calculate `expiresAt` from `expiresIn` or use 24-hour default
2. On app start: Check `isExpired()` before restoring session
3. On API call: Check `isExpiringSoon()` and proactively refresh
4. On expiration: Clear session, redirect to login

### Security Features

| Feature | Implementation | Status |
|---------|---------------|--------|
| Database Encryption | SQLCipher (256-bit) | ✅ |
| Key Storage | AndroidKeyStore | ✅ |
| TLS | TLS 1.3 minimum | ✅ |
| Certificate Pinning | SHA-256 pins | ✅ |
| Biometric Auth | AndroidX Biometric | ✅ |
| Secure Clipboard | Encrypted, auto-clear | ✅ |
| Deep Link Validation | URI validation, confirmation dialogs | ✅ |
| Push Security | No content in push payload | ✅ |

### OWASP Mobile Top 10 Compliance

| Risk | Status | Mitigation |
|------|--------|------------|
| M1: Improper Platform Usage | ✅ | Secure intent handling, validated deep links |
| M2: Insecure Data Storage | ✅ | SQLCipher, AndroidKeyStore |
| M3: Insecure Communication | ✅ | TLS 1.3, Certificate Pinning |
| M4: Insecure Authentication | ✅ | Biometric, secure token storage |
| M5: Insufficient Cryptography | ✅ | AES-256-GCM, proper key management |
| M6: Insecure Authorization | ✅ | Server-authoritative roles |
| M7: Client Code Quality | ✅ | Clean architecture, code review |
| M8: Code Tampering | ✅ | ProGuard, signature verification |
| M9: Reverse Engineering | ✅ | Code obfuscation, native crypto |
| M10: Extraneous Functionality | ✅ | Debug logging disabled in release |

---

## 10. Communication Architecture

> **Migration Status:** ✅ COMPLETE
> ArmorChat now uses **Matrix Protocol** as the primary channel for all messaging operations.
> Bridge RPC is reserved for **admin operations only**.

ArmorChat communicates through **three primary channels**:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    COMMUNICATION ARCHITECTURE                            │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │                        ArmorChat (Android App)                   │   │
│   │                                                                  │   │
│   │  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐ │   │
│   │  │ BridgeAdminClient│  │   MatrixClient   │  │  Matrix /sync  │ │   │
│   │  │  (Admin RPC)     │  │  (Messaging)     │  │  (Real-time)   │ │   │
│   │  └────────┬─────────┘  └────────┬─────────┘  └───────┬────────┘ │   │
│   │           │                     │                     │          │   │
│   └───────────┼─────────────────────┼─────────────────────┼──────────┘   │
│               │                     │                     │               │
│               ▼                     ▼                     ▼               │
│   ┌─────────────────────┐  ┌───────────────────────────────────────────┐ │
│   │  ArmorClaw Server   │  │           Matrix Homeserver               │ │
│   │  (VPS :8080)        │  │           (Conduit/Synapse)               │ │
│   │                     │  │                                           │ │
│   │  • JSON-RPC 2.0     │  │  • Client API (Messaging, Rooms)         │ │
│   │  • Admin Operations │  │  • /sync (Real-time Events)              │ │
│   │  • Platform Bridges │  │  • Media Repository (Uploads)            │ │
│   │  • Agent/Keystore   │  │  • E2EE Keys (Megolm)                    │ │
│   │  :8080 (HTTP/RPC)   │  │  :8008 (HTTPS)                           │ │
│   └─────────────────────┘  └───────────────────────────────────────────┘ │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────────┐│
│   │  PRIMARY CHANNEL: Matrix Protocol (E2E Encrypted)                   ││
│   │  ADMIN CHANNEL:  Bridge RPC (JSON-RPC 2.0)                          ││
│   │  REAL-TIME:      Matrix /sync (Long-poll, NOT WebSocket)            ││
│   └─────────────────────────────────────────────────────────────────────┘│
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Communication Endpoints

| Endpoint | Protocol | Purpose | Status |
|----------|----------|---------|--------|
| **Matrix Client API** | HTTPS | Messaging, rooms, presence | ✅ Primary |
| **Matrix /sync** | HTTPS (long-poll) | Real-time events | ✅ Primary |
| **Matrix Media** | HTTPS | File uploads/downloads | ✅ Primary |
| Bridge RPC | HTTP/JSON-RPC 2.0 | Admin operations only | ✅ Admin |
| ~~Bridge WebSocket~~ | ~~WebSocket~~ | ~~Real-time events~~ | ❌ Not available |

### What Goes Where (Data Flow Summary)

| Operation | Channel | Endpoint | Notes |
|-----------|---------|----------|-------|
| Login | Matrix SDK | `/_matrix/client/v3/login` | E2EE keys retrieved |
| Send Message | Matrix SDK | `/_matrix/client/v3/rooms/.../send` | E2E encrypted |
| Receive Message | Matrix /sync | `/_matrix/client/v3/sync` | Long-poll (30s) |
| Typing/Presence | Matrix SDK | `/_matrix/client/v3/...` | Direct to Matrix |
| File Upload | Matrix Media | `/_matrix/media/v3/upload` | Direct to Matrix |
| Start Bridge Session | Bridge RPC | `bridge.start` | Admin via RPC |
| Platform Connect | Bridge RPC | `platform.connect` | Admin via RPC |
| Push Register | Bridge RPC | `push.register_token` | Admin via RPC |
| Recovery Flow | Bridge RPC | `recovery.*` | Admin via RPC |
| WebRTC Signaling | Bridge RPC | `webrtc.*` | Admin via RPC |
| Agent Status | Bridge RPC | `agent.status` | Admin via RPC |
| Keystore Mgmt | Bridge RPC | `keystore.*` | Admin via RPC |

### Bridge RPC Method Coverage (Admin Only)

| Category | Methods | Count |
|----------|---------|-------|
| Bridge Lifecycle | `bridge.start`, `bridge.stop`, `bridge.status`, `bridge.health` | 4 |
| Platform Integration | `platform.connect`, `platform.disconnect`, `platform.list`, `platform.status`, `platform.test` | 5 |
| Push Notifications | `push.register_token`, `push.unregister_token`, `push.update_settings` | 3 |
| Recovery | `recovery.generate_phrase`, `recovery.store_phrase`, `recovery.verify`, `recovery.status`, `recovery.complete`, `recovery.is_device_valid` | 6 |
| WebRTC Signaling | `webrtc.offer`, `webrtc.answer`, `webrtc.ice_candidate`, `webrtc.hangup` | 4 |
| Agent Management | `agent.status`, `agent.status_history`, subscribe flows | 5 |
| Keystore | `keystore.status`, `keystore.unseal_challenge`, `keystore.unseal_respond`, `keystore.extend_session`, subscribe flow | 5 |
| **Total** | | **32** |

### Real-Time Events (Matrix /sync)

> **Important:** Real-time events come from Matrix /sync, NOT from Bridge WebSocket.
> The Bridge WebSocket is not available. Use `MatrixSyncManager` to subscribe to events.

Events received via Matrix /sync long-polling:

| Event Type | Matrix Event | Purpose |
|------------|--------------|---------|
| `m.room.message` | Timeline | New message in room |
| `m.room.encrypted` | Timeline | Encrypted message (decrypt client-side) |
| `m.typing` | Ephemeral | Typing indicator |
| `m.receipt` | Ephemeral | Read receipt |
| `m.presence` | Presence | User presence change |
| `m.room.member` | State | Room membership change |
| `m.call.*` | Timeline | VoIP call signaling |
| `com.armorclaw.*` | Timeline | ArmorClaw control plane events |

### ArmorClaw Control Plane Events

> **Note:** These are `BridgeEvent` types from `platform/bridge/BridgeEvent.kt`, originally designed
> for the Bridge WebSocket channel. Since ArmorChat uses Matrix /sync (not WebSocket), real-time
> equivalents of these events arrive as standard Matrix events (`m.room.message`, `m.typing`, etc.)
> or as ArmorClaw custom events (`com.armorclaw.*` — see Section 11).

| Event Type | Source | Purpose |
|------------|--------|---------|
| `message.received` | BridgeEvent | New message in room |
| `message.status` | BridgeEvent | Message delivery status |
| `room.created` | BridgeEvent | New room created |
| `room.membership` | BridgeEvent | User joined/left room |
| `typing` | Matrix `m.typing` | Typing indicator |
| `receipt.read` | Matrix `m.receipt` | Read receipt |
| `presence` | Matrix `m.presence` | User presence change |
| `call` | Matrix `m.call.*` | WebRTC call signaling |
| `platform.message` | BridgeEvent | External platform message |
| `session.expired` | BridgeEvent | Bridge session ended |
| `bridge.status` | BridgeEvent | Bridge health update |
| `recovery` | BridgeEvent | Recovery flow events |
| `app.armorclaw.alert` | BridgeEvent | System alerts |
| `license` | BridgeEvent | License state changes |
| `budget` | BridgeEvent | Budget usage alerts |
| `compliance` | BridgeEvent | Compliance notifications |

---

## 11. Control Plane & Agent System

### Control Plane Events

ArmorClaw extends Matrix with custom event types for control plane operations:

```kotlin
object ArmorClawEventType {
    // Workflow events
    const val WORKFLOW_STARTED = "com.armorclaw.workflow.started"
    const val WORKFLOW_STEP = "com.armorclaw.workflow.step"
    const val WORKFLOW_COMPLETED = "com.armorclaw.workflow.completed"
    const val WORKFLOW_FAILED = "com.armorclaw.workflow.failed"

    // Agent events
    const val AGENT_TASK_STARTED = "com.armorclaw.agent.task_started"
    const val AGENT_TASK_PROGRESS = "com.armorclaw.agent.task_progress"
    const val AGENT_TASK_COMPLETE = "com.armorclaw.agent.task_complete"
    const val AGENT_THINKING = "com.armorclaw.agent.thinking"

    // System events
    const val BUDGET_WARNING = "com.armorclaw.budget.warning"
    const val BRIDGE_CONNECTED = "com.armorclaw.bridge.connected"
    const val BRIDGE_DISCONNECTED = "com.armorclaw.bridge.disconnected"
    const val PLATFORM_MESSAGE = "com.armorclaw.platform.message"

    fun isArmorClawEvent(eventType: String): Boolean = eventType.startsWith("com.armorclaw.")
    fun isWorkflowEvent(eventType: String): Boolean = eventType.startsWith("com.armorclaw.workflow.")
    fun isAgentEvent(eventType: String): Boolean = eventType.startsWith("com.armorclaw.agent.")
}
```

### Agent Management (NEW in v3.3.0)

ArmorChat now includes a dedicated **Agent Management Screen** for viewing and controlling AI agents:

| Feature | Description | Status |
|---------|-------------|--------|
| Agent List | View all running agents with status | ✅ |
| Agent Status | Real-time status (idle/busy/error) | ✅ |
| Stop Agent | Safely stop a running agent | ✅ |
| Room Association | See which room an agent is active in | ✅ |

**New ViewModels (planned — routes defined, screens not yet built):**
| ViewModel | Purpose |
|-----------|---------|
| `AgentManagementViewModel` | Manage AI agents |
| `HitlViewModel` | Handle HITL approvals |
| `WorkflowViewModel` | Workflow management |

**New RPC Methods:**
| Method | Purpose |
|--------|---------|
| `agent.list` | List all running agents |
| `agent.status` | Get agent status details |
| `agent.get_status` | Get detailed status for a specific agent |
| `agent.status_history` | Get status history for an agent |
| `agent.stop` | Stop a running agent |

### HITL (Human-in-the-Loop) Approvals (NEW in v3.3.0)

The **HITL Approval Screen** allows users to review and approve/reject AI action requests:

| Feature | Description | Status |
|---------|-------------|--------|
| Pending List | View all pending approvals | ✅ |
| Approve/Reject | One-click approval or rejection | ✅ |
| Priority Levels | Critical, High, Medium, Low | ✅ |
| Reason Input | Optional rejection reason | ✅ |

**New RPC Methods:**
| Method | Purpose |
|--------|---------|
| `hitl.pending` | Get pending approvals |
| `hitl.approve` | Approve a request |
| `hitl.reject` | Reject a request |

### Workflow Management (NEW in v3.3.0)

**New RPC Methods:**
| Method | Purpose |
|--------|---------|
| `workflow.templates` | List available workflow templates |
| `workflow.start` | Start a new workflow |
| `workflow.status` | Get workflow status |

### Blocker Resolution (NEW in v3)

When a container step hits a blocker (missing PII, approval required, CAPTCHA), the workflow pauses and the user must provide input through ArmorChat. The `resolveBlocker` RPC method sends the user response back to the Bridge Orchestrator, which unblocks the container step.

**RPC Method:**
| Method | Purpose |
|--------|---------|
| `resolve_blocker` | Submit user input to resolve a workflow blocker |

**Parameters:**
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `workflow_id` | string | Yes | ID of the blocked workflow |
| `step_id` | string | Yes | ID of the blocked step |
| `input` | string | Yes | User-provided response (may contain PII) |
| `note` | string | No | Optional context or instructions |

**Response:** `Result<Map<String, String>>` containing status and workflow metadata.

> **PII Safety:** The `input` parameter may contain secrets. It is never logged or cached by the Bridge API client. Sensitive fields (password, card, key, token) are automatically masked in the BlockerResponseDialog.

**Blocker resolution flow:**
1. Container step encounters blocker (PII needed, approval required)
2. Bridge emits `workflow.blocker_warning` event to Matrix room
3. ArmorChat receives blocker event via `/sync`
4. `GovernanceBanner` shows `BLOCKED` state
5. User taps banner, `BlockerResponseDialog` opens
6. User enters response, `BridgeApi.resolveBlocker()` sends JSON-RPC call
7. Bridge Orchestrator unblocks workflow, container retries step

### Budget Tracking (NEW in v3.3.0)

**New RPC Method:**
| Method | Purpose |
|--------|---------|
| `budget.status` | Get budget usage and limits |

### Agent Types

```kotlin
enum class AgentType {
    GENERAL,           // General purpose assistant
    ANALYSIS,          // Document/data analysis
    CODE_REVIEW,       // Code review and suggestions
    RESEARCH,          // Research and information gathering
    WRITING,           // Content writing
    TRANSLATION,       // Translation services
    SCHEDULING,        // Calendar and scheduling
    WORKFLOW,          // Workflow orchestration
    PLATFORM_BRIDGE    // External platform integration
}
```

### Agent Status

```kotlin
enum class AgentStatus {
    ONLINE,
    BUSY,
    OFFLINE,
    ERROR
}

enum class AgentTaskStatus {
    PENDING,
    RUNNING,
    COMPLETED,
    FAILED,
    CANCELLED
}
```

---

## 12. Feature Reference

### Core Features (150+)

#### Authentication
- ✅ Login (username/email + password)
- ✅ Registration (`RegistrationScreenV2`, 427 lines)
- ✅ Forgot password (`ForgotPasswordScreenV2`, 323 lines)
- ✅ Key recovery (`KeyRecoveryScreenV2`)
- ✅ Biometric authentication
- ✅ Secure session management

#### Onboarding
- ✅ Welcome screen
- ✅ Security explanation (4 steps)
- ✅ Server connection
- ✅ Permission requests
- ✅ Completion celebration
- ✅ Tutorial (optional)
- ✅ State persistence

#### Chat
- ✅ Message list with all states
- ✅ Message bubbles with status
- ✅ Reply/Forward/React
- ✅ File/Image attachments
- ✅ Voice messages
- ✅ Typing indicators
- ✅ Read receipts
- ✅ Thread support
- ✅ Encryption indicators
- ✅ Command detection

#### Workflow & Governance (v3)
- ✅ Workflow timeline viewing (`WorkflowTimeline`, scrollable vertical timeline with step icons and durations)
- ✅ Blocker resolution (`BlockerResponseDialog`, modal with PII-safe input masking)
- ✅ Governance status display (`GovernanceBanner`, persistent banner for RUN/BLOCK/COMPLETE/FAIL states)
- ✅ Blocker RPC (`resolve_blocker`, sends user response to Bridge Orchestrator)

#### Rooms
- ✅ Room list (active/favorites/archived)
- ✅ Create/Join rooms
- ✅ Room details
- ✅ Room settings
- ✅ Pull-to-refresh

#### Calls
- ✅ Voice calls
- ✅ Video calls
- ✅ Incoming call dialog
- ✅ Call controls (mute, speaker, video, hold)

#### Profile
- ✅ Profile display/edit (`ProfileScreenV2`, `UserProfileScreenV2`)
- ✅ Change password (`ChangePasswordScreen` + `ChangePasswordViewModel`)
- ✅ Change phone (`ChangePhoneNumberScreen` + `ChangePhoneViewModel`)
- ✅ Edit bio (`EditBioScreen` + `EditBioViewModel`)
- ⚠️ Delete account (`DeleteAccountScreen` — screen exists, ViewModel pending)

#### Settings
- ✅ Settings screen
- ✅ Security settings
- ✅ Notification settings
- ✅ Appearance settings
- ✅ Device management
- ✅ Verification flow
- ✅ About screen
- ✅ Report bug

### Platform Integrations

| Platform | Bridge | License Tier |
|----------|--------|--------------|
| Slack | ✅ | Essential+ |
| Discord | ✅ | Professional+ |
| Teams | ✅ | Professional+ |
| WhatsApp | ✅ | Enterprise only |

---

## 13. Build & Configuration

### Build Commands

```bash
# Clean
./gradlew clean

# Build Debug APK
./gradlew assembleDebug

# Build Release APK
./gradlew assembleRelease

# Install Debug
./gradlew installDebug

# Unit Tests (requires JAVA_HOME pointing to JDK 17)
JAVA_HOME=~/.asdf/installs/java/adoptopenjdk-17.0.0+35 ./gradlew :androidApp:testDebugUnitTest

# Instrumented Tests (requires connected device)
./gradlew :androidApp:connectedDebugAndroidTest

# Static Analysis
./gradlew detekt

# Maestro UI Flows
maestro test maestro/flows/

# Appium Tests
cd appium && ANDROID_HOME=$HOME/Android/Sdk pytest tests/ -v
```

### Build Variants

| Type | Application ID | Purpose |
|------|---------------|---------|
| Debug | `com.armorclaw.app.debug` | Development (adds `.debug` suffix) |
| Release | `com.armorclaw.app` | Production (R8 + resource shrinking) |

### Build Optimizations

- **R8**: Code shrinking and obfuscation
- **ProGuard**: Additional rules in `androidApp/proguard-rules.pro`
- **Resource Shrinking**: `isShrinkResources = true`
- **Native Library Optimization**: Symbol stripping

### Dependency Versions

| Dependency | Version |
|------------|---------|
| Kotlin | 1.9.20 |
| Compose | 1.5.0 |
| Material 3 | 1.1.2 |
| Koin | 3.5.0 |
| SQLDelight | 2.0.0 |
| Ktor | 2.3.5 |
| Coroutines | 1.7.3 |
| Sentry | 7.6.0 |
| Coil | 2.4.0 |

---

## 14. Development Guidelines

### Code Style

- Follow Kotlin coding conventions
- Use meaningful variable names
- Prefer immutability (val over var)
- Use data classes for models
- Use sealed classes for state

### Architecture Guidelines

- **Domain layer**: No Android dependencies
- **Data layer**: Repository implementations
- **Presentation layer**: ViewModels + Compose
- **Platform layer**: expect/actual for platform code

### Dependency Injection

```kotlin
// Define module
val appModule = module {
    viewModel { ChatViewModel(get(), get(), get()) }
    single<MatrixClient> { MatrixClientFactory.create(get()) }
    single<BridgeRpcClient> { BridgeClientFactory.createClient(get()) }
}

// Use in Compose
val viewModel: ChatViewModel = koinViewModel()
val client: MatrixClient = koinInject()
```

### Navigation

```kotlin
// Navigate with parameters
navController.navigate(AppNavigation.createChatRoute(roomId))

// Navigate with pop up
navController.navigate(AppNavigation.HOME) {
    popUpTo(AppNavigation.SPLASH) { inclusive = true }
}
```

### Logging

```kotlin
// Use LoggerDelegate
private val logger = LoggerDelegate(LogTag.Domain.Chat)

logger.logInfo("Message sent", mapOf("roomId" to roomId))
logger.logError("Failed to send", exception)
```

### Error Handling

```kotlin
// Use AppResult
suspend fun sendMessage(...): AppResult<String> {
    return try {
        val eventId = matrixClient.sendTextMessage(roomId, text).getOrThrow()
        AppResult.success(eventId)
    } catch (e: Exception) {
        AppResult.error(ArmorClawErrorCode.MESSAGE_SEND_FAILED, e.message)
    }
}
```

---

## 15. ArmorChat ↔ ArmorClaw Communication

This section describes how ArmorChat (the Android app) communicates with ArmorClaw (the Bridge server, deployed separately on VPS) and the Matrix homeserver.

> **Migration Status:** ✅ COMPLETE
> ArmorChat now uses **Matrix Protocol** as the primary communication channel with **E2E encryption**.
> Bridge RPC is used **only for admin operations** (lifecycle, recovery, platforms, push).
> See `doc/MATRIX_MIGRATION.md` for full migration details.

### 15.1 Server Discovery Flow

Before ArmorChat can communicate, it must discover server endpoints:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCHAT SERVER DISCOVERY FLOW                              │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  DISCOVERY PRIORITY (Highest to Lowest Trust):                                   │
│                                                                                  │
│  1. SIGNED QR/DEEP LINK ────────────────────────────────────────────────         │
│     │                                                                            │
│     │  User scans QR or taps deep link                                          │
│     │  Format: armorclaw://config?d=<base64-config>                             │
│     │  Contains: matrix_homeserver, rpc_url, ws_url, push_gateway               │
│     │  Signed by Bridge, expires after 24h                                       │
│     │                                                                            │
│     └────────────────────────────────────────────────────────────────────────▶  │
│                                                                                  │
│  2. WELL-KNOWN DISCOVERY ───────────────────────────────────────────────        │
│     │                                                                            │
│     │  User enters domain: "armorclaw.app"                                      │
│     │  App fetches: https://armorclaw.app/.well-known/matrix/client             │
│     │  Response includes Bridge URLs under "com.armorclaw.bridge"               │
│     │  Standard Matrix discovery mechanism                                       │
│     │                                                                            │
│     └────────────────────────────────────────────────────────────────────────▶  │
│                                                                                  │
│  3. mDNS DISCOVERY (Local Network) ─────────────────────────────────────        │
│     │                                                                            │
│     │  App scans for _armorclaw._tcp.local                                      │
│     │  Bridge advertises itself via mDNS                                        │
│     │  Auto-discovers IP, port, and basic config                                │
│     │  May require QR for Matrix URL if not included                            │
│     │                                                                            │
│     └────────────────────────────────────────────────────────────────────────▶  │
│                                                                                  │
│  4. MANUAL ENTRY ───────────────────────────────────────────────────────        │
│     │                                                                            │
│     │  User enters: https://matrix.armorclaw.app                                │
│     │  App derives: https://bridge.armorclaw.app/api                            │
│     │  Fallback when other methods fail                                         │
│     │                                                                            │
│     └────────────────────────────────────────────────────────────────────────▶  │
│                                                                                  │
│  5. FALLBACK SERVERS ───────────────────────────────────────────────────        │
│     │                                                                            │
│     │  Pre-configured backup servers:                                           │
│     │  • bridge.armorclaw.app                                                   │
│     │  • bridge-backup.armorclaw.app                                            │
│     │  • bridge-eu.armorclaw.app                                                │
│     │                                                                            │
│     └────────────────────────────────────────────────────────────────────────▶  │
│                                                                                  │
│  6. BUILDCONFIG FEATURE FLAGS ────────────────────────────────────────        │
│     │                                                                            │
│     │  Build-time feature toggles in build.gradle.kts:                          │
│     │  • FEATURE_PASSWORD_RESET_ENABLED = false                                  │
│     │  • FEATURE_REGISTRATION_ENABLED = false                                    │
│     │  • FEATURE_CHANGE_PASSWORD_ENABLED = false                                 │
│     │  • FEATURE_CHANGE_PHONE_ENABLED = false                                    │
│     │  • FEATURE_EDIT_BIO_ENABLED = false                                        │
│     │  • FEATURE_DELETE_ACCOUNT_ENABLED = false                                  │
│     │  • FEATURE_KEY_BACKUP_ENABLED = false                                      │
│     │  • FEATURE_INVITE_ENABLED = false                                          │
│     │  • FEATURE_ADD_DEVICE_ENABLED = false                                      │
│     │  (Server URLs come from SetupService, not BuildConfig)                     │
│     │                                                                            │
│     └────────────────────────────────────────────────────────────────────────▶  │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.2 Configuration Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                     CONFIGURATION STORAGE & PRIORITY                             │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                        SetupService (Shared)                              │   │
│  │                                                                            │   │
│  │  parseSignedConfig(deepLink) → SetupConfig                                │   │
│  │  startSetupWithDiscovery(input) → SetupConfig                             │   │
│  │  useDemoServer() → SetupConfig                                            │   │
│  │                                                                            │   │
│  │  SetupConfig:                                                              │   │
│  │  • homeserver: String        (Matrix server URL)                          │   │
│  │  • bridgeUrl: String?        (Bridge RPC URL: /api)                       │   │
│  │  • wsUrl: String?            (WebSocket URL: /ws)                         │   │
│  │  • pushGateway: String?      (Push gateway URL)                           │   │
│  │  • serverName: String?       (Human-readable name)                        │   │
│  │  • configSource: ConfigSource (How config was obtained)                   │   │
│  │  • expiresAt: Long?          (Expiration timestamp)                       │   │
│  │                                                                            │   │
│  └────────────────────────────────────────────────────────────────────────────┘   │
│                                      │                                          │
│                                      ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │               SetupService / SetupConfig (Shared Module)                  │   │
│  │                                                                            │   │
│  │  Configuration is managed by SetupService in the shared module.           │   │
│  │  SetupConfig holds: homeserver, bridgeUrl, wsUrl, pushGateway,            │   │
│  │  serverName, configSource, expiresAt.                                     │   │
│  │  ConfigSource enum tracks discovery method.                               │   │
│  │                                                                            │   │
│  └────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  ConfigSource Priority:                                                          │
│  SIGNED_URL > WELL_KNOWN > MDNS > MANUAL > CACHED > DEFAULT                     │
│  (INVITE exists as a source but is not in the priority chain)                    │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.3 Communication Architecture Overview

ArmorChat uses **three parallel communication channels**:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCHAT COMMUNICATION ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                        ArmorChat (Android App)                          │   │
│   │                                                                         │   │
│   │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐      │   │
│   │  │ BridgeRpcClient  │  │ MatrixSyncManager│  │ MatrixClient     │      │   │
│   │  │ (Admin Ops)      │  │ (Real-time)      │  │ (Direct API)     │      │   │
│   │  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘      │   │
│   │           │                     │                     │                │   │
│   └───────────┼─────────────────────┼─────────────────────┼────────────────┘   │
│               │                     │                     │                     │
│               ▼                     ▼                     ▼                     │
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                        COMMUNICATION CHANNELS                            │   │
│   │                                                                          │   │
│   │  CHANNEL 1: Bridge RPC (HTTP/JSON-RPC 2.0)                              │   │
│   │  ────────────────────────────────────────                                │   │
│   │  • Bridge lifecycle (bridge.start/stop/status)                          │   │
│   │  • Recovery operations (recovery.generate_phrase, etc.)                 │   │
│   │  • Platform integration (platform.connect/list/status)                  │   │
│   │  • Push notifications (push.register_token/unregister_token)            │   │
│   │  • WebRTC signaling (webrtc.offer/answer/ice_candidate/hangup)          │   │
  │  │  • Agent management (agent.status/status_history)                       │   │
  │  │  • Keystore management (keystore.status/unseal/session)                 │   │
  │  │  • Provisioning (provisioning.claim/rotate/revoke/get_config/set_config)│   │
  │  │  • Blocker resolution (resolve_blocker)                                │   │
│   │                                                                          │   │
│   │  Endpoint: https://bridge.armorclaw.app/api                             │   │
│   │  Protocol: JSON-RPC 2.0 over HTTPS POST                                 │   │
│   │  ⚠️  IMPORTANT: Endpoint is /api NOT /rpc                               │   │
│   │                                                                          │   │
│   └──────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                                                                          │   │
│   │  CHANNEL 2: Matrix /sync (Direct HTTPS Long-Poll)                       │   │
│   │  ─────────────────────────────────────────────                          │   │
│   │  • Message received events                                              │   │
│   │  • Message status updates (sent/delivered/read)                         │   │
│   │  • Typing indicators                                                    │   │
│   │  • Presence updates                                                     │   │
│   │  • Read receipts                                                        │   │
│   │  • Room membership changes                                              │   │
│   │  • Room state changes (name, topic, avatar, encryption)                 │   │
│   │  • Call signaling (m.call.* events)                                     │   │
│   │  • To-device messages (E2EE key exchange)                               │   │
│   │  • Device list changes (cross-signing)                                  │   │
│   │                                                                          │   │
│   │  Endpoint: https://matrix.armorclaw.app/_matrix/client/v3/sync          │   │
│   │  Protocol: HTTPS GET with long-polling (30s timeout)                    │   │
│   │                                                                          │   │
│   └──────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                                                                          │   │
│   │  CHANNEL 3: Matrix Media (Direct HTTPS Upload)                          │   │
│   │  ─────────────────────────────────────────                              │   │
│   │  • File uploads (images, videos, documents)                             │   │
│   │  • Thumbnail generation                                                  │   │
│   │  • Avatar uploads                                                        │   │
│   │                                                                          │   │
│   │  Endpoint: https://matrix.armorclaw.app/_matrix/media/v3/upload         │   │
│   │  Protocol: HTTPS POST (multipart/form-data)                             │   │
│   │                                                                          │   │
│   └──────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│                                   ▼                                              │
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                          SERVER INFRASTRUCTURE                           │   │
│   │                                                                          │   │
│   │  ┌─────────────────────┐       ┌─────────────────────┐                  │   │
│   │  │  ArmorClaw Server   │       │  Matrix Homeserver  │                  │   │
│   │  │  (VPS :8080)        │       │  (Conduit/Synapse)  │                  │   │
│   │  │                     │       │                     │                  │   │
│   │  │  • HTTP :8443/api   │◄─────►│  • Client API       │                  │   │
│   │  │  • JSON-RPC 2.0     │       │  • Federation       │                  │   │
│   │  │  • Platform bridges │       │  • Media repo       │                  │   │
│   │  │  • License mgmt     │       │  • E2EE keys        │                  │   │
│   │  │  • QR generation    │       │                     │                  │   │
│   │  └─────────────────────┘       └─────────────────────┘                  │   │
│   │                                                                          │   │
│   └──────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.4 Communication Endpoints

| Endpoint | Protocol | Purpose | Default URL |
|----------|----------|---------|-------------|
| **Bridge RPC** | HTTP/JSON-RPC 2.0 | Admin operations, message sending | `https://bridge.armorclaw.app/api` |
| **Matrix Client API** | HTTPS | Direct Matrix operations | `https://matrix.armorclaw.app/_matrix/client/v3/` |
| **Matrix Sync** | HTTPS (long-poll) | Real-time events | `https://matrix.armorclaw.app/_matrix/client/v3/sync` |
| **Matrix Media** | HTTPS | File uploads/downloads | `https://matrix.armorclaw.app/_matrix/media/v3/` |
| ~~Bridge WebSocket~~ | ~~WebSocket~~ | ~~Real-time events~~ | **NOT AVAILABLE** (use Matrix /sync) |

### 15.5 Configuration

The app supports multiple configuration sources with priority ordering:

| Priority | Source | Description |
|----------|--------|-------------|
| 1 (Highest) | Runtime Config | Set via `BridgeConfig.setRuntimeConfig()` |
| 2 | Well-Known Discovery | Auto-discovered from `/.well-known/matrix/client` |
| 3 | SetupService | QR code, manual entry, URL derivation |
| 4 (Lowest) | Hardcoded Fallbacks | `BridgeConfig.PRODUCTION` defaults |

```kotlin
data class BridgeConfig(
    val baseUrl: String,                          // Bridge server URL (no /api suffix)
    val homeserverUrl: String,                    // Matrix homeserver URL
    val wsUrl: String? = null,                    // WebSocket URL (null — not available)
    val timeoutMs: Long = 30000,
    val enableCertificatePinning: Boolean = true,
    val certificatePins: List<String> = emptyList(),
    val retryCount: Int = 3,
    val retryDelayMs: Long = 1000,
    val useDirectMatrixSync: Boolean = true,      // Use Matrix /sync
    val environment: Environment = Environment.PRODUCTION,
    val serverName: String? = null
) {
    enum class Environment {
        DEVELOPMENT,    // Local development
        STAGING,        // Test server
        PRODUCTION,     // Live server
        CUSTOM          // User-configured (self-hosted VPS)
    }

    val isDebug: Boolean  // baseUrl contains localhost/10.0.2.2/192.168.
    val isSecure: Boolean // both URLs use https://
    val displayName: String  // serverName or derived from environment

    companion object {
        fun setRuntimeConfig(config: BridgeConfig)
        fun getCurrent(): BridgeConfig   // runtime > PRODUCTION
        fun clearRuntimeConfig()
        fun createCustom(bridgeUrl: String, homeserverUrl: String, serverName: String? = null): BridgeConfig
        fun deriveBridgeUrl(homeserver: String): String

        val PRODUCTION = BridgeConfig(
            baseUrl = "https://bridge.armorclaw.app",
            homeserverUrl = "https://matrix.armorclaw.app",
            enableCertificatePinning = true,
            useDirectMatrixSync = true,
            environment = Environment.PRODUCTION,
            serverName = "ArmorClaw"
        )

        val DEVELOPMENT = BridgeConfig(
            baseUrl = "http://10.0.2.2:8080",      // Android emulator
            homeserverUrl = "http://10.0.2.2:8008", // Local Matrix
            enableCertificatePinning = false,
            retryCount = 1,
            useDirectMatrixSync = true,
            environment = Environment.DEVELOPMENT,
            serverName = "Development Server"
        )

        val STAGING = BridgeConfig(
            baseUrl = "https://bridge-staging.armorclaw.app",
            homeserverUrl = "https://matrix-staging.armorclaw.app",
            enableCertificatePinning = false,
            useDirectMatrixSync = true,
            environment = Environment.STAGING,
            serverName = "Staging Server"
        )

        val DEMO = BridgeConfig(
            baseUrl = "https://bridge-demo.armorclaw.app",
            homeserverUrl = "https://matrix-demo.armorclaw.app",
            enableCertificatePinning = false,
            useDirectMatrixSync = true,
            environment = Environment.STAGING,
            serverName = "Demo Server"
        )
    }
}
```

### 15.5.1 Well-Known Discovery (NEW)

The app supports automatic server configuration via Matrix well-known:

```
GET https://example.com/.well-known/matrix/client
```

Response format:
```json
{
  "m.homeserver": {
    "base_url": "https://matrix.example.com"
  },
  "com.armorclaw": {
    "bridge_url": "https://bridge.example.com",
    "rpc_url": "https://bridge.example.com/api",
    "ws_url": "wss://bridge.example.com/ws",
    "push_gateway": "https://bridge.example.com/_matrix/push/v1/notify",
    "server_name": "Example Server",
    "version": "1.6.2",
    "region": "us-east",
    "supports_e2ee": true,
    "supports_recovery": true
  }
}
```

### 15.5.2 Self-Hosted VPS Configuration (NEW)

For connecting to your own Matrix homeserver:

**Option 1: Runtime Configuration**
```kotlin
BridgeConfig.setRuntimeConfig(
    BridgeConfig.createCustom(
        bridgeUrl = "https://your-vps.com:8080",
        homeserverUrl = "https://your-vps.com:8008",
        serverName = "My VPS"
    )
)
```

**Option 2: Well-Known Auto-Discovery**
Set up `/.well-known/matrix/client` on your domain.

**Option 3: Manual Entry in App**
Enter server address in Connect screen - app will auto-derive endpoints.

### 15.6 Setup Flow (First Connection)

When a user first sets up ArmorChat:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          SETUP FLOW COMMUNICATION                                │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  STEP 1: USER ENTERS SERVER INFO                                                 │
│  ────────────────────────────────                                                │
│  User Input:                                                                     │
│  • Homeserver URL (e.g., "https://matrix.example.com")                          │
│  • Bridge URL (optional, auto-derived if not provided)                          │
│                                                                                  │
│  STEP 2: SERVER HEALTH CHECK                                                     │
│  ────────────────────────────                                                    │
│  ArmorChat ──HTTP──▶ POST /rpc (JSON-RPC 2.0)                                   │
│                                                                                  │
│  Request:                                                                        │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "method": "bridge.health",                                                   │
│    "params": {                                                                   │
│      "correlation_id": "corr_abc123"                                            │
│    },                                                                           │
│    "id": "req_001"                                                              │
│  }                                                                              │
│                                                                                  │
│  Response:                                                                       │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "result": {                                                                   │
│      "version": "1.6.2",                                                        │
│      "supports_e2ee": true,                                                     │
│      "supports_recovery": true,                                                 │
│      "region": "us-east"                                                        │
│    },                                                                           │
│    "id": "req_001"                                                              │
│  }                                                                              │
│                                                                                  │
│  STEP 3: USER ENTERS CREDENTIALS                                                 │
│  ──────────────────────────────────                                              │
│  User Input:                                                                     │
│  • Username (e.g., "@alice:example.com")                                        │
│  • Password                                                                     │
│                                                                                  │
│  STEP 4: BRIDGE SESSION START                                                    │
│  ────────────────────────────                                                    │
│  ArmorChat ──RPC───▶ bridge.start(userId, deviceId)                             │
│                                                                                  │
│  Request:                                                                        │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "method": "bridge.start",                                                    │
│    "params": {                                                                   │
│      "user_id": "@alice:example.com",                                           │
│      "device_id": "ANDROID_abc123",                                             │
│      "correlation_id": "corr_def456"                                            │
│    },                                                                           │
│    "id": "req_002"                                                              │
│  }                                                                              │
│                                                                                  │
│  Response:                                                                       │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "result": {                                                                   │
│      "session_id": "sess_xyz789",                                               │
│      "container_id": "container_abc",                                           │
│      "status": "running",                                                       │
│      "ice_servers": [                                                           │
│        {"urls": ["stun:stun.l.google.com:19302"]},                              │
│        {"urls": ["turn:turn.armorclaw.app:3478"], "username": "...", ...}       │
│      ]                                                                          │
│    },                                                                           │
│    "id": "req_002"                                                              │
│  }                                                                              │
│                                                                                  │
│  STEP 5: MATRIX AUTHENTICATION (via MatrixClient)                                │
│  ────────────────────────────────────────────                                    │
│  ⚠️  NOTE: matrix.login RPC is @Deprecated. Use MatrixClient.login() instead.   │
│  ArmorChat ──HTTPS──▶ POST /_matrix/client/v3/login                             │
│                                                                                  │
│  Request (direct Matrix Client API — NOT JSON-RPC):                              │
│  POST https://matrix.example.com/_matrix/client/v3/login                         │
│  {                                                                               │
│    "type": "m.login.password",                                                   │
│    "identifier": { "type": "m.id.user", "user": "@alice:example.com" },          │
│    "password": "********",                                                       │
│    "device_id": "ANDROID_abc123"                                                 │
│  }                                                                              │
│                                                                                  │
│  Response:                                                                       │
│  {                                                                               │
│    "user_id": "@alice:example.com",                                             │
│    "access_token": "syt_abc123...",                                             │
│    "device_id": "ANDROID_abc123",                                               │
│    "refresh_token": "ref_xyz...",                                                │                                             │
│      "display_name": "Alice",                                                   │
│      "avatar_url": "mxc://matrix.example.com/avatar123"                         │
│    },                                                                           │
│    "id": "req_003"                                                              │
│  }                                                                              │
│                                                                                  │
│  STEP 6: START MATRIX SYNC (Real-time Events)                                    │
│  ─────────────────────────────────────────────                                   │
│  ArmorChat ──HTTPS──▶ GET /_matrix/client/v3/sync                               │
│                                                                                  │
│  Request:                                                                        │
│  GET https://matrix.example.com/_matrix/client/v3/sync                          │
│    ?access_token=syt_abc123...                                                  │
│    &timeout=30000                                                               │
│    &since=<empty for initial sync>                                              │
│                                                                                  │
│  Response:                                                                       │
│  {                                                                               │
│    "next_batch": "batch_token_abc123",                                          │
│    "rooms": {                                                                    │
│      "join": {                                                                   │
│        "!room1:example.com": {                                                  │
│          "timeline": { "events": [...] },                                       │
│          "state": { "events": [...] },                                          │
│          "ephemeral": { "events": [...] }                                       │
│        }                                                                         │
│      }                                                                           │
│    },                                                                           │
│    "presence": { "events": [...] },                                             │
│    "device_lists": { "changed": [...], "left": [...] }                          │
│  }                                                                              │
│                                                                                  │
│  ⚠️  NOTE: This replaces WebSocket. The sync loop continues indefinitely.       │
│                                                                                  │
│  STEP 7: REGISTER PUSH TOKEN                                                     │
│  ────────────────────────────                                                    │
│  ArmorChat ──RPC───▶ push.register_token(token, platform, deviceId)             │
│                                                                                  │
│  Request:                                                                        │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "method": "push.register_token",                                             │
│    "params": {                                                                   │
│      "push_token": "fcm_xyz789...",                                             │
│      "push_platform": "fcm",                                                    │
│      "device_id": "ANDROID_abc123"                                              │
│    },                                                                           │
│    "id": "req_004"                                                              │
│  }                                                                              │
│                                                                                  │
│  STEP 8: SETUP COMPLETE                                                          │
│  ─────────────────────                                                           │
│  • Session persisted in encrypted storage (SQLCipher)                           │
│  • Push token registered with bridge                                            │
│  • Matrix sync loop running in background                                       │
│  • User navigated to home screen                                                 │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.7 Message Sending Flow

When a user sends a text message:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        MESSAGE SENDING FLOW                                      │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  USER SENDS MESSAGE                                                              │
│  ────────────────────                                                            │
│  User types "Hello!" and presses send in room !abc123:example.com              │
│                                                                                  │
│  STEP 1: LOCAL STATE UPDATE (Optimistic UI)                                      │
│  ─────────────────────────────────────────────                                   │
│  ChatViewModel updates local state:                                              │
│  • Add message to list with status SENDING                                       │
│  • Message appears immediately in UI with clock icon                             │
│                                                                                  │
│  STEP 2: SEND VIA MATRIX CLIENT (Primary — NOT via Bridge RPC)                   │
│  ───────────────────────────────────────────────────────                          │
│  ⚠️  NOTE: matrix.send RPC is @Deprecated. Use MatrixClient.sendTextMessage().  │
│  ArmorChat ──HTTPS──▶ PUT /_matrix/client/v3/rooms/{roomId}/send/{txnId}        │
│                                                                                  │
│  Request (direct Matrix Client API — NOT JSON-RPC):                              │
│  PUT https://matrix.example.com/_matrix/client/v3/rooms/!abc123:example.com/     │
│      send/m.room.encrypted/txn_1708123456789                                     │
│  {                                                                               │
│    "algorithm": "m.megolm.v1.aes-sha2",                                          │
│    "ciphertext": "encrypted_payload...",                                          │
│    "sender_key": "device_key...",                                                 │
│    "session_id": "session_id..."                                                  │
│  }                                                                              │
│  (Message encrypted client-side with Megolm before sending)                      │
│                                                                                  │
│  STEP 3: MATRIX HOMESERVER RESPONSE                                              │
│  ───────────────────────────────────                                             │
│  Response:                                                                       │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│  Response (direct Matrix API — NOT JSON-RPC):                                    │
│  {                                                                               │
│    "event_id": "$event_xyz789"                                                   │
│  }                                                                              │
│                                                                                  │
│  STEP 4: LOCAL STATE UPDATE (Confirmed)                                          │
│  ─────────────────────────────────────────                                       │
│  ChatViewModel updates message:                                                  │
│  • Status: SENDING → SENT                                                        │
│  • Event ID: $event_xyz789                                                       │
│  • Single tick icon (✓)                                                          │
│                                                                                  │
│  STEP 5: DELIVERY CONFIRMATION VIA MATRIX /SYNC                                  │
│  ────────────────────────────────────────────────                                │
│  Matrix sync response includes the sent event (echo):                            │
│                                                                                  │
│  {                                                                               │
│    "next_batch": "batch_token_def456",                                          │
│    "rooms": {                                                                    │
│      "join": {                                                                   │
│        "!abc123:example.com": {                                                 │
│          "timeline": {                                                           │
│            "events": [                                                           │
│              {                                                                   │
│                "event_id": "$event_xyz789",                                     │
│                "type": "m.room.message",                                        │
│                "sender": "@alice:example.com",                                  │
│                "content": {"msgtype": "m.text", "body": "Hello!"},              │
│                "origin_server_ts": 1708123456789                                │
│              }                                                                   │
│            ]                                                                     │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  STEP 6: READ RECEIPT VIA MATRIX /SYNC                                           │
│  ─────────────────────────────────────                                           │
│  When recipient reads the message:                                               │
│                                                                                  │
│  {                                                                               │
│    "rooms": {                                                                    │
│      "join": {                                                                   │
│        "!abc123:example.com": {                                                 │
│          "ephemeral": {                                                          │
│            "events": [                                                           │
│              {                                                                   │
│                "type": "m.receipt",                                             │
│                "content": {                                                      │
│                  "$event_xyz789": {                                             │
│                    "m.read": {                                                   │
│                      "@bob:example.com": {"ts": 1708123457890}                  │
│                    }                                                             │
│                  }                                                               │
│                }                                                                 │
│              }                                                                   │
│            ]                                                                     │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  ChatViewModel updates message:                                                  │
│  • Status: SENT → READ                                                           │
│  • Double tick icon (✓✓)                                                         │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.8 File/Media Upload Flow

When a user sends an image or file:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        FILE/MEDIA UPLOAD FLOW                                    │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  USER SELECTS FILE                                                               │
│  ────────────────────                                                            │
│  User selects "vacation.jpg" (2.5 MB) to send                                   │
│                                                                                  │
│  STEP 1: LOCAL PREVIEW & COMPRESSION                                             │
│  ─────────────────────────────────────                                           │
│  • Generate thumbnail for preview                                                │
│  • Compress if needed (configurable quality)                                     │
│  • Display preview in UI                                                         │
│  • Show upload progress indicator                                                │
│  • Message status: UPLOADING                                                     │
│                                                                                  │
│  STEP 2: UPLOAD TO MATRIX MEDIA REPOSITORY (Direct)                              │
│  ────────────────────────────────────────────────────                            │
│  ArmorChat ──HTTPS──▶ POST /_matrix/media/v3/upload                              │
│                                                                                  │
│  Request:                                                                        │
│  POST https://matrix.example.com/_matrix/media/v3/upload                        │
│    ?access_token=syt_abc123...                                                  │
│    &filename=vacation.jpg                                                       │
│  Headers:                                                                        │
│    Content-Type: image/jpeg                                                     │
│    Content-Length: 2621440                                                      │
│  Body:                                                                           │
│    <binary file data>                                                           │
│                                                                                  │
│  Upload Progress:                                                                │
│  • Ktor tracks bytes sent                                                       │
│  • Progress emitted to UI (0% → 100%)                                           │
│  • User can cancel upload                                                       │
│                                                                                  │
│  STEP 3: MATRIX RESPONSE                                                         │
│  ─────────────────────                                                           │
│  Response:                                                                       │
│  {                                                                               │
│    "content_uri": "mxc://matrix.example.com/abc123def456"                       │
│  }                                                                              │
│                                                                                  │
│  STEP 4: GENERATE THUMBNAIL (Client-Side)                                        │
│  ────────────────────────────────────────                                        │
│  • Create thumbnail (320x180 or similar)                                        │
│  • Upload thumbnail to Matrix media repo                                        │
│  • Get thumbnail MXC URL                                                        │
│                                                                                  │
│  STEP 5: SEND MATRIX MESSAGE WITH ATTACHMENT (via MatrixClient)                 │
│  ─────────────────────────────────────────────────────────                       │
│  ⚠️  NOTE: matrix.send RPC is @Deprecated. Use MatrixClient.sendMediaMessage(). │
│  ArmorChat ──HTTPS──▶ PUT /_matrix/client/v3/rooms/{roomId}/send/{txnId}        │
│                                                                                  │
│  Request (direct Matrix Client API — NOT JSON-RPC):                              │
│  PUT https://matrix.example.com/_matrix/client/v3/rooms/!abc123:example.com/     │
│      send/m.room.encrypted/txn_media_123                                         │
│  {                                                                               │
│    "content": {                                                                  │
│        "msgtype": "m.image",                                                    │
│        "body": "vacation.jpg",                                                  │
│        "url": "mxc://matrix.example.com/abc123def456",                          │
│        "info": {                                                                 │
│          "mimetype": "image/jpeg",                                              │
│          "size": 2621440,                                                       │
│          "w": 1920,                                                             │
│          "h": 1080,                                                             │
│          "thumbnail_url": "mxc://matrix.example.com/thumb_abc123",              │
│          "thumbnail_info": {                                                    │
│            "mimetype": "image/jpeg",                                            │
│            "size": 51200,                                                       │
│            "w": 320,                                                            │
│            "h": 180                                                             │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                            │
│  }                                                                              │
│                                                                                  │
│  STEP 6: LOCAL STATE UPDATE                                                      │
│  ───────────────────────                                                         │
│  • Message status: UPLOADING → SENT                                              │
│  • Image displayed with MXC URL                                                  │
│  • Thumbnail loaded from thumbnail_url                                           │
│                                                                                  │
│  STEP 7: ENCRYPTED ROOM HANDLING (If Applicable)                                 │
│  ────────────────────────────────────────────                                    │
│  For encrypted rooms:                                                            │
│  • Client encrypts file with Megolm room key                                    │
│  • Uses Matrix encrypted media format                                           │
│  • URL points to encrypted file                                                 │
│  • Key sent via m.room.encrypted event                                          │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.9 Real-Time Events (Matrix /sync)

Real-time events come from Matrix /sync endpoint, NOT from WebSocket:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                      MATRIX /SYNC EVENT TYPES                                    │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  MatrixSyncManager polls: GET /_matrix/client/v3/sync?timeout=30000             │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  MESSAGE RECEIVED (Timeline Event)                                               │
│  ───────────────────────────────────                                             │
│  {                                                                               │
│    "rooms": {                                                                    │
│      "join": {                                                                   │
│        "!abc123:example.com": {                                                 │
│          "timeline": {                                                           │
│            "events": [                                                           │
│              {                                                                   │
│                "event_id": "$event_new123",                                     │
│                "type": "m.room.message",                                        │
│                "sender": "@bob:example.com",                                    │
│                "content": {                                                      │
│                  "msgtype": "m.text",                                           │
│                  "body": "Hey Alice!"                                           │
│                },                                                                │
│                "origin_server_ts": 1708123456789                                │
│              }                                                                   │
│            ]                                                                     │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  ENCRYPTED MESSAGE RECEIVED                                                      │
│  ──────────────────────────────────                                              │
│  {                                                                               │
│    "event_id": "$encrypted_event",                                              │
│    "type": "m.room.encrypted",                                                  │
│    "sender": "@bob:example.com",                                                │
│    "content": {                                                                  │
│      "algorithm": "m.megolm.v1.aes-sha2",                                       │
│      "ciphertext": "encrypted...",                                               │
│      "sender_key": "device_key...",                                              │
│      "session_id": "session_id..."                                               │
│    }                                                                             │
│  }                                                                              │
│  → Bridge decrypts using room key                                                │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  TYPING INDICATOR (Ephemeral Event)                                              │
│  ───────────────────────────────────                                             │
│  {                                                                               │
│    "rooms": {                                                                    │
│      "join": {                                                                   │
│        "!abc123:example.com": {                                                 │
│          "ephemeral": {                                                          │
│            "events": [                                                           │
│              {                                                                   │
│                "type": "m.typing",                                              │
│                "content": {                                                      │
│                  "user_ids": ["@bob:example.com", "@charlie:example.com"]       │
│                }                                                                 │
│              }                                                                   │
│            ]                                                                     │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  PRESENCE UPDATE (Top-Level Presence)                                            │
│  ────────────────────────────────────                                            │
│  {                                                                               │
│    "presence": {                                                                 │
│      "events": [                                                                 │
│        {                                                                         │
│          "type": "m.presence",                                                  │
│          "sender": "@bob:example.com",                                          │
│          "content": {                                                            │
│            "presence": "online",                                                │
│            "status_msg": "Working from home",                                    │
│            "last_active_ago": 60000                                              │
│          }                                                                       │
│        }                                                                         │
│      ]                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  READ RECEIPT (Ephemeral Event)                                                  │
│  ────────────────────────────────                                                │
│  {                                                                               │
│    "rooms": {                                                                    │
│      "join": {                                                                   │
│        "!abc123:example.com": {                                                 │
│          "ephemeral": {                                                          │
│            "events": [                                                           │
│              {                                                                   │
│                "type": "m.receipt",                                             │
│                "content": {                                                      │
│                  "$event_xyz789": {                                             │
│                    "m.read": {                                                   │
│                      "@bob:example.com": {"ts": 1708123457890}                  │
│                    }                                                             │
│                  }                                                               │
│                }                                                                 │
│              }                                                                   │
│            ]                                                                     │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  ROOM MEMBERSHIP CHANGE (State Event)                                            │
│  ────────────────────────────────────                                            │
│  {                                                                               │
│    "rooms": {                                                                    │
│      "join": {                                                                   │
│        "!abc123:example.com": {                                                 │
│          "timeline": {                                                           │
│            "events": [                                                           │
│              {                                                                   │
│                "type": "m.room.member",                                         │
│                "state_key": "@dave:example.com",                                │
│                "sender": "@alice:example.com",                                  │
│                "content": {                                                      │
│                  "membership": "join",                                          │
│                  "displayname": "Dave"                                          │
│                }                                                                 │
│              }                                                                   │
│            ]                                                                     │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  CALL SIGNALING (Timeline Event - Vo6)                                           │
│  ─────────────────────────────────────────                                       │
│  {                                                                               │
│    "event_id": "$call_invite_123",                                              │
│    "type": "m.call.invite",                                                     │
│    "sender": "@bob:example.com",                                                │
│    "content": {                                                                  │
│      "call_id": "call_abc123",                                                  │
│      "version": "1",                                                            │
│      "lifetime": 60000,                                                          │
│      "offer": {                                                                  │
│        "type": "offer",                                                          │
│        "sdp": "v=0\r\no=- 4611731400430051336 2 IN IP4 127.0.0.1..."            │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  Also: m.call.candidates, m.call.answer, m.call.hangup                          │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  ROOM INVITE (Invite Section)                                                    │
│  ────────────────────────────                                                    │
│  {                                                                               │
│    "rooms": {                                                                    │
│      "invite": {                                                                 │
│        "!newroom:example.com": {                                                │
│          "invite_state": {                                                       │
│            "events": [                                                           │
│              {                                                                   │
│                "type": "m.room.member",                                         │
│                "state_key": "@alice:example.com",                               │
│                "sender": "@bob:example.com",                                    │
│                "content": {"membership": "invite"}                              │
│              }                                                                   │
│            ]                                                                     │
│          }                                                                       │
│        }                                                                         │
│      }                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  DEVICE LIST CHANGES (For E2EE)                                                  │
│  ──────────────────────────────────                                              │
│  {                                                                               │
│    "device_lists": {                                                             │
│      "changed": ["@bob:example.com"],                                           │
│      "left": ["@charlie:example.com"]                                           │
│    },                                                                           │
│    "device_one_time_keys_count": {                                              │
│      "signed_curve25519": 50                                                    │
│    }                                                                            │
│  }                                                                              │
│                                                                                  │
│  ─────────────────────────────────────────────────────────────────────────────  │
│  TO-DEVICE MESSAGES (For E2EE Key Exchange)                                      │
│  ───────────────────────────────────────────                                     │
│  {                                                                               │
│    "to_device": {                                                                │
│      "events": [                                                                 │
│        {                                                                         │
│          "type": "m.room_key",                                                  │
│          "sender": "@bob:example.com",                                          │
│          "content": {                                                            │
│            "algorithm": "m.megolm.v1.aes-sha2",                                 │
│            "room_id": "!abc123:example.com",                                    │
│            "session_key": "..."                                                  │
│          }                                                                       │
│        }                                                                         │
│      ]                                                                           │
│    }                                                                             │
│  }                                                                              │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.10 MatrixSyncManager Events

The MatrixSyncManager converts raw Matrix sync events to typed Kotlin events:

```kotlin
sealed class MatrixSyncEvent {
    abstract val timestamp: Long

    // Messages
    data class MessageReceived(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Typing
    data class TypingNotification(
        val roomId: String,
        val userIds: List<String>,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Presence
    data class PresenceUpdate(
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Receipts
    data class ReceiptEvent(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class FullyReadMarker(
        val roomId: String,
        val eventId: String?,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Room membership
    data class RoomMembership(
        val roomId: String,
        val userId: String,
        val membership: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    data class InviteReceived(
        val roomId: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Room state changes
    data class RoomNameChanged(val roomId: String, val name: String?, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())
    data class RoomTopicChanged(val roomId: String, val topic: String?, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())
    data class RoomAvatarChanged(val roomId: String, val avatarUrl: String?, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())
    data class RoomEncryptionEnabled(val roomId: String, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())
    data class RoomPowerLevelsChanged(val roomId: String, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())
    data class RoomTagsUpdated(val roomId: String, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())

    // Reactions & Redactions
    data class ReactionEvent(val roomId: String, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())
    data class RedactionEvent(val roomId: String, val event: MatrixEventRaw, override val timestamp: Long = System.currentTimeMillis())

    // Calls
    data class CallSignaling(
        val roomId: String,
        val eventType: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // To-device messages (E2EE key exchange)
    data class ToDeviceMessage(
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Browser automation events (ArmorClaw custom events)
    data class BrowserCommandEvent(
        val roomId: String,
        val eventType: String,
        val event: MatrixEventRaw,
        override val timestamp: Long = System.currentTimeMillis()
    ) : MatrixSyncEvent()

    // Errors
    data class SyncError(val error: Throwable, override val timestamp: Long = System.currentTimeMillis()) : MatrixSyncEvent()
}
```

### 15.11 Platform Integration Flow

Connecting external platforms (Slack, Discord, Teams):

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    PLATFORM INTEGRATION FLOW                                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  USER WANTS TO CONNECT SLACK                                                     │
│  ───────────────────────────────────                                             │
│                                                                                  │
│  STEP 1: INITIATE CONNECTION                                                     │
│  ────────────────────────────                                                    │
│  ArmorChat ──RPC───▶ platform.connect("slack", config)                          │
│                                                                                  │
│  Request:                                                                        │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "method": "platform.connect",                                                │
│    "params": {                                                                   │
│      "platform_type": "slack",                                                  │
│      "config": {                                                                 │
│        "workspace": "mycompany",                                                │
│        "channels": ["general", "random"]                                        │
│      }                                                                           │
│    },                                                                           │
│    "id": "req_007"                                                              │
│  }                                                                              │
│                                                                                  │
│  STEP 2: OAUTH FLOW                                                              │
│  ───────────────────                                                             │
│  Response:                                                                       │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "result": {                                                                   │
│      "success": false,                                                          │
│      "auth_url": "https://slack.com/oauth/authorize?client_id=...",             │
│      "platform_id": null,                                                       │
│      "message": "Authorization required"                                        │
│    },                                                                           │
│    "id": "req_007"                                                              │
│  }                                                                              │
│                                                                                  │
│  STEP 3: USER AUTHORIZES IN BROWSER                                              │
│  ──────────────────────────────────────                                          │
│  • ArmorChat opens auth_url in browser                                          │
│  • User logs into Slack and authorizes                                          │
│  • Slack redirects to callback URL on Bridge                                    │
│  • Bridge receives callback with auth code                                      │
│  • Bridge exchanges code for access token                                       │
│                                                                                  │
│  STEP 4: CONNECTION COMPLETE (Polling or Next RPC Call)                          │
│  ────────────────────────────────────────────────────────                        │
│  Check connection status:                                                       │
│  ArmorChat ──RPC───▶ platform.list()                                            │
│                                                                                  │
│  Response:                                                                       │
│  {                                                                               │
│    "jsonrpc": "2.0",                                                            │
│    "result": {                                                                   │
│      "platforms": [                                                              │
│        {                                                                         │
│          "id": "slack_mycompany",                                               │
│          "type": "slack",                                                       │
│          "name": "MyCompany Workspace",                                         │
│          "status": "connected",                                                 │
│          "connected_at": 1708123460000,                                         │
│          "last_sync": 1708123470000                                             │
│        }                                                                         │
│      ]                                                                           │
│    },                                                                           │
│    "id": "req_008"                                                              │
│  }                                                                              │
│                                                                                  │
│  STEP 5: BIDIRECTIONAL MESSAGE FLOW                                              │
│  ──────────────────────────────────────                                          │
│                                                                                  │
│  Slack → Matrix:                                                                 │
│  • User posts in #general on Slack                                              │
│  • Bridge receives via Slack Events API                                         │
│  • Bridge creates Matrix event in !slack_general:example.com                    │
│  • ArmorChat receives via Matrix /sync (NOT WebSocket)                          │
│                                                                                  │
│  Matrix → Slack:                                                                 │
│  • User posts in !slack_general:example.com                                     │
│  • Bridge receives via Matrix appservice                                        │
│  • Bridge sends to Slack #general via Web API                                   │
│  • Message appears in Slack                                                     │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.12 RPC Method Reference

All RPC methods available in BridgeRpcClient. **Note: Methods use snake_case to match Bridge API.**

> **Migration Status:** ✅ COMPLETE
> Matrix methods (login, sync, send, rooms) are **DEPRECATED** - use `MatrixClient` instead.
> Only use Bridge RPC for **admin operations** (lifecycle, recovery, platforms, push, license).

#### ✅ Available Methods (Bridge Admin)

| Kotlin Method | RPC Method | Category | Description |
|---------------|------------|----------|-------------|
| `startBridge()` | `bridge.start` | Lifecycle | Start bridge session |
| `stopBridge()` | `bridge.stop` | Lifecycle | Stop bridge session |
| `getBridgeStatus()` | `bridge.status` | Lifecycle | Get session status |
| `healthCheck()` | `bridge.health` | Lifecycle | Health check |
| `recoveryGeneratePhrase()` | `recovery.generate_phrase` | Recovery | Generate BIP39 phrase |
| `recoveryStorePhrase()` | `recovery.store_phrase` | Recovery | Store encrypted phrase |
| `recoveryVerify()` | `recovery.verify` | Recovery | Verify phrase |
| `recoveryStatus()` | `recovery.status` | Recovery | Get recovery status |
| `recoveryComplete()` | `recovery.complete` | Recovery | Complete recovery |
| `recoveryIsDeviceValid()` | `recovery.is_device_valid` | Recovery | Check device validity |
| `platformConnect()` | `platform.connect` | Platforms | Connect platform |
| `platformDisconnect()` | `platform.disconnect` | Platforms | Disconnect platform |
| `platformList()` | `platform.list` | Platforms | List platforms |
| `platformStatus()` | `platform.status` | Platforms | Get platform status |
| `platformTest()` | `platform.test` | Platforms | Test connection |
| `pushRegister()` | `push.register_token` | Push | Register FCM/APNs |
| `pushUnregister()` | `push.unregister_token` | Push | Unregister push |
| `pushUpdateSettings()` | `push.update_settings` | Push | Update settings |
| `webrtcOffer()` | `webrtc.offer` | WebRTC | WebRTC call offer |
| `webrtcAnswer()` | `webrtc.answer` | WebRTC | WebRTC call answer |
| `webrtcIceCandidate()` | `webrtc.ice_candidate` | WebRTC | ICE candidate exchange |
| `webrtcHangup()` | `webrtc.hangup` | WebRTC | End WebRTC call |
| `agentList()` | `agent.list` | Agent | List all running agents |
| `agentStatus()` | `agent.status` | Agent | Get agent status |
| `agentGetStatus()` | `agent.get_status` | Agent | Get detailed agent status |
| `agentStatusHistory()` | `agent.status_history` | Agent | Get status history |
| `agentStop()` | `agent.stop` | Agent | Stop a running agent |
| `subscribeToAgentStatus()` | `agent.status` | Agent | Subscribe to status flow |
| `subscribeToAllAgentStatuses()` | `agent.status` | Agent | Subscribe to all agents |
| `getKeystoreStatus()` | `keystore.status` | Keystore | Get keystore status |
| `generateUnsealChallenge()` | `keystore.unseal_challenge` | Keystore | Generate unseal challenge |
| `respondToUnseal()` | `keystore.unseal_respond` | Keystore | Respond to unseal |
| `extendSession()` | `keystore.extend_session` | Keystore | Extend keystore session |
| `subscribeToKeystoreState()` | `keystore.status` | Keystore | Subscribe to keystore state |
| `resolveBlocker()` | `resolve_blocker` | Workflow | Resolve a workflow blocker with user input |

#### ⚠️ Deprecated Methods (Use MatrixClient)

| Kotlin Method | RPC Method | Replacement |
|---------------|------------|-------------|
| `matrixLogin()` | `matrix.login` | `MatrixClient.login()` |
| `matrixSync()` | `matrix.sync` | `MatrixClient.startSync()` |
| `matrixSend()` | `matrix.send` | `MatrixClient.sendTextMessage()` |
| `matrixRefreshToken()` | `matrix.refresh_token` | SDK handles automatically |
| `matrixCreateRoom()` | `matrix.create_room` | `MatrixClient.createRoom()` |
| `matrixJoinRoom()` | `matrix.join_room` | `MatrixClient.joinRoom()` |
| `matrixLeaveRoom()` | `matrix.leave_room` | `MatrixClient.leaveRoom()` |
| `matrixInviteUser()` | `matrix.invite_user` | `MatrixClient.inviteUser()` |
| `matrixSendTyping()` | `matrix.send_typing` | `MatrixClient.sendTyping()` |
| `matrixSendReadReceipt()` | `matrix.send_read_receipt` | `MatrixClient.sendReadReceipt()` |

### 15.13 Key Implementation Classes

| Class | Package | Purpose |
|-------|---------|---------|
| **`MatrixClient`** | `platform.matrix` | **Primary**: Matrix protocol operations (53 methods) |
| **`MatrixSyncManager`** | `platform.matrix` | **Primary**: Matrix /sync long-poll for real-time events |
| `MatrixSessionStorage` | `platform.matrix` | Encrypted session persistence |
| `ControlPlaneStore` | `data.store` | ArmorClaw event processing |
| `BridgeAdminClient` | `platform.bridge` | Admin-only subset of BridgeRpcClient (31 methods) |
| `BridgeRpcClient` | `platform.bridge` | Full RPC interface (includes deprecated methods) |
| `BridgeRpcClientImpl` | `platform.bridge` | RPC implementation with retry logic |
| `BridgeRepository` | `platform.bridge` | Integration layer (domain ↔ bridge) |
| `RealTimeEventStore` | `data.store` | Event distribution to UI |
| `CheckFeatureUseCase` | `domain.usecase` | Feature availability with caching |
| `SetupService` | `domain.service` | First-time setup flow |
| `UnifiedMessage` | `domain.model` | Message model for all types (Regular, Agent, System, Command) |
| `AppNavigation` | `androidApp` | All navigation routes (59 routes) |
| `MatrixClient` | `platform.matrix` | Direct Matrix API |
| `BridgeConfig` | `platform.bridge` | Configuration |
| `BridgeWebSocketClient` | `platform.bridge` | WebSocket (stub - not used) |
| **`BridgeApi`** | `app.armorclaw.network` | Bridge RPC methods (resolveBlocker, hardening, etc.) |
| **`WorkflowTimelineState`** | `app.armorclaw.ui.components` | Timeline aggregate state (events, progress, isRunning) |
| **`WorkflowEvent`** | `app.armorclaw.ui.components` | Single timeline event (seq, type, name, tsMs, detail, durationMs) |
| **`BlockerInfo`** | `app.armorclaw.ui.components` | Blocker metadata (blockerType, message, field, workflowId, stepId) |
| **`WorkflowStatus`** | `app.armorclaw.ui.components` | Enum matching Go WorkflowStatus (IDLE/RUNNING/BLOCKED/COMPLETED/FAILED/CANCELLED) |

### 15.14 Error Handling & Fallbacks

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    ERROR HANDLING & FALLBACKS                                    │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  CONNECTION ERRORS                                                               │
│  ───────────────────                                                             │
│                                                                                  │
│  Scenario: Primary bridge server unreachable                                     │
│                                                                                  │
│  1. ArmorChat detects connection failure                                         │
│  2. Shows "Connecting..." indicator                                              │
│  3. Attempts reconnection with exponential backoff:                              │
│     • Attempt 1: Wait 1s                                                         │
│     • Attempt 2: Wait 2s                                                         │
│     • Attempt 3: Wait 4s                                                         │
│     • ... up to 30s max                                                          │
│  4. After 3 failed attempts, try fallback servers:                               │
│     • https://bridge-backup.armorclaw.app                                        │
│     • https://bridge-eu.armorclaw.app                                            │
│  5. If all fail, show "Offline Mode" banner                                      │
│  6. Queue operations in offline queue                                            │
│  7. Sync when connection restored                                                │
│                                                                                  │
│  RPC ERRORS                                                                      │
│  ──────────                                                                      │
│                                                                                  │
│  JSON-RPC Error Codes:                                                           │
│  • -32700: Parse error (invalid JSON)                                            │
│  • -32600: Invalid Request                                                       │
│  • -32601: Method not found                                                      │
│  • -32602: Invalid params                                                        │
│  • -32603: Internal error                                                        │
│  • -32001: Session expired                                                       │
│  • -32002: Authentication failed                                                 │
│  • -32003: Network error                                                         │
│                                                                                  │
│  Handling:                                                                       │
│  • Session expired → Re-authenticate                                             │
│  • Network error → Queue for retry                                               │
│  • Invalid params → Show user error                                              │
│  • Method not found → Log and degrade gracefully                                 │
│                                                                                  │
│  MESSAGE SEND FAILURES                                                           │
│  ─────────────────────────                                                       │
│                                                                                  │
│  1. Message queued locally with status FAILED                                    │
│  2. Show retry button in UI                                                      │
│  3. User can tap to retry                                                        │
│  4. Or auto-retry when connection restored                                       │
│  5. After max retries (3), mark as permanently failed                            │
│  6. User can delete or retry manually                                            │
│                                                                                  │
│  FILE UPLOAD FAILURES                                                            │
│  ────────────────────────────                                                    │
│                                                                                  │
│  1. Upload paused, shows "Retry" button                                          │
│  2. Supports resume for large files                                              │
│  3. Chunk-based upload with progress tracking                                    │
│  4. Failed chunks retried individually                                           │
│  5. After complete failure, show "Upload failed" with retry option               │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.15 Configuration Classes

```kotlin
// Setup Configuration (Shared)
data class SetupConfig(
    val homeserver: String = "",
    val bridgeUrl: String? = null,        // Bridge RPC URL: /api (NOT /rpc!)
    val serverVersion: String? = null,
    val supportsE2EE: Boolean = true,
    val supportsRecovery: Boolean = true,
    val detectedRegion: String? = null,
    val isDemo: Boolean = false,
    val configSource: ConfigSource = ConfigSource.DEFAULT,
    val wsUrl: String? = null,            // WebSocket URL
    val pushGateway: String? = null,      // Push gateway URL
    val serverName: String? = null,
    val expiresAt: Long? = null
) {
    companion object {
        fun createDefault(): SetupConfig = SetupConfig(
            homeserver = "https://matrix.armorclaw.app",
            bridgeUrl = "https://bridge.armorclaw.app/api",  // ⚠️ /api NOT /rpc
            wsUrl = "wss://bridge.armorclaw.app/ws",
            pushGateway = "https://bridge.armorclaw.app/_matrix/push/v1/notify",
            serverName = "ArmorClaw",
            configSource = ConfigSource.DEFAULT
        )

        fun createDebug(): SetupConfig = SetupConfig(
            homeserver = "http://10.0.2.2:8008",
            bridgeUrl = "http://10.0.2.2:8080/api",  // ⚠️ /api NOT /rpc
            wsUrl = "ws://10.0.2.2:8080/ws",
            pushGateway = "http://10.0.2.2:8080/_matrix/push/v1/notify",
            serverName = "Debug Server",
            configSource = ConfigSource.DEFAULT,
            isDemo = true
        )
    }
}

// Bridge Configuration (Legacy - kept for compatibility)
data class BridgeConfig(
    val baseUrl: String,                    // Bridge server URL (no /api suffix)
    val homeserverUrl: String,              // Matrix homeserver URL
    val wsUrl: String? = null,
    val timeoutMs: Long = 30000,
    val enableCertificatePinning: Boolean = true,
    val certificatePins: List<String> = emptyList(),
    val retryCount: Int = 3,
    val retryDelayMs: Long = 1000,
    val useDirectMatrixSync: Boolean = true,
    val environment: Environment = Environment.PRODUCTION,
    val serverName: String? = null
) {
    companion object {
        val PRODUCTION = BridgeConfig(
            baseUrl = "https://bridge.armorclaw.app",
            homeserverUrl = "https://matrix.armorclaw.app",
            wsUrl = null,
            enableCertificatePinning = true,
            useDirectMatrixSync = true,
            environment = Environment.PRODUCTION,
            serverName = "ArmorClaw"
        )

        val DEVELOPMENT = BridgeConfig(
            baseUrl = "http://10.0.2.2:8080",
            homeserverUrl = "http://10.0.2.2:8008",
            wsUrl = null,
            enableCertificatePinning = false,
            retryCount = 1,
            useDirectMatrixSync = true,
            environment = Environment.DEVELOPMENT,
            serverName = "Development Server"
        )

        val STAGING = BridgeConfig(
            baseUrl = "https://bridge-staging.armorclaw.app",
            homeserverUrl = "https://matrix-staging.armorclaw.app",
            wsUrl = null,
            enableCertificatePinning = false,
            useDirectMatrixSync = true,
            environment = Environment.STAGING,
            serverName = "Staging Server"
        )

        val DEMO = BridgeConfig(
            baseUrl = "https://bridge-demo.armorclaw.app",
            homeserverUrl = "https://matrix-demo.armorclaw.app",
            wsUrl = null,
            enableCertificatePinning = false,
            useDirectMatrixSync = true,
            environment = Environment.STAGING,
            serverName = "Demo Server"
        )
    }
}
```

### 15.16 URL Reference Table

| Service | Production URL | Development URL |
|---------|---------------|-----------------|
| Matrix Homeserver | `https://matrix.armorclaw.app` | `http://10.0.2.2:8008` |
| Bridge RPC | `https://bridge.armorclaw.app/api` | `http://10.0.2.2:8080/api` |
| Bridge WebSocket | `wss://bridge.armorclaw.app/ws` | `ws://10.0.2.2:8080/ws` |
| Push Gateway | `https://bridge.armorclaw.app/_matrix/push/v1/notify` | `http://10.0.2.2:8080/_matrix/push/v1/notify` |
| Well-Known | `https://armorclaw.app/.well-known/matrix/client` | N/A |
| QR Config | `https://bridge.armorclaw.app/qr/config` | `http://10.0.2.2:8080/qr/config` |
| Discovery | `https://bridge.armorclaw.app/discover` | `http://10.0.2.2:8080/discover` |

### 15.17 Android Emulator Localhost

When developing with Android emulator, use these addresses:

| Service | Address | Notes |
|---------|---------|-------|
| Bridge RPC (local) | `http://10.0.2.2:8080/api` | Emulator's localhost mapping |
| Bridge WebSocket (local) | `ws://10.0.2.2:8080/ws` | Real-time events |
| Matrix Server (local) | `http://10.0.2.2:8008` | Matrix API |
| Matrix Media (local) | `http://10.0.2.2:8008/_matrix/media/` | Media uploads |
| Discovery (local) | `http://10.0.2.2:8080/discover` | Server discovery |
| QR Config (local) | `http://10.0.2.2:8080/qr/config` | QR generation |

### 15.18 Security During Communication

| Security Layer | Implementation |
|----------------|----------------|
| Transport | TLS 1.3 with certificate pinning |
| Authentication | Bearer token + session ID |
| Message Encryption | E2EE via Matrix (Megolm/vodozemac) |
| Session Encryption | AES-256 encrypted session storage (SQLCipher) |
| Request Signing | Correlation IDs + trace IDs for tracing |
| Local Storage | SQLCipher with biometric-protected key |
| QR Config Signing | HMAC signature from Bridge |

### 15.19 Communication Summary

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    COMMUNICATION QUICK REFERENCE                                 │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │  WHAT ARMORCHAT USES FOR EACH OPERATION                                  │    │
│  ├─────────────────────────────────────────────────────────────────────────┤    │
│  │                                                                          │    │
│  │  Operation                    │ Channel        │ Endpoint               │    │
│  │  ─────────────────────────────┼────────────────┼─────────────────────── │    │
│  │  Discovery                    │ Multiple       │ See 15.1               │    │
│  │  Login (Auth)                 │ Matrix SDK     │ /_matrix/client/login  │    │
│  │  Send Message                 │ Matrix SDK     │ /rooms/.../send (E2E)  │    │
│  │  Receive Message              │ Matrix /sync   │ /_matrix/client/sync   │    │
│  │  Create/Join/Leave Room       │ Matrix SDK     │ /_matrix/client/...    │    │
│  │  Typing/Read Receipts         │ Matrix SDK     │ /_matrix/client/...    │    │
│  │  Upload File/Media            │ Matrix Media   │ /_matrix/media/upload  │    │
│  │  Real-time Events             │ Matrix /sync   │ Long-poll (30s)        │    │
│  │  WebRTC Call Signaling        │ Matrix Events  │ m.call.* in timeline   │    │
│  │  ─────────────────────────────┼────────────────┼─────────────────────── │    │
│  │  Start/Stop Bridge            │ RPC            │ bridge.start/stop      │    │
│  │  Platform Connect             │ RPC            │ platform.connect       │    │
│  │  Push Notifications           │ RPC            │ push.register_token    │    │
│  │  Recovery                     │ RPC            │ recovery.*             │    │
│  │  WebRTC Signaling             │ RPC            │ webrtc.*               │    │
  │  │  Agent Status                 │ RPC            │ agent.status           │    │
  │  │  Keystore Mgmt                │ RPC            │ keystore.*             │    │
  │  │  Blocker Resolution           │ RPC            │ resolve_blocker        │    │
  │  │  Workflow Events              │ Matrix /sync   │ workflow.* events      │    │
│  │                                                                          │    │
│  │  ✅ PRIMARY: Matrix Protocol (E2E Encrypted)                             │    │
│  │  🔧 ADMIN:   Bridge RPC /api (JSON-RPC 2.0)                              │    │
│  │                                                                          │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 15.20 Complete Data Flow Example

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    MESSAGE SENDING FLOW (End-to-End)                             │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  USER SENDS MESSAGE IN ARMORCHAT                                                 │
│                                                                                  │
│  1. USER TYPES MESSAGE                                                           │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  ChatScreen.kt                                                       │    │
│     │  onSendMessage(text: String)                                         │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  ChatViewModel.onAction(ChatAction.SendMessage)                      │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Create local message (status: SENDING)                              │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  2. MESSAGE ENCRYPTION (E2EE)                                                    │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  MatrixClient.sendMessage()                                          │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Encrypt message with Megolm (room key)                              │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Create m.room.encrypted event                                       │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  3. SEND TO MATRIX SERVER                                                        │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  POST https://matrix.armorclaw.app/_matrix/client/v3/rooms/.../send  │    │
│     │  Body: { type: "m.room.encrypted", content: { ... } }                │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Response: { event_id: "$abc123" }                                   │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  4. RECIPIENT RECEIVES MESSAGE                                                   │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  GET https://matrix.armorclaw.app/_matrix/client/v3/sync             │    │
│     │  (Long-poll, recipient's ArmorChat app)                              │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Response includes m.room.encrypted event                            │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  MatrixSyncManager processes event                                   │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Decrypt message with room key (Megolm)                              │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Display decrypted message in ChatScreen                             │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  ⚠️  IMPORTANT: Bridge RPC is NOT used for message sending/receiving            │
│     Messages go directly through Matrix protocol (E2E encrypted)                │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

### 15.21 Workflow UI Components (v3)

Three Jetpack Compose components render agent workflow state in ArmorChat. They consume state from Matrix `/sync` events via ViewModel/StateFlow and do not manage network connections.

#### WorkflowTimeline

Scrollable vertical timeline showing agent step progress with icons, progress bars, and durations.

```kotlin
// Data models
data class WorkflowEvent(
    val seq: Int,           // Sequential step number (1-based)
    val type: String,       // Event type — matches Go bridge stepIcon() values
    val name: String,       // Human-readable step name
    val tsMs: Long,         // Epoch millis when the event occurred
    val detail: String = "",// Optional detail line (e.g. file path, command)
    val durationMs: Long? = null // Optional duration in milliseconds
)

data class WorkflowTimelineState(
    val events: List<WorkflowEvent>,
    val progress: Float,     // 0.0 – 1.0
    val isRunning: Boolean,
    val workflowName: String = ""
)

// Composable
@Composable
fun WorkflowTimeline(
    state: WorkflowTimelineState,
    modifier: Modifier = Modifier
)
```

**Event icon mapping** (must stay in sync with Go `stepIcon()` in `notifications.go`):

| Type | Icon | Go Constant |
|------|------|-------------|
| `step` | 🔹 | `WorkflowEventStepProgress` |
| `file_read` | 📄 | — |
| `file_write` | ✏️ | — |
| `file_delete` | 🗑️ | — |
| `command_run` | ⌨️ | — |
| `observation` | 💭 | — |
| `blocker` | 🚧 | `WorkflowEventBlockerWarning` |
| `error` | ❌ | `WorkflowEventStepError` |
| `artifact` | 📦 | — |
| `checkpoint` | 🏁 | — |

Layout structure: workflow title, animated progress bar, status line (Live/Complete/Paused), then a `LazyColumn` of timeline rows. Each row shows an icon, vertical track line, name + detail, and an optional duration badge.

#### BlockerResponseDialog

Modal dialog for resolving workflow blockers. Handles input, loading, error, and dismissed states via a state machine.

```kotlin
// Data model
data class BlockerInfo(
    val blockerType: String,  // e.g. "missing_field", "auth_required", "credential_required"
    val message: String,      // What the agent needs
    val suggestion: String,   // Hint for the user
    val field: String,        // Field name (triggers PII masking if sensitive)
    val workflowId: String,
    val stepId: String
)

enum class BlockerDialogState { INPUT, LOADING, ERROR, DISMISSED }

// Composable
@Composable
fun BlockerResponseDialog(
    blocker: BlockerInfo,
    onDismiss: () -> Unit,
    onResolve: (workflowId: String, stepId: String, input: String, note: String) -> Unit,
    dialogState: BlockerDialogState,
    errorMessage: String = ""
)
```

**PII safety:** Fields named `password`, `card`, `key`, `token`, `secret`, `cvv`, `pin`, or `ssn` are automatically masked with `VisualTransformation.Password()`. Input is cleared after sending and never logged.

**State machine flow:**
- `INPUT` — shows text field, optional collapsible note field, Send button (enabled when input is non-blank)
- `LOADING` — shows `CircularProgressIndicator` and "Resolving blocker..." text
- `ERROR` — shows error message and Retry button
- `DISMISSED` — composable returns early (invisible)

#### GovernanceBanner

Compact status banner that maps to Go `WorkflowStatus` values. Shows persistent state indicators at the top of relevant screens.

```kotlin
@Immutable
enum class WorkflowStatus {
    IDLE,        // No banner displayed
    RUNNING,     // Pulsing indicator dot, step counter (e.g. "Step 3 of 7")
    BLOCKED,     // Amber warning, "Action Required", tappable to open BlockerResponseDialog
    COMPLETED,   // Green check, subtle
    FAILED,      // Error container, red ✖
    CANCELLED;   // Grey, ⏹

    companion object {
        fun fromGo(value: String): WorkflowStatus =
            entries.firstOrNull { it.name.equals(value, ignoreCase = true) } ?: IDLE
    }
}

@Composable
fun GovernanceBanner(
    status: WorkflowStatus,
    currentStep: Int = 0,
    totalSteps: Int = 0,
    onBlockedTap: (() -> Unit)? = null
)
```

When `status == BLOCKED` and `onBlockedTap` is provided, the banner is tappable and shows an arrow icon. Tapping it typically opens the `BlockerResponseDialog`.

---

### 15.22 Blocker Resolution Flow (v3)

End-to-end flow from container blocker through Bridge, Matrix, ArmorChat, and back:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    BLOCKER RESOLUTION FLOW (End-to-End)                          │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  1. CONTAINER HITS BLOCKER                                                       │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  Agent container step requires user input (PII, approval, CAPTCHA)  │    │
│     │  Writes event to _events.jsonl with type "blocker"                  │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  2. BRIDGE PROCESSES EVENT                                                       │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  EventReader reads _events.jsonl                                     │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  OrchestratorIntegration converts to WorkflowEvent                   │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  WorkflowEventEmitter.EmitBlockerWarning()                           │    │
│     │  publishes workflow.blocker_warning to Matrix room                   │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  3. ARMORCHAT RECEIVES BLOCKER                                                   │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  MatrixSyncManager receives event via /sync                          │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  GovernanceBanner shows BLOCKED state (amber, tappable)              │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  4. USER RESPONDS                                                                │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  User taps banner → BlockerResponseDialog opens                      │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  User enters response (PII-safe masking for sensitive fields)        │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  BridgeApi.resolveBlocker(workflowId, stepId, input, note)           │    │
│     │  sends JSON-RPC: resolve_blocker to Bridge /api                      │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  5. BRIDGE UNBLOCKS WORKFLOW                                                     │
│     ┌──────────────────────────────────────────────────────────────────────┐    │
│     │  Bridge Orchestrator receives resolve_blocker RPC                    │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Injects user input into container step                              │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  Container retries step, workflow resumes                            │    │
│     │      │                                                               │    │
│     │      ▼                                                               │    │
│     │  WorkflowEventEmitter.EmitProgress() publishes workflow.progress     │    │
│     │  GovernanceBanner transitions to RUNNING                             │    │
│     └──────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Error path:** If `resolveBlocker()` RPC fails, `BlockerResponseDialog` transitions to `ERROR` state showing the error message. The user can retry or dismiss. The workflow remains blocked on the Bridge side until a successful resolution or timeout.

> See `communication-infra.md` for event type definitions and `secretary-workflow.md` for the blocker protocol details.

---

### 15.23 Workflow Event Bridge Synchronization (v3)

How workflow events flow from the container through the Bridge to ArmorChat via Matrix:

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    WORKFLOW EVENT BRIDGE SYNCHRONIZATION                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  Container          Bridge (Go)                    Matrix             ArmorChat  │
│  ─────────          ──────────                    ──────             ──────────  │
│                                                                                  │
│  _events.jsonl  →  EventReader                                                                   │
│  (StepEvent)        │                                                            │
│                     ▼                                                            │
│                  OrchestratorIntegration                                                          │
│                  (StepEvent → WorkflowEvent)                                      │
│                     │                                                            │
│                     ▼                                                            │
│                  WorkflowEventEmitter                                                             │
│                  ┌───────────────────────────────────┐                           │
│                  │ EmitStarted    → workflow.started  │────→ Matrix room ──→ /sync│
│                  │ EmitProgress   → workflow.progress │────→ Matrix room ──→ /sync│
│                  │ EmitBlocked    → workflow.blocked  │────→ Matrix room ──→ /sync│
│                  │ EmitCompleted  → workflow.completed│────→ Matrix room ──→ /sync│
│                  │ EmitFailed     → workflow.failed   │────→ Matrix room ──→ /sync│
│                  │ EmitCancelled  → workflow.cancelled│────→ Matrix room ──→ /sync│
│                  │ StepProgress   → workflow.step_progress  → /sync              │
│                  │ StepError      → workflow.step_error       → /sync            │
│                  │ BlockerWarning → workflow.blocker_warning   → /sync           │
│                  └───────────────────────────────────┘                           │
│                                                  │                               │
│                                                  ▼                               │
│                                           MatrixEventBus.Publish()               │
│                                                  │                               │
│                                                  ▼                               │
│                                           processEvents()                        │
│                                           (rooms/sync.go)                        │
│                                                  │                               │
│                                                  ▼                               │
│                                           Matrix room timeline                   │
│                                                  │                               │
│                                                  ▼                               │
│                                           ArmorChat /sync                        │
│                                           MatrixSyncManager                      │
│                                                  │                               │
│                                                  ▼                               │
│                                           ViewModel → StateFlow                  │
│                                                  │                               │
│                                                  ▼                               │
│                                           WorkflowTimeline / GovernanceBanner    │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**Go-side event types** (defined in `orchestrator_events.go`):

| Constant | Value | Description |
|----------|-------|-------------|
| `WorkflowEventStarted` | `workflow.started` | Workflow began executing |
| `WorkflowEventProgress` | `workflow.progress` | Step progress update |
| `WorkflowEventBlocked` | `workflow.blocked` | Workflow paused, needs user input |
| `WorkflowEventCompleted` | `workflow.completed` | Workflow finished successfully |
| `WorkflowEventFailed` | `workflow.failed` | Workflow errored out |
| `WorkflowEventCancelled` | `workflow.cancelled` | User or system cancelled |
| `WorkflowEventStepProgress` | `workflow.step_progress` | Granular step event from container |
| `WorkflowEventStepError` | `workflow.step_error` | Step-level error from container |
| `WorkflowEventBlockerWarning` | `workflow.blocker_warning` | Blocker needs user resolution |

**Kotlin-side WorkflowStatus mapping:**

| Go Status | Kotlin Enum | UI Behavior |
|-----------|-------------|-------------|
| `pending` / `running` | `RUNNING` | Pulsing indicator, step counter |
| `blocked` | `BLOCKED` | Amber banner, tappable for resolution |
| `completed` | `COMPLETED` | Green check, subtle |
| `failed` | `FAILED` | Red error banner |
| `cancelled` | `CANCELLED` | Grey stopped banner |

> See `communication-infra.md` for the full MatrixEventBus and EventReader architecture.

---

## 16. Quick Reference

### File Locations

| What | Location |
|------|----------|
| Navigation routes | `androidApp/.../navigation/AppNavigation.kt` |
| Deep link handler | `androidApp/.../navigation/DeepLinkHandler.kt` |
| ViewModels | `androidApp/.../viewmodels/` |
| Screens | `androidApp/.../screens/` |
| Domain models | `shared/.../domain/model/` |
| Repository interfaces | `shared/.../domain/repository/` |
| Setup service | `shared/.../platform/bridge/SetupService.kt` |
| Setup config | `shared/.../platform/bridge/SetupService.kt` |
| Matrix client | `shared/.../platform/matrix/MatrixClient.kt` |
| Matrix sync manager | `shared/.../platform/matrix/MatrixSyncManager.kt` |
| Bridge client | `shared/.../platform/bridge/BridgeRpcClient.kt` |
| Bridge RPC impl | `shared/.../platform/bridge/BridgeRpcClientImpl.kt` |
| Bridge repository | `shared/.../platform/bridge/BridgeRepository.kt` |
| Real-time event store | `shared/.../data/store/RealTimeEventStore.kt` |
| Check feature use case | `shared/.../domain/usecase/CheckFeatureUseCase.kt` |
| Control plane | `shared/.../data/store/ControlPlaneStore.kt` |
| Unified message | `shared/.../domain/model/UnifiedMessage.kt` |
| RPC models | `shared/.../platform/bridge/RpcModels.kt` |
| Android Manifest | `androidApp/src/main/AndroidManifest.xml` |
| Workflow timeline | `androidApp/.../ui/components/WorkflowTimeline.kt` |
| Blocker dialog | `androidApp/.../ui/components/BlockerResponseDialog.kt` |
| Governance banner | `androidApp/.../ui/components/GovernanceBanner.kt` |
| Bridge API | `androidApp/.../network/BridgeApi.kt` |

### Key Classes

| Class | Package | Purpose |
|-------|---------|---------|
| **`SetupService`** | `platform.bridge` | **Discovery**: Server discovery, signed config, setup flow |
| **`SetupConfig`** | `platform.bridge` | Server configuration (homeserver, bridgeUrl, etc.) |
| **`ConfigSource`** | `platform.bridge` | How config was obtained (SIGNED_URL, MDNS, etc.) |
| **`MatrixClient`** | `platform.matrix` | **Primary**: Matrix protocol operations (53 methods) |
| **`MatrixSyncManager`** | `platform.matrix` | **Primary**: Matrix /sync long-poll for real-time events |
| `MatrixSessionStorage` | `platform.matrix` | Encrypted session persistence |
| `ControlPlaneStore` | `data.store` | ArmorClaw event processing |
| `BridgeAdminClient` | `platform.bridge` | Admin-only subset of BridgeRpcClient (31 methods) |
| `BridgeRpcClient` | `platform.bridge` | Full RPC interface (74 methods, includes deprecated Matrix ops) |
| `BridgeRpcClientImpl` | `platform.bridge` | RPC implementation with retry logic |
| `BridgeRepository` | `platform.bridge` | Integration layer (domain ↔ bridge) |
| `RealTimeEventStore` | `data.store` | Event distribution to UI |
| `CheckFeatureUseCase` | `domain.usecase` | Feature availability with caching |
| `UnifiedMessage` | `domain.model` | Message model (Regular, Agent, System, Command) |
| `AppNavigation` | `androidApp` | All navigation routes (59 routes) |
| `DeepLinkHandler` | `androidApp` | Deep link parsing and routing |
| `ChatViewModel` | `androidApp` | Chat screen state (unified messages) |
| `HomeViewModel` | `androidApp` | Home screen state (rooms + workflows) |
| **`WorkflowTimeline`** | `app.armorclaw.ui.components` | Vertical timeline composable for agent steps |
| **`BlockerResponseDialog`** | `app.armorclaw.ui.components` | Modal dialog for HITL blocker resolution |
| **`GovernanceBanner`** | `app.armorclaw.ui.components` | Status banner (RUNNING/BLOCKED/COMPLETED/FAILED) |
| **`BridgeApi`** | `app.armorclaw.network` | Bridge RPC client with `resolveBlocker()` |

### Related Documentation

| Document | Location | Purpose |
|----------|----------|---------|
| **Matrix Migration** | `doc/MATRIX_MIGRATION.md` | **Primary**: RPC → Matrix migration guide (✅ COMPLETE) |
| Communication Fixes | `doc/COMMUNICATION_FIXES.md` | Details on RPC/Matrix sync implementation |
| Architecture | `doc/ARCHITECTURE.md` | System architecture overview |
| Security | `doc/SECURITY.md` | Security implementation details |
| API Reference | `doc/API.md` | Public API documentation |

### Bridge Server Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api` | POST | JSON-RPC 2.0 API (all RPC methods) |
| `/ws` | WebSocket | Real-time events (optional, Matrix sync preferred) |
| `/health` | GET | Health check |
| `/discover` | GET | Server discovery info |
| `/.well-known/matrix/client` | GET | Matrix well-known discovery |
| `/qr/config` | GET | Generate signed config QR |
| `/qr/image` | GET | QR code image |
| `/_matrix/push/v1/notify` | POST | Push gateway |

### Deep Link Formats

| Scheme | Format | Purpose |
|--------|--------|---------|
| Config | `armorclaw://config?d=<base64>` | Signed server configuration (handled by `SignedConfigParser`) |
| Room | `armorclaw://room/<room_id>` | Navigate to agent room (v0.7.0) |
| Email Approval | `armorclaw://email/approve/<approval_id>` | HITL email approval (v0.7.0) |
| Setup | `armorclaw://setup?token=xxx&server=xxx` | Device setup with token |
| Invite | `armorclaw://invite?code=xxx` | Room/server invite |
| Bond | `armorclaw://bond?token=xxx` | Device bonding (admin pairing) |
| Web Config | `https://armorclaw.app/config?d=<base64>` | Web-based config |
| Web Invite | `https://armorclaw.app/invite/<code>` | Web-based invite |

### Deep Link Handler (v0.7.0)

> **Added in v0.7.0** — Deep link routing for notification taps (room navigation and email HITL approval).

`DeepLinkHandler.kt` (`app.armorclaw.navigation`) provides `resolveRoute(uri: Uri): Route?` which maps incoming `armorclaw://` URIs to navigation routes:

- `armorclaw://room/{id}` → `Route.Room(roomId)`
- `armorclaw://email/approve/{id}` → `Route.EmailApproval(approvalId)`
- `armorclaw://config?d=...` → `null` (delegates to `SignedConfigParser`)
- Unknown hosts → `null`

**Cold-start handling:** `MainActivity.kt` uses a `mutableStateOf<Uri?>` field. On launch, `LaunchedEffect` reads `intent` for the initial deep link. `onNewIntent()` handles warm-resume notification taps. Both paths call `DeepLinkHandler.resolveRoute()` and navigate via `NavController`.

**Intent filters** in `AndroidManifest.xml` declare `armorclaw://room` and `armorclaw://email/approve` with `launchMode="singleTop"` to ensure `onNewIntent()` fires for subsequent taps.

### ConfigManager (v0.7.0)

> **Added in v0.7.0** — Server configuration persistence via `EncryptedSharedPreferences`.

`ConfigManager.kt` (`app.armorclaw.config`) persists `ServerConfig` (server URL, Matrix homeserver, device ID) to `EncryptedSharedPreferences`. This replaces the in-memory-only `BridgeRepository` credentials.

- `saveConfig(config: ServerConfig)` — persists to encrypted storage
- `loadConfig(): ServerConfig?` — reads from encrypted storage
- `clearConfig()` — removes stored config
- `isConfigExpired(): Boolean` — checks 24-hour TTL

> **Security note:** Auth tokens (accessToken) are NOT persisted. They remain in memory only, pending v0.8.0 security review.

### Config Expiration (v0.7.0)

`ArmorClawNavHost.kt` checks config expiration at startup. If `ConfigManager.isConfigExpired()` returns true, the app redirects to the bonding/provisioning screen for re-provisioning. This prevents stale configurations from causing cryptic connection failures.

---

*This document provides a comprehensive overview of the ArmorChat project. For implementation details, see the source code and additional documentation in the `doc/` directory.*

**Document Version:** 3.8
**Last Updated:** 2026-04-18
**Matrix Migration:** ✅ COMPLETE
**Discovery System:** ✅ ENHANCED
**Governor Strategy:** ✅ COMPLETE
**Workflow UI (v3):** ✅ COMPLETE (WorkflowTimeline, BlockerResponseDialog, GovernanceBanner, resolveBlocker RPC)
**Deep Links (v0.7.0):** ✅ COMPLETE (DeepLinkHandler, cold-start/warm-resume, ConfigManager, config expiration)

---

## 17. Governor Strategy

> **Status:** ✅ COMPLETE (All 4 Phases)
> **Version:** 1.0.0
> **Completion Date:** 2026-02-22

The Governor Strategy transforms ArmorChat from a messaging client to a **secure, authoritative controller** for ArmorClaw agents. The app becomes the "Governor" - authorizing, monitoring, and revoking agent actions.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARMORCLAW GOVERNOR STACK                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  PHASE 4: COMMERCIAL POLISH                                     │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ ArmorClawTheme │ StatusIcons │ RiskLevelBadge │ StatusBar  ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│  PHASE 3: AUDIT & TRANSPARENCY                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ ArmorTerminal │ TaskReceipt │ RevocationPanel │ PiiAccess  ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│  PHASE 2: GOVERNOR UI                                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ CommandBlock │ CapabilityRibbon │ HITLAuthorization         ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│  PHASE 1: COLD VAULT                                            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ SQLCipher │ KeystoreManager │ VaultRepository │ ShadowMap  ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 1: Cold Vault (Identity & Security) ✅

**Goal:** Encrypt all PII at rest using SQLCipher via SQLDelight 2.0.0

| Component | Status | Location |
|-----------|--------|----------|
| SQLCipher Integration | ✅ Complete | `androidApp/.../security/` |
| KeystoreManager | ✅ Complete | Hardware-backed key management |
| SqlCipherProvider | ✅ Complete | Encrypted database factory |
| VaultRepository | ✅ Complete | PII CRUD operations |
| PiiRegistry | ✅ Complete | PII key registry |
| ShadowMap | ✅ Complete | Placeholder mapping |
| AgentRequestInterceptor | ✅ Complete | Outbound request middleware |

**Shadow Mapping:**
- Placeholder format: `{{VAULT:field_name:hash}}`
- Replaces raw PII before agent transmission
- Hash verification for integrity

### Phase 2: Governor UI ✅

**Goal:** Technical "Command Block" UI and capability visualization

| Component | Status | Location |
|-----------|--------|----------|
| CommandBlock | ✅ Complete | `armorclaw-ui/.../governor/` |
| CommandStatusBadge | ✅ Complete | Status indicators |
| CapabilityRibbon | ✅ Complete | Horizontal capability display |
| CapabilityChip | ✅ Complete | Individual capabilities |
| CapabilityIndicator | ✅ Complete | Context-aware indicators |
| HITLAuthorizationCard | ✅ Complete | Human-in-the-loop approvals |
| SimpleApprovalDialog | ✅ Complete | Quick approve/reject |

### Phase 3: Audit & Transparency ✅

**Goal:** Immutable audit trail and one-click revocation

| Component | Status | Location |
|-----------|--------|----------|
| TaskReceipt | ✅ Complete | `armorclaw-ui/.../audit/` |
| ActionType | ✅ Complete | Action type enumeration |
| TaskStatus | ✅ Complete | Status tracking |
| CapabilityUsage | ✅ Complete | Usage tracking |
| PiiAccess | ✅ Complete | PII access logging |
| RevocationRecord | ✅ Complete | Revocation history |
| AuditSession | ✅ Complete | Session audit |
| RiskSummary | ✅ Complete | Risk assessment |
| ArmorTerminal | ✅ Complete | Real-time activity log |
| RevocationPanel | ✅ Complete | One-click revocation |

### Phase 4: Commercial Polish ✅

**Goal:** Consistent branding and status indicators

| Component | Status | Location |
|-----------|--------|----------|
| ArmorClawTheme | ✅ Complete | `armorclaw-ui/.../theme/` |
| SecurityStatusIcon | ✅ Complete | Security state indicator |
| AgentStatusIcon | ✅ Complete | Agent state indicator |
| CapabilityStatusIcon | ✅ Complete | Capability state |
| NetworkStatusIcon | ✅ Complete | Connection state |
| RiskLevelBadge | ✅ Complete | Risk level display |
| ActivityPulseIndicator | ✅ Complete | Animated activity |
| StatusBar | ✅ Complete | Combined status bar |

### Files Created (19 Total)

```
armorclaw-ui/src/commonMain/kotlin/
├── components/
│   ├── vault/
│   │   ├── VaultModels.kt           ✅ UI data models
│   │   ├── VaultPulseIndicator.kt   ✅ Pulse animation
│   │   └── VaultKeyPanel.kt         ✅ Sidebar panel
│   ├── governor/
│   │   ├── GovernorModels.kt        ✅ Governor data models
│   │   ├── CommandBlock.kt          ✅ Action card
│   │   ├── CapabilityRibbon.kt      ✅ Capability display
│   │   ├── HITLAuthorization.kt     ✅ Hold-to-approve
│   │   └── GovernorComponents.kt    ✅ Barrel export
│   └── audit/
│       ├── AuditModels.kt           ✅ Audit data models
│       ├── ArmorTerminal.kt         ✅ Activity log
│       └── RevocationControls.kt    ✅ Revocation UI
├── theme/
│   ├── ArmorClawTheme.kt            ✅ Brand theme
│   └── StatusIcons.kt               ✅ Status icons

androidApp/src/main/kotlin/com/armorclaw/app/security/
├── KeystoreManager.kt               ✅ Keystore management
├── SqlCipherProvider.kt             ✅ Database encryption
└── VaultRepository.kt               ✅ PII storage

shared/src/commonMain/kotlin/domain/security/
├── PiiRegistry.kt                   ✅ PII key registry
├── ShadowMap.kt                     ✅ Placeholder mapping
└── AgentRequestInterceptor.kt       ✅ Request middleware
```

### Key Features Delivered

| Feature | Description |
|---------|-------------|
| 🔐 Cold Vault | Hardware-backed encryption (Android Keystore + AES-256) |
| 🔒 SQLCipher | Encrypted database with 256-bit passphrase |
| 👥 Shadow Mapping | PII replaced with `{{VAULT:field:hash}}` placeholders |
| 📋 Command Blocks | Technical UI replacing chat bubbles |
| 🎛️ Capability Ribbon | Quick visibility of active capabilities |
| ✋ HITL Authorization | Hold-to-approve for sensitive actions |
| 📊 ArmorTerminal | Real-time activity log (terminal-style) |
| ⏪ One-Click Revocation | Immediate capability/session undo |
| 🎨 Brand Theme | Consistent Teal #14F0C8 / Navy #0A1428 |
| 🚧 Blocker Protocol | GovernanceBanner + BlockerResponseDialog for workflow blocker HITL (v3) |

### Security Model

| Layer | Implementation |
|-------|----------------|
| Transport | TLS 1.3 + Certificate Pinning |
| Database | SQLCipher 256-bit encryption |
| Keys | Android Keystore (hardware-backed) |
| PII | Shadow placeholders in transit |
| Audit | Immutable TaskReceipt records |

---

## 18. Deployment Status & Issues Fixed

> **Status:** 🟢 READY FOR DEPLOYMENT - All critical issues fixed
> **Last Reviewed:** 2026-02-20
> **Reviewer:** System Review

### ✅ Issues Fixed

| Issue | Severity | Status | File Changed |
|-------|----------|--------|--------------|
| Domain inconsistency | 🔴 Critical | ✅ FIXED | Multiple files |
| WebSocket URL mismatch | 🔴 Critical | ✅ FIXED | `BridgeWebSocketClient.kt` |
| AppModules Matrix URL | 🔴 Critical | ✅ FIXED | `AppModules.kt` |
| **API endpoint mismatch** | 🔴 **CRITICAL** | ✅ FIXED | `BridgeRpcClientImpl.kt` |
| ExternalLinkHandler URLs | 🟡 Medium | ✅ FIXED | `ExternalLinkHandler.kt` |
| ErrorReportingService URL | 🟡 Medium | ✅ FIXED | `ErrorReportingService.kt` |
| ReleaseConfig Matrix URL | 🟡 Medium | ✅ FIXED | `ReleaseConfig.kt` |
| TermsOfService email | 🟡 Medium | ✅ FIXED | `TermsOfServiceScreen.kt` |
| Test assertions | 🟡 Medium | ✅ FIXED | Test files |

### ⚠️ Critical Bug Fixed: API Endpoint Mismatch

**Problem:** ArmorChat client was posting to `/rpc` but Bridge server listens at `/api`

**Fixed:** Changed client to use `/api` endpoint:
- `BridgeRpcClientImpl.kt`: Changed `httpClient.post("/rpc")` → `httpClient.post("/api")`

### 📋 Domain Configuration (All Consistent)

All URLs now use `*.armorclaw.app`:

| Component | URL | File |
|-----------|-----|------|
| Bridge RPC | `https://bridge.armorclaw.app` | `RpcModels.kt` |
| Matrix Homeserver | `https://matrix.armorclaw.app` | `RpcModels.kt` |
| WebSocket | `wss://bridge.armorclaw.app` | `BridgeWebSocketClient.kt` |
| Demo Server | `https://demo.armorclaw.app` | `SetupService.kt` |
| Demo Bridge | `https://bridge-demo.armorclaw.app` | `SetupService.kt` |
| Fallback Bridge 1 | `https://bridge.armorclaw.app` | `SetupService.kt` |
| Fallback Bridge 2 | `https://bridge-backup.armorclaw.app` | `SetupService.kt` |
| Fallback Bridge 3 | `https://bridge-eu.armorclaw.app` | `SetupService.kt` |
| Website | `https://armorclaw.app` | `ExternalLinkHandler.kt` |
| Privacy Policy | `https://armorclaw.app/privacy` | `ExternalLinkHandler.kt` |
| Terms of Service | `https://armorclaw.app/terms` | `ExternalLinkHandler.kt` |
| Error Reporting API | `https://api.armorclaw.app` | `ErrorReportingService.kt` |
| Support Email | `support@armorclaw.app` | `ExternalLinkHandler.kt` |
| Legal Email | `legal@armorclaw.app` | `TermsOfServiceScreen.kt` |

### 🔗 Server Endpoints Required

| Service | Endpoint | Protocol | Purpose |
|---------|----------|----------|---------|
| Matrix Homeserver | `matrix.armorclaw.app` | HTTPS | User accounts, messaging |
| Bridge RPC | `bridge.armorclaw.app/api` | HTTPS | Admin operations |
| Bridge Health | `bridge.armorclaw.app/health` | HTTPS | Health check |
| Bridge Discovery | `bridge.armorclaw.app/discover` | HTTPS | Server discovery |
| Error Reporting | `api.armorclaw.app` | HTTPS | Error telemetry |

### Required Server URLs for First Deployment

| Service | URL | Notes |
|---------|-----|-------|
| Matrix Homeserver | `https://matrix.armorclaw.app` | Client connects directly |
| Bridge RPC | `https://bridge.armorclaw.app` | Admin operations via JSON-RPC |
| Bridge WebSocket | ❌ NOT USED | Use Matrix /sync instead |

### Files Updated for Deployment

| File | Change |
|------|--------|
| `RpcModels.kt` | ✅ Already correct - uses `.armorclaw.app` |
| `BridgeWebSocketClient.kt` | ✅ Fixed - changed `.com` to `.app` |
| `AppModules.kt` | ✅ Fixed - changed `.com` to `.app` |
| `BridgeRpcClientTest.kt` | ✅ Fixed - updated assertions |
| `BridgeWebSocketClientTest.kt` | ✅ Fixed - updated assertions |

### Configuration Checklist

- [x] All URLs use consistent domain (`.armorclaw.app`)
- [ ] Certificate pins configured for production
- [x] Demo server URLs defined
- [x] Fallback server URLs defined
- [ ] Matrix homeserver is accessible at `https://matrix.armorclaw.app`
- [ ] Bridge RPC endpoint is accessible at `https://bridge.armorclaw.app`
- [ ] SSL/TLS certificates valid for both domains

### ✅ Pre-Deployment Validation Steps

1. **DNS Verification:**
   ```bash
   # Verify DNS records exist
   nslookup matrix.armorclaw.app
   nslookup bridge.armorclaw.app
   nslookup api.armorclaw.app
   ```

2. **HTTPS Connectivity:**
   ```bash
   # Test Matrix homeserver
   curl -v https://matrix.armorclaw.app/_matrix/client/versions

   # Test Bridge health endpoint
   curl -v https://bridge.armorclaw.app/health

   # Test Bridge RPC endpoint (NOTE: uses /api, not /rpc!)
   curl -X POST https://bridge.armorclaw.app/api \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"system.health","id":"1"}'
   ```

3. **Certificate Validation:**
   ```bash
   openssl s_client -connect matrix.armorclaw.app:443
   openssl s_client -connect bridge.armorclaw.app:443
   openssl s_client -connect api.armorclaw.app:443
   ```

4. **Certificate Pinning (if enabled):**
   ```bash
   # Get certificate SHA-256 pins
   openssl s_client -connect bridge.armorclaw.app:443 | openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64
   ```

### 📱 Onboarding Flow Summary

```
Splash Screen
     │
     ├── hasLegacyBridgeSession? ──YES──▶ MigrationScreen (v2.5 → v3.0)
     │
     ├── hasValidSession && !isBackupComplete? ──YES──▶ KeyBackupSetupScreen
     │
     ├── hasValidSession && isLoggedIn? ──YES──▶ HomeScreen
     │
     ├── hasCompletedOnboarding && !isLoggedIn? ──YES──▶ LoginScreen
     │
     └── First time user ──▶ ConnectServerScreen (QR-first onboarding)
                                    │
                                    ├── User enters server URL / scans QR
                                    ├── App calls SetupService.startSetup()
                                    ├── Bridge health check at /health
                                    ├── If fails: Try fallback servers
                                    │
                                    ▼
                              User enters credentials
                                    │
                                    ▼
                              SetupService.connectWithCredentials()
                                    │
                                    ├── Start Bridge session
                                    ├── Login to Matrix
                                    ├── Get user role from server
                                    │
                                    ▼
                              PermissionsScreen
                                    │
                                    ▼
                              CompletionScreen
                                    │
                                    ▼
                              KeyBackupSetupScreen
                                    │
                                    ▼
                              HomeScreen
```

### 🔧 Server Discovery Mechanism

1. **User enters homeserver URL** (e.g., `https://matrix.example.com`)
2. **App derives bridge URL**:
   - `https://matrix.example.com` → `https://bridge.example.com`
3. **Health check** on bridge at `/health` endpoint
4. **If primary fails**, try fallback servers:
   - `https://bridge.armorclaw.app`
   - `https://bridge-backup.armorclaw.app`
   - `https://bridge-eu.armorclaw.app`

### 🌐 Communication Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        ArmorChat Client                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌───────────────┐    ┌──────────────┐    ┌────────────────┐   │
│  │  Matrix SDK   │    │ BridgeRpc    │    │ SetupService   │   │
│  │  (Messages)   │    │ Client       │    │ (Onboarding)   │   │
│  └───────┬───────┘    └──────┬───────┘    └────────┬───────┘   │
│          │                   │                     │           │
└──────────┼───────────────────┼─────────────────────┼───────────┘
           │                   │                     │
           ▼                   ▼                     ▼
    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
    │   Matrix     │    │   Bridge     │    │   Bridge     │
    │ Homeserver   │    │   /api       │    │   /health    │
    │ /_matrix/... │    │  (JSON-RPC)  │    │              │
    └──────────────┘    └──────────────┘    └──────────────┘
         matrix.            bridge.           bridge.
         armorclaw.app      armorclaw.app     armorclaw.app
```

### 📡 Bridge Server Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api` | POST | JSON-RPC 2.0 API |
| `/health` | GET | Health check (public) |
| `/discover` | GET | Server discovery info |
| `/fingerprint` | GET | Certificate fingerprint |
| `/ws` | WebSocket | Real-time events (not used by ArmorChat) |

**Note:** ArmorChat uses Matrix `/sync` for real-time events, not Bridge WebSocket.

---

## 19. ArmorChat ↔ ArmorClaw Compatibility

### ✅ RPC Methods Compatibility Matrix

| Category | Client Method | Server Handler | Status |
|----------|---------------|----------------|--------|
| **Bridge Lifecycle** | `bridge.start` | ✅ `handleBridgeStart` | Compatible |
| | `bridge.stop` | ✅ `handleBridgeStop` | Compatible |
| | `bridge.status` | ✅ `handleBridgeStatus` | Compatible |
| | `bridge.health` | ✅ `handleBridgeHealth` | Compatible |
| **Matrix Operations** | `matrix.login` | ⚠️ `handleMatrixLogin` | Deprecated — use `MatrixClient.login()` |
| | `matrix.send` | ⚠️ `handleMatrixSend` | Deprecated — use `MatrixClient.sendTextMessage()` |
| | `matrix.create_room` | ⚠️ `handleMatrixCreateRoom` | Deprecated — use `MatrixClient.createRoom()` |
| | `matrix.join_room` | ⚠️ `handleMatrixJoinRoom` | Deprecated — use `MatrixClient.joinRoom()` |
| | Other matrix.* | ⚠️ Various | Deprecated — use `MatrixClient` directly |
| **Platform** | `platform.connect` | ✅ `handlePlatformConnect` | Compatible |
| | `platform.disconnect` | ✅ `handlePlatformDisconnect` | Compatible |
| | `platform.list` | ✅ `handlePlatformList` | Compatible |
| | `platform.status` | ✅ `handlePlatformStatus` | Compatible |
| | `platform.test` | ✅ `handlePlatformTest` | Compatible |
| **Push Notifications** | `push.register_token` | ✅ `handlePushRegisterToken` | Compatible |
| | `push.unregister_token` | ✅ `handlePushUnregisterToken` | Compatible |
| | `push.update_settings` | ✅ `handlePushUpdateSettings` | Compatible |
| **Recovery** | `recovery.*` | ✅ All recovery methods | Compatible |
| **WebRTC** | `webrtc.offer` | ✅ `handleWebrtcOffer` | Compatible |
| | `webrtc.answer` | ✅ `handleWebrtcAnswer` | Compatible |
| | `webrtc.ice_candidate` | ✅ `handleWebrtcIceCandidate` | Compatible |
| | `webrtc.hangup` | ✅ `handleWebrtcHangup` | Compatible |
| **Agent Management** | `agent.status` | ✅ `handleAgentStatus` | Compatible |
| | `agent.status_history` | ✅ `handleAgentStatusHistory` | Compatible |
| **Keystore** | `keystore.status` | ✅ `handleKeystoreStatus` | Compatible |
| | `keystore.unseal_challenge` | ✅ `handleUnsealChallenge` | Compatible |
| | `keystore.unseal_respond` | ✅ `handleUnsealRespond` | Compatible |
| | `keystore.extend_session` | ✅ `handleExtendSession` | Compatible |
| **Blocker Resolution** | `resolve_blocker` | ✅ Orchestrator unblocks workflow | Compatible (v3) |

### 📡 Communication Protocol Stack

| Layer | Protocol | Implementation |
|-------|----------|----------------|
| Application | JSON-RPC 2.0 | HTTP POST to `/api` |
| Transport | HTTPS | TLS 1.3 |
| Real-time Events | Matrix /sync | Long-poll (30s timeout) |
| E2EE | AES-256-GCM | ECDH key exchange + VaultCryptoManager |
| Authentication | Bearer Token | Matrix access token |

### ⚠️ Known Limitations

1. **WebSocket Not Used:** Bridge WebSocket exists at `/ws` but ArmorChat uses Matrix `/sync` for real-time events
2. **Matrix SDK Required:** For messaging functionality, use `MatrixClient` interface directly
3. **Admin Operations via RPC:** Bridge RPC is for admin/container operations, not messaging
4. **First-User Admin:** Admin privileges are server-authoritative (no client-side race conditions)

---

## 20. Deployment Order

### 1. Infrastructure Setup

```bash
# DNS Records (create these first)
matrix.armorclaw.app     A     <matrix-server-ip>
bridge.armorclaw.app     A     <bridge-server-ip>
api.armorclaw.app        A     <api-server-ip>
```

### 2. Deploy Matrix Homeserver

```bash
# Option A: Conduit (recommended for simplicity)
docker run -d \
  --name conduit \
  -v /opt/conduit:/data \
  -p 6167:6167 \
  girlbossceo/conduit:latest

# Option B: Synapse (full-featured)
docker run -d \
  --name synapse \
  -v /opt/synapse:/data \
  -p 8008:8008 \
  matrixdotorg/synapse:latest
```

### 3. Deploy ArmorClaw Bridge

> **Note:** The ArmorClaw Bridge is a separate server-side component not included in this repository.
> Deploy from the ArmorClaw server repository following its deployment documentation.
> Expected endpoints: `/api` (JSON-RPC 2.0), `/health` (GET), `/ws` (WebSocket).

### 4. Configure Reverse Proxy (nginx/Caddy)

```nginx
# /etc/nginx/sites-enabled/bridge.armorclaw.app
server {
    listen 443 ssl http2;
    server_name bridge.armorclaw.app;

    ssl_certificate /etc/letsencrypt/live/bridge.armorclaw.app/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/bridge.armorclaw.app/privkey.pem;

    location /api {
        proxy_pass http://127.0.0.1:8443;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /health {
        proxy_pass http://127.0.0.1:8443;
    }

    location /discover {
        proxy_pass http://127.0.0.1:8443;
    }
}
```

### 5. SSL Certificates

```bash
# Get Let's Encrypt certificates
certbot certonly --nginx -d matrix.armorclaw.app
certbot certonly --nginx -d bridge.armorclaw.app
certbot certonly --nginx -d api.armorclaw.app
```

### 6. Build ArmorChat APK

```bash
cd ArmorChat

# Update certificate pins in CertificatePinner.kt if using pinning

# Build release APK
./gradlew assembleRelease

# APK will be at:
# androidApp/build/outputs/apk/release/androidApp-release.apk
```

---

## 21. Server Discovery Implementation

> **Status:** 🚧 PROPOSED — Not yet implemented
> The code below represents proposed changes for the Bridge server (Go) and a new `DiscoveryService.kt`.
> Currently, discovery is handled by `SetupService.kt` which supports: manual entry, URL derivation,
> demo server, localhost, and fallback servers. Well-known, mDNS, and QR scanning in `DiscoveryService`
> are aspirational.

### Discovery Methods Available

| Method | Status | Description |
|--------|--------|-------------|
| Manual Entry | ✅ Implemented (SetupService) | User enters homeserver URL |
| URL Derivation | ✅ Implemented (SetupService) | Auto-derive bridge URL from matrix URL |
| Demo Server | ✅ Implemented (SetupService) | Quick option for demo |
| Localhost | ✅ Implemented (SetupService) | Quick option for development |
| Fallback Servers | ✅ Implemented (SetupService) | Auto-retry with backup servers |
| Deep Links | ✅ Implemented (SetupService/DeepLinkHandler) | `armorclaw://` scheme |
| **Matrix Well-Known** | 🚧 PROPOSED | Standard Matrix discovery (not yet implemented) |
| **QR Code Scanning** | 🚧 PROPOSED | Scan QR from Bridge server (not yet implemented) |
| **mDNS/Bonjour** | 🚧 PROPOSED | Discover local servers (not yet implemented) |

### Discovery Flow

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      ArmorChat Server Discovery                          │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. QR CODE SCANNING (Preferred for local setup)                         │
│     ┌─────────┐      ┌─────────────┐      ┌──────────────────┐          │
│     │ Bridge  │ ───▶ │ QR Code     │ ───▶ │ ArmorChat        │          │
│     │ Server  │      │ (config)    │      │ scans & connects │          │
│     └─────────┘      └─────────────┘      └──────────────────┘          │
│                                                                          │
│  2. mDNS DISCOVERY (For local network)                                   │
│     ┌─────────┐      ┌─────────────┐      ┌──────────────────┐          │
│     │ Bridge  │ ───▶ │ _armorclaw  │ ───▶ │ ArmorChat        │          │
│     │ Server  │      │ ._tcp.local │      │ discovers        │          │
│     └─────────┘      └─────────────┘      └──────────────────┘          │
│                                                                          │
│  3. WELL-KNOWN DISCOVERY (Standard Matrix)                               │
│     ┌────────────────┐      ┌──────────────────┐      ┌────────────┐    │
│     │ armorclaw.app  │ ───▶ │ /.well-known/    │ ───▶ │ ArmorChat  │    │
│     │ (server name)  │      │ matrix/client    │      │ connects   │    │
│     └────────────────┘      └──────────────────┘      └────────────┘    │
│                                                                          │
│  4. MANUAL ENTRY (Fallback)                                              │
│     ┌────────────┐      ┌──────────────────────────────────────┐        │
│     │ User types │ ───▶ │ https://matrix.armorclaw.app         │        │
│     │ server URL │      │ → derives: https://bridge.armorclaw.app/api │ │
│     └────────────┘      └──────────────────────────────────────┘        │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### ArmorClaw Bridge Server - Required Changes (PROPOSED)

> **Note:** These are proposed server-side changes. They are NOT in the current codebase.

#### 1. Add Well-Known Endpoint

Create `bridge/pkg/http/wellknown.go`:

```go
package http

import (
    "encoding/json"
    "net/http"
)

// WellKnownResponse is the Matrix well-known discovery response
type WellKnownResponse struct {
    Homeserver     HomeserverInfo     `json:"m.homeserver"`
    IdentityServer *IdentityServerInfo `json:"m.identity_server,omitempty"`
    Bridge         *BridgeInfo         `json:"com.armorclaw.bridge,omitempty"`
}

type HomeserverInfo struct {
    BaseURL string `json:"base_url"`
}

type IdentityServerInfo struct {
    BaseURL string `json:"base_url"`
}

type BridgeInfo struct {
    BaseURL      string `json:"base_url"`
    APIEndpoint  string `json:"api_endpoint"`
    PushGateway  string `json:"push_gateway,omitempty"`
}

// handleWellKnown serves the Matrix well-known discovery document
// This should be served at /.well-known/matrix/client
func (s *Server) handleWellKnown(w http.ResponseWriter, r *http.Request) {
    response := WellKnownResponse{
        Homeserver: HomeserverInfo{
            BaseURL: "https://matrix.armorclaw.app",
        },
        IdentityServer: &IdentityServerInfo{
            BaseURL: "https://matrix.armorclaw.app",
        },
        Bridge: &BridgeInfo{
            BaseURL:     "https://bridge.armorclaw.app",
            APIEndpoint: "https://bridge.armorclaw.app/api",
            PushGateway: "https://bridge.armorclaw.app/_matrix/push/v1/notify",
        },
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    json.NewEncoder(w).Encode(response)
}
```

Add to `bridge/pkg/http/server.go`:
```go
// In Start() function, add:
mux.HandleFunc("/.well-known/matrix/client", s.handleWellKnown)
```

#### 2. Enhanced Discovery Endpoint

Update `bridge/pkg/http/server.go` handleDiscover:

```go
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
    hostname, _ := os.Hostname()
    ips, _ := getLocalIPs()
    fingerprint, _ := s.GetCertificateFingerprint()

    response := map[string]interface{}{
        "name":        hostname,
        "hostname":    s.config.Hostname,
        "port":        s.config.Port,
        "ips":         ips,
        "version":     "1.0.0",
        "fingerprint": fingerprint,
        "endpoints": map[string]string{
            "matrix": "https://matrix.armorclaw.app",
            "rpc":    "https://bridge.armorclaw.app/api",
            "ws":     "wss://bridge.armorclaw.app/ws",
            "push":   "https://bridge.armorclaw.app/_matrix/push/v1/notify",
        },
        "features": map[string]bool{
            "e2ee":  true,
            "voice": true,
            "push":  true,
        },
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    json.NewEncoder(w).Encode(response)
}
```

#### 3. QR Code Endpoints

The Bridge already has QR code support in `bridge/pkg/qr/public.go`. Add HTTP endpoints:

```go
// In bridge/pkg/http/server.go, add:
func (s *Server) handleQRConfig(w http.ResponseWriter, r *http.Request) {
    result, err := s.qrManager.GenerateConfigQR(24 * time.Hour)
    if err != nil {
        http.Error(w, "Failed to generate QR", 500)
        return
    }

    // Return JSON with QR image as base64
    response := map[string]interface{}{
        "url":        result.URL,
        "deep_link":  result.DeepLink,
        "qr_image":   base64.StdEncoding.EncodeToString(result.QRImage),
        "expires_at": result.ExpiresAt,
        "config":     result.Config,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *Server) handleQRImage(w http.ResponseWriter, r *http.Request) {
    result, err := s.qrManager.GenerateConfigQR(24 * time.Hour)
    if err != nil {
        http.Error(w, "Failed to generate QR", 500)
        return
    }

    w.Header().Set("Content-Type", "image/png")
    w.Write(result.QRImage)
}
```

Add routes in Start():
```go
mux.HandleFunc("/qr/config", s.handleQRConfig)
mux.HandleFunc("/qr/image", s.handleQRImage)
```

### ArmorChat - Required Changes (PROPOSED)

> **Note:** `DiscoveryService.kt` does NOT exist yet. Discovery is currently handled by `SetupService.kt`.

#### 1. Create Discovery Service

Create `shared/src/commonMain/kotlin/platform/discovery/DiscoveryService.kt`:

```kotlin
package com.armorclaw.shared.platform.discovery

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.platform.bridge.DetectServerInfo
import com.armorclaw.shared.platform.logging.repositoryLogger
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.request.*
import io.ktor.http.*
import kotlinx.coroutines.flow.*
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * Server Discovery Service
 *
 * Provides multiple methods for discovering ArmorClaw servers:
 * 1. Matrix Well-Known discovery (standard)
 * 2. mDNS/Bonjour discovery (local network)
 * 3. QR code scanning (direct configuration)
 * 4. Manual entry with URL derivation
 */
class DiscoveryService(
    private val httpClient: HttpClient = HttpClient(),
    private val mdnsDiscovery: MDNSDiscovery = MDNSDiscoveryStub(),
    private val qrScanner: QRScanner = QRScannerStub()
) {
    private val logger = repositoryLogger("DiscoveryService")
    private val json = Json { ignoreUnknownKeys = true }

    private val _discoveredServers = MutableStateFlow<List<DiscoveredServer>>(emptyList())
    val discoveredServers: StateFlow<List<DiscoveredServer>> = _discoveredServers.asStateFlow()

    private val _isScanning = MutableStateFlow(false)
    val isScanning: StateFlow<Boolean> = _isScanning.asStateFlow()

    /**
     * Discover server using Matrix well-known
     * Standard Matrix discovery: GET https://server/.well-known/matrix/client
     */
    suspend fun discoverWellKnown(
        serverName: String,
        context: OperationContext? = null
    ): DiscoveryResult {
        return try {
            val url = "https://$serverName/.well-known/matrix/client"
            logger.logInfo("Discovering via well-known", mapOf("url" to url))

            val response = httpClient.get(url) {
                header("Accept", "application/json")
            }

            if (!response.status.isSuccess()) {
                return DiscoveryResult.Error("Server returned ${response.status}")
            }

            val wellKnown = json.decodeFromString<WellKnownResponse>(response.body())

            DiscoveryResult.Success(
                DiscoveredServer(
                    homeserver = wellKnown.homeserver.baseUrl,
                    bridgeUrl = wellKnown.bridge?.apiEndpoint
                        ?: deriveBridgeUrl(wellKnown.homeserver.baseUrl),
                    serverName = serverName,
                    discoveryMethod = DiscoveryMethod.WELL_KNOWN,
                    pushGateway = wellKnown.bridge?.pushGateway
                )
            )
        } catch (e: Exception) {
            logger.logError("Well-known discovery failed", e)
            DiscoveryResult.Error(e.message ?: "Discovery failed")
        }
    }

    /**
     * Discover local servers via mDNS
     * Returns immediately with cached results, updates via flow
     */
    suspend fun discoverLocal(): List<DiscoveredServer> {
        _isScanning.value = true
        return try {
            val servers = mdnsDiscovery.discover()
            _discoveredServers.value = servers
            servers
        } catch (e: Exception) {
            logger.logError("mDNS discovery failed", e)
            emptyList()
        } finally {
            _isScanning.value = false
        }
    }

    /**
     * Parse QR code data
     * Supports: armorclaw://config?d=<base64>
     */
    fun parseQRCode(qrData: String): DiscoveryResult {
        return try {
            when {
                qrData.startsWith("armorclaw://config?") -> parseConfigDeepLink(qrData)
                qrData.startsWith("armorclaw://setup?") -> parseSetupDeepLink(qrData)
                qrData.startsWith("armorclaw://invite?") -> parseInviteDeepLink(qrData)
                qrData.startsWith("https://armorclaw.app/") -> parseWebLink(qrData)
                else -> DiscoveryResult.Error("Unknown QR code format")
            }
        } catch (e: Exception) {
            logger.logError("QR parse failed", e)
            DiscoveryResult.Error("Invalid QR code: ${e.message}")
        }
    }

    /**
     * Start QR scanning session
     */
    suspend fun startQRScanning(): QRScanResult {
        return qrScanner.scan()
    }

    /**
     * Derive bridge URL from homeserver URL
     */
    fun deriveBridgeUrl(homeserver: String): String {
        val url = homeserver.removeSuffix("/")
        return when {
            url.contains("://matrix.") -> url.replace("://matrix.", "://bridge.")
            url.contains("://chat.") -> url.replace("://chat.", "://bridge.")
            else -> url.replace("://", "://bridge.")
        }
    }

    private fun parseConfigDeepLink(uri: String): DiscoveryResult {
        // armorclaw://config?d=<base64-encoded-config>
        val params = uri.substringAfter("?").parseQueryParams()
        val data = params["d"] ?: return DiscoveryResult.Error("Missing config data")

        val configJson = java.util.Base64.getUrlDecoder().decode(data).decodeToString()
        val config = json.decodeFromString<QRConfigPayload>(configJson)

        // Validate signature (optional, for security)
        if (config.expiresAt < System.currentTimeMillis() / 1000) {
            return DiscoveryResult.Error("Config expired")
        }

        return DiscoveryResult.Success(
            DiscoveredServer(
                homeserver = config.matrixHomeserver,
                bridgeUrl = config.rpcUrl,
                serverName = config.serverName,
                discoveryMethod = DiscoveryMethod.QR_CODE,
                pushGateway = config.pushGateway
            )
        )
    }

    private fun parseSetupDeepLink(uri: String): DiscoveryResult {
        // armorclaw://setup?token=xxx&server=xxx
        val params = uri.substringAfter("?").parseQueryParams()
        val token = params["token"] ?: return DiscoveryResult.Error("Missing token")
        val server = params["server"] ?: return DiscoveryResult.Error("Missing server")

        return DiscoveryResult.Success(
            DiscoveredServer(
                homeserver = server,
                bridgeUrl = deriveBridgeUrl(server),
                serverName = server,
                discoveryMethod = DiscoveryMethod.QR_CODE,
                setupToken = token
            )
        )
    }

    private fun parseInviteDeepLink(uri: String): DiscoveryResult {
        // armorclaw://invite?code=xxx
        val params = uri.substringAfter("?").parseQueryParams()
        val code = params["code"] ?: return DiscoveryResult.Error("Missing invite code")

        // Invite codes contain server info
        return DiscoveryResult.InviteCode(code)
    }

    private fun parseWebLink(uri: String): DiscoveryResult {
        // https://armorclaw.app/config?d=xxx
        // https://armorclaw.app/invite/xxx
        // https://armorclaw.app/setup?token=xxx

        return when {
            uri.contains("/config?") -> parseConfigDeepLink(
                uri.replace("https://armorclaw.app/", "armorclaw://")
            )
            uri.contains("/invite/") -> {
                val code = uri.substringAfter("/invite/").substringBefore("?")
                DiscoveryResult.InviteCode(code)
            }
            uri.contains("/setup?") -> parseSetupDeepLink(
                uri.replace("https://armorclaw.app/", "armorclaw://")
            )
            else -> DiscoveryResult.Error("Unknown web link format")
        }
    }

    private fun String.parseQueryParams(): Map<String, String> {
        return split("&")
            .mapNotNull { param ->
                val parts = param.split("=", limit = 2)
                if (parts.size == 2) parts[0] to parts[1] else null
            }
            .toMap()
    }
}

// ============================================================================
// Models
// ============================================================================

@Serializable
data class WellKnownResponse(
    @SerialName("m.homeserver")
    val homeserver: HomeserverInfo,
    @SerialName("m.identity_server")
    val identityServer: HomeserverInfo? = null,
    @SerialName("com.armorclaw.bridge")
    val bridge: BridgeWellKnownInfo? = null
)

@Serializable
data class HomeserverInfo(
    @SerialName("base_url")
    val baseUrl: String
)

@Serializable
data class BridgeWellKnownInfo(
    @SerialName("base_url")
    val baseUrl: String,
    @SerialName("api_endpoint")
    val apiEndpoint: String? = null,
    @SerialName("push_gateway")
    val pushGateway: String? = null
)

@Serializable
data class QRConfigPayload(
    val version: Int,
    @SerialName("matrix_homeserver")
    val matrixHomeserver: String,
    @SerialName("rpc_url")
    val rpcUrl: String,
    @SerialName("ws_url")
    val wsUrl: String? = null,
    @SerialName("push_gateway")
    val pushGateway: String? = null,
    @SerialName("server_name")
    val serverName: String,
    val region: String? = null,
    @SerialName("expires_at")
    val expiresAt: Long,
    val signature: String? = null
)

data class DiscoveredServer(
    val homeserver: String,
    val bridgeUrl: String,
    val serverName: String,
    val discoveryMethod: DiscoveryMethod,
    val pushGateway: String? = null,
    val setupToken: String? = null,
    val host: String? = null,        // For mDNS
    val port: Int? = null,           // For mDNS
    val fingerprint: String? = null  // For TLS verification
)

enum class DiscoveryMethod {
    WELL_KNOWN,
    MDNS,
    QR_CODE,
    MANUAL,
    DEEP_LINK
}

sealed class DiscoveryResult {
    data class Success(val server: DiscoveredServer) : DiscoveryResult()
    data class Error(val message: String) : DiscoveryResult()
    data class InviteCode(val code: String) : DiscoveryResult()

    val isSuccess: Boolean get() = this is Success
}

sealed class QRScanResult {
    data class Success(val data: String) : QRScanResult()
    data class Cancelled(val reason: String? = null) : QRScanResult()
    data class Error(val message: String) : QRScanResult()
}

// ============================================================================
// Platform Interfaces (expect/actual pattern)
// ============================================================================

expect class MDNSDiscovery() {
    suspend fun discover(): List<DiscoveredServer>
    fun startScanning()
    fun stopScanning()
}

expect class QRScanner() {
    suspend fun scan(): QRScanResult
    fun isAvailable(): Boolean
}

// Stub implementations for common code
class MDNSDiscoveryStub : MDNSDiscovery {
    override suspend fun discover(): List<DiscoveredServer> = emptyList()
    override fun startScanning() {}
    override fun stopScanning() {}
}

class QRScannerStub : QRScanner {
    override suspend fun scan(): QRScanResult = QRScanResult.Error("QR scanning not available")
    override fun isAvailable(): Boolean = false
}
```

#### 2. Android mDNS Implementation

Create `shared/src/androidMain/kotlin/platform/discovery/MDNSDiscovery.android.kt`:

```kotlin
package com.armorclaw.shared.platform.discovery

import android.content.Context
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

actual class MDNSDiscovery actual constructor(
    private val context: Context? = null
) {
    private val SERVICE_TYPE = "_armorclaw._tcp"

    private var nsdManager: NsdManager? = null
    private var isScanning = false

    init {
        context?.let {
            nsdManager = it.getSystemService(Context.NSD_SERVICE) as NsdManager
        }
    }

    actual suspend fun discover(): List<DiscoveredServer> = suspendCancellableCoroutine { cont ->
        if (nsdManager == null) {
            cont.resume(emptyList())
            return@suspendCancellableCoroutine
        }

        val discoveredServers = mutableListOf<DiscoveredServer>()
        var resolveCount = 0
        var expectedResolves = 0

        val discoveryListener = object : NsdManager.DiscoveryListener {
            override fun onDiscoveryStarted(regType: String) {
                isScanning = true
            }

            override fun onServiceFound(service: NsdServiceInfo) {
                if (service.serviceType == SERVICE_TYPE) {
                    expectedResolves++
                    nsdManager?.resolveService(service, object : NsdManager.ResolveListener {
                        override fun onResolveFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {
                            resolveCount++
                            checkComplete()
                        }

                        override fun onServiceResolved(serviceInfo: NsdServiceInfo) {
                            resolveCount++
                            discoveredServers.add(
                                DiscoveredServer(
                                    homeserver = "https://${serviceInfo.host.hostAddress ?: serviceInfo.serviceName}",
                                    bridgeUrl = "https://${serviceInfo.host.hostAddress ?: serviceInfo.serviceName}:${serviceInfo.port}",
                                    serverName = serviceInfo.serviceName,
                                    discoveryMethod = DiscoveryMethod.MDNS,
                                    host = serviceInfo.host.hostAddress,
                                    port = serviceInfo.port
                                )
                            )
                            checkComplete()
                        }

                        fun checkComplete() {
                            // Wait a bit for all resolves to complete
                            if (resolveCount >= expectedResolves && expectedResolves > 0) {
                                nsdManager?.stopServiceDiscovery(this@discoveryListener)
                            }
                        }
                    })
                }
            }

            override fun onServiceLost(service: NsdServiceInfo) {}
            override fun onDiscoveryStopped(serviceType: String) {
                isScanning = false
                cont.resume(discoveredServers)
            }
            override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
                nsdManager?.stopServiceDiscovery(this)
                cont.resume(emptyList())
            }
            override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {
                cont.resume(discoveredServers)
            }
        }

        nsdManager?.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, discoveryListener)

        cont.invokeOnCancellation {
            nsdManager?.stopServiceDiscovery(discoveryListener)
        }
    }

    actual fun startScanning() {
        // Continuous scanning not implemented yet
    }

    actual fun stopScanning() {
        isScanning = false
    }
}
```

#### 3. Android QR Scanner Implementation

Create `shared/src/androidMain/kotlin/platform/discovery/QRScanner.android.kt`:

```kotlin
package com.armorclaw.shared.platform.discovery

import android.app.Activity
import android.content.Context
import android.content.Intent
import androidx.activity.result.contract.ActivityResultContracts
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanIntentResult
import com.journeyapps.barcodescanner.ScanOptions
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume

actual class QRScanner actual constructor(
    private val context: Context? = null
) {
    actual suspend fun scan(): QRScanResult = suspendCancellableCoroutine { cont ->
        // This needs to be called from an Activity with lifecycle awareness
        // For now, return a result that indicates the caller should use the Activity-based API
        cont.resume(QRScanResult.Error("Use scanWithActivity() for Android"))
    }

    /**
     * Activity-based QR scanning for Android
     * Call this from your Activity/Fragment
     */
    fun scanWithActivity(
        activity: Activity,
        callback: (QRScanResult) -> Unit
    ) {
        val options = ScanOptions()
            .setDesiredBarcodeFormats(ScanOptions.QR_CODE)
            .setPrompt("Scan ArmorClaw QR Code")
            .setCameraId(0)
            .setBeepEnabled(false)
            .setBarcodeImageEnabled(true)

        // This requires integration with ActivityResultLauncher
        // See: https://github.com/journeyapps/zxing-android-embedded
        callback(QRScanResult.Error("Integrate with ActivityResultLauncher"))
    }

    actual fun isAvailable(): Boolean = context != null
}
```

#### 4. Update SetupService

Add to `SetupService.kt`:

```kotlin
class SetupService(
    private val rpcClient: BridgeRpcClient,
    private val wsClient: BridgeWebSocketClient,
    private val discoveryService: DiscoveryService = DiscoveryService()
) {
    // ... existing code ...

    /**
     * Start setup with automatic discovery
     * Tries well-known first, then falls back to URL derivation
     */
    suspend fun startSetupWithDiscovery(
        serverNameOrUrl: String,
        context: OperationContext? = null
    ): SetupResult {
        val ctx = context ?: OperationContext.create()

        _setupState.value = SetupState.DetectingServer

        // Try well-known discovery first
        if (!serverNameOrUrl.startsWith("http")) {
            val wellKnownResult = discoveryService.discoverWellKnown(serverNameOrUrl, ctx)
            if (wellKnownResult is DiscoveryResult.Success) {
                val server = wellKnownResult.server
                return startSetup(server.homeserver, server.bridgeUrl, ctx)
            }
        }

        // Fall back to manual entry
        val homeserver = if (serverNameOrUrl.startsWith("http")) {
            serverNameOrUrl
        } else {
            "https://matrix.$serverNameOrUrl"
        }

        return startSetup(homeserver, null, ctx)
    }

    /**
     * Setup from QR code
     */
    suspend fun setupFromQRCode(
        qrData: String,
        context: OperationContext? = null
    ): SetupResult {
        val ctx = context ?: OperationContext.create()

        val result = discoveryService.parseQRCode(qrData)
        return when (result) {
            is DiscoveryResult.Success -> {
                val server = result.server
                startSetup(server.homeserver, server.bridgeUrl, ctx)
            }
            is DiscoveryResult.Error -> {
                SetupResult.Error(result.message)
            }
            is DiscoveryResult.InviteCode -> {
                // Handle invite code
                parseInviteAndSetup(result.code, ctx)
            }
        }
    }

    /**
     * Discover local servers via mDNS
     */
    suspend fun discoverLocalServers(): List<DiscoveredServer> {
        return discoveryService.discoverLocal()
    }
}
```

### Add Dependencies

Add to `shared/build.gradle.kts`:
```kotlin
// In androidMain dependencies:
implementation("com.journeyapps:zxing-android-embedded:4.3.0")
```

### Update ConnectServerScreen

Add discovery options to the UI:

```kotlin
// Add to ConnectServerScreen.kt

// Scan QR button
OutlinedButton(
    onClick = { showQRScanner = true },
    modifier = Modifier.fillMaxWidth()
) {
    Icon(Icons.Default.QrCodeScanner, contentDescription = null)
    Spacer(modifier = Modifier.width(8.dp))
    Text("Scan QR Code")
}

// Discover local servers button
OutlinedButton(
    onClick = {
        scope.launch {
            val servers = viewModel.discoverLocalServers()
            if (servers.isNotEmpty()) {
                discoveredServers = servers
            }
        }
    },
    modifier = Modifier.fillMaxWidth()
) {
    Icon(Icons.Default.Wifi, contentDescription = null)
    Spacer(modifier = Modifier.width(8.dp))
    Text("Find Local Servers")
}

// Show discovered servers
if (discoveredServers.isNotEmpty()) {
    Text("Discovered Servers:", style = MaterialTheme.typography.labelMedium)
    discoveredServers.forEach { server ->
        Card(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 4.dp)
                .clickable {
                    homeserver = server.homeserver
                    customBridgeUrl = server.bridgeUrl
                }
        ) {
            ListItem(
                headlineContent = { Text(server.serverName) },
                supportingContent = { Text(server.host ?: server.homeserver) },
                leadingContent = {
                    Icon(
                        when (server.discoveryMethod) {
                            DiscoveryMethod.MDNS -> Icons.Default.Wifi
                            else -> Icons.Default.Dns
                        },
                        contentDescription = null
                    )
                }
            )
        }
    }
}
```

---

## 22. Signed Configuration System Review

### Proposal Analysis

The signed configuration system is a **good addition** but has issues:

| Aspect | Status | Issue |
|--------|--------|-------|
| Domain | ❌ | Uses `.com` instead of `.app` |
| API Path | ❌ | Uses `/rpc` instead of `/api` |
| Architecture | ⚠️ | Creates parallel system instead of integrating |
| Security | ✅ | HMAC signature verification is good |
| UX Flow | ✅ | QR/deep link flow is user-friendly |

### Recommended: Unified Discovery Architecture

Instead of creating a separate system, integrate signed config into `SetupService`:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    SetupService (Single Entry Point)                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  startSetupWithDiscovery(input: String)                                 │
│       │                                                                 │
│       ├─── armorclaw:// or https://armorclaw.app?                       │
│       │    └──▶ parseSignedConfig() → startSetup()                      │
│       │                                                                 │
│       ├─── Domain without protocol? (e.g. "example.com")                │
│       │    └──▶ tryWellKnownDiscovery() → startSetup()                  │
│       │                                                                 │
│       └─── Full URL? (e.g. "https://matrix.example.com")               │
│            └──▶ startSetup() → detectServer()                           │
│                                                                         │
│  startSetup(homeserver, bridgeUrl?)                                     │
│       └──▶ detectServer() → deriveBridgeUrl() → healthCheck()           │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Proposed BuildConfig Fields

> **Note:** These BuildConfig fields do NOT currently exist in `androidApp/build.gradle.kts`.
> Configuration is handled via `BridgeConfig` static presets (PRODUCTION, DEVELOPMENT, STAGING, DEMO)
> and `SetupService` with runtime override via `BridgeConfig.setRuntimeConfig()`.

```kotlin
// PROPOSED BuildConfig fields (not yet implemented)
android {
    defaultConfig {
        // Production defaults (release builds) - USE .app DOMAIN
        buildConfigField("String", "MATRIX_HOMESERVER", "\"https://matrix.armorclaw.app\"")
        buildConfigField("String", "ARMORCLAW_RPC_URL", "\"https://bridge.armorclaw.app/api\"")
        buildConfigField("String", "ARMORCLAW_WS_URL", "\"wss://bridge.armorclaw.app/ws\"")
        buildConfigField("String", "PUSH_GATEWAY", "\"https://bridge.armorclaw.app/_matrix/push/v1/notify\"")
    }

    buildTypes {
        debug {
            // Debug defaults (emulator)
            buildConfigField("String", "MATRIX_HOMESERVER", "\"http://10.0.2.2:8008\"")
            buildConfigField("String", "ARMORCLAW_RPC_URL", "\"http://10.0.2.2:8080/api\"")
            buildConfigField("String", "ARMORCLAW_WS_URL", "\"ws://10.0.2.2:8080/ws\"")
            buildConfigField("String", "PUSH_GATEWAY", "\"http://10.0.2.2:8080/_matrix/push/v1/notify\"")
        }
    }
}
```

### Bridge Server: Corrected QR Config

The Bridge's `pkg/qr/public.go` already has the right structure but needs endpoint fix:

```go
// CORRECTED in GenerateConfigQR
config := &ConfigPayload{
    Version:          1,
    MatrixHomeserver: m.serverURL,
    RpcURL:           m.bridgeURL + "/api",    // FIXED: was /rpc
    WsURL:            m.bridgeURL + "/ws",
    PushGateway:      m.bridgeURL + "/_matrix/push/v1/notify",
    ServerName:       m.serverName,
    ExpiresAt:        time.Now().Add(expiration).Unix(),
}
```

### Integration with SetupService

> **Note:** `parseSignedConfig()` and `startSetupWithDiscovery()` are already implemented
> in `SetupService.kt`. The code below was the original proposal and is superseded by the
> actual implementation which uses `ConfigParseResult`, handles invite links, and stores
> setup tokens for first-boot provisioning.

Original proposal for `SetupService.kt`:

```kotlin
/**
 * Parse signed configuration from QR/deep link
 * Returns server info if valid, error if invalid/expired
 */
suspend fun parseSignedConfig(
    deepLinkOrQR: String,
    context: OperationContext? = null
): SetupResult {
    val ctx = context ?: OperationContext.create()

    return try {
        // Parse the deep link
        val config = when {
            deepLinkOrQR.startsWith("armorclaw://config?") -> {
                parseConfigDeepLink(deepLinkOrQR)
            }
            deepLinkOrQR.startsWith("https://armorclaw.app/config?") -> {
                parseConfigDeepLink(deepLinkOrQR)
            }
            else -> return SetupResult.Error("Invalid config format")
        }

        // Validate signature (if bridge provides public key)
        // For now, trust the config (could add signature verification)

        // Check expiration
        if (config.expiresAt != null && config.expiresAt < Clock.System.now().epochSeconds) {
            return SetupResult.Error(
                "Configuration has expired. Please scan a new QR code.",
                listOf(FallbackOption.CONTACT_SUPPORT)
            )
        }

        // Apply configuration
        _config.value = SetupConfig(
            homeserver = config.matrixHomeserver,
            bridgeUrl = config.rpcUrl,
            serverVersion = "1.0.0",
            supportsE2EE = true,
            supportsRecovery = true,
            configSource = ConfigSource.SIGNED_URL  // was SIGNED_QR in original proposal
        )

        _serverInfo.value = DetectServerInfo(
            homeserver = config.matrixHomeserver,
            bridgeUrl = config.rpcUrl,
            version = "1.0.0",
            supportsE2EE = true,
            supportsRecovery = true,
            region = config.region ?: "us-east"
        )

        _setupState.value = SetupState.ReadyForCredentials

        SetupResult.Success(_serverInfo.value!!)
    } catch (e: Exception) {
        logger.logOperationError("parseSignedConfig", e)
        SetupResult.Error("Invalid configuration: ${e.message}")
    }
}

// Actual ConfigSource enum (in SetupService.kt)
enum class ConfigSource {
    DEFAULT,      // BuildConfig defaults (lowest priority)
    MANUAL,       // User entered manually
    WELL_KNOWN,   // Matrix well-known discovery
    MDNS,         // mDNS/Bonjour discovery
    SIGNED_URL,   // Signed QR code or deep link from Bridge (highest priority)
    CACHED,       // Previously cached configuration
    INVITE        // From invite code
}
```

### Proposed AndroidManifest.xml (Corrected)

```xml
<activity android:name=".MainActivity">
    <!-- armorclaw:// deep links -->
    <intent-filter>
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="armorclaw" android:host="config" />
    </intent-filter>

    <intent-filter>
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="armorclaw" android:host="setup" />
    </intent-filter>

    <intent-filter>
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="armorclaw" android:host="invite" />
    </intent-filter>

    <!-- https:// web links -->
    <intent-filter android:autoVerify="true">
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="https"
              android:host="armorclaw.app"
              android:pathPrefix="/config" />
    </intent-filter>

    <intent-filter android:autoVerify="true">
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="https"
              android:host="armorclaw.app"
              android:pathPrefix="/invite" />
    </intent-filter>

    <intent-filter android:autoVerify="true">
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="https"
              android:host="armorclaw.app"
              android:pathPrefix="/setup" />
    </intent-filter>
</activity>
```

### Complete Discovery Priority Order

```
1. SIGNED QR/DEEP LINK  ← Highest trust, pre-verified by bridge (implemented)
       ↓ (if not available)
2. WELL-KNOWN          ← Standard Matrix discovery (implemented)
       ↓ (if not available)
3. MDNS                ← Local network discovery (ConfigSource defined, not yet wired)
       ↓ (if not available)
4. MANUAL ENTRY        ← User provides URL (implemented)
       ↓
5. FALLBACK SERVERS    ← Pre-configured backup servers
```

### File Changes Summary

| File | Action | Priority |
|------|--------|----------|
| `SetupService.kt` | Add `parseSignedConfig()` | ✅ Done |
| `SetupService.kt` | `ConfigSource` enum | ✅ Done |
| `AndroidManifest.xml` | Add intent filters | ✅ Done |
| `DeepLinkHandler.kt` | Route to SetupService | ✅ Done |
| `build.gradle.kts` | Add BuildConfig URLs (.app, /api) | PROPOSED |
| `ConnectServerScreen.kt` | Add QR scan button | MEDIUM |
| `bridge/pkg/qr/public.go` | Fix `/rpc` → `/api` | ✅ Done |

### Bridge Server Required Changes

1. **Fix QR endpoint path** in `pkg/qr/public.go`:
   - Change `RpcURL: m.bridgeURL + "/rpc"` to `RpcURL: m.bridgeURL + "/api"`
   - ✅ DONE

2. **Add QR HTTP endpoints** in `pkg/http/server.go`:
   - `GET /qr/config` - Returns JSON with QR data
   - `GET /qr/image` - Returns PNG image (or redirect info)
   - ✅ DONE

3. **Add well-known endpoint** for Matrix standard discovery:
   - `GET /.well-known/matrix/client`
   - ✅ DONE

---

## 23. OpenClaw Suggestions

OpenClaw is a self-hosted, open-source version of ArmorClaw. Here are suggestions for making discovery work:

### Discovery Architecture for OpenClaw

```
┌──────────────────────────────────────────────────────────────────────────┐
│                     OpenClaw Discovery Options                            │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. MANUAL ENTRY (Simplest)                                              │
│     - User enters: https://matrix.yourdomain.com                         │
│     - App derives: https://bridge.yourdomain.com/api                     │
│     - Works for any self-hosted deployment                               │
│                                                                          │
│  2. WELL-KNOWN DISCOVERY (Recommended)                                   │
│     - Serve at: https://yourdomain.com/.well-known/matrix/client         │
│     - Include Bridge URLs in response                                    │
│     - Standard Matrix discovery                                          │
│                                                                          │
│  3. mDNS DISCOVERY (Local Network)                                       │
│     - OpenClaw Bridge advertises via mDNS                                │
│     - ArmorChat discovers local servers automatically                     │
│     - Great for home/office deployments                                  │
│                                                                          │
│  4. QR CODE (Self-Hosted)                                                │
│     - OpenClaw generates signed config QR                                │
│     - Users scan QR to auto-configure                                    │
│     - Config can be signed with your own key                             │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### OpenClaw Bridge Server Setup

#### 1. Well-Known Endpoint (nginx/Caddy)

```nginx
# /etc/nginx/sites-enabled/openclaw.conf
server {
    listen 443 ssl;
    server_name yourdomain.com;

    # Well-known discovery
    location /.well-known/matrix/client {
        default_type application/json;
        add_header Access-Control-Allow-Origin *;
        return 200 '{
            "m.homeserver": {"base_url": "https://matrix.yourdomain.com"},
            "com.armorclaw.bridge": {
                "base_url": "https://bridge.yourdomain.com",
                "api_endpoint": "https://bridge.yourdomain.com/api"
            }
        }';
    }
}
```

#### 2. OpenClaw Bridge Config

```toml
# /etc/openclaw/config.toml

[server]
hostname = "bridge.yourdomain.com"
port = 8443

[discovery]
# Enable mDNS advertisement
mdns_enabled = true
mdns_name = "OpenClaw Bridge"

# Matrix homeserver URL
matrix_homeserver = "https://matrix.yourdomain.com"

# Human-readable server name
server_name = "My OpenClaw Server"

[qr]
# QR code signing key (generate with: openssl rand -hex 32)
signing_key = "your-32-byte-hex-key"
```

#### 3. QR Code Generation

OpenClaw should expose:
- `GET /qr/config` - Generate config QR for ArmorChat
- `GET /qr/setup` - Generate setup QR for new device pairing

#### 4. Self-Signed Certificates

For local deployments without proper certificates:

```bash
# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Or use the Bridge's built-in cert generation
openclaw-bridge --generate-certs
```

ArmorChat will show a warning for self-signed certificates but allow connection.

### OpenClaw Docker Compose Example

```yaml
version: '3.8'

services:
  # Matrix homeserver (Conduit - lightweight)
  conduit:
    image: girlbossceo/conduit:latest
    volumes:
      - conduit-data:/data
    environment:
      - CONDUIT_SERVER_NAME=yourdomain.com
    ports:
      - "6167:6167"

  # OpenClaw Bridge
  bridge:
    image: openclaw/bridge:latest
    volumes:
      - bridge-data:/data
      - bridge-certs:/certs
    environment:
      - OPENCLAW_HOSTNAME=bridge.yourdomain.com
      - OPENCLAW_MATRIX_URL=http://conduit:6167
      - OPENCLAW_MDNS_ENABLED=true
    ports:
      - "8443:8443"
    depends_on:
      - conduit

  # Reverse proxy
  caddy:
    image: caddy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy-data:/data
    depends_on:
      - conduit
      - bridge

volumes:
  conduit-data:
  bridge-data:
  bridge-certs:
  caddy-data:
```

### Caddyfile for OpenClaw

```
yourdomain.com {
    # Well-known discovery
    respond /.well-known/matrix/client `{
        "m.homeserver": {"base_url": "https://matrix.yourdomain.com"},
        "com.armorclaw.bridge": {
            "base_url": "https://bridge.yourdomain.com",
            "api_endpoint": "https://bridge.yourdomain.com/api"
        }
    }` 200

    header Access-Control-Allow-Origin *
}

matrix.yourdomain.com {
    reverse_proxy conduit:6167
}

bridge.yourdomain.com {
    reverse_proxy bridge:8443
}
```

### Self-Hosted Discovery Considerations

| Aspect | ArmorClaw (Cloud) | OpenClaw (Self-Hosted) |
|--------|-------------------|------------------------|
| Domain | `*.armorclaw.app` | Your own domain |
| SSL | Let's Encrypt (auto) | Let's Encrypt or self-signed |
| Discovery | Built-in well-known | Configure in reverse proxy |
| QR Signing | ArmorClaw signs | Your own key |
| Fallback | Multiple cloud servers | Your own fallback(s) |
| mDNS | Optional | Recommended for local |

### OpenClaw-Specific Recommendations

1. **Use Well-Known Discovery**
   - Easiest for users
   - Standard Matrix approach
   - One-time setup in reverse proxy

2. **Enable mDNS for Local Networks**
   - Auto-discovery in home/office
   - No DNS configuration needed
   - Works with local IP addresses

3. **Generate Config QR Codes**
   - Create QR for initial setup
   - Include in admin panel
   - Share via email/link

4. **Document Manual Setup**
   - Fallback if discovery fails
   - Clear instructions for users

5. **Certificate Warnings**
   - ArmorChat handles self-signed certs
   - Show warning but allow connection
   - Consider using Let's Encrypt for trusted certs

- [ ] All DNS records created and propagated
- [ ] Matrix homeserver running and accessible
- [ ] Bridge server running and accessible
- [ ] SSL certificates valid for all domains
- [ ] Health check returns 200 OK: `curl https://bridge.armorclaw.app/health`
- [ ] RPC endpoint works: `curl -X POST https://bridge.armorclaw.app/api -d '{"jsonrpc":"2.0","method":"system.health","id":1}'`
- [ ] Matrix login works via app
- [ ] Certificate pins added to `CertificatePinner.kt` (if using pinning)
- [ ] APK signed with release key
- [ ] APK tested on physical device

---

## 24. Onboarding Flow

### User Onboarding Steps

```
Splash Screen
     │
     ├── hasLegacyBridgeSession? ──YES──▶ MigrationScreen (v2.5 → v3.0)
     │
     ├── hasValidSession && !isBackupComplete? ──YES──▶ KeyBackupSetupScreen
     │
     ├── hasValidSession && isLoggedIn? ──YES──▶ HomeScreen
     │
     ├── hasCompletedOnboarding && !isLoggedIn? ──YES──▶ LoginScreen
     │
     └── First time user ──▶ ConnectServerScreen (QR-first onboarding)
                                   │
                                   ├── Manual entry path:
                                   │      ▼
                                   │  PermissionsScreen
                                   │      │
                                   │      ▼
                                   │  CompletionScreen
                                   │      │
                                   │      ▼
                                   │  KeyBackupSetupScreen
                                   │      │
                                   │      ▼
                                   │  HomeScreen
                                   │
                                   └── QR scan path:
                                          ▼
                                    QRScanScreen
                                          │
                                          ▼
                                    ExpressCompleteScreen
                                          │
                                          ▼
                                    HomeScreen
```

### Server Connection Flow

```
User enters homeserver URL
         │
         ▼
   SetupService.startSetup()
         │
         ├──▶ detectServer() - Health check on bridge
         │         │
         │         ├── SUCCESS: Server info captured
         │         │
         │         └── FAIL: Try fallback servers
         │                   │
         │                   ├── bridge.armorclaw.app
         │                   ├── bridge-backup.armorclaw.app
         │                   └── bridge-eu.armorclaw.app
         │
         ▼
   User enters credentials
         │
         ▼
   SetupService.connectWithCredentials()
         │
         ├──▶ startBridge() - Start bridge session
         ├──▶ matrixLogin() - Login to Matrix (deprecated RPC)
         │         │
         │         NOTE: Should use MatrixClient.login() instead!
         │
         ├──▶ wsClient.connect() - WebSocket connection (NON-FATAL)
         │         │
         │         NOTE: WS failure is caught and logged but does NOT
         │         block setup. ArmorChat uses Matrix /sync, not WS.
         │
         ├──▶ provisioningClaim() - Claim admin if setup token present
         │         │
         │         NOTE: First-boot flow. Skipped if no token or already claimed.
         │
         └──▶ getUserPrivilegesFromServer() - Get admin role
         │
         ▼
   Setup Complete
```

### How ArmorChat Finds ArmorClaw

1. **User-Provided URL:** User enters homeserver URL in ConnectServerScreen
2. **Bridge URL Derivation:** App derives bridge URL from homeserver:
   - `https://matrix.example.com` → `https://bridge.example.com`
3. **Health Check:** App calls `bridge.health` RPC to verify server
4. **Fallback:** If primary fails, tries known fallback servers
5. **Manual Override:** User can specify custom bridge URL in Advanced Options

### Quick Options Available

| Option | Homeserver | Bridge URL | Source |
|--------|------------|------------|--------|
| Production Default | `https://matrix.armorclaw.app` | `https://bridge.armorclaw.app/api` | `SetupConfig.createDefault()` |
| Demo Server | `https://demo.armorclaw.app` | `https://bridge-demo.armorclaw.app` | `SetupService.useDemoServer()` |
| Local Development | `http://10.0.2.2:8008` | `http://10.0.2.2:8080/api` | `SetupConfig.createDebug()` |

---

## 25. ArmorChat: Signed Config Integration

### ✅ Implementation Status

| Component | File | Status |
|-----------|------|--------|
| SetupConfig (enhanced) | `shared/.../bridge/SetupService.kt` | ✅ Done |
| ConfigSource enum (7 values) | `shared/.../bridge/SetupService.kt` | ✅ Done |
| SetupState (enhanced) | `shared/.../bridge/SetupService.kt` | ✅ Done |
| SignedServerConfig | `shared/.../bridge/SetupService.kt` | ✅ Done |
| ConfigParseResult | `shared/.../bridge/SetupService.kt` | ✅ Done |
| parseSignedConfig() | `shared/.../bridge/SetupService.kt` | ✅ Done |
| startSetupWithDiscovery() | `shared/.../bridge/SetupService.kt` | ✅ Done |
| DiscoveredServer | `shared/.../bridge/SetupService.kt` | ✅ Done |
| Deep Link Actions | `androidApp/.../navigation/DeepLinkHandler.kt` | ✅ Done |
| Intent Filters | `androidApp/.../AndroidManifest.xml` | ✅ Done |
| ConfigRepository | `shared/.../config/ConfigRepository.android.kt` | 📋 Not created |

### 📋 Android Components (To Create)

> **Note:** These components are proposed but not yet implemented. Config persistence
> is currently handled by `SetupService` in the shared module.

#### 1. ConfigRepository (Persistence)

Create `shared/src/androidMain/kotlin/platform/config/ConfigRepository.android.kt`:

```kotlin
package com.armorclaw.shared.platform.config

import android.content.Context
import android.content.SharedPreferences
import com.armorclaw.shared.platform.bridge.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

class ConfigRepository(context: Context) {

    companion object {
        private const val PREFS_NAME = "armorclaw_config"
        private const val KEY_HOMESERVER = "matrix_homeserver"
        private const val KEY_BRIDGE_URL = "bridge_url"
        private const val KEY_WS_URL = "ws_url"
        private const val KEY_PUSH_GATEWAY = "push_gateway"
        private const val KEY_SERVER_NAME = "server_name"
        private const val KEY_CONFIG_SOURCE = "config_source"
        private const val KEY_EXPIRES_AT = "expires_at"

        @Volatile
        private var instance: ConfigRepository? = null

        fun getInstance(context: Context): ConfigRepository {
            return instance ?: synchronized(this) {
                instance ?: ConfigRepository(context.applicationContext).also {
                    instance = it
                }
            }
        }
    }

    private val prefs: SharedPreferences =
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    private val _config = MutableStateFlow(loadConfig())
    val config: StateFlow<SetupConfig> = _config.asStateFlow()

    fun saveConfig(config: SetupConfig) {
        prefs.edit()
            .putString(KEY_HOMESERVER, config.homeserver)
            .putString(KEY_BRIDGE_URL, config.bridgeUrl)
            .putString(KEY_WS_URL, config.wsUrl)
            .putString(KEY_PUSH_GATEWAY, config.pushGateway)
            .putString(KEY_SERVER_NAME, config.serverName)
            .putString(KEY_CONFIG_SOURCE, config.configSource.name)
            .putLong(KEY_EXPIRES_AT, config.expiresAt ?: Long.MAX_VALUE)
            .apply()
        _config.value = config
    }

    fun loadConfig(): SetupConfig {
        if (!prefs.contains(KEY_HOMESERVER)) {
            return SetupConfig.createDefault()
        }
        val sourceName = prefs.getString(KEY_CONFIG_SOURCE, null)
        val source = try {
            sourceName?.let { ConfigSource.valueOf(it) } ?: ConfigSource.CACHED
        } catch (e: IllegalArgumentException) {
            ConfigSource.CACHED
        }
        return SetupConfig(
            homeserver = prefs.getString(KEY_HOMESERVER, "") ?: "",
            bridgeUrl = prefs.getString(KEY_BRIDGE_URL, null),
            wsUrl = prefs.getString(KEY_WS_URL, null),
            pushGateway = prefs.getString(KEY_PUSH_GATEWAY, null),
            serverName = prefs.getString(KEY_SERVER_NAME, null),
            configSource = source,
            expiresAt = prefs.getLong(KEY_EXPIRES_AT, Long.MAX_VALUE)
        )
    }

    fun clearConfig() {
        prefs.edit().clear().apply()
        _config.value = SetupConfig.createDefault()
    }

    fun isConfigured(): Boolean = _config.value.isConfigured()
}
```

#### 2. SetupViewModel (Android)

> **Note:** SetupViewModel already exists at `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SetupViewModel.kt`.
> The code below is a proposed refactor with `ConfigRepository` integration.

Proposed `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/SetupViewModel.kt` (enhanced):

```kotlin
package com.armorclaw.app.ui.setup

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.platform.bridge.*
import com.armorclaw.shared.platform.config.ConfigRepository
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch

class SetupViewModel(
    private val setupService: SetupService,
    private val configRepository: ConfigRepository
) : ViewModel() {

    private val _state = MutableStateFlow<SetupState>(SetupState.Idle)
    val state: StateFlow<SetupState> = _state.asStateFlow()

    val config: StateFlow<SetupConfig> = configRepository.config

    init {
        if (configRepository.isConfigured()) {
            _state.value = SetupState.ReadyForCredentials
        }
    }

    fun startDiscovery(input: String) {
        viewModelScope.launch {
            _state.value = SetupState.Discovering
            val result = setupService.startSetupWithDiscovery(input)
            when (result) {
                is SetupResult.Success -> {
                    configRepository.saveConfig(setupService.config.value)
                    _state.value = SetupState.ReadyForCredentials
                }
                is SetupResult.Error -> {
                    _state.value = SetupState.Error(result.message, result.fallbackOptions)
                }
            }
        }
    }

    fun handleDeepLink(deepLink: String) {
        viewModelScope.launch {
            _state.value = SetupState.Discovering
            val result = setupService.parseSignedConfig(deepLink)
            when (result) {
                is SetupResult.Success -> {
                    configRepository.saveConfig(setupService.config.value)
                    _state.value = SetupState.ReadyForCredentials
                }
                is SetupResult.Error -> {
                    _state.value = SetupState.Error(result.message, result.fallbackOptions)
                }
            }
        }
    }

    fun useDemoServer() {
        viewModelScope.launch {
            val result = setupService.useDemoServer()
            when (result) {
                is SetupResult.Success -> {
                    configRepository.saveConfig(setupService.config.value)
                    _state.value = SetupState.ReadyForCredentials
                }
                is SetupResult.Error -> {
                    _state.value = SetupState.Error(result.message)
                }
            }
        }
    }

    fun reset() {
        setupService.resetSetup()
        configRepository.clearConfig()
        _state.value = SetupState.Idle
    }
}
```

#### 3. Koin DI Module

> **Note:** This module does NOT exist. DI is configured in `ArmorClawApplication.kt`.
> Below is a proposed `ConfigModule` for clean separation.

Proposed `androidApp/src/main/kotlin/com/armorclaw/app/di/ConfigModule.kt`:

```kotlin
package com.armorclaw.app.di

import com.armorclaw.shared.platform.config.ConfigRepository
import com.armorclaw.shared.platform.bridge.SetupService
import com.armorclaw.shared.platform.bridge.BridgeRpcClient
import com.armorclaw.shared.platform.bridge.BridgeWebSocketClient
import org.koin.android.ext.koin.androidContext
import org.koin.dsl.module

val configModule = module {
    single { ConfigRepository.getInstance(androidContext()) }

    factory { BridgeRpcClient(get()) }
    factory { BridgeWebSocketClient() }
    factory { SetupService(get(), get()) }
}
```

### Actual Default URLs

Verified from `SetupConfig.createDefault()` in `SetupService.kt`:

```kotlin
SetupConfig(
    homeserver = "https://matrix.armorclaw.app",      // .app domain
    bridgeUrl = "https://bridge.armorclaw.app/api",   // /api suffix
    wsUrl = "wss://bridge.armorclaw.app/ws",
    pushGateway = "https://bridge.armorclaw.app/_matrix/push/v1/notify",
    serverName = "ArmorClaw",
    configSource = ConfigSource.DEFAULT
)
```

### Deep Link Flow Integration

```kotlin
// In MainActivity.kt or NavHost
when (action) {
    is DeepLinkAction.ApplySignedConfig -> {
        setupViewModel.handleDeepLink("armorclaw://config?d=${action.encodedData}")
        navController.navigate("onboarding/config")
    }
    is DeepLinkAction.SetupWithToken -> {
        setupViewModel.handleDeepLink(
            "armorclaw://setup?token=${action.token}&server=${action.server}"
        )
        navController.navigate("onboarding/setup")
    }
    is DeepLinkAction.AcceptInvite -> {
        navController.navigate("onboarding/invite?code=${action.code}")
    }
    // ... other actions
}
```

---
