//! CDP Interceptor for BlindFill™ Secret Injection
//!
//! Intercepts Chrome DevTools Protocol (CDP) requests to filter network resources
//! and inject secrets directly into browser without exposing them to agents.
//!
//! # Security Model
//!
//! - Only processes XHR and Fetch requests
//! - Resolves `{{VAULT:field:hash}}` placeholders with actual values
//! - Real values are injected directly into browser, never logged
//! - Agents never see the actual secret values

use crate::blindfill::placeholder::{
    parse_placeholders, replace_placeholders, PlaceholderParseError,
};
use serde_json::Value;
use std::collections::HashMap;

/// CDP Interceptor for filtering and modifying network requests
///
/// Filters requests by resourceType (XHR, Fetch only) and resolves
/// placeholders in request/response bodies.
pub struct CdpInterceptor {
    _allowed_resource_types: Vec<String>,
}

impl CdpInterceptor {
    /// Creates a new CDP interceptor with default XHR and Fetch filters
    pub fn new() -> Self {
        CdpInterceptor {
            _allowed_resource_types: vec!["XHR".to_string(), "Fetch".to_string()],
        }
    }

    /// Generates Fetch.enable parameters for CDP
    ///
    /// Returns a JSON structure with patterns that filter by resourceType
    /// (XHR and Fetch only) to intercept only relevant requests.
    ///
    /// # Returns
    ///
    /// A JSON Value containing of Fetch.enable command with patterns array.
    pub fn enable_params() -> Value {
        serde_json::json!({
            "method": "Fetch.enable",
            "params": {
                "patterns": [
                    {
                        "urlPattern": "*",
                        "resourceType": "XHR",
                        "requestStage": "Request"
                    },
                    {
                        "urlPattern": "*",
                        "resourceType": "Fetch",
                        "requestStage": "Request"
                    }
                ]
            }
        })
    }

    /// Resolves placeholders in a JSON value with provided secrets
    ///
    /// # Security Guarantees
    ///
    /// - Resolves `{{VAULT:field:hash}}` placeholders with actual values
    /// - Real values are only injected into the JSON value, never logged
    /// - Errors don't expose secret values
    ///
    /// # Arguments
    ///
    /// * `json_value` - Mutable reference to JSON value containing placeholders
    /// * `secrets` - Map of field:hash keys to their values
    ///
    /// # Returns
    ///
    /// Ok(()) if all placeholders were resolved successfully
    /// Err(PlaceholderParseError) if a placeholder is malformed or secret not found
    pub fn resolve_placeholders(
        &self,
        json_value: &mut Value,
        secrets: &HashMap<String, String>,
    ) -> Result<(), PlaceholderParseError> {
        if let Some(text) = json_value.as_str() {
            let placeholders = parse_placeholders(text)?;
            let replaced = replace_placeholders(text, &placeholders, secrets)?;
            *json_value = Value::String(replaced);
        } else if let Some(obj) = json_value.as_object_mut() {
            for (_, value) in obj.iter_mut() {
                self.resolve_placeholders(value, secrets)?;
            }
        } else if let Some(arr) = json_value.as_array_mut() {
            for value in arr.iter_mut() {
                self.resolve_placeholders(value, secrets)?;
            }
        }

        Ok(())
    }
}

impl Default for CdpInterceptor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_enable_params_structure() {
        let params = CdpInterceptor::enable_params();

        assert_eq!(params["method"], "Fetch.enable");
        assert!(params["params"]["patterns"].is_array());
        assert_eq!(params["params"]["patterns"].as_array().unwrap().len(), 2);
    }

    #[test]
    fn test_resolve_placeholders_simple_string() {
        let interceptor = CdpInterceptor::new();
        let mut json_value = Value::String("{{VAULT:api_key:a1b2c3}}".to_string());

        let mut secrets = HashMap::new();
        secrets.insert("api_key:a1b2c3".to_string(), "secret123".to_string());

        let result = interceptor.resolve_placeholders(&mut json_value, &secrets);
        assert!(result.is_ok());
        assert_eq!(json_value, Value::String("secret123".to_string()));
    }

    #[test]
    fn test_resolve_placeholders_nested_object() {
        let interceptor = CdpInterceptor::new();
        let mut json_value = serde_json::json!({
            "payment": {
                "card_number": "{{VAULT:card_number:d4e5f6}}"
            }
        });

        let mut secrets = HashMap::new();
        secrets.insert(
            "card_number:d4e5f6".to_string(),
            "4242424242424242".to_string(),
        );

        let result = interceptor.resolve_placeholders(&mut json_value, &secrets);
        assert!(result.is_ok());
        assert_eq!(json_value["payment"]["card_number"], "4242424242424242");
    }

    #[test]
    fn test_resolve_placeholders_multiple_fields() {
        let interceptor = CdpInterceptor::new();
        let mut json_value = serde_json::json!({
            "email": "{{VAULT:email:a1b2c3}}",
            "password": "{{VAULT:password:d4e5f6}}"
        });

        let mut secrets = HashMap::new();
        secrets.insert("email:a1b2c3".to_string(), "user@example.com".to_string());
        secrets.insert("password:d4e5f6".to_string(), "secret123".to_string());

        let result = interceptor.resolve_placeholders(&mut json_value, &secrets);
        assert!(result.is_ok());
        assert_eq!(json_value["email"], "user@example.com");
        assert_eq!(json_value["password"], "secret123");
    }

    #[test]
    fn test_old_secret_format_rejected() {
        let interceptor = CdpInterceptor::new();
        let mut json_value = Value::String("{{secret:api_key}}".to_string());

        let mut secrets = HashMap::new();
        secrets.insert("api_key".to_string(), "secret123".to_string());

        let result = interceptor.resolve_placeholders(&mut json_value, &secrets);
        assert!(result.is_err());
    }
}
