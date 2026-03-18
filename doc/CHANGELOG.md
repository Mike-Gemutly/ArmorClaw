# ArmorClaw Changelog

> Complete changelog for ArmorClaw - Secure E2E Encrypted Chat Application

---

## [4.1.1-alpha01] - 2026-02-25 - Frontend UX Review Fixes

### Fixed

#### UX Fix #1: Key Backup Force-Quit Bypass (HIGH)
- Added `KEY_BACKUP_COMPLETE` flag to `AppPreferences` with `isBackupComplete()` / `setBackupComplete()` methods
- `SplashViewModel.checkInitialState()` now checks backup completion — if logged in but backup incomplete, re-routes to `KeyBackupSetupScreen`
- Added `SplashTarget.KeyBackupSetup` navigation target
- Wired `SplashViewModel` into SPLASH composable in `AppNavigation.kt` for proper target-based routing
- `KeyBackupSetupScreen` "Skip for now" replaced with scary `AlertDialog` confirmation ("I Accept the Risk")
- Skip does NOT mark backup as complete — user will be prompted again on next launch

#### UX Fix #2: Migration "Skip" Terminology (MEDIUM)
- Renamed `MigrationScreen` skip button from "Skip (you may lose old message history)" to "Start Fresh (Delete Old Data)"
- Added descriptive subtitle: "Skips migration and permanently erases v2.5 message history"
- Red warning color for destructive label

#### UX Fix #3: Bridge Verification Visibility in ChatScreen (HIGH)
- Added `isBridgeVerified` StateFlow to `ChatViewModel` with `checkBridgeVerification()` init check
- Added yellow shield icon (`Icons.Default.Shield`) to `ChatTopBar` when bridge is unverified
- Added first-entry `AlertDialog` modal for unverified bridged rooms with "I Understand" / "Leave Room" actions
- Bridge verification state passed through `ChatScreenEnhanced` → `ChatTopBar`

#### UX Fix #4: Feature Suppression Feedback (LOW)
- Added `UiEvent.ShowSnackbar` to shared `UiEvent` sealed class
- `ChatViewModel.toggleReaction()` now emits Snackbar when `canReact()` returns false
- `ChatViewModel.replyToMessage()` now emits Snackbar when `canReply()` returns false
- Replaces silent disabling with visible user feedback

#### UX Fix #5: Deep Link Backstack (MEDIUM)
- `handleDeepLinkAction()` in `ArmorClawNavHost.kt` now verifies Home is in the backstack before navigating
- If Home is missing, navigates to Home first (clearing previous stack) then pushes deep link target
- Ensures Back button from deep-linked screens returns to Home, not exits app
- `SplashTarget.DeepLink` targets also construct Home-root backstack in AppNavigation SPLASH composable

#### UX Fix #6: Push Notification Fallback (MEDIUM)
- `ArmorFirebaseMessagingService.onMessageReceived()` catch block for `syncOnce()` now calls `showFallbackNotification()`
- Fallback notification uses push metadata only ("You may have new messages. Tap to open.")
- Ensures user is notified even when background sync fails (e.g., Doze mode, network issues)

#### UX Fix #7: Governance Banner Dismissal Persistence (LOW)
- Added `getGovernanceDismissedAt()`, `setGovernanceDismissed()`, `isGovernanceDismissed()` to `AppPreferences`
- CRITICAL severity: dismiss for 24 hours, then re-shows automatically
- INFO/WARNING: dismiss permanently (until next event)
- License Expired events (`id.startsWith("license-expired")`) cannot be dismissed — close button hidden
- `GovernanceBanner` now accepts `isDismissed` parameter for persistence integration

### Changed
- `AppPreferences` companion: added `KEY_BACKUP_COMPLETE`, `KEY_GOVERNANCE_DISMISSED_PREFIX`, `GOVERNANCE_CRITICAL_RESHOW_MS`
- `SplashTarget` sealed class: added `KeyBackupSetup` target
- `UiEvent` sealed class: added `ShowSnackbar` subclass
- `ChatViewModel`: added `isBridgeVerified` state and `checkBridgeVerification()` method
- `GovernanceBanner`: `canDismiss` logic based on event severity; `isDismissed` parameter

---

## [4.1.0-alpha01] - 2026-02-24 - ArmorClaw Integration Fixes

### Fixed

#### Bug Fix #6: Push Notification Dual Registration (CRITICAL)
- `PushNotificationRepositoryImpl` now registers FCM tokens with **both** Matrix Homeserver (`MatrixClient.setPusher()`) and Bridge Server (`BridgeRpcClient.pushRegister()`)
- Added `setPusher()` and `removePusher()` to `MatrixClient` interface and `MatrixClientImpl`
- Graceful partial-failure handling — if one channel fails, the other still works
- Updated DI in `AppModules.kt` to inject both `BridgeRpcClient` and `MatrixClient`

#### Bug Fix #7: Bridge Verification UX (HIGH)
- Added `BridgeVerificationBanner` composable to `RoomDetailsScreen.kt`
- Warning banner with "Verify Bridge Device" button when bridge device is unverified
- Added `BRIDGE_VERIFICATION` route to `AppNavigation.kt`
- Wired navigation: Room Details → Bridge Verification → `EmojiVerificationScreen`

#### Bug Fix #8: User Migration v2.5→v3.0 (CRITICAL)
- Created `MigrationScreen.kt` with auto-detection of legacy Bridge sessions
- Manual recovery phrase entry fallback
- Success/failure states with clear user guidance
- Added `SplashTarget.Migration` to `SplashViewModel`
- Added `hasLegacyBridgeSession()` and `clearLegacyBridgeSession()` to `AppPreferences`
- Added `MIGRATION` route to `AppNavigation.kt`

#### Bug Fix #9: Key Backup Setup (HIGH)
- Created `KeyBackupSetupScreen.kt` with 6-step guided flow
  - Explain → Generate → Display → Verify → Store → Success
- Mandatory in onboarding (CompletionScreen → KeyBackupSetup → Home)
- Optional re-entry from `SecuritySettingsScreen`
- Added `KEY_BACKUP_SETUP` route to `AppNavigation.kt`

### Changed
- Navigation route count: 37 → 42 (added MIGRATION, KEY_BACKUP_SETUP, BRIDGE_VERIFICATION, KEY_RECOVERY)
- `AppPreferences` constructor parameter renamed from `context` to `appContext` (private val)

### Removed
- **`EncryptionService`** (expect/actual) — stripped legacy pass-through encryption layer. Matrix Rust SDK now handles all encryption. Type definitions preserved in `EncryptionTypes.kt`.
- **`encryptionModule`** from `AppModules.kt` DI configuration.

### Added

#### Architecture Review Fix #1: Strip Legacy Encryption
- Deleted `EncryptionService.kt` (common) and `EncryptionService.android.kt`
- Created `EncryptionTypes.kt` retaining `EncryptionMode`, `RoomEncryptionStatus`, `DeviceVerificationStatus`
- Removed `encryptionModule` from DI

#### Architecture Review Fix #2: Origin Badges for Bridged Users
- Added `BridgePlatform` enum (SLACK, DISCORD, TEAMS, WHATSAPP, TELEGRAM, SIGNAL, MATRIX_NATIVE)
- Added `bridgePlatform` field to `MessageSender.UserSender`
- Created `BridgedOriginBadge.kt` composable — platform icon overlay on avatars

#### Architecture Review Fix #3: Feature Suppression in Bridged Rooms
- Added `BridgeRoomCapabilities` data class (`supportsEdit`, `supportsReactions`, `supportsThreads`, `supportsReadReceipts`)
- Added `bridgeCapabilities` field to `Room` model
- Updated `canReact()`, `canEdit()`, `canThread()` extension functions to check bridge capabilities

#### Architecture Review Fix #4: Enterprise Governance Feedback UI
- Added `LICENSE_WARNING`, `LICENSE_EXPIRED`, `CONTENT_POLICY_APPLIED` to `SystemEventType`
- Created `GovernanceBanner.kt` — dismissable info/warning/critical banner for HomeScreen
- Created `ContentPolicyPlaceholder.kt` — inline placeholder for HIPAA/DLP-scrubbed messages

#### Architecture Review Fix #5: Key Recovery on Login Screen
- Added `onRecoverKeys` callback to `LoginScreen`
- Added "Recover encryption keys" link below biometric login
- Created `KeyRecoveryScreen.kt` — 3-step flow (Enter Phrase → Verify → Restore)
- Added `KEY_RECOVERY` route to `AppNavigation`

#### Critical Fix: Push Notification setPusher() Was a No-Op Stub
- `MatrixClientImpl.setPusher()` was entirely a stub — logged success but never made an HTTP call. Homeserver never received pusher registration, meaning **zero background push notifications**.
- Replaced with real HTTP POST to `/_matrix/client/v3/pushers/set` (kind=http, data.url for Sygnal gateway, data.format=event_id_only)
- Added real `removePusher()` (kind="" per spec) replacing matching no-op stub
- Added `httpPost()` helper using `HttpURLConnection` (interim; Ktor migration planned)
- Injected `MatrixClient` into `ArmorFirebaseMessagingService` — calls `syncOnce()` on push receipt so the SDK fetches and decrypts event content before showing notification

#### Legacy Data Wipe in Migration Flow
- `MigrationScreen` now calls `wipeLegacyData()` on success, skip, and logout
- Added `clearLegacyDatabase()` to `AppPreferences` — deletes v2.5 SQLCipher files (`bridge_store.db` + journal/WAL/SHM)
- Added `wipeLegacyData()` combining Bridge SharedPreferences + SQLCipher DB + legacy onboarding prefs cleanup
- Prevents stale v2.5 data from accumulating on disk after v3.0 migration

### Security
- Push notifications now reach devices via Matrix push gateway (Sygnal) in addition to Bridge RPC
- **setPusher() actually calls the homeserver** — previously a silent no-op that broke all background push
- Key backup ensures encryption keys survive device loss
- Bridge device verification discoverable from Room Details UI
- Legacy encryption pass-through layer removed — Matrix Rust SDK is sole encryption provider
- Key recovery accessible from login screen for returning users
- Legacy Bridge data (SQLCipher DB, SharedPreferences) wiped during migration to prevent data leakage

---

## [4.0.0-alpha01] - 2026-02-22 - Governor Strategy Complete

### Added

#### Phase 1: Cold Vault
- SQLCipher integration for encrypted PII storage
- `KeystoreManager` - Hardware-backed key management (Android Keystore)
- `SqlCipherProvider` - Encrypted database factory
- `VaultRepository` - PII CRUD operations
- `PiiRegistry` - PII key registry with predefined fields
- `ShadowMap` - Placeholder mapping with `{{VAULT:field:hash}}` format
- `AgentRequestInterceptor` - Outbound request middleware for PII shadowing
- `VaultPulseIndicator` - Visual indicator for agent PII access requests
- `VaultKeyPanel` - Sidebar component for key management
- `VaultStore` - State management for vault operations

#### Phase 2: Governor UI
- `CommandBlock` - Technical action card (replaces message bubbles for agents)
- `CommandStatusBadge` - Status indicators (pending, running, completed, failed)
- `CapabilityRibbon` - Horizontal display of active capabilities
- `CapabilityChip` - Individual capability indicators
- `CapabilityIndicator` - Context-aware capability status
- `HITLAuthorizationCard` - Human-in-the-loop approval component
- `SimpleApprovalDialog` - Quick approve/reject dialog

#### Phase 3: Audit & Transparency
- `TaskReceipt` - Immutable action receipt with redacted diffs
- `ActionType` - Action type enumeration
- `TaskStatus` - Task status tracking
- `CapabilityUsage` - Capability usage tracking
- `PiiAccess` - PII access logging
- `RevocationRecord` - Revocation history
- `AuditSession` - Session audit trail
- `RiskSummary` - Risk assessment display
- `ArmorTerminal` - Real-time activity log (terminal-style)
- `RevocationPanel` - One-click capability revocation
- `QuickRevocationButton` - Immediate session undo

#### Phase 4: Commercial Polish
- `ArmorClawTheme` - Unified brand theme (Teal #14F0C8, Navy #0A1428)
- `SecurityStatusIcon` - Security state indicator
- `AgentStatusIcon` - Agent state indicator
- `CapabilityStatusIcon` - Capability state indicator
- `NetworkStatusIcon` - Connection state indicator
- `RiskLevelBadge` - Risk level display (low, medium, high, critical)
- `ActivityPulseIndicator` - Animated activity indicator
- `StatusBar` - Combined status bar component

### Changed
- ArmorChat is now the "Governor" - authoritative controller for ArmorClaw agents
- All PII encrypted at rest with SQLCipher
- Agent messages rendered as Command Blocks (technical UI)
- Real-time activity logging in ArmorTerminal

### Security
- Hardware-backed encryption (Android Keystore + AES-256)
- PII shadow mapping prevents raw data transmission
- Immutable audit trail for all agent actions
- One-click revocation for immediate capability disable

---

## [1.2.0] - 2026-02-21 - Unified Theme Module

### Added

#### armorclaw-ui Module
- New Kotlin Multiplatform module `:armorclaw-ui` for unified branding
- ArmorClawColor.kt - Teal (#14F0C8), Navy (#0A1428) color palette
- ArmorClawTypography.kt - Inter + JetBrains Mono typography
- ArmorClawShapes.kt - Unified shape definitions
- ArmorClawTheme.kt - Theme wrapper composable
- GlowModifiers.kt - Teal glow effect modifiers

#### Navigation Updates
- 47 total routes (up from 44)
- Added SERVER_CONNECTION route for server settings
- Added LICENSES route for open source licenses
- Added TERMS_OF_SERVICE route

### Changed
- Updated project structure to include armorclaw-ui module
- Branding assets moved to styling/ directory

### Documentation
- Added styling.md with unified theme implementation guide
- Updated REVIEW.md with armorclaw-ui module documentation
- Updated BUILD_STATUS.md with current module status

---

## [1.1.0] - 2026-02-18 - Matrix Migration Complete

### Added

#### Matrix SDK Integration (Phase 1)
- MatrixClient interface with 40+ methods
- MatrixClientFactory (expect/actual pattern)
- MatrixClientAndroidImpl using Rust SDK via FFI
- MatrixSessionStorage with encrypted persistence
- ArmorClaw custom Matrix event types
- Full sync state management

#### Transport Layer Split (Phase 2)
- BridgeAdminClient for admin-only operations (22 methods)
- BridgeAdminClientImpl delegating to BridgeRpcClient
- 10 RPC methods deprecated with @Deprecated annotations

#### Control Plane Events (Phase 3)
- ControlPlaneStore for workflow/agent event processing
- WorkflowRepository for workflow state persistence
- AgentRepository for agent task state
- WorkflowProgressBanner component
- AgentThinkingIndicator component
- WorkflowCard component
- HomeViewModel workflow integration
- ChatViewModel workflow/agent state

#### UI Unification (Phase 4)
- UnifiedMessage model (Regular, Agent, System, Command)
- UnifiedMessageList component for all message types
- UnifiedChatInput with command mode detection (!)
- ChatViewModel unified state
- Agent action handlers (regenerate, copy, follow-up)
- System action handlers (cancel, retry, verify)
- Command detection and execution
- Workflow system messages

#### Testing
- 130+ unit tests added
- MatrixClientTest (25+ tests)
- ControlPlaneStoreTest (15+ tests)
- UI component tests (45+ tests)
- UnifiedMessageTest (25+ tests)
- ChatViewModelUnifiedTest integration tests (20+ tests)

### Changed
- ChatViewModel now uses UnifiedMessage model
- HomeScreen includes workflow section
- ChatScreen uses unified components
- Navigation simplified (no terminal route needed)
- Session storage uses EncryptedSharedPreferences

### Deprecated
- `matrixLogin` - Use `MatrixClient.login()`
- `matrixSync` - Use `MatrixClient.startSync()`
- `matrixSend` - Use `MatrixClient.sendTextMessage()`
- `matrixRefreshToken` - SDK handles automatically
- `matrixCreateRoom` - Use `MatrixClient.createRoom()`
- `matrixJoinRoom` - Use `MatrixClient.joinRoom()`
- `matrixLeaveRoom` - Use `MatrixClient.leaveRoom()`
- `matrixInviteUser` - Use `MatrixClient.inviteUser()`
- `matrixSendTyping` - Use `MatrixClient.sendTyping()`
- `matrixSendReadReceipt` - Use `MatrixClient.sendReadReceipt()`

### Removed
- Separate Terminal screen (commands now in any chat)
- Legacy message handling in ChatViewModel

### Migration Impact
- Client now uses proper Matrix protocol
- E2E encryption keys held by client (not server)
- Federation ready (can connect to any Matrix server)
- Single unified chat interface

---

## [1.0.0] - 2026-02-10 - Initial Release

### Added

#### Core Features
- ✅ End-to-end encryption (ECDH+AES-GCM)
- ✅ Biometric authentication (Fingerprint/FaceID)
- ✅ Offline support with sync queue
- ✅ Message expiration (ephemeral messages)
- ✅ Conflict resolution (local, server, manual)

#### Onboarding Flow
- ✅ Splash screen with animations
- ✅ Welcome screen with feature list
- ✅ Security explanation screen (animated diagram, 4 steps)
- ✅ Connect server screen (URL, Connect, Demo option)
- ✅ Permissions screen (required/optional with progress)
- ✅ Completion screen (celebration, confetti, what's next)

#### Authentication
- ✅ Login screen (username/email, password)
- ✅ Biometric login (Fingerprint/FaceID)
- ✅ Forgot password flow (placeholder)
- ✅ Registration flow (placeholder)

#### Home Screen
- ✅ Room list with categories (Favorites, Chats, Archived)
- ✅ Unread badge count
- ✅ Room avatar with encryption indicator
- ✅ Last message preview
- ✅ Timestamp display (relative time)
- ✅ Expandable sections (Favorites, Archived)
- ✅ Join room button
- ✅ Create room floating action button
- ✅ Search button
- ✅ Profile button
- ✅ Settings button
- ✅ Top app bar with navigation

#### Chat Features
- ✅ Enhanced message list (loading, empty, error, pull-to-refresh)
- ✅ Message status indicators (Sending, Sent, Delivered, Read, Failed)
- ✅ Timestamp formatting (relative time)
- ✅ Reply to message
- ✅ Forward message
- ✅ Message reactions (emoji)
- ✅ File attachments
- ✅ Image attachments
- ✅ Voice messages
- ✅ Voice input
- ✅ Search within chat
- ✅ Message encryption indicators
- ✅ Typing indicators
- ✅ Reply preview
- ✅ Chat search bar

#### Profile Features
- ✅ Profile screen
- ✅ Profile avatar with edit overlay
- ✅ Status indicator (Online, Away, Busy, Invisible)
- ✅ Status dropdown
- ✅ Profile information (Name, Email, Status)
- ✅ Edit mode (Edit/Save)
- ✅ Account options (Change Password, Phone, Bio, Delete)
- ✅ Privacy settings (Privacy Policy, My Data)
- ✅ Logout button

#### Settings Features
- ✅ Settings screen
- ✅ User profile section (avatar, name, email)
- ✅ App settings (Notifications, Sound, Vibration)
- ✅ Appearance settings (Theme)
- ✅ Security settings (Biometric Auth, Encryption)
- ✅ Privacy section (Privacy Policy, Data & Storage)
- ✅ About section (About, Report Bug, Rate App)
- ✅ Logout button
- ✅ Version info

#### Room Management
- ✅ Room management screen
- ✅ Tab navigation (Create Room / Join Room)
- ✅ Create room form (Name, Topic, Privacy, Avatar)
- ✅ Join room form (Room ID, Alias)
- ✅ Form validation
- ✅ Privacy toggle
- ✅ Info cards

#### Platform Integrations
- ✅ Biometric authentication (AndroidX Biometric)
- ✅ Secure clipboard (encryption, hash verification, auto-clear)
- ✅ Push notifications (FCM, channels, grouped)
- ✅ Certificate pinning (OkHttp, SHA-256)
- ✅ Crash reporting (Sentry, breadcrumbs, performance)
- ✅ Analytics (event tracking, screen tracking, user tracking)
- ✅ Network monitoring (ConnectivityManager)

#### Offline Sync
- ✅ SQLCipher database (256-bit passphrase)
- ✅ Offline queue (enqueue, priority, retry)
- ✅ Sync engine (state machine, operation execution)
- ✅ Conflict resolver (detection, resolution strategies)
- ✅ Background sync worker (WorkManager, constraints)
- ✅ Message expiration manager (expiration, auto-check, deletion)

#### Performance
- ✅ Performance profiler (tracing, memory tracking, strict mode)
- ✅ Memory monitor (memory usage, pressure detection, leak detection)
- ✅ Method execution tracing
- ✅ Memory allocation tracking
- ✅ Heap dumping
- ✅ Strict mode enforcement

#### Accessibility
- ✅ Accessibility config (screen reader, high contrast, large text)
- ✅ Accessibility extensions (Compose modifiers)
- ✅ Content descriptions
- ✅ Heading levels
- ✅ Traversal order
- ✅ Screen reader support (TalkBack)
- ✅ High contrast detection
- ✅ Large text detection
- ✅ Font scale detection
- ✅ Reduced motion detection

#### Release
- ✅ Release config (build types, release channels, feature flags)
- ✅ Feature flags (20+ features)
- ✅ Build variants (debug, release, demo, alpha, beta, stable)
- ✅ R8/ProGuard rules
- ✅ Code shrinking
- ✅ Resource shrinking
- ✅ APK optimization

#### Navigation
- ✅ Animated transitions (fade in/out)
- ✅ 20+ navigation routes
- ✅ Route parameter support (roomId)
- ✅ Pop-up-to handling
- ✅ Deep linking support
- ✅ Nested navigation support

#### Design System
- ✅ Color palette (custom colors)
- ✅ Typography (custom fonts)
- ✅ Shapes (custom rounded corners)
- ✅ Theme (Material Design 3)
- ✅ Dark mode (automatic)
- ✅ Animations (10+)

#### UI Components
- ✅ Atomic components (Button, InputField, Card, Badge, Icon)
- ✅ Molecular components (MessageBubble, TypingIndicator, EncryptionStatus, ReplyPreview, ChatSearchBar)
- ✅ Organism components (MessageList, RoomItemCard, ProfileAvatar)
- ✅ Screen components (19 screens)

#### Testing
- ✅ Unit tests (50+)
- ✅ Integration tests (15+)
- ✅ E2E tests (11 scenarios)
- ✅ Compose UI tests
- ✅ Instrumented tests

#### Documentation
- ✅ README.md (project overview, getting started)
- ✅ FEATURES.md (complete feature list)
- ✅ ARCHITECTURE.md (architecture overview)
- ✅ COMPONENTS.md (UI component catalog)
- ✅ API.md (public API documentation)
- ✅ USER_GUIDE.md (user guide)
- ✅ DEVELOPER_GUIDE.md (developer guide)
- ✅ CHANGELOG.md (complete changelog)

### Changed
- (No changes)

### Fixed
- (No fixes)

### Removed
- (No removals)

---

## [0.9.0] - Never Released (Development Version)

### Added
- Foundation project structure
- KMP shared module
- Android application module
- Domain models (Message, Room, User)
- Repository interfaces
- Use case interfaces
- Platform interfaces (BiometricAuth, SecureClipboard, NotificationManager, NetworkMonitor)
- Base ViewModel
- Design system (Theme, Colors, Typography, Shapes)
- Atomic UI components (Button, InputField, Card)
- Molecular UI components
- Onboarding flow (5 screens)
- Chat screen (enhanced)
- Platform implementations (Android)
- Database layer (Room + SQLCipher)
- Offline sync layer
- Performance monitoring
- Accessibility compliance
- E2E testing suite
- Release configuration

### Changed
- (No changes)

### Fixed
- (No fixes)

### Removed
- (No removals)

---

## [0.1.0] - Never Released (Initial Prototype)

### Added
- Initial project setup
- Basic KMP configuration
- Basic Compose configuration
- Placeholder screens
- Placeholder components

### Changed
- (No changes)

### Fixed
- (No fixes)

### Removed
- (No removals)

---

## Future Releases

### [1.1.0] - Planned

#### Planned Features
- Real Matrix client integration
- Real authentication flow
- Repository implementations
- Use case implementations
- Real-time messaging (WebSocket)
- Push notification handling
- Registration screen
- Forgot password screen
- Biometric enrollment flow
- Real FCM integration
- Real Amplitude/Mixpanel integration
- Real certificate pins
- Incremental sync
- Delta sync
- Conflict UI
- Message expiration configuration
- Sync priority UI
- Sync history and analytics

#### Planned Improvements
- iOS platform integrations
- Background sync on cellular
- Message search optimization
- Performance optimizations
- Memory optimizations
- APK size optimization
- Accessibility improvements
- UI/UX improvements

#### Planned Fixes
- (No fixes yet)

---

## Version History

| Version | Date | Status | Notes |
|---------|------|---------|-------|
| 1.0.0 | 2026-02-10 | ✅ Released | Initial stable release |
| 0.9.0 | Never | 🚫 Cancelled | Development version |
| 0.1.0 | Never | 🚫 Cancelled | Initial prototype |

---

## Semantic Versioning

ArmorClaw follows [Semantic Versioning](https://semver.org/):

**MAJOR.MINOR.PATCH**

- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible functionality additions
- **PATCH**: Backwards-compatible bug fixes

---

## Changelog Format

This changelog follows the [Keep a Changelog](https://keepachangelog.com/) format.

### Sections

- **Added**: New features
- **Changed**: Changes in existing functionality
- **Deprecated**: Soon-to-be removed features
- **Removed**: Removed features
- **Fixed**: Bug fixes
- **Security**: Vulnerability fixes

---

## Contributors

- **ArmorClaw Team** - Initial development

---

## Support

- **Email**: support@armorclaw.app
- **Issues**: [GitHub Issues](https://github.com/armorclaw/ArmorClaw/issues)
- **Discussions**: [GitHub Discussions](https://github.com/armorclaw/ArmorClaw/discussions)

---

**Made with ❤️ for privacy**

*ArmorClaw - Secure. Private. Encrypted.*
