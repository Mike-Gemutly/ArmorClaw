package com.armorclaw.app.screens.splash

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.material3.MaterialTheme
import androidx.compose.animation.core.*
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.scale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.Navy
import com.armorclaw.shared.ui.theme.Teal
import com.armorclaw.app.R
import kotlinx.coroutines.delay

/**
 * Splash screen with branding and loading
 *
 * This is the first screen users see when opening the app.
 * It displays the app logo, name, and performs initialization.
 *
 * Navigation flow:
 * - If has valid session → onNavigateToHome
 * - If completed onboarding but not logged in → onNavigateToLogin
 * - Otherwise (first time user) → onNavigateToConnect (QR-first onboarding)
 */
@Composable
fun SplashScreen(
    onNavigateToOnboarding: () -> Unit,
    onNavigateToLogin: () -> Unit,
    onNavigateToHome: () -> Unit,
    onNavigateToConnect: () -> Unit = onNavigateToOnboarding,  // Default to onboarding for backward compat
    modifier: Modifier = Modifier
) {
    // Animation states
    var alpha by remember { mutableFloatStateOf(0f) }
    var scale by remember { mutableFloatStateOf(0.8f) }

    // Animations
    LaunchedEffect(Unit) {
        // Fade in and scale up logo
        alpha = 1f
        scale = 1f

        // Simulate initialization and check onboarding status
        delay(1500)

        // Navigation is now handled by SplashViewModel via StateFlow
        // This screen just shows the splash animation
        // The actual navigation is triggered from the parent via ViewModel observation
    }
    
    // Animation specs
    val alphaSpec = tween<Float>(durationMillis = 800, easing = FastOutSlowInEasing)
    val scaleSpec = spring<Float>(dampingRatio = Spring.DampingRatioMediumBouncy)
    
    // Animate alpha
    val animatedAlpha by animateFloatAsState(
        targetValue = alpha,
        animationSpec = alphaSpec,
        label = "alpha"
    )
    
    // Animate scale
    val animatedScale by animateFloatAsState(
        targetValue = scale,
        animationSpec = scaleSpec,
        label = "scale"
    )
    
    // UI
    Box(
        modifier = modifier
            .fillMaxSize()
            .background(Navy)
            .padding(32.dp),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            // Logo
            Logo(
                modifier = Modifier
                    .scale(animatedScale)
                    .alpha(animatedAlpha)
            )
            
            // App name
            AppName(
                modifier = Modifier.alpha(animatedAlpha)
            )
            
            // Tagline
            Tagline(
                modifier = Modifier.alpha(animatedAlpha)
            )
            
            // Loading indicator
            LoadingIndicator(
                modifier = Modifier
                    .padding(top = 48.dp)
                    .alpha(animatedAlpha * 0.7f)
            )
        }
    }
}

@Composable
private fun Logo(
    modifier: Modifier = Modifier
) {
    Image(
        painter = painterResource(id = R.drawable.ic_crab_mascot),
        contentDescription = "ArmorClaw",
        modifier = modifier.size(120.dp)
    )
}

@Composable
private fun AppName(
    modifier: Modifier = Modifier
) {
    Text(
        text = "ArmorClaw",
        style = MaterialTheme.typography.headlineLarge,
        fontWeight = FontWeight.Bold,
        color = Teal,
        textAlign = TextAlign.Center,
        modifier = modifier
    )
}

@Composable
private fun Tagline(
    modifier: Modifier = Modifier
) {
    Text(
        text = "Secure End-to-End Encrypted Chat",
        style = MaterialTheme.typography.bodyLarge,
        color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.7f),
        textAlign = TextAlign.Center,
        modifier = modifier
    )
}

@Composable
private fun LoadingIndicator(
    modifier: Modifier = Modifier
) {
    CircularProgressIndicator(
        modifier = modifier.size(32.dp),
        color = Teal,
        strokeWidth = 3.dp
    )
}

@Preview(showBackground = true)
@Composable
private fun SplashScreenPreview() {
    ArmorClawTheme {
        SplashScreen(
            onNavigateToOnboarding = {},
            onNavigateToLogin = {},
            onNavigateToHome = {}
        )
    }
}
