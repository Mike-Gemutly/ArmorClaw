package com.armorclaw.app.secretary.ui

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.*
import com.armorclaw.shared.secretary.SecretaryState

/**
 * SecretaryAvatar - UI component
 *
 * Simple visual indicator of current Secretary state.
 * Phase 1: Shows different icons based on state:
 * - Idle: Clock icon (gray)
 * - Observing: Eye icon (blue)
 * - Thinking: Brain icon (purple)
 * - Proposing: Lightbulb icon (orange)
 * - Executing: Progress indicator
 * - Error: Error icon (red)
 */
@Composable
fun SecretaryAvatar(
    state: SecretaryState,
    modifier: Modifier = Modifier
) {
    // Color and icon based on state
    val (icon, iconColor) = when (state) {
        is SecretaryState.Idle -> Pair(Icons.Filled.Clock, MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f))
        is SecretaryState.Observing -> Pair(Icons.Filled.Eye, MaterialTheme.colorScheme.primary.copy(alpha = 0.8f))
        is SecretaryState.Thinking -> Pair(Icons.Filled.Brain, MaterialTheme.colorScheme.tertiary.copy(alpha = 0.8f))
        is SecretaryState.Proposing -> Pair(Icons.Filled.Lightbulb, MaterialTheme.colorScheme.tertiary)
        is SecretaryState.Error -> Pair(Icons.Filled.Error, MaterialTheme.colorScheme.error)
        is SecretaryState.Executing -> Icons.Filled.Settings(
            Icons.Filled.AccountBox,
            MaterialTheme.colorScheme.tertiary.copy(alpha = 0.6f)
        )
        else -> Pair(Icons.Filled.QuestionMark, MaterialTheme.colorScheme.onSurface)
    }

    Box(
        modifier = modifier.size(40.dp),
        contentAlignment = Alignment.Center
    ) {
        Icon(
            imageVector = icon,
            tint = iconColor,
            contentDescription = "Secretary state indicator",
            modifier = Modifier.size(24.dp)
        )
    }
}
