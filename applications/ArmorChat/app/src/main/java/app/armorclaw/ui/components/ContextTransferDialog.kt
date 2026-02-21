package app.armorclaw.ui.components

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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog

/**
 * Context Transfer Cost Estimation Dialog
 *
 * Resolves: Gap - Context Transfer Quota (Cost Control)
 *
 * Shows a warning dialog before transferring context that would consume
 * significant token budget. Prevents accidental budget exhaustion.
 */

/**
 * Context transfer cost estimate
 */
data class ContextTransferEstimate(
    val sourceAgentName: String,
    val targetAgentName: String,
    val contentType: ContentType,
    val contentSizeBytes: Long,
    val estimatedTokens: Int,
    val estimatedCostUSD: Double,
    val currentBudgetRemaining: Double,
    val budgetAfterTransfer: Double,
    val willExhaustBudget: Boolean
) {
    /**
     * Risk level for the transfer
     */
    val riskLevel: RiskLevel
        get() = when {
            willExhaustBudget -> RiskLevel.CRITICAL
            budgetAfterTransfer < 1.0 -> RiskLevel.HIGH
            budgetAfterTransfer < currentBudgetRemaining * 0.2 -> RiskLevel.MEDIUM
            else -> RiskLevel.LOW
        }

    /**
     * Whether the transfer should be blocked
     */
    val shouldBlock: Boolean
        get() = willExhaustBudget || budgetAfterTransfer < 0
}

/**
 * Content type being transferred
 */
enum class ContentType(val displayName: String, val icon: androidx.compose.ui.graphics.vector.ImageVector) {
    TEXT("Text", Icons.Default.TextFields),
    FILE("File", Icons.Default.AttachFile),
    IMAGE("Image", Icons.Default.Image),
    PDF("PDF", Icons.Default.PictureAsPdf),
    CONVERSATION("Conversation History", Icons.Default.Chat),
    CODE("Code Snippet", Icons.Default.Code)
}

/**
 * Risk level for the transfer
 */
enum class RiskLevel(val color: Color, val displayName: String) {
    LOW(Color(0xFF4CAF50), "Low Impact"),
    MEDIUM(Color(0xFFFFC107), "Moderate Impact"),
    HIGH(Color(0xFFFF9800), "High Impact"),
    CRITICAL(Color(0xFFF44336), "Critical Impact")
}

/**
 * Context Transfer Warning Dialog
 */
@Composable
fun ContextTransferWarningDialog(
    estimate: ContextTransferEstimate,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier
) {
    Dialog(onDismissRequest = onDismiss) {
        Card(
            modifier = modifier
                .fillMaxWidth()
                .padding(16.dp),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(
                containerColor = if (estimate.willExhaustBudget)
                    Color(0xFFFFEBEE) // Red 50
                else
                    MaterialTheme.colorScheme.surface
            )
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                // Icon
                Icon(
                    imageVector = if (estimate.willExhaustBudget)
                        Icons.Default.Warning
                    else
                        Icons.Default.Info,
                    contentDescription = null,
                    tint = estimate.riskLevel.color,
                    modifier = Modifier.size(48.dp)
                )

                Spacer(modifier = Modifier.height(16.dp))

                // Title
                Text(
                    text = if (estimate.shouldBlock)
                        "Context Transfer Blocked"
                    else
                        "Context Transfer Warning",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold,
                    color = if (estimate.willExhaustBudget)
                        Color(0xFFB71C1C)
                    else
                        MaterialTheme.colorScheme.onSurface
                )

                Spacer(modifier = Modifier.height(8.dp))

                // Transfer summary
                Text(
                    text = "Transfer ${estimate.contentType.displayName} from ${estimate.sourceAgentName} to ${estimate.targetAgentName}",
                    style = MaterialTheme.typography.bodyMedium,
                    textAlign = TextAlign.Center,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                Spacer(modifier = Modifier.height(24.dp))

                // Cost breakdown card
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.surfaceVariant
                    )
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp)
                    ) {
                        // Token estimate
                        CostRow(
                            label = "Estimated Tokens",
                            value = "~${formatNumber(estimate.estimatedTokens)}",
                            icon = Icons.Default.Token
                        )

                        Spacer(modifier = Modifier.height(8.dp))

                        // Cost estimate
                        CostRow(
                            label = "Estimated Cost",
                            value = "$${String.format("%.4f", estimate.estimatedCostUSD)}",
                            icon = Icons.Default.AttachMoney
                        )

                        Spacer(modifier = Modifier.height(8.dp))

                        HorizontalDivider()

                        Spacer(modifier = Modifier.height(8.dp))

                        // Budget remaining
                        CostRow(
                            label = "Current Budget",
                            value = "$${String.format("%.2f", estimate.currentBudgetRemaining)}",
                            icon = Icons.Default.AccountBalanceWallet,
                            valueColor = MaterialTheme.colorScheme.primary
                        )

                        Spacer(modifier = Modifier.height(8.dp))

                        // Budget after transfer
                        CostRow(
                            label = "After Transfer",
                            value = "$${String.format("%.2f", estimate.budgetAfterTransfer)}",
                            icon = Icons.Default.RemoveCircleOutline,
                            valueColor = if (estimate.budgetAfterTransfer < estimate.currentBudgetRemaining * 0.2)
                                estimate.riskLevel.color
                            else
                                MaterialTheme.colorScheme.onSurface
                        )
                    }
                }

                Spacer(modifier = Modifier.height(16.dp))

                // Risk indicator
                if (!estimate.shouldBlock) {
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = estimate.riskLevel.color.copy(alpha = 0.1f)
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(6.dp)
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(8.dp)
                                    .padding(1.dp),
                                contentAlignment = Alignment.Center
                            ) {
                                Box(
                                    modifier = Modifier
                                        .fillMaxSize()
                                        .background(estimate.riskLevel.color, RoundedCornerShape(50))
                                )
                            }
                            Text(
                                text = estimate.riskLevel.displayName,
                                style = MaterialTheme.typography.labelMedium,
                                color = estimate.riskLevel.color
                            )
                        }
                    }
                }

                // Warning message for blocked transfer
                if (estimate.shouldBlock) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = "This transfer would exhaust your token budget. " +
                               "Please add funds or wait for budget reset before transferring.",
                        style = MaterialTheme.typography.bodySmall,
                        color = Color(0xFFB71C1C),
                        textAlign = TextAlign.Center
                    )
                }

                Spacer(modifier = Modifier.height(24.dp))

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

                    if (!estimate.shouldBlock) {
                        Button(
                            onClick = onConfirm,
                            modifier = Modifier.weight(1f),
                            colors = ButtonDefaults.buttonColors(
                                containerColor = if (estimate.riskLevel == RiskLevel.HIGH ||
                                                     estimate.riskLevel == RiskLevel.CRITICAL)
                                    Color(0xFFFF9800)
                                else
                                    MaterialTheme.colorScheme.primary
                            )
                        ) {
                            Text("Transfer Anyway")
                        }
                    } else {
                        Button(
                            onClick = onDismiss,
                            modifier = Modifier.weight(1f),
                            colors = ButtonDefaults.buttonColors(
                                containerColor = MaterialTheme.colorScheme.primary
                            )
                        ) {
                            Text("OK")
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun CostRow(
    label: String,
    value: String,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    valueColor: Color = MaterialTheme.colorScheme.onSurface
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                modifier = Modifier.size(18.dp),
                tint = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Text(
                text = label,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium,
            color = valueColor
        )
    }
}

/**
 * Utility function to estimate tokens from content
 */
fun estimateTokens(content: String, contentType: ContentType): Int {
    // Rough estimation: ~4 characters per token for English text
    // This is a simplified estimate; actual tokenization varies by model
    val baseChars = content.length

    val multiplier = when (contentType) {
        ContentType.TEXT -> 1.0
        ContentType.CODE -> 1.2 // Code tends to have more tokens per character
        ContentType.CONVERSATION -> 1.0
        ContentType.FILE -> 1.5 // Files often contain structured data
        ContentType.PDF -> 2.0 // PDFs may have formatting overhead
        ContentType.IMAGE -> 0 // Images are handled separately (vision models)
    }

    return (baseChars / 4.0 * multiplier).toInt()
}

/**
 * Utility function to estimate tokens from file size
 */
fun estimateTokensFromFileSize(sizeBytes: Long, contentType: ContentType): Int {
    // Rough estimation based on file size
    // Text files: ~1 byte per character
    // Assume average of 4 characters per token
    return when (contentType) {
        ContentType.TEXT -> (sizeBytes / 4).toInt()
        ContentType.CODE -> (sizeBytes / 3.5).toInt()
        ContentType.PDF -> (sizeBytes / 2).toInt() // PDFs are denser
        else -> (sizeBytes / 4).toInt()
    }
}

/**
 * Format large numbers with K/M suffix
 */
private fun formatNumber(num: Int): String {
    return when {
        num >= 1_000_000 -> "${num / 1_000_000}M"
        num >= 1_000 -> "${num / 1_000}K"
        else -> num.toString()
    }
}

// Add missing import
import androidx.compose.foundation.background
