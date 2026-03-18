package com.armorclaw.app.screens.settings
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.Devices
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.armorclaw.app.screens.settings.components.*
import com.armorclaw.app.viewmodels.DeviceListViewModel
import com.armorclaw.shared.domain.model.DeviceInfo
import com.armorclaw.shared.domain.model.TrustLevel
import com.armorclaw.shared.ui.theme.*
import kotlinx.coroutines.launch
import kotlinx.coroutines.delay

/**
 * Screen displaying list of user's devices with verification status
 *
 * Features:
 * - Shows all trusted devices with verification badges
 * - Highlights unverified devices requiring action
 * - Allows device verification and removal
 * - Shows device details on tap
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeviceListScreen(
    viewModel: DeviceListViewModel,
    onNavigateBack: () -> Unit = {},
    onAddDeviceClick: () -> Unit = {},
    onVerifyDeviceClick: ((deviceId: String) -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val trustedDevices by viewModel.trustedDevices.collectAsStateWithLifecycle()
    val unverifiedDevices by viewModel.unverifiedDevices.collectAsStateWithLifecycle()

    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    // Load devices on first composition
    LaunchedEffect(Unit) {
        viewModel.loadDevices()
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = "My Devices",
                        style = MaterialTheme.typography.titleMedium
                    )
                },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Default.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    // Unverified count badge
                    if (uiState.unverifiedCount > 0) {
                        Box(
                            modifier = Modifier
                                .padding(end = 8.dp)
                                .size(32.dp)
                                .clip(CircleShape)
                                .background(BrandRed),
                            contentAlignment = Alignment.Center
                        ) {
                            Text(
                                text = uiState.unverifiedCount.toString(),
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.Bold,
                                color = OnPrimary
                            )
                        }
                    }

                    IconButton(onClick = { viewModel.refresh() }) {
                        Icon(
                            imageVector = Icons.Default.Refresh,
                            contentDescription = "Refresh"
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface
                )
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = onAddDeviceClick,
                containerColor = BrandPurple,
                contentColor = OnPrimary
            ) {
                Icon(
                    imageVector = Icons.Default.Add,
                    contentDescription = "Add device"
                )
            }
        },
        modifier = modifier
    ) { paddingValues ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            when {
                uiState.isLoading -> {
                    // Loading state
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center
                    ) {
                        CircularProgressIndicator(
                            color = BrandPurple
                        )
                    }
                }

                uiState.devices.isEmpty() && !uiState.isLoading -> {
                    // Empty state
                    DeviceListEmptyState(
                        onRefresh = { viewModel.refresh() },
                        modifier = Modifier.fillMaxSize()
                    )
                }

                else -> {
                    // Device list
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(vertical = 16.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        // Unverified devices section
                        if (unverifiedDevices.isNotEmpty()) {
                            item {
                                DeviceSectionHeader(
                                    title = "Requires Verification",
                                    count = unverifiedDevices.size
                                )
                            }

                            items(
                                items = unverifiedDevices,
                                key = { it.deviceId }
                            ) { device ->
                                var showDetailsDialog by remember { mutableStateOf(false) }

                                DeviceListItem(
                                    device = device,
                                    onClick = { showDetailsDialog = true },
                                    onVerifyClick = {
                                        scope.launch {
                                            viewModel.verifyDevice(device.deviceId)
                                            snackbarHostState.showSnackbar("Device verified")
                                        }
                                    }
                                )

                                if (showDetailsDialog) {
                                    DeviceDetailsDialog(
                                        device = device,
                                        onDismiss = { showDetailsDialog = false },
                                        onVerify = {
                                            viewModel.verifyDevice(device.deviceId)
                                            showDetailsDialog = false
                                            scope.launch {
                                                snackbarHostState.showSnackbar("Device verified")
                                            }
                                        },
                                        onRemove = {
                                            viewModel.removeDevice(device.deviceId)
                                            showDetailsDialog = false
                                            scope.launch {
                                                snackbarHostState.showSnackbar("Device removed")
                                            }
                                        }
                                    )
                                }
                            }

                            item {
                                Spacer(modifier = Modifier.height(8.dp))
                            }
                        }

                        // Trusted devices section
                        if (trustedDevices.isNotEmpty()) {
                            item {
                                DeviceSectionHeader(
                                    title = "Trusted Devices",
                                    count = trustedDevices.size
                                )
                            }

                            items(
                                items = trustedDevices,
                                key = { it.deviceId }
                            ) { device ->
                                var showDetailsDialog by remember { mutableStateOf(false) }

                                DeviceListItem(
                                    device = device,
                                    onClick = { showDetailsDialog = true }
                                )

                                if (showDetailsDialog) {
                                    DeviceDetailsDialog(
                                        device = device,
                                        onDismiss = { showDetailsDialog = false },
                                        onVerify = {
                                            showDetailsDialog = false
                                            // Navigate to verification screen if callback provided
                                            // Otherwise use viewModel
                                            if (onVerifyDeviceClick != null) {
                                                onVerifyDeviceClick.invoke(device.deviceId)
                                            } else {
                                                viewModel.verifyDevice(device.deviceId)
                                            }
                                        },
                                        onRemove = {
                                            viewModel.removeDevice(device.deviceId)
                                            showDetailsDialog = false
                                            scope.launch {
                                                snackbarHostState.showSnackbar("Device removed")
                                            }
                                        }
                                    )
                                }
                            }
                        }
                    }
                }
            }

            // Error snackbar
            uiState.error?.let { error ->
                LaunchedEffect(error) {
                    snackbarHostState.showSnackbar(error)
                }
            }
        }
    }
}

/**
 * Device details dialog
 */
@Composable
private fun DeviceDetailsDialog(
    device: DeviceInfo,
    onDismiss: () -> Unit,
    onVerify: () -> Unit,
    onRemove: () -> Unit,
    modifier: Modifier = Modifier
) {
    var showRemoveConfirmation by remember { mutableStateOf(false) }

    AlertDialog(
        onDismissRequest = onDismiss,
        modifier = modifier,
        title = {
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(
                            when {
                                device.isCurrentDevice -> BrandPurple.copy(alpha = 0.15f)
                                device.trustLevel.isTrusted() -> BrandGreen.copy(alpha = 0.15f)
                                else -> OnBackground.copy(alpha = 0.1f)
                            }
                        ),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = if (device.isCurrentDevice)
                            Icons.Default.PhoneAndroid
                        else
                            Icons.Outlined.Devices,
                        contentDescription = null,
                        tint = when {
                            device.isCurrentDevice -> BrandPurple
                            device.trustLevel.isTrusted() -> BrandGreen
                            else -> OnBackground.copy(alpha = 0.6f)
                        }
                    )
                }

                Text(
                    text = device.displayName ?: "Unknown Device",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Medium
                )
            }
        },
        text = {
            Column(
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Trust status
                if (!device.isCurrentDevice) {
                    TrustStatusCard(
                        trustLevel = device.trustLevel,
                        deviceName = device.displayName,
                        onActionClick = {
                            if (device.trustLevel == TrustLevel.UNVERIFIED) {
                                onVerify()
                            }
                        }
                    )
                }

                // Device details
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = SurfaceVariant.copy(alpha = 0.5f)
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(12.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        DetailRow(
                            label = "Device ID",
                            value = device.deviceId.take(16) + "..."
                        )

                        device.lastSeenIp?.let { ip ->
                            // Mask IP for privacy
                            val maskedIp = ip.replaceAfterLast(".", "***")
                            DetailRow(
                                label = "Last IP",
                                value = maskedIp
                            )
                        }

                        device.lastSeenTimestamp?.let { timestamp ->
                            DetailRow(
                                label = "Last seen",
                                value = formatLastSeen(timestamp)
                            )
                        }

                        if (device.isCurrentDevice) {
                            Surface(
                                shape = RoundedCornerShape(4.dp),
                                color = BrandPurple.copy(alpha = 0.15f)
                            ) {
                                Text(
                                    text = "This is your current device",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = BrandPurple,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                )
                            }
                        }
                    }
                }

                // Warning for unverified devices
                if (device.trustLevel == TrustLevel.UNVERIFIED && !device.isCurrentDevice) {
                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = StatusWarning.copy(alpha = 0.1f)
                    ) {
                        Row(
                            modifier = Modifier.padding(12.dp),
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Default.Warning,
                                contentDescription = null,
                                tint = StatusWarning,
                                modifier = Modifier.size(20.dp)
                            )
                            Text(
                                text = "This device hasn't been verified. Messages sent to this device may not be secure.",
                                style = MaterialTheme.typography.bodyMedium,
                                color = StatusWarning
                            )
                        }
                    }
                }
            }
        },
        confirmButton = {
            if (showRemoveConfirmation) {
                TextButton(onClick = { showRemoveConfirmation = false }) {
                    Text("Cancel")
                }
                Button(
                    onClick = onRemove,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = BrandRed
                    )
                ) {
                    Text("Remove")
                }
            } else {
                if (device.trustLevel == TrustLevel.UNVERIFIED && !device.isCurrentDevice) {
                    Button(
                        onClick = onVerify,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = BrandGreen
                        )
                    ) {
                        Icon(
                            imageVector = Icons.Default.VerifiedUser,
                            contentDescription = null,
                            modifier = Modifier.size(18.dp)
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Verify")
                    }
                }

                if (!device.isCurrentDevice) {
                    IconButton(onClick = { showRemoveConfirmation = true }) {
                        Icon(
                            imageVector = Icons.Default.Delete,
                            contentDescription = "Remove device",
                            tint = BrandRed
                        )
                    }
                }
            }
        },
        dismissButton = {
            if (!showRemoveConfirmation) {
                TextButton(onClick = onDismiss) {
                    Text("Close")
                }
            }
        }
    )
}

@Composable
private fun DetailRow(
    label: String,
    value: String,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodySmall,
            color = OnBackground.copy(alpha = 0.6f)
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Medium
        )
    }
}

@Composable
private fun DeviceListEmptyState(
    onRefresh: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier.padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = Icons.Outlined.Devices,
            contentDescription = null,
            tint = OnBackground.copy(alpha = 0.3f),
            modifier = Modifier.size(80.dp)
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = "No Devices Found",
            style = MaterialTheme.typography.titleMedium,
            color = OnBackground.copy(alpha = 0.7f)
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Add a device to start syncing your encrypted messages",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.5f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(24.dp))

        Button(
            onClick = onRefresh,
            colors = ButtonDefaults.buttonColors(
                containerColor = BrandPurple
            )
        ) {
            Icon(
                imageVector = Icons.Default.Refresh,
                contentDescription = null,
                modifier = Modifier.size(18.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text("Refresh")
        }
    }
}

private fun formatLastSeen(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

    return when {
        diff < 60_000 -> "Just now"
        diff < 3_600_000 -> "${diff / 60_000} minutes ago"
        diff < 86_400_000 -> "${diff / 3_600_000} hours ago"
        diff < 604_800_000 -> "${diff / 86_400_000} days ago"
        else -> {
            val sdf = java.text.SimpleDateFormat("MMM d, yyyy", java.util.Locale.getDefault())
            sdf.format(java.util.Date(timestamp))
        }
    }
}
