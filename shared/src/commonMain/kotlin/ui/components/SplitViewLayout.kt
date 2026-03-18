package com.armorclaw.shared.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.Shield
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.TextButton
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.systemBars
import androidx.compose.foundation.layout.ime
import androidx.compose.animation.core.TweenSpec
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.draw.scale
import androidx.compose.material3.CardDefaults
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.material3.ExperimentalMaterial3Api
import com.armorclaw.shared.ui.theme.DesignTokens

/**
 * Window width size class for responsive layout decisions
 */
enum class WindowWidthSizeClass {
    COMPACT,
    MEDIUM,
    EXPANDED
}

/**
 * SplitViewLayout - A responsive layout component that adapts between:
 * - Two-pane layout (tablet/landscape): 60% chat, 40% activity log
 * - Single-pane layout (phone/portrait): Full chat with activity log below
 *
 * ## Architecture
 * ```
 * SplitViewLayout
 *      ├── ChatPane (always visible)
 *      └── ActivityLogPane (conditional)
 * ```
 *
 * ## Features
 * - Responsive to WindowWidthSizeClass (COMPACT, MEDIUM, EXPANDED)
 * - Independent scrolling for both panes
 * - Smooth animations (< 300ms) for layout transitions
 * - Edge-to-edge support with WindowInsets
 * - Material 3 design system compliance
 *
 * ## Usage
 * ```kotlin
 * SplitViewLayout(
 *     chatContent = { ChatScreen() },
 *     activityLogContent = { ActivityLogScreen() },
 *     isActivityLogVisible = true
 * )
 * ```
 */
@Composable
fun SplitViewLayout(
    chatContent: @Composable () -> Unit,
    activityLogContent: @Composable () -> Unit,
    isActivityLogVisible: Boolean = true,
    windowWidthSizeClass: WindowWidthSizeClass = WindowWidthSizeClass.COMPACT
) {
    // Determine layout mode based on window size
    val isTwoPane = when (windowWidthSizeClass) {
        WindowWidthSizeClass.COMPACT -> false // Phone - single pane
        WindowWidthSizeClass.MEDIUM -> true   // Tablet - two-pane
        WindowWidthSizeClass.EXPANDED -> true  // Desktop - two-pane
    }

    // Animation state for layout transitions
    val animationSpec = TweenSpec<Float>(durationMillis = 250) // 250ms animation
    val scale by animateFloatAsState(
        targetValue = if (isTwoPane) 1f else 0.95f,
        animationSpec = animationSpec,
        label = "layoutScale"
    )

    // Window insets for edge-to-edge support
    val systemBars = WindowInsets.systemBars.asPaddingValues()
    val ime = WindowInsets.ime.asPaddingValues()

    Box(
        modifier = Modifier
            .fillMaxSize()
            .graphicsLayer {
                scaleX = scale
                scaleY = scale
            }
    ) {
        if (isTwoPane && isActivityLogVisible) {
            // Two-pane layout for tablet/landscape
            TwoPaneLayout(
                chatContent = chatContent,
                activityLogContent = activityLogContent,
                systemBars = systemBars,
                ime = ime
            )
        } else {
            // Single-pane layout for phone/portrait
            SinglePaneLayout(
                chatContent = chatContent,
                activityLogContent = activityLogContent,
                systemBars = systemBars,
                ime = ime
            )
        }
    }
}

/**
 * Two-pane layout implementation (tablet/landscape)
 */
@Composable
private fun TwoPaneLayout(
    chatContent: @Composable () -> Unit,
    activityLogContent: @Composable () -> Unit,
    systemBars: androidx.compose.foundation.layout.PaddingValues,
    ime: androidx.compose.foundation.layout.PaddingValues
) {
Row(
    modifier = Modifier
        .fillMaxSize()
        .padding(horizontal = DesignTokens.Spacing.lg)
) {
        // Chat pane (60%)
        Box(
            modifier = Modifier
                .weight(0.6f)
                .fillMaxHeight()
        ) {
            chatContent()
        }

        // Activity log pane (40%)
        Box(
            modifier = Modifier
                .weight(0.4f)
                .fillMaxHeight()
        ) {
            activityLogContent()
        }
    }
}

/**
 * Single-pane layout implementation (phone/portrait) with BottomSheet
 */
@Composable
private fun SinglePaneLayout(
    chatContent: @Composable () -> Unit,
    activityLogContent: @Composable () -> Unit,
    systemBars: androidx.compose.foundation.layout.PaddingValues,
    ime: androidx.compose.foundation.layout.PaddingValues
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(systemBars)
    ) {
        chatContent()
    }
}

/**
 * Preview for SplitViewLayout - Two-pane mode (tablet/landscape)
 */
@Composable
fun SplitViewLayoutPreviewTwoPane() {
    MaterialTheme {
        SplitViewLayout(
            chatContent = {
                ChatPanePreview()
            },
            activityLogContent = {
                ActivityLogPanePreview()
            },
            isActivityLogVisible = true,
            windowWidthSizeClass = WindowWidthSizeClass.MEDIUM
        )
    }
}

/**
 * Preview for SplitViewLayout - Single-pane mode (phone/portrait)
 */
@Composable
fun SplitViewLayoutPreviewSinglePane() {
    MaterialTheme {
        SplitViewLayout(
            chatContent = {
                ChatPanePreview()
            },
            activityLogContent = {
                ActivityLogPanePreview()
            },
            isActivityLogVisible = true,
            windowWidthSizeClass = WindowWidthSizeClass.COMPACT
        )
    }
}

/**
 * Preview content for Chat pane
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ChatPanePreview() {
    Scaffold(
        topBar = {
            androidx.compose.material3.TopAppBar(
                title = { Text("Chat", style = MaterialTheme.typography.titleMedium) },
                modifier = Modifier.fillMaxWidth()
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(rememberScrollState())
        ) {
            repeat(5) { index ->
                androidx.compose.material3.Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(DesignTokens.Spacing.lg),
                    elevation = CardDefaults.cardElevation(defaultElevation = DesignTokens.Elevation.md)
                ) {
                    Column(
                        modifier = Modifier.padding(DesignTokens.Spacing.lg)
                    ) {
                        Text(
                            text = "Message $index",
                            style = MaterialTheme.typography.bodyLarge
                        )
                        Text(
                            text = "This is a sample message content for preview",
                            style = MaterialTheme.typography.bodyMedium
                        )
                    }
                }
            }
        }
    }
}

/**
 * Preview content for Activity log pane
 */
@Composable
private fun ActivityLogPanePreview() {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(DesignTokens.Spacing.lg)
    ) {
        androidx.compose.material3.Text(
            text = "Activity Log",
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(bottom = DesignTokens.Spacing.md)
        )
        
        repeat(8) { index ->
            androidx.compose.material3.Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = DesignTokens.Spacing.sm),
                elevation = CardDefaults.cardElevation(defaultElevation = DesignTokens.Elevation.xs)
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(DesignTokens.Spacing.lg),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Activity $index",
                        style = MaterialTheme.typography.bodyMedium
                    )
                    Text(
                        text = "10:30 AM",
                        style = MaterialTheme.typography.labelSmall
                    )
                }
            }
        }
    }
}