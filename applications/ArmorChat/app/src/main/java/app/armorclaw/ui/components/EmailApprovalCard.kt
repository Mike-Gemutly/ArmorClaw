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

@Composable
fun EmailApprovalCard(
    approvalId: String,
    emailId: String,
    to: String,
    piiFieldCount: Int,
    timeoutSeconds: Int = 300,
    onApprove: (String) -> Unit,
    onDeny: (String, String) -> Unit,
    modifier: Modifier = Modifier
) {
    var remainingSeconds by remember { mutableIntStateOf(timeoutSeconds) }
    var isExpired by remember { mutableStateOf(false) }

    LaunchedEffect(approvalId) {
        while (remainingSeconds > 0) {
            delay(1000)
            remainingSeconds--
        }
        isExpired = true
    }

    val isUrgent = remainingSeconds < 60
    val borderColor = when {
        isExpired -> Color(0xFF9E9E9E)
        isUrgent -> Color(0xFFF44336)
        else -> Color(0xFFFFC107)
    }
    val bgColor = when {
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
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Email,
                    contentDescription = null,
                    tint = when {
                        isExpired -> Color(0xFF9E9E9E)
                        isUrgent -> Color(0xFFD32F2F)
                        else -> Color(0xFFF57C00)
                    },
                    modifier = Modifier.size(28.dp)
                )
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = if (isExpired) "Email Approval Expired" else "Email Approval Request",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Text(
                        text = "To: $to",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                if (!isExpired) {
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = if (isUrgent) Color(0xFFFFCDD2) else Color(0xFFFFF9C4)
                    ) {
                        Text(
                            text = formatDuration(remainingSeconds),
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Bold,
                            color = if (isUrgent) Color(0xFFB71C1C) else Color(0xFFF57F17),
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            if (isExpired) {
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = Color(0xFFEEEEEE)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(12.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = Icons.Default.Schedule,
                            contentDescription = null,
                            tint = Color(0xFF757575),
                            modifier = Modifier.size(20.dp)
                        )
                        Text(
                            text = "This approval request has expired.",
                            style = MaterialTheme.typography.bodyMedium,
                            color = Color(0xFF757575)
                        )
                    }
                }
            } else {
                Text(
                    text = "An agent wants to send an email with $piiFieldCount PII field(s).",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium
                )

                Spacer(modifier = Modifier.height(4.dp))

                Text(
                    text = "Subject: [masked]",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                Spacer(modifier = Modifier.height(16.dp))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    OutlinedButton(
                        onClick = { onDeny(approvalId, "User denied email send") },
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

                    Button(
                        onClick = { onApprove(approvalId) },
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

private fun formatDuration(seconds: Int): String {
    val m = seconds / 60
    val s = seconds % 60
    return String.format("%d:%02d", m, s)
}
