//! Placeholder Parser for BlindFill™ Secret Injection
//!
//! Parses placeholders in the format `{{secret:name}}` for flat secret lookups.

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
}

/// Parses a string and extracts all placeholder names in `{{secret:name}}` format
///
/// # Arguments
///
/// * `input` - The string containing placeholders
///
/// # Returns
///
/// A vector of secret names if parsing succeeds, or an error if format is invalid
///
/// # Examples
///
/// ```
/// use rust_vault::blindfill::placeholder::parse_placeholders;
///
/// let input = "{{secret:payment.card_number}}";
/// let result = parse_placeholders(input).unwrap();
/// assert_eq!(result, vec!["payment.card_number"]);
/// ```
pub fn parse_placeholders(input: &str) -> Result<Vec<String>, PlaceholderParseError> {
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

                let prefix: String = chars.by_ref().take(7).collect();
                if prefix != "secret:" {
                    if prefix.starts_with(' ') || prefix.starts_with('\t') || prefix.starts_with('\n') {
                        return Err(PlaceholderParseError::InvalidFormat(
                            "whitespace not allowed in placeholder".to_string()
                        ));
                    }
                    return Err(PlaceholderParseError::InvalidFormat(
                        format!("missing 'secret:' prefix at position {}", start_pos)
                    ));
                }
                pos += 7;

                let mut secret_name = String::new();
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

                    secret_name.push(c);
                    chars.next();
                    pos += 1;
                }

                if !found_closing {
                    return Err(PlaceholderParseError::InvalidFormat(
                        format!("missing closing delimiter '}}' at position {}", start_pos)
                    ));
                }

                if secret_name.is_empty() {
                    return Err(PlaceholderParseError::InvalidFormat(
                        format!("empty secret name at position {}", start_pos)
                    ));
                }

                if secret_name.contains(' ') || secret_name.contains('\t') || secret_name.contains('\n') {
                    return Err(PlaceholderParseError::InvalidFormat(
                        "whitespace not allowed in placeholder".to_string()
                    ));
                }

                if secret_name.contains("{{") {
                    return Err(PlaceholderParseError::NestedPlaceholder(
                        "nested placeholders not allowed".to_string()
                    ));
                }

                let lower_name = secret_name.to_lowercase();
                if lower_name == "if" || lower_name == "else" || lower_name == "endif"
                    || lower_name.starts_with("if ") || lower_name.starts_with("else ") || lower_name.starts_with("endif ") {
                    return Err(PlaceholderParseError::ConditionalNotSupported(
                        "conditionals not supported".to_string()
                    ));
                }

                if lower_name == "for" || lower_name == "endfor"
                    || lower_name.starts_with("for ") || lower_name.starts_with("endfor ") {
                    return Err(PlaceholderParseError::LoopNotSupported(
                        "loops not supported".to_string()
                    ));
                }

                placeholders.push(secret_name);
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

/// Replaces placeholders in the input string with secret values
///
/// # Arguments
///
/// * `input` - The string containing placeholders
/// * `placeholders` - List of placeholder names to replace
/// * `secrets` - Map of secret names to their values
///
/// # Returns
///
/// The string with placeholders replaced, or an error if a secret is not found
pub fn replace_placeholders(
    input: &str,
    placeholders: &[String],
    secrets: &HashMap<String, String>,
) -> Result<String, PlaceholderParseError> {
    let mut output = input.to_string();

    for placeholder in placeholders {
        if let Some(secret) = secrets.get(placeholder) {
            let placeholder_str = format!("{{{{secret:{}}}}}", placeholder);
            output = output.replace(&placeholder_str, secret);
        } else {
            return Err(PlaceholderParseError::SecretNotFound(placeholder.clone()));
        }
    }

    Ok(output)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_valid_placeholder() {
        let input = "{{secret:payment.card_number}}";
        let result = parse_placeholders(input);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), vec!["payment.card_number"]);
    }

    #[test]
    fn test_parse_multiple_placeholders() {
        let input = "{{secret:a}} {{secret:b}}";
        let result = parse_placeholders(input);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), vec!["a", "b"]);
    }

    #[test]
    fn test_parse_no_placeholders() {
        let input = "just text";
        let result = parse_placeholders(input);
        assert!(result.is_ok());
        assert!(result.unwrap().is_empty());
    }

    #[test]
    fn test_error_missing_secret_prefix() {
        let input = "{{name}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::InvalidFormat(msg)) => {
                assert!(msg.contains("missing 'secret:' prefix"));
            }
            _ => panic!("Expected InvalidFormat error"),
        }
    }

    #[test]
    fn test_error_nested_placeholder() {
        let input = "{{secret:{{other}}}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::NestedPlaceholder(_)) => {}
            _ => panic!("Expected NestedPlaceholder error"),
        }
    }

    #[test]
    fn test_error_conditional() {
        let input = "{{secret:if}}";
        let result = parse_placeholders(input);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::ConditionalNotSupported(_)) => {}
            _ => panic!("Expected ConditionalNotSupported error"),
        }
    }

    #[test]
    fn test_replace_placeholders() {
        let input = "{{secret:a}}";
        let mut secrets = HashMap::new();
        secrets.insert("a".to_string(), "value".to_string());
        let placeholders = vec!["a".to_string()];

        let result = replace_placeholders(input, &placeholders, &secrets);
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), "value");
    }

    #[test]
    fn test_replace_missing_secret() {
        let input = "{{secret:a}}";
        let secrets = HashMap::new();
        let placeholders = vec!["a".to_string()];

        let result = replace_placeholders(input, &placeholders, &secrets);
        assert!(result.is_err());
        match result {
            Err(PlaceholderParseError::SecretNotFound(secret)) => {
                assert_eq!(secret, "a");
            }
            _ => panic!("Expected SecretNotFound error"),
        }
    }
}
