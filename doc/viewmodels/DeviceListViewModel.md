# DeviceListViewModel

> State management for DeviceListScreen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/viewmodels/DeviceListViewModel.kt`

## Overview

DeviceListViewModel manages the state for the device management screen, handling device loading, verification, and revocation.

## Class Definition

```kotlin
class DeviceListViewModel(
    private val deviceRepository: DeviceRepository
) : ViewModel()
```

---

## State Flows

### uiState
```kotlin
private val _uiState = MutableStateFlow<DeviceListUiState>(DeviceListUiState.Loading)
val uiState: StateFlow<DeviceListUiState> = _uiState.asStateFlow()
```

**Description:** Overall screen state.

**States:**
| State | Description |
|-------|-------------|
| `Loading` | Fetching devices |
| `Loaded` | Devices loaded successfully |
| `Error(message)` | Error occurred |

---

### devices
```kotlin
private val _devices = MutableStateFlow<List<Device>>(emptyList())
val devices: StateFlow<List<Device>> = _devices.asStateFlow()
```

**Description:** List of all user's devices.

---

### currentDevice
```kotlin
private val _currentDevice = MutableStateFlow<Device?>(null)
val currentDevice: StateFlow<Device?> = _currentDevice.asStateFlow()
```

**Description:** The current device the user is using.

---

### verificationState
```kotlin
private val _verificationState = MutableStateFlow<VerificationState>(VerificationState.Idle)
val verificationState: StateFlow<VerificationState> = _verificationState.asStateFlow()
```

**Description:** Current verification flow state.

**States:**
```kotlin
sealed class VerificationState {
    object Idle : VerificationState()
    data class InProgress(val device: Device, val emojis: List<String>) : VerificationState()
    data class Success(val deviceId: String) : VerificationState()
    data class Error(val message: String) : VerificationState()
}
```

---

## Actions

### loadDevices
```kotlin
fun loadDevices()
```

**Description:** Fetches all devices for the current user.

**Flow:**
1. Set loading state
2. Fetch devices from repository
3. Identify current device
4. Update devices list
5. Set loaded state

---

### startVerification
```kotlin
fun startVerification(deviceId: String)
```

**Description:** Initiates the verification process for a device.

**Flow:**
1. Find device in list
2. Generate verification emojis
3. Set verification state to InProgress
4. Wait for user confirmation

---

### confirmVerification
```kotlin
fun confirmVerification()
```

**Description:** Confirms the verification after user confirms emoji match.

**Flow:**
1. Get current verification state
2. Call repository to verify device
3. Update device trust level
4. Set success state
5. Clear verification state

---

### rejectVerification
```kotlin
fun rejectVerification()
```

**Description:** Cancels the verification process.

---

### revokeDevice
```kotlin
fun revokeDevice(deviceId: String)
```

**Description:** Revokes access for a device.

**Flow:**
1. Show confirmation dialog (in UI)
2. Call repository to revoke
3. Remove device from list
4. Show success message

---

### renameDevice
```kotlin
fun renameDevice(deviceId: String, newName: String)
```

**Description:** Updates the display name for a device.

---

## Data Models

### DeviceListUiState
```kotlin
sealed class DeviceListUiState {
    object Loading : DeviceListUiState()
    data class Loaded(
        val devices: List<Device>,
        val currentDevice: Device
    ) : DeviceListUiState()
    data class Error(val message: String) : DeviceListUiState()
}
```

### Device
```kotlin
data class Device(
    val id: String,
    val name: String,
    val platform: DevicePlatform,
    val trustLevel: TrustLevel,
    val lastActiveAt: Instant,
    val addedAt: Instant,
    val isCurrent: Boolean
)

enum class DevicePlatform {
    ANDROID, IOS, WEB_CHROME, WEB_FIREFOX,
    WEB_SAFARI, DESKTOP_WINDOWS, DESKTOP_MAC,
    DESKTOP_LINUX, UNKNOWN
}

enum class TrustLevel {
    VERIFIED, UNVERIFIED, BLOCKED
}
```

---

## Usage Example

```kotlin
@Composable
fun DeviceListScreen(
    onNavigateBack: () -> Unit,
    viewModel: DeviceListViewModel = viewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    val devices by viewModel.devices.collectAsState()
    val currentDevice by viewModel.currentDevice.collectAsState()
    val verificationState by viewModel.verificationState.collectAsState()

    when (uiState) {
        is DeviceListUiState.Loading -> {
            LoadingIndicator()
        }
        is DeviceListUiState.Loaded -> {
            DeviceListContent(
                devices = devices,
                currentDevice = currentDevice,
                onVerify = { viewModel.startVerification(it) },
                onRevoke = { viewModel.revokeDevice(it) }
            )
        }
        is DeviceListUiState.Error -> {
            ErrorState(
                message = (uiState as DeviceListUiState.Error).message,
                onRetry = { viewModel.loadDevices() }
            )
        }
    }

    // Verification dialog
    if (verificationState is VerificationState.InProgress) {
        EmojiVerificationDialog(
            emojis = (verificationState as VerificationState.InProgress).emojis,
            onConfirm = { viewModel.confirmVerification() },
            onReject = { viewModel.rejectVerification() }
        )
    }
}
```

---

## Testing

### Unit Tests
```kotlin
@Test
fun `loadDevices updates state with devices`() = runTest {
    // Given
    val mockDevices = listOf(
        Device(id = "1", name = "Phone", platform = ANDROID, trustLevel = VERIFIED, ...)
    )
    coEvery { deviceRepository.getDevices() } returns mockDevices

    // When
    val viewModel = DeviceListViewModel(deviceRepository)
    viewModel.loadDevices()

    // Then
    val state = viewModel.uiState.value
    assertTrue(state is DeviceListUiState.Loaded)
    assertEquals(1, (state as DeviceListUiState.Loaded).devices.size)
}

@Test
fun `revokeDevice removes device from list`() = runTest {
    // Given
    val viewModel = DeviceListViewModel(deviceRepository)
    viewModel.loadDevices()

    // When
    viewModel.revokeDevice("1")

    // Then
    val devices = viewModel.devices.value
    assertFalse(devices.any { it.id == "1" })
}
```

---

## Related Documentation

- [Device Management](../features/device-management.md) - Feature overview
- [DeviceListScreen](../screens/DeviceListScreen.md) - Device list screen
- [EmojiVerificationScreen](../screens/EmojiVerificationScreen.md) - Verification screen
