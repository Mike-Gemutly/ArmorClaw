package com.armorclaw.app.notifications

import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag

/**
 * Handles deep links from push notifications
 *
 * Parses notification payloads and creates navigation intents.
 */
class NotificationDeepLinkHandler(
    private val context: Context
) {
    private val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

    /**
     * Handle notification tap
     *
     * @param data Notification data payload
     * @return Navigation action to perform, or null if not applicable
     */
    fun handleNotificationTap(data: Map<String, String?>): NotificationAction? {
        AppLogger.info(
            LogTag.Domain.Notification,
            "Handling notification tap",
            mapOf("data" to data.toString().take(200))
        )

        val type = data["type"] ?: data["notification_type"] ?: return null

        return when (type) {
            "message", "m.room.message" -> {
                val roomId = data["room_id"] ?: return null
                val eventId = data["event_id"]
                NotificationAction.NavigateToRoom(roomId, eventId)
            }
            "call", "m.call.invite" -> {
                val callId = data["call_id"] ?: data["room_id"] ?: return null
                val callerName = data["sender_display_name"] ?: "Unknown"
                NotificationAction.IncomingCall(callId, callerName)
            }
            "verification", "m.key.verification.request" -> {
                val deviceId = data["device_id"] ?: return null
                val deviceName = data["device_display_name"] ?: "Unknown Device"
                NotificationAction.DeviceVerification(deviceId, deviceName)
            }
            "mention" -> {
                val roomId = data["room_id"] ?: return null
                val eventId = data["event_id"]
                NotificationAction.NavigateToRoom(roomId, eventId, scrollMentionIntoView = true)
            }
            "reaction" -> {
                val roomId = data["room_id"] ?: return null
                val eventId = data["event_id"]
                NotificationAction.NavigateToRoom(roomId, eventId)
            }
            "invite", "m.room.member" -> {
                val roomId = data["room_id"] ?: return null
                NotificationAction.RoomInvite(roomId)
            }
            else -> {
                AppLogger.warning(
                    LogTag.Domain.Notification,
                    "Unknown notification type: $type"
                )
                null
            }
        }
    }

    /**
     * Create intent from notification action
     */
    fun createIntent(action: NotificationAction): Intent {
        val uri = when (action) {
            is NotificationAction.NavigateToRoom -> {
                val uriBuilder = Uri.Builder()
                    .scheme("armorclaw")
                    .authority("room")
                    .appendPath(action.roomId)
                if (action.eventId != null) {
                    uriBuilder.appendQueryParameter("eventId", action.eventId)
                }
                if (action.scrollMentionIntoView) {
                    uriBuilder.appendQueryParameter("highlight", "true")
                }
                uriBuilder.build()
            }
            is NotificationAction.IncomingCall -> {
                Uri.Builder()
                    .scheme("armorclaw")
                    .authority("call")
                    .appendPath(action.callId)
                    .appendQueryParameter("caller", action.callerName)
                    .build()
            }
            is NotificationAction.DeviceVerification -> {
                Uri.Builder()
                    .scheme("armorclaw")
                    .authority("verification")
                    .appendPath(action.deviceId)
                    .appendQueryParameter("name", action.deviceName)
                    .build()
            }
            is NotificationAction.RoomInvite -> {
                Uri.Builder()
                    .scheme("armorclaw")
                    .authority("invite")
                    .appendPath(action.roomId)
                    .build()
            }
        }

        return Intent(Intent.ACTION_VIEW, uri).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
        }
    }

    /**
     * Cancel notification
     */
    fun cancelNotification(notificationId: Int) {
        notificationManager.cancel(notificationId)
    }

    /**
     * Cancel all notifications
     */
    fun cancelAllNotifications() {
        notificationManager.cancelAll()
    }

    companion object {
        const val EXTRA_NOTIFICATION_ACTION = "notification_action"
        const val EXTRA_ROOM_ID = "room_id"
        const val EXTRA_EVENT_ID = "event_id"
        const val EXTRA_CALL_ID = "call_id"
        const val EXTRA_DEVICE_ID = "device_id"

        /**
         * Create an intent to navigate to a room
         */
        fun createRoomIntent(context: Context, roomId: String, eventId: String?): Intent {
            val uriBuilder = Uri.Builder()
                .scheme("armorclaw")
                .authority("room")
                .appendPath(roomId)
            if (eventId != null) {
                uriBuilder.appendQueryParameter("eventId", eventId)
            }
            return Intent(Intent.ACTION_VIEW, uriBuilder.build()).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
                setPackage(context.packageName)
            }
        }

        /**
         * Create an intent to join a call
         */
        fun createCallIntent(context: Context, callId: String): Intent {
            val uri = Uri.Builder()
                .scheme("armorclaw")
                .authority("call")
                .appendPath(callId)
                .build()
            return Intent(Intent.ACTION_VIEW, uri).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
                setPackage(context.packageName)
            }
        }
    }
}

/**
 * Notification navigation actions
 */
sealed class NotificationAction {
    /**
     * Navigate to a specific room
     */
    data class NavigateToRoom(
        val roomId: String,
        val eventId: String? = null,
        val scrollMentionIntoView: Boolean = false
    ) : NotificationAction()

    /**
     * Show incoming call screen
     */
    data class IncomingCall(
        val callId: String,
        val callerName: String
    ) : NotificationAction()

    /**
     * Start device verification flow
     */
    data class DeviceVerification(
        val deviceId: String,
        val deviceName: String
    ) : NotificationAction()

    /**
     * Show room invite
     */
    data class RoomInvite(
        val roomId: String
    ) : NotificationAction()
}

/**
 * Notification data for building notifications
 */
data class NotificationData(
    val id: Int,
    val title: String,
    val body: String,
    val channelId: String,
    val type: String,
    val roomId: String? = null,
    val eventId: String? = null,
    val senderId: String? = null,
    val senderName: String? = null,
    val icon: String? = null,
    val sound: Boolean = true,
    val vibration: Boolean = true,
    val deepLink: Uri? = null
)
