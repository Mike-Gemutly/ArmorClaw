package components.vault

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

/**
 * Vault Pulse Indicator
 *
 * Visual indicator that pulses when the Cold Vault is active or
 * when an agent needs PII access. Uses a teal glow animation.
 *
 * Phase 1 Implementation - Governor Strategy
 *
 * @param isActive Whether the vault is currently active
 * @param requiredKeyCount Number of keys required by active agents
 * @param modifier Optional modifier
 */
@Composable
fun VaultPulseIndicator(
    isActive: Boolean,
    requiredKeyCount: Int = 0,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "vault_pulse")

    // Scale animation for pulse effect
    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (isActive) 1.3f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(1000, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    // Alpha animation for glow effect
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.6f,
        targetValue = if (isActive) 1f else 0.6f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    // Teal color for vault indicator
    val vaultTeal = Color(0xFF00BCD4)
    val vaultTealVariant = Color(0xFF0097A7)

    Box(
        modifier = modifier,
        contentAlignment = Alignment.Center
    ) {
        // Outer glow ring
        if (isActive) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .scale(scale)
                    .background(
                        color = vaultTeal.copy(alpha = alpha * 0.3f),
                        shape = CircleShape
                    )
            )
        }

        // Main indicator circle
        Box(
            modifier = Modifier
                .size(24.dp)
                .background(
                    color = if (isActive) vaultTeal else vaultTealVariant.copy(alpha = 0.5f),
                    shape = CircleShape
                ),
            contentAlignment = Alignment.Center
        ) {
            // Lock icon or key count
            if (requiredKeyCount > 0) {
                Text(
                    text = requiredKeyCount.toString(),
                    style = MaterialTheme.typography.labelSmall,
                    color = Color.White
                )
            }
        }
    }
}

/**
 * Vault Status Badge
 *
 * Shows the current vault status with a label
 *
 * @param status Current vault status
 * @param keyCount Number of keys stored
 * @param modifier Optional modifier
 */
@Composable
fun VaultStatusBadge(
    status: VaultStatus,
    keyCount: Int = 0,
    modifier: Modifier = Modifier
) {
    val (color, label) = when (status) {
        VaultStatus.SECURED -> Color(0xFF4CAF50) to "Secured"
        VaultStatus.ACTIVE -> Color(0xFF00BCD4) to "Active"
        VaultStatus.LOCKED -> Color(0xFF9E9E9E) to "Locked"
        VaultStatus.ERROR -> Color(0xFFF44336) to "Error"
    }

    Surface(
        modifier = modifier,
        shape = MaterialTheme.shapes.small,
        color = color.copy(alpha = 0.15f),
        border = androidx.compose.foundation.BorderStroke(1.dp, color)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            VaultPulseIndicator(
                isActive = status == VaultStatus.ACTIVE,
                requiredKeyCount = if (status == VaultStatus.ACTIVE) keyCount else 0,
                modifier = Modifier.size(16.dp)
            )
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = color
            )
        }
    }
}

/**
 * Preview composable for VaultPulseIndicator
 */
@Composable
fun VaultPulseIndicatorPreview() {
    Column(
        modifier = Modifier.padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            VaultPulseIndicator(isActive = false)
            Text("Inactive")
        }
        Row(
            horizontalArrangement = Arrangement.spacedBy(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            VaultPulseIndicator(isActive = true, requiredKeyCount = 3)
            Text("Active (3 keys)")
        }

        Divider(modifier = Modifier.padding(vertical = 8.dp))

        VaultStatusBadge(status = VaultStatus.SECURED)
        VaultStatusBadge(status = VaultStatus.ACTIVE, keyCount = 5)
        VaultStatusBadge(status = VaultStatus.LOCKED)
        VaultStatusBadge(status = VaultStatus.ERROR)
    }
}
