package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.blur
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.InterventionType

/**
 * Intervention Preview
 *
 * Displays a screenshot preview with blur overlay for sensitive content.
 * Used when showing CAPTCHA, 2FA, or error intervention points.
 *
 * ## Architecture
 * ```
 * InterventionPreview
 *      ├── Screenshot with blur (if sensitive)
 *      ├── Gradient overlay
 *      ├── Intervention type badge
 *      └── Action hints
 * ```
 *
 * ## Usage
 * ```kotlin
 * InterventionPreview(
 *     screenshotPath = event.screenshotPath,
 *     interventionType = InterventionType.CAPTCHA,
 *     hasSensitiveContent = true,
 *     onUnblurRequest = { viewModel.unblurForPreview() }
 * )
 * ```
 */
@Composable
fun InterventionPreview(
    screenshotPath: String?,
    interventionType: InterventionType,
    modifier: Modifier = Modifier,
    hasSensitiveContent: Boolean = true,
    isUnblurred: Boolean = false,
    onUnblurRequest: () -> Unit = {},
    contextText: String? = null
) {
    var showUnblurred by remember { mutableStateOf(isUnblurred) }

    // Animation for blur transition
    val blurAmount by animateDpAsState(
        targetValue = if (showUnblurred || !hasSensitiveContent) 0.dp else 20.dp,
        animationSpec = tween(300),
        label = "blur"
    )

    Box(
        modifier = modifier
            .fillMaxWidth()
            .heightIn(min = 200.dp, max = 300.dp)
            .clip(RoundedCornerShape(12.dp))
    ) {
        // Screenshot placeholder (would be actual image in production)
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    Brush.verticalGradient(
                        colors = listOf(
                            MaterialTheme.colorScheme.surfaceVariant,
                            MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.7f)
                        )
                    )
                )
                .blur(blurAmount)
        ) {
            // Placeholder for screenshot - in production this would be an AsyncImage
            Box(
                modifier = Modifier.fillMaxSize(),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Image,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.outline,
                        modifier = Modifier.size(48.dp)
                    )
                    Text(
                        text = "Screenshot captured",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.outline
                    )
                    if (screenshotPath != null) {
                        Text(
                            text = screenshotPath.takeLast(30),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.outline.copy(alpha = 0.5f)
                        )
                    }
                }
            }
        }

        // Gradient overlay
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    Brush.verticalGradient(
                        colors = listOf(
                            Color.Transparent,
                            Color.Black.copy(alpha = 0.3f),
                            Color.Black.copy(alpha = 0.7f)
                        ),
                        startY = 0f,
                        endY = Float.POSITIVE_INFINITY
                    )
                )
        )

        // Intervention badge
        InterventionBadge(
            type = interventionType,
            modifier = Modifier
                .align(Alignment.TopStart)
                .padding(12.dp)
        )

        // Unblur button
        if (hasSensitiveContent && !showUnblurred) {
            FilledTonalButton(
                onClick = {
                    showUnblurred = true
                    onUnblurRequest()
                },
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(12.dp),
                shape = RoundedCornerShape(8.dp),
                colors = ButtonDefaults.filledTonalButtonColors(
                    containerColor = Color.White.copy(alpha = 0.9f)
                )
            ) {
                Icon(
                    imageVector = Icons.Default.Visibility,
                    contentDescription = null,
                    modifier = Modifier.size(16.dp)
                )
                Spacer(modifier = Modifier.width(4.dp))
                Text(
                    text = "Show Content",
                    style = MaterialTheme.typography.labelMedium
                )
            }
        }

        // Context text overlay
        if (contextText != null) {
            Text(
                text = contextText,
                style = MaterialTheme.typography.bodyMedium,
                color = Color.White,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .align(Alignment.Center)
                    .padding(16.dp)
            )
        }
    }
}

/**
 * Intervention type badge
 */
@Composable
private fun InterventionBadge(
    type: InterventionType,
    modifier: Modifier = Modifier
) {
    val (icon, text, color) = when (type) {
        InterventionType.CAPTCHA -> Triple(
            Icons.Default.Security,
            "CAPTCHA Required",
            MaterialTheme.colorScheme.error
        )
        InterventionType.TWO_FA -> Triple(
            Icons.Default.Key,
            "2FA Code Needed",
            MaterialTheme.colorScheme.tertiary
        )
        InterventionType.ERROR -> Triple(
            Icons.Default.Error,
            "Error Occurred",
            MaterialTheme.colorScheme.error
        )
    }

    // Animated pulse for attention
    val infiniteTransition = rememberInfiniteTransition(label = "badge_pulse")
    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = 1.05f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    Surface(
        modifier = modifier.graphicsLayer { scaleX = scale; scaleY = scale },
        shape = RoundedCornerShape(8.dp),
        color = color
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = Color.White,
                modifier = Modifier.size(16.dp)
            )
            Spacer(modifier = Modifier.width(6.dp))
            Text(
                text = text,
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = Color.White
            )
        }
    }
}

/**
 * Compact intervention indicator for inline use
 */
@Composable
fun InterventionIndicator(
    type: InterventionType,
    modifier: Modifier = Modifier,
    onClick: () -> Unit = {}
) {
    val color = when (type) {
        InterventionType.CAPTCHA -> MaterialTheme.colorScheme.error
        InterventionType.TWO_FA -> MaterialTheme.colorScheme.tertiary
        InterventionType.ERROR -> MaterialTheme.colorScheme.error
    }

    val icon = when (type) {
        InterventionType.CAPTCHA -> Icons.Default.Security
        InterventionType.TWO_FA -> Icons.Default.Key
        InterventionType.ERROR -> Icons.Default.Error
    }

    val text = when (type) {
        InterventionType.CAPTCHA -> "CAPTCHA"
        InterventionType.TWO_FA -> "2FA"
        InterventionType.ERROR -> "Error"
    }

    Surface(
        onClick = onClick,
        modifier = modifier,
        shape = RoundedCornerShape(6.dp),
        color = color.copy(alpha = 0.15f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(14.dp)
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = text,
                style = MaterialTheme.typography.labelSmall,
                fontWeight = FontWeight.Medium,
                color = color
            )
        }
    }
}

/**
 * Intervention resolution card
 */
@Composable
fun InterventionResolutionCard(
    type: InterventionType,
    onResolve: () -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
    instructions: String? = null
) {
    val (title, description, icon) = when (type) {
        InterventionType.CAPTCHA -> Triple(
            "Complete CAPTCHA",
            "The agent detected a CAPTCHA challenge and needs your help to proceed.",
            Icons.Default.Security
        )
        InterventionType.TWO_FA -> Triple(
            "Enter 2FA Code",
            "Please provide the verification code from your authenticator app.",
            Icons.Default.Key
        )
        InterventionType.ERROR -> Triple(
            "Resolve Error",
            "The agent encountered an error and needs guidance.",
            Icons.Default.Error
        )
    }

    ElevatedCard(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp)
    ) {
        Column(
            modifier = Modifier.padding(20.dp)
        ) {
            // Header
            Row(
                verticalAlignment = Alignment.CenterVertically
            ) {
                Surface(
                    shape = RoundedCornerShape(10.dp),
                    color = MaterialTheme.colorScheme.errorContainer
                ) {
                    Box(
                        modifier = Modifier.padding(10.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        Icon(
                            imageVector = icon,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.error,
                            modifier = Modifier.size(24.dp)
                        )
                    }
                }

                Spacer(modifier = Modifier.width(12.dp))

                Column {
                    Text(
                        text = title,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Text(
                        text = "Agent needs your help",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            // Description
            Spacer(modifier = Modifier.height(16.dp))
            Text(
                text = description,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )

            // Instructions
            if (instructions != null) {
                Spacer(modifier = Modifier.height(12.dp))
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
                ) {
                    Text(
                        text = instructions,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(12.dp)
                    )
                }
            }

            // Actions
            Spacer(modifier = Modifier.height(20.dp))
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                OutlinedButton(
                    onClick = onDismiss,
                    modifier = Modifier.weight(1f)
                ) {
                    Text("Skip")
                }

                Button(
                    onClick = onResolve,
                    modifier = Modifier.weight(1f)
                ) {
                    Text("Resolve Now")
                }
            }
        }
    }
}

/**
 * Preview helper
 */
@Composable
fun InterventionPreviewPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        InterventionPreview(
            screenshotPath = "/screenshots/captcha_001.png",
            interventionType = InterventionType.CAPTCHA,
            hasSensitiveContent = true,
            contextText = "Please solve the CAPTCHA to continue"
        )

        InterventionPreview(
            screenshotPath = "/screenshots/2fa_001.png",
            interventionType = InterventionType.TWO_FA,
            hasSensitiveContent = false
        )
    }
}
