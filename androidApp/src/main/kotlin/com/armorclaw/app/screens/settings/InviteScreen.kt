package com.armorclaw.app.screens.settings

import kotlinx.coroutines.launch
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.DesignTokens

/**
 * Screen for generating and sharing invite links
 *
 * Part of the viral growth strategy - allows users to invite
 * friends to ArmorClaw via QR code or shareable link.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InviteScreen(
    onNavigateBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val clipboardManager = LocalClipboardManager.current
    val coroutineScope = rememberCoroutineScope()

    // State
    var inviteLink by remember { mutableStateOf<String?>(null) }
    var isGenerating by remember { mutableStateOf(false) }
    var showCopiedToast by remember { mutableStateOf(false) }
    var selectedExpiration by remember { mutableStateOf("7d") }

    // Generate invite on first load
    LaunchedEffect(Unit) {
        isGenerating = true
        // TODO: Call InviteService to generate actual invite
        // For now, use a placeholder
        kotlinx.coroutines.delay(1000)
        inviteLink = "armorclaw://config?d=eyJtYXRyaXhfaG9tZXNlcnZlciI6Imh0dHBzOi8vbWF0cml4LmFybW9yY2xhdy5hcHAiLCJycGNfdXJsIjoiaHR0cHM6Ly9icmlkZ2UuYXJtb3JjbGF3LmFwcCIsInNlcnZlcl9uYW1lIjoiQXJtb3JDbGF3In0="
        isGenerating = false
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Invite to ArmorClaw") },
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
                .padding(DesignTokens.Spacing.lg)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.lg)
        ) {
            // Header
            Icon(
                imageVector = Icons.Default.PersonAdd,
                contentDescription = null,
                modifier = Modifier.size(64.dp),
                tint = BrandPurple
            )

            Text(
                text = "Invite Friends",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )

            Text(
                text = "Share secure, encrypted chat with your friends and family.",
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
            )

            // QR Code placeholder
            if (isGenerating) {
                Box(
                    modifier = Modifier
                        .size(240.dp)
                        .clip(RoundedCornerShape(16.dp))
                        .background(MaterialTheme.colorScheme.surfaceVariant),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator(color = BrandPurple)
                }
            } else if (inviteLink != null) {
                // QR Code display
                Box(
                    modifier = Modifier
                        .size(240.dp)
                        .clip(RoundedCornerShape(16.dp))
                        .border(2.dp, BrandPurple, RoundedCornerShape(16.dp))
                        .background(Color.White),
                    contentAlignment = Alignment.Center
                ) {
                    // TODO: Generate actual QR code image
                    // For now, show a placeholder
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.Center
                    ) {
                        Icon(
                            Icons.Default.QrCode2,
                            contentDescription = "QR Code",
                            modifier = Modifier.size(180.dp),
                            tint = Color.Black
                        )
                    }
                }

                Text(
                    text = "Scan this QR code to connect",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                )
            }

            // Expiration selector
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
                        text = "Link Expires After",
                        style = MaterialTheme.typography.labelLarge,
                        fontWeight = FontWeight.Bold
                    )

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                    ) {
                        ExpirationChip(
                            label = "1 hour",
                            value = "1h",
                            selected = selectedExpiration == "1h",
                            onClick = { selectedExpiration = "1h" }
                        )
                        ExpirationChip(
                            label = "1 day",
                            value = "1d",
                            selected = selectedExpiration == "1d",
                            onClick = { selectedExpiration = "1d" }
                        )
                        ExpirationChip(
                            label = "7 days",
                            value = "7d",
                            selected = selectedExpiration == "7d",
                            onClick = { selectedExpiration = "7d" }
                        )
                        ExpirationChip(
                            label = "30 days",
                            value = "30d",
                            selected = selectedExpiration == "30d",
                            onClick = { selectedExpiration = "30d" }
                        )
                    }
                }
            }

            // Copy link button
            if (inviteLink != null) {
                Button(
                    onClick = {
                        clipboardManager.setText(AnnotatedString(inviteLink!!))
                        showCopiedToast = true
                    },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Icon(Icons.Default.ContentCopy, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Copy Invite Link")
                }

                // Share button
                OutlinedButton(
                    onClick = {
                        // TODO: Open system share sheet
                    },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Icon(Icons.Default.Share, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Share Link")
                }
            }

            // Regenerate button
            TextButton(
                onClick = {
                    isGenerating = true
                    // TODO: Regenerate invite
                    coroutineScope.launch {
                        kotlinx.coroutines.delay(1000)
                        // Generate new link
                        isGenerating = false
                    }
                },
                enabled = !isGenerating
            ) {
                Icon(Icons.Default.Refresh, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("Generate New Link")
            }

            // Security note
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = BrandGreen.copy(alpha = 0.1f)
                )
            ) {
                Row(
                    modifier = Modifier.padding(DesignTokens.Spacing.md),
                    verticalAlignment = Alignment.Top
                ) {
                    Icon(
                        Icons.Default.VerifiedUser,
                        contentDescription = null,
                        tint = BrandGreen,
                        modifier = Modifier.size(20.dp)
                    )
                    Spacer(modifier = Modifier.width(DesignTokens.Spacing.sm))
                    Text(
                        text = "Invite links are cryptographically signed and expire automatically. " +
                                "Only share with people you trust.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
                    )
                }
            }
        }
    }

    // Toast notification
    if (showCopiedToast) {
        LaunchedEffect(showCopiedToast) {
            kotlinx.coroutines.delay(2000)
            showCopiedToast = false
        }
        Snackbar(
            modifier = Modifier.padding(DesignTokens.Spacing.lg),
            action = {
                TextButton(onClick = { showCopiedToast = false }) {
                    Text("Dismiss")
                }
            }
        ) {
            Text("Link copied to clipboard")
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ExpirationChip(
    label: String,
    value: String,
    selected: Boolean,
    onClick: () -> Unit
) {
    FilterChip(
        selected = selected,
        onClick = onClick,
        label = { Text(label) },
        modifier = Modifier.height(36.dp)
    )
}
