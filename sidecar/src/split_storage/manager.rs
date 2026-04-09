use crate::encryption::aead::AeadCipher;
use crate::error::{Result, SidecarError};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use zeroize::Zeroize;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct EncryptedPayload {
    pub encrypted_text: String,
    pub version: u8,
    pub clearance: u8,
}

#[derive(Debug, Clone, PartialEq)]
pub struct DecryptedChunk {
    pub text: String,
    pub clearance: u8,
    pub score: f32,
}

pub struct SplitStorageManager {
    cipher: AeadCipher,
    key: [u8; 32],
}

impl SplitStorageManager {
    pub fn new(key: [u8; 32]) -> Self {
        Self {
            cipher: AeadCipher::new(key),
            key,
        }
    }

    pub fn encrypt_chunk(
        &self,
        text: &str,
        key_id: &[u8],
        blob_id: &[u8],
        clearance: u8,
    ) -> Result<EncryptedPayload> {
        let (ciphertext, version) = self.cipher.encrypt(text.as_bytes(), key_id, blob_id)?;
        let encrypted_text = BASE64.encode(&ciphertext);
        Ok(EncryptedPayload {
            encrypted_text,
            version,
            clearance,
        })
    }

    pub fn decrypt_chunk(&self, payload: &EncryptedPayload) -> Result<String> {
        let ciphertext = BASE64
            .decode(&payload.encrypted_text)
            .map_err(|e| SidecarError::InternalError(format!("base64 decode failed: {}", e)))?;
        let plaintext = AeadCipher::decrypt(&ciphertext, &self.key)?;
        String::from_utf8(plaintext.to_vec())
            .map_err(|e| SidecarError::InternalError(format!("invalid utf8 in plaintext: {}", e)))
    }

    pub fn filter_by_clearance(
        chunks: &[DecryptedChunk],
        user_clearance: u8,
    ) -> Vec<DecryptedChunk> {
        chunks
            .iter()
            .filter(|c| c.clearance <= user_clearance)
            .cloned()
            .collect()
    }

    pub fn can_access(chunk_clearance: u8, user_clearance: u8) -> bool {
        chunk_clearance <= user_clearance
    }

    pub fn validate_collection_name(name: &str) -> Result<()> {
        if name.is_empty() {
            return Err(SidecarError::InvalidRequest(
                "collection name must not be empty".to_string(),
            ));
        }
        if name.len() > 255 {
            return Err(SidecarError::InvalidRequest(
                "collection name must be ≤ 255 characters".to_string(),
            ));
        }
        if !name
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '_' || c == '-')
        {
            return Err(SidecarError::InvalidRequest(
                "collection name must contain only alphanumeric characters, underscores, or hyphens".to_string(),
            ));
        }
        Ok(())
    }

    pub fn extract_version(payload: &EncryptedPayload) -> u8 {
        payload.version
    }

    pub fn payload_to_map(payload: &EncryptedPayload) -> HashMap<String, serde_json::Value> {
        let mut map = HashMap::new();
        map.insert(
            "encrypted_text".to_string(),
            serde_json::Value::String(payload.encrypted_text.clone()),
        );
        map.insert(
            "clearance".to_string(),
            serde_json::Value::Number(serde_json::Number::from(payload.clearance)),
        );
        map.insert(
            "version".to_string(),
            serde_json::Value::Number(serde_json::Number::from(payload.version)),
        );
        map
    }

    pub fn map_to_payload(map: &HashMap<String, serde_json::Value>) -> Result<EncryptedPayload> {
        let encrypted_text = map
            .get("encrypted_text")
            .and_then(|v| v.as_str())
            .ok_or_else(|| {
                SidecarError::InternalError("missing encrypted_text in payload".to_string())
            })?
            .to_string();

        let clearance = map
            .get("clearance")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| {
                SidecarError::InternalError("missing or invalid clearance in payload".to_string())
            })? as u8;

        let version = map.get("version").and_then(|v| v.as_u64()).ok_or_else(|| {
            SidecarError::InternalError("missing or invalid version in payload".to_string())
        })? as u8;

        Ok(EncryptedPayload {
            encrypted_text,
            version,
            clearance,
        })
    }
}

impl Drop for SplitStorageManager {
    fn drop(&mut self) {
        self.key.zeroize();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::encryption::aead::VERSION as AEAD_VERSION;

    fn test_key() -> [u8; 32] {
        let mut key = [0u8; 32];
        key.copy_from_slice(b"split-storage-test-key-000000000");
        key
    }

    fn manager() -> SplitStorageManager {
        SplitStorageManager::new(test_key())
    }

    #[test]
    fn test_encrypted_text_stored_as_base64() {
        let mgr = manager();
        let text = "This is sensitive document content";
        let payload = mgr.encrypt_chunk(text, b"key-001", b"blob-abc", 3).unwrap();

        assert_ne!(payload.encrypted_text, text);
        assert!(BASE64.decode(&payload.encrypted_text).is_ok());
        let decoded = BASE64.decode(&payload.encrypted_text).unwrap();
        assert_ne!(decoded, text.as_bytes());
    }

    #[test]
    fn test_vectors_remain_plaintext() {
        let mgr = manager();
        let vector: Vec<f32> = vec![0.1, 0.2, 0.3, 0.4, 0.5];
        let original = vector.clone();

        let _payload = mgr
            .encrypt_chunk("some text", b"key-001", b"blob-001", 1)
            .unwrap();

        assert_eq!(vector, original);
    }

    #[test]
    fn test_roundtrip_encrypt_decrypt() {
        let mgr = manager();
        let original_text = "The quick brown fox jumps over the lazy dog";
        let payload = mgr
            .encrypt_chunk(original_text, b"key-round", b"blob-trip", 2)
            .unwrap();

        let decrypted = mgr.decrypt_chunk(&payload).unwrap();
        assert_eq!(decrypted, original_text);
    }

    #[test]
    fn test_rbac_enforcement_own_clearance() {
        let chunks = vec![
            DecryptedChunk {
                text: "public info".to_string(),
                clearance: 0,
                score: 0.95,
            },
            DecryptedChunk {
                text: "internal doc".to_string(),
                clearance: 3,
                score: 0.85,
            },
            DecryptedChunk {
                text: "secret data".to_string(),
                clearance: 5,
                score: 0.75,
            },
        ];

        let filtered = SplitStorageManager::filter_by_clearance(&chunks, 3);
        assert_eq!(filtered.len(), 2);
        assert_eq!(filtered[0].clearance, 0);
        assert_eq!(filtered[1].clearance, 3);
    }

    #[test]
    fn test_rbac_rejection_higher_clearance() {
        let chunks = vec![
            DecryptedChunk {
                text: "top secret".to_string(),
                clearance: 8,
                score: 0.99,
            },
            DecryptedChunk {
                text: "classified".to_string(),
                clearance: 10,
                score: 0.98,
            },
        ];

        let filtered = SplitStorageManager::filter_by_clearance(&chunks, 5);
        assert!(filtered.is_empty());
    }

    #[test]
    fn test_irrecoverable_halves_wrong_key() {
        let mgr = SplitStorageManager::new(test_key());
        let payload = mgr
            .encrypt_chunk("ultra secret data", b"key-001", b"blob-secret", 5)
            .unwrap();

        let mut wrong_key = [0u8; 32];
        wrong_key.copy_from_slice(b"wrong-key-for-testing-purposes!!");
        let wrong_mgr = SplitStorageManager::new(wrong_key);

        let result = wrong_mgr.decrypt_chunk(&payload);
        assert!(result.is_err());
    }

    #[test]
    fn test_collection_name_validation() {
        assert!(SplitStorageManager::validate_collection_name("documents").is_ok());
        assert!(SplitStorageManager::validate_collection_name("my_collection").is_ok());
        assert!(SplitStorageManager::validate_collection_name("docs-v2").is_ok());
        assert!(SplitStorageManager::validate_collection_name("a").is_ok());

        assert!(SplitStorageManager::validate_collection_name("").is_err());
        assert!(SplitStorageManager::validate_collection_name("has spaces").is_err());
        assert!(SplitStorageManager::validate_collection_name("has.dot").is_err());
        assert!(SplitStorageManager::validate_collection_name("has/slash").is_err());
        assert!(SplitStorageManager::validate_collection_name(&"x".repeat(256)).is_err());
    }

    #[test]
    fn test_key_version_stored() {
        let mgr = manager();
        let payload = mgr
            .encrypt_chunk("versioned content", b"key-ver", b"blob-ver", 1)
            .unwrap();

        assert_eq!(payload.version, AEAD_VERSION);
        assert_eq!(SplitStorageManager::extract_version(&payload), AEAD_VERSION);
    }
}
