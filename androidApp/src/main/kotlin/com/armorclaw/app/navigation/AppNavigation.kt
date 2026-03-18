package com.armorclaw.app.navigation

import androidx.compose.animation.*
import androidx.compose.animation.core.tween
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.navigation.*
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import com.armorclaw.app.screens.auth.ForgotPasswordScreen
import com.armorclaw.app.screens.auth.KeyRecoveryScreen
import com.armorclaw.app.screens.auth.LoginScreen
import com.armorclaw.app.screens.auth.RegistrationScreen
import com.armorclaw.app.screens.chat.ChatScreenEnhanced
import com.armorclaw.app.screens.home.HomeScreen
import com.armorclaw.app.screens.onboarding.*
import com.armorclaw.app.screens.profile.ChangePasswordScreen
import com.armorclaw.app.screens.profile.ChangePhoneNumberScreen
import com.armorclaw.app.screens.profile.DeleteAccountScreen
import com.armorclaw.app.screens.profile.EditBioScreen
import com.armorclaw.app.screens.profile.ProfileScreen
import com.armorclaw.app.screens.profile.UserProfileScreen
import com.armorclaw.app.screens.profile.SharedRoomsScreen
import com.armorclaw.app.screens.room.RoomDetailsScreen
import com.armorclaw.app.screens.room.RoomManagementScreen
import com.armorclaw.app.screens.room.RoomSettingsScreen
import com.armorclaw.app.screens.search.SearchScreen
import com.armorclaw.app.screens.settings.AboutScreen
import com.armorclaw.app.screens.settings.AppearanceSettingsScreen
import com.armorclaw.app.screens.settings.DataSafetyScreen
import com.armorclaw.app.screens.settings.EmojiVerificationScreen
import com.armorclaw.app.screens.settings.MyDataScreen
import com.armorclaw.app.screens.settings.NotificationSettingsScreen
import com.armorclaw.app.screens.settings.PrivacyPolicyScreen
import com.armorclaw.app.screens.settings.ReportBugScreen
import com.armorclaw.app.screens.settings.SecuritySettingsScreen
import com.armorclaw.app.screens.settings.SettingsScreen
import com.armorclaw.app.screens.settings.DeviceListScreen
import com.armorclaw.app.screens.settings.AddDeviceScreen
import com.armorclaw.app.screens.settings.AgentManagementScreen
import com.armorclaw.app.screens.settings.HitlApprovalScreen
import com.armorclaw.app.screens.keystore.UnsealScreen
import com.armorclaw.app.screens.call.ActiveCallScreen
import com.armorclaw.app.screens.call.IncomingCallDialog
import com.armorclaw.app.screens.chat.ThreadViewScreen
import com.armorclaw.app.screens.media.ImageViewerScreen
import com.armorclaw.app.screens.media.FilePreviewScreen
import com.armorclaw.app.screens.splash.SplashScreen
import com.armorclaw.app.screens.vault.VaultScreen
import com.armorclaw.app.screens.studio.AgentStudioScreen
import com.armorclaw.app.viewmodels.DeviceListViewModel
import com.armorclaw.app.viewmodels.SettingsViewModel
import com.armorclaw.app.viewmodels.SettingsUiState
import com.armorclaw.app.viewmodels.AppPreferences
import com.armorclaw.app.viewmodels.ChatViewModel
import com.armorclaw.app.viewmodels.SplashViewModel
import com.armorclaw.app.viewmodels.SplashTarget
import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.data.store.ControlPlaneStore
import org.koin.compose.koinInject

/**
 * App navigation configuration
 *
 * This class defines all navigation routes and handles
 * navigation between screens in the app.
 *
 * Total Routes: 55
 */
object AppNavigation {
    
    // ==================== CORE ROUTES ====================
    
    /**
     * Splash screen route
     */
    const val SPLASH = "splash"
    
    // ==================== ONBOARDING ROUTES ====================
    
    /**
     * Welcome screen route
     */
    const val WELCOME = "welcome"

    /**
     * Migration screen route (v2.5 → v3.0 upgrade)
     */
    const val MIGRATION = "migration"

    /**
     * Key backup setup screen route
     */
    const val KEY_BACKUP_SETUP = "key_backup_setup"
    
    /**
     * Security explanation screen route
     */
    const val SECURITY = "security"
    
    /**
     * Connect server screen route
     */
    const val CONNECT = "connect"
    
    /**
     * Permissions screen route
     */
    const val PERMISSIONS = "permissions"
    
    /**
     * Completion screen route
     */
    const val COMPLETION = "completion"

    /**
     * Tutorial screen route
     */
    const val TUTORIAL = "tutorial"

    /**
     * Onboarding config deep link route (for QR provisioning)
     */
    const val ONBOARDING_CONFIG = "onboarding/config"

    /**
     * Onboarding setup deep link route (for token-based setup)
     */
    const val ONBOARDING_SETUP = "onboarding/setup"

    /**
     * Onboarding invite deep link route (for invite codes)
     */
    const val ONBOARDING_INVITE = "onboarding/invite"
    
    // ==================== AUTHENTICATION ROUTES ====================
    
    /**
     * Login screen route
     */
    const val LOGIN = "login"
    
    /**
     * Forgot password screen route
     */
    const val FORGOT_PASSWORD = "forgot_password"
    
    /**
     * Registration screen route
     */
    const val REGISTRATION = "registration"
    
    // ==================== MAIN APP ROUTES ====================
    
    /**
     * Home screen route
     */
    const val HOME = "home"
    
    /**
     * Chat screen route with roomId parameter
     */
    const val CHAT = "chat/{roomId}"
    
    /**
     * Profile screen route
     */
    const val PROFILE = "profile"
    
    /**
     * Settings screen route
     */
    const val SETTINGS = "settings"
    
    /**
     * Vault screen route
     */
    const val VAULT = "vault"
    
    /**
     * Agent Studio screen route
     */
    const val AGENT_STUDIO = "agent_studio"
    
    // ==================== ROOM ROUTES ====================
    
    /**
     * Room management screen route
     */
    const val ROOM_MANAGEMENT = "room_management"
    
    /**
     * Room details screen route with roomId parameter
     */
    const val ROOM_DETAILS = "room_details/{roomId}"
    
    /**
     * Room settings screen route with roomId parameter
     */
    const val ROOM_SETTINGS = "room_settings/{roomId}"
    
    // ==================== SEARCH ROUTES ====================
    
    /**
     * Search screen route
     */
    const val SEARCH = "search"
    
    // ==================== PROFILE ROUTES ====================
    
    /**
     * Change password screen route
     */
    const val CHANGE_PASSWORD = "change_password"
    
    /**
     * Change phone number screen route
     */
    const val CHANGE_PHONE = "change_phone"
    
    /**
     * Edit bio screen route
     */
    const val EDIT_BIO = "edit_bio"
    
    /**
     * Delete account screen route
     */
    const val DELETE_ACCOUNT = "delete_account"
    
    // ==================== SETTINGS ROUTES ====================
    
    /**
     * Security settings screen route
     */
    const val SECURITY_SETTINGS = "security_settings"
    
    /**
     * Notification settings screen route
     */
    const val NOTIFICATION_SETTINGS = "notification_settings"
    
    /**
     * Appearance settings screen route
     */
    const val APPEARANCE = "appearance"
    
    /**
     * Privacy policy screen route
     */
    const val PRIVACY_POLICY = "privacy_policy"
    
    /**
     * My data screen route
     */
    const val MY_DATA = "my_data"

    /**
     * Data safety screen route
     */
    const val DATA_SAFETY = "data_safety"

    /**
     * About screen route
     */
    const val ABOUT = "about"

    /**
     * Report bug screen route
     */
    const val REPORT_BUG = "report_bug"

    /**
     * Invite screen route (for generating QR codes)
     */
    const val INVITE = "invite"

    /**
     * Server connection screen route
     * Used for post-onboarding discovery activation
     */
    const val SERVER_CONNECTION = "settings/server_connection"

    // ==================== AGENT & WORKFLOW ROUTES (NEW) ====================

    /**
     * Agent management screen route
     */
    const val AGENT_MANAGEMENT = "settings/agents"

    /**
     * HITL approval screen route
     */
    const val HITL_APPROVALS = "settings/approvals"

    /**
     * Workflow management screen route
     */
    const val WORKFLOW_MANAGEMENT = "settings/workflows"

    /**
     * Budget status screen route
     */
    const val BUDGET_STATUS = "settings/budget"

    // ==================== KEYSTORE ROUTES ====================

    /**
     * Keystore unseal screen route
     * Used when VPS keystore needs to be unsealed for credential access
     */
    const val KEYSTORE = "keystore"

    // ==================== SYNC & DEVICE ROUTES ====================

    /**
     * Device list screen route
     */
    const val DEVICES = "devices"

    /**
     * Add device screen route
     */
    const val ADD_DEVICE = "add_device"

    // ==================== VERIFICATION ROUTES ====================

    /**
     * Emoji verification screen route with deviceId parameter
     */
    const val EMOJI_VERIFICATION = "verification/{deviceId}"

    /**
     * Bridge verification screen route (launches emoji verification for bridge device)
     */
    const val BRIDGE_VERIFICATION = "bridge_verification/{deviceId}"

    /**
     * Helper function to create bridge verification route with deviceId
     */
    fun createBridgeVerificationRoute(deviceId: String): String {
        return BRIDGE_VERIFICATION.replace("{deviceId}", deviceId)
    }

    /**
     * Helper function to create verification route with deviceId
     */
    fun createVerificationRoute(deviceId: String): String {
        return EMOJI_VERIFICATION.replace("{deviceId}", deviceId)
    }

    // ==================== CALL ROUTES ====================

    /**
     * Active call screen route with callId parameter
     */
    const val ACTIVE_CALL = "call/{callId}"

    /**
     * Incoming call screen route with call arguments
     */
    const val INCOMING_CALL = "incoming_call/{callId}/{callerId}/{callerName}/{callType}"

    /**
     * Helper function to create active call route with callId
     */
    fun createCallRoute(callId: String): String {
        return ACTIVE_CALL.replace("{callId}", callId)
    }

    /**
     * Helper function to create incoming call route with arguments
     */
    fun createIncomingCallRoute(
        callId: String,
        callerId: String,
        callerName: String,
        callType: String = "voice"
    ): String {
        return INCOMING_CALL
            .replace("{callId}", callId)
            .replace("{callerId}", callerId)
            .replace("{callerName}", callerName)
            .replace("{callType}", callType)
    }

    // ==================== THREAD ROUTES ====================

    /**
     * Thread view screen route
     */
    const val THREAD = "thread/{roomId}/{rootMessageId}"

    /**
     * Helper function to create thread route
     */
    fun createThreadRoute(roomId: String, rootMessageId: String): String {
        return THREAD
            .replace("{roomId}", roomId)
            .replace("{rootMessageId}", rootMessageId)
    }

    // ==================== MEDIA ROUTES ====================

    /**
     * Image viewer screen route
     */
    const val IMAGE_VIEWER = "image/{imageId}"

    /**
     * File preview screen route
     */
    const val FILE_PREVIEW = "file/{fileId}"

    /**
     * Helper function to create image viewer route
     */
    fun createImageViewerRoute(imageId: String): String {
        return IMAGE_VIEWER.replace("{imageId}", imageId)
    }

    /**
     * Helper function to create file preview route
     */
    fun createFilePreviewRoute(fileId: String): String {
        return FILE_PREVIEW.replace("{fileId}", fileId)
    }

    // ==================== KEY RECOVERY ROUTE ====================

    /**
     * Key recovery screen route (recover encryption keys via recovery phrase)
     */
    const val KEY_RECOVERY = "key_recovery"

    // ==================== ADDITIONAL SETTINGS ROUTES ====================

    /**
     * Open source licenses screen route
     */
    const val LICENSES = "licenses"

    /**
     * Terms of service screen route
     */
    const val TERMS_OF_SERVICE = "terms"

    // ==================== USER PROFILE ROUTE ====================

    /**
     * User profile screen route
     */
    const val USER_PROFILE = "user/{userId}"

    /**
     * Shared rooms screen route
     */
    const val SHARED_ROOMS = "shared_rooms/{userId}"

    /**
     * Helper function to create user profile route
     */
    fun createUserProfileRoute(userId: String): String {
        return USER_PROFILE.replace("{userId}", userId)
    }

    /**
     * Helper function to create shared rooms route
     */
    fun createSharedRoomsRoute(userId: String): String {
        return SHARED_ROOMS.replace("{userId}", userId)
    }

    // ==================== HELPER FUNCTIONS ====================

    /**
     * Helper function to create chat route with roomId
     */
    fun createChatRoute(roomId: String): String {
        return CHAT.replace("{roomId}", roomId)
    }

    /**
     * Helper function to create room details route with roomId
     */
    fun createRoomDetailsRoute(roomId: String): String {
        return ROOM_DETAILS.replace("{roomId}", roomId)
    }

    /**
     * Helper function to create room settings route with roomId
     */
    fun createRoomSettingsRoute(roomId: String): String {
        return ROOM_SETTINGS.replace("{roomId}", roomId)
    }
}

/**
 * Animated fade transition
 */
private val FadeInAnimation = fadeIn(
    animationSpec = tween(durationMillis = 300)
)

private val FadeOutAnimation = fadeOut(
    animationSpec = tween(durationMillis = 300)
)

/**
 * Main app navigation host
 * 
 * This composable defines the navigation graph for the app,
 * including all screens and transitions between them.
 * 
 * Total Routes: 55
 */
@OptIn(ExperimentalAnimationApi::class)
@Composable
fun AppNavHost(
    navController: NavHostController,
    modifier: Modifier = Modifier,
    startDestination: String = AppNavigation.SPLASH
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val linkHandler = remember { com.armorclaw.app.util.ExternalLinkHandler(context) }

    NavHost(
        navController = navController,
        startDestination = startDestination,
        modifier = modifier,
        enterTransition = { FadeInAnimation },
        exitTransition = { FadeOutAnimation },
        popEnterTransition = { FadeInAnimation },
        popExitTransition = { FadeOutAnimation }
    ) {
        // ==================== SPLASH SCREEN ====================
        composable(AppNavigation.SPLASH) {
            val preferences: AppPreferences = koinInject()
            val splashViewModel = remember { SplashViewModel(preferences) }
            val target by splashViewModel.navigationTarget.collectAsState()

            // Observe SplashViewModel navigation target
            LaunchedEffect(target) {
                when (target) {
                    is SplashTarget.Connect, is SplashTarget.Onboarding -> {
                        navController.navigate(AppNavigation.CONNECT) {
                            popUpTo(AppNavigation.SPLASH) { inclusive = true }
                        }
                    }
                    is SplashTarget.Login -> {
                        navController.navigate(AppNavigation.LOGIN) {
                            popUpTo(AppNavigation.SPLASH) { inclusive = true }
                        }
                    }
                    is SplashTarget.Home -> {
                        navController.navigate(AppNavigation.HOME) {
                            popUpTo(AppNavigation.SPLASH) { inclusive = true }
                        }
                    }
                    is SplashTarget.Migration -> {
                        navController.navigate(AppNavigation.MIGRATION) {
                            popUpTo(AppNavigation.SPLASH) { inclusive = true }
                        }
                    }
                    is SplashTarget.KeyBackupSetup -> {
                        navController.navigate(AppNavigation.KEY_BACKUP_SETUP) {
                            popUpTo(AppNavigation.SPLASH) { inclusive = true }
                        }
                    }
                    is SplashTarget.DeepLink -> {
                        // Deep links: ensure Home is root, then navigate to target
                        navController.navigate(AppNavigation.HOME) {
                            popUpTo(AppNavigation.SPLASH) { inclusive = true }
                        }
                        when (val dl = target) {
                            is SplashTarget.DeepLink.Room -> {
                                navController.navigate("chat/${dl.roomId}")
                            }
                            is SplashTarget.DeepLink.User -> {
                                navController.navigate("user_profile/${dl.userId}")
                            }
                            is SplashTarget.DeepLink.Call -> {
                                navController.navigate("active_call/${dl.callId}")
                            }
                            is SplashTarget.DeepLink.Config -> {
                                navController.navigate("${AppNavigation.ONBOARDING_CONFIG}?d=${dl.configData}")
                            }
                            else -> {}
                        }
                    }
                    null -> { /* Still loading */ }
                }
            }

            SplashScreen(
                onNavigateToOnboarding = {},
                onNavigateToLogin = {},
                onNavigateToHome = {},
                onNavigateToConnect = {}
            )
        }
        
        // ==================== ONBOARDING FLOW ====================
        
        // Welcome screen
        composable(AppNavigation.WELCOME) {
            WelcomeScreen(
                onGetStarted = {
                    navController.navigate("${AppNavigation.SECURITY}/0")
                },
                onSkip = {
                    // Skip to home (for demo purposes)
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(AppNavigation.WELCOME) { inclusive = true }
                    }
                }
            )
        }
        
        // Security explanation screen (4 steps)
        composable(
            route = "${AppNavigation.SECURITY}/{step}",
            arguments = listOf(
                navArgument("step") { type = NavType.IntType; defaultValue = 0 }
            )
        ) {
            val currentStep = it.arguments?.getInt("step") ?: 0
            SecurityExplanationScreen(
                initialStep = currentStep,
                onNext = {
                    if (currentStep < 3) {
                        navController.navigate("${AppNavigation.SECURITY}/${currentStep + 1}")
                    } else {
                        navController.navigate(AppNavigation.CONNECT)
                    }
                },
                onBack = {
                    if (currentStep > 0) {
                        navController.popBackStack()
                    } else {
                        navController.navigate(AppNavigation.WELCOME)
                    }
                },
                onSkipToHome = {
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }

        // Connect server screen
        composable(AppNavigation.CONNECT) {
            ConnectServerScreen(
                onConnected = { _ ->
                    navController.navigate(AppNavigation.PERMISSIONS)
                },
                onBack = {
                    navController.popBackStack()
                }
            )
        }

        // Permissions screen
        composable(AppNavigation.PERMISSIONS) {
            PermissionsScreen(
                onComplete = {
                    navController.navigate(AppNavigation.COMPLETION)
                },
                onBack = {
                    navController.popBackStack()
                }
            )
        }
        
        // Completion screen
        composable(AppNavigation.COMPLETION) {
            val preferences: AppPreferences = koinInject()

            CompletionScreen(
                onStartChatting = {
                    // Mark onboarding as completed, then prompt for key backup
                    preferences.setOnboardingCompleted(true)
                    navController.navigate(AppNavigation.KEY_BACKUP_SETUP) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onTakeTutorial = {
                    // Mark onboarding as completed and navigate to tutorial
                    preferences.setOnboardingCompleted(true)
                    navController.navigate(AppNavigation.TUTORIAL) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }

        // Key Backup Setup screen (Bug 4 fix + force-quit bypass)
        composable(AppNavigation.KEY_BACKUP_SETUP) {
            val preferences: AppPreferences = koinInject()

            KeyBackupSetupScreen(
                onComplete = {
                    preferences.setBackupComplete(true)
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onSkip = {
                    // User explicitly chose to skip with scary dialog confirmation
                    // Do NOT mark backup as complete — they'll be prompted again next launch
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }

        // Migration screen (Bug 3 fix: v2.5 → v3.0 upgrade)
        composable(AppNavigation.MIGRATION) {
            val preferences: AppPreferences = koinInject()

            MigrationScreen(
                onMigrationComplete = {
                    preferences.clearLegacyBridgeSession()
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onSkipMigration = {
                    preferences.clearLegacyBridgeSession()
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onLogout = {
                    preferences.clearAll()
                    navController.navigate(AppNavigation.LOGIN) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }

        // Tutorial screen
        composable(AppNavigation.TUTORIAL) {
            TutorialScreen(
                onNavigateBack = { navController.popBackStack() },
                onComplete = {
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }

        // Onboarding config deep link (armorclaw://config?d=...)
        composable(
            route = "${AppNavigation.ONBOARDING_CONFIG}?d={data}",
            arguments = listOf(navArgument("data") { type = NavType.StringType; nullable = true })
        ) { backStackEntry ->
            val data = backStackEntry.arguments?.getString("data") ?: ""
            // Navigate to CONNECT with the QR data pre-loaded
            // The ConnectServerScreen will handle the QR provisioning
            val preferences: AppPreferences = koinInject()

            ConnectServerScreen(
                onConnected = { info ->
                    preferences.setOnboardingCompleted(true)
                    preferences.setLoggedIn(true)
                    preferences.setUserId(info.userId)
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onBack = {
                    navController.navigate(AppNavigation.CONNECT) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }

        // Onboarding setup deep link (armorclaw://setup?token=...&server=...)
        composable(
            route = "${AppNavigation.ONBOARDING_SETUP}?token={token}&server={server}",
            arguments = listOf(
                navArgument("token") { type = NavType.StringType; nullable = true },
                navArgument("server") { type = NavType.StringType; nullable = true }
            )
        ) { backStackEntry ->
            val token = backStackEntry.arguments?.getString("token") ?: ""
            val server = backStackEntry.arguments?.getString("server") ?: ""
            val preferences: AppPreferences = koinInject()

            ConnectServerScreen(
                onConnected = { info ->
                    preferences.setOnboardingCompleted(true)
                    preferences.setLoggedIn(true)
                    preferences.setUserId(info.userId)
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onBack = {
                    navController.navigate(AppNavigation.CONNECT) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }

        // Onboarding invite deep link (armorclaw://invite?code=...)
        composable(
            route = "${AppNavigation.ONBOARDING_INVITE}?code={code}",
            arguments = listOf(navArgument("code") { type = NavType.StringType; nullable = true })
        ) { backStackEntry ->
            val code = backStackEntry.arguments?.getString("code") ?: ""
            val preferences: AppPreferences = koinInject()

            ConnectServerScreen(
                onConnected = { info ->
                    preferences.setOnboardingCompleted(true)
                    preferences.setLoggedIn(true)
                    preferences.setUserId(info.userId)
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onBack = {
                    navController.navigate(AppNavigation.CONNECT) {
                        popUpTo(0) { inclusive = true }
                    }
                }
            )
        }
        
        // ==================== AUTHENTICATION FLOW ====================
        
        // Login screen
        composable(AppNavigation.LOGIN) {
            val preferences: AppPreferences = koinInject()

            LoginScreen(
                onLogin = { username, _ ->
                    // TODO: Implement actual login via ViewModel
                    // For now, save login state and navigate
                    preferences.setLoggedIn(true)
                    preferences.setUserId(username)
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onBiometricLogin = {
                    // TODO: Implement biometric login
                    preferences.setLoggedIn(true)
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onForgotPassword = {
                    navController.navigate(AppNavigation.FORGOT_PASSWORD)
                },
                onRegister = {
                    navController.navigate(AppNavigation.REGISTRATION)
                },
                onRecoverKeys = {
                    navController.navigate(AppNavigation.KEY_RECOVERY)
                }
            )
        }
        
        // Forgot password screen
        composable(AppNavigation.FORGOT_PASSWORD) {
            ForgotPasswordScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onResetPassword = { email ->
                    // TODO: Implement password reset
                    // Display success message
                }
            )
        }
        
        // Key Recovery screen
        composable(AppNavigation.KEY_RECOVERY) {
            val preferences: AppPreferences = koinInject()

            KeyRecoveryScreen(
                onRecoveryComplete = {
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(0) { inclusive = true }
                    }
                },
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }

        // Registration screen
        composable(AppNavigation.REGISTRATION) {
            RegistrationScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onRegister = { username, email, password ->
                    // TODO: Implement registration
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(AppNavigation.REGISTRATION) { inclusive = true }
                    }
                }
            )
        }
        
        // ==================== MAIN APP FLOW ====================

        composable(AppNavigation.HOME) {
            HomeScreen(
                onRoomClick = { roomId ->
                    try {
                        navController.navigate(AppNavigation.createChatRoute(roomId))
                    } catch (e: Exception) {
                        e.printStackTrace()
                    }
                }
            )
        }

        // Chat screen
        composable(
            route = AppNavigation.CHAT,
            arguments = listOf(
                navArgument("roomId") { type = NavType.StringType }
            )
        ) {
            val roomId = it.arguments?.getString("roomId") ?: ""
            val matrixClient: MatrixClient = koinInject()
            val controlPlaneStore: ControlPlaneStore = koinInject()
            val messageRepository: MessageRepository = koinInject()
            ChatScreenEnhanced(
                roomId = roomId,
                viewModel = remember { ChatViewModel(roomId, matrixClient, controlPlaneStore, messageRepository) },
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToRoomDetails = { id ->
                    navController.navigate(AppNavigation.createRoomDetailsRoute(id))
                },
                onNavigateToVoiceCall = { _ ->
                    // Navigate to voice call
                    val callId = "voice_${System.currentTimeMillis()}"
                    navController.navigate(AppNavigation.createCallRoute(callId))
                },
                onNavigateToVideoCall = { _ ->
                    // Navigate to video call
                    val callId = "video_${System.currentTimeMillis()}"
                    navController.navigate(AppNavigation.createCallRoute(callId))
                },
                onNavigateToThread = { roomId, rootMessageId ->
                    navController.navigate(AppNavigation.createThreadRoute(roomId, rootMessageId))
                },
                onNavigateToImage = { imageId ->
                    navController.navigate(AppNavigation.createImageViewerRoute(imageId))
                },
                onNavigateToFile = { fileId ->
                    navController.navigate(AppNavigation.createFilePreviewRoute(fileId))
                },
                onNavigateToUserProfile = { userId ->
                    navController.navigate(AppNavigation.createUserProfileRoute(userId))
                }
            )
        }

        // Profile screen
        composable(AppNavigation.PROFILE) {
            ProfileScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToSettings = {
                    navController.navigate(AppNavigation.SETTINGS)
                },
                onNavigateToChangePassword = {
                    navController.navigate(AppNavigation.CHANGE_PASSWORD)
                },
                onNavigateToChangePhone = {
                    navController.navigate(AppNavigation.CHANGE_PHONE)
                },
                onNavigateToEditBio = {
                    navController.navigate(AppNavigation.EDIT_BIO)
                },
                onNavigateToDeleteAccount = {
                    navController.navigate(AppNavigation.DELETE_ACCOUNT)
                },
                onNavigateToLogin = {
                    // Clear back stack and navigate to login
                    navController.navigate(AppNavigation.LOGIN) {
                        popUpTo(AppNavigation.HOME) { inclusive = true }
                    }
                }
            )
        }

        // Settings screen
        composable(AppNavigation.SETTINGS) {
            val preferences: AppPreferences = koinInject()
            val logoutUseCase: com.armorclaw.shared.domain.usecase.LogoutUseCase = koinInject()
            val controlPlaneStore: ControlPlaneStore = koinInject()
            val viewModel: SettingsViewModel = remember {
                SettingsViewModel(logoutUseCase)
            }
            val uiState by viewModel.uiState.collectAsState()
            val keystoreStatus by controlPlaneStore.keystoreStatus.collectAsState()

            // Handle logout success
            LaunchedEffect(uiState) {
                when (uiState) {
                    is SettingsUiState.LogoutSuccess -> {
                        // Clear preferences
                        preferences.clearSession()
                        // Navigate to login, clearing back stack
                        navController.navigate(AppNavigation.LOGIN) {
                            popUpTo(0) { inclusive = true }
                        }
                        viewModel.resetState()
                    }
                    is SettingsUiState.LogoutError -> {
                        // Show error but still navigate to login for safety
                        preferences.clearSession()
                        navController.navigate(AppNavigation.LOGIN) {
                            popUpTo(0) { inclusive = true }
                        }
                        viewModel.resetState()
                    }
                    else -> { /* No action needed */ }
                }
            }

            val isLoggedIn by preferences.isLoggedIn.collectAsState()

            SettingsScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToProfile = {
                    navController.navigate(AppNavigation.PROFILE)
                },
                onNavigateToSecurity = {
                    navController.navigate(AppNavigation.SECURITY_SETTINGS)
                },
                onNavigateToNotifications = {
                    navController.navigate(AppNavigation.NOTIFICATION_SETTINGS)
                },
                onNavigateToAppearance = {
                    navController.navigate(AppNavigation.APPEARANCE)
                },
                onNavigateToPrivacy = {
                    navController.navigate(AppNavigation.PRIVACY_POLICY)
                },
                onNavigateToAbout = {
                    navController.navigate(AppNavigation.ABOUT)
                },
                onNavigateToMyData = {
                    navController.navigate(AppNavigation.MY_DATA)
                },
                onNavigateToReportBug = {
                    navController.navigate(AppNavigation.REPORT_BUG)
                },
                onNavigateToDevices = {
                    navController.navigate(AppNavigation.DEVICES)
                },
                onNavigateToAgents = {
                    navController.navigate(AppNavigation.AGENT_MANAGEMENT)
                },
                onNavigateToApprovals = {
                    navController.navigate(AppNavigation.HITL_APPROVALS)
                },
                onNavigateToDataSafety = {
                    navController.navigate(AppNavigation.DATA_SAFETY)
                },
                onNavigateToServerConnection = {
                    navController.navigate(AppNavigation.SERVER_CONNECTION)
                },
                onNavigateToInvite = {
                    navController.navigate(AppNavigation.INVITE)
                },
                onNavigateToUnseal = {
                    navController.navigate(AppNavigation.KEYSTORE)
                },
                onNavigateToLogin = {
                    navController.navigate(AppNavigation.LOGIN)
                },
                onNavigateToRegister = {
                    navController.navigate(AppNavigation.REGISTRATION)
                },
                isLoggedIn = isLoggedIn,
                loggedInUserName = preferences.getUserDisplayName(),
                loggedInUserEmail = preferences.getCurrentUserId(),
                keystoreStatus = keystoreStatus,
                onResealKeystore = {
                    controlPlaneStore.resealKeystore()
                },
                onLogout = {
                    viewModel.logout()
                }
            )
        }
        
        // ==================== ROOM MANAGEMENT FLOW ====================
        
        // Room management screen
        composable(AppNavigation.ROOM_MANAGEMENT) {
            RoomManagementScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToHome = {
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(AppNavigation.ROOM_MANAGEMENT) { inclusive = true }
                    }
                },
                onCreateRoom = { name, topic, isPrivate ->
                    // Create room and navigate to it
                    // For now, generate a temporary room ID
                    val newRoomId = "!${System.currentTimeMillis()}:example.com"
                    navController.navigate(AppNavigation.createChatRoute(newRoomId)) {
                        popUpTo(AppNavigation.HOME) { inclusive = false }
                    }
                },
                onJoinRoom = { roomId, alias ->
                    // Join room and navigate to it
                    navController.navigate(AppNavigation.createChatRoute(roomId)) {
                        popUpTo(AppNavigation.HOME) { inclusive = false }
                    }
                }
            )
        }
        
        // Room details screen (Bug 2 fix: Bridge verification banner added)
        composable(
            route = AppNavigation.ROOM_DETAILS,
            arguments = listOf(
                navArgument("roomId") { type = NavType.StringType }
            )
        ) {
            val roomId = it.arguments?.getString("roomId") ?: ""
            RoomDetailsScreen(
                roomId = roomId,
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToSettings = {
                    navController.navigate(AppNavigation.createRoomSettingsRoute(roomId))
                },
                onLeaveRoom = {
                    // TODO: Leave room
                    navController.popBackStack()
                },
                onArchiveRoom = {
                    // TODO: Archive room
                    navController.popBackStack()
                },
                onVerifyBridge = { deviceId ->
                    // Launch emoji verification for the Bridge device
                    navController.navigate(AppNavigation.createVerificationRoute(deviceId))
                }
            )
        }

        // Bridge verification (reuses emoji verification with bridge device ID)
        composable(
            AppNavigation.BRIDGE_VERIFICATION,
            arguments = listOf(navArgument("deviceId") { type = NavType.StringType })
        ) { backStackEntry ->
            val deviceId = backStackEntry.arguments?.getString("deviceId") ?: ""

            EmojiVerificationScreen(
                state = com.armorclaw.shared.domain.model.VerificationState.Ready,
                deviceName = "SDTW Bridge ($deviceId)",
                onConfirmMatch = {
                    // TODO: Confirm bridge verification
                    navController.popBackStack()
                },
                onDenyMatch = {
                    // TODO: Deny bridge verification
                    navController.popBackStack()
                },
                onCancel = {
                    navController.popBackStack()
                }
            )
        }
        
        // Room settings screen
        composable(
            route = AppNavigation.ROOM_SETTINGS,
            arguments = listOf(
                navArgument("roomId") { type = NavType.StringType }
            )
        ) {
            val roomId = it.arguments?.getString("roomId") ?: ""
            RoomSettingsScreen(
                roomId = roomId,
                roomName = "Room Name", // TODO: Get actual room name
                onNavigateBack = {
                    navController.popBackStack()
                },
                onSave = { name, topic ->
                    // TODO: Save room settings
                    navController.popBackStack()
                },
                onChangeAvatar = {
                    // TODO: Change room avatar
                },
                onArchiveRoom = {
                    // TODO: Archive room
                    navController.popBackStack()
                },
                onLeaveRoom = {
                    // TODO: Leave room
                    navController.navigate(AppNavigation.HOME) {
                        popUpTo(AppNavigation.ROOM_SETTINGS) { inclusive = true }
                    }
                }
            )
        }
        
        // ==================== SEARCH FLOW ====================
        
        // Search screen
        composable(AppNavigation.SEARCH) {
            SearchScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToRoom = { roomId ->
                    navController.navigate(AppNavigation.createChatRoute(roomId)) {
                        popUpTo(AppNavigation.SEARCH) { inclusive = true }
                    }
                },
                onNavigateToMessage = { roomId, messageId ->
                    navController.navigate(AppNavigation.createChatRoute(roomId)) {
                        popUpTo(AppNavigation.SEARCH) { inclusive = true }
                    }
                    // TODO: Navigate to specific message
                }
            )
        }
        
        // ==================== PROFILE OPTIONS FLOW ====================
        
        // Change password screen
        composable(AppNavigation.CHANGE_PASSWORD) {
            ChangePasswordScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onChangePassword = { currentPassword, newPassword ->
                    // TODO: Change password
                    navController.popBackStack()
                }
            )
        }
        
        // Change phone number screen
        composable(AppNavigation.CHANGE_PHONE) {
            ChangePhoneNumberScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onChangePhoneNumber = { phoneNumber ->
                    // TODO: Change phone number
                    navController.popBackStack()
                }
            )
        }
        
        // Edit bio screen
        composable(AppNavigation.EDIT_BIO) {
            EditBioScreen(
                currentBio = "Your bio here...", // TODO: Get actual bio
                onNavigateBack = {
                    navController.popBackStack()
                },
                onSaveBio = { bio ->
                    // TODO: Save bio
                    navController.popBackStack()
                }
            )
        }
        
        // Delete account screen
        composable(AppNavigation.DELETE_ACCOUNT) {
            DeleteAccountScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onDeleteAccount = {
                    // TODO: Delete account
                    navController.navigate(AppNavigation.LOGIN) {
                        popUpTo(AppNavigation.HOME) { inclusive = true }
                    }
                }
            )
        }
        
        // ==================== SETTINGS OPTIONS FLOW ====================
        
        // Security settings screen
        composable(AppNavigation.SECURITY_SETTINGS) {
            SecuritySettingsScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToDevices = {
                    navController.navigate(AppNavigation.DEVICES)
                }
            )
        }
        
        // Notification settings screen
        composable(AppNavigation.NOTIFICATION_SETTINGS) {
            NotificationSettingsScreen(
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }
        
        // Appearance settings screen
        composable(AppNavigation.APPEARANCE) {
            AppearanceSettingsScreen(
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }
        
        // Privacy policy screen
        composable(AppNavigation.PRIVACY_POLICY) {
            PrivacyPolicyScreen(
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }

        // My data screen
        composable(AppNavigation.MY_DATA) {
            MyDataScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onRequestData = {
                    // TODO: Request data download
                }
            )
        }

        // Data safety screen
        composable(AppNavigation.DATA_SAFETY) {
            DataSafetyScreen(
                onNavigateBack = {
                    navController.popBackStack()
                }
            )
        }

        // About screen
        composable(AppNavigation.ABOUT) {
            AboutScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onNavigateToWebsite = {
                    linkHandler.openInCustomTab(com.armorclaw.app.util.ExternalLinkHandler.WEBSITE_URL)
                },
                onNavigateToGitHub = {
                    linkHandler.openInCustomTab(com.armorclaw.app.util.ExternalLinkHandler.GITHUB_URL)
                },
                onNavigateToTwitter = {
                    linkHandler.openInCustomTab(com.armorclaw.app.util.ExternalLinkHandler.TWITTER_URL)
                },
                onNavigateToTerms = {
                    navController.navigate(AppNavigation.TERMS_OF_SERVICE)
                },
                onNavigateToPrivacy = {
                    navController.navigate(AppNavigation.PRIVACY_POLICY)
                },
                onNavigateToLicenses = {
                    navController.navigate(AppNavigation.LICENSES)
                }
            )
        }
        
        // Licenses screen
        composable(AppNavigation.LICENSES) {
            com.armorclaw.app.screens.settings.OpenSourceLicensesScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        // Terms of service screen
        composable(AppNavigation.TERMS_OF_SERVICE) {
            com.armorclaw.app.screens.settings.TermsOfServiceScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }
        
        // Report bug screen
        composable(AppNavigation.REPORT_BUG) {
            ReportBugScreen(
                onNavigateBack = {
                    navController.popBackStack()
                },
                onSubmitReport = { summary, description, category ->
                    // TODO: Submit bug report
                    navController.popBackStack()
                }
            )
        }

        // Invite screen (for generating QR codes and sharing)
        composable(AppNavigation.INVITE) {
            com.armorclaw.app.screens.settings.InviteScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }

        // Server connection screen (for post-onboarding discovery activation)
        composable(AppNavigation.SERVER_CONNECTION) {
            com.armorclaw.app.screens.settings.ServerConnectionScreen(
                onBack = { navController.popBackStack() }
            )
        }

// ==================== AGENT & WORKFLOW SCREENS (NEW) ====================
    
    // Agent management screen
    composable(AppNavigation.AGENT_MANAGEMENT) {
        AgentManagementScreen(
            onBack = { navController.popBackStack() }
        )
    }

    // HITL approval screen
    composable(AppNavigation.HITL_APPROVALS) {
        HitlApprovalScreen(
            onBack = { navController.popBackStack() }
        )
    }

    // Vault screen
    composable(AppNavigation.VAULT) {
        VaultScreen()
    }

    // Agent Studio screen
    composable(AppNavigation.AGENT_STUDIO) {
        AgentStudioScreen()
    }

        // ==================== KEYSTORE FLOW ====================

        // Keystore unseal screen
        composable(AppNavigation.KEYSTORE) {
            UnsealScreen(
                onUnsealed = { navController.popBackStack() }
            )
        }

        // ==================== SYNC & DEVICE FLOW ====================

        // Device list screen
        composable(AppNavigation.DEVICES) {
            DeviceListScreen(
                viewModel = remember { DeviceListViewModel() },
                onNavigateBack = {
                    navController.popBackStack()
                },
                onAddDeviceClick = {
                    // Navigate to add device flow
                    navController.navigate(AppNavigation.ADD_DEVICE)
                },
                onVerifyDeviceClick = { deviceId ->
                    navController.navigate(AppNavigation.createVerificationRoute(deviceId))
                }
            )
        }

        // Add device screen
        composable(AppNavigation.ADD_DEVICE) {
            AddDeviceScreen(
                onNavigateBack = { navController.popBackStack() },
                onDeviceAdded = {
                    // Navigate back to device list
                    navController.popBackStack()
                }
            )
        }

        // ==================== VERIFICATION FLOW ====================

        // Emoji verification screen
        composable(
            AppNavigation.EMOJI_VERIFICATION,
            arguments = listOf(navArgument("deviceId") { type = NavType.StringType })
        ) { backStackEntry ->
            val deviceId = backStackEntry.arguments?.getString("deviceId") ?: ""

            EmojiVerificationScreen(
                state = com.armorclaw.shared.domain.model.VerificationState.Ready,
                deviceName = deviceId,
                onConfirmMatch = {
                    // TODO: Confirm verification match
                    navController.popBackStack()
                },
                onDenyMatch = {
                    // TODO: Deny verification match
                    navController.popBackStack()
                },
                onCancel = {
                    navController.popBackStack()
                }
            )
        }

        // ==================== CALL FLOW ====================

        // Incoming call dialog
        composable(
            AppNavigation.INCOMING_CALL,
            arguments = listOf(
                navArgument("callId") { type = NavType.StringType },
                navArgument("callerId") { type = NavType.StringType },
                navArgument("callerName") { type = NavType.StringType },
                navArgument("callType") { type = NavType.StringType; defaultValue = "voice" }
            )
        ) { backStackEntry ->
            val callId = backStackEntry.arguments?.getString("callId") ?: ""
            val callerId = backStackEntry.arguments?.getString("callerId") ?: ""
            val callerName = backStackEntry.arguments?.getString("callerName") ?: "Unknown Caller"
            val callTypeStr = backStackEntry.arguments?.getString("callType") ?: "voice"
            val callType = if (callTypeStr == "video") 
                com.armorclaw.shared.domain.model.CallType.VIDEO 
            else 
                com.armorclaw.shared.domain.model.CallType.VOICE

            IncomingCallDialog(
                callerName = callerName,
                callerAvatarUrl = null, // TODO: Get from user repository
                roomName = null, // TODO: Get from room repository
                callType = callType,
                onAnswer = {
                    // Accept call and navigate to active call
                    navController.navigate(AppNavigation.createCallRoute(callId)) {
                        popUpTo(AppNavigation.INCOMING_CALL) { inclusive = true }
                    }
                },
                onReject = {
                    // Reject call and go back
                    navController.popBackStack()
                }
            )
        }

        // Active call screen
        composable(
            AppNavigation.ACTIVE_CALL,
            arguments = listOf(navArgument("callId") { type = NavType.StringType })
        ) { backStackEntry ->
            val callId = backStackEntry.arguments?.getString("callId") ?: ""

            // Create a mock call session for UI preview
            val callSession = com.armorclaw.shared.domain.model.CallSession(
                id = callId,
                roomId = "",
                callerId = "unknown",
                callerName = "Unknown Caller",
                state = com.armorclaw.shared.domain.model.CallState.Active,
                participants = listOf(),
                isMuted = false,
                isSpeakerOn = true,
                isLocalVideoEnabled = false
            )

            ActiveCallScreen(
                callSession = callSession,
                onEndCall = {
                    navController.popBackStack()
                },
                onMuteToggle = {
                    // TODO: Toggle mute
                },
                onSpeakerToggle = {
                    // TODO: Toggle speaker
                },
                onVideoToggle = {
                    // TODO: Toggle video
                },
                onHoldToggle = {
                    // TODO: Toggle hold
                },
                onAudioDeviceSelect = { _ ->
                    // TODO: Select audio device
                }
            )
        }

        // ==================== THREAD FLOW ====================

        // Thread view screen
        composable(
            AppNavigation.THREAD,
            arguments = listOf(
                navArgument("roomId") { type = NavType.StringType },
                navArgument("rootMessageId") { type = NavType.StringType }
            )
        ) { backStackEntry ->
            val roomId = backStackEntry.arguments?.getString("roomId") ?: ""
            val rootMessageId = backStackEntry.arguments?.getString("rootMessageId") ?: ""

            com.armorclaw.app.screens.chat.ThreadViewScreen(
                roomId = roomId,
                rootMessageId = rootMessageId,
                rootMessage = null, // TODO: Load from repository
                replies = emptyList(), // TODO: Load from repository
                onNavigateBack = { navController.popBackStack() },
                onSendReply = { text ->
                    // TODO: Send reply via ViewModel
                },
                onMessageClick = { message ->
                    // TODO: Handle message click
                },
                onUserProfileClick = { userId ->
                    navController.navigate(AppNavigation.createUserProfileRoute(userId))
                }
            )
        }

        // ==================== MEDIA VIEWER FLOW ====================

        // Image viewer screen
        composable(
            AppNavigation.IMAGE_VIEWER,
            arguments = listOf(navArgument("imageId") { type = NavType.StringType })
        ) { backStackEntry ->
            val imageId = backStackEntry.arguments?.getString("imageId") ?: ""

            com.armorclaw.app.screens.media.ImageViewerScreen(
                imageUrl = "", // TODO: Get from repository
                fileName = null,
                mimeType = null,
                fileSize = null,
                senderName = null,
                timestamp = null,
                onNavigateBack = { navController.popBackStack() },
                onDownload = {
                    // TODO: Download image
                },
                onShare = {
                    // TODO: Share image
                }
            )
        }

        // File preview screen
        composable(
            AppNavigation.FILE_PREVIEW,
            arguments = listOf(navArgument("fileId") { type = NavType.StringType })
        ) { backStackEntry ->
            val fileId = backStackEntry.arguments?.getString("fileId") ?: ""

            com.armorclaw.app.screens.media.FilePreviewScreen(
                fileId = fileId,
                fileName = "Unknown File", // TODO: Get from repository
                mimeType = "application/octet-stream",
                fileSize = 0,
                fileUrl = null,
                senderName = null,
                timestamp = null,
                isDownloaded = false,
                downloadProgress = null,
                onNavigateBack = { navController.popBackStack() },
                onDownload = {
                    // TODO: Download file
                },
                onOpen = {
                    // TODO: Open file
                },
                onShare = {
                    // TODO: Share file
                },
                onDelete = {
                    // TODO: Delete file
                }
            )
        }

        // ==================== USER PROFILE FLOW ====================

        // User profile screen
        composable(
            AppNavigation.USER_PROFILE,
            arguments = listOf(navArgument("userId") { type = NavType.StringType })
        ) { backStackEntry ->
            val userId = backStackEntry.arguments?.getString("userId") ?: ""

            com.armorclaw.app.screens.profile.UserProfileScreen(
                userId = userId,
                displayName = null, // TODO: Get from repository
                avatarUrl = null,
                bio = null,
                presence = null,
                isVerified = null,
                sharedRoomsCount = 0,
                onNavigateBack = { navController.popBackStack() },
                onSendMessage = {
                    // Navigate to DM with this user (create or find existing room)
                    val dmRoomId = "!dm_$userId:example.com"
                    navController.navigate(AppNavigation.createChatRoute(dmRoomId)) {
                        popUpTo(AppNavigation.USER_PROFILE) { inclusive = true }
                    }
                },
                onStartCall = {
                    // Start call with this user
                    val callId = "call_${System.currentTimeMillis()}_$userId"
                    navController.navigate(AppNavigation.createCallRoute(callId))
                },
                onViewSharedRooms = {
                    // Navigate to shared rooms screen
                    navController.navigate(AppNavigation.createSharedRoomsRoute(userId))
                },
                onBlockUser = {
                    // TODO: Block user
                },
                onReportUser = {
                    // TODO: Report user
                }
            )
        }

        // Shared rooms screen
        composable(
            AppNavigation.SHARED_ROOMS,
            arguments = listOf(navArgument("userId") { type = NavType.StringType })
        ) { backStackEntry ->
            val userId = backStackEntry.arguments?.getString("userId") ?: ""

            SharedRoomsScreen(
                userId = userId,
                onNavigateBack = { navController.popBackStack() },
                onRoomClick = { roomId ->
                    navController.navigate(AppNavigation.createChatRoute(roomId)) {
                        popUpTo(AppNavigation.SHARED_ROOMS) { inclusive = true }
                    }
                }
            )
        }
    }
}
