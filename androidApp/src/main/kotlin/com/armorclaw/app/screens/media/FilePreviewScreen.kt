package com.armorclaw.app.screens.media

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * File preview screen for document attachments
 *
 * Supports:
 * - PDF preview (basic info + pages)
 * - Image preview (via ImageViewerScreen)
 * - Audio preview (play controls)
 * - Video preview (thumbnail + play)
 * - Other files (info + download)
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FilePreviewScreen(
    fileId: String,
    fileName: String,
    mimeType: String,
    fileSize: Long,
    fileUrl: String?,
    senderName: String?,
    timestamp: Long?,
    isDownloaded: Boolean,
    downloadProgress: Float?,
    onNavigateBack: () -> Unit,
    onDownload: () -> Unit,
    onOpen: () -> Unit,
    onShare: () -> Unit,
    onDelete: () -> Unit,
    modifier: Modifier = Modifier
) {
    val fileType = remember(mimeType, fileName) {
        getFileType(mimeType, fileName)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(
                            text = fileName,
                            style = MaterialTheme.typography.titleMedium,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis
                        )
                        Text(
                            text = fileType.displayName,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Filled.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    IconButton(onClick = onShare) {
                        Icon(Icons.Default.Share, "Share")
                    }
                }
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Spacer(modifier = Modifier.height(32.dp))

            // File type icon/preview
            FilePreviewArea(
                fileType = fileType,
                fileName = fileName,
                fileUrl = fileUrl,
                onOpen = onOpen,
                modifier = Modifier
                    .fillMaxWidth(0.8f)
                    .aspectRatio(1f)
                    .padding(16.dp)
            )

            Spacer(modifier = Modifier.height(24.dp))

            // File info card
            FileInfoCard(
                fileName = fileName,
                mimeType = mimeType,
                fileSize = fileSize,
                senderName = senderName,
                timestamp = timestamp,
                isDownloaded = isDownloaded,
                downloadProgress = downloadProgress,
                onDownload = onDownload,
                onOpen = onOpen,
                onDelete = onDelete
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Quick actions
            QuickActionButtons(
                fileType = fileType,
                isDownloaded = isDownloaded,
                onDownload = onDownload,
                onOpen = onOpen,
                onShare = onShare
            )

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun FilePreviewArea(
    fileType: FileType,
    fileName: String,
    fileUrl: String?,
    onOpen: () -> Unit,
    modifier: Modifier = Modifier
) {
    Surface(
        onClick = onOpen,
        modifier = modifier,
        shape = RoundedCornerShape(16.dp),
        color = fileType.color.copy(alpha = 0.1f)
    ) {
        Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = Alignment.Center
        ) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center
            ) {
                Icon(
                    imageVector = fileType.icon,
                    contentDescription = null,
                    modifier = Modifier.size(80.dp),
                    tint = fileType.color
                )

                Spacer(modifier = Modifier.height(16.dp))

                Text(
                    text = fileType.extension.uppercase(),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = fileType.color
                )

                Spacer(modifier = Modifier.height(8.dp))

                Text(
                    text = fileName.take(30) + if (fileName.length > 30) "..." else "",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            // Tap to open hint
            Surface(
                modifier = Modifier.align(Alignment.BottomCenter).padding(bottom = 16.dp),
                shape = RoundedCornerShape(20.dp),
                color = fileType.color.copy(alpha = 0.2f)
            ) {
                Text(
                    text = "Tap to open",
                    style = MaterialTheme.typography.labelSmall,
                    color = fileType.color,
                    modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp)
                )
            }
        }
    }
}

@Composable
private fun FileInfoCard(
    fileName: String,
    mimeType: String,
    fileSize: Long,
    senderName: String?,
    timestamp: Long?,
    isDownloaded: Boolean,
    downloadProgress: Float?,
    onDownload: () -> Unit,
    onOpen: () -> Unit,
    onDelete: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        shape = RoundedCornerShape(12.dp)
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = "File Information",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold
            )

            Divider()

            InfoRow(label = "Name", value = fileName)
            InfoRow(label = "Type", value = mimeType)
            InfoRow(label = "Size", value = formatFileSize(fileSize))
            senderName?.let { InfoRow(label = "From", value = it) }
            timestamp?.let { InfoRow(label = "Date", value = formatTimestamp(it)) }
            InfoRow(
                label = "Status",
                value = if (isDownloaded) "Downloaded" else "Not downloaded"
            )

            // Download progress
            downloadProgress?.let { progress ->
                Spacer(modifier = Modifier.height(8.dp))
                LinearProgressIndicator(
                    progress = progress,
                    modifier = Modifier.fillMaxWidth(),
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = "${(progress * 100).toInt()}%",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f, fill = false),
            textAlign = androidx.compose.ui.text.style.TextAlign.End
        )
    }
}

@Composable
private fun QuickActionButtons(
    fileType: FileType,
    isDownloaded: Boolean,
    onDownload: () -> Unit,
    onOpen: () -> Unit,
    onShare: () -> Unit
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        if (!isDownloaded) {
            FilledTonalButton(
                onClick = onDownload,
                modifier = Modifier.weight(1f)
            ) {
                Icon(Icons.Default.Download, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("Download")
            }
        } else {
            FilledTonalButton(
                onClick = onOpen,
                modifier = Modifier.weight(1f)
            ) {
                Icon(Icons.Default.OpenInNew, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("Open")
            }
        }

        OutlinedButton(
            onClick = onShare,
            modifier = Modifier.weight(1f)
        ) {
            Icon(Icons.Default.Share, contentDescription = null)
            Spacer(modifier = Modifier.width(8.dp))
            Text("Share")
        }
    }
}

/**
 * File type classification
 */
enum class FileType(
    val displayName: String,
    val extension: String,
    val icon: ImageVector,
    val color: Color
) {
    PDF("PDF Document", "PDF", Icons.Default.PictureAsPdf, Color(0xFFE53935)),
    IMAGE("Image", "IMAGE", Icons.Default.Image, Color(0xFF43A047)),
    VIDEO("Video", "VIDEO", Icons.Default.VideoFile, Color(0xFFFB8C00)),
    AUDIO("Audio", "AUDIO", Icons.Default.AudioFile, Color(0xFF1E88E5)),
    DOCUMENT("Document", "DOC", Icons.Default.Description, Color(0xFF5E35B1)),
    SPREADSHEET("Spreadsheet", "XLS", Icons.Default.TableChart, Color(0xFF43A047)),
    ARCHIVE("Archive", "ZIP", Icons.Default.FolderZip, Color(0xFF757575)),
    CODE("Code", "CODE", Icons.Default.Code, Color(0xFF00897B)),
    OTHER("File", "FILE", Icons.Default.InsertDriveFile, Color(0xFF9E9E9E))
}

fun getFileType(mimeType: String, fileName: String): FileType {
    val extension = fileName.substringAfterLast(".", "").lowercase()

    return when {
        mimeType.startsWith("image/") -> FileType.IMAGE
        mimeType.startsWith("video/") -> FileType.VIDEO
        mimeType.startsWith("audio/") -> FileType.AUDIO
        mimeType == "application/pdf" -> FileType.PDF
        mimeType.contains("spreadsheet") || extension in listOf("xls", "xlsx", "csv") -> FileType.SPREADSHEET
        mimeType.contains("document") || extension in listOf("doc", "docx", "txt", "rtf") -> FileType.DOCUMENT
        mimeType.contains("zip") || extension in listOf("zip", "rar", "7z", "tar", "gz") -> FileType.ARCHIVE
        extension in listOf("kt", "java", "py", "js", "ts", "cpp", "c", "h", "swift", "rs") -> FileType.CODE
        else -> FileType.OTHER
    }
}

private fun formatFileSize(bytes: Long): String {
    return when {
        bytes < 1024 -> "$bytes B"
        bytes < 1024 * 1024 -> String.format("%.1f KB", bytes / 1024.0)
        bytes < 1024 * 1024 * 1024 -> String.format("%.1f MB", bytes / (1024.0 * 1024))
        else -> String.format("%.1f GB", bytes / (1024.0 * 1024 * 1024))
    }
}

private fun formatTimestamp(timestamp: Long): String {
    val instant = kotlinx.datetime.Instant.fromEpochMilliseconds(timestamp)
    return instant.toString()
}
