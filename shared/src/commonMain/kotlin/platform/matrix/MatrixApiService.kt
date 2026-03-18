package com.armorclaw.shared.platform.matrix

import com.armorclaw.shared.platform.logging.LoggerDelegate
import com.armorclaw.shared.platform.logging.LogTag
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.util.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.IO
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject

/**
 * Matrix Client-Server API Service
 *
 * Low-level HTTP API service for Matrix Client-Server API communication.
 * Uses Ktor HttpClient for all network operations.
 *
 * ## Architecture
 * ```
 * MatrixClientImpl
 *      │
 *      └── MatrixApiService (this class)
 *           │
 *           ├── HttpClient (Ktor)
 *           │
 *           └── Matrix Homeserver
 * ```
 *
 * ## Usage
 * ```kotlin
 * val apiService = MatrixApiService(httpClient, json)
 *
 * // Login
 * val loginResult = apiService.login(
 *     homeserver = "https://matrix.org",
 *     username = "user",
 *     password = "password",
 *     deviceId = "DEVICE_ID"
 * )
 *
 * // Send message
 * val sendResult = apiService.sendMessage(
 *     homeserver = "https://matrix.org",
 *     accessToken = "token",
 *     roomId = "!room:matrix.org",
 *     content = RoomMessageContent(...)
 * )
 * ```
 *
 * ## Error Handling
 * All methods return `Result<T>` to handle both network errors and
 * Matrix API errors (M_FORBIDDEN, M_UNKNOWN_TOKEN, etc.).
 */
class MatrixApiService(
    private val httpClient: HttpClient,
    private val json: Json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
    }
) {
    private val logger = LoggerDelegate(LogTag.Network.MatrixClient)

    // ========================================================================
    // Authentication
    // ========================================================================

    /**
     * Login to Matrix homeserver
     *
     * POST /_matrix/client/v3/login
     */
    suspend fun login(
        homeserver: String,
        username: String,
        password: String,
        deviceId: String?,
        initialDeviceDisplayName: String? = null
    ): Result<LoginResponse> = withContext(Dispatchers.IO) {
        val url = "$homeserver/_matrix/client/v3/login"

        logger.logInfo("Logging in to Matrix", mapOf(
            "homeserver" to homeserver,
            "username" to username
        ))

        return@withContext try {
            val request = LoginRequest(
                type = "m.login.password",
                user = username,
                password = password,
                deviceId = deviceId,
                initialDeviceDisplayName = initialDeviceDisplayName ?: "ArmorClaw Android"
            )

            val response = httpClient.post(url) {
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                val loginResponse = json.decodeFromString<LoginResponse>(response.bodyAsText())
                logger.logInfo("Login successful", mapOf("userId" to loginResponse.userId))
                Result.success(loginResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Login failed", e, mapOf("homeserver" to homeserver))
            Result.failure(e)
        }
    }

    /**
     * Logout from Matrix homeserver
     *
     * POST /_matrix/client/v3/logout
     */
    suspend fun logout(
        homeserver: String,
        accessToken: String
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val url = "$homeserver/_matrix/client/v3/logout"

        logger.logInfo("Logging out from Matrix")

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
            }

            if (response.status.isSuccess()) {
                logger.logInfo("Logout successful")
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Logout failed", e)
            Result.failure(e)
        }
    }

    /**
     * Discover homeserver via well-known
     *
     * GET /.well-known/matrix/client
     */
    suspend fun discoverServer(
        serverName: String
    ): Result<WellKnownResponse> = withContext(Dispatchers.IO) {
        val url = "https://$serverName/.well-known/matrix/client"

        logger.logInfo("Discovering homeserver via well-known", mapOf("serverName" to serverName))

        return@withContext try {
            val response = httpClient.get(url)

            if (response.status.isSuccess()) {
                val wellKnown = json.decodeFromString<WellKnownResponse>(response.bodyAsText())
                logger.logInfo("Well-known discovery successful", mapOf(
                    "baseUrl" to (wellKnown.homeserver?.baseUrl ?: "null")
                ))
                Result.success(wellKnown)
            } else if (response.status == HttpStatusCode.NotFound) {
                // Try https://matrix.serverName as fallback
                logger.logDebug("No well-known found, using default pattern")
                Result.success(WellKnownResponse(
                    homeserver = HomeserverInfo(baseUrl = "https://matrix.$serverName")
                ))
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            // Network error - try default pattern
            logger.logWarning("Well-known discovery failed, using default", mapOf(
                "error" to e.message.orEmpty()
            ))
            Result.success(WellKnownResponse(
                homeserver = HomeserverInfo(baseUrl = "https://matrix.$serverName")
            ))
        }
    }

    /**
     * Refresh access token
     *
     * POST /_matrix/client/v3/refresh
     */
    suspend fun refreshToken(
        homeserver: String,
        refreshToken: String
    ): Result<RefreshTokenResponse> = withContext(Dispatchers.IO) {
        val url = "$homeserver/_matrix/client/v3/refresh"

        return@withContext try {
            val request = RefreshTokenRequest(refreshToken = refreshToken)

            val response = httpClient.post(url) {
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                val refreshResponse = json.decodeFromString<RefreshTokenResponse>(response.bodyAsText())
                Result.success(refreshResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Token refresh failed", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Room Operations
    // ========================================================================

    /**
     * Create a new room
     *
     * POST /_matrix/client/v3/createRoom
     */
    suspend fun createRoom(
        homeserver: String,
        accessToken: String,
        request: CreateRoomRequest
    ): Result<CreateRoomResponse> = withContext(Dispatchers.IO) {
        val url = "$homeserver/_matrix/client/v3/createRoom"

        logger.logInfo("Creating room", mapOf(
            "name" to (request.name ?: "unnamed"),
            "preset" to (request.preset?.name ?: "default")
        ))

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                val createResponse = json.decodeFromString<CreateRoomResponse>(response.bodyAsText())
                logger.logInfo("Room created", mapOf("roomId" to createResponse.roomId))
                Result.success(createResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to create room", e)
            Result.failure(e)
        }
    }

    /**
     * Join a room by ID or alias
     *
     * POST /_matrix/client/v3/join/{roomIdOrAlias}
     */
    suspend fun joinRoom(
        homeserver: String,
        accessToken: String,
        roomIdOrAlias: String,
        serverName: List<String>? = null
    ): Result<JoinRoomResponse> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomIdOrAlias.encodeURLPath()
        var url = "$homeserver/_matrix/client/v3/join/$encodedRoomId"

        serverName?.let { servers ->
            val serverParams = servers.joinToString("&") { "server_name=${it.encodeURLPath()}" }
            url = "$url?$serverParams"
        }

        logger.logInfo("Joining room", mapOf("roomId" to roomIdOrAlias))

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
            }

            if (response.status.isSuccess()) {
                val joinResponse = json.decodeFromString<JoinRoomResponse>(response.bodyAsText())
                logger.logInfo("Joined room", mapOf("roomId" to joinResponse.roomId))
                Result.success(joinResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to join room", e, mapOf("roomId" to roomIdOrAlias))
            Result.failure(e)
        }
    }

    /**
     * Leave a room
     *
     * POST /_matrix/client/v3/rooms/{roomId}/leave
     */
    suspend fun leaveRoom(
        homeserver: String,
        accessToken: String,
        roomId: String
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/leave"

        logger.logInfo("Leaving room", mapOf("roomId" to roomId))

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
            }

            if (response.status.isSuccess()) {
                logger.logInfo("Left room", mapOf("roomId" to roomId))
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to leave room", e, mapOf("roomId" to roomId))
            Result.failure(e)
        }
    }

    /**
     * Invite a user to a room
     *
     * POST /_matrix/client/v3/rooms/{roomId}/invite
     */
    suspend fun inviteUser(
        homeserver: String,
        accessToken: String,
        roomId: String,
        userId: String,
        reason: String? = null
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/invite"

        logger.logInfo("Inviting user to room", mapOf(
            "roomId" to roomId,
            "userId" to userId
        ))

        return@withContext try {
            val request = InviteUserRequest(userId = userId, reason = reason)

            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                logger.logInfo("User invited", mapOf("userId" to userId))
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to invite user", e)
            Result.failure(e)
        }
    }

    /**
     * Kick a user from a room
     *
     * POST /_matrix/client/v3/rooms/{roomId}/kick
     */
    suspend fun kickUser(
        homeserver: String,
        accessToken: String,
        roomId: String,
        userId: String,
        reason: String? = null
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/kick"

        logger.logInfo("Kicking user from room", mapOf(
            "roomId" to roomId,
            "userId" to userId,
            "reason" to (reason ?: "none")
        ))

        return@withContext try {
            val request = KickUserRequest(userId = userId, reason = reason)

            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                logger.logInfo("User kicked", mapOf("userId" to userId))
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to kick user", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Message Operations
    // ========================================================================

    /**
     * Send a message event to a room
     *
     * PUT /_matrix/client/v3/rooms/{roomId}/send/{eventType}/{txnId}
     */
    suspend fun sendMessage(
        homeserver: String,
        accessToken: String,
        roomId: String,
        eventType: String = "m.room.message",
        content: Any,
        txnId: String? = null
    ): Result<SendEventResponse> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val transactionId = txnId ?: generateTxnId()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/send/$eventType/$transactionId"

        logger.logDebug("Sending message", mapOf(
            "roomId" to roomId,
            "eventType" to eventType
        ))

        return@withContext try {
            val contentJson = when (content) {
                is String -> content
                else -> json.encodeToString(content)
            }

            val response = httpClient.put(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(contentJson)
            }

            if (response.status.isSuccess()) {
                val sendResponse = json.decodeFromString<SendEventResponse>(response.bodyAsText())
                logger.logDebug("Message sent", mapOf("eventId" to sendResponse.eventId))
                Result.success(sendResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to send message", e)
            Result.failure(e)
        }
    }

    /**
     * Get messages from a room (pagination)
     *
     * GET /_matrix/client/v3/rooms/{roomId}/messages
     */
    suspend fun getMessages(
        homeserver: String,
        accessToken: String,
        roomId: String,
        from: String? = null,
        to: String? = null,
        dir: String = "b",  // b = backwards, f = forwards
        limit: Int = 50,
        filter: String? = null
    ): Result<MessagesResponse> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val urlBuilder = StringBuilder("$homeserver/_matrix/client/v3/rooms/$encodedRoomId/messages")

        urlBuilder.append("?dir=$dir")
        urlBuilder.append("&limit=$limit")
        from?.let { urlBuilder.append("&from=$it") }
        to?.let { urlBuilder.append("&to=$it") }
        filter?.let { urlBuilder.append("&filter=$it") }

        logger.logDebug("Getting messages", mapOf(
            "roomId" to roomId,
            "limit" to limit
        ))

        return@withContext try {
            val response = httpClient.get(urlBuilder.toString()) {
                header("Authorization", "Bearer $accessToken")
            }

            if (response.status.isSuccess()) {
                val messagesResponse = json.decodeFromString<MessagesResponse>(response.bodyAsText())
                logger.logDebug("Got messages", mapOf(
                    "count" to messagesResponse.chunk.size
                ))
                Result.success(messagesResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to get messages", e)
            Result.failure(e)
        }
    }

    /**
     * Redact an event
     *
     * PUT /_matrix/client/v3/rooms/{roomId}/redact/{eventId}/{txnId}
     */
    suspend fun redactEvent(
        homeserver: String,
        accessToken: String,
        roomId: String,
        eventId: String,
        reason: String? = null,
        txnId: String? = null
    ): Result<SendEventResponse> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val encodedEventId = eventId.encodeURLPath()
        val transactionId = txnId ?: generateTxnId()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/redact/$encodedEventId/$transactionId"

        logger.logInfo("Redacting event", mapOf(
            "roomId" to roomId,
            "eventId" to eventId
        ))

        return@withContext try {
            val request = RedactEventRequest(reason = reason)

            val response = httpClient.put(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                val redactResponse = json.decodeFromString<SendEventResponse>(response.bodyAsText())
                Result.success(redactResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to redact event", e)
            Result.failure(e)
        }
    }

    /**
     * Send a state event to a room
     *
     * PUT /_matrix/client/v3/rooms/{roomId}/state/{eventType}/{stateKey}
     */
    suspend fun sendStateEvent(
        homeserver: String,
        accessToken: String,
        roomId: String,
        eventType: String,
        stateKey: String = "",
        content: Any
    ): Result<SendEventResponse> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val encodedStateKey = stateKey.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/state/$eventType/$encodedStateKey"

        logger.logDebug("Sending state event", mapOf(
            "roomId" to roomId,
            "eventType" to eventType
        ))

        return@withContext try {
            val contentJson = when (content) {
                is String -> content
                else -> json.encodeToString(content)
            }

            val response = httpClient.put(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(contentJson)
            }

            if (response.status.isSuccess()) {
                val sendResponse = json.decodeFromString<SendEventResponse>(response.bodyAsText())
                Result.success(sendResponse)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to send state event", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Presence / Typing / Receipts
    // ========================================================================

    /**
     * Set user presence
     *
     * PUT /_matrix/client/v3/presence/{userId}/status
     */
    suspend fun setPresence(
        homeserver: String,
        accessToken: String,
        userId: String,
        presence: String,
        statusMessage: String? = null
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedUserId = userId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/presence/$encodedUserId/status"

        logger.logDebug("Setting presence", mapOf(
            "presence" to presence
        ))

        return@withContext try {
            val request = SetPresenceRequest(
                presence = presence,
                statusMsg = statusMessage
            )

            val response = httpClient.put(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to set presence", e)
            Result.failure(e)
        }
    }

    /**
     * Get user presence
     *
     * GET /_matrix/client/v3/presence/{userId}/status
     */
    suspend fun getPresence(
        homeserver: String,
        accessToken: String,
        userId: String
    ): Result<PresenceResponse> = withContext(Dispatchers.IO) {
        val encodedUserId = userId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/presence/$encodedUserId/status"

        return@withContext try {
            val response = httpClient.get(url) {
                header("Authorization", "Bearer $accessToken")
            }

            if (response.status.isSuccess()) {
                val presence = json.decodeFromString<PresenceResponse>(response.bodyAsText())
                Result.success(presence)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to get presence", e)
            Result.failure(e)
        }
    }

    /**
     * Send typing notification
     *
     * PUT /_matrix/client/v3/rooms/{roomId}/typing/{userId}
     */
    suspend fun sendTyping(
        homeserver: String,
        accessToken: String,
        roomId: String,
        userId: String,
        typing: Boolean,
        timeout: Long? = if (typing) 30000L else null
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val encodedUserId = userId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/typing/$encodedUserId"

        return@withContext try {
            val request = SetTypingRequest(typing = typing, timeout = timeout)

            val response = httpClient.put(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to send typing", e)
            Result.failure(e)
        }
    }

    /**
     * Send read receipt
     *
     * POST /_matrix/client/v3/rooms/{roomId}/receipt/{receiptType}/{eventId}
     */
    suspend fun sendReadReceipt(
        homeserver: String,
        accessToken: String,
        roomId: String,
        eventId: String,
        receiptType: String = ReceiptType.READ
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val encodedEventId = eventId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/receipt/$receiptType/$encodedEventId"

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody("{}")
            }

            if (response.status.isSuccess()) {
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to send read receipt", e)
            Result.failure(e)
        }
    }

    /**
     * Set fully read marker
     *
     * POST /_matrix/client/v3/rooms/{roomId}/read_markers
     */
    suspend fun setReadMarkers(
        homeserver: String,
        accessToken: String,
        roomId: String,
        fullyRead: String? = null,
        read: String? = null
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedRoomId = roomId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/rooms/$encodedRoomId/read_markers"

        return@withContext try {
            val request = SetReadMarkersRequest(
                fullyRead = fullyRead,
                read = read
            )

            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to set read markers", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Profile Operations
    // ========================================================================

    /**
     * Get user profile
     *
     * GET /_matrix/client/v3/profile/{userId}
     */
    suspend fun getProfile(
        homeserver: String,
        userId: String
    ): Result<ProfileResponse> = withContext(Dispatchers.IO) {
        val encodedUserId = userId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/profile/$encodedUserId"

        return@withContext try {
            val response = httpClient.get(url)

            if (response.status.isSuccess()) {
                val profile = json.decodeFromString<ProfileResponse>(response.bodyAsText())
                Result.success(profile)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to get profile", e)
            Result.failure(e)
        }
    }

    /**
     * Set display name
     *
     * PUT /_matrix/client/v3/profile/{userId}/displayname
     */
    suspend fun setDisplayName(
        homeserver: String,
        accessToken: String,
        userId: String,
        displayName: String
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedUserId = userId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/profile/$encodedUserId/displayname"

        return@withContext try {
            val request = SetDisplayNameRequest(displayName = displayName)

            val response = httpClient.put(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to set display name", e)
            Result.failure(e)
        }
    }

    /**
     * Set avatar URL
     *
     * PUT /_matrix/client/v3/profile/{userId}/avatar_url
     */
    suspend fun setAvatarUrl(
        homeserver: String,
        accessToken: String,
        userId: String,
        avatarUrl: String?
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val encodedUserId = userId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/profile/$encodedUserId/avatar_url"

        return@withContext try {
            val request = SetAvatarUrlRequest(avatarUrl = avatarUrl)

            val response = httpClient.put(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to set avatar URL", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Media Operations
    // ========================================================================

    /**
     * Upload media to homeserver
     *
     * POST /_matrix/media/v3/upload
     */
    suspend fun uploadMedia(
        homeserver: String,
        accessToken: String,
        mimeType: String,
        data: ByteArray,
        filename: String? = null
    ): Result<String> = withContext(Dispatchers.IO) {
        var url = "$homeserver/_matrix/media/v3/upload"
        filename?.let { url = "$url?filename=${it.encodeURLPath()}" }

        logger.logInfo("Uploading media", mapOf(
            "mimeType" to mimeType,
            "size" to data.size
        ))

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.parse(mimeType))
                setBody(data)
            }

            if (response.status.isSuccess()) {
                val uploadResponse = json.decodeFromString<MediaUploadResponse>(response.bodyAsText())
                logger.logInfo("Media uploaded", mapOf("contentUri" to uploadResponse.contentUri))
                Result.success(uploadResponse.contentUri)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to upload media", e)
            Result.failure(e)
        }
    }

    /**
     * Download media from homeserver
     *
     * GET /_matrix/media/v3/download/{serverName}/{mediaId}
     */
    suspend fun downloadMedia(
        homeserver: String,
        accessToken: String,
        mxcUrl: String
    ): Result<ByteArray> = withContext(Dispatchers.IO) {
        // Parse mxc://server/mediaId
        val mxcMatch = Regex("mxc://([^/]+)/(.+)").find(mxcUrl)
            ?: return@withContext Result.failure(IllegalArgumentException("Invalid MXC URL: $mxcUrl"))

        val (serverName, mediaId) = mxcMatch.destructured
        val url = "$homeserver/_matrix/media/v3/download/$serverName/$mediaId"

        logger.logDebug("Downloading media", mapOf("mxcUrl" to mxcUrl))

        return@withContext try {
            val response = httpClient.get(url) {
                header("Authorization", "Bearer $accessToken")
            }

            if (response.status.isSuccess()) {
                val data = response.body<ByteArray>()
                Result.success(data)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to download media", e)
            Result.failure(e)
        }
    }

    /**
     * Get thumbnail URL for a media
     *
     * GET /_matrix/media/v3/thumbnail/{serverName}/{mediaId}
     */
    fun getThumbnailUrl(
        homeserver: String,
        mxcUrl: String,
        width: Int,
        height: Int,
        method: String = "scale"
    ): String? {
        val mxcMatch = Regex("mxc://([^/]+)/(.+)").find(mxcUrl)
            ?: return null

        val (serverName, mediaId) = mxcMatch.destructured
        return "$homeserver/_matrix/media/v3/thumbnail/$serverName/$mediaId?width=$width&height=$height&method=$method"
    }

    // ========================================================================
    // Pusher Operations
    // ========================================================================

    /**
     * Set or delete a pusher
     *
     * POST /_matrix/client/v3/pushers/set
     */
    suspend fun setPusher(
        homeserver: String,
        accessToken: String,
        request: SetPusherRequest
    ): Result<Unit> = withContext(Dispatchers.IO) {
        val url = "$homeserver/_matrix/client/v3/pushers/set"

        logger.logInfo("Setting pusher", mapOf(
            "appId" to request.appId,
            "kind" to request.kind
        ))

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(request))
            }

            if (response.status.isSuccess()) {
                Result.success(Unit)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to set pusher", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Filter Operations
    // ========================================================================

    /**
     * Create a filter for sync operations
     *
     * POST /_matrix/client/v3/user/{userId}/filter
     */
    suspend fun createFilter(
        homeserver: String,
        accessToken: String,
        userId: String,
        filter: FilterRequest
    ): Result<String> = withContext(Dispatchers.IO) {
        val encodedUserId = userId.encodeURLPath()
        val url = "$homeserver/_matrix/client/v3/user/$encodedUserId/filter"

        return@withContext try {
            val response = httpClient.post(url) {
                header("Authorization", "Bearer $accessToken")
                contentType(ContentType.Application.Json)
                setBody(json.encodeToString(filter))
            }

            if (response.status.isSuccess()) {
                val filterResponse = json.decodeFromString<FilterResponse>(response.bodyAsText())
                Result.success(filterResponse.filterId)
            } else {
                handleError(response)
            }
        } catch (e: Exception) {
            logger.logError("Failed to create filter", e)
            Result.failure(e)
        }
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    /**
     * Generate a unique transaction ID
     */
    private fun generateTxnId(): String {
        return "txn_${System.currentTimeMillis()}_${(0..9999).random()}"
    }

    /**
     * Handle HTTP error responses
     */
    private suspend fun <T> handleError(response: HttpResponse): Result<T> {
        return try {
            val errorBody = response.bodyAsText()
            val errorResponse = json.decodeFromString<MatrixErrorResponse>(errorBody)

            logger.logWarning("Matrix API error", mapOf(
                "status" to response.status.value,
                "errcode" to (errorResponse.errcode ?: "unknown"),
                "error" to (errorResponse.error ?: errorBody)
            ))

            val exception = when (errorResponse.errcode) {
                MatrixErrorResponse.M_UNKNOWN_TOKEN,
                MatrixErrorResponse.M_MISSING_TOKEN -> MatrixAuthException(
                    errorResponse.error ?: "Invalid or expired token",
                    errorResponse.errcode ?: "M_UNKNOWN_TOKEN",
                    errorResponse.softLogout ?: false
                )
                MatrixErrorResponse.M_FORBIDDEN -> MatrixForbiddenException(
                    errorResponse.error ?: "Forbidden"
                )
                MatrixErrorResponse.M_LIMIT_EXCEEDED -> MatrixRateLimitException(
                    errorResponse.error ?: "Rate limit exceeded",
                    errorResponse.retryAfterMs
                )
                else -> MatrixApiException(
                    errorResponse.error ?: "API error: ${response.status}",
                    errorResponse.errcode ?: "M_UNKNOWN",
                    response.status.value
                )
            }

            Result.failure(exception)
        } catch (e: Exception) {
            logger.logError("Failed to parse error response", e)
            Result.failure(MatrixApiException(
                "HTTP ${response.status}: ${response.bodyAsText()}",
                "HTTP_ERROR",
                response.status.value
            ))
        }
    }
}

// ============================================================================
// Exceptions
// ============================================================================

/**
 * Base Matrix API exception
 */
open class MatrixApiException(
    message: String,
    val errcode: String,
    val httpStatus: Int
) : Exception(message)

/**
 * Authentication exception (M_UNKNOWN_TOKEN, M_MISSING_TOKEN)
 */
class MatrixAuthException(
    message: String,
    errcode: String,
    val softLogout: Boolean = false
) : MatrixApiException(message, errcode, 401)

/**
 * Forbidden exception (M_FORBIDDEN)
 */
class MatrixForbiddenException(
    message: String
) : MatrixApiException(message, "M_FORBIDDEN", 403)

/**
 * Rate limit exception (M_LIMIT_EXCEEDED)
 */
class MatrixRateLimitException(
    message: String,
    val retryAfterMs: Long?
) : MatrixApiException(message, "M_LIMIT_EXCEEDED", 429)
