package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.SyncConfig
import com.armorclaw.shared.domain.model.SyncResult
import com.armorclaw.shared.domain.model.SyncState
import kotlinx.coroutines.flow.Flow

interface SyncRepository {
    suspend fun syncWhenOnline(): SyncResult
    suspend fun syncRoom(roomId: String): SyncResult

    /**
     * Start the sync process
     */
    suspend fun startSync()

    /**
     * Stop the sync process
     */
    suspend fun stopSync()

    /**
     * Clear all sync state
     */
    suspend fun clearSyncState()

    fun observeSyncState(): Flow<SyncState>
    fun isOnline(): Boolean

    suspend fun getConfig(): SyncConfig
    suspend fun updateConfig(config: SyncConfig): Result<Unit>
}
