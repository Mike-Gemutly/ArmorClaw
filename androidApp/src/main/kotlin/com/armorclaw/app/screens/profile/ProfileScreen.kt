package com.armorclaw.app.screens.profile
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.koin.androidx.compose.koinViewModel
import com.armorclaw.app.viewmodels.ProfileViewModel
import com.armorclaw.app.viewmodels.ProfileUiState
import com.armorclaw.app.viewmodels.SyncStatusViewModel
import com.armorclaw.app.components.sync.SyncStatusWrapper
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor

/**
 * Profile screen for user account management
 *
 * This screen allows users to view and edit their profile,
 * manage account settings, and view account information.
 *
 * Uses ProfileViewModel for state management, so state survives
 * configuration changes.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(
    onNavigateBack: () -> Unit,
    onNavigateToSettings: () -> Unit,
    onNavigateToChangePassword: () -> Unit = {},
    onNavigateToChangePhone: () -> Unit = {},
    onNavigateToEditBio: () -> Unit = {},
    onNavigateToDeleteAccount: () -> Unit = {},
    onNavigateToLogin: () -> Unit = {},
    viewModel: ProfileViewModel = koinViewModel(),
    syncStatusViewModel: SyncStatusViewModel = koinViewModel(),
    modifier: Modifier = Modifier
) {
    val scrollState = rememberScrollState()
    val uiState by viewModel.uiState.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }
    
    // Connect ProfileViewModel errors to SyncStatusViewModel
    LaunchedEffect(uiState.error) {
        uiState.error?.let { error ->
            syncStatusViewModel.setError(error)
        }
    }
    
    // Handle error recovery retry
    val onRetry = {
        viewModel.clearError()
        syncStatusViewModel.clearError()
        viewModel.saveProfile()
    }

    // Handle logout success - navigate to login
    LaunchedEffect(uiState.logoutSuccess) {
        if (uiState.logoutSuccess) {
            onNavigateToLogin()
            viewModel.resetLogoutState()
        }
    }

    // Handle error display
    uiState.error?.let { error ->
        LaunchedEffect(error) {
            snackbarHostState.showSnackbar(
                message = error,
                duration = SnackbarDuration.Short
            )
            viewModel.clearError()
        }
    }

    // Profile state from ViewModel
    val userName = uiState.userName
    val userEmail = uiState.userEmail
    val userStatus = uiState.userStatus
    val userAvatar = uiState.userAvatar
    val isEditing = uiState.isEditing
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Profile") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Default.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    if (isEditing) {
                        TextButton(
                            onClick = {
                                viewModel.saveProfile()
                            }
                        ) {
                            Text(
                                text = "Save",
                                fontWeight = FontWeight.SemiBold,
                                color = AccentColor
                            )
                        }
                    } else {
                        IconButton(onClick = { viewModel.toggleEditMode() }) {
                            Icon(
                                imageVector = Icons.Default.Edit,
                                contentDescription = "Edit"
                            )
                        }
                    }
                    
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(
                            imageVector = Icons.Default.Settings,
                            contentDescription = "Settings"
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = SurfaceColor
                )
            )
        },
        snackbarHost = {
            SnackbarHost(hostState = snackbarHostState)
        },
        modifier = modifier
    ) { paddingValues ->
        // Add SyncStatusWrapper at the top of the content
        SyncStatusWrapper(
            viewModel = syncStatusViewModel,
            onRetry = onRetry,
            modifier = Modifier.fillMaxWidth()
        )
        
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(scrollState)
        ) {
            // Avatar section
            AvatarSection(
                avatar = userAvatar,
                name = userName,
                status = userStatus,
                isEditing = isEditing,
                onChangeAvatar = { viewModel.changeAvatar() }
            )
            
            Spacer(modifier = Modifier.height(32.dp))
            
            // Profile information
            ProfileInformation(
                userName = userName,
                userEmail = userEmail,
                userStatus = userStatus,
                isEditing = isEditing,
                onNameChange = { viewModel.updateName(it) },
                onEmailChange = { viewModel.updateEmail(it) },
                onStatusChange = { viewModel.updateStatus(it) }
            )
            
            Spacer(modifier = Modifier.height(32.dp))

            // Account options
            AccountOptions(
                onChangePassword = onNavigateToChangePassword,
                onChangePhone = onNavigateToChangePhone,
                onChangeBio = onNavigateToEditBio,
                onDeleteAccount = onNavigateToDeleteAccount
            )

            Spacer(modifier = Modifier.height(32.dp))

            // Privacy settings
            PrivacySettings(
                onNavigateToPrivacy = onNavigateToSettings,
                onNavigateToData = onNavigateToSettings
            )

            Spacer(modifier = Modifier.height(16.dp))

            // Logout button
            LogoutButton(
                onLogout = { viewModel.logout() },
                isLoading = uiState.isLoggingOut
            )
        }
    }
}

@Composable
private fun AvatarSection(
    avatar: String?,
    name: String,
    status: String,
    isEditing: Boolean,
    onChangeAvatar: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Avatar
        Box(
            modifier = Modifier.size(120.dp)
        ) {
            if (avatar != null) {
                // TODO: Load actual avatar image
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Text(text = "📷", fontSize = 48.sp)
                }
            } else {
                // Initial avatar
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = name.firstOrNull()?.toString() ?: "?",
                        fontSize = 48.sp,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                }
            }
            
            // Edit overlay
            if (isEditing) {
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(AccentColor)
                        .align(Alignment.BottomEnd)
                        .clickable(onClick = onChangeAvatar),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = Icons.Default.CameraAlt,
                        contentDescription = "Change avatar",
                        tint = Color.White,
                        modifier = Modifier.size(20.dp)
                    )
                }
            }
        }
        
        // Status indicator
        StatusIndicator(status = status)
        
        // Status text
        if (isEditing) {
            StatusDropdown(
                currentStatus = status,
                onStatusChange = {}
            )
        }
    }
}

@Composable
private fun StatusIndicator(
    status: String,
    modifier: Modifier = Modifier
) {
    val statusColor = when (status) {
        "Online" -> Color(0xFF4CAF50)
        "Away" -> Color(0xFFFF9800)
        "Busy" -> Color(0xFFF44336)
        "Invisible" -> Color(0xFF9E9E9E)
        else -> Color(0xFF4CAF50)
    }
    
    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Box(
            modifier = Modifier
                .size(12.dp)
                .background(statusColor, CircleShape)
        )
        Text(
            text = status,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
        )
    }
}

@Composable
private fun StatusDropdown(
    currentStatus: String,
    onStatusChange: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    var expanded by remember { mutableStateOf(false) }
    
    val statuses = listOf("Online", "Away", "Busy", "Invisible")
    
    Box(modifier = modifier) {
        OutlinedButton(
            onClick = { expanded = !expanded },
            colors = ButtonDefaults.outlinedButtonColors(
                contentColor = MaterialTheme.colorScheme.onSurface
            )
        ) {
            Text(currentStatus)
            Spacer(modifier = Modifier.width(8.dp))
            Icon(
                imageVector = Icons.Default.ExpandMore,
                contentDescription = null,
                modifier = Modifier.size(18.dp)
            )
        }
        
        // Dropdown menu (simplified)
        if (expanded) {
            DropdownMenu(
                expanded = expanded,
                onDismissRequest = { expanded = false }
            ) {
                statuses.forEach { status ->
                    DropdownMenuItem(
                        text = { Text(status) },
                        onClick = {
                            onStatusChange(status)
                            expanded = false
                        }
                    )
                }
            }
        }
    }
}

@Composable
private fun ProfileInformation(
    userName: String,
    userEmail: String,
    userStatus: String,
    isEditing: Boolean,
    onNameChange: (String) -> Unit,
    onEmailChange: (String) -> Unit,
    onStatusChange: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 32.dp),
        verticalArrangement = Arrangement.spacedBy(24.dp)
    ) {
        // Name field
        ProfileField(
            label = "Display Name",
            value = userName,
            isEditing = isEditing,
            onValueChange = onNameChange,
            placeholder = "Enter your name"
        )
        
        // Email field
        ProfileField(
            label = "Email Address",
            value = userEmail,
            isEditing = isEditing,
            onValueChange = onEmailChange,
            placeholder = "Enter your email",
            keyboardType = androidx.compose.ui.text.input.KeyboardType.Email
        )
        
        // Status field (read-only)
        ProfileField(
            label = "Status",
            value = userStatus,
            isEditing = false,
            onValueChange = {},
            placeholder = ""
        )
    }
}

@Composable
private fun ProfileField(
    label: String,
    value: String,
    isEditing: Boolean,
    onValueChange: (String) -> Unit,
    placeholder: String = "",
    keyboardType: androidx.compose.ui.text.input.KeyboardType = androidx.compose.ui.text.input.KeyboardType.Text
) {
    Column(
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
        )
        
        if (isEditing) {
            OutlinedTextField(
                value = value,
                onValueChange = onValueChange,
                placeholder = { Text(placeholder) },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = keyboardType),
                shape = RoundedCornerShape(12.dp)
            )
        } else {
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Text(
                    text = value.ifEmpty { "Not set" },
                    modifier = Modifier.padding(16.dp),
                    style = MaterialTheme.typography.bodyLarge
                )
            }
        }
    }
}

@Composable
private fun AccountOptions(
    onChangePassword: () -> Unit,
    onChangePhone: () -> Unit,
    onChangeBio: () -> Unit,
    onDeleteAccount: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 32.dp),
        verticalArrangement = Arrangement.spacedBy(1.dp)
    ) {
        Text(
            text = "Account",
            style = MaterialTheme.typography.labelLarge,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(vertical = 8.dp)
        )
        
        AccountOption(
            icon = Icons.Default.Lock,
            title = "Change Password",
            description = "Update your password",
            onClick = onChangePassword
        )
        
        AccountOption(
            icon = Icons.Default.Phone,
            title = "Change Phone Number",
            description = "Add or update your phone",
            onClick = onChangePhone
        )
        
        AccountOption(
            icon = Icons.Default.EditNote,
            title = "Edit Bio",
            description = "Add information about yourself",
            onClick = onChangeBio
        )
        
        AccountOption(
            icon = Icons.Default.Delete,
            title = "Delete Account",
            description = "Permanently delete your account",
            onClick = onDeleteAccount,
            isDanger = true
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AccountOption(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    title: String,
    description: String,
    onClick: () -> Unit,
    isDanger: Boolean = false
) {
    Card(
        onClick = onClick,
        colors = CardDefaults.cardColors(
            containerColor = if (isDanger)
                MaterialTheme.colorScheme.errorContainer
            else
                MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = if (isDanger)
                    MaterialTheme.colorScheme.onErrorContainer
                else
                    AccentColor
            )
            
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(2.dp)
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium
                )
                Text(
                    text = description,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                )
            }
            
            Icon(
                imageVector = Icons.Default.ChevronRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
            )
        }
    }
}

@Composable
private fun PrivacySettings(
    onNavigateToPrivacy: () -> Unit,
    onNavigateToData: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 32.dp),
        verticalArrangement = Arrangement.spacedBy(1.dp)
    ) {
        Text(
            text = "Privacy",
            style = MaterialTheme.typography.labelLarge,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(vertical = 8.dp)
        )
        
        AccountOption(
            icon = Icons.Default.Security,
            title = "Privacy Policy",
            description = "Learn how we protect your data",
            onClick = onNavigateToPrivacy
        )
        
        AccountOption(
            icon = Icons.Default.Storage,
            title = "My Data",
            description = "Download your data",
            onClick = onNavigateToData
        )
    }
}

@Composable
private fun LogoutButton(
    onLogout: () -> Unit,
    isLoading: Boolean = false,
    modifier: Modifier = Modifier
) {
    Button(
        onClick = onLogout,
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 32.dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = MaterialTheme.colorScheme.errorContainer,
            contentColor = MaterialTheme.colorScheme.onErrorContainer
        ),
        shape = RoundedCornerShape(12.dp),
        enabled = !isLoading
    ) {
        if (isLoading) {
            CircularProgressIndicator(
                modifier = Modifier.size(24.dp),
                color = MaterialTheme.colorScheme.onErrorContainer,
                strokeWidth = 2.dp
            )
        } else {
            Text(
                text = "Log Out",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(12.dp)
            )
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun ProfileScreenPreview() {
    ArmorClawTheme {
        ProfileScreen(
            onNavigateBack = {},
            onNavigateToSettings = {}
        )
    }
}
