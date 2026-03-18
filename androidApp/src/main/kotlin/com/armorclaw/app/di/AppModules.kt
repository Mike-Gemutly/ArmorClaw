package com.armorclaw.app.di

import com.armorclaw.app.BuildConfig
import com.armorclaw.shared.data.repository.AuthRepositoryImpl

// Placeholder imports for Phase 1 security components
// These will be uncommented when the files are created
// import com.armorclaw.shared.domain.repository.VaultKey
// import com.armorclaw.shared.domain.repository.ShadowPlaceholder
// import com.armorclaw.shared.domain.repository.VaultKeyCategory
// import com.armorclaw.shared.domain.repository.VaultKeySensitivity
// import com.armorclaw.app.security.KeystoreManager
// import com.armorclaw.app.security.SqlCipherProvider
// import com.armorclaw.app.security.VaultRepository
// import com.armorclaw.shared.domain.security.PiiRegistry
// import com.armorclaw.shared.domain.security.ShadowMap
// import com.armorclaw.shared.domain.security.AgentRequestInterceptor

// Security imports - Cold Vault Phase 1
import com.armorclaw.app.security.KeystoreManager
import com.armorclaw.app.security.SqlCipherProvider
import com.armorclaw.app.security.VaultRepository
import com.armorclaw.shared.domain.security.PiiRegistry
import com.armorclaw.shared.domain.security.ShadowMap
import com.armorclaw.shared.domain.security.AgentRequestInterceptor
import com.armorclaw.shared.domain.store.vault.VaultStore
import com.armorclaw.app.data.repository.MessageRepositoryImpl
import com.armorclaw.app.data.repository.RoomRepositoryImpl
import com.armorclaw.app.viewmodels.AppPreferences
import com.armorclaw.app.viewmodels.ProfileViewModel
import com.armorclaw.app.viewmodels.SetupViewModel
import com.armorclaw.app.viewmodels.ChatViewModel
import com.armorclaw.app.viewmodels.AgentManagementViewModel
import com.armorclaw.app.viewmodels.HitlViewModel
import com.armorclaw.app.viewmodels.WorkflowViewModel
import com.armorclaw.app.viewmodels.ServerConnectionViewModel
import com.armorclaw.app.viewmodels.SyncStatusViewModel
import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.domain.repository.AgentRepository
import com.armorclaw.shared.domain.repository.AuthRepository
import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.domain.repository.RoomRepository
import com.armorclaw.shared.domain.repository.SyncRepository
import com.armorclaw.shared.domain.repository.UserRepository
import com.armorclaw.shared.domain.repository.WorkflowRepository
import com.armorclaw.shared.domain.model.SyncConfig
import com.armorclaw.shared.domain.model.SyncResult
import com.armorclaw.shared.domain.model.SyncState
import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserPresence
import com.armorclaw.shared.domain.model.UserSession
import com.armorclaw.shared.domain.usecase.LogoutUseCase
import com.armorclaw.shared.platform.bridge.BridgeAdminClient
import com.armorclaw.shared.platform.bridge.BridgeAdminClientImpl
import com.armorclaw.shared.platform.bridge.BridgeClientFactory
import com.armorclaw.shared.platform.bridge.BridgeConfig
import com.armorclaw.shared.platform.bridge.BridgeRepository
import com.armorclaw.shared.platform.bridge.BridgeRpcClient
import com.armorclaw.shared.platform.bridge.BridgeWebSocketClient
import com.armorclaw.shared.platform.bridge.BridgeWebSocketClientImpl
import com.armorclaw.shared.platform.bridge.InviteService
import com.armorclaw.shared.platform.bridge.SetupService
import com.armorclaw.shared.platform.bridge.WebSocketConfig
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.matrix.MatrixClientConfig
import com.armorclaw.shared.platform.matrix.MatrixClientFactory
import com.armorclaw.shared.platform.matrix.MatrixClientImpl
import com.armorclaw.shared.platform.matrix.MatrixSessionStorage
import com.armorclaw.shared.platform.matrix.MatrixSessionStorageFactory
import com.armorclaw.shared.platform.matrix.MatrixSyncManager
import com.armorclaw.shared.platform.notification.PushNotificationRepository
import com.armorclaw.shared.platform.notification.PushNotificationRepositoryImpl
import com.armorclaw.app.viewmodels.InviteViewModel
import com.armorclaw.app.viewmodels.HomeViewModel
import io.ktor.client.HttpClient
import kotlinx.coroutines.runBlocking
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.coroutines.flow.flowOf
import kotlinx.serialization.json.Json
import org.koin.android.ext.koin.androidContext
import org.koin.androidx.viewmodel.dsl.viewModel
import org.koin.dsl.module

/**
 * Security Module - Cold Vault Infrastructure
 *
 * Provides encrypted storage for PII using SQLCipher with keys
 * managed by Android Keystore.
 *
 * Phase 1 Implementation:
 * - SQLCipher for encrypted storage
 * - Android Keystore for key management
 * - PII Shadow Mapping middleware
 */
val securityModule = module {
    // Keystore Manager - manages encryption keys via Android Keystore
    single { KeystoreManager(androidContext()) }

    // Biometric Auth - wraps BiometricAuth singleton with proper context
    single { com.armorclaw.app.platform.BiometricAuthImpl(androidContext()) }

    // SQLCipher Provider - creates encrypted database connections
    single { SqlCipherProvider(androidContext(), get()) }

    // Vault Repository - manages PII storage and retrieval
    single { VaultRepository(get(), get()) }

    // PII Registry - manages PII keys and placeholders
    single { PiiRegistry() }

    // Shadow Map - placeholder mapping middleware
    single { ShadowMap(get()) }

    // Agent Request Interceptor - shadows PII in outgoing requests
    single { AgentRequestInterceptor(get(), get()) }

    // Vault Store - state management for vault keys
    single { VaultStore.create() }
}

val viewModelModule = module {
    viewModel { ProfileViewModel(get(), get()) }
    viewModel { SetupViewModel(get(), get()) }
    viewModel { InviteViewModel(get(), get()) }
    // LogoutUseCase - handles user logout
    factory { LogoutUseCase(get(), get()) }

    // HomeViewModel - displays rooms and active workflows
    viewModel { HomeViewModel(get(), get()) }

    // Agent Management ViewModel - NEW
    viewModel { AgentManagementViewModel(get()) }

    // HITL Approval ViewModel - NEW
    viewModel { HitlViewModel(get()) }

    // Workflow ViewModel - NEW
    viewModel { WorkflowViewModel(get()) }

    // Server Connection ViewModel - NEW (for post-onboarding discovery)
    viewModel { ServerConnectionViewModel(get(), get(), get()) }

    viewModel { SyncStatusViewModel(androidContext()) }

    // ChatViewModel factory - requires roomId parameter
    // Usage: viewModel { parameters -> ChatViewModel(parameters.get(), get(), get(), get()) }
    factory { (roomId: String) ->
        ChatViewModel(
            roomId = roomId,
            matrixClient = get(),
            controlPlaneStore = get(),
            messageRepository = get()
        )
    }
}

val repositoryModule = module {
    // Message Repository - uses BridgeRepository
    single<MessageRepository> { MessageRepositoryImpl(get()) }

    // Room Repository - uses BridgeRepository
    single<RoomRepository> { RoomRepositoryImpl(get()) }
}

val useCaseModule = module {
    // Use cases
    factory { LogoutUseCase(get(), get()) }
}

val bridgeModule = module {
    // Bridge RPC Client configuration
    single<BridgeConfig> {
        // Use development config for debug builds, production for release
        if (BuildConfig.DEBUG) {
            BridgeConfig.DEVELOPMENT
        } else {
            BridgeConfig.PRODUCTION
        }
    }

    // WebSocket configuration
    single<WebSocketConfig> {
        if (BuildConfig.DEBUG) {
            WebSocketConfig.DEVELOPMENT
        } else {
            WebSocketConfig.PRODUCTION
        }
    }

    // Shared HTTP client with WebSocket support
    single<HttpClient> {
        BridgeClientFactory.createHttpClient(get())
    }

    // Bridge RPC Client
    single<BridgeRpcClient> {
        BridgeClientFactory.createClient(get())
    }

    // Bridge Admin Client - admin-only operations
    // Use this for non-messaging operations (license, recovery, platform, push, webrtc)
    single<BridgeAdminClient> {
        BridgeAdminClientImpl(get())
    }

    // Bridge WebSocket Client
    single<BridgeWebSocketClient> {
        BridgeWebSocketClientImpl(get(), get())
    }

    // Matrix Sync Manager - handles direct Matrix /sync for real-time events
    // Created lazily with homeserver URL from config
    single<MatrixSyncManager> {
        val config: BridgeConfig = get()
        val client: HttpClient = get()
        MatrixSyncManager(
            homeserverUrl = config.homeserverUrl,
            httpClient = client
        )
    }

    // Bridge Repository - integrates RPC, WebSocket, and Matrix sync
    single<BridgeRepository> {
        BridgeRepository(
            rpcClient = get(),
            wsClient = get(),
            syncManager = get(),
            config = get()
        )
    }

    // Setup Service - handles initial setup flow
    single<SetupService> {
        SetupService(get(), get(), get())
    }

    // Invite Service - handles invite link generation
    single<InviteService> {
        InviteService(get(), get())
    }

    // Push Notification Repository - handles FCM token registration
    // Uses dual registration: Matrix SDK pusher + Bridge RPC
    // RC-07: Wire pushGatewayUrlProvider so dynamic push gateway URL from
    // QR/discovery/well-known is used instead of the hardcoded default.
    single<PushNotificationRepository> {
        val setupService: SetupService = get()
        PushNotificationRepositoryImpl(
            rpcClient = get(),
            matrixClient = get(),
            pushGatewayUrlProvider = { setupService.config.value.pushGateway }
        )
    }
}

// NOTE: encryptionModule removed in v4.1.0 - EncryptionService stripped.
// Matrix Rust SDK now handles all encryption. Type definitions in EncryptionTypes.kt.

/**
 * Matrix SDK Module
 *
 * Provides the Matrix client for all messaging operations.
 * This replaces the RPC-based messaging in BridgeRpcClient.
 *
 * Migration Status:
 * - [x] MatrixClient interface created
 * - [x] Placeholder implementation
 * - [x] Rust SDK factory (Android)
 * - [x] Session persistence (EncryptedSharedPreferences)
 * - [ ] Full matrix-rust-sdk integration
 */
val matrixModule = module {
    // Matrix Client configuration
    single<MatrixClientConfig> {
        MatrixClientConfig(
            defaultHomeserver = if (BuildConfig.DEBUG) {
                "https://matrix.conduit.local"
            } else {
                "https://matrix.armorclaw.app"
            },
            enableEncryption = true,
            enablePresence = true,
            enableTypingIndicators = true,
            enableReadReceipts = true,
            backgroundSync = true
        )
    }

    // Matrix Client - primary interface for messaging
    // Uses factory pattern to support platform-specific implementations
    single<MatrixClient> {
        // Ensure factory is initialized
        val httpClient: HttpClient = get()
        val json: Json = get()
        val sessionStorage: MatrixSessionStorage = get()
        val syncManager: MatrixSyncManager = get()
        val config: MatrixClientConfig = get()

        // Initialize factory if not already done
        try {
            MatrixClientFactory.initialize(
                httpClient = httpClient,
                json = json,
                sessionStorage = sessionStorage,
                syncManager = syncManager
            )
        } catch (e: Exception) {
            // Already initialized, ignore
        }

        // Try to restore session from storage
        // Check for stored session
        val storedSession = runCatching {
            kotlinx.coroutines.runBlocking {
                sessionStorage.loadSession().getOrNull()
            }
        }.getOrNull()

        if (storedSession != null) {
            // Restore from stored session
            try {
                MatrixClientFactory.createFromSession(storedSession, config)
            } catch (e: Exception) {
                // Fallback to new client
                MatrixClientFactory.create(config)
            }
        } else {
            MatrixClientFactory.create(config)
        }
    }

    // Control Plane Store - processes ArmorClaw-specific events
    single<ControlPlaneStore> {
        ControlPlaneStore(
            matrixClient = get(),
            workflowRepository = get(),
            agentRepository = get(),
            json = get()
        )
    }
}

/**
 * Workflow and Agent Repositories
 *
 * These are placeholder implementations that will be replaced
 * with proper database-backed implementations.
 */
val workflowAgentModule = module {
    // Workflow Repository - manages workflow state
    single<WorkflowRepository> {
        object : WorkflowRepository {
            override suspend fun startWorkflow(workflowId: String, type: String, roomId: String, parameters: Map<String, String>) {}
            override suspend fun updateStep(workflowId: String, stepId: String, status: com.armorclaw.shared.platform.matrix.event.StepStatus, output: String?, error: String?) {}
            override suspend fun completeWorkflow(workflowId: String, success: Boolean, result: String?, error: String?) {}
            override suspend fun failWorkflow(workflowId: String, failedAtStep: String, error: String, recoverable: Boolean) {}
            override suspend fun getActiveWorkflows(roomId: String) = emptyList<com.armorclaw.shared.domain.repository.WorkflowInfo>()
            override suspend fun getWorkflowHistory(roomId: String?, limit: Int) = emptyList<com.armorclaw.shared.domain.repository.WorkflowInfo>()
            override suspend fun getWorkflow(workflowId: String): com.armorclaw.shared.domain.repository.WorkflowInfo? = null
        }
    }

    // Agent Repository - manages agent state
    single<AgentRepository> {
        object : AgentRepository {
            override suspend fun taskStarted(agentId: String, taskId: String, taskType: String, roomId: String) {}
            override suspend fun taskCompleted(agentId: String, taskId: String, success: Boolean, result: String?, error: String?) {}
            override suspend fun getAgent(agentId: String): com.armorclaw.shared.domain.repository.AgentInfo? = null
            override suspend fun getRoomAgents(roomId: String) = emptyList<com.armorclaw.shared.domain.repository.AgentInfo>()
            override suspend fun getAgentTasks(agentId: String, limit: Int) = emptyList<com.armorclaw.shared.domain.repository.AgentTaskInfo>()
            override fun observeAgentTasks(agentId: String) = flowOf(emptyList<com.armorclaw.shared.domain.repository.AgentTaskInfo>())
            override suspend fun getActiveRoomTasks(roomId: String) = emptyList<com.armorclaw.shared.domain.repository.AgentTaskInfo>()
            override suspend fun updateAgentCapabilities(agentId: String, capabilities: List<String>) {}
            override suspend fun recordUsage(agentId: String, taskId: String, tokensUsed: Int?, durationMs: Long) {}
        }
    }
}

val platformModule = module {
    // Platform-specific dependencies
    single { AppPreferences(androidContext()) }

    // Simple HTTP Client for Matrix
    single<io.ktor.client.HttpClient> { io.ktor.client.HttpClient() }

    // JSON serializer for event parsing
    single<Json> {
        Json {
            ignoreUnknownKeys = true
            isLenient = true
            encodeDefaults = true
        }
    }

    // Initialize session storage factory
    single<MatrixSessionStorage> {
        MatrixSessionStorageFactory.initialize(androidContext())
        MatrixSessionStorageFactory.create()
    }

    // Matrix Sync Manager - actual implementation
    single<MatrixSyncManager> {
        MatrixSyncManager(
            homeserverUrl = get<MatrixClientConfig>().defaultHomeserver,
            httpClient = get(),
            json = get()
        )
    }
}

// Temporary mock implementations until real ones are added
val mockModule = module {
    // AuthRepository uses Matrix SDK with secure session storage
    single<AuthRepository> { AuthRepositoryImpl(get(), get()) }

    single<SyncRepository> {
        object : SyncRepository {
            override suspend fun syncWhenOnline() = SyncResult()
            override suspend fun syncRoom(roomId: String) = SyncResult()
            override suspend fun startSync() { /* No-op */ }
            override suspend fun stopSync() { /* No-op */ }
            override suspend fun clearSyncState() { /* No-op */ }
            override fun observeSyncState() = flowOf(SyncState.Idle)
            override fun isOnline() = true
            override suspend fun getConfig() = SyncConfig()
            override suspend fun updateConfig(config: SyncConfig) = Result.success(Unit)
        }
    }

    single<UserRepository> {
        object : UserRepository {
            private var currentUser = User(
                id = "mock_user",
                displayName = "John Doe",
                email = "john@example.com",
                avatar = null,
                presence = UserPresence.ONLINE,
                lastActive = Clock.System.now(),
                isVerified = true
            )

            override suspend fun getUser(userId: String) = Result.success(currentUser)
            override suspend fun getCurrentUser() = Result.success(currentUser)
            override suspend fun updateUser(user: User): Result<Unit> {
                currentUser = user
                return Result.success(Unit)
            }
            override fun observeUser(userId: String) = flowOf(currentUser)
            override fun observeCurrentUser() = flowOf(currentUser)
            override suspend fun updatePresence(isOnline: Boolean): Result<Unit> {
                currentUser = currentUser.copy(
                    presence = if (isOnline) UserPresence.ONLINE else UserPresence.OFFLINE
                )
                return Result.success(Unit)
            }
        }
    }
}

val appModules = listOf(
    securityModule,        // Cold Vault - encrypted PII storage (Phase 1)
    viewModelModule,
    repositoryModule,
    platformModule,
    bridgeModule,
    matrixModule,
    workflowAgentModule,
    mockModule
)

// Version tracking for Governor Strategy implementation
val GOVERNOR_VERSION = "4.0.0-alpha01"
val COLD_VAULT_ENABLED = true // Phase 1 complete - Cold Vault operational

/**
 * Phase 1 Implementation Complete:
 *
 * ✅ SQLCipher dependencies added
 * ✅ VaultKey/ShadowPlaceholder data models created
 * ✅ Security module configured in DI
 * ✅ KeystoreManager.kt created - Android Keystore key management
 * ✅ SqlCipherProvider.kt created - Encrypted database factory
 * ✅ VaultRepository.kt created - PII CRUD operations
 * ✅ PiiRegistry.kt created - PII key registry with predefined fields
 * ✅ ShadowMap.kt created - Placeholder mapping for agent transmission
 * ✅ AgentRequestInterceptor.kt created - Outbound request middleware
 */
