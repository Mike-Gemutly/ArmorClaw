package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

/**
 * Keystore Status
 *
 * Represents the sealed/unsealed state of the VPS keystore.
 * The keystore contains encrypted credentials that must be unsealed
 * before agents can access them.
 *
 * ## Security Model
 * - Sealed: All credentials are encrypted and inaccessible
 * - Unsealed: Credentials are decrypted and available for use (time-limited)
 * - Error: Keystore is in an error state
 *
 * ## Session Duration
 * Default unsealed session duration is 4 hours.
 */
@Serializable
sealed class KeystoreStatus {
    abstract val lastUpdated: Long

    /**
     * Keystore is sealed - credentials are encrypted
     */
    @Serializable
    data class Sealed(
        override val lastUpdated: Long = System.currentTimeMillis()
    ) : KeystoreStatus()

    /**
     * Keystore is unsealed - credentials are accessible
     * @param expiresAt Timestamp when the session will expire
     * @param unsealedBy Method used to unseal (password/biometric)
     */
    @Serializable
    data class Unsealed(
        val expiresAt: Long,
        val unsealedBy: UnsealMethod,
        override val lastUpdated: Long = System.currentTimeMillis()
    ) : KeystoreStatus() {
        /**
         * Check if the unsealed session has expired
         */
        fun isExpired(): Boolean {
            return System.currentTimeMillis() > expiresAt
        }

        /**
         * Get remaining time in milliseconds
         */
        fun remainingTimeMs(): Long {
            return (expiresAt - System.currentTimeMillis()).coerceAtLeast(0)
        }

        /**
         * Get remaining time as human-readable string
         */
        fun remainingTimeString(): String {
            val remainingMs = remainingTimeMs()
            val hours = remainingMs / (1000 * 60 * 60)
            val minutes = (remainingMs % (1000 * 60 * 60)) / (1000 * 60)
            return if (hours > 0) {
                "${hours}h ${minutes}m remaining"
            } else {
                "${minutes}m remaining"
            }
        }
    }

    /**
     * Keystore is in an error state
     */
    @Serializable
    data class Error(
        val message: String,
        override val lastUpdated: Long = System.currentTimeMillis()
    ) : KeystoreStatus()

    /**
     * Check if the keystore is currently accessible
     */
    fun isAccessible(): Boolean {
        return this is Unsealed && !isExpired()
    }
}

/**
 * Method used to unseal the keystore
 */
@Serializable
enum class UnsealMethod {
    PASSWORD,
    BIOMETRIC;

    fun toDisplayString(): String {
        return name.lowercase().replaceFirstChar { it.uppercase() }
    }
}

/**
 * Default session duration for unsealed keystores (4 hours)
 */
const val KEYSTORE_SESSION_DURATION_MS: Long = 4 * 60 * 60 * 1000
