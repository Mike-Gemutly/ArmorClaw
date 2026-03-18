package com.armorclaw.app.screens.call
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.animation.animateColorAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
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
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.interaction.collectIsHoveredAsState
import com.armorclaw.shared.platform.voice.AudioDevice
import com.armorclaw.shared.ui.theme.*

/**
 * Call control buttons for active calls
 */
@Composable
fun CallControls(
    isMuted: Boolean,
    isSpeakerOn: Boolean,
    isVideoEnabled: Boolean,
    isOnHold: Boolean,
    selectedAudioDevice: AudioDevice?,
    onMuteToggle: () -> Unit,
    onSpeakerToggle: () -> Unit,
    onVideoToggle: () -> Unit,
    onHoldToggle: () -> Unit,
    onEndCall: () -> Unit,
    onAudioDeviceSelect: (AudioDevice) -> Unit,
    modifier: Modifier = Modifier,
    showVideoToggle: Boolean = false
) {
    val haptic = LocalHapticFeedback.current

    Column(
        modifier = modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(24.dp)
    ) {
        // Secondary controls row
        Row(
            horizontalArrangement = Arrangement.spacedBy(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Mute button
            CallControlButton(
                icon = if (isMuted) Icons.Default.MicOff else Icons.Default.Mic,
                isActive = isMuted,
                activeColor = StatusWarning,
                onClick = {
                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                    onMuteToggle()
                },
                contentDescription = if (isMuted) "Unmute" else "Mute"
            )

            // Speaker button
            CallControlButton(
                icon = if (isSpeakerOn) Icons.Default.VolumeUp else Icons.Default.VolumeDown,
                isActive = isSpeakerOn,
                activeColor = BrandPurple,
                onClick = {
                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                    onSpeakerToggle()
                },
                contentDescription = if (isSpeakerOn) "Turn off speaker" else "Turn on speaker"
            )

            // Video toggle (if supported)
            if (showVideoToggle) {
                CallControlButton(
                    icon = if (isVideoEnabled) Icons.Default.Videocam else Icons.Default.VideocamOff,
                    isActive = isVideoEnabled,
                    activeColor = BrandGreen,
                    onClick = {
                        haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                        onVideoToggle()
                    },
                    contentDescription = if (isVideoEnabled) "Turn off video" else "Turn on video"
                )
            }

            // Hold button
            CallControlButton(
                icon = Icons.Default.Pause,
                isActive = isOnHold,
                activeColor = StatusWarning,
                onClick = {
                    haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                    onHoldToggle()
                },
                contentDescription = if (isOnHold) "Resume call" else "Hold call"
            )
        }

        // Audio device selector
        AudioDeviceSelector(
            selectedDevice = selectedAudioDevice,
            onDeviceSelect = onAudioDeviceSelect
        )

        // End call button
        EndCallButton(
            onClick = {
                haptic.performHapticFeedback(HapticFeedbackType.LongPress)
                onEndCall()
            }
        )
    }
}

@Composable
fun CallControlButton(
    icon: ImageVector,
    isActive: Boolean,
    activeColor: Color,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    contentDescription: String? = null,
    enabled: Boolean = true,
    disabledReason: String? = null
) {
    val backgroundColor by animateColorAsState(
        targetValue = if (isActive) activeColor.copy(alpha = 0.2f) else OnBackground.copy(alpha = 0.1f),
        label = "background"
    )

    val iconColor by animateColorAsState(
        targetValue = if (isActive) activeColor else OnBackground.copy(alpha = 0.8f),
        label = "icon_color"
    )

    val borderColor by animateColorAsState(
        targetValue = if (isActive) activeColor.copy(alpha = 0.5f) else Color.Transparent,
        label = "border"
    )

    // Simple disabled button (tooltip removed for Material3 1.1.2 compatibility)
    Box(
        modifier = modifier
            .size(64.dp)
            .clip(CircleShape)
            .background(backgroundColor)
            .border(2.dp, borderColor, CircleShape)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                enabled = enabled,
                onClick = onClick
            ),
        contentAlignment = Alignment.Center
    ) {
        Icon(
            imageVector = icon,
            contentDescription = if (!enabled && disabledReason != null) {
                "$contentDescription ($disabledReason)"
            } else {
                contentDescription
            },
            tint = if (enabled) iconColor else OnBackground.copy(alpha = 0.3f),
            modifier = Modifier.size(28.dp)
        )
    }
}

@Composable
private fun EndCallButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    var pressed by remember { mutableStateOf(false) }

    val scale by animateFloatAsState(
        targetValue = if (pressed) 0.9f else 1f,
        animationSpec = spring(stiffness = Spring.StiffnessLow),
        label = "scale"
    )

    Box(
        modifier = modifier
            .size(72.dp)
            .scale(scale)
            .clip(CircleShape)
            .background(StatusError)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick
            ),
        contentAlignment = Alignment.Center
    ) {
        Icon(
            imageVector = Icons.Default.CallEnd,
            contentDescription = "End call",
            tint = Color.White,
            modifier = Modifier.size(32.dp)
        )
    }
}

@Composable
private fun AudioDeviceSelector(
    selectedDevice: AudioDevice?,
    onDeviceSelect: (AudioDevice) -> Unit,
    modifier: Modifier = Modifier
) {
    var expanded by remember { mutableStateOf(false) }

    Card(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = OnBackground.copy(alpha = 0.08f)
        ),
        elevation = CardDefaults.cardElevation(
            defaultElevation = 0.dp
        )
    ) {
        Column {
            Row(
                modifier = Modifier
                    .clickable { expanded = !expanded }
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = getAudioDeviceIcon(selectedDevice),
                    contentDescription = null,
                    tint = OnBackground.copy(alpha = 0.7f),
                    modifier = Modifier.size(20.dp)
                )

                Text(
                    text = getAudioDeviceLabel(selectedDevice),
                    style = MaterialTheme.typography.bodyMedium,
                    color = OnBackground.copy(alpha = 0.8f),
                    modifier = Modifier.weight(1f)
                )

                Icon(
                    imageVector = if (expanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore,
                    contentDescription = null,
                    tint = OnBackground.copy(alpha = 0.5f)
                )
            }

            AnimatedVisibility(
                visible = expanded,
                enter = expandVertically() + fadeIn(),
                exit = shrinkVertically() + fadeOut()
            ) {
                Column {
                    Divider(
                        color = OnBackground.copy(alpha = 0.1f),
                        thickness = 1.dp
                    )

                    AudioDevice.values().forEach { device ->
                        if (device != AudioDevice.UNKNOWN) {
                            AudioDeviceOption(
                                device = device,
                                isSelected = device == selectedDevice,
                                onClick = {
                                    onDeviceSelect(device)
                                    expanded = false
                                }
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AudioDeviceOption(
    device: AudioDevice,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    val backgroundColor by animateColorAsState(
        targetValue = if (isSelected) BrandPurple.copy(alpha = 0.15f) else Color.Transparent,
        label = "background"
    )

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(backgroundColor)
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = getAudioDeviceIcon(device),
            contentDescription = null,
            tint = if (isSelected) BrandPurple else OnBackground.copy(alpha = 0.6f),
            modifier = Modifier.size(20.dp)
        )

        Text(
            text = getAudioDeviceLabel(device),
            style = MaterialTheme.typography.bodyMedium,
            color = if (isSelected) BrandPurple else OnBackground.copy(alpha = 0.8f),
            modifier = Modifier.weight(1f)
        )

        if (isSelected) {
            Icon(
                imageVector = Icons.Default.Check,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(20.dp)
            )
        }
    }
}

@Composable
fun CallDurationLabel(
    startTime: Long,
    modifier: Modifier = Modifier
) {
    var duration by remember { mutableStateOf(0L) }

    LaunchedEffect(startTime) {
        while (true) {
            duration = System.currentTimeMillis() - startTime
            kotlinx.coroutines.delay(1000)
        }
    }

    Text(
        text = formatCallDuration(duration),
        style = MaterialTheme.typography.titleMedium,
        fontWeight = FontWeight.Medium,
        color = OnBackground.copy(alpha = 0.8f),
        modifier = modifier
    )
}

@Composable
fun CallStatusIndicator(
    stateText: String,
    isConnecting: Boolean,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        if (isConnecting) {
            CircularProgressIndicator(
                modifier = Modifier.size(16.dp),
                color = BrandPurple,
                strokeWidth = 2.dp
            )
        }

        Text(
            text = stateText,
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f)
        )
    }
}

private fun getAudioDeviceIcon(device: AudioDevice?): ImageVector = when (device) {
    AudioDevice.EARPIECE -> Icons.Default.PhoneAndroid
    AudioDevice.SPEAKER_PHONE -> Icons.Default.VolumeUp
    AudioDevice.WIRED_HEADSET -> Icons.Default.Headset
    AudioDevice.BLUETOOTH -> Icons.Default.Bluetooth
    else -> Icons.Default.VolumeUp
}

private fun getAudioDeviceLabel(device: AudioDevice?): String = when (device) {
    AudioDevice.EARPIECE -> "Earpiece"
    AudioDevice.SPEAKER_PHONE -> "Speaker"
    AudioDevice.WIRED_HEADSET -> "Wired Headset"
    AudioDevice.BLUETOOTH -> "Bluetooth"
    else -> "Unknown"
}

private fun formatCallDuration(millis: Long): String {
    val seconds = (millis / 1000) % 60
    val minutes = (millis / (1000 * 60)) % 60
    val hours = millis / (1000 * 60 * 60)

    return if (hours > 0) {
        String.format("%02d:%02d:%02d", hours, minutes, seconds)
    } else {
        String.format("%02d:%02d", minutes, seconds)
    }
}
