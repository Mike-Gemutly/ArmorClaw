use hmac::{Hmac, Mac};
use sha2::Sha256;
use zeroize::Zeroizing;

type HmacSha256 = Hmac<Sha256>;

/// HMAC-SHA256 provenance signer with zeroizing key storage.
///
/// Produces 8-byte truncated signatures for lightweight provenance tracking.
pub struct ProvenanceSigner {
    key: Zeroizing<Vec<u8>>,
}

impl ProvenanceSigner {
    /// Create a new signer. Key must be at least 32 bytes.
    pub fn new(key: &[u8]) -> Self {
        assert!(key.len() >= 32, "HMAC key must be at least 32 bytes");
        Self {
            key: Zeroizing::new(key.to_vec()),
        }
    }

    pub fn generate_signature(&self, data: &[u8]) -> String {
        let mut mac = HmacSha256::new_from_slice(&self.key).expect("HMAC accepts any key length");
        mac.update(data);
        let result = mac.finalize().into_bytes();
        hex::encode(&result[..8])
    }

    pub fn verify_signature(&self, data: &[u8], signature: &str) -> bool {
        let expected = self.generate_signature(data);
        constant_time_compare(expected.as_bytes(), signature.as_bytes())
    }
}

pub fn format_provenance(signature: &str, session_id: &str) -> String {
    format!(
        "[Provenance: AC-v6-Sig:{} | Sess:{}]",
        signature, session_id
    )
}

/// Constant-time byte comparison to prevent timing attacks.
fn constant_time_compare(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    let mut result: u8 = 0;
    for (x, y) in a.iter().zip(b.iter()) {
        result |= x ^ y;
    }
    result == 0
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_key() -> Vec<u8> {
        vec![0u8; 32]
    }

    #[test]
    fn test_sign_and_verify() {
        let signer = ProvenanceSigner::new(&make_key());
        let data = b"hello world";
        let sig = signer.generate_signature(data);
        assert!(signer.verify_signature(data, &sig));
    }

    #[test]
    fn test_tampered_data_fails() {
        let signer = ProvenanceSigner::new(&make_key());
        let data = b"hello world";
        let sig = signer.generate_signature(data);
        assert!(!signer.verify_signature(b"hello world!", &sig));
    }

    #[test]
    fn test_wrong_key_fails() {
        let signer_a = ProvenanceSigner::new(&make_key());
        let mut key_b = make_key();
        key_b[0] = 1;
        let signer_b = ProvenanceSigner::new(&key_b);
        let data = b"hello world";
        let sig = signer_a.generate_signature(data);
        assert!(!signer_b.verify_signature(data, &sig));
    }

    #[test]
    fn test_format_provenance() {
        let sig = "abcdef0123456789";
        let session_id = "sess-123";
        let formatted = format_provenance(sig, session_id);
        assert_eq!(
            formatted,
            "[Provenance: AC-v6-Sig:abcdef0123456789 | Sess:sess-123]"
        );
    }

    #[test]
    fn test_deterministic() {
        let signer = ProvenanceSigner::new(&make_key());
        let data = b"deterministic test data";
        let sig1 = signer.generate_signature(data);
        let sig2 = signer.generate_signature(data);
        assert_eq!(sig1, sig2);
        assert_eq!(sig1.len(), 16);
    }
}
