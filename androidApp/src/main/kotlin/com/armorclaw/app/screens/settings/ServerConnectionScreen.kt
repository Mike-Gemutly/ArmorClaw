package com.armorclaw.app.screens.settings

import androidx.compose.animation.*
import androidx.compose.foundation.BorderStroke
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.armorclaw.app.viewmodels.*
import com.armorclaw.shared.platform.bridge.*
import com.armorclaw.shared.ui.theme.*
import org.koin.androidx.compose.koinViewModel
import java.text.SimpleDateFormat
import java.util.*

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ServerConnectionScreen(
    onBack: () -> Unit,
    viewModel: ServerConnectionViewModel = koinViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Server Connection") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            ConnectionStatusCard(
                status = uiState.connectionStatus,
                homeserver = uiState.currentHomeserver,
                bridgeUrl = uiState.currentBridgeUrl,
                serverVersion = uiState.serverVersion,
                lastConnected = uiState.lastConnected,
                isDemoMode = uiState.isDemoMode,
                onRefresh = { viewModel.checkConnectionStatus() }
            )

            AnimatedVisibility(visible = uiState.isDiscovering) {
                DiscoveryProgressCard(progress = uiState.discoveryProgress)
            }

            AnimatedVisibility(visible = uiState.securityWarnings.isNotEmpty()) {
                SecurityWarningsCard(warnings = uiState.securityWarnings, onDismiss = { viewModel.dismissWarning(it) })
            }

            AnimatedVisibility(visible = uiState.errorMessage != null && !uiState.isDiscovering) {
                ErrorCard(
                    message = uiState.errorMessage ?: "",
                    fallbackOptions = uiState.fallbackOptions,
                    onRetry = { viewModel.startDiscovery() },
                    onDemoServer = { viewModel.useDemoServer() }
                )
            }

            if (!uiState.isDiscovering) {
                ActionButtonsCard(
                    connectionStatus = uiState.connectionStatus,
                    isDemoMode = uiState.isDemoMode,
                    onDiscover = { viewModel.startDiscovery() },
                    onDemoMode = { viewModel.useDemoServer() },
                    onManualConfig = { h, b -> viewModel.updateServerConfig(h, b) }
                )
            }

            ManualConfigurationCard(
                currentHomeserver = uiState.currentHomeserver,
                currentBridgeUrl = uiState.currentBridgeUrl,
                isDiscovering = uiState.isDiscovering,
                onSave = { h, b -> viewModel.updateServerConfig(h, b) }
            )

            DeepLinkActivationCard(
                isDiscovering = uiState.isDiscovering,
                onDeepLinkReceived = { viewModel.processDeepLink(it) }
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
        }
    }
}

@Composable
private fun ConnectionStatusCard(
    status: ConnectionStatus,
    homeserver: String?,
    bridgeUrl: String?,
    serverVersion: String?,
    lastConnected: Long?,
    isDemoMode: Boolean,
    onRefresh: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = when (status) {
                ConnectionStatus.CONNECTED -> BrandGreen.copy(alpha = 0.1f)
                ConnectionStatus.DISCONNECTED, ConnectionStatus.ERROR -> BrandRed.copy(alpha = 0.1f)
                ConnectionStatus.CONNECTING, ConnectionStatus.DISCOVERING -> BrandYellow.copy(alpha = 0.1f)
                ConnectionStatus.UNKNOWN -> MaterialTheme.colorScheme.surfaceVariant
            }
        ),
        border = BorderStroke(1.dp, when (status) {
            ConnectionStatus.CONNECTED -> BrandGreen
            ConnectionStatus.DISCONNECTED, ConnectionStatus.ERROR -> BrandRed
            ConnectionStatus.CONNECTING, ConnectionStatus.DISCOVERING -> BrandYellow
            ConnectionStatus.UNKNOWN -> MaterialTheme.colorScheme.outline
        })
    ) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(DesignTokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    StatusIcon(status)
                    Text(
                        text = when (status) {
                            ConnectionStatus.CONNECTED -> "Connected"
                            ConnectionStatus.DISCONNECTED -> "Disconnected"
                            ConnectionStatus.ERROR -> "Connection Error"
                            ConnectionStatus.CONNECTING -> "Connecting..."
                            ConnectionStatus.DISCOVERING -> "Discovering..."
                            ConnectionStatus.UNKNOWN -> "Not Configured"
                        },
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                }
                IconButton(onClick = onRefresh) {
                    Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                }
            }
            Divider()
            if (homeserver != null) DetailRow(label = "Homeserver", value = homeserver.substringAfter("://"))
            if (bridgeUrl != null) DetailRow(label = "Bridge URL", value = bridgeUrl.substringAfter("://"))
            if (serverVersion != null) DetailRow(label = "Server Version", value = serverVersion)
            if (lastConnected != null) DetailRow(label = "Last Connected", value = formatTimestamp(lastConnected))
            if (isDemoMode) {
                Surface(
                    color = BrandPurple.copy(alpha = 0.2f),
                    shape = MaterialTheme.shapes.small
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            Icons.Default.Science,
                            contentDescription = null,
                            tint = BrandPurple,
                            modifier = Modifier.size(16.dp)
                        )
                        Text(
                            text = "Demo Mode",
                            style = MaterialTheme.typography.labelMedium,
                            color = BrandPurple,
                            fontWeight = FontWeight.SemiBold
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun StatusIcon(status: ConnectionStatus) {
    when (status) {
        ConnectionStatus.CONNECTED -> Icon(
            imageVector = Icons.Default.CheckCircle,
            contentDescription = null,
            tint = BrandGreen,
            modifier = Modifier.size(28.dp)
        )
        ConnectionStatus.DISCONNECTED -> Icon(
            imageVector = Icons.Default.Cancel,
            contentDescription = null,
            tint = BrandRed,
            modifier = Modifier.size(28.dp)
        )
        ConnectionStatus.ERROR -> Icon(
            imageVector = Icons.Default.Error,
            contentDescription = null,
            tint = BrandRed,
            modifier = Modifier.size(28.dp)
        )
        ConnectionStatus.CONNECTING, ConnectionStatus.DISCOVERING -> CircularProgressIndicator(
            modifier = Modifier.size(24.dp),
            strokeWidth = 2.dp,
            color = BrandYellow
        )
        ConnectionStatus.UNKNOWN -> Icon(
            imageVector = Icons.Default.HelpOutline,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.size(28.dp)
        )
    }
}

@Composable
private fun DetailRow(label: String, value: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(
            label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
        )
        Text(
            value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium
        )
    }
}

@Composable
private fun DiscoveryProgressCard(progress: DiscoveryProgress) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = BrandPurple.copy(alpha = 0.1f))
    ) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(DesignTokens.Spacing.lg),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            CircularProgressIndicator(Modifier.size(48.dp), color = BrandPurple)
            when (progress) {
                is DiscoveryProgress.Idle -> Text("Ready to discover", style = MaterialTheme.typography.bodyLarge)
                is DiscoveryProgress.Discovering -> Text(progress.step, style = MaterialTheme.typography.bodyLarge, textAlign = TextAlign.Center)
                is DiscoveryProgress.FoundServer -> {
                    Icon(imageVector = Icons.Default.CheckCircle, contentDescription = null, tint = BrandGreen, modifier = Modifier.size(48.dp))
                    Text("Server Found!", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold, color = BrandGreen)
                    Text(progress.serverInfo.homeserver, style = MaterialTheme.typography.bodyMedium)
                }
                is DiscoveryProgress.Error -> {
                    Icon(imageVector = Icons.Default.Error, contentDescription = null, tint = BrandRed, modifier = Modifier.size(48.dp))
                    Text("Discovery Failed", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold, color = BrandRed)
                    Text(progress.message, style = MaterialTheme.typography.bodyMedium, color = BrandRed)
                }
            }
        }
    }
}

@Composable
private fun SecurityWarningsCard(warnings: List<SecurityWarning>, onDismiss: (String) -> Unit) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = BrandYellow.copy(alpha = 0.1f)),
        border = BorderStroke(1.dp, BrandYellow)
    ) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(DesignTokens.Spacing.md),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
        ) {
            warnings.forEach { warning ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(warning.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                        Text(warning.message, style = MaterialTheme.typography.bodySmall)
                    }
                    if (warning.canDismiss) {
                        IconButton(onClick = { onDismiss(warning.id) }) {
                            Icon(Icons.Default.Close, contentDescription = "Dismiss")
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ErrorCard(
    message: String,
    fallbackOptions: List<FallbackOption>,
    onRetry: () -> Unit,
    onDemoServer: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = BrandRed.copy(alpha = 0.1f)),
        border = BorderStroke(1.dp, BrandRed)
    ) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(DesignTokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(Icons.Default.Error, contentDescription = null, tint = BrandRed)
                Text("Connection Failed", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold, color = BrandRed)
            }
            Text(message, style = MaterialTheme.typography.bodyMedium)
            Divider()
            Button(onClick = onRetry, modifier = Modifier.fillMaxWidth()) {
                Icon(Icons.Default.Refresh, contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text("Retry Discovery")
            }
            if (fallbackOptions.isNotEmpty()) {
                Text("Alternative Options", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
                fallbackOptions.forEach { option ->
                    when (option) {
                        FallbackOption.USE_DEMO -> {
                            OutlinedButton(onClick = onDemoServer, modifier = Modifier.fillMaxWidth()) {
                                Icon(Icons.Default.Science, contentDescription = null)
                                Spacer(Modifier.width(8.dp))
                                Text("Try Demo Server")
                            }
                        }
                        FallbackOption.RETRY -> {
                            // Already handled by main retry button
                        }
                        FallbackOption.CONTACT_SUPPORT -> {
                            OutlinedButton(onClick = { /* TODO: Open support */ }, modifier = Modifier.fillMaxWidth()) {
                                Icon(Icons.Default.Support, contentDescription = null)
                                Spacer(Modifier.width(8.dp))
                                Text("Contact Support")
                            }
                        }
                        FallbackOption.CHECK_INTERNET -> {
                            Text("Please check your internet connection", style = MaterialTheme.typography.bodySmall)
                        }
                        FallbackOption.TRY_LATER -> {
                            Text("Server may be temporarily unavailable. Try again later.", style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ActionButtonsCard(
    connectionStatus: ConnectionStatus,
    isDemoMode: Boolean,
    onDiscover: () -> Unit,
    onDemoMode: () -> Unit,
    onManualConfig: (String, String?) -> Unit
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(DesignTokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            Text("Actions", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            Button(
                onClick = onDiscover,
                modifier = Modifier.fillMaxWidth(),
                colors = ButtonDefaults.buttonColors(containerColor = BrandPurple)
            ) {
                Icon(Icons.Default.Search, contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text("Discover Server")
            }
            if (!isDemoMode) {
                OutlinedButton(onClick = onDemoMode, modifier = Modifier.fillMaxWidth()) {
                    Icon(Icons.Default.Science, contentDescription = null)
                    Spacer(Modifier.width(8.dp))
                    Text("Use Demo Server")
                }
            }
        }
    }
}

@Composable
private fun ManualConfigurationCard(
    currentHomeserver: String?,
    currentBridgeUrl: String?,
    isDiscovering: Boolean,
    onSave: (String, String?) -> Unit
) {
    var showAdvanced by remember { mutableStateOf(false) }
    var homeserver by remember { mutableStateOf(currentHomeserver ?: "") }
    var bridgeUrl by remember { mutableStateOf(currentBridgeUrl ?: "") }

    LaunchedEffect(currentHomeserver, currentBridgeUrl) {
        homeserver = currentHomeserver ?: ""
        bridgeUrl = currentBridgeUrl ?: ""
    }

    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(DesignTokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text("Manual Configuration", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                TextButton(onClick = { showAdvanced = !showAdvanced }) {
                    Icon(if (showAdvanced) Icons.Default.ExpandLess else Icons.Default.ExpandMore, contentDescription = null)
                    Spacer(Modifier.width(4.dp))
                    Text(if (showAdvanced) "Hide" else "Show")
                }
            }
            AnimatedVisibility(visible = showAdvanced) {
                Column(verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)) {
                    OutlinedTextField(
                        value = homeserver,
                        onValueChange = { homeserver = it },
                        label = { Text("Homeserver URL") },
                        placeholder = { Text("https://matrix.armorclaw.app") },
                        leadingIcon = { Icon(Icons.Default.Dns, contentDescription = null) },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !isDiscovering
                    )
                    OutlinedTextField(
                        value = bridgeUrl,
                        onValueChange = { bridgeUrl = it },
                        label = { Text("Bridge URL (optional)") },
                        placeholder = { Text("https://bridge.armorclaw.app") },
                        leadingIcon = { Icon(Icons.Default.Router, contentDescription = null) },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !isDiscovering
                    )
                    Button(
                        onClick = { onSave(homeserver, bridgeUrl.ifBlank { null }) },
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !isDiscovering && homeserver.isNotBlank()
                    ) {
                        Icon(Icons.Default.Save, contentDescription = null)
                        Spacer(Modifier.width(8.dp))
                        Text("Save Configuration")
                    }
                }
            }
        }
    }
}

@Composable
private fun DeepLinkActivationCard(
    isDiscovering: Boolean,
    onDeepLinkReceived: (String) -> Unit
) {
    var deepLinkInput by remember { mutableStateOf("") }

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
        )
    ) {
        Column(
            modifier = Modifier.fillMaxWidth().padding(DesignTokens.Spacing.lg),
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(Icons.Default.QrCodeScanner, contentDescription = null, tint = BrandPurple)
                Text("QR Code / Deep Link Activation", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            }
            Text(
                "When ArmorClaw becomes available, scan the QR code or paste the activation link to automatically configure your connection.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
            )
            OutlinedTextField(
                value = deepLinkInput,
                onValueChange = { deepLinkInput = it },
                label = { Text("Paste activation link") },
                placeholder = { Text("armorclaw://config?d=... or https://armorclaw.app/...") },
                leadingIcon = { Icon(Icons.Default.Link, contentDescription = null) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
                enabled = !isDiscovering
            )
            Button(
                onClick = {
                    if (deepLinkInput.isNotBlank()) {
                        onDeepLinkReceived(deepLinkInput)
                        deepLinkInput = ""
                    }
                },
                modifier = Modifier.fillMaxWidth(),
                enabled = !isDiscovering && deepLinkInput.isNotBlank(),
                colors = ButtonDefaults.buttonColors(containerColor = BrandPurple)
            ) {
                Icon(Icons.Default.Bolt, contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text("Activate")
            }
        }
    }
}

private fun formatTimestamp(timestamp: Long): String {
    val sdf = SimpleDateFormat("MMM dd, yyyy HH:mm", Locale.getDefault())
    return sdf.format(Date(timestamp))
}
