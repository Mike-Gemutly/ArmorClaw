package com.armorclaw.armorterminal.viewmodel

import android.util.Base64
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.armorterminal.network.BridgeApi
import com.armorclaw.armorterminal.network.BridgeDiscovery
import com.armorclaw.armorterminal.network.DiscoveredBridge
import com.armorclaw.armorterminal.network.ManualConnection
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import java.security.KeyPairGenerator
import java.security.Signature
import java.util.UUID

/**
 * Pairing ViewModel
 *
 * Handles the device pairing flow:
 * 1. Bridge discovery (mDNS or manual)
 * 2. QR code parsing
 * 3. Certificate fingerprint verification
 * 4. Device registration
 * 5. Waiting for admin approval
 */
class PairingViewModel(
    private val bridgeDiscovery: BridgeDiscovery,
    private val sessionManager: SessionManager,
    private val cryptoService: CryptoService
) : ViewModel() {

    private val _state = MutableStateFlow(PairingState())
    val state: StateFlow<PairingState> = _state.asStateFlow()

    private var approvalCheckJob: Job? = null
    private var bridgeApi: BridgeApi? = null

    //region Public API

    /**
     * Start bridge discovery
     */
    fun startDiscovery() {
        viewModelScope.launch {
            _state.update { it.copy(isDiscovering = true, error = null) }

            bridgeDiscovery.discover().collect { bridges ->
                _state.update { it.copy(
                    discoveredBridges = bridges,
                    isDiscovering = false
                )}
            }
        }
    }

    /**
     * Select a discovered bridge
     */
    fun selectBridge(bridge: DiscoveredBridge) {
        _state.update { it.copy(
            selectedBridge = bridge,
            step = PairingStep.CERTIFICATE_VERIFY
        )}

        // Fetch certificate fingerprint
        fetchCertificateFingerprint(bridge)
    }

    /**
     * Set manual connection
     */
    fun setManualConnection(host: String, port: Int) {
        val connection = ManualConnection(host, port)
        val validation = bridgeDiscovery.validateManualConnection(host, port.toString())

        when (validation) {
            is BridgeDiscovery.ValidationResult.Valid -> {
                val bridge = connection.toDiscoveredBridge()
                _state.update { it.copy(
                    selectedBridge = bridge,
                    step = PairingStep.CERTIFICATE_VERIFY,
                    error = null
                )}
                fetchCertificateFingerprint(bridge)
            }
            is BridgeDiscovery.ValidationResult.Error -> {
                _state.update { it.copy(error = validation.message) }
            }
        }
    }

    /**
     * Verify certificate fingerprint
     */
    fun verifyCertificate(accepted: Boolean) {
        if (!accepted) {
            _state.update { it.copy(
                step = PairingStep.BRIDGE_SELECT,
                selectedBridge = null,
                certificateFingerprint = null,
                error = "Certificate rejected"
            )}
            return
        }

        _state.update { it.copy(step = PairingStep.QR_SCAN)}
    }

    /**
     * Process scanned QR code
     */
    fun processQRCode(qrData: String) {
        viewModelScope.launch {
            _state.update { it.copy(isProcessingQR = true, error = null) }

            try {
                val pairingInfo = parsePairingQR(qrData)
                _state.update { it.copy(
                    pairingInfo = pairingInfo,
                    isProcessingQR = false
                )}

                // Start device registration
                registerDevice(pairingInfo)
            } catch (e: Exception) {
                _state.update { it.copy(
                    isProcessingQR = false,
                    error = "Invalid QR code: ${e.message}"
                )}
            }
        }
    }

    /**
     * Skip QR scan and use direct pairing token
     */
    fun setPairingToken(token: String) {
        val pairingInfo = PairingInfo(
            token = token,
            server = _state.value.selectedBridge?.getHttpsUrl() ?: "",
            userId = "",
            expiresAt = System.currentTimeMillis() + 300000 // 5 minutes
        )
        _state.update { it.copy(pairingInfo = pairingInfo) }
        registerDevice(pairingInfo)
    }

    /**
     * Cancel pairing
     */
    fun cancelPairing() {
        approvalCheckJob?.cancel()
        approvalCheckJob = null

        _state.update { PairingState() }
    }

    /**
     * Retry from error state
     */
    fun retry() {
        val currentState = _state.value

        when {
            currentState.selectedBridge != null -> {
                _state.update { it.copy(
                    step = PairingStep.QR_SCAN,
                    error = null
                )}
            }
            else -> {
                _state.update { PairingState() }
            }
        }
    }

    //endregion

    //region Private Implementation

    /**
     * Fetch certificate fingerprint from bridge
     */
    private fun fetchCertificateFingerprint(bridge: DiscoveredBridge) {
        viewModelScope.launch {
            val api = BridgeApi(bridge.getHttpsUrl())
            val result = api.getCertificateFingerprint()

            result.onSuccess { fingerprint ->
                _state.update { it.copy(
                    certificateFingerprint = fingerprint,
                    bridgeApi = api
                )}
                this@PairingViewModel.bridgeApi = api
            }.onFailure { error ->
                _state.update { it.copy(
                    error = "Failed to fetch certificate: ${error.message}",
                    step = PairingStep.BRIDGE_SELECT
                )}
            }
        }
    }

    /**
     * Register device with bridge
     */
    private fun registerDevice(pairingInfo: PairingInfo) {
        viewModelScope.launch {
            _state.update { it.copy(step = PairingStep.DEVICE_REGISTRATION) }

            val api = bridgeApi ?: return@launch

            // Generate device keypair
            val keypair = cryptoService.generateKeyPair()
            val publicKeyBase64 = Base64.encodeToString(
                keypair.public.encoded,
                Base64.NO_WRAP
            )

            val deviceName = "${android.os.Build.MANUFACTURER} ${android.os.Build.MODEL}"
            val deviceType = "android"

            val result = api.registerDevice(
                pairingToken = pairingInfo.token,
                deviceName = deviceName,
                deviceType = deviceType,
                publicKey = publicKeyBase64
            )

            result.onSuccess { registration ->
                // Store session
                sessionManager.saveSession(
                    deviceId = registration.device_id,
                    sessionToken = registration.session_token,
                    privateKey = keypair.private.encoded
                )

                _state.update { it.copy(
                    deviceId = registration.device_id,
                    sessionToken = registration.session_token,
                    step = PairingStep.AWAITING_APPROVAL
                )}

                // Start waiting for approval
                if (registration.next_step == "awaiting_approval") {
                    waitForApproval(registration.device_id, registration.session_token)
                } else {
                    // Already approved
                    _state.update { it.copy(step = PairingStep.COMPLETE) }
                }
            }.onFailure { error ->
                _state.update { it.copy(
                    step = PairingStep.QR_SCAN,
                    error = "Registration failed: ${error.message}"
                )}
            }
        }
    }

    /**
     * Wait for admin approval via WebSocket
     */
    private fun waitForApproval(deviceId: String, sessionToken: String) {
        approvalCheckJob?.cancel()

        approvalCheckJob = viewModelScope.launch {
            val bridge = _state.value.selectedBridge ?: return@launch

            // Poll for approval status
            var attempts = 0
            val maxAttempts = 60 // 5 minutes at 5 second intervals

            while (isActive && attempts < maxAttempts) {
                delay(5000)
                attempts++

                val api = bridgeApi ?: continue
                val result = api.waitForApproval(deviceId, sessionToken, timeout = 5)

                result.onSuccess { status ->
                    when (status.status) {
                        "approved" -> {
                            _state.update { it.copy(
                                step = PairingStep.COMPLETE,
                                approvalMessage = "Device approved!"
                            )}
                            return@launch
                        }
                        "rejected" -> {
                            _state.update { it.copy(
                                step = PairingStep.ERROR,
                                error = status.rejection_reason ?: "Device rejected"
                            )}
                            return@launch
                        }
                        "expired" -> {
                            _state.update { it.copy(
                                step = PairingStep.ERROR,
                                error = "Approval request expired"
                            )}
                            return@launch
                        }
                    }
                }
            }

            // Timeout
            if (isActive) {
                _state.update { it.copy(
                    step = PairingStep.ERROR,
                    error = "Approval timeout - please try again"
                )}
            }
        }
    }

    /**
     * Parse QR code data
     */
    private fun parsePairingQR(qrData: String): PairingInfo {
        // Try armorclaw:// format
        if (qrData.startsWith("armorclaw://pair/")) {
            return parseArmorClawFormat(qrData)
        }

        // Try JSON format
        if (qrData.startsWith("{")) {
            return parseJsonFormat(qrData)
        }

        // Try base64 encoded JSON
        try {
            val decoded = Base64.decode(qrData, Base64.DEFAULT).toString(Charsets.UTF_8)
            if (decoded.startsWith("{")) {
                return parseJsonFormat(decoded)
            }
        } catch (e: Exception) {
            // Not base64
        }

        throw IllegalArgumentException("Unknown QR code format")
    }

    private fun parseArmorClampFormat(qrData: String): PairingInfo {
        // armorclaw://pair/{base64(json)} or armorclaw://pair?token=...&server=...
        val content = qrData.removePrefix("armorclaw://pair/")

        return if (content.contains("?")) {
            // Query format
            val params = content.substringAfter("?")
                .split("&")
                .associate {
                    val (key, value) = it.split("=")
                    key to value
                }

            PairingInfo(
                token = params["token"] ?: "",
                server = params["server"] ?: "",
                userId = params["user"] ?: "",
                expiresAt = params["expires"]?.toLongOrNull() ?: (System.currentTimeMillis() + 300000)
            )
        } else {
            // Base64 format
            val decoded = Base64.decode(content, Base64.URL_SAFE).toString(Charsets.UTF_8)
            parseJsonFormat(decoded)
        }
    }

    private fun parseJsonFormat(json: String): PairingInfo {
        // Simple JSON parsing (in production, use kotlinx.serialization)
        val token = json.extractValue("token")
        val server = json.extractValue("server")
        val userId = json.extractValue("user_id") ?: json.extractValue("userId") ?: ""
        val expires = json.extractValue("expires")?.toLongOrNull() ?: (System.currentTimeMillis() + 300000)

        return PairingInfo(
            token = token ?: throw IllegalArgumentException("Missing token"),
            server = server ?: "",
            userId = userId,
            expiresAt = expires
        )
    }

    private fun String.extractValue(key: String): String? {
        val regex = """"$key"\s*:\s*"([^"]+)"""".toRegex()
        return regex.find(this)?.groupValues?.get(1)
    }

    //endregion
}

/**
 * Pairing state
 */
data class PairingState(
    val step: PairingStep = PairingStep.BRIDGE_SELECT,
    val isDiscovering: Boolean = false,
    val discoveredBridges: List<DiscoveredBridge> = emptyList(),
    val selectedBridge: DiscoveredBridge? = null,
    val certificateFingerprint: String? = null,
    val isProcessingQR: Boolean = false,
    val pairingInfo: PairingInfo? = null,
    val deviceId: String? = null,
    val sessionToken: String? = null,
    val approvalMessage: String? = null,
    val error: String? = null,
    internal val bridgeApi: BridgeApi? = null
)

/**
 * Pairing step
 */
enum class PairingStep {
    BRIDGE_SELECT,       // Select discovered bridge or enter manually
    CERTIFICATE_VERIFY,  // Verify bridge certificate
    QR_SCAN,             // Scan QR code for pairing token
    DEVICE_REGISTRATION, // Registering device with bridge
    AWAITING_APPROVAL,   // Waiting for admin approval
    COMPLETE,            // Pairing complete
    ERROR                // Error occurred
}

/**
 * Parsed QR pairing information
 */
data class PairingInfo(
    val token: String,
    val server: String,
    val userId: String,
    val expiresAt: Long
) {
    fun isExpired(): Boolean = System.currentTimeMillis() > expiresAt
}

/**
 * Session manager interface
 */
interface SessionManager {
    fun saveSession(deviceId: String, sessionToken: String, privateKey: ByteArray)
    fun getSession(): SavedSession?
    fun clearSession()
}

data class SavedSession(
    val deviceId: String,
    val sessionToken: String
)

/**
 * Crypto service interface
 */
interface CryptoService {
    fun generateKeyPair(): KeyPairData
    fun sign(data: ByteArray, privateKey: ByteArray): ByteArray
    fun verify(data: ByteArray, signature: ByteArray, publicKey: ByteArray): Boolean
}

data class KeyPairData(
    val public: PublicKeyData,
    val private: PrivateKeyData
)

data class PublicKeyData(val encoded: ByteArray)
data class PrivateKeyData(val encoded: ByteArray)
