package com.armorclaw.shared.platform.matrix

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

/**
 * Matrix Client-Server API Data Transfer Objects
 *
 * These models represent the request/response structures for the
 * Matrix Client-Server API (https://spec.matrix.org/v1.6/client-server-api/).
 *
 * ## Organization
 * - Authentication models (login, logout, well-known)
 * - Room models (create, join, leave, invite, kick)
 * - Message models (send, get, redact, reaction)
 * - Presence/Typing/Receipt models
 * - Profile models
 * - Media models
 * - Pusher models
 */

// ============================================================================
// Authentication Models
// ============================================================================

/**
 * Login request to Matrix homeserver
 *
 * POST /_matrix/client/v3/login
 */
@Serializable
data class LoginRequest(
    val type: String = "m.login.password",
    val user: String? = null,
    val username: String? = null,  // Some servers use this
    val password: String,
    val deviceId: String? = null,
    @SerialName("initial_device_display_name")
    val initialDeviceDisplayName: String? = null,
    val address: String? = null,   // For m.login.email or m.login.msisdn
    val medium: String? = null     // "email" or "msisdn"
)

/**
 * Login response from Matrix homeserver
 */
@Serializable
data class LoginResponse(
    @SerialName("user_id")
    val userId: String,
    @SerialName("access_token")
    val accessToken: String,
    @SerialName("device_id")
    val deviceId: String,
    @SerialName("refresh_token")
    val refreshToken: String? = null,
    @SerialName("expires_in_ms")
    val expiresInMs: Long? = null,
    @SerialName("well_known")
    val wellKnown: WellKnownResponse? = null
)

/**
 * Well-known discovery response
 *
 * GET /.well-known/matrix/client
 */
@Serializable
data class WellKnownResponse(
    @SerialName("m.homeserver")
    val homeserver: HomeserverInfo? = null,
    @SerialName("m.identity_server")
    val identityServer: IdentityServerInfo? = null
)

@Serializable
data class HomeserverInfo(
    @SerialName("base_url")
    val baseUrl: String
)

@Serializable
data class IdentityServerInfo(
    @SerialName("base_url")
    val baseUrl: String
)

/**
 * Logout response
 */
@Serializable
data class LogoutResponse(
    @SerialName("logout_success")
    val success: Boolean = true
)

/**
 * Refresh token request
 */
@Serializable
data class RefreshTokenRequest(
    @SerialName("refresh_token")
    val refreshToken: String
)

/**
 * Refresh token response
 */
@Serializable
data class RefreshTokenResponse(
    @SerialName("access_token")
    val accessToken: String,
    @SerialName("refresh_token")
    val refreshToken: String? = null,
    @SerialName("expires_in_ms")
    val expiresInMs: Long? = null
)

// ============================================================================
// Room Models
// ============================================================================

/**
 * Create room request
 *
 * POST /_matrix/client/v3/createRoom
 */
@Serializable
data class CreateRoomRequest(
    val name: String? = null,
    val topic: String? = null,
    val preset: RoomPreset? = null,
    val visibility: RoomVisibility? = null,
    @SerialName("room_alias_name")
    val roomAliasName: String? = null,
    val invite: List<String>? = null,
    @SerialName("invite_3pid")
    val invite3pid: List<Invite3pid>? = null,
    @SerialName("is_direct")
    val isDirect: Boolean? = null,
    @SerialName("initial_state")
    val initialState: List<InitialStateEvent>? = null,
    @SerialName("power_level_content_override")
    val powerLevelContentOverride: JsonObject? = null
)

@Serializable
enum class RoomPreset {
    @SerialName("private_chat")
    PRIVATE_CHAT,

    @SerialName("trusted_private_chat")
    TRUSTED_PRIVATE_CHAT,

    @SerialName("public_chat")
    PUBLIC_CHAT
}

@Serializable
enum class RoomVisibility {
    @SerialName("public")
    PUBLIC,

    @SerialName("private")
    PRIVATE
}

@Serializable
data class Invite3pid(
    val medium: String,
    val address: String,
    @SerialName("id_server")
    val idServer: String? = null,
    @SerialName("id_access_token")
    val idAccessToken: String? = null
)

@Serializable
data class InitialStateEvent(
    val type: String,
    @SerialName("state_key")
    val stateKey: String? = "",
    val content: JsonObject
)

/**
 * Create room response
 */
@Serializable
data class CreateRoomResponse(
    @SerialName("room_id")
    val roomId: String
)

/**
 * Join room response
 *
 * POST /_matrix/client/v3/join/{roomIdOrAlias}
 */
@Serializable
data class JoinRoomResponse(
    @SerialName("room_id")
    val roomId: String
)

/**
 * Invite user request
 *
 * POST /_matrix/client/v3/rooms/{roomId}/invite
 */
@Serializable
data class InviteUserRequest(
    @SerialName("user_id")
    val userId: String,
    val reason: String? = null,
    @SerialName("is_direct")
    val isDirect: Boolean? = null
)

/**
 * Kick user request
 *
 * POST /_matrix/client/v3/rooms/{roomId}/kick
 */
@Serializable
data class KickUserRequest(
    @SerialName("user_id")
    val userId: String,
    val reason: String? = null
)

/**
 * Leave room response
 */
@Serializable
data class LeaveRoomResponse(
    @SerialName("leave_success")
    val success: Boolean = true
)

// ============================================================================
// Message Models
// ============================================================================

/**
 * Send event response
 */
@Serializable
data class SendEventResponse(
    @SerialName("event_id")
    val eventId: String
)

/**
 * Room message event content
 */
@Serializable
data class RoomMessageContent(
    val msgtype: String,
    val body: String,
    @SerialName("formatted_body")
    val formattedBody: String? = null,
    @SerialName("format")
    val format: String? = null,
    @SerialName("m.relates_to")
    val relatesTo: RelationInfo? = null,
    @SerialName("m.new_content")
    val newContent: NewContentInfo? = null
)

@Serializable
data class RelationInfo(
    @SerialName("m.in_reply_to")
    val inReplyTo: InReplyToInfo? = null,
    @SerialName("rel_type")
    val relType: String? = null,
    @SerialName("event_id")
    val eventId: String? = null
)

@Serializable
data class InReplyToInfo(
    @SerialName("event_id")
    val eventId: String
)

@Serializable
data class NewContentInfo(
    val msgtype: String,
    val body: String,
    @SerialName("formatted_body")
    val formattedBody: String? = null,
    @SerialName("format")
    val format: String? = null
)

/**
 * Reaction event content
 */
@Serializable
data class ReactionContent(
    @SerialName("m.relates_to")
    val relatesTo: ReactionRelationInfo
)

@Serializable
data class ReactionRelationInfo(
    @SerialName("rel_type")
    val relType: String = "m.annotation",
    @SerialName("event_id")
    val eventId: String,
    val key: String
)

/**
 * Redact event request
 *
 * PUT /_matrix/client/v3/rooms/{roomId}/redact/{eventId}/{txnId}
 */
@Serializable
data class RedactEventRequest(
    val reason: String? = null
)

/**
 * Messages response (pagination)
 *
 * GET /_matrix/client/v3/rooms/{roomId}/messages
 */
@Serializable
data class MessagesResponse(
    val start: String,
    val end: String? = null,
    val chunk: List<MatrixEventRaw>,
    @SerialName("state")
    val state: List<MatrixEventRaw>? = null
)

/**
 * Context response (around an event)
 *
 * GET /_matrix/client/v3/rooms/{roomId}/context/{eventId}
 */
@Serializable
data class ContextResponse(
    val event: MatrixEventRaw,
    @SerialName("events_before")
    val eventsBefore: List<MatrixEventRaw>,
    @SerialName("events_after")
    val eventsAfter: List<MatrixEventRaw>,
    val state: List<MatrixEventRaw>? = null
)

// ============================================================================
// State Event Models
// ============================================================================

/**
 * Room name state event content
 */
@Serializable
data class RoomNameContent(
    val name: String
)

/**
 * Room topic state event content
 */
@Serializable
data class RoomTopicContent(
    val topic: String
)

/**
 * Room avatar state event content
 */
@Serializable
data class RoomAvatarContent(
    val url: String? = null
)

/**
 * Room encryption state event content
 */
@Serializable
data class RoomEncryptionContent(
    val algorithm: String = "m.megolm.v1.aes-sha2",
    @SerialName("rotation_period_ms")
    val rotationPeriodMs: Long? = null,
    @SerialName("rotation_period_msgs")
    val rotationPeriodMsgs: Long? = null
)

/**
 * Room power levels content
 */
@Serializable
data class RoomPowerLevelsContent(
    val users: Map<String, Int>? = null,
    val usersDefault: Int? = null,
    val events: Map<String, Int>? = null,
    val eventsDefault: Int? = null,
    val stateDefault: Int? = null,
    val ban: Int? = null,
    val kick: Int? = null,
    val redact: Int? = null,
    val invite: Int? = null,
    @SerialName("notifications")
    val notifications: PowerLevelNotifications? = null
)

@Serializable
data class PowerLevelNotifications(
    val room: Int? = null
)

// ============================================================================
// Presence / Typing / Receipt Models
// ============================================================================

/**
 * Presence state
 */
@Serializable
enum class PresenceState {
    @SerialName("online")
    ONLINE,

    @SerialName("unavailable")
    UNAVAILABLE,

    @SerialName("offline")
    OFFLINE
}

/**
 * Set presence request
 *
 * PUT /_matrix/client/v3/presence/{userId}/status
 */
@Serializable
data class SetPresenceRequest(
    val presence: String,
    @SerialName("status_msg")
    val statusMsg: String? = null
)

/**
 * Get presence response
 */
@Serializable
data class PresenceResponse(
    val presence: String,
    @SerialName("status_msg")
    val statusMsg: String? = null,
    @SerialName("last_active_ago")
    val lastActiveAgo: Long? = null,
    @SerialName("currently_active")
    val currentlyActive: Boolean? = null
)

/**
 * Set typing request
 *
 * PUT /_matrix/client/v3/rooms/{roomId}/typing/{userId}
 */
@Serializable
data class SetTypingRequest(
    val typing: Boolean,
    val timeout: Long? = null
)

/**
 * Read receipt types
 */
object ReceiptType {
    const val READ = "m.read"
    const val READ_PRIVATE = "m.read.private"
    const val FULLY_READ = "m.fully_read"
}

/**
 * Set read receipt request (receipt content)
 */
@Serializable
data class ReadReceiptContent(
    @SerialName("thread_id")
    val threadId: String? = null
)

/**
 * Set fully read marker request
 *
 * POST /_matrix/client/v3/rooms/{roomId}/read_markers
 */
@Serializable
data class SetReadMarkersRequest(
    @SerialName("m.fully_read")
    val fullyRead: String? = null,
    @SerialName("m.read")
    val read: String? = null,
    @SerialName("m.read.private")
    val readPrivate: String? = null
)

// ============================================================================
// Profile Models
// ============================================================================

/**
 * Get profile response
 */
@Serializable
data class ProfileResponse(
    @SerialName("displayname")
    val displayName: String? = null,
    @SerialName("avatar_url")
    val avatarUrl: String? = null
)

/**
 * Set display name request
 *
 * PUT /_matrix/client/v3/profile/{userId}/displayname
 */
@Serializable
data class SetDisplayNameRequest(
    @SerialName("displayname")
    val displayName: String
)

/**
 * Set avatar URL request
 *
 * PUT /_matrix/client/v3/profile/{userId}/avatar_url
 */
@Serializable
data class SetAvatarUrlRequest(
    @SerialName("avatar_url")
    val avatarUrl: String?
)

// ============================================================================
// Media Models
// ============================================================================

/**
 * Media upload response
 *
 * POST /_matrix/media/v3/upload
 */
@Serializable
data class MediaUploadResponse(
    @SerialName("content_uri")
    val contentUri: String
)

/**
 * Media thumbnail configuration
 */
@Serializable
data class ThumbnailConfig(
    val width: Int,
    val height: Int,
    val method: ThumbnailMethod? = ThumbnailMethod.SCALE
)

@Serializable
enum class ThumbnailMethod {
    @SerialName("crop")
    CROP,

    @SerialName("scale")
    SCALE
}

// ============================================================================
// Pusher Models
// ============================================================================

/**
 * Set pusher request
 *
 * POST /_matrix/client/v3/pushers/set
 */
@Serializable
data class SetPusherRequest(
    val pushkey: String,
    val kind: String,  // "http" or null to delete
    @SerialName("app_id")
    val appId: String,
    @SerialName("app_display_name")
    val appDisplayName: String,
    @SerialName("device_display_name")
    val deviceDisplayName: String,
    @SerialName("profile_tag")
    val profileTag: String? = null,
    val lang: String,
    val data: PusherData,
    val append: Boolean = false
)

@Serializable
data class PusherData(
    val url: String? = null,
    val format: String? = null,
    @SerialName("default_payload")
    val defaultPayload: JsonObject? = null
)

// ============================================================================
// Error Models
// ============================================================================

/**
 * Matrix API error response
 */
@Serializable
data class MatrixErrorResponse(
    val errcode: String? = null,
    val error: String? = null,
    @SerialName("soft_logout")
    val softLogout: Boolean? = null,
    @SerialName("retry_after_ms")
    val retryAfterMs: Long? = null
) {
    companion object {
        // Standard error codes
        const val M_FORBIDDEN = "M_FORBIDDEN"
        const val M_UNKNOWN_TOKEN = "M_UNKNOWN_TOKEN"
        const val M_MISSING_TOKEN = "M_MISSING_TOKEN"
        const val M_BAD_JSON = "M_BAD_JSON"
        const val M_NOT_JSON = "M_NOT_JSON"
        const val M_NOT_FOUND = "M_NOT_FOUND"
        const val M_LIMIT_EXCEEDED = "M_LIMIT_EXCEEDED"
        const val M_UNKNOWN = "M_UNKNOWN"
        const val M_UNRECOGNIZED = "M_UNRECOGNIZED"
        const val M_UNAUTHORIZED = "M_UNAUTHORIZED"
        const val M_USER_DEACTIVATED = "M_USER_DEACTIVATED"
        const val M_USER_IN_USE = "M_USER_IN_USE"
        const val M_INVALID_USERNAME = "M_INVALID_USERNAME"
        const val M_ROOM_IN_USE = "M_ROOM_IN_USE"
        const val M_INVALID_ROOM_STATE = "M_INVALID_ROOM_STATE"
        const val M_THREEPID_IN_USE = "M_THREEPID_IN_USE"
        const val M_THREEPID_NOT_FOUND = "M_THREEPID_NOT_FOUND"
        const val M_THREEPID_AUTH_FAILED = "M_THREEPID_AUTH_FAILED"
        const val M_THREEPID_DENIED = "M_THREEPID_DENIED"
        const val M_SERVER_NOT_TRUSTED = "M_SERVER_NOT_TRUSTED"
        const val M_UNSUPPORTED_ROOM_VERSION = "M_UNSUPPORTED_ROOM_VERSION"
        const val M_INCOMPATIBLE_ROOM_VERSION = "M_INCOMPATIBLE_ROOM_VERSION"
        const val M_EXCLUSIVE = "M_EXCLUSIVE"
        const val M_RESOURCE_LIMIT_EXCEEDED = "M_RESOURCE_LIMIT_EXCEEDED"
        const val M_GUEST_ACCESS_FORBIDDEN = "M_GUEST_ACCESS_FORBIDDEN"
        const val M_CONSENT_NOT_GIVEN = "M_CONSENT_NOT_GIVEN"
        const val M_UNSUPPORTED_MATRIX_VERSION = "M_UNSUPPORTED_MATRIX_VERSION"
    }
}

// ============================================================================
// Filter Models
// ============================================================================

/**
 * Filter for sync requests
 *
 * POST /_matrix/client/v3/user/{userId}/filter
 */
@Serializable
data class FilterRequest(
    val room: RoomFilter? = null,
    val presence: EventFilter? = null,
    @SerialName("account_data")
    val accountData: EventFilter? = null,
    @SerialName("event_format")
    val eventFormat: String? = null,
    @SerialName("event_fields")
    val eventFields: List<String>? = null
)

@Serializable
data class RoomFilter(
    @SerialName("not_rooms")
    val notRooms: List<String>? = null,
    val rooms: List<String>? = null,
    val ephemeral: EventFilter? = null,
    @SerialName("include_leave")
    val includeLeave: Boolean? = null,
    val state: StateFilter? = null,
    val timeline: TimelineFilter? = null,
    @SerialName("account_data")
    val accountData: EventFilter? = null
)

@Serializable
data class EventFilter(
    val limit: Int? = null,
    val notSenders: List<String>? = null,
    val senders: List<String>? = null,
    val notTypes: List<String>? = null,
    val types: List<String>? = null
)

@Serializable
data class StateFilter(
    val limit: Int? = null,
    @SerialName("not_senders")
    val notSenders: List<String>? = null,
    val senders: List<String>? = null,
    @SerialName("not_types")
    val notTypes: List<String>? = null,
    val types: List<String>? = null,
    @SerialName("lazy_load_members")
    val lazyLoadMembers: Boolean? = null,
    @SerialName("include_redundant_members")
    val includeRedundantMembers: Boolean? = null
)

@Serializable
data class TimelineFilter(
    val limit: Int? = null,
    @SerialName("not_senders")
    val notSenders: List<String>? = null,
    val senders: List<String>? = null,
    @SerialName("not_types")
    val notTypes: List<String>? = null,
    val types: List<String>? = null,
    @SerialName("unread_thread_notifications")
    val unreadThreadNotifications: Boolean? = null
)

/**
 * Filter response
 */
@Serializable
data class FilterResponse(
    @SerialName("filter_id")
    val filterId: String
)
