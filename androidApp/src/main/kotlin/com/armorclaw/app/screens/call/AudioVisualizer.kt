package com.armorclaw.app.screens.call
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.*
import kotlinx.coroutines.delay

/**
 * Animated audio level visualizer for voice calls
 */
@Composable
fun AudioVisualizer(
    audioLevel: Float,
    isActive: Boolean,
    modifier: Modifier = Modifier,
    barCount: Int = 5,
    color: Color = BrandPurple
) {
    // Generate animated levels for each bar
    val barLevels = remember { mutableStateListOf<Float>() }
    repeat(barCount) { barLevels.add(0.3f) }

    LaunchedEffect(isActive, audioLevel) {
        if (isActive) {
            while (isActive) {
                // Update each bar with randomized values based on audio level
                repeat(barCount) { index ->
                    val baseLevel = audioLevel.coerceIn(0f, 1f)
                    val randomFactor = (0.3f..1.0f).random()
                    val targetLevel = baseLevel * randomFactor

                    // Animate to new level
                    animateBarLevel(
                        currentLevel = barLevels[index],
                        targetLevel = targetLevel
                    ).let { animatedLevel ->
                        barLevels[index] = animatedLevel
                    }
                }
                delay(100)
            }
        } else {
            // Reset bars when not active
            repeat(barCount) { barLevels[it] = 0.2f }
        }
    }

    Row(
        modifier = modifier
            .height(80.dp)
            .padding(horizontal = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        barLevels.forEachIndexed { index, level ->
            AudioBar(
                level = if (isActive) level else 0.2f,
                color = color,
                delay = index * 50L,
                modifier = Modifier.weight(1f)
            )
        }
    }
}

@Composable
private fun AudioBar(
    level: Float,
    color: Color,
    delay: Long,
    modifier: Modifier = Modifier
) {
    var animatedLevel by remember { mutableStateOf(0.2f) }

    LaunchedEffect(level) {
        delay(delay)
        animatedLevel = level
    }

    val animatedHeight by animateFloatAsState(
        targetValue = animatedLevel.coerceIn(0.1f, 1f),
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioMediumBouncy,
            stiffness = Spring.StiffnessLow
        ),
        label = "height"
    )

    val animatedColor by animateColorAsState(
        targetValue = when {
            animatedLevel > 0.8f -> StatusWarning
            animatedLevel > 0.5f -> color
            else -> color.copy(alpha = 0.6f)
        },
        label = "color"
    )

    Box(
        modifier = modifier
            .fillMaxHeight(animatedHeight)
            .clip(RoundedCornerShape(4.dp))
            .background(animatedColor)
    )
}

private fun animateBarLevel(currentLevel: Float, targetLevel: Float): Float {
    // Smooth transition towards target
    val diff = targetLevel - currentLevel
    return currentLevel + diff * 0.3f
}

/**
 * Circular audio visualizer variant
 */
@Composable
fun CircularAudioVisualizer(
    audioLevel: Float,
    isActive: Boolean,
    modifier: Modifier = Modifier,
    color: Color = BrandPurple
) {
    var animatedLevel by remember { mutableStateOf(0f) }

    LaunchedEffect(isActive, audioLevel) {
        if (isActive) {
            while (isActive) {
                val randomFactor = (0.7f..1.0f).random()
                animatedLevel = (audioLevel * randomFactor).coerceIn(0f, 1f)
                delay(100)
            }
        } else {
            animatedLevel = 0f
        }
    }

    val animatedScale by animateFloatAsState(
        targetValue = 1f + animatedLevel * 0.3f,
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioMediumBouncy,
            stiffness = Spring.StiffnessLow
        ),
        label = "scale"
    )

    val animatedAlpha by animateFloatAsState(
        targetValue = if (isActive) 0.3f + animatedLevel * 0.4f else 0.1f,
        label = "alpha"
    )

    Box(
        modifier = modifier,
        contentAlignment = Alignment.Center
    ) {
        // Outer pulse ring
        Box(
            modifier = Modifier
                .matchParentSize()
                .scale(animatedScale)
                .clip(RoundedCornerShape(50))
                .background(color.copy(alpha = animatedAlpha * 0.5f))
        )

        // Middle ring
        Box(
            modifier = Modifier
                .matchParentSize()
                .scale(0.8f + animatedLevel * 0.1f)
                .clip(RoundedCornerShape(50))
                .background(color.copy(alpha = animatedAlpha))
        )

        // Inner circle
        Box(
            modifier = Modifier
                .fillMaxSize(0.6f)
                .clip(RoundedCornerShape(50))
                .background(color)
        )
    }
}

/**
 * Waveform-style audio visualizer
 */
@Composable
fun WaveformVisualizer(
    audioLevel: Float,
    isActive: Boolean,
    modifier: Modifier = Modifier,
    waveCount: Int = 20,
    color: Color = BrandPurple
) {
    val waveHeights = remember { mutableStateListOf<Float>() }
    repeat(waveCount) { waveHeights.add(0.2f) }

    LaunchedEffect(isActive, audioLevel) {
        if (isActive) {
            while (isActive) {
                repeat(waveCount) { index ->
                    val centerIndex = waveCount / 2
                    val distanceFromCenter = kotlin.math.abs(index - centerIndex)
                    val centerWeight = 1f - (distanceFromCenter.toFloat() / centerIndex)

                    val baseLevel = audioLevel.coerceIn(0f, 1f)
                    val randomFactor = (0.3f..1.0f).random()
                    val targetLevel = baseLevel * randomFactor * centerWeight

                    waveHeights[index] = 0.2f + targetLevel * 0.8f
                }
                delay(80)
            }
        } else {
            repeat(waveCount) { waveHeights[it] = 0.2f }
        }
    }

    Row(
        modifier = modifier
            .height(60.dp)
            .padding(horizontal = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(2.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        waveHeights.forEachIndexed { index, height ->
            WaveformBar(
                height = if (isActive) height else 0.2f,
                color = color,
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
            )
        }
    }
}

@Composable
private fun WaveformBar(
    height: Float,
    color: Color,
    modifier: Modifier = Modifier
) {
    var animatedHeight by remember { mutableStateOf(0.2f) }

    LaunchedEffect(height) {
        animatedHeight = height
    }

    val animatedHeightState by animateFloatAsState(
        targetValue = animatedHeight.coerceIn(0.1f, 1f),
        animationSpec = tween(
            durationMillis = 80,
            easing = FastOutSlowInEasing
        ),
        label = "height"
    )

    Box(
        modifier = modifier
            .fillMaxHeight(animatedHeightState)
            .padding(vertical = 4.dp)
            .clip(RoundedCornerShape(2.dp))
            .background(color.copy(alpha = 0.6f + animatedHeightState * 0.4f))
    )
}

/**
 * Speaking indicator that pulses when audio is detected
 */
@Composable
fun SpeakingIndicator(
    isSpeaking: Boolean,
    modifier: Modifier = Modifier,
    color: Color = BrandGreen
) {
    val infiniteTransition = rememberInfiniteTransition(label = "speaking")

    val pulseScale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = if (isSpeaking) 1.2f else 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    val pulseAlpha by infiniteTransition.animateFloat(
        initialValue = 0.3f,
        targetValue = if (isSpeaking) 0.8f else 0.3f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Box(
        modifier = modifier
            .scale(if (isSpeaking) pulseScale else 1f)
            .background(color.copy(alpha = pulseAlpha), RoundedCornerShape(4.dp))
    )
}

// Extension function for random range
private fun ClosedRange<Float>.random(): Float {
    return (Math.random() * (endInclusive - start) + start).toFloat()
}
