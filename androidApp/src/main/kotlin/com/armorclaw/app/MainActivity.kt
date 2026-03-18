package com.armorclaw.app

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Modifier
import androidx.compose.ui.tooling.preview.Preview
import androidx.navigation.compose.rememberNavController
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.app.navigation.AppNavHost
import com.armorclaw.app.navigation.AppNavigation
import com.armorclaw.app.navigation.DeepLinkHandler
import com.armorclaw.app.navigation.DeepLinkAction
import com.armorclaw.app.navigation.DeepLinkResult
import com.armorclaw.app.components.DeepLinkConfirmationDialog
import com.armorclaw.app.viewmodels.AppPreferences
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Shared deep link state that can be updated from onNewIntent
 * and observed in Compose
 */
object DeepLinkState {
    private val _pendingResult = MutableStateFlow<DeepLinkResult?>(null)
    val pendingResult: StateFlow<DeepLinkResult?> = _pendingResult.asStateFlow()

    fun setPendingResult(result: DeepLinkResult?) {
        _pendingResult.value = result
    }

    fun clearPendingResult() {
        _pendingResult.value = null
    }
}

class MainActivity : ComponentActivity() {

    private lateinit var preferences: AppPreferences
    private var initialDeepLinkUri: Uri? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        preferences = AppPreferences(this)


        // Handle initial intent (deep link from cold start)
        initialDeepLinkUri = intent?.data

        setContent {
            val darkTheme = isSystemInDarkTheme()
            var darkMode by rememberSaveable { mutableStateOf(darkTheme) }

            val navController = rememberNavController()
            val startDestination = getStartDestination()

            // Collect deep link state from shared state
            val pendingDeepLinkResult by DeepLinkState.pendingResult.collectAsState()

            // Handle initial deep link on first composition
            LaunchedEffect(initialDeepLinkUri) {
                initialDeepLinkUri?.let { uri ->
                    DeepLinkState.setPendingResult(DeepLinkHandler.parseDeepLinkUri(uri))
                }
            }

            // Track confirmed deep link action
            var confirmedAction by remember { mutableStateOf<DeepLinkAction?>(null) }

            // Handle deep link navigation
            LaunchedEffect(confirmedAction, pendingDeepLinkResult) {
                val action = confirmedAction
                    ?: (pendingDeepLinkResult as? DeepLinkResult.Valid)?.action

                if (action != null) {
                    val route = action.toRoute()
                    if (route != null) {
                        AppLogger.info(
                            LogTag.UI.Navigation,
                            "Navigating from deep link",
                            mapOf("route" to route)
                        )
                        navController.navigate(route) {
                            popUpTo(AppNavigation.HOME) { inclusive = false }
                        }
                        confirmedAction = null
                        DeepLinkState.clearPendingResult()
                    }
                }
            }

            // Show confirmation dialog if needed
            val resultToShow = pendingDeepLinkResult
            if (resultToShow is DeepLinkResult.RequiresConfirmation) {
                DeepLinkConfirmationDialog(
                    action = resultToShow.action,
                    securityCheck = resultToShow.securityCheck,
                    message = resultToShow.message,
                    details = resultToShow.details,
                    onConfirm = {
                        confirmedAction = resultToShow.action
                        DeepLinkState.clearPendingResult()
                    },
                    onDismiss = {
                        DeepLinkState.clearPendingResult()
                    }
                )
            }

            ArmorClawTheme(darkTheme = darkMode) {
                AppNavHost(
                    navController = navController,
                    modifier = Modifier,
                    startDestination = startDestination
                )
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        // Handle deep link from warm/hot start
        handleNewDeepLink(intent)
    }

    /**
     * Handle incoming deep link while app is running
     */
    private fun handleNewDeepLink(intent: Intent?) {
        val result = DeepLinkHandler.parseDeepLink(intent)
        if (result != null) {
            when (result) {
                is DeepLinkResult.Valid -> {
                    AppLogger.info(
                        LogTag.Platform.Android,
                        "Valid deep link received",
                        mapOf("route" to (result.action.toRoute() ?: "unknown"))
                    )
                }
                is DeepLinkResult.RequiresConfirmation -> {
                    AppLogger.info(
                        LogTag.Platform.Android,
                        "Deep link requires confirmation",
                        mapOf("check" to result.securityCheck.name)
                    )
                }
                is DeepLinkResult.Invalid -> {
                    AppLogger.warning(
                        LogTag.Platform.Android,
                        "Invalid deep link rejected",
                        mapOf("reason" to result.reason)
                    )
                }
            }
            // Update shared state to trigger navigation
            DeepLinkState.setPendingResult(result)
        }
    }

    /**
     * Determine start destination based on app state
     */
    private fun getStartDestination(): String {
        val hasCompletedOnboarding = preferences.hasCompletedOnboarding()
        val isLoggedIn = preferences.isLoggedIn()

        return when {
            initialDeepLinkUri != null -> {
                // If we have a deep link, go to appropriate screen after login check
                if (isLoggedIn) {
                    AppNavigation.HOME // Will handle deep link from there
                } else {
                    AppNavigation.LOGIN
                }
            }
            !hasCompletedOnboarding -> AppNavigation.WELCOME
            !isLoggedIn -> AppNavigation.LOGIN
            else -> AppNavigation.HOME
        }
    }
}

/**
 * Collect StateFlow as Compose state
 */
@Composable
private fun <T> StateFlow<T>.collectAsState(): State<T> {
    return androidx.compose.runtime.produceState(initialValue = value, this) {
        collect { value = it }
    }
}

@Preview(showBackground = true)
@Composable
fun DefaultPreview() {
    ArmorClawTheme(darkTheme = false) {
        // Preview content
    }
}
