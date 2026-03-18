package com.armorclaw.app.screens.chat.components

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandPurpleLight

data class TypingIndicator(
    val isTyping: Boolean,
    val typers: List<String> = emptyList()
)

@Composable
fun TypingIndicatorComponent(
    indicator: TypingIndicator,
    modifier: Modifier = Modifier
) {
    if (!indicator.isTyping) return
    
    Surface(
        modifier = modifier
            .padding(horizontal = 16.dp, vertical = 8.dp),
        shape = CircleShape,
        color = BrandPurpleLight.copy(alpha = 0.3f),
        tonalElevation = 2.dp,
        shadowElevation = 2.dp
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Animated dots
            TypingDots()
            
            // Text
            Text(
                text = when {
                    indicator.typers.isEmpty() -> "typing..."
                    indicator.typers.size == 1 -> "${indicator.typers[0]} is typing..."
                    indicator.typers.size == 2 -> "${indicator.typers[0]} and ${indicator.typers[1]} are typing..."
                    else -> "${indicator.typers.size} people are typing..."
                },
                style = MaterialTheme.typography.bodySmall.copy(
                    fontWeight = FontWeight.Medium,
                    color = BrandPurple
                )
            )
        }
    }
}

@Composable
private fun TypingDots() {
    val infiniteTransition = rememberInfiniteTransition(label = "typing_dots")
    
    val dot1Scale by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(
                durationMillis = 600,
                easing = LinearEasing
            ),
            repeatMode = RepeatMode.Reverse
        ),
        label = "dot1_scale"
    )
    
    val dot2Scale by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(
                durationMillis = 600,
                delayMillis = 200,
                easing = LinearEasing
            ),
            repeatMode = RepeatMode.Reverse
        ),
        label = "dot2_scale"
    )
    
    val dot3Scale by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(
                durationMillis = 600,
                delayMillis = 400,
                easing = LinearEasing
            ),
            repeatMode = RepeatMode.Reverse
        ),
        label = "dot3_scale"
    )
    
    Row(
        horizontalArrangement = Arrangement.spacedBy(2.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        TypingDot(scale = dot1Scale)
        TypingDot(scale = dot2Scale)
        TypingDot(scale = dot3Scale)
    }
}

@Composable
private fun TypingDot(scale: Float) {
    Box(
        modifier = Modifier
            .size(6.dp)
            .background(
                color = BrandPurple,
                shape = CircleShape
            ),
        contentAlignment = Alignment.Center
    ) {}
}
