package app.armorclaw.push

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.media.RingtoneManager
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.Person
import androidx.core.app.TaskStackBuilder
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import app.armorclaw.MainActivity
import app.armorclaw.R
import app.armorclaw.data.local.entity.RoomEntity
import app.armorclaw.data.repository.BridgeRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * Firebase Cloud Messaging Service for ArmorChat
 *
 * Handles:
 * - FCM token refresh and registration with Bridge
 * - Incoming message notifications
 * - Deep linking to specific chat rooms
 */
class ArmorClawMessagingService : FirebaseMessagingService() {

    companion object {
        private const val TAG = "ArmorClawFCM"
        const val CHANNEL_ID_MESSAGES = "messages"
        const val CHANNEL_ID_MENTIONS = "mentions"
        const val CHANNEL_ID_SYSTEM = "system"

        // Notification extras for deep linking
        const val EXTRA_ROOM_ID = "room_id"
        const val EXTRA_EVENT_ID = "event_id"
        const val EXTRA_SENDER_ID = "sender_id"
    }

    private val serviceScope = CoroutineScope(Dispatchers.IO)

    /**
     * Called when a new FCM token is generated
     */
    override fun onNewToken(token: String) {
        Log.d(TAG, "Received new FCM token: ${token.take(10)}...")

        // Register token with Bridge Server
        serviceScope.launch {
            try {
                val repository = BridgeRepository.getInstance(applicationContext)
                repository.registerPushToken(token)
                Log.d(TAG, "FCM token registered with bridge")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to register FCM token", e)
                // Retry logic handled by repository
            }
        }
    }

    /**
     * Called when a message is received
     */
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        Log.d(TAG, "Received message from: ${remoteMessage.from}")

        // Check if message contains data payload
        if (remoteMessage.data.isNotEmpty()) {
            handleDataMessage(remoteMessage)
        }

        // Check if message contains notification payload
        remoteMessage.notification?.let {
            handleNotificationMessage(it)
        }
    }

    /**
     * Handle data message (background sync trigger)
     */
    private fun handleDataMessage(message: RemoteMessage) {
        val data = message.data

        when (data["type"]) {
            "message" -> handleMessageNotification(data)
            "mention" -> handleMentionNotification(data)
            "invite" -> handleInviteNotification(data)
            "sync" -> triggerBackgroundSync(data)
            else -> Log.w(TAG, "Unknown message type: ${data["type"]}")
        }
    }

    /**
     * Handle incoming message notification
     */
    private fun handleMessageNotification(data: Map<String, String>) {
        val roomId = data["room_id"] ?: return
        val eventId = data["event_id"] ?: return
        val senderName = data["sender_name"] ?: "Unknown"
        val senderId = data["sender_id"] ?: ""
        val roomName = data["room_name"] ?: roomId
        val messageBody = data["body"] ?: "New message"
        val isEncrypted = data["encrypted"]?.toBoolean() ?: false

        // For encrypted messages, body might be placeholder
        val displayBody = if (isEncrypted && messageBody.isEmpty()) {
            getString(R.string.notification_encrypted_message)
        } else {
            messageBody
        }

        // Create intent for deep link
        val intent = createChatIntent(roomId, eventId)
        val pendingIntent = PendingIntent.getActivity(
            this,
            roomId.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        // Build notification
        val notification = NotificationCompat.Builder(this, CHANNEL_ID_MESSAGES)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(if (roomName != roomId) roomName else senderName)
            .setContentText(displayBody)
            .setStyle(
                NotificationCompat.MessagingStyle(Person.Builder().setName("You").build())
                    .addMessage(
                        NotificationCompat.MessagingStyle.Message(
                            displayBody,
                            System.currentTimeMillis(),
                            Person.Builder().setName(senderName).build()
                        )
                    )
            )
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setSound(RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION))
            .build()

        // Show notification
        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(roomId.hashCode(), notification)
    }

    /**
     * Handle mention notification (higher priority)
     */
    private fun handleMentionNotification(data: Map<String, String>) {
        val roomId = data["room_id"] ?: return
        val senderName = data["sender_name"] ?: "Someone"
        val messageBody = data["body"] ?: "mentioned you"

        val intent = createChatIntent(roomId, data["event_id"])
        val pendingIntent = PendingIntent.getActivity(
            this,
            roomId.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID_MENTIONS)
            .setSmallIcon(R.drawable.ic_notification_mention)
            .setContentTitle("$senderName mentioned you")
            .setContentText(messageBody)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setVibrate(longArrayOf(0, 250, 250, 250))
            .setLights(0xFF00FF00.toInt(), 1000, 1000)
            .build()

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(roomId.hashCode(), notification)
    }

    /**
     * Handle invite notification
     */
    private fun handleInviteNotification(data: Map<String, String>) {
        val roomId = data["room_id"] ?: return
        val inviterName = data["inviter_name"] ?: "Someone"
        val roomName = data["room_name"] ?: "a room"

        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("open_invites", true)
        }
        val pendingIntent = PendingIntent.getActivity(
            this,
            "invites".hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID_SYSTEM)
            .setSmallIcon(R.drawable.ic_notification_invite)
            .setContentTitle("Room Invitation")
            .setContentText("$inviterName invited you to $roomName")
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setCategory(NotificationCompat.CATEGORY_SOCIAL)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.notify(roomId.hashCode(), notification)
    }

    /**
     * Trigger background sync when receiving sync wake-up
     */
    private fun triggerBackgroundSync(data: Map<String, String>) {
        Log.d(TAG, "Triggering background sync")

        serviceScope.launch {
            try {
                val repository = BridgeRepository.getInstance(applicationContext)
                // Perform quick sync to fetch pending messages
                repository.performQuickSync()
            } catch (e: Exception) {
                Log.e(TAG, "Background sync failed", e)
            }
        }
    }

    /**
     * Handle notification payload (displayed by system when app is in background)
     */
    private fun handleNotificationMessage(notification: RemoteMessage.Notification) {
        // When app is in background, FCM displays notifications automatically
        // This is called when app is in foreground
        Log.d(TAG, "Notification title: ${notification.title}")
        Log.d(TAG, "Notification body: ${notification.body}")
    }

    /**
     * Create intent for deep linking to chat
     */
    private fun createChatIntent(roomId: String, eventId: String?): Intent {
        return Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra(EXTRA_ROOM_ID, roomId)
            eventId?.let { putExtra(EXTRA_EVENT_ID, it) }
            action = Intent.ACTION_VIEW
            data = android.net.Uri.parse("armorclaw://room/$roomId")
        }
    }

    /**
     * Create notification channels (required for Android 8.0+)
     */
    fun createNotificationChannels(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = context.getSystemService(NotificationManager::class.java)

            // Messages channel
            val messagesChannel = NotificationChannel(
                CHANNEL_ID_MESSAGES,
                context.getString(R.string.channel_messages),
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = context.getString(R.string.channel_messages_desc)
                enableLights(true)
                lightColor = 0xFF00FF00.toInt()
                enableVibration(true)
            }

            // Mentions channel (higher priority)
            val mentionsChannel = NotificationChannel(
                CHANNEL_ID_MENTIONS,
                context.getString(R.string.channel_mentions),
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = context.getString(R.string.channel_mentions_desc)
                enableLights(true)
                lightColor = 0xFFFF0000.toInt()
                enableVibration(true)
                vibrationPattern = longArrayOf(0, 250, 250, 250)
            }

            // System channel (invites, etc.)
            val systemChannel = NotificationChannel(
                CHANNEL_ID_SYSTEM,
                context.getString(R.string.channel_system),
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = context.getString(R.string.channel_system_desc)
            }

            notificationManager.createNotificationChannels(
                listOf(messagesChannel, mentionsChannel, systemChannel)
            )
        }
    }
}
