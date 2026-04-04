use crate::error::SidecarError;
use std::collections::HashMap;
use serde_json::Value;
use qdrant_client::qdrant::{CreateCollection, VectorsConfig, PointStruct, UpsertPoints, SearchPoints, ScoredPoint};

pub struct QdrantClient {
    client: Qdrant,
    collection_name: String,
}

impl QdrantClient {
    pub async fn new(url: &str, collection_name: &str) -> Result<Self, SidecarError> {
        let client = Qdrant::from_url(url)
            .map_err(|e| SidecarError::DatabaseError(format!("Failed to connect to Qdrant: {}", e)))?;

        Ok(Self {
            client,
            collection_name: collection_name.to_string(),
        })
    }

    pub async fn create_collection(&self, vector_size: usize) -> Result<(), SidecarError> {
        let vector_params = qdrant_client::qdrant::VectorParams {
            size: vector_size as u64,
            distance: Some(qdrant_client::qdrant::Distance::Cosine.into()),
            hnsw_config: None,
            quantization_config: None,
            on_disk: None,
        };

        let collection_params = CreateCollection {
            collection_name: self.collection_name.clone(),
            vectors_config: Some(VectorsConfig::Params(vector_params)),
            hnsw_config: None,
            optimizers_config: None,
            wal_config: None,
            quantization_config: None,
            init_from: None,
            on_disk_payload: None,
            replication_factor: None,
            write_consistency_factor: None,
            sparse_vectors_config: None,
        };

        self.client.create_collection(collection_params).await
            .map_err(|e| SidecarError::DatabaseError(format!("Failed to create collection: {}", e)))?;

        Ok(())
    }

    pub async fn upsert_vectors(
        &self,
        id: String,
        vectors: Vec<f32>,
        payload: HashMap<String, Value>,
    ) -> Result<(), SidecarError> {
        let points = vec![PointStruct {
            id: qdrant_client::qdrant::PointId::from(id),
            vector: Some(qdrant_client::qdrant::Vector::from(vectors)),
            payload: payload.into_iter().map(|(k, v)| (k, Some(v.into()))).collect(),
            shard_key: None,
        }];

        let upsert_request = UpsertPoints {
            collection_name: self.collection_name.clone(),
            points,
            wait: None,
            ordering: None,
        };

        self.client.upsert_points_blocking(upsert_request).await
            .map_err(|e| SidecarError::DatabaseError(format!("Failed to upsert vectors: {}", e)))?;

        Ok(())
    }

    pub async fn search(
        &self,
        query_vector: Vec<f32>,
        limit: usize,
    ) -> Result<Vec<ScoredPoint>, SidecarError> {
        let search_request = SearchPoints {
            collection_name: self.collection_name.clone(),
            vector: Some(qdrant_client::qdrant::Vector::from(query_vector)),
            limit: limit as u64,
            offset: None,
            with_payload: None,
            with_vectors: None,
            score_threshold: None,
            filter: None,
            params: None,
            hnsw_ef: None,
        };

        let result = self.client.search_points(search_request).await
            .map_err(|e| SidecarError::DatabaseError(format!("Search failed: {}", e)))?;

        Ok(result.result)
    }

    pub async fn delete_collection(&self) -> Result<(), SidecarError> {
        self.client.delete_collection(self.collection_name.clone()).await
            .map_err(|e| SidecarError::DatabaseError(format!("Failed to delete collection: {}", e)))?;

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
