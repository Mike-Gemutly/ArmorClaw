package app.armorclaw.config

import android.content.Context
import android.content.SharedPreferences
import android.os.Build
import android.util.Log
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

/**
 * Configuration Manager for ArmorChat
 *
 * Persists ServerConfig to encrypted storage using AndroidX EncryptedSharedPreferences.
 * Only non-sensitive fields are persisted (homeserver URL, device ID, expiresAt).
 * Auth tokens remain in-memory only (BridgeRepository) until v0.8.0 security review.
 *
 * Lifecycle:
 * - saveConfig(): called after successful QR scan / SignedConfigParser parse
 * - loadConfig(): called on app startup to restore previous session
 * - isConfigExpired(): checked before using cached config; triggers re-provisioning
 * - clearConfig(): called on logout or when config is invalidated
 */
class ConfigManager(
    context: Context
) {
    companion object {
        private const val TAG = "ConfigManager"
        private const val PREFS_NAME = "armorchat_server_config"
        private const val PREFS_VERSION = 1

        private const val KEY_MATRIX_HOMESERVER = "matrix_homeserver"
        private const val KEY_RPC_URL = "rpc_url"
        private const val KEY_WS_URL = "ws_url"
        private const val KEY_PUSH_GATEWAY = "push_gateway"
        private const val KEY_SERVER_NAME = "server_name"
        private const val KEY_REGION = "region"
        private const val KEY_CONFIG_SOURCE = "config_source"
        private const val KEY_EXPIRES_AT = "expires_at"
        private const val KEY_BRIDGE_PUBLIC_KEY = "bridge_public_key"
        private const val KEY_PREFS_VERSION = "prefs_version"

        @Volatile
        private var instance: ConfigManager? = null

        fun getInstance(context: Context): ConfigManager {
            return instance ?: synchronized(this) {
                instance ?: ConfigManager(context.applicationContext).also {
                    instance = it
                }
            }
        }
    }

    private val masterKey: MasterKey = MasterKey.Builder(context)
        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
        .build()

    private val prefs: SharedPreferences = try {
        EncryptedSharedPreferences.create(
            context,
            PREFS_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    } catch (e: Exception) {
        Log.e(TAG, "Failed to create EncryptedSharedPreferences, falling back to clear prefs", e)
        // Fallback: if encrypted prefs creation fails (e.g. corrupted master key on
        // older devices), use plain SharedPreferences so the app doesn't crash.
        // This should never happen in normal operation.
        context.getSharedPreferences(PREFS_NAME + "_fallback", Context.MODE_PRIVATE)
    }

    /**
     * Persist a ServerConfig after successful QR scan or SignedConfigParser parse.
     * Only non-sensitive fields are stored; auth tokens are intentionally excluded.
     */
    fun saveConfig(config: ServerConfig) {
        try {
            prefs.edit()
                .putString(KEY_MATRIX_HOMESERVER, config.matrixHomeserver)
                .putString(KEY_RPC_URL, config.rpcUrl)
                .putString(KEY_WS_URL, config.wsUrl)
                .putString(KEY_PUSH_GATEWAY, config.pushGateway)
                .putString(KEY_SERVER_NAME, config.serverName)
                .putString(KEY_REGION, config.region)
                .putString(KEY_CONFIG_SOURCE, config.configSource.name)
                .putLong(KEY_EXPIRES_AT, config.expiresAt)
                .putString(KEY_BRIDGE_PUBLIC_KEY, config.bridgePublicKey)
                .putInt(KEY_PREFS_VERSION, PREFS_VERSION)
                .apply()

            Log.i(TAG, "ServerConfig saved: server=${config.serverName}, source=${config.configSource}")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to save ServerConfig", e)
        }
    }

    /**
     * Load previously persisted ServerConfig.
     * Returns null if no config exists or if stored config is corrupted.
     * Callers should also check isConfigExpired() after loading.
     */
    fun loadConfig(): ServerConfig? {
        return try {
            if (!prefs.contains(KEY_MATRIX_HOMESERVER)) {
                Log.d(TAG, "No saved config found")
                return null
            }

            val version = prefs.getInt(KEY_PREFS_VERSION, 0)
            if (version != PREFS_VERSION) {
                Log.w(TAG, "Config version mismatch (stored=$version, current=$PREFS_VERSION), clearing")
                clearConfig()
                return null
            }

            val sourceName = prefs.getString(KEY_CONFIG_SOURCE, null)
            val source = try {
                sourceName?.let { ConfigSource.valueOf(it) } ?: ConfigSource.CACHED
            } catch (e: IllegalArgumentException) {
                ConfigSource.CACHED
            }

            val config = ServerConfig(
                matrixHomeserver = prefs.getString(KEY_MATRIX_HOMESERVER, "") ?: "",
                rpcUrl = prefs.getString(KEY_RPC_URL, "") ?: "",
                wsUrl = prefs.getString(KEY_WS_URL, "") ?: "",
                pushGateway = prefs.getString(KEY_PUSH_GATEWAY, "") ?: "",
                serverName = prefs.getString(KEY_SERVER_NAME, "") ?: "",
                region = prefs.getString(KEY_REGION, "unknown") ?: "unknown",
                configSource = source,
                expiresAt = prefs.getLong(KEY_EXPIRES_AT, Long.MAX_VALUE),
                bridgePublicKey = prefs.getString(KEY_BRIDGE_PUBLIC_KEY, null)
            )

            // Validate essential fields are present
            if (config.matrixHomeserver.isBlank()) {
                Log.w(TAG, "Saved config has empty homeserver, treating as missing")
                return null
            }

            Log.d(TAG, "Config loaded: server=${config.serverName}, source=${config.configSource}")
            config
        } catch (e: Exception) {
            Log.e(TAG, "Failed to load ServerConfig, clearing corrupted data", e)
            clearConfig()
            null
        }
    }

    /**
     * Clear all persisted config data.
     * Called on logout or when config is invalidated.
     */
    fun clearConfig() {
        try {
            prefs.edit().clear().apply()
            Log.i(TAG, "ServerConfig cleared")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to clear ServerConfig", e)
        }
    }

    /**
     * Check if the currently persisted config has expired.
     * Returns true if no config exists or if config has passed its expiresAt timestamp.
     * Callers should trigger re-provisioning when this returns true.
     */
    fun isConfigExpired(): Boolean {
        val config = loadConfig() ?: return true
        return config.isExpired()
    }
}
