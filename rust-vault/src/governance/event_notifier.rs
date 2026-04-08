use std::fmt;
use std::future::Future;
use std::pin::Pin;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, OnceLock};
use std::task::{Context, Poll};

use prometheus::{IntCounterVec, IntGauge, Opts};
use tokio::sync::broadcast;
use tokio_stream::Stream;

fn events_emitted() -> &'static IntCounterVec {
    static M: OnceLock<IntCounterVec> = OnceLock::new();
    M.get_or_init(|| {
        let counter = IntCounterVec::new(
            Opts::new("armorclaw_vault_events_emitted_total", "Total vault events emitted"),
            &["type"],
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(counter.clone()));
        counter
    })
}

fn events_missed_total() -> &'static IntGauge {
    static M: OnceLock<IntGauge> = OnceLock::new();
    M.get_or_init(|| {
        let gauge = IntGauge::new(
            "armorclaw_vault_events_missed_total",
            "Number of events missed by slow consumers (last observed lag)",
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(gauge.clone()));
        gauge
    })
}

fn events_subscribers_gauge() -> &'static IntGauge {
    static M: OnceLock<IntGauge> = OnceLock::new();
    M.get_or_init(|| {
        let gauge = IntGauge::new(
            "armorclaw_vault_events_subscribers",
            "Current number of active event subscribers",
        )
        .expect("metric creation must succeed");
        let _ = prometheus::register(Box::new(gauge.clone()));
        gauge
    })
}

// ── Errors ──────────────────────────────────────────────────────────────────

#[derive(Debug, PartialEq, Eq)]
pub enum StreamError {
    EventsMissed(usize),
    ChannelClosed,
}

impl fmt::Display for StreamError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            StreamError::EventsMissed(n) => write!(f, "missed {} events due to slow consumer", n),
            StreamError::ChannelClosed => write!(f, "event channel closed"),
        }
    }
}

impl std::error::Error for StreamError {}

// ── VaultEvent ──────────────────────────────────────────────────────────────

/// Valid `event_type` values: "token_issued", "token_consumed", "token_expired",
/// "secrets_zeroized", "skill_gate_denied", "pii_detected_in_output".
#[derive(Debug, Clone, PartialEq)]
pub struct VaultEvent {
    pub event_type: String,
    pub session_id: String,
    pub timestamp: i64,
    pub payload: String,
}

impl VaultEvent {
    pub fn new(event_type: &str, session_id: &str, timestamp: i64, payload: &str) -> Self {
        Self {
            event_type: event_type.to_string(),
            session_id: session_id.to_string(),
            timestamp,
            payload: payload.to_string(),
        }
    }
}

// ── SecureEventStream ───────────────────────────────────────────────────────

pub struct SecureEventStream {
    rx: broadcast::Receiver<VaultEvent>,
}

impl Stream for SecureEventStream {
    type Item = Result<VaultEvent, StreamError>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let recv = self.rx.recv();
        match std::pin::pin!(recv).poll(cx) {
            Poll::Ready(Ok(event)) => Poll::Ready(Some(Ok(event))),
            Poll::Ready(Err(broadcast::error::RecvError::Lagged(n))) => {
                let missed_count = n as usize;
                events_missed_total().set(n as i64);
                tracing::warn!(missed = missed_count, "Slow consumer: events missed");
                Poll::Ready(Some(Err(StreamError::EventsMissed(missed_count))))
            }
            Poll::Ready(Err(broadcast::error::RecvError::Closed)) => Poll::Ready(None),
            Poll::Pending => Poll::Pending,
        }
    }
}

impl Drop for SecureEventStream {
    fn drop(&mut self) {
        events_subscribers_gauge().dec();
    }
}

// ── EventNotifier ───────────────────────────────────────────────────────────

pub struct EventNotifier {
    sender: broadcast::Sender<VaultEvent>,
    subscribers: Arc<AtomicUsize>,
}

impl EventNotifier {
    pub fn new(capacity: usize) -> Self {
        let (sender, _) = broadcast::channel(capacity);
        Self {
            sender,
            subscribers: Arc::new(AtomicUsize::new(0)),
        }
    }

    pub fn subscribe(&self) -> SecureEventStream {
        self.subscribers.fetch_add(1, Ordering::Relaxed);
        events_subscribers_gauge().inc();
        SecureEventStream {
            rx: self.sender.subscribe(),
        }
    }

    pub fn notify(&self, event: VaultEvent) {
        events_emitted().with_label_values(&[&event.event_type]).inc();
        tracing::debug!(
            event_type = %event.event_type,
            session = %event.session_id,
            "Vault event emitted"
        );
        let _ = self.sender.send(event);
    }

    pub fn subscriber_count(&self) -> usize {
        self.sender.receiver_count()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio_stream::StreamExt;

    #[tokio::test]
    async fn slow_consumer_detection_returns_events_missed() {
        let notifier = EventNotifier::new(2);
        let mut stream = notifier.subscribe();

        for i in 0..5 {
            notifier.notify(VaultEvent::new(
                "token_issued",
                "sess1",
                1000 + i as i64,
                &format!("event {}", i),
            ));
        }

        let result = stream.next().await.expect("stream should yield a value");
        assert_eq!(result, Err(StreamError::EventsMissed(3)));
    }

    #[tokio::test]
    async fn normal_event_delivery_works() {
        let notifier = EventNotifier::new(10);
        let mut stream = notifier.subscribe();

        let event = VaultEvent::new("token_issued", "sess1", 1000, r#"{"key":"val"}"#);
        notifier.notify(event.clone());

        let received = stream.next().await.expect("stream should yield event").expect("should be Ok");
        assert_eq!(received.event_type, "token_issued");
        assert_eq!(received.session_id, "sess1");
        assert_eq!(received.timestamp, 1000);
        assert_eq!(received.payload, r#"{"key":"val"}"#);
    }

    #[tokio::test]
    async fn multiple_subscribers_receive_same_event() {
        let notifier = EventNotifier::new(10);
        let mut stream1 = notifier.subscribe();
        let mut stream2 = notifier.subscribe();

        assert_eq!(notifier.subscriber_count(), 2);

        let event = VaultEvent::new("secrets_zeroized", "sess2", 2000, "");
        notifier.notify(event.clone());

        let r1 = stream1.next().await.expect("stream1 should yield").expect("should be Ok");
        let r2 = stream2.next().await.expect("stream2 should yield").expect("should be Ok");
        assert_eq!(r1.event_type, "secrets_zeroized");
        assert_eq!(r2.event_type, "secrets_zeroized");
    }

    #[tokio::test]
    async fn channel_closed_when_notifier_dropped() {
        let notifier = EventNotifier::new(10);
        let mut stream = notifier.subscribe();

        notifier.notify(VaultEvent::new("token_issued", "s1", 1, ""));

        let _ = stream.next().await;

        drop(notifier);

        let result = stream.next().await;
        assert!(result.is_none());
    }

    #[test]
    fn events_missed_display_format() {
        let err = StreamError::EventsMissed(5);
        assert_eq!(format!("{}", err), "missed 5 events due to slow consumer");
    }

    #[test]
    fn channel_closed_display_format() {
        let err = StreamError::ChannelClosed;
        assert_eq!(format!("{}", err), "event channel closed");
    }

    #[test]
    fn vault_event_clone_independence() {
        let event = VaultEvent::new("token_issued", "s1", 100, "payload");
        let cloned = event.clone();
        assert_eq!(event.event_type, cloned.event_type);
        assert_eq!(event.session_id, cloned.session_id);
        assert_eq!(event.timestamp, cloned.timestamp);
        assert_eq!(event.payload, cloned.payload);
    }

    #[tokio::test]
    async fn single_capacity_misses_events_when_overwhelmed() {
        let notifier = EventNotifier::new(1);
        let mut stream = notifier.subscribe();

        notifier.notify(VaultEvent::new("token_issued", "s1", 1, ""));
        notifier.notify(VaultEvent::new("token_consumed", "s1", 2, ""));

        let result = stream.next().await.expect("stream should yield");
        assert_eq!(result, Err(StreamError::EventsMissed(1)));
    }
}
