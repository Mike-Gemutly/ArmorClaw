use hmac::{Hmac, Mac};
use sha2::Sha256;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

type HmacSha256 = Hmac<Sha256>;

pub const TOKEN_TTL_SECONDS: u64 = 1800; // 30 minutes
pub const MAX_TIMESTAMP_AGE_SECONDS: u64 = 300; // 5 minutes

#[derive(Debug, Clone, PartialEq)]
pub struct TokenInfo {
    pub request_id: String,
    pub timestamp: i64,
    pub operation: String,
    pub signature: String,
    pub expiration: i64,
}

#[derive(Debug, thiserror::Error)]
pub enum TokenError {
    #[error("invalid token format: expected 4 parts, got {0}")]
    InvalidFormat(usize),
    #[error("invalid timestamp: {0}")]
    InvalidTimestamp(#[from] std::num::ParseIntError),
    #[error("token timestamp is too old (> {0:?})")]
    TokenTooOld(Duration),
    #[error("token has expired (TTL: {0:?})")]
    TokenExpired(Duration),
    #[error("invalid token signature")]
    InvalidSignature,
    #[error("HMAC error: {0}")]
    HmacError(String),
}

/// Parses a token into its components.
/// Token format: {request_id}:{timestamp}:{operation}:{signature}
pub fn parse_token(token: &str) -> Result<TokenInfo, TokenError> {
    let parts: Vec<&str> = token.split(':').collect();

    if parts.len() != 4 {
        return Err(TokenError::InvalidFormat(parts.len()));
    }

    let timestamp = parts[1].parse::<i64>()?;
    let expiration = timestamp + TOKEN_TTL_SECONDS as i64;

    Ok(TokenInfo {
        request_id: parts[0].to_string(),
        timestamp,
        operation: parts[2].to_string(),
        signature: parts[3].to_string(),
        expiration,
    })
}

/// Validates a token's signature using HMAC-SHA256.
pub fn validate_token_signature(
    token: &str,
    shared_secret: &[u8],
) -> Result<bool, TokenError> {
    let info = parse_token(token)?;

    let data_to_sign = format!("{}{}{}", info.request_id, info.timestamp, info.operation);

    let mut mac = HmacSha256::new_from_slice(shared_secret)
        .map_err(|e| TokenError::HmacError(e.to_string()))?;

    mac.update(data_to_sign.as_bytes());

    let expected_signature = hex::encode(mac.finalize().into_bytes());

    Ok(info.signature == expected_signature)
}

/// Checks if a token has expired based on its TTL.
pub fn is_token_expired(info: &TokenInfo) -> bool {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    now > info.expiration
}

/// Checks if a token's timestamp is too old (beyond MAX_TIMESTAMP_AGE_SECONDS).
pub fn is_token_too_old(info: &TokenInfo) -> bool {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let age = now - info.timestamp;
    age > MAX_TIMESTAMP_AGE_SECONDS as i64
}

/// Performs full validation of a token including signature, expiration, and age.
pub fn validate_token(
    token: &str,
    shared_secret: &[u8],
) -> Result<TokenInfo, TokenError> {
    let info = parse_token(token)?;

    // Check if token is too old (timestamp > 5 minutes ago)
    if is_token_too_old(&info) {
        return Err(TokenError::TokenTooOld(Duration::from_secs(
            MAX_TIMESTAMP_AGE_SECONDS,
        )));
    }

    // Check if token has expired (TTL exceeded)
    if is_token_expired(&info) {
        return Err(TokenError::TokenExpired(Duration::from_secs(TOKEN_TTL_SECONDS)));
    }

    // Verify signature
    let valid = validate_token_signature(token, shared_secret)?;
    if !valid {
        return Err(TokenError::InvalidSignature);
    }

    Ok(info)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_token_valid() {
        let token = "test-request-id:1234567890:test-operation:abcdef123456";
        let result = parse_token(token);

        assert!(result.is_ok());
        let info = result.unwrap();
        assert_eq!(info.request_id, "test-request-id");
        assert_eq!(info.timestamp, 1234567890);
        assert_eq!(info.operation, "test-operation");
        assert_eq!(info.signature, "abcdef123456");
    }

    #[test]
    fn test_parse_token_invalid_format() {
        let token = "invalid:token";
        let result = parse_token(token);

        assert!(result.is_err());
        match result {
            Err(TokenError::InvalidFormat(n)) => assert_eq!(n, 2),
            _ => panic!("Expected InvalidFormat error"),
        }
    }

    #[test]
    fn test_parse_token_invalid_timestamp() {
        let token = "test-request-id:not-a-number:test-operation:abcdef";
        let result = parse_token(token);

        assert!(result.is_err());
        assert!(matches!(result, Err(TokenError::InvalidTimestamp(_))));
    }

    #[test]
    fn test_validate_token_signature_valid() {
        let shared_secret = b"test-secret-key-32-bytes-long!";
        let request_id = "test-request-id";
        let timestamp = 1234567890;
        let operation = "test-operation";

        let data_to_sign = format!("{}{}{}", request_id, timestamp, operation);

        let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
        mac.update(data_to_sign.as_bytes());
        let signature = hex::encode(mac.finalize().into_bytes());

        let token = format!("{}:{}:{}:{}", request_id, timestamp, operation, signature);

        let result = validate_token_signature(&token, shared_secret);
        assert!(result.is_ok());
        assert!(result.unwrap());
    }

    #[test]
    fn test_validate_token_signature_invalid() {
        let shared_secret = b"test-secret-key-32-bytes-long!";
        let token = "test-request-id:1234567890:test-operation:invalid-signature";

        let result = validate_token_signature(token, shared_secret);
        assert!(result.is_ok());
        assert!(!result.unwrap());
    }

    #[test]
    fn test_is_token_expired_true() {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;

        let info = TokenInfo {
            request_id: "test".to_string(),
            timestamp: now - TOKEN_TTL_SECONDS as i64 - 100,
            operation: "test".to_string(),
            signature: "test".to_string(),
            expiration: now - 100,
        };

        assert!(is_token_expired(&info));
    }

    #[test]
    fn test_is_token_expired_false() {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;

        let info = TokenInfo {
            request_id: "test".to_string(),
            timestamp: now,
            operation: "test".to_string(),
            signature: "test".to_string(),
            expiration: now + TOKEN_TTL_SECONDS as i64,
        };

        assert!(!is_token_expired(&info));
    }

    #[test]
    fn test_is_token_too_old_true() {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;

        let info = TokenInfo {
            request_id: "test".to_string(),
            timestamp: now - MAX_TIMESTAMP_AGE_SECONDS as i64 - 100,
            operation: "test".to_string(),
            signature: "test".to_string(),
            expiration: now + 1000,
        };

        assert!(is_token_too_old(&info));
    }

    #[test]
    fn test_is_token_too_old_false() {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;

        let info = TokenInfo {
            request_id: "test".to_string(),
            timestamp: now,
            operation: "test".to_string(),
            signature: "test".to_string(),
            expiration: now + 1000,
        };

        assert!(!is_token_too_old(&info));
    }

    #[test]
    fn test_validate_token_success() {
        let shared_secret = b"test-secret-key-32-bytes-long!";
        let request_id = "test-request-id";
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        let timestamp = now;
        let operation = "test-operation";

        let data_to_sign = format!("{}{}{}", request_id, timestamp, operation);

        let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
        mac.update(data_to_sign.as_bytes());
        let signature = hex::encode(mac.finalize().into_bytes());

        let token = format!("{}:{}:{}:{}", request_id, timestamp, operation, signature);

        let result = validate_token(&token, shared_secret);
        assert!(result.is_ok());
    }

    #[test]
    fn test_validate_token_too_old() {
        let shared_secret = b"test-secret-key-32-bytes-long!";
        let request_id = "test-request-id";
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        let timestamp = now - MAX_TIMESTAMP_AGE_SECONDS as i64 - 100;
        let operation = "test-operation";

        let data_to_sign = format!("{}{}{}", request_id, timestamp, operation);

        let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
        mac.update(data_to_sign.as_bytes());
        let signature = hex::encode(mac.finalize().into_bytes());

        let token = format!("{}:{}:{}:{}", request_id, timestamp, operation, signature);

        let result = validate_token(&token, shared_secret);
        assert!(result.is_err());
        assert!(matches!(result, Err(TokenError::TokenTooOld(_))));
    }

    #[test]
    fn test_validate_token_expired() {
        let shared_secret = b"test-secret-key-32-bytes-long!";
        let request_id = "test-request-id";
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        let timestamp = now - TOKEN_TTL_SECONDS as i64 - 100;
        let operation = "test-operation";

        let data_to_sign = format!("{}{}{}", request_id, timestamp, operation);

        let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
        mac.update(data_to_sign.as_bytes());
        let signature = hex::encode(mac.finalize().into_bytes());

        let token = format!("{}:{}:{}:{}", request_id, timestamp, operation, signature);

        let result = validate_token(&token, shared_secret);
        assert!(result.is_err());
        assert!(matches!(result, Err(TokenError::TokenExpired(_))));
    }

    #[test]
    fn test_validate_token_invalid_signature() {
        let shared_secret = b"test-secret-key-32-bytes-long!";
        let request_id = "test-request-id";
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        let timestamp = now;
        let operation = "test-operation";

        let token = format!("{}:{}:{}:{}", request_id, timestamp, operation, "invalid-signature");

        let result = validate_token(&token, shared_secret);
        assert!(result.is_err());
        assert!(matches!(result, Err(TokenError::InvalidSignature)));
    }
}
