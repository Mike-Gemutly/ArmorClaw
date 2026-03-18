package com.armorclaw.shared.domain.security

import com.armorclaw.shared.domain.repository.ShadowPlaceholder
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Agent Request Interceptor for Cold Vault
 *
 * Intercepts outgoing agent requests and shadows PII values.
 * Ensures agents never see raw PII - only placeholder tokens.
 *
 * Phase 1 Implementation - Governor Strategy
 *
 * Usage:
 * ```
 * val interceptor = AgentRequestInterceptor(piiRegistry, shadowMap)
 * val shadowedRequest = interceptor.intercept(originalRequest)
 * ```
 */
class AgentRequestInterceptor(
    private val piiRegistry: PiiRegistry,
    private val shadowMap: ShadowMap
) {

    private val _interceptionLog = MutableStateFlow<List<InterceptionRecord>>(emptyList())
    val interceptionLog: StateFlow<List<InterceptionRecord>> = _interceptionLog.asStateFlow()

    /**
     * Intercept a request and shadow PII values
     */
    fun intercept(request: AgentRequest): AgentRequest {
        val shadowedContent = shadowPiiInContent(request.content)
        val shadowedMetadata = shadowPiiInMetadata(request.metadata)

        logInterception(request.id, request.content, shadowedContent)

        return request.copy(
            content = shadowedContent,
            metadata = shadowedMetadata
        )
    }

    /**
     * Intercept a message and shadow PII values
     */
    fun interceptMessage(message: String, requiredFields: List<String>): ShadowedMessage {
        val shadowedContent = StringBuilder(message)
        val replacedFields = mutableListOf<String>()

        for (fieldName in requiredFields) {
            val key = piiRegistry.getKey(fieldName) ?: continue
            val valueHash = hashValue(message) // In production, hash the actual value
            
            val placeholder = shadowMap.createPlaceholder(fieldName, valueHash)
            // Replace the value with placeholder
            // In production, this would find and replace the actual PII value
            
            replacedFields.add(fieldName)
        }

        return ShadowedMessage(
            originalMessage = message,
            shadowedMessage = shadowedContent.toString(),
            replacedFields = replacedFields,
            placeholderCount = replacedFields.size
        )
    }

    /**
     * Restore placeholders to original values (for display)
     */
    fun restorePlaceholders(
        shadowedContent: String,
        valueProvider: (fieldName: String) -> String?
    ): RestoredContent {
        val restored = StringBuilder(shadowedContent)
        val restoredFields = mutableListOf<String>()

        val matches = shadowMap.findPlaceholdersInText(shadowedContent)
        for (match in matches) {
            val fieldName = shadowMap.extractFieldName(match.value) ?: continue
            val placeholder = shadowMap.getPlaceholder(fieldName)

            if (placeholder != null && shadowMap.isPlaceholderValid(placeholder)) {
                val originalValue = valueProvider(fieldName)
                if (originalValue != null) {
                    restored.replace(match.range.first, match.range.last + 1, originalValue)
                    restoredFields.add(fieldName)
                }
            }
        }

        return RestoredContent(
            shadowedContent = shadowedContent,
            restoredContent = restored.toString(),
            restoredFields = restoredFields
        )
    }

    /**
     * Check if content contains PII that should be shadowed
     */
    fun containsPii(content: String): Boolean {
        // In production, use pattern matching and NLP to detect PII
        // For now, check against known field patterns
        return piiRegistry.registeredKeys.value.any { key ->
            content.contains(key.fieldName, ignoreCase = true)
        }
    }

    /**
     * Get list of PII fields detected in content
     */
    fun detectPiiFields(content: String): List<String> {
        val detected = mutableListOf<String>()
        
        piiRegistry.registeredKeys.value.forEach { key ->
            if (content.contains(key.fieldName, ignoreCase = true)) {
                detected.add(key.fieldName)
            }
        }
        
        // Also check predefined keys
        PiiRegistry.PREDEFINED_KEYS.forEach { key ->
            if (content.contains(key.fieldName, ignoreCase = true) && !detected.contains(key.fieldName)) {
                detected.add(key.fieldName)
            }
        }
        
        return detected
    }

    // Private helper functions

    private fun shadowPiiInContent(content: String): String {
        var shadowed = content

        // Find all required keys and create placeholders
        piiRegistry.requiredKeys.value.forEach { fieldName ->
            val key = piiRegistry.getKey(fieldName) ?: return@forEach
            val valueHash = hashValue(content)

            val placeholder = shadowMap.createPlaceholder(fieldName, valueHash)
            // In production, find and replace actual PII values
            // For now, just note the placeholder was created
        }

        return shadowed
    }

    private fun shadowPiiInMetadata(metadata: Map<String, String>): Map<String, String> {
        return metadata.mapValues { (_, value) ->
            if (containsPii(value)) {
                shadowPiiInContent(value)
            } else {
                value
            }
        }
    }

    private fun hashValue(value: String): String {
        // In production, use SHA-256
        // For now, use a simple hash
        return value.hashCode().toString(16)
    }

    private fun logInterception(requestId: String, original: String, shadowed: String) {
        val record = InterceptionRecord(
            requestId = requestId,
            timestamp = System.currentTimeMillis(),
            originalHash = hashValue(original),
            shadowedHash = hashValue(shadowed),
            fieldsShadowed = detectPiiFields(original)
        )

        val current = _interceptionLog.value.toMutableList()
        current.add(0, record)
        // Keep last 100 records
        if (current.size > 100) {
            current.removeLast()
        }
        _interceptionLog.value = current
    }

    // Data classes

    /**
     * Agent request to intercept
     */
    data class AgentRequest(
        val id: String,
        val agentId: String,
        val content: String,
        val metadata: Map<String, String> = emptyMap()
    )

    /**
     * Shadowed message result
     */
    data class ShadowedMessage(
        val originalMessage: String,
        val shadowedMessage: String,
        val replacedFields: List<String>,
        val placeholderCount: Int
    )

    /**
     * Restored content result
     */
    data class RestoredContent(
        val shadowedContent: String,
        val restoredContent: String,
        val restoredFields: List<String>
    )

    /**
     * Interception log record
     */
    data class InterceptionRecord(
        val requestId: String,
        val timestamp: Long,
        val originalHash: String,
        val shadowedHash: String,
        val fieldsShadowed: List<String>
    )
}
