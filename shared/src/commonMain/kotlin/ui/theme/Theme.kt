package com.armorclaw.shared.ui.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.runtime.Composable
import com.armorclaw.ui.theme.ArmorClawTheme as UiTheme
import com.armorclaw.ui.theme.AppTheme as UiAppTheme

/**
 * ArmorClaw unified theme - wrapper around armorclaw-ui theme.
 * Dark mode is the only default experience.
 */
@Composable
fun ArmorClawTheme(
    darkTheme: Boolean = true,
    content: @Composable () -> Unit
) {
    UiTheme(darkTheme = darkTheme, content = content)
}

/**
 * Provides easy access to theme values.
 */
object AppTheme {
    val colors: ColorScheme
        @Composable get() = UiAppTheme.colors

    val typography: Typography
        @Composable get() = UiAppTheme.typography

    val shapes: Shapes
        @Composable get() = UiAppTheme.shapes
}

// Re-export unified colors for backwards compatibility
val Teal = com.armorclaw.ui.theme.Teal
val TealGlow = com.armorclaw.ui.theme.TealGlow
val Navy = com.armorclaw.ui.theme.Navy
val PrecisionBlue = com.armorclaw.ui.theme.PrecisionBlue
val SuccessGreen = com.armorclaw.ui.theme.SuccessGreen
val WarningAmber = com.armorclaw.ui.theme.WarningAmber
val ErrorRed = com.armorclaw.ui.theme.ErrorRed
