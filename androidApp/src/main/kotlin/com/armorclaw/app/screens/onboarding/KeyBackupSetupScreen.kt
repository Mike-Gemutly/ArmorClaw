package com.armorclaw.app.screens.onboarding

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Key
import androidx.compose.material.icons.filled.Shield
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.OnBackground

/**
 * Key Backup Setup screen for creating a recovery passphrase.
 *
 * This screen guides users through creating and verifying a 12-word
 * recovery phrase that protects their encryption keys. Without this,
 * losing the device means losing all message history.
 *
 * ## Flow
 * 1. Explain importance of key backup
 * 2. Generate recovery phrase (via BridgeAdminClient.recoveryGeneratePhrase())
 * 3. Display 12-word phrase for user to write down
 * 4. Verify user wrote it down (ask for 3 random words)
 * 5. Store phrase (via BridgeAdminClient.recoveryStorePhrase())
 * 6. Success confirmation
 *
 * ## Entry Points
 * - Onboarding flow (after CompletionScreen, before Home)
 * - Security Settings (manual setup)
 */

private sealed class BackupStep {
    object Explain : BackupStep()
    object Generating : BackupStep()
    data class DisplayPhrase(val words: List<String>) : BackupStep()
    data class VerifyPhrase(
        val words: List<String>,
        val verifyIndices: List<Int>,
        val answers: Map<Int, String>
    ) : BackupStep()
    object Storing : BackupStep()
    object Success : BackupStep()
    data class Error(
        val message: String,
        val words: List<String>? = null, // Store words for retry
        val verifyIndices: List<Int>? = null // Store indices for retry
    ) : BackupStep()
}

@Composable
fun KeyBackupSetupScreen(
    onComplete: () -> Unit,
    onSkip: () -> Unit,
    modifier: Modifier = Modifier
) {
    var currentStep by remember { mutableStateOf<BackupStep>(BackupStep.Explain) }

    Scaffold(modifier = modifier) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(24.dp)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            when (val step = currentStep) {
                is BackupStep.Explain -> {
                    ExplainStep(
                        onContinue = {
                            currentStep = BackupStep.Generating
                            // TODO: Call bridgeAdminClient.recoveryGeneratePhrase()
                            // For now, simulate with placeholder words
                        },
                        onSkip = onSkip
                    )
                }
                is BackupStep.Generating -> {
                    GeneratingStep()
                    // Simulate phrase generation
                    LaunchedEffect(Unit) {
                        kotlinx.coroutines.delay(1500)
                        val words = listOf(
                            "armor", "shield", "claw", "matrix", "secure",
                            "trust", "verify", "bridge", "cipher", "vault",
                            "token", "guard"
                        )
                        currentStep = BackupStep.DisplayPhrase(words)
                    }
                }
                is BackupStep.DisplayPhrase -> {
                    DisplayPhraseStep(
                        words = step.words,
                        onContinue = {
                            // Pick 3 random indices to verify
                            val indices = step.words.indices.shuffled().take(3).sorted()
                            currentStep = BackupStep.VerifyPhrase(
                                words = step.words,
                                verifyIndices = indices,
                                answers = emptyMap()
                            )
                        }
                    )
                }
                is BackupStep.VerifyPhrase -> {
                    VerifyPhraseStep(
                        words = step.words,
                        verifyIndices = step.verifyIndices,
                        answers = step.answers,
                        onAnswerChanged = { index, answer ->
                            currentStep = step.copy(
                                answers = step.answers + (index to answer)
                            )
                        },
                        onVerify = {
                            // Check answers
                            val allCorrect = step.verifyIndices.all { index ->
                                step.answers[index]?.trim()?.lowercase() ==
                                    step.words[index].lowercase()
                            }
                            if (allCorrect) {
                                currentStep = BackupStep.Storing
                                // TODO: Call bridgeAdminClient.recoveryStorePhrase(words.joinToString(" "))
                            } else {
                                // Store words and indices for retry
                                currentStep = BackupStep.Error(
                                    message = "Some words didn't match. Please try again.",
                                    words = step.words,
                                    verifyIndices = step.verifyIndices
                                )
                            }
                        }
                    )
                }
                is BackupStep.Storing -> {
                    StoringStep()
                    LaunchedEffect(Unit) {
                        kotlinx.coroutines.delay(2000)
                        currentStep = BackupStep.Success
                    }
                }
                is BackupStep.Success -> {
                    SuccessStep(onComplete = onComplete)
                }
                is BackupStep.Error -> {
                    ErrorStep(
                        message = step.message,
                        onRetry = {
                            // If we have stored words, retry verification directly
                            if (step.words != null && step.verifyIndices != null) {
                                currentStep = BackupStep.VerifyPhrase(
                                    words = step.words,
                                    verifyIndices = step.verifyIndices,
                                    answers = emptyMap()
                                )
                            } else {
                                currentStep = BackupStep.Explain
                            }
                        }
                    )
                }
            }
        }
    }
}

@Composable
private fun ExplainStep(
    onContinue: () -> Unit,
    onSkip: () -> Unit
) {
    var showSkipDialog by remember { mutableStateOf(false) }

    // Scary confirmation dialog for skipping backup
    if (showSkipDialog) {
        AlertDialog(
            onDismissRequest = { showSkipDialog = false },
            icon = {
                Icon(
                    imageVector = Icons.Default.Key,
                    contentDescription = null,
                    tint = Color(0xFFC62828),
                    modifier = Modifier.size(32.dp)
                )
            },
            title = {
                Text(
                    text = "Skip Key Backup?",
                    fontWeight = FontWeight.Bold,
                    color = Color(0xFFC62828)
                )
            },
            text = {
                Text(
                    text = "Without a recovery phrase, if you lose this device " +
                           "or uninstall the app, ALL of your encrypted message " +
                           "history will be permanently lost. This cannot be undone.\n\n" +
                           "You will be asked again next time you open the app.",
                    style = MaterialTheme.typography.bodyMedium
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    showSkipDialog = false
                    onSkip()
                }) {
                    Text(
                        text = "I Accept the Risk",
                        color = Color(0xFFC62828),
                        fontWeight = FontWeight.Bold
                    )
                }
            },
            dismissButton = {
                Button(onClick = { showSkipDialog = false }) {
                    Text("Go Back")
                }
            }
        )
    }

    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Icon(
            imageVector = Icons.Default.Shield,
            contentDescription = null,
            modifier = Modifier.size(80.dp),
            tint = BrandPurple
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = "Back Up Your Vault",
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(12.dp))

        Text(
            text = "Create a recovery phrase to back up your vault keys. " +
                   "If you lose your device, this phrase is the only way to " +
                   "restore access to your encrypted data.",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(24.dp))

        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = Color(0xFFFCE4EC)
            )
        ) {
            Row(
                modifier = Modifier.padding(16.dp),
                verticalAlignment = Alignment.Top
            ) {
                Icon(
                    imageVector = Icons.Default.Key,
                    contentDescription = null,
                    tint = Color(0xFFC62828),
                    modifier = Modifier.size(20.dp)
                )
                Spacer(modifier = Modifier.width(12.dp))
                Text(
                    text = "Without a recovery phrase, losing your device means " +
                           "permanently losing access to your vault.",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFFC62828)
                )
            }
        }

        Spacer(modifier = Modifier.height(32.dp))

        Button(
            onClick = onContinue,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Create Vault Backup")
        }

        Spacer(modifier = Modifier.height(8.dp))

        TextButton(onClick = { showSkipDialog = true }) {
            Text(
                text = "Skip (Not Recommended)",
                color = Color(0xFFC62828).copy(alpha = 0.7f)
            )
        }
    }
}

@Composable
private fun GeneratingStep() {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        CircularProgressIndicator(
            modifier = Modifier.size(48.dp),
            color = BrandPurple
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "Generating recovery phrase...",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f)
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun DisplayPhraseStep(
    words: List<String>,
    onContinue: () -> Unit
) {
    var isVisible by remember { mutableStateOf(true) }

    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = "Your Recovery Phrase",
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Write down these 12 words in order. Keep them safe and private.",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(24.dp))

        // Word grid
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceVariant
            )
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Recovery Phrase",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold
                    )
                    IconButton(onClick = { isVisible = !isVisible }) {
                        Icon(
                            imageVector = if (isVisible)
                                Icons.Default.VisibilityOff
                            else
                                Icons.Default.Visibility,
                            contentDescription = if (isVisible) "Hide" else "Show"
                        )
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    words.forEachIndexed { index, word ->
                        WordChip(
                            index = index + 1,
                            word = if (isVisible) word else "••••••",
                            modifier = Modifier
                        )
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(24.dp))

        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = Color(0xFFFFF3E0)
            )
        ) {
            Row(
                modifier = Modifier.padding(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.ContentCopy,
                    contentDescription = null,
                    tint = Color(0xFFE65100),
                    modifier = Modifier.size(18.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = "Write these words on paper. Do NOT screenshot or store digitally.",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFF795548)
                )
            }
        }

        Spacer(modifier = Modifier.height(32.dp))

        Button(
            onClick = onContinue,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("I've Written It Down")
        }
    }
}

@Composable
private fun WordChip(
    index: Int,
    word: String,
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(BrandPurple.copy(alpha = 0.1f))
            .border(1.dp, BrandPurple.copy(alpha = 0.3f), RoundedCornerShape(8.dp))
            .padding(horizontal = 12.dp, vertical = 8.dp)
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "$index.",
                style = MaterialTheme.typography.bodySmall,
                color = BrandPurple.copy(alpha = 0.6f),
                fontWeight = FontWeight.Bold
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = word,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium
            )
        }
    }
}

@Composable
private fun VerifyPhraseStep(
    words: List<String>,
    verifyIndices: List<Int>,
    answers: Map<Int, String>,
    onAnswerChanged: (Int, String) -> Unit,
    onVerify: () -> Unit
) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = "Verify Your Phrase",
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = "Enter the requested words to confirm you've saved your recovery phrase.",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(24.dp))

        verifyIndices.forEach { index ->
            OutlinedTextField(
                value = answers[index] ?: "",
                onValueChange = { onAnswerChanged(index, it) },
                label = { Text("Word #${index + 1}") },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 4.dp),
                singleLine = true
            )
        }

        Spacer(modifier = Modifier.height(24.dp))

        Button(
            onClick = onVerify,
            enabled = verifyIndices.all { answers.containsKey(it) && answers[it]?.isNotBlank() == true },
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Verify")
        }
    }
}

@Composable
private fun StoringStep() {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        CircularProgressIndicator(
            modifier = Modifier.size(48.dp),
            color = BrandPurple
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "Encrypting and storing your recovery phrase...",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f)
        )
    }
}

@Composable
private fun SuccessStep(onComplete: () -> Unit) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Icon(
            imageVector = Icons.Default.CheckCircle,
            contentDescription = null,
            modifier = Modifier.size(80.dp),
            tint = Color(0xFF4CAF50)
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = "Vault Backup Complete!",
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold
        )

        Spacer(modifier = Modifier.height(12.dp))

        Text(
            text = "Your vault keys are now backed up. You can restore your " +
                   "vault on a new device using your recovery phrase.",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(32.dp))

        Button(
            onClick = onComplete,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Continue")
        }
    }
}

@Composable
private fun ErrorStep(
    message: String,
    onRetry: () -> Unit
) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Icon(
            imageVector = Icons.Default.Warning,
            contentDescription = null,
            modifier = Modifier.size(48.dp),
            tint = Color(0xFFF44336)
        )

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = message,
            style = MaterialTheme.typography.bodyMedium,
            color = Color(0xFFF44336),
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(24.dp))

        Button(
            onClick = onRetry,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Try Again")
        }
    }
}
