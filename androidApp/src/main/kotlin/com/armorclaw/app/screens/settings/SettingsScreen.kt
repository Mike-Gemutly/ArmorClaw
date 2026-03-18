package com.armorclaw.app.screens.settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.material3.MaterialTheme
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor
import com.armorclaw.shared.ui.components.SealedIndicator
import com.armorclaw.shared.domain.model.KeystoreStatus
import com.armorclaw.app.viewmodels.SettingsViewModel
import com.armorclaw.app.viewmodels.SettingsUiState
import org.koin.androidx.compose.koinViewModel
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SnackbarResult
import androidx.compose.ui.platform.LocalContext

/**
 * Settings screen for app configuration
 *
 * This screen allows users to configure app settings,
 * notifications, security, and account.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    onNavigateBack: () -> Unit,
    onNavigateToProfile: () -> Unit,
    onNavigateToSecurity: () -> Unit,
    onNavigateToNotifications: () -> Unit,
    onNavigateToAppearance: () -> Unit,
    onNavigateToPrivacy: () -> Unit,
    onNavigateToAbout: () -> Unit,
    onNavigateToMyData: () -> Unit = {},
    onNavigateToReportBug: () -> Unit = {},
    onNavigateToDevices: () -> Unit = {},
    onNavigateToDataSafety: () -> Unit = {},
    onNavigateToAgents: () -> Unit = {},
    onNavigateToApprovals: () -> Unit = {},
    onNavigateToServerConnection: () -> Unit = {},
    onNavigateToInvite: () -> Unit = {},
    onNavigateToRateApp: () -> Unit = {},
    onNavigateToUnseal: () -> Unit = {},
    onNavigateToLogin: () -> Unit = {},
    onNavigateToRegister: () -> Unit = {},
    onLogout: () -> Unit,
    viewModel: SettingsViewModel = koinViewModel(),
    snackbarHostState: SnackbarHostState = remember { SnackbarHostState() },
    isLoggedIn: Boolean = false,
    loggedInUserName: String? = null,
    loggedInUserEmail: String? = null,
    keystoreStatus: KeystoreStatus = KeystoreStatus.Sealed(),
    onResealKeystore: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val scrollState = rememberScrollState()
    
    val userName = loggedInUserName ?: ""
    val userEmail = loggedInUserEmail ?: ""
    val userAvatar: String? = null
    
    // Settings values
    val notificationsEnabled = remember { mutableStateOf(true) }
    val soundEnabled = remember { mutableStateOf(true) }
    val vibrationEnabled = remember { mutableStateOf(true) }
    val darkMode = remember { mutableStateOf(true) }
    val biometricAuth = remember { mutableStateOf(true) }
    
    // Observe UI state for error handling
    val uiState by viewModel.uiState.collectAsState()
    
    // Handle logout errors
    LaunchedEffect(uiState) {
        when (uiState) {
            is SettingsUiState.LogoutError -> {
                val errorMessage = (uiState as SettingsUiState.LogoutError).message
                snackbarHostState.showSnackbar(
                    message = errorMessage,
                    actionLabel = "Retry",
                    duration = SnackbarDuration.Long
                ).let { result ->
                    if (result == SnackbarResult.ActionPerformed) {
                        viewModel.resetState()
                        onLogout()
                    }
                }
            }
            is SettingsUiState.LogoutSuccess -> {
                snackbarHostState.showSnackbar(
                    message = "Logout successful",
                    duration = SnackbarDuration.Short
                )
                viewModel.resetState()
            }
            else -> {}
        }
    }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Settings") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Default.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = SurfaceColor
                )
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(scrollState)
        ) {
            if (isLoggedIn) {
                // User profile section
                ProfileSection(
                    userName = userName,
                    userEmail = userEmail,
                    userAvatar = userAvatar,
                    onClick = onNavigateToProfile
                )
            } else {
                // Login / Create Account section
                LoginPromptSection(
                    onLogin = onNavigateToLogin,
                    onRegister = onNavigateToRegister
                )
            }
            
            // App settings section
            SettingsSection(
                title = "App Settings",
                items = listOf(
                    SettingItem(
                        icon = Icons.Default.Notifications,
                        title = "Notifications",
                        description = "Manage push notifications",
                        value = notificationsEnabled.value,
                        onToggle = { notificationsEnabled.value = it }
                    ),
                    SettingItem(
                        icon = Icons.Default.Palette,
                        title = "Appearance",
                        description = "Theme and display settings",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToAppearance
                    ),
                    SettingItem(
                        icon = Icons.Default.Lock,
                        title = "Security",
                        description = "Biometric auth, encryption, devices",
                        value = biometricAuth.value,
                        onToggle = { biometricAuth.value = it },
                        onClick = onNavigateToSecurity
                    ),
                    SettingItem(
                        icon = Icons.Default.Devices,
                        title = "Devices",
                    description = "Manage logged in devices",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToDevices
                    ),
                    SettingItem(
                        icon = Icons.Default.Dns,
                        title = "Server Connection",
                        description = "Reconnect or change server",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToServerConnection
                    )
                )
            )

            // Keystore Status Section (Phase 2)
            Column(
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
            ) {
                Text(
                    text = "VPS Keystore",
                    style = MaterialTheme.typography.labelLarge,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                )
                SealedIndicator(
                    status = keystoreStatus,
                    onUnsealClick = onNavigateToUnseal,
                    onResealClick = onResealKeystore,
                    modifier = Modifier.fillMaxWidth()
                )
            }

            // Privacy section
            SettingsSection(
                title = "Privacy",
                items = listOf(
                    SettingItem(
                        icon = Icons.Default.Security,
                        title = "Privacy Policy",
                        description = "Learn how we protect your data",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToPrivacy
                    ),
                    SettingItem(
                        icon = Icons.Default.VerifiedUser,
                        title = "Data Safety",
                        description = "How your data is handled",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToDataSafety
                    ),
                    SettingItem(
                        icon = Icons.Default.Storage,
                        title = "Data & Storage",
                        description = "Manage your data and storage",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToMyData
                    )
                )
            )

            // Invite section (NEW - for viral growth)
            SettingsSection(
                title = "Invite",
                items = listOf(
                    SettingItem(
                        icon = Icons.Default.PersonAdd,
                        title = "Invite to ArmorClaw",
                        description = "Share secure chat with friends",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToInvite
                    )
                )
            )

            // AI & Agents section (NEW)
            SettingsSection(
                title = "AI & Agents",
                items = listOf(
                    SettingItem(
                        icon = Icons.Default.SmartToy,
                        title = "Agent Management",
                        description = "View and manage AI agents",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToAgents
                    ),
                    SettingItem(
                        icon = Icons.Default.PendingActions,
                        title = "Pending Approvals",
                        description = "Review AI action requests",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToApprovals
                    )
                )
            )

            // About section
            SettingsSection(
                title = "About",
                items = listOf(
                    SettingItem(
                        icon = Icons.Default.Info,
                        title = "About ArmorClaw",
                        description = "Version 1.0.0",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToAbout
                    ),
                    SettingItem(
                        icon = Icons.Default.BugReport,
                        title = "Report a Bug",
                        description = "Help us improve the app",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToReportBug
                    ),
                    SettingItem(
                        icon = Icons.Default.Favorite,
                        title = "Rate App",
                        description = "Leave a review on Play Store",
                        value = null,
                        onToggle = null,
                        onClick = onNavigateToRateApp
                    )
                )
            )
            
            // Logout button (only when logged in)
            if (isLoggedIn) {
                LogoutButton(
                    onClick = onLogout,
                    modifier = Modifier.padding(16.dp)
                )
            }
            
            // Version info
            VersionInfo(
                version = "1.0.0 (1)",
                modifier = Modifier.padding(16.dp)
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ProfileSection(
    userName: String,
    userEmail: String,
    userAvatar: String?,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        onClick = onClick,
        modifier = modifier
            .fillMaxWidth()
            .padding(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Avatar
            Box(
                modifier = Modifier
                    .size(64.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primaryContainer),
                contentAlignment = Alignment.Center
            ) {
                if (userAvatar != null) {
                    // TODO: Load actual avatar
                    Text(text = "📷", fontSize = 32.sp)
                } else {
                    Text(
                        text = userName.firstOrNull()?.toString() ?: "?",
                        fontSize = 32.sp,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                }
            }
            
            // User info
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Text(
                    text = userName,
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.SemiBold
                )
                Text(
                    text = userEmail,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                )
            }
            
            // Chevron
            Icon(
                imageVector = Icons.Default.ChevronRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
            )
        }
    }
}

@Composable
private fun LoginPromptSection(
    onLogin: () -> Unit,
    onRegister: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Icon(
                imageVector = Icons.Default.AccountCircle,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                modifier = Modifier.size(64.dp)
            )
            Text(
                text = "Sign in to access your profile",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                OutlinedButton(
                    onClick = onRegister,
                    shape = RoundedCornerShape(12.dp)
                ) {
                    Text("Create Account")
                }
                Button(
                    onClick = onLogin,
                    shape = RoundedCornerShape(12.dp),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = AccentColor
                    )
                ) {
                    Text("Log In")
                }
            }
        }
    }
}

@Composable
private fun SettingsSection(
    title: String,
    items: List<SettingItem>,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier.padding(vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // Section title
        Text(
            text = title,
            style = MaterialTheme.typography.labelLarge,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(horizontal = 32.dp, vertical = 8.dp)
        )
        
        // Setting items
        Column(
            modifier = Modifier.padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(1.dp)
        ) {
            items.forEach { item ->
                SettingItemCard(item)
            }
        }
    }
}

@Composable
private fun SettingItemCard(
    item: SettingItem
) {
    Surface(
        shape = RoundedCornerShape(8.dp),
        color = MaterialTheme.colorScheme.surfaceVariant,
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(enabled = item.onClick != null) {
                    item.onClick?.invoke()
                }
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Icon
            Icon(
                imageVector = item.icon,
                contentDescription = null,
                tint = AccentColor,
                modifier = Modifier.size(24.dp)
            )
            
            // Info
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(2.dp)
            ) {
                Text(
                    text = item.title,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium
                )
                if (item.description != null) {
                    Text(
                        text = item.description,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
            }
            
            // Toggle switch
            if (item.value != null) {
                Switch(
                    checked = item.value,
                    onCheckedChange = { checked ->
                        item.onToggle?.invoke(checked)
                    }
                )
            } else {
                // Chevron
                Icon(
                    imageVector = Icons.Default.ChevronRight,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
                    modifier = Modifier.size(20.dp)
                )
            }
        }
    }
}

@Composable
private fun LogoutButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Button(
        onClick = onClick,
        modifier = modifier.fillMaxWidth(),
        colors = ButtonDefaults.buttonColors(
            containerColor = MaterialTheme.colorScheme.errorContainer,
            contentColor = MaterialTheme.colorScheme.onErrorContainer
        ),
        shape = RoundedCornerShape(12.dp)
    ) {
        Text(
            text = "Log Out",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(8.dp)
        )
    }
}

@Composable
private fun VersionInfo(
    version: String,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier,
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = "ArmorClaw",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold
        )
        Text(
            text = version,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
        )
        Text(
            text = "Made with ❤️ for privacy",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
        )
    }
}

/**
 * Setting item data class
 */
data class SettingItem(
    val icon: ImageVector,
    val title: String,
    val description: String?,
    val value: Boolean?,
    val onToggle: ((Boolean) -> Unit)?,
    val onClick: (() -> Unit)? = null
)

@Preview(showBackground = true, name = "Settings - Logged Out")
@Composable
private fun SettingsScreenLoggedOutPreview() {
    ArmorClawTheme {
        SettingsScreen(
            onNavigateBack = {},
            onNavigateToProfile = {},
            onNavigateToSecurity = {},
            onNavigateToNotifications = {},
            onNavigateToAppearance = {},
            onNavigateToPrivacy = {},
            onNavigateToAbout = {},
            onLogout = {},
            isLoggedIn = false,
            keystoreStatus = KeystoreStatus.Sealed()
        )
    }
}

@Preview(showBackground = true, name = "Settings - Logged In")
@Composable
private fun SettingsScreenLoggedInPreview() {
    ArmorClawTheme {
        SettingsScreen(
            onNavigateBack = {},
            onNavigateToProfile = {},
            onNavigateToSecurity = {},
            onNavigateToNotifications = {},
            onNavigateToAppearance = {},
            onNavigateToPrivacy = {},
            onNavigateToAbout = {},
            onLogout = {},
            isLoggedIn = true,
            loggedInUserName = "Alice",
            loggedInUserEmail = "alice@example.com",
            keystoreStatus = KeystoreStatus.Sealed()
        )
    }
}
