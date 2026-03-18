package com.armorclaw.shared.domain.features

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.OperationContext
import kotlinx.coroutines.flow.Flow

/**
 * Service interface for artifact rendering
 *
 * Provides rendering, preview generation, and export
 * capabilities for various artifact types (documents, images, code, etc.).
 *
 * TODO: Implement artifact renderer registry
 * TODO: Add export format support
 * TODO: Integrate with platform-specific rendering APIs
 */
interface ArtifactRenderer {

    /**
     * Render artifact to preview
     * @param artifactId The artifact ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun renderPreview(
        artifactId: String,
        context: OperationContext? = null
    ): AppResult<RenderedArtifact>

    /**
     * Render full artifact
     * @param artifactId The artifact ID
     * @param options Rendering options
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun renderFull(
        artifactId: String,
        options: RenderOptions,
        context: OperationContext? = null
    ): AppResult<RenderedArtifact>

    /**
     * Export artifact
     * @param artifactId The artifact ID
     * @param format Export format
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun exportArtifact(
        artifactId: String,
        format: ExportFormat,
        context: OperationContext? = null
    ): AppResult<ExportedArtifact>

    /**
     * Get supported formats for artifact type
     * @param artifactType The artifact type
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun getSupportedFormats(
        artifactType: ArtifactType,
        context: OperationContext? = null
    ): AppResult<List<ExportFormat>>

    /**
     * Check if artifact can be rendered
     * @param artifactId The artifact ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun canRender(
        artifactId: String,
        context: OperationContext? = null
    ): AppResult<Boolean>

    /**
     * Observe rendering progress (reactive)
     * @param artifactId The artifact ID
     */
    fun observeRenderingProgress(artifactId: String): Flow<RenderingProgress>

    /**
     * Cancel ongoing rendering
     * @param artifactId The artifact ID
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun cancelRendering(
        artifactId: String,
        context: OperationContext? = null
    ): AppResult<Unit>
}

/**
 * Rendered artifact result
 *
 * TODO: Add metadata support
 * TODO: Add thumbnail support
 */
@kotlinx.serialization.Serializable
data class RenderedArtifact(
    val artifactId: String,
    val type: ArtifactType,
    val content: String,
    val format: RenderFormat,
    val size: Long
)

/**
 * Exported artifact result
 */
@kotlinx.serialization.Serializable
data class ExportedArtifact(
    val artifactId: String,
    val format: ExportFormat,
    val data: String,
    val size: Long
)

/**
 * Rendering options
 *
 * TODO: Add quality settings
 * TODO: Add dimension settings
 */
@kotlinx.serialization.Serializable
data class RenderOptions(
    val includeMetadata: Boolean = true,
    val optimizeSize: Boolean = false
)

/**
 * Rendering progress
 */
@kotlinx.serialization.Serializable
data class RenderingProgress(
    val artifactId: String,
    val status: RenderingStatus,
    val progress: Float = 0f,
    val bytesProcessed: Long = 0,
    val totalBytes: Long = 0
)

/**
 * Rendering status
 */
@kotlinx.serialization.Serializable
enum class RenderingStatus {
    PENDING,
    PROCESSING,
    COMPLETED,
    FAILED,
    CANCELLED
}

/**
 * Artifact types
 */
@kotlinx.serialization.Serializable
enum class ArtifactType {
    DOCUMENT,
    IMAGE,
    CODE,
    CHAT,
    WORKFLOW,
    UNKNOWN
}

/**
 * Render format
 */
@kotlinx.serialization.Serializable
enum class RenderFormat {
    HTML,
    MARKDOWN,
    PLAIN_TEXT,
    JSON,
    PREVIEW
}

/**
 * Export format
 */
@kotlinx.serialization.Serializable
enum class ExportFormat {
    PDF,
    PNG,
    JPG,
    DOCX,
    TXT,
    JSON,
    HTML
}
