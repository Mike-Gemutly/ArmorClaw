package com.armorclaw.app.screens.keystore

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
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.armorclaw.app.R
import com.armorclaw.app.viewmodels.UnsealViewModel
import com.armorclaw.app.viewmodels.UnsealUiState

/**
 * Unseal Screen
 *
 * Displays the VPS keystore unseal interface where users can enter
 * their master password or use biometric authentication to decrypt
 * stored credentials.
 *
 * ## States
 * - Sealed: Shows password input and optional biometric button
 * - Loading: Shows progress indicator during unseal operation
 * - Error: Shows error message with retry option
 *
 * ## Usage
 * ```kotlin
 * // In navigation
 * composable("unseal") {
 *     UnsealScreen(
 *         onUnsealed = { navController.popBackStack() }
 *     )
 * }
 * ```
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UnsealScreen(
    onUnsealed: () -> Unit,
    viewModel: UnsealViewModel = viewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    val password by viewModel.password.collectAsState()
    val useBiometric by viewModel.useBiometric.collectAsState()
    val passwordVisible by viewModel.passwordVisible.collectAsState()
    val biometricAvailable by viewModel.biometricAvailable.collectAsState()

    val focusRequester = remember { FocusRequester() }
    val scrollState = rememberScrollState()

    // Handle successful unseal
    LaunchedEffect(uiState) {
        if (uiState is UnsealUiState.Unsealed) {
            onUnsealed()
        }
    }

    // Request focus on password field when screen loads
    LaunchedEffect(Unit) {
        focusRequester.requestFocus()
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(scrollState)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        // Lock icon
        Icon(
            imageVector = if (uiState is UnsealUiState.Loading) {
                Icons.Default.LockOpen
            } else {
                Icons.Default.Lock
            },
            contentDescription = null,
            modifier = Modifier.size(72.dp),
            tint = when (uiState) {
                is UnsealUiState.Error -> MaterialTheme.colorScheme.error
                is UnsealUiState.Loading -> MaterialTheme.colorScheme.primary
                else -> MaterialTheme.colorScheme.onSurfaceVariant
            }
        )

        Spacer(Modifier.height(24.dp))

        // Title
        Text(
            text = "VPS Keystore Sealed",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center
        )

        Spacer(Modifier.height(12.dp))

        // Description
        Text(
            text = "The VPS cannot access credentials until you authorize.",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center
        )

        Spacer(Modifier.height(32.dp))

        // Error message
        if (uiState is UnsealUiState.Error) {
            Surface(
                color = MaterialTheme.colorScheme.errorContainer,
                shape = MaterialTheme.shapes.small
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(12.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.Warning,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.error
                    )
                    Spacer(Modifier.width(12.dp))
                    Text(
                        text = (uiState as UnsealUiState.Error).message,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onErrorContainer
                    )
                }
            }
            Spacer(Modifier.height(16.dp))
        }

        // Password field
        OutlinedTextField(
            value = password,
            onValueChange = viewModel::setPassword,
            label = { Text("Master Password") },
            placeholder = { Text("Enter your master password") },
            visualTransformation = if (passwordVisible) {
                VisualTransformation.None
            } else {
                PasswordVisualTransformation()
            },
            trailingIcon = {
                IconButton(onClick = viewModel::togglePasswordVisibility) {
                    Icon(
                        imageVector = if (passwordVisible) {
                            Icons.Default.VisibilityOff
                        } else {
                            Icons.Default.Visibility
                        },
                        contentDescription = if (passwordVisible) "Hide password" else "Show password"
                    )
                }
            },
            singleLine = true,
            enabled = uiState !is UnsealUiState.Loading,
            isError = uiState is UnsealUiState.Error,
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Password,
                imeAction = ImeAction.Go
            ),
            keyboardActions = KeyboardActions(
                onGo = {
                    if (password.isNotBlank()) {
                        viewModel.unsealWithPassword()
                    }
                }
            ),
            modifier = Modifier
                .fillMaxWidth()
                .focusRequester(focusRequester)
        )

        Spacer(Modifier.height(16.dp))

        // Biometric toggle (if available)
        if (biometricAvailable) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth()
            ) {
                Checkbox(
                    checked = useBiometric,
                    onCheckedChange = viewModel::setUseBiometric,
                    enabled = uiState !is UnsealUiState.Loading
                )
                Spacer(Modifier.width(8.dp))
                Icon(
                    imageVector = Icons.Default.Lock,
                    contentDescription = null,
                    tint = if (useBiometric) {
                        MaterialTheme.colorScheme.primary
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    }
                )
                Spacer(Modifier.width(8.dp))
                Text("Use Biometric Instead")
            }
        }

        Spacer(Modifier.height(24.dp))

        // Unseal button
        if (uiState is UnsealUiState.Loading) {
            CircularProgressIndicator(
                modifier = Modifier.size(48.dp)
            )
            Spacer(Modifier.height(16.dp))
            Text(
                text = "Decrypting keystore...",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        } else {
            Button(
                onClick = {
                    if (useBiometric && biometricAvailable) {
                        viewModel.unsealWithBiometric()
                    } else {
                        viewModel.unsealWithPassword()
                    }
                },
                enabled = password.isNotBlank() || (useBiometric && biometricAvailable),
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp)
            ) {
                Icon(
                    imageVector = if (useBiometric && biometricAvailable) {
                        Icons.Default.Lock
                    } else {
                        Icons.Default.LockOpen
                    },
                    contentDescription = null
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    text = if (useBiometric && biometricAvailable) {
                        "Unseal with Biometric"
                    } else {
                        "Unseal with Password"
                    },
                    style = MaterialTheme.typography.titleMedium
                )
            }
        }

        Spacer(Modifier.height(24.dp))

        // Session info
        Surface(
            color = MaterialTheme.colorScheme.surfaceVariant,
            shape = MaterialTheme.shapes.small
        ) {
            Row(
                modifier = Modifier.padding(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Schedule,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.size(20.dp)
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    text = "Session will remain unsealed for 4 hours",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }

        Spacer(Modifier.height(16.dp))

        // Security note
        Text(
            text = "Your credentials are encrypted with AES-256-GCM and stored in the Android Keystore",
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.outline,
            textAlign = TextAlign.Center
        )
    }
}
