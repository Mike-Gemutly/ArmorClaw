package com.armorclaw.app.viewmodels

import android.content.Intent
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.Loggable
import com.armorclaw.shared.platform.logging.LoggerDelegate
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * ViewModel for Splash screen
 *
 * Handles initialization, state checking, and deep link routing.
 */
class SplashViewModel(
    private val preferences: AppPreferences
) : ViewModel(), Loggable by LoggerDelegate(LogTag.ViewModel.Splash) {

    private val _navigationTarget = MutableStateFlow<SplashTarget?>(null)
    val navigationTarget: StateFlow<SplashTarget?> = _navigationTarget.asStateFlow()

    private val _isLoading = MutableStateFlow(true)
    val isLoading: StateFlow<Boolean> = _isLoading.asStateFlow()

    init {
        logInfo("SplashViewModel initialized")
        checkInitialState()
    }

    /**
     * Check app state and determine where to navigate
     *
     * Flow:
     * - If has valid session → HOME
     * - If has completed onboarding but not logged in → LOGIN
     * - Otherwise → CONNECT (QR-first onboarding)
     */
    private fun checkInitialState() {
        viewModelScope.launch {
            logDebug("Checking initial app state")

            // Minimum splash display time for branding
            delay(MIN_SPLASH_DURATION)

            // Check states in order of priority
            val hasCompletedOnboarding = preferences.hasCompletedOnboarding()
            val isLoggedIn = preferences.isLoggedIn()
            val hasValidSession = preferences.hasValidSession()

            logDebug(
                "App state checked",
                mapOf(
                    "hasCompletedOnboarding" to hasCompletedOnboarding,
                    "isLoggedIn" to isLoggedIn,
                    "hasValidSession" to hasValidSession
                )
            )

            // Check for legacy Bridge session (v2.5) without Matrix session (v3.0)
            val hasLegacyBridgeSession = preferences.hasLegacyBridgeSession()
            val hasMatrixSession = hasValidSession

            logDebug(
                "Migration check",
                mapOf(
                    "hasLegacyBridgeSession" to hasLegacyBridgeSession,
                    "hasMatrixSession" to hasMatrixSession
                )
            )

            // Check key backup completion
            val isBackupComplete = preferences.isBackupComplete()

            logDebug(
                "Backup state",
                mapOf("isBackupComplete" to isBackupComplete)
            )

            val target = when {
                // Legacy v2.5 user upgrading to v3.0 → MIGRATION
                hasLegacyBridgeSession && !hasMatrixSession -> {
                    logInfo("Navigating to migration (v2.5 → v3.0 upgrade detected)")
                    SplashTarget.Migration
                }
                // Has valid session but backup not completed → KEY_BACKUP_SETUP
                // Prevents force-quit bypass of key backup during onboarding
                hasValidSession && isLoggedIn && !isBackupComplete -> {
                    logInfo("Navigating to key backup setup (backup incomplete)")
                    SplashTarget.KeyBackupSetup
                }
                // Has valid Matrix session → go directly to HOME
                hasValidSession && isLoggedIn -> {
                    logInfo("Navigating to home (valid session)")
                    SplashTarget.Home
                }
                // Completed onboarding but session expired → LOGIN
                hasCompletedOnboarding && !isLoggedIn -> {
                    logInfo("Navigating to login (session expired)")
                    SplashTarget.Login
                }
                // First time user → CONNECT (QR-first onboarding)
                // Skip welcome/security screens - go straight to QR scan
                else -> {
                    logInfo("Navigating to connect (QR-first onboarding)")
                    SplashTarget.Connect
                }
            }

            _navigationTarget.value = target
            _isLoading.value = false
        }
    }

    /**
     * Process incoming intent for deep links
     */
    fun processDeepLink(intent: Intent) {
        val data = intent.data ?: return

        logInfo(
            "Processing deep link",
            mapOf("uri" to (data.toString().take(100)))
        )

        viewModelScope.launch {
            // Parse matrix.to links
            val deepLinkTarget = when {
                data.scheme == "https" && data.host == "matrix.to" -> {
                    parseMatrixToLink(data)
                }
                data.scheme == "armorclaw" -> {
                    parseAppDeepLink(data)
                }
                else -> null
            }

            if (deepLinkTarget != null) {
                // Wait for initial navigation to complete
                delay(100)
                _navigationTarget.value = deepLinkTarget
            }
        }
    }

    /**
     * Parse matrix.to deep link
     *
     * Format: https://matrix.to/#/!roomId:server.com
     * or: https://matrix.to/#/@user:server.com
     */
    private fun parseMatrixToLink(uri: android.net.Uri): SplashTarget? {
        val fragment = uri.fragment ?: return null

        return when {
            fragment.startsWith("!") -> {
                // Room ID
                val roomId = fragment.substringBefore("?")
                logInfo("Deep link to room", mapOf("roomId" to roomId))
                SplashTarget.DeepLink.Room(roomId)
            }
            fragment.startsWith("@") -> {
                // User ID
                val userId = fragment.substringBefore("?")
                logInfo("Deep link to user", mapOf("userId" to userId))
                SplashTarget.DeepLink.User(userId)
            }
            else -> null
        }
    }

    /**
     * Parse app-specific deep link
     *
     * Format:
     * - armorclaw://room/{roomId}
     * - armorclaw://user/{userId}
     * - armorclaw://call/{callId}
     * - armorclaw://config?d=<base64-encoded-json> (QR provisioning)
     * - armorclaw://invite?code=<invite-code>
     */
    private fun parseAppDeepLink(uri: android.net.Uri): SplashTarget? {
        val pathSegments = uri.pathSegments

        return when (uri.host) {
            "room" -> {
                val roomId = pathSegments.firstOrNull()
                if (roomId != null) {
                    logInfo("App deep link to room", mapOf("roomId" to roomId))
                    SplashTarget.DeepLink.Room(roomId)
                } else null
            }
            "user" -> {
                val userId = pathSegments.firstOrNull()
                if (userId != null) {
                    logInfo("App deep link to user", mapOf("userId" to userId))
                    SplashTarget.DeepLink.User(userId)
                } else null
            }
            "call" -> {
                val callId = pathSegments.firstOrNull()
                if (callId != null) {
                    logInfo("App deep link to call", mapOf("callId" to callId))
                    SplashTarget.DeepLink.Call(callId)
                } else null
            }
            "config" -> {
                // QR provisioning deep link
                val configData = uri.getQueryParameter("d")
                if (configData != null) {
                    logInfo("App deep link to config (QR provisioning)")
                    // Reconstruct full URI for SetupService parsing
                    SplashTarget.DeepLink.Config(uri.toString())
                } else null
            }
            "invite" -> {
                // Invite code deep link
                val code = uri.getQueryParameter("code")
                if (code != null) {
                    logInfo("App deep link to invite")
                    SplashTarget.DeepLink.Config(uri.toString())
                } else null
            }
            else -> null
        }
    }

    /**
     * Clear navigation target after handling
     */
    fun clearNavigationTarget() {
        _navigationTarget.value = null
    }

    companion object {
        private const val MIN_SPLASH_DURATION = 1500L
    }
}

/**
 * Navigation targets from splash screen
 */
sealed class SplashTarget {
    object Onboarding : SplashTarget()
    object Connect : SplashTarget()  // QR-first onboarding
    object Login : SplashTarget()
    object Home : SplashTarget()
    object Migration : SplashTarget()  // v2.5 → v3.0 upgrade
    object KeyBackupSetup : SplashTarget()  // Force-quit bypass: backup incomplete

    sealed class DeepLink : SplashTarget() {
        data class Room(val roomId: String) : DeepLink()
        data class User(val userId: String) : DeepLink()
        data class Call(val callId: String) : DeepLink()
        data class Config(val configData: String) : DeepLink()  // QR config deep link
    }
}
