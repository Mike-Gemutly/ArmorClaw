package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

/**
 * Comprehensive error code taxonomy for ArmorClaw
 * Organized by category with user-friendly messages
 */
@Serializable
enum class ArmorClawErrorCode(
    val code: String,
    val userMessage: String,
    val recoverable: Boolean,
    val category: ErrorCategory
) {
    // Voice/Call Errors (E001-E010)
    VOICE_MIC_DENIED(
        "E001",
        "Microphone access denied. Please enable in settings.",
        true,
        ErrorCategory.PERMISSION
    ),
    VOICE_PERMISSION_REQUIRED(
        "E002",
        "Microphone permission required for voice calls.",
        true,
        ErrorCategory.PERMISSION
    ),
    TURN_SERVER_TIMEOUT(
        "E003",
        "Call server unreachable. Please try again.",
        true,
        ErrorCategory.NETWORK
    ),
    ICE_CONNECTION_FAILED(
        "E004",
        "Network connection failed. Check your internet.",
        true,
        ErrorCategory.NETWORK
    ),
    CALL_ALREADY_ACTIVE(
        "E005",
        "You already have an active call.",
        true,
        ErrorCategory.STATE
    ),
    CALL_NOT_FOUND(
        "E006",
        "Call no longer exists.",
        false,
        ErrorCategory.STATE
    ),
    CALL_DECLINED(
        "E007",
        "Call was declined.",
        false,
        ErrorCategory.STATE
    ),
    CALL_TIMEOUT(
        "E008",
        "No answer. Call timed out.",
        false,
        ErrorCategory.STATE
    ),
    AUDIO_DEVICE_ERROR(
        "E009",
        "Audio device error. Please check your microphone.",
        true,
        ErrorCategory.HARDWARE
    ),
    CAMERA_DENIED(
        "E010",
        "Camera access denied.",
        true,
        ErrorCategory.PERMISSION
    ),

    // Network Errors (E011-E020)
    HOMESERVER_UNREACHABLE(
        "E011",
        "Server temporarily unavailable.",
        true,
        ErrorCategory.NETWORK
    ),
    CONNECTION_TIMEOUT(
        "E012",
        "Connection timed out. Please try again.",
        true,
        ErrorCategory.NETWORK
    ),
    NETWORK_CHANGED(
        "E013",
        "Network changed. Reconnecting...",
        true,
        ErrorCategory.NETWORK
    ),
    NO_NETWORK(
        "E014",
        "No internet connection.",
        true,
        ErrorCategory.NETWORK
    ),
    DNS_RESOLUTION_FAILED(
        "E015",
        "Could not resolve server address.",
        true,
        ErrorCategory.NETWORK
    ),
    SSL_ERROR(
        "E016",
        "Secure connection failed.",
        true,
        ErrorCategory.NETWORK
    ),
    RATE_LIMITED(
        "E017",
        "Too many requests. Please wait.",
        true,
        ErrorCategory.NETWORK
    ),
    SERVER_ERROR(
        "E018",
        "Server error. Please try again later.",
        true,
        ErrorCategory.NETWORK
    ),
    PROXY_ERROR(
        "E019",
        "Proxy configuration error.",
        true,
        ErrorCategory.NETWORK
    ),
    WEBSOCKET_FAILED(
        "E020",
        "Real-time connection failed. Using fallback.",
        true,
        ErrorCategory.NETWORK
    ),

    // Trust/Verification Errors (E021-E030)
    DEVICE_UNVERIFIED(
        "E021",
        "Device verification required for this action.",
        true,
        ErrorCategory.TRUST
    ),
    CROSS_SIGNING_REQUIRED(
        "E022",
        "Please set up cross-signing first.",
        true,
        ErrorCategory.TRUST
    ),
    VERIFICATION_FAILED(
        "E023",
        "Verification failed. Please try again.",
        true,
        ErrorCategory.TRUST
    ),
    SESSION_EXPIRED(
        "E024",
        "Session expired. Please log in again.",
        true,
        ErrorCategory.AUTH
    ),
    VERIFICATION_CANCELLED(
        "E025",
        "Verification was cancelled.",
        true,
        ErrorCategory.TRUST
    ),
    VERIFICATION_TIMEOUT(
        "E026",
        "Verification timed out.",
        true,
        ErrorCategory.TRUST
    ),
    KEY_SIGNATURE_INVALID(
        "E027",
        "Invalid key signature.",
        false,
        ErrorCategory.TRUST
    ),
    TRUST_CHAIN_BROKEN(
        "E028",
        "Trust chain verification failed.",
        true,
        ErrorCategory.TRUST
    ),
    DEVICE_BLOCKED(
        "E029",
        "This device has been blocked.",
        false,
        ErrorCategory.TRUST
    ),
    RECOVERY_REQUIRED(
        "E030",
        "Account recovery required.",
        true,
        ErrorCategory.TRUST
    ),

    // Encryption Errors (E031-E040)
    ENCRYPTION_KEY_ERROR(
        "E031",
        "Encryption error. Please restart the app.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    DECRYPTION_FAILED(
        "E032",
        "Could not decrypt message.",
        false,
        ErrorCategory.ENCRYPTION
    ),
    KEY_BACKUP_REQUIRED(
        "E033",
        "Please back up your encryption keys.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    MEGOLM_SESSION_ERROR(
        "E034",
        "Message encryption session error.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    OLM_SESSION_ERROR(
        "E035",
        "Key exchange session error.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    KEY_NOT_TRUSTED(
        "E036",
        "Encryption key not trusted.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    KEY_ROTATION_REQUIRED(
        "E037",
        "Encryption keys need rotation.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    CLIPBOARD_ENCRYPTION_FAILED(
        "E038",
        "Could not encrypt clipboard content.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    DATABASE_ENCRYPTION_ERROR(
        "E039",
        "Database encryption error.",
        true,
        ErrorCategory.ENCRYPTION
    ),
    SECURE_STORAGE_ERROR(
        "E040",
        "Secure storage error.",
        true,
        ErrorCategory.ENCRYPTION
    ),

    // Sync Errors (E041-E050)
    SYNC_QUEUE_OVERFLOW(
        "E041",
        "Too many pending operations. Please wait.",
        true,
        ErrorCategory.SYNC
    ),
    SYNC_CONFLICT(
        "E042",
        "Data conflict detected. Resolving...",
        true,
        ErrorCategory.SYNC
    ),
    OFFLINE_MODE(
        "E043",
        "You are offline. Changes will sync when connected.",
        true,
        ErrorCategory.SYNC
    ),
    SYNC_VERSION_MISMATCH(
        "E044",
        "Sync version mismatch. Updating...",
        true,
        ErrorCategory.SYNC
    ),
    SYNC_TOKEN_INVALID(
        "E045",
        "Sync position invalid. Re-syncing...",
        true,
        ErrorCategory.SYNC
    ),
    SYNC_TIMEOUT(
        "E046",
        "Sync timed out. Will retry.",
        true,
        ErrorCategory.SYNC
    ),
    SYNC_STORAGE_FULL(
        "E047",
        "Local storage full. Please free up space.",
        true,
        ErrorCategory.SYNC
    ),
    BATCH_SYNC_FAILED(
        "E048",
        "Batch sync failed. Retrying individually...",
        true,
        ErrorCategory.SYNC
    ),
    SYNC_FORCED_RESET(
        "E049",
        "Sync reset required. Clearing cache...",
        true,
        ErrorCategory.SYNC
    ),
    BACKGROUND_SYNC_DISABLED(
        "E050",
        "Background sync disabled in settings.",
        true,
        ErrorCategory.SYNC
    ),

    // Thread Errors (E051-E060)
    THREAD_NOT_FOUND(
        "E051",
        "Thread not found or deleted.",
        false,
        ErrorCategory.CONTENT
    ),
    THREAD_REPLY_FAILED(
        "E052",
        "Could not send reply. Please try again.",
        true,
        ErrorCategory.CONTENT
    ),
    THREAD_ROOT_DELETED(
        "E053",
        "Original message was deleted.",
        false,
        ErrorCategory.CONTENT
    ),
    THREAD_PERMISSION_DENIED(
        "E054",
        "You don't have permission to reply.",
        false,
        ErrorCategory.CONTENT
    ),
    THREAD_TOO_LONG(
        "E055",
        "Thread is too long. Some messages may not load.",
        true,
        ErrorCategory.CONTENT
    ),
    THREAD_PARTICIPANTS_LIMIT(
        "E056",
        "Thread participant limit reached.",
        true,
        ErrorCategory.CONTENT
    ),

    // Room Errors (E061-E070)
    ROOM_NOT_FOUND(
        "E061",
        "Room not found.",
        false,
        ErrorCategory.CONTENT
    ),
    ROOM_ACCESS_DENIED(
        "E062",
        "Access to room denied.",
        false,
        ErrorCategory.CONTENT
    ),
    ROOM_FULL(
        "E063",
        "Room is full.",
        false,
        ErrorCategory.CONTENT
    ),
    ROOM_ARCHIVED(
        "E064",
        "Room has been archived.",
        true,
        ErrorCategory.CONTENT
    ),
    NOT_MEMBER(
        "E065",
        "You are no longer a member of this room.",
        false,
        ErrorCategory.CONTENT
    ),
    BANNED_FROM_ROOM(
        "E066",
        "You have been banned from this room.",
        false,
        ErrorCategory.CONTENT
    ),
    ROOM_CREATION_FAILED(
        "E067",
        "Could not create room.",
        true,
        ErrorCategory.CONTENT
    ),
    INVITE_FAILED(
        "E068",
        "Could not send invite.",
        true,
        ErrorCategory.CONTENT
    ),

    // Message Errors (E071-E080)
    MESSAGE_NOT_FOUND(
        "E071",
        "Message not found.",
        false,
        ErrorCategory.CONTENT
    ),
    MESSAGE_EDIT_FAILED(
        "E072",
        "Could not edit message.",
        true,
        ErrorCategory.CONTENT
    ),
    MESSAGE_DELETE_FAILED(
        "E073",
        "Could not delete message.",
        true,
        ErrorCategory.CONTENT
    ),
    MESSAGE_TOO_LONG(
        "E074",
        "Message is too long.",
        true,
        ErrorCategory.CONTENT
    ),
    MESSAGE_SEND_FAILED(
        "E075",
        "Could not send message.",
        true,
        ErrorCategory.CONTENT
    ),
    ATTACHMENT_TOO_LARGE(
        "E076",
        "Attachment is too large.",
        true,
        ErrorCategory.CONTENT
    ),
    UNSUPPORTED_FILE_TYPE(
        "E077",
        "File type not supported.",
        true,
        ErrorCategory.CONTENT
    ),
    REACTION_FAILED(
        "E078",
        "Could not add reaction.",
        true,
        ErrorCategory.CONTENT
    ),

    // Authentication Errors (E081-E090)
    LOGIN_FAILED(
        "E081",
        "Login failed. Please check your credentials.",
        true,
        ErrorCategory.AUTH
    ),
    INVALID_CREDENTIALS(
        "E082",
        "Invalid username or password.",
        true,
        ErrorCategory.AUTH
    ),
    ACCOUNT_LOCKED(
        "E083",
        "Account is locked. Please contact support.",
        false,
        ErrorCategory.AUTH
    ),
    PASSWORD_RESET_FAILED(
        "E084",
        "Password reset failed.",
        true,
        ErrorCategory.AUTH
    ),
    REGISTRATION_FAILED(
        "E085",
        "Registration failed.",
        true,
        ErrorCategory.AUTH
    ),
    TOKEN_REFRESH_FAILED(
        "E086",
        "Session refresh failed. Please log in again.",
        true,
        ErrorCategory.AUTH
    ),
    MFA_REQUIRED(
        "E087",
        "Two-factor authentication required.",
        true,
        ErrorCategory.AUTH
    ),
    MFA_FAILED(
        "E088",
        "Invalid verification code.",
        true,
        ErrorCategory.AUTH
    ),
    BIOMETRIC_FAILED(
        "E089",
        "Biometric authentication failed.",
        true,
        ErrorCategory.AUTH
    ),
    LOGOUT_FAILED(
        "E090",
        "Logout failed. Please try again.",
        true,
        ErrorCategory.AUTH
    ),

    // Generic Errors (E099)
    UNKNOWN_ERROR(
        "E099",
        "An unexpected error occurred.",
        true,
        ErrorCategory.UNKNOWN
    );

    companion object {
        fun fromCode(code: String): ArmorClawErrorCode {
            return entries.find { it.code == code } ?: UNKNOWN_ERROR
        }

        fun fromCategory(category: ErrorCategory): List<ArmorClawErrorCode> {
            return entries.filter { it.category == category }
        }
    }
}

@Serializable
enum class ErrorCategory {
    PERMISSION,
    NETWORK,
    TRUST,
    AUTH,
    ENCRYPTION,
    SYNC,
    CONTENT,
    STATE,
    HARDWARE,
    UNKNOWN
}

/**
 * Wrapper for errors with additional context
 */
@Serializable
data class ArmorClawError(
    val code: ArmorClawErrorCode,
    val details: String? = null,
    val timestamp: Long = System.currentTimeMillis(),
    val recoverableAction: RecoverableAction? = null
) {
    fun toDisplayString(): String {
        return if (details != null) {
            "${code.userMessage}\n$details"
        } else {
            code.userMessage
        }
    }

    fun isRecoverable(): Boolean = code.recoverable && recoverableAction != null
}

@Serializable
sealed class RecoverableAction {
    @Serializable
    data object Retry : RecoverableAction()

    @Serializable
    data object OpenSettings : RecoverableAction()

    @Serializable
    data object ReLogin : RecoverableAction()

    @Serializable
    data object VerifyDevice : RecoverableAction()

    @Serializable
    data object CheckNetwork : RecoverableAction()

    @Serializable
    data class Custom(val actionId: String, val label: String) : RecoverableAction()
}
