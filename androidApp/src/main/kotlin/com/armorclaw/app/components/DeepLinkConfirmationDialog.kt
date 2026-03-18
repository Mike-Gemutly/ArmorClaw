package com.armorclaw.app.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Devices
import androidx.compose.material.icons.filled.Forum
import androidx.compose.material.icons.filled.Language
import androidx.compose.material.icons.filled.PersonAdd
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.armorclaw.app.navigation.DeepLinkAction
import com.armorclaw.app.navigation.DeepLinkSecurityCheck

/**
 * Confirmation dialog for deep link navigation
 *
 * Shows a warning when navigating to potentially untrusted destinations,
 * such as rooms from external links.
 */
@Composable
fun DeepLinkConfirmationDialog(
    action: DeepLinkAction,
    securityCheck: DeepLinkSecurityCheck,
    message: String,
    details: String?,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier
) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(
            dismissOnBackPress = true,
            dismissOnClickOutside = true
        )
    ) {
        Card(
            modifier = modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surface
            )
        ) {
            Column(
                modifier = Modifier.padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Icon based on security check type
                Icon(
                    imageVector = when (securityCheck) {
                        DeepLinkSecurityCheck.ROOM_MEMBERSHIP -> Icons.Default.Forum
                        DeepLinkSecurityCheck.CALL_JOIN -> Icons.Default.Call
                        DeepLinkSecurityCheck.EXTERNAL_LINK -> Icons.Default.Language
                        DeepLinkSecurityCheck.INVITE_ACCEPT -> Icons.Default.PersonAdd
                        DeepLinkSecurityCheck.DEVICE_BONDING -> Icons.Default.Devices
                    },
                    contentDescription = null,
                    modifier = Modifier.size(48.dp),
                    tint = when (securityCheck) {
                        DeepLinkSecurityCheck.CALL_JOIN -> MaterialTheme.colorScheme.primary
                        else -> MaterialTheme.colorScheme.secondary
                    }
                )

                // Title
                Text(
                    text = message,
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold,
                    textAlign = TextAlign.Center
                )

                // Details/warning message
                if (details != null) {
                    Text(
                        text = details,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        textAlign = TextAlign.Center
                    )
                }

                // Room ID info for room navigation
                when (action) {
                    is DeepLinkAction.NavigateToRoom -> {
                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = MaterialTheme.colorScheme.surfaceVariant
                        ) {
                            Text(
                                text = action.roomId,
                                style = MaterialTheme.typography.bodySmall,
                                modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                                maxLines = 1
                            )
                        }
                    }
                    else -> { /* No extra info */ }
                }

                // Action buttons
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    OutlinedButton(
                        onClick = onDismiss,
                        modifier = Modifier.weight(1f)
                    ) {
                        Text("Cancel")
                    }

                    Button(
                        onClick = onConfirm,
                        modifier = Modifier.weight(1f),
                        colors = when (securityCheck) {
                            DeepLinkSecurityCheck.CALL_JOIN -> ButtonDefaults.buttonColors()
                            else -> ButtonDefaults.buttonColors(
                                containerColor = MaterialTheme.colorScheme.primary
                            )
                        }
                    ) {
                        Text(
                            when (securityCheck) {
                                DeepLinkSecurityCheck.ROOM_MEMBERSHIP -> "Join"
                                DeepLinkSecurityCheck.CALL_JOIN -> "Join Call"
                                DeepLinkSecurityCheck.EXTERNAL_LINK -> "Continue"
                                DeepLinkSecurityCheck.INVITE_ACCEPT -> "Accept"
                                DeepLinkSecurityCheck.DEVICE_BONDING -> "Pair Device"
                            }
                        )
                    }
                }

                // Security reminder
                Text(
                    text = "Only join rooms and calls from sources you trust.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f),
                    textAlign = TextAlign.Center
                )
            }
        }
    }
}

/**
 * Preview of the deep link confirmation dialog
 */
@Composable
fun DeepLinkConfirmationDialogPreview() {
    MaterialTheme {
        DeepLinkConfirmationDialog(
            action = DeepLinkAction.NavigateToRoom("!abc123:matrix.org"),
            securityCheck = DeepLinkSecurityCheck.ROOM_MEMBERSHIP,
            message = "Join room?",
            details = "You're about to join a chat room. Only join rooms from trusted sources.",
            onConfirm = {},
            onDismiss = {}
        )
    }
}
