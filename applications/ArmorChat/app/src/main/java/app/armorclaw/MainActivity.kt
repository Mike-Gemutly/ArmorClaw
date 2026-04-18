package app.armorclaw

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.navigation.NavHostController
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import app.armorclaw.navigation.ArmorClawNavHost
import app.armorclaw.navigation.DeepLinkHandler
import app.armorclaw.navigation.Route

/**
 * Deep links arriving while the user is on any of these routes are queued instead
 * of navigated immediately — mid-setup navigation would strand the user.
 */
private val SETUP_ROUTES = setOf(
    Route.Bonding.route,
    Route.SecurityConfig.route,
    Route.HardeningPassword.route,
    Route.HardeningDevice.route,
    Route.HardeningBiometrics.route,
    Route.KeyBackup.route,
)

class MainActivity : ComponentActivity() {

    private val pendingDeepLink = mutableStateOf<Route?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        intent?.data?.let { uri ->
            DeepLinkHandler.handle(uri)?.let { pendingDeepLink.value = it }
        }

        setContent {
            ArmorClawTheme {
                val navController = rememberNavController()
                val queuedDeepLink = remember { mutableStateOf<Route?>(null) }

                val currentEntry = navController.currentBackStackEntryAsState()
                val currentRoute = currentEntry.value?.destination?.route

                LaunchedEffect(pendingDeepLink.value) {
                    val route = pendingDeepLink.value ?: return@LaunchedEffect
                    val isOnSetup = currentRoute != null && currentRoute in SETUP_ROUTES

                    if (isOnSetup) {
                        queuedDeepLink.value = route
                    } else {
                        navigateToRoute(navController, route)
                    }
                    pendingDeepLink.value = null
                }

                LaunchedEffect(currentRoute) {
                    val queued = queuedDeepLink.value ?: return@LaunchedEffect
                    if (currentRoute !in SETUP_ROUTES) {
                        navigateToRoute(navController, queued)
                        queuedDeepLink.value = null
                    }
                }

                ArmorClawNavHost(navController = navController)
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        intent.data?.let { uri ->
            DeepLinkHandler.handle(uri)?.let { pendingDeepLink.value = it }
        }
    }

    private fun navigateToRoute(navController: NavHostController, route: Route) {
        navController.navigate(route.route) {
            popUpTo(navController.graph.startDestinationId) { saveState = true }
            launchSingleTop = true
            restoreState = true
        }
    }
}

@Composable
fun ArmorClawTheme(
    content: @Composable () -> Unit
) {
    val darkColorScheme = darkColorScheme()

    MaterialTheme(
        colorScheme = darkColorScheme,
        content = content
    )
}
