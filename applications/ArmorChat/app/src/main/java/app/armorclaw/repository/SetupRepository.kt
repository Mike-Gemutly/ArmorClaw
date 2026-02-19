package app.armorclaw.repository

import app.armorclaw.network.BridgeApi
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.security.MessageDigest

/**
 * Repository for setup and bonding operations
 */
class SetupRepository(private val api: BridgeApi = BridgeApi()) {

    /**
     * Get current lockdown status
     */
    suspend fun getLockdownStatus(): Result<BridgeApi.LockdownStatus> = withContext(Dispatchers.IO) {
        api.getLockdownStatus()
    }

    /**
     * Check if we can claim ownership
     */
    suspend fun canClaimOwnership(): Boolean {
        val status = api.getLockdownStatus().getOrNull() ?: return false
        return !status.admin_established &&
                (status.mode == "lockdown" || status.mode == "bonding")
    }

    /**
     * Start the claiming process
     */
    suspend fun startClaiming(
        displayName: String,
        deviceName: String,
        passphrase: String
    ): Result<ClaimingState> = withContext(Dispatchers.IO) {
        // Get a challenge first
        val challenge = api.getChallenge().getOrElse { error ->
            return@withContext Result.failure(error)
        }

        // Generate device fingerprint (simplified - in production use device-specific data)
        val deviceFingerprint = generateDeviceFingerprint(deviceName)

        // Generate passphrase commitment (argon2id hash in production, simplified here)
        val passphraseCommitment = hashPassphrase(passphrase)

        // Initiate claim
        val response = api.claimOwnership(
            displayName = displayName,
            deviceName = deviceName,
            deviceFingerprint = deviceFingerprint,
            passphraseCommitment = passphraseCommitment,
            challengeResponse = challenge.nonce
        ).getOrElse { error ->
            return@withContext Result.failure(error)
        }

        Result.success(ClaimingState.Success(
            adminId = response.admin_id,
            deviceId = response.device_id,
            sessionToken = response.session_token,
            nextStep = response.next_step
        ))
    }

    /**
     * Get security categories for configuration
     */
    suspend fun getSecurityCategories(): Result<List<BridgeApi.DataCategory>> = withContext(Dispatchers.IO) {
        api.getSecurityCategories()
    }

    /**
     * Set security category permission
     */
    suspend fun setCategoryPermission(category: String, permission: String): Result<Unit> = withContext(Dispatchers.IO) {
        api.setSecurityCategory(category, permission).map { }
    }

    /**
     * Complete setup and transition to operational mode
     */
    suspend fun completeSetup(): Result<Unit> = withContext(Dispatchers.IO) {
        api.transitionMode("operational").map { }
    }

    // Helper functions

    private fun generateDeviceFingerprint(deviceName: String): String {
        val data = "$deviceName:${System.currentTimeMillis()}"
        val bytes = MessageDigest.getInstance("SHA-256").digest(data.toByteArray())
        return bytes.joinToString("") { "%02x".format(it) }
    }

    private fun hashPassphrase(passphrase: String): String {
        // In production, use argon2id with proper salt
        val bytes = MessageDigest.getInstance("SHA-256").digest(passphrase.toByteArray())
        return bytes.joinToString("") { "%02x".format(it) }
    }
}

/**
 * State of the claiming process
 */
sealed class ClaimingState {
    data class Success(
        val adminId: String,
        val deviceId: String,
        val sessionToken: String,
        val nextStep: String
    ) : ClaimingState()

    data class Error(val message: String) : ClaimingState()
    object Loading : ClaimingState()
}
