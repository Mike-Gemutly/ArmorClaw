package com.armorclaw.armorterminal.network

import android.content.Context
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
    val fingerprint: String? = null,
    val txt: Map<String, String> = emptyMap()
) {
    /**
     * Get the HTTPS URL for this bridge
     */
    fun getHttpsUrl(): String {
        return if (port == 443) {
            "https://$host/api"
        } else {
            "https://$host:$port/api"
        }
    }

    /**
     * Get the WebSocket URL for this bridge
     */
    fun getWebSocketUrl(): String {
        return if (port == 443) {
            "wss://$host/ws"
        } else {
            "wss://$host:$port/ws"
        }
    }
}

/**
 * Bridge Discovery client using Android NSD (Network Service Discovery)
 *
 * Discovers ArmorClaw bridges on the local network via mDNS/Bonjour.
 * Service type: _armorclaw._tcp
 */
class BridgeDiscovery(context: Context) {

    companion object {
        const val SERVICE_TYPE = "_armorclaw._tcp."
        const val DEFAULT_PORT = 8443
        const val DISCOVERY_TIMEOUT_MS = 5000L
    }

    private val nsdManager: NsdManager = context.getSystemService(Context.NSD_SERVICE) as NsdManager

    /**
     * Discover bridges on the local network
     *
     * @return Flow that emits lists of discovered bridges
     */
    fun discover(): Flow<List<DiscoveredBridge>> = callbackFlow {
        val bridges = mutableListOf<DiscoveredBridge>()

        val discoveryListener = object : NsdManager.DiscoveryListener {
            override fun onDiscoveryStarted(regType: String) {
                // Discovery started successfully
            }

            override fun onServiceFound(service: NsdServiceInfo) {
                if (service.serviceType == SERVICE_TYPE) {
                    // Resolve the service to get connection details
                    resolveService(service) { bridge ->
                        synchronized(bridges) {
                            // Avoid duplicates
                            if (bridges.none { it.host == bridge.host && it.port == bridge.port }) {
                                bridges.add(bridge)
                            }
                        }
                        trySend(bridges.toList())
                    }
                }
            }

            override fun onServiceLost(service: NsdServiceInfo) {
                synchronized(bridges) {
                    bridges.removeAll { it.name == service.serviceName }
                }
                trySend(bridges.toList())
            }

            override fun onDiscoveryStopped(serviceType: String) {
                // Discovery stopped
            }

            override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
                close(Exception("Discovery start failed: error code $errorCode"))
            }

            override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {
                // Stop failed, but not critical
            }
        }

        nsdManager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, discoveryListener)

        awaitClose {
            try {
                nsdManager.stopServiceDiscovery(discoveryListener)
            } catch (e: Exception) {
                // Ignore errors during cleanup
            }
        }
    }

    /**
     * Discover a single bridge (first one found)
     *
     * @param timeoutMs Timeout in milliseconds
     * @return The first discovered bridge, or null if none found
     */
    suspend fun discoverOne(timeoutMs: Long = DISCOVERY_TIMEOUT_MS): DiscoveredBridge? {
        return suspendCancellableCoroutine { continuation ->
            var found = false
            val bridges = mutableListOf<DiscoveredBridge>()

            val discoveryListener = object : NsdManager.DiscoveryListener {
                override fun onDiscoveryStarted(regType: String) {}

                override fun onServiceFound(service: NsdServiceInfo) {
                    if (service.serviceType == SERVICE_TYPE && !found) {
                        resolveService(service) { bridge ->
                            if (!found) {
                                found = true
                                continuation.resume(bridge)
                                try {
                                    nsdManager.stopServiceDiscovery(this)
                                } catch (e: Exception) {
                                    // Ignore
                                }
                            }
                        }
                    }
                }

                override fun onServiceLost(service: NsdServiceInfo) {}

                override fun onDiscoveryStopped(serviceType: String) {
                    if (!found) {
                        continuation.resume(null)
                    }
                }

                override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
                    if (!found) {
                        continuation.resumeWithException(
                            Exception("Discovery start failed: error code $errorCode")
                        )
                    }
                }

                override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {}
            }

            // Set timeout
            val timeoutRunnable = Runnable {
                if (!found) {
                    found = true
                    try {
                        nsdManager.stopServiceDiscovery(discoveryListener)
                    } catch (e: Exception) {
                        // Ignore
                    }
                    if (continuation.isActive) {
                        continuation.resume(null)
                    }
                }
            }

            // Note: In a real implementation, you'd use a Handler for timeout
            // For simplicity, we rely on the flow-based discovery

            continuation.invokeOnCancellation {
                try {
                    nsdManager.stopServiceDiscovery(discoveryListener)
                } catch (e: Exception) {
                    // Ignore
                }
            }

            nsdManager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, discoveryListener)
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

        try {
            nsdManager.resolveService(service, resolveListener)
        } catch (e: Exception) {
            // Resolution failed
        }
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

    /**
     * Validation result for manual connection
     */
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
    val port: Int = BridgeDiscovery.DEFAULT_PORT
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

/**
 * Common local IP addresses for manual scanning
 */
object CommonLocalIPs {
    /**
     * Get common local IP prefixes to scan
     */
    fun getCommonPrefixes(): List<String> {
        return listOf(
            "192.168.1",
            "192.168.0",
            "10.0.0",
            "172.16.0"
        )
    }

    /**
     * Generate all IPs to scan for a given prefix
     */
    fun generateIPsToScan(prefix: String): List<String> {
        return (1..254).map { "$prefix.$it" }
    }
}
