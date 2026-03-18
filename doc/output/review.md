# ArmorChat Application Reference

> **Version:** 4.1.1-alpha05
> **Last Updated:** 2026-03-01
> **Purpose:** Comprehensive reference for understanding the ArmorChat application architecture, components, and implementation details.

---

## Table of Contents

1. [Application Overview](#1-application-overview)
2. [Architecture](#2-architecture)
3. [Tech Stack](#3-tech-stack)
4. [Module Structure](#4-module-structure)
5. [Domain Concepts](#5-domain-concepts)
6. [Data Flow](#6-data-flow)
7. [Navigation System](#7-navigation-system)
8. [State Management](#8-state-management)
9. [Security Implementation](#9-security-implementation)
10. [Communication Channels](#10-communication-channels)
    - [10.1 Bridge Discovery & Connection](#101-bridge-discovery--connection)
    - [10.2 JSON-RPC 2.0 Protocol](#102-json-rpc-20-protocol)
    - [10.3 Complete RPC Method Reference](#103-complete-rpc-method-reference)
    - [10.4 Push Notification Flow](#104-push-notification-flow)
    - [10.5 Matrix Protocol Integration](#105-matrix-protocol-integration)
    - [10.6 Health Check Protocol](#106-health-check-protocol)
    - [10.7 Error Handling](#107-error-handling)
    - [10.8 Security Considerations](#108-security-considerations)
    - [10.9 Implementation Files](#109-implementation-files)
11. [UI Component Library](#11-ui-component-library)
12. [UI/UX Design & User Navigation Guide](#115-uiux-design--user-navigation-guide)
    - [App Structure & Navigation Architecture](#app-structure--navigation-architecture)
    - [User Journey Flows](#user-journey-flows)
    - [Screen Catalog](#screen-catalog)
    - [Interaction Patterns](#interaction-patterns)
    - [Visual Design Language](#visual-design-language)
    - [Accessibility Features](#accessibility-features)
    - [Specialized UI Components](#specialized-ui-components)
    - [How UI Elements Help Users](#how-ui-elements-help-users)
    - [VPS Secretary Mode](#116-vps-secretary-mode-supervisory-ui)
13. [Key Files Reference](#12-key-files-reference)
14. [ArmorClaw Compatibility](#13-armorclaw-compatibility)

---

## 1. Application Overview

### What is ArmorChat?

ArmorChat is a **secure, end-to-end encrypted chat application** for Android that bridges external messaging platforms (Slack, Discord, Teams, Signal, WhatsApp) into a single Matrix-native client. It provides enterprise-grade security with hardware-backed encryption and AI agent task monitoring.

### Primary Users
- Enterprise users requiring secure cross-platform messaging
- Organizations with compliance requirements (HIPAA, DLP)
- Users who want unified access to multiple messaging platforms

### Core Value Propositions
1. **E2EE by Default** — All messages encrypted via Matrix Rust SDK
2. **Cross-Platform Bridging** — Single inbox for Slack/Discord/Teams/Signal/WhatsApp
3. **Hardware-Backed Security** — Android Keystore + SQLCipher + Biometrics
4. **AI Agent Monitoring** — Track and control AI assistants with capability revocation
5. **PII Protection** — BlindFill system for sensitive data access requests

---

## 2. Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER (Android)                     │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Screens (Compose)  │  ViewModels  │  Navigation  │  Theme     ││
│  └─────────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────────┤
│                      DOMAIN LAYER (Shared)                          │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Models  │  Repository Interfaces  │  Use Cases  │  Services   ││
│  └─────────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────────┤
│                       DATA LAYER (Shared + Android)                 │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Repository Impls  │  Stores  │  RPC Clients  │  Platform APIs ││
│  └─────────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────────┤
│                     PLATFORM LAYER (Android)                        │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  Keystore  │  SQLCipher  │  Biometric  │  Firebase  │  Network ││
│  └─────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘
```

### Key Architectural Patterns

| Pattern | Implementation |
|---------|---------------|
| **MVVM** | ViewModels expose `StateFlow<UiState>`, UI consumes via `collectAsState()` |
| **UDF** | Unidirectional Data Flow — State down, Events up |
| **Repository** | Interfaces in `domain/repository/`, implementations in `data/` |
| **Expect/Actual** | Platform services declared in shared with Android implementations |
| **Dependency Injection** | Koin modules in `di/` directories |

---

## 3. Tech Stack

### Core Technologies

| Category | Technology | Version |
|----------|------------|---------|
| **Language** | Kotlin | 1.9.20 |
| **UI Framework** | Jetpack Compose | 1.5.0 |
| **Design System** | Material 3 | Latest |
| **DI** | Koin | 3.5.0 |
| **Database** | SQLDelight + SQLCipher | 2.0.0 |
| **Networking** | Ktor | 2.3.5 |
| **Async** | Kotlin Coroutines + Flow | 1.7.3 |
| **Image Loading** | Coil | 2.4.0 |
| **Encryption** | Matrix Rust SDK | Latest |
| **Push** | Firebase Cloud Messaging | Latest |
| **Crash Reporting** | Sentry + Firebase Crashlytics | Latest |

### Build Configuration

```toml
# Key versions from libs.versions.toml
kotlin = "1.9.20"
android-gradle = "8.2.0"
compose = "1.5.0"
coroutines = "1.7.3"
koin = "3.5.0"
ktor = "2.3.5"
sqldelight = "2.0.0"
```

---

## 4. Module Structure

```
ArmorChat/
├── shared/                          # KMP Shared Module
│   └── src/
│       ├── commonMain/kotlin/
│       │   ├── domain/              # Business logic
│       │   │   ├── model/           # Data models
│       │   │   ├── repository/      # Repository interfaces
│       │   │   └── usecase/         # Use cases
│       │   ├── platform/            # Expect declarations
│       │   │   ├── matrix/          # Matrix client interface
│       │   │   ├── bridge/          # Bridge RPC client
│       │   │   ├── encryption/      # Encryption types
│       │   │   ├── notification/    # Push notification
│       │   │   └── logging/         # Logging abstraction
│       │   ├── data/                # Data layer
│       │   │   ├── store/           # State stores (ControlPlaneStore)
│       │   │   └── repository/      # Repository implementations
│       │   └── ui/                  # Shared UI
│       │       ├── components/      # Reusable composables
│       │       ├── theme/           # Theme, colors, typography
│       │       └── base/            # BaseViewModel, UiState, UiEvent
│       └── androidMain/kotlin/      # Android actual implementations
│
├── androidApp/                      # Android Application
│   └── src/main/kotlin/com/armorclaw/app/
│       ├── screens/                 # Screen composables by feature
│       │   ├── auth/                # Login, Registration, Recovery
│       │   ├── onboarding/          # Welcome, Setup, Migration
│       │   ├── home/                # Home, RoomList
│       │   ├── chat/                # ChatScreen, Message composables
│       │   ├── room/                # RoomDetails, Settings
│       │   ├── settings/            # Security, Preferences
│       │   ├── keystore/            # Unseal, Seal operations
│       │   └── governance/          # Admin, License screens
│       ├── viewmodels/              # Screen ViewModels
│       ├── navigation/              # AppNavigation (55 routes)
│       ├── di/                      # Koin modules
│       ├── platform/                # Android-specific services
│       │   ├── matrix/              # MatrixClientImpl
│       │   ├── bridge/              # BridgeRpcClientImpl
│       │   ├── security/            # KeystoreManager, SqlCipherProvider
│       │   └── notification/        # FirebaseMessagingService
│       └── data/                    # Android data implementations
│
└── armorclaw-ui/                    # Shared UI Components Module
    └── src/commonMain/kotlin/
        ├── components/              # Feature components
        │   ├── vault/               # Vault security components
        │   ├── governor/            # AI agent governance
        │   └── audit/               # Audit trail components
        └── theme/                   # ArmorClawTheme, StatusIcons
```

---

## 5. Domain Concepts

### Core Models

#### User & Authentication
```kotlin
// User representation
data class User(
    val userId: String,           // Matrix user ID (@user:server.com)
    val displayName: String,
    val avatarUrl: String?,
    val isBridged: Boolean,       // From external platform?
    val bridgePlatform: BridgePlatform? // SLACK, DISCORD, TEAMS, etc.
)

// Bridge platform identification
enum class BridgePlatform {
    SLACK, DISCORD, TEAMS, SIGNAL, WHATSAPP, MATRIX
}
```

#### Rooms & Messages
```kotlin
// Chat room
data class Room(
    val roomId: String,
    val name: String,
    val isEncrypted: Boolean,
    val isDirect: Boolean,
    val bridgeCapabilities: BridgeRoomCapabilities?, // Feature support
    val unreadCount: Int,
    val lastMessage: UnifiedMessage?
)

// Unified message format
data class UnifiedMessage(
    val eventId: String,
    val sender: MessageSender,
    val content: MessageContent,
    val timestamp: Long,
    val editCount: Int,
    val isRedacted: Boolean
)

// Room capabilities for bridged rooms
data class BridgeRoomCapabilities(
    val canEdit: Boolean,      // Can edit messages?
    val canReact: Boolean,     // Can add reactions?
    val canThread: Boolean,    // Can create threads?
    val canReply: Boolean      // Can reply?
)
```

#### Security & Encryption
```kotlin
// Trust levels for devices
enum class TrustLevel {
    UNVERIFIED,                 // Unknown device
    VERIFIED_IN_PERSON,         // Verified via SAS emoji
    CROSS_SIGNED,              // Cross-signed by user
    BLACKLISTED                // Blocked device
}

// Keystore status for VPS operations
sealed class KeystoreStatus {
    object Sealed : KeystoreStatus()      // Locked, requires unseal
    data class Unsealed(                  // Active session
        val unsealedAt: Long,
        val sessionDurationMs: Long = 4 * 60 * 60 * 1000L // 4 hours
    ) : KeystoreStatus()
    data class Error(val message: String) : KeystoreStatus()
}
```

#### AI Agent Monitoring (Control Plane)
```kotlin
// Agent task status
data class AgentTaskStatusEvent(
    val taskId: String,
    val status: AgentStatus,
    val capability: String?,
    val metadata: Map<String, String>?
)

enum class AgentStatus {
    IDLE, BROWSING, FORM_FILLING, PROCESSING_PAYMENT,
    AWAITING_CAPTCHA, AWAITING_2FA, AWAITING_APPROVAL,
    COMPLETED, FAILED, CANCELLED
}

// PII access request from agent
data class PiiAccessRequest(
    val requestId: String,
    val taskId: String,
    val requestedFields: List<PiiField>,
    val expiresAt: Long
)

data class PiiField(
    val fieldName: String,
    val sensitivity: SensitivityLevel
)

enum class SensitivityLevel {
    LOW, MEDIUM, HIGH, CRITICAL  // CRITICAL requires biometric
}
```

---

## 6. Data Flow

### Message Flow (Incoming)
```
1. Push Notification received (FCM)
   ↓
2. FirebaseMessagingService.onMessageReceived()
   ↓
3. MatrixClient.syncOnce() — Fetch and decrypt event
   ↓
4. MessageRepository.processEvent()
   ↓
5. RoomStore.update() — Emit new state
   ↓
6. ChatScreen UI recomposes with new message
```

### Message Flow (Outgoing)
```
1. User types message in ChatScreen
   ↓
2. ChatViewModel.sendMessage(text)
   ↓
3. MatrixClient.sendMessage(roomId, content)
   ↓
4. Rust SDK encrypts and sends to homeserver
   ↓
5. Optimistic update in RoomStore
   ↓
6. Send confirmation updates message status
```

### Bridge Event Flow
```
1. External platform event (e.g., Slack message)
   ↓
2. Bridge server receives via platform API
   ↓
3. Bridge converts to Matrix event format
   ↓
4. Bridge sends to Matrix homeserver
   ↓
5. Homeserver pushes to ArmorChat via FCM
   ↓
6. Standard message flow continues
```

---

## 7. Navigation System

### Route Categories (55 Total Routes)

```kotlin
// Navigation structure in AppNavigation.kt
object Routes {
    // Splash & Auth (5 routes)
    const val SPLASH = "splash"
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val FORGOT_PASSWORD = "forgot_password"
    const val KEY_RECOVERY = "key_recovery"

    // Onboarding (9 routes)
    const val WELCOME = "welcome"
    const val MIGRATION = "migration"           // v2.5 → v4.1
    const val SECURITY_EXPLANATION = "security_explanation"
    const val CONNECT_SERVER = "connect_server"
    const val PERMISSIONS = "permissions"
    const val COMPLETION = "completion"
    const val KEY_BACKUP_SETUP = "key_backup_setup"
    const val TUTORIAL = "tutorial"
    const val ONBOARDING_CONFIG = "onboarding_config"

    // Main App (15 routes)
    const val HOME = "home"
    const val CHAT = "chat/{roomId}"
    const val ROOM_DETAILS = "room_details/{roomId}"
    const val SETTINGS = "settings"
    const val SECURITY_SETTINGS = "security_settings"
    const val BRIDGE_VERIFICATION = "bridge_verification/{deviceId}"
    const val USER_PROFILE = "user_profile/{userId}"
    // ... more

    // Agent & Workflow (5 routes)
    const val AGENT_MANAGEMENT = "agent_management"
    const val HITL_APPROVALS = "hitl_approvals"
    const val WORKFLOW_MANAGEMENT = "workflow_management"
    const val BUDGET_STATUS = "budget_status"

    // Keystore (2 routes)
    const val UNSEAL = "unseal"
    const val SEAL = "seal"

    // Admin & Governance (4 routes)
    const val LICENSES = "licenses"
    const val TERMS_OF_SERVICE = "terms_of_service"
    const val GOVERNANCE = "governance"
    const val AUDIT_LOG = "audit_log"
}
```

### Screen Hierarchy
```
SplashScreen
├── Migration Flow (legacy users)
│   └── MigrationScreen → wipeLegacyData()
│
├── Onboarding Flow (new users)
│   ├── WelcomeScreen
│   ├── SecurityExplanationScreen
│   ├── ConnectServerScreen (with health check)
│   ├── PermissionsScreen
│   ├── CompletionScreen
│   └── KeyBackupSetupScreen (mandatory)
│
├── Authentication Flow (returning users)
│   ├── LoginScreen
│   │   └── → KeyRecoveryScreen
│   ├── RegistrationScreen
│   └── ForgotPasswordScreen
│
└── Main App (post-auth)
    ├── HomeScreen
    │   ├── RoomList
    │   └── GovernanceBanner
    ├── ChatScreen
    │   ├── MessageList
    │   ├── BridgedOriginBadge (on bridged users)
    │   ├── AgentStatusBanner (when agent active)
    │   ├── BlindFillCard (PII requests)
    │   └── ContentPolicyPlaceholder (DLP/HIPAA)
    ├── RoomDetailsScreen
    │   └── BridgeVerificationBanner
    ├── SettingsScreen
    │   ├── SecuritySettingsScreen
    │   │   └── → KeyBackupSetupScreen
    │   └── KeystoreScreen
    │       └── → UnsealScreen
    └── Governance screens
```

---

## 8. State Management

### ViewModel Pattern
```kotlin
// Base ViewModel with logging and error handling
abstract class BaseViewModel : ViewModel() {
    protected val _uiState = MutableStateFlow<UiState>(UiState.Idle)
    val uiState: StateFlow<UiState> = _uiState.asStateFlow()

    private val _events = MutableSharedFlow<UiEvent>()
    val events: SharedFlow<UiEvent> = _events.asSharedFlow()

    protected suspend fun emitEvent(event: UiEvent)
    protected fun <T> execute(operation: String, block: suspend () -> T): Result<T>
}

// UI State sealed class
sealed class UiState {
    object Idle : UiState()
    object Loading : UiState()
    data class Error(val message: String?) : UiState()
}

// UI Events for one-time actions
sealed class UiEvent {
    data class NavigateTo(val route: String, val data: Map<String, Any?>? = null) : UiEvent()
    data class NavigateBack(val result: Map<String, Any?>? = null) : UiEvent()
    data class ShowError(val message: String) : UiEvent()
    data class ShowSnackbar(val message: String) : UiEvent()
    data class CopyToClipboard(val text: String) : UiEvent()
}
```

### Key ViewModels

| ViewModel | Purpose | Key State |
|-----------|---------|-----------|
| `SplashViewModel` | App startup routing | `SplashTarget` enum |
| `LoginViewModel` | Authentication | `LoginUiState` |
| `ChatViewModel` | Chat screen | `ChatUiState` with messages, input |
| `HomeViewModel` | Room list | `HomeUiState` with rooms, filters |
| `SetupViewModel` | Server connection | `SetupUiState` with health status |
| `UnsealViewModel` | Keystore unseal | `UnsealUiState` with password/biometric |
| `ControlPlaneStore` | Agent/PII state | Agent statuses, PII requests, keystore |

### StateFlow Usage
```kotlin
// In ViewModel
private val _uiState = MutableStateFlow(ChatUiState())
val uiState: StateFlow<ChatUiState> = _uiState.asStateFlow()

// In Compose Screen
@Composable
fun ChatScreen(viewModel: ChatViewModel = koinViewModel()) {
    val uiState by viewModel.uiState.collectAsState()

    // Use uiState directly in composables
    MessageList(messages = uiState.messages)
}
```

---

## 9. Security Implementation

### Encryption Stack

```
┌─────────────────────────────────────────────────────────────┐
│                    ENCRYPTION LAYERS                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Transport Layer                                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  TLS 1.3 with Certificate Pinning (SHA-256)            ││
│  └─────────────────────────────────────────────────────────┘│
│                                                             │
│  Application Layer                                          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Matrix E2EE via Rust SDK (Olm/Megolm)                 ││
│  │  - Double Ratchet Algorithm                            ││
│  │  - Curve25519 Key Exchange                             ││
│  │  - AES-256-CBC + HMAC-SHA256                           ││
│  └─────────────────────────────────────────────────────────┘│
│                                                             │
│  Storage Layer                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  SQLCipher (256-bit passphrase)                        ││
│  │  Android Keystore (hardware-backed)                    ││
│  │  Encrypted SharedPreferences                           ││
│  └─────────────────────────────────────────────────────────┘│
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Key Management

```kotlin
// KeystoreManager handles hardware-backed key storage
class KeystoreManager {
    // Generate key in Android Keystore (hardware-backed)
    fun generateKey(alias: String): SecretKey

    // Encrypt data with Keystore key
    fun encrypt(alias: String, plaintext: ByteArray): EncryptedData

    // Decrypt data with Keystore key
    fun decrypt(alias: String, encrypted: EncryptedData): ByteArray

    // Check if key exists
    fun hasKey(alias: String): Boolean
}

// SSSS Key Backup (12-word BIP39 recovery phrase)
class KeyBackupSetupScreen {
    // 1. Explain importance
    // 2. Generate 12-word phrase
    // 3. Display phrase (copy support)
    // 4. Verify user knows phrase
    // 5. Upload encrypted backup to homeserver
    // 6. Confirm success
}
```

### Biometric Authorization

```kotlin
// BiometricAuthorizer for sensitive operations
class BiometricAuthorizer(private val context: Context) {
    // Authenticate for CRITICAL PII fields
    suspend fun authenticateForPiiAccess(): Result<Unit>

    // Authenticate for keystore unseal
    suspend fun authenticateForUnseal(): Result<Unit>

    // Check if biometric is available
    fun isBiometricAvailable(): Boolean
}
```

### PII Sensitivity Model

| Level | Fields | Authentication Required |
|-------|--------|------------------------|
| LOW | Email, Name, Company | None (auto-approved) |
| MEDIUM | Address, Phone | Explicit button tap |
| HIGH | Credit Card (masked), DOB | Explicit button tap |
| CRITICAL | SSN, CVV, Password | **Biometric required** |

---

## 10. Communication Channels

### What is ArmorClaw?

**ArmorClaw** is the bridge server that ArmorChat connects to. It is a **zero-trust security bridge** written in Go that enables:
1. **AI Agent Management** — Run isolated agents in Docker containers
2. **Platform Bridging** — Connect Slack/Discord/Teams/WhatsApp to Matrix
3. **PII Protection** — BlindFill system for sensitive data handling
4. **Push Notifications** — Sygnal gateway for mobile push

### ArmorClaw Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ARMORCLAW BRIDGE ARCHITECTURE                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   CLIENTS                           BRIDGE                      EXTERNAL    │
│   ┌─────────────┐                  ┌──────────────────────┐    ┌─────────┐ │
│   │  ArmorChat  │◄──Matrix /sync──►│   Matrix Homeserver  │    │  Slack  │ │
│   │  (Android)  │◄──JSON-RPC 2.0──►│      (Conduit)       │    │ Discord │ │
│   │             │◄──FCM Push───────►│                      │    │ Teams   │ │
│   └─────────────┘                  └──────────┬───────────┘    └─────────┘ │
│                                              │                             │
│   ┌─────────────┐                  ┌──────────▼───────────┐                │
│   │ArmorTerminal│◄──WebSocket─────►│   ArmorClaw Bridge   │                │
│   │  (Desktop)  │◄──JSON-RPC 2.0──►│      (Go Binary)     │                │
│   └─────────────┘                  │                      │                │
│                                    │  ┌────────────────┐  │                │
│                                    │  │ JSON-RPC 2.0   │  │                │
│                                    │  │ (122 methods)  │  │                │
│                                    │  └────────┬───────┘  │                │
│                                    │           │          │                │
│                                    │  ┌────────▼───────┐  │                │
│                                    │  │   Keystore     │  │                │
│                                    │  │  (SQLCipher)   │  │                │
│                                    │  └────────────────┘  │                │
│                                    │           │          │                │
│                                    │  ┌────────▼───────┐  │                │
│                                    │  │ Docker Runtime │  │                │
│                                    │  │  (AI Agents)   │  │                │
│                                    │  └────────────────┘  │                │
│                                    └──────────────────────┘                │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Channel Overview

| Channel | Protocol | Purpose |
|---------|----------|---------|
| Matrix /sync | HTTPS long-poll | Primary E2EE messaging |
| JSON-RPC 2.0 | HTTPS | Bridge admin operations |
| FCM Push | Google FCM | Background wake-up |
| WebSocket | NOT USED | (Reserved for ArmorTerminal only) |

---

### 10.1 Bridge Discovery & Connection

ArmorChat discovers and connects to ArmorClaw through multiple methods:

#### Discovery Methods

| Method | How It Works | Use Case |
|--------|--------------|----------|
| **QR Code** | Scan `armorclaw://config?d=...` deep link | Primary onboarding |
| **Manual URL** | User enters `https://bridge.example.com` | Manual setup |
| **mDNS** | Auto-discover `_armorclaw._tcp.local.` | Local network |
| **Deep Link** | Tap `armorclaw://` URL from email/SMS | Invitation flow |

#### QR Code Payload Structure

```json
{
  "version": 1,
  "setup_token": "stp_<48-hex-chars>",
  "matrix_homeserver": "https://matrix.example.com:8448",
  "rpc_url": "https://bridge.example.com:8443/api",
  "ws_url": "wss://bridge.example.com:8443/ws",
  "push_gateway": "https://push.example.com:5000",
  "server_name": "example.com",
  "region": "us-east-1",
  "bridge_public_key": "<optional-TOFU-key>",
  "expires_at": 1709000000,
  "signature": "hmac-sha256:<hex-digest>"
}
```

#### Connection Flow

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    ARMORCHAT → ARMORCLAW CONNECTION FLOW                  │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  1. DISCOVERY                                                            │
│     ┌─────────────┐      QR Scan / Manual Entry      ┌─────────────┐     │
│     │  ArmorChat  │ ─────────────────────────────────▶│ Parse URL   │     │
│     └─────────────┘                                    └──────┬──────┘     │
│                                                               │            │
│  2. WELL-KNOWN DISCOVERY                                     ▼            │
│     ┌─────────────────────────────────────────────────────────────────┐  │
│     │ GET https://server.com/.well-known/matrix/server               │  │
│     │ Response: { "m.server": "matrix.example.com:8448" }            │  │
│     └─────────────────────────────────────────────────────────────────┘  │
│                                                               │            │
│  3. HEALTH CHECK                                             ▼            │
│     ┌─────────────────────────────────────────────────────────────────┐  │
│     │ POST /rpc  {"method": "bridge.health", "id": 1}                │  │
│     │ Response: {                                                     │  │
│     │   "healthy": true,                                              │  │
│     │   "status": "ok",                                               │  │
│     │   "bridge_ready": true,                                         │  │
│     │   "is_new_server": false,                                       │  │
│     │   "provisioning_available": true,                               │  │
│     │   "server_name": "example.com",                                 │  │
│     │   "version": "4.1.0"                                            │  │
│     │ }                                                               │  │
│     └─────────────────────────────────────────────────────────────────┘  │
│                                                               │            │
│  4. PROVISIONING (if setup_token present)                    ▼            │
│     ┌─────────────────────────────────────────────────────────────────┐  │
│     │ POST /rpc  {"method": "provisioning.claim",                    │  │
│     │             "params": {"setup_token": "stp_..."}}              │  │
│     │ Response: {                                                     │  │
│     │   "success": true,                                              │  │
│     │   "admin_token": "atk_<token>",     // Persist this!           │  │
│     │   "role": "OWNER",                                              │  │
│     │   "matrix_user": "@admin:example.com"                          │  │
│     │ }                                                               │  │
│     └─────────────────────────────────────────────────────────────────┘  │
│                                                               │            │
│  5. MATRIX LOGIN                                            ▼             │
│     ┌─────────────────────────────────────────────────────────────────┐  │
│     │ POST /rpc  {"method": "matrix.login",                          │  │
│     │             "params": {"username": "user", "password": "..."}} │  │
│     │ Response: {                                                     │  │
│     │   "access_token": "syt_...",                                    │  │
│     │   "device_id": "DEVICEID",                                      │  │
│     │   "user_id": "@user:example.com"                               │  │
│     │ }                                                               │  │
│     └─────────────────────────────────────────────────────────────────┘  │
│                                                               │            │
│  6. PUSH REGISTRATION                                       ▼             │
│     ┌─────────────────────────────────────────────────────────────────┐  │
│     │ Matrix: POST /_matrix/client/v3/pushers/set                    │  │
│     │ Bridge: POST /rpc {"method": "push.register_token",            │  │
│     │                    "params": {"token": "FCM_TOKEN"}}           │  │
│     └─────────────────────────────────────────────────────────────────┘  │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘
```

---

### 10.2 JSON-RPC 2.0 Protocol

All ArmorChat ↔ ArmorClaw communication uses **JSON-RPC 2.0** over HTTPS.

#### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "bridge.health",
  "params": {
    // Method-specific parameters
  }
}
```

#### Response Format (Success)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    // Method-specific response
  }
}
```

#### Response Format (Error)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid request",
    "data": {
      "details": "Additional error information"
    }
  }
}
```

---

### 10.3 Complete RPC Method Reference

ArmorChat uses these RPC methods to communicate with ArmorClaw:

#### Bridge Lifecycle Methods

| Method | Purpose | Called By |
|--------|---------|-----------|
| `bridge.health` | Health check with all fields | `SetupViewModel`, startup |
| `bridge.status` | Operational status | Home refresh |
| `bridge.discover` | mDNS discovery helper | Local network setup |

**`bridge.health` Response (ArmorClaw 0.3.4):**
```json
{
  "result": {
    "healthy": true,           // Primary health indicator
    "status": "ok",            // String fallback ("ok" | "degraded" | "unhealthy")
    "bridge_ready": true,      // Bridge fully initialized?
    "is_new_server": false,    // First-time setup?
    "provisioning_available": true,  // Can provision new users?
    "server_name": "example.com",
    "version": "4.1.0",
    "capabilities": ["matrix", "slack", "discord"],
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

#### Provisioning Methods

| Method | Purpose | Params |
|--------|---------|--------|
| `provisioning.start` | Begin provisioning flow | `server_name`, `admin_email` |
| `provisioning.claim` | Claim bridge with setup token | `setup_token` |
| `provisioning.status` | Check provisioning state | — |
| `provisioning.rotate` | Rotate setup token | — |
| `provisioning.cancel` | Cancel provisioning | — |

**`provisioning.claim` Response:**
```json
{
  "result": {
    "success": true,
    "admin_token": "atk_<64-hex-chars>",  // MUST persist!
    "role": "OWNER",                        // OWNER | ADMIN | MODERATOR
    "matrix_user": "@admin:example.com",
    "expires_at": 1709000000
  }
}
```

#### Matrix Integration Methods

| Method | Purpose | Params |
|--------|---------|--------|
| `matrix.login` | Bridge-assisted Matrix login | `username`, `password` |
| `matrix.invite_user` | Invite user to room | `room_id`, `user_id` |
| `matrix.send_typing` | Typing indicator | `room_id`, `is_typing` |
| `matrix.send_read_receipt` | Read marker | `room_id`, `event_id` |

**`matrix.login` Response:**
```json
{
  "result": {
    "access_token": "syt_<token>",
    "device_id": "<device_id>",
    "user_id": "@user:example.com",
    "well_known": {
      "m.homeserver": { "base_url": "https://matrix.example.com" }
    }
  }
}
```

#### Push Notification Methods

| Method | Purpose | Params |
|--------|---------|--------|
| `push.register_token` | Register FCM token | `token`, `platform` |
| `push.unregister` | Remove FCM registration | `token` |
| `push.settings` | Get/set push preferences | `enabled`, `rooms` |

**Dual Registration (CRITICAL):**
```kotlin
// ArmorChat must register with BOTH Matrix AND Bridge
// 1. Matrix Homeserver (for Matrix-native messages)
matrixClient.setPusher(fcmToken, sygnalGatewayUrl)

// 2. ArmorClaw Bridge (for SDTW bridged messages)
bridgeRpcClient.pushRegister(fcmToken)
```

#### Recovery Methods

| Method | Purpose | Params |
|--------|---------|--------|
| `recovery.generate` | Generate recovery codes | `user_id` |
| `recovery.verify` | Verify recovery code | `code`, `user_id` |
| `recovery.store` | Store encrypted backup | `backup_data` |
| `recovery.complete` | Complete recovery flow | `session_id` |

#### Agent Status Methods (Phase 2)

| Method | Purpose | Params |
|--------|---------|--------|
| `agent.get_status` | Get current agent status | `agent_id` |
| `agent.status_history` | Get status change history | `agent_id`, `limit` |

**`agent.get_status` Response:**
```json
{
  "result": {
    "agent_id": "agent-123",
    "status": "IDLE",              // IDLE | RUNNING | PAUSED | COMPLETED | FAILED | CANCELLED
    "timestamp": 1709000000000,
    "metadata": { "task": "browser_automation" },
    "running_since": null,
    "current_task": null
  }
}
```

**`agent.status_history` Response:**
```json
{
  "result": {
    "agent_id": "agent-123",
    "history": [
      {
        "id": "entry-1",
        "agent_id": "agent-123",
        "status": "RUNNING",
        "timestamp": 1709000000000,
        "metadata": { "task": "form_fill" },
        "duration_ms": 5000
      }
    ],
    "total_count": 42
  }
}
```

#### Keystore / Zero-Trust Methods (Phase 2)

| Method | Purpose | Params |
|--------|---------|--------|
| `keystore.sealed` | Check if keystore is sealed | — |
| `keystore.unseal_challenge` | Get challenge for unsealing | — |
| `keystore.unseal_respond` | Respond with wrapped key | `challenge_id`, `wrapped_key` |
| `keystore.extend_session` | Extend unsealed session | — |

**`keystore.sealed` Response:**
```json
{
  "result": {
    "sealed": true,
    "seal_state": "sealed",        // sealed | unsealed | error
    "remaining_seconds": null,     // Only if unsealed
    "session_extensions_used": 0,
    "max_extensions": 3,
    "last_unsealed_at": null,
    "last_unsealed_by": null,
    "error_message": null
  }
}
```

**`keystore.unseal_challenge` Response:**
```json
{
  "result": {
    "challenge_id": "chal_abc123",
    "nonce": "base64_nonce",
    "server_public_key": "-----BEGIN RSA PUBLIC KEY-----...",
    "expires_at": 1709000300000,
    "key_derivation": {
      "algorithm": "argon2id",
      "memory_kb": 65536,
      "parallelism": 4,
      "salt": "base64_salt"
    }
  }
}
```

**`keystore.unseal_respond` Request/Response:**
```json
// Request
{
  "method": "keystore.unseal_respond",
  "params": {
    "challenge_id": "chal_abc123",
    "wrapped_key": "base64_rsa_oaep_wrapped_kek",
    "client_public_key": "-----BEGIN RSA PUBLIC KEY-----...",
    "unseal_method": "password"    // password | biometric
  }
}

// Response (Success)
{
  "result": {
    "success": true,
    "error": null,
    "error_code": null,
    "session_expires_at": 1709003900000,
    "session_duration_seconds": 3600
  }
}

// Response (Failure)
{
  "result": {
    "success": false,
    "error": "Invalid password",
    "error_code": "invalid_password",  // invalid_password | challenge_expired | keystore_error
    "session_expires_at": null,
    "session_duration_seconds": null
  }
}
```

**`keystore.extend_session` Response:**
```json
{
  "result": {
    "success": true,
    "new_expires_at": 1709007500000,
    "error": null,
    "max_extensions_reached": false
  }
}
```

---

### 10.4 Push Notification Flow

ArmorChat uses a **dual-registration** push notification system:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    PUSH NOTIFICATION FLOW                                 │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  REGISTRATION (occurs at login/setup)                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                                                                      │ │
│  │   ArmorChat                                                          │ │
│  │       │                                                              │ │
│  │       ├──▶ Matrix Homeserver                                        │ │
│  │       │    POST /_matrix/client/v3/pushers/set                      │ │
│  │       │    {                                                        │ │
│  │       │      "kind": "http",                                        │ │
│  │       │      "app_id": "com.armorclaw.app",                         │ │
│  │       │      "pushkey": "<FCM_TOKEN>",                              │ │
│  │       │      "data": {                                              │ │
│  │       │        "url": "https://push.example.com/_matrix/push/v1/notify",
│  │       │        "format": "event_id_only"                            │ │
│  │       │      }                                                      │ │
│  │       │    }                                                        │ │
│  │       │                                                              │ │
│  │       └──▶ ArmorClaw Bridge                                         │ │
│  │            POST /rpc {"method": "push.register_token",              │ │
│  │                     "params": {"token": "<FCM_TOKEN>"}}             │ │
│  │                                                                      │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
│  NOTIFICATION DELIVERY                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │                                                                      │ │
│  │   Message sent to room                                              │ │
│  │       │                                                              │ │
│  │       ▼                                                              │ │
│  │   Matrix Homeserver                                                 │ │
│  │       │                                                              │ │
│  │       ├──▶ Sygnal Push Gateway                                      │ │
│  │       │    POST /_matrix/push/v1/notify                             │ │
│  │       │    {                                                        │ │
│  │       │      "notification": {                                      │ │
│  │       │        "event_id": "$event_id",                             │ │
│  │       │        "room_id": "!room:example.com",                      │ │
│  │       │        "counts": { "unread": 5 }                            │ │
│  │       │      },                                                     │ │
│  │       │      "pushkey": "<FCM_TOKEN>"                               │ │
│  │       │    }                                                        │ │
│  │       │                                                              │ │
│  │       ▼                                                              │ │
│  │   Firebase Cloud Messaging                                          │ │
│  │       │                                                              │ │
│  │       ▼                                                              │ │
│  │   ArmorChat (Android)                                               │ │
│  │       │                                                              │ │
│  │       └──▶ FirebaseMessagingService.onMessageReceived()             │ │
│  │            │                                                         │ │
│  │            ├──▶ matrixClient.syncOnce()  // Fetch + decrypt event  │ │
│  │            │                                                         │ │
│  │            └──▶ Show notification with decrypted content            │ │
│  │                                                                      │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘
```

---

### 10.5 Matrix Protocol Integration

ArmorChat uses **Matrix protocol** as the primary messaging channel:

#### Client-Server API Endpoints

| Endpoint | Purpose | Auth |
|----------|---------|------|
| `/_matrix/client/v3/sync` | Long-poll for events | Access token |
| `/_matrix/client/v3/sendToDevice` | Send E2EE keys | Access token |
| `/_matrix/client/v3/keys/query` | Query device keys | Access token |
| `/_matrix/client/v3/keys/claim` | Claim one-time keys | Access token |
| `/_matrix/client/v3/pushers/set` | Register pusher | Access token |
| `/_matrix/client/r0/rooms/{roomId}/send` | Send message | Access token |

#### E2EE Flow (Olm/Megolm)

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    E2EE MESSAGE FLOW (Matrix Rust SDK)                    │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  SENDER (ArmorChat)                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │  1. Get room Megolm session (or create new)                         │ │
│  │  2. Encrypt message content with session key                        │ │
│  │  3. POST /_matrix/client/r0/rooms/{roomId}/send/m.room.encrypted   │ │
│  │     {                                                               │ │
│  │       "algorithm": "m.megolm.v1.aes-sha2",                          │ │
│  │       "sender_key": "<curve25519_key>",                             │ │
│  │       "session_id": "<megolm_session_id>",                          │ │
│  │       "ciphertext": "<base64_encrypted_content>"                    │ │
│  │     }                                                               │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
│  MATRIX HOMESERVER                                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │  - Stores encrypted event                                           │ │
│  │  - Delivers to all room members via /sync                           │ │
│  │  - Pushes notification via Sygnal                                   │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
│  RECEIVER (ArmorChat)                                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐ │
│  │  1. Receive encrypted event via /sync or push wake                  │ │
│  │  2. Look up Megolm session in Rust SDK cache                        │ │
│  │  3. Decrypt ciphertext with session key                             │ │
│  │  4. Display plaintext message                                       │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘
```

---

### 10.6 Health Check Protocol

ArmorChat performs health checks at multiple points:

#### Health Check Timing

| When | Method | Purpose |
|------|--------|---------|
| Server URL entry | `bridge.health` | Validate server before credential entry |
| App startup | `bridge.health` | Verify bridge is reachable |
| Periodic (5 min) | `bridge.health` | Monitor connection health |
| Before sensitive ops | `bridge.health` | Ensure bridge is ready |

#### Health Status Handling

```kotlin
enum class BridgeHealthStatus {
    HEALTHY,       // healthy: true, bridge_ready: true
    NOT_READY,     // healthy: true, bridge_ready: false → Block credential entry
    UNHEALTHY,     // healthy: false OR status != "ok"
    ERROR          // Network/RPC error
}

// Health check logic (SetupViewModel.kt)
fun checkHealth(data: JsonObject): BridgeHealthStatus {
    val healthy = data["healthy"] as? Boolean
        ?: (data["status"] as? String == "ok")
        ?: true

    if (!healthy) return BridgeHealthStatus.UNHEALTHY

    val bridgeReady = data["bridge_ready"] as? Boolean ?: true
    if (!bridgeReady) return BridgeHealthStatus.NOT_READY

    return BridgeHealthStatus.HEALTHY
}
```

---

### 10.7 Error Handling

#### RPC Error Codes

| Code | Meaning | Handling |
|------|---------|----------|
| `-32700` | Parse error | Show generic error |
| `-32600` | Invalid request | Retry with valid params |
| `-32601` | Method not found | Update required |
| `-32602` | Invalid params | Show validation error |
| `-32603` | Internal error | Retry with backoff |
| `-32001` | Server error | Show error, retry |
| `-32002` | Unauthorized | Re-authenticate |

#### Retry Strategy

```kotlin
// Exponential backoff for transient failures
class BridgeRpcClient {
    private val maxRetries = 3

    suspend fun <T> withRetry(block: suspend () -> T): Result<T> {
        repeat(maxRetries) { attempt ->
            try {
                return Result.success(block())
            } catch (e: IOException) {
                if (attempt == maxRetries - 1) return Result.failure(e)
                delay(1000L * (1 shl attempt))  // 1s, 2s, 4s
            }
        }
        return Result.failure(IOException("Max retries exceeded"))
    }
}
```

---

### 10.8 Security Considerations

#### TLS Requirements

- **Minimum TLS 1.2** for all connections
- **Certificate pinning** recommended for production
- **Self-signed certs** supported for IP-only mode

#### Token Storage

| Token | Storage | Lifetime |
|-------|---------|----------|
| `access_token` | Encrypted SharedPreferences | Until logout |
| `admin_token` | Encrypted SharedPreferences | Until re-provision |
| `FCM token` | SharedPreferences | Until app reinstall |

#### Authentication Flow

```
1. Health check (no auth)
2. provisioning.claim (setup_token) → admin_token
3. matrix.login (username/password) → access_token
4. All subsequent calls use access_token
```

---

### 10.9 Implementation Files

| File | Purpose |
|------|---------|
| `shared/.../platform/bridge/BridgeRpcClient.kt` | RPC interface |
| `shared/.../platform/bridge/BridgeRpcClientImpl.kt` | HTTP implementation |
| `shared/.../platform/bridge/SetupService.kt` | Provisioning flow |
| `shared/.../platform/bridge/RpcModels.kt` | Request/response models |
| `androidApp/.../viewmodels/SetupViewModel.kt` | Health check logic |
| `shared/.../platform/matrix/MatrixClient.kt` | Matrix interface |
| `shared/.../platform/matrix/MatrixClientImpl.kt` | Matrix HTTP implementation |
| `androidApp/.../service/FirebaseMessagingService.kt` | Push handling |
| `shared/.../platform/notification/PushNotificationRepository.kt` | Dual registration |

### Matrix Protocol Flow
```kotlin
// MatrixClient interface
interface MatrixClient {
    // Connection
    suspend fun startSync()
    suspend fun stopSync()
    suspend fun syncOnce()  // Used on push wake

    // Messaging
    suspend fun sendMessage(roomId: String, content: MessageContent)
    suspend fun sendTyping(roomId: String, isTyping: Boolean)
    suspend fun sendReadReceipt(roomId: String, eventId: String)

    // Push
    suspend fun setPusher(token: String, gatewayUrl: String)
    suspend fun removePusher()

    // Device verification
    suspend fun startVerification(deviceId: String)
    suspend fun confirmVerification(deviceId: String, emojis: List<String>)
}
```

### Bridge RPC Methods

| Method | Purpose | Called By |
|--------|---------|-----------|
| `push.register_token` | Register FCM for bridge events | `PushNotificationRepository` |
| `push.unregister_token` | Remove FCM registration | Logout flow |
| `provisioning.claim` | Claim bridge with setup token | `SetupService` |
| `bridge.health` | Health check (all fields) | `SetupViewModel` |
| `bridge.status` | Bridge operational status | Home refresh |
| `matrix.login` | Bridge-assisted Matrix login | `LoginViewModel` |
| `matrix.invite_user` | Invite user to room | Room creation |
| `matrix.send_typing` | Typing indicator via bridge | `ChatViewModel` |
| `matrix.send_read_receipt` | Read marker via bridge | `ChatViewModel` |
| `recovery.generate` | Generate recovery codes | `KeyRecoveryScreen` |
| `recovery.verify` | Verify recovery code | `KeyRecoveryScreen` |

### Health Check Fields (ArmorClaw 0.3.4)
```kotlin
data class BridgeHealthDetails(
    val healthy: Boolean,           // Primary health indicator
    val status: String,             // "ok" fallback
    val bridgeReady: Boolean,       // Bridge initialized?
    val isNewServer: Boolean,       // First-time setup?
    val provisioningAvailable: Boolean,  // Can provision new users?
    val serverName: String,         // Display name
    val version: String             // Bridge version
)

// Health check logic
fun isHealthy(data: JsonObject): Boolean {
    return data["healthy"] as? Boolean
        ?: (data["status"] as? String == "ok")
        ?: true
}
```

---

## 11. UI Component Library

### Shared Components (shared/ui/components/)

#### Vault Components
| Component | Purpose |
|-----------|---------|
| `VaultPulseIndicator` | Animated vault status indicator |
| `VaultKeyPanel` | Key management UI panel |

#### Governor Components
| Component | Purpose |
|-----------|---------|
| `CommandBlockCard` | Agent command display card |
| `CommandStatusBadge` | Status indicator (pending/running/done) |
| `CapabilityRibbon` | Quick capability overview |
| `CapabilityChip` | Individual capability badge |
| `HITLAuthorizationCard` | Human-in-the-loop approval UI |

#### Audit Components
| Component | Purpose |
|-----------|---------|
| `ArmorTerminal` | Terminal-style activity log |
| `RevocationPanel` | Capability revocation controls |
| `QuickRevocationButton` | One-click revoke |

#### Control Plane Phase 2 Components
| Component | Location | Purpose |
|-----------|----------|---------|
| `AgentStatusBanner` | `AgentStatusBanner.kt` | Animated agent task status |
| `BlindFillCard` | `BlindFillCard.kt` | PII access request approval |
| `PiiFieldItem` | `BlindFillCard.kt` | Individual PII field checkbox |
| `SensitivityBadge` | `SensitivityBadge.kt` | LOW/MEDIUM/HIGH/CRITICAL indicator |
| `SealedIndicator` | `SealedIndicator.kt` | Keystore status display |

### Theme System

```kotlin
// ArmorClawTheme colors
object ArmorClawColors {
    val Teal = Color(0xFF14F0C8)      // Primary brand
    val Navy = Color(0xFF0A1428)      // Dark background
    val NavyLight = Color(0xFF1A2438) // Card background
    val Error = Color(0xFFFF5252)     // Error state
    val Warning = Color(0xFFFFB74D)   // Warning state
    val Success = Color(0xFF4CAF50)   // Success state
}

// Sensitivity colors
object SensitivityColors {
    val Low = Color(0xFF4CAF50)       // Green
    val Medium = Color(0xFFFFB74D)    // Amber
    val High = Color(0xFFFF9800)      // Orange
    val Critical = Color(0xFFFF5252)  // Red
}
```

---

## 11.5 UI/UX Design & User Navigation Guide

This section provides a comprehensive guide to ArmorChat's user interface, navigation patterns, and how the UI helps users accomplish key tasks.

### App Structure & Navigation Architecture

#### Bottom Navigation (Main App Sections)

ArmorChat uses a **5-tab bottom navigation** pattern for primary navigation:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        ARMORCHAT MAIN UI                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    CONTENT AREA                                 │ │
│  │                                                                 │ │
│  │   (Home / Chat / Search / Settings / Profile screens)          │ │
│  │                                                                 │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐          │
│  │   🏠     │   💬     │   🔍     │   ⚙️     │   👤     │          │
│  │  Home    │  Chat    │  Search  │ Settings │ Profile  │          │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

| Tab | Icon | Purpose | Primary Action |
|-----|------|---------|----------------|
| **Home** | 🏠 | Room list, recent conversations | View all chat rooms |
| **Chat** | 💬 | Active chat (when in conversation) | Continue active conversation |
| **Search** | 🔍 | Global search across messages/rooms | Find content quickly |
| **Settings** | ⚙️ | App configuration, security, devices | Manage app preferences |
| **Profile** | 👤 | User profile, account management | View/edit profile |

### User Journey Flows

#### 1. First-Time Setup Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    FIRST-TIME USER ONBOARDING                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │
│  │ Splash  │───▶│ Welcome │───▶│Discover │───▶│ Provision│         │
│  │ Screen  │    │ Screen  │    │ Bridge  │    │  Setup   │         │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘         │
│       │              │              │              │                │
│       │              │              │              ▼                │
│       │              │              │       ┌─────────┐            │
│       │              │              │       │  QR     │            │
│       │              │              │       │  Scan   │            │
│       │              │              │       └─────────┘            │
│       │              │              │              │                │
│       │              │              ▼              ▼                │
│       │              │       ┌─────────────────────────┐           │
│       │              │       │   Matrix Login          │           │
│       │              │       │   (Username/Password)   │           │
│       │              │       └─────────────────────────┘           │
│       │              │                      │                       │
│       │              ▼                      ▼                       │
│       │       ┌─────────────────────────────────────┐              │
│       │       │        Key Backup Setup             │              │
│       │       │  (Recovery phrase generation)       │              │
│       │       └─────────────────────────────────────┘              │
│       │                             │                               │
│       ▼                             ▼                               │
│  ┌─────────────────────────────────────────────────────────┐       │
│  │                    HOME SCREEN                          │       │
│  └─────────────────────────────────────────────────────────┘       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**UI Elements That Guide Users:**

| Screen | UI Element | User Benefit |
|--------|------------|--------------|
| Welcome | "Get Started" button with progress dots | Clear call-to-action, shows journey length |
| Discover | Animated scanning indicator + manual entry | Visual feedback during network discovery |
| Provision | QR code scanner with guide overlay | Easy scanning, alignment guides help accuracy |
| Key Backup | Word cards with copy buttons | Easy phrase recording, one-tap copy |

#### 2. Daily Messaging Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                      DAILY MESSAGING FLOW                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  HOME SCREEN                                                         │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ 🔍 Search rooms...                              ┌─────┬─────┐  │ │
│  │                                                  │  +  │ ⋮   │  │ │
│  │  ┌────────────────────────────────────────────┐ └─────┴─────┘  │ │
│  │  │ 🔒 #general                    2:34 PM     │                │ │
│  │  │    Alice: Did you see the...               │  + = New Room  │ │
│  │  │                                    [ unread: 3 ]             │ │
│  │  ├────────────────────────────────────────────┤  ⋮ = Menu      │ │
│  │  │ 🟢 Sarah (Direct)              1:15 PM     │                │ │
│  │  │    Typing...                                │                │ │
│  │  ├────────────────────────────────────────────┤                │ │
│  │  │ 🔵 Slack: #engineering         12:00 PM    │                │ │
│  │  │    Bob: Build passed!                       │                │ │
│  │  │                              [ 🔀 Bridged ] │                │ │
│  │  └────────────────────────────────────────────┘                │ │
│  └────────────────────────────────────────────────────────────────┘ │
│         │                                                            │
│         │ Tap room                                                   │
│         ▼                                                            │
│  CHAT SCREEN                                                         │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ ← #general                              🔍  📞  📹  ⋮          │ │
│  │   🔒 Encrypted                           Search Call  More     │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │                                                                 │ │
│  │  ┌──────────────────────────────────────┐                      │ │
│  │  │ 👤 Alice                             │                      │ │
│  │  │ "Did you see the new design?"        │ 2:34 PM              │ │
│  │  │                            ❤️ 👍 ↩️   │                      │ │
│  │  └──────────────────────────────────────┘                      │ │
│  │                                                                 │ │
│  │  ┌──────────────────────────────────────┐                      │ │
│  │  │ 👤 You                               │                      │ │
│  │  │ "Yes! It looks great"                │ 2:35 PM ✓✓          │ │
│  │  └──────────────────────────────────────┘                      │ │
│  │                                                                 │ │
│  │  ┌──────────────────────────────────────┐                      │ │
│  │  │ 🤖 Assistant (Agent)                 │                      │ │
│  │  │ ▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░ Processing...  │ 2:36 PM              │ │
│  │  │ [Status: RUNNING - Browser Task]     │                      │ │
│  │  └──────────────────────────────────────┘                      │ │
│  │                                                                 │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │ 👤 Sarah is typing...                                          │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │ 📎  │ Type a message...                        😊  │ 🎤  │ ➤  │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Key UI Indicators:**

| Indicator | Meaning | User Action |
|-----------|---------|-------------|
| 🔒 Lock icon | End-to-end encrypted | Trust message security |
| ✓✓ Double check | Message delivered & read | Confirm delivery |
| 🔀 Bridged | From external platform | Understand message origin |
| 🟢 Green dot | User online | Know availability |
| ▓▓▓ Progress | Agent working | Wait for completion |
| 👤 Typing... | User composing | Expect incoming message |

### Screen Catalog

#### Home Screen (Room List)

**Purpose:** Central hub for all conversations

**Key UI Elements:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                         HOME SCREEN                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🔍 Search messages and rooms...              [+]  [⋮]     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  FILTER CHIPS (Horizontal scroll)                                   │
│  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐                │
│  │ All   │ │ Unread│ │ Direct│ │ Groups│ │ Muted │                │
│  └───────┘ └───────┘ └───────┘ └───────┘ └───────┘                │
│                                                                      │
│  ROOM CARDS                                                          │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [Avatar]  Room Name                              timestamp  │   │
│  │           Last message preview...                  [badge]  │   │
│  │           [🔒 Encrypted] [🔀 Bridged] [🤖 Agent]            │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  FAB (Floating Action Button)                                       │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                              [➕]  ← Create new room         │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**How UI Helps Users:**
- **Filter chips**: Quickly narrow down to relevant conversations
- **Room badges**: Unread count visibility without opening room
- **Platform indicators**: Know which platform a bridged room uses
- **Swipe actions**: Archive/mute rooms without entering them
- **Long-press**: Quick actions menu (mark read, pin, mute)

#### Chat Screen

**Purpose:** Core messaging interface

**Layout Zones:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CHAT SCREEN ZONES                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ZONE 1: TOP BAR (Fixed)                                            │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ ← [Room Name]              [🔍] [📞] [📹] [⋮]              │   │
│  │    [Status indicators]                                       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ZONE 2: CONTEXT BANNERS (Conditional)                              │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [Workflow Progress Banner] - Shows when workflow active     │   │
│  │ [Agent Status Banner] - Shows agent task status             │   │
│  │ [PII Request Card] - BlindFill approval UI                  │   │
│  │ [Agent Thinking Indicator] - Animated thinking state        │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ZONE 3: MESSAGE LIST (Scrollable)                                  │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                               │   │
│  │   [Message bubbles - incoming/outgoing differentiated]      │   │
│  │   [Date separators]                                          │   │
│  │   [Read receipts]                                            │   │
│  │   [Reaction indicators]                                      │   │
│  │   [Reply threads]                                            │   │
│  │                                                               │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ZONE 4: INPUT AREA (Fixed)                                         │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [📎] [Text field...] [😊] [🎤] [➤]                         │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Message Bubble Types:**

| Bubble Type | Visual Style | Purpose |
|-------------|--------------|---------|
| **User Message** | Right-aligned, primary color | Your sent messages |
| **Other User** | Left-aligned, surface color | Received messages |
| **Agent Command** | Card style with status badge | AI agent task display |
| **System Message** | Centered, muted text | Room events (joins, leaves) |
| **Encrypted** | Lock icon overlay | E2EE verification |
| **Bridged** | Platform icon badge | External platform origin |

#### Settings Screen

**Purpose:** App configuration and account management

**Organization:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SETTINGS SCREEN                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ACCOUNT SECTION                                                     │
│  ├── Profile                    → Edit display name, avatar         │
│  ├── Security & Privacy         → Biometrics, key management       │
│  ├── Devices                    → Manage logged-in devices          │
│  └── Data & Storage             → Cache, export data                │
│                                                                      │
│  CHAT SECTION                                                        │
│  ├── Appearance                → Theme, font size                   │
│  ├── Notifications             → Push, sound, quiet hours           │
│  ├── Chat Settings             → Read receipts, typing indicators  │
│  └── Blocked Users             → Manage blocklist                   │
│                                                                      │
│  BRIDGE SECTION                                                      │
│  ├── Connected Platforms        → Slack, Discord, etc.              │
│  ├── Agent Management           → AI agent control                  │
│  ├── Keystore Status            → VPS credential vault              │
│  └── HITL Approvals             → Pending human approvals           │
│                                                                      │
│  SUPPORT SECTION                                                     │
│  ├── Help & FAQ                → Documentation                      │
│  ├── Report a Bug              → Bug report form                    │
│  ├── About                     → Version, licenses                  │
│  └── Privacy Policy            → Legal information                  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### Security & Privacy Screen

**Purpose:** Manage security settings and view trust status

**UI Elements:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                    SECURITY & PRIVACY SCREEN                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  SECURITY STATUS CARD                                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🛡️ Security Status: PROTECTED                              │   │
│  │                                                              │   │
│  │  ✓ End-to-end encryption enabled                            │   │
│  │  ✓ Database encrypted (SQLCipher)                           │   │
│  │  ✓ Biometric unlock configured                              │   │
│  │  ⚠ Key backup recommended                                   │   │
│  │                                                              │   │
│  │  [View Details]                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  AUTHENTICATION                                                       │
│  ├── [Toggle] Biometric Unlock (Fingerprint/Face)                   │
│  ├── [Toggle] Require Auth on Launch                                │
│  └── [Button] Change PIN/Password                                   │
│                                                                      │
│  ENCRYPTION KEYS                                                     │
│  ├── [Button] Verify Encryption Keys                                │
│  ├── [Button] Export Keys (Encrypted)                               │
│  └── [Button] Recovery Phrase Setup                                 │
│                                                                      │
│  KEYSTORE (VPS Credentials)                                          │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🔐 Keystore Status: [SEALED/UNSEALED]                      │   │
│  │     Session expires: [time remaining]                       │   │
│  │                                                              │   │
│  │  [Unseal Keystore]  [Extend Session]                        │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Interaction Patterns

#### Gestures & Shortcuts

| Gesture | Context | Action |
|---------|---------|--------|
| **Swipe right** | Message | Reply to message |
| **Swipe left** | Message | React with emoji |
| **Long press** | Message | Context menu (copy, forward, delete) |
| **Long press** | Room in list | Quick actions (mute, archive, pin) |
| **Double tap** | Message | Quick reaction (❤️) |
| **Pull down** | Message list | Refresh/sync |
| **Swipe from edge** | Any screen | Navigate back |

#### Quick Actions

```
┌─────────────────────────────────────────────────────────────────────┐
│                      MESSAGE CONTEXT MENU                            │
│                    (Long press on message)                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  📋 Copy                                                    │   │
│  │  ↩️ Reply                                                   │   │
│  │  ↗️ Forward                                                 │   │
│  │  😊 React                                                   │   │
│  │  📌 Pin                                                     │   │
│  │  ⚠️ Report                                                  │   │
│  │  🗑️ Delete                                                  │   │
│  │  ℹ️ View Source                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### Reaction Picker

```
┌─────────────────────────────────────────────────────────────────────┐
│                       REACTION PICKER                                │
│                    (Tap reaction on message)                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│      👍    👎    ❤️    😂    😮    😢    👏    🔥    🎉    👀      │
│                                                                      │
│      [ frequently used reactions - horizontal scroll ]              │
│                                                                      │
│      [😀] [😃] [😄] [😁] [😅] [😂] [🤣] [😊] [😇] ...              │
│                                                                      │
│      [ full emoji picker - categorical tabs ]                       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Visual Design Language

#### Color System

```
┌─────────────────────────────────────────────────────────────────────┐
│                      ARMORCHAT COLOR SYSTEM                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  PRIMARY BRAND                                                       │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐                    │
│  │  Teal      │  │  Navy      │  │  NavyLight │                    │
│  │  #14F0C8   │  │  #0A1428   │  │  #1A2438   │                    │
│  │  Accent    │  │  Dark BG   │  │  Card BG   │                    │
│  └────────────┘  └────────────┘  └────────────┘                    │
│                                                                      │
│  SEMANTIC COLORS                                                     │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐    │
│  │  Success   │  │  Warning   │  │  Error     │  │  Info      │    │
│  │  #4CAF50   │  │  #FFB74D   │  │  #FF5252   │  │  #2196F3   │    │
│  │  Positive  │  │  Caution   │  │  Destruct  │  │  Neutral   │    │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘    │
│                                                                      │
│  SENSITIVITY COLORS (PII Classification)                            │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐    │
│  │  LOW       │  │  MEDIUM    │  │  HIGH      │  │  CRITICAL  │    │
│  │  #4CAF50   │  │  #FFB74D   │  │  #FF9800   │  │  #FF5252   │    │
│  │  Green     │  │  Amber     │  │  Orange    │  │  Red       │    │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘    │
│                                                                      │
│  PLATFORM COLORS (Bridged Platforms)                                │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐    │
│  │  Matrix    │  │  Slack     │  │  Discord   │  │  Teams     │    │
│  │  #0DBD8B   │  │  #4A154B   │  │  #5865F2   │  │  #6264A7   │    │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### Typography Scale

| Style | Size | Weight | Usage |
|-------|------|--------|-------|
| **Headline Large** | 32sp | Bold | Screen titles |
| **Headline Medium** | 24sp | SemiBold | Section headers |
| **Title Large** | 20sp | Medium | Card titles |
| **Title Medium** | 16sp | Medium | List item titles |
| **Body Large** | 16sp | Regular | Message content |
| **Body Medium** | 14sp | Regular | Secondary text |
| **Label** | 12sp | Medium | Captions, badges |
| **Caption** | 11sp | Regular | Timestamps, metadata |

#### Iconography

ArmorChat uses **Material Symbols** with consistent sizing:

| Size | Usage |
|------|-------|
| 48dp | Hero icons, empty states |
| 32dp | App bar actions |
| 24dp | Standard icons, list items |
| 20dp | Small indicators |
| 16dp | Inline icons, badges |

### Accessibility Features

#### Visual Accessibility

- **High Contrast Mode**: Enhanced text/background contrast
- **Font Scaling**: Respects system font size preferences
- **Color Blind Support**: Icons accompany all color-coded status
- **Dark Theme**: Reduces eye strain, saves battery

#### Interaction Accessibility

- **TalkBack Support**: Full screen reader compatibility
- **Switch Access**: Navigation without touch
- **Keyboard Navigation**: External keyboard support
- **Voice Input**: Speech-to-text for messages

#### Cognitive Accessibility

- **Clear Labels**: All buttons have descriptive text
- **Consistent Navigation**: Predictable UI patterns
- **Confirmation Dialogs**: Destructive actions require confirmation
- **Progress Indicators**: Loading states always visible

### Specialized UI Components

#### Agent Command Block

Used when AI agents perform tasks in chat:

```
┌─────────────────────────────────────────────────────────────────────┐
│                      AGENT COMMAND BLOCK                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🤖 Assistant                                    [RUNNING]  │   │
│  │                                                              │   │
│  │  Task: Fill insurance claim form                            │   │
│  │  Progress: ████████████░░░░░░░░ 60%                         │   │
│  │                                                              │   │
│  │  ┌────────────────────────────────────────────────────────┐ │   │
│  │  │  CAPABILITIES                                          │ │   │
│  │  │  [🌐 Web] [📝 Forms] [📋 Clipboard] [👁️ Screen]      │ │   │
│  │  └────────────────────────────────────────────────────────┘ │   │
│  │                                                              │   │
│  │  [Cancel Task]                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### BlindFill Card (PII Request)

When agent requests access to sensitive data:

```
┌─────────────────────────────────────────────────────────────────────┐
│                       BLINDFILL CARD                                 │
│              (PII Access Request Approval)                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🔐 Agent requests access to personal data                  │   │
│  │                                                              │   │
│  │  Agent: Assistant                                            │   │
│  │  Purpose: Auto-fill insurance form                           │   │
│  │                                                              │   │
│  │  ┌────────────────────────────────────────────────────────┐ │   │
│  │  │  Select fields to approve:                             │ │   │
│  │  │                                                        │ │   │
│  │  │  [✓] Full Name                           [LOW]         │ │   │
│  │  │  [✓] Email Address                       [LOW]         │ │   │
│  │  │  [ ] Date of Birth                       [MEDIUM]      │ │   │
│  │  │  [ ] Social Security Number              [CRITICAL]    │ │   │
│  │  │  [ ] Credit Card Number                  [CRITICAL]    │ │   │
│  │  └────────────────────────────────────────────────────────┘ │   │
│  │                                                              │   │
│  │  ⚠️ Tip: Only approve fields necessary for the task         │   │
│  │                                                              │   │
│  │  [Deny All]                        [Approve Selected]       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### Workflow Progress Banner

Shows active workflow status:

```
┌─────────────────────────────────────────────────────────────────────┐
│                   WORKFLOW PROGRESS BANNER                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  📋 Expense Report Workflow                      Step 2/5  │   │
│  │                                                              │   │
│  │  ●━━━━━○━━━━━○━━━━━○━━━━━○                                │   │
│  │  Done  Upload  Review  Approve  Complete                   │   │
│  │                                                              │   │
│  │  Current: Uploading receipt images...                       │   │
│  │                                                              │   │
│  │                                              [Cancel]       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### Keystore Status Indicator

Shows VPS keystore state:

```
┌─────────────────────────────────────────────────────────────────────┐
│                   KEYSTORE STATUS INDICATOR                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  SEALED State:                                                       │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🔐 Keystore Sealed                                         │   │
│  │     Credentials encrypted until needed                      │   │
│  │                                              [Unseal]       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  UNSEALED State:                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🔓 Keystore Unsealed                      ⏱️ 45 min left  │   │
│  │     Session active, agents can access credentials           │   │
│  │                                              [Extend]       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ERROR State:                                                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  ⚠️ Keystore Error                                          │   │
│  │     Failed to unseal: Invalid password                      │   │
│  │                                              [Retry]        │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Empty States & Onboarding

#### Empty Inbox

```
┌─────────────────────────────────────────────────────────────────────┐
│                       EMPTY INBOX STATE                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│                          ┌─────────────┐                            │
│                          │     💬      │                            │
│                          │   (large)   │                            │
│                          └─────────────┘                            │
│                                                                      │
│                    No conversations yet                              │
│                                                                      │
│           Start a new chat or join a room to begin                   │
│                                                                      │
│                    [Create Room]  [Join Room]                        │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### First Run Tips

```
┌─────────────────────────────────────────────────────────────────────┐
│                      FIRST RUN TOOLTIP                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  💡 Tip: Swipe left on a message to quickly react          │   │
│  │                                                              │   │
│  │                                              [Got it!]      │   │
│  │                                               ○ ○ ● ○       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Error Handling UI

#### Connection Error

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CONNECTION ERROR BANNER                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  ⚠️ Connection lost                                         │   │
│  │     Attempting to reconnect...                              │   │
│  │                                          [Retry Now]        │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### Encryption Warning

```
┌─────────────────────────────────────────────────────────────────────┐
│                   ENCRYPTION WARNING DIALOG                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🛡️  Unverified Encryption                                  │   │
│  │                                                              │   │
│  │  This room's encryption keys have not been verified.        │   │
│  │  Messages may not be fully secure.                          │   │
│  │                                                              │   │
│  │  [Verify Keys]                          [I Understand]      │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### How UI Elements Help Users

#### Security Visibility

| UI Element | Security Benefit |
|------------|------------------|
| 🔒 Lock icon | Confirms E2EE is active |
| ✓✓ Read receipts | Confirms message delivery |
| 🔀 Bridge badge | Shows message origin |
| ⚠️ Warning banners | Alerts to security issues |
| 🔐/🔓 Keystore indicator | Shows credential vault status |

#### Task Efficiency

| UI Element | Efficiency Benefit |
|------------|-------------------|
| Swipe gestures | Quick actions without menus |
| Long-press menus | Context-sensitive options |
| Filter chips | Fast room/message filtering |
| Search bar | Global search access |
| FAB (+) button | Quick room creation |

#### Agent Monitoring

| UI Element | Monitoring Benefit |
|------------|-------------------|
| Command Block | See what agent is doing |
| Progress bar | Track task completion |
| Capability ribbon | Know agent permissions |
| Status badge | Current task state |
| Cancel button | Stop unwanted actions |

#### Data Protection

| UI Element | Protection Benefit |
|------------|-------------------|
| Sensitivity badges | Understand data importance |
| BlindFill card | Approve specific fields only |
| Field checkboxes | Granular control |
| Critical highlighting | Extra caution for sensitive data |
| Deny button | Reject all access |

---

## 11.6 VPS Secretary Mode (Supervisory UI)

### The Paradigm Shift

ArmorChat is evolving from a **passive chat client** to an **active supervisory interface**. In the "VPS Secretary" model, AI agents work autonomously on a remote server while the user monitors and directs them via mobile.

```
┌─────────────────────────────────────────────────────────────────────┐
│                  PARADIGM COMPARISON                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  STANDARD CHAT CLIENT          vs       VPS SECRETARY MODE          │
│                                                                      │
│  ┌──────────────────┐                ┌──────────────────┐           │
│  │  Passive         │                │  Active          │           │
│  │  ────────        │                │  ──────          │           │
│  │  • Read messages │                │  • Monitor tasks │           │
│  │  • Reply to chat │                │  • Approve actions│          │
│  │  • Scroll history│                │  • Intervene     │           │
│  │                  │                │  • Direct agents │           │
│  └──────────────────┘                └──────────────────┘           │
│                                                                      │
│  User asks: "What's new?"            User asks: "Is it working?"   │
│  Interface: List of messages         Interface: Status dashboard    │
│  Action: Reply                       Action: Approve/Intervene      │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Mission Control Dashboard (Redesigned Home)

The home screen transforms from a simple room list into a **Mission Control** dashboard that immediately answers the user's primary question: *"Is my secretary working?"*

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MISSION CONTROL DASHBOARD                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  Good morning, User.                          ⚙️ Settings  │   │
│  │  You have 2 active tasks, 1 needs attention.                │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  🟢 ACTIVE TASKS (2)                               [View All →]     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  ✈️ Travel Booker                                  10:34 AM │   │
│  │  ────────────────────────────────────────────────────────── │   │
│  │  ⏳ Booking flight to NYC...                                │   │
│  │  Progress: ████████░░░░░░░░ 65%                            │   │
│  │                                                              │   │
│  │  [View Timeline]  [Pause]  [Cancel]                         │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  📊 Report Generator                               10:28 AM │   │
│  │  ────────────────────────────────────────────────────────── │   │
│  │  ✅ Completed: Q4 Financial Report                          │   │
│  │                                                              │   │
│  │  [View Result]  [Export PDF]                                │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  🔔 NEEDS ATTENTION (1)                            [View All →]     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  ⚠️ Contracts Agent                                10:15 AM │   │
│  │  ────────────────────────────────────────────────────────── │   │
│  │  🔐 Needs approval for $450 payment                         │   │
│  │  Vendor: Acme Corp                                          │   │
│  │                                                              │   │
│  │  [Review Now ← Requires Biometric]                          │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  +  Create New Agent Task                                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Dashboard Components:**

| Component | Purpose | User Benefit |
|-----------|---------|--------------|
| **Status Header** | Greeting + task summary | Instant status awareness |
| **Active Tasks** | Running agent jobs | Know what's being worked on |
| **Progress Bars** | Visual completion % | Estimate time remaining |
| **Attention Queue** | Items needing approval | Never miss critical decisions |
| **Quick Actions** | Pause/Cancel/View | Immediate control |

### Full-Screen Consent Experience

Critical PII approvals demand **focused attention**. Text messages in chat streams can be missed or scrolled past. The consent experience uses a **full-screen overlay** that interrupts the flow.

**UX Flow:**
```
┌─────────────────────────────────────────────────────────────────────┐
│                    CONSENT EXPERIENCE FLOW                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  STEP 1: Push Notification                                          │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🔔 Contracts Agent needs your approval                     │   │
│  │     Payment request for $450                                │   │
│  │                                           [Tap to review]   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                           │                                          │
│                           ▼                                          │
│  STEP 2: Full-Screen Consent Overlay                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  🔐 SECURE APPROVAL REQUIRED                                │   │
│  │  ────────────────────────────────────────────────────────── │   │
│  │                                                              │   │
│  │  Agent: Contracts Agent                                      │   │
│  │  Task: Pay Invoice #2024-0847                               │   │
│  │  Vendor: Acme Corporation                                    │   │
│  │                                                              │   │
│  │  ────────────────────────────────────────────────────────── │   │
│  │  REQUESTING ACCESS TO:                                       │   │
│  │                                                              │   │
│  │  • Client Name                           [LOW]    ✓         │   │
│  │  • Project Code                          [LOW]    ✓         │   │
│  │  • Contract Value                        [MEDIUM] ✓         │   │
│  │  • Credit Card Number                    [CRITICAL]         │   │
│  │  • Billing Address                       [MEDIUM]           │   │
│  │                                                              │   │
│  │  ────────────────────────────────────────────────────────── │   │
│  │  ⚠️ Critical fields require explicit selection              │   │
│  │                                                              │   │
│  │  ┌────────────────┐    ┌────────────────┐                   │   │
│  │  │    ❌ DENY     │    │   ✓ APPROVE    │                   │   │
│  │  │   (No auth)    │    │  (Biometric)   │                   │   │
│  │  └────────────────┘    └────────────────┘                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                           │                                          │
│                           ▼                                          │
│  STEP 3: Biometric Gate (on Approve)                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │                    👆                                        │   │
│  │              Touch sensor to approve                        │   │
│  │                                                              │   │
│  │  This binds approval to your physical presence              │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Security Features:**

| Feature | Purpose |
|---------|---------|
| **Full-screen takeover** | Cannot be dismissed without action |
| **Critical field highlighting** | Red/amber colors draw attention |
| **Explicit selection required** | Must tap each critical field |
| **Biometric binding** | Proves user was physically present |
| **Deny = No auth needed** | Low friction for rejection |

### Live Activity Timeline

The "Black Box" anxiety—users don't trust agents if they can't see work in progress. The **Timeline View** replaces simple message bubbles with a live activity stream.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    LIVE ACTIVITY TIMELINE                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Agent: ✈️ Travel Booker                               [Running]    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │  10:01  ✅ Navigated to united.com                          │   │
│  │         └── ✓ Page loaded successfully                      │   │
│  │                                                              │   │
│  │  10:02  ⏳ Filling passenger information...                 │   │
│  │         └── ● Currently processing                          │   │
│  │                                                              │   │
│  │  10:03  ⚠️ CAPTCHA DETECTED                                 │   │
│  │         ┌──────────────────────────────────────────────┐    │   │
│  │         │  [Screenshot Preview]                        │    │   │
│  │         │                                               │    │   │
│  │         │  "I need help with this CAPTCHA"             │    │   │
│  │         │                                               │    │   │
│  │         │  [View Full Screen]  [I'll handle it]        │    │   │
│  │         └──────────────────────────────────────────────┘    │   │
│  │                                                              │   │
│  │  10:04  ⏳ Waiting for user intervention...                │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  💬 Send intervention message...                  [Send]    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Timeline Event Types:**

| Icon | Status | Meaning |
|------|--------|---------|
| ✅ | Success | Step completed successfully |
| ⏳ | In Progress | Currently working on this step |
| ⚠️ | Intervention | Needs user help (CAPTCHA, 2FA) |
| ❌ | Failed | Step failed, needs retry |
| 🔄 | Retrying | Attempting again automatically |
| ⏸️ | Paused | User paused the task |

**Intervention Flow:**
```
┌─────────────────────────────────────────────────────────────────────┐
│                    INTERVENTION HANDLING                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Agent State: AWAITING_INTERVENTION                                  │
│                                                                      │
│  Options Available to User:                                          │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  1. VIEW SCREENSHOT                                         │   │
│  │     See exactly what the agent sees                         │   │
│  │                                                              │   │
│  │  2. TAKE OVER                                               │   │
│  │     Remote control the browser session                      │   │
│  │     (Agent pauses, user drives)                              │   │
│  │                                                              │   │
│  │  3. PROVIDE INFO                                            │   │
│  │     Type the CAPTCHA code / 2FA token                        │   │
│  │     Agent continues with provided input                      │   │
│  │                                                              │   │
│  │  4. CANCEL TASK                                             │   │
│  │     Stop the agent completely                                │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Voice-First Mode (Glass Interface)

The "Mobile Secretary" use case implies the user is mobile—walking, driving, shopping. Reading text is secondary. **Voice Mode** transforms the interface for eyes-free interaction.

```
┌─────────────────────────────────────────────────────────────────────┐
│                    VOICE-FIRST MODE                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  NORMAL MODE                          VOICE MODE                     │
│  ┌──────────────────┐                ┌──────────────────┐           │
│  │ Chat history     │                │                  │           │
│  │ visible          │    ──────▶     │    ┌────────┐    │           │
│  │                  │                │    │   🎤   │    │           │
│  │ Text input       │                │    │ LISTEN │    │           │
│  │ at bottom        │                │    └────────┘    │           │
│  └──────────────────┘                │                  │           │
│                                      │ Status: "Agent   │           │
│                                      │ is thinking..."  │           │
│                                      │                  │           │
│                                      │ [Tap for visual] │           │
│                                      └──────────────────┘           │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Voice Mode Features:**

| Feature | Behavior |
|---------|----------|
| **Auto-TTS** | All agent responses read aloud automatically |
| **Push-to-Talk** | Large "Hold to Speak" button |
| **Status Announcements** | "Agent started task: Booking flight" |
| **Intervention Alerts** | "Agent needs help with a CAPTCHA" |
| **Approval Requests** | "Approve $450 payment? Say 'Approve' or 'Deny'" |
| **Voice Confirmation** | "Approved. Payment processing." |

**Voice Mode States:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                      │
│     🎤 IDLE              🎙️ LISTENING            💭 THINKING        │
│     ──────────           ─────────────           ─────────────      │
│     Tap to speak         User is talking         Processing         │
│                            (animated)            speech              │
│                                                                      │
│     💬 SPEAKING          ⚠️ ATTENTION            🔔 NOTIFICATION    │
│     ────────────         ─────────────           ───────────────    │
│     Agent talking        Needs user              Background         │
│     (TTS active)         intervention            alert              │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### "Initialize Vault" Onboarding Flow

The architecture specifies **User-Held Keys**. The onboarding must communicate that *the user* holds the key, not the server. This sets expectations of **Ownership** from the first interaction.

```
┌─────────────────────────────────────────────────────────────────────┐
│                 INITIALIZE VAULT ONBOARDING                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  STEP 1: Welcome & Concept                                           │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │                        🔐                                    │   │
│  │                                                              │   │
│  │              Your Secrets, Your Control                      │   │
│  │                                                              │   │
│  │  ArmorChat uses a "Vault" to store your sensitive           │   │
│  │  credentials. Only YOU hold the key to this vault.          │   │
│  │                                                              │   │
│  │  • Your key never leaves your device                        │   │
│  │  • Not even we can access your vault                        │   │
│  │  • Your agents work on your behalf, securely                │   │
│  │                                                              │   │
│  │                        [Continue]                            │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  STEP 2: Create Master Key                                          │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │              Create Your Master Key                          │   │
│  │                                                              │   │
│  │  This key unlocks your secrets on the VPS.                  │   │
│  │  It is never sent to our servers.                           │   │
│  │                                                              │   │
│  │  ┌──────────────────────────────────────────────────────┐   │   │
│  │  │  Passphrase: ••••••••••••                            │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  │  ┌──────────────────────────────────────────────────────┐   │   │
│  │  │  Confirm:    ••••••••••••                            │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  │                                                              │   │
│  │  🔐 Add Biometric (Face/Fingerprint)?                       │   │
│  │  [Yes, Enable]  [Skip for Now]                              │   │
│  │                                                              │   │
│  │                        [Continue]                            │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  STEP 3: Recovery Phrase                                            │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │              Save Your Recovery Phrase                       │   │
│  │                                                              │   │
│  │  If you lose your device, this phrase restores your vault.  │   │
│  │  Write it down. Store it safely. Never share it.            │   │
│  │                                                              │   │
│  │  ┌──────────────────────────────────────────────────────┐   │   │
│  │  │  1. apple    4. river    7. cloud   10. music        │   │   │
│  │  │  2. dance    5. zebra    8. flame   11. tower        │   │   │
│  │  │  3. ghost    6. piano    9. storm   12. bread        │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  │                                                              │   │
│  │           [Copy to Clipboard]  [I've Saved It]              │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  STEP 4: Verify Recovery Phrase                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │              Verify Your Phrase                              │   │
│  │                                                              │   │
│  │  Tap the words in the correct order:                        │   │
│  │                                                              │   │
│  │  Word 3:  [ghost] ✓                                         │   │
│  │  Word 7:  [cloud] ✓                                         │   │
│  │  Word 11: [______]                                          │   │
│  │                                                              │   │
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐    │   │
│  │  │apple │ │tower │ │bread │ │zebra │ │dance │ │river │    │   │
│  │  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘    │   │
│  │                                                              │   │
│  │                        [Verify]                               │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  STEP 5: Sync with VPS                                              │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │              Connect to Your VPS                             │   │
│  │                                                              │   │
│  │  Scan the QR code displayed on your VPS terminal,           │   │
│  │  or enter the connection code manually.                     │   │
│  │                                                              │   │
│  │  ┌──────────────────────────────────────────────────────┐   │   │
│  │  │                                                       │   │   │
│  │  │              [QR Scanner View]                        │   │   │
│  │  │                                                       │   │   │
│  │  │          Align QR code within frame                   │   │   │
│  │  │                                                       │   │   │
│  │  └──────────────────────────────────────────────────────┘   │   │
│  │                                                              │   │
│  │           - or -                                             │   │
│  │                                                              │   │
│  │  Enter code: [____-____-____]                               │   │
│  │                                                              │   │
│  │                        [Connect]                             │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  STEP 6: Vault Ready                                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │                        ✅                                    │   │
│  │                                                              │   │
│  │               Your Vault is Ready                            │   │
│  │                                                              │   │
│  │  • Master key: ✓ Created                                     │   │
│  │  • Biometric: ✓ Enabled                                      │   │
│  │  • Recovery: ✓ Verified                                      │   │
│  │  • VPS Link: ✓ Connected                                     │   │
│  │                                                              │   │
│  │  Your agents can now securely access credentials            │   │
│  │  with your approval.                                         │   │
│  │                                                              │   │
│  │                   [Go to Dashboard]                          │   │
│  │                                                              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Summary: VPS Secretary UI Improvements

| Area | Standard Chat | VPS Secretary Mode |
|:-----|:--------------|:-------------------|
| **Home Screen** | Passive list of rooms | **Mission Control** dashboard with live status |
| **Consent** | Text-based approval in chat | **Full-screen overlay** with biometric gating |
| **Feedback** | "Black box" waiting for reply | **Live Timeline** with screenshots & intervention |
| **Input** | Text-centric | **Voice-First Mode** with TTS/STT |
| **Onboarding** | Generic "Sign Up" | **"Initialize Vault"** emphasizing user-held keys |

### Implementation Priority

| Priority | Feature | Complexity | Impact |
|----------|---------|------------|--------|
| **P0** | Mission Control Dashboard | Medium | High - Core paradigm shift |
| **P0** | Full-Screen Consent Overlay | Medium | Critical - Security UX |
| **P1** | Live Activity Timeline | High | High - Trust building |
| **P1** | Intervention Handling | High | High - User control |
| **P2** | Voice-First Mode | High | Medium - Accessibility |
| **P2** | Vault Onboarding Flow | Medium | High - User expectations |

---

## 12. Key Files Reference

### Critical Entry Points

| File | Purpose |
|------|---------|
| `androidApp/.../ArmorClawApplication.kt` | Application class, Koin initialization |
| `androidApp/.../MainActivity.kt` | Main activity, Compose setup |
| `androidApp/.../navigation/AppNavigation.kt` | 55 routes, NavHost setup |
| `androidApp/.../viewmodels/SplashViewModel.kt` | Startup routing logic |

### Authentication & Security

| File | Purpose |
|------|---------|
| `androidApp/.../screens/auth/LoginScreen.kt` | Login UI |
| `androidApp/.../screens/auth/KeyRecoveryScreen.kt` | Key recovery flow |
| `androidApp/.../platform/security/KeystoreManager.kt` | Hardware key storage |
| `androidApp/.../platform/security/SqlCipherProvider.kt` | Encrypted database |
| `shared/.../platform/encryption/EncryptionTypes.kt` | Encryption type definitions |

### Messaging

| File | Purpose |
|------|---------|
| `shared/.../platform/matrix/MatrixClient.kt` | Matrix client interface |
| `shared/.../platform/matrix/MatrixClientImpl.kt` | HTTP implementation |
| `androidApp/.../screens/chat/ChatScreen_enhanced.kt` | Main chat UI |
| `androidApp/.../viewmodels/ChatViewModel.kt` | Chat state management |
| `shared/.../domain/model/UnifiedMessage.kt` | Message model |

### Bridge Communication

| File | Purpose |
|------|---------|
| `shared/.../platform/bridge/BridgeRpcClient.kt` | RPC interface |
| `shared/.../platform/bridge/BridgeRpcClientImpl.kt` | RPC implementation |
| `shared/.../platform/bridge/SetupService.kt` | Server setup/provisioning |
| `shared/.../platform/bridge/RpcModels.kt` | RPC request/response models |

### Push Notifications

| File | Purpose |
|------|---------|
| `androidApp/.../service/FirebaseMessagingService.kt` | FCM message handling |
| `shared/.../platform/notification/PushNotificationRepository.kt` | Dual registration |

### Control Plane (AI Agent Monitoring)

| File | Purpose |
|------|---------|
| `shared/.../data/store/ControlPlaneStore.kt` | Agent/PII/Keystore state |
| `shared/.../domain/model/AgentStatusEvent.kt` | Agent task status model |
| `shared/.../domain/model/PiiAccessRequest.kt` | PII request model |
| `shared/.../domain/model/KeystoreStatus.kt` | VPS keystore state |
| `androidApp/.../data/BiometricAuthorizer.kt` | Biometric auth service |
| `androidApp/.../screens/keystore/UnsealScreen.kt` | Keystore unseal UI |
| `androidApp/.../viewmodels/UnsealViewModel.kt` | Unseal state management |

### Data Layer

| File | Purpose |
|------|---------|
| `shared/.../data/store/ControlPlaneStore.kt` | Central state store |
| `androidApp/.../viewmodels/AppPreferences.kt` | SharedPreferences wrapper |
| `androidApp/.../viewmodels/DeviceListViewModel.kt` | Device management |
| `androidApp/.../viewmodels/SyncStatusViewModel.kt` | Sync state tracking |

---

## 13. ArmorClaw Compatibility

### Integration Status: ✅ Production Ready

All critical integration gaps, architecture issues, and ArmorClaw 0.3.4 spec alignment issues have been resolved.

### Resolved Integration Gaps

| Gap ID | Issue | Resolution |
|--------|-------|------------|
| G-01 | Push notifications dropped | Real HTTP pusher + dual registration |
| G-03 | Bridge verification undiscoverable | Banner in Room Details |
| G-07 | Key backup not surfaced | 6-step onboarding flow |
| G-09 | No migration path v2.5→v4.1 | Auto-detect + legacy wipe |
| A-01 | Dead EncryptionService | Stripped, Rust SDK only |
| A-02 | Bridged users indistinguishable | Origin badges |
| A-03 | Edit/React in unsupported rooms | Capability suppression |
| A-04 | No governance UI | Banners + placeholders |
| A-05 | Key recovery from login | Recovery screen entry |

### Spec Alignment (0.3.4)

| Field | Consumer | Status |
|-------|----------|--------|
| `healthy`/`status` | Dual-check fallback | ✅ |
| `bridge_ready` | NOT_READY gating | ✅ |
| `provisioning_available` | BridgeHealthDetails | ✅ |
| `is_new_server` | SetupUiState | ✅ |
| `admin_token` | SetupCompleteInfo | ✅ |
| QR `version`/`bridge_public_key` | SignedServerConfig | ✅ |

### Feature Suppression Matrix

| Platform | Can Edit | Can React | Can Thread | Can Reply |
|----------|----------|-----------|------------|-----------|
| Matrix | ✅ | ✅ | ✅ | ✅ |
| Slack | ✅ | ✅ | ✅ | ✅ |
| Discord | ✅ | ✅ | ✅ | ✅ |
| Teams | ❌ | ✅ | ❌ | ✅ |
| Signal | ❌ | ❌ | ❌ | ✅ |
| WhatsApp | ❌ | ❌ | ❌ | ✅ |

### Phase 2 Implementation Status

#### Agent Status (✅ Implemented)

| Component | File | Status |
|-----------|------|--------|
| Domain Models | `shared/.../domain/model/AgentStatusHistory.kt` | ✅ |
| RPC Interface | `shared/.../platform/bridge/BridgeRpcClient.kt` | ✅ |
| RPC Implementation | `shared/.../platform/bridge/BridgeRpcClientImpl.kt` | ✅ |
| Admin Interface | `shared/.../platform/bridge/BridgeAdminClient.kt` | ✅ |
| Admin Implementation | `shared/.../platform/bridge/BridgeAdminClientImpl.kt` | ✅ |
| UI Component | `shared/.../ui/components/AgentStatusBanner.kt` | ✅ |
| Control Plane Store | `shared/.../data/store/ControlPlaneStore.kt` | ✅ |

**Available Methods:**
- `agentGetStatus(agentId)` - Get current agent status
- `agentStatusHistory(agentId, limit)` - Get status change history
- `subscribeToAgentStatus(agentId)` - Flow for real-time status updates
- `subscribeToAllAgentStatuses()` - Flow for all agent status changes

#### Keystore / Zero-Trust (✅ Implemented)

| Component | File | Status |
|-----------|------|--------|
| Domain Models | `shared/.../domain/model/KeystoreUnseal.kt` | ✅ |
| Domain Models | `shared/.../domain/model/KeystoreStatus.kt` | ✅ |
| RPC Interface | `shared/.../platform/bridge/BridgeRpcClient.kt` | ✅ |
| RPC Implementation | `shared/.../platform/bridge/BridgeRpcClientImpl.kt` | ✅ |
| Admin Interface | `shared/.../platform/bridge/BridgeAdminClient.kt` | ✅ |
| Admin Implementation | `shared/.../platform/bridge/BridgeAdminClientImpl.kt` | ✅ |
| UI Component | `shared/.../ui/components/SealedIndicator.kt` | ✅ |
| Unseal Screen | `androidApp/.../screens/keystore/UnsealScreen.kt` | ✅ |
| Unseal ViewModel | `androidApp/.../viewmodels/UnsealViewModel.kt` | ✅ |

**Available Methods:**
- `keystoreSealed()` - Check if keystore is sealed
- `keystoreUnsealChallenge()` - Get challenge for unsealing
- `keystoreUnsealRespond(request)` - Respond with wrapped key
- `keystoreExtendSession()` - Extend unsealed session
- `subscribeToKeystoreState()` - Flow for keystore state changes

#### PII Access Requests (✅ Domain Ready)

| Component | File | Status |
|-----------|------|--------|
| Domain Models | `shared/.../domain/model/PiiAccessRequest.kt` | ✅ |
| UI Component | `shared/.../ui/components/BlindFillCard.kt` | ✅ |
| UI Component | `shared/.../ui/components/SensitivityBadge.kt` | ✅ |
| Control Plane Store | `shared/.../data/store/ControlPlaneStore.kt` | ✅ |

#### Pending Integration

| Feature | Status | Notes |
|---------|--------|-------|
| Agent Status in ChatScreen | ✅ | AgentTaskStatusBanner wired in ChatScreenEnhanced |
| PII Request Handling | ✅ | ChatViewModel has approvePiiRequest/denyPiiRequest |
| Keystore Navigation | ✅ | UnsealScreen registered at route KEYSTORE |

---

## Appendix: Build Commands

```bash
# Build
./gradlew assembleDebug          # Debug APK
./gradlew assembleRelease        # Release APK

# Testing
./gradlew test                   # Unit tests
./gradlew connectedAndroidTest   # Instrumented tests

# Installation
./gradlew installDebug           # Install on device/emulator

# Clean
./gradlew clean
```

---

*Last Updated: 2026-02-28*
*Document Version: 2.0.0*
