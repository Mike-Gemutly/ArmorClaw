package com.armorclaw.shared.platform.network

/**
 * Shared network utility functions
 *
 * Consolidates helpers that were previously duplicated across:
 * - SetupService.kt
 * - SetupViewModel.kt
 * - ConnectServerScreen.kt
 * - RpcModels.kt (BridgeConfig.Companion)
 */
object NetworkUtils {

    private val IPV4_REGEX = Regex(
        "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
    )

    /**
     * Check if the given string is a valid IPv4 address.
     *
     * Uses strict dotted-decimal regex only — no heuristics.
     * Returns false for blank strings, hostnames, and IPv6 addresses.
     */
    fun isIpAddress(host: String): Boolean {
        if (host.isBlank()) return false
        return IPV4_REGEX.matches(host)
    }
}
