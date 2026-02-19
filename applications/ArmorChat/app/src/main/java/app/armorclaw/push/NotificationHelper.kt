package app.armorclaw.push

import android.app.Notification
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.Person
import androidx.core.app.TaskStackBuilder
import app.armorclaw.MainActivity
import app.armorclaw.R
import java.net.URL

/**
 * Helper class for building and displaying notifications
 */
class NotificationHelper(private val context: Context) {

    private val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

    /**
     * Notification data for building notifications
     */
    data class MessageNotification(
        val roomId: String,
        val roomName: String,
        val eventId: String,
        val senderId: String,
        val senderName: String,
        val senderAvatar: String? = null,
        val body: String,
        val timestamp: Long = System.currentTimeMillis(),
        val isMention: Boolean = false,
        val isEncrypted: Boolean = false
    )

    /**
     * Show a message notification
     */
    fun showMessageNotification(message: MessageNotification) {
        val channelId = if (message.isMention) {
            ArmorClawMessagingService.CHANNEL_ID_MENTIONS
        } else {
            ArmorClawMessagingService.CHANNEL_ID_MESSAGES
        }

        // Create deep link intent
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra(ArmorClawMessagingService.EXTRA_ROOM_ID, message.roomId)
            putExtra(ArmorClawMessagingService.EXTRA_EVENT_ID, message.eventId)
            action = Intent.ACTION_VIEW
            data = android.net.Uri.parse("armorclaw://room/${message.roomId}")
        }

        val pendingIntent = TaskStackBuilder.create(context).run {
            addNextIntentWithParentStack(intent)
            getPendingIntent(
                message.roomId.hashCode(),
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
            )
        }

        // Build the person for messaging style
        val senderPerson = Person.Builder()
            .setName(message.senderName)
            .setKey(message.senderId)
            .build()

        // Build notification
        val builder = NotificationCompat.Builder(context, channelId)
            .setSmallIcon(
                if (message.isMention) R.drawable.ic_notification_mention
                else R.drawable.ic_notification
            )
            .setContentTitle(
                if (message.roomName.isNotEmpty() && message.roomName != message.roomId) {
                    message.roomName
                } else {
                    message.senderName
                }
            )
            .setContentText(message.body)
            .setStyle(
                NotificationCompat.MessagingStyle(senderPerson)
                    .setConversationTitle(if (message.roomName != message.roomId) message.roomName else null)
                    .addMessage(
                        NotificationCompat.MessagingStyle.Message(
                            message.body,
                            message.timestamp,
                            Person.Builder().setName(message.senderName).build()
                        )
                    )
            )
            .setPriority(
                if (message.isMention) NotificationCompat.PRIORITY_HIGH
                else NotificationCompat.PRIORITY_DEFAULT
            )
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setWhen(message.timestamp)
            .setShowWhen(true)

        // Add sound/vibration for mentions
        if (message.isMention) {
            builder.setVibrate(longArrayOf(0, 250, 250, 250))
            builder.setLights(0xFFFF0000.toInt(), 1000, 1000)
        }

        // Add encrypted indicator if needed
        if (message.isEncrypted) {
            builder.setSubText(context.getString(R.string.notification_encrypted))
        }

        // Set group for bundling
        builder.setGroup(message.roomId)
        builder.setGroupAlertBehavior(NotificationCompat.GROUP_ALERT_SUMMARY)

        notificationManager.notify(message.roomId.hashCode(), builder.build())
    }

    /**
     * Show a summary notification for multiple messages in a room
     */
    fun showSummaryNotification(
        roomId: String,
        roomName: String,
        messageCount: Int,
        latestMessage: String
    ) {
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra(ArmorClawMessagingService.EXTRA_ROOM_ID, roomId)
            action = Intent.ACTION_VIEW
            data = android.net.Uri.parse("armorclaw://room/$roomId")
        }

        val pendingIntent = PendingIntent.getActivity(
            context,
            roomId.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val summary = NotificationCompat.Builder(context, ArmorClawMessagingService.CHANNEL_ID_MESSAGES)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(roomName)
            .setContentText("$messageCount new messages")
            .setStyle(
                NotificationCompat.InboxStyle()
                    .setBigContentTitle(roomName)
                    .setSummaryText("$messageCount new messages")
            )
            .setGroup(roomId)
            .setGroupSummary(true)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        notificationManager.notify(roomId.hashCode() + 1000, summary)
    }

    /**
     * Cancel notifications for a room
     */
    fun cancelRoomNotifications(roomId: String) {
        notificationManager.cancel(roomId.hashCode())
        notificationManager.cancel(roomId.hashCode() + 1000) // Summary
    }

    /**
     * Cancel all notifications
     */
    fun cancelAllNotifications() {
        notificationManager.cancelAll()
    }

    companion object {
        /**
         * Create notification channels - call this at app startup
         */
        fun createChannels(context: Context) {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                val manager = context.getSystemService(NotificationManager::class.java)

                // Messages channel
                val messagesChannel = android.app.NotificationChannel(
                    ArmorClawMessagingService.CHANNEL_ID_MESSAGES,
                    context.getString(R.string.channel_messages),
                    android.app.NotificationManager.IMPORTANCE_HIGH
                ).apply {
                    description = context.getString(R.string.channel_messages_desc)
                    enableLights(true)
                    lightColor = 0xFF00FF00.toInt()
                    enableVibration(true)
                    setShowBadge(true)
                }

                // Mentions channel
                val mentionsChannel = android.app.NotificationChannel(
                    ArmorClawMessagingService.CHANNEL_ID_MENTIONS,
                    context.getString(R.string.channel_mentions),
                    android.app.NotificationManager.IMPORTANCE_HIGH
                ).apply {
                    description = context.getString(R.string.channel_mentions_desc)
                    enableLights(true)
                    lightColor = 0xFFFF0000.toInt()
                    enableVibration(true)
                    vibrationPattern = longArrayOf(0, 250, 250, 250)
                    setShowBadge(true)
                }

                // System channel
                val systemChannel = android.app.NotificationChannel(
                    ArmorClawMessagingService.CHANNEL_ID_SYSTEM,
                    context.getString(R.string.channel_system),
                    android.app.NotificationManager.IMPORTANCE_DEFAULT
                ).apply {
                    description = context.getString(R.string.channel_system_desc)
                    setShowBadge(true)
                }

                manager.createNotificationChannels(listOf(
                    messagesChannel,
                    mentionsChannel,
                    systemChannel
                ))
            }
        }
    }
}
