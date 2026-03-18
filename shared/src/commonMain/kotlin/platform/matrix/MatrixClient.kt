package com.armorclaw.shared.platform.matrix

import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.model.Room
import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserPresence
import com.armorclaw.shared.platform.matrix.event.MatrixEvent
import com.armorclaw.shared.platform.matrix.event.ArmorClawEventType
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.serialization.Serializable

/**
 * Matrix SDK Client Interface
 *
 * This interface abstracts the Matrix Rust SDK, providing a clean API for
 * the domain layer. All messaging, room management, and real-time communication
 * should go through this interface instead of the RPC client.
 *
 * ## Architecture Position
 * ```
 * Domain Layer (Use Cases)
 *        ↓
 * MatrixClient (this interface)
 *        ↓
 * Matrix Rust SDK (FFI)
 *        ↓
 * Matrix Homeserver (Conduit)
 * ```
 *
 * ## Migration Note
 * This replaces the RPC-based messaging in BridgeRpcClient. The RPC client
 * should only be used for admin functions (license validation, budget tracking).
 */
interface MatrixClient {

    // ========================================================================
    // Connection State
    // ========================================================================

    /**
     * Current sync state of the Matrix client
     */
    val syncState: StateFlow<SyncState>

    /**
     * Whether the client is currently logged in
     */
    val isLoggedIn: StateFlow<Boolean>

    /**
     * Current logged-in user
     */
    val currentUser: StateFlow<User?>

    /**
     * Network connectivity state
     */
    val connectionState: StateFlow<ConnectionState>

    // ========================================================================
    // Authentication
    // ========================================================================

    /**
     * Login to the Matrix homeserver
     *
     * This establishes a true Matrix session with client-side key management.
     * The client connects directly to the homeserver, not through the bridge.
     *
     * @param homeserver The Matrix homeserver URL (e.g., "https://matrix.example.com")
     * @param username The Matrix username (localpart or full MXID)
     * @param password The user's password
     * @param deviceId Optional device identifier
     * @return Login result with session information
     */
    suspend fun login(
        homeserver: String,
        username: String,
        password: String,
        deviceId: String? = null
    ): Result<MatrixSession>

    /**
     * Login using a well-known URL discovery
     *
     * @param serverName The server name (e.g., "example.com")
     * @param username The username
     * @param password The password
     * @return Login result
     */
    suspend fun loginWithWellKnown(
        serverName: String,
        username: String,
        password: String
    ): Result<MatrixSession>

    /**
     * Restore a previous session from stored credentials
     *
     * @param session The session to restore
     * @return Success or failure
     */
    suspend fun restoreSession(session: MatrixSession): Result<Unit>

    /**
     * Logout from the homeserver
     *
     * This invalidates the session on the server and clears local data.
     */
    suspend fun logout(): Result<Unit>

    // ========================================================================
    // Sync Operations
    // ========================================================================

    /**
     * Start syncing with the homeserver
     *
     * This begins the long-poll sync loop. Events will be emitted via
     * [syncState] and room/event flows.
     */
    fun startSync()

    /**
     * Stop syncing with the homeserver
     */
    fun stopSync()

    /**
     * Force an immediate sync
     */
    suspend fun syncOnce(): Result<Unit>

    // ========================================================================
    // Room Operations
    // ========================================================================

    /**
     * Get all rooms the user is a member of
     */
    val rooms: StateFlow<List<Room>>

    /**
     * Get a specific room by ID
     */
    suspend fun getRoom(roomId: String): Room?

    /**
     * Observe changes to a specific room
     */
    fun observeRoom(roomId: String): Flow<Room>

    /**
     * Create a new room
     *
     * @param name Room name
     * @param topic Optional room topic
     * @param isDirect Whether this is a direct message
     * @param invite User IDs to invite
     * @param isEncrypted Whether to enable E2EE (default: true)
     */
    suspend fun createRoom(
        name: String? = null,
        topic: String? = null,
        isDirect: Boolean = false,
        invite: List<String> = emptyList(),
        isEncrypted: Boolean = true
    ): Result<Room>

    /**
     * Join a room by ID or alias
     */
    suspend fun joinRoom(roomIdOrAlias: String): Result<Room>

    /**
     * Leave a room
     */
    suspend fun leaveRoom(roomId: String): Result<Unit>

    /**
     * Invite a user to a room
     */
    suspend fun inviteUser(roomId: String, userId: String): Result<Unit>

    /**
     * Kick a user from a room
     */
    suspend fun kickUser(roomId: String, userId: String, reason: String? = null): Result<Unit>

    /**
     * Set room name
     */
    suspend fun setRoomName(roomId: String, name: String): Result<Unit>

    /**
     * Set room topic
     */
    suspend fun setRoomTopic(roomId: String, topic: String): Result<Unit>

    // ========================================================================
    // Message Operations
    // ========================================================================

    /**
     * Get messages for a room
     *
     * @param roomId The room ID
     * @param limit Maximum number of messages to return
     * @param fromToken Pagination token for older messages
     * @return List of messages and optional pagination token
     */
    suspend fun getMessages(
        roomId: String,
        limit: Int = 50,
        fromToken: String? = null
    ): Result<MessageBatch>

    /**
     * Observe messages in a room (real-time updates)
     */
    fun observeMessages(roomId: String): Flow<List<Message>>

    /**
     * Send a text message
     *
     * @param roomId The room ID
     * @param text The message text
     * @param html Optional HTML formatted body
     * @return The event ID of the sent message
     */
    suspend fun sendTextMessage(
        roomId: String,
        text: String,
        html: String? = null
    ): Result<String>

    /**
     * Send an emote message (/me action)
     */
    suspend fun sendEmote(roomId: String, text: String): Result<String>

    /**
     * Send a reply to a message
     */
    suspend fun sendReply(
        roomId: String,
        replyToEventId: String,
        text: String
    ): Result<String>

    /**
     * Edit a message
     */
    suspend fun editMessage(
        roomId: String,
        eventId: String,
        newText: String
    ): Result<String>

    /**
     * Redact (delete) a message
     */
    suspend fun redactMessage(
        roomId: String,
        eventId: String,
        reason: String? = null
    ): Result<Unit>

    /**
     * Send a reaction to a message
     */
    suspend fun sendReaction(
        roomId: String,
        eventId: String,
        key: String
    ): Result<String>

    // ========================================================================
    // Event Handling
    // ========================================================================

    /**
     * Observe all Matrix events (raw)
     */
    fun observeEvents(): Flow<MatrixEvent>

    /**
     * Observe events for a specific room
     */
    fun observeRoomEvents(roomId: String): Flow<MatrixEvent>

    /**
     * Observe ArmorClaw-specific events (workflows, agent tasks, etc.)
     *
     * These are custom Matrix events with types starting with "com.armorclaw."
     */
    fun observeArmorClawEvents(roomId: String? = null): Flow<MatrixEvent>

    // ========================================================================
    // Presence Operations
    // ========================================================================

    /**
     * Set user presence
     */
    suspend fun setPresence(presence: UserPresence, statusMessage: String? = null): Result<Unit>

    /**
     * Get user presence
     */
    suspend fun getUserPresence(userId: String): Result<UserPresence>

    /**
     * Observe presence changes
     */
    fun observePresence(): Flow<PresenceUpdate>

    // ========================================================================
    // Typing Indicators
    // ========================================================================

    /**
     * Send typing notification
     */
    suspend fun sendTyping(roomId: String, typing: Boolean, timeout: Long = 30000): Result<Unit>

    /**
     * Observe typing notifications in a room
     */
    fun observeTyping(roomId: String): Flow<List<String>> // List of user IDs

    // ========================================================================
    // Read Receipts
    // ========================================================================

    /**
     * Send read receipt
     */
    suspend fun sendReadReceipt(roomId: String, eventId: String): Result<Unit>

    /**
     * Get unread count for a room
     */
    suspend fun getUnreadCount(roomId: String): Result<UnreadCount>

    // ========================================================================
    // Encryption Operations
    // ========================================================================

    /**
     * Check if a room is encrypted
     */
    suspend fun isRoomEncrypted(roomId: String): Boolean

    /**
     * Get encryption status for a room
     */
    fun getRoomEncryptionStatus(roomId: String): Flow<RoomEncryptionStatus>

    /**
     * Request verification with another user/device
     */
    suspend fun requestVerification(userId: String, deviceId: String? = null): Result<VerificationRequest>

    /**
     * Observe verification requests
     */
    fun observeVerificationRequests(): Flow<VerificationRequest>

    // ========================================================================
    // User Operations
    // ========================================================================

    /**
     * Get user profile
     */
    suspend fun getUser(userId: String): Result<User>

    /**
     * Get display name
     */
    suspend fun getDisplayName(userId: String): Result<String?>

    /**
     * Set display name
     */
    suspend fun setDisplayName(name: String): Result<Unit>

    /**
     * Get avatar URL
     */
    suspend fun getAvatarUrl(userId: String): Result<String?>

    /**
     * Set avatar
     *
     * @param mimeType The MIME type of the image
     * @param data The image data
     */
    suspend fun setAvatar(mimeType: String, data: ByteArray): Result<Unit>

    // ========================================================================
    // Push Notification Operations
    // ========================================================================

    /**
     * Register a pusher with the Matrix homeserver
     *
     * This configures the homeserver to send push notifications via FCM/APNs
     * when the user receives new events. This is the standard Matrix push
     * mechanism and must be used alongside any custom Bridge push registration.
     *
     * @param pushKey The FCM/APNs token
     * @param appId The application ID (e.g., "com.armorclaw.app")
     * @param appDisplayName The application display name
     * @param deviceDisplayName The device display name
     * @param pushGatewayUrl The push gateway URL (e.g., "https://push.armorclaw.app/_matrix/push/v1/notify")
     * @param profileTag Optional profile tag for multi-account support
     * @return Success or failure
     */
    suspend fun setPusher(
        pushKey: String,
        appId: String = "com.armorclaw.app",
        appDisplayName: String = "ArmorClaw",
        deviceDisplayName: String = "Android",
        pushGatewayUrl: String = "https://push.armorclaw.app/_matrix/push/v1/notify",
        profileTag: String? = null
    ): Result<Unit>

    /**
     * Remove a pusher from the Matrix homeserver
     *
     * Called on logout or when push notifications are disabled.
     *
     * @param pushKey The FCM/APNs token to remove
     * @param appId The application ID
     * @return Success or failure
     */
    suspend fun removePusher(
        pushKey: String,
        appId: String = "com.armorclaw.app"
    ): Result<Unit>

    // ========================================================================
    // Media Operations
    // ========================================================================

    /**
     * Upload media to the homeserver
     */
    suspend fun uploadMedia(mimeType: String, data: ByteArray): Result<String> // MXC URL

    /**
     * Download media from the homeserver
     */
    suspend fun downloadMedia(mxcUrl: String): Result<ByteArray>

    /**
     * Get thumbnail URL for a media
     */
    fun getThumbnailUrl(mxcUrl: String, width: Int, height: Int): String?
}

// ========================================================================
// Data Classes
// ========================================================================

/**
 * Matrix session information
 *
 * @property userId The Matrix user ID (e.g., @user:example.com)
 * @property deviceId The device ID
 * @property accessToken The access token for API calls
 * @property refreshToken Optional refresh token for token renewal
 * @property homeserver The homeserver URL
 * @property displayName Optional display name
 * @property avatarUrl Optional avatar MXC URL
 * @property expiresIn Token validity duration in seconds (from login response)
 * @property expiresAt Absolute expiration timestamp (calculated from expiresIn)
 * @property loginAt Timestamp when the session was created
 */
@Serializable
data class MatrixSession(
    val userId: String,
    val deviceId: String,
    val accessToken: String,
    val refreshToken: String? = null,
    val homeserver: String,
    val displayName: String? = null,
    val avatarUrl: String? = null,
    val expiresIn: Long? = null,          // Seconds until token expires (from server)
    val expiresAt: Long? = null,          // Absolute epoch timestamp when session expires
    val loginAt: Long? = null             // Epoch timestamp when session was created
) {
    /**
     * Check if the session has expired
     */
    fun isExpired(): Boolean {
        val expiry = expiresAt ?: return false
        return System.currentTimeMillis() / 1000 >= expiry
    }

    /**
     * Check if the session will expire within the given number of seconds
     */
    fun isExpiringSoon(withinSeconds: Long = 300): Boolean {
        val expiry = expiresAt ?: return false
        val now = System.currentTimeMillis() / 1000
        return (expiry - now) <= withinSeconds
    }

    /**
     * Get remaining time until expiration in seconds
     */
    fun remainingTimeSeconds(): Long? {
        val expiry = expiresAt ?: return null
        val now = System.currentTimeMillis() / 1000
        return (expiry - now).coerceAtLeast(0)
    }

    companion object {
        /**
         * Default session validity duration (24 hours)
         * Used when server doesn't provide expiresIn
         */
        const val DEFAULT_SESSION_DURATION_SECONDS = 24 * 60 * 60L

        /**
         * Create a session with calculated expiration
         */
        fun withExpiration(
            userId: String,
            deviceId: String,
            accessToken: String,
            refreshToken: String? = null,
            homeserver: String,
            displayName: String? = null,
            avatarUrl: String? = null,
            expiresIn: Long? = null
        ): MatrixSession {
            val now = System.currentTimeMillis() / 1000
            val duration = expiresIn ?: DEFAULT_SESSION_DURATION_SECONDS
            return MatrixSession(
                userId = userId,
                deviceId = deviceId,
                accessToken = accessToken,
                refreshToken = refreshToken,
                homeserver = homeserver,
                displayName = displayName,
                avatarUrl = avatarUrl,
                expiresIn = duration,
                expiresAt = now + duration,
                loginAt = now
            )
        }
    }
}

/**
 * Sync state
 */
sealed class SyncState {
    object Idle : SyncState()
    object Connecting : SyncState()
    data class Syncing(val since: String?) : SyncState()
    data class Error(val error: Throwable) : SyncState()
    object Stopped : SyncState()
}

/**
 * Connection state
 */
sealed class ConnectionState {
    object Online : ConnectionState()
    object Offline : ConnectionState()
    object Reconnecting : ConnectionState()
    data class Error(val error: Throwable) : ConnectionState()
}

/**
 * Message batch with pagination
 */
data class MessageBatch(
    val messages: List<Message>,
    val nextToken: String? = null,
    val prevToken: String? = null
)

/**
 * Room encryption status
 */
sealed class RoomEncryptionStatus {
    object Unencrypted : RoomEncryptionStatus()
    object Encrypted : RoomEncryptionStatus()
    object Verified : RoomEncryptionStatus()
    data class Warning(val reason: String) : RoomEncryptionStatus()
    object Unknown : RoomEncryptionStatus()
}

/**
 * Unread count information
 */
data class UnreadCount(
    val notificationCount: Int,
    val highlightCount: Int,
    val markedUnread: Boolean
)

/**
 * Presence update
 */
data class PresenceUpdate(
    val userId: String,
    val presence: UserPresence,
    val statusMessage: String? = null,
    val lastActiveAgo: Long? = null
)

/**
 * Verification request
 */
data class VerificationRequest(
    val requestId: String,
    val userId: String,
    val deviceId: String,
    val methods: List<String>,
    val timestamp: Long
)

/**
 * Configuration for Matrix client
 */
data class MatrixClientConfig(
    val defaultHomeserver: String = "https://matrix.org",
    val syncTimeout: Long = 30000,
    val enableEncryption: Boolean = true,
    val enablePresence: Boolean = true,
    val enableTypingIndicators: Boolean = true,
    val enableReadReceipts: Boolean = true,
    val sessionPersistenceKey: String = "matrix_session",
    val enableCrossProcessLock: Boolean = true,
    val autoJoinInvites: Boolean = false,
    val backgroundSync: Boolean = true
)
