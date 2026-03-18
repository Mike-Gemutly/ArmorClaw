package com.armorclaw.shared.ui.components.tutorial

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.AnimationSpec
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.boundsInWindow
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.layout.positionInWindow
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Info
import com.armorclaw.shared.domain.features.TutorialStep
import com.armorclaw.shared.domain.features.TutorialAction
import com.armorclaw.shared.domain.features.TutorialAction.Highlight
import com.armorclaw.shared.domain.features.TutorialAction.Navigate
import com.armorclaw.shared.domain.features.TutorialAction.ShowMessage
import com.armorclaw.shared.domain.features.TutorialAction.Wait

/**
 * Coachmark overlay component for in-app tutorials
 *
 * This component displays a tutorial overlay with a highlighted area and tutorial content.
 * It supports positioning around different UI elements and includes proper Material 3 styling.
 *
 * @param tutorialStep The current tutorial step to display
 * @param targetElementBounds The bounds of the element to highlight
 * @param onDismiss Callback when the tutorial is dismissed
 * @param onNavigate Callback for navigation actions
 * @param modifier Modifier for the coachmark
 */
@Composable
fun CoachmarkOverlay(
    tutorialStep: TutorialStep,
    targetElementBounds: Rect,
    onDismiss: () -> Unit,
    onNavigate: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    val density = LocalDensity.current
    var overlayPosition by remember { mutableStateOf(IntOffset.Zero) }
    var overlaySize by remember { mutableStateOf(0.dp) }
    
    // Calculate overlay position and size based on target element
    val (position, size) = calculateOverlayPosition(
        targetBounds = targetElementBounds,
        density = density,
        padding = 16.dp
    )
    
    overlayPosition = position
    overlaySize = size
    
    // Animate the overlay appearance
    val animatedAlpha = remember { Animatable(0f) }
    LaunchedEffect(Unit) {
        animatedAlpha.animateTo(1f, animationSpec = tween(durationMillis = 300))
    }
    
    Surface(
        modifier = modifier
            .fillMaxSize()
            .clickable(onClick = onDismiss)
            .graphicsLayer(alpha = animatedAlpha.value)
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            // Background overlay
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.Black.copy(alpha = 0.5f))
            )
            
            // Highlighted area around target element
            Box(
                modifier = Modifier
                    .offset {
                        IntOffset(
                            x = (targetElementBounds.left * density.density).toInt(),
                            y = (targetElementBounds.top * density.density).toInt()
                        )
                    }
                    .size(
                        width = (targetElementBounds.width * density.density).dp,
                        height = (targetElementBounds.height * density.density).dp
                    )
                    .graphicsLayer {
                        shadowElevation = 8.dp.toPx()
                        shape = RoundedCornerShape(8.dp)
                        clip = true
                    }
                    .background(Color(0xFF6366F1).copy(alpha = 0.3f))
            )
            
            // Tutorial content card
            Card(
                modifier = Modifier
                    .offset {
                        IntOffset(
                            x = (overlayPosition.x).toInt(),
                            y = (overlayPosition.y).toInt()
                        )
                    }
                    .width(overlaySize)
                    .shadow(elevation = 16.dp, shape = RoundedCornerShape(16.dp)),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surface
                )
            ) {
                Column(
                    modifier = Modifier.padding(16.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
// Tutorial icon
Icon(
    imageVector = Icons.Filled.Info,
    contentDescription = "Tutorial",
    modifier = Modifier.size(48.dp),
    tint = MaterialTheme.colorScheme.primary
)
                    
                    Spacer(modifier = Modifier.height(16.dp))
                    
                    // Title
                    Text(
                        text = tutorialStep.title,
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.onSurface,
                        textAlign = TextAlign.Center
                    )
                    
                    Spacer(modifier = Modifier.height(8.dp))
                    
                    // Description
                    Text(
                        text = tutorialStep.description,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.padding(horizontal = 16.dp)
                    )
                    
                    Spacer(modifier = Modifier.height(24.dp))
                    
                    // Action button
                    when (tutorialStep.action) {
                        is ShowMessage -> {
                            TutorialActionButton(
                                text = "Next",
                                onClick = onDismiss
                            )
                        }
                        is Navigate -> {
                            TutorialActionButton(
                                text = "Continue",
                                onClick = { onNavigate(tutorialStep.action.route) }
                            )
                        }
                        is Highlight -> {
                            TutorialActionButton(
                                text = "Got it",
                                onClick = onDismiss
                            )
                        }
                        is Wait -> {
                            TutorialActionButton(
                                text = "Wait",
                                onClick = onDismiss
                            )
                        }
                    }
                }
            }
        }
    }
}

/**
 * Tutorial action button with Material 3 styling
 */
@Composable
private fun TutorialActionButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    com.armorclaw.shared.ui.components.atom.ArmorClawButton(
        text = text,
        onClick = onClick,
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        enabled = true,
        loading = false
    )
}

/**
 * Calculates the optimal position and size for the tutorial overlay
 */
private fun calculateOverlayPosition(
    targetBounds: Rect,
    density: androidx.compose.ui.unit.Density,
    padding: Dp
): Pair<IntOffset, Dp> {
    val screenWidth = 400.dp // Approximate screen width for positioning
    val screenHeight = 200.dp // Approximate screen height for positioning
    
    // Position the overlay below the target element
    val x = ((targetBounds.left + targetBounds.right) / 2 - screenWidth.value / 2) * density.density
    val y = (targetBounds.bottom + padding.value) * density.density
    
    return Pair(
        IntOffset(x.toInt(), y.toInt()),
        screenWidth
    )
}

/**
 * Modifier to track the bounds of a composable element for highlighting
 */
fun Modifier.trackBoundsForHighlight(
    onBoundsCalculated: (Rect) -> Unit
): Modifier = this.then(
    Modifier.onGloballyPositioned { coordinates ->
        val bounds = coordinates.boundsInWindow()
        onBoundsCalculated(bounds)
    }
)