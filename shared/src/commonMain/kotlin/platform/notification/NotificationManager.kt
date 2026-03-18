package com.armorclaw.shared.platform.notification

import com.armorclaw.shared.domain.model.Notification
import kotlinx.coroutines.flow.Flow

expect class NotificationManager() {
    suspend fun showNotification(notification: Notification): Result<Unit>
    suspend fun registerDevice(token: String): Result<Unit>
    suspend fun unregisterDevice(): Result<Unit>
    suspend fun requestPermission(): Result<Boolean>
    fun hasPermission(): Boolean
    fun observePermission(): Flow<Boolean>
    
    suspend fun cancelNotification(id: String): Result<Unit>
    suspend fun cancelAllNotifications(): Result<Unit>
    
    suspend fun createChannel(
        channelId: String,
        channelName: String,
        channelDescription: String? = null,
        importance: NotificationImportance = NotificationImportance.DEFAULT
    ): Result<Unit>
}

enum class NotificationImportance {
    MIN,
    LOW,
    DEFAULT,
    HIGH,
    MAX
}
