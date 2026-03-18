package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.Notification

interface NotificationRepository {
    suspend fun showNotification(notification: Notification): Result<Unit>
    suspend fun registerDevice(token: String): Result<Unit>
    suspend fun unregisterDevice(): Result<Unit>
    
    suspend fun requestPermission(): Result<Boolean>
    fun hasPermission(): Boolean
}
