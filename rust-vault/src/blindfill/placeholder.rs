//! Placeholder Parser for BlindFill™ Secret Injection
//!
//! Parses placeholders in the format `{{VAULT:field:hash}}` for secure secret lookups.
//!
//! # Security Model
//!
//! The placeholder format `{{VAULT:field:hash}}` ensures that:
//! - Real secret values NEVER appear in agent prompts or logs
//! - Only placeholders reach the agent context
//! - Actual values are injected directly into browser via CDP
//!
//! # Format Specification
//!
//! - Prefix: `{{VAULT:` (required)
//! - Field: Secret field identifier (e.g., `payment.card_number`)
//! - Hash: Lowercase hexadecimal hash of the secret value
//! - Suffix: `}}` (required)
//!
//! Example: `{{VAULT:payment.card_number:a1b2c3d4e5f6}}`

use std::collections::HashMap;

/// Errors that can occur during placeholder parsing
#[derive(Debug, PartialEq, Eq, thiserror::Error)]
pub enum PlaceholderParseError {
    #[error("Invalid placeholder format: {0}")]
    InvalidFormat(String),

    #[error("Nested placeholders are not allowed: {0}")]
    NestedPlaceholder(String),

    #[error("Conditionals are not supported: {0}")]
    ConditionalNotSupported(String),

    #[error("Loops are not supported: {0}")]
    LoopNotSupported(String),

    #[error("Secret not found: {0}")]
    SecretNotFound(String),

    #[error("Hash must be lowercase hexadecimal: {0}")]
    InvalidHash(String),

    #[error("Field name cannot be empty")]
    EmptyFieldName,

    #[error("Hash cannot be empty")]
    EmptyHash,
}

/// Represents a parsed placeholder with field and hash components
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Placeholder {
    /// The field identifier (e.g., "payment.card_number")
    pub field: String,
    /// The lowercase hexadecimal hash of the secret value
    pub hash: String,
}

impl Placeholder {
    /// Creates the placeholder string in `{{VAULT:field:hash}}` format
    pub fn to_string(&self) -> String {
        format!("{{{{VAULT:{}:{}}}}}", self.field, self.hash)
    }
}

/// Parses a string and extracts all placeholders in `{{VAULT:field:hash}}` format
///
/// # Security Guarantees
///
/// - Only accepts `{{VAULT:field:hash}}` format
/// - Hash must be lowercase hexadecimal
/// - Field and hash cannot be empty
/// - No nested placeholders, conditionals, or loops
///
/// # Arguments
///
/// * `input` - The string containing placeholders
///
/// # Returns
///
/// A vector of Placeholder objects if parsing succeeds, or an error if format is invalid
///
/// # Examples
///
/// ```
/// use rust_vault::blindfill::placeholder::{parse_placeholders, Placeholder};
///
/// let input = "{{VAULT:payment.card_number:a1b2c3d4e5f6}}";
/// let result = parse_placeholders(input).unwrap();
/// assert_eq!(result[0].field, "payment.card_number");
/// assert_eq!(result[0].hash, "a1b2c3d4e5f6");
/// ```
pub fn parse_placeholders(input: &str) -> Result<Vec<Placeholder>, PlaceholderParseError> {
    let mut placeholders = Vec::new();
    let mut chars = input.chars().peekable();
    let mut pos = 0;

    while let Some(&ch) = chars.peek() {
        if ch == '{' {
            let start_pos = pos;
            chars.next();

            if chars.peek() == Some(&'{') {
                chars.next();
                pos += 2;

                // Check for VAULT: prefix (case-sensitive for security)
                let prefix: String = chars.by_ref().take(6).collect();
                if prefix != "VAULT:" {
                    if prefix.starts_with(' ') || prefix.starts_with('\t') || prefix.starts_with('\n') {
                        return Err(PlaceholderParseError::InvalidFormat(
                            "whitespace not allowed in placeholder".to_string()
                        ));
                    }
                    return Err(PlaceholderParseError::InvalidFormat(
                        format!("missing 'VAULT:' prefix at position {} (found: {})", start_pos, prefix)
                    ));
                }
                pos += 6;

                // Parse field:hash format
                let mut content = String::new();
                let mut found_closing = false;

                while let Some(&c) = chars.peek() {
                    if c == '}' {
                        chars.next();
                        pos += 1;

                        if chars.peek() == Some(&'}') {
                            chars.next();
                            pos += 1;
                            found_closing = true;
                            break;
                        }
                    }

                    content.push(c);
                    chars.next();
                    pos += 1;
                }

                if !found_closing {
                    return Err(PlaceholderParseError::InvalidFormat(
                        format!("missing closing delimiter '}}' at position {}", start_pos)
                    ));
                }

                // Split content into field:hash
                let parts: Vec<&str> = content.splitn(2, ':').collect();
                if parts.len() != 2 {
                    return Err(PlaceholderParseError::InvalidFormat(
                        format!("placeholder must be in format {{VAULT:field:hash}}, found: {{VAULT:{}}}", content)
                    ));
                }

                let field = parts[0].trim();
                let hash = parts[1].trim();

                // Validate field is not empty
                if field.is_empty() {
                    return Err(PlaceholderParseError::EmptyFieldName);
                }

                // Validate hash is not empty
                if hash.is_empty() {
                    return Err(PlaceholderParseError::EmptyHash);
                }

                // Validate field doesn't contain whitespace
                if field.contains(' ') || field.contains('\t') || field.contains('\n') {
                    return Err(PlaceholderParseError::InvalidFormat(
                        "whitespace not allowed in field name".to_string()
                    ));
                }

                // Validate field doesn't contain nested placeholders
                if field.contains("{{") || field.contains("}}") {
                    return Err(PlaceholderParseError::NestedPlaceholder(
                        "nested placeholders not allowed".to_string()
                    ));
                }

                // Validate hash is lowercase hexadecimal only
                if !hash.chars().all(|c| c.is_ascii_hexdigit() && c.is_ascii_lowercase()) {
                    return Err(PlaceholderParseError::InvalidHash(
                        format!("hash must be lowercase hexadecimal, found: {}", hash)
                    ));
                }

                // Validate hash doesn't contain nested placeholders
                if hash.contains("{{") || hash.contains("}}") {
                    return Err(PlaceholderParseError::NestedPlaceholder(
                        "nested placeholders not allowed".to_string()
                    ));
                }

                // Validate against conditionals and loops
                let lower_field = field.to_lowercase();
                if lower_field == "if" || lower_field == "else" || lower_field == "endif"
                    || lower_field.starts_with("if ") || lower_field.starts_with("else ") || lower_field.starts_with("endif ") {
                    return Err(PlaceholderParseError::ConditionalNotSupported(
                        "conditionals not supported".to_string()
                    ));
                }

                if lower_field == "for" || lower_field == "endfor"
                    || lower_field.starts_with("for ") || lower_field.starts_with("endfor ") {
                    return Err(PlaceholderParseError::LoopNotSupported(
                        "loops not supported".to_string()
                    ));
                }

                placeholders.push(Placeholder {
                    field: field.to_string(),
                    hash: hash.to_string(),
                });
            } else {
                pos += 1;
            }
        } else {
            chars.next();
            pos += 1;
        }
    }

    if input.contains("}}") && !input.contains("{{") {
        return Err(PlaceholderParseError::InvalidFormat(
            "missing opening delimiter".to_string()
        ));
    }

    Ok(placeholders)
}
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_valid_placeholder() {
        let input = "{{VAULT:payment.card_number:a1b2c3d4e5f6}}";
        let result = parse_placeholders(input);
        assert!(result.is_ok());
        let placeholders = result.unwrap();
        assert_eq!(placeholders[0].field, "payment.card_number");
        assert_eq!(placeholders[0].hash, "a1b2c3d4e5f6");
    }

    #[test]
    fn test_parse_multiple_placeholders() {
        let input = "{{VAULT:email:a1b2c3}} {{VAULT:password:d4e5f6a}}";
        let result = parse_placeholders(input);
        assert!(result.is_ok());
        let placeholders = result.unwrap();
        assert_eq!(placeholders.len(), 2);
        assert_eq!(placeholders[0].field, "email");
        assert_eq!(placeholders[0].hash, "a1b2c3");
        assert_eq!(placeholders[1].field, "password");
        assert_eq!(placeholders[1].hash, "d4e5f6a");
    }

    #[test]
    fn test_parse_no_placeholders() {
        let input = "just text";
        let result = parse_placeholders(input);
        assert!(result.is_ok());
        assert!(result.unwrap().is_empty());
    }

    #[test]
    fn test_error_missing_vault_prefix() {
        let input = "{{name}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::InvalidFormat(msg)) => {
                assert!(msg.contains("missing 'VAULT:' prefix"));
            }
            _ => panic!("Expected InvalidFormat error"),
        }
    }

    #[test]
    fn test_error_secret_prefix_rejected() {
        let input = "{{secret:payment.card_number}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::InvalidFormat(msg)) => {
                assert!(msg.contains("missing 'VAULT:' prefix"));
            }
            _ => panic!("Expected InvalidFormat error for old secret: format"),
        }
    }

    #[test]
    fn test_error_missing_hash() {
        let input = "{{VAULT:payment.card_number}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::InvalidFormat(msg)) => {
                assert!(msg.contains("must be in format {{VAULT:field:hash}}"));
            }
            _ => panic!("Expected InvalidFormat error"),
        }
    }

    #[test]
    fn test_error_empty_field() {
        let input = "{{VAULT::a1b2c3}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::EmptyFieldName) => {}
            _ => panic!("Expected EmptyFieldName error"),
        }
    }

    #[test]
    fn test_error_empty_hash() {
        let input = "{{VAULT:email:}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::EmptyHash) => {}
            _ => panic!("Expected EmptyHash error"),
        }
    }

    #[test]
    fn test_error_hash_uppercase() {
        let input = "{{VAULT:email:A1B2C3}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::InvalidHash(msg)) => {
                assert!(msg.contains("must be lowercase hexadecimal"));
            }
            _ => panic!("Expected InvalidHash error"),
        }
    }

    #[test]
    fn test_error_hash_invalid_chars() {
        let input = "{{VAULT:email:xyz123}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::InvalidHash(_)) => {}
            _ => panic!("Expected InvalidHash error"),
        }
    }

    #[test]
    fn test_error_nested_placeholder() {
        let input = "{{VAULT:{{other}}:abc}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::NestedPlaceholder(_)) => {}
            _ => panic!("Expected NestedPlaceholder error"),
        }
    }

    #[test]
    fn test_error_conditional() {
        let input = "{{VAULT:if:abc}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::ConditionalNotSupported(_)) => {}
            _ => panic!("Expected ConditionalNotSupported error"),
        }
    }

    #[test]
    fn test_error_loop() {
        let input = "{{VAULT:for:abc}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::LoopNotSupported(_)) => {}
            _ => panic!("Expected LoopNotSupported error"),
        }
    }

    #[test]
    fn test_replace_placeholders() {
        let input = "{{VAULT:email:a1b2c3}}";
        let mut secrets = HashMap::new();
        secrets.insert("email:a1b2c3".to_string(), "user@example.com".to_string());
        let placeholders = parse_placeholders(input).unwrap();

        let result = replace_placeholders(input, &placeholders, &secrets);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "user@example.com");
    }

    #[test]
    fn test_replace_missing_secret() {
        let input = "{{VAULT:email:a1b2c3}}";
        let secrets = HashMap::new();
        let placeholders = parse_placeholders(input).unwrap();

        let result = replace_placeholders(input, &placeholders, &secrets);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::SecretNotFound(key)) => {
                assert_eq!(key, "email:a1b2c3");
            }
            _ => panic!("Expected SecretNotFound error"),
        }
    }

    #[test]
    fn test_placeholder_to_string() {
        let placeholder = Placeholder {
            field: "payment.card_number".to_string(),
            hash: "a1b2c3d4e5f6".to_string(),
        };
        assert_eq!(placeholder.to_string(), "{{VAULT:payment.card_number:a1b2c3d4e5f6}}");
    }
}
