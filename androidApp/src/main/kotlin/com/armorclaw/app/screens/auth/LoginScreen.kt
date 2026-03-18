package com.armorclaw.app.screens.auth
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.text.input.VisualTransformation

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor
import com.armorclaw.app.components.sync.SyncStatusWrapper
import com.armorclaw.app.viewmodels.SyncStatusViewModel

// TODO: Replace with BuildConfig.VERSION_NAME when build is configured
private const val APP_VERSION = "1.0.0"

/**
 * Login screen for user authentication
 * 
 * This screen allows users to log in with email/password,
 * register new account, or use biometric auth.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LoginScreen(
    onLogin: (username: String, password: String) -> Unit,
    onRegister: () -> Unit,
    onForgotPassword: () -> Unit,
    onBiometricLogin: () -> Unit,
    onRecoverKeys: () -> Unit = {},
    modifier: Modifier = Modifier,
    isLoading: Boolean = false,
    isBiometricLoading: Boolean = false,
    errorMessage: String? = null,
    syncStatusViewModel: SyncStatusViewModel? = null,
    onRetry: () -> Unit = {}
) {
    val scrollState = rememberScrollState()

    // Form state
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var isPasswordVisible by remember { mutableStateOf(false) }
    var isBiometricAvailable by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    // Validation
    val isValid = username.isNotBlank() && password.isNotBlank()

    // Focus management
    val passwordFocusRequester = FocusRequester()
    
    // Form submission with error handling
    fun submit() {
        if (isValid) {
            // Clear any previous errors before attempting login
            syncStatusViewModel?.clearError()
            
            onLogin(username, password)
        }
    }
    
    // Handle login failure from authentication flow
    fun handleLoginFailure(error: String) {
        errorMessage = error
        syncStatusViewModel?.setError(error, isRecoverable = true)
    }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Color.Transparent
                )
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .background(SurfaceColor)
                .verticalScroll(scrollState)
                .imePadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            // Sync status wrapper - handles offline indicator and error recovery
            syncStatusViewModel?.let {
                SyncStatusWrapper(
                    viewModel = it,
                    onRetry = onRetry,
                    modifier = Modifier.fillMaxWidth()
                )
            }
            
            // Top spacing
            Spacer(modifier = Modifier.weight(1f))
            // Top spacing
            Spacer(modifier = Modifier.weight(1f))
            
            // Logo and branding
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Logo
                Logo()
                
                // App name
                Text(
                    text = "ArmorClaw",
                    style = MaterialTheme.typography.headlineLarge,
                    fontWeight = FontWeight.Bold
                )
                
                // Tagline
                Text(
                    text = "Secure. Private. Encrypted.",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                )
            }
            
            Spacer(modifier = Modifier.height(32.dp))
            
            // Login form
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 24.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Column(
                    modifier = Modifier.padding(24.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    // Username field
                    OutlinedTextField(
                        value = username,
                        onValueChange = { username = it },
                        label = { Text("Username or Email") },
                        placeholder = { Text("Enter your username or email") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Email,
                            imeAction = ImeAction.Next
                        ),
                        leadingIcon = {
                            Icon(
                                imageVector = Icons.Default.Person,
                                contentDescription = "Username or email"
                            )
                        },
                        trailingIcon = {
                            if (username.isNotEmpty()) {
                                IconButton(onClick = { username = "" }) {
                                    Icon(
                                        imageVector = Icons.Default.Clear,
                                        contentDescription = "Clear"
                                    )
                                }
                            }
                        },
                        shape = RoundedCornerShape(12.dp)
                    )
                    
                    // Password field
                    OutlinedTextField(
                        value = password,
                        onValueChange = { password = it },
                        label = { Text("Password") },
                        placeholder = { Text("Enter your password") },
                        modifier = Modifier
                            .fillMaxWidth()
                            .focusRequester(passwordFocusRequester),
                        singleLine = true,
                        visualTransformation = if (isPasswordVisible)
                            androidx.compose.ui.text.input.VisualTransformation.None
                        else
                            PasswordVisualTransformation(),
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Password,
                            imeAction = ImeAction.Done
                        ),
                        leadingIcon = {
                            Icon(
                                imageVector = Icons.Default.Lock,
                                contentDescription = "Password"
                            )
                        },
                        trailingIcon = {
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                // Toggle password visibility
                                IconButton(
                                    onClick = { isPasswordVisible = !isPasswordVisible }
                                ) {
                                    Icon(
                                        imageVector = if (isPasswordVisible)
                                            Icons.Default.Visibility
                                        else
                                            Icons.Default.VisibilityOff,
                                        contentDescription = if (isPasswordVisible)
                                            "Hide password"
                                        else
                                            "Show password"
                                    )
                                }
                                
                                // Clear password
                                if (password.isNotEmpty()) {
                                    IconButton(onClick = { password = "" }) {
                                        Icon(
                                            imageVector = Icons.Default.Clear,
                                            contentDescription = "Clear"
                                        )
                                    }
                                }
                            }
                        },
                        shape = RoundedCornerShape(12.dp)
                    )
                    
                    // Forgot password link
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.End
                    ) {
                        TextButton(
                            onClick = onForgotPassword,
                            colors = ButtonDefaults.textButtonColors(
                                contentColor = AccentColor
                            )
                        ) {
                            Text(
                                text = "Forgot password?",
                                style = MaterialTheme.typography.bodyMedium
                            )
                        }
                    }
                    
                    // Error message is handled by SyncStatusWrapper's ErrorRecoveryBanner

                    // Login button
                    Button(
                        onClick = { submit() },
                        enabled = isValid && !isLoading,
                        modifier = Modifier.fillMaxWidth(),
                        colors = ButtonDefaults.buttonColors(
                            containerColor = AccentColor,
                            disabledContainerColor = AccentColor.copy(alpha = 0.3f)
                        ),
                        shape = RoundedCornerShape(12.dp)
                    ) {
                        if (isLoading) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(24.dp),
                                color = Color.White,
                                strokeWidth = 2.dp
                            )
                        } else {
                            Text(
                                text = "Log In",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.SemiBold,
                                modifier = Modifier.padding(12.dp)
                            )
                        }
                    }
                    
                    // Divider
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Divider(modifier = Modifier.weight(1f))
                        Text(
                            text = "OR",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
                        )
                        Divider(modifier = Modifier.weight(1f))
                    }
                    
                    // Biometric login button
                    if (isBiometricAvailable) {
                        OutlinedButton(
                            onClick = onBiometricLogin,
                            enabled = !isBiometricLoading,
                            modifier = Modifier.fillMaxWidth(),
                            colors = ButtonDefaults.outlinedButtonColors(
                                contentColor = MaterialTheme.colorScheme.onSurface
                            ),
                            shape = RoundedCornerShape(12.dp)
                        ) {
                            if (isBiometricLoading) {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(20.dp),
                                    strokeWidth = 2.dp
                                )
                            } else {
                                Icon(
                                    imageVector = Icons.Default.Fingerprint,
                                    contentDescription = "Biometric authentication",
                                    modifier = Modifier.size(20.dp)
                                )
                            }
                            Spacer(modifier = Modifier.width(8.dp))
                            Text(if (isBiometricLoading) "Authenticating..." else "Log in with Biometrics")
                        }
                    }

                    // Recover encryption keys link
                    TextButton(
                        onClick = onRecoverKeys,
                        modifier = Modifier.fillMaxWidth(),
                        colors = ButtonDefaults.textButtonColors(
                            contentColor = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                        )
                    ) {
                        Icon(
                            imageVector = Icons.Default.Lock,
                            contentDescription = "Key recovery",
                            modifier = Modifier.size(16.dp)
                        )
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(
                            text = "Recover encryption keys",
                            style = MaterialTheme.typography.bodySmall
                        )
                    }
                }
            }
            
            // Register link
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Text(
                    text = "Don't have an account?",
                    style = MaterialTheme.typography.bodyMedium
                )
                TextButton(
                    onClick = onRegister,
                    colors = ButtonDefaults.textButtonColors(
                        contentColor = AccentColor
                    )
                ) {
                    Text(
                        text = "Register",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.SemiBold
                    )
                }
            }
            
            // Terms and privacy
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Text(
                        text = "By continuing, you agree to our",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                    TextButton(
                        onClick = { /* Navigate to terms */ },
                        contentPadding = PaddingValues(4.dp)
                    ) {
                        Text(
                            text = "Terms of Service",
                            style = MaterialTheme.typography.bodySmall,
                            fontWeight = FontWeight.Medium
                        )
                    }
                }
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    Text(
                        text = "and",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                    TextButton(
                        onClick = { /* Navigate to privacy */ },
                        contentPadding = PaddingValues(4.dp)
                    ) {
                        Text(
                            text = "Privacy Policy",
                            style = MaterialTheme.typography.bodySmall,
                            fontWeight = FontWeight.Medium
                        )
                    }
                }
            }
            
            // Bottom spacing
            Spacer(modifier = Modifier.weight(2f))
            
            // Version
            Text(
                text = "Version $APP_VERSION",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f)
            )
        }
    }
}

@Composable
private fun Logo() {
    Box(
        modifier = Modifier.size(100.dp),
        contentAlignment = Alignment.Center
    ) {
        Box(
            modifier = Modifier
                .size(80.dp)
                .clip(CircleShape)
                .background(AccentColor.copy(alpha = 0.1f)),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = "🛡️",
                fontSize = 40.sp
            )
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun LoginScreenPreview() {
    ArmorClawTheme {
        LoginScreen(
            onLogin = { _, _ -> },
            onRegister = {},
            onForgotPassword = {},
            onBiometricLogin = {},
            onRecoverKeys = {},
            isLoading = false,
            isBiometricLoading = false,
            errorMessage = null,
            syncStatusViewModel = null,
            onRetry = {}
        )
    }
}

@Preview(showBackground = true)
@Composable
private fun LoginScreenLoadingPreview() {
    ArmorClawTheme {
        LoginScreen(
            onLogin = { _, _ -> },
            onRegister = {},
            onForgotPassword = {},
            onBiometricLogin = {},
            onRecoverKeys = {},
            isLoading = true,
            isBiometricLoading = false,
            errorMessage = null
        )
    }
}

@Preview(showBackground = true)
@Composable
private fun LoginScreenErrorPreview() {
    ArmorClawTheme {
        LoginScreen(
            onLogin = { _, _ -> },
            onRegister = {},
            onForgotPassword = {},
            onBiometricLogin = {},
            onRecoverKeys = {},
            isLoading = false,
            isBiometricLoading = false,
            errorMessage = "Invalid username or password. Please try again."
        )
    }
}
