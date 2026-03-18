package com.armorclaw.app.screens.call
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.armorclaw.shared.domain.model.CallType
import com.armorclaw.shared.ui.theme.*

/**
 * Full-screen incoming call dialog
 */
@Composable
fun IncomingCallDialog(
    callerName: String,
    callerAvatarUrl: String?,
    roomName: String?,
    callType: CallType,
    onAnswer: () -> Unit,
    onReject: () -> Unit,
    modifier: Modifier = Modifier,
    isVideo: Boolean = false
) {
    Dialog(
        onDismissRequest = onReject,
        properties = DialogProperties(
            dismissOnBackPress = false,
            dismissOnClickOutside = false,
            usePlatformDefaultWidth = false
        )
    ) {
        IncomingCallContent(
            callerName = callerName,
            callerAvatarUrl = callerAvatarUrl,
            roomName = roomName,
            callType = callType,
            onAnswer = onAnswer,
            onReject = onReject,
            modifier = modifier,
            isVideo = isVideo
        )
    }
}

@Composable
private fun IncomingCallContent(
    callerName: String,
    callerAvatarUrl: String?,
    roomName: String?,
    callType: CallType,
    onAnswer: () -> Unit,
    onReject: () -> Unit,
    modifier: Modifier,
    isVideo: Boolean
) {
    Box(
        modifier = modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    colors = listOf(
                        BrandPurple.copy(alpha = 0.9f),
                        BrandPurple.copy(alpha = 0.95f),
                        Color.Black
                    )
                )
            )
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .navigationBarsPadding(),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Spacer(modifier = Modifier.weight(1f))

            // Caller info
            CallerInfo(
                callerName = callerName,
                callerAvatarUrl = callerAvatarUrl,
                roomName = roomName,
                callType = callType
            )

            Spacer(modifier = Modifier.weight(1f))

            // Call status
            RingingIndicator()

            Spacer(modifier = Modifier.height(64.dp))

            // Action buttons
            IncomingCallActions(
                onAnswer = onAnswer,
                onReject = onReject,
                isVideo = isVideo
            )

            Spacer(modifier = Modifier.height(48.dp))
        }
    }
}

@Composable
private fun CallerInfo(
    callerName: String,
    callerAvatarUrl: String?,
    roomName: String?,
    callType: CallType
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Avatar
        CallerAvatar(
            avatarUrl = callerAvatarUrl,
            callerName = callerName
        )

        // Caller name
        Text(
            text = callerName,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = Color.White,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis
        )

        // Room name
        if (!roomName.isNullOrBlank()) {
            Text(
                text = roomName,
                style = MaterialTheme.typography.bodyLarge,
                color = Color.White.copy(alpha = 0.7f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
        }

        // Call type badge
        CallTypeBadge(callType = callType)
    }
}

@Composable
private fun CallerAvatar(
    avatarUrl: String?,
    callerName: String
) {
    var pulseScale by remember { mutableStateOf(1f) }

    LaunchedEffect(Unit) {
        while (true) {
            pulseScale = 1.05f
            kotlinx.coroutines.delay(1000)
            pulseScale = 1f
            kotlinx.coroutines.delay(1000)
        }
    }

    val scale by animateFloatAsState(
        targetValue = pulseScale,
        animationSpec = tween(500, easing = FastOutSlowInEasing),
        label = "pulse"
    )

    Box(
        modifier = Modifier
            .scale(scale)
            .size(120.dp)
            .clip(CircleShape)
            .background(Color.White.copy(alpha = 0.2f)),
        contentAlignment = Alignment.Center
    ) {
        if (avatarUrl.isNullOrBlank()) {
            // Placeholder avatar
            Text(
                text = callerName.take(2).uppercase(),
                style = ArmorClawTypography.h3,
                fontWeight = FontWeight.Bold,
                color = Color.White
            )
        } else {
            // TODO: Load avatar image with Coil
            // For now, show placeholder
            Text(
                text = callerName.take(2).uppercase(),
                style = ArmorClawTypography.h3,
                fontWeight = FontWeight.Bold,
                color = Color.White
            )
        }
    }
}

@Composable
private fun CallTypeBadge(
    callType: CallType
) {
    Surface(
        shape = RoundedCornerShape(20.dp),
        color = Color.White.copy(alpha = 0.2f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = when (callType) {
                    CallType.VIDEO -> Icons.Default.Videocam
                    CallType.VOICE -> Icons.Default.Phone
                },
                contentDescription = null,
                tint = Color.White,
                modifier = Modifier.size(18.dp)
            )

            Text(
                text = when (callType) {
                    CallType.VIDEO -> "Video Call"
                    CallType.VOICE -> "Voice Call"
                },
                style = MaterialTheme.typography.bodyMedium,
                color = Color.White
            )
        }
    }
}

@Composable
private fun RingingIndicator() {
    var dotCount by remember { mutableStateOf(0) }

    LaunchedEffect(Unit) {
        while (true) {
            kotlinx.coroutines.delay(500)
            dotCount = (dotCount + 1) % 4
        }
    }

    val dots = when (dotCount) {
        0 -> ""
        1 -> "."
        2 -> ".."
        else -> "..."
    }

    Text(
        text = "Incoming call$dots",
        style = MaterialTheme.typography.bodyLarge,
        color = Color.White.copy(alpha = 0.7f)
    )
}

@Composable
private fun IncomingCallActions(
    onAnswer: () -> Unit,
    onReject: () -> Unit,
    isVideo: Boolean
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Reject button
        CallActionCircleButton(
            icon = Icons.Default.CallEnd,
            label = "Decline",
            backgroundColor = StatusError,
            onClick = onReject
        )

        // Answer button
        CallActionCircleButton(
            icon = if (isVideo) Icons.Default.Videocam else Icons.Default.Call,
            label = if (isVideo) "Video" else "Answer",
            backgroundColor = BrandGreen,
            onClick = onAnswer
        )
    }
}

@Composable
private fun CallActionCircleButton(
    icon: ImageVector,
    label: String,
    backgroundColor: Color,
    onClick: () -> Unit
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        FAB(
            icon = icon,
            backgroundColor = backgroundColor,
            onClick = onClick
        )

        Text(
            text = label,
            style = MaterialTheme.typography.bodySmall,
            color = Color.White,
            fontWeight = FontWeight.Medium
        )
    }
}

@Composable
private fun FAB(
    icon: ImageVector,
    backgroundColor: Color,
    onClick: () -> Unit
) {
    var pressed by remember { mutableStateOf(false) }

    val scale by animateFloatAsState(
        targetValue = if (pressed) 0.9f else 1f,
        animationSpec = spring(stiffness = Spring.StiffnessLow),
        label = "scale"
    )

    FloatingActionButton(
        onClick = onClick,
        modifier = Modifier.scale(scale),
        shape = CircleShape,
        containerColor = backgroundColor,
        elevation = FloatingActionButtonDefaults.elevation(
            defaultElevation = 8.dp,
            pressedElevation = 4.dp
        )
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = Color.White,
            modifier = Modifier.size(28.dp)
        )
    }
}

/**
 * Mini incoming call notification (for notification shade)
 */
@Composable
fun IncomingCallMiniNotification(
    callerName: String,
    callType: CallType,
    onAnswer: () -> Unit,
    onReject: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = BrandPurple,
        tonalElevation = 4.dp,
        shadowElevation = 4.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = when (callType) {
                        CallType.VIDEO -> Icons.Default.Videocam
                        CallType.VOICE -> Icons.Default.Phone
                    },
                    contentDescription = null,
                    tint = Color.White
                )

                Column {
                    Text(
                        text = callerName,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                        color = Color.White
                    )

                    Text(
                        text = "Incoming ${callType.name.lowercase()} call",
                        style = MaterialTheme.typography.bodySmall,
                        color = Color.White.copy(alpha = 0.7f)
                    )
                }
            }

            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                IconButton(
                    onClick = onReject,
                    modifier = Modifier
                        .size(40.dp)
                        .background(StatusError, CircleShape)
                ) {
                    Icon(
                        imageVector = Icons.Default.CallEnd,
                        contentDescription = "Decline",
                        tint = Color.White,
                        modifier = Modifier.size(20.dp)
                    )
                }

                IconButton(
                    onClick = onAnswer,
                    modifier = Modifier
                        .size(40.dp)
                        .background(BrandGreen, CircleShape)
                ) {
                    Icon(
                        imageVector = Icons.Default.Call,
                        contentDescription = "Answer",
                        tint = Color.White,
                        modifier = Modifier.size(20.dp)
                    )
                }
            }
        }
    }
}
