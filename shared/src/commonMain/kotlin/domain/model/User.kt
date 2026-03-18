package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable
import kotlinx.datetime.Instant

@Serializable
data class User(
    val id: String,
    val displayName: String,
    val avatar: String? = null,
    val email: String? = null,
    val presence: UserPresence = UserPresence.OFFLINE,
    val lastActive: Instant? = null,
    val isVerified: Boolean = false
)

@Serializable
data class UserSession(
    val userId: String,
    val accessToken: String,
    val refreshToken: String,
    val deviceId: String,
    val homeserver: String,
    val expiresAt: Instant
)

@Serializable
enum class UserPresence {
    ONLINE,
    UNAVAILABLE,
    OFFLINE,
    UNKNOWN
}
