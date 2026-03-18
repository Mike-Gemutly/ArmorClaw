package com.armorclaw.shared.platform.bridge

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Debug/Development settings for QA and testing
 *
 * Access: Long-press version number in About screen 5 times
 *
 * These settings allow overriding bridge connection parameters for testing
 * against different server configurations without code changes.
 *
 * CTO Note (2026-02-23): This prevents blocking legitimate testing when
 * server port changes in future sprints.
 */
object DevSettings {

    private val _enabled = MutableStateFlow(false)
    val enabled: StateFlow<Boolean> = _enabled.asStateFlow()

    private val _overrides = MutableStateFlow(BridgeOverrides())
    val overrides: StateFlow<BridgeOverrides> = _overrides.asStateFlow()

    /**
     * Enable/disable dev mode
     */
    fun setEnabled(enabled: Boolean) {
        _enabled.value = enabled
        if (!enabled) {
            // Clear overrides when disabling
            _overrides.value = BridgeOverrides()
        }
    }

    /**
     * Toggle dev mode
     */
    fun toggle() {
        setEnabled(!_enabled.value)
    }

    /**
     * Update bridge URL override
     */
    fun setBridgeUrlOverride(url: String?) {
        _overrides.value = _overrides.value.copy(
            bridgeUrlOverride = url?.trim()?.takeIf { it.isNotEmpty() }
        )
    }

    /**
     * Update protocol override (http/https)
     */
    fun setProtocolOverride(protocol: String?) {
        _overrides.value = _overrides.value.copy(
            protocolOverride = protocol?.trim()?.takeIf { it == "http" || it == "https" }
        )
    }

    /**
     * Update RPC path override
     */
    fun setRpcPathOverride(path: String?) {
        _overrides.value = _overrides.value.copy(
            rpcPathOverride = path?.trim()?.takeIf { it.isNotEmpty() }
        )
    }

    /**
     * Update bridge port override
     */
    fun setPortOverride(port: Int?) {
        _overrides.value = _overrides.value.copy(
            portOverride = port
        )
    }

    /**
     * Clear all overrides
     */
    fun clearOverrides() {
        _overrides.value = BridgeOverrides()
    }

    /**
     * Apply overrides to a derived bridge URL
     *
     * @param derivedUrl The URL derived from homeserver
     * @return The overridden URL if overrides are active, otherwise the derived URL
     */
    fun applyOverrides(derivedUrl: String): String {
        val currentOverrides = _overrides.value
        if (!_enabled.value || !currentOverrides.hasAnyOverride) {
            return derivedUrl
        }

        var url = derivedUrl

        // Apply protocol override
        currentOverrides.protocolOverride?.let { protocol ->
            url = url.replace("https://", "$protocol://")
                     .replace("http://", "$protocol://")
        }

        // Apply port override
        currentOverrides.portOverride?.let { port ->
            // Replace existing port or add one
            val hostPart = url
                .removePrefix("https://")
                .removePrefix("http://")
                .split("/").first()

            val host = hostPart.split(":").first()
            val protocol = if (url.startsWith("https://")) "https://" else "http://"
            val path = url.removePrefix(protocol).removePrefix(hostPart)

            url = "$protocol$host:$port$path"
        }

        // Apply RPC path override
        currentOverrides.rpcPathOverride?.let { rpcPath ->
            // Remove trailing slash from URL and add RPC path
            val baseUrl = url.removeSuffix("/")
            val normalizedPath = if (rpcPath.startsWith("/")) rpcPath else "/$rpcPath"
            url = baseUrl + normalizedPath
        }

        // Apply full URL override (highest priority)
        currentOverrides.bridgeUrlOverride?.let { overrideUrl ->
            url = overrideUrl
        }

        return url
    }

    /**
     * Get effective bridge URL considering dev overrides
     * 
     * @param homeserver The Matrix homeserver URL
     * @param derivedUrl The bridge URL derived from homeserver (use SetupService.deriveBridgeUrl)
     * @return The overridden URL if overrides are active, otherwise the derived URL
     */
    fun getEffectiveBridgeUrl(homeserver: String, derivedUrl: String): String {
        return applyOverrides(derivedUrl)
    }
}

/**
 * Bridge connection overrides for testing
 */
data class BridgeOverrides(
    val bridgeUrlOverride: String? = null,
    val protocolOverride: String? = null,
    val rpcPathOverride: String? = null,
    val portOverride: Int? = null
) {
    val isEmpty: Boolean
        get() = bridgeUrlOverride == null &&
                protocolOverride == null &&
                rpcPathOverride == null &&
                portOverride == null

    val hasAnyOverride: Boolean
        get() = !isEmpty
}
