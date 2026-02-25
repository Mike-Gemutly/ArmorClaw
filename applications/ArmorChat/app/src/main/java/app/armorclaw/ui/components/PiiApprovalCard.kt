package app.armorclaw.ui.components

import androidx.compose.foundation.background
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import app.armorclaw.data.model.SystemAlertContent

/**
 * PII Approval Card - Batched HITL Approval for BlindFill Fields
 *
 * Renders a single card with all requested PII fields, sensitivity badges,
 * and per-field toggle controls. Prevents "approval fatigue" by consolidating
 * multiple field requests into one interactive card.
 *
 * Wires to: pii.approve_access / pii.reject_access RPC methods
 */

/**
 * Sensitivity level for visual badges
 */
enum class PiiSensitivityLevel(val displayName: String, val color: Color, val bgColor: Color) {
    LOW("Low", Color(0xFF388E3C), Color(0xFFE8F5E9)),
    MEDIUM("Medium", Color(0xFFF57C00), Color(0xFFFFF3E0)),
    HIGH("High", Color(0xFFE64A19), Color(0xFFFBE9E7)),
    CRITICAL("Critical", Color(0xFFB71C1C), Color(0xFFFFEBEE));

    companion object {
        fun fromString(s: String): PiiSensitivityLevel {
            return values().find { it.name.equals(s, ignoreCase = true) } ?: MEDIUM
        }
    }
}

/**
 * Represents a single PII field in the approval request
 */
data class PiiFieldRequest(
    val key: String,
    val description: String,
    val required: Boolean,
    val sensitivity: PiiSensitivityLevel
)

/**
 * Main PII Approval Card composable
 *
 * @param alert The system alert containing PII request metadata
 * @param onApprove Called with (requestId, approvedFieldKeys) when user approves
 * @param onDeny Called with (requestId, reason) when user denies all
 */
@Composable
fun PiiApprovalCard(
    alert: SystemAlertContent,
    onApprove: (String, List<String>) -> Unit,
    onDeny: (String, String) -> Unit,
    modifier: Modifier = Modifier
) {
    val metadata = alert.metadata ?: return
    val requestId = metadata["request_id"] as? String ?: return
    val skillName = metadata["skill_name"] as? String ?: "Unknown Skill"
    val hasCritical = metadata["has_critical"] as? Boolean ?: false

    // Parse fields from metadata
    val fields = remember(metadata) {
        @Suppress("UNCHECKED_CAST")
        val rawFields = metadata["fields"] as? List<Map<String, Any>> ?: emptyList()
        rawFields.map { fieldMap ->
            PiiFieldRequest(
                key = fieldMap["key"] as? String ?: "unknown",
                description = fieldMap["description"] as? String ?: "",
                required = fieldMap["required"] == true,
                sensitivity = PiiSensitivityLevel.fromString(
                    fieldMap["sensitivity"] as? String ?: "medium"
                )
            )
        }
    }

    // Track per-field approval state
    val fieldApprovals = remember(fields) {
        mutableStateMapOf<String, Boolean>().apply {
            fields.forEach { field ->
                // Required fields default to approved, optional default to approved too
                put(field.key, true)
            }
        }
    }

    var showReviewMode by remember { mutableStateOf(false) }

    val borderColor = if (hasCritical) Color(0xFFF44336) else Color(0xFFFFC107)
    val bgColor = if (hasCritical) Color(0xFFFFF8E1) else Color(0xFFF3F8FF)

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = bgColor),
        border = androidx.compose.foundation.BorderStroke(1.5.dp, borderColor)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        ) {
            // Header
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.PrivacyTip,
                    contentDescription = null,
                    tint = if (hasCritical) Color(0xFFD32F2F) else Color(0xFFF57C00),
                    modifier = Modifier.size(28.dp)
                )
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "PII Access Request",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Text(
                        text = "Skill: $skillName",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                if (hasCritical) {
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = Color(0xFFFFCDD2)
                    ) {
                        Text(
                            text = "CRITICAL",
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Bold,
                            color = Color(0xFFB71C1C),
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // Field list
            if (showReviewMode) {
                // Detailed per-field review with toggles
                fields.forEach { field ->
                    PiiFieldRow(
                        field = field,
                        isApproved = fieldApprovals[field.key] ?: true,
                        onToggle = { approved ->
                            // Don't allow toggling off required fields
                            if (!field.required || approved) {
                                fieldApprovals[field.key] = approved
                            }
                        }
                    )
                }
            } else {
                // Compact summary
                Text(
                    text = "Requesting ${fields.size} field(s):",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium
                )
                Spacer(modifier = Modifier.height(4.dp))

                // Compact field chips
                FlowRow(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    fields.forEach { field ->
                        PiiFieldChip(field = field)
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Action buttons
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                // Deny button
                OutlinedButton(
                    onClick = { onDeny(requestId, "User denied PII access") },
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

                // Review / Approve toggle
                if (!showReviewMode) {
                    OutlinedButton(
                        onClick = { showReviewMode = true },
                        modifier = Modifier.weight(1f)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Visibility,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Review")
                    }
                }

                // Approve button
                Button(
                    onClick = {
                        val approvedKeys = fieldApprovals
                            .filter { it.value }
                            .map { it.key }
                        onApprove(requestId, approvedKeys)
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
                    Text(if (showReviewMode) "Approve Selected" else "Approve All")
                }
            }
        }
    }
}

/**
 * Single field row in review mode with toggle
 */
@Composable
private fun PiiFieldRow(
    field: PiiFieldRequest,
    isApproved: Boolean,
    onToggle: (Boolean) -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.weight(1f)
        ) {
            // Sensitivity badge
            Surface(
                shape = RoundedCornerShape(4.dp),
                color = field.sensitivity.bgColor
            ) {
                Text(
                    text = field.sensitivity.displayName.uppercase(),
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Bold,
                    color = field.sensitivity.color,
                    modifier = Modifier.padding(horizontal = 4.dp, vertical = 1.dp)
                )
            }

            Column {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = field.key,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                    if (field.required) {
                        Text(
                            text = " *",
                            style = MaterialTheme.typography.bodyMedium,
                            color = Color(0xFFD32F2F),
                            fontWeight = FontWeight.Bold
                        )
                    }
                }
                if (field.description.isNotEmpty()) {
                    Text(
                        text = field.description,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }
        }

        Switch(
            checked = isApproved,
            onCheckedChange = onToggle,
            enabled = !field.required || isApproved, // Required fields can't be unchecked
            colors = SwitchDefaults.colors(
                checkedThumbColor = Color(0xFF388E3C),
                checkedTrackColor = Color(0xFFA5D6A7)
            )
        )
    }
}

/**
 * Compact chip for field display in summary mode
 */
@Composable
private fun PiiFieldChip(field: PiiFieldRequest) {
    Surface(
        shape = RoundedCornerShape(16.dp),
        color = field.sensitivity.bgColor,
        border = androidx.compose.foundation.BorderStroke(
            0.5.dp,
            field.sensitivity.color.copy(alpha = 0.3f)
        )
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = field.key,
                style = MaterialTheme.typography.labelSmall,
                color = field.sensitivity.color,
                fontWeight = if (field.required) FontWeight.Bold else FontWeight.Normal
            )
            if (field.required) {
                Text(
                    text = "*",
                    style = MaterialTheme.typography.labelSmall,
                    color = Color(0xFFD32F2F),
                    fontWeight = FontWeight.Bold
                )
            }
        }
    }
}
