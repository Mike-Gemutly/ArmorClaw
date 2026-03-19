package com.armorclaw.app.secretary.ui

import androidx.compose.foundation.background.modifier.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import com.armorclaw.shared.secretary.*

/**
 * ProactiveCard - Compose UI component
 *
 * Displays a notification card with:
 * - Title and description
 * - Priority indicator
 * - Primary action button
 * - Dismiss button (if dismissible)
 *
 * Phase 1 UI component - presentational only, no business logic.
 */
@Composable
fun ProactiveCard(
    card: ProactiveCard,
    onPrimaryAction: () -> Unit = {},
    onDismiss: () -> Unit = {}
) {
    // Color based on priority
    val cardColor = when (card.priority) {
        SecretaryPriority.LOW -> MaterialTheme.colorScheme.secondary
        SecretaryPriority.NORMAL -> MaterialTheme.colorScheme.primary
        SecretaryPriority.HIGH -> MaterialTheme.colorScheme.tertiary
        SecretaryPriority.CRITICAL -> MaterialTheme.colorScheme.error
    }

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp)
            .background(cardColor)
            .clickable { onPrimaryAction() },
        shape = MaterialTheme.shapes.medium,
        elevation = CardDefaults.cardElevation(defaultElevation = 4.dp),
    ) {
        Column(
            modifier = Modifier.padding(12.dp)
        ) {
            // Header
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Priority icon
                Box(
                    modifier = Modifier
                        .size(24.dp)
                        .background(
                            when (card.priority) {
                                SecretaryPriority.LOW -> MaterialTheme.colorScheme.secondary.copy(alpha = 0.2f)
                                SecretaryPriority.NORMAL -> MaterialTheme.colorScheme.primary.copy(alpha = 0.4f)
                                SecretaryPriority.HIGH -> MaterialTheme.colorScheme.tertiary.copy(alpha = 0.6f)
                                SecretaryPriority.CRITICAL -> MaterialTheme.colorScheme.error.copy(alpha = 0.8f)
                            }
                        ),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = when (card.priority) {
                            SecretaryPriority.LOW -> Icons.Outlined.Info
                            SecretaryPriority.NORMAL -> Icons.Outlined.Info
                            SecretaryPriority.HIGH -> Icons.Outlined.Warning
                            SecretaryPriority.CRITICAL -> Icons.Outlined.Error
                        },
                        contentDescription = null,
                        tint = when (card.priority) {
                            SecretaryPriority.LOW -> cardColor
                            SecretaryPriority.NORMAL -> cardColor
                            SecretaryPriority.HIGH -> cardColor
                            SecretaryPriority.CRITICAL -> cardColor
                        }
                    )
                }

                // Title
                Text(
                    text = card.title,
                    style = MaterialTheme.typography.titleMedium,
                    color = cardColor,
                    fontWeight = FontWeight.SemiBold
                )

                // Description
                if (card.description.isNotBlank()) {
                    Text(
                        text = card.description,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.padding(top = 8.dp)
                    )
                }

                // Primary action button
                Button(
                    onClick = onPrimaryAction,
                    modifier = Modifier
                        .padding(horizontal = 8.dp)
                        .background(
                            color = cardColor,
                            shape = RoundedCornerShape(8.dp)
                        )
                        .height(48.dp),
                    enabled = true,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = cardColor
                    ),
                ) {
                    Text(
                        "View Details",
                        style = MaterialTheme.typography.labelMedium
                    )
                }

                // Spacer
                Spacer(modifier = Modifier.height(16.dp))
            }
        }

        // Dismiss button (if card is dismissible)
        if (card.dismissible) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalArrangement = Arrangement.End,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Spacer(modifier = Modifier.weight(1f))
                TextButton(
                    text = "Dismiss",
                    onClick = onDismiss,
                    colors = TextButtonDefaults.textButtonColors(
                        contentColor = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                )
            }
        }
    }
}
