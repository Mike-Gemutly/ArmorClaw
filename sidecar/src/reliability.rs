use crate::error::{Result, SidecarError};
use prometheus::{Counter, IntGauge, IntGaugeVec, Registry};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::{Mutex, RwLock};
use tracing::{debug, info, warn};

pub struct Metrics {
    registry: Registry,
    circuit_breaker_state: IntGaugeVec,
    circuit_breaker_failures: IntGaugeVec,
    circuit_breaker_calls_total: Counter,
    circuit_breaker_errors_total: Counter,
    rate_limit_allowed: IntGaugeVec,
    rate_limit_denied: Counter,
}

impl Metrics {
    pub fn new() -> Self {
        let registry = Registry::new();
        
        let circuit_breaker_state = IntGaugeVec::new(
            prometheus::opts!(
                "circuit_breaker_state",
                "Current state of the circuit breaker (0=Closed, 1=Open, 2=Half-Open)"
            ),
            &["operation"]
        ).expect("Failed to create circuit_breaker_state metric");
        
        let circuit_breaker_failures = IntGaugeVec::new(
            prometheus::opts!(
                "circuit_breaker_failures",
                "Number of consecutive failures"
            ),
            &["operation"]
        ).expect("Failed to create circuit_breaker_failures metric");
        
        let circuit_breaker_calls_total = Counter::new(
            prometheus::opts!(
                "circuit_breaker_calls_total",
                "Total number of calls through circuit breaker"
            ),
        ).expect("Failed to create circuit_breaker_calls_total metric");
        
        let circuit_breaker_errors_total = Counter::new(
            prometheus::opts!(
                "circuit_breaker_errors_total",
                "Total number of errors through circuit breaker"
            ),
        ).expect("Failed to create circuit_breaker_errors_total metric");
        
        let rate_limit_allowed = IntGaugeVec::new(
            prometheus::opts!(
                "rate_limit_allowed",
                "Number of allowed requests in current window"
            ),
            &["operation"]
        ).expect("Failed to create rate_limit_allowed metric");
        
        let rate_limit_denied = Counter::new(
            prometheus::opts!(
                "rate_limit_denied",
                "Total number of requests denied by rate limiter"
            ),
        ).expect("Failed to create rate_limit_denied metric");
        
        registry.register(Box::new(circuit_breaker_state.clone())).unwrap();
        registry.register(Box::new(circuit_breaker_failures.clone())).unwrap();
        registry.register(Box::new(circuit_breaker_calls_total.clone())).unwrap();
        registry.register(Box::new(circuit_breaker_errors_total.clone())).unwrap();
        registry.register(Box::new(rate_limit_allowed.clone())).unwrap();
        registry.register(Box::new(rate_limit_denied.clone())).unwrap();
        
        Self {
            registry,
            circuit_breaker_state,
            circuit_breaker_failures,
            circuit_breaker_calls_total,
            circuit_breaker_errors_total,
            rate_limit_allowed,
            rate_limit_denied,
        }
    }
    
    pub fn registry(&self) -> &Registry {
        &self.registry
    }
    
    fn operation_name(operation: &str) -> String {
        operation.to_string()
    }
}

#[derive(Debug, Clone, Copy, PartialEq)]
enum CircuitBreakerState {
    Closed = 0,
    Open = 1,
    HalfOpen = 2,
}

pub struct CircuitBreaker {
    name: String,
    state: Arc<RwLock<CircuitBreakerState>>,
    failure_count: Arc<Mutex<u32>>,
    last_failure_time: Arc<Mutex<Option<Instant>>>,
    failure_threshold: u32,
    recovery_timeout: Duration,
    metrics: Arc<Metrics>,
}

impl CircuitBreaker {
    pub fn new(
        name: String,
        failure_threshold: u32,
        recovery_timeout_secs: u64,
        metrics: Arc<Metrics>,
    ) -> Self {
        let state = CircuitBreakerState::Closed;
        let op_name = Metrics::operation_name(&name);
        
        metrics.circuit_breaker_state
            .with_label_values(&[&op_name])
            .set(state as i64);
        
        Self {
            name,
            state: Arc::new(RwLock::new(state)),
            failure_count: Arc::new(Mutex::new(0)),
            last_failure_time: Arc::new(Mutex::new(None)),
            failure_threshold,
            recovery_timeout: Duration::from_secs(recovery_timeout_secs),
            metrics,
        }
    }
    
    pub async fn call<F, T, E>(&self, operation: F) -> Result<T>
    where
        F: std::future::Future<Output = std::result::Result<T, E>>,
        E: std::error::Error + Send + Sync + 'static,
    {
        self.metrics.circuit_breaker_calls_total.inc();
        
        {
            let state = self.state.read().await;
            if *state == CircuitBreakerState::Open {
                if let Some(last_failure) = *self.last_failure_time.lock().await {
                    if last_failure.elapsed() >= self.recovery_timeout {
                        drop(state);
                        *self.state.write().await = CircuitBreakerState::HalfOpen;
                        let op_name = Metrics::operation_name(&self.name);
                        self.metrics.circuit_breaker_state
                            .with_label_values(&[&op_name])
                            .set(CircuitBreakerState::HalfOpen as i64);
                        info!("Circuit breaker {} transitioning to Half-Open", self.name);
                    } else {
                        self.metrics.circuit_breaker_errors_total.inc();
                        return Err(SidecarError::CircuitBreakerOpen(format!(
                            "Circuit breaker {} is open",
                            self.name
                        )));
                    }
                }
            }
        }
        
        let result = operation.await;
        
        match result {
            Ok(value) => self.on_success().await,
            Err(err) => self.on_failure().await,
        }
        
        result.map_err(|e| SidecarError::StorageError(e.to_string()))
    }
    
    async fn on_success(&self) {
        let mut state = self.state.write().await;
        match *state {
            CircuitBreakerState::Open => {}
            CircuitBreakerState::HalfOpen => {
                *state = CircuitBreakerState::Closed;
                *self.failure_count.lock().await = 0;
                let op_name = Metrics::operation_name(&self.name);
                self.metrics.circuit_breaker_state
                    .with_label_values(&[&op_name])
                    .set(CircuitBreakerState::Closed as i64);
                self.metrics.circuit_breaker_failures
                    .with_label_values(&[&op_name])
                    .set(0);
                info!("Circuit breaker {} reset to Closed after successful call", self.name);
            }
            CircuitBreakerState::Closed => {
                *self.failure_count.lock().await = 0;
                let op_name = Metrics::operation_name(&self.name);
                self.metrics.circuit_breaker_failures
                    .with_label_values(&[&op_name])
                    .set(0);
            }
        }
    }
    
    async fn on_failure(&self) {
        self.metrics.circuit_breaker_errors_total.inc();
        
        let mut failure_count = self.failure_count.lock().await;
        *failure_count += 1;
        let op_name = Metrics::operation_name(&self.name);
        self.metrics.circuit_breaker_failures
            .with_label_values(&[&op_name])
            .set(*failure_count as i64);
        
        if *failure_count >= self.failure_threshold {
            let mut state = self.state.write().await;
            *state = CircuitBreakerState::Open;
            *self.last_failure_time.lock().await = Some(Instant::now());
            self.metrics.circuit_breaker_state
                .with_label_values(&[&op_name])
                .set(CircuitBreakerState::Open as i64);
            warn!(
                "Circuit breaker {} opened after {} failures",
                self.name, *failure_count
            );
        }
    }
    
    pub fn state(&self) -> CircuitBreakerState {
        let state = self.state.blocking_read();
        *state
    }
}

pub struct RateLimiter {
    name: String,
    max_requests: u32,
    window_duration: Duration,
    requests: Arc<Mutex<Vec<Instant>>>,
    metrics: Arc<Metrics>,
}

impl RateLimiter {
    pub fn new(name: String, max_requests_per_second: u32, metrics: Arc<Metrics>) -> Self {
        Self {
            name,
            max_requests: max_requests_per_second,
            window_duration: Duration::from_secs(1),
            requests: Arc::new(Mutex::new(Vec::new())),
            metrics,
        }
    }
    
    pub async fn acquire(&self) -> Result<()> {
        let mut requests = self.requests.lock().await;
        let now = Instant::now();
        
        requests.retain(|&timestamp| now.duration_since(timestamp) < self.window_duration);
        
        let op_name = Metrics::operation_name(&self.name);
        self.metrics.rate_limit_allowed
            .with_label_values(&[&op_name])
            .set(requests.len() as i64);
        
        if requests.len() >= self.max_requests as usize {
            self.metrics.rate_limit_denied.inc();
            return Err(SidecarError::RateLimitExceeded(format!(
                "Rate limit exceeded for {}: {} requests/sec, max: {}",
                self.name,
                requests.len(),
                self.max_requests
            )));
        }
        
        requests.push(now);
        
        let count = requests.len();
        self.metrics.rate_limit_allowed
            .with_label_values(&[&op_name])
            .set(count as i64);
        
        debug!(
            "Rate limiter {} allowed request (current: {}, max: {})",
            self.name, count, self.max_requests
        );
        
        Ok(())
    }
    
    pub async fn try_acquire(&self) -> bool {
        self.acquire().await.is_ok()
    }
}

pub struct S3Reliability {
    upload_circuit_breaker: Arc<CircuitBreaker>,
    download_circuit_breaker: Arc<CircuitBreaker>,
    list_circuit_breaker: Arc<CircuitBreaker>,
    delete_circuit_breaker: Arc<CircuitBreaker>,
    rate_limiter: Arc<RateLimiter>,
    metrics: Arc<Metrics>,
}

impl S3Reliability {
    pub fn new(
        failure_threshold: u32,
        recovery_timeout_secs: u64,
        max_requests_per_second: u32,
    ) -> Self {
        let metrics = Arc::new(Metrics::new());
        
        let upload_circuit_breaker = Arc::new(CircuitBreaker::new(
            "s3_upload".to_string(),
            failure_threshold,
            recovery_timeout_secs,
            metrics.clone(),
        ));
        
        let download_circuit_breaker = Arc::new(CircuitBreaker::new(
            "s3_download".to_string(),
            failure_threshold,
            recovery_timeout_secs,
            metrics.clone(),
        ));
        
        let list_circuit_breaker = Arc::new(CircuitBreaker::new(
            "s3_list".to_string(),
            failure_threshold,
            recovery_timeout_secs,
            metrics.clone(),
        ));
        
        let delete_circuit_breaker = Arc::new(CircuitBreaker::new(
            "s3_delete".to_string(),
            failure_threshold,
            recovery_timeout_secs,
            metrics.clone(),
        ));
        
        let rate_limiter = Arc::new(RateLimiter::new(
            "s3".to_string(),
            max_requests_per_second,
            metrics.clone(),
        ));
        
        Self {
            upload_circuit_breaker,
            download_circuit_breaker,
            list_circuit_breaker,
            delete_circuit_breaker,
            rate_limiter,
            metrics,
        }
    }
    
    pub fn metrics(&self) -> &Arc<Metrics> {
        &self.metrics
    }
    
    pub async fn upload<F, T, E>(&self, operation: F) -> Result<T>
    where
        F: std::future::Future<Output = std::result::Result<T, E>>,
        E: std::error::Error + Send + Sync + 'static,
    {
        self.rate_limiter.acquire().await?;
        self.upload_circuit_breaker.call(operation).await
    }
    
    pub async fn download<F, T, E>(&self, operation: F) -> Result<T>
    where
        F: std::future::Future<Output = std::result::Result<T, E>>,
        E: std::error::Error + Send + Sync + 'static,
    {
        self.rate_limiter.acquire().await?;
        self.download_circuit_breaker.call(operation).await
    }
    
    pub async fn list<F, T, E>(&self, operation: F) -> Result<T>
    where
        F: std::future::Future<Output = std::result::Result<T, E>>,
        E: std::error::Error + Send + Sync + 'static,
    {
        self.rate_limiter.acquire().await?;
        self.list_circuit_breaker.call(operation).await
    }
    
    pub async fn delete<F, T, E>(&self, operation: F) -> Result<T>
    where
        F: std::future::Future<Output = std::result::Result<T, E>>,
        E: std::error::Error + Send + Sync + 'static,
    {
        self.rate_limiter.acquire().await?;
        self.delete_circuit_breaker.call(operation).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU32, Ordering};
    use std::time::Duration;
    
    fn create_test_metrics() -> Arc<Metrics> {
        Arc::new(Metrics::new())
    }
    
    #[tokio::test]
    async fn test_circuit_breaker_closed_on_success() {
        let metrics = create_test_metrics();
        let cb = CircuitBreaker::new("test".to_string(), 3, 1, metrics);
        
        let result: std::result::Result<(), String> = cb.call(async { Ok(()) }).await;
        
        assert!(result.is_ok());
        assert_eq!(cb.state(), CircuitBreakerState::Closed);
    }
    
    #[tokio::test]
    async fn test_circuit_breaker_opens_after_threshold() {
        let metrics = create_test_metrics();
        let cb = CircuitBreaker::new("test".to_string(), 3, 1, metrics);
        
        for _ in 0..3 {
            let result: std::result::Result<(), String> = cb.call(async { Err("error".to_string()) }).await;
            assert!(result.is_err());
        }
        
        assert_eq!(cb.state(), CircuitBreakerState::Open);
    }
    
    #[tokio::test]
    async fn test_circuit_breaker_blocks_when_open() {
        let metrics = create_test_metrics();
        let cb = CircuitBreaker::new("test".to_string(), 3, 2, metrics);
        
        for _ in 0..3 {
            let _result: std::result::Result<(), String> = cb.call(async { Err("error".to_string()) }).await;
        }
        
        let result: std::result::Result<(), String> = cb.call(async { Ok(()) }).await;
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), SidecarError::CircuitBreakerOpen(_)));
    }
    
    #[tokio::test]
    async fn test_circuit_breaker_transitions_to_half_open() {
        let metrics = create_test_metrics();
        let cb = CircuitBreaker::new("test".to_string(), 2, 1, metrics);
        
        for _ in 0..2 {
            let _result: std::result::Result<(), String> = cb.call(async { Err("error".to_string()) }).await;
        }
        
        assert_eq!(cb.state(), CircuitBreakerState::Open);
        
        tokio::time::sleep(Duration::from_secs(1) + Duration::from_millis(100)).await;
        
        let _result: std::result::Result<(), String> = cb.call(async { Ok(()) }).await;
        
        assert_eq!(cb.state(), CircuitBreakerState::HalfOpen);
    }
    
    #[tokio::test]
    async fn test_circuit_breaker_resets_to_closed_on_half_open_success() {
        let metrics = create_test_metrics();
        let cb = CircuitBreaker::new("test".to_string(), 2, 1, metrics);
        
        for _ in 0..2 {
            let _result: std::result::Result<(), String> = cb.call(async { Err("error".to_string()) }).await;
        }
        
        tokio::time::sleep(Duration::from_secs(1) + Duration::from_millis(100)).await;
        
        let _result: std::result::Result<(), String> = cb.call(async { Ok(()) }).await;
        
        assert_eq!(cb.state(), CircuitBreakerState::Closed);
    }
    
    #[tokio::test]
    async fn test_rate_limiter_allows_within_limit() {
        let metrics = create_test_metrics();
        let limiter = RateLimiter::new("test".to_string(), 10, metrics);
        
        for _ in 0..10 {
            let result = limiter.acquire().await;
            assert!(result.is_ok());
        }
    }
    
    #[tokio::test]
    async fn test_rate_limiter_blocks_exceeds_limit() {
        let metrics = create_test_metrics();
        let limiter = RateLimiter::new("test".to_string(), 5, metrics);
        
        for _ in 0..5 {
            let _result = limiter.acquire().await;
        }
        
        let result = limiter.acquire().await;
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), SidecarError::RateLimitExceeded(_)));
    }
    
    #[tokio::test]
    async fn test_rate_limiter_resets_after_window() {
        let metrics = create_test_metrics();
        let limiter = RateLimiter::new("test".to_string(), 2, metrics);
        
        let _ = limiter.acquire().await;
        let _ = limiter.acquire().await;
        assert!(limiter.acquire().await.is_err());
        
        tokio::time::sleep(Duration::from_secs(1) + Duration::from_millis(100)).await;
        
        let result = limiter.acquire().await;
        assert!(result.is_ok());
    }
    
    #[tokio::test]
    async fn test_s3_reliability_wraps_operations() {
        let reliability = S3Reliability::new(3, 10, 10);
        
        let call_count = Arc::new(AtomicU32::new(0));
        let call_count_clone = call_count.clone();
        
        let result = reliability
            .upload(async move {
                call_count_clone.fetch_add(1, Ordering::SeqCst);
                Ok::<(), String>(())
            })
            .await;
        
        assert!(result.is_ok());
        assert_eq!(call_count.load(Ordering::SeqCst), 1);
    }
    
    #[tokio::test]
    async fn test_s3_reliability_rate_limits() {
        let reliability = S3Reliability::new(3, 10, 2);
        
        let results = futures::future::join_all(
            (0..3)
                .map(|_| reliability.upload(async { Ok::<(), String>(()) }))
        ).await;
        
        assert_eq!(results.len(), 3);
        let success_count = results.into_iter().filter(|r| r.is_ok()).count();
        assert_eq!(success_count, 2);
    }
}
