package com.armorclaw.app.navigation

import androidx.compose.material3.SnackbarDuration
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SnackbarResult
import androidx.navigation.NavController
import androidx.navigation.NavOptionsBuilder
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * Safe navigation wrapper with error handling
 *
 * Provides navigation functions that catch and handle errors gracefully,
 * showing user feedback instead of crashing.
 */

/**
 * Navigate safely with error handling
 *
 * @param route Destination route
 * @param builder Optional NavOptionsBuilder for configuration
 * @param snackbarHostState SnackbarHostState for showing error messages
 * @param scope CoroutineScope for launching snackbar
 * @param onError Optional callback for error logging
 */
fun NavController.navigateSafely(
    route: String,
    builder: NavOptionsBuilder.() -> Unit = {},
    snackbarHostState: SnackbarHostState? = null,
    scope: CoroutineScope? = null,
    onError: ((Throwable) -> Unit)? = null
) {
    try {
        navigate(route, builder)
        AppLogger.debug(
            LogTag.UI.Navigation,
            "Navigation succeeded",
            mapOf("route" to route)
        )
    } catch (e: Exception) {
        handleNavigationError(
            error = e,
            route = route,
            snackbarHostState = snackbarHostState,
            scope = scope,
            onError = onError
        )
    }
}

/**
 * Navigate safely with error handling (simple version)
 */
fun NavController.navigateSafely(
    route: String,
    snackbarHostState: SnackbarHostState?,
    scope: CoroutineScope?
) {
    navigateSafely(
        route = route,
        builder = {},
        snackbarHostState = snackbarHostState,
        scope = scope
    )
}

/**
 * Pop back stack safely with error handling
 */
fun NavController.popBackStackSafely(
    snackbarHostState: SnackbarHostState? = null,
    scope: CoroutineScope? = null,
    onError: ((Throwable) -> Unit)? = null
): Boolean {
    return try {
        val result = popBackStack()
        AppLogger.debug(
            LogTag.UI.Navigation,
            "Pop back stack succeeded",
            mapOf("result" to result)
        )
        result
    } catch (e: Exception) {
        handleNavigationError(
            error = e,
            route = "pop_back_stack",
            snackbarHostState = snackbarHostState,
            scope = scope,
            onError = onError
        )
        false
    }
}

/**
 * Navigate up safely with error handling
 */
fun NavController.navigateUpSafely(
    snackbarHostState: SnackbarHostState? = null,
    scope: CoroutineScope? = null
): Boolean {
    return try {
        val result = navigateUp()
        AppLogger.debug(
            LogTag.UI.Navigation,
            "Navigate up succeeded",
            mapOf("result" to result)
        )
        result
    } catch (e: Exception) {
        handleNavigationError(
            error = e,
            route = "navigate_up",
            snackbarHostState = snackbarHostState,
            scope = scope
        )
        false
    }
}

/**
 * Handle navigation error
 */
private fun handleNavigationError(
    error: Throwable,
    route: String,
    snackbarHostState: SnackbarHostState?,
    scope: CoroutineScope?,
    onError: ((Throwable) -> Unit)? = null
) {
    AppLogger.error(
        LogTag.UI.Navigation,
        "Navigation failed: ${error.message}",
        error,
        mapOf("route" to route)
    )

    onError?.invoke(error)

    scope?.launch {
        snackbarHostState?.showSnackbar(
            message = "Could not navigate to screen. Please try again.",
            actionLabel = "Retry",
            duration = SnackbarDuration.Short
        )
    }
}

/**
 * Navigation state holder for tracking navigation events
 */
data class NavigationState(
    val lastNavigationTime: Long = 0,
    val lastRoute: String? = null,
    val navigationCount: Int = 0,
    val errorCount: Int = 0
)

/**
 * Navigation event for logging and analytics
 */
sealed class NavigationEvent {
    data class Navigate(val route: String, val timestamp: Long) : NavigationEvent()
    data class PopBackStack(val success: Boolean, val timestamp: Long) : NavigationEvent()
    data class Error(val route: String, val error: Throwable, val timestamp: Long) : NavigationEvent()
}
