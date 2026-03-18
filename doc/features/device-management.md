# Device Management Feature

> Trusted device management and verification
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/settings/`

## Overview

Device management allows users to view and manage all devices logged into their account, verify new devices for end-to-end encryption, and revoke access from compromised devices.

## Feature Components

### DeviceListScreen
**Location:** `settings/DeviceListScreen.kt`

Main screen displaying all connected devices.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `DeviceListScreen()` | Main device list | `onNavigateBack`, `onVerifyDevice` |
| `DeviceListItem()` | Individual device card | `device`, `isCurrent`, `onVerify`, `onRevoke` |
| `CurrentDeviceSection()` | Highlight current device | `device` |
| `OtherDevicesSection()` | Other logged-in devices | `devices` |
| `TrustBadge()` | Device trust indicator | `trustLevel` |
| `AddDeviceButton()` | Add new device | `onClick` |

#### Screen Layout
```
┌────────────────────────────────────┐
│  ← Devices                         │
├────────────────────────────────────┤
│  THIS DEVICE                       │
│  ┌──────────────────────────────┐  │
│  │ 📱 Samsung Galaxy S23        │  │
│  │    ✅ Verified               │  │
│  │    Last active: Now          │  │
│  │    Added: Jan 15, 2026       │  │
│  └──────────────────────────────┘  │
├────────────────────────────────────┤
│  OTHER DEVICES (2)                 │
│  ┌──────────────────────────────┐  │
│  │ 💻 Chrome on Windows         │  │
│  │    ⚠️ Unverified    [Verify] │  │
│  │    Last active: 2 hours ago  │  │
│  │    [Revoke Access]           │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ 📱 iPhone 15                 │  │
│  │    ✅ Verified               │  │
│  │    Last active: Yesterday    │  │
│  │    [Revoke Access]           │  │
│  └──────────────────────────────┘  │
├────────────────────────────────────┤
│  [   Add New Device   ]            │
└────────────────────────────────────┘
```

---

### EmojiVerificationScreen
**Location:** `settings/EmojiVerificationScreen.kt`

Device verification using emoji comparison.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `EmojiVerificationScreen()` | Main verification screen | `deviceName`, `onConfirm`, `onCancel` |
| `EmojiGrid()` | Display verification emojis | `emojis` |
| `VerificationInstructions()` | How to verify | - |
| `ConfirmButtons()` | Confirm/Reject buttons | `onConfirm`, `onReject` |

#### Verification Layout
```
┌────────────────────────────────────┐
│  ← Verify Device                   │
├────────────────────────────────────┤
│                                    │
│  Verify your new device:           │
│  Samsung Galaxy S23                │
│                                    │
│  Compare these emojis on both      │
│  devices:                          │
│                                    │
│  ┌────┐ ┌────┐ ┌────┐ ┌────┐      │
│  │ 🦁 │ │ 🚀 │ │ 🎸 │ │ 🌻 │      │
│  └────┘ └────┘ └────┘ └────┘      │
│                                    │
│  ┌────┐ ┌────┐ ┌────┐ ┌────┐      │
│  │ 🐧 │ │ 🌈 │ │ ⚽ │ │ 🎂 │      │
│  └────┘ └────┘ └────┘ └────┘      │
│                                    │
│  Do the emojis match?              │
│                                    │
│  [  They match  ]  [ They don't ]  │
│                                    │
└────────────────────────────────────┘
```

---

### DeviceListItem Component
**Location:** `settings/components/DeviceListItem.kt`

Individual device list item with actions.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `DeviceListItem()` | Device card | `device`, `isCurrent`, `onVerify`, `onRevoke` |
| `DeviceIcon()` | Platform icon | `platform` |
| `DeviceStatusBadge()` | Trust status | `trustLevel` |
| `DeviceActions()` | Action buttons | `canVerify`, `canRevoke` |

---

### TrustBadge Component
**Location:** `settings/components/TrustBadge.kt`

Visual indicator for device trust level.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `TrustBadge()` | Trust indicator | `trustLevel`, `modifier` |

#### Trust Levels

| Level | Icon | Color | Description |
|-------|------|-------|-------------|
| VERIFIED | ✅ | Green | Device verified |
| UNVERIFIED | ⚠️ | Yellow | Pending verification |
| BLOCKED | ❌ | Red | Device blocked |

---

## Data Models

### Device
```kotlin
data class Device(
    val id: String,
    val userId: String,
    val name: String,
    val platform: DevicePlatform,
    val trustLevel: TrustLevel,
    val lastActiveAt: Instant,
    val addedAt: Instant,
    val publicKey: String?,
    val isCurrent: Boolean
)

enum class DevicePlatform {
    ANDROID,
    IOS,
    WEB_CHROME,
    WEB_FIREFOX,
    WEB_SAFARI,
    DESKTOP_WINDOWS,
    DESKTOP_MAC,
    DESKTOP_LINUX,
    UNKNOWN
}

enum class TrustLevel {
    VERIFIED,
    UNVERIFIED,
    BLOCKED
}
```

### VerificationData
```kotlin
data class VerificationData(
    val deviceId: String,
    val emojis: List<String>,
    val expiresAt: Instant,
    val verificationCode: String
)
```

---

## State Management

### DeviceListState
```kotlin
data class DeviceListState(
    val devices: List<Device>,
    val currentDevice: Device?,
    val isLoading: Boolean,
    val isRevoking: Boolean,
    val error: String?,
    val pendingVerification: Device?
)
```

### DeviceListActions
| Action | Description |
|--------|-------------|
| `loadDevices()` | Fetch all devices |
| `startVerification(deviceId)` | Begin emoji verification |
| `confirmVerification()` | Complete verification |
| `rejectVerification()` | Cancel verification |
| `revokeDevice(deviceId)` | Remove device access |
| `renameDevice(deviceId, name)` | Update device name |

---

## Verification Process

### New Device Verification Flow
1. User logs in on new device
2. Device appears as "Unverified"
3. User initiates verification on trusted device
4. Emoji grid generated and displayed on both devices
5. User compares emojis
6. If match: Device becomes verified
7. If no match: Device remains unverified (potential MITM)

### Emoji Verification
- 7-8 random emojis from standardized set
- Generated from device public keys
- Cannot be forged without private keys
- 5-minute expiration time

---

## Security Features

### Device Revocation
- Instantly removes device access
- Deletes encryption keys for that device
- Sends notification to affected device
- Cannot be undone

### Key Management
- Each device has unique key pair
- Keys stored in AndroidKeyStore
- Cross-signing between verified devices
- Key rotation on compromise

---

## Bridge Verification Banner (NEW 2026-02-24)

### BridgeVerificationBanner
**Location:** `screens/room/RoomDetailsScreen.kt`

A warning banner shown in Room Details when the ArmorClaw bridge device is unverified.

#### Layout
```
┌────────────────────────────────────┐
│  ⚠️ Bridge Device Unverified       │
│                                    │
│  The ArmorClaw bridge device for   │
│  this room has not been verified.  │
│  Messages may not be secure.       │
│                                    │
│  [  Verify Bridge Device  ]        │
└────────────────────────────────────┘
```

#### Behavior
- Visible when `bridgeDevice.trustLevel == UNVERIFIED`
- "Verify Bridge Device" button navigates to `BRIDGE_VERIFICATION` route
- Route resolves to `EmojiVerificationScreen` with bridge device ID
- Banner hidden once verification completes

---

## Related Documentation

- [Encryption](encryption.md) - Encryption implementation
- [Biometric Auth](biometric-auth.md) - Biometric authentication
- [Security Settings](settings.md#security-settings) - Security preferences
