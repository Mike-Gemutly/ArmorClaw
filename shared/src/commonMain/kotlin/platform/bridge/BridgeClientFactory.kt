package com.armorclaw.shared.platform.bridge

import io.ktor.client.*

/**
 * Factory for creating BridgeRpcClient instances
 *
 * Use this factory to create clients with proper platform-specific configuration.
 */
expect object BridgeClientFactory {
    /**
     * Create a BridgeRpcClient with the given configuration
     *
     * On Android, this will use OkHttp with certificate pinning if enabled.
     * On other platforms, it will use Ktor's default HttpClient.
     *
     * @param config Bridge configuration
     * @return Configured BridgeRpcClient instance
     */
    fun createClient(config: BridgeConfig): BridgeRpcClient

    /**
     * Create a default HTTP client with platform-specific optimizations
     *
     * @param config Bridge configuration
     * @return Configured HttpClient
     */
    fun createHttpClient(config: BridgeConfig): HttpClient
}
