package com.armorclaw.app.screens.onboarding

import androidx.compose.animation.*
import androidx.compose.animation.core.tween
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.DesignTokens

/**
 * Tutorial screen that guides users through app features
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TutorialScreen(
    onNavigateBack: () -> Unit,
    onComplete: () -> Unit,
    modifier: Modifier = Modifier
) {
    var currentPage by remember { mutableIntStateOf(0) }

    val tutorialPages = listOf(
        TutorialPage(
            title = "Welcome to ArmorClaw",
            description = "Secure messaging with end-to-end encryption",
            icon = Icons.Default.Lock,
            color = BrandPurple
        ),
        TutorialPage(
            title = "Chat with Anyone",
            description = "Send messages, share files, and make calls securely",
            icon = Icons.Default.Chat,
            color = BrandGreen
        ),
        TutorialPage(
            title = "Manage Your Privacy",
            description = "Control who can contact you and see your presence",
            icon = Icons.Default.Security,
            color = BrandPurple
        ),
        TutorialPage(
            title = "You're All Set!",
            description = "You're ready to start using ArmorClaw securely",
            icon = Icons.Default.CheckCircle,
            color = BrandGreen
        )
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Tutorial") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    TextButton(onClick = onComplete) {
                        Text("Skip")
                    }
                }
            )
        }
    ) { paddingValues ->
        Column(
            modifier = modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(DesignTokens.Spacing.lg),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Page indicator
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.Center
            ) {
                tutorialPages.forEachIndexed { index, _ ->
                    Box(
                        modifier = Modifier
                            .padding(4.dp)
                            .size(if (index == currentPage) 10.dp else 8.dp)
                            .padding(2.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        if (index == currentPage) {
                            Icon(
                                Icons.Default.FiberManualRecord,
                                contentDescription = null,
                                tint = BrandPurple,
                                modifier = Modifier.size(10.dp)
                            )
                        } else {
                            Icon(
                                Icons.Default.Circle,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.outline,
                                modifier = Modifier.size(8.dp)
                            )
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))

            // Tutorial content
            AnimatedContent(
                targetState = currentPage,
                transitionSpec = {
                    fadeIn(animationSpec = tween(300)) togetherWith
                            fadeOut(animationSpec = tween(300))
                },
                label = "tutorial_page_transition"
            ) { page ->
                TutorialPageContent(
                    page = tutorialPages[page],
                    modifier = Modifier.weight(1f)
                )
            }

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

            // Navigation buttons
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
            ) {
                if (currentPage > 0) {
                    OutlinedButton(
                        onClick = { currentPage-- },
                        modifier = Modifier.weight(1f)
                    ) {
                        Icon(Icons.Default.ArrowBack, contentDescription = null)
                        Spacer(modifier = Modifier.width(8.dp))
                        Text("Previous")
                    }
                } else {
                    Spacer(modifier = Modifier.weight(1f))
                }

                Button(
                    onClick = {
                        if (currentPage < tutorialPages.size - 1) {
                            currentPage++
                        } else {
                            onComplete()
                        }
                    },
                    modifier = Modifier.weight(1f)
                ) {
                    Text(if (currentPage < tutorialPages.size - 1) "Next" else "Get Started")
                    if (currentPage < tutorialPages.size - 1) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Icon(Icons.Default.ArrowForward, contentDescription = null)
                    }
                }
            }
        }
    }
}

@Composable
private fun TutorialPageContent(
    page: TutorialPage,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(DesignTokens.Spacing.xxl))

        // Icon
        Surface(
            modifier = Modifier.size(120.dp),
            shape = MaterialTheme.shapes.extraLarge,
            color = page.color.copy(alpha = 0.1f)
        ) {
            Box(
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = page.icon,
                    contentDescription = null,
                    modifier = Modifier.size(64.dp),
                    tint = page.color
                )
            }
        }

        Spacer(modifier = Modifier.height(DesignTokens.Spacing.xxl))

        // Title
        Text(
            text = page.title,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))

        // Description
        Text(
            text = page.description,
            style = MaterialTheme.typography.bodyLarge,
            textAlign = TextAlign.Center,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
        )
    }
}

private data class TutorialPage(
    val title: String,
    val description: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector,
    val color: androidx.compose.ui.graphics.Color
)
