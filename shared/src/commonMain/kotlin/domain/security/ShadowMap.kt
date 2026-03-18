package com.armorclaw.shared.domain.security

import com.armorclaw.shared.domain.repository.ShadowPlaceholder
import com.armorclaw.shared.domain.repository.VaultKey
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant

/**
 * Shadow Map for Cold Vault
 *
 * Maps real PII values to placeholder tokens for safe agent transmission.
 * Placeholders are formatted as {{VAULT:field_name:hash}}
 *
 * Phase 1 Implementation - Governor Strategy
 */
class ShadowMap(
    private val piiRegistry: PiiRegistry
) {

    private val _placeholders = MutableStateFlow<Map<String, ShadowPlaceholder>>(emptyMap())
    val placeholders: StateFlow<Map<String, ShadowPlaceholder>> = _placeholders.asStateFlow()

    companion object {
        private const val PLACEHOLDER_PREFIX = "{{VAULT:"
        private const val PLACEHOLDER_SUFFIX = "}}"
        private const val PLACEHOLDER_EXPIRY_MS = 60_000L * 5 // 5 minutes

        // Regex to find placeholders in text
        val PLACEHOLDER_REGEX = Regex("""\{\{VAULT:([^:]+):([^}]+)\}\}""")
    }

    /**
     * Create a shadow placeholder for a PII field
     */
    fun createPlaceholder(fieldName: String, valueHash: String): ShadowPlaceholder {
        val nowMs = Clock.System.now().toEpochMilliseconds()
        val placeholder = ShadowPlaceholder(
            keyId = generateKeyId(fieldName),
            placeholder = formatPlaceholder(fieldName, valueHash),
            hash = valueHash,
            createdAt = nowMs,
            expiresAt = nowMs + PLACEHOLDER_EXPIRY_MS
        )

        // Store the mapping
        val current = _placeholders.value.toMutableMap()
        current[fieldName] = placeholder
        _placeholders.value = current

        return placeholder
    }

    /**
     * Get a placeholder by field name
     */
    fun getPlaceholder(fieldName: String): ShadowPlaceholder? {
        return _placeholders.value[fieldName]
    }

    /**
     * Check if a placeholder is valid (not expired)
     */
    fun isPlaceholderValid(placeholder: ShadowPlaceholder): Boolean {
        val now = Clock.System.now().toEpochMilliseconds()
        return now < placeholder.expiresAt
    }

    /**
     * Validate a placeholder hash
     */
    fun validateHash(placeholder: ShadowPlaceholder, expectedHash: String): Boolean {
        return placeholder.hash == expectedHash && isPlaceholderValid(placeholder)
    }

    /**
     * Remove expired placeholders
     */
    fun cleanExpired() {
        val now = Clock.System.now().toEpochMilliseconds()
        val current = _placeholders.value.toMutableMap()
        current.entries.removeAll { now >= it.value.expiresAt }
        _placeholders.value = current
    }

    /**
     * Clear all placeholders (for logout)
     */
    fun clear() {
        _placeholders.value = emptyMap()
    }

    /**
     * Find all placeholders in text
     */
    fun findPlaceholdersInText(text: String): List<MatchResult> {
        return PLACEHOLDER_REGEX.findAll(text).toList()
    }

    /**
     * Check if text contains any placeholders
     */
    fun containsPlaceholders(text: String): Boolean {
        return PLACEHOLDER_REGEX.containsMatchIn(text)
    }

    /**
     * Get the field name from a placeholder
     */
    fun extractFieldName(placeholder: String): String? {
        val match = PLACEHOLDER_REGEX.find(placeholder)
        return match?.groupValues?.get(1)
    }

    /**
     * Get the hash from a placeholder
     */
    fun extractHash(placeholder: String): String? {
        val match = PLACEHOLDER_REGEX.find(placeholder)
        return match?.groupValues?.get(2)
    }

    // Helper functions

    private fun generateKeyId(fieldName: String): String {
        return "shadow_${fieldName}_${Clock.System.now().toEpochMilliseconds()}"
    }

    private fun formatPlaceholder(fieldName: String, hash: String): String {
        return "$PLACEHOLDER_PREFIX$fieldName:$hash$PLACEHOLDER_SUFFIX"
    }

    /**
     * Data class for tracking shadow mappings
     */
    data class ShadowMapping(
        val fieldName: String,
        val originalValueHash: String,
        val placeholder: String,
        val createdAt: Instant,
        val expiresAt: Instant,
        val accessedBy: List<String> = emptyList()
    )
}
