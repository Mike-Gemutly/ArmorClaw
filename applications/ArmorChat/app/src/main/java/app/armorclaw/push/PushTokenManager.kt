package app.armorclaw.push

import android.content.Context
import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.tasks.await

/**
 * Manages FCM token lifecycle and registration with Matrix Push Gateway
 *
 * Updated (v4.5.0): Now uses native Matrix HTTP Pusher instead of legacy Bridge API.
 * This resolves the "Split-Brain" push notification issue (G-01).
 */
object PushTokenManager {

    private const val TAG = "PushTokenManager"
    private const val PREFS_NAME = "push_tokens"
    private const val KEY_FCM_TOKEN = "fcm_token"
    private const val KEY_REGISTERED = "token_registered"
    private const val KEY_LEGACY_MODE = "legacy_push_mode" // Migration flag

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
     * Register FCM token using native Matrix HTTP Pusher
     *
     * This replaces the legacy Bridge API push registration.
     * The token is registered with the Matrix homeserver which routes
     * push notifications through Sygnal.
     *
     * @param context Application context
     * @param repository Bridge repository for Matrix pusher registration
     * @param deviceDisplayName Human-readable device name for pusher
     * @param force Force re-registration even if already registered
     */
    suspend fun registerWithMatrixPusher(
        context: Context,
        repository: app.armorclaw.data.repository.BridgeRepository,
        deviceDisplayName: String = "Android Device",
        force: Boolean = false
    ): Boolean {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

        // Check if already registered
        if (!force && prefs.getBoolean(KEY_REGISTERED, false)) {
            Log.d(TAG, "Pusher already registered")
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

        // Register with Matrix pusher
        return try {
            val result = repository.registerPushToken(token, deviceDisplayName)

            if (result.isSuccess) {
                prefs.edit()
                    .putString(KEY_FCM_TOKEN, token)
                    .putBoolean(KEY_REGISTERED, true)
                    .putBoolean(KEY_LEGACY_MODE, false) // Mark as migrated
                    .apply()
                Log.i(TAG, "Matrix pusher registered successfully")
                true
            } else {
                Log.e(TAG, "Failed to register pusher: ${result.exceptionOrNull()?.message}")
                prefs.edit().putBoolean(KEY_REGISTERED, false).apply()
                false
            }
        } catch (e: Exception) {
            Log.e(TAG, "Exception during pusher registration", e)
            prefs.edit().putBoolean(KEY_REGISTERED, false).apply()
            false
        }
    }

    /**
     * Legacy method - Register FCM token with the Bridge Server
     *
     * @deprecated Use registerWithMatrixPusher instead.
     * This method is kept for migration compatibility.
     */
    @Deprecated(
        message = "Use registerWithMatrixPusher for native Matrix pusher support",
        replaceWith = ReplaceWith("registerWithMatrixPusher(context, repository, deviceDisplayName, force)")
    )
    suspend fun registerWithBridge(
        context: Context,
        repository: app.armorclaw.data.repository.BridgeRepository,
        force: Boolean = false
    ): Boolean {
        // Redirect to new method
        return registerWithMatrixPusher(context, repository, force = force)
    }

    /**
     * Unregister push token from Matrix (e.g., on logout)
     */
    suspend fun unregisterFromMatrix(
        context: Context,
        repository: app.armorclaw.data.repository.BridgeRepository
    ) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val token = prefs.getString(KEY_FCM_TOKEN, null)

        if (token != null) {
            try {
                repository.unregisterPushToken(token)
                Log.d(TAG, "Pusher unregistered from Matrix")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to unregister pusher", e)
            }
        }

        prefs.edit()
            .remove(KEY_FCM_TOKEN)
            .putBoolean(KEY_REGISTERED, false)
            .apply()
    }

    /**
     * Legacy method - Unregister push token from bridge
     *
     * @deprecated Use unregisterFromMatrix instead.
     */
    @Deprecated(
        message = "Use unregisterFromMatrix for native Matrix pusher support",
        replaceWith = ReplaceWith("unregisterFromMatrix(context, repository)")
    )
    suspend fun unregisterFromBridge(
        context: Context,
        repository: app.armorclaw.data.repository.BridgeRepository
    ) {
        unregisterFromMatrix(context, repository)
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

    /**
     * Check if using legacy push mode
     */
    fun isLegacyMode(context: Context): Boolean {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        return prefs.getBoolean(KEY_LEGACY_MODE, false)
    }

    /**
     * Mark migration complete
     */
    fun markMigrated(context: Context) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit().putBoolean(KEY_LEGACY_MODE, false).apply()
    }
}
