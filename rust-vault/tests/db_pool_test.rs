use rust_vault::db::pool::DbPool;
use std::fs;
use std::path::Path;
use base64::prelude::*;

#[tokio::test]
async fn test_db_pool_structure() {
    let test_db_path = "test_structure.db";
    let test_master_key = "test_master_key_structure";

    let _ = fs::remove_file(test_db_path);
    let salt_path = format!("{}.salt", test_db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(test_db_path, test_master_key)
        .await
        .expect("Failed to create pool");

    let conn = pool.get().await.expect("Failed to get connection");

    conn.execute("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)", [])
        .expect("Failed to create table");

    conn.execute("INSERT INTO test (id, value) VALUES (1, 'test')", [])
        .expect("Failed to insert");

    let value: String = conn
        .query_row("SELECT value FROM test WHERE id = 1", [], |row| row.get(0))
        .expect("Failed to query");

    assert_eq!(value, "test");

    drop(pool);
    let _ = fs::remove_file(test_db_path);
    let _ = fs::remove_file(salt_path);
}

#[tokio::test]
async fn test_salt_generation_and_persistence() {
    let test_db_path = "test_salt.db";
    let test_master_key = "test_master_key_salt";

    let _ = fs::remove_file(test_db_path);
    let salt_path = format!("{}.salt", test_db_path);
    let _ = fs::remove_file(&salt_path);

    let pool1 = DbPool::new(test_db_path, test_master_key)
        .await
        .expect("Failed to create first pool");
    drop(pool1);

    assert!(
        Path::new(&salt_path).exists(),
        "Salt file should be created after pool creation"
    );

    let salt_content = fs::read_to_string(&salt_path)
        .expect("Failed to read salt file");

    let salt_bytes = BASE64_STANDARD.decode(&salt_content)
        .expect("Salt content should be valid base64");
    assert_eq!(
        salt_bytes.len(),
        32,
        "Salt should be 32 bytes"
    );

    let pool2 = DbPool::new(test_db_path, test_master_key)
        .await
        .expect("Failed to create second pool");

    let conn = pool2.get().await.expect("Failed to get connection");

    conn.execute(
        "CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY)",
        [],
    )
    .expect("Failed to create test table");

    conn.execute("INSERT INTO test_table (id) VALUES (1)", [])
        .expect("Failed to insert test data");

    let count: i64 = conn
        .query_row("SELECT COUNT(*) FROM test_table", [], |row| row.get(0))
        .expect("Failed to query test table");
    assert_eq!(count, 1, "Should be able to read data from database");

    drop(pool2);
    let _ = fs::remove_file(test_db_path);
    let _ = fs::remove_file(salt_path);
}

#[tokio::test]
async fn test_connection_pool_works() {
    let test_db_path = "test_pool.db";
    let test_master_key = "test_master_key_pool";

    let _ = fs::remove_file(test_db_path);
    let salt_path = format!("{}.salt", test_db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(test_db_path, test_master_key)
        .await
        .expect("Failed to create pool");

    let conn1 = pool.get().await.expect("Failed to get first connection");

    conn1
        .execute("CREATE TABLE test (id INTEGER)", [])
        .expect("Failed to create table with conn1");

    drop(conn1);

    let conn2 = pool.get().await.expect("Failed to get second connection");

    conn2
        .execute("INSERT INTO test (id) VALUES (1)", [])
        .expect("Failed to insert with conn2");

    let count: i64 = conn2
        .query_row("SELECT COUNT(*) FROM test", [], |row| row.get(0))
        .expect("Failed to query");
    assert_eq!(count, 1, "Connection pool should work correctly");

    drop(pool);
    let _ = fs::remove_file(test_db_path);
    let _ = fs::remove_file(salt_path);
}

#[tokio::test]
async fn test_key_derivation_correctness() {
    use pbkdf2::pbkdf2_hmac;
    use sha2::Sha512;

    let master_key = b"test_master_key";
    let salt = vec![1u8; 32];
    const PBKDF2_ITERATIONS: u32 = 256_000;
    const KEY_LENGTH: usize = 32;

    let mut key1 = [0u8; KEY_LENGTH];
    pbkdf2_hmac::<Sha512>(master_key, &salt, PBKDF2_ITERATIONS, &mut key1);

    let mut key2 = [0u8; KEY_LENGTH];
    pbkdf2_hmac::<Sha512>(master_key, &salt, PBKDF2_ITERATIONS, &mut key2);

    assert_eq!(key1, key2, "Key derivation should be deterministic");

    let different_salt = vec![2u8; 32];
    let mut key3 = [0u8; KEY_LENGTH];
    pbkdf2_hmac::<Sha512>(master_key, &different_salt, PBKDF2_ITERATIONS, &mut key3);

    assert_ne!(key1, key3, "Different salt should produce different key");
}

#[tokio::test]
async fn test_wal_mode_enabled() {
    let test_db_path = "test_wal.db";
    let test_master_key = "test_master_key_wal";

    let _ = fs::remove_file(test_db_path);
    let salt_path = format!("{}.salt", test_db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(test_db_path, test_master_key)
        .await
        .expect("Failed to create pool");

    let conn = pool.get().await.expect("Failed to get connection");

    let journal_mode: String = conn
        .query_row("PRAGMA journal_mode", [], |row| row.get::<_, String>(0))
        .expect("Failed to query journal_mode");
    assert_eq!(journal_mode, "wal", "journal_mode should be wal");

    drop(pool);
    let _ = fs::remove_file(test_db_path);
    let _ = fs::remove_file(salt_path);
    let _ = fs::remove_file(format!("{}-wal", test_db_path));
    let _ = fs::remove_file(format!("{}-shm", test_db_path));
}
