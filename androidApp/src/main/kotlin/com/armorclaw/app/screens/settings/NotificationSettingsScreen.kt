package com.armorclaw.app.screens.settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor

/**
 * Notification settings screen
 * 
 * This screen allows users to configure notification preferences
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NotificationSettingsScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    // Settings state
    var notificationsEnabled by remember { mutableStateOf(true) }
    var soundEnabled by remember { mutableStateOf(true) }
    var vibrationEnabled by remember { mutableStateOf(true) }
    var mentionsEnabled by remember { mutableStateOf(true) }
    var keywordsEnabled by remember { mutableStateOf(true) }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Notifications") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, "Back")
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
            // Main notifications
            item {
                SettingsCard(title = "Notifications") {
                    SettingToggle(
                        title = "Enable Notifications",
                        description = "Get notified about messages",
                        checked = notificationsEnabled,
                        onCheckedChange = { notificationsEnabled = it }
                    )
                    SettingToggle(
                        title = "Sound",
                        description = "Play sound on notification",
                        checked = soundEnabled,
                        onCheckedChange = { soundEnabled = it },
                        enabled = notificationsEnabled
                    )
                    SettingToggle(
                        title = "Vibration",
                        description = "Vibrate on notification",
                        checked = vibrationEnabled,
                        onCheckedChange = { vibrationEnabled = it },
                        enabled = notificationsEnabled
                    )
                }
            }
            
            // Advanced
            item {
                SettingsCard(title = "Advanced") {
                    SettingToggle(
                        title = "Mentions",
                        description = "Get notified when mentioned",
                        checked = mentionsEnabled,
                        onCheckedChange = { mentionsEnabled = it },
                        enabled = notificationsEnabled
                    )
                    SettingToggle(
                        title = "Keywords",
                        description = "Get notified for keywords",
                        checked = keywordsEnabled,
                        onCheckedChange = { keywordsEnabled = it },
                        enabled = notificationsEnabled
                    )
                }
            }
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun NotificationSettingsScreenPreview() {
    ArmorClawTheme {
        NotificationSettingsScreen(onNavigateBack = {})
    }
}
