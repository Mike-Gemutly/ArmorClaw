//! Placeholder Parser Tests
//!
//! TDD tests for placeholder parsing functionality.
//! Tests cover all examples from Wave 0 and error scenarios.

use rust_vault::blindfill::placeholder::{
    parse_placeholders, replace_placeholders, Placeholder, PlaceholderParseError,
};
use std::collections::HashMap;

// ============================================================================
// Valid Placeholder Tests (Wave 0 Examples)
// ============================================================================

#[test]
fn test_replace_placeholders_simple() {
    let input = "{{VAULT:payment.card_number:a1b2c3d4}}";
    let mut secrets = HashMap::new();
    secrets.insert(
        "payment.card_number:a1b2c3d4".to_string(),
        "4242424242424242".to_string(),
    );

    let result = parse_placeholders(input);
    assert!(result.is_ok());

    let placeholders = result.unwrap();
    let output = replace_placeholders(input, &placeholders, &secrets).unwrap();

    assert_eq!(output, "4242424242424242");
}

#[test]
fn test_parse_payment_cvv() {
    let input = "{{secret:payment.cvv}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["payment.cvv"]);
}

#[test]
fn test_parse_payment_expiry_month() {
    let input = "{{secret:payment.expiry_month}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["payment.expiry_month"]);
}

#[test]
fn test_parse_payment_expiry_year() {
    let input = "{{secret:payment.expiry_year}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["payment.expiry_year"]);
}

#[test]
fn test_parse_billing_address_city() {
    let input = "{{secret:billing.address.city}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["billing.address.city"]);
}

#[test]
fn test_parse_billing_address_state() {
    let input = "{{secret:billing.address.state}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["billing.address.state"]);
}

#[test]
fn test_parse_billing_address_zip() {
    let input = "{{secret:billing.address.zip}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["billing.address.zip"]);
}

#[test]
fn test_parse_user_email() {
    let input = "{{secret:user.email}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["user.email"]);
}

#[test]
fn test_parse_user_phone() {
    let input = "{{secret:user.phone}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["user.phone"]);
}

#[test]
fn test_parse_bank_account_number() {
    let input = "{{secret:bank.account_number}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["bank.account_number"]);
}

#[test]
fn test_parse_bank_routing_number() {
    let input = "{{secret:bank.routing_number}}";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["bank.routing_number"]);
}

// ============================================================================
// Multiple Placeholders Tests
// ============================================================================

#[test]
fn test_replace_placeholders_multiple() {
    let input = "{{VAULT:payment.card_number:a1b2c3d4}} {{VAULT:payment.cvv:e5f6g7h8}}";
    let mut secrets = HashMap::new();
    secrets.insert(
        "payment.card_number:a1b2c3d4".to_string(),
        "4242424242424242".to_string(),
    );
    secrets.insert("payment.cvv:e5f6g7h8".to_string(), "123".to_string());

    let result = parse_placeholders(input);
    assert!(result.is_ok());

    let placeholders = result.unwrap();
    let output = replace_placeholders(input, &placeholders, &secrets).unwrap();

    assert_eq!(output, "4242424242424242 123");
}

#[test]
fn test_replace_placeholders_with_text() {
    let input = "Card: {{VAULT:payment.card_number:a1b2c3d4}}, CVV: {{VAULT:payment.cvv:e5f6g7h8}}";
    let mut secrets = HashMap::new();
    secrets.insert(
        "payment.card_number:a1b2c3d4".to_string(),
        "4242424242424242".to_string(),
    );
    secrets.insert("payment.cvv:e5f6g7h8".to_string(), "123".to_string());

    let result = parse_placeholders(input);
    assert!(result.is_ok());

    let placeholders = result.unwrap();
    let output = replace_placeholders(input, &placeholders, &secrets).unwrap();

    assert_eq!(output, "Card: 4242424242424242, CVV: 123");
}

#[test]
fn test_parse_html_form_with_placeholders() {
    let input = r#"<input name="card_number" value="{{secret:payment.card_number}}">"#;
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert_eq!(placeholders, vec!["payment.card_number"]);
}

// ============================================================================
// Error Cases - Invalid Format
// ============================================================================

#[test]
fn test_error_missing_opening_brace() {
    let input = "secret:name}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::InvalidFormat(msg)) => {
            assert!(msg.contains("missing opening delimiter"));
        }
        _ => panic!("Expected InvalidFormat error"),
    }
}

#[test]
fn test_error_missing_closing_brace() {
    let input = "{{secret:name";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::InvalidFormat(msg)) => {
            assert!(msg.contains("missing closing delimiter"));
        }
        _ => panic!("Expected InvalidFormat error"),
    }
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
fn test_error_empty_secret_name() {
    let input = "{{secret:}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::InvalidFormat(msg)) => {
            assert!(msg.contains("empty secret name"));
        }
        _ => panic!("Expected InvalidFormat error"),
    }
}

// ============================================================================
// Error Cases - Nested Placeholders (FORBIDDEN)
// ============================================================================

#[test]
fn test_error_nested_placeholder() {
    let input = "{{secret:{{other}}}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::NestedPlaceholder(msg)) => {
            assert!(msg.contains("nested placeholders not allowed"));
        }
        _ => panic!("Expected NestedPlaceholder error"),
    }
}

#[test]
fn test_error_deeply_nested_placeholder() {
    let input = "{{secret:{{secret:nested}}}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::NestedPlaceholder(msg)) => {
            assert!(msg.contains("nested placeholders not allowed"));
        }
        _ => panic!("Expected NestedPlaceholder error"),
    }
}

// ============================================================================
// Error Cases - Conditional Placeholders (FORBIDDEN)
// ============================================================================

#[test]
fn test_error_conditional_if() {
    let input = "{{secret:if}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::ConditionalNotSupported(msg)) => {
            assert!(msg.contains("conditionals not supported"));
        }
        _ => panic!("Expected ConditionalNotSupported error"),
    }
}

#[test]
fn test_error_conditional_else() {
    let input = "{{secret:else}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::ConditionalNotSupported(msg)) => {
            assert!(msg.contains("conditionals not supported"));
        }
        _ => panic!("Expected ConditionalNotSupported error"),
    }
}

#[test]
fn test_error_conditional_endif() {
    let input = "{{secret:endif}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::ConditionalNotSupported(msg)) => {
            assert!(msg.contains("conditionals not supported"));
        }
        _ => panic!("Expected ConditionalNotSupported error"),
    }
}

// ============================================================================
// Error Cases - Loops (FORBIDDEN)
// ============================================================================

#[test]
fn test_error_loop_for() {
    let input = "{{secret:for}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::LoopNotSupported(msg)) => {
            assert!(msg.contains("loops not supported"));
        }
        _ => panic!("Expected LoopNotSupported error"),
    }
}

#[test]
fn test_error_loop_endfor() {
    let input = "{{secret:endfor}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::LoopNotSupported(msg)) => {
            assert!(msg.contains("loops not supported"));
        }
        _ => panic!("Expected LoopNotSupported error"),
    }
}

// ============================================================================
// Error Cases - Whitespace (STRICT - no whitespace allowed)
// ============================================================================

#[test]
fn test_error_whitespace_after_opening_brace() {
    let input = "{{ secret:name}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::InvalidFormat(msg)) => {
            assert!(msg.contains("whitespace not allowed"));
        }
        _ => panic!("Expected InvalidFormat error"),
    }
}

#[test]
fn test_error_whitespace_before_closing_brace() {
    let input = "{{secret:name }}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::InvalidFormat(msg)) => {
            assert!(msg.contains("whitespace not allowed"));
        }
        _ => panic!("Expected InvalidFormat error"),
    }
}

#[test]
fn test_error_whitespace_in_secret_name() {
    let input = "{{secret:name with spaces}}";
    let result = parse_placeholders(input);
    assert!(result.is_err());
    match result {
        Err(PlaceholderParseError::InvalidFormat(msg)) => {
            assert!(msg.contains("whitespace not allowed"));
        }
        _ => panic!("Expected InvalidFormat error"),
    }
}

// ============================================================================
// No Placeholders Tests
// ============================================================================

#[test]
fn test_no_placeholders() {
    let input = "Just plain text without any placeholders";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert!(placeholders.is_empty());
}

#[test]
fn test_empty_string() {
    let input = "";
    let result = parse_placeholders(input);
    assert!(result.is_ok());
    let placeholders = result.unwrap();
    assert!(placeholders.is_empty());
}

// ============================================================================
// Replacement Function Tests
// ============================================================================

#[test]
fn test_replace_placeholders_simple() {
    let input = "{{VAULT:payment.card_number:a1b2c3d4}}";
    let mut secrets = HashMap::new();
    secrets.insert(
        "payment.card_number:a1b2c3d4".to_string(),
        "4242424242424242".to_string(),
    );

    let result = parse_placeholders(input);
    assert!(result.is_ok());

    let placeholders = result.unwrap();
    let output = replace_placeholders(input, &placeholders, &secrets).unwrap();

    assert_eq!(output, "4242424242424242");
}

#[test]
fn test_replace_placeholders_multiple() {
    let input = "{{VAULT:payment.card_number:a1b2c3d4}} {{VAULT:payment.cvv:e5f6g7h8}}";
    let mut secrets = HashMap::new();
    secrets.insert(
        "payment.card_number:a1b2c3d4".to_string(),
        "4242424242424242".to_string(),
    );
    secrets.insert("payment.cvv:e5f6g7h8".to_string(), "123".to_string());

    let result = parse_placeholders(input);
    assert!(result.is_ok());

    let placeholders = result.unwrap();
    let output = replace_placeholders(input, &placeholders, &secrets).unwrap();

    assert_eq!(output, "4242424242424242 123");
}

#[test]
fn test_replace_placeholder_missing_secret() {
    let input = "{{VAULT:payment.card_number:a1b2c3d4}}";
    let secrets = HashMap::new();

    let result = parse_placeholders(input);
    assert!(result.is_ok());

    let placeholders = result.unwrap();
    let output = replace_placeholders(input, &placeholders, &secrets);

    assert!(output.is_err());
    match output {
        Err(PlaceholderParseError::SecretNotFound(secret)) => {
            assert_eq!(secret, "payment.card_number:a1b2c3d4");
        }
        _ => panic!("Expected SecretNotFound error"),
    }
}

#[test]
fn test_replace_placeholders_with_text() {
    let input = "Card: {{VAULT:payment.card_number:a1b2c3d4}}, CVV: {{VAULT:payment.cvv:e5f6g7h8}}";
    let mut secrets = HashMap::new();
    secrets.insert(
        "payment.card_number:a1b2c3d4".to_string(),
        "4242424242424242".to_string(),
    );
    secrets.insert("payment.cvv:e5f6g7h8".to_string(), "123".to_string());

    let result = parse_placeholders(input);
    assert!(result.is_ok());

    let placeholders = result.unwrap();
    let output = replace_placeholders(input, &placeholders, &secrets).unwrap();

    assert_eq!(output, "Card: 4242424242424242, CVV: 123");
}
