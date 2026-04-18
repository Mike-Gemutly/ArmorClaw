use std::pin::Pin;
use std::task::{Context, Poll};
use std::time::SystemTime;

use tokio_stream::Stream;
use tonic::{Request, Response, Status};

use crate::governance::ephemeral::{EphemeralTokenStore, TokenError};
use crate::governance::event_notifier::{EventNotifier, SecureEventStream, VaultEvent};
use crate::grpc::governance::governance::{
    governance_server::Governance, ConsumeTokenRequest, ConsumeTokenResponse,
    IssueTokenRequest, IssueTokenResponse, SubscribeRequest, VaultEventStream, ZeroizeRequest,
    ZeroizeResponse,
};

pub struct VaultGovernanceService {
    token_store: EphemeralTokenStore,
    event_notifier: EventNotifier,
}

impl VaultGovernanceService {
    pub fn new() -> Self {
        Self {
            token_store: EphemeralTokenStore::new(),
            event_notifier: EventNotifier::new(16),
        }
    }

    fn now_timestamp_ms() -> i64 {
        SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .map(|d| d.as_millis() as i64)
            .unwrap_or(0)
    }
}

fn token_error_to_status(err: TokenError) -> Status {
    match err {
        TokenError::TokenNotFound => Status::not_found("token not found"),
        TokenError::Unauthorized => Status::permission_denied("session mismatch"),
        TokenError::WrongTool => Status::permission_denied("wrong tool"),
        TokenError::Expired => Status::deadline_exceeded("token expired"),
    }
}

struct EventStreamAdapter {
    rx: tokio::sync::broadcast::Receiver<VaultEvent>,
}

impl EventStreamAdapter {
    fn from_secure_stream(stream: SecureEventStream) -> Self {
        let rx = unsafe {
            let ptr = &stream as *const SecureEventStream as *const tokio::sync::broadcast::Receiver<VaultEvent>;
            std::ptr::read(ptr)
        };
        std::mem::forget(stream);
        Self { rx }
    }
}

impl Stream for EventStreamAdapter {
    type Item = Result<VaultEventStream, Status>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = unsafe { self.get_unchecked_mut() };
        match this.rx.try_recv() {
            Ok(event) => Poll::Ready(Some(Ok(VaultEventStream {
                event_type: event.event_type,
                session_id: event.session_id,
                timestamp: event.timestamp,
                payload: event.payload,
            }))),
            Err(tokio::sync::broadcast::error::TryRecvError::Lagged(_)) => {
                cx.waker().wake_by_ref();
                Poll::Pending
            }
            Err(tokio::sync::broadcast::error::TryRecvError::Empty) => {
                cx.waker().wake_by_ref();
                Poll::Pending
            }
            Err(tokio::sync::broadcast::error::TryRecvError::Closed) => Poll::Ready(None),
        }
    }
}

#[tonic::async_trait]
impl Governance for VaultGovernanceService {
    async fn issue_ephemeral_token(
        &self,
        request: Request<IssueTokenRequest>,
    ) -> Result<Response<IssueTokenResponse>, Status> {
        tracing::debug!(rpc = "IssueEphemeralToken", "Governance RPC called");
        let req = request.into_inner();

        let ttl = tokio::time::Duration::from_millis(req.ttl_ms as u64);

        self.token_store
            .issue_token(&req.token_id, &req.plaintext, &req.session_id, &req.tool_name, ttl, req.capability_scope)
            .map_err(token_error_to_status)?;

        let timestamp = Self::now_timestamp_ms();
        self.event_notifier.notify(VaultEvent::new(
            "token_issued",
            &req.session_id,
            timestamp,
            &format!(
                r#"{{"token_id":"{}","tool_name":"{}"}}"#,
                req.token_id, req.tool_name
            ),
        ));

        Ok(Response::new(IssueTokenResponse { success: true }))
    }

    async fn consume_ephemeral_token(
        &self,
        request: Request<ConsumeTokenRequest>,
    ) -> Result<Response<ConsumeTokenResponse>, Status> {
        let req = request.into_inner();

        let plaintext = self
            .token_store
            .consume_token(&req.token_id, &req.session_id, &req.tool_name)
            .map_err(|err| {
                tracing::warn!(token_id = %req.token_id, error = %err, "Token consume failed");
                token_error_to_status(err)
            })?;

        let timestamp = Self::now_timestamp_ms();
        self.event_notifier.notify(VaultEvent::new(
            "token_consumed",
            &req.session_id,
            timestamp,
            &format!(r#"{{"token_id":"{}","tool_name":"{}"}}"#, req.token_id, req.tool_name),
        ));

        Ok(Response::new(ConsumeTokenResponse { plaintext }))
    }

    async fn zeroize_tool_secrets(
        &self,
        request: Request<ZeroizeRequest>,
    ) -> Result<Response<ZeroizeResponse>, Status> {
        let req = request.into_inner();

        let count = self
            .token_store
            .zeroize_for_tool(&req.tool_name, &req.session_id);

        let timestamp = Self::now_timestamp_ms();
        self.event_notifier.notify(VaultEvent::new(
            "secrets_zeroized",
            &req.session_id,
            timestamp,
            &format!(
                r#"{{"tool_name":"{}","secrets_destroyed":{}}}"#,
                req.tool_name, count
            ),
        ));

        Ok(Response::new(ZeroizeResponse {
            secrets_destroyed: count as i32,
        }))
    }

    type SubscribeEventsStream =
        Pin<Box<dyn Stream<Item = Result<VaultEventStream, Status>> + Send>>;

    async fn subscribe_events(
        &self,
        _request: Request<SubscribeRequest>,
    ) -> Result<Response<Self::SubscribeEventsStream>, Status> {
        let stream = self.event_notifier.subscribe();
        let adapter = EventStreamAdapter::from_secure_stream(stream);

        Ok(Response::new(Box::pin(adapter)))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn service_can_be_created() {
        let _service = VaultGovernanceService::new();
    }
}
