package com.armorclaw.shared.platform.clipboard

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

actual class SecureClipboard {

    private var context: Context? = null
    private var scope: CoroutineScope = CoroutineScope(Dispatchers.Main)
    private var clipboard: ClipboardManager? = null
    private var prefs: android.content.SharedPreferences? = null

    private val _autoClearState = MutableStateFlow<AutoClearState?>(null)
    actual val autoClearState = _autoClearState.asStateFlow()

    private var autoClearJob: Job? = null

    companion object {
        private const val SECURE_CLIPBOARD_PREFS = "secure_clipboard"
        private const val HASH_KEY = "clipboard_hash"

        @Volatile
        private var instance: SecureClipboard? = null

        fun getInstance(): SecureClipboard {
            return instance ?: synchronized(this) {
                instance ?: SecureClipboard().also { instance = it }
            }
        }

        fun setContext(context: Context) {
            getInstance().context = context.applicationContext
            getInstance().initialize()
        }
    }

    private fun initialize() {
        val ctx = context ?: return
        clipboard = ctx.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        prefs = ctx.getSharedPreferences(SECURE_CLIPBOARD_PREFS, Context.MODE_PRIVATE)
    }

    actual fun copySensitive(
        data: String,
        autoClearAfter: kotlin.time.Duration,
        showToast: Boolean
    ) {
        val clip = clipboard ?: return
        val p = prefs ?: return

        // Show warning if configured
        if (showToast) {
            showWarningToast(autoClearAfter)
        }

        // Store hash for later verification
        val dataHash = hashData(data)
        p.edit().putLong(HASH_KEY, dataHash).apply()

        // Copy to clipboard
        val clipData = ClipData.newPlainText("ArmorClaw", data)
        clip.setPrimaryClip(clipData)

        // Start auto-clear timer
        startAutoClearTimer(data, autoClearAfter)
    }

    actual fun copy(data: String) {
        val clip = clipboard ?: return
        val clipData = ClipData.newPlainText("ArmorClaw", data)
        clip.setPrimaryClip(clipData)
    }

    actual fun getClipboardContent(): String? {
        val clip = clipboard ?: return null
        val primaryClip = clip.primaryClip
        return if (primaryClip != null && primaryClip.itemCount > 0) {
            primaryClip.getItemAt(0).text?.toString()
        } else {
            null
        }
    }

    actual fun containsSensitiveData(): Boolean {
        val p = prefs ?: return false
        val currentHash = getClipboardContent()?.let { hashData(it) }
        val storedHash = p.getLong(HASH_KEY, -1)

        return currentHash == storedHash && storedHash != -1L
    }

    actual fun clear() {
        val clip = clipboard ?: return
        val p = prefs ?: return
        clip.setPrimaryClip(ClipData.newPlainText("", ""))
        p.edit().remove(HASH_KEY).apply()
        cancelAutoClearTimer()
    }

    private fun startAutoClearTimer(originalData: String, duration: kotlin.time.Duration) {
        // Cancel any existing timer
        cancelAutoClearTimer()

        val expiryTime = kotlinx.datetime.Clock.System.now() + duration

        autoClearJob = scope.launch {
            _autoClearState.value = AutoClearState.Active(
                expiresAt = expiryTime,
                originalData = originalData
            )

            delay(duration)

            // Check if clipboard still contains our data
            val currentContent = getClipboardContent()
            val currentHash = currentContent?.let { hashData(it) }
            val p = prefs
            val storedHash = p?.getLong(HASH_KEY, -1) ?: -1

            if (currentHash == storedHash) {
                // Clear the clipboard
                clipboard?.setPrimaryClip(ClipData.newPlainText("", ""))
                prefs?.edit()?.remove(HASH_KEY)?.apply()
                _autoClearState.value = AutoClearState.Cleared
            } else {
                // User has copied something else, don't clear
                _autoClearState.value = AutoClearState.UserOverridden
            }

            delay(2000)
            _autoClearState.value = null
        }
    }

    private fun cancelAutoClearTimer() {
        autoClearJob?.cancel()
        autoClearJob = null
        _autoClearState.value = null
    }

    private fun showWarningToast(duration: kotlin.time.Duration) {
        // Toast implementation would go here
        // For now, we'll use a simple println
        val message = "Sensitive data copied. Will be cleared after ${duration.inWholeSeconds} seconds."
        println(message)
    }

    private fun hashData(data: String): Long {
        // Simple hash for comparison (not for security)
        return data.hashCode().toLong()
    }
}
