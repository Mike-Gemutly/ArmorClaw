package com.armorclaw.app.viewmodels

import android.content.Context
import android.content.SharedPreferences
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * App preferences for storing user state
 *
 * Uses SharedPreferences for simple key-value storage.
 * For more complex data, consider using DataStore.
 */
class AppPreferences(private val appContext: Context) {

    private val prefs: SharedPreferences = appContext.getSharedPreferences(
        PREFS_NAME,
        Context.MODE_PRIVATE
    )

    private val _onboardingCompleted = MutableStateFlow(false)
    val onboardingCompleted: StateFlow<Boolean> = _onboardingCompleted.asStateFlow()

    private val _isLoggedIn = MutableStateFlow(false)
    val isLoggedIn: StateFlow<Boolean> = _isLoggedIn.asStateFlow()

    private val _isBackupComplete = MutableStateFlow(false)
    val backupComplete: StateFlow<Boolean> = _isBackupComplete.asStateFlow()

    init {
        // Initialize state from prefs
        _onboardingCompleted.value = prefs.getBoolean(KEY_ONBOARDING_COMPLETED, false)
        _isLoggedIn.value = prefs.getBoolean(KEY_IS_LOGGED_IN, false)
        _isBackupComplete.value = prefs.getBoolean(KEY_BACKUP_COMPLETE, false)
    }

    /**
     * Check if user has completed onboarding
     */
    fun hasCompletedOnboarding(): Boolean = _onboardingCompleted.value

    /**
     * Check if user is logged in
     */
    fun isLoggedIn(): Boolean = _isLoggedIn.value

    /**
     * Check if key backup has been completed
     */
    fun isBackupComplete(): Boolean = _isBackupComplete.value

    /**
     * Mark key backup as completed (or not)
     */
    fun setBackupComplete(complete: Boolean = true) {
        prefs.edit().putBoolean(KEY_BACKUP_COMPLETE, complete).apply()
        _isBackupComplete.value = complete
    }

    /**
     * Check if user has a valid session
     */
    fun hasValidSession(): Boolean {
        // Check if we have an access token
        val accessToken = prefs.getString(KEY_ACCESS_TOKEN, null)
        val tokenExpiry = prefs.getLong(KEY_TOKEN_EXPIRY, 0)
        val now = System.currentTimeMillis()

        return !accessToken.isNullOrEmpty() && tokenExpiry > now
    }

    /**
     * Mark onboarding as completed
     */
    fun setOnboardingCompleted(completed: Boolean = true) {
        prefs.edit().putBoolean(KEY_ONBOARDING_COMPLETED, completed).apply()
        _onboardingCompleted.value = completed
    }

    /**
     * Set logged in state
     */
    fun setLoggedIn(loggedIn: Boolean) {
        prefs.edit().putBoolean(KEY_IS_LOGGED_IN, loggedIn).apply()
        _isLoggedIn.value = loggedIn
    }

    /**
     * Save session tokens
     */
    fun saveSession(accessToken: String, refreshToken: String?, expiresIn: Long) {
        val expiryTime = System.currentTimeMillis() + (expiresIn * 1000)
        prefs.edit()
            .putString(KEY_ACCESS_TOKEN, accessToken)
            .putString(KEY_REFRESH_TOKEN, refreshToken)
            .putLong(KEY_TOKEN_EXPIRY, expiryTime)
            .putBoolean(KEY_IS_LOGGED_IN, true)
            .apply()
        _isLoggedIn.value = true
    }

    /**
     * Clear session (logout)
     */
    fun clearSession() {
        prefs.edit()
            .remove(KEY_ACCESS_TOKEN)
            .remove(KEY_REFRESH_TOKEN)
            .remove(KEY_TOKEN_EXPIRY)
            .putBoolean(KEY_IS_LOGGED_IN, false)
            .apply()
        _isLoggedIn.value = false
    }

    /**
     * Clear all app data (full logout)
     */
    fun clearAll() {
        prefs.edit().clear().apply()
        _onboardingCompleted.value = false
        _isLoggedIn.value = false
        _isBackupComplete.value = false
    }

    /**
     * Get access token
     */
    fun getAccessToken(): String? = prefs.getString(KEY_ACCESS_TOKEN, null)

    /**
     * Get refresh token
     */
    fun getRefreshToken(): String? = prefs.getString(KEY_REFRESH_TOKEN, null)

    /**
     * Get current user ID
     */
    fun getCurrentUserId(): String? = prefs.getString(KEY_USER_ID, null)

    /**
     * Set current user ID
     */
    fun setUserId(userId: String) {
        prefs.edit().putString(KEY_USER_ID, userId).apply()
    }

    /**
     * Get current user display name
     */
    fun getUserDisplayName(): String? = prefs.getString(KEY_USER_DISPLAY_NAME, null)

    /**
     * Set user display name
     */
    fun setUserDisplayName(name: String) {
        prefs.edit().putString(KEY_USER_DISPLAY_NAME, name).apply()
    }

    /**
     * Get homeserver URL
     */
    fun getHomeserver(): String? = prefs.getString(KEY_HOMESERVER, null)

    /**
     * Set homeserver URL
     */
    fun setHomeserver(url: String) {
        prefs.edit().putString(KEY_HOMESERVER, url).apply()
    }

    /**
     * Check for legacy Bridge session (v2.5 Thin Client)
     *
     * Detects if the user has a v2.5 Bridge session stored in the old
     * SharedPreferences format. This indicates they need the migration
     * flow to transition to the v3.0 Thick Client (Matrix SDK).
     */
    fun hasLegacyBridgeSession(): Boolean {
        // Check the legacy Bridge session prefs (used by v2.5 BridgeRpcClient)
        val legacyPrefs = appContext.getSharedPreferences(
            "bridge_session", Context.MODE_PRIVATE
        )
        val hasLegacyToken = legacyPrefs.getString("bridge_access_token", null) != null
        val hasLegacyUserId = legacyPrefs.getString("bridge_user_id", null) != null
        return hasLegacyToken && hasLegacyUserId
    }

    /**
     * Clear legacy Bridge session after successful migration
     */
    fun clearLegacyBridgeSession() {
        val legacyPrefs = appContext.getSharedPreferences(
            "bridge_session", Context.MODE_PRIVATE
        )
        legacyPrefs.edit().clear().apply()
    }

    /**
     * Wipe legacy SQLCipher database and related files from v2.5 Bridge Proxy.
     *
     * The old Thin Client stored encrypted message history in a SQLCipher database
     * (`bridge_store.db`). After migration (or skip), these files are stale and
     * incompatible with the v3.0 Matrix SDK crypto store. Deleting them avoids
     * disk bloat and prevents any accidental reads from the old schema.
     */
    fun clearLegacyDatabase() {
        val dbNames = listOf("bridge_store.db", "bridge_store.db-journal", "bridge_store.db-wal", "bridge_store.db-shm")
        dbNames.forEach { name ->
            val dbFile = appContext.getDatabasePath(name)
            if (dbFile.exists()) {
                dbFile.delete()
            }
        }
    }

    /**
     * Full legacy data wipe — call on migration success, skip, or fresh-start logout.
     *
     * Clears Bridge SharedPreferences, SQLCipher DB files, and the legacy
     * onboarding prefs so the v3.0 session starts with a clean slate.
     */
    fun wipeLegacyData() {
        clearLegacyBridgeSession()
        clearLegacyDatabase()
        // Also clear legacy onboarding prefs (separate SharedPreferences file)
        val legacyOnboarding = appContext.getSharedPreferences(
            "onboarding_prefs", Context.MODE_PRIVATE
        )
        legacyOnboarding.edit().clear().apply()
    }

    // ==================== GOVERNANCE BANNER PERSISTENCE ====================

    /**
     * Get the timestamp when a governance event was dismissed.
     * Returns 0L if never dismissed.
     */
    fun getGovernanceDismissedAt(eventId: String): Long {
        return prefs.getLong("${KEY_GOVERNANCE_DISMISSED_PREFIX}$eventId", 0L)
    }

    /**
     * Record dismissal of a governance event (stores current timestamp).
     */
    fun setGovernanceDismissed(eventId: String) {
        prefs.edit()
            .putLong("${KEY_GOVERNANCE_DISMISSED_PREFIX}$eventId", System.currentTimeMillis())
            .apply()
    }

    /**
     * Check if a governance event is currently dismissed.
     * CRITICAL events re-surface after 24 hours.
     */
    fun isGovernanceDismissed(eventId: String, severity: String): Boolean {
        val dismissedAt = getGovernanceDismissedAt(eventId)
        if (dismissedAt == 0L) return false

        // CRITICAL warnings re-show after 24 hours
        if (severity == "CRITICAL") {
            val elapsed = System.currentTimeMillis() - dismissedAt
            return elapsed < GOVERNANCE_CRITICAL_RESHOW_MS
        }
        return true
    }

    companion object {
        const val PREFS_NAME = "armorclaw_prefs"

        // Keys
        const val KEY_ONBOARDING_COMPLETED = "onboarding_completed"
        const val KEY_IS_LOGGED_IN = "is_logged_in"
        const val KEY_BACKUP_COMPLETE = "backup_complete"
        const val KEY_ACCESS_TOKEN = "access_token"
        const val KEY_REFRESH_TOKEN = "refresh_token"
        const val KEY_TOKEN_EXPIRY = "token_expiry"
        const val KEY_USER_ID = "user_id"
        const val KEY_USER_DISPLAY_NAME = "user_display_name"
        const val KEY_HOMESERVER = "homeserver"
        private const val KEY_GOVERNANCE_DISMISSED_PREFIX = "governance_dismissed_"

        /** CRITICAL governance warnings re-surface after 24 hours */
        private const val GOVERNANCE_CRITICAL_RESHOW_MS = 24 * 60 * 60 * 1000L
    }
}
