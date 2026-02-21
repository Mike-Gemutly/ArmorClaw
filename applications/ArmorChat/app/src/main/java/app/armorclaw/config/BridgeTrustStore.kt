package app.armorclaw.config

import android.content.Context
import android.util.Log
import org.json.JSONObject

/**
 * Trust-On-First-Use (TOFU) Store for Bridge Identity
 *
 * Stores known bridge identities (public keys) to detect MITM attacks.
 * When a user first provisions a device, the bridge's identity is stored.
 * On subsequent connections, any change in identity triggers a warning.
 */
object BridgeTrustStore {
    private const val TAG = "BridgeTrustStore"
    private const val PREFS_NAME = "armorclaw_bridge_trust"
    private const val KEY_KNOWN_BRIDGES = "known_bridges"
    private const val KEY_PROVISIONING_SECRETS = "provisioning_secrets"

    fun saveBridgeIdentity(
        context: Context,
        bridgeId: String,
        publicKey: String,
        serverName: String
    ) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)

        knownBridges.put(bridgeId, JSONObject().apply {
            put("public_key", publicKey)
            put("server_name", serverName)
            put("trusted_at", System.currentTimeMillis())
            put("last_seen", System.currentTimeMillis())
        })

        prefs.edit().putString(KEY_KNOWN_BRIDGES, knownBridges.toString()).apply()
        Log.i(TAG, "Saved bridge identity: $bridgeId")
    }

    fun getBridgePublicKey(context: Context, bridgeId: String): String? {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)

        if (!knownBridges.has(bridgeId)) return null

        val bridgeData = knownBridges.getJSONObject(bridgeId)
        bridgeData.put("last_seen", System.currentTimeMillis())
        prefs.edit().putString(KEY_KNOWN_BRIDGES, knownBridges.toString()).apply()

        return bridgeData.getString("public_key")
    }

    fun isBridgeKnown(context: Context, bridgeId: String): Boolean {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return getKnownBridgesJson(prefs).has(bridgeId)
    }

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

    fun removeBridge(context: Context, bridgeId: String) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val knownBridges = getKnownBridgesJson(prefs)
        knownBridges.remove(bridgeId)
        prefs.edit().putString(KEY_KNOWN_BRIDGES, knownBridges.toString()).apply()
        Log.i(TAG, "Removed bridge: $bridgeId")
    }

    fun clearAll(context: Context) {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE).edit().clear().apply()
        Log.w(TAG, "Cleared all trusted bridges")
    }

    fun storeProvisioningSecret(context: Context, bridgeId: String, secret: String) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val secrets = getProvisioningSecretsJson(prefs)
        secrets.put(bridgeId, secret)
        prefs.edit().putString(KEY_PROVISIONING_SECRETS, secrets.toString()).apply()
    }

    fun getProvisioningSecret(context: Context, bridgeId: String): String? {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val secrets = getProvisioningSecretsJson(prefs)
        return if (secrets.has(bridgeId)) secrets.getString(bridgeId) else null
    }

    private fun getKnownBridgesJson(prefs: android.content.SharedPreferences): JSONObject {
        val jsonStr = prefs.getString(KEY_KNOWN_BRIDGES, "{}") ?: "{}"
        return try { JSONObject(jsonStr) } catch (e: Exception) { JSONObject() }
    }

    private fun getProvisioningSecretsJson(prefs: android.content.SharedPreferences): JSONObject {
        val jsonStr = prefs.getString(KEY_PROVISIONING_SECRETS, "{}") ?: "{}"
        return try { JSONObject(jsonStr) } catch (e: Exception) { JSONObject() }
    }
}

data class KnownBridge(
    val bridgeId: String,
    val publicKey: String,
    val serverName: String,
    val trustedAt: Long,
    val lastSeen: Long
) {
    val trustedAtFormatted: String
        get() = java.text.DateFormat.getDateTimeInstance().format(java.util.Date(trustedAt))

    val lastSeenFormatted: String
        get() = java.text.DateFormat.getDateTimeInstance().format(java.util.Date(lastSeen))
}
