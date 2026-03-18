package com.armorclaw.app.viewmodels

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.domain.model.DeviceInfo
import com.armorclaw.shared.domain.model.SyncState
import com.armorclaw.shared.domain.model.TrustLevel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch

/**
 * UI State for sync status
 */
data class SyncStatusUiState(
    val syncState: SyncState = SyncState.Idle,
    val lastSyncTime: Long? = null,
    val queuedMessageCount: Int = 0,
    val isRetrying: Boolean = false,
    val retryCount: Int = 0,
    val maxRetries: Int = 3,
    val errorMessage: String? = null
)

/**
 * ViewModel for managing sync status and connection state
 */
class SyncStatusViewModel(private val context: Context) : ViewModel() {

    private val _uiState = MutableStateFlow(SyncStatusUiState())
    val uiState: StateFlow<SyncStatusUiState> = _uiState.asStateFlow()

    private var syncJob: Job? = null
    private var retryJob: Job? = null

    private val connectivityManager = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    private val _isOnline = MutableStateFlow(true)
    val isOnline: StateFlow<Boolean> = _isOnline.asStateFlow()

    private val networkCallback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            _isOnline.value = true
        }

        override fun onLost(network: Network) {
            _isOnline.value = false
        }

        override fun onCapabilitiesChanged(network: Network, networkCapabilities: NetworkCapabilities) {
            _isOnline.value = networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                              networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
        }
    }

    init {
        val networkRequest = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        connectivityManager.registerNetworkCallback(networkRequest, networkCallback)
    }

    val syncState: StateFlow<SyncState> = _uiState
        .map { uiState: SyncStatusUiState -> uiState.syncState }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), SyncState.Idle)

    val lastSyncTime: StateFlow<Long?> = _uiState
        .map { uiState: SyncStatusUiState -> uiState.lastSyncTime }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), null)

    fun updateSyncState(state: SyncState) {
        _uiState.update { current ->
            when (state) {
                is SyncState.Success -> current.copy(
                    syncState = state,
                    lastSyncTime = System.currentTimeMillis(),
                    queuedMessageCount = 0,
                    errorMessage = null,
                    isRetrying = false,
                    retryCount = 0
                )
                is SyncState.Error -> current.copy(
                    syncState = state,
                    errorMessage = state.message,
                    isRetrying = false
                )
                else -> current.copy(syncState = state)
            }
        }
    }

    fun updateQueuedMessageCount(count: Int) {
        _uiState.update { it.copy(queuedMessageCount = count) }
    }

    fun updateLastSyncTime(timestamp: Long) {
        _uiState.update { it.copy(lastSyncTime = timestamp) }
    }

    fun sync() {
        syncJob?.cancel()
        syncJob = viewModelScope.launch {
            _uiState.update { it.copy(syncState = SyncState.Syncing, isRetrying = false) }
            delay(2000L)
            _uiState.update {
                it.copy(
                    syncState = SyncState.Success(messagesSent = 0, messagesReceived = 0),
                    lastSyncTime = System.currentTimeMillis(),
                    queuedMessageCount = 0,
                    errorMessage = null,
                    retryCount = 0
                )
            }
        }
    }

    fun cancelSync() {
        syncJob?.cancel()
        syncJob = null
        _uiState.update { it.copy(syncState = SyncState.Idle, isRetrying = false) }
    }

    fun retryWithBackoff() {
        val currentState = _uiState.value
        if (currentState.retryCount >= currentState.maxRetries) {
            _uiState.update {
                it.copy(
                    errorMessage = "Max retries exceeded. Please try again later.",
                    isRetrying = false
                )
            }
            return
        }

        retryJob?.cancel()
        retryJob = viewModelScope.launch {
            _uiState.update { it.copy(isRetrying = true) }
            val delayMs: Long = (1000L * (1 shl currentState.retryCount)).coerceAtMost(30000L)
            delay(delayMs)
            _uiState.update { it.copy(retryCount = it.retryCount + 1) }
            sync()
        }
    }

    fun clearError() {
        _uiState.update {
            it.copy(
                syncState = SyncState.Idle,
                errorMessage = null,
                isRetrying = false
            )
        }
    }

    fun setOffline() {
        _uiState.update { it.copy(syncState = SyncState.Offline) }
    }

    fun setError(message: String, isRecoverable: Boolean = true) {
        _uiState.update {
            it.copy(
                syncState = SyncState.Error(message, isRecoverable),
                errorMessage = message
            )
        }
    }

    override fun onCleared() {
        super.onCleared()
        connectivityManager.unregisterNetworkCallback(networkCallback)
        syncJob?.cancel()
        retryJob?.cancel()
    }
}

/**
 * UI State for device list
 */
data class DeviceListUiState(
    val devices: List<DeviceInfo> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
    val currentDeviceId: String? = null,
    val unverifiedCount: Int = 0
)

/**
 * ViewModel for managing device list
 */
class DeviceListViewModel : ViewModel() {

    private val _uiState = MutableStateFlow(DeviceListUiState())
    val uiState: StateFlow<DeviceListUiState> = _uiState.asStateFlow()

    val unverifiedDevices: StateFlow<List<DeviceInfo>> = _uiState
        .map { state: DeviceListUiState -> state.devices }
        .map { devices: List<DeviceInfo> ->
            devices.filter { device: DeviceInfo ->
                !device.trustLevel.isTrusted() && !device.isCurrentDevice
            }
        }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())

    val trustedDevices: StateFlow<List<DeviceInfo>> = _uiState
        .map { state: DeviceListUiState -> state.devices }
        .map { devices: List<DeviceInfo> ->
            devices.filter { device: DeviceInfo -> device.trustLevel.isTrusted() }
        }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5000), emptyList())

    fun loadDevices() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            delay(1000L)

            val sampleDevices: List<DeviceInfo> = listOf(
                DeviceInfo(
                    deviceId = "DEVICE_001",
                    displayName = "This Device",
                    userId = "user@example.com",
                    trustLevel = TrustLevel.VERIFIED_IN_PERSON,
                    isCurrentDevice = true,
                    lastSeenTimestamp = System.currentTimeMillis()
                ),
                DeviceInfo(
                    deviceId = "DEVICE_002",
                    displayName = "iPhone 14 Pro",
                    userId = "user@example.com",
                    trustLevel = TrustLevel.CROSS_SIGNED,
                    lastSeenTimestamp = System.currentTimeMillis() - 3600000
                ),
                DeviceInfo(
                    deviceId = "DEVICE_003",
                    displayName = "Desktop App",
                    userId = "user@example.com",
                    trustLevel = TrustLevel.UNVERIFIED,
                    lastSeenTimestamp = System.currentTimeMillis() - 86400000
                )
            )

            val unverifiedCount = sampleDevices.count { device ->
                !device.trustLevel.isTrusted() && !device.isCurrentDevice
            }

            _uiState.update {
                it.copy(
                    devices = sampleDevices,
                    isLoading = false,
                    currentDeviceId = "DEVICE_001",
                    unverifiedCount = unverifiedCount
                )
            }
        }
    }

    fun verifyDevice(deviceId: String) {
        viewModelScope.launch {
            _uiState.update { state: DeviceListUiState ->
                val updatedDevices: List<DeviceInfo> = state.devices.map { device: DeviceInfo ->
                    if (device.deviceId == deviceId) {
                        device.copy(trustLevel = TrustLevel.CROSS_SIGNED)
                    } else {
                        device
                    }
                }
                val newUnverifiedCount = updatedDevices.count { device: DeviceInfo ->
                    !device.trustLevel.isTrusted() && !device.isCurrentDevice
                }
                state.copy(devices = updatedDevices, unverifiedCount = newUnverifiedCount)
            }
        }
    }

    fun removeDevice(deviceId: String) {
        viewModelScope.launch {
            _uiState.update { state: DeviceListUiState ->
                val updatedDevices: List<DeviceInfo> = state.devices.filter { device: DeviceInfo ->
                    device.deviceId != deviceId
                }
                val newUnverifiedCount = updatedDevices.count { device: DeviceInfo ->
                    !device.trustLevel.isTrusted() && !device.isCurrentDevice
                }
                state.copy(devices = updatedDevices, unverifiedCount = newUnverifiedCount)
            }
        }
    }

    fun refresh() {
        loadDevices()
    }
}
