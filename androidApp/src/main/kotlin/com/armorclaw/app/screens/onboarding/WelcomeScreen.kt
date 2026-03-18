package com.armorclaw.app.screens.onboarding

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
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Chat
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Shield
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.armorclaw.app.components.atom.ArmorClawButton
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.OnBackground
import com.armorclaw.shared.ui.theme.OnPrimary
import com.armorclaw.shared.ui.theme.Primary

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WelcomeScreen(
    onGetStarted: () -> Unit,
    onSkip: () -> Unit
) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Primary,
                    titleContentColor = OnPrimary
                )
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(DesignTokens.Spacing.xl)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            VaultKeyVisual()

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

            Text(
                text = "Your Personal Vault",
                style = MaterialTheme.typography.headlineMedium
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            Text(
                text = "ArmorClaw",
                style = MaterialTheme.typography.titleMedium,
                color = BrandPurple
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

            Text(
                text = "Control your AI agents with keys only you hold",
                style = MaterialTheme.typography.bodyLarge,
                color = OnBackground.copy(alpha = 0.7f)
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

            FeatureList(
                features = listOf(
                    Feature(
                        icon = Icons.Default.Lock,
                        title = "Your Vault, Your Keys",
                        description = "Encryption keys never leave your device"
                    ),
                    Feature(
                        icon = Icons.Default.Security,
                        title = "Agent Supervision",
                        description = "Monitor and control AI agents in real-time"
                    ),
                    Feature(
                        icon = Icons.Default.Chat,
                        title = "Secure Communication",
                        description = "End-to-end encrypted via Matrix protocol"
                    )
                )
            )

            Spacer(modifier = Modifier.weight(1f))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
            ) {
                OutlinedButton(
                    onClick = onSkip,
                    modifier = Modifier.weight(1f)
                ) {
                    Text("Skip")
                }

                ArmorClawButton(
                    text = "Get Started",
                    onClick = onGetStarted,
                    modifier = Modifier.weight(1f)
                )
            }
        }
    }
}

@Composable
private fun FeatureList(features: List<Feature>) {
    Column(
        verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
    ) {
        features.forEach { feature ->
            FeatureCard(feature)
        }
    }
}

@Composable
private fun FeatureCard(feature: Feature) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
    ) {
        Icon(
            imageVector = feature.icon,
            contentDescription = feature.title,
            tint = BrandPurple
        )

        Column {
            Text(
                text = feature.title,
                style = MaterialTheme.typography.titleSmall
            )
            Text(
                text = feature.description,
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.7f)
            )
        }
    }
}

data class Feature(
    val icon: ImageVector,
    val title: String,
    val description: String
)

@Composable
private fun VaultKeyVisual(
    modifier: Modifier = Modifier
) {
    val infiniteTransition = rememberInfiniteTransition(label = "vault_animation")

    val scale by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = 1.08f,
        animationSpec = infiniteRepeatable(
            animation = tween(1200, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "scale"
    )

    val colorAlpha by infiniteTransition.animateFloat(
        initialValue = 1f,
        targetValue = 0.6f,
        animationSpec = infiniteRepeatable(
            animation = tween(1500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "colorAlpha"
    )

    Box(
        modifier = modifier.size(140.dp),
        contentAlignment = Alignment.Center
    ) {
        Box(
            modifier = Modifier
                .size(120.dp)
                .graphicsLayer { scaleX = scale; scaleY = scale }
                .clip(CircleShape)
                .background(BrandPurple.copy(alpha = 0.1f))
        )

        Icon(
            imageVector = Icons.Default.Shield,
            contentDescription = "Vault",
            modifier = Modifier.size(100.dp),
            tint = BrandPurple.copy(alpha = colorAlpha)
        )

        Icon(
            imageVector = Icons.Outlined.Lock,
            contentDescription = null,
            modifier = Modifier
                .size(40.dp)
                .align(Alignment.BottomEnd)
                .graphicsLayer { translationX = -8f; translationY = -8f },
            tint = Primary
        )
    }
}
