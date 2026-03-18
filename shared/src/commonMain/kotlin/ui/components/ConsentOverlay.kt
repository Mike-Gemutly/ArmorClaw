package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.blur
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.armorclaw.shared.domain.model.PiiAccessRequest
import com.armorclaw.shared.domain.model.PiiField
import com.armorclaw.shared.domain.model.SensitivityLevel

/**
 * Full-Screen Consent Overlay
 *
 * A fullscreen modal for PII access requests with biometric gating.
 * Transforms the BlindFillCard into an immersive approval experience.
 *
 * ## Architecture
 * ```
 * ConsentOverlay (FullScreen Dialog)
 *      ├── Scrim with blur effect
 *      ├── ConsentContent
 *      │   ├── Header with agent info
 *      │   ├── SensitivityGroup[] (fields grouped by level)
 *      │   │   └── PiiFieldItem[]
 *      │   └── Action buttons
 *      └── BiometricGateOverlay (when critical fields approved)
 * ```
 *
 * ## Security Model
 * - LOW/MEDIUM fields: Standard approval
 * - HIGH fields: Warning displayed
 * - CRITICAL fields: Require biometric authentication
 *
 * ## Usage
 * ```kotlin
 * var showOverlay by remember { mutableStateOf(false) }
 *
 * if (showOverlay && pendingRequest != null) {
 *     ConsentOverlay(
 *         request = pendingRequest,
 *         onApprove = { fields -> viewModel.approve(fields) },
 *         onDeny = { viewModel.deny() },
 *         onDismiss = { showOverlay = false },
 *         onBiometricRequired = { showBiometric = true }
 *     )
 * }
 * ```
 */
@Composable
fun ConsentOverlay(
    request: PiiAccessRequest,
    onApprove: (Set<String>) -> Unit,
    onDeny: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
    onBiometricRequired: () -> Unit = {},
    isBiometricAuthenticating: Boolean = false,
    biometricError: String? = null
) {
    // Track approved fields
    var approvedFields by remember(request.requestId) {
        mutableStateOf(request.fields.map { it.name }.toSet())
    }

    // Check for critical fields
    val criticalFields = remember(request.fields) {
        request.fields.filter { it.sensitivity == SensitivityLevel.CRITICAL }
    }
    val hasApprovedCriticalFields = criticalFields.any { it.name in approvedFields }

    // Check if expired
    val isExpired = remember(request.expiresAt) {
        derivedStateOf { request.isExpired() }
    }

    // Group fields by sensitivity
    val fieldsBySensitivity = remember(request.fields) {
        request.fields.groupBy { it.sensitivity }
            .toSortedMap(compareByDescending { it.ordinal })
    }

    // Calculate summary
    val summary = remember(approvedFields, request.fields) {
        val total = request.fields.size
        val approved = approvedFields.size
        val critical = criticalFields.count { it.name in approvedFields }
        ConsentSummary(total, approved, critical)
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(
            dismissOnBackPress = true,
            dismissOnClickOutside = false,
            usePlatformDefaultWidth = false
        )
    ) {
        Box(
            modifier = modifier
                .fillMaxSize()
                .background(
                    Brush.verticalGradient(
                        colors = listOf(
                            MaterialTheme.colorScheme.scrim.copy(alpha = 0.9f),
                            MaterialTheme.colorScheme.scrim.copy(alpha = 0.95f)
                        )
                    )
                )
        ) {
            // Main content
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .statusBarsPadding()
                    .navigationBarsPadding()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Spacer(modifier = Modifier.height(32.dp))

                // Header icon with pulse animation
                ConsentHeader(
                    hasCriticalFields = hasApprovedCriticalFields,
                    isExpired = isExpired.value
                )

                Spacer(modifier = Modifier.height(24.dp))

                // Agent info
                Text(
                    text = "Data Access Request",
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = Color.White
                )

                Spacer(modifier = Modifier.height(8.dp))

                Text(
                    text = "Agent \"${request.agentId.take(20)}\" requests access to:",
                    style = MaterialTheme.typography.bodyLarge,
                    color = Color.White.copy(alpha = 0.8f),
                    textAlign = TextAlign.Center
                )

                Spacer(modifier = Modifier.height(4.dp))

                Text(
                    text = request.reason,
                    style = MaterialTheme.typography.bodyMedium,
                    color = Color.White.copy(alpha = 0.6f),
                    textAlign = TextAlign.Center
                )

                // Expired warning
                if (isExpired.value) {
                    Spacer(modifier = Modifier.height(16.dp))
                    ExpiredWarning()
                }

                // Biometric error
                if (biometricError != null) {
                    Spacer(modifier = Modifier.height(16.dp))
                    BiometricErrorBanner(message = biometricError)
                }

                Spacer(modifier = Modifier.height(24.dp))

                // Field groups by sensitivity
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    color = MaterialTheme.colorScheme.surface.copy(alpha = 0.95f)
                ) {
                    Column(
                        modifier = Modifier.padding(16.dp)
                    ) {
                        fieldsBySensitivity.forEach { (sensitivity, fields) ->
                            SensitivityGroup(
                                sensitivity = sensitivity,
                                fields = fields,
                                approvedFields = approvedFields,
                                onToggleField = { fieldName ->
                                    approvedFields = if (fieldName in approvedFields) {
                                        approvedFields - fieldName
                                    } else {
                                        approvedFields + fieldName
                                    }
                                },
                                enabled = !isExpired.value && !isBiometricAuthenticating
                            )

                            if (sensitivity != fieldsBySensitivity.keys.last()) {
                                Spacer(modifier = Modifier.height(16.dp))
                            }
                        }
                    }
                }

                // Summary section
                Spacer(modifier = Modifier.height(16.dp))
                ConsentSummarySection(summary = summary)

                // Batch indicator
                if (request.batchSize > 1) {
                    Spacer(modifier = Modifier.height(8.dp))
                    BatchIndicator(count = request.batchSize)
                }

                Spacer(modifier = Modifier.height(24.dp))

                // Action buttons
                ConsentActionButtons(
                    onApprove = {
                        if (hasApprovedCriticalFields) {
                            onBiometricRequired()
                        } else {
                            onApprove(approvedFields)
                        }
                    },
                    onDeny = onDeny,
                    onDismiss = onDismiss,
                    hasApprovedCriticalFields = hasApprovedCriticalFields,
                    hasApprovedFields = approvedFields.isNotEmpty(),
                    isExpired = isExpired.value,
                    isAuthenticating = isBiometricAuthenticating,
                    modifier = Modifier.fillMaxWidth()
                )

                Spacer(modifier = Modifier.height(16.dp))
            }

            // Biometric overlay
            if (isBiometricAuthenticating) {
                BiometricGateOverlay(
                    criticalFieldCount = criticalFields.count { it.name in approvedFields },
                    onCancel = onDismiss
                )
            }
        }
    }
}

/**
 * Consent header with animated icon
 */
@Composable
private fun ConsentHeader(
    hasCriticalFields: Boolean,
    isExpired: Boolean,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "consent_pulse")
    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (hasCriticalFields && !isExpired) 1.1f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    val iconColor = when {
        isExpired -> MaterialTheme.colorScheme.error
        hasCriticalFields -> MaterialTheme.colorScheme.error
        else -> MaterialTheme.colorScheme.primary
    }

    val icon = when {
        isExpired -> Icons.Default.Warning
        hasCriticalFields -> Icons.Default.Fingerprint
        else -> Icons.Default.Lock
    }

    Surface(
        modifier = modifier.graphicsLayer { scaleX = scale; scaleY = scale },
        shape = RoundedCornerShape(24.dp),
        color = iconColor.copy(alpha = 0.2f)
    ) {
        Box(
            modifier = Modifier.padding(16.dp),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = iconColor,
                modifier = Modifier.size(48.dp)
            )
        }
    }
}

/**
 * Expired request warning
 */
@Composable
private fun ExpiredWarning(modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.errorContainer
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Schedule,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(20.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "This request has expired and can no longer be approved",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onErrorContainer
            )
        }
    }
}

/**
 * Biometric error banner
 */
@Composable
private fun BiometricErrorBanner(
    message: String,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.errorContainer
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Error,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(20.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = message,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onErrorContainer
            )
        }
    }
}

/**
 * Fields grouped by sensitivity level
 */
@Composable
private fun SensitivityGroup(
    sensitivity: SensitivityLevel,
    fields: List<PiiField>,
    approvedFields: Set<String>,
    onToggleField: (String) -> Unit,
    enabled: Boolean,
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier) {
        // Group header
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            SensitivityBadge(sensitivity = sensitivity, compact = false)
            Text(
                text = getGroupDescription(sensitivity, fields.size),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }

        Spacer(modifier = Modifier.height(8.dp))

        // Field items
        fields.forEach { field ->
            PiiFieldItem(
                field = field,
                isApproved = approvedFields.contains(field.name),
                onToggle = { onToggleField(field.name) },
                enabled = enabled
            )
            Spacer(modifier = Modifier.height(4.dp))
        }
    }
}

/**
 * Get description for sensitivity group
 */
private fun getGroupDescription(sensitivity: SensitivityLevel, count: Int): String {
    val fieldWord = if (count == 1) "field" else "fields"
    return when (sensitivity) {
        SensitivityLevel.LOW -> "$count standard $fieldWord"
        SensitivityLevel.MEDIUM -> "$count sensitive $fieldWord"
        SensitivityLevel.HIGH -> "$count highly sensitive $fieldWord"
        SensitivityLevel.CRITICAL -> "$count critical $fieldWord (requires biometric)"
    }
}

/**
 * Consent summary section
 */
@Composable
private fun ConsentSummarySection(
    summary: ConsentSummary,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            SummaryItem(
                label = "Total",
                value = summary.total.toString(),
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            SummaryItem(
                label = "Selected",
                value = summary.approved.toString(),
                color = MaterialTheme.colorScheme.primary
            )
            if (summary.criticalApproved > 0) {
                SummaryItem(
                    label = "Critical",
                    value = summary.criticalApproved.toString(),
                    color = MaterialTheme.colorScheme.error
                )
            }
        }
    }
}

/**
 * Individual summary item
 */
@Composable
private fun SummaryItem(
    label: String,
    value: String,
    color: Color,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = value,
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            color = color
        )
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}

/**
 * Batch indicator
 */
@Composable
private fun BatchIndicator(
    count: Int,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.tertiaryContainer
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Layers,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onTertiaryContainer,
                modifier = Modifier.size(16.dp)
            )
            Spacer(modifier = Modifier.width(6.dp))
            Text(
                text = "$count similar requests pending",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onTertiaryContainer
            )
        }
    }
}

/**
 * Action buttons
 */
@Composable
private fun ConsentActionButtons(
    onApprove: () -> Unit,
    onDeny: () -> Unit,
    onDismiss: () -> Unit,
    hasApprovedCriticalFields: Boolean,
    hasApprovedFields: Boolean,
    isExpired: Boolean,
    isAuthenticating: Boolean,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier,
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        // Primary action
        Button(
            onClick = onApprove,
            enabled = hasApprovedFields && !isExpired && !isAuthenticating,
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp),
            shape = RoundedCornerShape(16.dp),
            colors = ButtonDefaults.buttonColors(
                containerColor = if (hasApprovedCriticalFields) {
                    MaterialTheme.colorScheme.error
                } else {
                    MaterialTheme.colorScheme.primary
                }
            )
        ) {
            if (isAuthenticating) {
                CircularProgressIndicator(
                    modifier = Modifier.size(24.dp),
                    color = MaterialTheme.colorScheme.onPrimary,
                    strokeWidth = 2.dp
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Authenticating...")
            } else {
                Icon(
                    imageVector = if (hasApprovedCriticalFields) Icons.Default.Fingerprint else Icons.Default.Check,
                    contentDescription = null,
                    modifier = Modifier.size(24.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = if (hasApprovedCriticalFields) "Verify & Approve" else "Approve Selected",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold
                )
            }
        }

        // Secondary actions
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            OutlinedButton(
                onClick = onDeny,
                modifier = Modifier
                    .weight(1f)
                    .height(48.dp),
                shape = RoundedCornerShape(12.dp),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = MaterialTheme.colorScheme.error
                )
            ) {
                Icon(Icons.Default.Close, contentDescription = null)
                Spacer(modifier = Modifier.width(4.dp))
                Text("Deny All")
            }

            TextButton(
                onClick = onDismiss,
                modifier = Modifier
                    .weight(1f)
                    .height(48.dp)
            ) {
                Text("Cancel")
            }
        }
    }
}

/**
 * Consent summary data
 */
private data class ConsentSummary(
    val total: Int,
    val approved: Int,
    val criticalApproved: Int
)

/**
 * Preview helper
 */
@Composable
fun ConsentOverlayPreview() {
    val sampleRequest = PiiAccessRequest(
        requestId = "req_123",
        agentId = "agent_checkout_001",
        fields = listOf(
            PiiField("Full Name", SensitivityLevel.LOW, "Required for shipping", "John Doe"),
            PiiField("Address", SensitivityLevel.MEDIUM, "Delivery address", "123 Main St"),
            PiiField("Credit Card", SensitivityLevel.HIGH, "Required for payment", "••••4242"),
            PiiField("CVV", SensitivityLevel.CRITICAL, "Card verification code")
        ),
        reason = "Complete checkout on example.com",
        expiresAt = System.currentTimeMillis() + 60000,
        batchSize = 2
    )

    ConsentOverlay(
        request = sampleRequest,
        onApprove = {},
        onDeny = {},
        onDismiss = {}
    )
}
