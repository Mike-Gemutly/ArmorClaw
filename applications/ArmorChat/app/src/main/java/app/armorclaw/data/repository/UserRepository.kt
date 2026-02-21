package app.armorclaw.data.repository

import android.content.Context
import android.util.Log
import app.armorclaw.data.local.entity.UserEntity
import app.armorclaw.data.local.entity.UserNamespace
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * User Repository - Identity & Autocomplete Logic
 *
 * Resolves: G-04 (Identity Consistency)
 *
 * Handles user identity management with namespace tagging for bridged users.
 * Supports autocomplete differentiation between Matrix users and Ghost Users.
 */
class UserRepository(private val context: Context) {

    companion object {
        private const val TAG = "UserRepository"

        // Namespace prefixes for bridged users
        const val NAMESPACE_SLACK = "@slack_"
        const val NAMESPACE_DISCORD = "@discord_"
        const val NAMESPACE_TEAMS = "@teams_"
        const val NAMESPACE_WHATSAPP = "@whatsapp_"
        const val NAMESPACE_BRIDGE = "@bridge_"
    }

    private val _users = MutableStateFlow<Map<String, UserEntity>>(emptyMap())
    val users: StateFlow<Map<String, UserEntity>> = _users.asStateFlow()

    private val _autocompleteResults = MutableStateFlow<List<AutocompleteUser>>(emptyList())
    val autocompleteResults: StateFlow<List<AutocompleteUser>> = _autocompleteResults.asStateFlow()

    /**
     * Get namespace for a user ID
     */
    fun getUserNamespace(userId: String): UserNamespace {
        return when {
            userId.startsWith(NAMESPACE_SLACK) -> UserNamespace.SLACK
            userId.startsWith(NAMESPACE_DISCORD) -> UserNamespace.DISCORD
            userId.startsWith(NAMESPACE_TEAMS) -> UserNamespace.TEAMS
            userId.startsWith(NAMESPACE_WHATSAPP) -> UserNamespace.WHATSAPP
            userId.startsWith(NAMESPACE_BRIDGE) -> UserNamespace.BRIDGE
            else -> UserNamespace.MATRIX
        }
    }

    /**
     * Check if a user is a bridged ghost user
     */
    fun isBridgedUser(userId: String): Boolean {
        return getUserNamespace(userId) != UserNamespace.MATRIX
    }

    /**
     * Get display name for autocomplete
     */
    fun getAutocompleteDisplayName(user: UserEntity): String {
        val namespace = getUserNamespace(user.userId)
        val baseName = user.displayName ?: extractLocalpart(user.userId)

        return if (namespace == UserNamespace.MATRIX) {
            baseName
        } else {
            "$baseName (${namespace.displayName})"
        }
    }

    /**
     * Extract localpart from Matrix ID
     */
    private fun extractLocalpart(userId: String): String {
        if (!userId.startsWith("@")) return userId

        val withoutPrefix = userId.substring(1)

        // Find the colon separator
        val colonIndex = withoutPrefix.indexOf(':')
        if (colonIndex > 0) {
            return withoutPrefix.substring(0, colonIndex)
        }

        return withoutPrefix
    }

    /**
     * Search users for autocomplete
     * Returns results with namespace indicators
     */
    fun searchUsers(query: String, roomId: String? = null) {
        val currentUserMap = _users.value
        val results = mutableListOf<AutocompleteUser>()

        val lowerQuery = query.lowercase().removePrefix("@")

        currentUserMap.values.forEach { user ->
            val displayName = user.displayName ?: ""
            val userId = user.userId

            // Match against display name or user ID
            val matchesDisplayName = displayName.lowercase().contains(lowerQuery)
            val matchesUserId = userId.lowercase().contains(lowerQuery)

            if (matchesDisplayName || matchesUserId) {
                val namespace = getUserNamespace(userId)
                results.add(
                    AutocompleteUser(
                        userId = userId,
                        displayName = displayName.ifEmpty { extractLocalpart(userId) },
                        namespace = namespace,
                        avatarUrl = user.avatarUrl,
                        isBridged = namespace != UserNamespace.MATRIX,
                        platformIcon = namespace.iconRes,
                        matchScore = calculateMatchScore(lowerQuery, displayName, userId)
                    )
                )
            }
        }

        // Sort by match score, then prioritize Matrix users
        results.sortWith(compareByDescending<AutocompleteUser> { it.matchScore }
            .thenBy { it.namespace.order })

        _autocompleteResults.value = results.take(10) // Limit to 10 results
    }

    /**
     * Calculate match score for sorting
     */
    private fun calculateMatchScore(query: String, displayName: String, userId: String): Int {
        var score = 0

        // Exact match on display name
        if (displayName.lowercase() == query) {
            score += 100
        }
        // Starts with query
        else if (displayName.lowercase().startsWith(query)) {
            score += 80
        }
        // Contains query
        else if (displayName.lowercase().contains(query)) {
            score += 60
        }

        // Exact match on user ID localpart
        val localpart = extractLocalpart(userId).lowercase()
        if (localpart == query) {
            score += 90
        } else if (localpart.startsWith(query)) {
            score += 70
        }

        return score
    }

    /**
     * Add or update a user
     */
    fun upsertUser(user: UserEntity) {
        val currentMap = _users.value.toMutableMap()
        currentMap[user.userId] = user
        _users.value = currentMap
    }

    /**
     * Get user by ID
     */
    fun getUser(userId: String): UserEntity? {
        return _users.value[userId]
    }

    /**
     * Get all users in a specific namespace
     */
    fun getUsersByNamespace(namespace: UserNamespace): List<UserEntity> {
        return _users.value.values.filter { getUserNamespace(it.userId) == namespace }
    }

    /**
     * Clear autocomplete results
     */
    fun clearAutocompleteResults() {
        _autocompleteResults.value = emptyList()
    }
}

/**
 * Autocomplete user result with namespace information
 */
data class AutocompleteUser(
    val userId: String,
    val displayName: String,
    val namespace: UserNamespace,
    val avatarUrl: String? = null,
    val isBridged: Boolean = false,
    val platformIcon: Int? = null, // Resource ID for platform icon
    val matchScore: Int = 0
)

/**
 * User namespace enum with display information
 */
enum class UserNamespace(val displayName: String, val iconRes: Int?, val order: Int) {
    MATRIX("Matrix", null, 0),
    SLACK("Slack", android.R.drawable.ic_menu_send, 1), // Replace with actual icon
    DISCORD("Discord", android.R.drawable.ic_menu_send, 2),
    TEAMS("Teams", android.R.drawable.ic_menu_send, 3),
    WHATSAPP("WhatsApp", android.R.drawable.ic_menu_send, 4),
    BRIDGE("Bridge", android.R.drawable.ic_menu_send, 5);
}
