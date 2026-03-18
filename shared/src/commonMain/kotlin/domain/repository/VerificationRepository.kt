package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.*
import kotlinx.coroutines.flow.Flow

/**
 * Repository for device verification and trust management
 * Supports Matrix cross-signing and SAS verification (MSC1756)
 */
interface VerificationRepository {

    // ========== Device Management ==========

    /**
     * Get all devices for a user
     * @param userId The Matrix user ID
     * @return Result containing list of device info
     */
    suspend fun getDevices(userId: String): Result<List<DeviceInfo>>

    /**
     * Get a specific device
     * @param userId The Matrix user ID
     * @param deviceId The device ID
     * @return Result containing device info or null
     */
    suspend fun getDevice(userId: String, deviceId: String): Result<DeviceInfo?>

    /**
     * Get the current user's device ID
     * @return The current device ID or null if not available
     */
    suspend fun getCurrentDeviceId(): Result<String?>

    /**
     * Refresh device list from server
     * @param userId The Matrix user ID
     * @return Result indicating success or error
     */
    suspend fun refreshDevices(userId: String): Result<Unit>

    // ========== Cross-Signing Setup ==========

    /**
     * Check if cross-signing is set up for the current user
     * @return Cross-signing status
     */
    suspend fun getCrossSigningStatus(): Result<CrossSigningStatus>

    /**
     * Set up cross-signing for the current account
     * @param password Optional password for authentication (if required by server)
     * @return Result containing the cross-signing keys
     */
    suspend fun setupCrossSigning(password: String?): Result<CrossSigningKeys>

    /**
     * Verify cross-signing key with another device
     * @param masterKey The master cross-signing key to verify
     * @return Result indicating success or error
     */
    suspend fun verifyCrossSigningKey(masterKey: String): Result<Unit>

    /**
     * Get cross-signing keys for a user
     * @param userId The Matrix user ID
     * @return Result containing cross-signing keys or null
     */
    suspend fun getCrossSigningKeys(userId: String): Result<CrossSigningKeys?>

    // ========== Verification Requests ==========

    /**
     * Request verification with another user's device
     * @param userId The other user's Matrix ID
     * @param deviceId The device ID to verify
     * @return Result containing the transaction ID
     */
    suspend fun requestVerification(
        userId: String,
        deviceId: String
    ): Result<String>

    /**
     * Request verification using QR code
     * @param userId The other user's Matrix ID
     * @param deviceId The device ID to verify
     * @return Result containing QR code data
     */
    suspend fun requestQrVerification(
        userId: String,
        deviceId: String
    ): Result<String>

    /**
     * Accept an incoming verification request
     * @param transactionId The transaction ID from the request
     * @param method The verification method to use
     * @return Result indicating success or error
     */
    suspend fun acceptVerification(
        transactionId: String,
        method: VerificationMethod
    ): Result<Unit>

    /**
     * Cancel a verification request
     * @param transactionId The transaction ID
     * @param reason Optional reason for cancellation
     * @return Result indicating success or error
     */
    suspend fun cancelVerification(
        transactionId: String,
        reason: String? = null
    ): Result<Unit>

    // ========== SAS/Emoji Verification ==========

    /**
     * Start SAS (Short Authentication String) verification
     * @param transactionId The transaction ID
     * @return Result containing list of emoji information
     */
    suspend fun startSasVerification(transactionId: String): Result<List<EmojiInfo>>

    /**
     * Get the emoji challenge for a verification
     * @param transactionId The transaction ID
     * @return Result containing emoji info list
     */
    suspend fun getEmojiChallenge(transactionId: String): Result<List<EmojiInfo>>

    /**
     * Confirm whether the emojis match
     * @param transactionId The transaction ID
     * @param matches Whether the emojis match between devices
     * @return Result indicating success or error
     */
    suspend fun confirmEmojiMatch(
        transactionId: String,
        matches: Boolean
    ): Result<Unit>

    /**
     * Get verification code for numeric verification
     * @param transactionId The transaction ID
     * @return Result containing the verification code
     */
    suspend fun getVerificationCode(transactionId: String): Result<String>

    // ========== Verification State ==========

    /**
     * Observe the verification state for a transaction
     * @param transactionId The transaction ID
     * @return Flow emitting verification state updates
     */
    fun observeVerificationState(transactionId: String): Flow<VerificationState>

    /**
     * Get the current verification state
     * @param transactionId The transaction ID
     * @return Result containing current verification state
     */
    suspend fun getVerificationState(transactionId: String): Result<VerificationState>

    /**
     * Get all active verification transactions
     * @return Result containing list of active transactions
     */
    suspend fun getActiveVerifications(): Result<List<VerificationTransaction>>

    // ========== Trust Level Management ==========

    /**
     * Get the trust level for a device
     * @param userId The Matrix user ID
     * @param deviceId The device ID
     * @return Result containing the trust level
     */
    suspend fun getTrustLevel(
        userId: String,
        deviceId: String
    ): Result<TrustLevel>

    /**
     * Manually set trust level for a device
     * Use with caution - prefer verification flows
     * @param userId The Matrix user ID
     * @param deviceId The device ID
     * @param trustLevel The trust level to set
     * @return Result indicating success or error
     */
    suspend fun setTrustLevel(
        userId: String,
        deviceId: String,
        trustLevel: TrustLevel
    ): Result<Unit>

    /**
     * Mark a device as verified
     * @param userId The Matrix user ID
     * @param deviceId The device ID
     * @return Result indicating success or error
     */
    suspend fun verifyDevice(
        userId: String,
        deviceId: String
    ): Result<Unit>

    /**
     * Mark a device as compromised
     * @param userId The Matrix user ID
     * @param deviceId The device ID
     * @return Result indicating success or error
     */
    suspend fun markDeviceCompromised(
        userId: String,
        deviceId: String
    ): Result<Unit>

    // ========== Device Signing ==========

    /**
     * Sign another user's device with cross-signing key
     * @param userId The user ID of the device owner
     * @param deviceId The device ID to sign
     * @return Result indicating success or error
     */
    suspend fun signDevice(
        userId: String,
        deviceId: String
    ): Result<Unit>

    /**
     * Sign a user's master cross-signing key
     * @param userId The user ID
     * @param masterKey The master key to sign
     * @return Result indicating success or error
     */
    suspend fun signUserMasterKey(
        userId: String,
        masterKey: String
    ): Result<Unit>

    // ========== Key Backup ==========

    /**
     * Check key backup status
     * @return Result containing key backup status
     */
    suspend fun getKeyBackupStatus(): Result<KeyBackupStatus>

    /**
     * Get key backup information
     * @return Result containing key backup info or null
     */
    suspend fun getKeyBackupInfo(): Result<KeyBackupInfo?>

    // ========== Session Management ==========

    /**
     * Get current user session info
     * @return Result containing session info
     */
    suspend fun getCurrentSession(): Result<UserSession?>

    /**
     * Check if current session needs verification
     * @return Result containing boolean
     */
    suspend fun needsVerification(): Result<Boolean>

    /**
     * Logout other sessions
     * @param deviceIds List of device IDs to logout
     * @return Result indicating success or error
     */
    suspend fun logoutDevices(deviceIds: List<String>): Result<Unit>
}

/**
 * Verification-related operation types for offline sync
 */
enum class VerificationOperationType {
    REQUEST_VERIFICATION,
    ACCEPT_VERIFICATION,
    CANCEL_VERIFICATION,
    CONFIRM_EMOJI_MATCH,
    SET_TRUST_LEVEL,
    SIGN_DEVICE,
    SETUP_CROSS_SIGNING
}

/**
 * Exception for verification-related errors
 */
class VerificationException(
    val errorCode: ArmorClawErrorCode,
    message: String,
    cause: Throwable? = null
) : Exception(message, cause)
