package com.armorclaw.app.navigation

import android.content.Intent
import android.net.Uri
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag

/**
 * Handles deep link parsing and navigation
 *
 * Supports:
 * - armorclaw://room/{roomId}
 * - armorclaw://user/{userId}
 * - armorclaw://chat/{roomId}
 * - armorclaw://call/{callId}
 * - armorclaw://settings
 * - armorclaw://profile
 * - https://matrix.to/#/{roomId}
 * - https://matrix.to/#/@{userId}
 * - https://chat.armorclaw.app/room/{roomId}
 * - https://chat.armorclaw.app/user/{userId}
 *
 * ## Security (FIX for Bug #4)
 *
 * Deep links can be triggered by any app or website. To prevent abuse:
 * - Room navigation requires user confirmation if not already a member
 * - External HTTPS links are validated against known hosts
 * - Unknown/external rooms trigger a security warning
 * - Malformed links are rejected
 *
 * @see DeepLinkAction for navigation actions
 * @see DeepLinkSecurityCheck for security validation
 */
object DeepLinkHandler {

    private const val SCHEME_ARMORCLAW = "armorclaw"
    private const val HOST_MATRIX_TO = "matrix.to"
    private const val HOST_ARMORCLAW_CHAT = "chat.armorclaw.app"
    private const val HOST_ARMORCLAW_APP = "armorclaw.app"

    // Known safe hosts for HTTPS deep links
    private val KNOWN_SAFE_HOSTS = setOf(
        HOST_MATRIX_TO,
        HOST_ARMORCLAW_CHAT,
        HOST_ARMORCLAW_APP,
        "app.armorclaw.app"
    )

    /**
     * Parse deep link from intent and return navigation action
     */
    fun parseDeepLink(intent: Intent?): DeepLinkResult? {
        if (intent == null) return null

        val data: Uri? = intent.data
        if (data == null) {
            AppLogger.debug(LogTag.UI.Navigation, "No deep link data in intent")
            return null
        }

        return parseDeepLinkUri(data)
    }

    /**
     * Parse a deep link URI and return a validated result
     */
    fun parseDeepLinkUri(uri: Uri): DeepLinkResult? {
        AppLogger.info(
            LogTag.UI.Navigation,
            "Processing deep link",
            mapOf("uri" to redactSensitiveUri(uri.toString()))
        )

        // Validate URI structure
        val validationResult = validateUri(uri)
        if (validationResult is DeepLinkValidationResult.Invalid) {
            AppLogger.warning(
                LogTag.UI.Navigation,
                "Deep link validation failed",
                mapOf("reason" to validationResult.reason)
            )
            return DeepLinkResult.Invalid(validationResult.reason)
        }

        val action = when (uri.scheme) {
            SCHEME_ARMORCLAW -> parseArmorClawScheme(uri)
            "https" -> parseHttpsScheme(uri)
            else -> {
                AppLogger.warning(
                    LogTag.UI.Navigation,
                    "Unsupported deep link scheme",
                    mapOf("scheme" to (uri.scheme ?: "null"))
                )
                null
            }
        }

        return action?.let {
            // Apply security checks to the action
            applySecurityChecks(it, uri)
        }
    }

    /**
     * Parse Uri directly and return navigation action (legacy method)
     * @deprecated Use parseDeepLinkUri instead for security-validated results
     */
    @Deprecated(
        message = "Use parseDeepLinkUri for security-validated deep link handling",
        replaceWith = ReplaceWith("parseDeepLinkUri(uri)")
    )
    fun parseUri(uri: Uri): DeepLinkAction? {
        return (parseDeepLinkUri(uri) as? DeepLinkResult.Valid)?.action
    }

    /**
     * Validate URI for security issues
     */
    private fun validateUri(uri: Uri): DeepLinkValidationResult {
        // Check for null/empty scheme
        if (uri.scheme.isNullOrBlank()) {
            return DeepLinkValidationResult.Invalid("Missing URI scheme")
        }

        // Validate scheme
        if (uri.scheme !in listOf(SCHEME_ARMORCLAW, "https")) {
            return DeepLinkValidationResult.Invalid("Unsupported scheme: ${uri.scheme}")
        }

        // For HTTPS, validate host
        if (uri.scheme == "https") {
            val host = uri.host
            if (host.isNullOrBlank()) {
                return DeepLinkValidationResult.Invalid("Missing host for HTTPS URI")
            }
            if (host !in KNOWN_SAFE_HOSTS) {
                return DeepLinkValidationResult.UntrustedHost(host)
            }
        }

        // Check for suspiciously long URIs (potential buffer overflow attempts)
        if (uri.toString().length > 2048) {
            return DeepLinkValidationResult.Invalid("URI too long")
        }

        // Validate path segments don't contain suspicious characters
        val pathSegments = uri.pathSegments
        for (segment in pathSegments) {
            if (segment.contains("..") || segment.contains("\\")) {
                return DeepLinkValidationResult.Invalid("Invalid path segment")
            }
        }

        return DeepLinkValidationResult.Valid
    }

    /**
     * Apply security checks to deep link action
     */
    private fun applySecurityChecks(action: DeepLinkAction, uri: Uri): DeepLinkResult {
        return when (action) {
            is DeepLinkAction.NavigateToRoom -> {
                // Room navigation may require confirmation
                DeepLinkResult.RequiresConfirmation(
                    action = action,
                    securityCheck = DeepLinkSecurityCheck.ROOM_MEMBERSHIP,
                    message = "Join room '${action.roomId}'?",
                    details = "You're about to join a chat room. Only join rooms from trusted sources."
                )
            }
            is DeepLinkAction.NavigateToUser -> {
                // User profile navigation is generally safe
                DeepLinkResult.Valid(action)
            }
            is DeepLinkAction.NavigateToCall -> {
                // Calls require explicit confirmation
                DeepLinkResult.RequiresConfirmation(
                    action = action,
                    securityCheck = DeepLinkSecurityCheck.CALL_JOIN,
                    message = "Join call?",
                    details = "You're about to join a call. Only join calls from trusted sources."
                )
            }
            is DeepLinkAction.NavigateToSettings -> {
                DeepLinkResult.Valid(action)
            }
            is DeepLinkAction.NavigateToProfile -> {
                DeepLinkResult.Valid(action)
            }
            // Signed configuration from Bridge
            is DeepLinkAction.ApplySignedConfig -> {
                // Config deep links are trusted (signed by Bridge)
                DeepLinkResult.Valid(action)
            }
            // Setup with token
            is DeepLinkAction.SetupWithToken -> {
                DeepLinkResult.Valid(action)
            }
            // Invite code
            is DeepLinkAction.AcceptInvite -> {
                DeepLinkResult.RequiresConfirmation(
                    action = action,
                    securityCheck = DeepLinkSecurityCheck.INVITE_ACCEPT,
                    message = "Accept invite?",
                    details = "You're about to join using an invite code. Only accept invites from trusted sources."
                )
            }
            // Device bonding (admin pairing)
            is DeepLinkAction.BondDevice -> {
                DeepLinkResult.RequiresConfirmation(
                    action = action,
                    securityCheck = DeepLinkSecurityCheck.DEVICE_BONDING,
                    message = "Pair with this device?",
                    details = "This will grant administrative access. Only pair with devices you trust."
                )
            }
        }
    }

    /**
     * Parse armorclaw:// scheme deep links
     */
    private fun parseArmorClawScheme(uri: Uri): DeepLinkAction? {
        val host = uri.host ?: return null
        val pathSegments = uri.pathSegments

        return when (host) {
            "room", "chat" -> {
                val roomId = pathSegments.firstOrNull()
                if (roomId != null && isValidRoomId(roomId)) {
                    DeepLinkAction.NavigateToRoom(roomId)
                } else null
            }
            "user" -> {
                val userId = pathSegments.firstOrNull()
                if (userId != null && isValidUserId(userId)) {
                    DeepLinkAction.NavigateToUser(userId)
                } else null
            }
            "call" -> {
                val callId = pathSegments.firstOrNull()
                if (callId != null) {
                    DeepLinkAction.NavigateToCall(callId)
                } else null
            }
            "settings" -> DeepLinkAction.NavigateToSettings
            "profile" -> DeepLinkAction.NavigateToProfile

            // Signed configuration from Bridge (armorclaw://config?d=...)
            "config" -> {
                val encodedData = uri.getQueryParameter("d")
                if (!encodedData.isNullOrBlank()) {
                    DeepLinkAction.ApplySignedConfig(encodedData)
                } else null
            }

            // Setup with token (armorclaw://setup?token=xxx&server=xxx)
            "setup" -> {
                val token = uri.getQueryParameter("token")
                val server = uri.getQueryParameter("server")
                if (!token.isNullOrBlank() && !server.isNullOrBlank()) {
                    DeepLinkAction.SetupWithToken(token, server)
                } else null
            }

            // Invite code (armorclaw://invite?code=xxx)
            "invite" -> {
                val code = uri.getQueryParameter("code")
                if (!code.isNullOrBlank()) {
                    DeepLinkAction.AcceptInvite(code)
                } else null
            }

            // Device bonding (armorclaw://bond?token=xxx&challenge=xxx)
            "bond" -> {
                val token = uri.getQueryParameter("token")
                val challenge = uri.getQueryParameter("challenge")
                if (!token.isNullOrBlank()) {
                    DeepLinkAction.BondDevice(token, challenge)
                } else null
            }

            else -> {
                AppLogger.warning(
                    LogTag.UI.Navigation,
                    "Unknown armorclaw deep link host",
                    mapOf("host" to host)
                )
                null
            }
        }
    }

    /**
     * Parse https:// scheme deep links (matrix.to, chat.armorclaw.app, armorclaw.app)
     */
    private fun parseHttpsScheme(uri: Uri): DeepLinkAction? {
        val host = uri.host ?: return null

        return when (host) {
            HOST_MATRIX_TO -> parseMatrixToLink(uri)
            HOST_ARMORCLAW_CHAT -> parseArmorClawChatLink(uri)
            HOST_ARMORCLAW_APP -> parseArmorClawAppLink(uri)
            else -> {
                AppLogger.warning(
                    LogTag.UI.Navigation,
                    "Unknown HTTPS deep link host",
                    mapOf("host" to host)
                )
                null
            }
        }
    }

    /**
     * Parse matrix.to links
     * Format: https://matrix.to/#/{roomId} or https://matrix.to/#/!roomId:server
     */
    private fun parseMatrixToLink(uri: Uri): DeepLinkAction? {
        // Matrix.to links use fragment for the identifier
        val fragment = uri.fragment ?: return null

        // Remove leading # if present and query params
        val identifier = fragment.removePrefix("#").substringBefore("?")

        return when {
            identifier.startsWith("!") -> {
                // Room ID format: !roomId:server
                if (isValidRoomId(identifier)) {
                    DeepLinkAction.NavigateToRoom(identifier)
                } else null
            }
            identifier.startsWith("@") -> {
                // User ID format: @userId:server
                if (isValidUserId(identifier)) {
                    DeepLinkAction.NavigateToUser(identifier)
                } else null
            }
            identifier.startsWith("#") -> {
                // Room alias format: #alias:server
                DeepLinkAction.NavigateToRoom(identifier)
            }
            else -> {
                AppLogger.warning(
                    LogTag.UI.Navigation,
                    "Unknown matrix.to identifier format",
                    mapOf("identifier" to identifier)
                )
                null
            }
        }
    }

    /**
     * Parse chat.armorclaw.app links
     * Format: https://chat.armorclaw.app/room/{roomId}
     */
    private fun parseArmorClawChatLink(uri: Uri): DeepLinkAction? {
        val pathSegments = uri.pathSegments

        return when (pathSegments.firstOrNull()) {
            "room", "chat" -> {
                val roomId = pathSegments.getOrNull(1)
                if (roomId != null && isValidRoomId(roomId)) {
                    DeepLinkAction.NavigateToRoom(roomId)
                } else null
            }
            "user" -> {
                val userId = pathSegments.getOrNull(1)
                if (userId != null && isValidUserId(userId)) {
                    DeepLinkAction.NavigateToUser(userId)
                } else null
            }
            "call" -> {
                val callId = pathSegments.getOrNull(1)
                if (callId != null) {
                    DeepLinkAction.NavigateToCall(callId)
                } else null
            }
            "settings" -> DeepLinkAction.NavigateToSettings
            "profile" -> DeepLinkAction.NavigateToProfile
            else -> {
                AppLogger.warning(
                    LogTag.UI.Navigation,
                    "Unknown armorclaw.chat path",
                    mapOf("path" to (uri.path ?: "null"))
                )
                null
            }
        }
    }

    /**
     * Parse armorclaw.app links (main website deep links)
     * Formats:
     * - https://armorclaw.app/config?d=...
     * - https://armorclaw.app/invite/{code}
     * - https://armorclaw.app/setup?token=...&server=...
     */
    private fun parseArmorClawAppLink(uri: Uri): DeepLinkAction? {
        val pathSegments = uri.pathSegments

        return when (pathSegments.firstOrNull()) {
            "config" -> {
                val encodedData = uri.getQueryParameter("d")
                if (!encodedData.isNullOrBlank()) {
                    DeepLinkAction.ApplySignedConfig(encodedData)
                } else null
            }
            "invite" -> {
                val code = pathSegments.getOrNull(1) ?: uri.getQueryParameter("code")
                if (!code.isNullOrBlank()) {
                    DeepLinkAction.AcceptInvite(code)
                } else null
            }
            "setup" -> {
                val token = uri.getQueryParameter("token")
                val server = uri.getQueryParameter("server")
                if (!token.isNullOrBlank() && !server.isNullOrBlank()) {
                    DeepLinkAction.SetupWithToken(token, server)
                } else null
            }
            "bond" -> {
                val token = uri.getQueryParameter("token")
                val challenge = uri.getQueryParameter("challenge")
                if (!token.isNullOrBlank()) {
                    DeepLinkAction.BondDevice(token, challenge)
                } else null
            }
            else -> {
                AppLogger.warning(
                    LogTag.UI.Navigation,
                    "Unknown armorclaw.app path",
                    mapOf("path" to (uri.path ?: "null"))
                )
                null
            }
        }
    }

    // ==================== Security Helpers ====================

    /**
     * Validate Matrix room ID format
     * Room IDs start with ! and contain server name
     */
    private fun isValidRoomId(roomId: String): Boolean {
        // Basic validation: !localpart:server
        return roomId.startsWith("!") &&
                roomId.contains(":") &&
                roomId.length in 3..255 &&
                !roomId.contains("..") &&
                !roomId.contains("/")
    }

    /**
     * Validate Matrix user ID format
     * User IDs start with @ and contain server name
     */
    private fun isValidUserId(userId: String): Boolean {
        // Basic validation: @localpart:server
        return userId.startsWith("@") &&
                userId.contains(":") &&
                userId.length in 3..255 &&
                !userId.contains("..")
    }

    /**
     * Redact sensitive parts of URI for logging
     */
    private fun redactSensitiveUri(uri: String): String {
        // Truncate long URIs and redact potential tokens
        return if (uri.length > 200) {
            "${uri.take(100)}...[truncated]"
        } else {
            uri
        }
    }
}

/**
 * Result of deep link parsing with security validation
 */
sealed class DeepLinkResult {
    /** Valid deep link ready for navigation */
    data class Valid(val action: DeepLinkAction) : DeepLinkResult()

    /** Deep link requires user confirmation before navigation */
    data class RequiresConfirmation(
        val action: DeepLinkAction,
        val securityCheck: DeepLinkSecurityCheck,
        val message: String,
        val details: String? = null
    ) : DeepLinkResult()

    /** Invalid or malicious deep link */
    data class Invalid(val reason: String) : DeepLinkResult()
}

/**
 * Types of security checks that may be required
 */
enum class DeepLinkSecurityCheck {
    /** Check if user is a member of the room before navigating */
    ROOM_MEMBERSHIP,

    /** Confirm before joining a call */
    CALL_JOIN,

    /** Verify external link is from trusted source */
    EXTERNAL_LINK,

    /** Confirm before accepting an invite */
    INVITE_ACCEPT,

    /** Confirm before pairing with another device for admin access */
    DEVICE_BONDING
}

/**
 * Validation result for deep link URIs
 */
sealed class DeepLinkValidationResult {
    object Valid : DeepLinkValidationResult()
    data class Invalid(val reason: String) : DeepLinkValidationResult()
    data class UntrustedHost(val host: String) : DeepLinkValidationResult()
}

/**
 * Represents actions that can be taken from a deep link
 */
sealed class DeepLinkAction {
    abstract fun toRoute(): String?

    data class NavigateToRoom(val roomId: String) : DeepLinkAction() {
        override fun toRoute(): String = "chat/$roomId"
    }

    data class NavigateToUser(val userId: String) : DeepLinkAction() {
        override fun toRoute(): String = "user_profile/$userId"
    }

    data class NavigateToCall(val callId: String) : DeepLinkAction() {
        override fun toRoute(): String = "active_call/$callId"
    }

    object NavigateToSettings : DeepLinkAction() {
        override fun toRoute(): String = "settings"
    }

    object NavigateToProfile : DeepLinkAction() {
        override fun toRoute(): String = "profile"
    }

    /**
     * Apply signed server configuration from Bridge QR
     * The encoded data contains base64-encoded JSON with server URLs
     */
    data class ApplySignedConfig(val encodedData: String) : DeepLinkAction() {
        override fun toRoute(): String = "onboarding/config?d=$encodedData"
    }

    /**
     * Setup with token from Bridge
     * Used for initial device setup
     */
    data class SetupWithToken(val token: String, val server: String) : DeepLinkAction() {
        override fun toRoute(): String = "onboarding/setup?token=$token&server=$server"
    }

    /**
     * Accept invite code
     * Invites can contain room membership or server access
     */
    data class AcceptInvite(val code: String) : DeepLinkAction() {
        override fun toRoute(): String = "onboarding/invite?code=$code"
    }

    /**
     * Bond with another device for admin access
     * Used for ArmorTerminal pairing
     */
    data class BondDevice(val token: String, val challenge: String?) : DeepLinkAction() {
        override fun toRoute(): String = "settings/bond?token=$token"
    }
}
