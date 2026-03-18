package com.armorclaw.ui.theme

import androidx.compose.foundation.background
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Shape

/**
 * Applies a teal glow effect to a composable.
 * Use for accent elements like buttons, icons, or highlighted items.
 */
fun Modifier.glowTeal(
    shape: Shape = CircleShape,
    ambientAlpha: Float = 0.4f,
    spotAlpha: Float = 0.4f
): Modifier = this
    .shadow(
        elevation = androidx.compose.ui.unit.Dp(8f),
        shape = shape,
        ambientColor = TealGlow.copy(alpha = ambientAlpha),
        spotColor = TealGlow.copy(alpha = spotAlpha)
    )
    .background(
        color = Teal.copy(alpha = 0.08f),
        shape = shape
    )

/**
 * Applies a subtle teal border glow for interactive elements.
 */
fun Modifier.tealBorder(
    shape: Shape = CircleShape
): Modifier = this
    .shadow(
        elevation = androidx.compose.ui.unit.Dp(4f),
        shape = shape,
        ambientColor = Teal.copy(alpha = 0.2f),
        spotColor = Teal.copy(alpha = 0.2f)
    )
