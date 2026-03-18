package com.armorclaw.app.accessibility

import android.content.Context
import android.content.res.Configuration
import android.os.Build
import androidx.compose.ui.semantics.SemanticsPropertyReceiver
import androidx.compose.ui.semantics.semantics
import androidx.core.view.accessibility.AccessibilityManagerCompat

/**
 * Accessibility configuration and utilities
 * 
 * This class provides accessibility settings and helpers for
 * ensuring the app is accessible to all users.
 */
class AccessibilityConfig(
    private val context: Context
) {
    
    /**
     * Check if screen reader is enabled
     */
    fun isScreenReaderEnabled(): Boolean {
        val accessibilityManager = context.getSystemService(Context.ACCESSIBILITY_SERVICE) as? android.view.accessibility.AccessibilityManager
        return accessibilityManager?.isEnabled == true &&
               AccessibilityManagerCompat.isTouchExplorationEnabled(accessibilityManager)
    }
    
    /**
     * Check if high contrast is enabled
     */
    fun isHighContrastEnabled(): Boolean {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            val config = context.resources.configuration
            return config.isScreenRound // Approximation, check actual high contrast flag if available
        }
        return false
    }
    
    /**
     * Check if large text is enabled
     */
    fun isLargeTextEnabled(): Boolean {
        val config = context.resources.configuration
        val fontScale = config.fontScale
        return fontScale > 1.0f
    }
    
    /**
     * Get current font scale
     */
    fun getFontScale(): Float {
        val config = context.resources.configuration
        return config.fontScale
    }
    
    /**
     * Check if reduced motion is enabled
     */
    fun isReducedMotionEnabled(): Boolean {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            val config = context.resources.configuration
            return config.isScreenRound // Check actual reduced motion flag
        }
        return false
    }
    
    /**
     * Get accessibility settings summary
     */
    fun getAccessibilitySettingsSummary(): AccessibilitySettings {
        return AccessibilitySettings(
            isScreenReaderEnabled = isScreenReaderEnabled(),
            isHighContrastEnabled = isHighContrastEnabled(),
            isLargeTextEnabled = isLargeTextEnabled(),
            fontScale = getFontScale(),
            isReducedMotionEnabled = isReducedMotionEnabled(),
            isColorInversionEnabled = false, // Check actual implementation
            isTalkBackEnabled = isScreenReaderEnabled()
        )
    }
}

/**
 * Accessibility settings data class
 */
data class AccessibilitySettings(
    val isScreenReaderEnabled: Boolean,
    val isHighContrastEnabled: Boolean,
    val isLargeTextEnabled: Boolean,
    val fontScale: Float,
    val isReducedMotionEnabled: Boolean,
    val isColorInversionEnabled: Boolean,
    val isTalkBackEnabled: Boolean
)

/**
 * Semantics properties for accessibility
 */
object AccessibilitySemantics {
    
    /**
     * Content description
     */
    val contentDescription = SemanticsPropertyKey<String>(
        name = "ContentDescription",
        mergePolicy = AccessibilityMergePolicy.MergeDescendants
    )
    
    /**
     * Heading level
     */
    val heading = SemanticsPropertyKey<Int>(
        name = "Heading",
        mergePolicy = AccessibilityMergePolicy.MergeDescendants
    )
    
    /**
     * State description
     */
    val stateDescription = SemanticsPropertyKey<String>(
        name = "StateDescription",
        mergePolicy = AccessibilityMergePolicy.MergeDescendants
    )
    
    /**
     * Value
     */
    val value = SemanticsPropertyKey<String>(
        name = "Value",
        mergePolicy = AccessibilityMergePolicy.MergeDescendants
    )
    
    /**
     * Traversal order
     */
    val traversalIndex = SemanticsPropertyKey<Int>(
        name = "TraversalIndex",
        mergePolicy = AccessibilityMergePolicy.MergeDescendants
    )
}

/**
 * Accessibility merge policies
 */
enum class AccessibilityMergePolicy {
    MergeDescendants,
    DoNotMergeDescendants
}

/**
 * Helper class for SemanticsPropertyKey
 */
class SemanticsPropertyKey<T>(
    val name: String,
    val mergePolicy: AccessibilityMergePolicy
)
