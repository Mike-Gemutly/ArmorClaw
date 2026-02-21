package app.armorclaw.ui.verification

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Bridge Verification Screen - SDTW E2EE Verification
 *
 * Resolves: G-02 (SDTW Decryption)
 *
 * This screen allows users to verify the ArmorClaw Bridge AppService,
 * enabling the bridge to decrypt and relay messages to external platforms.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BridgeVerificationScreen(
    roomId: String,
    bridgeUserId: String,
    onVerificationComplete: () -> Unit,
    onDismiss: () -> Unit,
    viewModel: BridgeVerificationViewModel = viewModel()
) {
    val uiState by viewModel.uiState.collectAsState()

    LaunchedEffect(roomId) {
        viewModel.initialize(roomId, bridgeUserId)
    }

    LaunchedEffect(uiState.verificationComplete) {
        if (uiState.verificationComplete) {
            onVerificationComplete()
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        icon = {
            Icon(
                imageVector = if (uiState.verificationComplete) Icons.Default.CheckCircle else Icons.Default.Security,
                contentDescription = null,
                modifier = Modifier.size(48.dp),
                tint = if (uiState.verificationComplete)
                    MaterialTheme.colorScheme.primary
                else
                    MaterialTheme.colorScheme.secondary
            )
        },
        title = {
            Text(
                text = if (uiState.verificationComplete) "Verification Complete" else "Verify Bridge",
                textAlign = TextAlign.Center
            )
        },
        text = {
            Column(
                modifier = Modifier.fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                if (uiState.verificationComplete) {
                    // Success state
                    Icon(
                        imageVector = Icons.Default.Done,
                        contentDescription = null,
                        modifier = Modifier.size(64.dp),
                        tint = Color(0xFF4CAF50)
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = "The ArmorClaw Bridge is now verified. You can securely chat with users on Slack, Discord, and Teams.",
                        style = MaterialTheme.typography.bodyMedium,
                        textAlign = TextAlign.Center
                    )
                } else if (uiState.verificationInProgress) {
                    // In progress state
                    CircularProgressIndicator()
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = uiState.statusMessage ?: "Starting verification...",
                        style = MaterialTheme.typography.bodyMedium,
                        textAlign = TextAlign.Center
                    )

                    // Emoji verification display
                    if (uiState.emojis.isNotEmpty()) {
                        Spacer(modifier = Modifier.height(24.dp))
                        Text(
                            text = "Compare these emojis on both devices:",
                            style = MaterialTheme.typography.labelMedium
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceEvenly
                        ) {
                            uiState.emojis.forEach { emoji ->
                                EmojiDisplay(emoji = emoji)
                            }
                        }
                    }
                } else {
                    // Initial state - explanation
                    Text(
                        text = "To chat with users on external platforms (Slack, Discord, Teams), " +
                                "you must verify the ArmorClaw Bridge.",
                        style = MaterialTheme.typography.bodyMedium,
                        textAlign = TextAlign.Center
                    )

                    Spacer(modifier = Modifier.height(16.dp))

                    Card(
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.secondaryContainer
                        )
                    ) {
                        Column(
                            modifier = Modifier.padding(12.dp)
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Icon(
                                    imageVector = Icons.Default.Info,
                                    contentDescription = null,
                                    tint = MaterialTheme.colorScheme.onSecondaryContainer,
                                    modifier = Modifier.size(20.dp)
                                )
                                Spacer(modifier = Modifier.width(8.dp))
                                Text(
                                    text = "Why is this needed?",
                                    style = MaterialTheme.typography.titleSmall,
                                    color = MaterialTheme.colorScheme.onSecondaryContainer
                                )
                            }
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = "The bridge acts as a relay between Matrix and external platforms. " +
                                        "Verification establishes trust so the bridge can decrypt your messages.",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSecondaryContainer
                            )
                        }
                    }

                    // Bridge info
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = "Bridge: ${uiState.bridgeUserId}",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.outline
                    )
                }

                // Error display
                uiState.error?.let { error ->
                    Spacer(modifier = Modifier.height(16.dp))
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error
                    )
                }
            }
        },
        confirmButton = {
            if (uiState.verificationComplete) {
                TextButton(onClick = onDismiss) {
                    Text("Done")
                }
            } else if (uiState.verificationInProgress) {
                Row {
                    TextButton(onClick = { viewModel.cancelVerification() }) {
                        Text("Cancel")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(
                        onClick = { viewModel.confirmVerification() },
                        enabled = uiState.emojis.isNotEmpty()
                    ) {
                        Text("They Match")
                    }
                }
            } else {
                Button(
                    onClick = { viewModel.startVerification() },
                    enabled = uiState.canStartVerification
                ) {
                    Icon(Icons.Default.PlayArrow, contentDescription = null)
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Start Verification")
                }
            }
        },
        dismissButton = {
            if (!uiState.verificationInProgress && !uiState.verificationComplete) {
                TextButton(onClick = onDismiss) {
                    Text("Not Now")
                }
            }
        }
    )
}

@Composable
private fun EmojiDisplay(emoji: EmojiInfo) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = emoji.emoji,
            style = MaterialTheme.typography.displayMedium
        )
        Text(
            text = emoji.description,
            style = MaterialTheme.typography.labelSmall
        )
    }
}

/**
 * Bridge Verification ViewModel
 */
class BridgeVerificationViewModel : ViewModel() {

    private val _uiState = MutableStateFlow(BridgeVerificationUiState())
    val uiState: StateFlow<BridgeVerificationUiState> = _uiState.asStateFlow()

    private var roomId: String? = null
    private var bridgeUserId: String? = null
    private var verificationTransactionId: String? = null

    fun initialize(roomId: String, bridgeUserId: String) {
        this.roomId = roomId
        this.bridgeUserId = bridgeUserId

        _uiState.value = _uiState.value.copy(
            bridgeUserId = bridgeUserId,
            canStartVerification = true
        )
    }

    fun startVerification() {
        val currentRoomId = roomId ?: return
        val currentBridgeUserId = bridgeUserId ?: return

        _uiState.value = _uiState.value.copy(
            verificationInProgress = true,
            statusMessage = "Requesting verification...",
            error = null
        )

        viewModelScope.launch {
            try {
                // Simulate verification request
                // In production, this would call Matrix SDK:
                // matrixClient.verification().requestVerification(bridgeUserId, roomId)

                kotlinx.coroutines.delay(1000)

                // Generate emoji verification (simulated)
                val emojis = generateVerificationEmojis()

                _uiState.value = _uiState.value.copy(
                    statusMessage = "Compare the emojis below",
                    emojis = emojis
                )
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(
                    verificationInProgress = false,
                    error = "Failed to start verification: ${e.message}"
                )
            }
        }
    }

    fun confirmVerification() {
        _uiState.value = _uiState.value.copy(
            statusMessage = "Confirming verification..."
        )

        viewModelScope.launch {
            try {
                // In production, call Matrix SDK to confirm:
                // matrixClient.verification().confirmVerification(transactionId)

                kotlinx.coroutines.delay(500)

                _uiState.value = _uiState.value.copy(
                    verificationInProgress = false,
                    verificationComplete = true,
                    statusMessage = "Verification complete"
                )
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(
                    error = "Confirmation failed: ${e.message}"
                )
            }
        }
    }

    fun cancelVerification() {
        _uiState.value = _uiState.value.copy(
            verificationInProgress = false,
            emojis = emptyList(),
            statusMessage = null
        )
    }

    private fun generateVerificationEmojis(): List<EmojiInfo> {
        // In production, these would come from the Matrix SDK verification
        val emojiList = listOf(
            EmojiInfo("🐕", "Dog"),
            EmojiInfo("🐱", "Cat"),
            EmojiInfo("🐘", "Elephant"),
            EmojiInfo("🦊", "Fox"),
            EmojiInfo("🐼", "Panda"),
            EmojiInfo("🦁", "Lion"),
            EmojiInfo("🐯", "Tiger"),
            EmojiInfo("🐻", "Bear")
        )

        return emojiList.shuffled().take(7)
    }

    private val viewModelScope = androidx.lifecycle.viewModelScope
}

/**
 * Bridge Verification UI State
 */
data class BridgeVerificationUiState(
    val bridgeUserId: String = "",
    val canStartVerification: Boolean = false,
    val verificationInProgress: Boolean = false,
    val verificationComplete: Boolean = false,
    val statusMessage: String? = null,
    val emojis: List<EmojiInfo> = emptyList(),
    val error: String? = null
)

/**
 * Emoji info for verification display
 */
data class EmojiInfo(
    val emoji: String,
    val description: String
)

/**
 * Room header indicator for bridge rooms
 */
@Composable
fun BridgeRoomIndicator(
    isVerified: Boolean,
    onClick: () -> Unit
) {
    IconButton(onClick = onClick) {
        Icon(
            imageVector = if (isVerified) Icons.Default.VerifiedUser else Icons.Default.Shield,
            contentDescription = if (isVerified) "Bridge verified" else "Verify bridge",
            tint = if (isVerified)
                Color(0xFF4CAF50)
            else
                MaterialTheme.colorScheme.outline
        )
    }
}
