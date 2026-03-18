package com.armorclaw.shared.data.repository

import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserSession
import com.armorclaw.shared.domain.repository.AuthRepository
import com.armorclaw.shared.domain.repository.ServerConfig
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryErrorBoundarySuspend
import com.armorclaw.shared.platform.logging.repositoryLogger
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.matrix.MatrixSession
import com.armorclaw.shared.platform.matrix.MatrixSessionStorage
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant

/**
 * Implementation of AuthRepository using Matrix SDK
 *
 * ## Architecture
 * ```
 * AuthRepositoryImpl
 *      ├── MatrixClient (primary authentication)
 *      │      └── matrix-rust-sdk → Conduit Homeserver
 *      │
 *      └── MatrixSessionStorage (encrypted persistence)
 *              └── EncryptedSharedPreferences (Android)
 * ```
 *
 * ## Migration Status
 * - [x] Login via Matrix SDK
 * - [x] Session persistence with encrypted storage
 * - [x] Session restoration on app start
 * - [ ] Logout with device deactivation
 * - [ ] Token refresh
 */
class AuthRepositoryImpl(
    private val matrixClient: MatrixClient,
    private val sessionStorage: MatrixSessionStorage
) : AuthRepository {

    private val logger = repositoryLogger("AuthRepository", LogTag.Data.AuthRepository)

    // In-memory session state (backed by secure storage)
    private val _session = MutableStateFlow<UserSession?>(null)
    private val _matrixSession = MutableStateFlow<MatrixSession?>(null)

    init {
        // Initialize from secure storage
        _matrixSession.value = run {
            // Synchronous load for initialization - handled in loadStoredSession()
            null
        }
    }

    /**
     * Load any stored session from secure storage
     * Should be called during app initialization
     */
    suspend fun loadStoredSession(): Result<UserSession?> {
        return sessionStorage.loadSession().fold(
            onSuccess = { matrixSession ->
                if (matrixSession != null) {
                    // Check if session has expired
                    if (matrixSession.isExpired()) {
                        logger.logOperationError("loadStoredSession",
                            SessionExpiredException("Session has expired"))

                        // Clear expired session
                        sessionStorage.clearSession()
                        return Result.success(null)
                    }

                    logger.logOperationSuccess("loadStoredSession", matrixSession.userId)

                    // Restore Matrix client with session
                    matrixClient.restoreSession(matrixSession)

                    // Update state with proper expiration
                    _matrixSession.value = matrixSession
                    _session.value = UserSession(
                        userId = matrixSession.userId,
                        accessToken = matrixSession.accessToken,
                        refreshToken = matrixSession.refreshToken ?: "",
                        deviceId = matrixSession.deviceId,
                        homeserver = matrixSession.homeserver,
                        expiresAt = matrixSession.expiresAt?.let { Instant.fromEpochSeconds(it) }
                            ?: Instant.DISTANT_FUTURE
                    )

                    // Warn if session is expiring soon
                    if (matrixSession.isExpiringSoon(300)) {
                        logger.logOperationStart("session_expiring_soon", mapOf(
                            "remaining_seconds" to (matrixSession.remainingTimeSeconds() ?: "unknown")
                        ))
                    }

                    Result.success(_session.value)
                } else {
                    logger.logOperationSuccess("loadStoredSession", "no stored session")
                    Result.success(null)
                }
            },
            onFailure = { error ->
                logger.logOperationError("loadStoredSession", error)
                Result.failure(error)
            }
        )
    }

    override suspend fun login(config: ServerConfig): Result<UserSession> = repositoryErrorBoundarySuspend(
        logger = logger,
        operation = "login"
    ) {
        logger.logOperationStart("login", mapOf(
            "homeserver" to config.homeserver,
            "username" to config.username,
            "authMethod" to "matrix_sdk"
        ))

        logger.logNetworkRequest("MatrixClient.login", "POST")

        val startTime = System.currentTimeMillis()

        // Use Matrix SDK for authentication
        val matrixResult = matrixClient.login(
            homeserver = config.homeserver,
            username = config.username,
            password = config.password,
            deviceId = config.deviceId
        )

        matrixResult.fold(
            onSuccess = { matrixSession ->
                logger.logNetworkResponse(
                    "MatrixClient.login",
                    200,
                    System.currentTimeMillis() - startTime
                )

                // Calculate expiration if not provided by server
                val effectiveSession = if (matrixSession.expiresAt == null) {
                    MatrixSession.withExpiration(
                        userId = matrixSession.userId,
                        deviceId = matrixSession.deviceId,
                        accessToken = matrixSession.accessToken,
                        refreshToken = matrixSession.refreshToken,
                        homeserver = matrixSession.homeserver,
                        displayName = matrixSession.displayName,
                        avatarUrl = matrixSession.avatarUrl,
                        expiresIn = matrixSession.expiresIn
                    )
                } else {
                    matrixSession
                }

                // Convert MatrixSession to UserSession with proper expiration
                val userSession = UserSession(
                    userId = effectiveSession.userId,
                    accessToken = effectiveSession.accessToken,
                    refreshToken = effectiveSession.refreshToken ?: "",
                    deviceId = effectiveSession.deviceId,
                    homeserver = effectiveSession.homeserver,
                    expiresAt = effectiveSession.expiresAt?.let { Instant.fromEpochSeconds(it) }
                        ?: Instant.DISTANT_FUTURE
                )

                // Store sessions in memory
                _session.value = userSession
                _matrixSession.value = effectiveSession

                // Save to secure storage
                sessionStorage.saveSession(effectiveSession)
                logger.logDatabaseQuery("INSERT INTO session", mapOf(
                    "userId" to effectiveSession.userId,
                    "storage" to "encrypted",
                    "expires_at" to (effectiveSession.expiresAt ?: "never")
                ))

                logger.logTransformation("MatrixSession", "UserSession")
                logger.logOperationSuccess("login", "User ${userSession.userId}")

                userSession
            },
            onFailure = { error ->
                logger.logNetworkResponse(
                    "MatrixClient.login",
                    401,
                    System.currentTimeMillis() - startTime
                )
                throw error
            }
        )
    }

    override suspend fun logout(): Result<Unit> = repositoryErrorBoundarySuspend(
        logger = logger,
        operation = "logout"
    ) {
        logger.logOperationStart("logout")

        val startTime = System.currentTimeMillis()

        // Logout via Matrix SDK (invalidates session on server)
        matrixClient.logout().fold(
            onSuccess = {
                logger.logNetworkResponse(
                    "MatrixClient.logout",
                    200,
                    System.currentTimeMillis() - startTime
                )

                // Clear local session state
                _session.value = null
                _matrixSession.value = null

                // Clear secure storage
                sessionStorage.clearSession()
                logger.logDatabaseQuery("DELETE FROM session", mapOf("storage" to "encrypted"))

                logger.logOperationSuccess("logout")
            },
            onFailure = { error ->
                // Even if server logout fails, clear local state
                _session.value = null
                _matrixSession.value = null
                sessionStorage.clearSession()
                throw error
            }
        )
    }

    override suspend fun refreshSession(): Result<UserSession> = repositoryErrorBoundarySuspend(
        logger = logger,
        operation = "refreshSession"
    ) {
        logger.logOperationStart("refreshSession")

        val currentSession = _matrixSession.value
            ?: throw IllegalStateException("No session to refresh")

        // Matrix SDK handles token refresh internally
        // This method is kept for compatibility with existing code
        logger.logOperationSuccess("refreshSession", "Session valid")

        _session.value ?: throw IllegalStateException("Session expired")
    }

    override suspend fun getCurrentUser(): Result<User?> = repositoryErrorBoundarySuspend(
        logger = logger,
        operation = "getCurrentUser"
    ) {
        logger.logOperationStart("getCurrentUser")

        // Get current user from Matrix client
        matrixClient.currentUser.value?.let { user ->
            logger.logOperationSuccess("getCurrentUser", user.id)
            user
        } ?: run {
            logger.logOperationSuccess("getCurrentUser", "null (not logged in)")
            null
        }
    }

    override fun observeSession(): Flow<UserSession?> {
        logger.logOperationStart("observeSession")
        return _session.asStateFlow()
    }

    override fun isLoggedIn(): Boolean {
        val loggedIn = _session.value != null && matrixClient.isLoggedIn.value
        logger.logOperationSuccess("isLoggedIn", loggedIn.toString())
        return loggedIn
    }

    override suspend fun clearLocalAuth() {
        logger.logOperationStart("clearLocalAuth")

        // Clear in-memory session state
        _session.value = null
        _matrixSession.value = null

        logger.logOperationSuccess("clearLocalAuth")
    }

    /**
     * Get the Matrix session for use with MatrixClient
     */
    fun getMatrixSession(): MatrixSession? = _matrixSession.value

    /**
     * Restore a previous session
     */
    suspend fun restoreSession(session: UserSession): Result<Unit> {
        return try {
            // Check if the session has expired
            val now = Clock.System.now()
            if (session.expiresAt < now) {
                return Result.failure(SessionExpiredException("Session has expired"))
            }

            val matrixSession = MatrixSession(
                userId = session.userId,
                deviceId = session.deviceId,
                accessToken = session.accessToken,
                refreshToken = session.refreshToken,
                homeserver = session.homeserver
            )

            matrixClient.restoreSession(matrixSession).fold(
                onSuccess = {
                    _session.value = session
                    _matrixSession.value = matrixSession
                    Result.success(Unit)
                },
                onFailure = { Result.failure(it) }
            )
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * Exception thrown when a session has expired
 */
class SessionExpiredException(message: String) : Exception(message)
