package com.armorclaw.app.screens.settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTheme

/**
 * Appearance settings screen
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AppearanceSettingsScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    var theme by remember { mutableStateOf(ThemeMode.AUTO) }
    var fontSize by remember { mutableStateOf(FontSize.MEDIUM) }
    var largeText by remember { mutableStateOf(false) }
    var highContrast by remember { mutableStateOf(false) }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Appearance") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            Icons.Default.ArrowBack,
                            contentDescription = "Back"
                        )
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
            // Theme
            item {
                SettingsCard(title = "Theme") {
                    RadioGroup(
                        title = "Choose Theme",
                        options = ThemeMode.values().toList(),
                        selected = theme,
                        onSelected = { theme = it }
                    )
                }
            }
            
            // Font size
            item {
                SettingsCard(title = "Font Size") {
                    RadioGroup(
                        title = "Font Size",
                        options = FontSize.values().toList(),
                        selected = fontSize,
                        onSelected = { fontSize = it }
                    )
                }
            }
            
            // Accessibility
            item {
                SettingsCard(title = "Accessibility") {
                    SettingToggle(
                        title = "Large Text",
                        description = "Increase text size for better readability",
                        checked = largeText,
                        onCheckedChange = { largeText = it }
                    )
                    SettingToggle(
                        title = "High Contrast",
                        description = "Increase contrast for better visibility",
                        checked = highContrast,
                        onCheckedChange = { highContrast = it }
                    )
                }
            }
        }
    }
}

enum class ThemeMode {
    LIGHT, DARK, AUTO
}

enum class FontSize {
    SMALL, MEDIUM, LARGE
}

@Preview(showBackground = true)
@Composable
private fun AppearanceSettingsScreenPreview() {
    ArmorClawTheme {
        AppearanceSettingsScreen(onNavigateBack = {})
    }
}
