package com.armorclaw.app.screens.onboarding

import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurpleLight
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.app.viewmodels.SetupViewModel
import com.armorclaw.app.viewmodels.SetupUiState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow

/**
 * Setup Mode Selection Screen
 *
 * Allows user to choose between Express setup with smart defaults
 * or Custom setup for full control.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SetupModeSelectionScreen(
    viewModel: SetupViewModel,
    onExpressSelected: () -> Unit,
    onCustomSelected: () -> Unit,
    onBack: () -> Unit
) {
    var selectedMode by remember { mutableStateOf<SetupMode?>(null) }
var showRetrySnackbar by remember { mutableStateOf(false) }

        LaunchedEffect(showRetrySnackbar) {
            if (showRetrySnackbar) {
                showRetrySnackbar = false
            }
        }

        ArmorClawTheme {
            Scaffold(
                topBar = {
                    TopAppBar(
                        title = { Text("Choose Your Setup") },
                        navigationIcon = {
                            IconButton(onClick = onBack) {
                                Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                            }
                        }
                    )
                },
                snackbarHost = {
                    SnackbarHost(hostState = SnackbarHostState()) { data ->
                        Snackbar(
                            modifier = Modifier.padding(DesignTokens.Spacing.md),
                            containerColor = MaterialTheme.colorScheme.errorContainer,
                            contentColor = MaterialTheme.colorScheme.onErrorContainer
                        ) {
                            Text(data.visuals.message, style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                }
            ) { paddingValues ->
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(paddingValues)
                        .padding(DesignTokens.Spacing.lg),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    // Header
                    Text(
                    text = "How would you like to set up?",
                    style = MaterialTheme.typography.headlineMedium.copy(
                        fontWeight = FontWeight.Bold
                    ),
                    textAlign = TextAlign.Center
                )

                Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

                Text(
                    text = "You can always change these settings later",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    textAlign = TextAlign.Center
                )

                Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

                // Mode cards
                Column(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                ) {
                    // Express Mode
                    SetupModeCard(
                        mode = SetupMode.EXPRESS,
                        title = "Express Setup",
                        subtitle = "Recommended for most users",
                        description = "Smart security defaults configured for you. Get started in seconds!",
                        icon = Icons.Default.Bolt,
                        features = listOf(
                            "Banking & Medical: Protected",
                            "PII & Identity: Allowed",
                            "Location: Protected"
                        ),
                        isSelected = selectedMode == SetupMode.EXPRESS,
                        recommended = true,
                        onClick = { selectedMode = SetupMode.EXPRESS }
                    )

                    // Custom Mode
                    SetupModeCard(
                        mode = SetupMode.CUSTOM,
                        title = "Custom Setup",
                        subtitle = "For power users",
                        description = "Configure every security setting yourself for maximum control.",
                        icon = Icons.Default.Settings,
                        features = listOf(
                            "Full control over all categories",
                            "Granular permission settings",
                            "Advanced security options"
                        ),
                        isSelected = selectedMode == SetupMode.CUSTOM,
                        recommended = false,
                        onClick = { selectedMode = SetupMode.CUSTOM }
                    )
                }

                Spacer(modifier = Modifier.weight(1f))

                // Smart Defaults Preview
                if (selectedMode == SetupMode.EXPRESS) {
                    SmartDefaultsPreview(
                        modifier = Modifier.animateContentSize(
                            animationSpec = spring(Spring.DampingRatioMediumBouncy)
                        )
                    )
                }

                Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

                // Action buttons
                Button(
                    onClick = {
                        when (selectedMode) {
                            SetupMode.EXPRESS -> onExpressSelected()
                            SetupMode.CUSTOM -> onCustomSelected()
                            null -> { }
                        }
                    },
                    enabled = selectedMode != null,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(56.dp)
                ) {
                    Text(
                        text = when (selectedMode) {
                            SetupMode.EXPRESS -> "Continue with Express Setup"
                            SetupMode.CUSTOM -> "Continue with Custom Setup"
                            null -> "Select a Setup Mode"
                        },
                        style = MaterialTheme.typography.labelLarge
                    )
                }
            }
        }
    }
}

@Composable
private fun SetupModeCard(
    mode: SetupMode,
    title: String,
    subtitle: String,
    description: String,
    icon: ImageVector,
    features: List<String>,
    isSelected: Boolean,
    recommended: Boolean,
    onClick: () -> Unit
) {
    val borderColor = if (isSelected) BrandPurple else Color.Gray.copy(alpha = 0.3f)
    val backgroundColor = if (isSelected) {
        BrandPurple.copy(alpha = 0.1f)
    } else {
        MaterialTheme.colorScheme.surface
    }

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .border(
                width = if (isSelected) 2.dp else 1.dp,
                color = borderColor,
                shape = RoundedCornerShape(16.dp)
            ),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = backgroundColor),
        elevation = CardDefaults.cardElevation(
            defaultElevation = if (isSelected) 4.dp else 1.dp
        )
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.lg)
        ) {
            // Header row
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Box(
                        modifier = Modifier
                            .size(48.dp)
                            .clip(CircleShape)
                            .background(
                                if (mode == SetupMode.EXPRESS) BrandGreen.copy(alpha = 0.2f)
                                else BrandPurple.copy(alpha = 0.2f)
                            ),
                        contentAlignment = Alignment.Center
                    ) {
                        Icon(
                            imageVector = icon,
                            contentDescription = null,
                            tint = if (mode == SetupMode.EXPRESS) BrandGreen else BrandPurple,
                            modifier = Modifier.size(24.dp)
                        )
                    }

                    Column {
                        Text(
                            text = title,
                            style = MaterialTheme.typography.titleMedium.copy(
                                fontWeight = FontWeight.Bold
                            )
                        )
                        Text(
                            text = subtitle,
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                        )
                    }
                }

                // Recommended badge or selection indicator
                if (recommended) {
                    Surface(
                        shape = RoundedCornerShape(12.dp),
                        color = BrandGreen.copy(alpha = 0.2f)
                    ) {
                        Text(
                            text = "Recommended",
                            style = MaterialTheme.typography.labelSmall,
                            color = BrandGreen,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                        )
                    }
                } else if (isSelected) {
                    Icon(
                        imageVector = Icons.Default.CheckCircle,
                        contentDescription = null,
                        tint = BrandPurple,
                        modifier = Modifier.size(24.dp)
                    )
                }
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

            // Description
            Text(
                text = description,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

            // Features
            features.forEach { feature ->
                Row(
                    modifier = Modifier.padding(vertical = 2.dp),
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.Check,
                        contentDescription = null,
                        tint = BrandGreen,
                        modifier = Modifier.size(16.dp)
                    )
                    Text(
                        text = feature,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
                    )
                }
            }
        }
    }
}

@Composable
private fun SmartDefaultsPreview(modifier: Modifier = Modifier) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = BrandGreen.copy(alpha = 0.1f)
        ),
        border = BorderStroke(1.dp, BrandGreen.copy(alpha = 0.3f)),
        shape = RoundedCornerShape(12.dp)
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Shield,
                    contentDescription = null,
                    tint = BrandGreen,
                    modifier = Modifier.size(20.dp)
                )
                Text(
                    text = "Smart Security Defaults",
                    style = MaterialTheme.typography.titleSmall.copy(
                        fontWeight = FontWeight.Bold
                    ),
                    color = BrandGreen
                )
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            // Default categories preview
            val defaults = listOf(
                "Banking" to SecurityDefault.DENY,
                "PII" to SecurityDefault.ALLOW,
                "Medical" to SecurityDefault.DENY,
                "Network" to SecurityDefault.ALLOW,
                "Location" to SecurityDefault.DENY,
                "Credentials" to SecurityDefault.DENY
            )

            defaults.chunked(3).forEach { row ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                ) {
                    row.forEach { (category, default) ->
                        DefaultChip(category = category, default = default)
                    }
                }
                Spacer(modifier = Modifier.height(4.dp))
            }
        }
    }
}

@Composable
private fun DefaultChip(category: String, default: SecurityDefault) {
    val (color, icon) = when (default) {
        SecurityDefault.ALLOW -> BrandGreen to Icons.Default.Check
        SecurityDefault.DENY -> Color(0xFFF44336) to Icons.Default.Close
        SecurityDefault.ALLOW_ALL -> BrandGreen to Icons.Default.DoneAll
    }

    Surface(
        shape = RoundedCornerShape(8.dp),
        color = color.copy(alpha = 0.1f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 3.dp),
            horizontalArrangement = Arrangement.spacedBy(3.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(12.dp)
            )
            Text(
                text = category,
                style = MaterialTheme.typography.labelSmall,
                color = color
            )
        }
    }
}

enum class SetupMode {
    EXPRESS,
    CUSTOM
}

enum class SecurityDefault {
    ALLOW,
    DENY,
    ALLOW_ALL
}
