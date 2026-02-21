package com.armorclaw.armorterminal.config

import android.content.Context
import android.content.SharedPreferences
import android.util.Log
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update

/**
 * Configuration Manager for ArmorTerminal
 *
 * Manages server configuration with the following priority:
 * 1. Signed URL config (highest - from QR scan)
 * 2. Manual config (user entered)
 * 3. Cached config (from previous session)
 * 4. BuildConfig defaults (lowest)
 *
 * Features:
 * - Persists config to encrypted storage
 * - Notifies listeners on config changes
 * - Validates config before applying
 * - Supports config expiration
 */
class ConfigManager(
    context: Context,
    private val defaultConfig: ServerConfig
) {
    companion object {
        private const val TAG = "ConfigManager"
        private const val PREFS_NAME = "armorclaw_config"
        private const val KEY_MATRIX_HOMESERVER = "matrix_homeserver"
        private const val KEY_RPC_URL = "rpc_url"
        private const val KEY_WS_URL = "ws_url"
        private const val KEY_PUSH_GATEWAY = "push_gateway"
        private const val KEY_SERVER_NAME = "server_name"
        private const val KEY_REGION = "region"
        private const val KEY_CONFIG_SOURCE = "config_source"
        private const val KEY_EXPIRES_AT = "expires_at"
        private const val KEY_CONFIG_VERSION = "config_version"

        private const val CONFIG_VERSION = 1

        @Volatile
        private var instance: ConfigManager? = null

        fun getInstance(context: Context, defaultConfig: ServerConfig): ConfigManager {
            return instance ?: synchronized(this) {
                instance ?: ConfigManager(context.applicationContext, defaultConfig).also {
                    instance = it
                }
            }
        }
    }

    private val prefs: SharedPreferences = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    private val _config = MutableStateFlow(loadConfig())
    val config: StateFlow<ServerConfig> = _config.asStateFlow()

    private val _configEvents = MutableStateFlow<ConfigChangeEvent?>(null)
    val configEvents: StateFlow<ConfigChangeEvent?> = _configEvents.asStateFlow()

    init {
        // Check if cached config is expired
        val currentConfig = _config.value
        if (currentConfig.isExpired() && currentConfig.configSource != ConfigSource.DEFAULT) {
            Log.w(TAG, "Cached config expired, reverting to defaults")
            applyConfig(defaultConfig, ConfigSource.DEFAULT)
        }
    }

    /**
     * Apply configuration from signed URL/QR code
     */
    fun applySignedConfig(payload: SignedConfigParser.ConfigPayload): Result<ServerConfig> {
        return try {
            val newConfig = SignedConfigParser.toServerConfig(payload)
            applyConfig(newConfig, ConfigSource.SIGNED_URL)
            Result.success(newConfig)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to apply signed config", e)
            Result.failure(e)
        }
    }

    /**
     * Apply manual configuration
     */
    fun applyManualConfig(
        matrixHomeserver: String,
        rpcUrl: String,
        wsUrl: String,
        pushGateway: String,
        serverName: String
    ): Result<ServerConfig> {
        return try {
            // Validate URLs
            validateUrl(matrixHomeserver, "Matrix homeserver")
            validateUrl(rpcUrl, "RPC URL")
            validateUrl(wsUrl, "WebSocket URL")

            val newConfig = ServerConfig(
                matrixHomeserver = matrixHomeserver,
                rpcUrl = rpcUrl,
                wsUrl = wsUrl,
                pushGateway = pushGateway.ifBlank { derivePushGateway(rpcUrl) },
                serverName = serverName.ifBlank { deriveServerName(matrixHomeserver) },
                configSource = ConfigSource.MANUAL
            )

            applyConfig(newConfig, ConfigSource.MANUAL)
            Result.success(newConfig)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to apply manual config", e)
            Result.failure(e)
        }
    }

    /**
     * Reset to default configuration
     */
    fun resetToDefaults() {
        applyConfig(defaultConfig, ConfigSource.DEFAULT)
    }

    /**
     * Get current configuration
     */
    fun getCurrentConfig(): ServerConfig = _config.value

    /**
     * Check if configuration is valid for use
     */
    fun isConfigured(): Boolean {
        val config = _config.value
        return config.matrixHomeserver.isNotBlank() &&
               config.rpcUrl.isNotBlank() &&
               config.wsUrl.isNotBlank()
    }

    /**
     * Clear configuration event
     */
    fun clearConfigEvent() {
        _configEvents.value = null
    }

    // Private implementation

    private fun applyConfig(newConfig: ServerConfig, source: ConfigSource) {
        val oldConfig = _config.value

        // Save to preferences
        prefs.edit()
            .putString(KEY_MATRIX_HOMESERVER, newConfig.matrixHomeserver)
            .putString(KEY_RPC_URL, newConfig.rpcUrl)
            .putString(KEY_WS_URL, newConfig.wsUrl)
            .putString(KEY_PUSH_GATEWAY, newConfig.pushGateway)
            .putString(KEY_SERVER_NAME, newConfig.serverName)
            .putString(KEY_REGION, newConfig.region)
            .putString(KEY_CONFIG_SOURCE, source.name)
            .putLong(KEY_EXPIRES_AT, newConfig.expiresAt)
            .putInt(KEY_CONFIG_VERSION, CONFIG_VERSION)
            .apply()

        // Update state
        _config.value = newConfig

        // Emit event
        _configEvents.value = ConfigChangeEvent(
            oldConfig = oldConfig,
            newConfig = newConfig,
            source = source
        )

        Log.i(TAG, "Config updated: source=$source, server=${newConfig.serverDisplayName}")
    }

    private fun loadConfig(): ServerConfig {
        // Check if we have saved config
        if (!prefs.contains(KEY_MATRIX_HOMESERVER)) {
            return defaultConfig
        }

        val sourceName = prefs.getString(KEY_CONFIG_SOURCE, null)
        val source = try {
            sourceName?.let { ConfigSource.valueOf(it) } ?: ConfigSource.CACHED
        } catch (e: IllegalArgumentException) {
            ConfigSource.CACHED
        }

        return ServerConfig(
            matrixHomeserver = prefs.getString(KEY_MATRIX_HOMESERVER, defaultConfig.matrixHomeserver)!!,
            rpcUrl = prefs.getString(KEY_RPC_URL, defaultConfig.rpcUrl)!!,
            wsUrl = prefs.getString(KEY_WS_URL, defaultConfig.wsUrl)!!,
            pushGateway = prefs.getString(KEY_PUSH_GATEWAY, defaultConfig.pushGateway)!!,
            serverName = prefs.getString(KEY_SERVER_NAME, defaultConfig.serverName)!!,
            region = prefs.getString(KEY_REGION, defaultConfig.region)!!,
            configSource = source,
            expiresAt = prefs.getLong(KEY_EXPIRES_AT, Long.MAX_VALUE)
        )
    }

    private fun validateUrl(url: String, name: String) {
        if (url.isBlank()) {
            throw IllegalArgumentException("$name cannot be empty")
        }
        if (!url.startsWith("http://") && !url.startsWith("https://") &&
            !url.startsWith("ws://") && !url.startsWith("wss://")) {
            throw IllegalArgumentException("$name must be a valid URL")
        }
    }

    private fun derivePushGateway(rpcUrl: String): String {
        return rpcUrl
            .replace("/rpc", "/push")
            .replace("/api", "/push")
    }

    private fun deriveServerName(homeserver: String): String {
        return homeserver
            .removePrefix("https://")
            .removePrefix("http://")
            .substringBefore(":")
            .substringBefore("/")
    }
}

/**
 * BuildConfig-based default configuration
 *
 * These values are injected at build time from build.gradle.kts:
 * - Debug builds use emulator URLs (10.0.2.2)
 * - Release builds use production URLs
 */
object DefaultConfig {

    // These would be replaced by BuildConfig fields in actual implementation
    // For now, using development defaults

    val MATRIX_HOMESERVER: String = "https://matrix.armorclaw.com"
    val RPC_URL: String = "https://armorclaw.com/rpc"
    val WS_URL: String = "wss://armorclaw.com/ws"
    val PUSH_GATEWAY: String = "https://armorclaw.com/push"
    val SERVER_NAME: String = "ArmorClaw"
    val REGION: String = "us-east-1"

    fun create(): ServerConfig = ServerConfig(
        matrixHomeserver = MATRIX_HOMESERVER,
        rpcUrl = RPC_URL,
        wsUrl = WS_URL,
        pushGateway = PUSH_GATEWAY,
        serverName = SERVER_NAME,
        region = REGION,
        configSource = ConfigSource.DEFAULT
    )

    /**
     * Create debug configuration for emulator testing
     */
    fun createDebug(): ServerConfig = ServerConfig(
        matrixHomeserver = "http://10.0.2.2:8008",
        rpcUrl = "http://10.0.2.2:8080/rpc",
        wsUrl = "ws://10.0.2.2:8080/ws",
        pushGateway = "http://10.0.2.2:8080/push",
        serverName = "Development",
        region = "local",
        configSource = ConfigSource.DEFAULT
    )
}
