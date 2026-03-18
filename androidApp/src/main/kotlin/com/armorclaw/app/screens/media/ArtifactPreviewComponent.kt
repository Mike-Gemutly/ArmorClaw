package com.armorclaw.app.screens.media

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.domain.features.ArtifactRenderer
import com.armorclaw.shared.domain.features.ArtifactType
import com.armorclaw.shared.domain.features.RenderedArtifact
import com.armorclaw.shared.domain.features.RenderFormat
import com.armorclaw.shared.domain.model.AppResult
import kotlinx.coroutines.launch

/**
 * Artifact preview component that renders artifact content using artifact renderers
 *
 * This component can be used alongside FilePreviewScreen to provide content rendering
 * without modifying the existing FilePreviewScreen structure
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ArtifactPreviewComponent(
    artifactId: String,
    artifactType: ArtifactType,
    artifactRenderer: ArtifactRenderer,
    modifier: Modifier = Modifier
) {
    var renderedArtifact by remember { mutableStateOf<RenderedArtifact?>(null) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(artifactId, artifactType) {
        isLoading = true
        error = null
        
        val result = artifactRenderer.renderPreview(artifactId, null)
        when (result) {
            is AppResult.Success -> {
                renderedArtifact = result.data
                isLoading = false
            }
            is AppResult.Error -> {
                error = result.error.message
                isLoading = false
            }
            is AppResult.Loading -> { }
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .padding(16.dp),
        contentAlignment = Alignment.Center
    ) {
        when {
            isLoading -> {
                CircularProgressIndicator()
            }
            error != null -> {
                Text(
                    text = "Error loading artifact: $error",
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodyLarge
                )
            }
            renderedArtifact != null -> {
                when (renderedArtifact!!.format) {
                    RenderFormat.JSON -> {
                        JsonPreviewContent(
                            content = renderedArtifact!!.content,
                            modifier = Modifier.fillMaxSize()
                        )
                    }
                    RenderFormat.PLAIN_TEXT -> {
                        TextPreviewContent(
                            content = renderedArtifact!!.content,
                            modifier = Modifier.fillMaxSize()
                        )
                    }
                    RenderFormat.MARKDOWN -> {
                        TextPreviewContent(
                            content = renderedArtifact!!.content,
                            modifier = Modifier.fillMaxSize()
                        )
                    }
                    else -> {
                        Text(
                            text = "Preview not available for this format",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            style = MaterialTheme.typography.bodyLarge
                        )
                    }
                }
            }
            else -> {
                Text(
                    text = "No artifact to preview",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    style = MaterialTheme.typography.bodyLarge
                )
            }
        }
    }
}

/**
 * JSON preview content with syntax highlighting
 */
@Composable
private fun JsonPreviewContent(
    content: String,
    modifier: Modifier = Modifier
) {
    // In a real implementation, you would parse and render the JSON with proper syntax highlighting
    // For now, we'll just display it as formatted text
    Text(
        text = content,
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        color = MaterialTheme.colorScheme.onSurface,
        fontSize = 14.sp,
        lineHeight = 1.5.sp
    )
}

/**
 * Text preview content for plain text and markdown
 */
@Composable
private fun TextPreviewContent(
    content: String,
    modifier: Modifier = Modifier
) {
    Text(
        text = content,
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        color = MaterialTheme.colorScheme.onSurface,
        fontSize = 14.sp,
        lineHeight = 1.5.sp
    )
}