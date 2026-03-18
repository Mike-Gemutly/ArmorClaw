package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

@Serializable
data class Notification(
    val id: String,
    val type: NotificationType,
    val title: String,
    val message: String,
    val data: Map<String, String> = emptyMap(),
    val timestamp: Long = System.currentTimeMillis()
)

@Serializable
sealed class NotificationType {
    @Serializable
    data class NewMessage(val roomId: String, val sender: String) : NotificationType()
    
    @Serializable
    data class BudgetAlert(val percentUsed: Float) : NotificationType()
    
    @Serializable
    data class SecurityAlert(val event: String) : NotificationType()
    
    @Serializable
    data class TaskUpdate(val taskId: String, val status: String) : NotificationType()
    
    @Serializable
    data class SyncUpdate(val state: SyncState) : NotificationType()
}
