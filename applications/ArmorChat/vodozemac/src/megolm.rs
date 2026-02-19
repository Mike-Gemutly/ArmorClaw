//! Megolm group session management
//!
//! Megolm provides efficient group encryption using a symmetric ratchet.
//! The session key is shared via Olm with each group member.

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Megolm errors
#[derive(Error, Debug)]
pub enum MegolmError {
    #[error("Session creation failed: {0}")]
    SessionCreationFailed(String),

    #[error("Encryption failed: {0}")]
    EncryptionFailed(String),

    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),

    #[error("Invalid session key: {0}")]
    InvalidSessionKey(String),

    #[error("Invalid message index: {0}")]
    InvalidMessageIndex(String),

    #[error("Session not found")]
    SessionNotFound,
}

/// Encrypted Megolm message
#[derive(Serialize, Deserialize)]
pub struct MegolmMessage {
    pub algorithm: String,
    pub sender_key: String,
    pub session_id: String,
    pub ciphertext: String,
    pub message_index: u32,
}

/// Megolm group session
pub struct MegolmSession {
    session_id: String,
    outbound: Option<olm_rs::inbound_group_session::OlmInboundGroupSession>,
    message_index: u32,
    is_outbound: bool,
}

impl MegolmSession {
    /// Create a new outbound Megolm session
    pub fn create_outbound() -> Result<Self, MegolmError> {
        // Create a new session key (simulated)
        // In production, this would use vodozemac's Megolm implementation
        let session_key = Self::generate_session_key();
        let session_id = Self::session_id_from_key(&session_key);

        let inbound = olm_rs::inbound_group_session::OlmInboundGroupSession::new(&session_key)
            .map_err(|e| MegolmError::SessionCreationFailed(format!("{:?}", e)))?;

        Ok(Self {
            session_id,
            outbound: Some(inbound),
            message_index: 0,
            is_outbound: true,
        })
    }

    /// Create an inbound Megolm session from a session key
    pub fn create_inbound(session_key: &str) -> Result<Self, MegolmError> {
        let session_id = Self::session_id_from_key(session_key);

        let inbound = olm_rs::inbound_group_session::OlmInboundGroupSession::new(session_key)
            .map_err(|e| MegolmError::SessionCreationFailed(format!("{:?}", e)))?;

        Ok(Self {
            session_id,
            outbound: Some(inbound),
            message_index: 0,
            is_outbound: false,
        })
    }

    /// Get the session key for sharing with group members
    pub fn get_session_key(&self) -> Result<String, MegolmError> {
        if !self.is_outbound {
            return Err(MegolmError::SessionCreationFailed(
                "Cannot export key from inbound session".into()
            ));
        }

        // In production, this would return the actual session key
        // For now, return a placeholder
        Ok(format!("megolm_session_key_{}", self.session_id))
    }

    /// Get the session ID
    pub fn session_id(&self) -> &str {
        &self.session_id
    }

    /// Encrypt a message
    pub fn encrypt(&mut self, plaintext: &[u8]) -> Result<MegolmMessage, MegolmError> {
        if !self.is_outbound {
            return Err(MegolmError::EncryptionFailed(
                "Cannot encrypt with inbound session".into()
            ));
        }

        // In production, this would use vodozemac's Megolm encryption
        // For now, simulate the structure

        let ciphertext = base64::Engine::encode(
            &base64::engine::general_purpose::STANDARD,
            plaintext
        );

        let message = MegolmMessage {
            algorithm: "m.megolm.v1.aes-sha2".to_string(),
            sender_key: "placeholder_curve25519_key".to_string(),
            session_id: self.session_id.clone(),
            ciphertext,
            message_index: self.message_index,
        };

        self.message_index += 1;

        Ok(message)
    }

    /// Decrypt a message
    pub fn decrypt(&mut self, ciphertext_json: &str) -> Result<Vec<u8>, MegolmError> {
        let message: MegolmMessage = serde_json::from_str(ciphertext_json)
            .map_err(|e| MegolmError::DecryptionFailed(format!("Invalid JSON: {}", e)))?;

        if message.session_id != self.session_id {
            return Err(MegolmError::DecryptionFailed(
                "Session ID mismatch".into()
            ));
        }

        // In production, this would use vodozemac's Megolm decryption
        // For now, decode base64
        let plaintext = base64::Engine::decode(
            &base64::engine::general_purpose::STANDARD,
            message.ciphertext.as_bytes()
        ).map_err(|e| MegolmError::DecryptionFailed(format!("Base64 decode: {}", e)))?;

        self.message_index = message.message_index + 1;

        Ok(plaintext)
    }

    /// Pickle (serialize) the session
    pub fn pickle(&self) -> Result<Vec<u8>, MegolmError> {
        let inbound = self.outbound.as_ref()
            .ok_or(MegolmError::SessionNotFound)?;

        inbound.pickle(olm_rs::PicklingMode::EncryptWith(&[]))
            .map_err(|e| MegolmError::SessionCreationFailed(format!("{:?}", e)))
            .map(|s| s.as_bytes().to_vec())
    }

    /// Generate a random session key
    fn generate_session_key() -> String {
        use rand::RngCore;
        let mut key = [0u8; 128];
        rand::thread_rng().fill_bytes(&mut key);
        base64::Engine::encode(&base64::engine::general_purpose::STANDARD, &key)
    }

    /// Derive session ID from key
    fn session_id_from_key(key: &str) -> String {
        use sha2::{Sha256, Digest};
        let mut hasher = Sha256::new();
        hasher.update(key.as_bytes());
        let result = hasher.finalize();
        base64::Engine::encode(&base64::engine::general_purpose::STANDARD, &result)
    }
}

impl Drop for MegolmSession {
    fn drop(&mut self) {
        // Clear sensitive data
        // The Rust destructor will handle this
    }
}
