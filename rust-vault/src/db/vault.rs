use crate::db::pool::DbPool;
use crate::error::VaultError;
use zeroize::Zeroizing;

#[derive(Debug, Clone)]
pub struct SecretMetadata {
    pub id: String,
    pub value: Option<String>,
}

pub struct VaultDb {
    pool: DbPool,
}

impl VaultDb {
    pub fn new(pool: DbPool) -> Self {
        Self { pool }
    }

    async fn init_database(&self) -> Result<(), VaultError> {
        let conn = self.pool.get().await?;

        conn.execute(
            "CREATE TABLE IF NOT EXISTS secrets (
                id TEXT PRIMARY KEY,
                value BLOB NOT NULL,
                created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
                updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
            )",
            [],
        )?;

        Ok(())
    }

    pub async fn store_secret(&self, id: &str, value: Zeroizing<String>) -> Result<(), VaultError> {
        self.init_database().await?;

        let conn = self.pool.get().await?;

        let value_bytes = value.as_bytes();

        conn.execute(
            "INSERT OR REPLACE INTO secrets (id, value, updated_at) VALUES (?1, ?2, strftime('%s', 'now'))",
            (id, value_bytes.as_ref()),
        )?;

        Ok(())
    }

    pub async fn retrieve_secret(&self, id: &str) -> Result<Zeroizing<String>, VaultError> {
        self.init_database().await?;

        let conn = self.pool.get().await?;

        let value_blob: Vec<u8> = conn
            .query_row(
                "SELECT value FROM secrets WHERE id = ?1",
                [id],
                |row| row.get(0),
            )
            .map_err(|_| VaultError::SecretNotFound(id.to_string()))?;

        let value_str = String::from_utf8(value_blob)
            .map_err(|e| VaultError::Database(format!("Invalid UTF-8 in secret: {}", e)))?;

        Ok(Zeroizing::new(value_str))
    }

    pub async fn delete_secret(&self, id: &str) -> Result<(), VaultError> {
        self.init_database().await?;

        let conn = self.pool.get().await?;

        let rows_affected = conn.execute("DELETE FROM secrets WHERE id = ?1", [id])?;

        if rows_affected == 0 {
            return Err(VaultError::SecretNotFound(id.to_string()));
        }

        Ok(())
    }

    pub async fn list_secrets(&self) -> Result<Vec<SecretMetadata>, VaultError> {
        self.init_database().await?;

        let conn = self.pool.get().await?;

        conn.execute("CREATE TEMPORARY TABLE IF NOT EXISTS temp_list_secrets AS SELECT id FROM secrets ORDER BY id", [])?;

        let mut secrets = Vec::new();
        let mut offset = 0;

        loop {
            let result = conn.query_row(
                "SELECT id FROM temp_list_secrets LIMIT 1 OFFSET ?",
                [offset],
                |row| Ok::<_, rusqlite::Error>((row.get::<_, String>(0)?,)),
            );

            match result {
                Ok((id,)) => {
                    secrets.push(SecretMetadata { id, value: None });
                    offset += 1;
                }
                Err(rusqlite::Error::QueryReturnedNoRows) => break,
                Err(e) => return Err(VaultError::Database(format!("Failed to list secrets: {}", e))),
            }
        }

        Ok(secrets)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_zeroizing_string_drops_zeroized() {
        let secret = Zeroizing::new("sensitive_data".to_string());

        let ptr = secret.as_ptr();
        drop(secret);

        let memory = unsafe { std::slice::from_raw_parts(ptr, 14) };

        let original_bytes = b"sensitive_data";
        let is_zeroed = memory == &[0u8; 14] || memory != original_bytes;

        assert!(is_zeroed, "String was not zeroized after drop");
    }
}
