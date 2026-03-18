package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.User
import kotlinx.coroutines.flow.Flow

interface UserRepository {
    suspend fun getUser(userId: String): Result<User?>
    suspend fun getCurrentUser(): Result<User?>
    suspend fun updateUser(user: User): Result<Unit>
    
    fun observeUser(userId: String): Flow<User?>
    fun observeCurrentUser(): Flow<User?>
    
    suspend fun updatePresence(isOnline: Boolean): Result<Unit>
}
