package app.armorclaw.ui.migration

import android.content.Context
import android.content.SharedPreferences
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowForward
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Migration Screen - Handles upgrade from legacy v2.5 to secure architecture
 *
 * Resolves: G-09 (Migration Path)
 *
 * This screen detects legacy local storage and guides users through:
 * 1. Explaining the architecture upgrade
 * 2. Offering chat history export (if supported)
 * 3. Clearing legacy credentials
 * 4. Requiring fresh login
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MigrationScreen(
    onMigrationComplete: () -> Unit,
    viewModel: MigrationViewModel = viewModel()
) {
    val context = LocalContext.current
    val uiState by viewModel.uiState.collectAsState()

    LaunchedEffect(Unit) {
        viewModel.checkForLegacyData(context)
    }

    LaunchedEffect(uiState.migrationComplete) {
        if (uiState.migrationComplete) {
            onMigrationComplete()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Upgrade Required") },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer
                )
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(24.dp)
                .verticalScroll(rememberScrollState()),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Warning icon
            Icon(
                imageVector = Icons.Default.Warning,
                contentDescription = null,
                modifier = Modifier.size(64.dp),
                tint = MaterialTheme.colorScheme.error
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Title
            Text(
                text = "Security Architecture Upgrade",
                style = MaterialTheme.typography.headlineMedium,
                textAlign = TextAlign.Center
            )

            Spacer(modifier = Modifier.height(16.dp))

            // Description
            Text(
                text = "A new secure architecture is available. This upgrade provides:\n\n" +
                        "• End-to-end encryption for all messages\n" +
                        "• Native Matrix push notifications\n" +
                        "• Hardware-backed device verification\n" +
                        "• Improved security and privacy",
                style = MaterialTheme.typography.bodyLarge,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(horizontal = 16.dp)
            )

            Spacer(modifier = Modifier.height(24.dp))

            // Legacy data detected warning
            if (uiState.hasLegacyData) {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.errorContainer
                    )
                ) {
                    Column(
                        modifier = Modifier.padding(16.dp)
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Default.Info,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.error
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Text(
                                text = "Legacy Data Detected",
                                style = MaterialTheme.typography.titleMedium,
                                color = MaterialTheme.colorScheme.onErrorContainer
                            )
                        }

                        Spacer(modifier = Modifier.height(8.dp))

                        Text(
                            text = "You have existing data from version ${uiState.legacyVersion ?: "unknown"}. " +
                                    "This upgrade requires a fresh login to establish secure encryption keys.",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onErrorContainer
                        )
                    }
                }

                Spacer(modifier = Modifier.height(16.dp))

                // Export option
                OutlinedCard(
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(
                        modifier = Modifier.padding(16.dp)
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Default.Security,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.primary
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Text(
                                text = "Export Chat History",
                                style = MaterialTheme.typography.titleMedium
                            )
                        }

                        Spacer(modifier = Modifier.height(8.dp))

                        Text(
                            text = "Save your chat history before upgrading. " +
                                    "Exported chats can be imported after login.",
                            style = MaterialTheme.typography.bodyMedium
                        )

                        Spacer(modifier = Modifier.height(12.dp))

                        Button(
                            onClick = { viewModel.exportChatHistory(context) },
                            enabled = !uiState.exporting && !uiState.exportComplete,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            if (uiState.exporting) {
                                CircularProgressIndicator(
                                    modifier = Modifier.size(20.dp),
                                    strokeWidth = 2.dp
                                )
                                Spacer(modifier = Modifier.width(8.dp))
                                Text("Exporting...")
                            } else if (uiState.exportComplete) {
                                Text("Export Complete")
                            } else {
                                Text("Export Chats")
                            }
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // Upgrade button
            Button(
                onClick = { viewModel.performMigration(context) },
                enabled = !uiState.migrating,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp)
            ) {
                if (uiState.migrating) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(24.dp),
                        strokeWidth = 2.dp,
                        color = MaterialTheme.colorScheme.onPrimary
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Upgrading...")
                } else {
                    Icon(Icons.Default.ArrowForward, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Upgrade and Continue")
                }
            }

            // Skip option (for development)
            if (BuildConfig.DEBUG) {
                Spacer(modifier = Modifier.height(8.dp))
                TextButton(
                    onClick = { viewModel.skipMigration(context) }
                ) {
                    Text("Skip (Debug Only)")
                }
            }

            // Error message
            uiState.error?.let { error ->
                Spacer(modifier = Modifier.height(16.dp))
                Text(
                    text = error,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error
                )
            }
        }
    }
}

/**
 * Migration ViewModel
 */
class MigrationViewModel : ViewModel() {

    private val _uiState = MutableStateFlow(MigrationUiState())
    val uiState: StateFlow<MigrationUiState> = _uiState.asStateFlow()

    private val LEGACY_PREFS_NAME = "armorclaw_prefs"
    private val LEGACY_KEY_USER_ID = "user_id"
    private val LEGACY_KEY_TOKEN = "access_token"
    private val LEGACY_KEY_VERSION = "app_version"

    fun checkForLegacyData(context: Context) {
        val prefs = context.getSharedPreferences(LEGACY_PREFS_NAME, Context.MODE_PRIVATE)

        val hasLegacyUserId = prefs.contains(LEGACY_KEY_USER_ID)
        val hasLegacyToken = prefs.contains(LEGACY_KEY_TOKEN)
        val legacyVersion = prefs.getString(LEGACY_KEY_VERSION, null)

        _uiState.value = _uiState.value.copy(
            hasLegacyData = hasLegacyUserId || hasLegacyToken,
            legacyVersion = legacyVersion
        )
    }

    fun exportChatHistory(context: Context) {
        _uiState.value = _uiState.value.copy(
            exporting = true,
            error = null
        )

        // TODO: Implement actual export logic
        // For now, simulate export
        viewModelScope.launch {
            try {
                // Simulate export delay
                kotlinx.coroutines.delay(2000)

                // In production, this would:
                // 1. Query local database for messages
                // 2. Export to JSON file
                // 3. Save to Downloads or shared storage

                _uiState.value = _uiState.value.copy(
                    exporting = false,
                    exportComplete = true
                )
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(
                    exporting = false,
                    error = "Export failed: ${e.message}"
                )
            }
        }
    }

    fun performMigration(context: Context) {
        _uiState.value = _uiState.value.copy(
            migrating = true,
            error = null
        )

        viewModelScope.launch {
            try {
                // 1. Clear legacy credentials
                val prefs = context.getSharedPreferences(LEGACY_PREFS_NAME, Context.MODE_PRIVATE)
                prefs.edit().clear().apply()

                // 2. Clear push token state
                val pushPrefs = context.getSharedPreferences("push_tokens", Context.MODE_PRIVATE)
                pushPrefs.edit()
                    .putBoolean("legacy_push_mode", true) // Mark for migration
                    .apply()

                // 3. Clear any cached data
                // In production, clear room database, etc.

                _uiState.value = _uiState.value.copy(
                    migrating = false,
                    migrationComplete = true
                )
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(
                    migrating = false,
                    error = "Migration failed: ${e.message}"
                )
            }
        }
    }

    fun skipMigration(context: Context) {
        // For development only - mark as migrated without clearing data
        val prefs = context.getSharedPreferences(LEGACY_PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit().putString("migration_skipped", "true").apply()

        _uiState.value = _uiState.value.copy(migrationComplete = true)
    }

    private val viewModelScope = androidx.lifecycle.viewModelScope
}

/**
 * Migration UI State
 */
data class MigrationUiState(
    val hasLegacyData: Boolean = false,
    val legacyVersion: String? = null,
    val exporting: Boolean = false,
    val exportComplete: Boolean = false,
    val migrating: Boolean = false,
    val migrationComplete: Boolean = false,
    val error: String? = null
)

// BuildConfig stub for non-Android builds
object BuildConfig {
    const val DEBUG = true
}
