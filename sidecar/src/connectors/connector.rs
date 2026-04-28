use crate::error::Result;
use async_trait::async_trait;
use std::collections::HashMap;

#[async_trait]
pub trait CloudConnector: Send + Sync {
    async fn upload(&self, container: &str, key: &str, data: &[u8]) -> Result<String>;
    async fn download(&self, container: &str, key: &str) -> Result<Vec<u8>>;
    async fn list(&self, container: &str, prefix: &str) -> Result<Vec<String>>;
    async fn delete(&self, container: &str, key: &str) -> Result<()>;
    async fn health_check(&self) -> Result<bool>;
    async fn get_config(&self) -> HashMap<String, String>;
}
