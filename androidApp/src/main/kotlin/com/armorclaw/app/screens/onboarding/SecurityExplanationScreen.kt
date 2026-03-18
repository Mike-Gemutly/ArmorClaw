package com.armorclaw.app.screens.onboarding

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Bolt
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.PhoneAndroid
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Shield
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandPurpleLight
import com.armorclaw.shared.ui.theme.DesignTokens

data class SecurityStep(
    val id: String,
    val title: String,
    val description: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector
)

@OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)
@Composable
fun SecurityExplanationScreen(
    initialStep: Int = 0,
    onNext: () -> Unit,
    onBack: () -> Unit,
    onSkipToHome: (() -> Unit)? = null,
    onUseDefaults: (() -> Unit)? = null
) {
    var selectedStep by remember { mutableStateOf(initialStep) }
    
    val steps = listOf(
        SecurityStep(
            id = "vault",
            title = "Your Vault",
            description = "Keys are generated and stored only on this device",
            icon = Icons.Default.Lock
        ),
        SecurityStep(
            id = "keys",
            title = "Encryption Keys",
            description = "You hold the keys - they never leave your device",
            icon = Icons.Default.Security
        ),
        SecurityStep(
            id = "agents",
            title = "Agent Supervision",
            description = "Monitor and approve agent actions in real-time",
            icon = Icons.Default.Shield
        ),
        SecurityStep(
            id = "control",
            title = "Full Control",
            description = "Pause, resume, or stop any agent at any time",
            icon = Icons.Default.PhoneAndroid
        )
    )
    
    ArmorClawTheme {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Setting Up Your Vault") },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                        }
                    },
                    actions = {
                        onSkipToHome?.let { skip ->
                            TextButton(onClick = skip) {
                                Text("Skip", color = BrandPurple)
                            }
                        }
                    }
                )
            }
        ) { paddingValues ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues)
                    .padding(DesignTokens.Spacing.lg)
                    .verticalScroll(rememberScrollState()),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                // Animated diagram
                SecurityDiagram(
                    steps = steps,
                    selectedStep = selectedStep,
                    onStepClick = { index -> selectedStep = index }
                )
                
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
                
                // Step details
                StepDetails(steps[selectedStep])
                
                Spacer(modifier = Modifier.weight(1f))
                
                // Actions
                Column(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                ) {
                    Button(
                        onClick = onNext,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(56.dp)
                    ) {
                        Text("Initialize Vault", style = MaterialTheme.typography.labelLarge)
                    }
                    
                    // Use defaults button for express setup
                    onUseDefaults?.let { useDefaults ->
                        OutlinedButton(
                            onClick = useDefaults,
                            modifier = Modifier.fillMaxWidth(),
                            colors = ButtonDefaults.outlinedButtonColors(
                                contentColor = BrandGreen
                            ),
                            border = androidx.compose.foundation.BorderStroke(
                                1.dp,
                                BrandGreen.copy(alpha = 0.5f)
                            )
                        ) {
                            Icon(
                                imageVector = Icons.Default.Bolt,
                                contentDescription = null,
                                modifier = Modifier.size(18.dp)
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Text("Use Smart Defaults")
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SecurityDiagram(
    steps: List<SecurityStep>,
    selectedStep: Int,
    onStepClick: (Int) -> Unit
) {
    val infiniteTransition = rememberInfiniteTransition(label = "security")
    
    val animatedOffset by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(2000, easing = androidx.compose.animation.core.LinearEasing),
            repeatMode = RepeatMode.Restart
        ),
        label = "offset"
    )
    
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(300.dp)
            .padding(DesignTokens.Spacing.md),
        contentAlignment = Alignment.Center
    ) {
        Canvas(
            modifier = Modifier.fillMaxSize()
        ) {
            val canvasWidth = size.width
            val canvasHeight = size.height
            val stepCount = steps.size
            val gapX = canvasWidth / (stepCount + 1)
            val centerY = canvasHeight / 2
            
            // Draw connections
            for (i in 0 until stepCount - 1) {
                val startX = gapX * (i + 1)
                val endX = gapX * (i + 2)
                val lineAlpha = if (i < selectedStep) 1f else 0.3f
                
                // Draw animated data flow
                val flowOffset = (animatedOffset * 100) % (endX - startX)
                val flowX = startX + flowOffset
                
                if (i < selectedStep) {
                    drawCircle(
                        color = BrandPurple,
                        radius = 4.dp.toPx(),
                        center = Offset(flowX, centerY)
                    )
                }
                
                // Draw line
                drawLine(
                    color = BrandPurple.copy(alpha = lineAlpha),
                    start = Offset(startX, centerY),
                    end = Offset(endX, centerY),
                    strokeWidth = 3.dp.toPx()
                )
            }
        }

        // Draw step nodes
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .align(Alignment.Center),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            steps.forEachIndexed { index, step ->
                StepNode(
                    step = step,
                    isSelected = index == selectedStep,
                    onClick = { onStepClick(index) }
                )
            }
        }
    }
}

@Composable
private fun StepNode(
    step: SecurityStep,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    val scale by animateFloatAsState(
        targetValue = if (isSelected) 1.2f else 1f,
        animationSpec = tween(300),
        label = "scale"
    )
    
    val borderColor by animateColorAsState(
        targetValue = if (isSelected) BrandPurple else BrandPurpleLight,
        animationSpec = tween(300),
        label = "borderColor"
    )
    
    Box(
        modifier = Modifier
            .size(if (isSelected) 80.dp else 60.dp)
            .scale(scale)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center
    ) {
        Card(
            modifier = Modifier
                .size(if (isSelected) 70.dp else 50.dp)
                .border(
                    width = if (isSelected) 3.dp else 1.dp,
                    color = borderColor,
                    shape = CircleShape
                ),
            colors = CardDefaults.cardColors(
                containerColor = if (isSelected) BrandPurpleLight.copy(alpha = 0.5f) else Color.White
            ),
            shape = CircleShape,
            elevation = CardDefaults.cardElevation(defaultElevation = if (isSelected) 8.dp else 2.dp)
        ) {
            Icon(
                imageVector = step.icon,
                contentDescription = step.title,
                tint = if (isSelected) BrandPurple else Color.Gray,
                modifier = Modifier.padding(12.dp)
            )
        }
    }
}

@Composable
private fun StepDetails(step: SecurityStep) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(DesignTokens.Spacing.lg),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Icon(
            imageVector = step.icon,
            contentDescription = null,
            tint = BrandPurple,
            modifier = Modifier.size(64.dp)
        )
        
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
        
        Text(
            text = step.title,
            style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold)
        )
        
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))
        
        Text(
            text = step.description,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
        )
    }
}
