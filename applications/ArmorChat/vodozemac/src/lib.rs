//! Vodozemac Android JNI Bindings
//!
//! This library provides JNI bindings for vodozemac, the Matrix E2EE
//! implementation in Rust. It enables Android apps to use proper
//! Matrix encryption compatible with Element and other clients.

use jni::JNIEnv;
use jni::objects::{JClass, JObject, JString, JValue};
use jni::sys::{jint, jlong, jboolean, jbyteArray, jstring};

mod olm;
mod megolm;
mod utilities;

use olm::OlmSession;
use megolm::MegolmSession;

/// Initialize the native library
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_initialize(
    mut env: JNIEnv,
    _class: JClass,
) -> jboolean {
    // Initialize logging for Android
    android_logger::init_once(
        android_logger::Config::default()
            .with_max_level(log::LevelFilter::Info)
            .with_tag("VodozemacNative"),
    );

    log::info!("Vodozemac native library initialized");
    true as jboolean
}

/// Get the library version
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_getVersion(
    mut env: JNIEnv,
    _class: JClass,
) -> jstring {
    let version = env.new_string("vodozemac-0.8.0-android").unwrap();
    version.into_raw()
}

/// Generate Curve25519 key pair for identity
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_generateIdentityKeyPair(
    mut env: JNIEnv,
    _class: JClass,
) -> jbyteArray {
    match utilities::generate_key_pair() {
        Ok(key_pair) => {
            let bytes = key_pair.to_bytes();
            env.byte_array_from_slice(&bytes).unwrap()
        }
        Err(e) => {
            log::error!("Failed to generate identity key pair: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Generate Ed25519 key pair for signing
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_generateSigningKeyPair(
    mut env: JNIEnv,
    _class: JClass,
) -> jbyteArray {
    match utilities::generate_signing_key_pair() {
        Ok(key_pair) => {
            let bytes = key_pair.to_bytes();
            env.byte_array_from_slice(&bytes).unwrap()
        }
        Err(e) => {
            log::error!("Failed to generate signing key pair: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Sign a message with Ed25519
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_sign(
    mut env: JNIEnv,
    _class: JClass,
    private_key: jbyteArray,
    message: jbyteArray,
) -> jbyteArray {
    let private_key = match env.convert_byte_array(private_key) {
        Ok(bytes) => bytes,
        Err(e) => {
            log::error!("Failed to get private key: {}", e);
            return std::ptr::null_mut();
        }
    };

    let message = match env.convert_byte_array(message) {
        Ok(bytes) => bytes,
        Err(e) => {
            log::error!("Failed to get message: {}", e);
            return std::ptr::null_mut();
        }
    };

    match utilities::sign(&private_key, &message) {
        Ok(signature) => {
            env.byte_array_from_slice(&signature).unwrap()
        }
        Err(e) => {
            log::error!("Failed to sign: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Verify an Ed25519 signature
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_verify(
    mut env: JNIEnv,
    _class: JClass,
    public_key: jbyteArray,
    message: jbyteArray,
    signature: jbyteArray,
) -> jboolean {
    let public_key = match env.convert_byte_array(public_key) {
        Ok(bytes) => bytes,
        Err(_) => return false as jboolean,
    };

    let message = match env.convert_byte_array(message) {
        Ok(bytes) => bytes,
        Err(_) => return false as jboolean,
    };

    let signature = match env.convert_byte_array(signature) {
        Ok(bytes) => bytes,
        Err(_) => return false as jboolean,
    };

    match utilities::verify(&public_key, &message, &signature) {
        Ok(valid) => valid as jboolean,
        Err(_) => false as jboolean,
    }
}

// ============================================================================
// Olm Session Management
// ============================================================================

/// Create an Olm account (generates identity and one-time keys)
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_createOlmAccount(
    mut env: JNIEnv,
    _class: JClass,
) -> jlong {
    match OlmSession::create_account() {
        Ok(account) => {
            Box::into_raw(Box::new(account)) as jlong
        }
        Err(e) => {
            log::error!("Failed to create Olm account: {}", e);
            0
        }
    }
}

/// Get identity keys from account
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_getIdentityKeys(
    mut env: JNIEnv,
    _class: JClass,
    account_ptr: jlong,
) -> jstring {
    let account = unsafe { &mut *(account_ptr as *mut OlmSession) };

    match account.get_identity_keys() {
        Ok(keys) => {
            match serde_json::to_string(&keys) {
                Ok(json) => env.new_string(&json).unwrap().into_raw(),
                Err(e) => {
                    log::error!("Failed to serialize identity keys: {}", e);
                    std::ptr::null_mut()
                }
            }
        }
        Err(e) => {
            log::error!("Failed to get identity keys: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Generate one-time keys
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_generateOneTimeKeys(
    mut env: JNIEnv,
    _class: JClass,
    account_ptr: jlong,
    count: jint,
) -> jstring {
    let account = unsafe { &mut *(account_ptr as *mut OlmSession) };

    match account.generate_one_time_keys(count as usize) {
        Ok(keys) => {
            match serde_json::to_string(&keys) {
                Ok(json) => env.new_string(&json).unwrap().into_raw(),
                Err(e) => {
                    log::error!("Failed to serialize one-time keys: {}", e);
                    std::ptr::null_mut()
                }
            }
        }
        Err(e) => {
            log::error!("Failed to generate one-time keys: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Create outbound session
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_createOutboundSession(
    mut env: JNIEnv,
    _class: JClass,
    account_ptr: jlong,
    their_identity_key: jbyteArray,
    their_one_time_key: jbyteArray,
) -> jlong {
    let account = unsafe { &mut *(account_ptr as *mut OlmSession) };

    let identity_key = match env.convert_byte_array(their_identity_key) {
        Ok(bytes) => bytes,
        Err(_) => return 0,
    };

    let one_time_key = match env.convert_byte_array(their_one_time_key) {
        Ok(bytes) => bytes,
        Err(_) => return 0,
    };

    match account.create_outbound_session(&identity_key, &one_time_key) {
        Ok(session_id) => session_id as jlong,
        Err(e) => {
            log::error!("Failed to create outbound session: {}", e);
            0
        }
    }
}

/// Encrypt message with Olm
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_encryptOlm(
    mut env: JNIEnv,
    _class: JClass,
    session_ptr: jlong,
    plaintext: jbyteArray,
) -> jbyteArray {
    let session = unsafe { &mut *(session_ptr as *mut OlmSession) };

    let plaintext = match env.convert_byte_array(plaintext) {
        Ok(bytes) => bytes,
        Err(_) => return std::ptr::null_mut(),
    };

    match session.encrypt(&plaintext) {
        Ok(ciphertext) => {
            env.byte_array_from_slice(&ciphertext).unwrap()
        }
        Err(e) => {
            log::error!("Failed to encrypt: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Decrypt message with Olm
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_decryptOlm(
    mut env: JNIEnv,
    _class: JClass,
    session_ptr: jlong,
    ciphertext: jbyteArray,
    message_type: jint,
) -> jbyteArray {
    let session = unsafe { &mut *(session_ptr as *mut OlmSession) };

    let ciphertext = match env.convert_byte_array(ciphertext) {
        Ok(bytes) => bytes,
        Err(_) => return std::ptr::null_mut(),
    };

    match session.decrypt(&ciphertext, message_type as usize) {
        Ok(plaintext) => {
            env.byte_array_from_slice(&plaintext).unwrap()
        }
        Err(e) => {
            log::error!("Failed to decrypt: {}", e);
            std::ptr::null_mut()
        }
    }
}

// ============================================================================
// Megolm Group Sessions
// ============================================================================

/// Create outbound Megolm session
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_createOutboundMegolmSession(
    mut env: JNIEnv,
    _class: JClass,
) -> jlong {
    match MegolmSession::create_outbound() {
        Ok(session) => {
            Box::into_raw(Box::new(session)) as jlong
        }
        Err(e) => {
            log::error!("Failed to create Megolm session: {}", e);
            0
        }
    }
}

/// Get Megolm session key for sharing
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_getMegolmSessionKey(
    mut env: JNIEnv,
    _class: JClass,
    session_ptr: jlong,
) -> jstring {
    let session = unsafe { &*(session_ptr as *const MegolmSession) };

    match session.get_session_key() {
        Ok(key) => env.new_string(&key).unwrap().into_raw(),
        Err(e) => {
            log::error!("Failed to get session key: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Encrypt message with Megolm
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_encryptMegolm(
    mut env: JNIEnv,
    _class: JClass,
    session_ptr: jlong,
    plaintext: jbyteArray,
) -> jstring {
    let session = unsafe { &mut *(session_ptr as *mut MegolmSession) };

    let plaintext = match env.convert_byte_array(plaintext) {
        Ok(bytes) => bytes,
        Err(_) => return std::ptr::null_mut(),
    };

    match session.encrypt(&plaintext) {
        Ok(encrypted) => {
            match serde_json::to_string(&encrypted) {
                Ok(json) => env.new_string(&json).unwrap().into_raw(),
                Err(e) => {
                    log::error!("Failed to serialize encrypted message: {}", e);
                    std::ptr::null_mut()
                }
            }
        }
        Err(e) => {
            log::error!("Failed to encrypt with Megolm: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Create inbound Megolm session
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_createInboundMegolmSession(
    mut env: JNIEnv,
    _class: JClass,
    session_key: jstring,
) -> jlong {
    let session_key: JString = unsafe { JObject::from_raw(session_key).into() };
    let session_key = match env.get_string(&session_key) {
        Ok(s) => s.to_str().unwrap().to_string(),
        Err(_) => return 0,
    };

    match MegolmSession::create_inbound(&session_key) {
        Ok(session) => {
            Box::into_raw(Box::new(session)) as jlong
        }
        Err(e) => {
            log::error!("Failed to create inbound Megolm session: {}", e);
            0
        }
    }
}

/// Decrypt message with Megolm
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_decryptMegolm(
    mut env: JNIEnv,
    _class: JClass,
    session_ptr: jlong,
    ciphertext: jstring,
) -> jbyteArray {
    let session = unsafe { &mut *(session_ptr as *mut MegolmSession) };

    let ciphertext: JString = unsafe { JObject::from_raw(ciphertext).into() };
    let ciphertext = match env.get_string(&ciphertext) {
        Ok(s) => s.to_str().unwrap().to_string(),
        Err(_) => return std::ptr::null_mut(),
    };

    match session.decrypt(&ciphertext) {
        Ok(plaintext) => {
            env.byte_array_from_slice(&plaintext).unwrap()
        }
        Err(e) => {
            log::error!("Failed to decrypt with Megolm: {}", e);
            std::ptr::null_mut()
        }
    }
}

/// Free Olm account
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_freeOlmAccount(
    _env: JNIEnv,
    _class: JClass,
    account_ptr: jlong,
) {
    if account_ptr != 0 {
        unsafe {
            let _ = Box::from_raw(account_ptr as *mut OlmSession);
        }
    }
}

/// Free Megolm session
#[no_mangle]
pub extern "system" fn Java_app_armorclaw_crypto_VodozemacNative_freeMegolmSession(
    _env: JNIEnv,
    _class: JClass,
    session_ptr: jlong,
) {
    if session_ptr != 0 {
        unsafe {
            let _ = Box::from_raw(session_ptr as *mut MegolmSession);
        }
    }
}
