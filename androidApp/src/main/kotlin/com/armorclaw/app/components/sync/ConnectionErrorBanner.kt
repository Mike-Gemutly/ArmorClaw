package com.armorclaw.app.components.sync
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.armorclaw.shared.ui.theme.*
import kotlinx.coroutines.delay

/**
 * Types of connection errors that can occur
 */
enum class ConnectionErrorType {
    CONNECTION_FAILED,
    CERTIFICATE_ERROR,
    SYNC_CONFLICT,
    RATE_LIMITED,
    UPDATE_REQUIRED,
    AUTHENTICATION_FAILED,
    SERVER_ERROR,
    NETWORK_TIMEOUT
}

/**
 * Connection error information
 */
data class ConnectionError(
    val type: ConnectionErrorType,
    val title: String,
    val message: String,
    val isRecoverable: Boolean = true,
    val retryAfterSeconds: Int? = null,
    val details: String? = null
)

/**
 * Banner that displays at top of screen for persistent connection errors
 * Stays visible until the error is resolved
 */
@Composable
fun ConnectionErrorBanner(
    error: ConnectionError,
    onRetryClick: () -> Unit = {},
    onDismissClick: () -> Unit = {},
    onDetailsClick: () -> Unit = {},
    modifier: Modifier = Modifier,
    isRetrying: Boolean = false
) {
    val config = getErrorConfig(error.type)

    val backgroundColor by animateColorAsState(
        targetValue = config.backgroundColor,
        animationSpec = tween(300),
        label = "background"
    )

    val contentColor by animateColorAsState(
        targetValue = config.contentColor,
        animationSpec = tween(300),
        label = "content"
    )

    Surface(
        modifier = modifier.fillMaxWidth(),
        color = backgroundColor,
        contentColor = contentColor
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Error icon with pulse animation
            Box(
                modifier = Modifier.size(24.dp),
                contentAlignment = Alignment.Center
            ) {
                if (isRetrying) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(20.dp),
                        color = contentColor,
                        strokeWidth = 2.dp
                    )
                } else {
                    val infiniteTransition = rememberInfiniteTransition(label = "pulse")
                    val alpha by infiniteTransition.animateFloat(
                        initialValue = 1f,
                        targetValue = 0.6f,
                        animationSpec = infiniteRepeatable(
                            animation = tween(800, easing = FastOutSlowInEasing),
                            repeatMode = RepeatMode.Reverse
                        ),
                        label = "alpha"
                    )

                    Icon(
                        imageVector = config.icon,
                        contentDescription = null,
                        tint = contentColor.copy(alpha = alpha),
                        modifier = Modifier.size(24.dp)
                    )
                }
            }

            // Error message
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(2.dp)
            ) {
                Text(
                    text = error.title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )

                Text(
                    text = error.message,
                    style = MaterialTheme.typography.bodySmall,
                    color = contentColor.copy(alpha = 0.8f),
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )

                // Retry after info for rate limiting
                error.retryAfterSeconds?.let { seconds ->
                    if (seconds > 0) {
                        Text(
                            text = "Try again in ${seconds}s",
                            style = MaterialTheme.typography.bodySmall,
                            color = contentColor.copy(alpha = 0.6f),
                            maxLines = 1
                        )
                    }
                }
            }

            // Action buttons
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                if (error.details != null) {
                    TextButton(
                        onClick = onDetailsClick,
                        colors = ButtonDefaults.textButtonColors(
                            contentColor = contentColor
                        )
                    ) {
                        Text(
                            text = "Details",
                            style = MaterialTheme.typography.bodySmall
                        )
                    }
                }

                if (error.isRecoverable) {
                    Button(
                        onClick = onRetryClick,
                        enabled = !isRetrying && (error.retryAfterSeconds ?: 0) <= 0,
                        colors = ButtonDefaults.buttonColors(
                            containerColor = contentColor,
                            contentColor = backgroundColor,
                            disabledContainerColor = contentColor.copy(alpha = 0.3f),
                            disabledContentColor = backgroundColor.copy(alpha = 0.5f)
                        ),
                        shape = RoundedCornerShape(8.dp),
                        modifier = Modifier.height(32.dp)
                    ) {
                        if (isRetrying) {
                            CircularProgressIndicator(
                                modifier = Modifier.size(16.dp),
                                color = backgroundColor,
                                strokeWidth = 2.dp
                            )
                        } else {
                            Text(
                                text = "Retry",
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.Medium
                            )
                        }
                    }
                }

                // Dismiss button
                IconButton(
                    onClick = onDismissClick,
                    modifier = Modifier.size(32.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = "Dismiss",
                        tint = contentColor.copy(alpha = 0.7f),
                        modifier = Modifier.size(18.dp)
                    )
                }
            }
        }
    }
}

/**
 * Toast-style notification for transient errors
 * Auto-dismisses after a timeout
 */
@Composable
fun ErrorToast(
    message: String,
    type: ConnectionErrorType = ConnectionErrorType.NETWORK_TIMEOUT,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier,
    durationMs: Long = 4000
) {
    val config = getErrorConfig(type)
    var visible by remember { mutableStateOf(true) }

    LaunchedEffect(Unit) {
        delay(durationMs)
        visible = false
        delay(300) // Wait for exit animation
        onDismiss()
    }

    AnimatedVisibility(
        visible = visible,
        enter = fadeIn() + slideInVertically { it },
        exit = fadeOut() + slideOutVertically { it },
        modifier = modifier
    ) {
        Surface(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(8.dp),
            color = config.backgroundColor,
            contentColor = config.contentColor,
            shadowElevation = 4.dp
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = config.icon,
                    contentDescription = null,
                    tint = config.contentColor,
                    modifier = Modifier.size(20.dp)
                )

                Text(
                    text = message,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.weight(1f)
                )

                IconButton(
                    onClick = {
                        visible = false
                        onDismiss()
                    },
                    modifier = Modifier.size(24.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = "Dismiss",
                        tint = config.contentColor.copy(alpha = 0.7f),
                        modifier = Modifier.size(16.dp)
                    )
                }
            }
        }
    }
}

/**
 * Full-screen error state for critical errors
 */
@Composable
fun ConnectionErrorScreen(
    error: ConnectionError,
    onRetryClick: () -> Unit = {},
    onHelpClick: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val config = getErrorConfig(error.type)

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        // Error icon
        Surface(
            modifier = Modifier.size(80.dp),
            shape = RoundedCornerShape(40.dp),
            color = config.backgroundColor
        ) {
            Box(
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = config.icon,
                    contentDescription = null,
                    tint = config.contentColor,
                    modifier = Modifier.size(40.dp)
                )
            }
        }

        Spacer(modifier = Modifier.height(24.dp))

        // Error title
        Text(
            text = error.title,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = OnBackground
        )

        Spacer(modifier = Modifier.height(8.dp))

        // Error message
        Text(
            text = error.message,
            style = MaterialTheme.typography.bodyLarge,
            color = OnBackground.copy(alpha = 0.7f)
        )

        if (error.details != null) {
            Spacer(modifier = Modifier.height(16.dp))

            Surface(
                shape = RoundedCornerShape(8.dp),
                color = SurfaceVariant
            ) {
                Text(
                    text = error.details,
                    style = MaterialTheme.typography.bodyMedium,
                    color = OnBackground.copy(alpha = 0.6f),
                    modifier = Modifier.padding(16.dp)
                )
            }
        }

        Spacer(modifier = Modifier.height(32.dp))

        // Action buttons
        Row(
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            if (error.isRecoverable) {
                Button(
                    onClick = onRetryClick,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = BrandPurple
                    ),
                    modifier = Modifier.fillMaxWidth(0.5f)
                ) {
                    Icon(
                        imageVector = Icons.Default.Refresh,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp)
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Try Again")
                }
            }

            OutlinedButton(
                onClick = onHelpClick,
                modifier = Modifier.fillMaxWidth(if (error.isRecoverable) 0.5f else 1f)
            ) {
                Icon(
                    imageVector = Icons.Outlined.Info,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Get Help")
            }
        }
    }
}

/**
 * Error details dialog for viewing full error information
 */
@Composable
fun ErrorDetailsDialog(
    error: ConnectionError,
    onDismiss: () -> Unit,
    onRetry: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val config = getErrorConfig(error.type)

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(
            dismissOnBackPress = true,
            dismissOnClickOutside = true
        )
    ) {
        Surface(
            modifier = modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            color = Surface
        ) {
            Column(
                modifier = Modifier.padding(24.dp)
            ) {
                // Header
                Row(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Surface(
                        modifier = Modifier.size(40.dp),
                        shape = RoundedCornerShape(20.dp),
                        color = config.backgroundColor
                    ) {
                        Box(contentAlignment = Alignment.Center) {
                            Icon(
                                imageVector = config.icon,
                                contentDescription = null,
                                tint = config.contentColor,
                                modifier = Modifier.size(20.dp)
                            )
                        }
                    }

                    Text(
                        text = error.title,
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.Bold
                    )
                }

                Spacer(modifier = Modifier.height(16.dp))

                // Error message
                Text(
                    text = error.message,
                    style = MaterialTheme.typography.bodyLarge,
                    color = OnBackground.copy(alpha = 0.8f)
                )

                // Details
                if (error.details != null) {
                    Spacer(modifier = Modifier.height(16.dp))

                    Text(
                        text = "Technical Details",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Medium,
                        color = OnBackground.copy(alpha = 0.6f)
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = SurfaceVariant
                    ) {
                        Text(
                            text = error.details,
                            style = MaterialTheme.typography.bodyMedium,
                            color = OnBackground.copy(alpha = 0.7f),
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(12.dp)
                                .verticalScroll(rememberScrollState())
                        )
                    }
                }

                // Retry after countdown
                error.retryAfterSeconds?.let { seconds ->
                    if (seconds > 0) {
                        Spacer(modifier = Modifier.height(16.dp))

                        Row(
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Default.DateRange,
                                contentDescription = null,
                                tint = StatusWarning,
                                modifier = Modifier.size(16.dp)
                            )
                            Text(
                                text = "Please wait ${seconds} seconds before retrying",
                                style = MaterialTheme.typography.bodyMedium,
                                color = StatusWarning
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.height(24.dp))

                // Actions
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(12.dp, Alignment.End)
                ) {
                    TextButton(onClick = onDismiss) {
                        Text("Close")
                    }

                    if (error.isRecoverable) {
                        Button(
                            onClick = {
                                onDismiss()
                                onRetry()
                            },
                            enabled = (error.retryAfterSeconds ?: 0) <= 0,
                            colors = ButtonDefaults.buttonColors(
                                containerColor = BrandPurple
                            )
                        ) {
                            Text("Retry")
                        }
                    }
                }
            }
        }
    }
}

// ========== Configuration ==========

private data class ErrorConfig(
    val icon: ImageVector,
    val backgroundColor: Color,
    val contentColor: Color
)

@Composable
private fun getErrorConfig(type: ConnectionErrorType): ErrorConfig {
    return when (type) {
        ConnectionErrorType.CONNECTION_FAILED -> ErrorConfig(
            icon = Icons.Default.Warning,
            backgroundColor = BrandRed.copy(alpha = 0.1f),
            contentColor = BrandRedDark
        )
        ConnectionErrorType.CERTIFICATE_ERROR -> ErrorConfig(
            icon = Icons.Default.Warning,
            backgroundColor = StatusWarning.copy(alpha = 0.1f),
            contentColor = Color(0xFFB45309)
        )
        ConnectionErrorType.SYNC_CONFLICT -> ErrorConfig(
            icon = Icons.Default.Refresh,
            backgroundColor = StatusWarning.copy(alpha = 0.1f),
            contentColor = Color(0xFFB45309)
        )
        ConnectionErrorType.RATE_LIMITED -> ErrorConfig(
            icon = Icons.Default.Info,
            backgroundColor = BrandPurple.copy(alpha = 0.1f),
            contentColor = BrandPurpleDark
        )
        ConnectionErrorType.UPDATE_REQUIRED -> ErrorConfig(
            icon = Icons.Default.Add,
            backgroundColor = BrandBlue.copy(alpha = 0.1f),
            contentColor = BrandBlueDark
        )
        ConnectionErrorType.AUTHENTICATION_FAILED -> ErrorConfig(
            icon = Icons.Default.Lock,
            backgroundColor = BrandRed.copy(alpha = 0.1f),
            contentColor = BrandRedDark
        )
        ConnectionErrorType.SERVER_ERROR -> ErrorConfig(
            icon = Icons.Default.Warning,
            backgroundColor = BrandRed.copy(alpha = 0.1f),
            contentColor = BrandRedDark
        )
        ConnectionErrorType.NETWORK_TIMEOUT -> ErrorConfig(
            icon = Icons.Default.Info,
            backgroundColor = StatusWarning.copy(alpha = 0.1f),
            contentColor = Color(0xFFB45309)
        )
    }
}

// ========== Factory Functions ==========

object ConnectionErrors {
    fun connectionFailed(details: String? = null) = ConnectionError(
        type = ConnectionErrorType.CONNECTION_FAILED,
        title = "Can't Connect",
        message = "Couldn't reach the server. Check your internet connection.",
        isRecoverable = true,
        details = details
    )

    fun certificateError(details: String? = null) = ConnectionError(
        type = ConnectionErrorType.CERTIFICATE_ERROR,
        title = "Security Alert",
        message = "The server's certificate has changed. This could indicate a security issue.",
        isRecoverable = false,
        details = details
    )

    fun syncConflict(details: String? = null) = ConnectionError(
        type = ConnectionErrorType.SYNC_CONFLICT,
        title = "Sync Conflict",
        message = "Messages were modified on another device. Pull down to refresh.",
        isRecoverable = true,
        details = details
    )

    fun rateLimited(retryAfter: Int = 30) = ConnectionError(
        type = ConnectionErrorType.RATE_LIMITED,
        title = "Too Many Requests",
        message = "You're making requests too quickly. Please wait a moment.",
        isRecoverable = true,
        retryAfterSeconds = retryAfter
    )

    fun updateRequired() = ConnectionError(
        type = ConnectionErrorType.UPDATE_REQUIRED,
        title = "Update Required",
        message = "A new version of ArmorClaw is available. Please update to continue.",
        isRecoverable = false
    )

    fun authenticationFailed() = ConnectionError(
        type = ConnectionErrorType.AUTHENTICATION_FAILED,
        title = "Authentication Failed",
        message = "Your session has expired. Please log in again.",
        isRecoverable = true
    )

    fun serverError(code: Int? = null) = ConnectionError(
        type = ConnectionErrorType.SERVER_ERROR,
        title = "Server Error",
        message = "The server encountered an error. Please try again later.",
        isRecoverable = true,
        details = code?.let { "Error code: $it" }
    )

    fun networkTimeout() = ConnectionError(
        type = ConnectionErrorType.NETWORK_TIMEOUT,
        title = "Connection Timeout",
        message = "The request took too long. Check your connection and try again.",
        isRecoverable = true
    )
}

// ========== Preview Composables ==========

@Composable
fun ConnectionErrorBannerPreview() {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        ConnectionErrorBanner(
            error = ConnectionErrors.connectionFailed(),
            onRetryClick = {}
        )
        ConnectionErrorBanner(
            error = ConnectionErrors.certificateError("Certificate SHA-256 fingerprint changed"),
            onRetryClick = {}
        )
        ConnectionErrorBanner(
            error = ConnectionErrors.rateLimited(30),
            onRetryClick = {}
        )
    }
}
