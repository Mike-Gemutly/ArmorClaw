use std::collections::HashMap;
use std::sync::{Arc, OnceLock, RwLock};
use thiserror::Error;
use tokio::time::{Duration, Instant};
use zeroize::Zeroizing;

use prometheus::{IntCounter, IntCounterVec, IntGauge, Opts};

// ── Prometheus metrics ─────────────────────────────────────────────────────

fn blindfill_issued() -> &'static IntCounterVec {
    static M: OnceLock<IntCounterVec> = OnceLock::new();
    M.get_or_init(|| {
        let counter = IntCounterVec::new(
            Opts::new("armorclaw_vault_blindfill_issued_total", "Total blindfill tokens issued"),
            &["tool"],
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(counter.clone()));
        counter
    })
}

fn blindfill_consumed() -> &'static IntCounterVec {
    static M: OnceLock<IntCounterVec> = OnceLock::new();
    M.get_or_init(|| {
        let counter = IntCounterVec::new(
            Opts::new("armorclaw_vault_blindfill_consumed_total", "Total blindfill tokens consumed"),
            &["tool"],
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(counter.clone()));
        counter
    })
}

fn blindfill_consume_failed() -> &'static IntCounterVec {
    static M: OnceLock<IntCounterVec> = OnceLock::new();
    M.get_or_init(|| {
        let counter = IntCounterVec::new(
            Opts::new(
                "armorclaw_vault_blindfill_consume_failed_total",
                "Total failed blindfill token consumptions",
            ),
            &["tool", "error"],
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(counter.clone()));
        counter
    })
}

fn blindfill_expired() -> &'static IntCounter {
    static M: OnceLock<IntCounter> = OnceLock::new();
    M.get_or_init(|| {
        let counter = IntCounter::new(
            "armorclaw_vault_blindfill_expired_total",
            "Total blindfill tokens expired during cleanup",
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(counter.clone()));
        counter
    })
}

fn blindfill_zeroized() -> &'static IntCounter {
    static M: OnceLock<IntCounter> = OnceLock::new();
    M.get_or_init(|| {
        let counter = IntCounter::new(
            "armorclaw_vault_blindfill_zeroized_total",
            "Total blindfill tokens proactively zeroized",
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(counter.clone()));
        counter
    })
}

fn blindfill_active_tokens() -> &'static IntGauge {
    static M: OnceLock<IntGauge> = OnceLock::new();
    M.get_or_init(|| {
        let gauge = IntGauge::new(
            "armorclaw_vault_blindfill_active_tokens",
            "Current number of active blindfill tokens in the store",
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(gauge.clone()));
        gauge
    })
}

/// Errors for ephemeral token operations.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum TokenError {
    #[error("token not found")]
    TokenNotFound,
    #[error("unauthorized: session mismatch")]
    Unauthorized,
    #[error("wrong tool")]
    WrongTool,
    #[error("token expired")]
    Expired,
}

/// Composite key for token lookup in the store.
#[derive(Debug, Clone, Hash, Eq, PartialEq)]
pub struct TokenKey {
    pub token_id: String,
    pub session_id: String,
    pub tool_name: String,
}

/// A single token entry with zeroizing plaintext and TTL metadata.
pub struct TokenEntry {
    pub plaintext: Zeroizing<String>,
    pub created_at: Instant,
    pub ttl: Duration,
}

/// In-memory store for short-lived, single-use secret tokens.
///
/// Uses `Arc<RwLock<HashMap>>` for sync access from tonic 0.10 interceptors.
/// All plaintext values are wrapped in `Zeroizing<String>` for secure memory handling.
pub struct EphemeralTokenStore {
    tokens: Arc<RwLock<HashMap<TokenKey, TokenEntry>>>,
}

impl EphemeralTokenStore {
    /// Create a new empty token store with a background TTL cleanup task.
    ///
    /// The cleanup task runs every 60 seconds and removes expired entries.
    /// Spawning is best-effort: if no tokio runtime is available, the store
    /// still works but expired entries are only cleaned on access.
    pub fn new() -> Self {
        let tokens: Arc<RwLock<HashMap<TokenKey, TokenEntry>>> =
            Arc::new(RwLock::new(HashMap::new()));

        if let Ok(handle) = tokio::runtime::Handle::try_current() {
            let tokens_clone = Arc::clone(&tokens);
            handle.spawn(async move {
                let mut interval = tokio::time::interval(Duration::from_secs(60));
                loop {
                    interval.tick().await;
                    if let Ok(mut map) = tokens_clone.write() {
                        let before = map.len();
                        map.retain(|_, entry| entry.created_at.elapsed() <= entry.ttl);
                        let expired = before - map.len();
                        if expired > 0 {
                            blindfill_expired().inc_by(expired as u64);
                            blindfill_active_tokens().set(map.len() as i64);
                        }
                    }
                }
            });
        }

        Self { tokens }
    }

    /// Issue a new token. Overwrites any existing token with the same composite key.
    ///
    /// # Arguments
    /// * `token_id` - Unique identifier for this token
    /// * `plaintext` - The secret value (stored as `Zeroizing<String>`)
    /// * `session_id` - Binding: only this session can consume the token
    /// * `tool_name` - Binding: only this tool can consume the token
    /// * `ttl` - Time-to-live before the token expires
    pub fn issue_token(
        &self,
        token_id: &str,
        plaintext: &str,
        session_id: &str,
        tool_name: &str,
        ttl: Duration,
    ) -> Result<(), TokenError> {
        let key = TokenKey {
            token_id: token_id.to_string(),
            session_id: session_id.to_string(),
            tool_name: tool_name.to_string(),
        };
        let entry = TokenEntry {
            plaintext: Zeroizing::new(plaintext.to_string()),
            created_at: Instant::now(),
            ttl,
        };
        tracing::info!(
            token_id = %token_id,
            tool = %tool_name,
            session = %session_id,
            ttl_ms = ttl.as_millis(),
            "Ephemeral token issued"
        );
        {
            let mut map = self.tokens.write().map_err(|_| TokenError::TokenNotFound)?;
            map.insert(key, entry);
            blindfill_active_tokens().set(map.len() as i64);
        }
        blindfill_issued().with_label_values(&[tool_name]).inc();
        Ok(())
    }

    /// Consume a token: validates bindings, checks TTL, returns plaintext, removes entry.
    ///
    /// Single-use: after successful consumption, the token is removed from the store.
    /// Thread-safe: uses write lock to ensure exactly one consumer succeeds under contention.
    pub fn consume_token(
        &self,
        token_id: &str,
        session_id: &str,
        tool_name: &str,
    ) -> Result<String, TokenError> {
        let mut map = match self.tokens.write() {
            Ok(guard) => guard,
            Err(_) => {
                blindfill_consume_failed()
                    .with_label_values(&[tool_name, "lock_error"])
                    .inc();
                return Err(TokenError::TokenNotFound);
            }
        };

        let found_key = map
            .keys()
            .find(|k| k.token_id == token_id)
            .cloned();

        let key = match found_key {
            None => {
                blindfill_consume_failed()
                    .with_label_values(&[tool_name, "token_not_found"])
                    .inc();
                return Err(TokenError::TokenNotFound);
            }
            Some(k) => k,
        };

        if key.session_id != session_id {
            blindfill_consume_failed()
                .with_label_values(&[tool_name, "unauthorized"])
                .inc();
            return Err(TokenError::Unauthorized);
        }

        if key.tool_name != tool_name {
            blindfill_consume_failed()
                .with_label_values(&[tool_name, "wrong_tool"])
                .inc();
            return Err(TokenError::WrongTool);
        }

        let entry = map.get(&key).expect("key exists (just found it)");
        if entry.created_at.elapsed() > entry.ttl {
            blindfill_consume_failed()
                .with_label_values(&[tool_name, "expired"])
                .inc();
            return Err(TokenError::Expired);
        }

        let entry = map.remove(&key).expect("key exists (just found it)");
        tracing::info!(token_id = %token_id, tool = %tool_name, "Ephemeral token consumed");
        blindfill_consumed().with_label_values(&[tool_name]).inc();
        blindfill_active_tokens().set(map.len() as i64);
        Ok(entry.plaintext.to_string())
    }

    /// Proactively zeroize all tokens for a given tool and session.
    ///
    /// Returns the number of tokens destroyed. Useful for session teardown
    /// or when a tool is no longer authorized.
    pub fn zeroize_for_tool(&self, tool_name: &str, session_id: &str) -> u32 {
        if let Ok(mut map) = self.tokens.write() {
            let before = map.len();
            map.retain(|key, _| {
                !(key.tool_name == tool_name && key.session_id == session_id)
            });
            let count = (before - map.len()) as u32;
            if count > 0 {
                blindfill_zeroized().inc_by(count as u64);
                blindfill_active_tokens().set(map.len() as i64);
            }
            tracing::info!(tool = %tool_name, session = %session_id, count = count, "Tool secrets zeroized");
            count
        } else {
            0
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU32, Ordering};

    // ── Test 1: Happy path ──────────────────────────────────────────────
    #[tokio::test]
    async fn happy_path_issue_consume_returns_plaintext() {
        let store = EphemeralTokenStore::new();
        store
            .issue_token(
                "tok1",
                "secret123",
                "sess1",
                "tool1",
                Duration::from_secs(1800),
            )
            .unwrap();

        let result = store.consume_token("tok1", "sess1", "tool1").unwrap();
        assert_eq!(result, "secret123");
    }

    // ── Test 2: Race condition ──────────────────────────────────────────
    #[tokio::test]
    async fn race_condition_1000_consumers_exactly_1_succeeds() {
        let store = Arc::new(EphemeralTokenStore::new());
        store
            .issue_token(
                "race_tok",
                "race_secret",
                "sess",
                "tool",
                Duration::from_secs(1800),
            )
            .unwrap();

        let success_count = Arc::new(AtomicU32::new(0));
        let not_found_count = Arc::new(AtomicU32::new(0));

        let mut handles = Vec::new();
        for _ in 0..1000 {
            let store = Arc::clone(&store);
            let success_count = Arc::clone(&success_count);
            let not_found_count = Arc::clone(&not_found_count);

            handles.push(tokio::spawn(async move {
                match store.consume_token("race_tok", "sess", "tool") {
                    Ok(_) => {
                        success_count.fetch_add(1, Ordering::Relaxed);
                    }
                    Err(TokenError::TokenNotFound) => {
                        not_found_count.fetch_add(1, Ordering::Relaxed);
                    }
                    Err(e) => panic!("unexpected error: {}", e),
                }
            }));
        }

        for handle in handles {
            handle.await.unwrap();
        }

        assert_eq!(success_count.load(Ordering::Relaxed), 1);
        assert_eq!(not_found_count.load(Ordering::Relaxed), 999);
    }

    // ── Test 3: Session binding ─────────────────────────────────────────
    #[tokio::test]
    async fn session_binding_wrong_session_returns_unauthorized() {
        let store = EphemeralTokenStore::new();
        store
            .issue_token(
                "tok1",
                "secret",
                "session_A",
                "tool1",
                Duration::from_secs(1800),
            )
            .unwrap();

        let result = store.consume_token("tok1", "session_B", "tool1");
        assert_eq!(result, Err(TokenError::Unauthorized));
    }

    // ── Test 4: Tool binding ────────────────────────────────────────────
    #[tokio::test]
    async fn tool_binding_wrong_tool_returns_wrong_tool() {
        let store = EphemeralTokenStore::new();
        store
            .issue_token(
                "tok1",
                "secret",
                "sess1",
                "agentmail",
                Duration::from_secs(1800),
            )
            .unwrap();

        let result = store.consume_token("tok1", "sess1", "evil_tool");
        assert_eq!(result, Err(TokenError::WrongTool));
    }

    // ── Test 5: TTL expiration ──────────────────────────────────────────
    #[tokio::test]
    async fn ttl_expiration_expired_token_returns_expired() {
        let store = EphemeralTokenStore::new();
        store
            .issue_token("tok1", "secret", "sess1", "tool1", Duration::from_millis(5))
            .unwrap();

        tokio::time::sleep(Duration::from_millis(10)).await;

        let result = store.consume_token("tok1", "sess1", "tool1");
        assert_eq!(result, Err(TokenError::Expired));
    }

    // ── Test 6: Proactive zeroize ───────────────────────────────────────
    #[tokio::test]
    async fn proactive_zeroize_removes_all_tokens_for_tool() {
        let store = EphemeralTokenStore::new();

        for i in 0..5 {
            store
                .issue_token(
                    &format!("tok_{}", i),
                    "secret",
                    "sess1",
                    "agentmail",
                    Duration::from_secs(1800),
                )
                .unwrap();
        }

        let destroyed = store.zeroize_for_tool("agentmail", "sess1");
        assert_eq!(destroyed, 5);

        // Verify the store is effectively empty for these tokens
        let result = store.consume_token("tok_0", "sess1", "agentmail");
        assert_eq!(result, Err(TokenError::TokenNotFound));
    }
}
