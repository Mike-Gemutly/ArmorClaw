pub mod server;
pub mod interceptor;
pub mod proto {
    pub mod sidecar {
        pub mod sidecar_service_server {
            tonic::include_proto!("sidecar");
        }
    }
}
