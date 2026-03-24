package app.armorclaw.ui.security

import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.biometric.BiometricPrompt.AuthenticationCallback
import androidx.biometric.BiometricPrompt.PromptInfo
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.viewmodel.compose.viewModel
import app.armorclaw.R
import app.armorclaw.viewmodel.HardeningWizardViewModel

/**
 * Biometric enable screen for optional biometric setup during hardening
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BiometricEnableScreen(
    viewModel: HardeningWizardViewModel = viewModel(),
    onSuccess: () -> Unit
) {
    var biometricState by remember { mutableStateOf<BiometricState>(BiometricState.Idle) }
    val context = LocalContext.current

    // Check biometric capability on composition
    LaunchedEffect(Unit) {
        checkBiometricCapability(context, biometricState) { state ->
            biometricState = state
        }
    }

    Scaffold(
        topBar = {
            CenterAlignedTopAppBar(
                title = { Text("Biometric Lock") }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Biometric benefits card
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.secondaryContainer
                )
            ) {
                Row(
                    modifier = Modifier.padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        Icons.Default.Fingerprint,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.onSecondaryContainer
                    )
                    Spacer(Modifier.width(12.dp))
                    Column {
                        Text(
                            text = "Quick Secure Access",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSecondaryContainer
                        )
                        Text(
                            text = "Use fingerprint or face to unlock your keystore. Faster than passcode and more secure than nothing.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSecondaryContainer
                        )
                    }
                }
            }

            Spacer(Modifier.height(32.dp))

            // Title
            Text(
                text = "Enable Biometric Lock",
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold
            )

            Spacer(Modifier.height(8.dp))

            Text(
                text = "Add an extra layer of security to your keystore. Biometrics provide quick, secure access without typing passcodes.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center
            )

            Spacer(Modifier.height(32.dp))

            // Biometric state handling
            when (biometricState) {
                is BiometricState.Idle -> {
                    CircularProgressIndicator()
                    Spacer(Modifier.height(16.dp))
                    Text("Checking biometric hardware...")
                }

                is BiometricState.HardwareUnavailable -> {
                    Icon(
                        Icons.Default.DeviceUnknown,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.error,
                        modifier = Modifier.size(64.dp)
                    )

                    Spacer(Modifier.height(16.dp))

                    Text(
                        text = "No Biometric Hardware",
                        style = MaterialTheme.typography.headlineMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.error
                    )

                    Spacer(Modifier.height(8.dp))

                    Text(
                        text = "This device doesn't have fingerprint or face recognition hardware.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Spacer(Modifier.height(24.dp))

                    Button(
                        onClick = {
                            viewModel.acknowledgeStep("skipped")
                            onSuccess()
                        },
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Continue Without Biometrics")
                        Spacer(Modifier.width(8.dp))
                        Icon(Icons.Default.ArrowForward, contentDescription = null)
                    }
                }

                is BiometricState.NoBiometricsEnrolled -> {
                    Icon(
                        Icons.Default.Fingerprint,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.warning,
                        modifier = Modifier.size(64.dp)
                    )

                    Spacer(Modifier.height(16.dp))

                    Text(
                        text = "No Biometrics Enrolled",
                        style = MaterialTheme.typography.headlineMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.warning
                    )

                    Spacer(Modifier.height(8.dp))

                    Text(
                        text = "You need to enroll at least one fingerprint or face before using biometrics.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Spacer(Modifier.height(24.dp))

                    Button(
                        onClick = {
                            viewModel.acknowledgeStep("skipped")
                            onSuccess()
                        },
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Continue Without Biometrics")
                        Spacer(Modifier.width(8.dp))
                        Icon(Icons.Default.ArrowForward, contentDescription = null)
                    }
                }

                is BiometricState.Available -> {
                    // Enable Biometrics button
                    Button(
                        onClick = {
                            showBiometricPrompt(context, viewModel, onSuccess)
                        },
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Icon(Icons.Default.Fingerprint, contentDescription = null)
                        Spacer(Modifier.width(8.dp))
                        Text("Enable Biometrics")
                    }

                    Spacer(Modifier.height(16.dp))

                    // Skip button
                    TextButton(
                        onClick = {
                            viewModel.acknowledgeStep("skipped")
                            onSuccess()
                        },
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Skip for now")
                    }
                }

                is BiometricState.Authenticating -> {
                    CircularProgressIndicator()
                    Spacer(Modifier.height(16.dp))
                    Text("Authenticating...")
                }

                is BiometricState.Success -> {
                    Icon(
                        Icons.Default.CheckCircle,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(64.dp)
                    )

                    Spacer(Modifier.height(16.dp))

                    Text(
                        text = "Biometrics Enabled!",
                        style = MaterialTheme.typography.headlineMedium,
                        fontWeight = FontWeight.Bold
                    )

                    Spacer(Modifier.height(8.dp))

                    Text(
                        text = "Your keystore is now protected with biometric authentication.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Spacer(Modifier.height(24.dp))

                    Button(
                        onClick = onSuccess,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Continue to Next Step")
                        Spacer(Modifier.width(8.dp))
                        Icon(Icons.Default.ArrowForward, contentDescription = null)
                    }
                }

                is BiometricState.Error -> {
                    Icon(
                        Icons.Default.Error,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.error,
                        modifier = Modifier.size(64.dp)
                    )

                    Spacer(Modifier.height(16.dp))

                    Text(
                        text = "Authentication Failed",
                        style = MaterialTheme.typography.headlineMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.error
                    )

                    Spacer(Modifier.height(8.dp))

                    Text(
                        text = biometricState.message,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Spacer(Modifier.height(24.dp))

                    Button(
                        onClick = {
                            showBiometricPrompt(context, viewModel, onSuccess)
                        },
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Try Again")
                        Spacer(Modifier.width(8.dp))
                        Icon(Icons.Default.Refresh, contentDescription = null)
                    }
                }
            }

            Spacer(Modifier.height(32.dp))
        }
    }
}

/**
 * Check biometric capability and update state
 */
private fun checkBiometricCapability(
    context: android.content.Context,
    currentState: BiometricState,
    onStateUpdate: (BiometricState) -> Unit
) {
    val biometricManager = BiometricManager.from(context)
    when (biometricManager.canAuthenticate()) {
        BiometricManager.BIOMETRIC_SUCCESS -> {
            if (currentState != BiometricState.Available) {
                onStateUpdate(BiometricState.Available)
            }
        }
        BiometricManager.BIOMETRIC_ERROR_NO_HARDWARE -> {
            if (currentState != BiometricState.HardwareUnavailable) {
                onStateUpdate(BiometricState.HardwareUnavailable)
            }
        }
        BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED -> {
            if (currentState != BiometricState.NoBiometricsEnrolled) {
                onStateUpdate(BiometricState.NoBiometricsEnrolled)
            }
        }
        else -> {
            if (currentState != BiometricState.Idle) {
                onStateUpdate(BiometricState.Idle)
            }
        }
    }
}

/**
 * Show biometric prompt for authentication
 */
private fun showBiometricPrompt(
    context: android.content.Context,
    viewModel: HardeningWizardViewModel,
    onSuccess: () -> Unit
) {
    val executor = ContextCompat.getMainExecutor(context)
    val biometricPrompt = BiometricPrompt(
        context as android.app.Activity,
        executor,
        object : AuthenticationCallback() {
            override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                super.onAuthenticationSucceeded(result)
                viewModel.acknowledgeStep("biometrics_enabled")
                onSuccess()
            }

            override fun onAuthenticationFailed() {
                super.onAuthenticationFailed()
                // This is called when authentication fails, but we don't need to handle it here
                // The system will show the error message automatically
            }

            override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                super.onAuthenticationError(errorCode, errString)
                // Handle specific error codes if needed
                // For now, we'll let the system show the error message
            }
        }
    )

    val promptInfo = PromptInfo.Builder()
        .setTitle("Biometric Authentication")
        .setSubtitle("Verify your identity to enable biometric lock")
        .setNegativeButtonText("Cancel")
        .build()

    biometricPrompt.authenticate(promptInfo)
}

/**
 * Biometric state for the enable screen
 */
sealed class BiometricState {
    object Idle : BiometricState()
    object HardwareUnavailable : BiometricState()
    object NoBiometricsEnrolled : BiometricState()
    object Available : BiometricState()
    object Authenticating : BiometricState()
    object Success : BiometricState()
    data class Error(val message: String) : BiometricState()
}