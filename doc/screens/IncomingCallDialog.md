# IncomingCallDialog

> Incoming call notification dialog
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/call/IncomingCallDialog.kt`

## Overview

IncomingCallDialog displays a full-screen dialog when receiving an incoming voice call, allowing the user to accept or decline the call.

## Functions

### IncomingCallDialog
```kotlin
@Composable
fun IncomingCallDialog(
    callerName: String,
    callerAvatar: String?,
    onAccept: () -> Unit,
    onDecline: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Full-screen incoming call UI with accept/decline actions.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `callerName` | `String` | Name of caller |
| `callerAvatar` | `String?` | Avatar URL |
| `onAccept` | `() -> Unit` | Accept call callback |
| `onDecline` | `() -> Unit` | Decline call callback |
| `modifier` | `Modifier` | Optional styling |

---

## Screen Layout

```
┌────────────────────────────────────┐
│                                    │
│         INCOMING CALL              │
│                                    │
│         ┌─────────┐                │
│         │   👤    │                │
│         │ Avatar  │                │
│         └─────────┘                │
│                                    │
│         Alice Johnson              │
│                                    │
│         🔒 End-to-end encrypted    │
│                                    │
│                                    │
│    ┌─────────┐    ┌─────────┐     │
│    │   📞    │    │   ✕     │     │
│    │ Accept  │    │ Decline │     │
│    │  (green)│    │  (red)  │     │
│    └─────────┘    └─────────┘     │
│                                    │
└────────────────────────────────────┘
```

---

## Implementation

```kotlin
@Composable
fun IncomingCallDialog(
    callerName: String,
    callerAvatar: String?,
    onAccept: () -> Unit,
    onDecline: () -> Unit,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "ring")

    val ringScale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = 1.1f,
        animationSpec = infiniteRepeatable(
            animation = tween(1000, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    Surface(
        modifier = modifier.fillMaxSize(),
        color = MaterialTheme.colorScheme.background
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.SpaceEvenly
        ) {
            // Header
            Text(
                text = "INCOMING CALL",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.primary,
                letterSpacing = 4.sp
            )

            // Caller info
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Avatar with pulse animation
                Box(
                    modifier = Modifier
                        .size(140.dp)
                        .scale(ringScale),
                    contentAlignment = Alignment.Center
                ) {
                    Surface(
                        modifier = Modifier.fillMaxSize(),
                        shape = CircleShape,
                        color = MaterialTheme.colorScheme.surfaceVariant
                    ) {
                        if (callerAvatar != null) {
                            AsyncImage(
                                model = callerAvatar,
                                contentDescription = "Caller avatar",
                                modifier = Modifier.fillMaxSize()
                            )
                        } else {
                            Box(
                                modifier = Modifier.fillMaxSize(),
                                contentAlignment = Alignment.Center
                            ) {
                                Text(
                                    text = callerName.firstOrNull()?.toString() ?: "?",
                                    style = MaterialTheme.typography.displayMedium,
                                    fontWeight = FontWeight.Bold
                                )
                            }
                        }
                    }
                }

                // Caller name
                Text(
                    text = callerName,
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold
                )
            }

            // Encryption indicator
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Lock,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(20.dp)
                )
                Text(
                    text = "End-to-end encrypted",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            // Action buttons
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly
            ) {
                // Decline button
                CallActionButton(
                    icon = Icons.Default.CallEnd,
                    label = "Decline",
                    backgroundColor = MaterialTheme.colorScheme.error,
                    onClick = onDecline
                )

                // Accept button
                CallActionButton(
                    icon = Icons.Default.Call,
                    label = "Accept",
                    backgroundColor = Color(0xFF4CAF50), // Green
                    onClick = onAccept
                )
            }
        }
    }
}
```

---

### CallActionButton
```kotlin
@Composable
private fun CallActionButton(
    icon: ImageVector,
    label: String,
    backgroundColor: Color,
    onClick: () -> Unit
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        FilledIconButton(
            onClick = onClick,
            modifier = Modifier.size(72.dp),
            colors = IconButtonDefaults.filledIconButtonColors(
                containerColor = backgroundColor
            )
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                modifier = Modifier.size(32.dp),
                tint = Color.White
            )
        }

        Text(
            text = label,
            style = MaterialTheme.typography.labelLarge
        )
    }
}
```

---

## Animations

### Ring Animation
- Avatar pulses between 1.0x and 1.1x scale
- 1 second animation duration
- Continuous repeat while ringing

### Button Ripple
- Material ripple effect on tap
- Scale feedback on press

---

## Call Decline Actions

### Quick Responses
```kotlin
@Composable
fun QuickResponseOptions(
    onSendResponse: (String) -> Unit
) {
    Column {
        TextButton(onClick = { onSendResponse("I'll call you back") }) {
            Text("I'll call you back")
        }
        TextButton(onClick = { onSendResponse("Can't talk right now") }) {
            Text("Can't talk right now")
        }
        TextButton(onClick = { onSendResponse("In a meeting") }) {
            Text("In a meeting")
        }
    }
}
```

---

## System Integration

### Full-Screen Intent
```kotlin
// In NotificationManager
fun showIncomingCallNotification(call: Call) {
    val fullScreenIntent = Intent(context, IncomingCallActivity::class.java).apply {
        putExtra("callId", call.id)
        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    }

    val fullScreenPendingIntent = PendingIntent.getActivity(
        context,
        0,
        fullScreenIntent,
        PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
    )

    val notification = NotificationCompat.Builder(context, CHANNEL_CALLS)
        .setSmallIcon(R.drawable.ic_call)
        .setContentTitle("Incoming call")
        .setContentText(call.callerName)
        .setPriority(NotificationCompat.PRIORITY_MAX)
        .setCategory(NotificationCompat.CATEGORY_CALL)
        .setFullScreenIntent(fullScreenPendingIntent, true)
        .build()

    notificationManager.notify(NOTIFICATION_ID, notification)
}
```

---

## Accessibility

### Content Descriptions
- Avatar: "Incoming call from [name]"
- Accept: "Accept call"
- Decline: "Decline call"

---

## Related Documentation

- [ActiveCallScreen](ActiveCallScreen.md) - Active call screen
- [CallControls](../components/CallControls.md) - Call control buttons
- [Voice Calls](../features/voice-calls.md) - Call feature
