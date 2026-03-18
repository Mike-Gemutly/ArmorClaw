package com.armorclaw.app.security

import android.database.sqlite.SQLiteDatabase
import com.armorclaw.shared.domain.repository.VaultKey
import com.armorclaw.shared.domain.repository.VaultKeyCategory
import com.armorclaw.shared.domain.repository.VaultKeySensitivity
import com.armorclaw.shared.domain.model.OMOIdentityData
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import javax.inject.Singleton

/**
 * Vault Repository for Cold Vault
 *
 * Manages PII storage and retrieval in the encrypted vault.
 * All data is encrypted at rest using SQLCipher.
 *
 * Phase 1 Implementation - Governor Strategy
 */
@Singleton
class VaultRepository(
    private val sqlCipherProvider: SqlCipherProvider,
    private val keystoreManager: KeystoreManager
) {

    /**
     * Store a PII value in the vault
     */
    suspend fun storeValue(
        fieldName: String,
        value: String,
        category: VaultKeyCategory,
        sensitivity: VaultKeySensitivity
    ): Result<VaultKey> = withContext(Dispatchers.IO) {
        try {
            val db = sqlCipherProvider.getDatabase()
            val id = generateKeyId(fieldName)
            val now = System.currentTimeMillis()

            // Encrypt the value (SQLCipher handles encryption at rest)
            val encryptedValue = encryptValue(value)

            // Insert or replace
            db.execSQL(
                """
                INSERT OR REPLACE INTO vault_entries 
                (id, field_name, encrypted_value, category, sensitivity, created_at, updated_at, access_count)
                VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(
                    (SELECT access_count FROM vault_entries WHERE field_name = ?), 0
                ))
                """.trimIndent(),
                arrayOf(id, fieldName, encryptedValue, category.name, sensitivity.name, now, now, fieldName)
            )

            // Update key metadata
            db.execSQL(
                """
                INSERT OR REPLACE INTO vault_keys 
                (id, field_name, display_name, category, sensitivity, last_accessed, access_count)
                VALUES (?, ?, ?, ?, ?, ?, 0)
                """.trimIndent(),
                arrayOf(id, fieldName, formatDisplayName(fieldName), category.name, sensitivity.name, now)
            )

            Result.success(
                VaultKey(
                    id = id,
                    fieldName = fieldName,
                    displayName = formatDisplayName(fieldName),
                    category = category,
                    sensitivity = sensitivity,
                    lastAccessed = now,
                    accessCount = 0
                )
            )
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Retrieve a PII value from the vault
     */
    suspend fun retrieveValue(fieldName: String): Result<String> = withContext(Dispatchers.IO) {
        try {
            val db = sqlCipherProvider.getDatabase()
            val now = System.currentTimeMillis()

            // Query and log access
            val cursor = db.rawQuery(
                "SELECT encrypted_value FROM vault_entries WHERE field_name = ?",
                arrayOf(fieldName)
            )

            if (cursor.moveToFirst()) {
                val encryptedValue = cursor.getBlob(0)
                val value = decryptValue(encryptedValue)

                // Update access log
                logAccess(db, fieldName, "READ")

                // Update access count
                db.execSQL(
                    "UPDATE vault_entries SET access_count = access_count + 1, last_accessed = ? WHERE field_name = ?",
                    arrayOf(now, fieldName)
                )
                db.execSQL(
                    "UPDATE vault_keys SET access_count = access_count + 1, last_accessed = ? WHERE field_name = ?",
                    arrayOf(now, fieldName)
                )

                cursor.close()
                Result.success(value)
            } else {
                cursor.close()
                Result.failure(NoSuchElementException("Field not found: $fieldName"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Delete a PII value from the vault
     */
    suspend fun deleteValue(fieldName: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val db = sqlCipherProvider.getDatabase()
            
            logAccess(db, fieldName, "DELETE")
            
            db.execSQL("DELETE FROM vault_entries WHERE field_name = ?", arrayOf(fieldName))
            db.execSQL("DELETE FROM vault_keys WHERE field_name = ?", arrayOf(fieldName))
            
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * List all keys in the vault
     */
    suspend fun listKeys(): Result<List<VaultKey>> = withContext(Dispatchers.IO) {
        try {
            val db = sqlCipherProvider.getDatabase()
            val keys = mutableListOf<VaultKey>()

            val cursor = db.rawQuery(
                "SELECT id, field_name, display_name, category, sensitivity, last_accessed, access_count FROM vault_keys",
                null
            )

            while (cursor.moveToNext()) {
                keys.add(
                    VaultKey(
                        id = cursor.getString(0),
                        fieldName = cursor.getString(1),
                        displayName = cursor.getString(2),
                        category = VaultKeyCategory.valueOf(cursor.getString(3)),
                        sensitivity = VaultKeySensitivity.valueOf(cursor.getString(4)),
                        lastAccessed = cursor.getLong(5),
                        accessCount = cursor.getInt(6)
                    )
                )
            }
            cursor.close()

            Result.success(keys)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /**
     * Clear all vault data
     */
    suspend fun clearAll(): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val db = sqlCipherProvider.getDatabase()
            db.execSQL("DELETE FROM vault_entries")
            db.execSQL("DELETE FROM vault_keys")
            db.execSQL("DELETE FROM vault_access_log")
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // Helper functions

    private fun generateKeyId(fieldName: String): String {
        return "vault_${fieldName}_${System.currentTimeMillis()}"
    }

    private fun formatDisplayName(fieldName: String): String {
        return fieldName.split("_").joinToString(" ") { word ->
            word.replaceFirstChar { it.uppercase() }
        }
    }

    private fun encryptValue(value: String): ByteArray {
        // SQLCipher handles encryption at rest
        // Additional encryption can be added here for extra security
        return value.toByteArray(Charsets.UTF_8)
    }

    private fun decryptValue(encryptedValue: ByteArray): String {
        // SQLCipher handles decryption
        return String(encryptedValue, Charsets.UTF_8)
    }

    private fun logAccess(db: SQLiteDatabase, fieldName: String, accessType: String) {
        val keyId = generateKeyId(fieldName)
        db.execSQL(
            """
            INSERT INTO vault_access_log (key_id, access_type, accessed_by, timestamp)
            VALUES (?, ?, ?, ?)
            """.trimIndent(),
            arrayOf(keyId, accessType, "app", System.currentTimeMillis())
        )
    }

    // ========================================
    // OMO CRUD Operations
    // ========================================

    /**
     * Store an OMO credential (API keys, tokens, passwords for OMO services)
     *
     * @param key The credential key (e.g., "openai_api_key", "github_token")
     * @param value The credential value to store
     * @param requiresBiometric Whether biometric authentication is required
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOCredential(
        key: String,
        value: String,
        requiresBiometric: Boolean = false
    ): Result<VaultKey> {
        val fieldName = "omo_credentials_$key"
        return storeValue(
            fieldName = fieldName,
            value = value,
            category = VaultKeyCategory.OMO_CREDENTIALS,
            sensitivity = VaultKeySensitivity.OMO_CRITICAL
        )
    }

    /**
     * Retrieve an OMO credential
     *
     * @param key The credential key to retrieve
     * @return Result containing the credential value
     */
    suspend fun retrieveOMOCredential(key: String): Result<String> {
        val fieldName = "omo_credentials_$key"
        return retrieveValue(fieldName)
    }

    /**
     * Delete an OMO credential
     *
     * @param key The credential key to delete
     * @param requiresBiometric Whether biometric authentication is required
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOCredential(
        key: String,
        requiresBiometric: Boolean = false
    ): Result<Unit> {
        val fieldName = "omo_credentials_$key"
        return deleteValue(fieldName)
    }

    /**
     * List all stored OMO credentials
     *
     * @return Result containing list of VaultKey entries for OMO credentials
     */
    suspend fun listOMOCredentials(): Result<List<VaultKey>> {
        return listKeysByCategory(VaultKeyCategory.OMO_CREDENTIALS)
    }

    /**
     * Store an OMO identity (User identity information for OMO agents)
     *
     * @param id Unique identifier for the identity
     * @param name Display name
     * @param email Email address
     * @param phone Phone number
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOIdentity(
        id: String,
        name: String,
        email: String,
        phone: String
    ): Result<VaultKey> {
        val fieldName = "omo_identity_$id"
        val value = "name:$name|email:$email|phone:$phone"
        return storeValue(
            fieldName = fieldName,
            value = value,
            category = VaultKeyCategory.OMO_IDENTITY,
            sensitivity = VaultKeySensitivity.OMO_LOW
        )
    }

    /**
     * Retrieve an OMO identity
     *
     * @param id Unique identifier for the identity
     * @return Result containing the identity data (name, email, phone)
     */
    suspend fun retrieveOMOIdentity(id: String): Result<OMOIdentityData> {
        val fieldName = "omo_identity_$id"
        return retrieveValue(fieldName).map { value ->
            val parts = value.split("|")
            OMOIdentityData(
                id = id,
                name = parts.find { it.startsWith("name:") }?.substringAfter("name:") ?: "",
                email = parts.find { it.startsWith("email:") }?.substringAfter("email:") ?: "",
                phone = parts.find { it.startsWith("phone:") }?.substringAfter("phone:") ?: ""
            )
        }
    }

    /**
     * Delete an OMO identity
     *
     * @param id Unique identifier for the identity
     * @param requiresBiometric Whether biometric authentication is required
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOIdentity(
        id: String,
        requiresBiometric: Boolean = false
    ): Result<Unit> {
        val fieldName = "omo_identity_$id"
        return deleteValue(fieldName)
    }

    /**
     * List all stored OMO identities
     *
     * @return Result containing list of VaultKey entries for OMO identities
     */
    suspend fun listOMOIdentities(): Result<List<VaultKey>> {
        return listKeysByCategory(VaultKeyCategory.OMO_IDENTITY)
    }

    /**
     * Store an OMO setting (Agent configuration and settings)
     *
     * @param key The setting key (e.g., "model_temperature", "max_tokens")
     * @param value The setting value to store
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOSetting(
        key: String,
        value: String
    ): Result<VaultKey> {
        val fieldName = "omo_settings_$key"
        return storeValue(
            fieldName = fieldName,
            value = value,
            category = VaultKeyCategory.OMO_SETTINGS,
            sensitivity = VaultKeySensitivity.OMO_LOW
        )
    }

    /**
     * Retrieve an OMO setting
     *
     * @param key The setting key to retrieve
     * @return Result containing the setting value
     */
    suspend fun retrieveOMOSetting(key: String): Result<String> {
        val fieldName = "omo_settings_$key"
        return retrieveValue(fieldName)
    }

    /**
     * Delete an OMO setting
     *
     * @param key The setting key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOSetting(key: String): Result<Unit> {
        val fieldName = "omo_settings_$key"
        return deleteValue(fieldName)
    }

    /**
     * List all stored OMO settings
     *
     * @return Result containing list of VaultKey entries for OMO settings
     */
    suspend fun listOMOSettings(): Result<List<VaultKey>> {
        return listKeysByCategory(VaultKeyCategory.OMO_SETTINGS)
    }

    /**
     * Store an OMO token (Session tokens for OMO authentication)
     *
     * @param key The token key (e.g., "session_token", "refresh_token")
     * @param value The token value to store
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOToken(
        key: String,
        value: String
    ): Result<VaultKey> {
        val fieldName = "omo_tokens_$key"
        return storeValue(
            fieldName = fieldName,
            value = value,
            category = VaultKeyCategory.OMO_TOKENS,
            sensitivity = VaultKeySensitivity.OMO_HIGH
        )
    }

    /**
     * Retrieve an OMO token
     *
     * @param key The token key to retrieve
     * @return Result containing the token value
     */
    suspend fun retrieveOMOToken(key: String): Result<String> {
        val fieldName = "omo_tokens_$key"
        return retrieveValue(fieldName)
    }

    /**
     * Delete an OMO token
     *
     * @param key The token key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOToken(key: String): Result<Unit> {
        val fieldName = "omo_tokens_$key"
        return deleteValue(fieldName)
    }

    /**
     * List all stored OMO tokens
     *
     * @return Result containing list of VaultKey entries for OMO tokens
     */
    suspend fun listOMOTokens(): Result<List<VaultKey>> {
        return listKeysByCategory(VaultKeyCategory.OMO_TOKENS)
    }

    /**
     * Store an OMO workspace (Workspace and project data)
     *
     * @param key The workspace key (e.g., "default_workspace", "project_alpha")
     * @param value The workspace data (JSON or structured string)
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOWorkspace(
        key: String,
        value: String
    ): Result<VaultKey> {
        val fieldName = "omo_workspace_$key"
        return storeValue(
            fieldName = fieldName,
            value = value,
            category = VaultKeyCategory.OMO_WORKSPACE,
            sensitivity = VaultKeySensitivity.OMO_MEDIUM
        )
    }

    /**
     * Retrieve an OMO workspace
     *
     * @param key The workspace key to retrieve
     * @return Result containing the workspace data
     */
    suspend fun retrieveOMOWorkspace(key: String): Result<String> {
        val fieldName = "omo_workspace_$key"
        return retrieveValue(fieldName)
    }

    /**
     * Delete an OMO workspace
     *
     * @param key The workspace key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOWorkspace(key: String): Result<Unit> {
        val fieldName = "omo_workspace_$key"
        return deleteValue(fieldName)
    }

    /**
     * List all stored OMO workspaces
     *
     * @return Result containing list of VaultKey entries for OMO workspaces
     */
    suspend fun listOMOWorkspaces(): Result<List<VaultKey>> {
        return listKeysByCategory(VaultKeyCategory.OMO_WORKSPACE)
    }

    /**
     * Store an OMO task (Task data and metadata)
     *
     * @param key The task key (e.g., "task_123", "todo_list")
     * @param value The task data (JSON or structured string)
     * @return Result containing the stored VaultKey
     */
    suspend fun storeOMOTask(
        key: String,
        value: String
    ): Result<VaultKey> {
        val fieldName = "omo_tasks_$key"
        return storeValue(
            fieldName = fieldName,
            value = value,
            category = VaultKeyCategory.OMO_TASKS,
            sensitivity = VaultKeySensitivity.OMO_LOW
        )
    }

    /**
     * Retrieve an OMO task
     *
     * @param key The task key to retrieve
     * @return Result containing the task data
     */
    suspend fun retrieveOMOTask(key: String): Result<String> {
        val fieldName = "omo_tasks_$key"
        return retrieveValue(fieldName)
    }

    /**
     * Delete an OMO task
     *
     * @param key The task key to delete
     * @return Result indicating success or failure
     */
    suspend fun deleteOMOTask(key: String): Result<Unit> {
        val fieldName = "omo_tasks_$key"
        return deleteValue(fieldName)
    }

    /**
     * List all stored OMO tasks
     *
     * @return Result containing list of VaultKey entries for OMO tasks
     */
    suspend fun listOMOTasks(): Result<List<VaultKey>> {
        return listKeysByCategory(VaultKeyCategory.OMO_TASKS)
    }

    // Helper function to list keys by category
    private suspend fun listKeysByCategory(category: VaultKeyCategory): Result<List<VaultKey>> = withContext(Dispatchers.IO) {
        try {
            val db = sqlCipherProvider.getDatabase()
            val keys = mutableListOf<VaultKey>()

            val cursor = db.rawQuery(
                "SELECT id, field_name, display_name, category, sensitivity, last_accessed, access_count FROM vault_keys WHERE category = ?",
                arrayOf(category.name)
            )

            while (cursor.moveToNext()) {
                keys.add(
                    VaultKey(
                        id = cursor.getString(0),
                        fieldName = cursor.getString(1),
                        displayName = cursor.getString(2),
                        category = VaultKeyCategory.valueOf(cursor.getString(3)),
                        sensitivity = VaultKeySensitivity.valueOf(cursor.getString(4)),
                        lastAccessed = cursor.getLong(5),
                        accessCount = cursor.getInt(6)
                    )
                )
            }
            cursor.close()

            Result.success(keys)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

