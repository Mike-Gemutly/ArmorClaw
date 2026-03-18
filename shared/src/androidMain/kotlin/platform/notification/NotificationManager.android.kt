package com.armorclaw.shared.platform.notification

import android.Manifest
import android.app.NotificationChannel
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.core.app.ActivityCompat
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.armorclaw.shared.domain.model.Notification
import com.armorclaw.shared.domain.model.NotificationType
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow

actual class NotificationManager {

    private var context: Context? = null
    private val _permission = MutableStateFlow(false)

    companion object {
        private const val CHANNEL_MESSAGES = "messages"
        private const val CHANNEL_ALERTS = "alerts"

        @Volatile
        private var instance: NotificationManager? = null

        fun getInstance(): NotificationManager {
            return instance ?: synchronized(this) {
                instance ?: NotificationManager().also { instance = it }
            }
        }

        fun setContext(context: Context) {
            getInstance().context = context.applicationContext
            getInstance().initialize()
        }
    }

    private fun initialize() {
        val ctx = context ?: return
        _permission.value = checkPermission(ctx)
        createNotificationChannels(ctx)
    }

    actual suspend fun showNotification(notification: Notification): Result<Unit> {
        val ctx = context ?: return Result.failure(Exception("Context not set"))

        return try {
            if (!checkPermission(ctx)) {
                return Result.failure(Exception("Notification permission not granted"))
            }

            val extras = Bundle()
            notification.data.forEach { (key, value) ->
                extras.putString(key, value)
            }

            val builder = NotificationCompat.Builder(ctx, getChannelId(notification.type))
                .setSmallIcon(android.R.drawable.ic_dialog_info)
                .setContentTitle(notification.title)
                .setContentText(notification.message)
                .setPriority(NotificationCompat.PRIORITY_DEFAULT)
                .setAutoCancel(true)
                .addExtras(extras)

            NotificationManagerCompat.from(ctx).notify(notification.id.hashCode(), builder.build())
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun registerDevice(token: String): Result<Unit> {
        val ctx = context ?: return Result.failure(Exception("Context not set"))

        return try {
            // Store token locally for Bridge registration
            val prefs = ctx.getSharedPreferences("fcm_prefs", Context.MODE_PRIVATE)
            prefs.edit().putString("fcm_token", token).apply()

            // TODO(#push): Call Bridge API to register FCM token
            // POST /api/v1/devices/register
            // {
            //   "fcm_token": token,
            //   "device_id": <current_device_id>,
            //   "user_id": <current_user_id>
            // }
            //
            // This should be implemented in BridgeRepository or a dedicated
            // PushNotificationRepository that calls the Bridge RPC method.
            //
            // Example:
            // bridgeRepository.registerPushToken(token, deviceId, userId)

            _permission.value = true
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Get the stored FCM token
     */
    fun getStoredToken(): String? {
        val ctx = context ?: return null
        val prefs = ctx.getSharedPreferences("fcm_prefs", Context.MODE_PRIVATE)
        return prefs.getString("fcm_token", null)
    }

    actual suspend fun unregisterDevice(): Result<Unit> {
        // FCM unregistration would go here
        return Result.success(Unit)
    }

    actual suspend fun requestPermission(): Result<Boolean> {
        val ctx = context ?: return Result.failure(Exception("Context not set"))

        return try {
            if (checkPermission(ctx)) {
                return Result.success(true)
            }

            // Request permission (would need Activity context)
            Result.success(false)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual fun hasPermission(): Boolean {
        return context?.let { checkPermission(it) } ?: false
    }

    actual fun observePermission(): Flow<Boolean> {
        return _permission.asStateFlow()
    }

    actual suspend fun cancelNotification(id: String): Result<Unit> {
        val ctx = context ?: return Result.failure(Exception("Context not set"))

        return try {
            NotificationManagerCompat.from(ctx).cancel(id.hashCode())
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun cancelAllNotifications(): Result<Unit> {
        val ctx = context ?: return Result.failure(Exception("Context not set"))

        return try {
            NotificationManagerCompat.from(ctx).cancelAll()
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun createChannel(
        channelId: String,
        channelName: String,
        channelDescription: String?,
        importance: NotificationImportance
    ): Result<Unit> {
        val ctx = context ?: return Result.failure(Exception("Context not set"))

        return try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                val channel = NotificationChannel(
                    channelId,
                    channelName,
                    mapImportance(importance)
                ).apply {
                    description = channelDescription
                }

                NotificationManagerCompat.from(ctx).createNotificationChannel(channel)
            }
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun createNotificationChannels(ctx: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channels = listOf(
                NotificationChannel(
                    CHANNEL_MESSAGES,
                    "Messages",
                    mapImportance(NotificationImportance.HIGH)
                ).apply {
                    description = "New message notifications"
                },
                NotificationChannel(
                    CHANNEL_ALERTS,
                    "Alerts",
                    mapImportance(NotificationImportance.DEFAULT)
                ).apply {
                    description = "Budget and security alerts"
                }
            )

            val notificationManager = ctx.getSystemService(Context.NOTIFICATION_SERVICE) as android.app.NotificationManager
            channels.forEach { channel ->
                notificationManager.createNotificationChannel(channel)
            }
        }
    }

    private fun getChannelId(type: NotificationType): String {
        return when (type) {
            is NotificationType.NewMessage -> CHANNEL_MESSAGES
            is NotificationType.BudgetAlert -> CHANNEL_ALERTS
            is NotificationType.SecurityAlert -> CHANNEL_ALERTS
            is NotificationType.TaskUpdate -> CHANNEL_MESSAGES
            is NotificationType.SyncUpdate -> CHANNEL_ALERTS
        }
    }

    private fun mapImportance(importance: NotificationImportance): Int {
        return when (importance) {
            NotificationImportance.MIN -> android.app.NotificationManager.IMPORTANCE_MIN
            NotificationImportance.LOW -> android.app.NotificationManager.IMPORTANCE_LOW
            NotificationImportance.DEFAULT -> android.app.NotificationManager.IMPORTANCE_DEFAULT
            NotificationImportance.HIGH -> android.app.NotificationManager.IMPORTANCE_HIGH
            NotificationImportance.MAX -> android.app.NotificationManager.IMPORTANCE_HIGH
        }
    }

    private fun checkPermission(ctx: Context): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            ActivityCompat.checkSelfPermission(
                ctx,
                Manifest.permission.POST_NOTIFICATIONS
            ) == PackageManager.PERMISSION_GRANTED
        } else {
            true
        }
    }
}
