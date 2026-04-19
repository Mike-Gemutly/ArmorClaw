package app.armorclaw.ui.account

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import app.armorclaw.network.BridgeApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

data class AccountDeletionState(
    val password: String = "",
    val confirmText: String = "",
    val isDeleting: Boolean = false,
    val isHardeningComplete: Boolean = false,
    val error: String? = null,
    val success: Boolean = false
)

class AccountDeletionViewModel(
    private val api: BridgeApi = BridgeApi()
) : ViewModel() {

    private val _state = MutableStateFlow(AccountDeletionState())
    val state: StateFlow<AccountDeletionState> = _state.asStateFlow()

    init {
        checkHardening()
    }

    private fun checkHardening() {
        viewModelScope.launch {
            val result = api.getHardeningStatus()
            result.onSuccess { status ->
                _state.value = _state.value.copy(isHardeningComplete = status.delegation_ready)
            }
        }
    }

    fun setPassword(password: String) {
        _state.value = _state.value.copy(password = password)
    }

    fun setConfirmText(text: String) {
        _state.value = _state.value.copy(confirmText = text)
    }

    fun deleteAccount() {
        viewModelScope.launch {
            _state.value = _state.value.copy(isDeleting = true, error = null)
            api.deleteAccount(_state.value.password, erase = true)
                .onSuccess {
                    _state.value = _state.value.copy(isDeleting = false, success = true)
                }
                .onFailure { e ->
                    _state.value = _state.value.copy(
                        isDeleting = false,
                        error = e.message ?: "Deletion failed"
                    )
                }
        }
    }

    fun clearError() {
        _state.value = _state.value.copy(error = null)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AccountDeletionScreen(
    viewModel: AccountDeletionViewModel = viewModel(),
    onBack: () -> Unit = {},
    onDeleted: () -> Unit = {}
) {
    val state by viewModel.state.collectAsState()

    if (state.success) {
        onDeleted()
        return
    }

    val canDelete = state.confirmText == "DELETE" &&
            state.password.isNotBlank() &&
            state.isHardeningComplete &&
            !state.isDeleting

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Delete Account") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Surface(
                color = MaterialTheme.colorScheme.errorContainer,
                shape = MaterialTheme.shapes.medium,
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Icon(
                        Icons.Default.Warning,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.error,
                        modifier = Modifier.size(32.dp)
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = "This action is permanent and cannot be undone.",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onErrorContainer
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        text = "Your account, agents, workflows, and all associated data will be permanently erased from the server.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onErrorContainer
                    )
                }
            }

            if (!state.isHardeningComplete) {
                Surface(
                    color = MaterialTheme.colorScheme.surfaceVariant,
                    shape = MaterialTheme.shapes.medium,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text(
                        text = "Account deletion requires security hardening to be complete. Complete the setup wizard first.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(16.dp)
                    )
                }
            }

            if (state.isHardeningComplete) {
                OutlinedTextField(
                    value = state.password,
                    onValueChange = { viewModel.setPassword(it) },
                    label = { Text("Password") },
                    visualTransformation = PasswordVisualTransformation(),
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )

                OutlinedTextField(
                    value = state.confirmText,
                    onValueChange = { viewModel.setConfirmText(it) },
                    label = { Text("Type DELETE to confirm") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    isError = state.confirmText.isNotBlank() && state.confirmText != "DELETE"
                )

                if (state.confirmText.isNotBlank() && state.confirmText != "DELETE") {
                    Text(
                        text = "You must type exactly DELETE to enable the delete button.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error
                    )
                }

                Button(
                    onClick = { viewModel.deleteAccount() },
                    enabled = canDelete,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.error,
                        contentColor = MaterialTheme.colorScheme.onError,
                        disabledContainerColor = MaterialTheme.colorScheme.error.copy(alpha = 0.3f),
                        disabledContentColor = MaterialTheme.colorScheme.onError.copy(alpha = 0.5f)
                    ),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    if (state.isDeleting) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(18.dp),
                            color = MaterialTheme.colorScheme.onError,
                            strokeWidth = 2.dp
                        )
                    } else {
                        Text("Permanently Delete Account")
                    }
                }
            }

            state.error?.let { error ->
                Surface(
                    color = MaterialTheme.colorScheme.errorContainer,
                    shape = MaterialTheme.shapes.small,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = error,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                            modifier = Modifier.weight(1f)
                        )
                        TextButton(onClick = { viewModel.clearError() }) {
                            Text("Dismiss")
                        }
                    }
                }
            }
        }
    }
}
