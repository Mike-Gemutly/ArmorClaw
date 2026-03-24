package app.armorclaw.navigation

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import app.armorclaw.ui.security.BiometricEnableScreen
import app.armorclaw.ui.security.BondingScreen
import app.armorclaw.ui.security.KeyBackupSetupScreen
import app.armorclaw.ui.security.PasswordRotationScreen
import app.armorclaw.ui.security.SecurityConfigScreen
import app.armorclaw.ui.verification.BridgeVerificationScreen
import androidx.lifecycle.viewmodel.compose.viewModel
import app.armorclaw.viewmodel.HardeningStep
import app.armorclaw.viewmodel.HardeningWizardViewModel

@Composable
fun ArmorClawNavHost(navController: NavHostController) {
    val viewModel: HardeningWizardViewModel = viewModel()
    var startRoute by rememberSaveable { mutableStateOf(Route.Bonding.route) }

    LaunchedEffect(Unit) {
        viewModel.loadState()
        startRoute = when (viewModel.getCurrentStep()) {
            HardeningStep.ROTATE_PASSWORD -> Route.HardeningPassword.route
            HardeningStep.WIPE_BOOTSTRAP -> Route.HardeningPassword.route
            HardeningStep.VERIFY_DEVICE -> Route.HardeningDevice.route
            HardeningStep.BACKUP_RECOVERY -> Route.KeyBackup.route
            HardeningStep.ENABLE_BIOMETRICS -> Route.HardeningBiometrics.route
            HardeningStep.COMPLETE -> Route.SecurityConfig.route
        }
    }

    NavHost(
        navController = navController,
        startDestination = startRoute
    ) {
        composable(Route.Bonding.route) {
            BondingScreen(
                bondingState = app.armorclaw.ui.security.BondingState.ReadyToClaim,
                onClaimOwnership = { _, _, _ -> },
                onContinue = {
                    when (viewModel.getCurrentStep()) {
                        HardeningStep.ROTATE_PASSWORD,
                        HardeningStep.WIPE_BOOTSTRAP -> navController.navigate(Route.HardeningPassword.route)
                        HardeningStep.VERIFY_DEVICE -> navController.navigate(Route.HardeningDevice.route)
                        HardeningStep.BACKUP_RECOVERY -> navController.navigate(Route.KeyBackup.route)
                        HardeningStep.ENABLE_BIOMETRICS -> navController.navigate(Route.HardeningBiometrics.route)
                        HardeningStep.COMPLETE -> navController.navigate(Route.SecurityConfig.route)
                    }
                }
            )
        }

        composable(Route.HardeningPassword.route) {
            PasswordRotationScreen(
                viewModel = viewModel,
                onSuccess = {
                    viewModel.acknowledgeStep("password_rotated")
                    navController.navigate(Route.HardeningDevice.route)
                }
            )
        }

        composable(Route.HardeningDevice.route) {
            BridgeVerificationScreen(
                roomId = "",
                bridgeUserId = "",
                onVerificationComplete = {
                    viewModel.acknowledgeStep("device_verified")
                    navController.navigate(Route.KeyBackup.route)
                },
                onDismiss = { navController.popBackStack() }
            )
        }

        composable(Route.KeyBackup.route) {
            KeyBackupSetupScreen(
                onComplete = {
                    viewModel.acknowledgeStep("recovery_backed_up")
                    navController.navigate(Route.HardeningBiometrics.route)
                },
                onSkip = {
                    // Skipping key backup is not recommended as it's a mandatory step.
                    // Production should prevent this, but we allow navigation for now.
                    navController.navigate(Route.HardeningBiometrics.route)
                }
            )
        }

        composable(Route.HardeningBiometrics.route) {
            BiometricEnableScreen(
                viewModel = viewModel,
                onSuccess = {
                    navController.navigate(Route.SecurityConfig.route)
                }
            )
        }

        composable(Route.SecurityConfig.route) {
            SecurityConfigScreen(
                currentStep = 1,
                totalSteps = 1,
                onComplete = {
                    navController.navigate(Route.Home.route)
                },
                onBack = { navController.popBackStack() }
            )
        }

        composable(Route.Home.route) {
            PlaceholderScreen(
                title = "ArmorClaw Home",
                description = "Welcome to your secure agent dashboard."
            )
        }

        // KeyRecovery is a separate recovery flow, not part of onboarding
        composable(Route.KeyRecovery.route) {
            PlaceholderScreen(
                title = "Key Recovery",
                description = "Recover your encryption keys using your recovery passphrase."
            )
        }
    }
}

@Composable
fun PlaceholderScreen(title: String, description: String) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = "$title\n\n$description",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface
        )
    }
}
