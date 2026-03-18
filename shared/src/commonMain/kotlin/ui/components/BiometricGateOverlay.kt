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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.zIndex

/**
 * Biometric Gate Overlay
 *
 * Animated full-screen overlay displayed during biometric authentication.
 * Shows fingerprint icon with pulsing animation and instructions.
 *
 * ## Architecture
 * ```
 * BiometricGateOverlay
 *      ├── Animated background gradient
 *      ├── Fingerprint icon with pulse
 *      ├── Status text
 *      └── Cancel button
 * ```
 *
 * ## Usage
 * ```kotlin
 * if (isAuthenticating) {
 *     BiometricGateOverlay(
 *         criticalFieldCount = 2,
 *         onCancel = { viewModel.cancelBiometric() }
 *     )
 * }
 * ```
 */
@Composable
fun BiometricGateOverlay(
    criticalFieldCount: Int,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier,
    statusMessage: String = "Touch the fingerprint sensor",
    errorMessage: String? = null
) {
    // Background animation
    val infiniteTransition = rememberInfiniteTransition(label = "biometric_bg")
    val bgAlpha by infiniteTransition.animateFloat(
        initialValue = 0.95f,
        targetValue = 0.98f,
        animationSpec = infiniteRepeatable(
            animation = tween(1500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "bg_alpha"
    )

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    colors = listOf(
                        MaterialTheme.colorScheme.scrim.copy(alpha = bgAlpha),
                        MaterialTheme.colorScheme.scrim.copy(alpha = bgAlpha + 0.02f)
                    )
                )
            )
            .zIndex(10f),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            // Animated fingerprint icon
            FingerprintIcon(
                isError = errorMessage != null,
                modifier = Modifier.size(120.dp)
            )

            // Title
            Text(
                text = if (errorMessage != null) "Authentication Failed" else "Biometric Verification",
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold,
                color = Color.White,
                textAlign = TextAlign.Center
            )

            // Description
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Text(
                    text = if (errorMessage != null) {
                        errorMessage
                    } else {
                        "$criticalFieldCount critical ${if (criticalFieldCount == 1) "field requires" else "fields require"} biometric verification"
                    },
                    style = MaterialTheme.typography.bodyLarge,
                    color = Color.White.copy(alpha = 0.8f),
                    textAlign = TextAlign.Center
                )

                if (errorMessage == null) {
                    Text(
                        text = statusMessage,
                        style = MaterialTheme.typography.bodyMedium,
                        color = Color.White.copy(alpha = 0.6f),
                        textAlign = TextAlign.Center
                    )
                }
            }

            // Progress indicator
            if (errorMessage == null) {
                BiometricProgressIndicator()
            }

            // Cancel button
            Spacer(modifier = Modifier.height(16.dp))
            OutlinedButton(
                onClick = onCancel,
                shape = RoundedCornerShape(12.dp),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = Color.White
                )
            ) {
                Icon(
                    imageVector = Icons.Default.Close,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Cancel")
            }
        }
    }
}

/**
 * Animated fingerprint icon
 */
@Composable
private fun FingerprintIcon(
    isError: Boolean,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "fingerprint")

    // Pulse animation
    val scale by infiniteTransition.animateFloat(
        initialValue = 0.95f,
        targetValue = 1.05f,
        animationSpec = infiniteRepeatable(
            animation = tween(1000, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    // Glow animation
    val glowAlpha by infiniteTransition.animateFloat(
        initialValue = 0.3f,
        targetValue = 0.7f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "glow"
    )

    // Rotation for error state
    val rotation by infiniteTransition.animateFloat(
        initialValue = -3f,
        targetValue = if (isError) 3f else 0f,
        animationSpec = infiniteRepeatable(
            animation = tween(100, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "rotation"
    )

    val color = if (isError) {
        MaterialTheme.colorScheme.error
    } else {
        MaterialTheme.colorScheme.primary
    }

    Box(
        modifier = modifier.graphicsLayer {
            scaleX = scale
            scaleY = scale
            rotationZ = rotation
        },
        contentAlignment = Alignment.Center
    ) {
        // Glow effect
        Box(
            modifier = Modifier
                .size(160.dp)
                .graphicsLayer { alpha = glowAlpha }
                .background(
                    color.copy(alpha = 0.2f),
                    RoundedCornerShape(percent = 50)
                )
        )

        // Icon container
        Surface(
            shape = RoundedCornerShape(32.dp),
            color = color.copy(alpha = 0.15f)
        ) {
            Box(
                modifier = Modifier.padding(24.dp),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = if (isError) Icons.Default.Error else Icons.Default.Fingerprint,
                    contentDescription = null,
                    tint = color,
                    modifier = Modifier.size(72.dp)
                )
            }
        }
    }
}

/**
 * Linear progress indicator with animation
 */
@Composable
private fun BiometricProgressIndicator(
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "progress")
    val progress by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(3000, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "progress"
    )

    Box(
        modifier = modifier
            .width(200.dp)
            .height(4.dp)
            .clip(RoundedCornerShape(2.dp))
            .background(Color.White.copy(alpha = 0.2f))
    ) {
        Box(
            modifier = Modifier
                .fillMaxHeight()
                .fillMaxWidth(progress)
                .background(
                    MaterialTheme.colorScheme.primary,
                    RoundedCornerShape(2.dp)
                )
        )
    }
}

/**
 * Compact biometric indicator for inline use
 */
@Composable
fun BiometricIndicator(
    isActive: Boolean,
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "biometric_inline")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = if (isActive) 1f else 0.5f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Fingerprint,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary.copy(alpha = alpha),
                modifier = Modifier.size(16.dp)
            )
            Spacer(modifier = Modifier.width(6.dp))
            Text(
                text = if (isActive) "Verifying..." else "Biometric required",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.primary.copy(alpha = alpha)
            )
        }
    }
}

/**
 * Preview helper
 */
@Composable
fun BiometricGateOverlayPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(16.dp)) {
        // Normal state
        BiometricGateOverlay(
            criticalFieldCount = 2,
            onCancel = {}
        )
    }
}
