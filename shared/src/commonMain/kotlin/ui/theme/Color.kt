package com.armorclaw.shared.ui.theme

import androidx.compose.ui.graphics.Color

// Primary colors (Brand)
val Primary = Color(0xFF6750A4)
val PrimaryContainer = Color(0xFFEADDFF)
val OnPrimary = Color(0xFFFFFFFF)
val OnPrimaryContainer = Color(0xFF21005D)

// Secondary colors
val Secondary = Color(0xFF625B71)
val SecondaryContainer = Color(0xFFE8DEF8)
val OnSecondary = Color(0xFFFFFFFF)
val OnSecondaryContainer = Color(0xFF1D192B)

// Tertiary colors
val Tertiary = Color(0xFF7D5260)
val TertiaryContainer = Color(0xFFFFD8E4)
val OnTertiary = Color(0xFFFFFFFF)
val OnTertiaryContainer = Color(0xFF31111D)

// Error colors
val Error = Color(0xFFB3261E)
val ErrorContainer = Color(0xFFF9DEDC)
val OnError = Color(0xFFFFFFFF)
val OnErrorContainer = Color(0xFF410E0B)

// Background colors
val Background = Color(0xFFFFFBFE)
val OnBackground = Color(0xFF1C1B1F)
val Surface = Color(0xFFFFFBFE)
val OnSurface = Color(0xFF1C1B1F)
val SurfaceVariant = Color(0xFFE7E0EC)
val OnSurfaceVariant = Color(0xFF49454F)

// Outline colors
val Outline = Color(0xFF79747E)
val OutlineVariant = Color(0xFFCAC4D0)

// Custom brand colors
val BrandPurple = Color(0xFF8B5CF6)
val BrandPurpleLight = Color(0xFFC4B5FD)
val BrandPurpleDark = Color(0xFF7C3AED)

val BrandGreen = Color(0xFF10B981)
val BrandGreenLight = Color(0xFF34D399)
val BrandGreenDark = Color(0xFF059669)

val BrandRed = Color(0xFFEF4444)
val BrandRedLight = Color(0xFFF87171)
val BrandRedDark = Color(0xFFDC2626)

val BrandBlue = Color(0xFF3B82F6)
val BrandBlueLight = Color(0xFF60A5FA)
val BrandBlueDark = Color(0xFF2563EB)

val BrandYellow = Color(0xFFF59E0B)
val BrandYellowLight = Color(0xFFFBBF24)
val BrandYellowDark = Color(0xFFD97706)

// Status colors
val StatusSuccess = BrandGreen
val StatusWarning = Color(0xFFF59E0B)
val StatusError = BrandRed
val StatusInfo = BrandBlue

// Sync state colors
val SyncOnline = BrandGreen
val SyncOffline = Color(0xFF6B7280)
val SyncSyncing = BrandBlue
val SyncError = BrandRed

// Legacy aliases for backwards compatibility
val AccentColor = BrandPurple
val SurfaceColor = Surface

// Dark mode colors
val PrimaryDark = Color(0xFFD0BCFF)
val PrimaryContainerDark = Color(0xFF4F378B)
val SecondaryDark = Color(0xFFCCC2DC)
val SecondaryContainerDark = Color(0xFF332D41)
val TertiaryDark = Color(0xFFEFB8C8)
val TertiaryContainerDark = Color(0xFF492532)
val ErrorDark = Color(0xFFF2B8B5)
val ErrorContainerDark = Color(0xFF8C1D18)
val BackgroundDark = Color(0xFF1C1B1F)
val SurfaceDark = Color(0xFF1C1B1F)
val SurfaceVariantDark = Color(0xFF49454F)
val OnSurfaceVariantDark = Color(0xFFCAC4D0)
val OutlineDark = Color(0xFF938F99)
val OutlineVariantDark = Color(0xFF49454F)

// Transparent colors
val Transparent10 = Color(0x1A000000)
val Transparent20 = Color(0x33000000)
val Transparent30 = Color(0x4D000000)
val Transparent40 = Color(0x66000000)
val Transparent50 = Color(0x80000000)
val Transparent60 = Color(0x99000000)
val Transparent70 = Color(0xCC000000)
val Transparent80 = Color(0xE6000000)
val Transparent90 = Color(0xF2000000)

object ArmorClawColors {
    // Brand
    val brandPrimary = BrandPurple
    val brandPrimaryLight = BrandPurpleLight
    val brandPrimaryDark = BrandPurpleDark
    
    // Status
    val success = StatusSuccess
    val warning = StatusWarning
    val error = StatusError
    val info = StatusInfo
    
    // Sync
    val syncOnline = SyncOnline
    val syncOffline = SyncOffline
    val syncSyncing = SyncSyncing
    val syncError = SyncError
}
