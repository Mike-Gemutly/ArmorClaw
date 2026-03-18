# Lessons Learned

## 2026-02-24: ArmorChat ↔ ArmorClaw Setup Integration Review

### Lesson 1: Document-Code Drift in RPC Methods
**Problem:** ArmorClaw.md documented `provisioning.*` RPC methods (start, status, cancel, claim, rotate) in the architecture diagram, but ArmorChat's `BridgeRpcClient` had zero provisioning methods implemented.
**Root Cause:** The bridge (Go) and client (Kotlin) were developed in parallel without a shared RPC contract file.
**Fix:** Added all 5 provisioning methods to `BridgeRpcClient`, `BridgeRpcClientImpl`, and `RpcModels.kt`.
**Prevention:** When new RPC methods are added to the bridge, immediately add stubs to the client interface. Consider generating client stubs from bridge RPC schema.

### Lesson 2: First-Boot Bootstrap Chicken-and-Egg Problem
**Problem:** Device registration required admin approval, but during first-boot there IS no admin to approve. This created a deadlock for the first user.
**Root Cause:** The setup flow assumed an admin always exists, which is false at installation time.
**Fix:** Added provisioning claim flow — the setup QR code includes a `setup_token` that ArmorChat uses to bypass admin approval and auto-claim OWNER role during first-boot.
**Prevention:** Always design onboarding flows starting from the "empty server" state. Test the very first user experience, not just subsequent user flows.

### Lesson 3: QR Payload Completeness
**Problem:** The QR code generated during ArmorClaw setup contained server URLs but no provisioning token. `SignedServerConfig.setupToken` field existed in ArmorChat but was never populated by the bridge.
**Root Cause:** The QR payload was designed for configuration only, not for provisioning/claiming.
**Fix:** Updated QR format documentation to include `setup_token`. ArmorChat's `parseSignedConfig()` now stores the token and uses it during the claim step.
**Prevention:** When adding new security flows, trace the full end-to-end path from server generation to client consumption. Ensure all payload fields are documented and tested.

### Lesson 4: Cryptographic Signing Must Be Production-Grade from Day One
**Problem:** `InviteService.signPayload()` used a simple fold-hash (`bytes.fold(0L) { acc * 31 + byte }`) instead of HMAC-SHA256. This is trivially forgeable.
**Root Cause:** Implemented as a "placeholder" during initial development and never upgraded.
**Fix:** Replaced with a proper HMAC-SHA256 construction using a pure-Kotlin SHA-256 implementation for cross-platform compatibility.
**Prevention:** Never ship placeholder cryptography. Even for MVPs, use proper constructions. Weak signing invites tampering of invite links.

### Lesson 5: Graceful Degradation for Missing RPC Endpoints
**Problem:** If ArmorChat connects to an older ArmorClaw bridge that doesn't support `provisioning.claim`, the setup would fail.
**Fix:** The provisioning claim is wrapped in a try/catch that falls back to the existing server-authoritative role check (`bridge.status` with `userRole`). Both old and new bridges work correctly.
**Prevention:** Always design new RPC features with fallback paths. Client should never hard-fail on a missing server method unless that method is absolutely required.

### Lesson 6: State Machine Completeness
**Problem:** `SetupState` had no states for the provisioning claim step, making it impossible for the UI to show progress during admin claim.
**Fix:** Added `SetupState.ClaimingAdmin`, `SetupState.ProvisioningExpired`, and `SetupState.AlreadyClaimed` states.
**Prevention:** When adding a new step to a multi-step flow, always add corresponding state machine states FIRST, then implement the logic.

### Lesson 7: Documentation Must Be Updated in ALL Sections, Not Just One
**Problem:** After adding `setup_token` to the code and QR format, only the v8.1 QR section was updated. Eight other locations in ArmorClaw.md still referenced the old payload without `setup_token`: the v7.1 config flow, v7.4.0 QR URL format, v7.4.2 QR fix payload, v0.2.0 provisioning protocol, client integration instructions, and the RPC method table (which used `provisioning.rotate_secret` instead of `provisioning.rotate`).
**Root Cause:** Large markdown documents often describe the same concept in multiple sections for different audiences. Updating only one section creates internal contradictions.
**Fix:** Grep'd the entire doc for all references to QR payloads, provisioning methods, and setup flow — then fixed all 8 locations in a single pass.
**Prevention:** After any payload or API change, search the full doc for ALL mentions of the changed fields/methods. Treat documentation as code: a single rename should be applied everywhere.

### Lesson 8: RPC Method Names Must Match Exactly Between Client and Server
**Problem:** ArmorChat called `push.register` and `push.unregister` but ArmorClaw bridge exposes `push.register_token` and `push.unregister_token`. This would cause `-32601 Method Not Found` at runtime, silently breaking push notifications.
**Root Cause:** The Kotlin method names (`pushRegister`, `pushUnregister`) were shortened during implementation without cross-referencing the bridge RPC spec. Similarly, `matrix.invite` should be `matrix.invite_user`, `matrix.typing` should be `matrix.send_typing`, and `matrix.read_receipt` should be `matrix.send_read_receipt`.
**Fix:** Updated all 5 method name strings in `BridgeRpcClientImpl.kt` to match the ArmorClaw.md RPC reference exactly.
**Prevention:** Maintain a single RPC method name registry. When implementing a client RPC call, copy the method string from the doc, never abbreviate it.

### Lesson 9: Non-Essential Connections Must Not Block Setup
**Problem:** `SetupService.connectWithCredentials()` threw `SetupException.WebSocketFailed` if the bridge WebSocket connection failed, completely blocking setup. But ArmorClaw.md states ArmorChat does NOT use Bridge WebSocket — only ArmorTerminal does.
**Root Cause:** The WebSocket connection was wired into the setup flow as a hard requirement, contradicting the architecture which says ArmorChat gets all real-time events from Matrix `/sync`.
**Fix:** Made WebSocket connection non-fatal — failures are logged but setup continues.
**Prevention:** Before making any external connection a hard requirement in setup, verify against the architecture doc whether the client actually needs it. Non-essential connections should always be best-effort.

### Lesson 10: Enum Serialization Requires Case-Sensitive Alignment
**Problem:** `BridgeStatusResponse.userRole` comment said lowercase values (`"owner"`, `"admin"`) but the `AdminLevel` enum expects uppercase (`OWNER`, `ADMIN`). If a bridge developer followed the comment, kotlinx.serialization would fail to deserialize the response.
**Root Cause:** The code comment was written based on conventional JSON naming (lowercase) rather than the actual Kotlin enum names.
**Fix:** Updated the comment to specify uppercase values and added `(uppercase strings)` clarification.
**Prevention:** When documenting enum fields in model comments, always reference the exact enum constant names. Consider adding `@SerialName` annotations to enums for explicit case control.

## 2026-02-24: Critical Bug Fixes — Push, Bridge Verification, Migration, Key Backup

### Lesson 11: Push Notification Registration Must Be Dual-Channel
**Problem:** `PushNotificationRepositoryImpl` only registered FCM tokens with the Bridge RPC (`push.register_token`) but never called the Matrix SDK's `setPusher()`. This meant the Matrix homeserver had no knowledge of the device's push gateway, so push notifications were silently dropped for all Matrix-native events.
**Root Cause:** The initial implementation assumed the bridge would relay all push triggers, but Matrix events (mentions, DMs, room invites) originate from the homeserver, which needs its own pusher registration.
**Fix:** Added `setPusher()` and `removePusher()` to `MatrixClient` interface and implementation. Refactored `PushNotificationRepositoryImpl` to accept both `BridgeRpcClient` and `MatrixClient`, performing dual registration on every token refresh.
**Prevention:** When integrating push in a bridged architecture, always map out which servers originate push-worthy events. Each origin server needs its own push registration path. Never assume a single relay covers all event sources.

### Lesson 12: Security Features Need Discoverable UI Entry Points
**Problem:** `EmojiVerificationScreen` existed and worked, but there was no way for a user to initiate bridge device verification from the room UI. The only path was deep-linking or developer tools.
**Root Cause:** The verification flow was built bottom-up (screen first, navigation never wired) without a corresponding top-down UX entry point.
**Fix:** Added `BridgeVerificationBanner` to `RoomDetailsScreen` with a visible warning banner and "Verify Bridge Device" button. Added `BRIDGE_VERIFICATION` route and wired navigation through to the existing `EmojiVerificationScreen`.
**Prevention:** Every security-critical flow must have at least one discoverable UI entry point defined BEFORE implementation. If the user can't find it, it doesn't exist. Design the trigger (banner, button, menu item) at the same time as the target screen.

### Lesson 13: Version Migration Must Be a First-Class Onboarding State
**Problem:** Users upgrading from ArmorChat v2.5 (Bridge-only sessions) to v3.0 (Matrix-native) had no migration path. The app would launch into a broken state — legacy credentials existed in `AppPreferences` but the new `MatrixClient` couldn't use them.
**Root Cause:** The upgrade path was never designed. `SplashViewModel.checkInitialState()` only checked for "logged in" vs "not logged in", with no intermediate "needs migration" state.
**Fix:** Added `SplashTarget.Migration` state, `hasLegacyBridgeSession()` detection in `AppPreferences`, and a full `MigrationScreen` with auto-migration, manual recovery phrase entry, and error handling.
**Prevention:** When planning a major architectural change (e.g., switching auth backends), design the migration flow FIRST — before any new feature work. Add migration detection to the app's startup state machine as a defined state, not an afterthought.

### Lesson 14: Critical Security Flows Must Be Wired Into the User Journey
**Problem:** The Matrix SDK supported key backup (recovery phrase generation, server-side encrypted backup), but ArmorChat never surfaced it in onboarding or settings. Users had no way to back up encryption keys, meaning a device loss would permanently lose message history.
**Root Cause:** Key backup was treated as a "nice-to-have" post-launch feature rather than a core security requirement. The SDK capability existed but no UI was built.
**Fix:** Created `KeyBackupSetupScreen` with a 6-step guided flow (Explain → Generate → Display → Verify → Store → Success). Wired it into onboarding completion (mandatory before reaching Home) and into `SecuritySettingsScreen` (optional re-entry).
**Prevention:** Inventory all SDK security capabilities (key backup, cross-signing, device verification) at project start and ensure each has a corresponding UI flow in the onboarding checklist. Security features without UI are effectively missing features.

## 2026-02-24: Architecture Review — Strip Legacy, Add Bridge UX, Governance UI, Key Recovery

### Lesson 15: Dead Code Creates Confusion About Encryption Ownership
**Problem:** `EncryptionService` (expect/actual) was a pass-through in SERVER_SIDE mode that did nothing — but its presence in the DI graph and codebase implied it was actively encrypting messages. Developers couldn't tell whether the Rust SDK or the legacy service was responsible for encryption.
**Root Cause:** During the Thin→Thick Client migration, the old encryption layer was left in place "just in case" rather than being decisively removed.
**Fix:** Deleted `EncryptionService.kt` and `EncryptionService.android.kt`. Created `EncryptionTypes.kt` retaining only the type definitions. Removed `encryptionModule` from DI.
**Prevention:** When migrating to a new system (e.g., Matrix Rust SDK for encryption), delete the old system immediately after validation. Legacy code that "does nothing" still creates cognitive overhead and potential confusion.

### Lesson 16: Bridged Users Must Be Visually Distinguishable
**Problem:** Ghost users bridged from Slack/Discord/Teams appeared identical to native Matrix users in the chat UI. Users couldn't tell if they were messaging someone on Slack vs. Matrix.
**Root Cause:** The `MessageSender.UserSender` model had no concept of platform origin. The bridge creates Matrix "ghost" accounts that look native.
**Fix:** Added `BridgePlatform` enum and `bridgePlatform` field to `UserSender`. Created `BridgedOriginBadge` composable as an avatar overlay showing the origin platform icon.
**Prevention:** When designing a multi-platform bridging system, always include origin metadata in the user/message model from day one. Visual disambiguation is not optional — it's a trust signal.

### Lesson 17: Feature Gates Must Respect Bridge Capabilities
**Problem:** Edit and Reaction buttons appeared in bridged rooms even when the target platform (e.g., Slack) doesn't support message editing or reactions via the bridge.
**Root Cause:** `canReact()` and `canEdit()` only checked the message type, not the room's bridge capabilities.
**Fix:** Added `BridgeRoomCapabilities` to `Room` model. Updated `canReact()`, `canEdit()`, and new `canThread()` to accept optional capabilities and suppress features when the bridge doesn't support them.
**Prevention:** Every interactive feature in a chat UI should check room-level capability flags, not just message-level properties. Bridge limitations must propagate to the UI layer.

### Lesson 18: Enterprise Governance Events Need Dedicated UI Components
**Problem:** Server-side governance events (license expiry, HIPAA content scrubbing) had no UI representation. Admins had no visibility into license status, and users saw no indication when a message was policy-redacted.
**Root Cause:** The `SystemEventType` enum covered room/workflow events but had no governance category. Enterprise features were treated as "future work" with no placeholder UI.
**Fix:** Added `LICENSE_WARNING`, `LICENSE_EXPIRED`, `CONTENT_POLICY_APPLIED` to `SystemEventType`. Created `GovernanceBanner` for HomeScreen and `ContentPolicyPlaceholder` for inline message redaction display.
**Prevention:** When building for enterprise deployment, governance UI is not optional. Every server-side policy action (content filtering, license enforcement) must have a corresponding client-side feedback component.

### Lesson 19: Key Recovery Must Be Accessible from Login, Not Just Onboarding
**Problem:** Key backup setup existed in onboarding, but there was no way for a returning user (who already completed onboarding) to recover keys from the login screen. A user who reinstalled the app or got a new device had no path to restore their encryption keys.
**Root Cause:** Key recovery was only considered in the context of first-time setup, not the returning-user flow.
**Fix:** Added `onRecoverKeys` callback and "Recover encryption keys" link to `LoginScreen`. Created `KeyRecoveryScreen` with 3-step flow (Enter Phrase → Verify → Restore). Added `KEY_RECOVERY` route.
**Prevention:** For any security credential flow (key backup, 2FA recovery), always design BOTH the creation path (onboarding) AND the recovery path (login/re-auth) at the same time. If you build "save" without "restore", the feature is incomplete.

## 2026-02-24: Push Notification Refactor — Stub-to-Real Implementation

### Lesson 20: Stub Implementations That Log Success Are the Most Dangerous Kind of Bug
**Problem:** `MatrixClientImpl.setPusher()` was a complete stub — it logged `"Pusher set successfully"` and returned `Result.success(Unit)` without making any HTTP call. The homeserver never received a pusher registration, so **no background push notifications worked**. The app appeared to register pushers correctly (dual-registration logic in `PushNotificationRepositoryImpl` ran without errors), but the underlying call was a no-op.
**Root Cause:** During initial Matrix SDK integration, `setPusher()` was stubbed with a `// TODO: Implement with Matrix SDK` comment. The stub returned success, so all callers assumed it worked. No integration test verified that the homeserver actually received the registration.
**Fix:** Replaced with real HTTP POST to `/_matrix/client/v3/pushers/set` using the session access token. Added `removePusher()` (kind="" per spec). Added `syncOnce()` call in `ArmorFirebaseMessagingService.onMessageReceived()` so the SDK fetches/decrypts event content on push wake.
**Prevention:** **Stubs must never return success.** A stub should either: (1) throw `NotImplementedError("setPusher not yet implemented")`, (2) return `Result.failure()` with a clear message, or (3) log at WARN/ERROR level. A stub that silently succeeds is indistinguishable from a working implementation and will never be caught by callers. Additionally, add integration tests that verify the actual HTTP call is made, not just that the return value is success.

### Lesson 21: Migration Screens Must Wipe Legacy Data, Not Just Read It
**Problem:** `MigrationScreen` detected legacy Bridge sessions and attempted key migration, but never cleaned up legacy data afterwards. The old SQLCipher database (`bridge_store.db`), Bridge SharedPreferences (`bridge_session`), and legacy onboarding prefs remained on disk after migration — wasting storage and risking accidental reads from stale schemas.
**Root Cause:** The migration flow was designed only for the "read and transfer" phase, not the "clean up old data" phase. `clearLegacyBridgeSession()` existed in `AppPreferences` but was never called from `MigrationScreen`.
**Fix:** Added `clearLegacyDatabase()` (deletes SQLCipher files) and `wipeLegacyData()` (combines all legacy cleanup) to `AppPreferences`. Wired `wipeLegacyData()` into all MigrationScreen exit paths (success, skip, logout).
**Prevention:** Every migration flow needs three phases: (1) detect, (2) transfer, (3) clean up. Design all three at the same time. If you build "detect + transfer" without "clean up", old data persists indefinitely.

## 2026-02-25: Frontend UX Review — 7 Fixes

### Lesson 22: Mandatory Security Steps Must Survive Process Death
**Problem:** Users could force-quit the app during key backup setup (between login and backup completion), then reopen it and skip straight to Home — permanently bypassing key backup.
**Root Cause:** `SplashViewModel` checked `isLoggedIn` and `hasValidSession` but had no concept of an "incomplete backup" state. Once the session was valid, the user went to Home regardless of backup status.
**Fix:** Added `isBackupComplete` flag to `AppPreferences`. `SplashViewModel` now checks this: logged in + valid session + backup incomplete → re-route to `KeyBackupSetupScreen`. Replaced "Skip for now" with a scary confirmation dialog that does NOT mark backup as complete.
**Prevention:** For any mandatory security step in onboarding, persist a separate completion flag. The splash/startup state machine must check ALL required steps, not just authentication. If a step is skippable, the skip must NOT mark the step as complete — the user should be re-prompted on next launch.

### Lesson 23: Destructive Actions Need Unambiguous Labels
**Problem:** The migration "Skip" button said "Skip (you may lose old message history)" — a vague label that undersells the severity. Users might interpret "may" as unlikely rather than certain.
**Root Cause:** The label was written from a technical perspective ("data loss is possible") rather than a user perspective ("your data will be deleted").
**Fix:** Renamed to "Start Fresh (Delete Old Data)" with a subtitle: "Skips migration and permanently erases v2.5 message history". Used red warning color.
**Prevention:** Labels for destructive actions should always state the concrete outcome, not the probability. Use explicit verbs ("Delete", "Erase") instead of vague ones ("Skip", "Remove").

### Lesson 24: Security Indicators Must Be Visible in the Primary UI, Not Buried
**Problem:** Bridge verification status was only visible in Room Details (a secondary screen). Users chatting in an unverified bridged room had no visual indication of the trust risk.
**Root Cause:** The verification banner was placed where it was convenient to implement (Room Details), not where users would actually see it (Chat screen).
**Fix:** Added a yellow shield icon to `ChatTopBar` when bridge is unverified. Added a first-entry modal dialog explaining the risk with "I Understand" / "Leave Room" options.
**Prevention:** Security indicators must be visible in the screen where the user performs the risky action. If encryption/verification status matters, it belongs in the chat bar — not in a settings page users visit once.

### Lesson 25: Silent Feature Disabling Creates Confusion
**Problem:** When bridge capabilities suppressed reactions/edits/threads, buttons were silently disabled with no feedback. Users tapped disabled buttons and nothing happened, with no explanation.
**Root Cause:** The `canReact()`/`canReply()` checks returned early without emitting any UI event.
**Fix:** Added `UiEvent.ShowSnackbar` and emit it before returning from suppressed feature checks. User now sees "Reactions are not available for this message type".
**Prevention:** Every feature gate (capability check, permission check) that disables a user-facing action must provide feedback when the user attempts the action. Silent disabling is a UX anti-pattern.

### Lesson 26: Deep Links Must Construct a Proper Backstack
**Problem:** Tapping a push notification deep link navigated directly to the chat screen with no Home in the backstack. Pressing Back exited the app instead of going to Home.
**Root Cause:** `handleDeepLinkAction()` used `popUpTo(HOME)` which assumes Home is already in the stack, but it wasn't when launching from a cold start.
**Fix:** Check if Home exists in the backstack; if not, navigate to Home first (clearing the stack), then push the deep link target on top.
**Prevention:** Deep link navigation should always construct the full expected backstack, never assume intermediate screens exist. Test deep links from both warm-start and cold-start scenarios.

### Lesson 27: Push Notification Handlers Need Fallback Paths
**Problem:** When `syncOnce()` failed in `onMessageReceived()`, the catch block only logged a warning. If the sync failed AND the push payload had no notification body, the user received no notification at all.
**Root Cause:** The sync failure handler assumed the push payload always contained a displayable notification, but Matrix push payloads often contain only metadata (`room_id`, `event_id`).
**Fix:** Added `showFallbackNotification()` in the catch block that posts a generic "You may have new messages" notification using only push metadata.
**Prevention:** Every background operation in a push handler needs a fallback path. If the primary operation fails, there must still be a visible result (notification, badge update, etc.).

### Lesson 28: Ephemeral UI State Must Be Persisted for Enterprise Features
**Problem:** `GovernanceBanner` dismiss state was stored in ephemeral `remember { mutableStateOf(true) }`, so dismissed banners reappeared on every screen re-composition or app restart.
**Root Cause:** Banner dismissal was implemented as a quick UI-only feature without considering persistence.
**Fix:** Added timestamp-based persistence in `AppPreferences`. CRITICAL banners re-show after 24 hours. License Expired banners cannot be dismissed at all (close button hidden).
**Prevention:** Any dismissible UI element for enterprise/compliance features must have persisted state. Ask: "Should this come back after restart?" and "Should this come back after a time period?" — then implement accordingly.

## 2026-02-26: Setup Flow Review — 8 Fixes

### Lesson 29: Async Operations Cannot Be Followed by Synchronous State Checks
**Problem:** `AdvancedSetupScreen.onConnect` called `viewModel.startSetup()` (which launches a coroutine) then immediately checked `uiState.canProceed` — which was always `false` because the coroutine hadn't completed. `connectWithCredentials` was **never called**, making Advanced mode completely broken.
**Root Cause:** Treating a ViewModel coroutine launch as a synchronous call. The state update happens on a future frame, not on the current one.
**Fix:** Implemented a two-phase connect pattern: store pending credentials in local state, then use `LaunchedEffect` to observe `canProceed` and auto-call `connectWithCredentials` when it becomes `true`.
**Prevention:** Never check reactive state immediately after launching an async operation. Use reactive observation (`LaunchedEffect`, `collect`, `snapshotFlow`) to respond to state changes.

### Lesson 30: Mandatory Security Gates Must Apply to ALL Setup Paths
**Problem:** Bridge health gating (CTO requirement) was enforced in `startSetup()` but completely bypassed by `handleQrProvision()` and `handleQrProvisionWithAuth()`. QR-scanned setup could proceed to credential entry with an unreachable bridge.
**Root Cause:** Health gating was added to the manual path but the QR paths were not updated. Code review missed the parallel paths.
**Fix:** Added `performBridgeHealthCheck()` to both QR methods. Also added a health pre-check to `connectWithCredentials()` as a defense-in-depth guard.
**Prevention:** When adding a mandatory check (security, validation, health), grep for ALL entry points that reach the gated step. A single path that bypasses the gate defeats the purpose entirely.

### Lesson 31: IP Address Detection Must Use Strict Validation
**Problem:** `isIpAddress()` used a heuristic `host.first().isDigit() && host.contains(".")` which matched domain names starting with digits (e.g., `1password.com`, `3com.net`), causing incorrect bridge URL derivation.
**Root Cause:** A lazy fallback added alongside the proper IPv4 regex, intended for edge cases but creating false positives.
**Fix:** Removed the heuristic check. Now only the strict IPv4 regex is used. Applied fix in all four copies: `SetupService.isIpAddress()`, `SetupViewModel.isIpAddress()`, `ConnectServerScreen.isIpAddress()`, and `BridgeConfig.Companion.isIpAddress()` in `RpcModels.kt`.
**Note:** The initial fix missed the `BridgeConfig.Companion` copy — see Lesson 37.
**Prevention:** IP address validation should only use standardized patterns. Heuristics like "starts with digit" are too broad. If you need to detect IPs that don't match standard patterns, handle those specific cases explicitly.

### Lesson 32: URL Decoding Must Be Atomic, Not Sequential
**Problem:** `decodeUrlPart()` decoded `%25` (literal percent) mid-sequence, meaning a URL like `%2525` would first become `%25` (correct), then the second pass of replacements could re-decode it. The sequential replace chain had ordering dependencies.
**Root Cause:** Manual character-by-character percent decoding using chained `.replace()` calls, with `%25` placed arbitrarily in the middle of the chain.
**Fix:** Replaced the entire chain with a single regex-based decoder: `Regex("%([0-9A-Fa-f]{2})")` that decodes all percent-encoded characters in one atomic pass.
**Prevention:** URL decoding must be done in a single pass (regex or state machine), never as sequential string replacements. Sequential replacement creates ordering bugs that are hard to spot.

### Lesson 33: Resource-Allocating `remember` Blocks Need Corresponding `DisposableEffect` Cleanup
**Problem:** `ConnectServerScreen` created `Executors.newSingleThreadExecutor()` and `BarcodeScanning.getClient()` via `remember {}` but never shut them down. Each navigation to/from the screen leaked a thread and native resources.
**Root Cause:** `remember {}` was used for convenience without considering the lifecycle implications. No corresponding cleanup was added.
**Fix:** Added `DisposableEffect(Unit) { onDispose { executor.shutdown(); barcodeScanner.close() } }`.
**Prevention:** Every `remember {}` that creates a closeable/disposable resource must be paired with a `DisposableEffect` for cleanup. Treat `remember` + resource allocation as a red flag during code review.

### Lesson 34: Permission UI Must Invoke System Permission APIs, Not Just Toggle State
**Problem:** `PermissionsScreen` showed Grant buttons that only toggled a local `granted` boolean — they never called Android's `ActivityResultContracts.RequestPermission()`. The screen always showed permissions as "granted" after tapping, but the OS never actually granted them.
**Root Cause:** The screen was built as a UI mockup and the actual permission request was never wired in.
**Fix:** Added `rememberLauncherForActivityResult` with `RequestPermission` contract. Grant button now calls `permissionLauncher.launch(manifestPermission)`. Added `LaunchedEffect` to check already-granted permissions on composition. Handles pre-Android 13 notification permission (no runtime request needed).
**Prevention:** Permission screens must be tested on a real device/emulator with permissions reset. A permission screen that works without the system dialog is a dead giveaway that it's not wired to the OS.

### Lesson 35: Bridge API Response Fields Must Be Consumed Explicitly, Not Silently Dropped
**Problem:** `bridge.health` RPC returned `bridge_ready` and `is_new_server` fields, but `performBridgeHealthCheck()` only consumed `healthy` and `reason`. `detectServer()` only consumed `version`, `supports_e2ee`, `supports_recovery`, and `region`. New fields were silently ignored, meaning the client couldn't distinguish a healthy-but-initializing bridge from a fully ready one, and couldn't detect first-boot servers for auto-claim.
**Root Cause:** `healthCheck()` returns a generic `Map<String, Any?>` from raw JSON. Consumers only extracted the fields they knew about at implementation time. When the bridge added new fields, no compile-time error flagged the unconsumed data.
**Fix:** (1) `performBridgeHealthCheck()` now extracts `bridge_ready` — if healthy but not ready, returns `NOT_READY` status with a "Bridge is starting up" message that blocks credential entry. (2) `detectServer()` now extracts `is_new_server` into `DetectServerInfo.isNewServer`. (3) Added `BridgeHealthStatus.NOT_READY` enum value. (4) `isNewServer` propagated through `SetupUiState` and `BridgeHealthDetails` for the provisioning claim flow.
**Prevention:** When consuming untyped API responses (`Map<String, Any?>`), document ALL expected fields at the call site, even if you don't use them yet. Better: define typed response models (like `BridgeStatusResponse`) so the compiler flags missing fields. When the bridge API adds new fields, audit all consumers.

### Lesson 36: Client-Server Field Name Contracts Must Be Validated Against the Spec, Not Assumed
**Problem:** ArmorChat's `performBridgeHealthCheck()` checked `data["healthy"] as? Boolean ?: true`, but the bridge's HTTP `/health` endpoint returns `"status": "ok"` (a string). If the `bridge.health` JSON-RPC method mirrors the HTTP format, the `healthy` key would be absent and ArmorChat would always default to `true` — silently masking unhealthy bridges. Additionally, `provisioning_available`, `server_name`, and `admin_token` from the claim response were never consumed despite being documented in the ArmorClaw spec.
**Root Cause:** The client code was written against an assumed API contract, not the actual spec document. When the bridge team enriched the `/health` endpoint in 0.3.4, the JSON-RPC method potentially changed to use `status` instead of `healthy` — but no client-side review was performed.
**Fix:** (1) Health check now checks both `data["healthy"]` (boolean) and falls back to `data["status"] == "ok"` (string). (2) `provisioning_available` consumed and stored in `BridgeHealthDetails`. (3) `admin_token` from `provisioning.claim` response persisted in `SetupCompleteInfo`. (4) `SignedServerConfig` gained `version` and `bridge_public_key` fields matching the QR payload spec.
**Prevention:** Whenever the bridge team publishes a spec or changelog (like ArmorClaw.md 0.3.4), the client team must diff every documented response field against the client's parsing code. Field name mismatches between server spec and client code are silent bugs that only manifest at runtime. Default values on missing fields (like `?: true`) are especially dangerous because they hide the mismatch.

### Lesson 37: Utility Function Fixes Must Be Applied to ALL Copies
**Problem:** Lesson 31 fixed `isIpAddress()` in `SetupService`, `SetupViewModel`, and `ConnectServerScreen` — but missed the fourth copy in `BridgeConfig.Companion.isIpAddress()` inside `RpcModels.kt`. The loose heuristic (`host.first().isDigit() && host.contains(".")`) survived there, meaning `deriveBridgeUrl()` could still misclassify domains like `3com.example.com` as IP addresses and derive the wrong bridge URL (e.g., `http://3com.example.com:8080` instead of `https://bridge.example.com`).
**Root Cause:** The codebase had four independent copies of the same utility function. When the fix was applied, a grep for `isIpAddress` found 3 of 4 call sites but missed the `BridgeConfig.Companion` private function because it was in a different file (`RpcModels.kt`) than the other three (all in the `bridge/` package but different classes).
**Fix:** Applied the same strict IPv4 regex-only pattern to `BridgeConfig.Companion.isIpAddress()`. Removed the heuristic fallback.
**Prevention:** (1) When fixing a bug in a utility function, grep the ENTIRE project for all copies of that function — not just the file where the bug was found. (2) Better yet, extract shared utility functions into a single location and call them from all sites. Four copies of the same function is a maintenance hazard. (3) When a lesson says "applied fix in N copies", verify the count by searching the full codebase.

## 2026-02-26: Compatibility Review Document Fixes

### Lesson 38: Review Documents Must Be Validated Against the Codebase Before Publishing
**Problem:** `ArmorChat.md` claimed 42 navigation routes, but the actual `AppNavigation.kt` contained 55 `const val` route definitions. The document also had broken markdown table syntax (extra leading `|` characters), contradictory statements (GovernanceBanner listed as both fixed and unfixed), migration version inconsistencies (v3.0 vs v4.6), and duplicate content across sections.
**Root Cause:** The review document was updated incrementally across multiple review passes without a final reconciliation pass. Each pass added new sections but didn't update earlier sections' counts and claims.
**Fix:** Corrected route count to 55, fixed table syntax, standardized migration version to v4.1, removed GovernanceBanner contradiction, deduplicated Future Considerations vs Required Changes, and reordered Spec Alignment before Production Readiness verdict.
**Prevention:** Before marking a review document as final: (1) grep the codebase to verify all numeric claims (route counts, field counts, copy counts), (2) search the document for internal contradictions between "Fixed" items and "Future" items, (3) run a markdown linter to catch syntax issues.

## 2026-02-26: ArmorClaw Compatibility Fixes — RC-01 through RC-08

### Lesson 39: DI Wiring Must Be Updated When Constructor Signatures Change
**Problem:** RC-04 added a `pushGatewayUrlProvider` parameter to `PushNotificationRepositoryImpl`, but the DI site (`AppModules.kt`) was not updated — it still passed only 2 positional arguments. The third parameter silently defaulted to `{ null }`, making the entire RC-04 fix dead code. Push notifications always used the hardcoded gateway URL.
**Root Cause:** The constructor change had a default value, so the Kotlin compiler didn't flag the DI site as a build error. No test verified that the dynamic URL was actually being read.
**Fix:** Updated `AppModules.kt` to inject `SetupService` and pass `{ setupService.config.value.pushGateway }` as the `pushGatewayUrlProvider`.
**Prevention:** When adding a new parameter to a dependency-injected class — even with a default value — always grep for ALL construction sites and update them. Default values on DI-injected parameters are dangerous because they compile cleanly but silently use the wrong value. Consider making critical provider parameters non-optional (no default) so the compiler forces all callers to supply them.

### Lesson 40: Data Mapping Between Models Must Transfer ALL Fields
**Problem:** `parseSignedConfig()` decoded a `SignedServerConfig` from the QR payload but then created a `SetupConfig` that dropped `pushGateway`, `wsUrl`, `serverName`, and `expiresAt`. These fields existed in both models but were not mapped. The secondary deep link parsers (`parseSetupDeepLink`, `parseSetupWebLink`) did derive pushGateway correctly, making this a first-path-only bug.
**Root Cause:** The `SetupConfig()` constructor call was written when `SetupConfig` had fewer fields. When `pushGateway`, `wsUrl`, etc. were added later, the constructor call was not revisited.
**Fix:** Added all missing field mappings: `wsUrl`, `pushGateway`, `serverName`, `expiresAt`, `serverVersion`.
**Prevention:** When two models share overlapping fields (e.g., API response → domain config), write a dedicated mapping function (e.g., `SignedServerConfig.toSetupConfig()`) rather than inline construction. This centralizes the mapping and makes it reviewable. When new fields are added to either model, the mapping function is the single place to update.

### Lesson 41: Bridge-Client Feature Parity Must Be Tracked Systematically
**Problem:** ArmorClaw spec defined `device.register` + `device.wait_for_approval` (Step 4b) for non-first-boot device registration with HITL approval. ArmorChat had no implementation of this flow — second devices silently fell through to the legacy `bridge.status` role check, which doesn't integrate with the provisioning system's admin approval workflow.
**Root Cause:** The original RC todo list was created from a targeted review of 5 specific issues. The device registration flow was in a different section of the spec and wasn't flagged because ArmorChat "worked" (via the fallback path) even without it.
**Fix:** Documented as RC-06 for future implementation. The fallback to `bridge.status` continues to work for now.
**Prevention:** Maintain a feature parity matrix between the bridge spec and client implementation. Each RPC method and flow in the spec should have a corresponding line item tracking its client-side status (implemented, stubbed, not started). A targeted issue list is not a substitute for systematic spec coverage tracking.
