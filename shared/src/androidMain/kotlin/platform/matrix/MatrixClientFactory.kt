package com.armorclaw.shared.platform.matrix

import com.armorclaw.shared.platform.logging.LoggerDelegate
import com.armorclaw.shared.platform.logging.LogTag
import io.ktor.client.*
import kotlinx.serialization.json.Json

/**
 * Android implementation of MatrixClientFactory
 *
 * Creates MatrixClientImpl instances with all required dependencies:
 * - MatrixApiService for HTTP API calls
 * - MatrixSessionStorage for secure session persistence
 * - MatrixSyncManager for real-time sync
 */
actual object MatrixClientFactory {
    private val logger = LoggerDelegate(LogTag.Network.MatrixClient)

    // Default dependencies (can be overridden for testing)
    private var defaultHttpClient: HttpClient? = null
    private var defaultJson: Json? = null
    private var defaultSessionStorage: MatrixSessionStorage? = null
    private var defaultSyncManager: MatrixSyncManager? = null

    /**
     * Initialize the factory with default dependencies
     *
     * Should be called during app initialization (e.g., in DI module)
     */
    fun initialize(
        httpClient: HttpClient,
        json: Json,
        sessionStorage: MatrixSessionStorage,
        syncManager: MatrixSyncManager
    ) {
        this.defaultHttpClient = httpClient
        this.defaultJson = json
        this.defaultSessionStorage = sessionStorage
        this.defaultSyncManager = syncManager
        logger.logInfo("MatrixClientFactory initialized")
    }

    /**
     * Create a new Matrix client instance
     */
    actual fun create(config: MatrixClientConfig): MatrixClient {
        logger.logInfo("Creating new Matrix client")

        val httpClient = defaultHttpClient ?: run {
            logger.logError("MatrixClientFactory not initialized. Call initialize() first.")
            throw IllegalStateException("MatrixClientFactory not initialized. Call initialize() first.")
        }
        val json = defaultJson ?: Json { ignoreUnknownKeys = true; isLenient = true }
        val sessionStorage = defaultSessionStorage ?: run {
            logger.logError("MatrixSessionStorage not provided")
            throw IllegalStateException("MatrixSessionStorage not provided")
        }
        val syncManager = defaultSyncManager ?: run {
            logger.logError("MatrixSyncManager not provided")
            throw IllegalStateException("MatrixSyncManager not provided")
        }

        val apiService = MatrixApiService(httpClient, json)
        return MatrixClientImpl(apiService, sessionStorage, syncManager, json, config)
    }

    /**
     * Create a Matrix client from a stored session
     */
    actual fun createFromSession(session: MatrixSession, config: MatrixClientConfig): MatrixClient {
        logger.logInfo("Creating Matrix client from stored session", mapOf(
            "userId" to session.userId
        ))

        val client = create(config)

        // Restore the session in the client
        kotlinx.coroutines.runBlocking {
            (client as? MatrixClientImpl)?.restoreSession(session)
        }

        return client
    }

    /**
     * Create a Matrix client with custom dependencies (for testing)
     */
    fun createWithDependencies(
        apiService: MatrixApiService,
        sessionStorage: MatrixSessionStorage,
        syncManager: MatrixSyncManager,
        json: Json,
        config: MatrixClientConfig
    ): MatrixClient {
        logger.logInfo("Creating Matrix client with custom dependencies")
        return MatrixClientImpl(apiService, sessionStorage, syncManager, json, config)
    }
}
