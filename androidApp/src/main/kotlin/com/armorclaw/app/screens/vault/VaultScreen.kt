package com.armorclaw.app.screens.vault

import android.content.res.Configuration
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.tooling.preview.PreviewParameterProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import com.armorclaw.app.security.VaultRepository
import com.armorclaw.shared.domain.repository.VaultKey
import com.armorclaw.shared.domain.repository.VaultKeyCategory
import com.armorclaw.shared.domain.repository.VaultKeySensitivity
import com.armorclaw.shared.platform.biometric.BiometricAuth
import com.armorclaw.shared.platform.biometric.BiometricResult
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch
import org.koin.compose.koinInject

private val VaultKeyCategory.displayName: String
    get() = when (this) {
        VaultKeyCategory.PERSONAL -> "Personal"
        VaultKeyCategory.FINANCIAL -> "Financial"
        VaultKeyCategory.CONTACT -> "Contact"
        VaultKeyCategory.AUTHENTICATION -> "Authentication"
        VaultKeyCategory.MEDICAL -> "Medical"
        VaultKeyCategory.OTHER -> "Other"
        VaultKeyCategory.OMO_CREDENTIALS -> "OMO Credentials"
        VaultKeyCategory.OMO_IDENTITY -> "OMO Identity"
        VaultKeyCategory.OMO_SETTINGS -> "OMO Settings"
        VaultKeyCategory.OMO_TOKENS -> "OMO Tokens"
        VaultKeyCategory.OMO_WORKSPACE -> "OMO Workspace"
        VaultKeyCategory.OMO_TASKS -> "OMO Tasks"
    }

private val VaultKeySensitivity.displayName: String
    get() = when (this) {
        VaultKeySensitivity.LOW -> "Low"
        VaultKeySensitivity.MEDIUM -> "Medium"
        VaultKeySensitivity.HIGH -> "High"
        VaultKeySensitivity.CRITICAL -> "Critical"
        VaultKeySensitivity.OMO_LOW -> "OMO Low"
        VaultKeySensitivity.OMO_MEDIUM -> "OMO Medium"
        VaultKeySensitivity.OMO_HIGH -> "OMO High"
        VaultKeySensitivity.OMO_CRITICAL -> "OMO Critical"
    }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VaultScreen(
    vaultRepository: VaultRepository = koinInject(),
    biometricAuth: BiometricAuth = koinInject<com.armorclaw.app.platform.BiometricAuthImpl>().delegate
) {
    var vaultKeys by remember { mutableStateOf<List<VaultKey>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showAddDialog by remember { mutableStateOf(false) }
    var showEditDialog by remember { mutableStateOf(false) }
    var selectedKey by remember { mutableStateOf<VaultKey?>(null) }
    var editValue by remember { mutableStateOf("") }

    val scope = rememberCoroutineScope()

    // Load vault data
    LaunchedEffect(Unit) {
        scope.launch {
            vaultRepository.listKeys().onSuccess { keys ->
                vaultKeys = keys
                isLoading = false
            }.onFailure { e ->
                error = "Failed to load vault: ${e.message}"
                isLoading = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Secure Vault") },
                actions = {
                    IconButton(onClick = { showAddDialog = true }) {
                        Icon(Icons.Default.Add, contentDescription = "Add new PII")
                    }
                }
            )
        }
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
        ) {
            when {
                isLoading -> {
                    CircularProgressIndicator(modifier = Modifier.align(Alignment.Center))
                }
                error != null -> {
                    Text(
                        text = error!!,
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier.align(Alignment.Center)
                    )
                }
                vaultKeys.isEmpty() -> {
                    Text(
                        text = "No PII stored yet",
                        modifier = Modifier.align(Alignment.Center)
                    )
                }
                else -> {
                    VaultContent(
                        vaultKeys = vaultKeys,
                        onEditClick = { key ->
                            selectedKey = key
                            editValue = key.fieldName
                            showEditDialog = true
                        }
                    )
                }
            }
        }
    }

    // Add New PII Dialog
    if (showAddDialog) {
        AddPiiDialog(
            onDismiss = { showAddDialog = false },
            onAdd = { fieldName, value, category, sensitivity ->
                scope.launch {
                    vaultRepository.storeValue(fieldName, value, category, sensitivity)
                        .onSuccess {
                            // Reload data
                            vaultRepository.listKeys().onSuccess { keys ->
                                vaultKeys = keys
                            }
                        }
                    showAddDialog = false
                }
            }
        )
    }

    // Edit PII Dialog
    if (showEditDialog && selectedKey != null) {
        EditPiiDialog(
            vaultKey = selectedKey!!,
            currentValue = editValue,
            onDismiss = { showEditDialog = false },
            onEdit = { newValue ->
                scope.launch {
                    vaultRepository.storeValue(
                        fieldName = selectedKey!!.fieldName,
                        value = newValue,
                        category = selectedKey!!.category,
                        sensitivity = selectedKey!!.sensitivity
                    ).onSuccess {
                        // Reload data
                        vaultRepository.listKeys().onSuccess { keys ->
                            vaultKeys = keys
                        }
                    }
                    showEditDialog = false
                }
            },
            biometricAuth = biometricAuth
        )
    }
}

@Composable
private fun VaultContent(
    vaultKeys: List<VaultKey>,
    onEditClick: (VaultKey) -> Unit
) {
    val groupedKeys = vaultKeys.groupBy { it.category }

    LazyColumn(
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        VaultKeyCategory.values().forEach { category ->
            groupedKeys[category]?.let { keys ->
                item {
                    Text(
                        text = category.displayName,
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.primary
                    )
                }
                items(keys) { key ->
                    VaultItem(
                        key = key,
                        onEditClick = onEditClick
                    )
                }
            }
        }
    }
}

@Composable
private fun VaultItem(
    key: VaultKey,
    onEditClick: (VaultKey) -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(
                modifier = Modifier.weight(1f)
            ) {
                Text(
                    text = key.displayName,
                    style = MaterialTheme.typography.bodyLarge
                )
                Text(
                    text = maskValue(key),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            IconButton(
                onClick = { onEditClick(key) },
                enabled = key.sensitivity != VaultKeySensitivity.OMO_CRITICAL
            ) {
                Icon(Icons.Default.Edit, contentDescription = "Edit ${key.displayName}")
            }
        }
    }
}

private fun maskValue(key: VaultKey): String {
    return when (key.sensitivity) {
        VaultKeySensitivity.CRITICAL, VaultKeySensitivity.OMO_CRITICAL -> "****"
        VaultKeySensitivity.HIGH, VaultKeySensitivity.OMO_HIGH -> "****12**"
        VaultKeySensitivity.MEDIUM, VaultKeySensitivity.OMO_MEDIUM -> "****1234"
        VaultKeySensitivity.LOW, VaultKeySensitivity.OMO_LOW -> {
            val name = key.fieldName
            if (name.length > 4) "${name.take(2)}****${name.takeLast(2)}" else "****"
        }
    }
}

@Composable
private fun AddPiiDialog(
    onDismiss: () -> Unit,
    onAdd: (String, String, VaultKeyCategory, VaultKeySensitivity) -> Unit
) {
    var fieldName by remember { mutableStateOf("") }
    var value by remember { mutableStateOf("") }
    var selectedCategory by remember { mutableStateOf(VaultKeyCategory.PERSONAL) }
    var selectedSensitivity by remember { mutableStateOf(VaultKeySensitivity.LOW) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add New PII") },
        text = {
            Column {
                TextField(
                    value = fieldName,
                    onValueChange = { fieldName = it },
                    label = { Text("Field Name") },
                    placeholder = { Text("e.g., api_key, github_token") }
                )
                Spacer(modifier = Modifier.height(8.dp))
                TextField(
                    value = value,
                    onValueChange = { value = it },
                    label = { Text("Value") },
                    placeholder = { Text("Enter the value") }
                )
                Spacer(modifier = Modifier.height(8.dp))
                Text("Category", style = MaterialTheme.typography.bodyMedium)
                VaultKeyCategory.values().forEach { category ->
                    RadioButton(
                        selected = selectedCategory == category,
                        onClick = { selectedCategory = category }
                    )
                    Text(
                        text = category.displayName,
                        modifier = Modifier.padding(start = 8.dp)
                    )
                }
                Spacer(modifier = Modifier.height(8.dp))
                Text("Sensitivity", style = MaterialTheme.typography.bodyMedium)
                VaultKeySensitivity.values().forEach { sensitivity ->
                    RadioButton(
                        selected = selectedSensitivity == sensitivity,
                        onClick = { selectedSensitivity = sensitivity }
                    )
                    Text(
                        text = sensitivity.displayName,
                        modifier = Modifier.padding(start = 8.dp)
                    )
                }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    onAdd(fieldName, value, selectedCategory, selectedSensitivity)
                }
            ) {
                Text("Add")
            }
        },
        dismissButton = {
            Button(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

@Composable
private fun EditPiiDialog(
    vaultKey: VaultKey,
    currentValue: String,
    onDismiss: () -> Unit,
    onEdit: (String) -> Unit,
    biometricAuth: BiometricAuth
) {
    var value by remember { mutableStateOf(currentValue) }
    var showBiometricPrompt by remember { mutableStateOf(false) }
    var biometricResult by remember { mutableStateOf<String?>(null) }

    if (showBiometricPrompt) {
        BiometricPrompt(
            vaultKey = vaultKey,
            onResult = { result ->
                when (result) {
                    is BiometricResult.Success -> {
                        biometricResult = "Authentication successful"
                        onEdit(value)
                    }
                    is BiometricResult.Error -> {
                        biometricResult = "Authentication failed: ${result.message}"
                    }
                    BiometricResult.Cancelled -> {
                        biometricResult = "Authentication cancelled"
                    }
                    else -> {
                        biometricResult = "Authentication unavailable"
                    }
                }
                showBiometricPrompt = false
            },
            biometricAuth = biometricAuth
        )
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Edit ${vaultKey.displayName}") },
        text = {
            Column {
                TextField(
                    value = value,
                    onValueChange = { value = it },
                    label = { Text("Value") },
                    placeholder = { Text("Enter the value") }
                )
                if (biometricResult != null) {
                    Text(
                        text = biometricResult!!,
                        color = when {
                            biometricResult!!.contains("successful") -> MaterialTheme.colorScheme.primary
                            biometricResult!!.contains("failed") -> MaterialTheme.colorScheme.error
                            else -> MaterialTheme.colorScheme.onSurfaceVariant
                        }
                    )
                }
            }
        },
        confirmButton = {
            Button(
                onClick = {
                    if (vaultKey.sensitivity == VaultKeySensitivity.CRITICAL ||
                        vaultKey.sensitivity == VaultKeySensitivity.OMO_CRITICAL ||
                        vaultKey.sensitivity == VaultKeySensitivity.HIGH ||
                        vaultKey.sensitivity == VaultKeySensitivity.OMO_HIGH) {
                        showBiometricPrompt = true
                    } else {
                        onEdit(value)
                    }
                }
            ) {
                Text("Save")
            }
        },
        dismissButton = {
            Button(onClick = onDismiss) {
                Text("Cancel")
            }
        }
    )
}

@Composable
private fun BiometricPrompt(
    vaultKey: VaultKey,
    onResult: (com.armorclaw.shared.platform.biometric.BiometricResult) -> Unit,
    biometricAuth: BiometricAuth
) {
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            biometricAuth.authenticate("Confirm to edit ${vaultKey.displayName}")
                .onSuccess {
                    onResult(com.armorclaw.shared.platform.biometric.BiometricResult.Success)
                }
                .onFailure { e ->
                    onResult(com.armorclaw.shared.platform.biometric.BiometricResult.Error(e.message ?: "Authentication failed"))
                }
        }
    }
}

// Preview Composables
@Preview(showBackground = true)
@Preview(uiMode = Configuration.UI_MODE_NIGHT_YES)
@Composable
private fun VaultScreenPreview() {
    com.armorclaw.shared.ui.theme.ArmorClawTheme {
        VaultScreen()
    }
}

@Preview(showBackground = true)
@Preview(uiMode = Configuration.UI_MODE_NIGHT_YES)
@Composable
private fun VaultItemPreview() {
    com.armorclaw.shared.ui.theme.ArmorClawTheme {
        VaultItem(
            key = VaultKey(
                id = "test",
                fieldName = "test_field",
                displayName = "Test Field",
                category = VaultKeyCategory.PERSONAL,
                sensitivity = VaultKeySensitivity.MEDIUM,
                lastAccessed = 0,
                accessCount = 0
            ),
            onEditClick = {}
        )
    }
}