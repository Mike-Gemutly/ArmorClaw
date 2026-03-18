package com.armorclaw.shared.platform.bridge

import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import kotlinx.coroutines.flow.*
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlin.time.Duration
import kotlin.time.Duration.Companion.hours
import kotlin.time.Duration.Companion.days
import kotlinx.serialization.*
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.*
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

/**
 * Service for creating and managing invite links
 *
 * Admins can generate time-limited, signed URLs that contain server configuration
 * for easy onboarding of new users.
 *
 * Features:
 * - Time-limited expiration (1 hour to 30 days)
 * - Cryptographic signature to prevent tampering
 * - Optional usage limits
 * - Server configuration embedded in link
 */
class InviteService(
    private val rpcClient: BridgeRpcClient,
    private val config: BridgeConfig
) {
    private val logger = repositoryLogger("InviteService", LogTag.Network.Bridge)
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    // Invite state
    private val _inviteLinks = MutableStateFlow<List<InviteLink>>(emptyList())
    val inviteLinks: StateFlow<List<InviteLink>> = _inviteLinks.asStateFlow()

    private val _generatedLink = MutableStateFlow<String?>(null)
    val generatedLink: StateFlow<String?> = _generatedLink.asStateFlow()

    /**
     * Generate a new invite link
     *
     * @param serverConfig Server configuration to embed
     * @param expiration How long the link is valid
     * @param maxUses Maximum number of times the link can be used (null = unlimited)
     * @param createdBy User ID of the admin creating the link
     * @param context Operation context for tracing
     * @return The generated invite URL
     */
    suspend fun generateInviteLink(
        serverConfig: ServerInviteConfig,
        expiration: InviteExpiration = InviteExpiration.SEVEN_DAYS,
        maxUses: Int? = null,
        createdBy: String,
        context: OperationContext? = null
    ): InviteResult {
        val ctx = context ?: OperationContext.create()

        logger.logOperationStart("generateInviteLink", mapOf(
            "expiration" to expiration.name,
            "max_uses" to (maxUses ?: "unlimited"),
            "created_by" to createdBy,
            "correlation_id" to ctx.correlationId
        ))

        return try {
            val now = Clock.System.now()
            val expiresAt = now.plus(expiration.duration)

            // Create invite payload
            val invite = InviteLink(
                id = generateInviteId(),
                serverConfig = serverConfig,
                createdAt = now,
                expiresAt = expiresAt,
                maxUses = maxUses,
                currentUses = 0,
                createdBy = createdBy,
                isActive = true
            )

            // Generate signed URL
            val signedUrl = createSignedInviteUrl(invite)

            // Store locally
            _inviteLinks.value = _inviteLinks.value + invite
            _generatedLink.value = signedUrl

            logger.logOperationSuccess("generateInviteLink", "invite_id=${invite.id}, expires_at=$expiresAt")

            InviteResult.Success(invite, signedUrl)
        } catch (e: Exception) {
            logger.logOperationError("generateInviteLink", e)
            InviteResult.Error("Failed to generate invite link: ${e.message}")
        }
    }

    /**
     * Parse and validate an invite URL
     *
     * @param inviteUrl The invite URL to parse
     * @param context Operation context for tracing
     * @return Parsed invite data if valid
     */
    fun parseInviteUrl(
        inviteUrl: String,
        context: OperationContext? = null
    ): InviteParseResult {
        val ctx = context ?: OperationContext.create()

        logger.logOperationStart("parseInviteUrl", mapOf(
            "correlation_id" to ctx.correlationId
        ))

        return try {
            // Extract the invite data from URL
            val inviteData = extractInviteDataFromUrl(inviteUrl)
                ?: return InviteParseResult.Error("Invalid invite URL format")

            // Verify signature
            if (!verifyInviteSignature(inviteData)) {
                logger.logOperationError("parseInviteUrl", Exception("Invalid signature"))
                return InviteParseResult.Error("Invite link has been tampered with")
            }

            // Parse the invite
            val invite = json.decodeFromString<InviteLink>(inviteData.payload)

            // Check expiration
            val now = Clock.System.now()
            if (now > invite.expiresAt) {
                return InviteParseResult.Expired(invite)
            }

            // Check usage limit
            if (invite.maxUses != null && invite.currentUses >= invite.maxUses) {
                return InviteParseResult.Exhausted(invite)
            }

            // Check if active
            if (!invite.isActive) {
                return InviteParseResult.Revoked(invite)
            }

            logger.logOperationSuccess("parseInviteUrl", "invite_id=${invite.id}")

            InviteParseResult.Valid(invite)
        } catch (e: Exception) {
            logger.logOperationError("parseInviteUrl", e)
            InviteParseResult.Error("Failed to parse invite: ${e.message}")
        }
    }

    /**
     * Revoke an invite link
     *
     * @param inviteId The invite ID to revoke
     * @param context Operation context for tracing
     */
    suspend fun revokeInviteLink(
        inviteId: String,
        context: OperationContext? = null
    ): InviteResult {
        val ctx = context ?: OperationContext.create()

        logger.logOperationStart("revokeInviteLink", mapOf(
            "invite_id" to inviteId,
            "correlation_id" to ctx.correlationId
        ))

        return try {
            val links = _inviteLinks.value.toMutableList()
            val index = links.indexOfFirst { it.id == inviteId }

            if (index >= 0) {
                links[index] = links[index].copy(isActive = false)
                _inviteLinks.value = links

                logger.logOperationSuccess("revokeInviteLink")
                InviteResult.Success(links[index], null)
            } else {
                InviteResult.Error("Invite not found")
            }
        } catch (e: Exception) {
            logger.logOperationError("revokeInviteLink", e)
            InviteResult.Error("Failed to revoke invite: ${e.message}")
        }
    }

    /**
     * Record usage of an invite link
     *
     * @param inviteId The invite ID that was used
     * @param context Operation context for tracing
     */
    suspend fun recordInviteUsage(
        inviteId: String,
        context: OperationContext? = null
    ) {
        val links = _inviteLinks.value.toMutableList()
        val index = links.indexOfFirst { it.id == inviteId }

        if (index >= 0) {
            links[index] = links[index].copy(currentUses = links[index].currentUses + 1)
            _inviteLinks.value = links
        }
    }

    /**
     * Get all invite links created by an admin
     */
    fun getInvitesByCreator(adminId: String): List<InviteLink> {
        return _inviteLinks.value.filter { it.createdBy == adminId }
    }

    /**
     * Clear the generated link
     */
    fun clearGeneratedLink() {
        _generatedLink.value = null
    }

    // Private implementation

    @OptIn(ExperimentalEncodingApi::class)
    private fun createSignedInviteUrl(invite: InviteLink): String {
        // Serialize the invite
        val payload = json.encodeToString(invite)

        // Create signature
        val signature = signPayload(payload)

        // Create invite data
        val inviteData = InviteData(
            version = INVITE_VERSION,
            payload = payload,
            signature = signature,
            createdAt = Clock.System.now()
        )

        // Encode to base64
        val encodedData = Base64.UrlSafe.encode(
            json.encodeToString(inviteData).encodeToByteArray()
        )

        // Create URL
        return "${config.baseUrl}/invite/$encodedData"
    }

    @OptIn(ExperimentalEncodingApi::class)
    private fun extractInviteDataFromUrl(url: String): InviteData? {
        return try {
            // Extract the base64 data from URL
            val encodedData = url.substringAfterLast("/invite/")
                .substringBefore("?")
                .substringBefore("#")

            // Decode from base64
            val jsonBytes = Base64.UrlSafe.decode(encodedData)
            val jsonData = jsonBytes.decodeToString()

            // Parse invite data
            json.decodeFromString(jsonData)
        } catch (e: Exception) {
            null
        }
    }

    private fun verifyInviteSignature(inviteData: InviteData): Boolean {
        return try {
            val expectedSignature = signPayload(inviteData.payload)
            expectedSignature == inviteData.signature
        } catch (e: Exception) {
            false
        }
    }

    /**
     * Sign a payload using HMAC-SHA256 (software implementation)
     *
     * Uses a double-pass keyed hash to approximate HMAC construction:
     * HMAC(key, message) = H((key XOR opad) || H((key XOR ipad) || message))
     *
     * The signing key is derived from the bridge config base URL to tie
     * signatures to a specific server instance.
     */
    private fun signPayload(payload: String): String {
        val key = deriveSigningKey()
        val message = payload.encodeToByteArray()

        // HMAC-like construction using two-pass keyed hashing
        // Pad key to 64 bytes (SHA-256 block size)
        val paddedKey = if (key.size > 64) {
            sha256(key)
        } else {
            key.copyOf(64)
        }

        // Inner and outer padding
        val ipad = ByteArray(64) { (paddedKey[it].toInt() xor 0x36).toByte() }
        val opad = ByteArray(64) { (paddedKey[it].toInt() xor 0x5C).toByte() }

        // Inner hash: H(ipad || message)
        val innerHash = sha256(ipad + message)

        // Outer hash: H(opad || innerHash)
        val hmac = sha256(opad + innerHash)

        return hmac.joinToString("") { "%02x".format(it) }
    }

    /**
     * Derive a signing key from the bridge configuration.
     * In production, this should use a proper server-side secret.
     */
    private fun deriveSigningKey(): ByteArray {
        val material = "armorclaw-invite-signing:${config.baseUrl}"
        return sha256(material.encodeToByteArray())
    }

    /**
     * Software SHA-256 implementation (cross-platform)
     *
     * This is a pure-Kotlin SHA-256 for invite signing only.
     * For high-throughput cryptographic operations, use platform-native APIs.
     */
    private fun sha256(input: ByteArray): ByteArray {
        // SHA-256 initial hash values
        val h = intArrayOf(
            0x6a09e667.toInt(), 0xbb67ae85.toInt(), 0x3c6ef372, 0xa54ff53a.toInt(),
            0x510e527f, 0x9b05688c.toInt(), 0x1f83d9ab, 0x5be0cd19
        )

        // SHA-256 round constants
        val k = intArrayOf(
            0x428a2f98, 0x71374491, -0x4a3f0431, -0x164a245b,
            0x3956c25b, 0x59f111f1, -0x6dc07d5c, -0x54e3a12b,
            -0x27f85568, 0x12835b01, 0x243185be, 0x550c7dc3,
            0x72be5d74, -0x7f214e02, -0x6423f959, -0x3e640e8c,
            -0x1b64963f, -0x1041b87a, 0x0fc19dc6, 0x240ca1cc,
            0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
            -0x67c1aeae, -0x57ce3993, -0x4ffcd838, -0x40a68039,
            -0x391ff40d, -0x2a586eb9, 0x06ca6351, 0x14292967,
            0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
            0x650a7354, 0x766a0abb, -0x7e3d36d2, -0x6d8dd37b,
            -0x5d40175f, -0x57e599b5, -0x3db47490, -0x3893ae5d,
            -0x2e6d17e7, -0x2966f9dc, -0x0bf1ca7b, 0x106aa070,
            0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
            0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
            0x748f82ee, 0x78a5636f, -0x7b3787ec, -0x7338fdf8,
            -0x6f410006, -0x5baf9315, -0x41065c09, -0x398e870e
        )

        fun Int.rightRotate(n: Int): Int = (this ushr n) or (this shl (32 - n))

        // Pre-processing: pad message
        val bitLen = input.size.toLong() * 8
        val padded = input.toMutableList()
        padded.add(0x80.toByte())
        while ((padded.size % 64) != 56) padded.add(0)
        for (i in 7 downTo 0) padded.add((bitLen ushr (i * 8)).toByte())

        val blocks = padded.toByteArray()

        // Process each 512-bit block
        for (blockStart in blocks.indices step 64) {
            val w = IntArray(64)
            for (i in 0 until 16) {
                w[i] = ((blocks[blockStart + i * 4].toInt() and 0xFF) shl 24) or
                        ((blocks[blockStart + i * 4 + 1].toInt() and 0xFF) shl 16) or
                        ((blocks[blockStart + i * 4 + 2].toInt() and 0xFF) shl 8) or
                        (blocks[blockStart + i * 4 + 3].toInt() and 0xFF)
            }
            for (i in 16 until 64) {
                val s0 = w[i - 15].rightRotate(7) xor w[i - 15].rightRotate(18) xor (w[i - 15] ushr 3)
                val s1 = w[i - 2].rightRotate(17) xor w[i - 2].rightRotate(19) xor (w[i - 2] ushr 10)
                w[i] = w[i - 16] + s0 + w[i - 7] + s1
            }

            var a = h[0]; var b = h[1]; var c = h[2]; var d = h[3]
            var e = h[4]; var f = h[5]; var g = h[6]; var hh = h[7]

            for (i in 0 until 64) {
                val s1 = e.rightRotate(6) xor e.rightRotate(11) xor e.rightRotate(25)
                val ch = (e and f) xor (e.inv() and g)
                val temp1 = hh + s1 + ch + k[i] + w[i]
                val s0 = a.rightRotate(2) xor a.rightRotate(13) xor a.rightRotate(22)
                val maj = (a and b) xor (a and c) xor (b and c)
                val temp2 = s0 + maj

                hh = g; g = f; f = e; e = d + temp1
                d = c; c = b; b = a; a = temp1 + temp2
            }

            h[0] += a; h[1] += b; h[2] += c; h[3] += d
            h[4] += e; h[5] += f; h[6] += g; h[7] += hh
        }

        // Produce final hash
        val result = ByteArray(32)
        for (i in 0 until 8) {
            result[i * 4] = (h[i] ushr 24).toByte()
            result[i * 4 + 1] = (h[i] ushr 16).toByte()
            result[i * 4 + 2] = (h[i] ushr 8).toByte()
            result[i * 4 + 3] = h[i].toByte()
        }
        return result
    }

    private fun generateInviteId(): String {
        val timestamp = Clock.System.now().toEpochMilliseconds().toString(36)
        val random = (0 until 8).map {
            "abcdefghijklmnopqrstuvwxyz0123456789".random()
        }.joinToString("")
        return "invite_${timestamp}_$random"
    }

    companion object {
        private const val INVITE_VERSION = 1
    }
}

// Data classes

@Serializable
data class InviteLink(
    val id: String,
    val serverConfig: ServerInviteConfig,
    val createdAt: Instant,
    val expiresAt: Instant,
    val maxUses: Int?,
    val currentUses: Int,
    val createdBy: String,
    val isActive: Boolean
) {
    val isExpired: Boolean
        get() = Clock.System.now() > expiresAt

    val isExhausted: Boolean
        get() = maxUses != null && currentUses >= maxUses

    val remainingUses: Int?
        get() = maxUses?.let { it - currentUses }

    val timeRemaining: kotlin.time.Duration
        get() = (expiresAt - Clock.System.now()).coerceAtLeast(kotlin.time.Duration.ZERO)
}

@Serializable
data class ServerInviteConfig(
    val homeserver: String,
    val bridgeUrl: String,
    val serverName: String? = null,
    val serverRegion: String? = null,
    val serverDescription: String? = null,
    val requiresAdminApproval: Boolean = false,
    val autoJoinRooms: List<String>? = null,
    val welcomeMessage: String? = null,
    val serverLogo: String? = null,
    val features: ServerFeatures? = null
)

@Serializable
data class ServerFeatures(
    val e2ee: Boolean = true,
    val voice: Boolean = true,
    val video: Boolean = true,
    val fileSharing: Boolean = true,
    val reactions: Boolean = true,
    val threads: Boolean = true,
    val platforms: List<String>? = null
)

@Serializable
data class InviteData(
    val version: Int,
    val payload: String,
    val signature: String,
    val createdAt: Instant
)

enum class InviteExpiration(val duration: kotlin.time.Duration) {
    ONE_HOUR(1.hours),
    SIX_HOURS(6.hours),
    ONE_DAY(1.days),
    THREE_DAYS(3.days),
    SEVEN_DAYS(7.days),
    FOURTEEN_DAYS(14.days),
    THIRTY_DAYS(30.days)
}

sealed class InviteResult {
    data class Success(
        val invite: InviteLink,
        val url: String?
    ) : InviteResult()

    data class Error(val message: String) : InviteResult()

    val isSuccess: Boolean get() = this is Success
    val isError: Boolean get() = this is Error
}

sealed class InviteParseResult {
    data class Valid(val invite: InviteLink) : InviteParseResult()
    data class Expired(val invite: InviteLink) : InviteParseResult()
    data class Exhausted(val invite: InviteLink) : InviteParseResult()
    data class Revoked(val invite: InviteLink) : InviteParseResult()
    data class Error(val message: String) : InviteParseResult()

    val isValid: Boolean get() = this is Valid
    val isInvalid: Boolean get() = this !is Valid
}
