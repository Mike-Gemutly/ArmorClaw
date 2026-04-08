use std::future::Future;
use std::path::PathBuf;
use std::pin::Pin;
use std::task::{Context, Poll};

use rust_vault::config::VaultConfig;
use rust_vault::grpc::governance::governance::governance_client::GovernanceClient;
use rust_vault::grpc::governance::governance::governance_server::GovernanceServer;
use rust_vault::grpc::governance::governance::{
    ConsumeTokenRequest, IssueTokenRequest, SubscribeRequest, ZeroizeRequest,
};
use rust_vault::grpc::server::GrpcServer;
use rust_vault::grpc::governance_service::VaultGovernanceService;
use tokio::net::UnixStream;
use tokio_stream::StreamExt;
use tonic::transport::{Endpoint, Server, Uri};

struct UnixConnector {
    socket_path: PathBuf,
}

impl UnixConnector {
    fn new(socket_path: PathBuf) -> Self {
        Self { socket_path }
    }
}

impl tonic::codegen::Service<Uri> for UnixConnector {
    type Response = UnixStream;
    type Error = std::io::Error;
    type Future = Pin<Box<dyn Future<Output = Result<Self::Response, Self::Error>> + Send>>;

    fn poll_ready(&mut self, _cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        Poll::Ready(Ok(()))
    }

    fn call(&mut self, _: Uri) -> Self::Future {
        let path = self.socket_path.clone();
        Box::pin(async move { UnixStream::connect(path).await })
    }
}

async fn make_client(socket_path: PathBuf) -> GovernanceClient<tonic::transport::Channel> {
    let endpoint = Endpoint::from_static("http://[::]:50051");

    let channel = endpoint
        .connect_with_connector(UnixConnector::new(socket_path))
        .await
        .expect("connected");

    GovernanceClient::new(channel)
}

#[tokio::test]
async fn issue_and_consume_token_round_trip() {
    let socket_path = PathBuf::from("/tmp/test_gov_issue_consume.sock");
    let _ = std::fs::remove_file(&socket_path);

    let mut config = VaultConfig::default();
    config.keystore_socket_path = socket_path.clone();

    let service = VaultGovernanceService::new();
    let mut server = GrpcServer::new(config).expect("server created");

    let handle = tokio::spawn(async move {
        let router = Server::builder().add_service(GovernanceServer::new(service));
        server.serve(router).await
    });

    for _ in 0..50 {
        if socket_path.exists() {
            break;
        }
        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
    }

    let mut client = make_client(socket_path.clone()).await;

    let issue_resp = client
        .issue_ephemeral_token(IssueTokenRequest {
            token_id: "tok_1".into(),
            plaintext: "super_secret".into(),
            session_id: "sess_A".into(),
            tool_name: "browser".into(),
            ttl_ms: 60_000,
        })
        .await
        .expect("issue succeeded");

    assert!(issue_resp.into_inner().success);

    let consume_resp = client
        .consume_ephemeral_token(ConsumeTokenRequest {
            token_id: "tok_1".into(),
            session_id: "sess_A".into(),
            tool_name: "browser".into(),
        })
        .await
        .expect("consume succeeded");

    assert_eq!(consume_resp.into_inner().plaintext, "super_secret");

    handle.abort();
    let _ = std::fs::remove_file(&socket_path);
}

#[tokio::test]
async fn consume_nonexistent_token_returns_not_found() {
    let socket_path = PathBuf::from("/tmp/test_gov_not_found.sock");
    let _ = std::fs::remove_file(&socket_path);

    let mut config = VaultConfig::default();
    config.keystore_socket_path = socket_path.clone();

    let service = VaultGovernanceService::new();
    let mut server = GrpcServer::new(config).expect("server created");

    let handle = tokio::spawn(async move {
        let router = Server::builder().add_service(GovernanceServer::new(service));
        server.serve(router).await
    });

    for _ in 0..50 {
        if socket_path.exists() {
            break;
        }
        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
    }

    let mut client = make_client(socket_path.clone()).await;

    let err = client
        .consume_ephemeral_token(ConsumeTokenRequest {
            token_id: "ghost".into(),
            session_id: "sess_X".into(),
            tool_name: "tool".into(),
        })
        .await
        .expect_err("should fail");

    assert_eq!(err.code(), tonic::Code::NotFound);

    handle.abort();
    let _ = std::fs::remove_file(&socket_path);
}

#[tokio::test]
async fn consume_with_wrong_session_returns_permission_denied() {
    let socket_path = PathBuf::from("/tmp/test_gov_wrong_session.sock");
    let _ = std::fs::remove_file(&socket_path);

    let mut config = VaultConfig::default();
    config.keystore_socket_path = socket_path.clone();

    let service = VaultGovernanceService::new();
    let mut server = GrpcServer::new(config).expect("server created");

    let handle = tokio::spawn(async move {
        let router = Server::builder().add_service(GovernanceServer::new(service));
        server.serve(router).await
    });

    for _ in 0..50 {
        if socket_path.exists() {
            break;
        }
        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
    }

    let mut client = make_client(socket_path.clone()).await;

    client
        .issue_ephemeral_token(IssueTokenRequest {
            token_id: "tok_sess".into(),
            plaintext: "secret".into(),
            session_id: "owner_session".into(),
            tool_name: "tool".into(),
            ttl_ms: 60_000,
        })
        .await
        .expect("issue succeeded");

    let err = client
        .consume_ephemeral_token(ConsumeTokenRequest {
            token_id: "tok_sess".into(),
            session_id: "impostor_session".into(),
            tool_name: "tool".into(),
        })
        .await
        .expect_err("should fail");

    assert_eq!(err.code(), tonic::Code::PermissionDenied);

    handle.abort();
    let _ = std::fs::remove_file(&socket_path);
}

#[tokio::test]
async fn double_consume_returns_not_found() {
    let socket_path = PathBuf::from("/tmp/test_gov_double_consume.sock");
    let _ = std::fs::remove_file(&socket_path);

    let mut config = VaultConfig::default();
    config.keystore_socket_path = socket_path.clone();

    let service = VaultGovernanceService::new();
    let mut server = GrpcServer::new(config).expect("server created");

    let handle = tokio::spawn(async move {
        let router = Server::builder().add_service(GovernanceServer::new(service));
        server.serve(router).await
    });

    for _ in 0..50 {
        if socket_path.exists() {
            break;
        }
        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
    }

    let mut client = make_client(socket_path.clone()).await;

    client
        .issue_ephemeral_token(IssueTokenRequest {
            token_id: "tok_single".into(),
            plaintext: "once_only".into(),
            session_id: "sess".into(),
            tool_name: "tool".into(),
            ttl_ms: 60_000,
        })
        .await
        .expect("issue succeeded");

    client
        .consume_ephemeral_token(ConsumeTokenRequest {
            token_id: "tok_single".into(),
            session_id: "sess".into(),
            tool_name: "tool".into(),
        })
        .await
        .expect("first consume succeeds");

    let err = client
        .consume_ephemeral_token(ConsumeTokenRequest {
            token_id: "tok_single".into(),
            session_id: "sess".into(),
            tool_name: "tool".into(),
        })
        .await
        .expect_err("second consume should fail");

    assert_eq!(err.code(), tonic::Code::NotFound);

    handle.abort();
    let _ = std::fs::remove_file(&socket_path);
}

#[tokio::test]
async fn zeroize_destroys_secrets() {
    let socket_path = PathBuf::from("/tmp/test_gov_zeroize.sock");
    let _ = std::fs::remove_file(&socket_path);

    let mut config = VaultConfig::default();
    config.keystore_socket_path = socket_path.clone();

    let service = VaultGovernanceService::new();
    let mut server = GrpcServer::new(config).expect("server created");

    let handle = tokio::spawn(async move {
        let router = Server::builder().add_service(GovernanceServer::new(service));
        server.serve(router).await
    });

    for _ in 0..50 {
        if socket_path.exists() {
            break;
        }
        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
    }

    let mut client = make_client(socket_path.clone()).await;

    for i in 0..3 {
        client
            .issue_ephemeral_token(IssueTokenRequest {
                token_id: format!("tok_z_{}", i),
                plaintext: format!("secret_{}", i),
                session_id: "sess".into(),
                tool_name: "browser".into(),
                ttl_ms: 60_000,
            })
            .await
            .expect("issue succeeded");
    }

    let zeroize_resp = client
        .zeroize_tool_secrets(ZeroizeRequest {
            tool_name: "browser".into(),
            session_id: "sess".into(),
        })
        .await
        .expect("zeroize succeeded");

    assert_eq!(zeroize_resp.into_inner().secrets_destroyed, 3);

    handle.abort();
    let _ = std::fs::remove_file(&socket_path);
}

#[tokio::test]
async fn subscribe_events_receives_issued_event() {
    let socket_path = PathBuf::from("/tmp/test_gov_subscribe.sock");
    let _ = std::fs::remove_file(&socket_path);

    let mut config = VaultConfig::default();
    config.keystore_socket_path = socket_path.clone();

    let service = VaultGovernanceService::new();
    let mut server = GrpcServer::new(config).expect("server created");

    let handle = tokio::spawn(async move {
        let router = Server::builder().add_service(GovernanceServer::new(service));
        server.serve(router).await
    });

    for _ in 0..50 {
        if socket_path.exists() {
            break;
        }
        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
    }

    let mut client = make_client(socket_path.clone()).await;

    let mut event_stream = client
        .subscribe_events(SubscribeRequest::default())
        .await
        .expect("subscribe succeeded")
        .into_inner();

    client
        .issue_ephemeral_token(IssueTokenRequest {
            token_id: "tok_ev".into(),
            plaintext: "secret".into(),
            session_id: "sess_ev".into(),
            tool_name: "tool_ev".into(),
            ttl_ms: 60_000,
        })
        .await
        .expect("issue succeeded");

    let event = tokio::time::timeout(std::time::Duration::from_secs(2), event_stream.next())
        .await
        .expect("received event within timeout")
        .expect("stream yielded item")
        .expect("event is Ok");

    assert_eq!(event.event_type, "token_issued");
    assert_eq!(event.session_id, "sess_ev");

    handle.abort();
    let _ = std::fs::remove_file(&socket_path);
}
