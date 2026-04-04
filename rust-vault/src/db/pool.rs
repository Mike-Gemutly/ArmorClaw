use crate::error::VaultError;
use base64::{engine::general_purpose::STANDARD, Engine};
use pbkdf2::pbkdf2_hmac;
use rand::rngs::OsRng;
use rand::RngCore;
use rusqlite::{Connection, OpenFlags};
use sha2::Sha512;
use std::fs;
use std::path::Path;
use std::sync::Arc;
use tokio::sync::Mutex;

const SALT_LENGTH: usize = 32;
const PBKDF2_ITERATIONS: u32 = 256_000;
const KEY_LENGTH: usize = 32;

#[derive(Clone)]
pub struct DbPool {
    inner: Arc<Mutex<Connection>>,
}

impl DbPool {
    pub async fn new(db_path: &str, master_key: &str) -> Result<Self, VaultError> {
        let salt_path = format!("{}.salt", db_path);
        let salt = Self::load_or_generate_salt(&salt_path)?;

        let derived_key = Self::derive_key(master_key.as_bytes(), &salt);

        let conn = Self::create_encrypted_connection(db_path, &derived_key)?;

        Ok(Self {
            inner: Arc::new(Mutex::new(conn)),
        })
    }

    pub async fn get(&self) -> Result<ConnectionGuard, VaultError> {
        let mutex = Arc::clone(&self.inner);
        Ok(ConnectionGuard {
            _guard: mutex.lock_owned().await,
        })
    }

    fn load_or_generate_salt(salt_path: &str) -> Result<Vec<u8>, VaultError> {
        if Path::new(salt_path).exists() {
            let salt_b64 = fs::read_to_string(salt_path)
                .map_err(|e| VaultError::Database(format!("Failed to read salt file: {}", e)))?;

            let salt = STANDARD.decode(&salt_b64)
                .map_err(|e| VaultError::Database(format!("Failed to decode salt: {}", e)))?;

            if salt.len() != SALT_LENGTH {
                return Err(VaultError::Database(
                    format!("Invalid salt length: expected {}, got {}", SALT_LENGTH, salt.len())
                ));
            }

            Ok(salt)
        } else {
            let mut salt = vec![0u8; SALT_LENGTH];
            OsRng.fill_bytes(&mut salt);

            let salt_b64 = STANDARD.encode(&salt);

            fs::write(salt_path, salt_b64)
                .map_err(|e| VaultError::Database(format!("Failed to write salt file: {}", e)))?;

            fs::set_permissions(salt_path, std::os::unix::fs::PermissionsExt::from_mode(0o600))
                .map_err(|e| VaultError::Database(format!("Failed to set salt file permissions: {}", e)))?;

            Ok(salt)
        }
    }

    fn derive_key(master_key: &[u8], salt: &[u8]) -> [u8; KEY_LENGTH] {
        let mut key = [0u8; KEY_LENGTH];
        pbkdf2_hmac::<Sha512>(master_key, salt, PBKDF2_ITERATIONS, &mut key);
        key
    }

    fn create_encrypted_connection(db_path: &str, derived_key: &[u8; KEY_LENGTH]) -> Result<Connection, VaultError> {
        let conn = Connection::open_with_flags(db_path, OpenFlags::SQLITE_OPEN_READ_WRITE | OpenFlags::SQLITE_OPEN_CREATE)?;

        let key_hex = hex::encode(derived_key);

        conn.execute_batch(&format!(
            "PRAGMA key = \"x'{}'\";
             PRAGMA cipher_page_size = 4096;
             PRAGMA kdf_iter = 256000;
             PRAGMA cipher_hmac_algorithm = HMAC_SHA512;
             PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA512;",
            key_hex
        ))?;

        conn.execute_batch(
            "PRAGMA cipher_plaintext_header_size = 32;
             PRAGMA synchronous = NORMAL;
             PRAGMA journal_mode = WAL;
             PRAGMA wal_autocheckpoint = 1000;"
        )?;

        Ok(conn)
    }
}

pub struct ConnectionGuard {
    _guard: tokio::sync::OwnedMutexGuard<Connection>,
}

impl ConnectionGuard {
    pub fn query_row<F, R, P>(&self, sql: &str, params: P, f: F) -> Result<R, rusqlite::Error>
    where
        P: rusqlite::Params,
        F: FnOnce(&rusqlite::Row) -> Result<R, rusqlite::Error>,
    {
        self._guard.query_row(sql, params, f)
    }

    pub fn execute<P>(&self, sql: &str, params: P) -> Result<usize, rusqlite::Error>
    where
        P: rusqlite::Params,
    {
        self._guard.execute(sql, params)
    }

    pub fn prepare(&self, sql: &str) -> Result<rusqlite::Statement<'_>, rusqlite::Error> {
        self._guard.prepare(sql)
    }
}
