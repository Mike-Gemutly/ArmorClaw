package com.armorclaw.ui.theme

import androidx.compose.material3.darkColorScheme
import androidx.compose.ui.graphics.Color

// Primary brand colors (from Component Catch)
val Teal = Color(0xFF14F0C8)
val TealGlow = Color(0xFF67F5D8)
val Navy = Color(0xFF0A1428)
val NavyLight = Color(0xFF111827)
val NavyVariant = Color(0xFF1E293B)

// Semantic colors
val PrecisionBlue = Color(0xFF0EA5E9)
val SuccessGreen = Color(0xFF22C55E)
val WarningAmber = Color(0xFFF59E0B)
val ErrorRed = Color(0xFFEF4444)

// Text colors
val TextPrimary = Color(0xFFE2E8F0)
val TextSecondary = Color(0xFFCBD5E1)
val TextMuted = Color(0xFF94A3B8)

// Dark color scheme (default and only mode)
val ArmorClawDarkColorScheme = darkColorScheme(
    primary = Teal,
    onPrimary = Navy,
    primaryContainer = TealGlow,
    inversePrimary = Teal,
    secondary = PrecisionBlue,
    onSecondary = Color.White,
    tertiary = SuccessGreen,
    error = ErrorRed,
    onError = Color.White,
    background = Navy,
    onBackground = TextPrimary,
    surface = NavyLight,
    onSurface = TextPrimary,
    surfaceVariant = NavyVariant,
    onSurfaceVariant = TextSecondary,
    outline = Teal.copy(alpha = 0.3f),
    outlineVariant = TextMuted
)
