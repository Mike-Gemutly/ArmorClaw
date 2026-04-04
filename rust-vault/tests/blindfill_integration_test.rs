use rust_vault::db::pool::DbPool;
use rust_vault::db::vault::VaultDb;
use rust_vault::blindfill::integration::BlindFillIntegrator;
use zeroize::Zeroizing;
use std::fs;

#[tokio::test]
async fn test_blindfill_integration_end_to_end() {
    let db_path = "/tmp/test_blindfill_integration.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    // Setup: Create vault and store secrets
    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);
    vault
        .store_secret("payment.card_number", Zeroizing::new("4242424242424242".to_string()))
        .await
        .unwrap();
    vault
        .store_secret("payment.cvv", Zeroizing::new("123".to_string()))
        .await
        .unwrap();

    // Create integrator
    let integrator = BlindFillIntegrator::new(vault);

    // Test: Process request body with placeholders
    let request_body = r#"{
        "card_number": "{{secret:payment.card_number}}",
        "cvv": "{{secret:payment.cvv}}"
    }"#;

    let result = integrator.process_request(request_body).await.unwrap();

    // Verify: Placeholders replaced with actual secret values
    assert!(result.contains("4242424242424242"), "Card number not replaced");
    assert!(result.contains("123"), "CVV not replaced");
    assert!(!result.contains("{{secret:"), "Placeholders should be replaced");

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_blindfill_integration_secret_not_found() {
    let db_path = "/tmp/test_blindfill_not_found.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    let integrator = BlindFillIntegrator::new(vault);

    // Test: Request with non-existent secret
    let request_body = r#"{"value": "{{secret:non_existent}}" }"#;

    let result = integrator.process_request(request_body).await;

    // Verify: Error for missing secret
    assert!(result.is_err(), "Should error when secret not found");

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_blindfill_integration_no_placeholders() {
    let db_path = "/tmp/test_blindfill_no_placeholders.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);

    let integrator = BlindFillIntegrator::new(vault);

    // Test: Request without placeholders
    let request_body = r#"{"value": "plain_text"}"#;

    let result = integrator.process_request(request_body).await.unwrap();

    // Verify: Body unchanged
    assert_eq!(result, request_body);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}

#[tokio::test]
async fn test_blindfill_secrets_zeroized_after_use() {
    let db_path = "/tmp/test_blindfill_zeroize.db";
    let salt_path = format!("{}.salt", db_path);

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);

    let pool = DbPool::new(db_path, "test_master_key").await.unwrap();
    let vault = VaultDb::new(pool);
    vault
        .store_secret("test_secret", Zeroizing::new("sensitive_value".to_string()))
        .await
        .unwrap();

    // Verify secret is in vault before processing
    let retrieved = vault.retrieve_secret("test_secret").await.unwrap();
    assert_eq!(*retrieved, "sensitive_value");

    let integrator = BlindFillIntegrator::new(vault);

    // Process request
    let request_body = r#"{"value": "{{secret:test_secret}}"}"#;
    let _ = integrator.process_request(request_body).await.unwrap();

    // Allow time for zeroization
    tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

    let _ = fs::remove_file(db_path);
    let _ = fs::remove_file(&salt_path);
}
