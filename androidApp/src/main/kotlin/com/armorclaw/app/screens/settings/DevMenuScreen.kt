package com.armorclaw.app.screens.settings

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.armorclaw.shared.platform.bridge.DevSettings
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandRed
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.DesignTokens

/**
 * Hidden Developer Menu for QA testing
 *
 * Access: Long-press version number in About screen 5 times
 *
 * Allows overriding:
 * - Bridge base URL
 * - Protocol (http/https)
 * - RPC path (/rpc vs /api)
 * - Port number
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DevMenuScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val devEnabled by DevSettings.enabled.collectAsStateWithLifecycle()
    val overrides by DevSettings.overrides.collectAsStateWithLifecycle()
    val scrollState = rememberScrollState()

    // Local state for inputs
    var bridgeUrlInput by remember { mutableStateOf(overrides.bridgeUrlOverride ?: "") }
    var protocolInput by remember { mutableStateOf(overrides.protocolOverride ?: "") }
    var rpcPathInput by remember { mutableStateOf(overrides.rpcPathOverride ?: "") }
    var portInput by remember { mutableStateOf(overrides.portOverride?.toString() ?: "") }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Developer Menu") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, "Back")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface
                )
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(scrollState)
                .padding(DesignTokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            // Warning banner
            Card(
                colors = CardDefaults.cardColors(
                    containerColor = BrandRed.copy(alpha = 0.1f)
                )
            ) {
                Row(
                    modifier = Modifier.padding(DesignTokens.Spacing.md),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        Icons.Default.Warning,
                        contentDescription = null,
                        tint = BrandRed
                    )
                    Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                    Text(
                        text = "These settings are for QA testing only. Incorrect values may break connectivity.",
                        style = MaterialTheme.typography.bodySmall,
                        color = BrandRed
                    )
                }
            }

            // Dev mode toggle
            Card(
                modifier = Modifier.fillMaxWidth()
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(DesignTokens.Spacing.md),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column {
                        Text(
                            text = "Enable Dev Overrides",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold
                        )
                        Text(
                            text = "When enabled, overrides will be applied to bridge connections",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                        )
                    }
                    Switch(
                        checked = devEnabled,
                        onCheckedChange = { DevSettings.setEnabled(it) }
                    )
                }
            }

            // Only show settings when enabled
            AnimatedVisibility(visible = devEnabled) {
                Column(verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)) {
                    // Status card
                    Card(
                        colors = CardDefaults.cardColors(
                            containerColor = if (overrides.hasAnyOverride) {
                                BrandGreen.copy(alpha = 0.1f)
                            } else {
                                MaterialTheme.colorScheme.surfaceVariant
                            }
                        ),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Row(
                            modifier = Modifier.padding(DesignTokens.Spacing.md),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                if (overrides.hasAnyOverride) Icons.Default.Check else Icons.Default.Info,
                                contentDescription = null,
                                tint = if (overrides.hasAnyOverride) BrandGreen else MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                            Text(
                                text = if (overrides.hasAnyOverride) "Overrides active" else "No overrides set",
                                style = MaterialTheme.typography.bodyMedium
                            )
                        }
                    }

                    // Bridge URL Override
                    OutlinedTextField(
                        value = bridgeUrlInput,
                        onValueChange = { 
                            bridgeUrlInput = it
                            DevSettings.setBridgeUrlOverride(it.ifBlank { null })
                        },
                        label = { Text("Bridge URL Override") },
                        placeholder = { Text("http://192.168.1.100:8080") },
                        leadingIcon = { Icon(Icons.Default.Router, contentDescription = null) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                        supportingText = { Text("Full URL to override derived bridge URL") }
                    )

                    // Protocol Override
                    OutlinedTextField(
                        value = protocolInput,
                        onValueChange = { 
                            protocolInput = it
                            DevSettings.setProtocolOverride(it.ifBlank { null })
                        },
                        label = { Text("Protocol Override") },
                        placeholder = { Text("http or https") },
                        leadingIcon = { Icon(Icons.Default.Http, contentDescription = null) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                        supportingText = { Text("Force http or https protocol") }
                    )

                    // Port Override
                    OutlinedTextField(
                        value = portInput,
                        onValueChange = { 
                            portInput = it
                            DevSettings.setPortOverride(it.toIntOrNull())
                        },
                        label = { Text("Port Override") },
                        placeholder = { Text("8080") },
                        leadingIcon = { Icon(Icons.Default.SettingsEthernet, contentDescription = null) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        supportingText = { Text("Override bridge port number") }
                    )

                    // RPC Path Override
                    OutlinedTextField(
                        value = rpcPathInput,
                        onValueChange = { 
                            rpcPathInput = it
                            DevSettings.setRpcPathOverride(it.ifBlank { null })
                        },
                        label = { Text("RPC Path Override") },
                        placeholder = { Text("/rpc or /api") },
                        leadingIcon = { Icon(Icons.Default.Api, contentDescription = null) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                        supportingText = { Text("Override RPC endpoint path") }
                    )

                    // Clear button
                    Button(
                        onClick = {
                            bridgeUrlInput = ""
                            protocolInput = ""
                            rpcPathInput = ""
                            portInput = ""
                            DevSettings.clearOverrides()
                        },
                        modifier = Modifier.fillMaxWidth(),
                        colors = ButtonDefaults.buttonColors(
                            containerColor = BrandRed
                        )
                    ) {
                        Icon(Icons.Default.Clear, contentDescription = null)
                        Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                        Text("Clear All Overrides")
                    }
                }
            }

            Spacer(modifier = Modifier.weight(1f))

            // Quick presets
            if (devEnabled) {
                Text(
                    text = "Quick Presets",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(top = DesignTokens.Spacing.md)
                )

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                ) {
                    OutlinedButton(
                        onClick = {
                            bridgeUrlInput = "http://10.0.2.2:8080"
                            DevSettings.setBridgeUrlOverride("http://10.0.2.2:8080")
                        },
                        modifier = Modifier.weight(1f)
                    ) {
                        Text("Android Emulator", style = MaterialTheme.typography.labelSmall)
                    }
                    OutlinedButton(
                        onClick = {
                            bridgeUrlInput = "http://192.168.1.100:8080"
                            DevSettings.setBridgeUrlOverride("http://192.168.1.100:8080")
                        },
                        modifier = Modifier.weight(1f)
                    ) {
                        Text("Local Dev", style = MaterialTheme.typography.labelSmall)
                    }
                }
            }
        }
    }
}
