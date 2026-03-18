package com.armorclaw.shared.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.SensitivityLevel
import com.armorclaw.shared.ui.theme.AppIcons

/**
 * Sensitivity Badge
 *
 * Displays the sensitivity level of PII data with appropriate color coding.
 * Used in BlindFill cards and PII management screens.
 *
 * ## Color Coding
 * - LOW: Primary container (blue) - Low risk
 * - MEDIUM: Tertiary container (green) - Medium risk
 * - HIGH: Secondary container (yellow/orange) - High risk
 * - CRITICAL: Error container (red) - Critical risk, requires biometric
 *
 * ## Usage
 * ```kotlin
 * PiiField(
 *     name = "Credit Card",
 *     sensitivity = SensitivityLevel.HIGH,
 *     ...
 * )
 *
 * // In UI:
 * SensitivityBadge(sensitivity = field.sensitivity)
 * ```
 */
@Composable
fun SensitivityBadge(
    sensitivity: SensitivityLevel,
    modifier: Modifier = Modifier,
    compact: Boolean = false
) {
    val (containerColor, contentColor, label) = when (sensitivity) {
        SensitivityLevel.LOW -> Triple(
            MaterialTheme.colorScheme.primaryContainer,
            MaterialTheme.colorScheme.onPrimaryContainer,
            if (compact) "L" else "LOW"
        )
        SensitivityLevel.MEDIUM -> Triple(
            MaterialTheme.colorScheme.tertiaryContainer,
            MaterialTheme.colorScheme.onTertiaryContainer,
            if (compact) "M" else "MEDIUM"
        )
        SensitivityLevel.HIGH -> Triple(
            MaterialTheme.colorScheme.secondaryContainer,
            MaterialTheme.colorScheme.onSecondaryContainer,
            if (compact) "H" else "HIGH"
        )
        SensitivityLevel.CRITICAL -> Triple(
            MaterialTheme.colorScheme.errorContainer,
            MaterialTheme.colorScheme.onErrorContainer,
            if (compact) "!" else "CRITICAL"
        )
    }

    Surface(
        modifier = modifier.clip(RoundedCornerShape(4.dp)),
        color = containerColor
    ) {
        Text(
            text = label,
            modifier = Modifier.padding(
                horizontal = if (compact) 4.dp else 6.dp,
                vertical = if (compact) 1.dp else 2.dp
            ),
            style = if (compact) {
                MaterialTheme.typography.labelSmall
            } else {
                MaterialTheme.typography.labelMedium
            },
            fontWeight = FontWeight.Bold,
            color = contentColor
        )
    }
}

/**
 * Extended sensitivity badge with icon and description
 */
@Composable
fun SensitivityBadgeExtended(
    sensitivity: SensitivityLevel,
    modifier: Modifier = Modifier
) {
    val (containerColor, contentColor, label, description) = when (sensitivity) {
        SensitivityLevel.LOW -> Tuple4(
            MaterialTheme.colorScheme.primaryContainer,
            MaterialTheme.colorScheme.onPrimaryContainer,
            "Low Sensitivity",
            "General information like name or email"
        )
        SensitivityLevel.MEDIUM -> Tuple4(
            MaterialTheme.colorScheme.tertiaryContainer,
            MaterialTheme.colorScheme.onTertiaryContainer,
            "Medium Sensitivity",
            "Contact information like address or phone"
        )
        SensitivityLevel.HIGH -> Tuple4(
            MaterialTheme.colorScheme.secondaryContainer,
            MaterialTheme.colorScheme.onSecondaryContainer,
            "High Sensitivity",
            "Financial or personal data like credit card"
        )
        SensitivityLevel.CRITICAL -> Tuple4(
            MaterialTheme.colorScheme.errorContainer,
            MaterialTheme.colorScheme.onErrorContainer,
            "Critical Sensitivity",
            "Highly sensitive data requiring biometric verification"
        )
    }

    Surface(
        modifier = modifier,
        color = containerColor,
        shape = RoundedCornerShape(8.dp)
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(
                text = label,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
                color = contentColor
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = description,
                style = MaterialTheme.typography.bodySmall,
                color = contentColor.copy(alpha = 0.8f)
            )
            if (sensitivity == SensitivityLevel.CRITICAL) {
                Spacer(Modifier.height(8.dp))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = AppIcons.Fingerprint,
                        contentDescription = null,
                        tint = contentColor,
                        modifier = Modifier.size(16.dp)
                    )
                    Text(
                        text = "Biometric verification required",
                        style = MaterialTheme.typography.labelSmall,
                        color = contentColor
                    )
                }
            }
        }
    }
}

/**
 * Helper data class for 4-tuple
 */
private data class Tuple4<A, B, C, D>(
    val first: A,
    val second: B,
    val third: C,
    val fourth: D
)

/**
 * Preview helper
 */
@Composable
fun SensitivityBadgePreview() {
    Column(
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // Compact badges
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            SensitivityBadge(SensitivityLevel.LOW, compact = true)
            SensitivityBadge(SensitivityLevel.MEDIUM, compact = true)
            SensitivityBadge(SensitivityLevel.HIGH, compact = true)
            SensitivityBadge(SensitivityLevel.CRITICAL, compact = true)
        }

        // Full badges
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            SensitivityBadge(SensitivityLevel.LOW)
            SensitivityBadge(SensitivityLevel.MEDIUM)
            SensitivityBadge(SensitivityLevel.HIGH)
            SensitivityBadge(SensitivityLevel.CRITICAL)
        }

        // Extended badges
        SensitivityBadgeExtended(SensitivityLevel.LOW)
        SensitivityBadgeExtended(SensitivityLevel.HIGH)
        SensitivityBadgeExtended(SensitivityLevel.CRITICAL)
    }
}
