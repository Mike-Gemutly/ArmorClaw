use chacha20poly1305::aead::{Aead, NewAead};
use chacha20poly1305::{Key, XChaCha20Poly1305, XNonce};
use hmac::{Hmac, Mac};
use sha2::Sha256;
use zeroize::Zeroize;
use zeroize::Zeroizing;

use crate::error::{Result, SidecarError};

type HmacSha256 = Hmac<Sha256>;

/// Current encryption format version.
pub const VERSION: u8 = 0x01;

/// Nonce length for XChaCha20Poly1305 (192-bit / 24 bytes).
pub const NONCE_LEN: usize = 24;

/// Fixed HMAC message for deterministic nonce derivation.
const NONCE_DERIVATION_MESSAGE: &[u8] = b"armorclaw-xchacha-nonce-v1";

/// AEAD cipher using XChaCha20-Poly1305 with deterministic nonce derivation.
///
/// The nonce is derived deterministically via HMAC-SHA256(key_id || blob_id, message)
/// to ensure the same blob always encrypts to the same ciphertext (idempotent encryption).
pub struct AeadCipher {
    key: [u8; 32],
}

impl AeadCipher {
    /// Create a new cipher with the given 32-byte key.
    pub fn new(key: [u8; 32]) -> Self {
        Self { key }
    }

    /// Derive a deterministic 24-byte nonce from key_id and blob_id.
    ///
    /// Uses HMAC-SHA256 with key = key_id || blob_id and message = NONCE_DERIVATION_MESSAGE.
    /// The first 24 bytes of the HMAC output are used as the XChaCha20 nonce.
    fn derive_nonce(key_id: &[u8], blob_id: &[u8]) -> XNonce {
        let hmac_key: Vec<u8> = [key_id, blob_id].concat();
        let mut mac = HmacSha256::new_from_slice(&hmac_key).expect("HMAC accepts any key size");
        mac.update(NONCE_DERIVATION_MESSAGE);
        let result = mac.finalize().into_bytes();
        let mut nonce_bytes = [0u8; NONCE_LEN];
        nonce_bytes.copy_from_slice(&result[..NONCE_LEN]);
        XNonce::from(nonce_bytes)
    }

    /// Encrypt plaintext with a deterministic nonce derived from key_id and blob_id.
    ///
    /// Returns `(output, version)` where output format is:
    /// `[version: 1 byte][nonce: 24 bytes][ciphertext + tag: variable]`
    pub fn encrypt(
        &self,
        plaintext: &[u8],
        key_id: &[u8],
        blob_id: &[u8],
    ) -> Result<(Vec<u8>, u8)> {
        let nonce = Self::derive_nonce(key_id, blob_id);
        let key = Key::from_slice(&self.key);
        let cipher = XChaCha20Poly1305::new(key);
        let ciphertext = cipher
            .encrypt(&nonce, plaintext)
            .map_err(|e| SidecarError::InternalError(format!("encryption failed: {}", e)))?;

        let mut output = Vec::with_capacity(1 + NONCE_LEN + ciphertext.len());
        output.push(VERSION);
        output.extend_from_slice(&nonce);
        output.extend_from_slice(&ciphertext);

        Ok((output, VERSION))
    }

    /// Decrypt ciphertext that was encrypted with the given key.
    ///
    /// Expects input format: `[version: 1 byte][nonce: 24 bytes][ciphertext + tag: variable]`
    ///
    /// Returns plaintext wrapped in `Zeroizing<Vec<u8>>` for secure memory handling.
    pub fn decrypt(ciphertext: &[u8], key: &[u8]) -> Result<Zeroizing<Vec<u8>>> {
        if ciphertext.len() < 1 + NONCE_LEN + 16 {
            return Err(SidecarError::InternalError(
                "ciphertext too short".to_string(),
            ));
        }

        let version = ciphertext[0];
        if version != VERSION {
            return Err(SidecarError::InternalError(format!(
                "unsupported version: {}",
                version
            )));
        }

        let nonce = XNonce::from_slice(&ciphertext[1..1 + NONCE_LEN]);
        let ct = &ciphertext[1 + NONCE_LEN..];

        let key = Key::from_slice(key);
        let cipher = XChaCha20Poly1305::new(key);
        let plaintext = cipher
            .decrypt(nonce, ct)
            .map_err(|e| SidecarError::InternalError(format!("decryption failed: {}", e)))?;

        Ok(Zeroizing::new(plaintext))
    }
}

impl Drop for AeadCipher {
    fn drop(&mut self) {
        self.key.zeroize();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_key() -> [u8; 32] {
        let mut key = [0u8; 32];
        key.copy_from_slice(b"test-key-00000000000000000000000");
        key
    }

    fn extract_nonce(output: &[u8]) -> &[u8] {
        &output[1..1 + NONCE_LEN]
    }

    #[test]
    fn test_roundtrip_encrypt_decrypt() {
        let cipher = AeadCipher::new(test_key());
        let plaintext = b"Hello, ArmorClaw!";
        let key_id = b"key-001";
        let blob_id = b"blob-abc";

        let (encrypted, version) = cipher.encrypt(plaintext, key_id, blob_id).unwrap();
        let decrypted = AeadCipher::decrypt(&encrypted, &test_key()).unwrap();

        assert_eq!(&*decrypted, plaintext);
        assert_eq!(version, VERSION);
    }

    #[test]
    fn test_nonce_is_24_bytes() {
        let cipher = AeadCipher::new(test_key());
        let (encrypted, _) = cipher
            .encrypt(b"test data", b"key-001", b"blob-001")
            .unwrap();

        let nonce = extract_nonce(&encrypted);
        assert_eq!(nonce.len(), NONCE_LEN, "nonce must be exactly 24 bytes");
    }

    #[test]
    fn test_ciphertext_differs_from_plaintext() {
        let cipher = AeadCipher::new(test_key());
        let plaintext = b"sensitive data that should be hidden";
        let (encrypted, _) = cipher.encrypt(plaintext, b"key-001", b"blob-001").unwrap();

        assert_ne!(encrypted, plaintext.to_vec());
    }

    #[test]
    fn test_tamper_detection() {
        let cipher = AeadCipher::new(test_key());
        let plaintext = b"untampered message";
        let (mut encrypted, _) = cipher.encrypt(plaintext, b"key-001", b"blob-001").unwrap();

        // Flip a byte in the ciphertext portion (after version + nonce)
        let ct_offset = 1 + NONCE_LEN;
        encrypted[ct_offset] ^= 0xFF;

        let result = AeadCipher::decrypt(&encrypted, &test_key());
        assert!(
            result.is_err(),
            "tampered ciphertext should fail to decrypt"
        );
    }

    #[test]
    fn test_different_nonces_for_different_blob_ids() {
        let cipher = AeadCipher::new(test_key());
        let key_id = b"key-shared";

        let (enc1, _) = cipher.encrypt(b"data", key_id, b"blob-A").unwrap();
        let (enc2, _) = cipher.encrypt(b"data", key_id, b"blob-B").unwrap();

        let nonce1 = extract_nonce(&enc1);
        let nonce2 = extract_nonce(&enc2);

        assert_ne!(
            nonce1, nonce2,
            "different blob_ids must produce different nonces"
        );
    }

    #[test]
    fn test_empty_plaintext() {
        let cipher = AeadCipher::new(test_key());
        let plaintext: &[u8] = b"";

        let (encrypted, version) = cipher
            .encrypt(plaintext, b"key-001", b"blob-empty")
            .unwrap();
        let decrypted = AeadCipher::decrypt(&encrypted, &test_key()).unwrap();

        assert_eq!(&*decrypted, plaintext);
        assert_eq!(version, VERSION);
    }

    #[test]
    fn test_large_plaintext_1mb() {
        let cipher = AeadCipher::new(test_key());
        let plaintext: Vec<u8> = (0..=255).cycle().take(1024 * 1024).collect();

        let (encrypted, version) = cipher
            .encrypt(&plaintext, b"key-001", b"blob-large")
            .unwrap();
        let decrypted = AeadCipher::decrypt(&encrypted, &test_key()).unwrap();

        assert_eq!(&*decrypted, &plaintext);
        assert_eq!(version, VERSION);
    }

    #[test]
    fn test_key_versioning() {
        let cipher = AeadCipher::new(test_key());
        let (encrypted, version) = cipher
            .encrypt(b"versioned data", b"key-001", b"blob-ver")
            .unwrap();

        assert_eq!(version, VERSION, "version byte must be 0x01");
        assert_eq!(
            encrypted[0], VERSION,
            "first byte of output must be version 0x01"
        );
    }

    #[test]
    fn test_deterministic_nonce() {
        let cipher = AeadCipher::new(test_key());
        let key_id = b"key-deterministic";
        let blob_id = b"blob-deterministic";

        let (enc1, _) = cipher.encrypt(b"data", key_id, blob_id).unwrap();
        let (enc2, _) = cipher.encrypt(b"other data", key_id, blob_id).unwrap();

        let nonce1 = extract_nonce(&enc1);
        let nonce2 = extract_nonce(&enc2);

        assert_eq!(
            nonce1, nonce2,
            "same key_id + blob_id must produce the same nonce"
        );
    }
}
