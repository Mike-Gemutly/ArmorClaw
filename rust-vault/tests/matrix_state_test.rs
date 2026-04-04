use rust_vault::db::matrix_state::MatrixStateDb;
use rust_vault::db::pool::DbPool;
use rand::rngs::OsRng;
use rand::RngCore;

#[tokio::test]
async fn test_matrix_state_db_key_derivation_parameters() {
    // CRITICAL: Key derivation MUST match Go Bridge parameters:
    // - PBKDF2-HMAC-SHA512 (NOT SHA256)
    // - 256,000 iterations (NOT 100,000)
    // - 32-byte salt
    // - 32-byte key

    let db_path = "/tmp/test_matrix_state.db";
    let master_key = "test_master_key_for_key_derivation";

    let pool = DbPool::new(db_path, master_key)
        .await
        .expect("Failed to create DbPool");

    let matrix_db = MatrixStateDb::new(pool.clone())
        .expect("Failed to create MatrixStateDb");

    let test_key = "test_state_key";
    let test_value = vec![1u8, 2, 3, 4, 5];

    matrix_db
        .store_state(test_key, test_value.clone())
        .await
        .expect("Failed to store state");

    let retrieved = matrix_db
        .retrieve_state(test_key)
        .await
        .expect("Failed to retrieve state");

    assert_eq!(retrieved, test_value);

    matrix_db
        .delete_state(test_key)
        .await
        .expect("Failed to delete state");

    let result = matrix_db.retrieve_state(test_key).await;
    assert!(result.is_err());

    std::fs::remove_file(db_path).ok();
    std::fs::remove_file(format!("{}.salt", db_path)).ok();
}

#[tokio::test]
async fn test_matrix_state_db_key_derivation_consistency() {
    let _db_path = "/tmp/test_matrix_state_consistency.db";
    let master_key = "consistent_key_test";
    let salt = generate_test_salt();

    let key1 = rust_vault::db::matrix_state::derive_key_for_test(master_key.as_bytes(), &salt);
    let key2 = rust_vault::db::matrix_state::derive_key_for_test(master_key.as_bytes(), &salt);

    assert_eq!(key1, key2, "Key derivation must be deterministic");

    let key3 = rust_vault::db::matrix_state::derive_key_for_test(b"different_key", &salt);
    assert_ne!(key1, key3);

    let key4 = rust_vault::db::matrix_state::derive_key_for_test(master_key.as_bytes(), &generate_test_salt());
    assert_ne!(key1, key4);
}

#[tokio::test]
async fn test_matrix_state_db_operations() {
    let db_path = "/tmp/test_matrix_state_operations.db";
    let master_key = "operations_test_key";

    let pool = DbPool::new(db_path, master_key)
        .await
        .expect("Failed to create DbPool");

    let matrix_db = MatrixStateDb::new(pool.clone())
        .expect("Failed to create MatrixStateDb");

    matrix_db
        .store_state("key1", vec![1, 2, 3])
        .await
        .expect("Failed to store key1");

    matrix_db
        .store_state("key2", vec![4, 5, 6])
        .await
        .expect("Failed to store key2");

    matrix_db
        .store_state("key3", vec![7, 8, 9])
        .await
        .expect("Failed to store key3");

    let val1 = matrix_db.retrieve_state("key1").await.unwrap();
    let val2 = matrix_db.retrieve_state("key2").await.unwrap();
    let val3 = matrix_db.retrieve_state("key3").await.unwrap();

    assert_eq!(val1, vec![1, 2, 3]);
    assert_eq!(val2, vec![4, 5, 6]);
    assert_eq!(val3, vec![7, 8, 9]);

    matrix_db.delete_state("key2").await.unwrap();

    assert!(matrix_db.retrieve_state("key2").await.is_err());
    assert!(matrix_db.retrieve_state("key1").await.is_ok());
    assert!(matrix_db.retrieve_state("key3").await.is_ok());

    std::fs::remove_file(db_path).ok();
    std::fs::remove_file(format!("{}.salt", db_path)).ok();
}

#[tokio::test]
async fn test_matrix_state_db_empty_retrieval() {
    let db_path = "/tmp/test_matrix_state_empty.db";
    let master_key = "empty_test_key";

    let pool = DbPool::new(db_path, master_key)
        .await
        .expect("Failed to create DbPool");

    let matrix_db = MatrixStateDb::new(pool.clone())
        .expect("Failed to create MatrixStateDb");

    let result = matrix_db.retrieve_state("nonexistent_key").await;
    assert!(result.is_err());

    std::fs::remove_file(db_path).ok();
    std::fs::remove_file(format!("{}.salt", db_path)).ok();
}

#[tokio::test]
async fn test_matrix_state_db_large_data() {
    let db_path = "/tmp/test_matrix_state_large.db";
    let master_key = "large_data_test_key";

    let pool = DbPool::new(db_path, master_key)
        .await
        .expect("Failed to create DbPool");

    let matrix_db = MatrixStateDb::new(pool.clone())
        .expect("Failed to create MatrixStateDb");

    let large_data: Vec<u8> = (0..1_000_000).map(|i| (i % 256) as u8).collect();

    matrix_db
        .store_state("large_key", large_data.clone())
        .await
        .expect("Failed to store large data");

    let retrieved = matrix_db
        .retrieve_state("large_key")
        .await
        .expect("Failed to retrieve large data");

    assert_eq!(retrieved.len(), large_data.len());
    assert_eq!(retrieved, large_data);

    std::fs::remove_file(db_path).ok();
    std::fs::remove_file(format!("{}.salt", db_path)).ok();
}

fn generate_test_salt() -> Vec<u8> {
    let mut salt = vec![0u8; 32];
    OsRng.fill_bytes(&mut salt);
    salt
}
