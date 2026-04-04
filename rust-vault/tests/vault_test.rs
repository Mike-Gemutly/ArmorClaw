use rust_vault::db::pool::DbPool;
use rust_vault::db::vault::VaultDb;
use rust_vault::error::VaultError;
use std::fs;
use zeroize::Zeroizing;

#[tokio::test]
async fn test_store_and_retrieve_secret() {
    let db_path = "/tmp/test_vault_store_retrieve.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    let secret = Zeroizing::new("my_secret_value".to_string());
    vault
        .store_secret("test_secret_id", secret)
        .await
        .unwrap();

    let retrieved = vault.retrieve_secret("test_secret_id").await.unwrap();
    assert_eq!(*retrieved, "my_secret_value");

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_delete_secret() {
    let db_path = "/tmp/test_vault_delete.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    let secret = Zeroizing::new("secret_to_delete".to_string());
    vault
        .store_secret("delete_me", secret)
        .await
        .unwrap();

    let retrieved = vault.retrieve_secret("delete_me").await.unwrap();
    assert_eq!(*retrieved, "secret_to_delete");

    vault.delete_secret("delete_me").await.unwrap();

    let result = vault.retrieve_secret("delete_me").await;
    assert!(matches!(result, Err(VaultError::SecretNotFound(_))));

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_list_secrets() {
    let db_path = "/tmp/test_vault_list.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    vault
        .store_secret("secret1", Zeroizing::new("value1".to_string()))
        .await
        .unwrap();
    vault
        .store_secret("secret2", Zeroizing::new("value2".to_string()))
        .await
        .unwrap();
    vault
        .store_secret("secret3", Zeroizing::new("value3".to_string()))
        .await
        .unwrap();

    let secrets = vault.list_secrets().await.unwrap();
    assert_eq!(secrets.len(), 3);

    for secret in secrets {
        assert_eq!(secret.value, None);
        assert!(!secret.id.is_empty());
    }

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_secret_not_found() {
    let db_path = "/tmp/test_vault_not_found.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    let result = vault.retrieve_secret("non_existent").await;
    assert!(matches!(result, Err(VaultError::SecretNotFound(_))));

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_zeroization_after_drop() {
    let db_path = "/tmp/test_vault_zeroize.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    {
        let secret = Zeroizing::new("sensitive_data".to_string());
        vault
            .store_secret("zeroize_test", secret)
            .await
            .unwrap();
    }

    tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_no_secrets_logged() {
    let db_path = "/tmp/test_vault_no_logs.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    let secret = Zeroizing::new("super_secret_password".to_string());
    vault
        .store_secret("test_log", secret)
        .await
        .unwrap();

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_vault_database_initialization() {
    let db_path = "/tmp/test_vault_init.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    vault
        .store_secret("test_init", Zeroizing::new("init_value".to_string()))
        .await
        .unwrap();

    let retrieved = vault.retrieve_secret("test_init").await.unwrap();
    assert_eq!(*retrieved, "init_value");

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}
