package components.vault

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Vault Key Panel
 *
 * Sidebar panel displaying all vault keys with their status.
 * Shows required keys with a pulsing indicator.
 *
 * Phase 1 Implementation - Governor Strategy
 *
 * @param keys List of vault keys to display
 * @param requiredKeys Set of field names currently required by agents
 * @param onKeyClick Callback when a key is clicked
 * @param modifier Optional modifier
 */
@Composable
fun VaultKeyPanel(
    keys: List<VaultKeyUi>,
    requiredKeys: Set<String> = emptySet(),
    onKeyClick: (VaultKeyUi) -> Unit = {},
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(8.dp)
    ) {
        // Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = "Cold Vault",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold
            )
            VaultStatusBadge(
                status = if (requiredKeys.isNotEmpty()) VaultStatus.ACTIVE else VaultStatus.SECURED,
                keyCount = requiredKeys.size
            )
        }

        Divider(modifier = Modifier.padding(vertical = 8.dp))

        // Key count summary
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            SummaryChip(
                label = "${keys.size} Keys",
                icon = Icons.Default.Lock
            )
            if (requiredKeys.isNotEmpty()) {
                SummaryChip(
                    label = "${requiredKeys.size} Required",
                    icon = Icons.Default.Info,
                    color = Color(0xFF00BCD4)
                )
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        // Key list by category
        val keysByCategory = keys.groupBy { it.category }

        LazyColumn(
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            keysByCategory.forEach { (category, categoryKeys) ->
                // Category header
                item {
                    Text(
                        text = getCategoryLabel(category),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 8.dp, bottom = 4.dp)
                    )
                }

                // Keys in this category
                items(categoryKeys) { key ->
                    VaultKeyCard(
                        key = key,
                        isRequired = requiredKeys.contains(key.fieldName),
                        onClick = { onKeyClick(key) }
                    )
                }
            }
        }
    }
}

/**
 * Vault Key Card
 *
 * Individual card for a vault key
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VaultKeyCard(
    key: VaultKeyUi,
    isRequired: Boolean = false,
    onClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val sensitivityColor = getSensitivityColor(key.sensitivity)

    Card(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (isRequired)
                Color(0xFF00BCD4).copy(alpha = 0.1f)
            else
                MaterialTheme.colorScheme.surface
        ),
        onClick = onClick
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                // Sensitivity indicator
                Box(
                    modifier = Modifier
                        .size(4.dp, 24.dp)
                        .background(sensitivityColor, RoundedCornerShape(2.dp))
                )

                Column {
                    Text(
                        text = key.displayName,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = if (isRequired) FontWeight.Bold else FontWeight.Normal
                    )
                    Text(
                        text = key.fieldName,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            // Status indicators
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                if (isRequired) {
                    VaultPulseIndicator(
                        isActive = true,
                        modifier = Modifier.size(16.dp)
                    )
                }
                if (key.accessCount > 0) {
                    Text(
                        text = "${key.accessCount}",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }
    }
}

/**
 * Summary Chip
 */
@Composable
private fun SummaryChip(
    label: String,
    icon: ImageVector,
    color: Color = MaterialTheme.colorScheme.primary,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = color.copy(alpha = 0.1f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                modifier = Modifier.size(14.dp),
                tint = color
            )
            Text(
                text = label,
                style = MaterialTheme.typography.labelSmall,
                color = color
            )
        }
    }
}

// Helper functions

private fun getCategoryLabel(category: VaultKeyCategory): String {
    return when (category) {
        VaultKeyCategory.PERSONAL -> "Personal"
        VaultKeyCategory.FINANCIAL -> "Financial"
        VaultKeyCategory.CONTACT -> "Contact"
        VaultKeyCategory.AUTHENTICATION -> "Authentication"
        VaultKeyCategory.MEDICAL -> "Medical"
        VaultKeyCategory.OTHER -> "Other"
    }
}

private fun getSensitivityColor(sensitivity: VaultKeySensitivity): Color {
    return when (sensitivity) {
        VaultKeySensitivity.LOW -> Color(0xFF4CAF50)
        VaultKeySensitivity.MEDIUM -> Color(0xFFFFC107)
        VaultKeySensitivity.HIGH -> Color(0xFFFF9800)
        VaultKeySensitivity.CRITICAL -> Color(0xFFF44336)
    }
}

/**
 * Preview composable for VaultKeyPanel
 */
@Composable
fun VaultKeyPanelPreview() {
    val sampleKeys = listOf(
        VaultKeyUi(
            id = "1",
            fieldName = "full_name",
            displayName = "Full Name",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = System.currentTimeMillis(),
            accessCount = 5
        ),
        VaultKeyUi(
            id = "2",
            fieldName = "email",
            displayName = "Email Address",
            category = VaultKeyCategory.CONTACT,
            sensitivity = VaultKeySensitivity.MEDIUM,
            lastAccessed = null,
            accessCount = 0
        ),
        VaultKeyUi(
            id = "3",
            fieldName = "ssn",
            displayName = "Social Security Number",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.CRITICAL,
            lastAccessed = System.currentTimeMillis(),
            accessCount = 2
        ),
        VaultKeyUi(
            id = "4",
            fieldName = "credit_card",
            displayName = "Credit Card",
            category = VaultKeyCategory.FINANCIAL,
            sensitivity = VaultKeySensitivity.CRITICAL,
            lastAccessed = null,
            accessCount = 0
        )
    )

    val requiredKeys = setOf("ssn", "email")

    MaterialTheme {
        Surface {
            VaultKeyPanel(
                keys = sampleKeys,
                requiredKeys = requiredKeys,
                modifier = Modifier.width(280.dp)
            )
        }
    }
}
