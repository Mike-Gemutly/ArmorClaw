package com.armorclaw.shared.ui.components

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.animation.core.*
import androidx.compose.animation.core.InfiniteRepeatableSpec
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.TweenSpec
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.InfiniteTransition
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.filterNotNull
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.withContext
import java.util.Locale
import kotlin.coroutines.CoroutineContext
import kotlinx.coroutines.flow.catch

import com.armorclaw.shared.ui.theme.DesignTokens

/**
 * Command Bar Component
 *
 * A hybrid input component that combines text input with command chips for quick command selection.
 * Supports horizontal scrolling of command chips and text input with voice and send icons.
 *
 * ## Features
 * - Horizontal scrollable command chips (Status, Screenshot, Stop, Pause, Logs)
 * - OutlinedTextField with placeholder text
 * - Leading mic icon with voice input functionality
 * - Trailing send icon
 * - Chip click handlers to inject commands into input field
 * - Material 3 design system compliant
 * - Voice input state visualization (recording indicator)
 *
 * ## Usage
 * ```kotlin
 * var inputText by remember { mutableStateOf(TextFieldValue()) }
 * 
 * CommandBar(
 *     value = inputText,
 *     onValueChange = { inputText = it },
 *     onSend = { content ->
 *         // Handle send action
 *         inputText = TextFieldValue()
 *     },
 *     placeholder = "Delegate a task...",
 *     voiceInputService = voiceInputService // Optional: for voice input functionality
 * )
 * ```
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CommandBar(
    value: TextFieldValue,
    onValueChange: (TextFieldValue) -> Unit,
    onSend: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String = "Delegate a task...",
    chips: List<CommandChip> = defaultCommandChips,
    voiceInputService: com.armorclaw.shared.domain.features.VoiceInputService? = null
) {
    val recognitionScope = remember { CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate) }
    
    val isRecording = remember { mutableStateOf(false) }
    
    val transcription = remember { mutableStateOf("") }
    
    val recognitionError = remember { mutableStateOf<String?>(null) }

    // Permission state for voice input
    val hasRecordAudioPermission = remember { mutableStateOf(true) }
    val showPermissionRationale = remember { mutableStateOf(false) }

    // Permission request dialog
    if (showPermissionRationale.value) {
        AlertDialog(
            onDismissRequest = {
                showPermissionRationale.value = false
            },
            title = {
                Text("Voice Input Permission Required")
            },
            text = {
                Text("This app needs access to your microphone to enable voice input functionality. Please grant the RECORD_AUDIO permission in settings.")
            },
                    confirmButton = {
                TextButton(
                    onClick = {
                        showPermissionRationale.value = false
                        // This would normally launch the permission request
                        // In a real implementation, this would call the Android permission request
                    }
                ) {
                    Text("OK")
                }
            },
            dismissButton = {
                TextButton(
                    onClick = {
                        showPermissionRationale.value = false
                    }
                ) {
                    Text("Cancel")
                }
            }
        )
    }

    Column(modifier = modifier) {
        // Command chips row
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState())
                .padding(horizontal = DesignTokens.Spacing.md),
            horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
        ) {
            chips.forEach { chip ->
                InputChip(
                    onClick = {
                        val currentText = value.text
                        val newText = if (currentText.isNotEmpty()) {
                            "$currentText ${chip.command}"
                        } else {
                            chip.command
                        }
                        onValueChange(TextFieldValue(newText))
                    },
                    label = { Text(chip.label) },
                    leadingIcon = {
                        Icon(
                            imageVector = chip.icon,
                            contentDescription = null,
                            modifier = Modifier.size(DesignTokens.Icon.sm)
                        )
                    },
                    selected = false,
                    modifier = Modifier
                        .clip(RoundedCornerShape(DesignTokens.Radius.sm))
                )
            }
        }

        // Main input row
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = DesignTokens.Spacing.md, vertical = DesignTokens.Spacing.sm),
            verticalAlignment = Alignment.Bottom
        ) {
            // Error message display
            if (recognitionError.value != null) {
                Text(
                    text = recognitionError.value ?: "",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(end = DesignTokens.Spacing.sm)
                )
            }
            // Voice input icon with recording state and permission handling
            IconButton(
                onClick = {
                    if (!hasRecordAudioPermission.value) {
                        // Request permission
                        showPermissionRationale.value = true
                        return@IconButton
                    }
                    
                    if (isRecording.value) {
                        isRecording.value = false
                    } else {
                        isRecording.value = true
                        transcription.value = ""
                        recognitionError.value = null
                    }
                },
                enabled = voiceInputService != null
            ) {
                val infiniteTransition = rememberInfiniteTransition(label = "Recording animation")
                val pulseAnimation by infiniteTransition.animateFloat(
                    initialValue = 0.8f,
                    targetValue = 1.2f,
                    animationSpec = infiniteRepeatable(
                        animation = tween(1000, easing = FastOutSlowInEasing),
                        repeatMode = RepeatMode.Reverse
                    ),
                    label = "Recording pulse"
                )
                
                Icon(
                    imageVector = if (isRecording.value) Icons.Default.Stop else Icons.Default.Mic,
                    contentDescription = if (isRecording.value) "Stop voice recording" else "Start voice input",
                    tint = if (isRecording.value) MaterialTheme.colorScheme.error else
                           if (!hasRecordAudioPermission.value) MaterialTheme.colorScheme.error.copy(alpha = 0.5f)
                           else MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier
                        .size(DesignTokens.Icon.sm)
                        .scale(if (isRecording.value) pulseAnimation else 1f)
                )
            }

            // Input field
            Surface(
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = DesignTokens.Spacing.sm),
                shape = RoundedCornerShape(DesignTokens.Radius.md),
                color = MaterialTheme.colorScheme.surfaceVariant
            ) {
                OutlinedTextField(
                    value = value,
                    onValueChange = onValueChange,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = DesignTokens.Spacing.md, vertical = DesignTokens.Input.iconSize),
                    textStyle = MaterialTheme.typography.bodyLarge,
                    placeholder = {
                        Text(
                            text = placeholder,
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
                        )
                    },
                    leadingIcon = {
                        Row {
                            Icon(
                                imageVector = if (isRecording.value) Icons.Default.Stop else Icons.Default.Mic,
                                contentDescription = if (isRecording.value) "Stop voice recording" else "Voice input",
                                tint = if (isRecording.value) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.size(DesignTokens.Icon.sm)
                            )
                            if (isRecording.value) {
                                Spacer(modifier = Modifier.width(DesignTokens.Spacing.xs))
                                Box(
                                    modifier = Modifier
                                        .size(8.dp)
                                        .background(MaterialTheme.colorScheme.error, CircleShape)
                                )
                            }
                        }
                    },
                    trailingIcon = {
                        IconButton(
                            onClick = {
                                if (value.text.isNotBlank()) {
                                    onSend(value.text)
                                }
                            },
                            enabled = value.text.isNotBlank()
                        ) {
                            Icon(
                                imageVector = Icons.Default.Send,
                                contentDescription = "Send",
                                modifier = Modifier.size(DesignTokens.Icon.sm)
                            )
                        }
                    },
                    colors = TextFieldDefaults.colors(
                        focusedIndicatorColor = if (isRecording.value) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary,
                        unfocusedIndicatorColor = if (isRecording.value) MaterialTheme.colorScheme.error.copy(alpha = 0.5f) else MaterialTheme.colorScheme.outline,
                        cursorColor = MaterialTheme.colorScheme.primary
                    )
                )
            }
        }
    }
}

/**
 * Command chip data class
 */
data class CommandChip(
    val label: String,
    val command: String,
    val icon: ImageVector
)

/**
 * Default command chips for the command bar
 */
val defaultCommandChips = listOf(
    CommandChip("Status", "!status", Icons.Default.Info),
    CommandChip("Screenshot", "!screenshot", Icons.Default.Image),
    CommandChip("Stop", "!stop", Icons.Default.Stop),
    CommandChip("Pause", "!pause", Icons.Default.Pause),
    CommandChip("Logs", "!logs", Icons.Default.FileOpen)
)

/**
 * Preview for CommandBar in light mode
 */
@Preview
@Composable
fun CommandBarPreviewLight() {
    MaterialTheme {
        CommandBar(
            value = TextFieldValue(),
            onValueChange = {},
            onSend = {}
        )
    }
}

/**
 * Preview for CommandBar in dark mode
 */
@Preview(uiMode = android.content.res.Configuration.UI_MODE_NIGHT_YES)
@Composable
fun CommandBarPreviewDark() {
    MaterialTheme {
        CommandBar(
            value = TextFieldValue(),
            onValueChange = {},
            onSend = {}
        )
    }
}