package com.armorclaw.app.screens.onboarding

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.*
import androidx.compose.animation.fadeIn
import androidx.compose.animation.slideInVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
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
import kotlinx.coroutines.delay

/**
 * Express Setup Complete Screen
 *
 * "You're all set!" screen shown after express setup completion.
 * Displays what was configured and offers quick actions.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ExpressSetupCompleteScreen(
    onStartChatting: () -> Unit,
    onViewSettings: () -> Unit,
    modifier: Modifier = Modifier
) {
    var showContent by remember { mutableStateOf(false) }
    val scale = remember { Animatable(0f) }

    LaunchedEffect(Unit) {
        delay(200)
        scale.animateTo(
            targetValue = 1f,
            animationSpec = spring(
                dampingRatio = Spring.DampingRatioMediumBouncy,
                stiffness = Spring.StiffnessLow
            )
        )
        showContent = true
    }

    ArmorClawTheme {
        Scaffold(
            containerColor = MaterialTheme.colorScheme.background
        ) { paddingValues ->
            Box(
                modifier = modifier
                    .fillMaxSize()
                    .background(
                        brush = Brush.verticalGradient(
                            colors = listOf(
                                BrandGreen.copy(alpha = 0.1f),
                                MaterialTheme.colorScheme.background
                            )
                        )
                    )
                    .padding(paddingValues)
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(DesignTokens.Spacing.lg),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

                    // Success animation
                    Box(
                        modifier = Modifier
                            .size(140.dp)
                            .scale(scale.value),
                        contentAlignment = Alignment.Center
                    ) {
                        // Outer ring
                        Box(
                            modifier = Modifier
                                .fillMaxSize()
                                .clip(CircleShape)
                                .background(
                                    brush = Brush.radialGradient(
                                        colors = listOf(
                                            BrandGreen.copy(alpha = 0.3f),
                                            BrandGreen.copy(alpha = 0.1f),
                                            Color.Transparent
                                        )
                                    )
                                )
                        )

                        // Inner circle with check
                        Box(
                            modifier = Modifier
                                .size(100.dp)
                                .clip(CircleShape)
                                .background(BrandGreen),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.Check,
                                contentDescription = null,
                                tint = Color.White,
                                modifier = Modifier.size(56.dp)
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

                    // Title
                    AnimatedVisibility(
                        visible = showContent,
                        enter = fadeIn() + slideInVertically()
                    ) {
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally
                        ) {
                            Text(
                                text = "You're all set!",
                                style = MaterialTheme.typography.headlineMedium.copy(
                                    fontWeight = FontWeight.Bold
                                ),
                                textAlign = TextAlign.Center
                            )

                            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

                            Text(
                                text = "Your security settings have been configured with smart defaults",
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
                                textAlign = TextAlign.Center
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

                    // Configured settings summary
                    AnimatedVisibility(
                        visible = showContent,
                        enter = fadeIn(animationSpec = tween(500, delayMillis = 200))
                    ) {
                        ConfiguredSettingsCard()
                    }

                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

                    // Security note
                    AnimatedVisibility(
                        visible = showContent,
                        enter = fadeIn(animationSpec = tween(500, delayMillis = 400))
                    ) {
                        SecurityNoteCard()
                    }

                    Spacer(modifier = Modifier.weight(1f))

                    // Action buttons
                    AnimatedVisibility(
                        visible = showContent,
                        enter = fadeIn(animationSpec = tween(500, delayMillis = 600))
                    ) {
                        Column(
                            modifier = Modifier.fillMaxWidth(),
                            verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                        ) {
                            Button(
                                onClick = onStartChatting,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .height(56.dp),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = BrandGreen
                                )
                            ) {
                                Icon(
                                    imageVector = Icons.Default.Chat,
                                    contentDescription = null,
                                    modifier = Modifier.size(20.dp)
                                )
                                Spacer(modifier = Modifier.width(8.dp))
                                Text(
                                    text = "Start Chatting",
                                    style = MaterialTheme.typography.labelLarge.copy(
                                        fontWeight = FontWeight.Bold
                                    )
                                )
                            }

                            OutlinedButton(
                                onClick = onViewSettings,
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Icon(
                                    imageVector = Icons.Default.Settings,
                                    contentDescription = null,
                                    modifier = Modifier.size(18.dp)
                                )
                                Spacer(modifier = Modifier.width(8.dp))
                                Text("View Security Settings")
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ConfiguredSettingsCard() {
    val configuredSettings = listOf(
        ConfiguredItem(
            category = "Banking",
            status = "Protected",
            icon = Icons.Default.AccountBalance,
            color = Color(0xFFF44336)
        ),
        ConfiguredItem(
            category = "Medical",
            status = "Protected",
            icon = Icons.Default.LocalHospital,
            color = Color(0xFFF44336)
        ),
        ConfiguredItem(
            category = "PII",
            status = "Allowed",
            icon = Icons.Default.Person,
            color = BrandGreen
        ),
        ConfiguredItem(
            category = "Network",
            status = "Allowed",
            icon = Icons.Default.Wifi,
            color = BrandGreen
        ),
        ConfiguredItem(
            category = "Location",
            status = "Protected",
            icon = Icons.Default.LocationOn,
            color = Color(0xFFF44336)
        ),
        ConfiguredItem(
            category = "Credentials",
            status = "Protected",
            icon = Icons.Default.Key,
            color = Color(0xFFF44336)
        )
    )

    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.lg)
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
                    text = "What's Protected",
                    style = MaterialTheme.typography.titleMedium.copy(
                        fontWeight = FontWeight.Bold
                    )
                )
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

            // Grid of configured items
            configuredSettings.chunked(2).forEach { row ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                ) {
                    row.forEach { item ->
                        ConfiguredItemCard(
                            item = item,
                            modifier = Modifier.weight(1f)
                        )
                    }
                    // Add spacer if odd number
                    if (row.size == 1) {
                        Spacer(modifier = Modifier.weight(1f))
                    }
                }
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.xs))
            }
        }
    }
}

@Composable
private fun ConfiguredItemCard(
    item: ConfiguredItem,
    modifier: Modifier = Modifier
) {
    val isProtected = item.status == "Protected"
    val backgroundColor = if (isProtected) {
        Color(0xFFF44336).copy(alpha = 0.1f)
    } else {
        BrandGreen.copy(alpha = 0.1f)
    }

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(12.dp),
        color = backgroundColor
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = item.icon,
                contentDescription = null,
                tint = item.color,
                modifier = Modifier.size(18.dp)
            )

            Column {
                Text(
                    text = item.category,
                    style = MaterialTheme.typography.labelMedium.copy(
                        fontWeight = FontWeight.Medium
                    )
                )
                Text(
                    text = item.status,
                    style = MaterialTheme.typography.labelSmall,
                    color = item.color
                )
            }
        }
    }
}

@Composable
private fun SecurityNoteCard() {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = BrandPurple.copy(alpha = 0.1f)
        ),
        border = androidx.compose.foundation.BorderStroke(
            1.dp,
            BrandPurple.copy(alpha = 0.3f)
        )
    ) {
        Row(
            modifier = Modifier.padding(DesignTokens.Spacing.md),
            horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Info,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(24.dp)
            )

            Column {
                Text(
                    text = "You can change these anytime",
                    style = MaterialTheme.typography.titleSmall.copy(
                        fontWeight = FontWeight.Medium
                    ),
                    color = BrandPurple
                )
                Text(
                    text = "Go to Settings > Security to customize",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                )
            }
        }
    }
}

private data class ConfiguredItem(
    val category: String,
    val status: String,
    val icon: ImageVector,
    val color: Color
)
