package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

// ========== Trust Levels ==========

/**
 * Trust level for devices and users
 * Used for encryption status and verification
 */
@Serializable
enum class TrustLevel {
    /** No verification performed */
    UNVERIFIED,
    /** Verified via cross-signing */
    CROSS_SIGNED,
    /** Manually verified (QR code, emoji comparison) */
    VERIFIED_IN_PERSON,
    /** Previously verified, no evidence of compromise */
    KNOWN_UNCOMPROMISED,
    /** Known to be compromised */
    COMPROMISED;

    fun isTrusted(): Boolean = this != UNVERIFIED && this != COMPROMISED

    fun requiresVerification(): Boolean = this == UNVERIFIED
}

// ========== Verification State ==========

/**
 * State machine for verification flow
 */
@Serializable
sealed class VerificationState {
    @Serializable
    object Unverified : VerificationState()

    @Serializable
    object Requested : VerificationState()

    @Serializable
    object Ready : VerificationState()

    @Serializable
    data class EmojiChallenge(
        val emojis: List<EmojiInfo>,
        val transactionId: String
    ) : VerificationState()

    @Serializable
    data class CodeChallenge(
        val code: String,
        val transactionId: String
    ) : VerificationState()

    @Serializable
    object Verifying : VerificationState()

    @Serializable
    object Verified : VerificationState()

    @Serializable
    data class Cancelled(
        val reason: String,
        val byMe: Boolean = false
    ) : VerificationState()

    @Serializable
    data class Failed(
        val reason: String,
        val errorCode: String? = null
    ) : VerificationState()

    fun isInProgress(): Boolean = when (this) {
        is Requested, is Ready, is EmojiChallenge, is CodeChallenge, is Verifying -> true
        else -> false
    }

    fun isComplete(): Boolean = this is Verified || this is Cancelled || this is Failed
}

// ========== Emoji Verification ==========

/**
 * Emoji information for verification
 */
@Serializable
data class EmojiInfo(
    val emoji: String,
    val description: String,
    val index: Int
) {
    companion object {
        /**
         * Standard verification emojis used by Matrix
         */
        val STANDARD_EMOJIS = listOf(
            EmojiInfo("🐶", "Dog", 0),
            EmojiInfo("🐱", "Cat", 1),
            EmojiInfo("🦁", "Lion", 2),
            EmojiInfo("🐎", "Horse", 3),
            EmojiInfo("🦄", "Unicorn", 4),
            EmojiInfo("🐷", "Pig", 5),
            EmojiInfo("🐘", "Elephant", 6),
            EmojiInfo("🐰", "Rabbit", 7),
            EmojiInfo("🐼", "Panda", 8),
            EmojiInfo("🐓", "Rooster", 9),
            EmojiInfo("🐧", "Penguin", 10),
            EmojiInfo("🐢", "Turtle", 11),
            EmojiInfo("🐟", "Fish", 12),
            EmojiInfo("🐙", "Octopus", 13),
            EmojiInfo("🦋", "Butterfly", 14),
            EmojiInfo("🌷", "Flower", 15),
            EmojiInfo("🌳", "Tree", 16),
            EmojiInfo("🌵", "Cactus", 17),
            EmojiInfo("🍄", "Mushroom", 18),
            EmojiInfo("🌏", "Globe", 19),
            EmojiInfo("🌙", "Moon", 20),
            EmojiInfo("☁️", "Cloud", 21),
            EmojiInfo("🔥", "Fire", 22),
            EmojiInfo("🍌", "Banana", 23),
            EmojiInfo("🍎", "Apple", 24),
            EmojiInfo("🍓", "Strawberry", 25),
            EmojiInfo("🌽", "Corn", 26),
            EmojiInfo("🍕", "Pizza", 27),
            EmojiInfo("🎂", "Cake", 28),
            EmojiInfo("❤️", "Heart", 29),
            EmojiInfo("😀", "Smiley", 30),
            EmojiInfo("🤖", "Robot", 31),
            EmojiInfo("🎩", "Hat", 32),
            EmojiInfo("👟", "Shoe", 33),
            EmojiInfo("💨", "Wind", 34),
            EmojiInfo("⛳", "Golf", 35),
            EmojiInfo("🚀", "Rocket", 36),
            EmojiInfo("🏆", "Trophy", 37),
            EmojiInfo("🔈", "Speaker", 38),
            EmojiInfo("🎸", "Guitar", 39),
            EmojiInfo("🎺", "Trumpet", 40),
            EmojiInfo("🔔", "Bell", 41),
            EmojiInfo("⚓", "Anchor", 42),
            EmojiInfo("💻", "Laptop", 43),
            EmojiInfo("💾", "Disk", 44),
            EmojiInfo("📷", "Camera", 45),
            EmojiInfo("🎥", "Video", 46),
            EmojiInfo("☎️", "Phone", 47),
            EmojiInfo("⏰", "Clock", 48),
            EmojiInfo("🎁", "Gift", 49),
            EmojiInfo("💡", "Light", 50),
            EmojiInfo("📖", "Book", 51),
            EmojiInfo("✏️", "Pencil", 52),
            EmojiInfo("📎", "Paperclip", 53),
            EmojiInfo("✂️", "Scissors", 54),
            EmojiInfo("🔑", "Key", 55),
            EmojiInfo("🔨", "Hammer", 56),
            EmojiInfo("🚿", "Shower", 57),
            EmojiInfo("🧲", "Magnet", 58),
            EmojiInfo("🛡️", "Shield", 59),
            EmojiInfo("⚽", "Soccer", 60),
            EmojiInfo("🎯", "Target", 61),
            EmojiInfo("🎸", "Guitar", 62),
            EmojiInfo("🚲", "Bicycle", 63)
        )
    }
}

// ========== Device Information ==========

/**
 * Information about a user's device
 */
@Serializable
data class DeviceInfo(
    val deviceId: String,
    val displayName: String?,
    val userId: String,
    val isVerified: Boolean = false,
    val trustLevel: TrustLevel = TrustLevel.UNVERIFIED,
    val lastSeenIp: String? = null,
    val lastSeenTimestamp: Long? = null,
    val isCurrentDevice: Boolean = false,
    val keys: DeviceKeys? = null
)

/**
 * Device keys for encryption
 */
@Serializable
data class DeviceKeys(
    val deviceId: String,
    val userId: String,
    val algorithms: List<String>,
    val keys: Map<String, String>,
    val signatures: Map<String, Map<String, String>>,
    val unsigned: Map<String, String>? = null
)

// ========== Cross-Signing ==========

/**
 * Cross-signing keys for user identity verification
 */
@Serializable
data class CrossSigningKeys(
    val masterKey: String,
    val selfSigningKey: String,
    val userSigningKey: String,
    val verifiedAt: Long? = null,
    val signatures: Map<String, Map<String, String>> = emptyMap()
)

/**
 * Cross-signing setup status
 */
@Serializable
enum class CrossSigningStatus {
    /** Cross-signing not set up */
    NOT_SETUP,
    /** Set up but not verified on this device */
    SETUP_UNVERIFIED,
    /** Set up and verified */
    VERIFIED,
    /** Keys need backup */
    NEEDS_BACKUP
}

// ========== Verification Transaction ==========

/**
 * Active verification transaction
 */
@Serializable
data class VerificationTransaction(
    val transactionId: String,
    val otherUserId: String,
    val otherDeviceId: String,
    val state: VerificationState,
    val startedAt: Long,
    val method: VerificationMethod,
    val cancellationReason: String? = null
)

/**
 * Verification method used
 */
@Serializable
enum class VerificationMethod {
    SAS,        // Short Authentication String (emoji/numbers)
    QR_CODE,    // QR code scanning
    RECIPROCATE // Reciprocal verification
}

// ========== Trust Actions ==========

/**
 * Actions that require trust verification
 */
@Serializable
enum class TrustAction {
    SETUP_CROSS_SIGNING,
    VERIFY_DEVICE,
    SIGN_DEVICE,
    VIEW_RECOVERY_KEY,
    RESET_KEYS,
    SEND_ENCRYPTED_MESSAGE,
    JOIN_ENCRYPTED_ROOM
}

// ========== Encryption Status ==========

/**
 * Overall encryption status for a room or conversation
 */
@Serializable
enum class EncryptionStatus {
    /** No encryption available */
    NONE,
    /** Messages not encrypted */
    UNENCRYPTED,
    /** Encrypted but keys not verified */
    UNVERIFIED,
    /** End-to-end encrypted and verified */
    VERIFIED,
    /** Cross-signed, highest trust level */
    CROSS_SIGNED;

    fun isEncrypted(): Boolean = this != NONE && this != UNENCRYPTED

    fun isTrusted(): Boolean = this == VERIFIED || this == CROSS_SIGNED
}

// ========== Key Backup ==========

/**
 * Key backup information
 */
@Serializable
data class KeyBackupInfo(
    val version: String,
    val algorithm: String,
    val authData: Map<String, String>,
    val count: Int,
    val etag: String
)

/**
 * Key backup status
 */
@Serializable
enum class KeyBackupStatus {
    /** No backup exists */
    NO_BACKUP,
    /** Backup exists but not verified */
    UNVERIFIED,
    /** Backup is enabled and verified */
    ENABLED,
    /** Backup needs to be created */
    REQUIRED
}

// ========== Session Information ==========

/**
 * Trust-related session information
 * Extends basic session with verification and cross-signing status
 */
@Serializable
data class TrustSessionInfo(
    val userId: String,
    val deviceId: String,
    val deviceIdVerified: Boolean = false,
    val crossSigningStatus: CrossSigningStatus = CrossSigningStatus.NOT_SETUP
)
