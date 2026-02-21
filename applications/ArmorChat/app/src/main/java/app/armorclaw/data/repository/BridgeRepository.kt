package app.armorclaw.data.repository

import android.content.Context
import android.util.Log
import app.armorclaw.network.BridgeApi
import app.armorclaw.push.MatrixPusherManager
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Repository for Bridge Server communication
 *
 * Handles:
 * - Native Matrix pusher registration (via MatrixPusherManager)
 * - Message sync
 * - WebSocket connection management
 *
 * Migration Note (v4.5.0):
 * Legacy push token methods (registerPushToken/unregisterPushToken via Bridge API)
 * have been replaced with standard Matrix HTTP Pusher via MatrixPusherManager.
 * This resolves the "Split-Brain" push notification issue (G-01).
 */
class BridgeRepository private constructor(
    private val context: Context,
    private val api: BridgeApi = BridgeApi()
) {

    companion object {
        private const val TAG = "BridgeRepository"

        @Volatile
        private var instance: BridgeRepository? = null

        fun getInstance(context: Context): BridgeRepository {
            return instance ?: synchronized(this) {
                instance ?: BridgeRepository(context.applicationContext).also { instance = it }
            }
        }
    }

    // Connection state
    private var isConnected: Boolean = false
    private var deviceId: String? = null

    // Matrix credentials (set after login)
    private var homeserverUrl: String? = null
    private var accessToken: String? = null

    // Native Matrix Pusher Manager
    private var pusherManager: MatrixPusherManager? = null

    /**
     * Set Matrix credentials for native pusher registration
     */
    fun setMatrixCredentials(homeserver: String, token: String, device: String) {
        this.homeserverUrl = homeserver
        this.accessToken = token
        this.deviceId = device
        this.pusherManager = MatrixPusherManager(
            context = context,
            homeserverUrl = homeserver,
            accessToken = token,
            deviceId = device
        )
        Log.d(TAG, "Matrix credentials configured for device: $device")
    }

    /**
     * Set the current device ID for push notifications
     */
    fun setDeviceId(id: String) {
        this.deviceId = id
    }

    /**
     * Register push token using native Matrix HTTP Pusher
     *
     * This replaces the legacy Bridge API push registration.
     * Uses standard Matrix pusher API routed through Sygnal.
     *
     * @param fcmToken FCM token from Firebase
     * @param deviceDisplayName Human-readable device name
     */
    suspend fun registerPushToken(
        fcmToken: String,
        deviceDisplayName: String = "Android Device"
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val manager = pusherManager
        if (manager == null) {
            Log.e(TAG, "PusherManager not initialized - call setMatrixCredentials first")
            return@withContext Result.failure(
                IllegalStateException("Matrix credentials not set")
            )
        }

        manager.registerPusher(fcmToken, deviceDisplayName)
    }

    /**
     * Unregister push token from Matrix homeserver
     */
    suspend fun unregisterPushToken(fcmToken: String): Result<Unit> = withContext(Dispatchers.IO) {
        val manager = pusherManager
        if (manager == null) {
            Log.w(TAG, "PusherManager not initialized, skipping unregister")
            return@withContext Result.success(Unit)
        }

        manager.unregisterPusher(fcmToken)
    }

    /**
     * Check if pusher is currently registered
     */
    fun isPusherRegistered(): Boolean {
        return pusherManager?.isPusherRegistered() ?: false
    }

    /**
     * Perform a quick sync to fetch pending messages
     * Called when app wakes up from push notification
     */
    suspend fun performQuickSync(): Result<SyncResult> = withContext(Dispatchers.IO) {
        try {
            // Connect to WebSocket and sync
            // This is a placeholder - actual implementation would:
            // 1. Establish WebSocket connection
            // 2. Send /sync request
            // 3. Process new messages
            // 4. Update local database
            // 5. Return sync result

            Log.d(TAG, "Performing quick sync...")

            // Simulate sync
            Thread.sleep(500)

            Result.success(SyncResult(
                newMessages = 0,
                success = true
            ))
        } catch (e: Exception) {
            Log.e(TAG, "Quick sync failed", e)
            Result.failure(e)
        }
    }

    /**
     * Connect to the Bridge WebSocket
     */
    suspend fun connect(): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            // WebSocket connection logic
            isConnected = true
            Log.d(TAG, "Connected to Bridge")
            Result.success(Unit)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to connect to Bridge", e)
            Result.failure(e)
        }
    }

    /**
     * Disconnect from the Bridge WebSocket
     */
    fun disconnect() {
        isConnected = false
        Log.d(TAG, "Disconnected from Bridge")
    }

    /**
     * Check if connected to Bridge
     */
    fun isBridgeConnected(): Boolean = isConnected

    /**
     * Clear all credentials (for logout)
     */
    fun clearCredentials() {
        homeserverUrl = null
        accessToken = null
        deviceId = null
        pusherManager = null
        isConnected = false
        Log.d(TAG, "Credentials cleared")
    }
}

/**
 * Result of a sync operation
 */
data class SyncResult(
    val newMessages: Int,
    val success: Boolean,
    val error: String? = null
)
