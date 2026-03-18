# ArmorChat — Required Changes for ArmorClaw Compatibility

> **Date:** 2026-02-26
> **ArmorClaw Bridge Version:** 4.1.0 (spec 0.3.4)
> **Status:** 7 of 8 fixed — RC-06 (device registration flow) remains open

These are changes that must be made in the **ArmorChat Android codebase** to fully work with the ArmorClaw bridge. Each item includes the problem, what the bridge actually returns, and exactly what to change in ArmorChat.

---

## RC-01: Wire `admin_token` into RPC auth headers — HIGH

**Problem:** ArmorChat stores `admin_token` from `provisioning.claim` into `SetupCompleteInfo.adminToken` (SA-03), but never sends it on subsequent RPC calls. The bridge generates per-user tokens prefixed `aat_` and will reject auth-gated RPC methods without them.

**What the bridge returns:**
```json
// provisioning.claim response
{
  "success": true,
  "role": "OWNER",
  "admin_token": "aat_7f3a9b...",   // <-- this must be sent back on future calls
  "user_id": "@abc123:server",
  "device_id": "DEV_abc123",
  "message": "Admin role claimed successfully"
}
```

**What to change:**
- **File:** `BridgeRpcClientImpl.kt`
- **Change:** Add `Authorization: Bearer <admin_token>` header to the `httpPost()` method. Read the token from the stored `adminToken` (persisted via `SetupCompleteInfo`).
- **Example:**
```kotlin
// In httpPost(), before sending:
val adminToken = setupConfig.adminToken
if (adminToken != null) {
    connection.setRequestProperty("Authorization", "Bearer $adminToken")
}
```

---

## RC-02: Gate provisioning claim on `provisioning_available` — MEDIUM

**Problem:** ArmorChat extracts `provisioning_available` from the health response into `BridgeHealthDetails.provisioningAvailable` (SA-02), but the claim flow in `connectWithCredentials()` never checks it. If the bridge runs without a provisioning secret, `provisioning_available` is `false` — calling `provisioning.claim` returns an RPC error ("Provisioning not configured") instead of a clean result, confusing the setup flow.

**What the bridge returns:**
```json
// bridge.health response (when provisioning is NOT configured)
{
  "status": "ok",
  "bridge_ready": true,
  "provisioning_available": false,   // <-- must check this before claiming
  "is_new_server": true
}
```

**What to change:**
- **File:** `SetupService.kt` → `connectWithCredentials()`
- **Change:** Before calling `provisioningClaim()`, check `bridgeHealthDetails.provisioningAvailable`. If `false`, skip the claim step entirely and fall back to the `bridge.status` role check path.
- **Example:**
```kotlin
// In connectWithCredentials(), before the claim attempt:
if (bridgeHealthDetails.provisioningAvailable) {
    val claimResult = rpcClient.provisioningClaim(setupToken, deviceId, deviceName)
    // ... handle claim result
} else {
    // Skip claim, fall back to bridge.status role check
    val statusResult = rpcClient.bridgeStatus(userId)
    // ... extract role from statusResult
}
```

---

## RC-03: Consume `matrix_homeserver` and `correlation_id` from claim response — MEDIUM

**Problem:** The bridge returns `matrix_homeserver` and `correlation_id` in `provisioning.claim` responses, but `ProvisioningClaimResponse` only has 6 fields — these 2 are silently dropped by Kotlinx.serialization. The `matrix_homeserver` field is critical for self-hosted deployments where the bridge's configured homeserver differs from what was embedded in the QR payload.

**What the bridge returns:**
```json
// provisioning.claim response (full)
{
  "success": true,
  "role": "OWNER",
  "admin_token": "aat_7f3a9b...",
  "user_id": "@abc123:server",
  "device_id": "DEV_abc123",
  "message": "Admin role claimed successfully",
  "matrix_homeserver": "https://matrix.myserver.com",  // <-- not consumed
  "correlation_id": "req_abc123"                        // <-- not consumed
}
```

**What to change:**
- **File:** `RpcModels.kt` → `ProvisioningClaimResponse`
- **Change:** Add the two missing fields:
```kotlin
@Serializable
data class ProvisioningClaimResponse(
    val success: Boolean,
    val role: String? = null,
    @SerialName("admin_token") val adminToken: String? = null,
    @SerialName("user_id") val userId: String? = null,
    @SerialName("device_id") val deviceId: String? = null,
    val message: String? = null,
    // NEW — add these:
    @SerialName("matrix_homeserver") val matrixHomeserver: String? = null,
    @SerialName("correlation_id") val correlationId: String? = null,
)
```

- **File:** `SetupService.kt`
- **Change:** After a successful claim, if `matrixHomeserver` is non-null, override the QR-derived homeserver URL with it:
```kotlin
if (claimResult.success && claimResult.matrixHomeserver != null) {
    serverConfig = serverConfig.copy(matrixHomeserver = claimResult.matrixHomeserver)
}
```

---

## RC-04: Use push gateway URL from discovery instead of hardcoded default — MEDIUM

**Problem:** `PushNotificationRepository` hardcodes `https://push.armorclaw.app/_matrix/push/v1/notify` as the Sygnal push gateway URL. For self-hosted deployments, Sygnal runs on a different host/port — the hardcoded URL will silently fail (push registration succeeds on the homeserver side, but Sygnal never receives the push).

**Where the bridge provides the correct URL:**

| Source | Field | Example |
|--------|-------|---------|
| QR payload | `push_gateway` | `https://myserver.com:8008/_matrix/push/v1/notify` |
| `/discover` endpoint | `push_gateway` | same |
| `/.well-known/matrix/client` | `com.armorclaw.push_gateway` | same |
| `provisioning.start` response | `server_config.push_gateway` | same |

**What to change:**
- **File:** `PushNotificationRepository.kt`
- **Change:** Replace the hardcoded `PUSH_GATEWAY_URL` constant with a value read from `SetupConfig` or `ServerConfig`. The push gateway URL is already available in the QR payload and discovery responses — it just needs to be plumbed through.
- **Example:**
```kotlin
// Before (hardcoded):
private const val PUSH_GATEWAY_URL = "https://push.armorclaw.app/_matrix/push/v1/notify"

// After (dynamic):
private fun getPushGatewayUrl(): String {
    return serverConfig.pushGateway
        ?: "https://push.armorclaw.app/_matrix/push/v1/notify" // fallback for older bridges
}
```

---

## RC-05: Consolidate `isIpAddress()` into shared utility — LOW

**Problem:** SA-06 fixed the 4th copy of `isIpAddress()` to use a strict IPv4 regex, but 4 independent copies still exist across the codebase. Any future fix or IPv6 support must be applied 4 times — a maintenance hazard.

**Current copies:**

| # | File | Location |
|---|------|----------|
| 1 | `SetupService.kt` | `isIpAddress()` helper |
| 2 | `SetupViewModel.kt` | `isIpAddress()` helper |
| 3 | `ConnectServerScreen.kt` | `isIpAddress()` helper |
| 4 | `RpcModels.kt` | `BridgeConfig.Companion.isIpAddress()` |

**What to change:**
- **Create:** `shared/.../util/NetworkUtils.kt`
```kotlin
object NetworkUtils {
    private val IPV4_REGEX = Regex(
        "^((25[0-5]|2[0-4]\\d|[01]?\\d\\d?)\\.){3}(25[0-5]|2[0-4]\\d|[01]?\\d\\d?)$"
    )

    fun isIpAddress(host: String): Boolean = IPV4_REGEX.matches(host)
}
```
- **Update:** All 4 files above to call `NetworkUtils.isIpAddress()` instead of their local copies.
- **Delete:** The 4 local `isIpAddress()` implementations.

---

## RC-06: Implement `device.register` + `device.wait_for_approval` for non-first-boot devices — MEDIUM

**Problem:** ArmorClaw spec (Step 4b, "Bridge Registration") defines a flow for non-first-boot devices: when no `setup_token` is present (second device, reinstall, etc.), the device should call `device.register` and then poll `device.wait_for_approval` until the admin approves via Matrix notification. ArmorChat skips this entirely — it falls directly to the legacy `bridge.status` role check, which only works with older bridges and doesn't integrate with the provisioning system's HITL approval flow.

**What the bridge expects:**
```json
// Step 1: Register device
{"method": "device.register", "params": {"device_name": "Pixel 7", "device_type": "android"}}
// Response: {"status": "pending", "request_id": "req_..."}

// Step 2: Poll for approval
{"method": "device.wait_for_approval", "params": {"request_id": "req_..."}}
// Response (approved): {"status": "approved", "session_token": "...", "capabilities": [...]}
// Response (pending): {"status": "pending"}
// Response (rejected): {"status": "rejected"}
```

**What to change:**
- **File:** `BridgeRpcClient.kt` + `BridgeRpcClientImpl.kt` — Add `deviceRegister()` and `deviceWaitForApproval()` methods
- **File:** `RpcModels.kt` — Add `DeviceRegisterResponse` and `DeviceApprovalResponse` data classes
- **File:** `SetupService.kt` — In the no-setup-token branch (line 288), attempt `device.register` + polling before falling back to `bridge.status`
- **File:** `SetupState.kt` — Add `WaitingForAdminApproval(requestId: String)` state
- **UI:** Show a "Waiting for admin approval…" screen with polling indicator

**Note:** This is a larger feature requiring new UI and polling logic. Falls back gracefully to `bridge.status` if `device.register` returns `-32601`.

---

## RC-07: Wire `pushGatewayUrlProvider` at DI construction site — HIGH (FIXED)

**Problem:** `PushNotificationRepositoryImpl` accepts a `pushGatewayUrlProvider` parameter (added in RC-04), but `AppModules.kt` line 237 constructed it with only 2 args: `PushNotificationRepositoryImpl(get(), get())`. The third parameter defaulted to `{ null }`, so `pushGatewayUrlProvider()` always returned `null` and the hardcoded `DEFAULT_PUSH_GATEWAY_URL` was always used — making the RC-04 fix dead code.

**Fix applied:** `AppModules.kt` now reads `SetupService.config.value.pushGateway` via a lambda, completing the pipeline from QR/discovery → SetupConfig → PushNotificationRepository.

---

## RC-08: `parseConfigDeepLink` drops QR payload fields when creating SetupConfig — MEDIUM (FIXED)

**Problem:** When parsing the primary QR format (`armorclaw://config?d=...`), `parseSignedConfig()` created a `SetupConfig` that omitted `pushGateway`, `wsUrl`, `serverName`, and `expiresAt` from the decoded `SignedServerConfig`. These fields were present in the QR payload and used by other deep link parsers (`parseSetupDeepLink`, `parseSetupWebLink`), but the primary config path dropped them.

**Impact:** Self-hosted deployments using QR provisioning would always fall back to the hardcoded push gateway URL and lose the server's display name.

**Fix applied:** `SetupConfig` construction in `parseSignedConfig()` now carries over all fields: `wsUrl`, `pushGateway`, `serverName`, `expiresAt`, and `serverVersion` from the signed payload.

---

## Summary

| # | Change | Severity | Files | Status |
|---|--------|----------|-------|--------|
| RC-01 | Wire `admin_token` into RPC auth | HIGH | `BridgeRpcClient.kt`, `BridgeRpcClientImpl.kt`, `SetupService.kt` | ✅ Fixed |
| RC-02 | Gate claim on `provisioning_available` | MEDIUM | `SetupService.kt` | ✅ Fixed |
| RC-03 | Consume `matrix_homeserver` + `correlation_id` | MEDIUM | `RpcModels.kt`, `SetupService.kt` | ✅ Fixed |
| RC-04 | Dynamic push gateway URL (provider param) | MEDIUM | `PushNotificationRepository.kt` | ✅ Fixed |
| RC-05 | Consolidate `isIpAddress()` | LOW | 4 files + 1 new `NetworkUtils.kt` | ✅ Fixed |
| RC-06 | `device.register` + approval flow | MEDIUM | RPC client, models, SetupService, new UI | 🔲 Open |
| RC-07 | Wire `pushGatewayUrlProvider` at DI site | HIGH | `AppModules.kt` | ✅ Fixed |
| RC-08 | Carry QR payload fields to SetupConfig | MEDIUM | `SetupService.kt` | ✅ Fixed |

7 of 8 items fixed. RC-06 requires new RPC methods, polling logic, and a waiting UI — it should be planned as a separate feature.
