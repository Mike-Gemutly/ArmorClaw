package app.armorclaw.network

import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.suspendCancellableCoroutine
import java.net.InetAddress
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * Discovered bridge information
 */
data class DiscoveredBridge(
    val name: String,
    val host: String,
    val port: Int,
    val ips: List<InetAddress> = emptyList(),
    val version: String? = null,
    val mode: String? = null,
    val hardware: String? = null,
    val txt: Map<String, String> = emptyMap()
)

/**
 * Discovery client using Android NSD (Network Service Discovery)
 */
class BridgeDiscovery(private val nsdManager: NsdManager) {

    companion object {
        const val SERVICE_TYPE = "_armorclaw._tcp."
        const val DEFAULT_PORT = 8080
    }

    /**
     * Discover bridges on the local network
     */
    fun discover(): Flow<List<DiscoveredBridge>> = callbackFlow {
        val bridges = mutableListOf<DiscoveredBridge>()

        val discoveryListener = object : NsdManager.DiscoveryListener {
            override fun onDiscoveryStarted(regType: String) {
                // Discovery started
            }

            override fun onServiceFound(service: NsdServiceInfo) {
                if (service.serviceType == SERVICE_TYPE) {
                    // Resolve the service to get connection details
                    resolveService(service) { bridge ->
                        bridges.add(bridge)
                        trySend(bridges.toList())
                    }
                }
            }

            override fun onServiceLost(service: NsdServiceInfo) {
                // Remove from list
                bridges.removeAll { it.name == service.serviceName }
                trySend(bridges.toList())
            }

            override fun onDiscoveryStopped(serviceType: String) {
                // Discovery stopped
            }

            override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
                close(Exception("Discovery start failed: $errorCode"))
            }

            override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {
                // Stop failed, but not critical
            }
        }

        nsdManager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, discoveryListener)

        awaitClose {
            nsdManager.stopServiceDiscovery(discoveryListener)
        }
    }

    /**
     * Resolve a discovered service to get connection details
     */
    private fun resolveService(
        service: NsdServiceInfo,
        onResolved: (DiscoveredBridge) -> Unit
    ) {
        val resolveListener = object : NsdManager.ResolveListener {
            override fun onResolveFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {
                // Resolution failed, skip this service
            }

            override fun onServiceResolved(serviceInfo: NsdServiceInfo) {
                val bridge = DiscoveredBridge(
                    name = serviceInfo.serviceName,
                    host = serviceInfo.host?.hostAddress ?: "",
                    port = serviceInfo.port,
                    ips = listOfNotNull(serviceInfo.host),
                    txt = parseTxtRecords(serviceInfo.attributes)
                )
                onResolved(bridge)
            }
        }

        nsdManager.resolveService(service, resolveListener)
    }

    /**
     * Parse TXT records from NSD attributes
     */
    private fun parseTxtRecords(attributes: Map<String, ByteArray>): Map<String, String> {
        return attributes.mapValues { (_, value) ->
            try {
                String(value, Charsets.UTF_8)
            } catch (e: Exception) {
                ""
            }
        }
    }

    /**
     * Test connection to a specific bridge
     */
    suspend fun testConnection(bridge: DiscoveredBridge): Boolean {
        return suspendCancellableCoroutine { continuation ->
            try {
                val address = InetAddress.getByName(bridge.host)
                val reachable = address.isReachable(3000) // 3 second timeout
                continuation.resume(reachable)
            } catch (e: Exception) {
                continuation.resume(false)
            }
        }
    }

    /**
     * Validate manual connection details
     */
    fun validateManualConnection(host: String, port: String): ValidationResult {
        if (host.isBlank()) {
            return ValidationResult.Error("Host is required")
        }

        // Validate IP address
        val ipPattern = Regex("^(\\d{1,3}\\.){3}\\d{1,3}$")
        if (ipPattern.matches(host)) {
            val parts = host.split(".").map { it.toIntOrNull() ?: 0 }
            if (parts.any { it > 255 }) {
                return ValidationResult.Error("Invalid IP address")
            }
        } else {
            // Validate hostname
            val hostnamePattern = Regex(
                "^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"
            )
            if (!hostnamePattern.matches(host)) {
                return ValidationResult.Error("Invalid hostname")
            }
        }

        // Validate port
        val portNum = port.toIntOrNull()
        if (portNum == null || portNum < 1 || portNum > 65535) {
            return ValidationResult.Error("Port must be between 1 and 65535")
        }

        return ValidationResult.Valid(portNum)
    }

    sealed class ValidationResult {
        data class Valid(val port: Int) : ValidationResult()
        data class Error(val message: String) : ValidationResult()
    }
}

/**
 * Manual connection details
 */
data class ManualConnection(
    val host: String,
    val port: Int = DEFAULT_PORT
) {
    fun toDiscoveredBridge(): DiscoveredBridge {
        return DiscoveredBridge(
            name = "Manual: $host",
            host = host,
            port = port,
            txt = mapOf("manual" to "true")
        )
    }
}
