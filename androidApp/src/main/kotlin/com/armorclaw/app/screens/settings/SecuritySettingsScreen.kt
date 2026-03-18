package com.armorclaw.app.screens.settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Devices
import androidx.compose.material.icons.filled.Key
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.KeyboardArrowRight
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.clickable
import androidx.compose.foundation.shape.RoundedCornerShape

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.BrandPurple

/**
 * Security settings screen
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SecuritySettingsScreen(
    onNavigateBack: () -> Unit,
    onNavigateToDevices: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    var biometricAuth by remember { mutableStateOf(true) }
    var twoFactorAuth by remember { mutableStateOf(false) }
    var autoDeleteMessages by remember { mutableStateOf(false) }
    var autoDeleteDays by remember { mutableStateOf(30) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Security") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        },
        modifier = modifier
    ) { paddingValues ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Device management - NEW
            item {
                SettingsCard(title = "Device Management") {
                    SecurityMenuItem(
                        icon = Icons.Default.Devices,
                        title = "Manage Devices",
                        description = "View and manage your logged in devices",
                        onClick = onNavigateToDevices
                    )
                }
            }

            // Biometric auth
            item {
                SettingsCard(title = "Biometric Authentication") {
                    SettingToggle(
                        title = "Enable Biometric",
                        description = "Use fingerprint or face to unlock",
                        checked = biometricAuth,
                        onCheckedChange = { biometricAuth = it }
                    )
                }
            }

            // Two-factor auth
            item {
                SettingsCard(title = "Two-Factor Authentication") {
                    SettingToggle(
                        title = "Enable 2FA",
                        description = "Add an extra layer of security",
                        checked = twoFactorAuth,
                        onCheckedChange = { twoFactorAuth = it }
                    )
                }
            }

            // Encryption info
            item {
                SettingsCard(title = "Encryption") {
                    SecurityMenuItem(
                        icon = Icons.Default.Lock,
                        title = "Encryption Keys",
                        description = "Manage your encryption keys",
                        onClick = { /* TODO */ }
                    )
                    SecurityMenuItem(
                        icon = Icons.Default.Key,
                        title = "Key Backup",
                        description = "Backup your encryption keys",
                        onClick = { /* TODO */ }
                    )
                }
            }

            // Auto-delete
            item {
                SettingsCard(title = "Auto-Delete Messages") {
                    SettingToggle(
                        title = "Auto-Delete",
                        description = "Automatically delete old messages",
                        checked = autoDeleteMessages,
                        onCheckedChange = { autoDeleteMessages = it }
                    )
                    SettingSlider(
                        title = "Delete After",
                        value = autoDeleteDays.toFloat(),
                        onValueChange = { autoDeleteDays = it.toInt() },
                        range = 7f..365f,
                        steps = 12,
                        enabled = autoDeleteMessages,
                        valueText = "$autoDeleteDays days"
                    )
                }
            }
        }
    }
}

@Composable
private fun SecurityMenuItem(
    icon: ImageVector,
    title: String,
    description: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = BrandPurple,
            modifier = Modifier.size(24.dp)
        )
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium
            )
            Text(
                text = description,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
            )
        }
        Icon(
            imageVector = Icons.Filled.KeyboardArrowRight,
            contentDescription = null
        )
    }
}

@Preview(showBackground = true)
@Composable
private fun SecuritySettingsScreenPreview() {
    ArmorClawTheme {
        SecuritySettingsScreen(onNavigateBack = {})
    }
}
