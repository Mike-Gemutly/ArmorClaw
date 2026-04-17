package app.armorclaw.ui.components

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp

/**
 * GovernanceBanner — Compact status banner for agent workflow state.
 *
 * Maps to Go WorkflowStatus values: pending, running, blocked, completed, failed, cancelled.
 * BLOCKED and RUNNING are visually prominent; other states are subtle or hidden.
 */
@Composable
fun GovernanceBanner(
    status: WorkflowStatus,
    currentStep: Int = 0,
    totalSteps: Int = 0,
    onBlockedTap: (() -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    when (status) {
        WorkflowStatus.IDLE -> {
            // No banner for idle
        }
        WorkflowStatus.RUNNING -> {
            RunningBanner(
                currentStep = currentStep,
                totalSteps = totalSteps,
                modifier = modifier
            )
        }
        WorkflowStatus.BLOCKED -> {
            BlockedBanner(
                onTap = onBlockedTap,
                modifier = modifier
            )
        }
        WorkflowStatus.COMPLETED -> {
            CompletedBanner(modifier = modifier)
        }
        WorkflowStatus.FAILED -> {
            FailedBanner(modifier = modifier)
        }
        WorkflowStatus.CANCELLED -> {
            CancelledBanner(modifier = modifier)
        }
    }
}

// ── RUNNING ──────────────────────────────────────────────────────────────

@Composable
private fun RunningBanner(
    currentStep: Int,
    totalSteps: Int,
    modifier: Modifier = Modifier
) {
    val containerColor = MaterialTheme.colorScheme.primaryContainer
    val contentColor = MaterialTheme.colorScheme.onPrimaryContainer
    val indicatorColor = MaterialTheme.colorScheme.primary

    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        color = containerColor,
        tonalElevation = 1.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            // Pulsing indicator dot
            PulsingIndicator(color = indicatorColor)

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "Running",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = contentColor
                )
                if (totalSteps > 0) {
                    Text(
                        text = "Step $currentStep of $totalSteps",
                        style = MaterialTheme.typography.bodySmall,
                        color = contentColor.copy(alpha = 0.8f)
                    )
                }
            }
        }
    }
}

// ── BLOCKED ──────────────────────────────────────────────────────────────

@Composable
private fun BlockedBanner(
    onTap: (() -> Unit)?,
    modifier: Modifier = Modifier
) {
    // Warning yellow palette — using tertiary colors when available,
    // falling back to explicit amber for predictability.
    val background = Color(0xFFFFF8E1)   // Amber 50
    val border = Color(0xFFFFC107)       // Amber 500
    val text = Color(0xFFE65100)         // Orange 900
    val iconColor = Color(0xFFF57F17)    // Yellow 900

    val targetModifier = if (onTap != null) {
        modifier
            .fillMaxWidth()
            .clickable(onClick = onTap)
    } else {
        modifier.fillMaxWidth()
    }

    Surface(
        modifier = targetModifier,
        shape = RoundedCornerShape(8.dp),
        color = background,
        border = BorderStroke(1.dp, border),
        tonalElevation = 2.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            Text(
                text = "\uD83D\uDEA7",  // 🚧
                style = MaterialTheme.typography.titleMedium
            )
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "Action Required",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold,
                    color = text
                )
                Text(
                    text = "Agent is waiting for your input",
                    style = MaterialTheme.typography.bodySmall,
                    color = text.copy(alpha = 0.75f),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }
            if (onTap != null) {
                Icon(
                    imageVector = androidx.compose.material.icons.Icons.Default.ArrowForward,
                    contentDescription = "Resolve blocker",
                    tint = iconColor,
                    modifier = Modifier.size(18.dp)
                )
            }
        }
    }
}

// ── COMPLETED ────────────────────────────────────────────────────────────

@Composable
private fun CompletedBanner(modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.secondaryContainer,
        tonalElevation = 0.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            Text(
                text = "\u2705",  // ✅
                style = MaterialTheme.typography.titleMedium
            )
            Text(
                text = "Completed",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSecondaryContainer
            )
        }
    }
}

// ── FAILED ───────────────────────────────────────────────────────────────

@Composable
private fun FailedBanner(modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.errorContainer,
        tonalElevation = 0.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            Text(
                text = "\u274C",  // ❌
                style = MaterialTheme.typography.titleMedium
            )
            Text(
                text = "Failed",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onErrorContainer
            )
        }
    }
}

// ── CANCELLED ────────────────────────────────────────────────────────────

@Composable
private fun CancelledBanner(modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.surfaceVariant,
        tonalElevation = 0.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp)
        ) {
            Text(
                text = "\u23F9\uFE0F",  // ⏹️
                style = MaterialTheme.typography.titleMedium
            )
            Text(
                text = "Cancelled",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

// ── Pulsing Indicator ────────────────────────────────────────────────────

@Composable
private fun PulsingIndicator(color: Color) {
    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = 0.3f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 800),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulseAlpha"
    )

    Box(
        modifier = Modifier
            .size(10.dp)
            .clip(RoundedCornerShape(5.dp))
            .background(color.copy(alpha = alpha))
    )
}

// ── Workflow Status Enum ──────────────────────────────────────────────────

/**
 * Maps to Go WorkflowStatus: pending, running, blocked, completed, failed, cancelled.
 */
@Immutable
enum class WorkflowStatus {
    IDLE,
    RUNNING,
    BLOCKED,
    COMPLETED,
    FAILED,
    CANCELLED;

    companion object {
        /** Parse from Go-side lowercase string. */
        fun fromGo(value: String): WorkflowStatus =
            entries.firstOrNull { it.name.equals(value, ignoreCase = true) } ?: IDLE
    }
}
