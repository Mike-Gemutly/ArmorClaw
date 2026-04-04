//! BlindFill Integration Layer
//!
//! Integrates placeholder parsing with secret retrieval from VaultDb.
//! Processes request bodies containing `{{secret:name}}` placeholders and
//! replaces them with actual secret values.

use crate::db::vault::VaultDb;
use crate::blindfill::placeholder::{parse_placeholders, replace_placeholders};
use crate::error::VaultError;
use std::collections::HashMap;

#[cfg(test)]
use zeroize::Zeroizing;

/// Integrates placeholder parsing with VaultDb secret retrieval
pub struct BlindFillIntegrator {
    vault: VaultDb,
}

impl BlindFillIntegrator {
    /// Creates a new BlindFillIntegrator with the given VaultDb
    pub fn new(vault: VaultDb) -> Self {
        Self { vault }
    }

    /// Processes a request body containing placeholders and replaces them with secret values
    ///
    /// # Arguments
    ///
    /// * `request_body` - The request body containing placeholders in `{{secret:name}}` format
    ///
    /// # Returns
    ///
    /// The request body with all placeholders replaced with actual secret values
    ///
    /// # Errors
    ///
    /// Returns an error if:
    /// - Placeholder parsing fails
    /// - A secret is not found in the vault
    /// - Database retrieval fails
    pub async fn process_request(&self, request_body: &str) -> Result<String, VaultError> {
        let placeholders = parse_placeholders(request_body)?;

        if placeholders.is_empty() {
            return Ok(request_body.to_string());
        }

        let mut secrets = HashMap::new();

        for placeholder in &placeholders {
            let secret_value = self.vault.retrieve_secret(placeholder).await?;
            secrets.insert(placeholder.clone(), secret_value.to_string());
        }

        let result = replace_placeholders(request_body, &placeholders, &secrets)?;

        Ok(result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::pool::DbPool;
    use std::fs;

    #[tokio::test]
    async fn test_process_request_with_placeholders() {
        let db_path = "/tmp/test_integration_process.db";
        let salt_path = format!("{}.salt", db_path);

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);

        let pool = DbPool::new(db_path, "test_key").await.unwrap();
        let vault = VaultDb::new(pool);
        vault
            .store_secret("secret1", Zeroizing::new("value1".to_string()))
            .await
            .unwrap();

        let integrator = BlindFillIntegrator::new(vault);
        let body = r#"{"key": "{{secret:secret1}}"}"#;

        let result = integrator.process_request(body).await.unwrap();

        assert!(result.contains("value1"));
        assert!(!result.contains("{{secret:"));

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);
    }

    #[tokio::test]
    async fn test_process_request_no_placeholders() {
        let db_path = "/tmp/test_integration_no_placeholders.db";
        let salt_path = format!("{}.salt", db_path);

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);

        let pool = DbPool::new(db_path, "test_key").await.unwrap();
        let vault = VaultDb::new(pool);
        let integrator = BlindFillIntegrator::new(vault);
        let body = r#"{"key": "value"}"#;

        let result = integrator.process_request(body).await.unwrap();

        assert_eq!(result, body);

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);
    }
}
