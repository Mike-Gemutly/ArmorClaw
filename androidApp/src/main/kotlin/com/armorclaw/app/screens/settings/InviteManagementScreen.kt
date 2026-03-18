package com.armorclaw.app.screens.settings

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.armorclaw.app.viewmodels.InviteViewModel
import com.armorclaw.app.viewmodels.InviteUiState
import com.armorclaw.shared.platform.bridge.*
import com.armorclaw.shared.ui.theme.*
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import org.koin.androidx.compose.koinViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InviteManagementScreen(
    onBack: () -> Unit,
    viewModel: InviteViewModel = koinViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val context = LocalContext.current

    var showGenerateDialog by remember { mutableStateOf(false) }
    var selectedExpiration by remember { mutableStateOf(InviteExpiration.SEVEN_DAYS) }
    var maxUses by remember { mutableStateOf("") }
    var serverName by remember { mutableStateOf("") }
    var welcomeMessage by remember { mutableStateOf("") }

    // Handle share intent
    val shareLauncher = remember { mutableStateOf<String?>(null) }
    LaunchedEffect(shareLauncher.value) {
        shareLauncher.value?.let { text ->
            val sendIntent = Intent().apply {
                action = Intent.ACTION_SEND
                putExtra(Intent.EXTRA_TEXT, text)
                type = "text/plain"
            }
            context.startActivity(Intent.createChooser(sendIntent, "Share Invite Link"))
            shareLauncher.value = null
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Invite Users") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface
                )
            )
        },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = { showGenerateDialog = true },
                icon = { Icon(Icons.Default.AddLink, contentDescription = null) },
                text = { Text("Create Invite") },
                containerColor = BrandPurple,
                contentColor = MaterialTheme.colorScheme.onPrimary
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(rememberScrollState())
        ) {
            // Success/Error messages
            AnimatedVisibility(
                visible = uiState.successMessage != null || uiState.error != null,
                enter = expandVertically() + fadeIn(),
                exit = shrinkVertically() + fadeOut()
            ) {
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(DesignTokens.Spacing.md),
                    colors = CardDefaults.cardColors(
                        containerColor = if (uiState.successMessage != null)
                            BrandGreen.copy(alpha = 0.1f)
                        else
                            BrandRed.copy(alpha = 0.1f)
                    ),
                    border = BorderStroke(
                        1.dp,
                        if (uiState.successMessage != null) BrandGreen else BrandRed
                    )
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(DesignTokens.Spacing.md),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = uiState.successMessage ?: uiState.error ?: "",
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (uiState.successMessage != null) BrandGreen else BrandRed,
                            modifier = Modifier.weight(1f)
                        )
                        IconButton(onClick = { viewModel.clearMessages() }) {
                            Icon(Icons.Default.Close, contentDescription = "Dismiss")
                        }
                    }
                }
            }

            // Generated link card
            AnimatedVisibility(
                visible = uiState.generatedLink != null,
                enter = expandVertically() + fadeIn(),
                exit = shrinkVertically() + fadeOut()
            ) {
                GeneratedLinkCard(
                    link = uiState.generatedLink ?: "",
                    invite = uiState.lastGeneratedInvite,
                    onCopy = {
                        copyToClipboard(context, uiState.generatedLink ?: "")
                        viewModel.clearMessages()
                    },
                    onShare = {
                        shareLauncher.value = viewModel.getShareText()
                    },
                    onClear = { viewModel.clearGeneratedLink() }
                )
            }

            // Info card
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(DesignTokens.Spacing.md)
            ) {
                Column(
                    modifier = Modifier.padding(DesignTokens.Spacing.md)
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                    ) {
                        Icon(
                            Icons.Default.Info,
                            contentDescription = null,
                            tint = BrandPurple
                        )
                        Text(
                            text = "Invite Links",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold
                        )
                    }

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

                    Text(
                        text = "Generate time-limited invite links to share your server configuration with new users. Links are cryptographically signed and cannot be tampered with.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                    )
                }
            }

            // Active invites section
            if (uiState.hasInviteLinks) {
                Text(
                    text = "Your Invite Links",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(
                        horizontal = DesignTokens.Spacing.lg,
                        vertical = DesignTokens.Spacing.sm
                    )
                )

                // Active invites
                if (uiState.activeInvites.isNotEmpty()) {
                    Text(
                        text = "Active (${uiState.activeInvites.size})",
                        style = MaterialTheme.typography.labelLarge,
                        color = BrandGreen,
                        modifier = Modifier.padding(
                            horizontal = DesignTokens.Spacing.lg,
                            vertical = DesignTokens.Spacing.xs
                        )
                    )

                    uiState.activeInvites.forEach { invite ->
                        InviteLinkCard(
                            invite = invite,
                            onRevoke = { viewModel.revokeInvite(invite.id) },
                            onCopy = { copyToClipboard(context, it) }
                        )
                    }
                }

                // Expired/Exhausted invites
                if (uiState.expiredInvites.isNotEmpty() || uiState.exhaustedInvites.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

                    Text(
                        text = "Inactive",
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                        modifier = Modifier.padding(
                            horizontal = DesignTokens.Spacing.lg,
                            vertical = DesignTokens.Spacing.xs
                        )
                    )

                    (uiState.expiredInvites + uiState.exhaustedInvites + uiState.revokedInvites)
                        .distinctBy { it.id }
                        .forEach { invite ->
                            InviteLinkCard(
                                invite = invite,
                                onRevoke = { viewModel.revokeInvite(invite.id) },
                                onCopy = { copyToClipboard(context, it) },
                                isInactive = true
                            )
                        }
                }
            } else {
                // Empty state
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(DesignTokens.Spacing.xl),
                    contentAlignment = Alignment.Center
                ) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                    ) {
                        Icon(
                            Icons.Default.Link,
                            contentDescription = null,
                            modifier = Modifier.size(64.dp),
                            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
                        )
                        Text(
                            text = "No invite links yet",
                            style = MaterialTheme.typography.titleMedium,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
                        )
                        Text(
                            text = "Tap the button below to create your first invite link",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(100.dp)) // FAB space
        }
    }

    // Generate dialog
    if (showGenerateDialog) {
        GenerateInviteDialog(
            selectedExpiration = selectedExpiration,
            onExpirationChange = { selectedExpiration = it },
            maxUses = maxUses,
            onMaxUsesChange = { maxUses = it },
            serverName = serverName,
            onServerNameChange = { serverName = it },
            welcomeMessage = welcomeMessage,
            onWelcomeMessageChange = { welcomeMessage = it },
            isGenerating = uiState.isGenerating,
            onGenerate = {
                viewModel.generateInviteLink(
                    expiration = selectedExpiration,
                    maxUses = maxUses.toIntOrNull(),
                    serverName = serverName.ifBlank { null },
                    welcomeMessage = welcomeMessage.ifBlank { null }
                )
                showGenerateDialog = false
                maxUses = ""
                serverName = ""
                welcomeMessage = ""
            },
            onDismiss = { showGenerateDialog = false }
        )
    }
}

@Composable
private fun GeneratedLinkCard(
    link: String,
    invite: InviteLink?,
    onCopy: () -> Unit,
    onShare: () -> Unit,
    onClear: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(DesignTokens.Spacing.md),
        colors = CardDefaults.cardColors(
            containerColor = BrandPurple.copy(alpha = 0.1f)
        ),
        border = BorderStroke(1.dp, BrandPurple)
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Invite Link Generated!",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = BrandPurple
                )
                IconButton(onClick = onClear) {
                    Icon(Icons.Default.Close, contentDescription = "Clear")
                }
            }

            if (invite != null) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                ) {
                    AssistChip(
                        onClick = {},
                        label = { Text(formatTimeRemaining(invite.expiresAt)) },
                        leadingIcon = {
                            Icon(
                                Icons.Default.Schedule,
                                contentDescription = null,
                                modifier = Modifier.size(16.dp)
                            )
                        }
                    )

                    invite.maxUses?.let { max ->
                        AssistChip(
                            onClick = {},
                            label = { Text("${invite.remainingUses ?: 0}/$max uses") },
                            leadingIcon = {
                                Icon(
                                    Icons.Default.People,
                                    contentDescription = null,
                                    modifier = Modifier.size(16.dp)
                                )
                            }
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            SelectionContainer {
                Text(
                    text = link,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
            ) {
                OutlinedButton(
                    onClick = onCopy,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(Icons.Default.ContentCopy, contentDescription = null)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Copy")
                }

                Button(
                    onClick = onShare,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(Icons.Default.Share, contentDescription = null)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Share")
                }
            }
        }
    }
}

@Composable
private fun InviteLinkCard(
    invite: InviteLink,
    onRevoke: () -> Unit,
    onCopy: (String) -> Unit,
    isInactive: Boolean = false
) {
    var showRevokeConfirm by remember { mutableStateOf(false) }

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .padding(
                horizontal = DesignTokens.Spacing.md,
                vertical = DesignTokens.Spacing.xs
            ),
        colors = CardDefaults.cardColors(
            containerColor = if (isInactive)
                MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
            else
                MaterialTheme.colorScheme.surface
        )
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = invite.serverConfig.serverName ?: "Server Invite",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold
                    )

                    Text(
                        text = "ID: ${invite.id.take(20)}...",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
                    )
                }

                // Status badge
                Surface(
                    color = when {
                        !invite.isActive -> BrandRed.copy(alpha = 0.1f)
                        invite.isExpired -> BrandYellow.copy(alpha = 0.1f)
                        invite.isExhausted -> BrandYellow.copy(alpha = 0.1f)
                        else -> BrandGreen.copy(alpha = 0.1f)
                    },
                    shape = MaterialTheme.shapes.small
                ) {
                    Text(
                        text = when {
                            !invite.isActive -> "Revoked"
                            invite.isExpired -> "Expired"
                            invite.isExhausted -> "Used up"
                            else -> "Active"
                        },
                        style = MaterialTheme.typography.labelSmall,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                        color = when {
                            !invite.isActive -> BrandRed
                            invite.isExpired -> BrandYellow
                            invite.isExhausted -> BrandYellow
                            else -> BrandGreen
                        }
                    )
                }
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.lg)
            ) {
                // Expiration
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Icon(
                        Icons.Default.Schedule,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp),
                        tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
                    )
                    Text(
                        text = formatTimeRemaining(invite.expiresAt),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                    )
                }

                // Usage
                invite.maxUses?.let { max ->
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(4.dp)
                    ) {
                        Icon(
                            Icons.Default.People,
                            contentDescription = null,
                            modifier = Modifier.size(16.dp),
                            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
                        )
                        Text(
                            text = "${invite.currentUses}/$max used",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                        )
                    }
                }
            }

            if (invite.isActive && !invite.isExpired) {
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                ) {
                    TextButton(onClick = { /* Copy invite URL */ }) {
                        Icon(Icons.Default.ContentCopy, contentDescription = null)
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Copy Link")
                    }

                    TextButton(
                        onClick = { showRevokeConfirm = true },
                        colors = ButtonDefaults.textButtonColors(
                            contentColor = BrandRed
                        )
                    ) {
                        Icon(Icons.Default.Block, contentDescription = null)
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("Revoke")
                    }
                }
            }
        }
    }

    // Revoke confirmation dialog
    if (showRevokeConfirm) {
        AlertDialog(
            onDismissRequest = { showRevokeConfirm = false },
            title = { Text("Revoke Invite Link?") },
            text = { Text("This action cannot be undone. Anyone with this link will no longer be able to join.") },
            confirmButton = {
                TextButton(
                    onClick = {
                        onRevoke()
                        showRevokeConfirm = false
                    },
                    colors = ButtonDefaults.textButtonColors(contentColor = BrandRed)
                ) {
                    Text("Revoke")
                }
            },
            dismissButton = {
                TextButton(onClick = { showRevokeConfirm = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun GenerateInviteDialog(
    selectedExpiration: InviteExpiration,
    onExpirationChange: (InviteExpiration) -> Unit,
    maxUses: String,
    onMaxUsesChange: (String) -> Unit,
    serverName: String,
    onServerNameChange: (String) -> Unit,
    welcomeMessage: String,
    onWelcomeMessageChange: (String) -> Unit,
    isGenerating: Boolean,
    onGenerate: () -> Unit,
    onDismiss: () -> Unit
) {
    var expanded by remember { mutableStateOf(false) }

    AlertDialog(
        onDismissRequest = { if (!isGenerating) onDismiss() },
        title = { Text("Create Invite Link") },
        text = {
            Column(
                verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
            ) {
                // Expiration
                ExposedDropdownMenuBox(
                    expanded = expanded,
                    onExpandedChange = { expanded = it }
                ) {
                    OutlinedTextField(
                        value = formatExpirationLabel(selectedExpiration),
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("Link expires after") },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
                        modifier = Modifier
                            .menuAnchor()
                            .fillMaxWidth()
                    )

                    ExposedDropdownMenu(
                        expanded = expanded,
                        onDismissRequest = { expanded = false }
                    ) {
                        InviteExpiration.values().forEach { expiration ->
                            DropdownMenuItem(
                                text = { Text(formatExpirationLabel(expiration)) },
                                onClick = {
                                    onExpirationChange(expiration)
                                    expanded = false
                                }
                            )
                        }
                    }
                }

                // Max uses
                OutlinedTextField(
                    value = maxUses,
                    onValueChange = {
                        if (it.isEmpty() || it.toIntOrNull() != null) {
                            onMaxUsesChange(it)
                        }
                    },
                    label = { Text("Maximum uses (optional)") },
                    placeholder = { Text("Leave empty for unlimited") },
                    keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(
                        keyboardType = androidx.compose.ui.text.input.KeyboardType.Number
                    ),
                    modifier = Modifier.fillMaxWidth()
                )

                Divider()

                // Optional server name
                OutlinedTextField(
                    value = serverName,
                    onValueChange = onServerNameChange,
                    label = { Text("Server name (optional)") },
                    placeholder = { Text("My ArmorClaw Server") },
                    modifier = Modifier.fillMaxWidth()
                )

                // Optional welcome message
                OutlinedTextField(
                    value = welcomeMessage,
                    onValueChange = onWelcomeMessageChange,
                    label = { Text("Welcome message (optional)") },
                    placeholder = { Text("Welcome to our secure chat!") },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                    maxLines = 4
                )
            }
        },
        confirmButton = {
            Button(
                onClick = onGenerate,
                enabled = !isGenerating
            ) {
                if (isGenerating) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(20.dp),
                        color = MaterialTheme.colorScheme.onPrimary
                    )
                } else {
                    Text("Generate")
                }
            }
        },
        dismissButton = {
            TextButton(
                onClick = onDismiss,
                enabled = !isGenerating
            ) {
                Text("Cancel")
            }
        }
    )
}

// Helpers

private fun copyToClipboard(context: Context, text: String) {
    val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    val clip = ClipData.newPlainText("Invite Link", text)
    clipboard.setPrimaryClip(clip)
}

private fun formatExpirationLabel(expiration: InviteExpiration): String {
    return when (expiration) {
        InviteExpiration.ONE_HOUR -> "1 hour"
        InviteExpiration.SIX_HOURS -> "6 hours"
        InviteExpiration.ONE_DAY -> "1 day"
        InviteExpiration.THREE_DAYS -> "3 days"
        InviteExpiration.SEVEN_DAYS -> "7 days"
        InviteExpiration.FOURTEEN_DAYS -> "14 days"
        InviteExpiration.THIRTY_DAYS -> "30 days"
    }
}

private fun formatTimeRemaining(expiresAt: Instant): String {
    val now = Clock.System.now()
    val duration = expiresAt - now

    return when {
        duration.isNegative() -> "Expired"
        duration.inWholeMinutes < 60 -> "${duration.inWholeMinutes}m left"
        duration.inWholeHours < 24 -> "${duration.inWholeHours}h left"
        duration.inWholeDays < 7 -> "${duration.inWholeDays}d left"
        else -> "${duration.inWholeDays}d left"
    }
}
