package app.armorclaw.ui.components

import androidx.compose.foundation.BorderStroke
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
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay

/**
 * BlindFill Secret Approval Card - HITL Approval for BlindFill credential injection
 *
 * Renders a card for approving credential injection into browser forms.
 * The agent requests a credential by name but never sees the actual value.
 * Shows target domain, credential name, risk badge, and countdown timer.
 *
 * Deep link: armorclaw://secret/approve/{request_id}
 *
 * Wires to: blindfill.approve / blindfill.reject RPC methods
 */

/**
 * Risk classification for BlindFill requests
 */
enum class BlindFillRiskClass(
    val displayName: String,
    val color: Color,
    val bgColor: Color
) {
    PAYMENT("Payment", Color(0xFFB71C1C), Color(0xFFFFEBEE)),
    IDENTITY_PII("Identity PII", Color(0xFFE65100), Color(0xFFFFF3E0)),
    CREDENTIAL_USE("Credential", Color(0xFFF57F17), Color(0xFFFFF9C4)),
    OTHER("Other", Color(0xFF1565C0), Color(0xFFE3F2FD));

    companion object {
        fun fromString(s: String): BlindFillRiskClass {
            return values().find {
                it.name.equals(s.replace("-", "_"), ignoreCase = true)
            } ?: OTHER
        }
    }
}

/**
 * Represents a BlindFill secret injection request
 */
data class BlindFillRequest(
    val requestId: String,
    val agentId: String,
    val credentialName: String,
    val targetDomain: String,
    val riskClass: BlindFillRiskClass,
    val reason: String,
    val timeoutSeconds: Int = 300
)

/**
 * BlindFill Approval Card composable
 *
 * @param request The BlindFill request details
 * @param onApprove Called with requestId when user approves injection
 * @param onDeny Called with requestId when user denies injection
 */
@Composable
fun BlindFillCard(
    request: BlindFillRequest,
    onApprove: (String) -> Unit,
    onDeny: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    var remainingSeconds by remember { mutableIntStateOf(request.timeoutSeconds) }
    var isExpired by remember { mutableStateOf(false) }
    var isApproved by remember { mutableStateOf(false) }
    var isDenied by remember { mutableStateOf(false) }

    // Countdown timer
    LaunchedEffect(request.requestId) {
        while (remainingSeconds > 0) {
            delay(1000)
            remainingSeconds--
        }
        isExpired = true
    }

    val isResolved = isApproved || isDenied || isExpired
    val isUrgent = remainingSeconds < 60

    val borderColor = when {
        isDenied -> Color(0xFFD32F2F)
        isApproved -> Color(0xFF388E3C)
        isExpired -> Color(0xFF9E9E9E)
        isUrgent -> Color(0xFFF44336)
        else -> Color(0xFFFFC107)
    }
    val bgColor = when {
        isDenied -> Color(0xFFFFEBEE)
        isApproved -> Color(0xFFE8F5E9)
        isExpired -> Color(0xFFF5F5F5)
        isUrgent -> Color(0xFFFFEBEE)
        else -> Color(0xFFFFF8E1)
    }

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = bgColor),
        border = BorderStroke(1.5.dp, borderColor)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        ) {
            // Header row
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Shield,
                    contentDescription = null,
                    tint = when {
                        isExpired -> Color(0xFF9E9E9E)
                        isDenied -> Color(0xFFD32F2F)
                        isApproved -> Color(0xFF388E3C)
                        isUrgent -> Color(0xFFD32F2F)
                        else -> Color(0xFFF57C00)
                    },
                    modifier = Modifier.size(28.dp)
                )
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = when {
                            isDenied -> "Secret Request Denied"
                            isApproved -> "Secret Approved"
                            isExpired -> "Secret Request Expired"
                            else -> "🔐 Secret Request"
                        },
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Text(
                        text = "Agent: ${request.agentId}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }

                // Risk badge
                if (!isResolved) {
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = request.riskClass.bgColor
                    ) {
                        Text(
                            text = request.riskClass.displayName.uppercase(),
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Bold,
                            color = request.riskClass.color,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }

                // Countdown timer
                if (!isResolved) {
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = if (isUrgent) Color(0xFFFFCDD2) else Color(0xFFFFF9C4)
                    ) {
                        Text(
                            text = formatBlindFillDuration(remainingSeconds),
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Bold,
                            color = if (isUrgent) Color(0xFFB71C1C) else Color(0xFFF57F17),
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            if (isResolved) {
                // Resolved state
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = when {
                        isApproved -> Color(0xFFC8E6C9)
                        isDenied -> Color(0xFFFFCDD2)
                        else -> Color(0xFFEEEEEE)
                    }
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(12.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = when {
                                isApproved -> Icons.Default.CheckCircle
                                isDenied -> Icons.Default.Cancel
                                else -> Icons.Default.Schedule
                            },
                            contentDescription = null,
                            tint = when {
                                isApproved -> Color(0xFF2E7D32)
                                isDenied -> Color(0xFFC62828)
                                else -> Color(0xFF757575)
                            },
                            modifier = Modifier.size(20.dp)
                        )
                        Text(
                            text = when {
                                isApproved -> "Credential injection approved."
                                isDenied -> "Credential injection denied."
                                else -> "This approval request has expired."
                            },
                            style = MaterialTheme.typography.bodyMedium,
                            color = when {
                                isApproved -> Color(0xFF2E7D32)
                                isDenied -> Color(0xFFC62828)
                                else -> Color(0xFF757575)
                            }
                        )
                    }
                }
            } else {
                // Target domain (prominent)
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.Language,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(18.dp)
                    )
                    Text(
                        text = request.targetDomain,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                }

                Spacer(modifier = Modifier.height(8.dp))

                // Credential name
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.VpnKey,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(18.dp)
                    )
                    Text(
                        text = "Credential: ${request.credentialName}",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium
                    )
                }

                // Reason text
                if (request.reason.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = request.reason,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }

                // Deep link reference comment:
                // armorclaw://secret/approve/{request_id}

                Spacer(modifier = Modifier.height(16.dp))

                // Action buttons
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    // Deny button
                    OutlinedButton(
                        onClick = {
                            isDenied = true
                            onDeny(request.requestId)
                        },
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.outlinedButtonColors(
                            contentColor = Color(0xFFD32F2F)
                        )
                    ) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Deny")
                    }

                    // Approve button
                    Button(
                        onClick = {
                            isApproved = true
                            onApprove(request.requestId)
                        },
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.buttonColors(
                            containerColor = Color(0xFF388E3C)
                        )
                    ) {
                        Icon(
                            imageVector = Icons.Default.Check,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Approve")
                    }
                }
            }
        }
    }
}

private fun formatBlindFillDuration(seconds: Int): String {
    val m = seconds / 60
    val s = seconds % 60
    return String.format("%d:%02d", m, s)
}

@Preview(showBackground = true)
@Composable
private fun BlindFillCardPreview() {
    MaterialTheme {
        BlindFillCard(
            request = BlindFillRequest(
                requestId = "req_abc123",
                agentId = "travel-booker",
                credentialName = "credit_card",
                targetDomain = "booking.example.com",
                riskClass = BlindFillRiskClass.PAYMENT,
                reason = "Agent needs to complete flight booking checkout.",
                timeoutSeconds = 300
            ),
            onApprove = {},
            onDeny = { _ -> }
        )
    }
}

@Preview(showBackground = true)
@Composable
private fun BlindFillCardLowRiskPreview() {
    MaterialTheme {
        BlindFillCard(
            request = BlindFillRequest(
                requestId = "req_def456",
                agentId = "research-agent",
                credentialName = "api_token",
                targetDomain = "api.service.io",
                riskClass = BlindFillRiskClass.CREDENTIAL_USE,
                reason = "Agent needs API access for data retrieval.",
                timeoutSeconds = 180
            ),
            onApprove = {},
            onDeny = { _ -> }
        )
    }
}
