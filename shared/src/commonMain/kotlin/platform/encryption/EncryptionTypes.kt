package com.armorclaw.shared.platform.encryption

/**
 * Encryption type definitions for ArmorChat
 *
 * NOTE: The EncryptionService class has been removed (v4.1.0-alpha01).
 * The Matrix Rust SDK now handles all encryption operations directly.
 * These types are retained for UI status display and room metadata.
 *
 * @see MatrixClient for encryption operations
 */

/**
 * Encryption mode determines where encryption/decryption happens
 *
 * ## Trust Models
 *
 * ### SERVER_SIDE (Legacy - Bridge handles E2EE)
 * - Bridge server handles Matrix E2EE (Megolm/Olm)
 * - **Deprecated**: Retained for migration status display only
 *
 * ### CLIENT_SIDE (Active - Matrix Rust SDK)
 * - Client directly encrypts/decrypts using Matrix Rust SDK
 * - True end-to-end encryption - only sender and recipients can read
 * - Keys stored in AndroidKeyStore
 *
 * ### NONE
 * - No encryption applied (development/debugging only)
 */
enum class EncryptionMode {
    /** No encryption - messages sent in plaintext (development only) */
    NONE,

    /** Server-side encryption - Bridge handles Matrix E2EE (deprecated) */
    SERVER_SIDE,

    /** Client-side encryption - Matrix Rust SDK (active) */
    CLIENT_SIDE
}

/**
 * Room encryption status
 */
sealed class RoomEncryptionStatus {
    /** Encryption not available for this room */
    object NotAvailable : RoomEncryptionStatus()

    /** Room is encrypted with verified devices */
    data class Encrypted(val verified: Boolean) : RoomEncryptionStatus()

    /**
     * Room is encrypted but some devices are unverified
     */
    data class EncryptedUnverified(
        val reason: String,
        val trustModel: String = "client-side"
    ) : RoomEncryptionStatus()
}

/**
 * Device verification status
 */
enum class DeviceVerificationStatus {
    /** Device is verified (cross-signed or manually) */
    VERIFIED,

    /** Device is unverified */
    UNVERIFIED,

    /** Verification is in progress */
    VERIFYING
}
