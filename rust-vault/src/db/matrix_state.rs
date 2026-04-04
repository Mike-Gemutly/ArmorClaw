use crate::db::pool::DbPool;
use crate::error::VaultError;
use pbkdf2::pbkdf2_hmac;
use sha2::Sha512;
use std::sync::Arc;

const PBKDF2_ITERATIONS: u32 = 256_000;
const KEY_LENGTH: usize = 32;

pub struct MatrixStateDb {
    pool: Arc<DbPool>,
}

impl MatrixStateDb {
    pub fn new(pool: DbPool) -> Result<Self, VaultError> {
        Ok(Self {
            pool: Arc::new(pool),
        })
    }

    async fn init_database(&self) -> Result<(), VaultError> {
        let guard = self.pool.get().await?;

        guard.execute(
            "CREATE TABLE IF NOT EXISTS matrix_state (
                key TEXT PRIMARY KEY,
                value BLOB NOT NULL,
                created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
                updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
            )",
            [],
        )
        .map_err(|e| VaultError::Database(format!("Failed to create matrix_state table: {}", e)))?;

        Ok(())
    }

    pub async fn store_state(&self, key: &str, value: Vec<u8>) -> Result<(), VaultError> {
        self.init_database().await?;

        let guard = self.pool.get().await?;

        guard.execute(
            "INSERT OR REPLACE INTO matrix_state (key, value, updated_at) VALUES (?, ?, strftime('%s', 'now'))",
            (key, value),
        )
        .map_err(|e| VaultError::Database(format!("Failed to store state: {}", e)))?;

        Ok(())
    }

    pub async fn retrieve_state(&self, key: &str) -> Result<Vec<u8>, VaultError> {
        self.init_database().await?;

        let guard = self.pool.get().await?;

        guard
            .query_row(
                "SELECT value FROM matrix_state WHERE key = ?",
                [key],
                |row| {
                    let value: Vec<u8> = row.get(0)?;
                    Ok(value)
                },
            )
            .map_err(|e| {
                if e.to_string().contains("QueryReturnedNoRows") {
                    VaultError::Database(format!("State not found: {}", key))
                } else {
                    VaultError::Database(format!("Failed to retrieve state: {}", e))
                }
            })
    }

    pub async fn delete_state(&self, key: &str) -> Result<(), VaultError> {
        self.init_database().await?;

        let guard = self.pool.get().await?;

        guard
            .execute("DELETE FROM matrix_state WHERE key = ?", [key])
            .map_err(|e| VaultError::Database(format!("Failed to delete state: {}", e)))?;

        Ok(())
    }
}

pub fn derive_key_for_test(master_key: &[u8], salt: &[u8]) -> [u8; KEY_LENGTH] {
    let mut key = [0u8; KEY_LENGTH];
    pbkdf2_hmac::<Sha512>(master_key, salt, PBKDF2_ITERATIONS, &mut key);
    key
}
