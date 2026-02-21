package com.armorclaw.armorterminal.config

import android.content.Context
import android.util.Log
import org.json.JSONObject

/**
 * Trust-On-First-Use (TOFU) Store for Bridge Identity
 *
 * Stores known bridge identities (public keys) to detect MITM attacks.
 * When a user first provisions a device, the bridge's identity is stored.
 * On subsequent connections, any change in identity triggers a warning.
 *
 * Security Model:
 * - First connection: Trust the bridge identity
 * - Subsequent connections: Verify identity matches stored value
 * - Identity change: Alert user, require explicit confirmation
 */
object BridgeTrustStore {
    private const val TAG = "BridgeTrustStore"
    private const val PREFS_NAME = "armorclaw_bridge_trust"
    private const val KEY_KNOWN_BRIDGES = "known_bridges"
    private const val KEY_PROVISIONING_SECRETS = "provisioning_secrets"

    /**
     * Save a bridge identity (public key) for future verification
     *
     * @param context Android context
     * @param bridgeId Unique identifier (server_name or public key hash)
     * @param publicKey The bridge's HMAC signing key (hex-encoded)
     * @param serverName Human-readable server name
     */
    fun saveBridgeIdentity(
        context: Context,
        bridgeId: String,
        publicKey: String,
        serverName: String
    ) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)

        // Add or update bridge
        knownBridges.put(bridgeId, JSONObject().apply {
            put("public_key", publicKey)
            put("server_name", serverName)
            put("trusted_at", System.currentTimeMillis())
            put("last_seen", System.currentTimeMillis())
        })

        prefs.edit()
            .putString(KEY_KNOWN_BRIDGES, knownBridges.toString())
            .apply()

        Log.i(TAG, "Saved bridge identity: $bridgeId")
    }

    /**
     * Get the stored public key for a bridge
     *
     * @param context Android context
     * @param bridgeId Bridge identifier
     * @return Public key hex string, or null if unknown
     */
    fun getBridgePublicKey(context: Context, bridgeId: String): String? {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)

        if (!knownBridges.has(bridgeId)) {
            return null
        }

        val bridgeData = knownBridges.getJSONObject(bridgeId)

        // Update last seen
        bridgeData.put("last_seen", System.currentTimeMillis())
        prefs.edit()
            .putString(KEY_KNOWN_BRIDGES, knownBridges.toString())
            .apply()

        return bridgeData.getString("public_key")
    }

    /**
     * Check if a bridge is known
     */
    fun isBridgeKnown(context: Context, bridgeId: String): Boolean {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)
        return knownBridges.has(bridgeId)
    }

    /**
     * Get all known bridges
     */
    fun getKnownBridges(context: Context): List<KnownBridge> {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)

        return (knownBridges.keys() as? Iterator<String>)
            ?.asSequence()
            ?.map { bridgeId ->
                val data = knownBridges.getJSONObject(bridgeId)
                KnownBridge(
                    bridgeId = bridgeId,
                    publicKey = data.getString("public_key"),
                    serverName = data.getString("server_name"),
                    trustedAt = data.getLong("trusted_at"),
                    lastSeen = data.getLong("last_seen")
                )
            }
            ?.toList() ?: emptyList()
    }

    /**
     * Remove a bridge from the trust store
     */
    fun removeBridge(context: Context, bridgeId: String) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)

        knownBridges.remove(bridgeId)

        prefs.edit()
            .putString(KEY_KNOWN_BRIDGES, knownBridges.toString())
            .apply()

        Log.i(TAG, "Removed bridge: $bridgeId")
    }

    /**
     * Clear all trusted bridges (use with caution!)
     */
    fun clearAll(context: Context) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit().clear().apply()
        Log.w(TAG, "Cleared all trusted bridges")
    }

    /**
     * Store a provisioning secret received from a bridge
     * This is used for verifying subsequent configurations
     */
    fun storeProvisioningSecret(
        context: Context,
        bridgeId: String,
        secret: String
    ) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val secrets = getProvisioningSecretsJson(prefs)

        secrets.put(bridgeId, secret)

        prefs.edit()
            .putString(KEY_PROVISIONING_SECRETS, secrets.toString())
            .apply()
    }

    /**
     * Get a stored provisioning secret
     */
    fun getProvisioningSecret(context: Context, bridgeId: String): String? {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val secrets = getProvisioningSecretsJson(prefs)
        return if (secrets.has(bridgeId)) secrets.getString(bridgeId) else null
    }

    // Helper to get known bridges as JSONObject
    private fun getKnownBridgesJson(prefs: android.content.SharedPreferences): JSONObject {
        val jsonStr = prefs.getString(KEY_KNOWN_BRIDGES, "{}") ?: "{}"
        return try {
            JSONObject(jsonStr)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse known bridges", e)
            JSONObject()
        }
    }

    // Helper to get provisioning secrets as JSONObject
    private fun getProvisioningSecretsJson(prefs: android.content.SharedPreferences): JSONObject {
        val jsonStr = prefs.getString(KEY_PROVISIONING_SECRETS, "{}") ?: "{}"
        return try {
            JSONObject(jsonStr)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse provisioning secrets", e)
            JSONObject()
        }
    }
}

/**
 * Represents a known/trusted bridge
 */
data class KnownBridge(
    val bridgeId: String,
    val publicKey: String,
    val serverName: String,
    val trustedAt: Long,
    val lastSeen: Long
) {
    val trustedAtFormatted: String
        get() = java.text.DateFormat.getDateTimeInstance()
            .format(java.util.Date(trustedAt))

    val lastSeenFormatted: String
        get() = java.text.DateFormat.getDateTimeInstance()
            .format(java.util.Date(lastSeen))
}
