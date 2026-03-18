package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

@Serializable
sealed class SyncState {
    @Serializable
    object Idle : SyncState()
    
    @Serializable
    object Syncing : SyncState()
    
    @Serializable
    object Offline : SyncState()
    
    @Serializable
    data class Error(val message: String, val isRecoverable: Boolean = true) : SyncState()
    
    @Serializable
    data class Success(val messagesSent: Int, val messagesReceived: Int) : SyncState()
}

@Serializable
sealed class LoadState {
    @Serializable
    object Loading : LoadState()
    
    @Serializable
    object Success : LoadState()
    
    @Serializable
    data class Error(val message: String?, val isRecoverable: Boolean = true) : LoadState()
    
    @Serializable
    data class Empty(val message: String) : LoadState()
}

@Serializable
data class SyncConfig(
    val maxOfflineMessages: Int = 1000,
    val maxOfflineDays: Int = 7,
    val syncBatchSize: Int = 50,
    val syncTimeout: Long = 30000, // 30 seconds in ms
    val messageExpiry: Long = 604800000, // 7 days in ms
    val initialRetryDelay: Long = 5000, // 5 seconds in ms
    val maxRetries: Int = 5,
    val periodicSyncInterval: Long = 300000 // 5 minutes in ms
)

@Serializable
data class SyncResult(
    val messagesSent: Int = 0,
    val messagesReceived: Int = 0,
    val conflicts: Int = 0
)
