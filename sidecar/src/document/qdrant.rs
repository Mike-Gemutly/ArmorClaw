use crate::error::SidecarError;
use std::collections::HashMap;
use serde_json::Value;
use qdrant_client::Qdrant;
use qdrant_client::qdrant::{
    CreateCollectionBuilder,
    Distance,
    PointStruct,
    ScoredPoint,
    SearchPointsBuilder,
    UpsertPointsBuilder,
    VectorParamsBuilder,
};

pub struct QdrantClient {
    client: Qdrant,
    collection_name: String,
}

impl QdrantClient {
    pub async fn new(url: &str, collection_name: &str) -> Result<Self, SidecarError> {
        let client = Qdrant::from_url(url)
            .build()
            .map_err(|e| SidecarError::StorageError(format!("Failed to connect to Qdrant: {}", e)))?;

        Ok(Self {
            client,
            collection_name: collection_name.to_string(),
        })
    }

    pub async fn create_collection(&self, vector_size: usize) -> Result<(), SidecarError> {
        self.client
            .create_collection(
                CreateCollectionBuilder::new(&self.collection_name)
                    .vectors_config(VectorParamsBuilder::new(vector_size as u64, Distance::Cosine)),
            )
            .await
            .map_err(|e| SidecarError::StorageError(format!("Failed to create collection: {}", e)))?;

        Ok(())
    }

    pub async fn upsert_vectors(
        &self,
        id: String,
        vectors: Vec<f32>,
        payload: HashMap<String, Value>,
    ) -> Result<(), SidecarError> {
        let json_payload: Value = payload.into_iter().collect();
        let qdrant_payload = qdrant_client::Payload::try_from(json_payload)
            .map_err(|e| SidecarError::StorageError(format!("Invalid payload: {}", e)))?;

        let points = vec![PointStruct::new(id, vectors, qdrant_payload)];

        self.client
            .upsert_points(UpsertPointsBuilder::new(&self.collection_name, points))
            .await
            .map_err(|e| SidecarError::StorageError(format!("Failed to upsert vectors: {}", e)))?;

        Ok(())
    }

    pub async fn search(
        &self,
        query_vector: Vec<f32>,
        limit: usize,
    ) -> Result<Vec<ScoredPoint>, SidecarError> {
        let result = self.client
            .search_points(
                SearchPointsBuilder::new(&self.collection_name, query_vector, limit as u64)
                    .with_payload(true),
            )
            .await
            .map_err(|e| SidecarError::StorageError(format!("Search failed: {}", e)))?;

        Ok(result.result)
    }

    pub async fn delete_collection(&self) -> Result<(), SidecarError> {
        self.client
            .delete_collection(&self.collection_name)
            .await
            .map_err(|e| SidecarError::StorageError(format!("Failed to delete collection: {}", e)))?;

        Ok(())
    }
}

pub async fn create_ephemeral_collection(
    url: &str,
    vector_size: usize,
) -> Result<(QdrantClient, String), SidecarError> {
    let collection_name = format!("temp_{}", uuid::Uuid::new_v4().to_string());
    let client = QdrantClient::new(url, &collection_name).await?;
    client.create_collection(vector_size).await?;
    Ok((client, collection_name))
}
