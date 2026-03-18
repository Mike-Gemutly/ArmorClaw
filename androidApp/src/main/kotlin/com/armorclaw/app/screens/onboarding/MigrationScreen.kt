package com.armorclaw.app.screens.onboarding

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Error
import androidx.compose.material.icons.filled.Key
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.armorclaw.app.viewmodels.AppPreferences
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.OnBackground
import org.koin.compose.koinInject

/**
 * Migration screen for users upgrading from v2.5 (Thin Client / Bridge Proxy)
 * to v3.0 (Thick Client / Matrix SDK).
 *
 * This screen handles the transition by:
 * 1. Detecting legacy Bridge session keys
 * 2. Attempting automatic key migration via Bridge recovery API
 * 3. Falling back to manual recovery phrase entry
 * 4. Creating a new Matrix SDK session on success
 *
 * ## When This Screen Appears
 * The SplashViewModel detects a legacy Bridge session (SharedPreferences `bridge_session`)
 * without a corresponding Matrix SDK session (MatrixSessionStorage). This indicates
 * the user upgraded the app from v2.5 to v3.0 without re-authenticating.
 */

sealed class MigrationState {
    object Detecting : MigrationState()
    object AutoMigrating : MigrationState()
    data class AutoMigrationProgress(val progress: Float, val step: String) : MigrationState()
    object NeedsManualRecovery : MigrationState()
    object ManualRecoveryInProgress : MigrationState()
    data class Success(val migratedKeys: Int) : MigrationState()
    data class Failed(val reason: String) : MigrationState()
}

@Composable
fun MigrationScreen(
    onMigrationComplete: () -> Unit,
    onSkipMigration: () -> Unit,
    onLogout: () -> Unit,
    modifier: Modifier = Modifier,
    appPreferences: AppPreferences = koinInject()
) {
    var migrationState by remember { mutableStateOf<MigrationState>(MigrationState.Detecting) }
    var recoveryPhrase by remember { mutableStateOf("") }

    // Wipe legacy data when migration completes (success or skip)
    val onMigrationCompleteWithWipe: () -> Unit = {
        appPreferences.wipeLegacyData()
        onMigrationComplete()
    }
    val onSkipMigrationWithWipe: () -> Unit = {
        appPreferences.wipeLegacyData()
        onSkipMigration()
    }
    val onLogoutWithWipe: () -> Unit = {
        appPreferences.wipeLegacyData()
        appPreferences.clearAll()
        onLogout()
    }

    // Simulate automatic migration attempt
    LaunchedEffect(Unit) {
        migrationState = MigrationState.AutoMigrating
        kotlinx.coroutines.delay(1000)
        migrationState = MigrationState.AutoMigrationProgress(0.3f, "Checking legacy session...")
        kotlinx.coroutines.delay(800)
        migrationState = MigrationState.AutoMigrationProgress(0.6f, "Retrieving encryption keys...")
        kotlinx.coroutines.delay(800)
        // In a real implementation, this would call:
        // bridgeAdminClient.recoveryIsDeviceValid(legacyDeviceId)
        // If valid, attempt automatic key migration
        // If not, fall back to manual recovery
        migrationState = MigrationState.NeedsManualRecovery
    }

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
            // Header
            Icon(
                imageVector = Icons.Default.Sync,
                contentDescription = null,
                modifier = Modifier.size(80.dp),
                tint = BrandPurple
            )

            Spacer(modifier = Modifier.height(24.dp))

            Text(
                text = "Upgrade to ArmorClaw 3.0",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                textAlign = TextAlign.Center
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "We're migrating your account to the new Thick Client architecture " +
                       "with true end-to-end encryption.",
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.7f),
                textAlign = TextAlign.Center
            )

            Spacer(modifier = Modifier.height(32.dp))

            // State-dependent content
            when (val state = migrationState) {
                is MigrationState.Detecting -> {
                    DetectingContent()
                }
                is MigrationState.AutoMigrating -> {
                    AutoMigratingContent()
                }
                is MigrationState.AutoMigrationProgress -> {
                    AutoMigrationProgressContent(
                        progress = state.progress,
                        step = state.step
                    )
                }
                is MigrationState.NeedsManualRecovery -> {
                    ManualRecoveryContent(
                        recoveryPhrase = recoveryPhrase,
                        onRecoveryPhraseChange = { recoveryPhrase = it },
                        onSubmit = {
                            migrationState = MigrationState.ManualRecoveryInProgress
                            // TODO: Call bridgeAdminClient.recoveryVerify(recoveryPhrase)
                            // On success: migrationState = MigrationState.Success(keyCount)
                            // On failure: migrationState = MigrationState.Failed(reason)
                        },
                        onSkip = onSkipMigrationWithWipe
                    )
                }
                is MigrationState.ManualRecoveryInProgress -> {
                    RecoveryInProgressContent()
                }
                is MigrationState.Success -> {
                    SuccessContent(
                        migratedKeys = state.migratedKeys,
                        onContinue = onMigrationCompleteWithWipe
                    )
                }
                is MigrationState.Failed -> {
                    FailedContent(
                        reason = state.reason,
                        onRetry = {
                            migrationState = MigrationState.NeedsManualRecovery
                            recoveryPhrase = ""
                        },
                        onSkip = onSkipMigrationWithWipe,
                        onLogout = onLogoutWithWipe
                    )
                }
            }
        }
    }
}

@Composable
private fun DetectingContent() {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        CircularProgressIndicator(
            modifier = Modifier.size(48.dp),
            color = BrandPurple
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "Detecting legacy session...",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f)
        )
    }
}

@Composable
private fun AutoMigratingContent() {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        CircularProgressIndicator(
            modifier = Modifier.size(48.dp),
            color = BrandPurple
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "Attempting automatic migration...",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f)
        )
    }
}

@Composable
private fun AutoMigrationProgressContent(progress: Float, step: String) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = Modifier.fillMaxWidth()
    ) {
        LinearProgressIndicator(
            progress = progress,
            modifier = Modifier.fillMaxWidth(),
            color = BrandPurple
        )
        Spacer(modifier = Modifier.height(12.dp))
        Text(
            text = step,
            style = MaterialTheme.typography.bodySmall,
            color = OnBackground.copy(alpha = 0.6f)
        )
    }
}

@Composable
private fun ManualRecoveryContent(
    recoveryPhrase: String,
    onRecoveryPhraseChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onSkip: () -> Unit
) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = Modifier.fillMaxWidth()
    ) {
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = Color(0xFFFFF3E0)
            )
        ) {
            Row(
                modifier = Modifier.padding(16.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Key,
                    contentDescription = null,
                    tint = Color(0xFFE65100),
                    modifier = Modifier.size(24.dp)
                )
                Spacer(modifier = Modifier.width(12.dp))
                Text(
                    text = "Automatic migration was not possible. Enter your recovery " +
                           "phrase to migrate your encryption keys.",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFF795548)
                )
            }
        }

        Spacer(modifier = Modifier.height(24.dp))

        OutlinedTextField(
            value = recoveryPhrase,
            onValueChange = onRecoveryPhraseChange,
            label = { Text("Recovery Phrase (12 words)") },
            placeholder = { Text("Enter your 12-word recovery phrase...") },
            modifier = Modifier.fillMaxWidth(),
            minLines = 3,
            maxLines = 5
        )

        Spacer(modifier = Modifier.height(24.dp))

        Button(
            onClick = onSubmit,
            enabled = recoveryPhrase.trim().split("\\s+".toRegex()).size >= 12,
            modifier = Modifier.fillMaxWidth()
        ) {
            Icon(
                imageVector = Icons.Default.Security,
                contentDescription = null,
                modifier = Modifier.size(18.dp)
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text("Migrate Keys")
        }

        Spacer(modifier = Modifier.height(12.dp))

        TextButton(onClick = onSkip) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "Start Fresh (Delete Old Data)",
                    color = Color(0xFFC62828).copy(alpha = 0.8f),
                    fontWeight = FontWeight.SemiBold
                )
                Text(
                    text = "Skips migration and permanently erases v2.5 message history",
                    style = MaterialTheme.typography.labelSmall,
                    color = OnBackground.copy(alpha = 0.4f),
                    textAlign = TextAlign.Center
                )
            }
        }
    }
}

@Composable
private fun RecoveryInProgressContent() {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        CircularProgressIndicator(
            modifier = Modifier.size(48.dp),
            color = BrandPurple
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "Verifying recovery phrase and migrating keys...",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.6f)
        )
    }
}

@Composable
private fun SuccessContent(migratedKeys: Int, onContinue: () -> Unit) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Icon(
            imageVector = Icons.Default.CheckCircle,
            contentDescription = null,
            modifier = Modifier.size(64.dp),
            tint = Color(0xFF4CAF50)
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "Migration Complete!",
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "$migratedKeys encryption keys migrated successfully. " +
                   "Your message history is preserved.",
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )
        Spacer(modifier = Modifier.height(32.dp))
        Button(
            onClick = onContinue,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Continue to ArmorClaw")
        }
    }
}

@Composable
private fun FailedContent(
    reason: String,
    onRetry: () -> Unit,
    onSkip: () -> Unit,
    onLogout: () -> Unit
) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Icon(
            imageVector = Icons.Default.Error,
            contentDescription = null,
            modifier = Modifier.size(64.dp),
            tint = Color(0xFFF44336)
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "Migration Failed",
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = reason,
            style = MaterialTheme.typography.bodyMedium,
            color = OnBackground.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )
        Spacer(modifier = Modifier.height(32.dp))

        Button(
            onClick = onRetry,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Try Again")
        }

        Spacer(modifier = Modifier.height(8.dp))

        OutlinedButton(
            onClick = onSkip,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text("Skip Migration")
        }

        Spacer(modifier = Modifier.height(8.dp))

        TextButton(onClick = onLogout) {
            Text(
                text = "Log Out & Start Fresh",
                color = Color(0xFFF44336)
            )
        }
    }
}
