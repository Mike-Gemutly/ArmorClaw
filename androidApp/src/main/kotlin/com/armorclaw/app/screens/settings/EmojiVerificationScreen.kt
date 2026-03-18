package com.armorclaw.app.screens.settings
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.domain.model.EmojiInfo
import com.armorclaw.shared.domain.model.VerificationState
import com.armorclaw.shared.ui.theme.*

/**
 * Screen for emoji verification flow
 * Users compare emoji sequences to verify device trust
 */
@Composable
fun EmojiVerificationScreen(
    state: VerificationState,
    deviceName: String,
    onConfirmMatch: () -> Unit,
    onDenyMatch: () -> Unit,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier
) {
    // Track cancellation state to await server response
    var isCancelling by remember { mutableStateOf(false) }

    // Wrapped cancel handler that shows loading state
    val handleCancel: () -> Unit = {
        isCancelling = true
        onCancel()
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Show cancelling overlay
        if (isCancelling) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(24.dp),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(48.dp),
                        color = BrandPurple,
                        strokeWidth = 4.dp
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = "Cancelling verification...",
                        style = MaterialTheme.typography.bodyMedium,
                        color = OnBackground.copy(alpha = 0.7f)
                    )
                }
            }
        } else {
            // Header
            VerificationHeader(
                state = state,
                deviceName = deviceName,
                onCancel = handleCancel
            )

        Spacer(modifier = Modifier.height(32.dp))

        // Main content based on state
        when (state) {
            is VerificationState.Unverified -> {
                VerificationWaitingState()
            }
            is VerificationState.Requested -> {
                VerificationWaitingState()
            }
            is VerificationState.Ready -> {
                VerificationWaitingState()
            }
            is VerificationState.EmojiChallenge -> {
                EmojiChallengeContent(
                    emojis = state.emojis,
                    onConfirmMatch = onConfirmMatch,
                    onDenyMatch = onDenyMatch
                )
            }
            is VerificationState.CodeChallenge -> {
                CodeChallengeContent(
                    code = state.code,
                    onConfirmMatch = onConfirmMatch,
                    onDenyMatch = onDenyMatch
                )
            }
            is VerificationState.Verifying -> {
                VerificationWaitingState()
            }
            is VerificationState.Verified -> {
                VerificationSuccessContent(
                    onDone = handleCancel
                )
            }
            is VerificationState.Cancelled -> {
                VerificationFailedContent(
                    reason = state.reason,
                    onRetry = handleCancel
                )
            }
            is VerificationState.Failed -> {
                VerificationFailedContent(
                    reason = state.reason,
                    onRetry = handleCancel
                )
            }
        }
        }
    }
}

@Composable
private fun VerificationHeader(
    state: VerificationState,
    deviceName: String,
    onCancel: () -> Unit
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        IconButton(onClick = onCancel) {
            Icon(
                imageVector = Icons.Default.Close,
                contentDescription = "Cancel",
                tint = OnBackground.copy(alpha = 0.7f)
            )
        }

        Column(
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(
                text = "Verify Device",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                color = OnBackground
            )
            Text(
                text = deviceName,
                style = MaterialTheme.typography.bodySmall,
                color = OnBackground.copy(alpha = 0.6f)
            )
        }

        // Spacer for symmetry
        Spacer(modifier = Modifier.size(48.dp))
    }
}

@Composable
private fun VerificationWaitingState() {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(64.dp),
            color = BrandPurple,
            strokeWidth = 4.dp
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = "Waiting for verification to start...",
            style = MaterialTheme.typography.bodyLarge,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )
    }
}

@Composable
private fun EmojiChallengeContent(
    emojis: List<EmojiInfo>,
    onConfirmMatch: () -> Unit,
    onDenyMatch: () -> Unit
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Instruction
        Text(
            text = "Compare these emojis",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Medium,
            color = OnBackground
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Make sure the emojis match exactly on both devices",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(32.dp))

        // Emoji display
        EmojiDisplay(emojis = emojis)

        Spacer(modifier = Modifier.height(32.dp))

        // Security note
        SecurityNote()

        Spacer(modifier = Modifier.height(32.dp))

        // Action buttons
        VerificationButtons(
            onConfirm = onConfirmMatch,
            onDeny = onDenyMatch
        )
    }
}

@Composable
private fun EmojiDisplay(
    emojis: List<EmojiInfo>
) {
    LazyRow(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        contentPadding = PaddingValues(horizontal = 16.dp)
    ) {
        items(emojis) { emojiInfo ->
            EmojiCard(emojiInfo = emojiInfo)
        }
    }
}

@Composable
private fun EmojiCard(
    emojiInfo: EmojiInfo
) {
    var animated by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        animated = true
    }

    val scale by animateFloatAsState(
        targetValue = if (animated) 1f else 0.5f,
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioMediumBouncy,
            stiffness = Spring.StiffnessLow
        ),
        label = "scale"
    )

    Column(
        modifier = Modifier
            .scale(scale)
            .width(80.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(BrandPurple.copy(alpha = 0.1f))
            .border(1.dp, BrandPurple.copy(alpha = 0.3f), RoundedCornerShape(12.dp))
            .padding(vertical = 16.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = emojiInfo.emoji,
            fontSize = 36.sp,
            modifier = Modifier.padding(bottom = 8.dp)
        )

        Text(
            text = emojiInfo.description,
            style = MaterialTheme.typography.bodySmall,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center,
            maxLines = 2,
            modifier = Modifier.padding(horizontal = 8.dp)
        )
    }
}

@Composable
private fun CodeChallengeContent(
    code: String,
    onConfirmMatch: () -> Unit,
    onDenyMatch: () -> Unit
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Instruction
        Text(
            text = "Compare the code",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Medium,
            color = OnBackground
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Make sure the code matches exactly on both devices",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(32.dp))

        // Code display
        CodeDisplay(code = code)

        Spacer(modifier = Modifier.height(32.dp))

        // Security note
        SecurityNote()

        Spacer(modifier = Modifier.height(32.dp))

        // Action buttons
        VerificationButtons(
            onConfirm = onConfirmMatch,
            onDeny = onDenyMatch
        )
    }
}

@Composable
private fun CodeDisplay(
    code: String
) {
    val codeGroups = code.chunked(4)

    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        codeGroups.forEach { group ->
            Surface(
                shape = RoundedCornerShape(8.dp),
                color = BrandPurple.copy(alpha = 0.1f),
                border = androidx.compose.foundation.BorderStroke(1.dp, BrandPurple.copy(alpha = 0.3f))
            ) {
                Text(
                    text = group,
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    color = BrandPurple,
                    letterSpacing = 4.sp,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)
                )
            }
        }
    }
}

@Composable
private fun SecurityNote() {
    Surface(
        shape = RoundedCornerShape(8.dp),
        color = StatusWarning.copy(alpha = 0.1f)
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Icon(
                imageVector = Icons.Default.Security,
                contentDescription = null,
                tint = StatusWarning,
                modifier = Modifier.size(20.dp)
            )

            Text(
                text = "Only confirm if you can verify the other device",
                style = MaterialTheme.typography.bodySmall,
                color = StatusWarning
            )
        }
    }
}

@Composable
private fun VerificationButtons(
    onConfirm: () -> Unit,
    onDeny: () -> Unit
) {
    Column(
        modifier = Modifier.fillMaxWidth(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Button(
            onClick = onConfirm,
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(
                containerColor = BrandGreen
            )
        ) {
            Icon(
                imageVector = Icons.Default.Check,
                contentDescription = null,
                modifier = Modifier.size(24.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "They Match",
                style = MaterialTheme.typography.labelLarge,
                fontWeight = FontWeight.Medium
            )
        }

        OutlinedButton(
            onClick = onDeny,
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.outlinedButtonColors(
                contentColor = StatusError
            ),
            border = androidx.compose.foundation.BorderStroke(1.dp, StatusError)
        ) {
            Icon(
                imageVector = Icons.Default.Close,
                contentDescription = null,
                modifier = Modifier.size(24.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "They Don't Match",
                style = MaterialTheme.typography.labelLarge,
                fontWeight = FontWeight.Medium
            )
        }
    }
}

@Composable
private fun VerificationSuccessContent(
    onDone: () -> Unit
) {
    var visible by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        visible = true
    }

    AnimatedVisibility(
        visible = visible,
        enter = fadeIn() + scaleIn(
            animationSpec = spring(
                dampingRatio = Spring.DampingRatioMediumBouncy,
                stiffness = Spring.StiffnessLow
            )
        )
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Success icon
            Box(
                modifier = Modifier
                    .size(100.dp)
                    .clip(CircleShape)
                    .background(BrandGreen.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = Icons.Default.VerifiedUser,
                    contentDescription = null,
                    tint = BrandGreen,
                    modifier = Modifier.size(56.dp)
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            Text(
                text = "Verification Complete",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                color = OnBackground
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "This device has been verified and is now trusted",
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.6f),
                textAlign = TextAlign.Center
            )

            Spacer(modifier = Modifier.height(32.dp))

            Button(
                onClick = onDone,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                shape = RoundedCornerShape(12.dp),
                colors = ButtonDefaults.buttonColors(
                    containerColor = BrandGreen
                )
            ) {
                Text(
                    text = "Done",
                    style = MaterialTheme.typography.labelLarge,
                    fontWeight = FontWeight.Medium
                )
            }
        }
    }
}

@Composable
private fun VerificationFailedContent(
    reason: String,
    onRetry: () -> Unit
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Error icon
        Box(
            modifier = Modifier
                .size(100.dp)
                .clip(CircleShape)
                .background(StatusError.copy(alpha = 0.15f)),
            contentAlignment = Alignment.Center
        ) {
            Icon(
                imageVector = Icons.Default.Error,
                contentDescription = null,
                tint = StatusError,
                modifier = Modifier.size(56.dp)
            )
        }

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = "Verification Failed",
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            color = OnBackground
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = reason,
            style = MaterialTheme.typography.bodyMedium,
            color = StatusError,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "The device could not be verified. Please try again.",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(32.dp))

        Button(
            onClick = onRetry,
            modifier = Modifier
                .fillMaxWidth()
                .height(56.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(
                containerColor = BrandPurple
            )
        ) {
            Icon(
                imageVector = Icons.Default.Refresh,
                contentDescription = null,
                modifier = Modifier.size(24.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = "Try Again",
                style = MaterialTheme.typography.labelLarge,
                fontWeight = FontWeight.Medium
            )
        }
    }
}
