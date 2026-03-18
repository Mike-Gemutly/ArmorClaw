package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

/**
 * PII Access Request
 *
 * Represents a request from an AI agent to access sensitive user data.
 * Used in the BlindFill flow where agents request permission to use personal
 * information for tasks like form filling or payment processing.
 *
 * ## Security Model
 * - LOW sensitivity: Email, name - approved by default
 * - MEDIUM sensitivity: Address, phone - requires explicit approval
 * - HIGH sensitivity: Credit card, DOB - requires explicit approval
 * - CRITICAL sensitivity: SSN, CVV, password - requires biometric approval
 *
 * ## Usage
 * ```kotlin
 * val request = PiiAccessRequest(
 *     requestId = "req_123",
 *     agentId = "agent_browse_001",
 *     fields = listOf(
 *         PiiField("Credit Card Number", SensitivityLevel.HIGH, "Required for payment", "••••4242")
 *     ),
 *     reason = "Complete checkout on amazon.com",
 *     expiresAt = System.currentTimeMillis() + 30000,
 *     batchSize = 1
 * )
 * ```
 */
@Serializable
data class PiiAccessRequest(
    val requestId: String,
    val agentId: String,
    val fields: List<PiiField>,
    val reason: String,
    val expiresAt: Long,
    val batchSize: Int = 1
) {
    /**
     * Check if this request has expired
     */
    fun isExpired(): Boolean {
        return System.currentTimeMillis() > expiresAt
    }

    /**
     * Check if this request contains any critical fields
     */
    fun hasCriticalFields(): Boolean {
        return fields.any { it.sensitivity == SensitivityLevel.CRITICAL }
    }

    /**
     * Check if this request contains any high sensitivity fields
     */
    fun hasHighSensitivityFields(): Boolean {
        return fields.any { it.sensitivity >= SensitivityLevel.HIGH }
    }

    /**
     * Get fields grouped by sensitivity level
     */
    fun getFieldsBySensitivity(): Map<SensitivityLevel, List<PiiField>> {
        return fields.groupBy { it.sensitivity }
    }
}

/**
 * Individual PII field being requested
 */
@Serializable
data class PiiField(
    val name: String,
    val sensitivity: SensitivityLevel,
    val description: String,
    val currentValue: String? = null
)

/**
 * Sensitivity level for PII data
 *
 * Ordered from lowest to highest sensitivity.
 */
@Serializable
enum class SensitivityLevel : Comparable<SensitivityLevel> {
    /** Low sensitivity: Email, name, general contact info */
    LOW,

    /** Medium sensitivity: Address, phone number */
    MEDIUM,

    /** High sensitivity: Credit card number, date of birth */
    HIGH,

    /** Critical sensitivity: SSN, CVV, passwords */
    CRITICAL;

    /**
     * Returns a user-friendly display string
     */
    fun toDisplayString(): String {
        return name.lowercase().replaceFirstChar { it.uppercase() }
    }

    /**
     * Returns true if this sensitivity level requires biometric authentication
     */
    fun requiresBiometric(): Boolean {
        return this == CRITICAL
    }
}
