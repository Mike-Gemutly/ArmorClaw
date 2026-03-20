package com.armorclaw.shared.secretary

import kotlinx.serialization.Serializable

/**
 * SecretaryMode defines the operating mode for the Secretary system.
 * Each mode determines how proactive cards are filtered and displayed.
 */
@Serializable
enum class SecretaryMode {
    MEETING,
    FOCUS,
    SLEEP,
    NORMAL
}
