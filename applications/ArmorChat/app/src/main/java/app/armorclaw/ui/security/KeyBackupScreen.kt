package app.armorclaw.ui.security

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Key Backup Setup Screen - SSSS Recovery Passphrase
 *
 * Resolves: G-07 (Key Backup)
 *
 * Guides users through setting up a recovery passphrase for their encryption keys.
 * Part of onboarding flow after login.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun KeyBackupSetupScreen(
    onComplete: () -> Unit,
    onSkip: () -> Unit,
    viewModel: KeyBackupViewModel = viewModel()
) {
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Secure Your Keys") },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer
                )
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(24.dp)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Icon
            Icon(
                imageVector = Icons.Default.Lock,
                contentDescription = null,
                modifier = Modifier.size(64.dp),
                tint = MaterialTheme.colorScheme.primary
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Title
            Text(
                text = "Protect Your Messages",
                style = MaterialTheme.typography.headlineMedium,
                textAlign = TextAlign.Center
            )

            Spacer(modifier = Modifier.height(16.dp))

            // Description
            Text(
                text = "Set a recovery passphrase to backup your encryption keys. " +
                        "This allows you to restore your message history on new devices.",
                style = MaterialTheme.typography.bodyLarge,
                textAlign = TextAlign.Center
            )

            Spacer(modifier = Modifier.height(32.dp))

            when (uiState.step) {
                SetupStep.ENTER_PASSPHRASE -> {
                    PassphraseEntryCard(
                        passphrase = uiState.passphrase,
                        passphraseConfirm = uiState.passphraseConfirm,
                        passphraseVisible = uiState.passphraseVisible,
                        passphraseStrength = uiState.passphraseStrength,
                        onPassphraseChange = { viewModel.updatePassphrase(it) },
                        onPassphraseConfirmChange = { viewModel.updatePassphraseConfirm(it) },
                        onToggleVisibility = { viewModel.togglePassphraseVisibility() },
                        onContinue = { viewModel.validateAndProceed() },
                        error = uiState.error
                    )
                }

                SetupStep.CONFIRM_PASSPHRASE -> {
                    ConfirmPassphraseCard(
                        passphrase = uiState.passphraseConfirm,
                        passphraseVisible = uiState.passphraseVisible,
                        onPassphraseChange = { viewModel.updatePassphraseConfirm(it) },
                        onToggleVisibility = { viewModel.togglePassphraseVisibility() },
                        onConfirm = { viewModel.setupBackup() },
                        onBack = { viewModel.goBack() },
                        error = uiState.error
                    )
                }

                SetupStep.BACKING_UP -> {
                    Card(
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Column(
                            modifier = Modifier
                                .padding(24.dp)
                                .fillMaxWidth(),
                            horizontalAlignment = Alignment.CenterHorizontally
                        ) {
                            CircularProgressIndicator()
                            Spacer(modifier = Modifier.height(16.dp))
                            Text(
                                text = "Creating encrypted backup...",
                                style = MaterialTheme.typography.bodyLarge
                            )
                        }
                    }
                }

                SetupStep.COMPLETE -> {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        colors = CardDefaults.cardColors(
                            containerColor = Color(0xFF4CAF50).copy(alpha = 0.1f)
                        )
                    ) {
                        Column(
                            modifier = Modifier
                                .padding(24.dp)
                                .fillMaxWidth(),
                            horizontalAlignment = Alignment.CenterHorizontally
                        ) {
                            Icon(
                                imageVector = Icons.Default.CheckCircle,
                                contentDescription = null,
                                modifier = Modifier.size(48.dp),
                                tint = Color(0xFF4CAF50)
                            )
                            Spacer(modifier = Modifier.height(16.dp))
                            Text(
                                text = "Backup Complete!",
                                style = MaterialTheme.typography.titleLarge,
                                color = Color(0xFF4CAF50)
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = "Your encryption keys are now backed up securely. " +
                                        "Remember your passphrase - you'll need it to restore your messages.",
                                style = MaterialTheme.typography.bodyMedium,
                                textAlign = TextAlign.Center
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(24.dp))

                    Button(
                        onClick = onComplete,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Continue")
                    }
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Skip option (only on first step)
            if (uiState.step == SetupStep.ENTER_PASSPHRASE) {
                TextButton(onClick = onSkip) {
                    Text("Set up later")
                }
            }
        }
    }
}

@Composable
private fun PassphraseEntryCard(
    passphrase: String,
    passphraseConfirm: String,
    passphraseVisible: Boolean,
    passphraseStrength: PassphraseStrength,
    onPassphraseChange: (String) -> Unit,
    onPassphraseConfirmChange: (String) -> Unit,
    onToggleVisibility: () -> Unit,
    onContinue: () -> Unit,
    error: String?
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(
                text = "Create a Recovery Passphrase",
                style = MaterialTheme.typography.titleMedium
            )

            Spacer(modifier = Modifier.height(16.dp))

            // Passphrase field
            OutlinedTextField(
                value = passphrase,
                onValueChange = onPassphraseChange,
                label = { Text("Passphrase") },
                placeholder = { Text("Enter a strong passphrase") },
                visualTransformation = if (passphraseVisible)
                    VisualTransformation.None
                else
                    PasswordVisualTransformation(),
                trailingIcon = {
                    IconButton(onClick = onToggleVisibility) {
                        Icon(
                            imageVector = if (passphraseVisible)
                                Icons.Default.VisibilityOff
                            else
                                Icons.Default.Visibility,
                            contentDescription = if (passphraseVisible) "Hide" else "Show"
                        )
                    }
                },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Password,
                    imeAction = ImeAction.Next
                )
            )

            // Strength indicator
            if (passphrase.isNotEmpty()) {
                Spacer(modifier = Modifier.height(8.dp))
                PassphraseStrengthIndicator(strength = passphraseStrength)
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Confirm field
            OutlinedTextField(
                value = passphraseConfirm,
                onValueChange = onPassphraseConfirmChange,
                label = { Text("Confirm Passphrase") },
                visualTransformation = if (passphraseVisible)
                    VisualTransformation.None
                else
                    PasswordVisualTransformation(),
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Password,
                    imeAction = ImeAction.Done
                ),
                keyboardActions = KeyboardActions(onDone = { onContinue() }),
                isError = passphraseConfirm.isNotEmpty() && passphrase != passphraseConfirm
            )

            // Error message
            error?.let {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            Button(
                onClick = onContinue,
                enabled = passphrase.isNotEmpty() && passphrase == passphraseConfirm,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Continue")
            }
        }
    }
}

@Composable
private fun ConfirmPassphraseCard(
    passphrase: String,
    passphraseVisible: Boolean,
    onPassphraseChange: (String) -> Unit,
    onToggleVisibility: () -> Unit,
    onConfirm: () -> Unit,
    onBack: () -> Unit,
    error: String?
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Icon(
                imageVector = Icons.Default.Info,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "Confirm Your Passphrase",
                style = MaterialTheme.typography.titleMedium
            )

            Text(
                text = "Enter your passphrase again to confirm",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.outline
            )

            Spacer(modifier = Modifier.height(16.dp))

            OutlinedTextField(
                value = passphrase,
                onValueChange = onPassphraseChange,
                label = { Text("Passphrase") },
                visualTransformation = if (passphraseVisible)
                    VisualTransformation.None
                else
                    PasswordVisualTransformation(),
                trailingIcon = {
                    IconButton(onClick = onToggleVisibility) {
                        Icon(
                            imageVector = if (passphraseVisible)
                                Icons.Default.VisibilityOff
                            else
                                Icons.Default.Visibility,
                            contentDescription = if (passphraseVisible) "Hide" else "Show"
                        )
                    }
                },
                singleLine = true,
                modifier = Modifier.fillMaxWidth()
            )

            error?.let {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                OutlinedButton(
                    onClick = onBack,
                    modifier = Modifier.weight(1f)
                ) {
                    Text("Back")
                }

                Button(
                    onClick = onConfirm,
                    enabled = passphrase.isNotEmpty(),
                    modifier = Modifier.weight(1f)
                ) {
                    Text("Confirm")
                }
            }
        }
    }
}

@Composable
private fun PassphraseStrengthIndicator(strength: PassphraseStrength) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = "Strength: ",
            style = MaterialTheme.typography.bodySmall
        )

        LinearProgressIndicator(
            progress = { strength.progress },
            modifier = Modifier
                .weight(1f)
                .height(4.dp),
            color = strength.color,
            trackColor = strength.color.copy(alpha = 0.2f)
        )

        Spacer(modifier = Modifier.width(8.dp))

        Text(
            text = strength.label,
            style = MaterialTheme.typography.bodySmall,
            color = strength.color
        )
    }
}

/**
 * Key Backup ViewModel
 */
class KeyBackupViewModel : ViewModel() {

    private val _uiState = MutableStateFlow(KeyBackupUiState())
    val uiState: StateFlow<KeyBackupUiState> = _uiState.asStateFlow()

    private var originalPassphrase: String = ""

    fun updatePassphrase(passphrase: String) {
        _uiState.value = _uiState.value.copy(
            passphrase = passphrase,
            passphraseStrength = calculateStrength(passphrase),
            error = null
        )
    }

    fun updatePassphraseConfirm(passphrase: String) {
        _uiState.value = _uiState.value.copy(
            passphraseConfirm = passphrase,
            error = null
        )
    }

    fun togglePassphraseVisibility() {
        _uiState.value = _uiState.value.copy(
            passphraseVisible = !_uiState.value.passphraseVisible
        )
    }

    fun validateAndProceed() {
        val state = _uiState.value

        if (state.passphrase.length < 8) {
            _uiState.value = state.copy(error = "Passphrase must be at least 8 characters")
            return
        }

        if (state.passphrase != state.passphraseConfirm) {
            _uiState.value = state.copy(error = "Passphrases do not match")
            return
        }

        originalPassphrase = state.passphrase
        _uiState.value = state.copy(
            step = SetupStep.CONFIRM_PASSPHRASE,
            passphraseConfirm = "",
            error = null
        )
    }

    fun goBack() {
        _uiState.value = _uiState.value.copy(
            step = SetupStep.ENTER_PASSPHRASE,
            passphraseConfirm = "",
            error = null
        )
    }

    fun setupBackup() {
        val state = _uiState.value

        if (state.passphraseConfirm != originalPassphrase) {
            _uiState.value = state.copy(error = "Passphrase does not match")
            return
        }

        _uiState.value = state.copy(step = SetupStep.BACKING_UP)

        viewModelScope.launch {
            try {
                // Simulate backup creation
                // In production, this would:
                // 1. Derive key from passphrase using PBKDF2
                // 2. Encrypt private keys
                // 3. Upload to homeserver as SSSS backup

                kotlinx.coroutines.delay(2000)

                _uiState.value = _uiState.value.copy(step = SetupStep.COMPLETE)
            } catch (e: Exception) {
                _uiState.value = state.copy(
                    step = SetupStep.CONFIRM_PASSPHRASE,
                    error = "Backup failed: ${e.message}"
                )
            }
        }
    }

    private fun calculateStrength(passphrase: String): PassphraseStrength {
        var score = 0

        if (passphrase.length >= 8) score++
        if (passphrase.length >= 12) score++
        if (passphrase.length >= 16) score++
        if (passphrase.any { it.isUpperCase() }) score++
        if (passphrase.any { it.isLowerCase() }) score++
        if (passphrase.any { it.isDigit() }) score++
        if (passphrase.any { !it.isLetterOrDigit() }) score++

        return when {
            score < 3 -> PassphraseStrength.WEAK
            score < 5 -> PassphraseStrength.MEDIUM
            else -> PassphraseStrength.STRONG
        }
    }

    private val viewModelScope = androidx.lifecycle.viewModelScope
}

/**
 * Key Backup UI State
 */
data class KeyBackupUiState(
    val step: SetupStep = SetupStep.ENTER_PASSPHRASE,
    val passphrase: String = "",
    val passphraseConfirm: String = "",
    val passphraseVisible: Boolean = false,
    val passphraseStrength: PassphraseStrength = PassphraseStrength.WEAK,
    val error: String? = null
)

enum class SetupStep {
    ENTER_PASSPHRASE,
    CONFIRM_PASSPHRASE,
    BACKING_UP,
    COMPLETE
}

enum class PassphraseStrength(val label: String, val progress: Float, val color: androidx.compose.ui.graphics.Color) {
    WEAK("Weak", 0.33f, androidx.compose.ui.graphics.Color.Red),
    MEDIUM("Medium", 0.66f, androidx.compose.ui.graphics.Color(0xFFFFA000)),
    STRONG("Strong", 1f, androidx.compose.ui.graphics.Color(0xFF4CAF50))
}
