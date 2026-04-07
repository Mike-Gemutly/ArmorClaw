//! BlindFill Integration Layer
//!
//! Integrates placeholder parsing with secret retrieval from VaultDb.
//! Processes request bodies containing `{{VAULT:field:hash}}` placeholders and
//! replaces them with actual secret values.
//!
//! # Security Model
//!
//! - Placeholders are in `{{VAULT:field:hash}}` format
//! - Real values are only injected into browser via CDP
//! - Agent context only sees placeholders, never actual values
//! - Secrets are retrieved by field:hash composite key

use crate::db::vault::VaultDb;
use crate::blindfill::placeholder::{parse_placeholders, replace_placeholders, Placeholder};
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
    /// # Security Guarantees
    ///
    /// - Real values are only replaced in output, never logged or returned
    /// - Secrets are retrieved by field:hash composite key
    /// - Input validation ensures only valid placeholder format is accepted
    ///
    /// # Arguments
    ///
    /// * `request_body` - The request body containing placeholders in `{{VAULT:field:hash}}` format
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
            // Create composite key from field:hash
            let key = format!("{}:{}", placeholder.field, placeholder.hash);
            let secret_value = self.vault.retrieve_secret(&key).await?;
            secrets.insert(key, secret_value.to_string());
        }

        let result = replace_placeholders(request_body, &placeholders, &secrets)?;

        Ok(result)
    }

    /// Extracts placeholders from a request body without resolving values
    ///
    /// # Security Guarantees
    ///
    /// - Returns only placeholder strings in `{{VAULT:field:hash}}` format
    /// - Never returns actual secret values
    /// - Safe to use in agent context
    ///
    /// # Arguments
    ///
    /// * `request_body` - The request body containing placeholders
    ///
    /// # Returns
    ///
    /// A vector of placeholder strings in `{{VAULT:field:hash}}` format
    ///
    /// # Errors
    ///
    /// Returns an error if placeholder parsing fails
    pub fn extract_placeholders(&self, request_body: &str) -> Result<Vec<String>, VaultError> {
        let placeholders = parse_placeholders(request_body)?;

        // Return only placeholder strings, never actual values
        Ok(placeholders.into_iter().map(|p| p.to_string()).collect())
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
            .store_secret("email:a1b2c3", Zeroizing::new("user@example.com".to_string()))
            .await
            .unwrap();

        let integrator = BlindFillIntegrator::new(vault);
        let body = r#"{"key": "{{VAULT:email:a1b2c3}}"}"#;

        let result = integrator.process_request(body).await.unwrap();

        assert!(result.contains("user@example.com"));
        assert!(!result.contains("{{VAULT:"));

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

    #[tokio::test]
    async fn test_extract_placeholders_for_agent_context() {
        let db_path = "/tmp/test_extract_placeholders.db";
        let salt_path = format!("{}.salt", db_path);

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);

        let pool = DbPool::new(db_path, "test_key").await.unwrap();
        let vault = VaultDb::new(pool);
        vault
            .store_secret("email:a1b2c3", Zeroizing::new("user@example.com".to_string()))
            .await
            .unwrap();

        let integrator = BlindFillIntegrator::new(vault);
        let body = r#"{"email": "{{VAULT:email:a1b2c3}}", "password": "{{VAULT:pass:d4e5f6}}"}"#;

        // Extract placeholders - should only return placeholder strings, not values
        let placeholders = integrator.extract_placeholders(body).unwrap();

        assert_eq!(placeholders.len(), 2);
        assert!(placeholders.contains(&"{{VAULT:email:a1b2c3}}".to_string()));
        assert!(placeholders.contains(&"{{VAULT:pass:d4e5f6}}".to_string()));

        // Verify actual values are NOT in the result
        for placeholder in &placeholders {
            assert!(!placeholder.contains("user@example.com"));
            assert!(!placeholder.contains("password123"));
        }

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);
    }

    #[tokio::test]
    async fn test_old_secret_format_rejected() {
        let db_path = "/tmp/test_old_format_rejected.db";
        let salt_path = format!("{}.salt", db_path);

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);

        let pool = DbPool::new(db_path, "test_key").await.unwrap();
        let vault = VaultDb::new(pool);
        let integrator = BlindFillIntegrator::new(vault);

        // Old format should be rejected
        let body = r#"{"key": "{{secret:email}}"}"#;

        let result = integrator.process_request(body).await;

        assert!(result.is_err());

        let _ = fs::remove_file(db_path);
        let _ = fs::remove_file(&salt_path);
    }
}
