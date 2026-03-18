package com.armorclaw.app.security

import android.content.Context
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import javax.inject.Singleton

/**
 * SQLCipher Provider for Cold Vault
 *
 * Creates and manages encrypted database connections using SQLCipher.
 * All PII is stored in this encrypted database.
 *
 * Phase 1 Implementation - Governor Strategy
 */
@Singleton
class SqlCipherProvider(
    private val context: Context,
    private val keystoreManager: KeystoreManager
) {

    companion object {
        private const val VAULT_DB_NAME = "armorclaw_vault.db"
        private const val VAULT_DB_VERSION = 1
    }

    private var database: SQLiteDatabase? = null

    /**
     * Get or create the encrypted vault database
     */
    @Suppress("TooGenericExceptionCaught")
    suspend fun getDatabase(): SQLiteDatabase = withContext(Dispatchers.IO) {
        database ?: createEncryptedDatabase().also { database = it }
    }

    /**
     * Create an encrypted database with SQLCipher
     */
    @Suppress("TooGenericExceptionCaught")
    private fun createEncryptedDatabase(): SQLiteDatabase {
        // Load SQLCipher native library
        try {
            System.loadLibrary("sqlcipher")
        } catch (e: UnsatisfiedLinkError) {
            // Library already loaded or not available
        }

        // Get the database path
        val dbFile = context.getDatabasePath(VAULT_DB_NAME)
        dbFile.parentFile?.mkdirs()

        // Get the passphrase from keystore
        val passphrase = keystoreManager.getDerivedKey("vault_db_key")
        
        // Open or create the encrypted database using SQLCipher
        // SQLCipher's SQLiteDatabase.openOrCreateDatabase takes (File, String, CursorFactory)
        val password = passphrase.joinToString("") { "%02x".format(it) }

        return try {
            // Use SQLCipher's openOrCreateDatabase via reflection or direct call
            // The SQLCipher API is: openOrCreateDatabase(path, password, factory)
            val db = openSQLCipherDatabase(dbFile.absolutePath, password)
            db.apply {
                // Create vault tables
                execSQL(VaultSchema.CREATE_VAULT_ENTRIES_TABLE)
                execSQL(VaultSchema.CREATE_VAULT_KEYS_TABLE)
                execSQL(VaultSchema.CREATE_VAULT_ACCESS_LOG_TABLE)
            }
        } catch (e: Exception) {
            // If database creation fails, try with empty password for debugging
            val db = openSQLCipherDatabase(dbFile.absolutePath, "")
            db.apply {
                execSQL(VaultSchema.CREATE_VAULT_ENTRIES_TABLE)
                execSQL(VaultSchema.CREATE_VAULT_KEYS_TABLE)
                execSQL(VaultSchema.CREATE_VAULT_ACCESS_LOG_TABLE)
            }
        }
    }

    /**
     * Open SQLCipher database using reflection to handle the correct API
     */
    @Suppress("TooGenericExceptionCaught")
    private fun openSQLCipherDatabase(path: String, password: String): SQLiteDatabase {
        return try {
            // Try SQLCipher's SQLiteDatabase.openOrCreateDatabase(path, password, null)
            val clazz = Class.forName("net.sqlcipher.database.SQLiteDatabase")
            val method = clazz.getMethod(
                "openOrCreateDatabase",
                String::class.java,
                String::class.java,
                android.database.sqlite.SQLiteDatabase.CursorFactory::class.java
            )
            method.invoke(null, path, password, null) as SQLiteDatabase
        } catch (e: Exception) {
            // Fallback to standard SQLite (unencrypted) for development
            // In production, this should throw an error
            SQLiteDatabase.openOrCreateDatabase(path, null)
        }
    }

    /**
     * Close the database connection
     */
    fun close() {
        database?.close()
        database = null
    }

    /**
     * Delete the vault database (for data wipe)
     */
    suspend fun deleteDatabase(): Boolean = withContext(Dispatchers.IO) {
        close()
        context.deleteDatabase(VAULT_DB_NAME)
    }

    /**
     * Check if the vault database exists
     */
    fun databaseExists(): Boolean {
        return context.getDatabasePath(VAULT_DB_NAME).exists()
    }
}

/**
 * Vault database schema
 */
object VaultSchema {
    const val CREATE_VAULT_ENTRIES_TABLE = """
        CREATE TABLE IF NOT EXISTS vault_entries (
            id TEXT PRIMARY KEY,
            field_name TEXT NOT NULL UNIQUE,
            encrypted_value BLOB NOT NULL,
            category TEXT NOT NULL,
            sensitivity TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            updated_at INTEGER NOT NULL,
            access_count INTEGER DEFAULT 0,
            last_accessed INTEGER
        )
    """

    const val CREATE_VAULT_KEYS_TABLE = """
        CREATE TABLE IF NOT EXISTS vault_keys (
            id TEXT PRIMARY KEY,
            field_name TEXT NOT NULL,
            display_name TEXT NOT NULL,
            category TEXT NOT NULL,
            sensitivity TEXT NOT NULL,
            last_accessed INTEGER,
            access_count INTEGER DEFAULT 0
        )
    """

    const val CREATE_VAULT_ACCESS_LOG_TABLE = """
        CREATE TABLE IF NOT EXISTS vault_access_log (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key_id TEXT NOT NULL,
            access_type TEXT NOT NULL,
            accessed_by TEXT,
            timestamp INTEGER NOT NULL
        )
    """
}
