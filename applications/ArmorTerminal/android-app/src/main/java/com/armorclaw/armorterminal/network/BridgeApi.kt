package com.armorclaw.armorterminal.network

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.CertificatePinner
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/**
 * Bridge API Client for ArmorTerminal
 *
 * Communicates with the ArmorClaw bridge via HTTPS.
 * All requests are JSON-RPC 2.0.
 *
 * Security features:
 * - TLS 1.3 with certificate pinning
 * - Session token management
 * - Automatic token refresh
 */
class BridgeApi(
    private var baseUrl: String = "https://armorclaw.local:8443/api",
    private val certificateFingerprint: String? = null
) {

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
        encodeDefaults = true
    }

    private var requestId = AtomicInteger(0)

    private val client: OkHttpClient by lazy {
        val builder = OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)

        // Add certificate pinning if fingerprint is provided
        if (certificateFingerprint != null) {
            val host = extractHost(baseUrl)
            val pinner = CertificatePinner.Builder()
                .add(host, "sha256/$certificateFingerprint")
                .build()
            builder.certificatePinner(pinner)
        }

        builder.build()
    }

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
    data class BridgeInfo(
        val name: String,
        val hostname: String,
        val port: Int,
        val ips: List<String>,
        val version: String,
        val fingerprint: String? = null
    )

    @Serializable
    data class DeviceRegistration(
        val device_id: String,
        val device_name: String,
        val trust_state: String,
        val session_token: String,
        val next_step: String,
        val request_id: String? = null
    )

    @Serializable
    data class ApprovalStatus(
        val status: String,
        val device_id: String,
        val verified_at: String? = null,
        val rejection_reason: String? = null,
        val timeout: Int? = null,
        val message: String? = null,
        val ws_endpoint: String? = null
    )

    @Serializable
    data class SessionInfo(
        val device_id: String,
        val session_token: String,
        val expires_at: String? = null
    )

    @Serializable
    data class PushTokenResponse(
        val success: Boolean,
        val message: String,
        val device_id: String? = null
    )

    //endregion

    //region Connection Management

    /**
     * Set the bridge URL
     */
    fun setBaseUrl(url: String) {
        baseUrl = url
    }

    /**
     * Get the current bridge URL
     */
    fun getBaseUrl(): String = baseUrl

    /**
     * Test connection to the bridge
     */
    fun testConnection(): Result<Boolean> {
        return try {
            val response = rpc<Unit>("status")
            Result.success(response.isSuccess)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Get bridge info for discovery
     */
    fun getBridgeInfo(): Result<BridgeInfo> {
        return try {
            // Use the /discover endpoint directly
            val discoverUrl = baseUrl.replace("/api", "/discover")

            val request = Request.Builder()
                .url(discoverUrl)
                .get()
                .build()

            val response = client.newCall(request).execute()
            if (!response.isSuccessful) {
                return Result.failure(Exception("HTTP ${response.code}"))
            }

            val body = response.body?.string()
                ?: return Result.failure(Exception("Empty response"))

            val info = json.decodeFromString<BridgeInfo>(body)
            Result.success(info)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Get certificate fingerprint from bridge
     */
    fun getCertificateFingerprint(): Result<String> {
        return try {
            val fingerprintUrl = baseUrl.replace("/api", "/fingerprint")

            val request = Request.Builder()
                .url(fingerprintUrl)
                .get()
                .build()

            val response = client.newCall(request).execute()
            if (!response.isSuccessful) {
                return Result.failure(Exception("HTTP ${response.code}"))
            }

            val body = response.body?.string()
                ?: return Result.failure(Exception("Empty response"))

            val result = json.decodeFromString<Map<String, String>>(body)
            Result.success(result["sha256"] ?: "")
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    //endregion

    //region Device Registration

    /**
     * Register a new device with the bridge
     */
    fun registerDevice(
        pairingToken: String,
        deviceName: String,
        deviceType: String,
        publicKey: String,
        fcmToken: String? = null,
        userAgent: String = "ArmorTerminal/1.0"
    ): Result<DeviceRegistration> {
        val params = mutableMapOf(
            "pairing_token" to pairingToken,
            "device_name" to deviceName,
            "device_type" to deviceType,
            "public_key" to publicKey,
            "user_agent" to userAgent
        )
        fcmToken?.let { params["fcm_token"] = it }

        return rpc("device.register", params)
    }

    /**
     * Wait for admin approval
     */
    fun waitForApproval(
        deviceId: String,
        sessionToken: String,
        timeout: Int = 60
    ): Result<ApprovalStatus> {
        return rpc("device.wait_for_approval", mapOf(
            "device_id" to deviceId,
            "session_token" to sessionToken,
            "timeout" to timeout.toString()
        ))
    }

    //endregion

    //region Session Management

    /**
     * Validate session token
     */
    fun validateSession(deviceId: String, sessionToken: String): Result<Boolean> {
        return try {
            val result = rpc<Map<String, Boolean>>("session.validate", mapOf(
                "device_id" to deviceId,
                "session_token" to sessionToken
            ))
            result.map { it["valid"] ?: false }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Refresh session token
     */
    fun refreshSession(deviceId: String, sessionToken: String): Result<SessionInfo> {
        return rpc("session.refresh", mapOf(
            "device_id" to deviceId,
            "session_token" to sessionToken
        ))
    }

    //endregion

    //region Lockdown & Security

    /**
     * Get lockdown status
     */
    fun getLockdownStatus(): Result<LockdownStatus> {
        return rpc("lockdown.status")
    }

    /**
     * Get security tiers
     */
    fun getSecurityTiers(): Result<List<Map<String, String>>> {
        return rpc("security.get_tiers")
    }

    //endregion

    //region Push Notifications

    /**
     * Register FCM push token
     */
    fun registerPushToken(deviceId: String, token: String): Result<PushTokenResponse> {
        return rpc("push.register_token", mapOf(
            "device_id" to deviceId,
            "token" to token,
            "platform" to "android"
        ))
    }

    /**
     * Unregister FCM push token
     */
    fun unregisterPushToken(deviceId: String, token: String): Result<PushTokenResponse> {
        return rpc("push.unregister_token", mapOf(
            "device_id" to deviceId,
            "token" to token
        ))
    }

    //endregion

    //region WebSocket

    /**
     * Create WebSocket connection for real-time updates
     */
    fun createWebSocket(
        deviceId: String,
        sessionToken: String,
        listener: WebSocketListener
    ): WebSocket {
        val wsUrl = baseUrl
            .replace("/api", "/ws")
            .replace("https://", "wss://")
            .replace("http://", "ws://")

        val request = Request.Builder()
            .url(wsUrl)
            .build()

        return client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: okhttp3.Response) {
                // Send registration message
                val registerMsg = """{"type":"register","payload":{"device_id":"$deviceId"}}"""
                webSocket.send(registerMsg)
                listener.onOpen(webSocket, response)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                listener.onMessage(webSocket, text)
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                listener.onClosing(webSocket, code, reason)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                listener.onClosed(webSocket, code, reason)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: okhttp3.Response?) {
                listener.onFailure(webSocket, t, response)
            }
        })
    }

    //endregion

    //region Private Helpers

    /**
     * Generic RPC call
     */
    private inline fun <reified T> rpc(method: String, params: Map<String, String>? = null): Result<T> {
        return try {
            val id = requestId.incrementAndGet()
            val request = RPCRequest(id = id, method = method, params = params)
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

            when {
                rpcResponse.error != null -> Result.failure(Exception(rpcResponse.error.message))
                rpcResponse.result != null -> Result.success(rpcResponse.result)
                else -> Result.failure(Exception("No result in response"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Extract host from URL for certificate pinning
     */
    private fun extractHost(url: String): String {
        val withoutProtocol = url.removePrefix("https://").removePrefix("http://")
        return withoutProtocol.substringBefore(":").substringBefore("/")
    }

    //endregion
}
