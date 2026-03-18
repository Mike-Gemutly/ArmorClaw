package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserSession
import kotlinx.coroutines.flow.Flow
import kotlinx.serialization.Serializable

interface AuthRepository {
    suspend fun login(config: ServerConfig): Result<UserSession>
    suspend fun logout(): Result<Unit>
    suspend fun refreshSession(): Result<UserSession>

    suspend fun getCurrentUser(): Result<User?>
    fun observeSession(): Flow<UserSession?>

    fun isLoggedIn(): Boolean

    /**
     * Clear all local authentication data
     * This removes tokens and session data without contacting the server
     */
    suspend fun clearLocalAuth()
}

@Serializable
data class ServerConfig(
    val homeserver: String,
    val username: String,
    val password: String,
    val deviceId: String? = null
)
