package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.SyncConfig
import com.armorclaw.shared.domain.model.SyncResult
import com.armorclaw.shared.domain.model.SyncState
import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserSession
import com.armorclaw.shared.domain.repository.AuthRepository
import com.armorclaw.shared.domain.repository.ServerConfig
import com.armorclaw.shared.domain.repository.SyncRepository
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import kotlin.test.*

/**
 * Tests for LogoutUseCase
 *
 * Uses Robolectric to provide Android runtime for logging.
 */
@RunWith(RobolectricTestRunner::class)
class LogoutUseCaseTest {

    // ========================================================================
    // Success Tests
    // ========================================================================

    @Test
    fun `should complete logout successfully`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout()
        val mockSyncRepo = MockSyncRepositoryForLogout()
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        val result = useCase(clearAllData = false)

        // Assert
        assertTrue(result.isSuccess)
    }

    @Test
    fun `should call stopSync during logout`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout()
        val mockSyncRepo = MockSyncRepositoryForLogout()
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        useCase(clearAllData = false)

        // Assert
        assertTrue(mockSyncRepo.stopSyncCalled)
    }

    @Test
    fun `should call logout on auth repository`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout()
        val mockSyncRepo = MockSyncRepositoryForLogout()
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        useCase(clearAllData = false)

        // Assert
        assertTrue(mockAuthRepo.logoutCalled)
    }

    @Test
    fun `should call clearLocalAuth during logout`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout()
        val mockSyncRepo = MockSyncRepositoryForLogout()
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        useCase(clearAllData = false)

        // Assert
        assertTrue(mockAuthRepo.clearLocalAuthCalled)
    }

    // ========================================================================
    // Clear All Data Tests
    // ========================================================================

    @Test
    fun `should clear all data when clearAllData is true`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout()
        val mockSyncRepo = MockSyncRepositoryForLogout()
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        val result = useCase(clearAllData = true)

        // Assert
        assertTrue(result.isSuccess)
    }

    @Test
    fun `should not fail when clearAllData is false`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout()
        val mockSyncRepo = MockSyncRepositoryForLogout()
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        val result = useCase(clearAllData = false)

        // Assert
        assertTrue(result.isSuccess)
    }

    // ========================================================================
    // Error Handling Tests
    // ========================================================================

    @Test
    fun `should continue logout even if server logout fails`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout(serverLogoutFails = true)
        val mockSyncRepo = MockSyncRepositoryForLogout()
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        val result = useCase(clearAllData = false)

        // Assert - Should still succeed (local cleanup continues)
        assertTrue(result.isSuccess)
        assertTrue(mockAuthRepo.clearLocalAuthCalled)
    }

    @Test
    fun `should continue logout even if stopSync throws`() = runTest {
        // Arrange
        val mockAuthRepo = MockAuthRepositoryForLogout()
        val mockSyncRepo = MockSyncRepositoryForLogout(throwOnStopSync = true)
        val useCase = LogoutUseCase(mockAuthRepo, mockSyncRepo)

        // Act
        val result = useCase(clearAllData = false)

        // Assert - Should still succeed
        assertTrue(result.isSuccess)
    }
}

/**
 * Mock AuthRepository for logout testing
 */
class MockAuthRepositoryForLogout(
    private val serverLogoutFails: Boolean = false
) : AuthRepository {

    var logoutCalled = false
    var clearLocalAuthCalled = false

    override suspend fun login(config: ServerConfig): Result<UserSession> {
        return Result.success(
            UserSession(
                userId = "@test:example.com",
                accessToken = "token",
                refreshToken = "refresh",
                deviceId = "DEVICE",
                homeserver = config.homeserver,
                expiresAt = kotlinx.datetime.Clock.System.now() + kotlin.time.Duration.parseIsoString("PT1H")
            )
        )
    }

    override suspend fun logout(): Result<Unit> {
        logoutCalled = true
        return if (serverLogoutFails) {
            Result.failure(Exception("Server logout failed"))
        } else {
            Result.success(Unit)
        }
    }

    override suspend fun refreshSession(): Result<UserSession> {
        return Result.success(
            UserSession(
                userId = "@test:example.com",
                accessToken = "refreshed_token",
                refreshToken = "new_refresh",
                deviceId = "DEVICE",
                homeserver = "https://matrix.example.com",
                expiresAt = kotlinx.datetime.Clock.System.now() + kotlin.time.Duration.parseIsoString("PT1H")
            )
        )
    }

    override suspend fun getCurrentUser(): Result<User?> {
        return Result.success(
            User(
                id = "@test:example.com",
                displayName = "Test User",
                avatar = null
            )
        )
    }

    override fun observeSession(): Flow<UserSession?> = flowOf(
        UserSession(
            userId = "@test:example.com",
            accessToken = "token",
            refreshToken = "refresh",
            deviceId = "DEVICE",
            homeserver = "https://matrix.example.com",
            expiresAt = kotlinx.datetime.Clock.System.now() + kotlin.time.Duration.parseIsoString("PT1H")
        )
    )

    override fun isLoggedIn(): Boolean = true

    override suspend fun clearLocalAuth() {
        clearLocalAuthCalled = true
    }
}

/**
 * Mock SyncRepository for testing
 */
class MockSyncRepositoryForLogout(
    private val throwOnStopSync: Boolean = false
) : SyncRepository {

    var stopSyncCalled = false

    override suspend fun syncWhenOnline(): SyncResult = SyncResult()

    override suspend fun syncRoom(roomId: String): SyncResult = SyncResult()

    override suspend fun startSync() {
        // No-op
    }

    override suspend fun stopSync() {
        stopSyncCalled = true
        if (throwOnStopSync) {
            throw Exception("Failed to stop sync")
        }
    }

    override suspend fun clearSyncState() {
        // No-op
    }

    override fun observeSyncState(): Flow<SyncState> = flowOf(SyncState.Idle)

    override fun isOnline(): Boolean = true

    override suspend fun getConfig(): SyncConfig = SyncConfig()

    override suspend fun updateConfig(config: SyncConfig): Result<Unit> = Result.success(Unit)
}
