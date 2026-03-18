package com.armorclaw.shared.platform.matrix

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import com.armorclaw.shared.platform.logging.LoggerDelegate
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import java.security.GeneralSecurityException

/**
 * Android implementation of MatrixSessionStorage
 *
 * Uses EncryptedSharedPreferences for secure session storage.
 * The storage is encrypted with Android's keystore-backed master key.
 *
 * ## Security Features
 * - AES256-GCM encryption for stored data
 * - Master key stored in Android Keystore
 * - Key rotation support via MasterKey
 *
 * ## Stored Data
 * - User ID
 * - Device ID
 * - Access Token (SENSITIVE)
 * - Refresh Token (SENSITIVE)
 * - Homeserver URL
 * - Display Name
 * - Avatar URL
 */
class MatrixSessionStorageAndroid(
    private val context: Context,
    private val json: Json = Json { ignoreUnknownKeys = true }
) : MatrixSessionStorage {

    private val logger = LoggerDelegate(LogTag.Security.SecureStorage)

    private val _sessionFlow = MutableStateFlow<MatrixSession?>(null)

    private val masterKey: MasterKey by lazy {
        MasterKey.Builder(context, PREFS_FILE_NAME)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
    }

    private val encryptedPrefs by lazy {
        EncryptedSharedPreferences.create(
            context,
            PREFS_FILE_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }

    override suspend fun saveSession(session: MatrixSession): Result<Unit> {
        return try {
            logger.logInfo("Saving Matrix session", mapOf(
                "userId" to session.userId,
                "deviceId" to session.deviceId
            ))

            val sessionJson = json.encodeToString(session)

            encryptedPrefs.edit()
                .putString(KEY_SESSION, sessionJson)
                .putString(KEY_USER_ID, session.userId)
                .putLong(KEY_LAST_UPDATED, System.currentTimeMillis())
                .apply()

            _sessionFlow.value = session

            logger.logInfo("Session saved successfully")
            Result.success(Unit)
        } catch (e: GeneralSecurityException) {
            logger.logError("Security error saving session", e)
            Result.failure(e)
        } catch (e: Exception) {
            logger.logError("Failed to save session", e)
            Result.failure(e)
        }
    }

    override suspend fun loadSession(): Result<MatrixSession?> {
        return try {
            logger.logDebug("Loading Matrix session")

            val sessionJson = encryptedPrefs.getString(KEY_SESSION, null)

            if (sessionJson == null) {
                logger.logDebug("No stored session found")
                return Result.success(null)
            }

            val session = json.decodeFromString<MatrixSession>(sessionJson)

            // Validate session has required fields
            if (session.userId.isEmpty() || session.accessToken.isEmpty()) {
                logger.logWarning("Stored session is invalid, clearing")
                clearSession()
                return Result.success(null)
            }

            _sessionFlow.value = session

            logger.logInfo("Session loaded successfully", mapOf(
                "userId" to session.userId,
                "deviceId" to session.deviceId
            ))

            Result.success(session)
        } catch (e: GeneralSecurityException) {
            logger.logError("Security error loading session", e)
            // Clear corrupted data
            clearSession()
            Result.failure(e)
        } catch (e: Exception) {
            logger.logError("Failed to load session", e)
            clearSession()
            Result.failure(e)
        }
    }

    override suspend fun clearSession(): Result<Unit> {
        return try {
            logger.logInfo("Clearing Matrix session")

            encryptedPrefs.edit()
                .remove(KEY_SESSION)
                .remove(KEY_USER_ID)
                .remove(KEY_LAST_UPDATED)
                .apply()

            _sessionFlow.value = null

            logger.logInfo("Session cleared successfully")
            Result.success(Unit)
        } catch (e: Exception) {
            logger.logError("Failed to clear session", e)
            Result.failure(e)
        }
    }

    override suspend fun hasSession(): Boolean {
        return try {
            val sessionJson = encryptedPrefs.getString(KEY_SESSION, null)
            !sessionJson.isNullOrEmpty()
        } catch (e: Exception) {
            false
        }
    }

    override fun observeSession(): StateFlow<MatrixSession?> {
        return _sessionFlow.asStateFlow()
    }

    /**
     * Get the last update timestamp
     */
    fun getLastUpdated(): Long {
        return encryptedPrefs.getLong(KEY_LAST_UPDATED, 0)
    }

    companion object {
        private const val PREFS_FILE_NAME = "matrix_session_storage"
        private const val KEY_SESSION = "matrix_session"
        private const val KEY_USER_ID = "user_id"
        private const val KEY_LAST_UPDATED = "last_updated"
    }
}

/**
 * Android factory implementation
 */
actual object MatrixSessionStorageFactory {
    private var instance: MatrixSessionStorage? = null

    fun initialize(context: Context) {
        instance = MatrixSessionStorageAndroid(context.applicationContext)
    }

    actual fun create(): MatrixSessionStorage {
        return instance ?: throw IllegalStateException(
            "MatrixSessionStorageFactory not initialized. Call initialize(context) first."
        )
    }
}
