package app.armorclaw.navigation

import android.net.Uri

/**
 * Parses armorclaw:// deep link URIs into navigation [Route] objects.
 *
 * Supported patterns:
 * - `armorclaw://room/{roomId}` → [Route.Room]
 * - `armorclaw://email/approve/{approvalId}` → [Route.EmailApproval]
 *
 * Config deep links (`armorclaw://config?d=...`) are intentionally NOT handled
 * here — [app.armorclaw.config.SignedConfigParser] owns that flow.
 */
object DeepLinkHandler {

    private const val SCHEME = "armorclaw"

    fun handle(uri: Uri): Route? {
        if (uri.scheme != SCHEME) return null
        // armorclaw://config is handled by SignedConfigParser
        if (uri.host == "config") return null

        return when (uri.host) {
            "room" -> parseRoom(uri)
            "email" -> parseEmailApproval(uri)
            else -> null
        }
    }

    private fun parseRoom(uri: Uri): Route.Room? {
        val segments = uri.pathSegments
        val roomId = segments.firstOrNull()?.takeIf { it.isNotBlank() } ?: return null
        return Route.Room(roomId)
    }

    private fun parseEmailApproval(uri: Uri): Route.EmailApproval? {
        val segments = uri.pathSegments
        if (segments.getOrNull(0) != "approve") return null
        val approvalId = segments.getOrNull(1)?.takeIf { it.isNotBlank() } ?: return null
        return Route.EmailApproval(approvalId)
    }
}
