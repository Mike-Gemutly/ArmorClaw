package com.armorclaw.app.screens.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Shield
import androidx.compose.material.icons.filled.VerifiedUser
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple

/**
 * Data Safety screen - Shows how user data is handled
 * Required for Google Play Data Safety disclosure transparency
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DataSafetyScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Data Safety") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Header card
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = BrandPurple.copy(alpha = 0.1f)
                ),
                shape = RoundedCornerShape(16.dp)
            ) {
                Row(
                    modifier = Modifier.padding(20.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Shield,
                        contentDescription = null,
                        tint = BrandPurple,
                        modifier = Modifier.size(48.dp)
                    )
                    Column {
                        Text(
                            text = "Your Data is Safe",
                            style = MaterialTheme.typography.titleLarge,
                            fontWeight = FontWeight.Bold,
                            color = BrandPurple
                        )
                        Text(
                            text = "We follow strict data protection practices",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                        )
                    }
                }
            }

            // Data Collection Section
            DataSafetySection(
                title = "Data We Collect",
                items = listOf(
                    DataSafetyItem(
                        title = "Account Information",
                        description = "Email, username, and profile name",
                        encrypted = true,
                        canDelete = true
                    ),
                    DataSafetyItem(
                        title = "Messages",
                        description = "End-to-end encrypted chat messages",
                        encrypted = true,
                        canDelete = true
                    ),
                    DataSafetyItem(
                        title = "Files & Media",
                        description = "Images, documents sent in chats",
                        encrypted = true,
                        canDelete = true
                    ),
                    DataSafetyItem(
                        title = "Device Information",
                        description = "Device type, OS version for debugging",
                        encrypted = true,
                        canDelete = true
                    )
                )
            )

            // Security Section
            DataSafetySection(
                title = "Security Practices",
                items = listOf(
                    DataSafetyItem(
                        title = "End-to-End Encryption",
                        description = "All messages encrypted with AES-256-GCM",
                        encrypted = true,
                        canDelete = false
                    ),
                    DataSafetyItem(
                        title = "Data at Rest Encryption",
                        description = "Local database encrypted with SQLCipher",
                        encrypted = true,
                        canDelete = false
                    ),
                    DataSafetyItem(
                        title = "Secure Transmission",
                        description = "All network traffic uses TLS 1.3",
                        encrypted = true,
                        canDelete = false
                    ),
                    DataSafetyItem(
                        title = "No Data Selling",
                        description = "We never sell your personal data",
                        encrypted = false,
                        canDelete = false
                    )
                )
            )

            // Sharing Section
            DataSafetySection(
                title = "Data Sharing",
                items = listOf(
                    DataSafetyItem(
                        title = "No Third-Party Sharing",
                        description = "Your data is not shared with advertisers",
                        encrypted = false,
                        canDelete = false
                    ),
                    DataSafetyItem(
                        title = "Matrix Federation",
                        description = "Messages shared only with recipients via federated servers",
                        encrypted = true,
                        canDelete = false
                    )
                )
            )

            // Your Rights Section
            DataSafetySection(
                title = "Your Rights",
                items = listOf(
                    DataSafetyItem(
                        title = "Access Your Data",
                        description = "Request a copy of all your data",
                        encrypted = false,
                        canDelete = true
                    ),
                    DataSafetyItem(
                        title = "Delete Your Data",
                        description = "Request complete data deletion",
                        encrypted = false,
                        canDelete = true
                    ),
                    DataSafetyItem(
                        title = "Export Your Data",
                        description = "Download your data in portable format",
                        encrypted = false,
                        canDelete = true
                    )
                )
            )

            // Contact info
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Column(
                    modifier = Modifier.padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Text(
                        text = "Questions about your data?",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Text(
                        text = "Contact us at privacy@armorclaw.app",
                        style = MaterialTheme.typography.bodyMedium,
                        color = BrandPurple
                    )
                }
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun DataSafetySection(
    title: String,
    items: List<DataSafetyItem>,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold
        )

        items.forEach { item ->
            DataSafetyItemCard(item = item)
        }
    }
}

@Composable
private fun DataSafetyItemCard(
    item: DataSafetyItem,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Icon
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .padding(8.dp)
            ) {
                if (item.encrypted) {
                    Icon(
                        imageVector = Icons.Default.Lock,
                        contentDescription = "Encrypted",
                        tint = BrandGreen,
                        modifier = Modifier.size(24.dp)
                    )
                } else {
                    Icon(
                        imageVector = Icons.Default.Check,
                        contentDescription = "Verified",
                        tint = BrandGreen,
                        modifier = Modifier.size(24.dp)
                    )
                }
            }

            // Content
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = item.title,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium
                )
                Text(
                    text = item.description,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                )
            }

            // Badges
            Column(horizontalAlignment = Alignment.End) {
                if (item.encrypted) {
                    Text(
                        text = "Encrypted",
                        style = MaterialTheme.typography.labelSmall,
                        color = BrandGreen,
                        fontWeight = FontWeight.Medium
                    )
                }
                if (item.canDelete) {
                    Text(
                        text = "Can delete",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.Medium
                    )
                }
            }
        }
    }
}

private data class DataSafetyItem(
    val title: String,
    val description: String,
    val encrypted: Boolean,
    val canDelete: Boolean
)

@Preview(showBackground = true)
@Composable
private fun DataSafetyScreenPreview() {
    ArmorClawTheme {
        DataSafetyScreen(onNavigateBack = {})
    }
}
