package app.armorclaw.data.repository

import android.content.Context
import android.util.Log
import app.armorclaw.network.BridgeApi
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Repository for Bridge Server communication
 *
 * Handles:
 * - Push token registration
 * - Message sync
 * - WebSocket connection management
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

    /**
     * Set the current device ID for push notifications
     */
    fun setDeviceId(id: String) {
        this.deviceId = id
    }

    /**
     * Register FCM push token with the Bridge Server
     */
    suspend fun registerPushToken(token: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = api.registerPushToken(
                deviceId = deviceId ?: "unknown",
                token = token,
                platform = "android"
            )

            if (response.success) {
                Log.d(TAG, "Push token registered: ${response.message}")
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to register push token", e)
            Result.failure(e)
        }
    }

    /**
     * Unregister FCM push token from the Bridge Server
     */
    suspend fun unregisterPushToken(token: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = api.unregisterPushToken(
                deviceId = deviceId ?: "unknown",
                token = token
            )

            if (response.success) {
                Log.d(TAG, "Push token unregistered: ${response.message}")
                Result.success(Unit)
            } else {
                Result.failure(Exception(response.message))
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to unregister push token", e)
            Result.failure(e)
        }
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
}

/**
 * Result of a sync operation
 */
data class SyncResult(
    val newMessages: Int,
    val success: Boolean,
    val error: String? = null
)
