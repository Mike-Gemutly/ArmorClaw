//! Olm session management for 1:1 encrypted messaging
//!
//! Olm provides the Double Ratchet algorithm for forward secrecy
//! in one-to-one conversations.

use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Olm errors
#[derive(Error, Debug)]
pub enum OlmError {
    #[error("Account creation failed: {0}")]
    AccountCreationFailed(String),

    #[error("Key generation failed: {0}")]
    KeyGenerationFailed(String),

    #[error("Session creation failed: {0}")]
    SessionCreationFailed(String),

    #[error("Encryption failed: {0}")]
    EncryptionFailed(String),

    #[error("Decryption failed: {0}")]
    DecryptionFailed(String),

    #[error("Invalid key: {0}")]
    InvalidKey(String),

    #[error("Session not found")]
    SessionNotFound,
}

/// Identity keys for an Olm account
#[derive(Serialize, Deserialize)]
pub struct IdentityKeys {
    pub curve25519: String,
    pub ed25519: String,
}

/// One-time key
#[derive(Serialize, Deserialize)]
pub struct OneTimeKey {
    pub key_id: String,
    pub key: String,
}

/// Encrypted message
#[derive(Serialize, Deserialize)]
pub struct EncryptedMessage {
    pub message_type: usize,
    pub body: String,
}

/// Olm session for 1:1 encryption
pub struct OlmSession {
    // In production, this would hold actual vodozemac types
    // For now, we use olm-rs as the underlying implementation
    account: Option<olm_rs::account::OlmAccount>,
    sessions: Vec<(String, olm_rs::session::OlmSession)>,
    current_session_id: usize,
}

impl OlmSession {
    /// Create a new Olm account
    pub fn create_account() -> Result<Self, OlmError> {
        let account = olm_rs::account::OlmAccount::new()
            .map_err(|e| OlmError::AccountCreationFailed(format!("{:?}", e)))?;

        Ok(Self {
            account: Some(account),
            sessions: Vec::new(),
            current_session_id: 0,
        })
    }

    /// Get identity keys (Curve25519 + Ed25519)
    pub fn get_identity_keys(&self) -> Result<IdentityKeys, OlmError> {
        let account = self.account.as_ref()
            .ok_or(OlmError::AccountCreationFailed("No account".into()))?;

        let curve25519 = account.parsed_identity_keys().curve25519;
        let ed25519 = account.parsed_identity_keys().ed25519;

        Ok(IdentityKeys {
            curve25519,
            ed25519,
        })
    }

    /// Generate one-time keys
    pub fn generate_one_time_keys(&mut self, count: usize) -> Result<Vec<OneTimeKey>, OlmError> {
        let account = self.account.as_mut()
            .ok_or(OlmError::AccountCreationFailed("No account".into()))?;

        account.generate_one_time_keys(count as u64)
            .map_err(|e| OlmError::KeyGenerationFailed(format!("{:?}", e)))?;

        let keys = account.parsed_one_time_keys();

        let one_time_keys: Vec<OneTimeKey> = keys.curve25519
            .iter()
            .map(|(key_id, key)| OneTimeKey {
                key_id: key_id.clone(),
                key: key.clone(),
            })
            .collect();

        Ok(one_time_keys)
    }

    /// Create an outbound session with a recipient
    pub fn create_outbound_session(
        &mut self,
        their_identity_key: &[u8],
        their_one_time_key: &[u8],
    ) -> Result<usize, OlmError> {
        let account = self.account.as_ref()
            .ok_or(OlmError::AccountCreationFailed("No account".into()))?;

        let their_identity = std::str::from_utf8(their_identity_key)
            .map_err(|_| OlmError::InvalidKey("Invalid identity key".into()))?;
        let their_otk = std::str::from_utf8(their_one_time_key)
            .map_err(|_| OlmError::InvalidKey("Invalid one-time key".into()))?;

        let session = olm_rs::session::OlmSession::new_outbound(
            account,
            their_identity,
            their_otk,
        ).map_err(|e| OlmError::SessionCreationFailed(format!("{:?}", e)))?;

        let session_id = self.sessions.len();
        self.sessions.push((session.session_id(), session));
        self.current_session_id = session_id;

        Ok(session_id)
    }

    /// Encrypt a message
    pub fn encrypt(&mut self, plaintext: &[u8]) -> Result<Vec<u8>, OlmError> {
        let session = self.sessions.get_mut(self.current_session_id)
            .map(|(_, s)| s)
            .ok_or(OlmError::SessionNotFound)?;

        let message_type = session.message_type();
        let ciphertext = session.encrypt(plaintext)
            .map_err(|e| OlmError::EncryptionFailed(format!("{:?}", e)))?;

        // Prepend message type byte
        let mut result = vec![message_type as u8];
        result.extend(ciphertext.as_bytes());

        Ok(result)
    }

    /// Decrypt a message
    pub fn decrypt(&mut self, ciphertext: &[u8], message_type: usize) -> Result<Vec<u8>, OlmError> {
        let session = self.sessions.get_mut(self.current_session_id)
            .map(|(_, s)| s)
            .ok_or(OlmError::SessionNotFound)?;

        let ciphertext_str = std::str::from_utf8(ciphertext)
            .map_err(|_| OlmError::DecryptionFailed("Invalid ciphertext".into()))?;

        let msg_type = match message_type {
            0 => olm_rs::session::OlmMessageType::PreKey,
            1 => olm_rs::session::OlmMessageType::Message,
            _ => return Err(OlmError::DecryptionFailed("Invalid message type".into())),
        };

        session.decrypt(&msg_type, ciphertext_str)
            .map_err(|e| OlmError::DecryptionFailed(format!("{:?}", e)))
    }

    /// Pickle (serialize) the account
    pub fn pickle(&self) -> Result<Vec<u8>, OlmError> {
        let account = self.account.as_ref()
            .ok_or(OlmError::AccountCreationFailed("No account".into()))?;

        account.pickle(olm_rs::PicklingMode::EncryptWith(&[]))
            .map_err(|e| OlmError::AccountCreationFailed(format!("{:?}", e)))
            .map(|s| s.as_bytes().to_vec())
    }

    /// Unpickle (deserialize) the account
    pub fn unpickle(data: &[u8]) -> Result<Self, OlmError> {
        let pickle = std::str::from_utf8(data)
            .map_err(|_| OlmError::AccountCreationFailed("Invalid pickle data".into()))?;

        let account = olm_rs::account::OlmAccount::unpickle(
            pickle.to_string(),
            olm_rs::PicklingMode::EncryptWith(&[]),
        ).map_err(|e| OlmError::AccountCreationFailed(format!("{:?}", e)))?;

        Ok(Self {
            account: Some(account),
            sessions: Vec::new(),
            current_session_id: 0,
        })
    }
}

impl Drop for OlmSession {
    fn drop(&mut self) {
        // Clear sensitive data
        if let Some(account) = &mut self.account {
            let _ = account.generate_one_time_keys(0); // Clear one-time keys
        }
    }
}
