package theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.graphics.Color

/**
 * ArmorClaw Brand Colors
 *
 * Core brand palette derived from the Governor Strategy design system.
 * Teal (#00BCD4) as the primary accent color representing trust and security.
 */

// Primary Brand Colors
val ArmorTeal = Color(0xFF00BCD4)
val ArmorTealDark = Color(0xFF0097A7)
val ArmorTealLight = Color(0xFF4DD0E1)

// Secondary Colors
val ArmorNavy = Color(0xFF1A1A2E)
val ArmorNavyLight = Color(0xFF252540)
val ArmorSlate = Color(0xFF4A4A6A)

// Status Colors
val ArmorSuccess = Color(0xFF3FB950)
val ArmorWarning = Color(0xFFFFC107)
val ArmorError = Color(0xFFF85149)
val ArmorInfo = Color(0xFF58A6FF)

// Neutral Colors
val ArmorSurface = Color(0xFF0D1117)
val ArmorSurfaceVariant = Color(0xFF161B22)
val ArmorBorder = Color(0xFF30363D)
val ArmorText = Color(0xFFE6EDF3)
val ArmorTextMuted = Color(0xFF8B949E)

// Light Theme Colors (for future use)
val ArmorTealLightTheme = Color(0xFF00BCD4)
val ArmorBackgroundLight = Color(0xFFF5F5F5)
val ArmorSurfaceLight = Color(0xFFFFFFFF)

/**
 * ArmorClaw Color Scheme - Dark Theme
 */
val ArmorClawDarkColorScheme = darkColorScheme(
    primary = ArmorTeal,
    onPrimary = Color.White,
    primaryContainer = ArmorTealDark,
    onPrimaryContainer = ArmorTealLight,
    
    secondary = ArmorSlate,
    onSecondary = Color.White,
    secondaryContainer = ArmorNavyLight,
    onSecondaryContainer = ArmorText,
    
    tertiary = ArmorInfo,
    onTertiary = Color.White,
    
    background = ArmorSurface,
    onBackground = ArmorText,
    
    surface = ArmorSurfaceVariant,
    onSurface = ArmorText,
    
    surfaceVariant = ArmorNavy,
    onSurfaceVariant = ArmorTextMuted,
    
    outline = ArmorBorder,
    outlineVariant = ArmorSlate,
    
    error = ArmorError,
    onError = Color.White,
    errorContainer = ArmorError.copy(alpha = 0.1f),
    onErrorContainer = ArmorError
)

/**
 * ArmorClaw Theme
 *
 * Wraps Material3 theme with ArmorClaw brand colors.
 * Enforces consistent theming across all components.
 *
 * @param darkTheme Whether to use dark theme (defaults to system setting)
 * @param dynamicColor Whether to use dynamic colors (disabled for brand consistency)
 * @param content Theme content
 */
@Composable
fun ArmorClawTheme(
    darkTheme: Boolean = true, // Always use dark theme for brand consistency
    dynamicColor: Boolean = false, // Disable dynamic colors for brand consistency
    content: @Composable () -> Unit
) {
    val colorScheme = ArmorClawDarkColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        typography = ArmorClawTypography,
        content = content
    )
}

/**
 * ArmorClaw Typography
 */
val ArmorClawTypography = Typography()

/**
 * ArmorClaw Shapes
 */
val ArmorClawShapes = Shapes()

/**
 * Extension properties for custom colors
 */

// Custom color extensions for Material3 ColorScheme
val ColorScheme.success: Color
    get() = ArmorSuccess

val ColorScheme.warning: Color
    get() = ArmorWarning

val ColorScheme.info: Color
    get() = ArmorInfo

val ColorScheme.teal: Color
    get() = ArmorTeal

val ColorScheme.navy: Color
    get() = ArmorNavy

val ColorScheme.surfaceDark: Color
    get() = ArmorSurface
