package com.armorclaw.app.service

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.armorclaw.app.MainActivity
import com.armorclaw.app.R
import com.armorclaw.app.notifications.NotificationDeepLinkHandler
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.notification.PushNotificationRepository
import com.google.firebase.messaging.FirebaseMessaging
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import org.koin.android.ext.android.inject

/**
 * Firebase Cloud Messaging service for handling push notifications
 *
 * This service handles:
 * - Receiving FCM messages from the Bridge Server
 * - Displaying local notifications
 * - Managing FCM token registration
 * - Deep link handling from notifications
 *
 * ## Server-Side Requirements
 *
 * The Bridge Server must:
 * 1. Store FCM tokens per user/device (via `push.register` RPC)
 * 2. Call FCM API to send push notifications when:
 *    - New message received
 *    - Room invite received
 *    - Call started
 *    - Security alert triggered
 *
 * ## Notification Payload Format
 *
 * ```json
 * {
 *   "data": {
 *     "type": "message|invite|call|alert",
 *     "room_id": "!roomId:server",
 *     "event_id": "$eventId",
 *     "sender_id": "@sender:server"
 *   },
 *   "notification": {
 *     "title": "Sender Name",
 *     "body": "Message preview..."
 *   }
 * }
 * ```
 */
class ArmorFirebaseMessagingService : FirebaseMessagingService() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    override fun onDestroy() {
        super.onDestroy()
        scope.cancel()
    }

    // Inject PushNotificationRepository via Koin
    private val pushRepository: PushNotificationRepository by inject()

    // Inject MatrixClient for sync-on-push
    private val matrixClient: MatrixClient by inject()

    companion object {
        private const val CHANNEL_MESSAGES = "messages"
        private const val CHANNEL_ALERTS = "alerts"
        private const val CHANNEL_CALLS = "calls"

        private var registeredToken: String? = null

        /**
         * Get the current FCM token, requesting a new one if necessary
         */
        fun getCurrentToken(): String? {
            return registeredToken ?: run {
                // Token will be delivered via onNewToken callback
                FirebaseMessaging.getInstance().token.addOnCompleteListener { task ->
                    if (task.isSuccessful) {
                        registeredToken = task.result
                    }
                }
                registeredToken
            }
        }
    }

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        registeredToken = token

        AppLogger.info(
            LogTag.Network.Fcm,
            "FCM token received",
            mapOf("token_prefix" to token.take(10) + "...")
        )

        // Register token with Bridge Server via PushNotificationRepository
        scope.launch {
            registerTokenWithBridge(token)
        }
    }

    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        super.onMessageReceived(remoteMessage)

        AppLogger.info(
            LogTag.Network.Fcm,
            "FCM message received",
            mapOf(
                "from" to (remoteMessage.from ?: "unknown"),
                "data_keys" to remoteMessage.data.keys.joinToString(","),
                "has_notification" to (remoteMessage.notification != null)
            )
        )

        // Trigger a one-time Matrix sync to fetch actual event content.
        // This is critical for background/Doze mode: the push payload contains
        // only metadata (room_id, event_id). The SDK needs to sync to decrypt
        // the real message content and update its internal state.
        scope.launch {
            try {
                matrixClient.syncOnce()
                AppLogger.info(LogTag.Network.Fcm, "Push-triggered sync completed")
            } catch (e: Exception) {
                AppLogger.warning(
                    LogTag.Network.Fcm,
                    "Push-triggered sync failed, posting fallback notification",
                    mapOf("error" to (e.message ?: "unknown"))
                )
                // Fix 6: Post a fallback notification so the user knows something
                // arrived even though we couldn't decrypt the content via sync.
                showFallbackNotification(remoteMessage.data)
            }
        }

        // Extract notification data
        val data = remoteMessage.data
        val notification = remoteMessage.notification

        val type = data["type"] ?: "unknown"
        val roomId = data["room_id"]
        val eventId = data["event_id"]
        val senderId = data["sender_id"]

        val title = notification?.title ?: data["title"] ?: getString(R.string.app_name)
        val body = notification?.body ?: data["body"] ?: ""

        // Show notification
        showNotification(
            type = type,
            title = title,
            body = body,
            roomId = roomId,
            eventId = eventId,
            senderId = senderId
        )
    }

    /**
     * Post a fallback notification when syncOnce() fails (Fix 6).
     * Uses only push metadata since we couldn't decrypt the real content.
     */
    private fun showFallbackNotification(data: Map<String, String>) {
        val roomId = data["room_id"]
        val eventId = data["event_id"]
        showNotification(
            type = data["type"] ?: "message",
            title = getString(R.string.app_name),
            body = "You may have new messages. Tap to open.",
            roomId = roomId,
            eventId = eventId,
            senderId = data["sender_id"]
        )
    }

    /**
     * Display a local notification
     */
    private fun showNotification(
        type: String,
        title: String,
        body: String,
        roomId: String?,
        eventId: String?,
        senderId: String?
    ) {
        val notificationId = (roomId ?: eventId ?: System.currentTimeMillis().toString()).hashCode()

        // Build deep link intent
        val deepLinkIntent = when (type) {
            "message", "invite" -> {
                roomId?.let {
                    NotificationDeepLinkHandler.createRoomIntent(this, it, eventId)
                } ?: Intent(this, MainActivity::class.java)
            }
            "call" -> {
                roomId?.let {
                    NotificationDeepLinkHandler.createCallIntent(this, it)
                } ?: Intent(this, MainActivity::class.java)
            }
            else -> {
                Intent(this, MainActivity::class.java)
            }
        }.apply {
            flags = Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP
            putExtra("from_notification", true)
            putExtra("notification_type", type)
            roomId?.let { putExtra("room_id", it) }
            eventId?.let { putExtra("event_id", it) }
        }

        val pendingIntentFlags = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        } else {
            PendingIntent.FLAG_UPDATE_CURRENT
        }

        val pendingIntent = PendingIntent.getActivity(
            this,
            notificationId,
            deepLinkIntent,
            pendingIntentFlags
        )

        // Build notification
        val channelId = when (type) {
            "call" -> CHANNEL_CALLS
            "alert", "security" -> CHANNEL_ALERTS
            else -> CHANNEL_MESSAGES
        }

        val builder = NotificationCompat.Builder(this, channelId)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle(title)
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setCategory(
                when (type) {
                    "call" -> NotificationCompat.CATEGORY_CALL
                    "message", "invite" -> NotificationCompat.CATEGORY_MESSAGE
                    else -> NotificationCompat.CATEGORY_SOCIAL
                }
            )

        // Add person for messaging style (Android 7+)
        if (type == "message" && senderId != null) {
            val person = androidx.core.app.Person.Builder()
                .setKey(senderId)
                .setName(title)
                .build()

            builder.setStyle(
                NotificationCompat.MessagingStyle(person)
                    .addMessage(body, System.currentTimeMillis(), person)
            )
        }

        // Check notification permission (Android 13+)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (checkSelfPermission(android.Manifest.permission.POST_NOTIFICATIONS) !=
                android.content.pm.PackageManager.PERMISSION_GRANTED) {
                AppLogger.warning(
                    LogTag.Network.Fcm,
                    "Notification permission not granted, skipping notification"
                )
                return
            }
        }

        // Show notification
        try {
            NotificationManagerCompat.from(this).notify(notificationId, builder.build())
            AppLogger.info(
                LogTag.Network.Fcm,
                "Notification displayed",
                mapOf(
                    "type" to type,
                    "room_id" to (roomId ?: "null"),
                    "notification_id" to notificationId
                )
            )
        } catch (e: SecurityException) {
            AppLogger.error(
                LogTag.Network.Fcm,
                "Failed to show notification",
                e
            )
        }
    }

    /**
     * Register FCM token with Bridge Server via PushNotificationRepository
     *
     * The Bridge Server needs this token to send push notifications.
     * This is called when:
     * - A new token is generated (onNewToken)
     * - User logs in (should be called from login flow)
     * - Periodically to ensure token is still valid
     */
    private suspend fun registerTokenWithBridge(token: String) {
        // Get device ID from preferences
        val prefs = getSharedPreferences("device_prefs", Context.MODE_PRIVATE)
        val deviceId = prefs.getString("device_id", "unknown") ?: "unknown"

        AppLogger.info(
            LogTag.Network.Fcm,
            "Registering FCM token with Bridge",
            mapOf(
                "token_prefix" to token.take(10) + "...",
                "device_id" to deviceId
            )
        )

        pushRepository.registerToken(
            pushToken = token,
            pushPlatform = "fcm",
            deviceId = deviceId
        ).onSuccess {
            AppLogger.info(LogTag.Network.Fcm, "FCM token registered with Bridge successfully")
        }.onError { error ->
            AppLogger.error(
                LogTag.Network.Fcm,
                "Failed to register FCM token with Bridge",
                error.toException()
            )
        }
    }
}
