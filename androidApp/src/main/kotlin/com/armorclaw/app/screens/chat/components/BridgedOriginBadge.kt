package com.armorclaw.app.screens.chat.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.domain.model.BridgePlatform

/**
 * Small platform-origin badge overlaid on a user avatar
 *
 * Displays a tiny icon (emoji) indicating the external platform
 * that a bridged "ghost user" originates from (Slack, Discord, etc.).
 *
 * Usage:
 * ```
 * Box {
 *     Avatar(...)
 *     BridgedOriginBadge(
 *         platform = sender.bridgePlatform,
 *         modifier = Modifier.align(Alignment.BottomEnd)
 *     )
 * }
 * ```
 */
@Composable
fun BridgedOriginBadge(
    platform: BridgePlatform?,
    modifier: Modifier = Modifier
) {
    if (platform == null || platform == BridgePlatform.MATRIX_NATIVE) return

    val (emoji, backgroundColor, accessibilityLabel) = when (platform) {
        BridgePlatform.SLACK -> Triple("💬", Color(0xFF4A154B), "Slack user")
        BridgePlatform.DISCORD -> Triple("🎮", Color(0xFF5865F2), "Discord user")
        BridgePlatform.TEAMS -> Triple("🟦", Color(0xFF6264A7), "Microsoft Teams user")
        BridgePlatform.WHATSAPP -> Triple("📱", Color(0xFF25D366), "WhatsApp user")
        BridgePlatform.TELEGRAM -> Triple("✈️", Color(0xFF0088CC), "Telegram user")
        BridgePlatform.SIGNAL -> Triple("🔒", Color(0xFF3A76F0), "Signal user")
        BridgePlatform.MATRIX_NATIVE -> return // No badge for native users
    }

    Box(
        contentAlignment = Alignment.Center,
        modifier = modifier
            .size(18.dp)
            .offset(x = 2.dp, y = 2.dp)
            .clip(CircleShape)
            .background(backgroundColor)
            .semantics { contentDescription = accessibilityLabel }
    ) {
        Text(
            text = emoji,
            fontSize = 10.sp
        )
    }
}

/**
 * Returns a human-readable platform name for display in tooltips or labels
 */
fun BridgePlatform.displayName(): String = when (this) {
    BridgePlatform.SLACK -> "Slack"
    BridgePlatform.DISCORD -> "Discord"
    BridgePlatform.TEAMS -> "Microsoft Teams"
    BridgePlatform.WHATSAPP -> "WhatsApp"
    BridgePlatform.TELEGRAM -> "Telegram"
    BridgePlatform.SIGNAL -> "Signal"
    BridgePlatform.MATRIX_NATIVE -> "Matrix"
}
