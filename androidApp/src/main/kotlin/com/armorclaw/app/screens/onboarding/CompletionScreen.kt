package com.armorclaw.app.screens.onboarding

import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.unit.IntOffset
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.ArrowForward
import androidx.compose.material.icons.filled.Book
import androidx.compose.material.icons.filled.Chat
import androidx.compose.material.icons.filled.Create
import androidx.compose.material.icons.filled.Star
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurpleLight
import com.armorclaw.shared.ui.theme.DesignTokens
import kotlinx.coroutines.delay

@Composable
fun CompletionScreen(
    onStartChatting: () -> Unit,
    onTakeTutorial: () -> Unit
) {
    var showAnimation by remember { mutableStateOf(false) }
    
    LaunchedEffect(Unit) {
        delay(300)
        showAnimation = true
    }
    
    val scale = remember { Animatable(0f) }
    val rotation = remember { Animatable(0f) }
    
    LaunchedEffect(showAnimation) {
        if (showAnimation) {
            scale.animateTo(
                targetValue = 1f,
                animationSpec = tween(
                    durationMillis = 800,
                    easing = androidx.compose.animation.core.FastOutSlowInEasing
                )
            )
            rotation.animateTo(
                targetValue = 360f,
                animationSpec = tween(durationMillis = 1200)
            )
        }
    }
    
    ArmorClawTheme {
        Scaffold { paddingValues ->
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(
                        brush = Brush.verticalGradient(
                            colors = listOf(
                                BrandPurpleLight.copy(alpha = 0.3f),
                                MaterialTheme.colorScheme.background
                            ),
                            startY = 0f,
                            endY = Float.POSITIVE_INFINITY
                        )
                    )
                    .padding(paddingValues)
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(DesignTokens.Spacing.lg)
                        .verticalScroll(rememberScrollState()),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
                    
                    // Celebration animation
                    Box(
                        modifier = Modifier
                            .size(200.dp)
                            .scale(scale.value),
                        contentAlignment = Alignment.Center
                    ) {
                        // Animated circle with check
                        Box(
                            modifier = Modifier
                                .size(160.dp)
                                .clip(CircleShape)
                                .background(BrandGreen)
                                .rotate(rotation.value),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.Check,
                                contentDescription = null,
                                tint = Color.White,
                                modifier = Modifier.size(80.dp)
                            )
                        }
                        
                        // Confetti particles (decorative)
                        ConfettiParticles(showAnimation = showAnimation)
                    }
                    
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
                    
                    // Success message
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = "🔐 Vault Initialized",
                            style = MaterialTheme.typography.displaySmall.copy(
                                fontWeight = FontWeight.Bold
                            ),
                            color = BrandGreen
                        )

                        Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

                        Text(
                            text = "Your encryption keys are secured. You're ready to supervise your AI agents!",
                            style = MaterialTheme.typography.titleMedium,
                            textAlign = TextAlign.Center,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
                        )
                    }
                    
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
                    
                    // What's next card
                    WhatsNextCard()
                    
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))
                    
                    // Quick tip card
                    QuickTipCard()
                    
                    Spacer(modifier = Modifier.weight(1f))
                    
                    // Actions
                    Column(
                        modifier = Modifier.fillMaxWidth(),
                        verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                    ) {
                        Button(
                            onClick = onStartChatting,
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(56.dp),
                        ) {
                            Row(
                                horizontalArrangement = Arrangement.Center,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = "Go to Mission Control →",
                                    style = MaterialTheme.typography.labelLarge.copy(
                                        fontWeight = FontWeight.Bold
                                    )
                                )
                                Spacer(modifier = Modifier.width(8.dp))
                                Icon(
                                    imageVector = Icons.Default.ArrowForward,
                                    contentDescription = null
                                )
                            }
                        }
                        
                        TextButton(
                            onClick = onTakeTutorial,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Text(
                                text = "Take the Tutorial",
                                style = MaterialTheme.typography.labelLarge
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ConfettiParticles(showAnimation: Boolean) {
    val positions = remember {
        listOf(
            Offset(0f, 0f),
            Offset(200f, 0f),
            Offset(0f, 200f),
            Offset(200f, 200f),
            Offset(100f, 0f),
            Offset(0f, 100f),
            Offset(200f, 100f),
            Offset(100f, 200f)
        )
    }
    
    val colors = listOf(
        BrandPurple,
        BrandGreen,
                        BrandPurpleLight.copy(alpha = 0.5f)
    )
    
    positions.forEachIndexed { index, offset ->
        val delay = index * 100
        val animatedAlpha = remember { Animatable(0f) }
        
        LaunchedEffect(showAnimation, delay) {
            if (showAnimation) {
                animatedAlpha.animateTo(
                    targetValue = 0.8f,
                    animationSpec = tween(durationMillis = 600)
                )
                delay(1000)
                animatedAlpha.animateTo(
                    targetValue = 0f,
                    animationSpec = tween(durationMillis = 400)
                )
            }
        }
        
        Box(
            modifier = Modifier
                .size(12.dp)
                .offset { IntOffset(offset.x.toInt(), offset.y.toInt()) }
                .alpha(animatedAlpha.value)
                .clip(CircleShape)
                .background(colors[index % colors.size])
        )
    }
}

@Composable
private fun WhatsNextCard() {
    Card(
        modifier = Modifier.fillMaxWidth(),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.lg)
        ) {
            Text(
                text = "What's Next:",
                style = MaterialTheme.typography.titleMedium.copy(
                    fontWeight = FontWeight.Bold
                ),
                color = BrandPurple
            )
            
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
            
            NextStepItem(
                icon = Icons.Default.Chat,
                text = "1. View your Mission Control"
            )
            NextStepItem(
                icon = Icons.Default.Star,
                text = "2. Monitor active agents"
            )
            NextStepItem(
                icon = Icons.Default.Create,
                text = "3. Approve or deny PII requests"
            )
        }
    }
}

@Composable
private fun NextStepItem(
    icon: ImageVector,
    text: String
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = DesignTokens.Spacing.xs),
        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Box(
            modifier = Modifier
                .size(32.dp)
                .clip(CircleShape)
                .background(BrandPurple.copy(alpha = 0.2f)),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(20.dp)
            )
        }
        
        Text(
            text = text,
            style = MaterialTheme.typography.bodyLarge
        )
    }
}

@Composable
private fun QuickTipCard() {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = BrandPurple.copy(alpha = 0.1f)
        ),
        border = androidx.compose.foundation.BorderStroke(
            1.dp,
            BrandPurple.copy(alpha = 0.3f)
        )
    ) {
        Row(
            modifier = Modifier.padding(DesignTokens.Spacing.md),
            horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Book,
                contentDescription = null,
                tint = BrandPurple,
                modifier = Modifier.size(32.dp)
            )
            
            Column {
                Text(
                    text = "Quick Tip:",
                    style = MaterialTheme.typography.titleSmall.copy(
                        fontWeight = FontWeight.Bold
                    ),
                    color = BrandPurple
                )
                Text(
                    text = "Swipe down on Mission Control to see all agent activity",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
                )
            }
        }
    }
}
