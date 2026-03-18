package com.armorclaw.app.screens.settings

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.DesignTokens

/**
 * Screen for adding a new device to the user's account
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddDeviceScreen(
    onNavigateBack: () -> Unit,
    onDeviceAdded: () -> Unit,
    modifier: Modifier = Modifier
) {
    var deviceName by remember { mutableStateOf("") }
    var isAdding by remember { mutableStateOf(false) }
    var showQRScanner by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Add Device") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        }
    ) { paddingValues ->
        Column(
            modifier = modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(DesignTokens.Spacing.lg),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Header
            Icon(
                imageVector = Icons.Default.AddCircleOutline,
                contentDescription = null,
                modifier = Modifier.size(80.dp),
                tint = BrandPurple
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

            Text(
                text = "Add a New Device",
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            Text(
                text = "Add another device to your account for secure access",
                style = MaterialTheme.typography.bodyLarge,
                textAlign = TextAlign.Center,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

            // Device name input
            OutlinedTextField(
                value = deviceName,
                onValueChange = { deviceName = it },
                label = { Text("Device Name") },
                placeholder = { Text("e.g., Work Laptop, Phone") },
                leadingIcon = { Icon(Icons.Default.Devices, contentDescription = null) },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

            // Add methods
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Column(
                    modifier = Modifier.padding(DesignTokens.Spacing.md)
                ) {
                    Text(
                        text = "Add Methods",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

                    // QR Code option
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            Icons.Default.QrCode2,
                            contentDescription = null,
                            tint = BrandPurple
                        )
                        Spacer(modifier = Modifier.width(DesignTokens.Spacing.md))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = "Scan QR Code",
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.Medium
                            )
                            Text(
                                text = "Fastest method - scan from existing device",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                            )
                        }
                        IconButton(onClick = { showQRScanner = true }) {
                            Icon(Icons.Default.ChevronRight, contentDescription = "Scan")
                        }
                    }

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

                    Divider()

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

                    // Recovery key option
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            Icons.Default.Key,
                            contentDescription = null,
                            tint = BrandPurple
                        )
                        Spacer(modifier = Modifier.width(DesignTokens.Spacing.md))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = "Use Recovery Key",
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.Medium
                            )
                            Text(
                                text = "Enter your recovery key manually",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                            )
                        }
                        IconButton(onClick = { /* Show recovery key input */ }) {
                            Icon(Icons.Default.ChevronRight, contentDescription = "Enter key")
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.weight(1f))

            // Add button
            Button(
                onClick = {
                    if (deviceName.isNotBlank()) {
                        isAdding = true
                        // TODO: Actually add device via ViewModel
                        // For now, just simulate and navigate back
                        onDeviceAdded()
                    }
                },
                modifier = Modifier.fillMaxWidth(),
                enabled = deviceName.isNotBlank() && !isAdding
            ) {
                if (isAdding) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(20.dp),
                        color = MaterialTheme.colorScheme.onPrimary
                    )
                } else {
                    Icon(Icons.Default.Add, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Add Device")
                }
            }
        }
    }

    // QR Scanner Dialog
    if (showQRScanner) {
        AlertDialog(
            onDismissRequest = { showQRScanner = false },
            title = { Text("Scan QR Code") },
            text = {
                Column(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    // Placeholder for QR scanner
                    Surface(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(200.dp),
                        color = MaterialTheme.colorScheme.surfaceVariant
                    ) {
                        Box(
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                Icons.Default.QrCodeScanner,
                                contentDescription = "QR Scanner",
                                modifier = Modifier.size(80.dp),
                                tint = MaterialTheme.colorScheme.outline
                            )
                        }
                    }
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = "Point your camera at the QR code displayed on your other device",
                        style = MaterialTheme.typography.bodyMedium,
                        textAlign = TextAlign.Center
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = { showQRScanner = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}
