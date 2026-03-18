package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.platform.tts.TtsEvent
import com.armorclaw.shared.platform.tts.TtsPriority
import com.armorclaw.shared.platform.tts.VoiceModeConfig
import com.armorclaw.shared.platform.tts.toSpokenText
import kotlinx.coroutines.delay

/**
 * Voice Mode Indicator
 *
 * Visual indicator showing voice-first mode status and current TTS activity.
 * Displays when voice mode is enabled with animated speaking state.
 *
 * ## Architecture
 * ```
 * VoiceModeIndicator
 *      ├── VoiceModeChip (compact status)
 *      └── VoiceModePanel (expanded controls)
 *          ├── Enable/disable toggle
 *          ├── Speech rate slider
 *          ├── Pitch slider
 *          └── Announcement toggles
 * ```
 *
 * ## Usage
 * ```kotlin
 * val voiceMode by viewModel.voiceModeConfig.collectAsState()
 * val isSpeaking by viewModel.isTtsSpeaking.collectAsState()
 *
 * VoiceModeIndicator(
 *     config = voiceMode,
 *     isSpeaking = isSpeaking,
 *     currentEvent = viewModel.currentTtsEvent,
 *     onConfigChange = { viewModel.updateVoiceMode(it) }
 * )
 * ```
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VoiceModeIndicator(
    config: VoiceModeConfig,
    isSpeaking: Boolean,
    modifier: Modifier = Modifier,
    currentEvent: TtsEvent? = null,
    onConfigChange: (VoiceModeConfig) -> Unit = {},
    onExpandClick: () -> Unit = {}
) {
    if (!config.enabled && !isSpeaking) {
        // Compact toggle chip when disabled
        VoiceModeToggleChip(
            enabled = false,
            onClick = { onConfigChange(config.copy(enabled = true)) },
            modifier = modifier
        )
    } else {
        // Active indicator with speaking animation
        VoiceModeActiveChip(
            config = config,
            isSpeaking = isSpeaking,
            currentEvent = currentEvent,
            onClick = onExpandClick,
            modifier = modifier
        )
    }
}

/**
 * Compact toggle chip for enabling voice mode
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun VoiceModeToggleChip(
    enabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    FilterChip(
        selected = enabled,
        onClick = onClick,
        label = { Text("Voice") },
        leadingIcon = {
            Icon(
                imageVector = Icons.Default.RecordVoiceOver,
                contentDescription = null,
                modifier = Modifier.size(16.dp)
            )
        },
        modifier = modifier
    )
}

/**
 * Active voice mode chip with speaking animation
 */
@Composable
private fun VoiceModeActiveChip(
    config: VoiceModeConfig,
    isSpeaking: Boolean,
    currentEvent: TtsEvent?,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "voice_pulse")

    val pulseAlpha by infiniteTransition.animateFloat(
        initialValue = 0.4f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(600, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse"
    )

    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(20.dp),
        color = if (isSpeaking) {
            MaterialTheme.colorScheme.primaryContainer
        } else {
            MaterialTheme.colorScheme.surfaceVariant
        }
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            // Animated voice icon
            Box(
                modifier = Modifier.size(20.dp),
                contentAlignment = Alignment.Center
            ) {
                if (isSpeaking) {
                    // Sound waves animation
                    Box(
                        modifier = Modifier
                            .size(20.dp)
                            .graphicsLayer { alpha = pulseAlpha }
                            .clip(CircleShape)
                            .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.3f))
                    )
                }
                Icon(
                    imageVector = if (isSpeaking) Icons.Default.VolumeUp else Icons.Default.VolumeUp,
                    contentDescription = if (isSpeaking) "Speaking" else "Voice mode enabled",
                    tint = if (isSpeaking) {
                        MaterialTheme.colorScheme.primary
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    },
                    modifier = Modifier.size(18.dp)
                )
            }

            // Speaking indicator or label
            if (isSpeaking && currentEvent != null) {
                Text(
                    text = getShortDescription(currentEvent),
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.widthIn(max = 80.dp)
                )
            } else {
                Text(
                    text = "Voice",
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

/**
 * Expanded voice mode panel with controls
 */
@Composable
fun VoiceModePanel(
    config: VoiceModeConfig,
    isSpeaking: Boolean,
    currentEvent: TtsEvent? = null,
    onConfigChange: (VoiceModeConfig) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        elevation = CardDefaults.cardElevation(defaultElevation = 4.dp)
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Header
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.RecordVoiceOver,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary
                    )
                    Text(
                        text = "Voice Mode",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                }

                // Enable/disable toggle
                Switch(
                    checked = config.enabled,
                    onCheckedChange = { onConfigChange(config.copy(enabled = it)) }
                )
            }

            if (config.enabled) {
                // Current announcement
                if (isSpeaking && currentEvent != null) {
                    CurrentAnnouncementCard(
                        event = currentEvent,
                        config = config
                    )
                }

                // Speech rate slider
                Column {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text(
                            text = "Speech Rate",
                            style = MaterialTheme.typography.bodyMedium
                        )
                        Text(
                            text = "${(config.speechRate * 100).toInt()}%",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.primary,
                            fontWeight = FontWeight.Medium
                        )
                    }
                    Slider(
                        value = config.speechRate,
                        onValueChange = { onConfigChange(config.copy(speechRate = it)) },
                        valueRange = 0.5f..2.0f,
                        steps = 5
                    )
                }

                // Pitch slider
                Column {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text(
                            text = "Pitch",
                            style = MaterialTheme.typography.bodyMedium
                        )
                        Text(
                            text = "${(config.pitch * 100).toInt()}%",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.primary,
                            fontWeight = FontWeight.Medium
                        )
                    }
                    Slider(
                        value = config.pitch,
                        onValueChange = { onConfigChange(config.copy(pitch = it)) },
                        valueRange = 0.5f..2.0f,
                        steps = 5
                    )
                }

                // Announcement toggles
                Text(
                    text = "Announcements",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                AnnouncementToggleRow(
                    label = "Agent Status",
                    checked = config.announceAgentStatus,
                    onCheckedChange = { onConfigChange(config.copy(announceAgentStatus = it)) }
                )

                AnnouncementToggleRow(
                    label = "Interventions",
                    checked = config.announceInterventions,
                    onCheckedChange = { onConfigChange(config.copy(announceInterventions = it)) }
                )

                AnnouncementToggleRow(
                    label = "PII Requests",
                    checked = config.announcePiiRequests,
                    onCheckedChange = { onConfigChange(config.copy(announcePiiRequests = it)) }
                )

                AnnouncementToggleRow(
                    label = "Errors",
                    checked = config.announceErrors,
                    onCheckedChange = { onConfigChange(config.copy(announceErrors = it)) }
                )

                AnnouncementToggleRow(
                    label = "Task Completions",
                    checked = config.announceCompletions,
                    onCheckedChange = { onConfigChange(config.copy(announceCompletions = it)) }
                )

                // Verbose mode toggle
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column {
                        Text(
                            text = "Verbose Mode",
                            style = MaterialTheme.typography.bodyMedium
                        )
                        Text(
                            text = "More detailed announcements",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    Switch(
                        checked = config.verboseMode,
                        onCheckedChange = { onConfigChange(config.copy(verboseMode = it)) }
                    )
                }
            }
        }
    }
}

@Composable
private fun CurrentAnnouncementCard(
    event: TtsEvent,
    config: VoiceModeConfig
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            // Speaking animation
            SpeakingDots()

            Text(
                text = event.toSpokenText(config),
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f)
            )
        }
    }
}

@Composable
private fun SpeakingDots() {
    val infiniteTransition = rememberInfiniteTransition(label = "speaking_dots")

    Row(
        horizontalArrangement = Arrangement.spacedBy(3.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        repeat(3) { index ->
            val scale by infiniteTransition.animateFloat(
                initialValue = 0.5f,
                targetValue = 1f,
                animationSpec = infiniteRepeatable(
                    animation = tween(300, easing = LinearEasing, delayMillis = index * 100),
                    repeatMode = RepeatMode.Reverse
                ),
                label = "dot_$index"
            )

            Box(
                modifier = Modifier
                    .size(6.dp)
                    .scale(scale)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primary)
            )
        }
    }
}

@Composable
private fun AnnouncementToggleRow(
    label: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium
        )
        Checkbox(
            checked = checked,
            onCheckedChange = onCheckedChange
        )
    }
}

/**
 * Get short description for current event
 */
private fun getShortDescription(event: TtsEvent): String {
    return when (event) {
        is TtsEvent.AgentStatus -> "${event.agentName}: ${event.status}"
        is TtsEvent.InterventionRequired -> event.type
        is TtsEvent.PiiRequest -> event.fieldName
        is TtsEvent.TaskComplete -> event.taskName
        is TtsEvent.Error -> "Error"
        is TtsEvent.SystemAnnouncement -> "System"
    }
}

/**
 * Floating voice mode button for quick access
 */
@Composable
fun VoiceModeFloatingButton(
    config: VoiceModeConfig,
    isSpeaking: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "voice_fab")

    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (isSpeaking) 1.1f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    FloatingActionButton(
        onClick = onClick,
        modifier = modifier.graphicsLayer {
            scaleX = scale
            scaleY = scale
        },
        containerColor = if (config.enabled) {
            if (isSpeaking) {
                MaterialTheme.colorScheme.primary
            } else {
                MaterialTheme.colorScheme.secondaryContainer
            }
        } else {
            MaterialTheme.colorScheme.surfaceVariant
        }
    ) {
        Icon(
            imageVector = if (isSpeaking) Icons.Default.VolumeUp else Icons.Default.RecordVoiceOver,
            contentDescription = if (config.enabled) "Voice mode enabled" else "Enable voice mode",
            tint = if (config.enabled && isSpeaking) {
                MaterialTheme.colorScheme.onPrimary
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            }
        )
    }
}

/**
 * Preview helper
 */
@Composable
fun VoiceModeIndicatorPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        // Disabled state
        VoiceModeIndicator(
            config = VoiceModeConfig.DEFAULT,
            isSpeaking = false
        )

        // Enabled, not speaking
        VoiceModeIndicator(
            config = VoiceModeConfig.DEFAULT.copy(enabled = true),
            isSpeaking = false
        )

        // Speaking
        VoiceModeIndicator(
            config = VoiceModeConfig.DEFAULT.copy(enabled = true),
            isSpeaking = true,
            currentEvent = TtsEvent.InterventionRequired(
                type = "CAPTCHA",
                context = "reCAPTCHA detected on checkout page"
            )
        )

        // Full panel
        VoiceModePanel(
            config = VoiceModeConfig.ACCESSIBILITY,
            isSpeaking = true,
            currentEvent = TtsEvent.PiiRequest(
                fieldName = "Credit Card",
                sensitivity = "HIGH"
            ),
            onConfigChange = {},
            onDismiss = {}
        )
    }
}
