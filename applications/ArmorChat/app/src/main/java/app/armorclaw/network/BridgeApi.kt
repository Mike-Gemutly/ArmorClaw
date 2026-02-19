package app.armorclaw.network

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.TimeUnit

/**
 * Bridge API Client for ArmorChat
 *
 * Communicates with the ArmorClaw bridge via HTTP.
 * All requests are JSON-RPC 2.0.
 */
class BridgeApi(private val baseUrl: String = "http://localhost:8080/api") {

    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    private var requestId = 0

    //region Types

    @Serializable
    data class RPCRequest(
        val jsonrpc: String = "2.0",
        val id: Int,
        val method: String,
        val params: Map<String, String>? = null
    )

    @Serializable
    data class RPCResponse<T>(
        val jsonrpc: String,
        val id: Int,
        val result: T? = null,
        val error: RPCError? = null
    )

    @Serializable
    data class RPCError(
        val code: Int,
        val message: String,
        val data: String? = null
    )

    @Serializable
    data class LockdownStatus(
        val mode: String,
        val admin_established: Boolean,
        val single_device_mode: Boolean,
        val setup_complete: Boolean,
        val security_configured: Boolean,
        val keystore_initialized: Boolean
    )

    @Serializable
    data class Challenge(
        val nonce: String,
        val created_at: String,
        val expires_at: String
    )

    @Serializable
    data class BondingRequest(
        val display_name: String,
        val device_name: String,
        val device_fingerprint: String,
        val passphrase_commitment: String,
        val challenge_response: String? = null
    )

    @Serializable
    data class BondingResponse(
        val status: String,
        val admin_id: String,
        val device_id: String,
        val certificate: String,
        val session_token: String,
        val expires_at: String,
        val next_step: String
    )

    @Serializable
    data class DataCategory(
        val id: String,
        val name: String,
        val description: String,
        val risk_level: String,
        val permission: String,
        val allowed_websites: List<String>,
        val requires_approval: Boolean
    )

    @Serializable
    data class SecurityTier(
        val id: String,
        val name: String,
        val description: String
    )

    @Serializable
    data class Device(
        val id: String,
        val name: String,
        val type: String,
        val trust_state: String,
        val last_seen: String,
        val is_current: Boolean
    )

    @Serializable
    data class Invite(
        val id: String,
        val code: String,
        val role: String,
        val status: String,
        val created_at: String,
        val expires_at: String?
    )

    //endregion

    //region API Methods

    /**
     * Generic RPC call
     */
    private inline fun <reified T> rpc(method: String, params: Map<String, String>? = null): Result<T> {
        return try {
            requestId++
            val request = RPCRequest(id = requestId, method = method, params = params)
            val requestBody = json.encodeToString(RPCRequest.serializer(), request)
                .toRequestBody("application/json".toMediaType())

            val httpRequest = Request.Builder()
                .url(baseUrl)
                .post(requestBody)
                .build()

            val response = client.newCall(httpRequest).execute()
            if (!response.isSuccessful) {
                return Result.failure(Exception("HTTP ${response.code}: ${response.message}"))
            }

            val responseBody = response.body?.string()
                ?: return Result.failure(Exception("Empty response body"))

            val rpcResponse = json.decodeFromString<RPCResponse<T>>(responseBody)

            if (rpcResponse.error != null) {
                Result.failure(Exception(rpcResponse.error.message))
            } else if (rpcResponse.result != null) {
                Result.success(rpcResponse.result)
            } else {
                Result.failure(Exception("No result in response"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Get lockdown status
     */
    fun getLockdownStatus(): Result<LockdownStatus> {
        return rpc("lockdown.status")
    }

    /**
     * Get a bonding challenge
     */
    fun getChallenge(): Result<Challenge> {
        return rpc("lockdown.get_challenge")
    }

    /**
     * Claim ownership (admin bonding)
     */
    fun claimOwnership(
        displayName: String,
        deviceName: String,
        deviceFingerprint: String,
        passphraseCommitment: String,
        challengeResponse: String? = null
    ): Result<BondingResponse> {
        val params = mutableMapOf(
            "display_name" to displayName,
            "device_name" to deviceName,
            "device_fingerprint" to deviceFingerprint,
            "passphrase_commitment" to passphraseCommitment
        )
        challengeResponse?.let { params["challenge_response"] = it }
        return rpc("lockdown.claim_ownership", params)
    }

    /**
     * Transition lockdown mode
     */
    fun transitionMode(target: String): Result<Map<String, String>> {
        return rpc("lockdown.transition", mapOf("target" to target))
    }

    /**
     * Get security categories
     */
    fun getSecurityCategories(): Result<List<DataCategory>> {
        return rpc("security.get_categories")
    }

    /**
     * Set security category permission
     */
    fun setSecurityCategory(category: String, permission: String): Result<Map<String, Boolean>> {
        return rpc("security.set_category", mapOf(
            "category" to category,
            "permission" to permission
        ))
    }

    /**
     * Get security tiers
     */
    fun getSecurityTiers(): Result<List<SecurityTier>> {
        return rpc("security.get_tiers")
    }

    /**
     * Set security tier
     */
    fun setSecurityTier(tier: String): Result<Map<String, Boolean>> {
        return rpc("security.set_tier", mapOf("tier" to tier))
    }

    /**
     * List devices
     */
    fun listDevices(): Result<List<Device>> {
        return rpc("device.list")
    }

    /**
     * Approve device
     */
    fun approveDevice(deviceId: String, approvedBy: String): Result<Map<String, Boolean>> {
        return rpc("device.approve", mapOf(
            "device_id" to deviceId,
            "approved_by" to approvedBy
        ))
    }

    /**
     * Reject device
     */
    fun rejectDevice(deviceId: String, rejectedBy: String, reason: String): Result<Map<String, Boolean>> {
        return rpc("device.reject", mapOf(
            "device_id" to deviceId,
            "rejected_by" to rejectedBy,
            "reason" to reason
        ))
    }

    /**
     * Create invite
     */
    fun createInvite(role: String, expiration: String, maxUses: Int, createdBy: String): Result<Invite> {
        return rpc("invite.create", mapOf(
            "role" to role,
            "expiration" to expiration,
            "max_uses" to maxUses.toString(),
            "created_by" to createdBy
        ))
    }

    /**
     * List invites
     */
    fun listInvites(): Result<List<Invite>> {
        return rpc("invite.list")
    }

    /**
     * Revoke invite
     */
    fun revokeInvite(inviteId: String, revokedBy: String): Result<Map<String, Boolean>> {
        return rpc("invite.revoke", mapOf(
            "invite_id" to inviteId,
            "revoked_by" to revokedBy
        ))
    }

    /**
     * Generate setup QR
     */
    fun generateSetupQR(): Result<Map<String, String>> {
        return rpc("qr.generate_setup")
    }

    //region Push Notification Methods

    @Serializable
    data class PushTokenResponse(
        val success: Boolean,
        val message: String,
        val device_id: String? = null
    )

    /**
     * Register FCM push token with the bridge
     */
    fun registerPushToken(deviceId: String, token: String, platform: String): Result<PushTokenResponse> {
        return rpc("push.register_token", mapOf(
            "device_id" to deviceId,
            "token" to token,
            "platform" to platform
        ))
    }

    /**
     * Unregister FCM push token from the bridge
     */
    fun unregisterPushToken(deviceId: String, token: String): Result<PushTokenResponse> {
        return rpc("push.unregister_token", mapOf(
            "device_id" to deviceId,
            "token" to token
        ))
    }

    //endregion
}
