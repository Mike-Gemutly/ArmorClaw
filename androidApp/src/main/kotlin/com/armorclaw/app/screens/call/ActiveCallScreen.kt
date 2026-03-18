package com.armorclaw.app.screens.call
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.platform.voice.AudioDevice
import com.armorclaw.shared.ui.theme.*

/**
 * Full-screen active call UI
 */
@Composable
fun ActiveCallScreen(
    callSession: CallSession,
    onEndCall: () -> Unit,
    onMuteToggle: () -> Unit,
    onSpeakerToggle: () -> Unit,
    onVideoToggle: () -> Unit,
    onHoldToggle: () -> Unit,
    onAudioDeviceSelect: (AudioDevice) -> Unit,
    modifier: Modifier = Modifier
) {
    val isConnecting = callSession.state == CallState.Connecting
    val isOnHold = callSession.state == CallState.OnHold
    val isActive = callSession.state == CallState.Active

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    colors = listOf(
                        Background,
                        Background.copy(alpha = 0.95f),
                        BrandPurple.copy(alpha = 0.1f)
                    )
                )
            )
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .navigationBarsPadding()
        ) {
            // Top bar
            CallTopBar(
                state = callSession.state,
                startTime = callSession.startedAt?.toEpochMilliseconds() ?: System.currentTimeMillis(),
                roomName = null // TODO: Get room name from session
            )

            // Main content
            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
            ) {
                when {
                    isOnHold -> OnHoldOverlay()
                    isConnecting -> ConnectingContent(onCancel = onEndCall)
                    isActive -> ActiveCallContent(
                        participants = callSession.participants,
                        isLocalMuted = callSession.isMuted
                    )
                }
            }

            // Bottom controls
            CallControls(
                isMuted = callSession.isMuted,
                isSpeakerOn = callSession.isSpeakerOn,
                isVideoEnabled = callSession.isLocalVideoEnabled,
                isOnHold = isOnHold,
                selectedAudioDevice = AudioDevice.SPEAKER_PHONE, // TODO: Get from session
                onMuteToggle = onMuteToggle,
                onSpeakerToggle = onSpeakerToggle,
                onVideoToggle = onVideoToggle,
                onHoldToggle = onHoldToggle,
                onEndCall = onEndCall,
                onAudioDeviceSelect = onAudioDeviceSelect,
                showVideoToggle = callSession.callType == CallType.VIDEO,
                modifier = Modifier.padding(bottom = 32.dp)
            )
        }
    }
}

@Composable
private fun CallTopBar(
    state: CallState,
    startTime: Long,
    roomName: String?
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        // Call type indicator
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Box(
                modifier = Modifier
                    .size(8.dp)
                    .clip(CircleShape)
                    .background(
                        when (state) {
                            CallState.Active -> BrandGreen
                            CallState.Connecting -> StatusWarning
                            CallState.OnHold -> StatusWarning
                            else -> OnBackground.copy(alpha = 0.5f)
                        }
                    )
            )

            Text(
                text = when (state) {
                    CallState.Active -> "Active Call"
                    CallState.Connecting -> "Connecting..."
                    CallState.OnHold -> "On Hold"
                    CallState.Ringing -> "Ringing..."
                    else -> "Call"
                },
                style = MaterialTheme.typography.titleSmall,
                color = OnBackground.copy(alpha = 0.7f)
            )
        }

        // Call Duration
        if (state == CallState.Active) {
            CallDurationLabel(startTime = startTime)
        }

        // Room name
        if (!roomName.isNullOrBlank()) {
            Text(
                text = roomName,
                style = MaterialTheme.typography.bodySmall,
                color = OnBackground.copy(alpha = 0.5f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false)
            )
        }
    }
}

@Composable
private fun ConnectingContent(
    onCancel: () -> Unit
) {
    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(48.dp),
            color = BrandPurple,
            strokeWidth = 4.dp
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "Connecting...",
            style = MaterialTheme.typography.titleMedium,
            color = OnBackground.copy(alpha = 0.7f)
        )

        Spacer(modifier = Modifier.height(24.dp))

        // Cancel button to abort connection attempt
        OutlinedButton(
            onClick = onCancel,
            colors = ButtonDefaults.outlinedButtonColors(
                contentColor = StatusError
            )
        ) {
            Icon(
                imageVector = Icons.Default.Close,
                contentDescription = null,
                modifier = Modifier.size(18.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text("Cancel")
        }
    }
}

@Composable
private fun ActiveCallContent(
    participants: List<CallParticipant>,
    isLocalMuted: Boolean
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        // Audio visualizer (for voice calls)
        AudioVisualizer(
            audioLevel = 0.5f, // TODO: Get actual audio level
            isActive = !isLocalMuted,
            modifier = Modifier.padding(vertical = 32.dp)
        )

        // Participants list
        if (participants.isNotEmpty()) {
            ParticipantsList(participants = participants)
        } else {
            Text(
                text = "Waiting for participants...",
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.5f)
            )
        }
    }
}

@Composable
private fun OnHoldOverlay() {
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(OnBackground.copy(alpha = 0.1f)),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Icon(
                imageVector = Icons.Default.Pause,
                contentDescription = "Call paused",
                tint = StatusWarning,
                modifier = Modifier.size(64.dp)
            )

            Text(
                text = "Call On Hold",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                color = OnBackground
            )

            Text(
                text = "Tap the hold button to resume",
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.6f)
            )
        }
    }
}

@Composable
private fun ParticipantsList(
    participants: List<CallParticipant>
) {
    LazyRow(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        contentPadding = PaddingValues(horizontal = 16.dp)
    ) {
        items(participants) { participant ->
            ParticipantCard(participant = participant)
        }
    }
}

@Composable
private fun ParticipantCard(
    participant: CallParticipant
) {
    Card(
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = OnBackground.copy(alpha = 0.05f)
        ),
        elevation = CardDefaults.cardElevation(
            defaultElevation = 0.dp
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            // Avatar
            Box(
                modifier = Modifier
                    .size(56.dp)
                    .clip(CircleShape)
                    .background(BrandPurple.copy(alpha = 0.2f)),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = participant.userId.take(2).uppercase(),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = BrandPurple
                )
            }

            // Name
            Text(
                text = participant.displayName ?: participant.userId.take(8),
                style = MaterialTheme.typography.titleSmall,
                color = OnBackground,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            // Status indicators
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                if (participant.isMuted) {
                    Icon(
                        imageVector = Icons.Default.MicOff,
                        contentDescription = "Muted",
                        tint = StatusWarning,
                        modifier = Modifier.size(16.dp)
                    )
                }

                if (participant.isSpeaking) {
                    Icon(
                        imageVector = Icons.Default.RecordVoiceOver,
                        contentDescription = "Speaking",
                        tint = BrandGreen,
                        modifier = Modifier.size(16.dp)
                    )
                }
            }
        }
    }
}

/**
 * Video call variant of active call screen
 */
@Composable
fun ActiveVideoCallScreen(
    callSession: CallSession,
    localVideoEnabled: Boolean,
    remoteVideoEnabled: Boolean,
    onEndCall: () -> Unit,
    onMuteToggle: () -> Unit,
    onVideoToggle: () -> Unit,
    onCameraSwitch: () -> Unit,
    modifier: Modifier = Modifier
) {
    var controlsVisible by remember { mutableStateOf(true) }

    LaunchedEffect(Unit) {
        // Auto-hide controls after 3 seconds
        while (true) {
            kotlinx.coroutines.delay(3000)
            controlsVisible = false
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(Color.Black)
    ) {
        // Remote video (full screen)
        Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = Alignment.Center
        ) {
            if (remoteVideoEnabled) {
                // TODO: Show remote video surface
                Text(
                    text = "Remote Video",
                    color = Color.White.copy(alpha = 0.5f)
                )
            } else {
                // Remote video disabled placeholder
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.VideocamOff,
                        contentDescription = null,
                        tint = Color.White.copy(alpha = 0.5f),
                        modifier = Modifier.size(48.dp)
                    )

                    Text(
                        text = "Video paused",
                        style = MaterialTheme.typography.bodyMedium,
                        color = Color.White.copy(alpha = 0.5f)
                    )
                }
            }
        }

        // Local video (picture-in-picture)
        if (localVideoEnabled) {
            LocalVideoPreview(
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(16.dp)
                    .size(120.dp)
            )
        }

        // Controls overlay
        AnimatedVisibility(
            visible = controlsVisible,
            enter = fadeIn(),
            exit = fadeOut(),
            modifier = Modifier.align(Alignment.BottomCenter)
        ) {
            VideoCallControls(
                isMuted = callSession.isMuted,
                isVideoEnabled = localVideoEnabled,
                onMuteToggle = onMuteToggle,
                onVideoToggle = onVideoToggle,
                onCameraSwitch = onCameraSwitch,
                onEndCall = onEndCall,
                modifier = Modifier.padding(bottom = 32.dp)
            )
        }

        // Tap to show/hide controls
        Box(
            modifier = Modifier
                .fillMaxSize()
                .clickableNoRipple { controlsVisible = !controlsVisible }
        )
    }
}

@Composable
private fun LocalVideoPreview(
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        elevation = CardDefaults.cardElevation(
            defaultElevation = 8.dp
        )
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(Color.Black),
            contentAlignment = Alignment.Center
        ) {
            // TODO: Show local video surface
            Icon(
                imageVector = Icons.Default.Person,
                contentDescription = null,
                tint = Color.White.copy(alpha = 0.5f),
                modifier = Modifier.size(32.dp)
            )
        }
    }
}

@Composable
private fun VideoCallControls(
    isMuted: Boolean,
    isVideoEnabled: Boolean,
    onMuteToggle: () -> Unit,
    onVideoToggle: () -> Unit,
    onCameraSwitch: () -> Unit,
    onEndCall: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .clip(RoundedCornerShape(28.dp))
            .background(Color.Black.copy(alpha = 0.7f))
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        CallControlButton(
            icon = if (isMuted) Icons.Default.MicOff else Icons.Default.Mic,
            isActive = isMuted,
            activeColor = StatusWarning,
            onClick = onMuteToggle,
            contentDescription = if (isMuted) "Unmute" else "Mute"
        )

        CallControlButton(
            icon = if (isVideoEnabled) Icons.Default.Videocam else Icons.Default.VideocamOff,
            isActive = !isVideoEnabled,
            activeColor = StatusWarning,
            onClick = onVideoToggle,
            contentDescription = if (isVideoEnabled) "Turn off video" else "Turn on video"
        )

        CallControlButton(
            icon = Icons.Default.FlipCameraAndroid,
            isActive = false,
            activeColor = BrandPurple,
            onClick = onCameraSwitch,
            contentDescription = "Switch camera"
        )

        CallControlButton(
            icon = Icons.Default.CallEnd,
            isActive = true,
            activeColor = StatusError,
            onClick = onEndCall,
            contentDescription = "End call"
        )
    }
}

private fun Modifier.clickableNoRipple(onClick: () -> Unit): Modifier =
    this.then(
        MutableInteractionSource().let { interactionSource ->
            clickable(
                interactionSource = interactionSource,
                indication = null,
                onClick = onClick
            )
        }
    )
