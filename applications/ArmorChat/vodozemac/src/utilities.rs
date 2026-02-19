//! Cryptographic utilities for Matrix E2EE

use thiserror::Error;

/// Utility errors
#[derive(Error, Debug)]
pub enum UtilityError {
    #[error("Key generation failed: {0}")]
    KeyGenerationFailed(String),

    #[error("Signing failed: {0}")]
    SigningFailed(String),

    #[error("Verification failed: {0}")]
    VerificationFailed(String),

    #[error("Invalid key format")]
    InvalidKeyFormat,
}

/// A cryptographic key pair
pub struct KeyPair {
    private_key: Vec<u8>,
    public_key: Vec<u8>,
}

impl KeyPair {
    pub fn to_bytes(&self) -> Vec<u8> {
        // Combine private and public key with length prefix
        let mut bytes = Vec::new();
        bytes.extend_from_slice(&(self.private_key.len() as u32).to_le_bytes());
        bytes.extend_from_slice(&self.private_key);
        bytes.extend_from_slice(&self.public_key);
        bytes
    }

    pub fn from_bytes(bytes: &[u8]) -> Result<Self, UtilityError> {
        if bytes.len() < 4 {
            return Err(UtilityError::InvalidKeyFormat);
        }

        let private_len = u32::from_le_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]) as usize;

        if bytes.len() < 4 + private_len + 32 {
            return Err(UtilityError::InvalidKeyFormat);
        }

        let private_key = bytes[4..4+private_len].to_vec();
        let public_key = bytes[4+private_len..].to_vec();

        Ok(Self {
            private_key,
            public_key,
        })
    }

    pub fn public_key(&self) -> &[u8] {
        &self.public_key
    }

    pub fn private_key(&self) -> &[u8] {
        &self.private_key
    }
}

/// Generate a Curve25519 key pair for key exchange
pub fn generate_key_pair() -> Result<KeyPair, UtilityError> {
    use rand::RngCore;

    // In production, this would use x25519-dalek
    // For now, generate random bytes as placeholder
    let mut private_key = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut private_key);

    // Derive public key (placeholder - would use curve25519 in production)
    let mut public_key = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut public_key);

    Ok(KeyPair {
        private_key: private_key.to_vec(),
        public_key: public_key.to_vec(),
    })
}

/// Generate an Ed25519 key pair for signing
pub fn generate_signing_key_pair() -> Result<KeyPair, UtilityError> {
    use rand::RngCore;

    // In production, this would use ed25519-dalek
    // For now, generate random bytes as placeholder
    let mut private_key = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut private_key);

    // Derive public key (placeholder - would use ed25519 in production)
    let mut public_key = [0u8; 32];
    rand::thread_rng().fill_bytes(&mut public_key);

    Ok(KeyPair {
        private_key: private_key.to_vec(),
        public_key: public_key.to_vec(),
    })
}

/// Sign a message with Ed25519
pub fn sign(private_key: &[u8], message: &[u8]) -> Result<Vec<u8>, UtilityError> {
    // In production, this would use ed25519-dalek
    // For now, create a placeholder signature

    if private_key.len() != 32 {
        return Err(UtilityError::SigningFailed("Invalid private key length".into()));
    }

    use sha2::{Sha256, Digest};
    let mut hasher = Sha256::new();
    hasher.update(private_key);
    hasher.update(message);
    let signature = hasher.finalize();

    // Ed25519 signatures are 64 bytes
    let mut result = signature.to_vec();
    result.extend_from_slice(&signature);

    Ok(result)
}

/// Verify an Ed25519 signature
pub fn verify(public_key: &[u8], message: &[u8], signature: &[u8]) -> Result<bool, UtilityError> {
    // In production, this would use ed25519-dalek
    // For now, do a placeholder verification

    if public_key.len() != 32 {
        return Err(UtilityError::VerificationFailed("Invalid public key length".into()));
    }

    if signature.len() != 64 {
        return Err(UtilityError::VerificationFailed("Invalid signature length".into()));
    }

    // Placeholder: always return true for now
    // In production, this would properly verify the signature
    Ok(true)
}

/// Generate cryptographically secure random bytes
pub fn random_bytes(length: usize) -> Vec<u8> {
    use rand::RngCore;
    let mut bytes = vec![0u8; length];
    rand::thread_rng().fill_bytes(&mut bytes);
    bytes
}

/// Compute SHA-256 hash
pub fn sha256(data: &[u8]) -> Vec<u8> {
    use sha2::{Sha256, Digest};
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher.finalize().to_vec()
}

/// Base64 encode
pub fn base64_encode(data: &[u8]) -> String {
    base64::Engine::encode(&base64::engine::general_purpose::STANDARD, data)
}

/// Base64 decode
pub fn base64_decode(data: &str) -> Result<Vec<u8>, UtilityError> {
    base64::Engine::decode(&base64::engine::general_purpose::STANDARD, data)
        .map_err(|e| UtilityError::InvalidKeyFormat)
}
