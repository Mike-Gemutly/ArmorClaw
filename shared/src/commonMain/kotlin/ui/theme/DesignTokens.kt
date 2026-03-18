package com.armorclaw.shared.ui.theme

import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

object DesignTokens {
    // Spacing
    object Spacing {
        val xs = 2.dp
        val sm = 4.dp
        val md = 8.dp
        val lg = 16.dp
        val xl = 24.dp
        val xxl = 32.dp
        val xxxl = 48.dp
    }
    
    // Border Radius
    object Radius {
        val none = 0.dp
        val xs = 2.dp
        val sm = 4.dp
        val md = 8.dp
        val lg = 16.dp
        val xl = 24.dp
        val full = 9999.dp
    }
    
    // Elevation
    object Elevation {
        val none = 0.dp
        val xs = 1.dp
        val sm = 2.dp
        val md = 4.dp
        val lg = 8.dp
        val xl = 16.dp
    }
    
    // Typography
    object FontSize {
        val xs = 10.sp
        val sm = 12.sp
        val md = 14.sp
        val lg = 16.sp
        val xl = 18.sp
        val xxl = 20.sp
        val xxxl = 24.sp
        val display = 30.sp
    }
    
    // Icon sizes
    object Icon {
        val xs = 16.dp
        val sm = 20.dp
        val md = 24.dp
        val lg = 32.dp
        val xl = 48.dp
    }
    
    // Animation durations
    object Duration {
        val fast = 150
        val normal = 250
        val slow = 400
    }
    
    // Button sizes
    object Button {
        val minWidth = 64.dp
        val minHeight = 36.dp
        val smallHeight = 32.dp
        val largeHeight = 48.dp
        val iconSize = 20.dp
        val smallIconSize = 18.dp
        val largeIconSize = 24.dp
    }
    
    // Input field sizes
    object Input {
        val minHeight = 56.dp
        val smallMinHeight = 40.dp
        val iconSize = 24.dp
        val smallIconSize = 20.dp
    }
    
    // Card sizes
    object Card {
        val padding = 16.dp
        val smallPadding = 12.dp
        val elevation = 2.dp
    }
    
    // List item sizes
    object ListItem {
        val height = 56.dp
        val padding = 16.dp
        val iconSize = 24.dp
        val avatarSize = 40.dp
    }
    
    // Avatar sizes
    object Avatar {
        val xs = 24.dp
        val sm = 32.dp
        val md = 40.dp
        val lg = 56.dp
        val xl = 80.dp
    }
    
    // Message bubble sizes
    object Message {
        val maxWidth = 280.dp
        val padding = 12.dp
        val minHeight = 40.dp
    }
}
