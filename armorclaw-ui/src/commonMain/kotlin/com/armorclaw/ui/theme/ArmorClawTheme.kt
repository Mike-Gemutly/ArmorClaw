package com.armorclaw.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.runtime.Composable

/**
 * ArmorClaw unified theme.
 * Dark mode is the only default experience.
 * Light mode is not supported per design constraints.
 */
@Composable
fun ArmorClawTheme(
    // Always use dark theme - light mode is hidden behind build flag
    darkTheme: Boolean = true,
    content: @Composable () -> Unit
) {
    val colorScheme = ArmorClawDarkColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        typography = ArmorClawTypography,
        shapes = ArmorClawShapes,
        content = content
    )
}

/**
 * Provides easy access to theme values.
 */
object AppTheme {
    val colors: ColorScheme
        @Composable get() = MaterialTheme.colorScheme

    val typography: Typography
        @Composable get() = MaterialTheme.typography

    val shapes: Shapes
        @Composable get() = MaterialTheme.shapes
}
