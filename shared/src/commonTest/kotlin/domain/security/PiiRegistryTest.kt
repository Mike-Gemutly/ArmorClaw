package com.armorclaw.shared.domain.security

import com.armorclaw.shared.domain.repository.VaultKeyCategory
import com.armorclaw.shared.domain.repository.VaultKeySensitivity
import com.armorclaw.shared.domain.repository.VaultKey
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for PiiRegistry
 *
 * Tests predefined PII fields, OMO categories, extension functions,
 * registerKey(), requireKey(), isKeyRequired() functions, and OMO-specific fields.
 */
class PiiRegistryTest {

    // ========================================================================
    // Predefined PII Fields Tests
    // ========================================================================

    @Test
    fun `FULL_NAME predefined field has correct properties`() {
        val fullName = PiiRegistry.FULL_NAME

        assertEquals("pii_full_name", fullName.id, "FULL_NAME should have correct id")
        assertEquals("full_name", fullName.fieldName, "FULL_NAME should have correct field name")
        assertEquals("Full Name", fullName.displayName, "FULL_NAME should have correct display name")
        assertEquals(VaultKeyCategory.PERSONAL, fullName.category, "FULL_NAME should be PERSONAL category")
        assertEquals(VaultKeySensitivity.LOW, fullName.sensitivity, "FULL_NAME should be LOW sensitivity")
        assertEquals(0, fullName.accessCount, "FULL_NAME should have zero access count")
    }

    @Test
    fun `EMAIL predefined field has correct properties`() {
        val email = PiiRegistry.EMAIL

        assertEquals("pii_email", email.id, "EMAIL should have correct id")
        assertEquals("email", email.fieldName, "EMAIL should have correct field name")
        assertEquals("Email Address", email.displayName, "EMAIL should have correct display name")
        assertEquals(VaultKeyCategory.CONTACT, email.category, "EMAIL should be CONTACT category")
        assertEquals(VaultKeySensitivity.MEDIUM, email.sensitivity, "EMAIL should be MEDIUM sensitivity")
        assertEquals(0, email.accessCount, "EMAIL should have zero access count")
    }

    @Test
    fun `PHONE predefined field has correct properties`() {
        val phone = PiiRegistry.PHONE

        assertEquals("pii_phone", phone.id, "PHONE should have correct id")
        assertEquals("phone", phone.fieldName, "PHONE should have correct field name")
        assertEquals("Phone Number", phone.displayName, "PHONE should have correct display name")
        assertEquals(VaultKeyCategory.CONTACT, phone.category, "PHONE should be CONTACT category")
        assertEquals(VaultKeySensitivity.MEDIUM, phone.sensitivity, "PHONE should be MEDIUM sensitivity")
        assertEquals(0, phone.accessCount, "PHONE should have zero access count")
    }

    @Test
    fun `DATE_OF_BIRTH predefined field has correct properties`() {
        val dob = PiiRegistry.DATE_OF_BIRTH

        assertEquals("pii_dob", dob.id, "DATE_OF_BIRTH should have correct id")
        assertEquals("date_of_birth", dob.fieldName, "DATE_OF_BIRTH should have correct field name")
        assertEquals("Date of Birth", dob.displayName, "DATE_OF_BIRTH should have correct display name")
        assertEquals(VaultKeyCategory.PERSONAL, dob.category, "DATE_OF_BIRTH should be PERSONAL category")
        assertEquals(VaultKeySensitivity.HIGH, dob.sensitivity, "DATE_OF_BIRTH should be HIGH sensitivity")
        assertEquals(0, dob.accessCount, "DATE_OF_BIRTH should have zero access count")
    }

    @Test
    fun `SSN predefined field has correct properties`() {
        val ssn = PiiRegistry.SSN

        assertEquals("pii_ssn", ssn.id, "SSN should have correct id")
        assertEquals("ssn", ssn.fieldName, "SSN should have correct field name")
        assertEquals("Social Security Number", ssn.displayName, "SSN should have correct display name")
        assertEquals(VaultKeyCategory.PERSONAL, ssn.category, "SSN should be PERSONAL category")
        assertEquals(VaultKeySensitivity.CRITICAL, ssn.sensitivity, "SSN should be CRITICAL sensitivity")
        assertEquals(0, ssn.accessCount, "SSN should have zero access count")
    }

    @Test
    fun `ADDRESS predefined field has correct properties`() {
        val address = PiiRegistry.ADDRESS

        assertEquals("pii_address", address.id, "ADDRESS should have correct id")
        assertEquals("address", address.fieldName, "ADDRESS should have correct field name")
        assertEquals("Street Address", address.displayName, "ADDRESS should have correct display name")
        assertEquals(VaultKeyCategory.CONTACT, address.category, "ADDRESS should be CONTACT category")
        assertEquals(VaultKeySensitivity.HIGH, address.sensitivity, "ADDRESS should be HIGH sensitivity")
        assertEquals(0, address.accessCount, "ADDRESS should have zero access count")
    }

    @Test
    fun `CREDIT_CARD predefined field has correct properties`() {
        val creditCard = PiiRegistry.CREDIT_CARD

        assertEquals("pii_credit_card", creditCard.id, "CREDIT_CARD should have correct id")
        assertEquals("credit_card", creditCard.fieldName, "CREDIT_CARD should have correct field name")
        assertEquals("Credit Card Number", creditCard.displayName, "CREDIT_CARD should have correct display name")
        assertEquals(VaultKeyCategory.FINANCIAL, creditCard.category, "CREDIT_CARD should be FINANCIAL category")
        assertEquals(VaultKeySensitivity.CRITICAL, creditCard.sensitivity, "CREDIT_CARD should be CRITICAL sensitivity")
        assertEquals(0, creditCard.accessCount, "CREDIT_CARD should have zero access count")
    }

    @Test
    fun `BANK_ACCOUNT predefined field has correct properties`() {
        val bankAccount = PiiRegistry.BANK_ACCOUNT

        assertEquals("pii_bank_account", bankAccount.id, "BANK_ACCOUNT should have correct id")
        assertEquals("bank_account", bankAccount.fieldName, "BANK_ACCOUNT should have correct field name")
        assertEquals("Bank Account Number", bankAccount.displayName, "BANK_ACCOUNT should have correct display name")
        assertEquals(VaultKeyCategory.FINANCIAL, bankAccount.category, "BANK_ACCOUNT should be FINANCIAL category")
        assertEquals(VaultKeySensitivity.CRITICAL, bankAccount.sensitivity, "BANK_ACCOUNT should be CRITICAL sensitivity")
        assertEquals(0, bankAccount.accessCount, "BANK_ACCOUNT should have zero access count")
    }

    @Test
    fun `PASSWORD predefined field has correct properties`() {
        val password = PiiRegistry.PASSWORD

        assertEquals("pii_password", password.id, "PASSWORD should have correct id")
        assertEquals("password", password.fieldName, "PASSWORD should have correct field name")
        assertEquals("Password", password.displayName, "PASSWORD should have correct display name")
        assertEquals(VaultKeyCategory.AUTHENTICATION, password.category, "PASSWORD should be AUTHENTICATION category")
        assertEquals(VaultKeySensitivity.CRITICAL, password.sensitivity, "PASSWORD should be CRITICAL sensitivity")
        assertEquals(0, password.accessCount, "PASSWORD should have zero access count")
    }

    // ========================================================================
    // OMO Categories Tests
    // ========================================================================

    @Test
    fun `OMO_CREDENTIALS predefined field has correct properties`() {
        val omoCredentials = PiiRegistry.OMO_CREDENTIALS

        assertEquals("pii_omo_credentials", omoCredentials.id, "OMO_CREDENTIALS should have correct id")
        assertEquals("omo_credentials", omoCredentials.fieldName, "OMO_CREDENTIALS should have correct field name")
        assertEquals("OMO Credentials", omoCredentials.displayName, "OMO_CREDENTIALS should have correct display name")
        assertEquals(VaultKeyCategory.OMO_CREDENTIALS, omoCredentials.category, "OMO_CREDENTIALS should be OMO_CREDENTIALS category")
        assertEquals(VaultKeySensitivity.OMO_CRITICAL, omoCredentials.sensitivity, "OMO_CREDENTIALS should be OMO_CRITICAL sensitivity")
        assertEquals(0, omoCredentials.accessCount, "OMO_CREDENTIALS should have zero access count")
    }

    @Test
    fun `OMO_IDENTITY predefined field has correct properties`() {
        val omoIdentity = PiiRegistry.OMO_IDENTITY

        assertEquals("pii_omo_identity", omoIdentity.id, "OMO_IDENTITY should have correct id")
        assertEquals("omo_identity", omoIdentity.fieldName, "OMO_IDENTITY should have correct field name")
        assertEquals("OMO Identity", omoIdentity.displayName, "OMO_IDENTITY should have correct display name")
        assertEquals(VaultKeyCategory.OMO_IDENTITY, omoIdentity.category, "OMO_IDENTITY should be OMO_IDENTITY category")
        assertEquals(VaultKeySensitivity.OMO_LOW, omoIdentity.sensitivity, "OMO_IDENTITY should be OMO_LOW sensitivity")
        assertEquals(0, omoIdentity.accessCount, "OMO_IDENTITY should have zero access count")
    }

    @Test
    fun `OMO_SETTINGS predefined field has correct properties`() {
        val omoSettings = PiiRegistry.OMO_SETTINGS

        assertEquals("pii_omo_settings", omoSettings.id, "OMO_SETTINGS should have correct id")
        assertEquals("omo_settings", omoSettings.fieldName, "OMO_SETTINGS should have correct field name")
        assertEquals("OMO Settings", omoSettings.displayName, "OMO_SETTINGS should have correct display name")
        assertEquals(VaultKeyCategory.OMO_SETTINGS, omoSettings.category, "OMO_SETTINGS should be OMO_SETTINGS category")
        assertEquals(VaultKeySensitivity.OMO_LOW, omoSettings.sensitivity, "OMO_SETTINGS should be OMO_LOW sensitivity")
        assertEquals(0, omoSettings.accessCount, "OMO_SETTINGS should have zero access count")
    }

    @Test
    fun `OMO_TOKENS predefined field has correct properties`() {
        val omoTokens = PiiRegistry.OMO_TOKENS

        assertEquals("pii_omo_tokens", omoTokens.id, "OMO_TOKENS should have correct id")
        assertEquals("omo_tokens", omoTokens.fieldName, "OMO_TOKENS should have correct field name")
        assertEquals("OMO Tokens", omoTokens.displayName, "OMO_TOKENS should have correct display name")
        assertEquals(VaultKeyCategory.OMO_TOKENS, omoTokens.category, "OMO_TOKENS should be OMO_TOKENS category")
        assertEquals(VaultKeySensitivity.OMO_HIGH, omoTokens.sensitivity, "OMO_TOKENS should be OMO_HIGH sensitivity")
        assertEquals(0, omoTokens.accessCount, "OMO_TOKENS should have zero access count")
    }

    @Test
    fun `OMO_WORKSPACE predefined field has correct properties`() {
        val omoWorkspace = PiiRegistry.OMO_WORKSPACE

        assertEquals("pii_omo_workspace", omoWorkspace.id, "OMO_WORKSPACE should have correct id")
        assertEquals("omo_workspace", omoWorkspace.fieldName, "OMO_WORKSPACE should have correct field name")
        assertEquals("OMO Workspace", omoWorkspace.displayName, "OMO_WORKSPACE should have correct display name")
        assertEquals(VaultKeyCategory.OMO_WORKSPACE, omoWorkspace.category, "OMO_WORKSPACE should be OMO_WORKSPACE category")
        assertEquals(VaultKeySensitivity.OMO_MEDIUM, omoWorkspace.sensitivity, "OMO_WORKSPACE should be OMO_MEDIUM sensitivity")
        assertEquals(0, omoWorkspace.accessCount, "OMO_WORKSPACE should have zero access count")
    }

    @Test
    fun `OMO_TASKS predefined field has correct properties`() {
        val omoTasks = PiiRegistry.OMO_TASKS

        assertEquals("pii_omo_tasks", omoTasks.id, "OMO_TASKS should have correct id")
        assertEquals("omo_tasks", omoTasks.fieldName, "OMO_TASKS should have correct field name")
        assertEquals("OMO Tasks", omoTasks.displayName, "OMO_TASKS should have correct display name")
        assertEquals(VaultKeyCategory.OMO_TASKS, omoTasks.category, "OMO_TASKS should be OMO_TASKS category")
        assertEquals(VaultKeySensitivity.OMO_LOW, omoTasks.sensitivity, "OMO_TASKS should be OMO_LOW sensitivity")
        assertEquals(0, omoTasks.accessCount, "OMO_TASKS should have zero access count")
    }

    // ========================================================================
    // PREDEFINED_KEYS Tests
    // ========================================================================

    @Test
    fun `PREDEFINED_KEYS contains all standard PII fields`() {
        val predefinedKeys = PiiRegistry.PREDEFINED_KEYS

        assertTrue(predefinedKeys.any { it.fieldName == "full_name" }, "PREDEFINED_KEYS should contain FULL_NAME")
        assertTrue(predefinedKeys.any { it.fieldName == "email" }, "PREDEFINED_KEYS should contain EMAIL")
        assertTrue(predefinedKeys.any { it.fieldName == "phone" }, "PREDEFINED_KEYS should contain PHONE")
        assertTrue(predefinedKeys.any { it.fieldName == "date_of_birth" }, "PREDEFINED_KEYS should contain DATE_OF_BIRTH")
        assertTrue(predefinedKeys.any { it.fieldName == "ssn" }, "PREDEFINED_KEYS should contain SSN")
        assertTrue(predefinedKeys.any { it.fieldName == "address" }, "PREDEFINED_KEYS should contain ADDRESS")
    }

    @Test
    fun `PREDEFINED_KEYS contains all financial and auth fields`() {
        val predefinedKeys = PiiRegistry.PREDEFINED_KEYS

        assertTrue(predefinedKeys.any { it.fieldName == "credit_card" }, "PREDEFINED_KEYS should contain CREDIT_CARD")
        assertTrue(predefinedKeys.any { it.fieldName == "bank_account" }, "PREDEFINED_KEYS should contain BANK_ACCOUNT")
        assertTrue(predefinedKeys.any { it.fieldName == "password" }, "PREDEFINED_KEYS should contain PASSWORD")
    }

    @Test
    fun `PREDEFINED_KEYS contains all OMO fields`() {
        val predefinedKeys = PiiRegistry.PREDEFINED_KEYS

        assertTrue(predefinedKeys.any { it.fieldName == "omo_credentials" }, "PREDEFINED_KEYS should contain OMO_CREDENTIALS")
        assertTrue(predefinedKeys.any { it.fieldName == "omo_identity" }, "PREDEFINED_KEYS should contain OMO_IDENTITY")
        assertTrue(predefinedKeys.any { it.fieldName == "omo_settings" }, "PREDEFINED_KEYS should contain OMO_SETTINGS")
        assertTrue(predefinedKeys.any { it.fieldName == "omo_tokens" }, "PREDEFINED_KEYS should contain OMO_TOKENS")
        assertTrue(predefinedKeys.any { it.fieldName == "omo_workspace" }, "PREDEFINED_KEYS should contain OMO_WORKSPACE")
        assertTrue(predefinedKeys.any { it.fieldName == "omo_tasks" }, "PREDEFINED_KEYS should contain OMO_TASKS")
    }

    @Test
    fun `PREDEFINED_KEYS count is fifteen`() {
        val predefinedKeys = PiiRegistry.PREDEFINED_KEYS

        assertEquals(15, predefinedKeys.size, "PREDEFINED_KEYS should contain exactly 15 keys")
    }

    // ========================================================================
    // registerKey() Tests
    // ========================================================================

    @Test
    fun `registerKey adds new key to registeredKeys`() {
        val registry = PiiRegistry()
        val customKey = VaultKey(
            id = "custom_1",
            fieldName = "custom_field",
            displayName = "Custom Field",
            category = VaultKeyCategory.OTHER,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = null,
            accessCount = 0
        )

        registry.registerKey(customKey)
        val registeredKeys = registry.registeredKeys.value

        assertTrue(registeredKeys.contains(customKey), "Registered keys should contain the custom key")
        assertEquals(1, registeredKeys.size, "Registered keys should have size 1")
    }

    @Test
    fun `registerKey updates existing key with same fieldName`() {
        val registry = PiiRegistry()
        val key1 = VaultKey(
            id = "key_1",
            fieldName = "same_field",
            displayName = "First Version",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = null,
            accessCount = 0
        )
        val key2 = VaultKey(
            id = "key_2",
            fieldName = "same_field",
            displayName = "Updated Version",
            category = VaultKeyCategory.FINANCIAL,
            sensitivity = VaultKeySensitivity.HIGH,
            lastAccessed = null,
            accessCount = 0
        )

        registry.registerKey(key1)
        registry.registerKey(key2)
        val registeredKeys = registry.registeredKeys.value

        assertEquals(1, registeredKeys.size, "Registered keys should still have size 1 after update")
        assertEquals("Updated Version", registeredKeys[0].displayName, "Registered key should have updated display name")
        assertEquals(VaultKeyCategory.FINANCIAL, registeredKeys[0].category, "Registered key should have updated category")
        assertEquals(VaultKeySensitivity.HIGH, registeredKeys[0].sensitivity, "Registered key should have updated sensitivity")
    }

    @Test
    fun `registerKey appends new keys to the list`() {
        val registry = PiiRegistry()
        val key1 = VaultKey(
            id = "key_1",
            fieldName = "field_1",
            displayName = "Field 1",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = null,
            accessCount = 0
        )
        val key2 = VaultKey(
            id = "key_2",
            fieldName = "field_2",
            displayName = "Field 2",
            category = VaultKeyCategory.CONTACT,
            sensitivity = VaultKeySensitivity.MEDIUM,
            lastAccessed = null,
            accessCount = 0
        )

        registry.registerKey(key1)
        registry.registerKey(key2)
        val registeredKeys = registry.registeredKeys.value

        assertEquals(2, registeredKeys.size, "Registered keys should have size 2")
        assertTrue(registeredKeys.contains(key1), "Registered keys should contain key1")
        assertTrue(registeredKeys.contains(key2), "Registered keys should contain key2")
    }

    // ========================================================================
    // requireKey() Tests
    // ========================================================================

    @Test
    fun `requireKey adds fieldName to requiredKeys`() {
        val registry = PiiRegistry()

        registry.requireKey("full_name")
        val requiredKeys = registry.requiredKeys.value

        assertTrue(requiredKeys.contains("full_name"), "Required keys should contain 'full_name'")
        assertEquals(1, requiredKeys.size, "Required keys should have size 1")
    }

    @Test
    fun `requireKey does not duplicate fieldName`() {
        val registry = PiiRegistry()

        registry.requireKey("email")
        registry.requireKey("email")
        val requiredKeys = registry.requiredKeys.value

        assertEquals(1, requiredKeys.size, "Required keys should have size 1 (no duplicates)")
        assertTrue(requiredKeys.contains("email"), "Required keys should contain 'email'")
    }

    @Test
    fun `requireKey adds multiple fieldNames to requiredKeys`() {
        val registry = PiiRegistry()

        registry.requireKey("full_name")
        registry.requireKey("email")
        registry.requireKey("phone")
        val requiredKeys = registry.requiredKeys.value

        assertEquals(3, requiredKeys.size, "Required keys should have size 3")
        assertTrue(requiredKeys.contains("full_name"), "Required keys should contain 'full_name'")
        assertTrue(requiredKeys.contains("email"), "Required keys should contain 'email'")
        assertTrue(requiredKeys.contains("phone"), "Required keys should contain 'phone'")
    }

    // ========================================================================
    // isKeyRequired() Tests
    // ========================================================================

    @Test
    fun `isKeyRequired returns true for required key`() {
        val registry = PiiRegistry()

        registry.requireKey("ssn")
        val isRequired = registry.isKeyRequired("ssn")

        assertTrue(isRequired, "isKeyRequired should return true for required key")
    }

    @Test
    fun `isKeyRequired returns false for non-required key`() {
        val registry = PiiRegistry()

        val isRequired = registry.isKeyRequired("address")

        assertFalse(isRequired, "isKeyRequired should return false for non-required key")
    }

    @Test
    fun `isKeyRequired returns false after unrequireKey`() {
        val registry = PiiRegistry()

        registry.requireKey("date_of_birth")
        registry.unrequireKey("date_of_birth")
        val isRequired = registry.isKeyRequired("date_of_birth")

        assertFalse(isRequired, "isKeyRequired should return false after unrequireKey")
    }

    @Test
    fun `isKeyRequired handles multiple required keys correctly`() {
        val registry = PiiRegistry()

        registry.requireKey("full_name")
        registry.requireKey("email")

        assertTrue(registry.isKeyRequired("full_name"), "full_name should be required")
        assertTrue(registry.isKeyRequired("email"), "email should be required")
        assertFalse(registry.isKeyRequired("phone"), "phone should not be required")
    }

    // ========================================================================
    // getKey() Tests
    // ========================================================================

    @Test
    fun `getKey returns registered key from registeredKeys`() {
        val registry = PiiRegistry()
        val customKey = VaultKey(
            id = "custom_1",
            fieldName = "custom_field",
            displayName = "Custom Field",
            category = VaultKeyCategory.OTHER,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = null,
            accessCount = 0
        )

        registry.registerKey(customKey)
        val retrievedKey = registry.getKey("custom_field")

        assertEquals(customKey, retrievedKey, "getKey should return the registered key")
    }

    @Test
    fun `getKey returns predefined key from PREDEFINED_KEYS`() {
        val registry = PiiRegistry()

        val retrievedKey = registry.getKey("full_name")

        assertEquals(PiiRegistry.FULL_NAME, retrievedKey, "getKey should return predefined FULL_NAME")
    }

    @Test
    fun `getKey returns null for non-existent key`() {
        val registry = PiiRegistry()

        val retrievedKey = registry.getKey("nonexistent_field")

        assertEquals(null, retrievedKey, "getKey should return null for non-existent key")
    }

    @Test
    fun `getKey prefers registered key over predefined key`() {
        val registry = PiiRegistry()
        val customKey = VaultKey(
            id = "custom_1",
            fieldName = "full_name",
            displayName = "Custom Full Name",
            category = VaultKeyCategory.FINANCIAL,
            sensitivity = VaultKeySensitivity.HIGH,
            lastAccessed = null,
            accessCount = 0
        )

        registry.registerKey(customKey)
        val retrievedKey = registry.getKey("full_name")

        assertEquals(customKey, retrievedKey, "getKey should prefer registered key over predefined key")
    }

    // ========================================================================
    // getKeysByCategory() Tests
    // ========================================================================

    @Test
    fun `getKeysByCategory returns all keys for PERSONAL category`() {
        val registry = PiiRegistry()

        val personalKeys = registry.getKeysByCategory(VaultKeyCategory.PERSONAL)

        assertTrue(personalKeys.contains(PiiRegistry.FULL_NAME), "PERSONAL keys should contain FULL_NAME")
        assertTrue(personalKeys.contains(PiiRegistry.DATE_OF_BIRTH), "PERSONAL keys should contain DATE_OF_BIRTH")
        assertTrue(personalKeys.contains(PiiRegistry.SSN), "PERSONAL keys should contain SSN")
    }

    @Test
    fun `getKeysByCategory returns all keys for CONTACT category`() {
        val registry = PiiRegistry()

        val contactKeys = registry.getKeysByCategory(VaultKeyCategory.CONTACT)

        assertTrue(contactKeys.contains(PiiRegistry.EMAIL), "CONTACT keys should contain EMAIL")
        assertTrue(contactKeys.contains(PiiRegistry.PHONE), "CONTACT keys should contain PHONE")
        assertTrue(contactKeys.contains(PiiRegistry.ADDRESS), "CONTACT keys should contain ADDRESS")
    }

    @Test
    fun `getKeysByCategory returns all keys for FINANCIAL category`() {
        val registry = PiiRegistry()

        val financialKeys = registry.getKeysByCategory(VaultKeyCategory.FINANCIAL)

        assertTrue(financialKeys.contains(PiiRegistry.CREDIT_CARD), "FINANCIAL keys should contain CREDIT_CARD")
        assertTrue(financialKeys.contains(PiiRegistry.BANK_ACCOUNT), "FINANCIAL keys should contain BANK_ACCOUNT")
    }

    @Test
    fun `getKeysByCategory returns all keys for AUTHENTICATION category`() {
        val registry = PiiRegistry()

        val authKeys = registry.getKeysByCategory(VaultKeyCategory.AUTHENTICATION)

        assertTrue(authKeys.contains(PiiRegistry.PASSWORD), "AUTHENTICATION keys should contain PASSWORD")
    }

    @Test
    fun `getKeysByCategory returns all keys for OMO_CREDENTIALS category`() {
        val registry = PiiRegistry()

        val omoCredentialKeys = registry.getKeysByCategory(VaultKeyCategory.OMO_CREDENTIALS)

        assertTrue(omoCredentialKeys.contains(PiiRegistry.OMO_CREDENTIALS), "OMO_CREDENTIALS keys should contain OMO_CREDENTIALS")
    }

    @Test
    fun `getKeysByCategory returns all keys for OMO_IDENTITY category`() {
        val registry = PiiRegistry()

        val omoIdentityKeys = registry.getKeysByCategory(VaultKeyCategory.OMO_IDENTITY)

        assertTrue(omoIdentityKeys.contains(PiiRegistry.OMO_IDENTITY), "OMO_IDENTITY keys should contain OMO_IDENTITY")
    }

    @Test
    fun `getKeysByCategory returns all keys for OMO_SETTINGS category`() {
        val registry = PiiRegistry()

        val omoSettingsKeys = registry.getKeysByCategory(VaultKeyCategory.OMO_SETTINGS)

        assertTrue(omoSettingsKeys.contains(PiiRegistry.OMO_SETTINGS), "OMO_SETTINGS keys should contain OMO_SETTINGS")
    }

    @Test
    fun `getKeysByCategory returns all keys for OMO_TOKENS category`() {
        val registry = PiiRegistry()

        val omoTokensKeys = registry.getKeysByCategory(VaultKeyCategory.OMO_TOKENS)

        assertTrue(omoTokensKeys.contains(PiiRegistry.OMO_TOKENS), "OMO_TOKENS keys should contain OMO_TOKENS")
    }

    @Test
    fun `getKeysByCategory returns all keys for OMO_WORKSPACE category`() {
        val registry = PiiRegistry()

        val omoWorkspaceKeys = registry.getKeysByCategory(VaultKeyCategory.OMO_WORKSPACE)

        assertTrue(omoWorkspaceKeys.contains(PiiRegistry.OMO_WORKSPACE), "OMO_WORKSPACE keys should contain OMO_WORKSPACE")
    }

    @Test
    fun `getKeysByCategory returns all keys for OMO_TASKS category`() {
        val registry = PiiRegistry()

        val omoTasksKeys = registry.getKeysByCategory(VaultKeyCategory.OMO_TASKS)

        assertTrue(omoTasksKeys.contains(PiiRegistry.OMO_TASKS), "OMO_TASKS keys should contain OMO_TASKS")
    }

    @Test
    fun `getKeysByCategory includes registered custom keys`() {
        val registry = PiiRegistry()
        val customKey = VaultKey(
            id = "custom_1",
            fieldName = "custom_field",
            displayName = "Custom Field",
            category = VaultKeyCategory.OTHER,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = null,
            accessCount = 0
        )

        registry.registerKey(customKey)
        val otherKeys = registry.getKeysByCategory(VaultKeyCategory.OTHER)

        assertTrue(otherKeys.contains(customKey), "OTHER keys should contain custom registered key")
    }

    // ========================================================================
    // clear() Tests
    // ========================================================================

    @Test
    fun `clear clears all registeredKeys`() {
        val registry = PiiRegistry()
        val key1 = VaultKey(
            id = "key_1",
            fieldName = "field_1",
            displayName = "Field 1",
            category = VaultKeyCategory.PERSONAL,
            sensitivity = VaultKeySensitivity.LOW,
            lastAccessed = null,
            accessCount = 0
        )

        registry.registerKey(key1)
        registry.clear()
        val registeredKeys = registry.registeredKeys.value

        assertEquals(0, registeredKeys.size, "Registered keys should be empty after clear")
    }

    @Test
    fun `clear clears all requiredKeys`() {
        val registry = PiiRegistry()

        registry.requireKey("full_name")
        registry.requireKey("email")
        registry.clear()
        val requiredKeys = registry.requiredKeys.value

        assertEquals(0, requiredKeys.size, "Required keys should be empty after clear")
    }

    // ========================================================================
    // StateFlow Tests
    // ========================================================================

    @Test
    fun `registeredKeys is initially empty`() {
        val registry = PiiRegistry()

        val registeredKeys = registry.registeredKeys.value

        assertTrue(registeredKeys.isEmpty(), "Registered keys should be empty initially")
    }

    @Test
    fun `requiredKeys is initially empty`() {
        val registry = PiiRegistry()

        val requiredKeys = registry.requiredKeys.value

        assertTrue(requiredKeys.isEmpty(), "Required keys should be empty initially")
    }
}
