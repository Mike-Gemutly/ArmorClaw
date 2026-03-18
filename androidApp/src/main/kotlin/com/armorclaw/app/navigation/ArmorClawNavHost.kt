package com.armorclaw.app.navigation

import android.net.Uri
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.navigation.NavHost
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.armorclaw.app.screens.chat.ChatScreenEnhanced
import com.armorclaw.app.screens.home.HomeScreen
import com.armorclaw.app.screens.onboarding.CompletionScreen
import com.armorclaw.app.screens.onboarding.ConnectServerScreen
import com.armorclaw.app.screens.onboarding.ExpressSetupCompleteScreen
import com.armorclaw.app.screens.onboarding.PermissionsScreen
import com.armorclaw.app.screens.onboarding.SecurityExplanationScreen
import com.armorclaw.app.screens.onboarding.SetupModeSelectionScreen
import com.armorclaw.app.screens.onboarding.WelcomeScreen
import com.armorclaw.app.viewmodels.ChatViewModel
import org.koin.androidx.compose.koinViewModel
import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import org.koin.compose.koinInject
import org.koin.core.parameter.parametersOf

@Composable
fun ArmorClawNavHost(
    modifier: Modifier = Modifier,
    startDestination: String = "welcome",
    deepLinkAction: DeepLinkAction? = null,
    onDeepLinkHandled: () -> Unit = {}
) {
    val navController = rememberNavController()

    // Handle deep link navigation
    LaunchedEffect(deepLinkAction) {
        deepLinkAction?.let { action ->
            handleDeepLinkAction(navController, action)
            onDeepLinkHandled()
        }
    }

    Scaffold(modifier = modifier) { paddingValues ->
        NavHost(
            navController = navController,
            startDestination = startDestination,
            modifier = Modifier.padding(paddingValues)
        ) {
            // Onboarding
            composable("welcome") {
                WelcomeScreen(
                    onGetStarted = {
                        navController.navigate("security")
                    },
                    onSkip = {
                        navController.navigate("home") {
                            popUpTo("welcome") { inclusive = true }
                        }
                    }
                )
            }

            composable("security") {
                SecurityExplanationScreen(
                    onNext = {
                        navController.navigate("setup_mode")
                    },
                    onBack = {
                        navController.popBackStack()
                    },
                    onUseDefaults = {
                        // Express setup - skip to express complete
                        navController.navigate("express_complete") {
                            popUpTo("security") { inclusive = false }
                        }
                    }
                )
            }

            composable("setup_mode") {
                SetupModeSelectionScreen(
                    viewModel = koinViewModel(),
                    onExpressSelected = {
                        navController.navigate("express_complete") {
                            popUpTo("security") { inclusive = false }
                        }
                    },
                    onCustomSelected = {
                        navController.navigate("connect")
                    },
                    onBack = {
                        navController.popBackStack()
                    }
                )
            }

            composable("express_complete") {
                ExpressSetupCompleteScreen(
                    onStartChatting = {
                        navController.navigate("home") {
                            popUpTo("welcome") { inclusive = true }
                        }
                    },
                    onViewSettings = {
                        // TODO: Navigate to security settings
                        navController.navigate("home") {
                            popUpTo("welcome") { inclusive = true }
                        }
                    }
                )
            }

            composable("connect") {
                ConnectServerScreen(
                    onConnected = { serverInfo ->
                        navController.navigate("permissions")
                    },
                    onBack = {
                        navController.popBackStack()
                    }
                )
            }

            composable("permissions") {
                PermissionsScreen(
                    onComplete = {
                        navController.navigate("complete")
                    },
                    onBack = {
                        navController.popBackStack()
                    }
                )
            }

            composable("complete") {
                CompletionScreen(
                    onStartChatting = {
                        navController.navigate("home") {
                            popUpTo("welcome") { inclusive = true }
                        }
                    },
                    onTakeTutorial = {
                        // TODO: Launch tutorial
                        navController.navigate("home") {
                            popUpTo("welcome") { inclusive = true }
                        }
                    }
                )
            }

            // Home
            composable("home") {
                HomeScreen(
                    onRoomClick = { roomId ->
                        navController.navigate("chat/$roomId")
                    }
                )
            }

            // Chat
            composable(
                route = "chat/{roomId}",
                arguments = listOf(
                    navArgument("roomId") { type = NavType.StringType }
                )
            ) { backStackEntry ->
                val roomId = backStackEntry.arguments?.getString("roomId") ?: return@composable
                val matrixClient: MatrixClient = koinInject()
                val controlPlaneStore: ControlPlaneStore = koinInject()
                val messageRepository: MessageRepository = koinInject()
                ChatScreenEnhanced(
                    roomId = roomId,
                    viewModel = remember { ChatViewModel(roomId, matrixClient, controlPlaneStore, messageRepository) },
                    onNavigateBack = {
                        navController.popBackStack()
                    }
                )
            }
        }
    }
}

/**
 * Handle deep link action navigation (Fix 5: proper backstack)
 *
 * Ensures Home is always the root of the backstack when handling deep links,
 * so pressing Back navigates to Home instead of exiting the app.
 */
private fun handleDeepLinkAction(
    navController: androidx.navigation.NavController,
    action: DeepLinkAction
) {
    val route = action.toRoute()
    if (route != null) {
        AppLogger.info(
            LogTag.UI.Navigation,
            "Navigating from deep link",
            mapOf("route" to route)
        )

        // First ensure Home is in the backstack as root
        val currentBackStack = navController.currentBackStack.value
        val hasHomeInStack = currentBackStack.any {
            it.destination.route == AppNavigation.HOME ||
            it.destination.route == "home"
        }

        if (!hasHomeInStack) {
            // Navigate to Home first, clearing any previous stack
            navController.navigate(AppNavigation.HOME) {
                popUpTo(0) { inclusive = true }
            }
        }

        // Then navigate to the deep link target on top of Home
        navController.navigate(route) {
            // Don't remove Home from the stack
            launchSingleTop = true
        }
    }
}
