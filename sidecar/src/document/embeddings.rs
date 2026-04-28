use crate::error::{Result, SidecarError};
use reqwest::Client;
use serde::{Deserialize, Serialize};
use async_trait::async_trait;

#[async_trait]
pub trait Embedder: Send + Sync {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>>;
    async fn generate_batch_embeddings(&self, texts: &[String]) -> Result<Vec<Vec<f32>>>;
}

pub struct OpenAIEmbedder {
    client: Client,
    api_key: String,
    model: String,
}

impl OpenAIEmbedder {
    pub fn new(api_key: String) -> Result<Self> {
        let client = Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .map_err(|e| SidecarError::HttpError(format!("Failed to create HTTP client: {}", e)))?;

        Ok(Self {
            client,
            api_key,
            model: "text-embedding-ada-002".to_string(),
        })
    }

    pub fn with_model(&mut self, model: &str) {
        self.model = model.to_string();
    }
}

#[async_trait]
impl Embedder for OpenAIEmbedder {
    async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>> {
        if text.is_empty() {
            return Ok(vec![]);
        }

        let response = self.client
            .post("https://api.openai.com/v1/embeddings")
            .header("Authorization", format!("Bearer {}", self.api_key))
            .json(&serde_json::json!({
                "input": text,
                "model": self.model
            }))
            .send()
            .await
            .map_err(|e| SidecarError::ApiError(format!("Embedding generation failed: {}", e)))?;

        let embedding_response: EmbeddingResponse = response.json()
            .await
            .map_err(|e| SidecarError::ApiError(format!("Invalid embedding response: {}", e)))?;

        Ok(embedding_response.embedding)
    }

    async fn generate_batch_embeddings(&self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        let mut embeddings = Vec::new();
        for text in texts {
            let embedding = self.generate_embedding(text).await?;
            embeddings.push(embedding);
        }
        Ok(embeddings)
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmbeddingResponse {
    pub object: String,
    pub embedding: Vec<f32>,
    pub index: i32,
}

pub struct EmbeddingGenerator {
    client: Client,
    api_key: String,
    model: String,
}

impl EmbeddingGenerator {
    pub fn new(api_key: String) -> Result<Self> {
        let client = Client::builder()
            .timeout(std::time::Duration::from_secs(30))
            .build()
            .map_err(|e| SidecarError::HttpError(format!("Failed to create HTTP client: {}", e)))?;

        Ok(Self {
            client,
            api_key,
            model: "text-embedding-ada-002".to_string(),
        })
    }

    pub async fn generate_embedding(&self, text: &str) -> Result<Vec<f32>> {
        if text.is_empty() {
            return Ok(vec![]);
        }

        let response = self.client
            .post("https://api.openai.com/v1/embeddings")
            .header("Authorization", format!("Bearer {}", self.api_key))
            .json(&serde_json::json!({
                "input": text,
                "model": self.model
            }))
            .send()
            .await
            .map_err(|e| SidecarError::ApiError(format!("Embedding generation failed: {}", e)))?;

        let embedding_response: EmbeddingResponse = response.json()
            .await
            .map_err(|e| SidecarError::ApiError(format!("Invalid embedding response: {}", e)))?;

        Ok(embedding_response.embedding)
    }

    pub async fn generate_batch_embeddings(&self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        let mut embeddings = Vec::new();
        for text in texts {
            let embedding = self.generate_embedding(text).await?;
            embeddings.push(embedding);
        }
        Ok(embeddings)
    }

    pub fn with_model(&mut self, model: &str) {
        self.model = model.to_string();
    }
}

pub async fn generate_text_embedding(text: &str, api_key: &str) -> Result<Vec<f32>> {
    EmbeddingGenerator::new(api_key.to_string())?.generate_embedding(text).await
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_embedding_generator_new() {
        let generator = EmbeddingGenerator::new("test_key".to_string());
        assert!(generator.is_ok());
        assert_eq!(generator.unwrap().model, "text-embedding-ada-002");
    }

    #[test]
    fn test_empty_embedding() {
        let rt = tokio::runtime::Runtime::new().unwrap();
        let result = rt.block_on(async {
            EmbeddingGenerator::new("test_key".to_string())?.generate_embedding("").await
        });
        assert!(result.is_ok());
        let embedding = result.unwrap();
        assert!(embedding.is_empty());
    }

    #[test]
    fn test_with_model() {
        let mut generator = EmbeddingGenerator::new("test_key".to_string()).unwrap();
        generator.with_model("text-embedding-ada-003");
        assert_eq!(generator.model, "text-embedding-ada-003");
    }
}
