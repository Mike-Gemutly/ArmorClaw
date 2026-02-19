package app.armorclaw.push

import android.content.Context
import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.tasks.await

/**
 * Manages FCM token lifecycle and registration with Bridge
 */
object PushTokenManager {

    private const val TAG = "PushTokenManager"
    private const val PREFS_NAME = "push_tokens"
    private const val KEY_FCM_TOKEN = "fcm_token"
    private const val KEY_REGISTERED = "token_registered"

    /**
     * Get the current FCM token, fetching if necessary
     */
    suspend fun getToken(): String? {
        return try {
            FirebaseMessaging.getInstance().token.await()
        } catch (e: Exception) {
            Log.e(TAG, "Failed to get FCM token", e)
            null
        }
    }

    /**
     * Register FCM token with the Bridge Server
     *
     * @param context Application context
     * @param repository Bridge repository for API calls
     * @param force Force re-registration even if already registered
     */
    suspend fun registerWithBridge(
        context: Context,
        repository: app.armorclaw.data.repository.BridgeRepository,
        force: Boolean = false
    ): Boolean {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

        // Check if already registered
        if (!force && prefs.getBoolean(KEY_REGISTERED, false)) {
            Log.d(TAG, "Token already registered with bridge")
            return true
        }

        // Get current token
        val token = getToken()
        if (token == null) {
            Log.w(TAG, "No FCM token available")
            return false
        }

        // Check if token changed
        val storedToken = prefs.getString(KEY_FCM_TOKEN, null)
        if (!force && token == storedToken) {
            Log.d(TAG, "Token unchanged, skipping registration")
            return true
        }

        // Register with bridge
        return try {
            repository.registerPushToken(token)
            prefs.edit()
                .putString(KEY_FCM_TOKEN, token)
                .putBoolean(KEY_REGISTERED, true)
                .apply()
            Log.d(TAG, "Token registered with bridge successfully")
            true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to register token with bridge", e)
            // Mark as not registered so we retry next time
            prefs.edit().putBoolean(KEY_REGISTERED, false).apply()
            false
        }
    }

    /**
     * Unregister push token from bridge (e.g., on logout)
     */
    suspend fun unregisterFromBridge(
        context: Context,
        repository: app.armorclaw.data.repository.BridgeRepository
    ) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val token = prefs.getString(KEY_FCM_TOKEN, null)

        if (token != null) {
            try {
                repository.unregisterPushToken(token)
                Log.d(TAG, "Token unregistered from bridge")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to unregister token", e)
            }
        }

        prefs.edit()
            .remove(KEY_FCM_TOKEN)
            .putBoolean(KEY_REGISTERED, false)
            .apply()
    }

    /**
     * Force refresh the FCM token
     */
    suspend fun refreshToken(): String? {
        return try {
            FirebaseMessaging.getInstance().deleteToken().await()
            FirebaseMessaging.getInstance().token.await()
        } catch (e: Exception) {
            Log.e(TAG, "Failed to refresh FCM token", e)
            null
        }
    }
}
