package components.governor

import androidx.compose.animation.core.*
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.detectTapGestures
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
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * HITL Authorization Component (The Pinch)
 *
 * Gesture-based approval component for sensitive actions.
 * Requires user to perform a long-press gesture to approve.
 *
 * Phase 2 Implementation - Governor Strategy
 *
 * @param command The command requiring approval
 * @param onApprove Callback when approval is complete
 * @param onReject Callback when rejected
 * @param modifier Optional modifier
 */
@Composable
fun HITLAuthorizationCard(
    command: CommandBlock,
    onApprove: () -> Unit,
    onReject: () -> Unit,
    modifier: Modifier = Modifier
) {
    var approvalProgress by remember { mutableFloatStateOf(0f) }
    var isHolding by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    // Animated progress
    val animatedProgress by animateFloatAsState(
        targetValue = approvalProgress,
        animationSpec = tween(100),
        label = "approval_progress"
    )

    // Pulse animation when holding
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val pulseScale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (isHolding) 1.05f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse_scale"
    )

    // Progress timer while holding
    LaunchedEffect(isHolding) {
        if (isHolding) {
            val startTime = System.currentTimeMillis()
            val holdDuration = 2000L // 2 seconds to approve
            
            while (isHolding && approvalProgress < 1f) {
                val elapsed = System.currentTimeMillis() - startTime
                approvalProgress = (elapsed.toFloat() / holdDuration).coerceIn(0f, 1f)
                delay(16) // ~60fps
            }
            
            if (approvalProgress >= 1f) {
                onApprove()
            }
        } else {
            // Decay progress when released
            while (approvalProgress > 0f && !isHolding) {
                approvalProgress = (approvalProgress - 0.05f).coerceAtLeast(0f)
                delay(16)
            }
        }
    }

    Card(
        modifier = modifier
            .fillMaxWidth()
            .graphicsLayer {
                scaleX = pulseScale
                scaleY = pulseScale
            },
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFF1A1A2E) // Dark authorization background
        ),
        border = BorderStroke(
            width = if (animatedProgress > 0) 2.dp else 1.dp,
            color = if (animatedProgress > 0)
                Color(0xFF00BCD4).copy(alpha = animatedProgress)
            else
                Color(0xFF4A4A6A)
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Header
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Lock,
                        contentDescription = null,
                        tint = Color(0xFF00BCD4),
                        modifier = Modifier.size(24.dp)
                    )
                    Text(
                        text = "Authorization Required",
                        style = MaterialTheme.typography.titleMedium,
                        color = Color.White,
                        fontWeight = FontWeight.Bold
                    )
                }

                // Risk badge
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = Color(0xFFF44336).copy(alpha = 0.2f)
                ) {
                    Text(
                        text = "HIGH RISK",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color(0xFFF44336),
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                    )
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Command details
            Text(
                text = command.title,
                style = MaterialTheme.typography.titleLarge,
                color = Color.White,
                textAlign = TextAlign.Center
            )

            if (command.description.isNotBlank()) {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = command.description,
                    style = MaterialTheme.typography.bodyMedium,
                    color = Color.White.copy(alpha = 0.7f),
                    textAlign = TextAlign.Center
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Required capabilities
            if (command.requiredCapabilities.isNotEmpty()) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    command.requiredCapabilities.forEach { cap ->
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = Color.White.copy(alpha = 0.1f)
                        ) {
                            Text(
                                text = cap.displayName,
                                style = MaterialTheme.typography.labelSmall,
                                color = Color.White,
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                            )
                        }
                        Spacer(modifier = Modifier.width(8.dp))
                    }
                }
                Spacer(modifier = Modifier.height(24.dp))
            }

            // Approval button with progress ring
            Box(
                modifier = Modifier.size(120.dp),
                contentAlignment = Alignment.Center
            ) {
                // Progress ring background
                Box(
                    modifier = Modifier
                        .size(120.dp)
                        .border(4.dp, Color.White.copy(alpha = 0.1f), CircleShape)
                )

                // Progress ring
                Box(
                    modifier = Modifier
                        .size(120.dp)
                        .clip(CircleShape)
                        .background(
                            brush = androidx.compose.ui.graphics.Brush.sweepGradient(
                                colors = listOf(
                                    Color(0xFF00BCD4).copy(alpha = animatedProgress),
                                    Color(0xFF00BCD4).copy(alpha = 0f),
                                    Color(0xFF00BCD4).copy(alpha = 0f)
                                )
                            )
                        )
                )

                // Inner button
                Box(
                    modifier = Modifier
                        .size(100.dp)
                        .clip(CircleShape)
                        .background(
                            if (isHolding)
                                Color(0xFF00BCD4).copy(alpha = 0.3f)
                            else
                                Color.White.copy(alpha = 0.1f)
                        )
                        .pointerInput(Unit) {
                            detectTapGestures(
                                onPress = {
                                    isHolding = true
                                    tryAwaitRelease()
                                    isHolding = false
                                }
                            )
                        },
                    contentAlignment = Alignment.Center
                ) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Icon(
                            imageVector = if (animatedProgress >= 1f)
                                Icons.Default.Check
                            else
                                Icons.Default.Done,
                            contentDescription = null,
                            tint = if (isHolding) Color(0xFF00BCD4) else Color.White,
                            modifier = Modifier.size(40.dp)
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = if (animatedProgress >= 1f) "Approved" else "Hold",
                            style = MaterialTheme.typography.labelSmall,
                            color = if (isHolding) Color(0xFF00BCD4) else Color.White
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Instructions
            Text(
                text = "Press and hold to authorize this action",
                style = MaterialTheme.typography.bodySmall,
                color = Color.White.copy(alpha = 0.5f),
                textAlign = TextAlign.Center
            )

            // Cancel button
            Spacer(modifier = Modifier.height(16.dp))
            TextButton(onClick = onReject) {
                Text(
                    text = "Cancel",
                    color = Color.White.copy(alpha = 0.5f)
                )
            }
        }
    }
}

/**
 * Simple Approval Dialog
 *
 * Lightweight approval dialog for non-critical actions
 */
@Composable
fun SimpleApprovalDialog(
    title: String,
    message: String,
    riskLevel: RiskLevel,
    onApprove: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        icon = {
            Icon(
                imageVector = when (riskLevel) {
                    RiskLevel.LOW -> Icons.Default.Info
                    RiskLevel.MEDIUM -> Icons.Default.Warning
                    RiskLevel.HIGH -> Icons.Default.Warning
                    RiskLevel.CRITICAL -> Icons.Default.Warning
                },
                contentDescription = null,
                tint = getRiskColor(riskLevel)
            )
        },
        title = { Text(title) },
        text = { Text(message) },
        confirmButton = {
            Button(
                onClick = onApprove,
                colors = ButtonDefaults.buttonColors(
                    containerColor = getRiskColor(riskLevel)
                )
            ) {
                Text("Approve")
            }
        },
        dismissButton = {
            OutlinedButton(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

// Helper function
private fun getRiskColor(riskLevel: RiskLevel): Color {
    return when (riskLevel) {
        RiskLevel.LOW -> Color(0xFF4CAF50)
        RiskLevel.MEDIUM -> Color(0xFFFFC107)
        RiskLevel.HIGH -> Color(0xFFFF9800)
        RiskLevel.CRITICAL -> Color(0xFFF44336)
    }
}

/**
 * Preview
 */
@Composable
fun HITLAuthorizationPreview() {
    val sampleCommand = CommandBlock(
        id = "cmd_1",
        agentId = "agent_1",
        agentName = "Payment Agent",
        commandType = CommandType.APPROVAL_REQUIRED,
        title = "Process Payment",
        description = "Authorize payment of \$149.99 to vendor@example.com",
        status = CommandStatus.PENDING,
        requiredCapabilities = listOf(
            Capability(
                id = "cap_1",
                name = "payment",
                displayName = "Payment",
                description = "Process payments",
                category = CapabilityCategory.EXTERNAL,
                riskLevel = RiskLevel.CRITICAL,
                requiresApproval = true
            )
        ),
        requiredPiiKeys = listOf("credit_card"),
        createdAt = System.currentTimeMillis(),
        isApproved = false
    )

    MaterialTheme {
        Surface(
            modifier = Modifier.fillMaxWidth().padding(16.dp),
            color = MaterialTheme.colorScheme.background
        ) {
            HITLAuthorizationCard(
                command = sampleCommand,
                onApprove = {},
                onReject = {}
            )
        }
    }
}
