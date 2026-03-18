package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.foundation.clickable
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.PiiAccessRequest
import com.armorclaw.shared.domain.model.PiiField
import com.armorclaw.shared.domain.model.SensitivityLevel

/**
 * BlindFill Card
 *
 * Displays a request from an AI agent to access sensitive user data.
 * Allows users to selectively approve or deny access to individual fields.
 *
 * ## Security Model
 * - All fields start selected (approved) by default
 * - Users can deselect fields they don't want to share
 * - CRITICAL fields require biometric authentication on approve
 *
 * ## Usage (Compact Mode)
 * ```kotlin
 * val request by viewModel.pendingPiiRequest.collectAsState()
 *
 * request?.let { piiRequest ->
 *     BlindFillCard(
 *         request = piiRequest,
 *         onApprove = { approvedFields -> viewModel.approvePiiRequest(approvedFields) },
 *         onDeny = { viewModel.denyPiiRequest() }
 *     )
 * }
 * ```
 *
 * ## Usage (Fullscreen Mode)
 * ```kotlin
 * var showOverlay by remember { mutableStateOf(false) }
 *
 * BlindFillCard(
 *     request = piiRequest,
 *     fullscreen = true,
 *     isBiometricAuthenticating = isAuthenticating,
 *     onApprove = { fields -> viewModel.approvePiiRequest(fields) },
 *     onDeny = { viewModel.denyPiiRequest() },
 *     onDismiss = { showOverlay = false }
 * )
 * ```
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BlindFillCard(
    request: PiiAccessRequest,
    onApprove: (Set<String>) -> Unit,
    onDeny: () -> Unit,
    modifier: Modifier = Modifier,
    onBiometricRequired: (() -> Unit)? = null,
    fullscreen: Boolean = false,
    onDismiss: (() -> Unit)? = null,
    isBiometricAuthenticating: Boolean = false,
    biometricError: String? = null
) {
    if (fullscreen) {
        // Use ConsentOverlay for fullscreen mode
        ConsentOverlay(
            request = request,
            onApprove = onApprove,
            onDeny = onDeny,
            onDismiss = onDismiss ?: {},
            onBiometricRequired = onBiometricRequired ?: {},
            isBiometricAuthenticating = isBiometricAuthenticating,
            biometricError = biometricError,
            modifier = modifier
        )
    } else {
        // Use compact card mode
        BlindFillCardCompact(
            request = request,
            onApprove = onApprove,
            onDeny = onDeny,
            onBiometricRequired = onBiometricRequired,
            modifier = modifier
        )
    }
}

/**
 * Compact card version (original implementation)
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun BlindFillCardCompact(
    request: PiiAccessRequest,
    onApprove: (Set<String>) -> Unit,
    onDeny: () -> Unit,
    modifier: Modifier = Modifier,
    onBiometricRequired: (() -> Unit)? = null
) {
    // Track approved fields (all start approved by default)
    var approvedFields by remember(request.requestId) {
        mutableStateOf(request.fields.map { it.name }.toSet())
    }

    // Check if any critical fields are approved
    val hasApprovedCriticalFields = request.fields.any {
        it.sensitivity == SensitivityLevel.CRITICAL && it.name in approvedFields
    }

    // Check if request is expired
    val isExpired = remember(request.expiresAt) {
        derivedStateOf { request.isExpired() }
    }

    ElevatedCard(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surface
        )
    ) {
        Column(
            modifier = Modifier
                .padding(16.dp)
                .fillMaxWidth()
        ) {
            // Header
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth()
            ) {
                Icon(
                    imageVector = Icons.Default.Lock,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(24.dp)
                )
                Spacer(Modifier.width(10.dp))
                Text(
                    text = "Remote Access Request",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold
                )
            }

            Spacer(Modifier.height(12.dp))

            // Agent and reason
            Text(
                text = "Agent \"${request.agentId.take(20)}\" requests access:",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = request.reason,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )

            // Expired warning
            if (isExpired.value) {
                Spacer(Modifier.height(8.dp))
                Surface(
                    color = MaterialTheme.colorScheme.errorContainer,
                    shape = RoundedCornerShape(6.dp)
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = Icons.Default.Warning,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.error,
                            modifier = Modifier.size(16.dp)
                        )
                        Spacer(Modifier.width(6.dp))
                        Text(
                            text = "This request has expired",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onErrorContainer
                        )
                    }
                }
            }

            Spacer(Modifier.height(16.dp))

            // Field list header
            Text(
                text = "Requested Fields:",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(Modifier.height(8.dp))

            // Field list
            Column(
                modifier = Modifier
                    .heightIn(max = 200.dp)
                    .verticalScroll(rememberScrollState())
            ) {
                request.fields.forEach { field ->
                    PiiFieldItem(
                        field = field,
                        isApproved = approvedFields.contains(field.name),
                        onToggle = {
                            approvedFields = if (approvedFields.contains(field.name)) {
                                approvedFields - field.name
                            } else {
                                approvedFields + field.name
                            }
                        },
                        enabled = !isExpired.value
                    )
                    Spacer(Modifier.height(4.dp))
                }
            }

            // Batch indicator
            if (request.batchSize > 1) {
                Spacer(Modifier.height(8.dp))
                Surface(
                    color = MaterialTheme.colorScheme.tertiaryContainer,
                    shape = RoundedCornerShape(6.dp)
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = Icons.Default.Layers,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onTertiaryContainer,
                            modifier = Modifier.size(16.dp)
                        )
                        Spacer(Modifier.width(6.dp))
                        Text(
                            text = "You have ${request.batchSize} similar requests pending",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onTertiaryContainer
                        )
                    }
                }
            }

            // Action buttons
            Spacer(Modifier.height(16.dp))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                OutlinedButton(
                    onClick = onDeny,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp)
                    )
                    Spacer(Modifier.width(4.dp))
                    Text("Deny All")
                }

                Button(
                    onClick = {
                        if (hasApprovedCriticalFields && onBiometricRequired != null) {
                            onBiometricRequired()
                        } else {
                            onApprove(approvedFields)
                        }
                    },
                    enabled = approvedFields.isNotEmpty() && !isExpired.value,
                    modifier = Modifier.weight(1f)
                ) {
                    if (hasApprovedCriticalFields) {
                        Icon(
                            imageVector = Icons.Default.Fingerprint,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                    } else {
                        Icon(
                            imageVector = Icons.Default.Check,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                    }
                    Spacer(Modifier.width(4.dp))
                    Text(if (hasApprovedCriticalFields) "Verify & Approve" else "Approve")
                }
            }

            // Approval count
            if (approvedFields.size != request.fields.size) {
                Spacer(Modifier.height(8.dp))
                Text(
                    text = "${approvedFields.size} of ${request.fields.size} fields selected",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.outline
                )
            }
        }
    }
}

/**
 * Individual PII field item with checkbox and sensitivity badge
 */
@Composable
fun PiiFieldItem(
    field: PiiField,
    isApproved: Boolean,
    onToggle: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true
) {
    Surface(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(enabled = enabled) { onToggle() },
        color = if (isApproved && enabled) {
            MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
        } else {
            MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
        }
    ) {
        Row(
            modifier = Modifier
                .padding(horizontal = 10.dp, vertical = 8.dp)
                .fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Checkbox(
                checked = isApproved,
                onCheckedChange = { onToggle() },
                enabled = enabled,
                modifier = Modifier.size(20.dp)
            )
            Spacer(Modifier.width(10.dp))

            Column(modifier = Modifier.weight(1f)) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Text(
                        text = field.name,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f, fill = false)
                    )
                    SensitivityBadge(
                        sensitivity = field.sensitivity,
                        compact = true
                    )
                }

                Text(
                    text = field.description,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                // Show masked current value if available
                if (field.currentValue != null) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = field.currentValue,
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.outline
                    )
                }
            }
        }
    }
}

/**
 * Preview helper
 */
@Composable
fun BlindFillCardPreview() {
    val sampleRequest = PiiAccessRequest(
        requestId = "req_123",
        agentId = "agent_browse_001",
        fields = listOf(
            PiiField(
                name = "Full Name",
                sensitivity = SensitivityLevel.LOW,
                description = "Required for shipping",
                currentValue = "John Doe"
            ),
            PiiField(
                name = "Credit Card Number",
                sensitivity = SensitivityLevel.HIGH,
                description = "Required for payment",
                currentValue = "••••4242"
            ),
            PiiField(
                name = "CVV",
                sensitivity = SensitivityLevel.CRITICAL,
                description = "Required for card verification"
            )
        ),
        reason = "Complete checkout on example.com",
        expiresAt = System.currentTimeMillis() + 30000,
        batchSize = 3
    )

    BlindFillCard(
        request = sampleRequest,
        onApprove = {},
        onDeny = {}
    )
}
