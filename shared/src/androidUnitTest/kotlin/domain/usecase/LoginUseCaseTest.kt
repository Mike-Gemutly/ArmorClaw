package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.User
import com.armorclaw.shared.domain.model.UserSession
import com.armorclaw.shared.domain.repository.AuthRepository
import com.armorclaw.shared.domain.repository.ServerConfig
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import kotlin.test.*

/**
 * Tests for LoginUseCase
 *
 * Uses Robolectric to provide Android runtime for logging.
 */
@RunWith(RobolectricTestRunner::class)
class LoginUseCaseTest {

    // ========================================================================
    // Validation Tests
    // ========================================================================

    @Test
    fun `should fail when homeserver is blank`() = runTest {
        // Arrange
        val useCase = LoginUseCase(MockAuthRepositoryForLogin())
        val config = ServerConfig(
            homeserver = "",
            username = "testuser",
            password = "password123"
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isFailure)
        val exception = result.exceptionOrNull()
        assertTrue(exception is ValidationException)
        assertTrue(exception?.message?.contains("Homeserver") == true)
    }

    @Test
    fun `should fail when homeserver has invalid format`() = runTest {
        // Arrange
        val useCase = LoginUseCase(MockAuthRepositoryForLogin())
        val config = ServerConfig(
            homeserver = "invalid-url",
            username = "testuser",
            password = "password123"
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isFailure)
        val exception = result.exceptionOrNull()
        assertTrue(exception is ValidationException)
        assertTrue(exception?.message?.contains("homeserver") == true)
    }

    @Test
    fun `should accept https homeserver URL`() = runTest {
        // Arrange
        val mockRepo = MockAuthRepositoryForLogin()
        val useCase = LoginUseCase(mockRepo)
        val config = ServerConfig(
            homeserver = "https://matrix.example.com",
            username = "testuser",
            password = "password123"
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isSuccess)
    }

    @Test
    fun `should accept http homeserver URL`() = runTest {
        // Arrange
        val mockRepo = MockAuthRepositoryForLogin()
        val useCase = LoginUseCase(mockRepo)
        val config = ServerConfig(
            homeserver = "http://localhost:8080",
            username = "testuser",
            password = "password123"
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isSuccess)
    }

    @Test
    fun `should fail when username is blank`() = runTest {
        // Arrange
        val useCase = LoginUseCase(MockAuthRepositoryForLogin())
        val config = ServerConfig(
            homeserver = "https://matrix.example.com",
            username = "",
            password = "password123"
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isFailure)
        val exception = result.exceptionOrNull()
        assertTrue(exception is ValidationException)
        assertTrue(exception?.message?.contains("Username") == true)
    }

    @Test
    fun `should fail when password is blank`() = runTest {
        // Arrange
        val useCase = LoginUseCase(MockAuthRepositoryForLogin())
        val config = ServerConfig(
            homeserver = "https://matrix.example.com",
            username = "testuser",
            password = ""
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isFailure)
        val exception = result.exceptionOrNull()
        assertTrue(exception is ValidationException)
        assertTrue(exception?.message?.contains("Password") == true)
    }

    // ========================================================================
    // Success Tests
    // ========================================================================

    @Test
    fun `should return UserSession on successful login`() = runTest {
        // Arrange
        val mockRepo = MockAuthRepositoryForLogin()
        val useCase = LoginUseCase(mockRepo)
        val config = ServerConfig(
            homeserver = "https://matrix.example.com",
            username = "testuser",
            password = "password123"
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isSuccess)
        val session = result.getOrNull()
        assertNotNull(session)
        assertEquals("@testuser:example.com", session?.userId)
        assertNotNull(session?.accessToken)
    }

    // ========================================================================
    // Error Handling Tests
    // ========================================================================

    @Test
    fun `should return failure when auth repository fails`() = runTest {
        // Arrange
        val mockRepo = MockAuthRepositoryForLogin(shouldFail = true)
        val useCase = LoginUseCase(mockRepo)
        val config = ServerConfig(
            homeserver = "https://matrix.example.com",
            username = "testuser",
            password = "password123"
        )

        // Act
        val result = useCase(config)

        // Assert
        assertTrue(result.isFailure)
    }
}

/**
 * Mock AuthRepository for testing
 */
class MockAuthRepositoryForLogin(
    private val shouldFail: Boolean = false
) : AuthRepository {

    override suspend fun login(config: ServerConfig): Result<UserSession> {
        if (shouldFail) {
            return Result.failure(Exception("Authentication failed"))
        }
        return Result.success(
            UserSession(
                userId = "@${config.username}:example.com",
                accessToken = "test_access_token",
                refreshToken = "test_refresh_token",
                deviceId = "TESTDEVICE",
                homeserver = config.homeserver,
                expiresAt = kotlinx.datetime.Clock.System.now() + kotlin.time.Duration.parseIsoString("PT1H")
            )
        )
    }

    override suspend fun logout(): Result<Unit> {
        return Result.success(Unit)
    }

    override suspend fun refreshSession(): Result<UserSession> {
        return Result.success(
            UserSession(
                userId = "@testuser:example.com",
                accessToken = "refreshed_token",
                refreshToken = "new_refresh_token",
                deviceId = "TESTDEVICE",
                homeserver = "https://matrix.example.com",
                expiresAt = kotlinx.datetime.Clock.System.now() + kotlin.time.Duration.parseIsoString("PT1H")
            )
        )
    }

    override suspend fun getCurrentUser(): Result<User?> {
        return if (shouldFail) {
            Result.success(null)
        } else {
            Result.success(
                User(
                    id = "@testuser:example.com",
                    displayName = "Test User",
                    avatar = null
                )
            )
        }
    }

    override fun observeSession(): Flow<UserSession?> = flowOf(
        if (shouldFail) null else UserSession(
            userId = "@testuser:example.com",
            accessToken = "test_access_token",
            refreshToken = "test_refresh_token",
            deviceId = "TESTDEVICE",
            homeserver = "https://matrix.example.com",
            expiresAt = kotlinx.datetime.Clock.System.now() + kotlin.time.Duration.parseIsoString("PT1H")
        )
    )

    override fun isLoggedIn(): Boolean = !shouldFail

    override suspend fun clearLocalAuth() {
        // No-op for mock
    }
}
