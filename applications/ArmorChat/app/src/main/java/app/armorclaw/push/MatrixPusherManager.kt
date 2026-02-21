package app.armorclaw.push

import android.content.Context
import android.util.Log
import app.armorclaw.crypto.VodozemacNative
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * Matrix Pusher Manager - Native HTTP Pusher Implementation
 *
 * Replaces legacy BridgeAdminClient push registration with standard Matrix pusher.
 * Uses the Matrix Rust SDK's HTTP Pusher API to register with Sygnal.
 *
 * Resolves: G-01 (Push Logic Conflict)
 */
class MatrixPusherManager(
    private val context: Context,
    private val homeserverUrl: String,
    private val accessToken: String,
    private val deviceId: String
) {
    companion object {
        private const val TAG = "MatrixPusherManager"

        // Push Gateway configuration
        const val PUSH_GATEWAY_URL = "https://push.armorclaw.app/_matrix/push/v1/notify"
        const val APP_ID = "app.armorclaw.armorchat"
        const val APP_DISPLAY_NAME = "ArmorChat"
        const val DEFAULT_LANG = "en"
    }

    /**
     * Register HTTP Pusher with Matrix homeserver
     *
     * This uses the standard Matrix pusher API instead of custom Bridge endpoints.
     * The homeserver will route push notifications through Sygnal.
     */
    suspend fun registerPusher(
        fcmToken: String,
        deviceDisplayName: String = "Android Device"
    ): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            Log.d(TAG, "Registering Matrix HTTP Pusher for device: $deviceId")

            val pusherPayload = buildPusherPayload(fcmToken, deviceDisplayName)

            // Use Matrix client API to set pusher
            val response = matrixSetPusher(pusherPayload)

            if (response.isSuccessful) {
                Log.i(TAG, "Matrix pusher registered successfully")
                savePusherState(fcmToken, true)
                Result.success(Unit)
            } else {
                val error = response.errorMessage ?: "Unknown error"
                Log.e(TAG, "Failed to register pusher: $error")
                Result.failure(PushException("Pusher registration failed: $error"))
            }
        } catch (e: Exception) {
            Log.e(TAG, "Exception during pusher registration", e)
            Result.failure(e)
        }
    }

    /**
     * Unregister HTTP Pusher from Matrix homeserver
     */
    suspend fun unregisterPusher(fcmToken: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            Log.d(TAG, "Unregistering Matrix HTTP Pusher")

            // Set pusher with empty pushkey to delete it
            val deletePayload = buildDeletePusherPayload(fcmToken)

            val response = matrixSetPusher(deletePayload)

            if (response.isSuccessful) {
                Log.i(TAG, "Matrix pusher unregistered successfully")
                savePusherState(fcmToken, false)
                Result.success(Unit)
            } else {
                Log.w(TAG, "Failed to unregister pusher: ${response.errorMessage}")
                // Still clear local state even if server fails
                savePusherState(fcmToken, false)
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Log.e(TAG, "Exception during pusher unregistration", e)
            // Clear local state anyway
            savePusherState(fcmToken, false)
            Result.failure(e)
        }
    }

    /**
     * Check if pusher is registered
     */
    fun isPusherRegistered(): Boolean {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return prefs.getBoolean(KEY_PUSHER_REGISTERED, false)
    }

    /**
     * Get registered FCM token
     */
    fun getRegisteredToken(): String? {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return prefs.getString(KEY_FCM_TOKEN, null)
    }

    /**
     * Build the Matrix pusher payload for registration
     */
    private fun buildPusherPayload(fcmToken: String, deviceDisplayName: String): JSONObject {
        return JSONObject().apply {
            put("pushkey", fcmToken)
            put("kind", "http")
            put("app_id", APP_ID)
            put("app_display_name", APP_DISPLAY_NAME)
            put("device_display_name", deviceDisplayName)
            put("profile_tag", generateProfileTag())
            put("lang", DEFAULT_LANG)
            put("data", JSONObject().apply {
                put("url", PUSH_GATEWAY_URL)
                put("format", "event_id_only")
                // Encrypted push support
                put("default_payload", JSONObject().apply {
                    put("aps", JSONObject().apply {
                        put("mutable-content", 1)
                        put("alert", JSONObject().apply {
                            put("loc-key", "NEW_MESSAGE")
                            put("loc-args", JSONObject())
                        })
                    })
                })
            })
            put("append", false) // Replace existing pusher
        }
    }

    /**
     * Build delete pusher payload (kind = null removes the pusher)
     */
    private fun buildDeletePusherPayload(fcmToken: String): JSONObject {
        return JSONObject().apply {
            put("pushkey", fcmToken)
            put("kind", null) // null kind deletes the pusher
            put("app_id", APP_ID)
        }
    }

    /**
     * Generate a profile tag for grouping pushers
     */
    private fun generateProfileTag(): String {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        var tag = prefs.getString(KEY_PROFILE_TAG, null)

        if (tag == null) {
            tag = "armorchat_${System.currentTimeMillis().toString(16)}"
            prefs.edit().putString(KEY_PROFILE_TAG, tag).apply()
        }

        return tag
    }

    /**
     * Call Matrix API to set pusher
     */
    private suspend fun matrixSetPusher(pusherPayload: JSONObject): PusherResponse {
        return try {
            val client = okhttp3.OkHttpClient.Builder()
                .connectTimeout(30, java.util.concurrent.TimeUnit.SECONDS)
                .readTimeout(30, java.util.concurrent.TimeUnit.SECONDS)
                .build()

            val url = "$homeserverUrl/_matrix/client/v3/pushers/set"

            val body = okhttp3.RequestBody.create(
                okhttp3.MediaType.parse("application/json"),
                pusherPayload.toString()
            )

            val request = okhttp3.Request.Builder()
                .url(url)
                .post(body)
                .addHeader("Authorization", "Bearer $accessToken")
                .addHeader("Content-Type", "application/json")
                .build()

            val response = client.newCall(request).execute()

            PusherResponse(
                isSuccessful = response.isSuccessful,
                errorMessage = if (!response.isSuccessful) {
                    response.body?.string() ?: "HTTP ${response.code}"
                } else null
            )
        } catch (e: Exception) {
            Log.e(TAG, "Network error during pusher API call", e)
            PusherResponse(
                isSuccessful = false,
                errorMessage = e.message ?: "Network error"
            )
        }
    }

    /**
     * Save pusher state locally
     */
    private fun savePusherState(fcmToken: String, registered: Boolean) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit()
            .putString(KEY_FCM_TOKEN, if (registered) fcmToken else null)
            .putBoolean(KEY_PUSHER_REGISTERED, registered)
            .apply()
    }

    /**
     * Response wrapper for pusher API
     */
    private data class PusherResponse(
        val isSuccessful: Boolean,
        val errorMessage: String? = null
    )

    /**
     * Exception for push-related errors
     */
    class PushException(message: String) : Exception(message)
}

// Constants
private const val PREFS_NAME = "matrix_pusher_prefs"
private const val KEY_PUSHER_REGISTERED = "pusher_registered"
private const val KEY_FCM_TOKEN = "fcm_token"
private const val KEY_PROFILE_TAG = "profile_tag"
