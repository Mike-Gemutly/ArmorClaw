package com.armorclaw.shared.platform.clipboard

import kotlinx.coroutines.flow.StateFlow
import kotlinx.datetime.Instant
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds

expect class SecureClipboard() {
    fun copySensitive(
        data: String,
        autoClearAfter: Duration = 30.seconds,
        showToast: Boolean = true
    )

    fun copy(data: String)
    fun getClipboardContent(): String?
    fun containsSensitiveData(): Boolean
    fun clear()

    val autoClearState: StateFlow<AutoClearState?>
}

sealed class AutoClearState {
    data class Active(
        val expiresAt: Instant,
        val originalData: String
    ) : AutoClearState()

    object Cleared : AutoClearState()
    object UserOverridden : AutoClearState()
}

data class ClipboardData(
    val content: String,
    val isSensitive: Boolean,
    val copiedAt: Instant,
    val expiresAt: Instant? = null
)
