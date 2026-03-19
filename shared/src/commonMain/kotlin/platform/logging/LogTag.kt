package com.armorclaw.shared.platform.logging

/**
 * Log tags for categorizing log messages by module and component
 *
 * This provides a hierarchical tagging system that makes it easy to:
 * - Filter logs by category
 * - Identify the source of errors
 * - Track issues across different parts of the app
 *
 * Structure: Category.Module.Component
 * Example: UI.Chat.MessageList, Network.Matrix.Sync
 */
open class LogTag(
    val category: String,
    val module: String,
    val component: String = ""
) {
    /**
     * Full tag string: Category.Module.Component
     */
    val fullTag: String
        get() = if (component.isNotEmpty()) {
            "$category.$module.$component"
        } else {
            "$category.$module"
        }

    /**
     * Short tag for display: [Category/Module]
     */
    val shortTag: String
        get() = "[$category/$module]"

    // ==================== UI CATEGORY ====================
    object UI {
        // Navigation
        object Navigation : LogTag("UI", "Navigation")

        // Screens
        object Splash : LogTag("UI", "Splash")
        object Welcome : LogTag("UI", "Welcome")
        object Onboarding : LogTag("UI", "Onboarding")
        object Login : LogTag("UI", "Auth", "Login")
        object Registration : LogTag("UI", "Auth", "Registration")
        object ForgotPassword : LogTag("UI", "Auth", "ForgotPassword")
        object Home : LogTag("UI", "Home")
        object Chat : LogTag("UI", "Chat")
        object ChatSearch : LogTag("UI", "Chat", "Search")
        object ChatInput : LogTag("UI", "Chat", "Input")
        object MessageList : LogTag("UI", "Chat", "MessageList")
        object MessageBubble : LogTag("UI", "Chat", "MessageBubble")
        object ThreadView : LogTag("UI", "Chat", "Thread")
        object Profile : LogTag("UI", "Profile")
        object EditProfile : LogTag("UI", "Profile", "Edit")
        object ChangePassword : LogTag("UI", "Profile", "ChangePassword")
        object DeleteAccount : LogTag("UI", "Profile", "DeleteAccount")
        object Settings : LogTag("UI", "Settings")
        object SecuritySettings : LogTag("UI", "Settings", "Security")
        object NotificationSettings : LogTag("UI", "Settings", "Notifications")
        object AppearanceSettings : LogTag("UI", "Settings", "Appearance")
        object PrivacyPolicy : LogTag("UI", "Settings", "Privacy")
        object DataSafety : LogTag("UI", "Settings", "DataSafety")
        object MyData : LogTag("UI", "Settings", "MyData")
        object DeviceList : LogTag("UI", "Settings", "Devices")
        object About : LogTag("UI", "Settings", "About")
        object ReportBug : LogTag("UI", "Settings", "ReportBug")
        object Search : LogTag("UI", "Search")
        object RoomManagement : LogTag("UI", "Rooms", "Management")
        object RoomDetails : LogTag("UI", "Rooms", "Details")
        object RoomSettings : LogTag("UI", "Rooms", "Settings")
        
        // Call screens
        object IncomingCall : LogTag("UI", "Call", "Incoming")
        object ActiveCall : LogTag("UI", "Call", "Active")
        object CallControls : LogTag("UI", "Call", "Controls")
        object AudioVisualizer : LogTag("UI", "Call", "Visualizer")
    }

    // ==================== VIEWMODEL CATEGORY ====================
    object ViewModel {
        object Chat : LogTag("VM", "Chat")
        object Home : LogTag("VM", "Home")
        object SyncStatus : LogTag("VM", "SyncStatus")
        object Welcome : LogTag("VM", "Welcome")
        object Login : LogTag("VM", "Auth", "Login")
        object Registration : LogTag("VM", "Auth", "Registration")
        object Profile : LogTag("VM", "Profile")
        object Settings : LogTag("VM", "Settings")
        object Splash : LogTag("VM", "Splash")
        object Room : LogTag("VM", "Room")
        object DeviceList : LogTag("VM", "DeviceList")
        object Secretary : LogTag("VM", "Secretary")
    }

    // ==================== DOMAIN CATEGORY ====================
    object Domain {
        object Auth : LogTag("Domain", "Auth")
        object Message : LogTag("Domain", "Message")
        object Room : LogTag("Domain", "Room")
        object User : LogTag("Domain", "User")
        object Sync : LogTag("Domain", "Sync")
        object Call : LogTag("Domain", "Call")
        object Notification : LogTag("Domain", "Notification")
        object Thread : LogTag("Domain", "Thread")
        object Verification : LogTag("Domain", "Verification")
        object ControlPlane : LogTag("Domain", "ControlPlane")
        object Workflow : LogTag("Domain", "Workflow")
        object Agent : LogTag("Domain", "Agent")
        object License : LogTag("Domain", "License")
        object Budget : LogTag("Domain", "Budget")
    }

    // ==================== USECASE CATEGORY ====================
    object UseCase {
        object Login : LogTag("UseCase", "Auth", "Login")
        object Logout : LogTag("UseCase", "Auth", "Logout")
        object Register : LogTag("UseCase", "Auth", "Register")
        object SendMessage : LogTag("UseCase", "Message", "Send")
        object LoadMessages : LogTag("UseCase", "Message", "Load")
        object GetRooms : LogTag("UseCase", "Room", "Get")
        object CreateRoom : LogTag("UseCase", "Room", "Create")
        object JoinRoom : LogTag("UseCase", "Room", "Join")
        object LeaveRoom : LogTag("UseCase", "Room", "Leave")
        object SyncWhenOnline : LogTag("UseCase", "Sync", "WhenOnline")
    }

    // ==================== DATA CATEGORY ====================
    object Data {
        object Database : LogTag("Data", "Database")
        object DatabaseDao : LogTag("Data", "Database", "Dao")
        object Preferences : LogTag("Data", "Preferences")
        object Cache : LogTag("Data", "Cache")
        object SecureStorage : LogTag("Data", "SecureStorage")
        
        // Repositories
        object AuthRepository : LogTag("Data", "Repository", "Auth")
        object MessageRepository : LogTag("Data", "Repository", "Message")
        object RoomRepository : LogTag("Data", "Repository", "Room")
        object UserRepository : LogTag("Data", "Repository", "User")
        object SyncRepository : LogTag("Data", "Repository", "Sync")
        object CallRepository : LogTag("Data", "Repository", "Call")
        object NotificationRepository : LogTag("Data", "Repository", "Notification")
        object ThreadRepository : LogTag("Data", "Repository", "Thread")
        object VerificationRepository : LogTag("Data", "Repository", "Verification")
        object AgentFlowRepository : LogTag("Data", "Repository", "AgentFlow")
    }

    // ==================== NETWORK CATEGORY ====================
    object Network {
        object Matrix : LogTag("Network", "Matrix")
        object MatrixSync : LogTag("Network", "Matrix", "Sync")
        object MatrixApi : LogTag("Network", "Matrix", "API")
        object MatrixClient : LogTag("Network", "Matrix", "Client")
        object MatrixE2EE : LogTag("Network", "Matrix", "E2EE")
        object HttpClient : LogTag("Network", "HTTP", "Client")
        object WebSocket : LogTag("Network", "WebSocket")
        object CertificatePinning : LogTag("Network", "Security", "CertPin")
        object NetworkMonitor : LogTag("Network", "Monitor")
        object Bridge : LogTag("Network", "Bridge")
        object BridgeRpc : LogTag("Network", "Bridge", "RPC")
        object BridgeWebSocket : LogTag("Network", "Bridge", "WebSocket")
        object Fcm : LogTag("Network", "FCM")
    }

    // ==================== PLATFORM CATEGORY ====================
    object Platform {
        object Android : LogTag("Platform", "Android")
        object Biometric : LogTag("Platform", "Security", "Biometric")
        object Notification : LogTag("Platform", "Notification")
        object Clipboard : LogTag("Platform", "Clipboard")
        object VoiceCall : LogTag("Platform", "VoiceCall")
        object Permission : LogTag("Platform", "Permission")
        object Camera : LogTag("Platform", "Camera")
        object Audio : LogTag("Platform", "Audio")
        object Navigation : LogTag("Platform", "Navigation")
        object Network : LogTag("Platform", "Network")
    }

    // ==================== SECURITY CATEGORY ====================
    object Security {
        object Encryption : LogTag("Security", "Encryption")
        object KeyManagement : LogTag("Security", "KeyManagement")
        object SecureStorage : LogTag("Security", "Storage")
        object CertificateVerification : LogTag("Security", "CertVerify")
        object Session : LogTag("Security", "Session")
    }

    // ==================== SYNC CATEGORY ====================
    object Sync {
        object EventProcessor : LogTag("Sync", "EventProcessor")
        object StateSync : LogTag("Sync", "State")
        object MessageSync : LogTag("Sync", "Message")
        object RoomSync : LogTag("Sync", "Room")
        object PresenceSync : LogTag("Sync", "Presence")
    }

    // ==================== DI CATEGORY ====================
    object DI {
        object Module : LogTag("DI", "Module")
        object Initialization : LogTag("DI", "Init")
    }

    // ==================== LIFECYCLE CATEGORY ====================
    object Lifecycle {
        object App : LogTag("Lifecycle", "App")
        object Activity : LogTag("Lifecycle", "Activity")
        object Service : LogTag("Lifecycle", "Service")
        object ViewModel : LogTag("Lifecycle", "ViewModel")
    }

    // ==================== CRASH REPORTING CATEGORY ====================
    object CrashReporting {
        object Initialization : LogTag("CrashReport", "Init")
        object Capture : LogTag("CrashReport", "Capture")
        object Breadcrumb : LogTag("CrashReport", "Breadcrumb")
    }

    // ==================== ANALYTICS CATEGORY ====================
    object Analytics {
        object Initialization : LogTag("Analytics", "Init")
        object Event : LogTag("Analytics", "Event")
        object Screen : LogTag("Analytics", "Screen")
    }

    // ==================== PERFORMANCE CATEGORY ====================
    object Performance {
        object Startup : LogTag("Performance", "Startup")
        object Memory : LogTag("Performance", "Memory")
        object Battery : LogTag("Performance", "Battery")
        object NetworkUsage : LogTag("Performance", "Network")
    }

    // ==================== WORKER CATEGORY ====================
    object Worker {
        object SyncWorker : LogTag("Worker", "Sync")
        object NotificationWorker : LogTag("Worker", "Notification")
        object CleanupWorker : LogTag("Worker", "Cleanup")
        object KeyRotationWorker : LogTag("Worker", "KeyRotation")
    }

    // ==================== TESTING CATEGORY ====================
    object Test {
        object Unit : LogTag("Test", "Unit")
        object Integration : LogTag("Test", "Integration")
        object Ui : LogTag("Test", "UI")
    }
}

/**
 * Custom log tag for specific use cases
 */
class CustomLogTag(
    category: String,
    module: String,
    component: String = ""
) : LogTag(category, module, component)

/**
 * Extension to create a component-specific tag from a base tag
 */
fun LogTag.withComponent(component: String): LogTag = CustomLogTag(
    category = this.category,
    module = this.module,
    component = component
)
