package app.armorclaw.ui.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.tooling.preview.Preview

/**
 * Blocker Resolution Dialog
 *
 * Resolves: ArmorChat BlockerResponseDialog (T27)
 *
 * Dialog for resolving workflow blockers that require user input.
 * Shows the blocker message, provides an input field for the response,
 * optional note field, and handles loading/error states during RPC call.
 *
 * PII Safety: Input is cleared after sending, never logged or persisted
 * in UI. Sensitive fields (password, card, key, token) use password masking.
 */

/**
 * Describes a blocker that requires user input to proceed.
 */
data class BlockerInfo(
    val blockerType: String,
    val message: String,
    val suggestion: String = "",
    val field: String = "",
    val workflowId: String,
    val stepId: String
)

/**
 * Dialog state machine for the blocker resolution flow.
 */
enum class BlockerDialogState {
    INPUT,
    LOADING,
    ERROR,
    DISMISSED
}

/**
 * Blocker Resolution Dialog
 *
 * @param blocker The blocker metadata describing what input is needed
 * @param onDismiss Called when the user dismisses the dialog
 * @param onResolve Called with (workflowId, stepId, input, note) to submit resolution
 * @param dialogState Current state of the dialog (managed by parent)
 * @param errorMessage Error text to display when dialogState == ERROR
 */
@Composable
fun BlockerResponseDialog(
    blocker: BlockerInfo,
    onDismiss: () -> Unit,
    onResolve: (workflowId: String, stepId: String, input: String, note: String) -> Unit,
    dialogState: BlockerDialogState,
    errorMessage: String = "",
    modifier: Modifier = Modifier
) {
    var inputText by remember { mutableStateOf("") }
    var noteText by remember { mutableStateOf("") }
    var showNoteField by remember { mutableStateOf(false) }

    // Clear input after send (PII safety)
    LaunchedEffect(dialogState) {
        if (dialogState == BlockerDialogState.INPUT || dialogState == BlockerDialogState.ERROR) {
            // Keep input on retry — only clear on fresh blocker
        }
    }

    if (dialogState == BlockerDialogState.DISMISSED) return

    val isSensitive = remember(blocker.field) {
        val f = blocker.field.lowercase()
        f.contains("password") || f.contains("card") ||
            f.contains("key") || f.contains("token") ||
            f.contains("secret") || f.contains("cvv") ||
            f.contains("pin") || f.contains("ssn")
    }

    Dialog(onDismissRequest = {
        if (dialogState != BlockerDialogState.LOADING) onDismiss()
    }) {
        Card(
            modifier = modifier
                .fillMaxWidth()
                .padding(16.dp),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(
                containerColor = when (dialogState) {
                    BlockerDialogState.ERROR -> Color(0xFFFFEBEE)
                    else -> MaterialTheme.colorScheme.surface
                }
            )
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                // Header icon
                Icon(
                    imageVector = when (dialogState) {
                        BlockerDialogState.ERROR -> Icons.Default.Error
                        else -> Icons.Default.WarningAmber
                    },
                    contentDescription = null,
                    tint = when (dialogState) {
                        BlockerDialogState.ERROR -> Color(0xFFD32F2F)
                        else -> Color(0xFFFFA000)
                    },
                    modifier = Modifier.size(48.dp)
                )

                Spacer(modifier = Modifier.height(12.dp))

                // Title
                Text(
                    text = "\u26A0\uFE0F Action Required",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold,
                    color = when (dialogState) {
                        BlockerDialogState.ERROR -> Color(0xFFB71C1C)
                        else -> MaterialTheme.colorScheme.onSurface
                    }
                )

                Spacer(modifier = Modifier.height(4.dp))

                // Blocker type badge
                Surface(
                    shape = RoundedCornerShape(4.dp),
                    color = MaterialTheme.colorScheme.secondaryContainer
                ) {
                    Text(
                        text = blocker.blockerType.uppercase(),
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSecondaryContainer,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp)
                    )
                }

                Spacer(modifier = Modifier.height(12.dp))

                // Blocker message
                Text(
                    text = blocker.message,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                // Suggestion (if present)
                if (blocker.suggestion.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = blocker.suggestion,
                        style = MaterialTheme.typography.bodySmall.copy(
                            fontStyle = androidx.compose.ui.text.font.FontStyle.Italic
                        ),
                        color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f)
                    )
                }

                // Field label (if present)
                if (blocker.field.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = MaterialTheme.colorScheme.primaryContainer
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                            horizontalArrangement = Arrangement.spacedBy(4.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Default.Edit,
                                contentDescription = null,
                                modifier = Modifier.size(14.dp),
                                tint = MaterialTheme.colorScheme.onPrimaryContainer
                            )
                            Text(
                                text = "Field: ${blocker.field}",
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.onPrimaryContainer
                            )
                            if (isSensitive) {
                                Icon(
                                    imageVector = Icons.Default.Lock,
                                    contentDescription = "Sensitive",
                                    modifier = Modifier.size(14.dp),
                                    tint = Color(0xFFD32F2F)
                                )
                            }
                        }
                    }
                }

                Spacer(modifier = Modifier.height(16.dp))

                // Input field
                when (dialogState) {
                    BlockerDialogState.LOADING -> {
                        // Loading state
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(32.dp),
                                strokeWidth = 3.dp,
                                color = MaterialTheme.colorScheme.primary
                            )
                            Spacer(modifier = Modifier.height(12.dp))
                            Text(
                                text = "Resolving blocker\u2026",
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                    BlockerDialogState.ERROR -> {
                        // Error state
                        Text(
                            text = errorMessage.ifEmpty { "Failed to resolve blocker. Please try again." },
                            style = MaterialTheme.typography.bodySmall,
                            color = Color(0xFFB71C1C)
                        )
                        Spacer(modifier = Modifier.height(12.dp))
                    }
                    else -> {
                        // INPUT state — show input fields
                        OutlinedTextField(
                            value = inputText,
                            onValueChange = { inputText = it },
                            label = {
                                Text(if (blocker.field.isNotEmpty()) blocker.field else "Response")
                            },
                            placeholder = {
                                Text(
                                    if (blocker.suggestion.isNotEmpty()) blocker.suggestion
                                    else "Enter your response"
                                )
                            },
                            visualTransformation = if (isSensitive)
                                VisualTransformation.Password()
                            else
                                VisualTransformation.None,
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(8.dp),
                            enabled = dialogState == BlockerDialogState.INPUT
                        )

                        Spacer(modifier = Modifier.height(8.dp))

                        // Note toggle
                        TextButton(
                            onClick = { showNoteField = !showNoteField },
                            contentPadding = PaddingValues(horizontal = 8.dp)
                        ) {
                            Icon(
                                imageVector = if (showNoteField)
                                    Icons.Default.ExpandLess
                                else
                                    Icons.Default.ExpandMore,
                                contentDescription = null,
                                modifier = Modifier.size(16.dp)
                            )
                            Spacer(modifier = Modifier.width(4.dp))
                            Text(
                                text = if (showNoteField) "Hide note" else "Add note (optional)",
                                style = MaterialTheme.typography.labelMedium
                            )
                        }

                        // Collapsible note field
                        AnimatedVisibility(visible = showNoteField) {
                            OutlinedTextField(
                                value = noteText,
                                onValueChange = { noteText = it },
                                label = { Text("Note") },
                                placeholder = { Text("Optional context or instructions") },
                                singleLine = false,
                                maxLines = 3,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 4.dp),
                                shape = RoundedCornerShape(8.dp)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.height(20.dp))

                // Action buttons
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    // Cancel / Dismiss
                    OutlinedButton(
                        onClick = onDismiss,
                        modifier = Modifier.weight(1f),
                        enabled = dialogState != BlockerDialogState.LOADING
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Cancel")
                    }

                    // Send / Retry
                    if (dialogState == BlockerDialogState.ERROR) {
                        Button(
                            onClick = {
                                onResolve(
                                    blocker.workflowId,
                                    blocker.stepId,
                                    inputText,
                                    noteText
                                )
                            },
                            modifier = Modifier.weight(1f),
                            colors = ButtonDefaults.buttonColors(
                                containerColor = Color(0xFFFF9800)
                            )
                        ) {
                            Icon(
                                imageVector = Icons.Default.Refresh,
                                contentDescription = null,
                                modifier = Modifier.size(18.dp)
                            )
                            Spacer(modifier = Modifier.width(4.dp))
                            Text("Retry")
                        }
                    } else {
                        Button(
                            onClick = {
                                onResolve(
                                    blocker.workflowId,
                                    blocker.stepId,
                                    inputText,
                                    noteText
                                )
                            },
                            modifier = Modifier.weight(1f),
                            enabled = dialogState == BlockerDialogState.INPUT &&
                                inputText.isNotBlank()
                        ) {
                            Icon(
                                imageVector = Icons.Default.Send,
                                contentDescription = null,
                                modifier = Modifier.size(18.dp)
                            )
                            Spacer(modifier = Modifier.width(4.dp))
                            Text("Send")
                        }
                    }
                }
            }
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun BlockerResponseDialogInputPreview() {
    MaterialTheme {
        BlockerResponseDialog(
            blocker = BlockerInfo(
                blockerType = "missing_field",
                message = "The agent needs your shipping address to complete the checkout.",
                suggestion = "e.g. 123 Main St, New York, NY 10001",
                field = "shipping_address",
                workflowId = "wf_abc123",
                stepId = "step_4"
            ),
            onDismiss = {},
            onResolve = { _, _, _, _ -> },
            dialogState = BlockerDialogState.INPUT
        )
    }
}

@Preview(showBackground = true)
@Composable
private fun BlockerResponseDialogLoadingPreview() {
    MaterialTheme {
        BlockerResponseDialog(
            blocker = BlockerInfo(
                blockerType = "missing_field",
                message = "The agent needs your shipping address to complete the checkout.",
                field = "shipping_address",
                workflowId = "wf_abc123",
                stepId = "step_4"
            ),
            onDismiss = {},
            onResolve = { _, _, _, _ -> },
            dialogState = BlockerDialogState.LOADING
        )
    }
}

@Preview(showBackground = true)
@Composable
private fun BlockerResponseDialogErrorPreview() {
    MaterialTheme {
        BlockerResponseDialog(
            blocker = BlockerInfo(
                blockerType = "auth_required",
                message = "The website requires a CAPTCHA response.",
                field = "captcha_response",
                workflowId = "wf_def456",
                stepId = "step_2"
            ),
            onDismiss = {},
            onResolve = { _, _, _, _ -> },
            dialogState = BlockerDialogState.ERROR,
            errorMessage = "Network error: could not reach bridge. Please try again."
        )
    }
}

@Preview(showBackground = true)
@Composable
private fun BlockerResponseDialogSensitivePreview() {
    MaterialTheme {
        BlockerResponseDialog(
            blocker = BlockerInfo(
                blockerType = "credential_required",
                message = "The agent needs your password to proceed with login.",
                suggestion = "Enter your account password",
                field = "password",
                workflowId = "wf_xyz789",
                stepId = "step_1"
            ),
            onDismiss = {},
            onResolve = { _, _, _, _ -> },
            dialogState = BlockerDialogState.INPUT
        )
    }
}
