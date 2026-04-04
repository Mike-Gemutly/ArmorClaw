use crate::error::SidecarError;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use async_trait::async_trait;
use std::collections::HashMap;

#[async_trait]
#[async_trait]
pub trait CloudConnector: Send + Sync {
    async fn upload(&self, bucket: &str, key: &str, data: &[u8]) -> Result<String, SidecarError>;
    async fn download(&self, bucket: &str, key: &str) -> Result<Vec<u8>, SidecarError>;
    async fn list(&self, bucket: &str, prefix: &str) -> Result<Vec<String>, SidecarError>;
    async fn delete(&self, bucket: &str, key: &str) -> Result<(), SidecarError>;
}

pub struct SharePointConfig {
    pub tenant_id: String,
    pub client_id: String,
    pub client_secret: String,
    pub site_url: String,
}

pub struct SharePointConnector {
    client: Client,
    config: SharePointConfig,
    access_token: Option<String>,
}

impl SharePointConnector {
    pub fn new(config: SharePointConfig) -> Result<Self, SidecarError> {
        let client = Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .map_err(|e| SidecarError::HttpError(format!("Failed to create HTTP client: {}", e)))?;

        Ok(Self {
            client,
            config,
            access_token: None,
        })
    }

    pub async fn authenticate(&mut self) -> Result<(), SidecarError> {
        let token_url = format!(
            "https://login.microsoftonline.com/{}/oauth2/v2.0/token",
            self.config.tenant_id
        );

        let params = [
            ("client_id", self.config.client_id.as_str()),
            ("client_secret", self.config.client_secret.as_str()),
            ("scope", "https://graph.microsoft.com/.default"),
            ("grant_type", "client_credentials"),
        ];

        let response = self.client
            .post(token_url)
            .form(&params)
            .send()
            .await
            .map_err(|e| SidecarError::AuthenticationError(format!("Authentication failed: {}", e)))?;

        let token_response: TokenResponse = response.json().await
            .map_err(|e| SidecarError::AuthenticationError(format!("Invalid token response: {}", e)))?;

        self.access_token = Some(token_response.access_token);
        Ok(())
    }

    async fn get_drive_item(&self, item_id: &str) -> Result<DriveItem, SidecarError> {
        if self.access_token.is_none() {
            self.authenticate().await?;
        }

        let url = format!(
            "https://graph.microsoft.com/v1.0/drives/{}/root:/children",
            self.config.site_url
        );

        let response = self.client
            .get(url)
            .bearer_auth(self.access_token.as_ref().unwrap())
            .send()
            .await
            .map_err(|e| SidecarError::CloudStorageError(format!("Failed to get drive item: {}", e)))?;

        response.json().await.map_err(|e| SidecarError::CloudStorageError(format!("Invalid response: {}", e)))
    }

    async fn upload(&self, bucket: &str, key: &str, data: &[u8]) -> Result<String, SidecarError> {
        if self.access_token.is_none() {
            self.authenticate().await?;
        }

        let item = self.get_drive_item(bucket).await?;
        let upload_url = format!(
            "{}/drive/items/{}/root:/children/{}/content",
            self.config.site_url,
            item.id,
            key
        );

        let response = self.client
            .put(upload_url)
            .bearer_auth(self.access_token.as_ref().unwrap())
            .body(data)
            .send()
            .await
            .map_err(|e| SidecarError::CloudStorageError(format!("Upload failed: {}", e)))?;

        Ok(key.to_string())
    }

    async fn download(&self, bucket: &str, key: &str) -> Result<Vec<u8>, SidecarError> {
        if self.access_token.is_none() {
            self.authenticate().await?;
        }

        let item = self.get_drive_item(bucket).await?;
        let download_url = format!(
            "{}/drive/items/{}/root:/children/{}/content",
            self.config.site_url,
            item.id,
            key
        );

        let response = self.client
            .get(download_url)
            .bearer_auth(self.access_token.as_ref().unwrap())
            .send()
            .await
            .map_err(|e| SidecarError::CloudStorageError(format!("Download failed: {}", e)))?;

        Ok(response.bytes().await.map(|b| b.to_vec()))
    }

    async fn list(&self, bucket: &str, prefix: &str) -> Result<Vec<String>, SidecarError> {
        if self.access_token.is_none() {
            self.authenticate().await?;
        }

        let item = self.get_drive_item(bucket).await?;
        let list_url = format!(
            "{}/drive/items/{}/root:/children?filter=startswith(name,'{}')",
            self.config.site_url,
            item.id,
            prefix
        );

        let response = self.client
            .get(list_url)
            .bearer_auth(self.access_token.as_ref().unwrap())
            .send()
            .await
            .map_err(|e| SidecarError::CloudStorageError(format!("List failed: {}", e)))?;

        let items: Vec<DriveItem> = response.json()
            .map_err(|e| SidecarError::CloudStorageError(format!("Invalid list response: {}", e)))?;

        Ok(items.into_iter().map(|item| item.name).collect())
    }

    async fn delete(&self, bucket: &str, key: &str) -> Result<(), SidecarError> {
        if self.access_token.is_none() {
            self.authenticate().await?;
        }

        let item = self.get_drive_item(bucket).await?;
        let delete_url = format!(
            "{}/drive/items/{}/root:/children/{}",
            self.config.site_url,
            item.id,
            key
        );

        self.client
            .delete(delete_url)
            .bearer_auth(self.access_token.as_ref().unwrap())
            .send()
            .await
            .map_err(|e| SidecarError::CloudStorageError(format!("Delete failed: {}", e)))?;

        Ok(())
    }
}

#[derive(Debug, Deserialize)]
struct TokenResponse {
    access_token: String,
    token_type: String,
    expires_in: i64,
}

#[derive(Debug, Deserialize)]
struct DriveItem {
    id: String,
    name: String,
    #[serde(rename = "driveType")]
    drive_type: Option<String>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sharepoint_config_creation() {
        let config = SharePointConfig {
            tenant_id: "test-tenant".to_string(),
            client_id: "test-client".to_string(),
            client_secret: "test-secret".to_string(),
            site_url: "test-site.sharepoint.com".to_string(),
        };
        assert_eq!(config.tenant_id, "test-tenant");
    }

    #[test]
    fn test_sharepoint_connector_creation() {
        let config = SharePointConfig {
            tenant_id: "test-tenant".to_string(),
            client_id: "test-client".to_string(),
            client_secret: "test-secret".to_string(),
            site_url: "test-site.sharepoint.com".to_string(),
        };
        let result = SharePointConnector::new(config);
        assert!(result.is_ok());
    }
}
