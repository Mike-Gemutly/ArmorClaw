# ArmorClaw Application Review

> **Document Purpose:** Comprehensive summary of ArmorClaw's structure, features, functions, user stories, and purpose.
>
> **Last Updated:** 2026-02-26
> **App Version:** 4.1.1-alpha01
> **Platform:** Android (Kotlin Multiplatform)
> **Target Protocol:** Matrix (matrix.org)
> **Architecture Status:** ✅ Production Ready with Bridge Integration + Setup Flow + Invite System + Security Hardening + Discovery System + ArmorClaw Gap Fixes + UX Polish
> **Build Status:** ✅ All critical bugs resolved, clean build
> **Discovery System:** ✅ Complete - See Section 0.5
> **ArmorClaw Integration:** ✅ All 16 gaps/issues resolved — See `doc/output/review.md`
> **Play Store Ready:** ✅ Documentation complete — See `docs/GOOGLE_PLAY_REQUIREMENTS.md`

---

## 0. Architecture Overview

> **Reference:** See `MATRIX_MIGRATION.md` for the complete migration history.

### Current Architecture (Production)
```
ArmorChat (Matrix Client) ↔ Conduit ↔ ArmorClaw (Bridge/Agent)
```

- ArmorChat communicates with Matrix via the Rust SDK (thick client)
- ArmorClaw Bridge handles platform bridging, provisioning, and agent orchestration
- Legacy Bridge-only RPC methods are deprecated (see `BridgeRpcClient`)
- Migration path from v2.5 (Bridge-only) to v3.0 (Matrix-native) via `MigrationScreen`

---

## 0.5. Server Discovery: How ArmorChat Finds ArmorClaw

ArmorChat uses a **multi-layered discovery system** to find and connect to ArmorClaw Bridge servers. This section explains the complete discovery flow.

### Discovery Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                     ARMORCHAT SERVER DISCOVERY FLOW                              │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  DISCOVERY PRIORITY (Highest to Lowest Trust):                                   │
│                                                                                  │
│  1. SIGNED QR/DEEP LINK ────────────────────────────────────────────────         │
│     • User scans QR code or taps deep link                                      │
│     • Format: armorclaw://config?d=<base64-encoded-json>                         │
│     • Contains: matrix_homeserver, rpc_url, ws_url, push_gateway, server_name   │
│     • Cryptographically signed by Bridge, expires after 24h                      │
│     • HIGHEST TRUST: Pre-verified by Bridge server                               │
│                                                                                  │
│  2. WELL-KNOWN DISCOVERY ───────────────────────────────────────────────        │
│     • Standard Matrix discovery mechanism                                        │
│     • User enters domain: "armorclaw.app"                                        │
│     • App fetches: https://armorclaw.app/.well-known/matrix/client              │
|     • Response includes Bridge URLs under "com.armorclaw" key                  │
│                                                                                  │
│  3. mDNS DISCOVERY (Local Network) ─────────────────────────────────────        │
│     • App scans for _armorclaw._tcp.local                                       │
│     • Bridge advertises itself via mDNS/Bonjour                                 │
│     • Auto-discovers IP, port, and basic config                                 │
│     • May require QR scan if Matrix URL not included                            │
│     • Great for home/office deployments                                          │
│                                                                                  │
│  4. MANUAL ENTRY ───────────────────────────────────────────────────────        │
│     • User enters Matrix homeserver URL                                          │
│     • App derives Bridge URL (matrix.example.com → bridge.example.com)          │
│     • Fallback when other discovery methods fail                                 │
│                                                                                  │
│  5. FALLBACK SERVERS ───────────────────────────────────────────────────        │
│     • Pre-configured backup servers:                                             │
│       - bridge.armorclaw.app                                                     │
│       - bridge-backup.armorclaw.app                                              │
│       - bridge-eu.armorclaw.app                                                  │
│                                                                                  │
│  6. BUILDCONFIG DEFAULTS ───────────────────────────────────────────────        │
│     • Production defaults compiled into app:                                     │
│       - MATRIX_HOMESERVER = "https://matrix.armorclaw.app"                       │
│       - ARMORCLAW_RPC_URL = "https://bridge.armorclaw.app/api"                   │
│       - ARMORCLAW_WS_URL = "wss://bridge.armorclaw.app/ws"                       │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Signed QR Code / Deep Link Format

The most trusted discovery method uses signed configuration embedded in QR codes or deep links:

```
armorclaw://config?d=<base64-encoded-json>
https://armorclaw.app/config?d=<base64-encoded-json>
```

**Decoded JSON Structure:**
```json
{
  "version": 1,
  "matrix_homeserver": "https://matrix.armorclaw.app",
  "rpc_url": "https://bridge.armorclaw.app/api",
  "ws_url": "wss://bridge.armorclaw.app/ws",
  "push_gateway": "https://bridge.armorclaw.app/_matrix/push/v1/notify",
  "server_name": "ArmorClaw",
  "region": "us-east",
  "expires_at": 1708123460,
  "signature": "..."
}
```

### Well-Known Discovery Response

Standard Matrix well-known with Bridge extension:

```json
{
  "m.homeserver": {
    "base_url": "https://matrix.armorclaw.app"
  },
  "m.identity_server": {
    "base_url": "https://matrix.armorclaw.app"
  },
  "com.armorclaw": {
    "base_url": "https://bridge.armorclaw.app",
    "api_endpoint": "https://bridge.armorclaw.app/api",
    "ws_endpoint": "wss://bridge.armorclaw.app/ws",
    "push_gateway": "https://bridge.armorclaw.app/_matrix/push/v1/notify"
  }
}
```

### Bridge HTTP Discovery Endpoints

The Bridge server exposes these discovery endpoints:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/.well-known/matrix/client` | GET | Matrix standard discovery |
| `/discover` | GET | Bridge discovery info (IP, port, fingerprint) |
| `/qr/config` | GET | Generate signed config QR for ArmorChat |
| `/qr/image` | GET | QR code PNG image |
| `/health` | GET | Health check for connectivity testing |

### Deep Link Schemes

ArmorChat handles these deep link schemes:

| Scheme | Format | Purpose |
|--------|--------|---------|
| Config | `armorclaw://config?d=<base64>` | Signed server configuration |
| Setup | `armorclaw://setup?token=xxx&server=xxx` | Device setup with token |
| Invite | `armorclaw://invite?code=xxx` | Room/server invite |
| Bond | `armorclaw://bond?token=xxx` | Device bonding (admin pairing) |
| Web Config | `https://armorclaw.app/config?d=<base64>` | Web-based config |
| Web Invite | `https://armorclaw.app/invite/<code>` | Web-based invite |

### Discovery Implementation

**Key Classes:**
- `SetupService` - Orchestrates discovery flow
- `SetupConfig` - Holds discovered configuration
- `ConfigSource` - Enum tracking how config was obtained
- `DeepLinkHandler` - Parses and routes deep links

**Discovery Flow in Code:**
```kotlin
// Priority order
when {
    input.startsWith("armorclaw://") -> parseSignedConfig(input)
    input.startsWith("https://armorclaw.app") -> parseSignedConfig(input)
    isDomain(input) -> tryWellKnownDiscovery(input)
    isUrl(input) -> deriveBridgeUrl(input)
    else -> useBuildConfigDefaults()
}
```

### Configuration Storage

Discovered configuration is persisted in Android SharedPreferences:

| Key | Description |
|-----|-------------|
| `matrix_homeserver` | Matrix server URL |
| `bridge_url` | Bridge RPC URL (/api) |
| `ws_url` | WebSocket URL (/ws) |
| `push_gateway` | Push notification gateway |
| `server_name` | Human-readable server name |
| `config_source` | How config was obtained (enum) |
| `expires_at` | Config expiration timestamp |

### Self-Hosted / OpenClaw Considerations

For self-hosted deployments (OpenClaw):

1. **Well-Known**: Configure `/.well-known/matrix/client` on your domain
2. **mDNS**: Enable mDNS advertisement on Bridge
3. **QR Codes**: Bridge generates self-signed QR configs
4. **Manual**: Users can manually enter server URLs
5. **Self-Signed Certs**: ArmorChat warns but allows connection

### Correct URLs (Important!)

| Service | Correct URL | ❌ Wrong (Legacy) |
|---------|-------------|------------------|
| Matrix | `https://matrix.armorclaw.app` | `.com` |
| Bridge RPC | `https://bridge.armorclaw.app/api` | `/rpc` |
| Bridge WS | `wss://bridge.armorclaw.app/ws` | - |
| Push Gateway | `https://bridge.armorclaw.app/_matrix/push/v1/notify` | `/push` |

---

## Executive Summary

ArmorClaw is a **secure end-to-end encrypted chat application** designed for privacy-conscious users who require confidential communication. Built with modern Android development practices, it combines enterprise-grade security with an intuitive user experience.

### Core Value Proposition
- **Privacy by Design**: All messages are end-to-end encrypted using AES-256-GCM
- **Open Standards**: Built on the Matrix protocol for interoperability
- **User Control**: Users own their data with full export and deletion capabilities
- **Offline-First**: Full functionality even without internet connectivity

### Setup Flow Review (2026-02-26)

This update fixes 8 issues in the ArmorClaw setup process, ensuring flawless first-time connection with excellent error handling:

| Bug # | Issue | Severity | Status |
|-------|-------|----------|--------|
| **#10** | Advanced connect race condition — `connectWithCredentials` never called | CRITICAL | ✅ Fixed |
| **#11** | QR provisioning bypassed mandatory bridge health check | CRITICAL | ✅ Fixed |
| **#12** | `isIpAddress()` false positives on digit-starting domains | MEDIUM | ✅ Fixed |
| **#13** | URL decoding order bug in `decodeUrlPart()` | MEDIUM | ✅ Fixed |
| **#14** | Camera executor resource leak in QR scanner | LOW | ✅ Fixed |
| **#15** | PermissionsScreen not requesting real Android permissions | HIGH | ✅ Fixed |
| **#16** | No health-check gate on `connectWithCredentials` | HIGH | ✅ Fixed |
| **#17** | Missing defense-in-depth across all setup entry points | MEDIUM | ✅ Fixed |

#### Bug Fix #10: Advanced Connect Race Condition
**Problem:** `AdvancedSetupScreen.onConnect` called `viewModel.startSetup()` (async coroutine) then immediately checked `uiState.canProceed` — always `false` because the coroutine hadn't completed. `connectWithCredentials` was **never called**.
**Solution:** Two-phase connect pattern: store pending credentials in local state, then use `LaunchedEffect` to observe `canProceed` and auto-call `connectWithCredentials` when it becomes `true`.

#### Bug Fix #11: QR Provisioning Health Bypass
**Problem:** Bridge health gating (CTO requirement) was enforced in `startSetup()` but completely bypassed by `handleQrProvision()` and `handleQrProvisionWithAuth()`. QR-scanned setup could proceed with an unreachable bridge.
**Solution:** Added `performBridgeHealthCheck()` to both QR methods. Added defense-in-depth guard to `connectWithCredentials()`.

#### Bug Fix #12: IP Address False Positives
**Problem:** Heuristic `host.first().isDigit() && host.contains(".")` matched domains like `1password.com`, causing incorrect bridge URL derivation.
**Solution:** Removed heuristic. Now only strict IPv4 regex is used. Applied in all 3 copies (SetupService, SetupViewModel, ConnectServerScreen).

#### Bug Fix #13: URL Decoding Order Bug
**Problem:** Sequential `.replace()` chain in `decodeUrlPart()` had ordering dependencies — `%25` decoded mid-sequence could corrupt subsequent replacements.
**Solution:** Replaced with single-pass regex decoder: `Regex("%([0-9A-Fa-f]{2})")`.

#### Bug Fix #14: Camera Executor Leak
**Problem:** `ConnectServerScreen` created `Executors.newSingleThreadExecutor()` via `remember {}` but never shut it down.
**Solution:** Added `DisposableEffect(Unit) { onDispose { executor.shutdown(); barcodeScanner.close() } }`.

#### Bug Fix #15: PermissionsScreen Not Requesting Real Permissions
**Problem:** Grant buttons only toggled a local `granted` boolean — never called Android's `ActivityResultContracts.RequestPermission()`.
**Solution:** Added `rememberLauncherForActivityResult` with `RequestPermission` contract, `LaunchedEffect` for already-granted permissions, and `permissionIdToManifest()` helper.

### Critical Integration Update (2026-02-24)

This update resolves 4 critical ArmorClaw integration gaps identified during compatibility review:

| Bug # | Issue | Severity | Status |
|-------|-------|----------|--------|
| **#6** | Push Notification Dual Registration — Matrix pusher missing | CRITICAL | ✅ Fixed |
| **#7** | Bridge Verification UX — No discoverable entry point | HIGH | ✅ Fixed |
| **#8** | User Migration v2.5→v3.0 — No upgrade path | CRITICAL | ✅ Fixed |
| **#9** | Key Backup Setup — Never surfaced in UI | HIGH | ✅ Fixed |

#### Bug Fix #6: Push Notification Dual Registration
**Problem:** `PushNotificationRepositoryImpl` only registered FCM tokens with Bridge RPC, not with the Matrix homeserver. Push notifications for Matrix-native events were silently dropped.
**Solution:** Refactored `PushNotificationRepositoryImpl` to accept both `BridgeRpcClient` and `MatrixClient`. Added `setPusher()`/`removePusher()` to `MatrixClient` interface. Dual registration on every token refresh with graceful partial-failure handling.

#### Bug Fix #7: Bridge Verification UX
**Problem:** `EmojiVerificationScreen` existed but had no UI entry point — users couldn't initiate bridge device verification.
**Solution:** Added `BridgeVerificationBanner` composable to `RoomDetailsScreen` with warning banner and "Verify Bridge Device" button. Added `BRIDGE_VERIFICATION` route.

#### Bug Fix #8: User Migration v2.5→v3.0
**Problem:** No migration path for users upgrading from legacy Bridge-only sessions to Matrix-native architecture.
**Solution:** Created `MigrationScreen.kt` with auto-detection, manual recovery phrase entry, and success/failure states. Added `SplashTarget.Migration` to startup state machine. Added `hasLegacyBridgeSession()` to `AppPreferences`.

#### Bug Fix #9: Key Backup Setup
**Problem:** Matrix SDK key backup was never surfaced in the app. Device loss = permanent message history loss.
**Solution:** Created `KeyBackupSetupScreen.kt` with 6-step guided flow (Explain→Generate→Display→Verify→Store→Success). Mandatory in onboarding, optional re-entry from Security Settings.

### Critical Security Update (2026-02-18)

This update addresses 5 critical security and reliability issues identified during code review:

| Bug # | Issue | Severity | Status |
|-------|-------|----------|--------|
| **#1** | Ghost Encryption Layer - Encryption status unclear | HIGH | ✅ Fixed |
| **#2** | Push Notification Placeholder - No FCM integration | HIGH | ✅ Fixed |
| **#3** | Admin Invite Race Condition - First-user detection flawed | MEDIUM | ✅ Fixed |
| **#4** | Deep Link Security - No validation of external links | MEDIUM | ✅ Fixed |
| **#5** | Sync State Deadlock - State machine could hang | HIGH | ✅ Fixed |

### Recent Improvements (2026-02-18)

#### Bug Fix #1: Ghost Encryption Layer
**Problem:** App showed encryption indicators but had no actual client-side encryption implementation.

**Solution:** 
- Documented encryption trust models (Server-Side vs Client-Side)
- Added `EncryptionMode` enum: `SERVER_SIDE`, `CLIENT_SIDE`, `NONE`
- Added `RoomEncryptionStatus` for accurate UI representation
- Documented path to Client-Side E2EE via vodozemac (future)

**Note:** The original `EncryptionService` (expect/actual) was later removed (see Lesson 15) as it was a pass-through that created confusion about encryption ownership. Type definitions retained in `EncryptionTypes.kt`.

#### Bug Fix #2: Push Notification Placeholder
**Problem:** `registerDevice()` was a stub, FCM messages were not processed.

**Solution:**
- Implemented `ArmorFirebaseMessagingService` with full FCM handling
- Created `PushNotificationRepository` for Bridge API integration
- Added `push.register`, `push.unregister`, `push.updateSettings` RPC methods
- Integrated with Koin DI for proper dependency injection
- Added notification channels (messages, alerts, calls)

**Files Changed:**
- `androidApp/src/main/kotlin/com/armorclaw/app/service/ArmorFirebaseMessagingService.kt` (new)
- `shared/src/commonMain/kotlin/platform/notification/PushNotificationRepository.kt` (new)
- `shared/src/commonMain/kotlin/platform/bridge/BridgeRpcClient.kt` (updated)
- `shared/src/commonMain/kotlin/platform/bridge/BridgeRpcClientImpl.kt` (updated)
- `shared/src/commonMain/kotlin/platform/bridge/RpcModels.kt` (updated)

#### Bug Fix #3: Admin Invite Race Condition
**Problem:** First-user admin detection used client-side `messageCount == 0` check, causing race conditions when multiple users connect simultaneously.

**Solution:**
- Moved admin privilege detection to server-side
- Added `userRole` field to `BridgeStatusResponse`
- Created `getUserPrivilegesFromServer()` method
- Deprecated client-side `checkUserPrivileges()` method
- Server now authoritatively assigns roles (OWNER, ADMIN, MODERATOR, NONE)

**Files Changed:**
- `shared/src/commonMain/kotlin/platform/bridge/SetupService.kt` (updated)
- `shared/src/commonMain/kotlin/platform/bridge/RpcModels.kt` (updated)

#### Bug Fix #4: Deep Link Security
**Problem:** Deep links from external sources had no validation, allowing potential phishing attacks.

**Solution:**
- Added `DeepLinkResult` sealed class: `Valid`, `RequiresConfirmation`, `Invalid`
- Implemented URI validation (scheme, host, length, path segments)
- Added known safe hosts whitelist for HTTPS links
- Added room ID and user ID format validation
- Created `DeepLinkConfirmationDialog` for user confirmation
- Updated `MainActivity` to handle security states

**Files Changed:**
- `androidApp/src/main/kotlin/com/armorclaw/app/navigation/DeepLinkHandler.kt` (rewritten)
- `androidApp/src/main/kotlin/com/armorclaw/app/components/DeepLinkConfirmationDialog.kt` (new)
- `androidApp/src/main/kotlin/com/armorclaw/app/MainActivity.kt` (updated)

#### Bug Fix #5: Sync State Deadlock
**Problem:** WebSocket state machine could get stuck in transient states (CONNECTING/DISCONNECTING) during network errors.

**Solution:**
- Added guaranteed state reset in `handleDisconnection()`
- Added `finally` block in `listenForEvents()` for consistent cleanup
- Fixed `subscribeToRoom()` and `unsubscribeFromRoom()` state consistency
- Added null-safety check at start of event listening

**Files Changed:**
- `shared/src/commonMain/kotlin/platform/bridge/BridgeWebSocketClientImpl.kt` (updated)

### Previous Improvements (2026-02-16)
- ✅ **Bridge Service Tests**: Comprehensive test suite added (`BridgeServiceTest.kt`)
  - InviteServiceTest: 8 tests for invite link generation and validation
  - SetupServiceTest: 4 tests for setup flow
  - SecurityWarningTest: 3 tests for warning types and severities
  - InviteExpirationTest: 1 test for duration parsing
  - AdminLevelTest: 1 test for admin levels
  - SetupStateTest: 3 tests for setup state transitions
  - Mock implementations for BridgeRpcClient and BridgeWebSocketClient
- ✅ **Code Quality Fixes**: Fixed compilation issues in InviteService.kt and SetupService.kt
  - Proper Duration imports (kotlin.time.Duration)
  - Logger API compatibility fixes
  - Null safety improvements

### Previous Improvements (2026-02-15)
- ✅ **Deep Link Support**: Full implementation with `DeepLinkHandler.kt`
  - `armorclaw://room/{id}`, `armorclaw://user/{id}`, `armorclaw://call/{id}`
  - `https://matrix.to/#/{roomId}` and `https://chat.armorclaw.app/room/{id}`
  - `MainActivity.onNewIntent()` handling for warm/hot deep link starts
- ✅ **Profile State Persistence**: `ProfileViewModel` with proper state management
  - State survives configuration changes (rotation, etc.)
  - Connected to `UserRepository` for data loading/saving
- ✅ **Logout Session Clearing**: Connected `ProfileScreen` to `LogoutUseCase`
  - Proper session cleanup: stops sync, clears tokens, clears local data
  - Navigation to login screen after successful logout
- ✅ **Navigation Error Handling**: `NavigationExtensions.kt` provides safe navigation
  - `navigateSafely()`, `popBackStackSafely()`, `navigateUpSafely()`
  - Error logging and user feedback via snackbar
- ✅ **All 15 User Journey Gaps Resolved**: FEATURE_REVIEW.md updated

### Previous Improvements (2026-02-14)
- ✅ **Repository Layer Complete**: MessageRepositoryImpl and RoomRepositoryImpl with full CRUD operations
- ✅ **Unified Logging**: All data layer components now use structured AppLogger with metadata
- ✅ **ChatViewModel Refactored**: Removed simulated data, now uses repository pattern with pagination
- ✅ **Build Status**: All compilation errors resolved, clean build

### Bridge Integration (2026-02-15)
ArmorChat now integrates with the **ArmorClaw Go Bridge Server** for secure Matrix protocol communication.

- ✅ **BridgeRepository**: Integration layer between domain repositories and bridge clients
  - Handles bridge lifecycle (start, stop)
  - Matrix login and authentication
  - Message sending/receiving via RPC
  - Room management (create, join, leave, invite)
  - Real-time event processing via WebSocket

- ✅ **BridgeRpcClient**: JSON-RPC 2.0 client for bridge communication
  - **58 RPC methods** across 15 categories:
    - Bridge Lifecycle (4), Matrix (10, deprecated), Provisioning (5), WebRTC (8)
    - Recovery (6), Platform (5), Push (3), License (3), Compliance (2)
    - Error Mgmt (2), Agent (3), Workflow (3), HITL (3), Budget (1)

- ✅ **BridgeWebSocketClient**: Real-time event streaming
  - **12 event types** for live updates:
    - `message.received`, `message.status`
    - `room.created`, `room.membership`
    - `typing`, `receipt.read`
    - `presence`, `call`
    - `platform.message`, `session.expired`
    - `bridge.status`, `recovery`
  - Automatic reconnection with exponential backoff
  - Keep-alive pings for connection health

- ✅ **RpcModels**: Complete RPC request/response models
  - `JsonRpcRequest`, `JsonRpcResponse`, `JsonRpcError`
  - `BridgeStartResponse`, `BridgeStatusResponse`, `BridgeStopResponse`
  - `MatrixLoginResponse`, `MatrixSyncResponse`, `MatrixSendResponse`
  - `MatrixCreateRoomResponse`, `MatrixJoinRoomResponse`
  - `WebRtcSignalingResponse`, `IceCandidate`
  - `RecoveryPhraseResponse`, `RecoveryVerifyResponse`
  - `PlatformConnectResponse`, `PlatformStatusResponse`

- ✅ **BridgeEvent**: Sealed class hierarchy for WebSocket events
  - Type-safe event handling
  - Full kotlinx.serialization support
  - Unknown event forwarding for forward compatibility

### Setup Flow (2026-02-15, updated 2026-02-26)
ArmorChat provides a flawless first-time setup experience with QR-first onboarding, mandatory bridge health gating, and comprehensive error handling.

**Onboarding Flow:**
```
Welcome → Security → ConnectServer (QR-first) → Permissions → KeyBackup → Home
                          |
                   +------+------+
                   | QR Scan     |---> Health Check -> Credentials -> Connected
                   | (default)   |
                   +-------------+
                   | Advanced    |---> Manual URL -> Health Check -> Credentials
                   | (toggle)    |    (two-phase connect pattern)
                   +------+------+
```

- ✅ **SetupService**: Orchestrates initial connection with fallbacks
  - Server detection and mandatory health checks on ALL paths
  - 3 fallback servers with automatic failover
  - Progress tracking through 10 setup steps
  - Single-pass URL decoding (regex-based, no ordering bugs)
  - Strict IPv4 regex for IP address detection (no heuristic false positives)

- ✅ **SetupViewModel**: UI state management for setup screens
  - QR provisioning with health check gating
  - Two-phase connect pattern for Advanced mode (async-safe)
  - Reactive state for Compose UI via `StateFlow`
  - Error handling with fallback options

- ✅ **ConnectServerScreen**: QR-first setup UI
  - QR scanner as default entry point (camera with barcode detection)
  - Advanced mode toggle for manual URL entry
  - Two-phase connect: pending credentials -> `LaunchedEffect` observes `canProceed`
  - Progress indicator, security warnings, admin notification
  - Camera executor properly cleaned up via `DisposableEffect`

- ✅ **PermissionsScreen**: Real Android permission requests
  - `rememberLauncherForActivityResult` with `RequestPermission` contract
  - `LaunchedEffect` checks already-granted permissions on composition

- ✅ **Admin Detection**: First user automatically becomes Owner
  - Admin levels: `NONE`, `MODERATOR`, `ADMIN`, `OWNER`
  - Provisioning claim flow for first-boot (setup token from QR)
  - Fallback to `bridge.status` for older bridges

- ✅ **Security Warnings**: 6 warning types with severity levels
  - `EXTERNAL_SERVER` (LOW), `SHARED_IP` (HIGH), `UNENCRYPTED_CONNECTION` (CRITICAL)
  - `CERTIFICATE_ISSUE` (HIGH), `SERVER_UNVERIFIED` (MEDIUM), `FALLBACK_SERVER` (LOW)

### Admin Invite System (2026-02-16)
Admins can generate time-limited, cryptographically signed invite URLs to share server configuration.

- ✅ **InviteService**: Generate and validate invite links
  - Time-limited expiration (1h to 30d)
  - Optional usage limits
  - Cryptographic signature prevents tampering
  - Server config embedding (URL, name, features)

- ✅ **InviteViewModel**: UI state management
  - Generate invites with configurable options
  - Parse incoming invite URLs
  - Track invite states (valid, expired, exhausted, revoked)

- ✅ **InviteManagementScreen**: Admin management UI
  - Generate new invite links
  - View all created invites with status
  - Copy/share invite links
  - Revoke active invites
  - Status badges (Active, Expired, Used up, Revoked)

**Invite URL Format:**
```
https://bridge.armorclaw.app/invite/{base64_signed_data}
```

**Embedded Data:**
- Homeserver and bridge URLs
- Server name and description
- Welcome message
- Feature flags (E2EE, voice, video, etc.)
- Expiration timestamp
- Usage count and limits

---

## 1. Application Purpose

### Mission Statement
ArmorClaw empowers individuals and organizations to communicate privately without compromising on features or user experience. We believe secure communication should be accessible to everyone, not just technical users.

### Target Users

| User Segment | Description | Primary Needs |
|--------------|-------------|---------------|
| **Privacy Advocates** | Individuals concerned about surveillance | E2E encryption, no data mining |
| **Journalists** | Reporters handling sensitive sources | Source protection, message expiration |
| **Healthcare Professionals** | Doctors, therapists, counselors | HIPAA compliance, secure file sharing |
| **Legal Professionals** | Lawyers, paralegals | Attorney-client privilege, audit logs |
| **Businesses** | Companies with IP or trade secrets | Enterprise security, device management |
| **Activists** | Political and social activists | Anonymity, secure group communications |

### Problem Statement
In an era of mass surveillance, data breaches, and privacy violations, users lack a chat application that:
1. Provides genuine end-to-end encryption without backdoors
2. Offers a modern, feature-rich experience comparable to mainstream apps
3. Gives users complete control over their data
4. Works reliably even in poor network conditions
5. Is built on open, auditable standards

---

## 2. Application Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           PRESENTATION LAYER                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Jetpack Compose UI                        │   │
│  │  Screens │ Components │ ViewModels │ Navigation │ Themes   │   │
│  │  ✅ Complete │ ✅ Complete │ ✅ Complete │ ✅ Deep Links   │   │
│  └─────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────┤
│                            DOMAIN LAYER                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              Business Logic & Models                         │   │
│  │  Use Cases │ Repository Interfaces │ Domain Models          │   │
│  │  ✅ Complete │ ✅ 9 Repository Interfaces │ ✅ Complete      │   │
│  └─────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────┤
│                              DATA LAYER                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │               Data Sources & Persistence                     │   │
│  │  Repository Impl │ Offline Queue │ Sync Engine │ Logging   │   │
│  │  ✅ MessageRepo │ ✅ RoomRepo │ ✅ AppLogger Unified        │   │
│  └─────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────┤
│                           BRIDGE LAYER                              │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │               ArmorClaw Go Bridge Integration                │   │
│  │  BridgeRepository │ BridgeRpcClient │ BridgeWebSocketClient│   │
│  │  ✅ 58 RPC Methods │ ✅ 12 Event Types │ ✅ Auto Reconnect  │   │
│  └─────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────┤
│                           PLATFORM LAYER                            │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                Android Platform Services                     │   │
│  │  Biometric Auth │ Notifications │ Clipboard │ Network      │   │
│  │  ✅ Complete │ ✅ Complete │ ✅ Complete │ ✅ Complete      │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘

                    ↓ JSON-RPC 2.0 + WebSocket ↓

┌─────────────────────────────────────────────────────────────────────┐
│                     ARMORCLAW GO BRIDGE SERVER                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │               Per-User Isolated Container                    │   │
│  │  Matrix SDK │ WebRTC │ Encryption │ Recovery Keys          │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘

                    ↓ Matrix Protocol ↓

┌─────────────────────────────────────────────────────────────────────┐
│                         MATRIX HOMESERVER                           │
│              (Synapse, Dendrite, Conduit, etc.)                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Module Structure

```
ArmorChat/
├── shared/                          # Kotlin Multiplatform Shared Module
│   └── src/
│       ├── commonMain/              # Platform-agnostic code
│       │   ├── domain/              # Business logic
│       │   │   ├── model/           # Data models ✅
│       │   │   ├── repository/      # Repository interfaces (9) ✅
│       │   │   └── usecase/         # Business use cases ✅
│       │   ├── platform/            # Platform service interfaces ✅
│       │   │   └── bridge/          # Bridge integration layer ✅ NEW
│       │   │       ├── BridgeRepository.kt
│       │   │       ├── BridgeRpcClient.kt
│       │   │       ├── BridgeRpcClientImpl.kt
│       │   │       ├── BridgeWebSocketClient.kt
│       │   │       ├── BridgeWebSocketClientImpl.kt
│       │   │       ├── BridgeEvent.kt
│       │   │       ├── RpcModels.kt
│       │   │       └── WebSocketConfig.kt
│       │   └── ui/                  # Shared UI components ✅
│       └── androidMain/             # Android-specific implementations
│
└── androidApp/                      # Android Application
    └── src/main/kotlin/com/armorclaw/app/
        ├── screens/                 # Screen composables (19 screens) ✅
        │   ├── onboarding/          # Welcome, Security, Server, Permissions
        │   ├── home/                # Room list, favorites
        │   ├── chat/                # Messaging, reactions, threads
        │   ├── profile/             # User profile, account management
        │   ├── settings/            # App settings, preferences
        │   ├── room/                # Room management, settings
        │   ├── call/                # Voice calls
        │   ├── search/              # Global search
        │   └── splash/              # App initialization
        ├── components/              # Reusable UI components ✅
        ├── viewmodels/              # Screen state management ✅
        ├── navigation/              # Navigation (42 routes + deep links) ✅
        ├── platform/                # Platform service implementations ✅
        └── data/                    # Data layer ✅
            ├── repository/          # Repository implementations ✅
            │   ├── MessageRepositoryImpl.kt  # Uses BridgeRepository
            │   └── RoomRepositoryImpl.kt     # Uses BridgeRepository
            ├── offline/             # Offline sync & queue ✅
            │   ├── BackgroundSyncWorker.kt
            │   ├── ConflictResolver.kt
            │   └── MessageExpirationManager.kt
            └── database/            # Database entities ✅
```

### Technology Stack

| Layer | Technology | Purpose | Status |
|-------|------------|---------|--------|
| **Language** | Kotlin 1.9.20 | Primary development language | ✅ |
| **UI Framework** | Jetpack Compose 1.5.0 | Declarative UI | ✅ |
| **Design System** | Material Design 3 | Visual design language | ✅ |
| **DI Framework** | Koin 3.5.0 | Dependency injection | ✅ |
| **Database** | SQLDelight 2.0.0 + SQLCipher | Encrypted local storage | ✅ |
| **Networking** | Ktor 2.3.5 | API communication, WebSocket | ✅ |
| **Bridge Protocol** | JSON-RPC 2.0 | ArmorClaw bridge communication | ✅ |
| **Serialization** | kotlinx.serialization | JSON encoding/decoding | ✅ |
| **Async** | Kotlin Coroutines + Flow | Reactive programming | ✅ |
| **Image Loading** | Coil 2.4.0 | Efficient image loading | ✅ |
| **Analytics** | Sentry 6.34.0 | Crash reporting | ✅ |

---

## 3. Feature Overview

### Feature Categories

```
┌─────────────────────────────────────────────────────────────────────┐
│                         ARMORCLAW FEATURES                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐  │
│  │ AUTH &      │  │ MESSAGING   │  │ SECURITY    │  │ PROFILE   │  │
│  │ ONBOARDING  │  │             │  │             │  │ & SETTINGS│  │
│  ├─────────────┤  ├─────────────┤  ├─────────────┤  ├───────────┤  │
│  │ • Login     │  │ • Messages  │  │ • E2E Enc   │  │ • Avatar  │  │
│  │ • Register  │  │ • Replies   │  │ • Biometric │  │ • Status  │  │
│  │ • Biometric │  │ • Reactions │  │ • Clipboard │  │ • Account │  │
│  │ • Password  │  │ • Threads   │  │ • Cert Pin  │  │ • Privacy │  │
│  │ • Session   │  │ • Search    │  │ • Keys      │  │ • Logout  │  │
│  └─────────────┘  └─────────────┘  └─────────────┘  └───────────┘  │
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐  │
│  │ ROOM        │  │ CALLS       │  │ OFFLINE     │  │ PLATFORM  │  │
│  │ MANAGEMENT  │  │             │  │             │  │           │  │
│  ├─────────────┤  ├─────────────┤  ├─────────────┤  ├───────────┤  │
│  │ • Create    │  │ • Voice     │  │ • Queue     │  │ • Notify  │  │
│  │ • Join      │  │ • Incoming  │  │ • Sync      │  │ • Network │  │
│  │ • Settings  │  │ • Controls  │  │ • Conflict  │  │ • Storage │  │
│  │ • Members   │  │ • Audio Viz │  │ • Background│  │ • Biometric│ │
│  │ • Invites   │  │ • Encrypt   │  │ • Expiration│  │ • Analytics│ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └───────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Feature Statistics

| Category | Features | Status |
|----------|----------|--------|
| Authentication & Onboarding | 12 | ✅ Complete |
| Home Screen & Navigation | 18 | ✅ Complete |
| Chat & Messaging | 28 | ✅ Complete |
| Profile & Account | 15 | ✅ Complete |
| Settings | 20 | ✅ Complete |
| Room Management | 14 | ✅ Complete |
| Security | 16 | ✅ Complete |
| Platform Integration | 12 | ✅ Complete |
| UI/UX & Accessibility | 20 | ✅ Complete |
| Developer Tools | 10 | ✅ Complete |
| **Total** | **165+** | **100%** |

---

## 4. User Stories

### 4.1 Onboarding & Authentication

#### Story 1: First-Time User Onboarding
**As a** new user downloading ArmorClaw,
**I want to** be guided through the app's security features and setup process,
**So that** I understand how my data is protected before I start using the app.

**Acceptance Criteria:**
- Welcome screen explains key features with visual icons
- Security explanation shows 4-step encryption process with animation
- Server connection allows custom Matrix server or demo mode
- Permissions request clearly explains why each permission is needed
- Completion screen celebrates successful setup
- Progress is persisted - user can close and resume

**User Flow:**
```
App Launch -> Welcome -> Security -> ConnectServer -> Permissions -> KeyBackup -> Home
                                          |
                                   +------+------+
                                   | QR Scan     | (default, camera-based)
                                   +-------------+
                                   | Advanced    | (manual URL, two-phase connect)
                                   +------+------+
                                          |
                                Health Check Gate (mandatory)
                                          |
                                  Credentials -> Connected
```

#### Story 2: Secure Login
**As a** returning user,
**I want to** log in quickly using biometrics or password,
**So that** I can access my messages securely without friction.

**Acceptance Criteria:**
- Username/email field with validation
- Password field with visibility toggle
- Biometric authentication option (fingerprint/FaceID)
- "Forgot password" link available
- Form validation with clear error messages
- Secure credential storage

#### Story 3: Password Recovery
**As a** user who forgot my password,
**I want to** reset my password via email,
**So that** I can regain access to my account.

**Acceptance Criteria:**
- Email input for password reset
- Verification code sent to email
- New password with strength requirements
- Confirmation field matching
- Success/error feedback

---

### 4.2 Messaging & Communication

#### Story 4: Send Encrypted Message
**As a** user in a chat room,
**I want to** send a message that is automatically encrypted,
**So that** only the intended recipients can read it.

**Acceptance Criteria:**
- Message input with multiline support
- Send button with keyboard action
- Message status progression (Sending → Sent → Delivered → Read)
- Visual encryption indicator (lock icon)
- Failed message handling with retry
- Message appears immediately in conversation

**Message Status Flow:**
```
User types → Send button → SENDING → Server confirms → SENT
                                          ↓
                               Recipient receives → DELIVERED
                                          ↓
                               Recipient reads → READ
```

#### Story 5: Reply to Message
**As a** user receiving a message,
**I want to** reply with context showing what I'm responding to,
**So that** conversations remain clear and organized.

**Acceptance Criteria:**
- Long-press message to show context menu
- "Reply" option in menu
- Original message preview shown above input
- Reply quote displayed in sent message
- Tap quote to scroll to original
- Cancel reply option

#### Story 6: React to Message
**As a** user viewing a message,
**I want to** quickly express my reaction with an emoji,
**So that** I can acknowledge messages without typing.

**Acceptance Criteria:**
- Long-press to show reaction picker
- Common emojis displayed (👍 ❤️ 😂 😮 😢 🎉)
- Emoji shown below message with count
- Tap existing reaction to toggle
- Reaction updates in real-time for all participants

#### Story 7: Threaded Conversations
**As a** user in a busy group chat,
**I want to** create and view threaded conversations,
**So that** side discussions don't disrupt the main conversation.

**Acceptance Criteria:**
- Thread indicator on messages with replies
- Tap to open thread view
- Thread shows parent message and all replies
- Reply within thread
- Badge showing unread thread messages
- Return to main conversation

#### Story 8: Search Messages
**As a** user looking for past information,
**I want to** search through all my messages,
**So that** I can find specific content quickly.

**Acceptance Criteria:**
- Global search from home screen
- Search within specific chat
- Results highlighted with query match
- Tap result to jump to message context
- Filter by date, sender, room
- Recent searches saved

---

### 4.3 Room Management

#### Story 9: Create Group Room
**As a** user wanting to start a group conversation,
**I want to** create a new room and invite others,
**So that** we can communicate as a team.

**Acceptance Criteria:**
- Create room from FAB on home screen
- Set room name (required) and topic (optional)
- Choose privacy level (public/private)
- Set room avatar
- Generate shareable invite link
- Invite via QR code or link

#### Story 10: Join Existing Room
**As a** user invited to a conversation,
**I want to** join using an invite code or link,
**So that** I can participate in the discussion.

**Acceptance Criteria:**
- Enter room ID or alias
- Paste invite link (auto-detect)
- Preview room before joining
- See member count and description
- Join with single tap
- Room appears in home list

#### Story 11: Manage Room Settings
**As a** room administrator,
**I want to** configure room settings and manage members,
**So that** the room serves its intended purpose.

**Acceptance Criteria:**
- Edit room name and description
- Change room avatar
- Manage member roles (Member/Mod/Admin)
- Remove members
- Generate new invite links
- Set message retention policy
- Archive or delete room

---

### 4.4 Voice Calls

#### Story 12: Receive Voice Call
**As a** user receiving a call,
**I want to** see a full-screen notification with caller info,
**So that** I can decide whether to answer.

**Acceptance Criteria:**
- Full-screen incoming call UI
- Caller name and avatar displayed
- Encrypted call indicator
- Accept button (green)
- Decline button (red)
- Quick decline with message option
- Works when app is in background

#### Story 13: Active Call Experience
**As a** user on a call,
**I want to** control audio settings and see call status,
**So that** I can manage my call experience.

**Acceptance Criteria:**
- Caller info with call duration
- Mute/unmute microphone
- Speaker phone toggle
- End call button
- Audio level visualization
- Encryption indicator
- Minimize call (picture-in-picture)

---

### 4.5 Profile & Account

#### Story 14: Manage Profile
**As a** user,
**I want to** customize my profile and status,
**So that** others can recognize me and know my availability.

**Acceptance Criteria:**
- View profile with avatar, name, status
- Edit display name
- Change avatar from camera/gallery
- Set status (Online/Away/Busy/Invisible)
- Add custom status message
- View email and phone (read-only)

#### Story 15: Account Security
**As a** security-conscious user,
**I want to** manage my account security settings,
**So that** I can protect my account from unauthorized access.

**Acceptance Criteria:**
- Change password with verification
- Enable/disable biometric login
- View and manage trusted devices
- Verify new devices via emoji comparison
- Revoke access from suspicious devices
- Two-factor authentication setup

#### Story 16: Delete Account
**As a** user leaving the platform,
**I want to** permanently delete my account and data,
**So that** my information is completely removed.

**Acceptance Criteria:**
- Clear warnings about data loss
- List what will be deleted
- Type "DELETE" to confirm
- 30-day grace period (optional)
- Confirmation email sent
- Complete data erasure

---

### 4.6 Settings & Preferences

#### Story 17: Notification Preferences
**As a** user,
**I want to** control when and how I receive notifications,
**So that** I'm not disturbed unnecessarily.

**Acceptance Criteria:**
- Enable/disable all notifications
- Per-room notification settings
- Mentions-only mode
- Quiet hours scheduling
- Sound and vibration toggles
- Notification preview settings

#### Story 18: Privacy Settings
**As a** privacy-conscious user,
**I want to** control what information is visible to others,
**So that** I maintain my desired privacy level.

**Acceptance Criteria:**
- Control read receipt visibility
- Manage online status visibility
- Block/unblock users
- View privacy policy
- Export personal data
- Manage data retention

---

### 4.7 Security & Encryption

#### Story 19: Verify Device Encryption
**As a** user concerned about security,
**I want to** verify that my messages are encrypted,
**So that** I can trust the communication is private.

**Acceptance Criteria:**
- Visual encryption indicator in chat header
- Lock icon on encrypted messages
- Tap icon for encryption details
- Four encryption states clearly indicated:
  - ✅ Verified (green) - All devices verified
  - ⚠️ Unverified (yellow) - Some devices unverified
  - ❌ Unencrypted (red) - No encryption
  - ℹ️ Unavailable (gray) - Cannot encrypt

#### Story 20: Device Verification
**As a** user with multiple devices,
**I want to** verify new devices securely,
**So that** I can ensure no man-in-the-middle attacks.

**Acceptance Criteria:**
- Emoji verification process
- Same emoji set shown on both devices
- User compares and confirms match
- Device marked as verified
- Option to reject if mismatch
- Verification logged in security history

---

### 4.8 Offline Support

#### Story 21: Offline Messaging
**As a** user with unreliable internet,
**I want to** compose and queue messages while offline,
**So that** they're sent automatically when connectivity returns.

**Acceptance Criteria:**
- Messages queued when offline
- Visual indicator for pending messages
- Automatic send when online
- Retry failed messages
- Order preserved on delivery
- Conflict resolution for simultaneous edits

#### Story 22: Background Sync
**As a** user who doesn't check the app frequently,
**I want to** receive new messages in the background,
**So that** I'm notified of important messages.

**Acceptance Criteria:**
- Periodic background sync (15 min intervals)
- Battery-optimized scheduling
- Network-aware sync
- Sync status indicator
- Manual refresh option
- New message notifications

---

## 5. Data Models

### Core Domain Models

```kotlin
// User representation
data class User(
    val id: String,
    val displayName: String,
    val email: String,
    val phoneNumber: String?,
    val avatar: String?,
    val status: UserStatus,
    val statusMessage: String?,
    val bio: String?
)

// Chat room
data class Room(
    val id: String,
    val name: String,
    val description: String?,
    val avatar: String?,
    val type: RoomType,           // DIRECT, GROUP, CHANNEL
    val isEncrypted: Boolean,
    val memberCount: Int,
    val unreadCount: Int,
    val lastMessage: Message?
)

// Message
data class Message(
    val id: String,
    val roomId: String,
    val senderId: String,
    val content: MessageContent,
    val timestamp: Instant,
    val status: MessageStatus,    // SENDING, SENT, DELIVERED, READ, FAILED
    val replyTo: String?,
    val reactions: List<Reaction>
)

// Thread
data class Thread(
    val id: String,
    val parentMessageId: String,
    val replyCount: Int,
    val lastReplyAt: Instant?,
    val isResolved: Boolean
)

// Device for verification
data class Device(
    val id: String,
    val name: String,
    val platform: DevicePlatform,
    val trustLevel: TrustLevel,   // VERIFIED, UNVERIFIED, BLOCKED
    val lastActiveAt: Instant
)
```

---

## 6. Security Implementation

### Encryption Stack

| Layer | Algorithm | Purpose |
|-------|-----------|---------|
| **Message** | AES-256-GCM | Message content encryption |
| **Key Exchange** | ECDH (Curve25519) | Shared secret generation |
| **Signing** | Ed25519 | Message authentication |
| **Database** | SQLCipher (256-bit) | Local storage encryption |
| **Transport** | TLS 1.3 + Pinning | Network security |
| **Key Storage** | AndroidKeyStore | Secure key storage |

### Security Features Matrix

| Feature | Implementation | User Benefit |
|---------|---------------|--------------|
| E2E Encryption | AES-256-GCM | Only recipients can read |
| Perfect Forward Secrecy | ECDH key rotation | Past messages stay secure |
| Biometric Auth | AndroidX Biometric | Quick secure access |
| Secure Clipboard | Auto-clear + encryption | Safe copy/paste |
| Certificate Pinning | SHA-256 pins | MITM protection |
| Device Verification | Emoji comparison | Trust establishment |
| Key Backup | Encrypted recovery | Account recovery |

### Encryption Trust Model (NEW 2026-02-18, updated 2026-02-24)

ArmorClaw supports two encryption trust models. **Note:** The original `EncryptionService` (expect/actual) was removed (Lesson 15) as it was a pass-through creating confusion about encryption ownership. Type definitions retained in `EncryptionTypes.kt`.

#### Server-Side Encryption (Current)
```
┌─────────────┐      ┌─────────────────────┐      ┌─────────────────┐
│ ArmorChat   │─────▶│ ArmorClaw Go Bridge │─────▶│ Matrix Server   │
│ (No Crypto) │      │ (E2EE via libolm)   │      │ (Transport Only)│
└─────────────┘      └─────────────────────┘      └─────────────────┘
```

- Bridge Server handles all Matrix E2EE operations
- Per-user container isolation
- Keys managed securely server-side
- App trusts Bridge as encryption authority

**When to use:**
- Default mode for all users
- Simpler key management
- No key backup required from user

#### Client-Side Encryption (Future)
```
┌─────────────────────┐      ┌─────────────┐      ┌─────────────────┐
│ ArmorChat           │─────▶│ Bridge      │─────▶│ Matrix Server   │
│ (E2EE via vodozemac)│      │ (Transport) │      │ (Transport Only)│
└─────────────────────┘      └─────────────┘      └─────────────────┘
```

- Client handles all encryption locally
- Keys never leave device
- vodozemac (Rust) for crypto operations
- User responsible for key backup

**When to use:**
- Maximum security requirements
- Zero-trust architecture
- User-controlled keys

#### Encryption Status Indicators

| Status | Icon | Description |
|--------|------|-------------|
| `SERVER_ENCRYPTED` | 🔒 Green | Encrypted by Bridge (current mode) |
| `CLIENT_ENCRYPTED` | 🔒 Green | Encrypted locally (future) |
| `UNENCRYPTED` | 🔓 Red | No encryption (warning) |
| `UNKNOWN` | ❓ Gray | Status unknown |

### Deep Link Security (NEW 2026-02-18)

Deep links can be triggered by any app or website. ArmorClaw validates all deep links before navigation:

#### Validation Steps

1. **Scheme Validation**: Only `armorclaw://` and whitelisted HTTPS hosts
2. **Host Validation**: HTTPS hosts must be in known safe list
3. **Length Check**: URIs over 2048 characters rejected
4. **Path Validation**: No `..` or backslash in path segments
5. **ID Format Validation**: Room IDs must match Matrix format

#### Security States

| State | Behavior |
|-------|----------|
| `Valid` | Navigate immediately |
| `RequiresConfirmation` | Show confirmation dialog |
| `Invalid` | Reject with logged reason |

#### Confirmation Required For

- Room navigation (join confirmation)
- Call join (accept call prompt)
- External HTTPS links (trust warning)

### Admin Role Assignment (NEW 2026-02-18)

User roles are now **server-authoritative** to prevent race conditions:

#### Role Hierarchy

| Level | Description | Permissions |
|-------|-------------|-------------|
| `OWNER` | First registered user | All admin + server config |
| `ADMIN` | Server administrator | Invite, room management |
| `MODERATOR` | Room moderator | Kick, mute users |
| `NONE` | Regular user | Basic messaging |

#### Server-Side Assignment

```kotlin
// Bridge Server determines role in bridge.status response
data class BridgeStatusResponse(
    val userRole: AdminLevel?,  // Server-assigned role
    val isNewServer: Boolean?   // First user becomes OWNER
)
```

**No more client-side role detection** - eliminates race conditions when multiple users connect simultaneously to a new server.

---

## 7. User Interface Structure

### Screen Hierarchy

```
SplashScreen
├── Migration Flow (NEW 2026-02-24)
│   └── MigrationScreen (auto-detect legacy Bridge sessions)
│
├── Onboarding Flow
│   ├── WelcomeScreen
│   ├── SecurityExplanationScreen
│   ├── ConnectServerScreen (QR-first, updated 2026-02-26)
│   │   ├── QR Scanner (default mode, camera barcode detection)
│   │   └── Advanced Mode (manual URL, two-phase connect)
│   ├── PermissionsScreen (real Android permission requests, fixed 2026-02-26)
│   ├── CompletionScreen
│   └── KeyBackupSetupScreen (NEW 2026-02-24 — mandatory before Home)
│
├── Authentication Flow
│   ├── LoginScreen
│   │   └── KeyRecoveryScreen (NEW 2026-02-24 — recover encryption keys)
│   ├── RegistrationScreen
│   └── ForgotPasswordScreen
│
└── Main App
    ├── HomeScreen (Room List)
    │   ├── SearchScreen
    │   ├── RoomManagementScreen
    │   │   ├── CreateRoomTab
    │   │   └── JoinRoomTab
    │   └── RoomItem → ChatScreen
    │
    ├── ChatScreen
    │   ├── MessageList
    │   ├── MessageBubble
    │   ├── ReplyPreview
    │   ├── TypingIndicator
    │   ├── SearchOverlay
    │   └── ThreadView
    │
    ├── ProfileScreen
    │   ├── ChangePasswordScreen
    │   ├── ChangePhoneNumberScreen
    │   ├── EditBioScreen
    │   └── DeleteAccountScreen
    │
    ├── SettingsScreen
    │   ├── NotificationSettingsScreen
    │   ├── SecuritySettingsScreen
    │   │   └── → KeyBackupSetupScreen (optional re-entry)
    │   ├── AppearanceSettingsScreen
    │   ├── DeviceListScreen
    │   │   └── EmojiVerificationScreen
    │   ├── AboutScreen
    │   ├── ReportBugScreen
    │   ├── PrivacyPolicyScreen
    │   └── MyDataScreen
    │
    ├── RoomSettingsScreen
    │   └── RoomDetailsScreen
    │       └── BridgeVerificationBanner (NEW 2026-02-24)
    │           └── → EmojiVerificationScreen
    │
    └── Call Flow
        ├── IncomingCallDialog
        └── ActiveCallScreen
```

### Navigation Graph

The app uses a single-activity architecture with Compose Navigation:

- **42 total routes** covering all screens (added KEY_RECOVERY, AGENT_MANAGEMENT, HITL_APPROVALS, WORKFLOW_MANAGEMENT, BUDGET_STATUS, and more)
- **Animated transitions** between screens
- **Deep linking** support with multiple schemes
- **Back stack management** with pop-up-to behavior
- **Nested navigation** for onboarding and settings
- **Security validation** for all deep links (NEW)

#### Deep Link Support

| URI Scheme | Format | Example |
|------------|--------|---------|
| App Schema | `armorclaw://room/{roomId}` | `armorclaw://room/!abc123:server.com` |
| App Schema | `armorclaw://user/{userId}` | `armorclaw://user/@alice:server.com` |
| App Schema | `armorclaw://call/{callId}` | `armorclaw://call/call_12345` |
| Matrix.to | `https://matrix.to/#/{roomId}` | `https://matrix.to/#/!abc123:server.com` |
| Matrix.to | `https://matrix.to/#/{userId}` | `https://matrix.to/#/@alice:server.com` |

#### Deep Link Security (See Section 6)

All deep links are validated before navigation:
- Room links require confirmation before joining
- Call links show accept/decline dialog
- Invalid links are rejected with logging
- Deep links construct proper backstack (Home always present, fixed 2026-02-25)

---

## 8. Performance & Quality

### Performance Metrics

| Metric | Target | Actual |
|--------|--------|--------|
| Cold Start | < 2s | ~1.8s |
| Message Send | < 500ms | ~200ms |
| Room List Load | < 300ms | ~150ms |
| Search Results | < 200ms | ~100ms |
| Memory Usage | < 200MB | ~156MB |
| APK Size | < 25MB | ~22MB |
| Frame Rate | 60 FPS | 58-60 FPS |

### Quality Assurance

| Test Type | Count | Coverage |
|-----------|-------|----------|
| Unit Tests | 50+ | Business logic |
| Integration Tests | 15+ | Data layer |
| UI Tests | 20+ | Screen flows |
| E2E Tests | 11 | User journeys |
| Accessibility Tests | 15+ | TalkBack, scaling |

### Code Quality Metrics

| Metric | Score | Assessment |
|--------|-------|------------|
| **Separation of Concerns** | ⭐⭐⭐⭐⭐ | Clean architecture with complete repository implementations |
| **Logging Coverage** | ⭐⭐⭐⭐⭐ | Unified AppLogger system with structured metadata |
| **Error Handling** | ⭐⭐⭐⭐⭐ | Comprehensive error taxonomy (120+ error codes) |
| **Build Status** | ✅ PASS | Zero compilation errors |

---

## 9. Future Roadmap

### Planned Enhancements

| Feature | Priority | Target |
|---------|----------|--------|
| Video Calls | High | Q2 2026 |
| Widget Support | Medium | Q2 2026 |
| Tablet Optimization | Medium | Q2 2026 |
| Wear OS Companion | Low | Q3 2026 |
| Desktop Client | Medium | Q3 2026 |
| iOS App | High | Q4 2026 |

---

## 10. Conclusion

ArmorClaw represents a comprehensive approach to secure messaging, combining:

1. **Enterprise-Grade Security**: Military-grade encryption with user-controlled keys
2. **Modern User Experience**: Intuitive Material Design 3 interface
3. **Offline Resilience**: Full functionality without constant connectivity
4. **Open Standards**: Built on Matrix protocol for interoperability
5. **Privacy by Design**: No data mining, no backdoors, user ownership
6. **Clean Architecture**: Complete repository layer with unified logging
7. **Bridge Integration**: Secure communication via ArmorClaw Go bridge server
8. **Setup Flow**: Flawless first-time experience with admin detection and security warnings
9. **Invite System**: Time-limited signed URLs for user invitations
10. **Security Hardening**: Deep link validation, server-authoritative roles, push notifications

### Architecture Status: ✅ Production Ready with Bridge Integration + Security Hardening + Setup Review

The application demonstrates that security and usability are not mutually exclusive. With 165+ features across 42 routes, ArmorClaw provides a complete communication platform while maintaining the highest standards of privacy and security.

**Setup Flow Review (2026-02-26):**
- ✅ Fixed Advanced connect race condition (two-phase connect pattern)
- ✅ Added bridge health gating to all setup paths (QR, manual, deep link)
- ✅ Fixed IP address detection false positives, URL decoding order bug
- ✅ Fixed camera executor leak, PermissionsScreen now requests real permissions
- ✅ See Lessons 29-34 in `LESSONS_LEARNED.md`

**Critical Security Fixes (2026-02-18):**
- ✅ **Encryption Trust Model**: Clear documentation of Server-Side vs Client-Side E2EE
- ✅ **Push Notifications**: Full FCM integration with dual registration (Bridge + Matrix)
- ✅ **Admin Role Assignment**: Server-authoritative role assignment prevents race conditions
- ✅ **Deep Link Security**: Validation and confirmation dialogs for external links
- ✅ **Sync State Machine**: Guaranteed state reset prevents deadlocks

**Bridge Integration Complete (2026-02-15):**
- ✅ 58 JSON-RPC methods across 15 categories
- ✅ 12 WebSocket event types for real-time updates
- ✅ BridgeRepository integration layer
- ✅ Automatic reconnection with exponential backoff

**Setup Flow Complete (2026-02-15, updated 2026-02-26):**
- ✅ QR-first onboarding with mandatory bridge health gating
- ✅ Admin detection via provisioning claim + server-authoritative fallback
- ✅ 6 security warning types with severity levels
- ✅ Two-phase connect pattern for async-safe credential handling
- ✅ Real Android permission requests with `RequestPermission` contract

**Invite System Complete (2026-02-16):**
- ✅ Time-limited invite links (1h to 30d expiration)
- ✅ Cryptographic signature prevents tampering
- ✅ Optional usage limits per invite
- ✅ Server config embedded in invite URL
- ✅ Admin can revoke invites
- ✅ Invite status tracking (active, expired, exhausted, revoked)

**Previous Improvements:**
- Repository pattern fully implemented with MessageRepositoryImpl and RoomRepositoryImpl
- All data layer components use unified AppLogger with structured metadata
- ChatViewModel properly uses repository layer with pagination support
- Deep linking support for armorclaw:// and matrix.to URI schemes
- Zero compilation errors, clean build

---

## 11. API Reference

### Bridge RPC Methods

The ArmorClaw Bridge server exposes **58 RPC methods** organized into 15 categories. All methods follow JSON-RPC 2.0 specification.

#### Bridge Lifecycle Methods (4)

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `bridge.start` | Start a new bridge session | `userId`, `deviceId` | `BridgeStartResponse` |
| `bridge.status` | Get current bridge status | None | `BridgeStatusResponse` |
| `bridge.stop` | Stop bridge session | `sessionId` | `BridgeStopResponse` |
| `bridge.health` | Health check | None | Health status map |

#### Matrix Methods (10) - DEPRECATED (use MatrixClient instead)

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `matrix.login` | Login to homeserver | `homeserver`, `username`, `password`, `deviceId` | `MatrixLoginResponse` |
| `matrix.sync` | Sync with homeserver | `since?`, `timeout?`, `filter?` | `MatrixSyncResponse` |
| `matrix.send` | Send message | `roomId`, `eventType`, `content`, `txnId?` | `MatrixSendResponse` |
| `matrix.refresh_token` | Refresh access token | `refreshToken` | `MatrixLoginResponse` |
| `matrix.create_room` | Create new room | `name?`, `topic?`, `isDirect?`, `invite?` | `MatrixCreateRoomResponse` |
| `matrix.join_room` | Join a room | `roomIdOrAlias` | `MatrixJoinRoomResponse` |
| `matrix.leave_room` | Leave a room | `roomId` | Boolean |
| `matrix.invite_user` | Invite user to room | `roomId`, `userId` | Boolean |
| `matrix.send_typing` | Send typing notification | `roomId`, `typing`, `timeout?` | Boolean |
| `matrix.send_read_receipt` | Mark message as read | `roomId`, `eventId` | Boolean |

#### WebRTC Methods (4)

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `webrtc.offer` | Create WebRTC offer | `callId`, `sdpOffer` | `WebRtcSignalingResponse` |
| `webrtc.answer` | Process WebRTC answer | `callId`, `sdpAnswer` | `WebRtcSignalingResponse` |
| `webrtc.ice_candidate` | Send ICE candidate | `callId`, `candidate`, `sdpMid?`, `sdpMlineIndex?` | Boolean |
| `webrtc.hangup` | End call | `callId` | Boolean |

#### Recovery Methods (6)

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `recovery.generate_phrase` | Generate recovery phrase | None | `RecoveryPhraseResponse` |
| `recovery.store_phrase` | Store encrypted phrase | `phrase` | Boolean |
| `recovery.verify` | Verify recovery phrase | `phrase` | `RecoveryVerifyResponse` |
| `recovery.status` | Get recovery status | `recoveryId` | `RecoveryStatusResponse` |
| `recovery.complete` | Complete recovery | `recoveryId`, `newDeviceName` | `RecoveryCompleteResponse` |
| `recovery.is_device_valid` | Check device validity | `deviceId` | `DeviceValidResponse` |

#### Platform Methods (5)

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `platform.connect` | Connect external platform | `platformType`, `config` | `PlatformConnectResponse` |
| `platform.disconnect` | Disconnect platform | `platformId` | Boolean |
| `platform.list` | List connected platforms | None | `PlatformListResponse` |
| `platform.status` | Get platform status | `platformId` | `PlatformStatusResponse` |
| `platform.test` | Test platform connection | `platformId` | `PlatformTestResponse` |

#### Push Notification Methods (3) - NEW 2026-02-18

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `push.register_token` | Register FCM/APNs token | `pushToken`, `pushPlatform`, `deviceId` | `PushRegisterResponse` |
| `push.unregister_token` | Remove push token | `pushToken` | Boolean |
| `push.update_settings` | Update notification settings | `enabled`, `quietHoursStart?`, `quietHoursEnd?` | Boolean |

**Push Notification Flow:**
```
1. FCM/APNs provides device token
2. App calls push.register via BridgeRpcClient
3. Bridge Server stores token for user/device
4. Server sends push when new message arrives
5. App receives push, shows notification
6. On logout, call push.unregister
```

**Supported Platforms:**
- `fcm` - Firebase Cloud Messaging (Android)
- `apns` - Apple Push Notification Service (iOS future)

#### Provisioning Methods (5) - NEW 2026-02-24

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `provisioning.start` | Start provisioning session | `expiration?` | `ProvisioningStartResponse` |
| `provisioning.status` | Check provisioning status | `provisioningId` | `ProvisioningStatusResponse` |
| `provisioning.claim` | Claim admin via setup token | `setupToken`, `deviceName`, `deviceType?` | `ProvisioningClaimResponse` |
| `provisioning.rotate` | Rotate provisioning secret | None | `ProvisioningRotateResponse` |
| `provisioning.cancel` | Cancel provisioning session | `provisioningId` | `ProvisioningCancelResponse` |

#### License Methods (3) - NEW

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `license.status` | Get license status | None | `LicenseStatusResponse` |
| `license.features` | Get features by tier | None | `LicenseFeaturesResponse` |
| `license.check_feature` | Check feature availability | `feature` | `FeatureCheckResponse` |

#### Compliance Methods (2) - NEW

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `compliance.status` | Get compliance mode | None | `ComplianceStatusResponse` |
| `compliance.platform_limits` | Get bridging limits | None | `PlatformLimitsResponse` |

#### Error Management Methods (2) - NEW

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `get_errors` | Get recent bridge errors | `limit?`, `component?` | `ErrorsResponse` |
| `resolve_error` | Resolve an error | `errorId`, `resolution?` | Boolean |

#### Agent Methods (3) - NEW

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `agent.list` | List running agents | None | `AgentListResponse` |
| `agent.status` | Get agent status | `agentId` | `AgentStatusResponse` |
| `agent.stop` | Stop an agent | `agentId` | Boolean |

#### Workflow Methods (3) - NEW

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `workflow.templates` | Get workflow templates | None | `WorkflowTemplatesResponse` |
| `workflow.start` | Start a workflow | `templateId`, `params?`, `roomId?` | `WorkflowStartResponse` |
| `workflow.status` | Get workflow status | `workflowId` | `WorkflowStatusResponse` |

#### HITL Methods (3) - NEW

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `hitl.pending` | Get pending approvals | None | `HitlPendingResponse` |
| `hitl.approve` | Approve a request | `gateId`, `notes?` | Boolean |
| `hitl.reject` | Reject a request | `gateId`, `reason?` | Boolean |

#### Budget Methods (1) - NEW

| Method | Description | Parameters | Response |
|--------|-------------|------------|----------|
| `budget.status` | Get budget usage | None | `BudgetStatusResponse` |

### WebSocket Event Types

Real-time events are pushed via WebSocket connection.

| Event Type | Description | Key Fields |
|------------|-------------|------------|
| `message.received` | New message in room | `eventId`, `roomId`, `senderId`, `content` |
| `message.status` | Message delivery update | `eventId`, `roomId`, `status` |
| `room.created` | New room created | `roomId`, `name`, `isDirect` |
| `room.membership` | User joined/left room | `roomId`, `userId`, `membership` |
| `typing` | Typing indicator | `roomId`, `userId`, `typing` |
| `receipt.read` | Read receipt | `roomId`, `userId`, `eventId` |
| `presence` | User presence update | `userId`, `presence`, `statusMsg` |
| `call` | WebRTC call signaling | `callId`, `roomId`, `action`, `sdp?` |
| `platform.message` | External platform message | `platformId`, `platformType`, `content` |
| `session.expired` | Session needs refresh | `expiredSessionId`, `reason` |
| `bridge.status` | Bridge health status | `status`, `message?`, `containerId?` |
| `recovery` | Account recovery event | `recoveryId`, `action` |

### Request/Response Examples

#### Login Request
```json
{
  "jsonrpc": "2.0",
  "method": "matrix.login",
  "params": {
    "homeserver": "https://matrix.example.com",
    "username": "alice",
    "password": "secret",
    "deviceId": "ANDROID_001"
  },
  "id": "req-12345"
}
```

#### Login Response
```json
{
  "jsonrpc": "2.0",
  "result": {
    "access_token": "sydt_...",
    "refresh_token": "syrt_...",
    "device_id": "ANDROID_001",
    "user_id": "@alice:example.com",
    "expires_in": 3600
  },
  "id": "req-12345"
}
```

#### Error Response
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32001,
    "message": "Authentication failed",
    "data": {
      "reason": "invalid_credentials",
      "retry_after": 5000
    }
  },
  "id": "req-12345"
}
```

---

## 12. Error Code Reference

ArmorClaw uses a comprehensive error taxonomy with **80+ error codes** organized by category.

### Error Categories

| Category | Description | Example Codes |
|----------|-------------|---------------|
| `PERMISSION` | User permission issues | E001, E002, E010 |
| `NETWORK` | Network/connectivity issues | E003, E011-E020 |
| `TRUST` | Device verification issues | E021-E030 |
| `ENCRYPTION` | Crypto/encryption issues | E031-E040 |
| `SYNC` | Synchronization issues | E041-E050 |
| `CONTENT` | Message/room content issues | E051-E080 |
| `AUTH` | Authentication issues | E081-E090 |
| `STATE` | Application state issues | E005-E008 |
| `HARDWARE` | Device hardware issues | E009 |

### Error Code Reference Table

#### Voice/Call Errors (E001-E010)

| Code | Name | Message | Recoverable |
|------|------|---------|-------------|
| E001 | `VOICE_MIC_DENIED` | Microphone access denied. Please enable in settings. | ✅ |
| E002 | `VOICE_PERMISSION_REQUIRED` | Microphone permission required for voice calls. | ✅ |
| E003 | `TURN_SERVER_TIMEOUT` | Call server unreachable. Please try again. | ✅ |
| E004 | `ICE_CONNECTION_FAILED` | Network connection failed. Check your internet. | ✅ |
| E005 | `CALL_ALREADY_ACTIVE` | You already have an active call. | ✅ |
| E006 | `CALL_NOT_FOUND` | Call no longer exists. | ❌ |
| E007 | `CALL_DECLINED` | Call was declined. | ❌ |
| E008 | `CALL_TIMEOUT` | No answer. Call timed out. | ❌ |
| E009 | `AUDIO_DEVICE_ERROR` | Audio device error. Please check your microphone. | ✅ |
| E010 | `CAMERA_DENIED` | Camera access denied. | ✅ |

#### Network Errors (E011-E020)

| Code | Name | Message | Recoverable |
|------|------|---------|-------------|
| E011 | `HOMESERVER_UNREACHABLE` | Server temporarily unavailable. | ✅ |
| E012 | `CONNECTION_TIMEOUT` | Connection timed out. Please try again. | ✅ |
| E013 | `NETWORK_CHANGED` | Network changed. Reconnecting... | ✅ |
| E014 | `NO_NETWORK` | No internet connection. | ✅ |
| E015 | `DNS_RESOLUTION_FAILED` | Could not resolve server address. | ✅ |
| E016 | `SSL_ERROR` | Secure connection failed. | ✅ |
| E017 | `RATE_LIMITED` | Too many requests. Please wait. | ✅ |
| E018 | `SERVER_ERROR` | Server error. Please try again later. | ✅ |
| E019 | `PROXY_ERROR` | Proxy configuration error. | ✅ |
| E020 | `WEBSOCKET_FAILED` | Real-time connection failed. Using fallback. | ✅ |

#### Trust/Verification Errors (E021-E030)

| Code | Name | Message | Recoverable |
|------|------|---------|-------------|
| E021 | `DEVICE_UNVERIFIED` | Device verification required for this action. | ✅ |
| E022 | `CROSS_SIGNING_REQUIRED` | Please set up cross-signing first. | ✅ |
| E023 | `VERIFICATION_FAILED` | Verification failed. Please try again. | ✅ |
| E024 | `SESSION_EXPIRED` | Session expired. Please log in again. | ✅ |
| E025 | `VERIFICATION_CANCELLED` | Verification was cancelled. | ✅ |
| E026 | `VERIFICATION_TIMEOUT` | Verification timed out. | ✅ |
| E027 | `KEY_SIGNATURE_INVALID` | Invalid key signature. | ❌ |
| E028 | `TRUST_CHAIN_BROKEN` | Trust chain verification failed. | ✅ |
| E029 | `DEVICE_BLOCKED` | This device has been blocked. | ❌ |
| E030 | `RECOVERY_REQUIRED` | Account recovery required. | ✅ |

#### Encryption Errors (E031-E040)

| Code | Name | Message | Recoverable |
|------|------|---------|-------------|
| E031 | `ENCRYPTION_KEY_ERROR` | Encryption error. Please restart the app. | ✅ |
| E032 | `DECRYPTION_FAILED` | Could not decrypt message. | ❌ |
| E033 | `KEY_BACKUP_REQUIRED` | Please back up your encryption keys. | ✅ |
| E034 | `MEGOLM_SESSION_ERROR` | Message encryption session error. | ✅ |
| E035 | `OLM_SESSION_ERROR` | Key exchange session error. | ✅ |
| E036 | `KEY_NOT_TRUSTED` | Encryption key not trusted. | ✅ |
| E037 | `KEY_ROTATION_REQUIRED` | Encryption keys need rotation. | ✅ |
| E038 | `CLIPBOARD_ENCRYPTION_FAILED` | Could not encrypt clipboard content. | ✅ |
| E039 | `DATABASE_ENCRYPTION_ERROR` | Database encryption error. | ✅ |
| E040 | `SECURE_STORAGE_ERROR` | Secure storage error. | ✅ |

#### Sync Errors (E041-E050)

| Code | Name | Message | Recoverable |
|------|------|---------|-------------|
| E041 | `SYNC_QUEUE_OVERFLOW` | Too many pending operations. Please wait. | ✅ |
| E042 | `SYNC_CONFLICT` | Data conflict detected. Resolving... | ✅ |
| E043 | `OFFLINE_MODE` | You are offline. Changes will sync when connected. | ✅ |
| E044 | `SYNC_VERSION_MISMATCH` | Sync version mismatch. Updating... | ✅ |
| E045 | `SYNC_TOKEN_INVALID` | Sync position invalid. Re-syncing... | ✅ |
| E046 | `SYNC_TIMEOUT` | Sync timed out. Will retry. | ✅ |
| E047 | `SYNC_STORAGE_FULL` | Local storage full. Please free up space. | ✅ |
| E048 | `BATCH_SYNC_FAILED` | Batch sync failed. Retrying individually... | ✅ |
| E049 | `SYNC_FORCED_RESET` | Sync reset required. Clearing cache... | ✅ |
| E050 | `BACKGROUND_SYNC_DISABLED` | Background sync disabled in settings. | ✅ |

#### Content Errors (E051-E080)

| Code | Name | Message | Recoverable |
|------|------|---------|-------------|
| E051 | `THREAD_NOT_FOUND` | Thread not found or deleted. | ❌ |
| E052 | `THREAD_REPLY_FAILED` | Could not send reply. Please try again. | ✅ |
| E061 | `ROOM_NOT_FOUND` | Room not found. | ❌ |
| E062 | `ROOM_ACCESS_DENIED` | Access to room denied. | ❌ |
| E063 | `ROOM_FULL` | Room is full. | ❌ |
| E067 | `ROOM_CREATION_FAILED` | Could not create room. | ✅ |
| E071 | `MESSAGE_NOT_FOUND` | Message not found. | ❌ |
| E075 | `MESSAGE_SEND_FAILED` | Could not send message. | ✅ |
| E076 | `ATTACHMENT_TOO_LARGE` | Attachment is too large. | ✅ |

#### Authentication Errors (E081-E090)

| Code | Name | Message | Recoverable |
|------|------|---------|-------------|
| E081 | `LOGIN_FAILED` | Login failed. Please check your credentials. | ✅ |
| E082 | `INVALID_CREDENTIALS` | Invalid username or password. | ✅ |
| E083 | `ACCOUNT_LOCKED` | Account is locked. Please contact support. | ❌ |
| E084 | `PASSWORD_RESET_FAILED` | Password reset failed. | ✅ |
| E085 | `REGISTRATION_FAILED` | Registration failed. | ✅ |
| E086 | `TOKEN_REFRESH_FAILED` | Session refresh failed. Please log in again. | ✅ |
| E087 | `MFA_REQUIRED` | Two-factor authentication required. | ✅ |
| E088 | `MFA_FAILED` | Invalid verification code. | ✅ |
| E089 | `BIOMETRIC_FAILED` | Biometric authentication failed. | ✅ |
| E090 | `LOGOUT_FAILED` | Logout failed. Please try again. | ✅ |

### Recoverable Actions

When errors are recoverable, the following actions are available:

| Action | Description | User Prompt |
|--------|-------------|-------------|
| `Retry` | Retry the failed operation | "Try Again" |
| `OpenSettings` | Open app settings | "Open Settings" |
| `ReLogin` | Re-authenticate | "Log In Again" |
| `VerifyDevice` | Complete device verification | "Verify Device" |
| `CheckNetwork` | Check network connection | "Check Connection" |
| `Custom` | Custom action | Developer-defined |

---

## 13. Database Schema

ArmorClaw uses SQLDelight with SQLCipher for encrypted local storage.

### Schema Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ARMORCLAW DATABASE (SQLCipher)                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────────┐   │
│  │  messages   │────▶│   rooms     │     │       users         │   │
│  │             │     │             │     │  (in shared pref)   │   │
│  └─────────────┘     └─────────────┘     └─────────────────────┘   │
│         │                   │                                       │
│         │                   │                                       │
│         ▼                   ▼                                       │
│  ┌─────────────┐     ┌─────────────────┐                           │
│  │sync_metadata│     │    devices      │                           │
│  │             │     │                 │                           │
│  └─────────────┘     └─────────────────┘                           │
│                             │                                       │
│                             ▼                                       │
│                    ┌────────────────────────┐                       │
│                    │ verification_transactions│                      │
│                    └────────────────────────┘                       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Messages Table

**Purpose:** Store all chat messages with thread support

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PK | Local message ID |
| `event_id` | TEXT | Matrix event ID |
| `room_id` | TEXT FK | Reference to room |
| `sender_id` | TEXT | Sender user ID |
| `content` | TEXT | Message body |
| `formatted_body` | TEXT | HTML-formatted body |
| `message_type` | TEXT | TEXT, IMAGE, FILE, etc. |
| `timestamp` | INTEGER | Local timestamp |
| `server_timestamp` | INTEGER | Server timestamp |
| `is_outgoing` | INTEGER | 1 if sent by current user |
| `sync_status` | TEXT | PENDING, SYNCED, FAILED |
| `edit_count` | INTEGER | Number of edits |
| `reply_to` | TEXT | Parent message ID |
| `is_deleted` | INTEGER | Soft delete flag |
| `thread_root_id` | TEXT | Root message for threads |
| `is_thread_reply` | INTEGER | 1 if thread reply |
| `thread_depth` | INTEGER | Nesting level |
| `thread_participants` | TEXT | JSON array of user IDs |
| `reactions` | TEXT | JSON array of reactions |

**Indexes:**
- `idx_messages_room_id` - Room lookup
- `idx_messages_event_id` - Event deduplication
- `idx_messages_sender_id` - Sender queries
- `idx_messages_timestamp` - Chronological ordering
- `idx_messages_thread_root_id` - Thread queries
- `idx_messages_thread_depth` - Thread depth queries

### Rooms Table

**Purpose:** Store room metadata and state

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PK | Room ID |
| `name` | TEXT | Display name |
| `topic` | TEXT | Room topic/description |
| `avatar_url` | TEXT | Avatar mxc:// URL |
| `is_direct` | INTEGER | 1 if DM |
| `is_encrypted` | INTEGER | 1 if E2EE enabled |
| `member_count` | INTEGER | Number of members |
| `unread_count` | INTEGER | Unread messages |
| `mention_count` | INTEGER | Unread mentions |
| `last_message_id` | TEXT | Latest message |
| `last_message_timestamp` | INTEGER | Last activity |
| `is_favourite` | INTEGER | Favorited flag |
| `is_low_priority` | INTEGER | Low priority flag |
| `join_state` | TEXT | JOINED, INVITED, LEFT |
| `thread_unread_count` | INTEGER | Unread thread replies |
| `active_thread_count` | INTEGER | Active threads |
| `last_thread_activity` | INTEGER | Last thread timestamp |

### Devices Table

**Purpose:** Device verification and trust management

| Column | Type | Description |
|--------|------|-------------|
| `device_id` | TEXT PK | Device identifier |
| `user_id` | TEXT PK | Owner user ID |
| `display_name` | TEXT | Device name |
| `is_verified` | INTEGER | Manual verification |
| `trust_level` | TEXT | UNVERIFIED, VERIFIED, WARNED, etc. |
| `last_seen_ip` | TEXT | Last IP address |
| `last_seen_timestamp` | INTEGER | Last activity |
| `is_current_device` | INTEGER | 1 if this device |
| `curve25519_key` | TEXT | Identity key |
| `ed25519_key` | TEXT | Signing key |
| `signatures` | TEXT | JSON signatures |
| `signed_by` | TEXT | Cross-signer device |
| `verified_at` | INTEGER | Verification timestamp |

### Sync Metadata Table

**Purpose:** Track sync state per room

| Column | Type | Description |
|--------|------|-------------|
| `room_id` | TEXT PK | Room identifier |
| `last_sync_token` | TEXT | Matrix sync token |
| `last_sync_timestamp` | INTEGER | Last sync time |
| `pending_count` | INTEGER | Pending operations |
| `last_error` | TEXT | Last error message |
| `retry_count` | INTEGER | Consecutive failures |

### Verification Transactions Table

**Purpose:** Track verification flows

| Column | Type | Description |
|--------|------|-------------|
| `transaction_id` | TEXT PK | Transaction ID |
| `other_user_id` | TEXT | User being verified |
| `other_device_id` | TEXT | Device being verified |
| `state` | TEXT | UNVERIFIED, REQUESTED, EMOJI_CHALLENGE, etc. |
| `method` | TEXT | SAS, QR, etc. |
| `started_at` | INTEGER | Start timestamp |
| `updated_at` | INTEGER | Last update |
| `cancellation_reason` | TEXT | If cancelled |
| `emoji_data` | TEXT | JSON emoji array |
| `completed_at` | INTEGER | Completion timestamp |

---

## 14. Troubleshooting Guide

### Common Issues and Solutions

#### Authentication Issues

| Issue | Symptoms | Solution |
|-------|----------|----------|
| Login fails | "Invalid credentials" error | Check homeserver URL, verify username format (@user:server.com) |
| Session expired | Redirected to login | Token refresh failed; clear app data and re-login |
| Biometric fails | "Biometric authentication failed" | Re-register biometric in device settings |
| MFA loop | Repeatedly asks for code | Check time sync on device; verify authenticator app |

#### Network Issues

| Issue | Symptoms | Solution |
|-------|----------|----------|
| Connection timeout | Long loading, timeouts | Check network connectivity; try different network |
| WebSocket disconnected | No real-time updates | Check firewall settings; verify WebSocket URL |
| Certificate error | SSL/TLS errors | Verify certificate pins; check system time |
| Rate limited | "Too many requests" | Wait and retry; reduce request frequency |

#### Sync Issues

| Issue | Symptoms | Solution |
|-------|----------|----------|
| Messages not syncing | Missing messages | Force sync from settings; check network |
| Duplicate messages | Same message appears twice | Clear cache; re-sync |
| Offline messages stuck | Pending messages not sent | Check network; force retry from settings |
| Storage full | "Storage full" error | Clear old messages; free device storage |

#### Encryption Issues

| Issue | Symptoms | Solution |
|-------|----------|----------|
| Can't decrypt | "Could not decrypt message" | Request key from sender; verify device |
| Key backup failed | Backup incomplete | Check network; verify recovery phrase |
| Verification fails | "Verification failed" | Ensure both devices online; retry |
| Cross-signing error | Trust chain broken | Reset cross-signing; re-verify devices |

### Debug Commands

```bash
# Check bridge connection
adb shell am broadcast -a com.armorclaw.DEBUG.BRIDGE_STATUS

# Force sync
adb shell am broadcast -a com.armorclaw.DEBUG.FORCE_SYNC

# Clear cache
adb shell pm clear com.armorclaw.app

# View logs
adb logcat -s ArmorClaw:V
```

### Log Analysis

Key log patterns to look for:

```
# Successful login
[ArmorClaw:Auth] Login successful userId=@user:server.com

# Bridge connection
[ArmorClaw:Bridge] RPC connected sessionId=abc123
[ArmorClaw:Bridge] WebSocket connected

# Sync events
[ArmorClaw:Sync] Sync started since=token_123
[ArmorClaw:Sync] Sync complete rooms=5 messages=23

# Errors
[ArmorClaw:Error] E075: MESSAGE_SEND_FAILED roomId=!abc details=network_timeout
```

---

## 15. Performance Optimization

### Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Cold Start | < 2s | Time to first frame |
| Message Send | < 500ms | Time to server ack |
| Room List Load | < 300ms | Time to display rooms |
| Search Results | < 200ms | Time to results |
| Memory Usage | < 200MB | Peak heap |
| APK Size | < 25MB | Release APK |
| Frame Rate | 60 FPS | UI smoothness |

### Optimization Strategies

#### Startup Optimization
- Lazy initialization of non-critical services
- Deferred bridge connection until after first render
- Parallel initialization of independent modules

#### List Performance
- Use `LazyColumn` with stable keys
- Implement `equals()` and `hashCode()` for items
- Avoid recomposition with `key()` and `derivedStateOf`

```kotlin
LazyColumn {
    items(
        items = messages,
        key = { it.id } // Stable key prevents recomposition
    ) { message ->
        MessageBubble(message)
    }
}
```

#### Database Optimization
- Use pagination for large datasets
- Batch operations when possible
- Index frequently queried columns

#### Network Optimization
- Connection pooling via Ktor
- Request deduplication
- Response caching where appropriate

---

## 16. Security Audit Checklist

### Pre-Release Security Review

#### Data Protection
- [ ] Database encrypted with SQLCipher (256-bit)
- [ ] Sensitive data in AndroidKeyStore
- [ ] No plaintext credentials in logs
- [ ] Secure deletion of sensitive data
- [ ] Memory cleared after use

#### Network Security
- [ ] TLS 1.3 for all connections
- [ ] Certificate pinning enabled
- [ ] No mixed HTTP/HTTPS content
- [ ] Proper certificate validation

#### Authentication
- [ ] Secure password handling (no logging)
- [ ] Token secure storage
- [ ] Session timeout implemented
- [ ] Biometric fallback available

#### Encryption
- [ ] E2E encryption enabled by default
- [ ] Proper key generation
- [ ] Key rotation implemented
- [ ] Perfect forward secrecy

### OWASP Mobile Top 10 Compliance

| Risk | Status | Mitigation |
|------|--------|------------|
| M1: Improper Platform Usage | ✅ | Proper intent handling |
| M2: Insecure Data Storage | ✅ | SQLCipher, AndroidKeyStore |
| M3: Insecure Communication | ✅ | TLS 1.3, Certificate Pinning |
| M4: Insecure Authentication | ✅ | Biometric, MFA, Token refresh |
| M5: Insufficient Cryptography | ✅ | AES-256-GCM, Curve25519 |
| M6: Insecure Authorization | ✅ | Role-based access, token validation |
| M7: Client Code Quality | ✅ | Detekt, code review |
| M8: Code Tampering | ✅ | R8, signature verification |
| M9: Reverse Engineering | ✅ | ProGuard, native libs |
| M10: Extraneous Functionality | ✅ | Debug code removed in release |

---

## 17. Accessibility Compliance

### WCAG 2.1 AA Compliance

#### Perceivable

| Guideline | Status | Implementation |
|-----------|--------|----------------|
| 1.1.1 Non-text content | ✅ | All images have content descriptions |
| 1.3.1 Info and relationships | ✅ | Semantic heading hierarchy |
| 1.4.3 Contrast (minimum) | ✅ | 4.5:1 for normal text |
| 1.4.4 Resize text | ✅ | Supports 200% scaling |

#### Operable

| Guideline | Status | Implementation |
|-----------|--------|----------------|
| 2.1.1 Keyboard | ✅ | Full keyboard/D-pad navigation |
| 2.4.3 Focus order | ✅ | Logical tab order |
| 2.4.7 Focus visible | ✅ | Clear focus indicators |

#### Understandable

| Guideline | Status | Implementation |
|-----------|--------|----------------|
| 3.3.1 Error identification | ✅ | Errors clearly described |
| 3.3.2 Labels or instructions | ✅ | Form labels present |

### Android Accessibility Features

#### TalkBack Support
```kotlin
// Content descriptions
Image(
    painter = painterResource(R.drawable.avatar),
    contentDescription = "User avatar: ${user.displayName}"
)

// Custom actions
Modifier.semantics {
    customActions = listOf(
        CustomAccessibilityAction("Reply to message") { /* ... */ true },
        CustomAccessibilityAction("React with emoji") { /* ... */ true }
    )
}
```

---

## 18. Deployment & DevOps

### Build Variants

| Variant | Purpose | Configuration |
|---------|---------|---------------|
| `debug` | Development | Debuggable, no minification, test server |
| `release` | Production | R8 minification, production server, signing |
| `benchmark` | Performance testing | Release config with profiling |

### Release Checklist

#### Pre-Release
- [ ] Update version code and name
- [ ] Update CHANGELOG.md
- [ ] Run all tests: `./gradlew test connectedAndroidTest`
- [ ] Run static analysis: `./gradlew detekt`
- [ ] Review security checklist
- [ ] Verify ProGuard rules

#### Build Release
```bash
# Clean build
./gradlew clean

# Build release APK
./gradlew assembleRelease

# Build release bundle (Play Store)
./gradlew bundleRelease
```

### CI/CD Pipeline (GitHub Actions)

```yaml
name: CI
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up JDK 17
        uses: actions/setup-java@v4
        with:
          java-version: '17'
          distribution: 'temurin'
      - name: Run tests
        run: ./gradlew test
      - name: Run detekt
        run: ./gradlew detekt
      - name: Build debug APK
        run: ./gradlew assembleDebug
```

### Monitoring & Alerting

#### Crash Reporting
- **Sentry** for crash tracking
- **Firebase Crashlytics** for crash analytics
- Automatic breadcrumb logging

#### Alerts
- Crash rate > 1% triggers alert
- ANR rate > 0.5% triggers alert
- Response time > 5s triggers alert

---

## Related Documentation

- [Architecture Documentation](../ARCHITECTURE.md)
- [Features List](../FEATURES.md)
- [API Documentation](../API.md)
- [User Guide](../USER_GUIDE.md)
- [Developer Guide](../DEVELOPER_GUIDE.md)
- [Separation of Concerns Assessment](./separation-of-concerns-assessment.md)
- [Bridge Integration](../bridge/README.md) - ArmorClaw Go bridge server integration
- [Documentation Index](../index.md)
