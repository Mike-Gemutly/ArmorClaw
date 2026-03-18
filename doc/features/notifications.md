# Notifications Feature

> Push notifications and notification management
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/platform/`

## Overview

The notifications feature handles push notifications for new messages, mentions, calls, and other app events. It includes notification channels, preferences, and rich notification support.

## Feature Components

### NotificationManager Service
**Location:** `platform/NotificationManager.kt`

Platform service for managing notifications.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `showNotification()` | Display notification | `notification` |
| `cancelNotification()` | Remove notification | `id` |
| `cancelAll()` | Clear all notifications | - |
| `createChannel()` | Create notification channel | `channel` |
| `updateBadge()` | Update app badge count | `count` |

#### Notification Types
| Type | Channel | Priority | Sound |
|------|---------|----------|-------|
| Message | messages | HIGH | Default |
| Mention | mentions | HIGH | Alert |
| Call | calls | MAX | Ringtone |
| System | system | DEFAULT | None |

---

### Notification Channels

#### Channel Configuration
```kotlin
data class NotificationChannel(
    val id: String,
    val name: String,
    val description: String,
    val importance: Int,
    val sound: Uri?,
    val vibration: Boolean,
    val lights: Boolean,
    val badge: Boolean
)
```

#### Default Channels
| Channel ID | Name | Importance |
|------------|------|------------|
| messages | Messages | HIGH |
| mentions | Mentions | HIGH |
| calls | Voice Calls | MAX |
| system | System | DEFAULT |

---

## Notification Types

### Message Notification
```
┌────────────────────────────────────┐
│ 👤 Alice                    2:30 PM│
│                                    │
│ Hey! Did you see the new design?   │
│                                    │
│ [Reply]  [Mark as Read]            │
└────────────────────────────────────┘
```

### Mention Notification
```
┌────────────────────────────────────┐
│ 💬 #general                 2:30 PM│
│                                    │
│ @John check this out!              │
│                                    │
│ [Reply]  [View in Chat]            │
└────────────────────────────────────┘
```

### Call Notification
```
┌────────────────────────────────────┐
│ 📞 Incoming Call                   │
│                                    │
│ Alice is calling you               │
│                                    │
│   [Decline]      [Answer]          │
└────────────────────────────────────┘
```

---

## Rich Notifications

### Actions
| Action | Notification Type | Behavior |
|--------|-------------------|----------|
| Reply | Message, Mention | Direct reply inline |
| Mark as Read | Message | Dismisses notification |
| View | All | Opens to message |
| Answer | Call | Accepts call |
| Decline | Call | Rejects call |

### Direct Reply
```kotlin
val replyIntent = Intent(context, ReplyReceiver::class.java)
val replyPendingIntent = PendingIntent.getBroadcast(
    context,
    0,
    replyIntent,
    PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE
)

val remoteInput = RemoteInput.Builder(KEY_REPLY)
    .setLabel("Reply")
    .build()

val replyAction = NotificationCompat.Action.Builder(
    R.drawable.ic_reply,
    "Reply",
    replyPendingIntent
).addRemoteInput(remoteInput).build()
```

---

## Notification Settings

### NotificationSettingsScreen
**Location:** `settings/NotificationSettingsScreen.kt`

User-configurable notification preferences.

#### Settings Categories
| Category | Options |
|----------|---------|
| General | Enable/disable all |
| Messages | Sound, vibration, preview |
| Mentions | Sound, vibration |
| Calls | Ringtone, vibration |
| Quiet Hours | Schedule, exceptions |

#### Settings Layout
```
┌────────────────────────────────────┐
│  ← Notifications                   │
├────────────────────────────────────┤
│  GENERAL                           │
│  ├─ Notifications         [ON]     │
│  ├─ Sound                 [ON]     │
│  └─ Vibration             [ON]     │
├────────────────────────────────────┤
│  MESSAGES                          │
│  ├─ Message Notifications [ON]     │
│  ├─ Show Preview          [ON]     │
│  └─ Sound                 Default  │
├────────────────────────────────────┤
│  MENTIONS                          │
│  ├─ Notify on @mention   [ON]      │
│  └─ Sound                 Alert    │
├────────────────────────────────────┤
│  CALLS                             │
│  ├─ Incoming Calls       [ON]      │
│  └─ Ringtone              Default  │
├────────────────────────────────────┤
│  QUIET HOURS                       │
│  ├─ Enable               [OFF]     │
│  ├─ Start Time           10:00 PM  │
│  └─ End Time             7:00 AM   │
└────────────────────────────────────┘
```

---

## Data Models

### AppNotification
```kotlin
data class AppNotification(
    val id: Int,
    val type: NotificationType,
    val title: String,
    val body: String,
    val roomId: String?,
    val messageId: String?,
    val senderId: String?,
    val senderName: String?,
    val senderAvatar: String?,
    val timestamp: Instant,
    val channelId: String,
    val priority: Int,
    val actions: List<NotificationAction>
)

enum class NotificationType {
    MESSAGE,
    MENTION,
    CALL_INCOMING,
    CALL_MISSED,
    SYSTEM,
    DEVICE_VERIFICATION
}
```

### NotificationPreferences
```kotlin
data class NotificationPreferences(
    val enabled: Boolean,
    val messageNotifications: Boolean,
    val mentionNotifications: Boolean,
    val callNotifications: Boolean,
    val showPreview: Boolean,
    val soundEnabled: Boolean,
    val vibrationEnabled: Boolean,
    val quietHoursEnabled: Boolean,
    val quietHoursStart: LocalTime?,
    val quietHoursEnd: LocalTime?
)
```

---

## State Management

### NotificationState
```kotlin
data class NotificationState(
    val preferences: NotificationPreferences,
    val channels: List<NotificationChannelInfo>,
    val isLoading: Boolean,
    val error: String?
)
```

### NotificationActions
| Action | Description |
|--------|-------------|
| `loadPreferences()` | Fetch user preferences |
| `updatePreferences(prefs)` | Save preferences |
| `testChannel(channelId)` | Send test notification |
| `openSystemSettings()` | Open Android notification settings |

---

## Push Notification Handling

### FCM Integration
```kotlin
class ArmorClawMessagingService : FirebaseMessagingService() {
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        val notification = parseNotification(remoteMessage)
        notificationManager.showNotification(notification)
    }

    override fun onNewToken(token: String) {
        // Send token to server
        repository.updatePushToken(token)
    }
}
```

### Dual Push Registration (NEW 2026-02-24)

FCM tokens are now registered with **both** the Matrix Homeserver and Bridge Server:

```kotlin
// PushNotificationRepositoryImpl registers both channels
suspend fun registerToken(pushToken: String, ...) {
    // Channel 1: Matrix Homeserver (Sygnal push gateway)
    matrixClient.setPusher(
        pushKey = pushToken,
        appId = "com.armorclaw.app",
        pushGatewayUrl = "https://push.armorclaw.app/_matrix/push/v1/notify"
    )

    // Channel 2: Bridge Server (SDTW bridging events)
    rpcClient.pushRegister(pushToken, "fcm", deviceId)
}
```

**Why dual registration?**
- Matrix-native events (mentions, DMs, invites) originate from the homeserver → requires Matrix pusher
- SDTW bridging events (Slack, Discord, Teams, WhatsApp) originate from the bridge → requires Bridge RPC
- Partial failure is handled: if one channel fails, the other still delivers push notifications

### Notification Payload
```json
{
  "data": {
    "type": "message",
    "roomId": "room_123",
    "messageId": "msg_456",
    "senderId": "user_789",
    "senderName": "Alice"
  },
  "notification": {
    "title": "Alice",
    "body": "Hey! Check this out..."
  }
}
```

---

## Quiet Hours

### Implementation
```kotlin
fun isInQuietHours(preferences: NotificationPreferences): Boolean {
    if (!preferences.quietHoursEnabled) return false

    val now = LocalTime.now()
    val start = preferences.quietHoursStart ?: return false
    val end = preferences.quietHoursEnd ?: return false

    return if (start.isBefore(end)) {
        now.isAfter(start) && now.isBefore(end)
    } else {
        now.isAfter(start) || now.isBefore(end)
    }
}
```

### Exceptions
- Calls can bypass quiet hours (configurable)
- Urgent mentions can bypass (configurable)
- System notifications always show

---

## Badge Management

### App Icon Badge
```kotlin
fun updateBadge(count: Int) {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
        val badgeCount = minOf(count, 999)
        ShortcutBadger.applyCount(context, badgeCount)
    }
}
```

### Badge Behavior
- Shows total unread message count
- Updates on message read
- Maximum display: 999+

---

## Related Documentation

- [Offline Sync](offline-sync.md) - Background sync
- [Settings](settings.md) - App settings
- [Voice Calls](voice-calls.md) - Call notifications
