use armorclaw_sidecar::security::shadowmap::{PiiCategory, ShadowMap};

#[test]
fn test_redact_email() {
    let mut sm = ShadowMap::new();
    let input = "Contact us at admin@example.com for support.";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("admin@example.com"));
    assert!(redacted.contains("[PII_"));
    assert!(sm.map().len() == 1);
}

#[test]
fn test_redact_ssn() {
    let mut sm = ShadowMap::new();
    let input = "SSN: 123-45-6789";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("123-45-6789"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_credit_card() {
    let mut sm = ShadowMap::new();
    let input = "Card: 4111 1111 1111 1111";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("4111 1111 1111 1111"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_credit_card_amex() {
    let mut sm = ShadowMap::new();
    let input = "Amex: 3782-822463-10005";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("3782-822463-10005"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_phone() {
    let mut sm = ShadowMap::new();
    let input = "Call (555) 123-4567 now";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("(555) 123-4567"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_ip_address() {
    let mut sm = ShadowMap::new();
    let input = "Server at 192.168.1.100 is down";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("192.168.1.100"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_ip_no_version_false_positive() {
    let mut sm = ShadowMap::new();
    let input = "Version 0.0.0 is the first release";
    let redacted = sm.redact(input);
    assert!(redacted.contains("0.0.0"));
}

#[test]
fn test_redact_api_key() {
    let mut sm = ShadowMap::new();
    let input = "Use key sk-abc123def456 for access";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("sk-abc123def456"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_bearer_token() {
    let mut sm = ShadowMap::new();
    let input = "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.test.signature";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("eyJhbGciOiJIUzI1NiJ9.test.signature"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_password_in_url() {
    let mut sm = ShadowMap::new();
    let input = "Connect with password=s3cretP@ss in config";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("s3cretP@ss"));
    assert!(redacted.contains("[PII_"));
}

#[test]
fn test_redact_multiple_pii_types() {
    let mut sm = ShadowMap::new();
    let input = "User john@example.com has SSN 987-65-4321 and card 5500-0000-0000-0004";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("john@example.com"));
    assert!(!redacted.contains("987-65-4321"));
    assert!(!redacted.contains("5500-0000-0000-0004"));
    assert!(sm.map().len() >= 3);
}

#[test]
fn test_redact_no_pii() {
    let mut sm = ShadowMap::new();
    let input = "Hello world, this is plain text.";
    let redacted = sm.redact(input);
    assert_eq!(redacted, input);
    assert!(sm.map().is_empty());
}

#[test]
fn test_redact_empty_string() {
    let mut sm = ShadowMap::new();
    let redacted = sm.redact("");
    assert_eq!(redacted, "");
}

#[test]
fn test_unredact_restores_originals() {
    let mut sm = ShadowMap::new();
    let input = "Email john@example.com and SSN 123-45-6789";
    let redacted = sm.redact(input);
    assert_ne!(redacted, input);

    let restored = sm.unredact(&redacted);
    assert_eq!(restored, input);
}

#[test]
fn test_bidirectional_map() {
    let mut sm = ShadowMap::new();
    let input = "Contact admin@example.com";
    let redacted = sm.redact(input);

    let token = redacted
        .split_whitespace()
        .find(|w| w.starts_with("[PII_"))
        .unwrap();

    let original = sm.unredact_token(token).unwrap();
    assert_eq!(original, "admin@example.com");
}

#[test]
fn test_unredact_token_not_found() {
    let sm = ShadowMap::new();
    let result = sm.unredact_token("[PII_999]");
    assert!(result.is_none());
}

#[test]
fn test_clear_map() {
    let mut sm = ShadowMap::new();
    let _ = sm.redact("admin@example.com");
    assert!(!sm.map().is_empty());
    sm.clear();
    assert!(sm.map().is_empty());
}

#[test]
fn test_pii_categories() {
    let mut sm = ShadowMap::new();
    let input = "admin@example.com";
    let redacted = sm.redact(input);
    let token = redacted
        .split_whitespace()
        .find(|w| w.starts_with("[PII_"))
        .unwrap();
    let category = sm.category(token).unwrap();
    assert_eq!(category, PiiCategory::Email);
}

#[test]
fn test_counter_increments() {
    let mut sm = ShadowMap::new();
    let r1 = sm.redact("user1@test.com");
    let r2 = sm.redact("user2@test.com");
    assert_ne!(r1, r2);
    assert!(sm.map().len() == 2);
}

#[test]
fn test_phone_not_misclassified_as_ssn() {
    let mut sm = ShadowMap::new();
    let input = "Phone: 555-123-4567";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("555-123-4567"));
    assert!(redacted.contains("[PII_"));
    let token = redacted
        .split_whitespace()
        .find(|w| w.starts_with("[PII_"))
        .unwrap();
    assert_eq!(sm.category(token).unwrap(), PiiCategory::Phone);
}

#[test]
fn test_ssn_still_detected_after_phone() {
    let mut sm = ShadowMap::new();
    let input = "SSN: 123-45-6789 is valid";
    let redacted = sm.redact(input);
    assert!(!redacted.contains("123-45-6789"));
    let token = redacted
        .split_whitespace()
        .find(|w| w.starts_with("[PII_"))
        .unwrap();
    assert_eq!(sm.category(token).unwrap(), PiiCategory::Ssn);
}

#[test]
fn test_contains_pii() {
    assert!(ShadowMap::contains_pii("test@example.com"));
    assert!(ShadowMap::contains_pii("123-45-6789"));
    assert!(!ShadowMap::contains_pii("Hello world"));
}

#[test]
fn test_token_count() {
    let mut sm = ShadowMap::new();
    assert_eq!(sm.token_count(), 0);
    sm.redact("john@example.com and 123-45-6789");
    assert_eq!(sm.token_count(), 2);
}
