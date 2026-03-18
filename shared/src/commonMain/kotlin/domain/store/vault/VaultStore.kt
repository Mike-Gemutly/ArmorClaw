package com.armorclaw.shared.domain.store.vault

import com.armorclaw.shared.domain.repository.VaultKey
import com.armorclaw.shared.domain.repository.VaultKeyCategory
import com.armorclaw.shared.domain.repository.VaultKeySensitivity
import com.armorclaw.shared.domain.security.PiiRegistry
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update

/**
 * Vault Store
 *
 * State management for the Cold Vault.
 * Tracks required keys, active sessions, and vault status.
 *
 * Phase 1 Implementation - Governor Strategy
 */
class VaultStore(
    private val piiRegistry: PiiRegistry
) {

    private val _state = MutableStateFlow(VaultState())
    val state: StateFlow<VaultState> = _state.asStateFlow()

    // Computed properties
    val isActive: Boolean get() = _state.value.requiredKeys.isNotEmpty()
    val requiredKeyCount: Int get() = _state.value.requiredKeys.size
    val storedKeyCount: Int get() = _state.value.storedKeys.size

    /**
     * Initialize the vault with stored keys
     */
    fun initialize(keys: List<VaultKey>) {
        _state.update { it.copy(storedKeys = keys, isInitialized = true) }
    }

    /**
     * Mark a key as required by an agent
     */
    fun requireKey(fieldName: String) {
        piiRegistry.requireKey(fieldName)
        _state.update { state ->
            state.copy(requiredKeys = state.requiredKeys + fieldName)
        }
    }

    /**
     * Unmark a key as required
     */
    fun unrequireKey(fieldName: String) {
        piiRegistry.unrequireKey(fieldName)
        _state.update { state ->
            state.copy(requiredKeys = state.requiredKeys - fieldName)
        }
    }

    /**
     * Mark multiple keys as required
     */
    fun requireKeys(fieldNames: Collection<String>) {
        fieldNames.forEach { piiRegistry.requireKey(it) }
        _state.update { state ->
            state.copy(requiredKeys = state.requiredKeys + fieldNames)
        }
    }

    /**
     * Clear all required keys (when agent session ends)
     */
    fun clearRequiredKeys() {
        _state.value.requiredKeys.forEach { piiRegistry.unrequireKey(it) }
        _state.update { state ->
            state.copy(requiredKeys = emptySet())
        }
    }

    /**
     * Add a stored key
     */
    fun addStoredKey(key: VaultKey) {
        piiRegistry.registerKey(key)
        _state.update { state ->
            state.copy(storedKeys = state.storedKeys + key)
        }
    }

    /**
     * Remove a stored key
     */
    fun removeStoredKey(fieldName: String) {
        _state.update { state ->
            state.copy(storedKeys = state.storedKeys.filter { it.fieldName != fieldName })
        }
    }

    /**
     * Update key access info
     */
    fun updateKeyAccess(fieldName: String, accessedAt: Long = System.currentTimeMillis()) {
        _state.update { state ->
            state.copy(
                storedKeys = state.storedKeys.map { key ->
                    if (key.fieldName == fieldName) {
                        key.copy(
                            lastAccessed = accessedAt,
                            accessCount = key.accessCount + 1
                        )
                    } else key
                }
            )
        }
    }

    /**
     * Set vault status
     */
    fun setStatus(status: VaultStatus) {
        _state.update { it.copy(status = status) }
    }

    /**
     * Set error state
     */
    fun setError(error: String?) {
        _state.update { it.copy(error = error, status = if (error != null) VaultStatus.ERROR else VaultStatus.SECURED) }
    }

    /**
     * Clear all state (for logout)
     */
    fun clear() {
        piiRegistry.clear()
        _state.value = VaultState()
    }

    /**
     * Get keys by category
     */
    fun getKeysByCategory(category: VaultKeyCategory): List<VaultKey> {
        return _state.value.storedKeys.filter { it.category == category }
    }

    /**
     * Get required keys that are not yet stored
     */
    fun getMissingRequiredKeys(): Set<String> {
        val storedFieldNames = _state.value.storedKeys.map { it.fieldName }.toSet()
        return _state.value.requiredKeys - storedFieldNames
    }

    /**
     * Check if all required keys are stored
     */
    fun hasAllRequiredKeys(): Boolean {
        return getMissingRequiredKeys().isEmpty()
    }

    companion object {
        /**
         * Create a VaultStore with predefined keys from PiiRegistry
         */
        fun create(): VaultStore {
            val registry = PiiRegistry()
            // Register predefined keys
            PiiRegistry.PREDEFINED_KEYS.forEach { registry.registerKey(it) }
            return VaultStore(registry)
        }
    }
}

/**
 * Vault State
 */
data class VaultState(
    val isInitialized: Boolean = false,
    val status: VaultStatus = VaultStatus.LOCKED,
    val storedKeys: List<VaultKey> = emptyList(),
    val requiredKeys: Set<String> = emptySet(),
    val activeSessionId: String? = null,
    val error: String? = null
)

/**
 * Vault Status
 */
enum class VaultStatus {
    LOCKED,     // Vault is locked, needs authentication
    SECURED,    // Vault is initialized and secured
    ACTIVE,     // Vault is actively being accessed
    ERROR       // Vault encountered an error
}
