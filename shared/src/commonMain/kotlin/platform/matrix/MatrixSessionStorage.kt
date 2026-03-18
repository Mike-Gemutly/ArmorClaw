package com.armorclaw.shared.platform.matrix

import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * Matrix Session Storage Interface
 *
 * Provides secure persistence for Matrix session data.
 * Implementations should use platform-specific secure storage.
 *
 * ## Security Requirements
 * - Sessions MUST be stored in encrypted storage
 * - Access tokens MUST be protected
 * - Storage SHOULD be protected by biometric/PIN
 */
interface MatrixSessionStorage {
    /**
     * Save a Matrix session
     */
    suspend fun saveSession(session: MatrixSession): Result<Unit>

    /**
     * Load the current Matrix session
     */
    suspend fun loadSession(): Result<MatrixSession?>

    /**
     * Clear the stored session
     */
    suspend fun clearSession(): Result<Unit>

    /**
     * Check if a session exists
     */
    suspend fun hasSession(): Boolean

    /**
     * Observe session changes
     */
    fun observeSession(): StateFlow<MatrixSession?>
}

/**
 * Factory for creating platform-specific session storage
 */
expect object MatrixSessionStorageFactory {
    fun create(): MatrixSessionStorage
}
