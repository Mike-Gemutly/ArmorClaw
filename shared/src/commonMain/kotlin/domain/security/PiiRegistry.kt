package com.armorclaw.shared.domain.security

import com.armorclaw.shared.domain.repository.VaultKey
import com.armorclaw.shared.domain.repository.VaultKeyCategory
import com.armorclaw.shared.domain.repository.VaultKeySensitivity
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * PII Registry for Cold Vault
 *
 * Manages PII keys and placeholders for shadow mapping.
 * Tracks what PII is stored and required by agents.
 *
 * Phase 1 Implementation - Governor Strategy
 */
class PiiRegistry {

    private val _registeredKeys = MutableStateFlow<List<VaultKey>>(emptyList())
    val registeredKeys: StateFlow<List<VaultKey>> = _registeredKeys.asStateFlow()

    private val _requiredKeys = MutableStateFlow<Set<String>>(emptySet())
    val requiredKeys: StateFlow<Set<String>> = _requiredKeys.asStateFlow()

    // Predefined PII fields
    companion object {
        // Personal
        val FULL_NAME = VaultKey(
            id = "pii_full_name",
            fieldName = "full_name",
            displayName = "Full Name",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = null,
            accessCount = 0
        )

        val EMAIL = VaultKey(
            id = "pii_email",
            fieldName = "email",
            displayName = "Email Address",
            category = VaultKeyCategory.CONTACT,
            sensitivity = VaultKeySensitivity.MEDIUM,
            lastAccessed = null,
            accessCount = 0
        )

        val PHONE = VaultKey(
            id = "pii_phone",
            fieldName = "phone",
            displayName = "Phone Number",
            category = VaultKeyCategory.CONTACT,
            sensitivity = VaultKeySensitivity.MEDIUM,
            lastAccessed = null,
            accessCount = 0
        )

        val DATE_OF_BIRTH = VaultKey(
            id = "pii_dob",
            fieldName = "date_of_birth",
            displayName = "Date of Birth",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.HIGH,
            lastAccessed = null,
            accessCount = 0
        )

        val SSN = VaultKey(
            id = "pii_ssn",
            fieldName = "ssn",
            displayName = "Social Security Number",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.CRITICAL,
            lastAccessed = null,
            accessCount = 0
        )

        val ADDRESS = VaultKey(
            id = "pii_address",
            fieldName = "address",
            displayName = "Street Address",
            category = VaultKeyCategory.CONTACT,
            sensitivity = VaultKeySensitivity.HIGH,
            lastAccessed = null,
            accessCount = 0
        )

        // Financial
        val CREDIT_CARD = VaultKey(
            id = "pii_credit_card",
            fieldName = "credit_card",
            displayName = "Credit Card Number",
            category = VaultKeyCategory.FINANCIAL,
            sensitivity = VaultKeySensitivity.CRITICAL,
            lastAccessed = null,
            accessCount = 0
        )

        val BANK_ACCOUNT = VaultKey(
            id = "pii_bank_account",
            fieldName = "bank_account",
            displayName = "Bank Account Number",
            category = VaultKeyCategory.FINANCIAL,
            sensitivity = VaultKeySensitivity.CRITICAL,
            lastAccessed = null,
            accessCount = 0
        )

        // Authentication
        val PASSWORD = VaultKey(
            id = "pii_password",
            fieldName = "password",
            displayName = "Password",
            category = VaultKeyCategory.AUTHENTICATION,
            sensitivity = VaultKeySensitivity.CRITICAL,
            lastAccessed = null,
            accessCount = 0
        )

        // OMO Credentials (API keys, tokens, passwords for OMO services)
        val OMO_CREDENTIALS = VaultKey(
            id = "pii_omo_credentials",
            fieldName = "omo_credentials",
            displayName = "OMO Credentials",
            category = VaultKeyCategory.OMO_CREDENTIALS,
            sensitivity = VaultKeySensitivity.OMO_CRITICAL,
            lastAccessed = null,
            accessCount = 0
        )

        // OMO Identity (User identity information for OMO agents)
        val OMO_IDENTITY = VaultKey(
            id = "pii_omo_identity",
            fieldName = "omo_identity",
            displayName = "OMO Identity",
            category = VaultKeyCategory.OMO_IDENTITY,
            sensitivity = VaultKeySensitivity.OMO_LOW,
            lastAccessed = null,
            accessCount = 0
        )

        // OMO Settings (Agent configuration and settings)
        val OMO_SETTINGS = VaultKey(
            id = "pii_omo_settings",
            fieldName = "omo_settings",
            displayName = "OMO Settings",
            category = VaultKeyCategory.OMO_SETTINGS,
            sensitivity = VaultKeySensitivity.OMO_LOW,
            lastAccessed = null,
            accessCount = 0
        )

        // OMO Tokens (Session tokens for OMO authentication)
        val OMO_TOKENS = VaultKey(
            id = "pii_omo_tokens",
            fieldName = "omo_tokens",
            displayName = "OMO Tokens",
            category = VaultKeyCategory.OMO_TOKENS,
            sensitivity = VaultKeySensitivity.OMO_HIGH,
            lastAccessed = null,
            accessCount = 0
        )

        // OMO Workspace (Workspace and project data)
        val OMO_WORKSPACE = VaultKey(
            id = "pii_omo_workspace",
            fieldName = "omo_workspace",
            displayName = "OMO Workspace",
            category = VaultKeyCategory.OMO_WORKSPACE,
            sensitivity = VaultKeySensitivity.OMO_MEDIUM,
            lastAccessed = null,
            accessCount = 0
        )

        // OMO Tasks (Task data and metadata)
        val OMO_TASKS = VaultKey(
            id = "pii_omo_tasks",
            fieldName = "omo_tasks",
            displayName = "OMO Tasks",
            category = VaultKeyCategory.OMO_TASKS,
            sensitivity = VaultKeySensitivity.OMO_LOW,
            lastAccessed = null,
            accessCount = 0
        )

        /**
         * Get all predefined keys
         */
        val PREDEFINED_KEYS = listOf(
            FULL_NAME, EMAIL, PHONE, DATE_OF_BIRTH, SSN,
            ADDRESS, CREDIT_CARD, BANK_ACCOUNT, PASSWORD,
            OMO_CREDENTIALS, OMO_IDENTITY, OMO_SETTINGS,
            OMO_TOKENS, OMO_WORKSPACE, OMO_TASKS
        )
    }

    /**
     * Register a custom PII key
     */
    fun registerKey(key: VaultKey) {
        val current = _registeredKeys.value.toMutableList()
        val existingIndex = current.indexOfFirst { it.fieldName == key.fieldName }
        
        if (existingIndex >= 0) {
            current[existingIndex] = key
        } else {
            current.add(key)
        }
        
        _registeredKeys.value = current
    }

    /**
     * Mark a key as required by an agent
     */
    fun requireKey(fieldName: String) {
        val current = _requiredKeys.value.toMutableSet()
        current.add(fieldName)
        _requiredKeys.value = current
    }

    /**
     * Unmark a key as required
     */
    fun unrequireKey(fieldName: String) {
        val current = _requiredKeys.value.toMutableSet()
        current.remove(fieldName)
        _requiredKeys.value = current
    }

    /**
     * Check if a key is required
     */
    fun isKeyRequired(fieldName: String): Boolean {
        return _requiredKeys.value.contains(fieldName)
    }

    /**
     * Get a key by field name
     */
    fun getKey(fieldName: String): VaultKey? {
        return _registeredKeys.value.find { it.fieldName == fieldName }
            ?: PREDEFINED_KEYS.find { it.fieldName == fieldName }
    }

    /**
     * Get all keys by category
     */
    fun getKeysByCategory(category: VaultKeyCategory): List<VaultKey> {
        return _registeredKeys.value.filter { it.category == category } +
                PREDEFINED_KEYS.filter { it.category == category }
    }

    /**
     * Clear all registered keys (for logout)
     */
    fun clear() {
        _registeredKeys.value = emptyList()
        _requiredKeys.value = emptySet()
    }
}
