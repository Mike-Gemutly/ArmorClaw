# ArmorClaw Mobile App - Gap Analysis & Fixes

> **Document Purpose:** Identify and fix gaps in the mobile app plan to ensure production-ready implementation.
> **Date Created:** 2026-02-10
> **Status:** Critical Gaps Identified | Action Items Defined

---

## Executive Summary

The original mobile app plan covers 15 sections with comprehensive feature mapping. This analysis identifies **32 critical gaps** across technical architecture, UX design, security, compliance, and infrastructure categories.

**Gap Distribution:**
- **Technical Architecture:** 9 gaps
- **UX Design:** 8 gaps
- **Security:** 7 gaps
- **Compliance:** 3 gaps
- **Infrastructure:** 5 gaps

---

## 1. Technical Architecture Gaps

### 1.1 Offline/Sync Strategy ⚠️ CRITICAL

**Gap:** Only brief mention of "graceful degradation" for offline fallback. No detailed sync strategy.

**Impact:** Users lose access to conversation history when offline; data conflicts when reconnecting.

**Solution:**

```kotlin
// Offline sync architecture
interface OfflineSyncManager {
    // Queue messages while offline
    suspend fun queueMessage(message: MatrixMessage)

    // Sync when connection restored
    suspend fun syncWhenOnline(): SyncResult

    // Resolve conflicts (last-write-wins with timestamps)
    suspend fun resolveConflict(local: Message, remote: Message): Message
}

// Local storage strategy
data class OfflineStorageConfig(
    val maxOfflineMessages: Int = 1000,
    val maxOfflineDays: Int = 7,
    val syncBatchSize: Int = 50
)
```

**Implementation Requirements:**
- SQLite/Room for local message cache
- Background sync worker (WorkManager)
- Conflict resolution algorithm
- Sync status indicators in UI

**Phase:** 1 (Foundation)

---

### 1.2 Matrix Disconnection Recovery ⚠️ CRITICAL

**Gap:** No detailed error recovery for Matrix connection drops.

**Impact:** App appears broken; users don't know if messages are sending.

**Solution:**

```kotlin
// Connection state machine
sealed class ConnectionState {
    object Connected : ConnectionState()
    object Connecting : ConnectionState()
    data class Disconnected(val reason: ErrorReason) : ConnectionState()
    data class Reconnecting(val attempt: Int, val maxAttempts: Int) : ConnectionState()
}

class MatrixConnectionManager(
    private val config: ReconnectionConfig(
        initialDelay = 1_000L,      // 1 second
        maxDelay = 30_000L,          // 30 seconds
        maxAttempts = 10
    )
) {
    // Exponential backoff reconnection
    suspend fun reconnectWithBackoff(): Result<Connection>
}
```

**UI Requirements:**
- Connection status banner
- Retry button with exponential backoff
- Queued message indicator
- Last successful sync timestamp

**Phase:** 1 (Foundation)

---

### 1.3 Push Notification Architecture ⚠️ CRITICAL

**Gap:** No push notification system for new messages/alerts.

**Impact:** Users miss important messages unless app is open.

**Solution:**

```kotlin
// Push notification via Matrix Push Gateway
interface PushNotificationManager {
    // Register device with Matrix Push Gateway
    suspend fun registerDevice(pushToken: String, lang: String = "en")

    // Handle incoming push
    fun handlePushNotification(data: PushData): Notification

    // Unregister
    suspend fun unregisterDevice()
}

// Notification types
sealed class PushType {
    data class NewMessage(val roomId: String, val sender: String) : PushType()
    data class BudgetAlert(val percentUsed: Float) : PushType()
    data class SecurityAlert(val event: SecurityEvent) : PushType()
    data class TaskUpdate(val taskId: String, val status: TaskStatus) : PushType()
}
```

**Implementation:**
- Firebase Cloud Messaging (Android) + APNs (iOS)
- Matrix Push Gateway integration
- Notification categories (message, budget, security, task)
- Doze mode exemptions for critical alerts

**Phase:** 1 (Foundation)

---

### 1.4 App Updates & OTA Configuration

**Gap:** No mechanism for over-the-air configuration updates without app store release.

**Impact:** Config changes require app updates; slower iteration.

**Solution:**

```kotlin
// Remote config via Matrix room metadata
class RemoteConfigManager(
    private val matrixClient: MatrixClient,
    private val configRoomId: String
) {
    // Fetch config from Matrix room state events
    suspend fun fetchRemoteConfig(): RemoteConfig

    // Validate config signature
    fun validateConfig(config: RemoteConfig, signature: String): Boolean

    // Apply config with rollback support
    suspend fun applyConfig(config: RemoteConfig): Result<Unit>
}

data class RemoteConfig(
    val version: Int,
    val features: Map<String, Boolean>,
    val budgetDefaults: BudgetConfig,
    val themeOverrides: ThemeConfig,
    val minAppVersion: String
)
```

**Phase:** 2 (Intelligence)

---

### 1.5 Biometric Authentication Integration

**Gap:** Mentioned but no integration details for secure Matrix token storage.

**Impact:** Tokens stored in less secure storage; security compliance risk.

**Solution:**

```kotlin
// Biometric authentication for Matrix tokens
interface BiometricTokenManager {
    // Store token encrypted with biometric
    suspend fun storeToken(
        token: String,
        prompt: String = "Authenticate to access chat"
    ): Result<Unit>

    // Retrieve token (requires biometric)
    suspend fun retrieveToken(prompt: String = "Unlock chat"): Result<String>

    // Clear token
    suspend fun clearToken(): Result<Unit>

    // Check if biometric available
    fun isBiometricAvailable(): Boolean
}

// Platform-specific implementation
// Android: BiometricPrompt + AndroidKeyStore
// iOS: LocalAuthentication + Keychain
```

**Phase:** 1 (Foundation)

---

### 1.6 App Lifecycle Handling

**Gap:** No discussion of background/foreground transitions and their effects.

**Impact:** Sync issues, missed messages, battery drain.

**Solution:**

```kotlin
// Lifecycle-aware sync manager
class AppLifecycleManager(
    private val syncManager: OfflineSyncManager,
    private val matrixClient: MatrixClient
) : LifecycleObserver {

    @OnLifecycleEvent(Lifecycle.Event.ON_RESUME)
    fun onForeground() {
        // Resume sync, check for new messages
        syncManager.syncWhenOnline()
    }

    @OnLifecycleEvent(Lifecycle.Event.ON_PAUSE)
    fun onBackground() {
        // Pause sync, release resources
        matrixClient.pauseSync()
    }

    @OnLifecycleEvent(Lifecycle.Event.ON_STOP)
    fun onStop() {
        // Full cleanup if needed
    }
}
```

**Phase:** 1 (Foundation)

---

### 1.7 Memory Leak Prevention

**Gap:** No strategies mentioned for preventing memory leaks in long-running sessions.

**Impact:** App crashes, performance degradation.

**Solution:**

```kotlin
// Memory management strategies
class MemoryManager {
    // Monitor memory usage
    fun getMemoryUsage(): MemoryStats

    // Clear caches when threshold exceeded
    fun clearCachesIfNeeded()

    // Release image cache
    fun releaseImageCache()

    // Clear message history beyond limit
    fun trimMessageHistory(maxMessages: Int = 1000)
}

// Lifecycle-aware components
class MessageListViewModel : ViewModel() {
    override fun onCleared() {
        // Clean up resources
        messageAdapter.release()
        imageLoader.clearCache()
    }
}
```

**Phase:** 1 (Foundation)

---

### 1.8 Crash Reporting Integration

**Gap:** Missing crash reporting for production monitoring.

**Impact:** Silent failures, poor user experience, no debugging data.

**Solution:**

```kotlin
// Crash reporting with ArmorClaw-specific context
interface CrashReporter {
    // Log crash with context
    fun logCrash(throwable: Throwable, context: CrashContext)

    // Log non-fatal errors
    fun logError(error: Error, context: ErrorContext)

    // Set user identifier (hashed Matrix ID)
    fun setUserIdentifier(userId: String)
}

data class CrashContext(
    val matrixRoomId: String?,
    val lastAction: String,
    val connectionState: ConnectionState,
    val memoryUsage: MemoryStats
)

// Integration: Sentry, Firebase Crashlytics
```

**Phase:** 1 (Foundation)

---

### 1.9 App Size Optimization

**Gap:** No discussion of minimizing app size (critical for user adoption).

**Impact:** Low install rates, especially in emerging markets.

**Solution:**

```kotlin
// App size optimization strategies
// 1. Dynamic feature modules
// 2. ProGuard/R8 obfuscation
// 3. Resource shrinking
// 4. APK splits by ABI

android {
    buildTypes {
        release {
            minifyEnabled true
            shrinkResources true
            proguardFiles getDefaultProguardFile('proguard-android-optimize.txt')
        }
    }

    splits {
        abi {
            enable true
            reset()
            include 'x86', 'armeabi-v7a', 'arm64-v8a'
            universalApk false
        }
    }
}

// Target: < 50MB base APK
```

**Phase:** 4 (Polish & Launch)

---

## 2. UX Design Gaps

### 2.1 Onboarding Flow ⚠️ CRITICAL

**Gap:** No onboarding experience for first-time users.

**Impact:** High drop-off rate, confusion about app purpose.

**Solution:**

```kotlin
// Onboarding flow
sealed class OnboardingStep {
    object Welcome : OnboardingStep()      // What is ArmorClaw?
    object Security : OnboardingStep()      // How your data is protected
    object Connect : OnboardingStep()       // Connect to your server
    object Permissions : OnboardingStep()   // Notification permissions
    object Complete : OnboardingStep()      // Ready to chat
}

class OnboardingManager {
    suspend fun completeOnboarding(steps: List<OnboardingStep>)
    fun hasCompletedOnboarding(): Boolean
    fun skipOnboarding() // For power users
}
```

**UI Requirements:**
- 3-5 screen carousel with skip option
- Server connection test
- Permission requests with explanations
- Completion celebration

**Phase:** 1 (Foundation)

---

### 2.2 Empty States

**Gap:** No discussion of empty states (first launch, no conversations).

**Impact:** Confusing blank screens, no guidance.

**Solution:**

```kotlin
// Empty state definitions
sealed class EmptyState(
    val title: String,
    val message: String,
    val action: EmptyAction?
) {
    object NoConversations : EmptyState(
        title = "No conversations yet",
        message = "Start chatting with your AI agent by creating a room or joining an existing one.",
        action = EmptyAction.CreateRoom
    )

    object NoMessages : EmptyState(
        title = "No messages",
        message = "Send a message to start the conversation. Try 'Hello!' or ask a question.",
        action = EmptyAction.SendMessage
    )

    object Offline : EmptyState(
        title = "You're offline",
        message = "Check your internet connection. Messages will be sent when you're back online.",
        action = null
    )
}
```

**Phase:** 1 (Foundation)

---

### 2.3 Loading States & Skeletons

**Gap:** No loading states or skeleton screens for better perceived performance.

**Impact:** App feels sluggish, poor UX.

**Solution:**

```kotlin
// Skeleton screens
@Composable
fun MessageListSkeleton() {
    LazyColumn {
        items(10) {
            SkeletonMessage(
                showAvatar = true,
                lines = (2..4).random()
            )
        }
    }
}

// Loading states
sealed class LoadState {
    object Idle : LoadState()
    object Loading : LoadState()
    data class Success(val hasData: Boolean) : LoadState()
    data class Error(val error: Throwable) : LoadState()
}
```

**Phase:** 1 (Foundation)

---

### 2.4 Input Validation UX

**Gap:** No pre-send validation of user input before sending to backend.

**Impact:** Wasted API calls, poor error messages, user frustration.

**Solution:**

```kotlin
// Input validation before sending
class InputValidator {
    fun validateMessage(input: String): ValidationResult {
        return when {
            input.isBlank() -> ValidationResult.Error("Message cannot be empty")
            input.length > MAX_MESSAGE_LENGTH ->
                ValidationResult.Error("Message too long (max ${MAX_MESSAGE_LENGTH} chars)")
            containsPotentialPII(input) ->
                ValidationResult.Warning("This may contain sensitive information. Continue?")
            else -> ValidationResult.Valid
        }
    }

    private fun containsPotentialPII(input: String): Boolean {
        // Check for emails, SSNs, etc.
        return PII_PATTERTERS.any { it.containsMatchIn(input) }
    }
}

sealed class ValidationResult {
    object Valid : ValidationResult()
    data class Error(val message: String) : ValidationResult()
    data class Warning(val message: String) : ValidationResult()
}
```

**UI Requirements:**
- Real-time character count
- PII warnings with visual indicators
- Send button disabled for invalid input

**Phase:** 1 (Foundation)

---

### 2.5 Typing Indicators

**Gap:** No typing indicators to show when agent is processing.

**Impact:** Uncertainty about whether agent is working, duplicate sends.

**Solution:**

```kotlin
// Typing indicators via Matrix
interface TypingIndicatorManager {
    // Send typing notification
    suspend fun sendTypingNotification(roomId: String, isTyping: Boolean)

    // Subscribe to typing events
    fun observeTypingUsers(roomId: String): Flow<Set<UserId>>

    // Show agent processing indicator
    fun showAgentProcessing(roomId: String, isProcessing: Boolean)
}

// UI: Animated dots "..." when agent is thinking
```

**Phase:** 2 (Intelligence)

---

### 2.6 Read Receipts

**Gap:** No read receipts for message delivery confirmation.

**Impact:** Uncertainty about message delivery, confusing UX.

**Solution:**

```kotlin
// Read receipts via Matrix receipts
interface ReceiptManager {
    // Send read receipt
    suspend fun sendReadReceipt(roomId: String, eventId: String)

    // Get read status
    suspend fun getReadStatus(eventId: String): ReceiptStatus

    // Observe receipt updates
    fun observeReceipts(eventId: String): Flow<ReceiptStatus>
}

data class ReceiptStatus(
    val sent: Boolean,
    val delivered: Boolean,
    val read: Boolean,
    val readBy: List<UserId>
)
```

**UI Requirements:**
- Single check (sent)
- Double check (delivered)
- Filled double check (read)

**Phase:** 2 (Intelligence)

---

### 2.7 Search Functionality

**Gap:** No search across conversations and messages.

**Impact:** Difficult to find past information, poor usability.

**Solution:**

```kotlin
// Full-text search via Matrix
interface SearchManager {
    // Search across all rooms
    suspend fun searchAllRooms(query: String): SearchResults

    // Search specific room
    suspend fun searchRoom(roomId: String, query: String): SearchResults

    // Search filters
    data class SearchFilters(
        val rooms: List<String>? = null,
        val sender: String? = null,
        val dateRange: ClosedRange<Instant>? = null,
        val messageType: MessageType? = null
    )
}

data class SearchResults(
    val results: List<SearchResult>,
    val totalCount: Int,
    val hasNextPage: Boolean
)
```

**Phase:** 2 (Intelligence)

---

### 2.8 Pull-to-Refresh

**Gap:** No pull-to-refresh mechanics for manual sync.

**Impact:** No way to force sync, poor perceived control.

**Solution:**

```kotlin
// Pull-to-refresh implementation
@Composable
fun MessageList(
    viewModel: MessageListViewModel,
    onRefresh: () -> Unit
) {
    val pullRefreshState = rememberPullRefreshState(
        refreshing = viewModel.isRefreshing,
        onRefresh = onRefresh
    )

    Box(modifier = Modifier.pullRefresh(pullRefreshState)) {
        LazyColumn {
            // Messages
        }

        PullRefreshIndicator(
            refreshing = viewModel.isRefreshing,
            state = pullRefreshState,
            modifier = Modifier.align(Alignment.TopCenter)
        )
    }
}
```

**Phase:** 1 (Foundation)

---

## 3. Security Gaps

### 3.1 Certificate Pinning ⚠️ CRITICAL

**Gap:** No certificate pinning for Matrix connections.

**Impact:** Vulnerable to MITM attacks, especially in enterprise environments.

**Solution:**

```kotlin
// Certificate pinning for Matrix homeserver
class CertificatePinningInterceptor(
    private val pins: List<CertificatePin>
) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val certificate = chain.request()
            .url("https://matrix.example.com")
            .let { chain.connection()?.socket() }
            ?.let { getCertificate(it) }

        if (!pins.any { it.matches(certificate) }) {
            throw CertificatePinningException("Certificate not pinned!")
        }

        return chain.proceed(chain.request())
    }
}

data class CertificatePin(
    val hash: String,  // SPKI hash (base64)
    val expiry: Instant?
)
```

**Implementation:**
- OkHttp CertificatePinner for Android
- URLSession delegate for iOS
- Fallback to system certificates in dev builds

**Phase:** 1 (Foundation)

---

### 3.2 Screen Capture Prevention

**Gap:** No prevention of screen capture for sensitive content.

**Impact:** Screenshots of sensitive conversations, security risk.

**Solution:**

```kotlin
// Screen capture prevention (Android)
class SecureActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Prevent screenshots
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
            window.setFlags(
                LayoutParams.FLAG_SECURE,
                LayoutParams.FLAG_SECURE
            )
        }
    }
}

// iOS: Prevent screen recording in specific view controllers
// override func viewWillAppear(_ animated: Bool) {
//     isScreenRecordingDisabled = true
// }
```

**Note:** Make this configurable per room/message type.

**Phase:** 1 (Foundation)

---

### 3.3 App Shielding (Anti-Tampering)

**Gap:** No app shielding against reverse engineering or tampering.

**Impact:** Easier extraction of keys/tokens, security vulnerability.

**Solution:**

```kotlin
// App integrity checks
interface AppIntegrityChecker {
    // Check for rooted/jailbroken device
    fun isDeviceSecure(): Boolean

    // Check for debuggers
    fun isDebuggerAttached(): Boolean

    // Check for emulator
    fun isEmulator(): Boolean

    // Check app signature
    fun verifyAppSignature(): Boolean
}

// Integration with app shielding solutions:
// - Android: ProGuard, R8, potentially DexGuard
// - iOS: Swift obfuscation, potentially iOS App Protection
```

**Phase:** 3 (Collaboration) - Can defer to enterprise builds

---

### 3.4 Secure Clipboard Handling

**Gap:** No discussion of secure clipboard handling for sensitive data.

**Impact:** Sensitive data accessible from other apps.

**Solution:**

```kotlin
// Secure clipboard
interface SecureClipboard {
    // Copy with auto-clear
    fun copySensitive(text: String, autoClearAfter: Duration = 30.seconds)

    // Clear clipboard
    fun clear()

    // Check if contains sensitive data
    fun containsSensitiveData(): Boolean
}

// Auto-clear timer clears clipboard after use
// Warning when pasting sensitive data
```

**Phase:** 1 (Foundation)

---

### 3.5 Screenshot Detection

**Gap:** No detection when user takes screenshots (for security audit).

**Impact:** No audit trail of potential data leaks.

**Solution:**

```kotlin
// Screenshot detection (Android)
class ScreenshotDetector(private val context: Context) {
    private val handler = Handler(Looper.getMainLooper())

    fun startListening() {
        context.contentResolver.registerContentObserver(
            MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
            true,
            screenshotObserver
        )
    }

    private val screenshotObserver = object : ContentObserver(handler) {
        override fun onChange(selfChange: Boolean) {
            // Check if screenshot was taken
            val lastScreenshot = getLastScreenshot()
            if (lastScreenshot != null && lastScreenshot.time > lastCheckedTime) {
                // Log security event
                logSecurityEvent(SecurityEvent.ScreenshotDetected)
            }
        }
    }
}
```

**Phase:** 2 (Intelligence)

---

### 3.6 Biometric Timeout Policies

**Gap:** No discussion of biometric authentication timeout policies.

**Impact:** Balance between security and usability unclear.

**Solution:**

```kotlin
// Biometric timeout configuration
data class BiometricPolicy(
    val sessionTimeout: Duration = 5.minutes,
    val biometricPromptInterval: Duration = 1.minute,
    val requireOnAppLaunch: Boolean = true,
    val requireOnBackground: Boolean = true,
    val requireOnSensitiveActions: Boolean = true
)

// Enforce policy
class BiometricPolicyEnforcer(
    private val policy: BiometricPolicy
) {
    fun shouldAuthenticate(action: AppAction): Boolean {
        return when (action) {
            is AppAction.Launch -> policy.requireOnAppLaunch
            is AppAction.ResumeFromBackground -> policy.requireOnBackground
            is AppAction.SensitiveAction -> policy.requireOnSensitiveActions
            else -> false
        }
    }
}
```

**Phase:** 1 (Foundation)

---

### 3.7 Secure Time Handling

**Gap:** No discussion of secure time verification for TTL enforcement.

**Impact:** Device time manipulation could bypass TTL limits.

**Solution:**

```kotlin
// Secure time from network, not device clock
interface SecureTimeProvider {
    // Get time from trusted source (Matrix server, NTP)
    suspend fun getSecureTime(): Instant

    // Verify device time hasn't been manipulated
    fun verifyDeviceTime(): Boolean
}

// Fallback to Matrix server time if device time untrusted
class SecureTimeManager(
    private val matrixClient: MatrixClient
) : SecureTimeProvider {
    override suspend fun getSecureTime(): Instant {
        return try {
            matrixClient.getServerTime()
        } catch (e: Exception) {
            // Fallback to device time with warning
            Instant.now()
        }
    }
}
```

**Phase:** 2 (Intelligence)

---

## 4. Compliance Gaps

### 4.1 GDPR Data Export ⚠️ CRITICAL

**Gap:** No data export functionality (GDPR right to portability).

**Impact:** Non-compliance with GDPR, legal risk.

**Solution:**

```kotlin
// GDPR data export
interface DataExportManager {
    // Export all user data
    suspend fun exportUserData(format: ExportFormat): ExportResult

    // Generate export (may take time)
    suspend fun requestExport(): ExportJob

    // Check export status
    suspend fun getExportStatus(jobId: String): ExportStatus
}

data class ExportResult(
    val messages: List<Message>,
    val rooms: List<Room>,
    val profile: UserProfile,
    val metadata: ExportMetadata
)

enum class ExportFormat { JSON, CSV, HTML }
```

**Phase:** 3 (Collaboration)

---

### 4.2 Account Deletion Flow ⚠️ CRITICAL

**Gap:** No account deletion flow (GDPR right to be forgotten).

**Impact:** Non-compliance with GDPR, legal risk.

**Solution:**

```kotlin
// GDPR account deletion
interface AccountDeletionManager {
    // Request account deletion
    suspend fun requestDeletion(): DeletionRequest

    // Confirm deletion (after cooldown)
    suspend fun confirmDeletion(requestId: String): Result<Unit>

    // Cancel deletion request
    suspend fun cancelDeletion(requestId: String): Result<Unit>

    // Check deletion status
    suspend fun getDeletionStatus(): DeletionStatus
}

data class DeletionRequest(
    val id: String,
    val createdAt: Instant,
    val expiresAt: Instant,  // 7-30 day cooldown
    val willDelete: List<DataType>
)
```

**UI Requirements:**
- Clear warning about data loss
- Cooldown period for change of mind
- Confirmation required
- Export data option before deletion

**Phase:** 1 (Foundation) - basic flow, Phase 3 - full automation

---

### 4.3 Cookie/Consent Management

**Gap:** No discussion of cookie/consent management for tracking.

**Impact:** GDPR/CCPA compliance risk, user trust issues.

**Solution:**

```kotlin
// Consent management
interface ConsentManager {
    // Request consent
    suspend fun requestConsent(type: ConsentType): ConsentResult

    // Check consent status
    fun hasConsented(type: ConsentType): Boolean

    // Withdraw consent
    suspend fun withdrawConsent(type: ConsentType): Result<Unit>

    // Get all consents
    fun getAllConsents(): Map<ConsentType, Boolean>
}

enum class ConsentType {
    Analytics,      // Usage analytics
    CrashReporting, // Crash reports
    Personalization,// Personalized features
    Marketing       // Marketing communications
}
```

**UI Requirements:**
- First-launch consent dialog
- Settings screen for managing consents
- Clear explanation of each consent type

**Phase:** 1 (Foundation)

---

## 5. Infrastructure Gaps

### 5.1 CI/CD Pipeline

**Gap:** No CI/CD pipeline for mobile builds.

**Impact:** Manual builds, slow iteration, human error.

**Solution:**

```yaml
# GitHub Actions workflow
name: Mobile CI/CD

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  build-android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up JDK
        uses: actions/setup-java@v3
        with:
          java-version: '17'
          distribution: 'temurin'

      - name: Build debug APK
        run: ./gradlew assembleDebug

      - name: Run tests
        run: ./gradlew test

      - name: Upload APK
        uses: actions/upload-artifact@v3
        with:
          name: app-debug
          path: app/build/outputs/apk/debug/*.apk

      - name: Deploy to Play Store (internal)
        if: github.ref == 'refs/heads/main'
        run: ./gradlew publishRelease

  build-ios:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Xcode
        uses: maxim-lobanov/setup-xcode@v1
        with:
          xcode-version: latest-stable

      - name: Build and test
        run: |
          xcodebuild -scheme ArmorClaw -sdk iphonesimulator \
            -destination 'platform=iOS Simulator,name=iPhone 14' \
            clean build test
```

**Phase:** 1 (Foundation)

---

### 5.2 Code Signing & Distribution

**Gap:** No code signing strategy for release builds.

**Impact:** Can't distribute to app stores, security issues.

**Solution:**

```kotlin
// Android signing configuration
android {
    signingConfigs {
        create("release") {
            // Use environment variables for security
            storeFile = file(System.getenv("KEYSTORE_FILE") ?: "keystore.jks")
            storePassword = System.getenv("KEYSTORE_PASSWORD")
            keyAlias = System.getenv("KEY_ALIAS")
            keyPassword = System.getenv("KEY_PASSWORD")
        }
    }

    buildTypes {
        release {
            signingConfig = signingConfigs.getByName("release")
            isMinifyEnabled = true
            isShrinkResources = true
        }
    }
}

// iOS: Use Xcode managed signing or fastlane match
```

**Phase:** 1 (Foundation)

---

### 5.3 Feature Flag System

**Gap:** No feature flag system for gradual rollouts.

**Impact:** Can't A/B test, all-or-nothing releases.

**Solution:**

```kotlin
// Feature flag system
interface FeatureFlagManager {
    // Check if feature enabled
    fun isFeatureEnabled(feature: Feature): Boolean

    // Get feature config
    fun getFeatureConfig(feature: Feature): Map<String, Any>

    // Refresh flags from remote
    suspend fun refreshFlags()
}

enum class Feature {
    VOICE_INPUT,
    TASK_AUTOMATION,
    LONG_TERM_MEMORY,
    SHARED_CONVERSATIONS,
    HUMAN_HANDOFF,
    ANALYTICS_DASHBOARD
}

// Integration with remote config
class RemoteFeatureFlagManager(
    private val remoteConfig: RemoteConfigManager
) : FeatureFlagManager {
    override fun isFeatureEnabled(feature: Feature): Boolean {
        return remoteConfig.getBoolean("features.${feature.name.lowercase()}", false)
    }
}
```

**Phase:** 2 (Intelligence)

---

### 5.4 Analytics Integration

**Gap:** Mentioned but no detailed analytics implementation.

**Impact:** No insight into user behavior, can't make data-driven decisions.

**Solution:**

```kotlin
// Privacy-preserving analytics
interface AnalyticsManager {
    // Track event (PII-free)
    fun trackEvent(event: AnalyticsEvent)

    // Track screen view
    fun trackScreen(screen: String)

    // Set user properties (hashed, no PII)
    fun setUserProperty(key: String, value: String)

    // Reset analytics (logout)
    fun resetAnalytics()
}

data class AnalyticsEvent(
    val name: String,
    val properties: Map<String, Any>,
    val piiFree: Boolean = true  // Enforce PII-free
)

// Integration: Firebase Analytics, Mixpanel, etc.
```

**Privacy Requirements:**
- No PII in analytics
- Hash user IDs
- Aggregate data only
- Allow opt-out

**Phase:** 3 (Collaboration)

---

### 5.5 A/B Testing Infrastructure

**Gap:** No A/B testing infrastructure for UX optimization.

**Impact:** Can't optimize UX, rely on guesses.

**Solution:**

```kotlin
// A/B testing framework
interface ExperimentManager {
    // Get experiment variant
    fun getVariant(experiment: String): String?

    // Track experiment exposure
    fun trackExposure(experiment: String, variant: String)

    // Track conversion
    fun trackConversion(experiment: String, variant: String)
}

// Example experiment
data class ButtonPlacementExperiment(
    val variant: String  // "top", "bottom", "inline"
)

// Integration with remote config
class RemoteExperimentManager(
    private val remoteConfig: RemoteConfigManager
) : ExperimentManager {
    override fun getVariant(experiment: String): String? {
        return remoteConfig.getString("experiments.$experiment")
    }
}
```

**Phase:** 4 (Polish & Launch)

---

## 6. Updated Delivery Phases

### Phase 0: Prep (2 weeks - extended from 1)

**Additions:**
- Onboarding flow wireframes
- Empty states and loading states
- Error state designs
- Security audit requirements
- Performance benchmarks

### Phase 1: Foundation (6-8 weeks - extended from 4-6)

**Additions:**
- ✅ Matrix auth, chat UI, text/voice inputs, security baseline
- ➕ Offline/sync strategy
- ➕ Push notifications
- ➕ Connection recovery
- ➕ Biometric authentication
- ➕ App lifecycle handling
- ➕ Empty states and loading states
- ➕ Input validation
- ➕ Account deletion flow (basic)
- ➕ Consent management
- ➕ Crash reporting
- ➕ CI/CD pipeline

### Phase 2: Intelligence (5-6 weeks - extended from 4)

**Additions:**
- ✅ Rich responses, tasks, memory/feedback
- ➕ Typing indicators
- ➕ Read receipts
- ➕ Search functionality
- ➕ Screenshot detection
- ➕ Secure time handling
- ➕ Feature flags
- ➕ Certificate pinning

### Phase 3: Collaboration (5-6 weeks - extended from 3-4)

**Additions:**
- ✅ Sharing, handoff, analytics, branding
- ➕ GDPR data export
- ➕ Full account deletion
- ➕ Analytics integration
- ➕ A/B testing infrastructure
- ➕ OTA configuration

### Phase 4: Polish & Launch (3-4 weeks - extended from 2)

**Additions:**
- ✅ Performance profiling, rollback plans, store submissions
- ➕ App size optimization
- ➕ Security audit completion
- ➕ Performance benchmarking

**Total Timeline:** 21-30 weeks (up from 14-20 weeks)

---

## 7. Priority Matrix

| Gap | Criticality | Effort | Phase |
|-----|-------------|--------|-------|
| Offline/Sync Strategy | ⚠️ HIGH | HIGH | 1 |
| Matrix Disconnection Recovery | ⚠️ HIGH | MEDIUM | 1 |
| Push Notifications | ⚠️ HIGH | MEDIUM | 1 |
| Onboarding Flow | ⚠️ HIGH | MEDIUM | 1 |
| Certificate Pinning | ⚠️ HIGH | LOW | 1 |
| GDPR Data Export | ⚠️ HIGH | HIGH | 3 |
| Account Deletion | ⚠️ HIGH | MEDIUM | 1 |
| Input Validation UX | MEDIUM | LOW | 1 |
| Typing Indicators | MEDIUM | LOW | 2 |
| Read Receipts | MEDIUM | LOW | 2 |
| Search Functionality | MEDIUM | HIGH | 2 |
| Feature Flags | MEDIUM | MEDIUM | 2 |
| Biometric Integration | MEDIUM | MEDIUM | 1 |
| Crash Reporting | MEDIUM | LOW | 1 |
| CI/CD Pipeline | LOW | MEDIUM | 1 |
| A/B Testing | LOW | HIGH | 4 |

---

## 8. Next Steps

1. **Review this gap analysis** with stakeholders
2. **Prioritize gaps** based on target market and timeline
3. **Update wireframes** to include missing UX elements
4. **Create technical specifications** for each gap solution
5. **Adjust sprint plan** to include new tasks
6. **Update delivery timeline** with realistic estimates

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-10
**Status:** Awaiting Review
