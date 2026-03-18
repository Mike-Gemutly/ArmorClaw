# ArmorChat Communication Fixes

> **Last Updated:** 2025-01-20  
> **Version:** 1.0

This document summarizes the fixes made to ArmorChat's communication with the ArmorClaw Bridge server.

## Problem Summary

The ArmorChat app had several communication issues with the Bridge server:

1. **RPC Method Name Mismatch** - Some RPC methods used camelCase while Bridge expects snake_case
2. **matrix.sync Not Available** - Bridge doesn't expose matrix.sync, requiring direct Matrix /sync
3. **WebSocket Stub** - Bridge WebSocket is not implemented, preventing real-time events
4. **WebRTC Method Mismatch** - ArmorChat expected different WebRTC methods than Bridge provides
5. **License Feature Check** - Bridge doesn't have license.checkFeature, need local caching
6. **Missing homeserverUrl** - Config didn't include direct Matrix server access

## Changes Made

### Step 1: Update BridgeConfig

**File:** `shared/src/commonMain/kotlin/platform/bridge/RpcModels.kt`

```kotlin
data class BridgeConfig(
    val baseUrl: String,
    val homeserverUrl: String,           // NEW: Matrix direct access
    val wsUrl: String? = null,
    val useDirectMatrixSync: Boolean = true  // NEW: Use Matrix /sync directly
)
```

### Step 2: Create MatrixSyncManager

**File:** `shared/src/commonMain/kotlin/platform/matrix/MatrixSyncManager.kt` (NEW)

A client-side Matrix /sync implementation that:
- Performs long-polling directly to Matrix homeserver
- Emits typed events (MessageReceived, TypingNotification, etc.)
- Provides connection state management
- Replaces WebSocket for real-time events

### Step 3: Create RealTimeEventStore

**File:** `shared/src/commonMain/kotlin/data/store/RealTimeEventStore.kt` (NEW)

A high-level event store that:
- Wraps MatrixSyncManager
- Provides filtered event flows by type
- Supports per-room subscriptions
- Emits typed events for UI consumption

### Step 4: Create CheckFeatureUseCase

**File:** `shared/src/commonMain/kotlin/domain/usecase/CheckFeatureUseCase.kt` (NEW)

A use case for feature availability checking that:
- Uses license.features RPC (available on Bridge)
- Caches results locally (5-minute TTL)
- Provides isAvailable() and isWithinLimit() helpers
- Handles stale cache gracefully on errors

### Step 5: Add Bridge-Compatible WebRTC Methods

**File:** `shared/src/commonMain/kotlin/platform/bridge/BridgeRpcClient.kt`

Added new methods matching Bridge API:
- `webrtcStart(roomId, callType)` → maps to `webrtc.start`
- `webrtcSendIceCandidate(sessionId, ...)` → maps to `webrtc.ice_candidate`
- `webrtcEnd(sessionId)` → maps to `webrtc.end`
- `webrtcList()` → maps to `webrtc.list`

### Step 6: Add WebRtcCallSession Model

**File:** `shared/src/commonMain/kotlin/platform/bridge/RpcModels.kt`

```kotlin
@Serializable
data class WebRtcCallSession(
    @SerialName("session_id") val sessionId: String,
    @SerialName("room_id") val roomId: String,
    @SerialName("call_type") val callType: String,
    @SerialName("started_at") val startedAt: Long,
    @SerialName("started_by") val startedBy: String,
    val status: String,
    ...
)
```

### Step 7: Update BridgeRepository

**File:** `shared/src/commonMain/kotlin/platform/bridge/BridgeRepository.kt`

Changes:
- Added `syncManager: MatrixSyncManager?` parameter
- Added `config: BridgeConfig` parameter
- Added `connectForEvents()` method that uses Matrix /sync when configured
- Updated `stopBridge()` to stop sync manager
- Added `getSyncEventFlow()` for direct sync events

### Step 8: Update RPC Method Names

All RPC methods now use snake_case in their Bridge method names:

| Kotlin Method | RPC Method |
|---------------|------------|
| `recoveryGeneratePhrase` | `recovery.generate_phrase` |
| `recoveryStorePhrase` | `recovery.store_phrase` |
| `recoveryIsDeviceValid` | `recovery.is_device_valid` |
| `pushRegister` | `push.register_token` |
| `pushUnregister` | `push.unregister_token` |
| `getErrors` | `get_errors` |
| `resolveError` | `resolve_error` |
| `webrtcStart` | `webrtc.start` |
| `webrtcEnd` | `webrtc.end` |
| `webrtcList` | `webrtc.list` |

## Communication Flow (After Fixes)

```
┌─────────────────────────────────────────────────────────────────┐
│  ArmorChat (Android App)                                        │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │ BridgeRpcClient │    │ MatrixSyncManager│                    │
│  │ (Admin Ops)     │    │ (Real-time)     │                    │
│  └────────┬────────┘    └────────┬────────┘                    │
│           │                      │                              │
└───────────┼──────────────────────┼──────────────────────────────┘
            │                      │
            ▼                      ▼
┌─────────────────────┐    ┌─────────────────────┐
│  Bridge RPC         │    │  Matrix Homeserver  │
│  (HTTP/JSON-RPC)    │    │  (HTTPS /sync)      │
│                     │    │                     │
│  • bridge.start     │    │  • /sync (long-poll)│
│  • webrtc.start     │    │  • Messages         │
│  • recovery.*       │    │  • Typing           │
│  • platform.*       │    │  • Presence         │
│  • license.*        │    │  • Calls            │
└─────────────────────┘    └─────────────────────┘
```

## Usage Examples

### Starting a Session with Real-Time Events

```kotlin
// Create components
val config = BridgeConfig.PRODUCTION
val rpcClient = BridgeRpcClientImpl(config)
val syncManager = MatrixSyncManager(config.homeserverUrl, httpClient)
val repository = BridgeRepository(rpcClient, wsClient, syncManager, config)

// Login
val loginResult = repository.matrixLogin(homeserver, username, password, deviceId)
val accessToken = loginResult.getOrThrow().accessToken

// Start receiving real-time events
repository.connectForEvents()

// Subscribe to messages
syncManager.events.collect { event ->
    when (event) {
        is MatrixSyncEvent.MessageReceived -> showMessage(event)
        is MatrixSyncEvent.TypingNotification -> showTyping(event)
        // ...
    }
}
```

### Checking Feature Availability

```kotlin
val checkFeature = CheckFeatureUseCase(bridgeClient)

// Check if Slack bridging is available
val result = checkFeature("platform_slack")
if (result.getOrThrow().available) {
    // Enable Slack integration
}

// Check with limit
val limitResult = checkFeature.isWithinLimit("platform_bridging", currentChannelCount)
if (limitResult.getOrThrow()) {
    // Add another channel
}
```

### WebRTC Calls

```kotlin
// Start a call
val session = bridgeClient.webrtcStart(roomId, "video").getOrThrow()

// Send ICE candidate
bridgeClient.webrtcSendIceCandidate(
    sessionId = session.sessionId,
    candidate = candidate.candidate,
    sdpMid = candidate.sdpMid,
    sdpMLineIndex = candidate.sdpMLineIndex
)

// End call
bridgeClient.webrtcEnd(session.sessionId)
```

## Testing

1. **Unit Tests**: Test RPC method name formatting
2. **Integration Tests**: Test MatrixSyncManager with mock server
3. **E2E Tests**: Test full login + sync flow

## Migration Notes

### For Existing Code

1. Replace `connectWebSocket()` with `connectForEvents()`
2. Use `getSyncEventFlow()` instead of `getEventFlow()` for Matrix events
3. Add `homeserverUrl` to BridgeConfig
4. Use `CheckFeatureUseCase` instead of `licenseCheckFeature` RPC

### Breaking Changes

- `BridgeRepository` constructor now accepts additional parameters
- `connectWebSocket()` is deprecated, use `connectForEvents()`
- WebRTC methods have new signatures matching Bridge API

## Future Work

1. **WebSocket Implementation**: When Bridge implements WebSocket, add fallback support
2. **E2EE**: Implement vodozemac for client-side encryption
3. **Background Sync**: Use WorkManager for background sync on Android
4. **Push Notifications**: Integrate FCM/APNs with Matrix push gateway
