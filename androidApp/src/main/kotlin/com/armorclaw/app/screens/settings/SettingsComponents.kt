package com.armorclaw.app.screens.settings
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.draw.clip

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.armorclaw.app.viewmodels.AgentManagementViewModel
import com.armorclaw.app.viewmodels.AgentManagementUiState
import com.armorclaw.app.viewmodels.HitlViewModel
import com.armorclaw.app.viewmodels.HitlUiState
import com.armorclaw.shared.platform.bridge.AgentInfo
import com.armorclaw.shared.platform.bridge.HitlApproval
import com.armorclaw.shared.ui.theme.BrandPurple
import org.koin.androidx.compose.koinViewModel

// ============================================================================
// Agent Management Screen - NEW
// ============================================================================

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AgentManagementScreen(
    onBack: () -> Unit,
    viewModel: AgentManagementViewModel = koinViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val agents by viewModel.agents.collectAsStateWithLifecycle()
    var showStopDialog by remember { mutableStateOf<String?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Agent Management") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.loadAgents() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                }
            )
        }
    ) { paddingValues ->
        when (val state = uiState) {
            is AgentManagementUiState.Loading -> {
                Box(
                    modifier = Modifier.fillMaxSize().padding(paddingValues),
                    contentAlignment = Alignment.Center
                ) { CircularProgressIndicator() }
            }
            is AgentManagementUiState.Error -> {
                Box(
                    modifier = Modifier.fillMaxSize().padding(paddingValues),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Icon(Icons.Default.Error, contentDescription = null,
                            modifier = Modifier.size(48.dp), tint = MaterialTheme.colorScheme.error)
                        Spacer(Modifier.height(16.dp))
                        Text(state.message, style = MaterialTheme.typography.bodyLarge)
                        Spacer(Modifier.height(16.dp))
                        Button(onClick = { viewModel.loadAgents() }) { Text("Retry") }
                    }
                }
            }
            else -> {
                if (agents.isEmpty()) {
                    Box(
                        modifier = Modifier.fillMaxSize().padding(paddingValues),
                        contentAlignment = Alignment.Center
                    ) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Icon(Icons.Default.SmartToy, contentDescription = null,
                                modifier = Modifier.size(64.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant)
                            Spacer(Modifier.height(16.dp))
                            Text("No agents running", style = MaterialTheme.typography.titleMedium)
                            Text("AI agents will appear here when active",
                                style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                } else {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize().padding(paddingValues)
                    ) {
                        items(agents, key = { it.agentId }) { agent ->
                            AgentCard(
                                agent = agent,
                                isStopping = uiState is AgentManagementUiState.StoppingAgent &&
                                    (uiState as AgentManagementUiState.StoppingAgent).agentId == agent.agentId,
                                onStop = { showStopDialog = agent.agentId },
                                onRefresh = { viewModel.refreshAgentStatus(agent.agentId) }
                            )
                        }
                    }
                }
            }
        }

        showStopDialog?.let { agentId ->
            AlertDialog(
                onDismissRequest = { showStopDialog = null },
                title = { Text("Stop Agent?") },
                text = { Text("Are you sure you want to stop this agent? This action cannot be undone.") },
                confirmButton = {
                    Button(onClick = { viewModel.stopAgent(agentId); showStopDialog = null },
                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error))
                    { Text("Stop") }
                },
                dismissButton = { TextButton(onClick = { showStopDialog = null }) { Text("Cancel") } }
            )
        }
    }
}

@Composable
private fun AgentCard(
    agent: AgentInfo,
    isStopping: Boolean,
    onStop: () -> Unit,
    onRefresh: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(modifier = Modifier.fillMaxWidth().padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Default.SmartToy, contentDescription = null,
                        tint = when (agent.status) {
                            "idle" -> MaterialTheme.colorScheme.primary
                            "busy" -> BrandPurple
                            "error" -> MaterialTheme.colorScheme.error
                            else -> MaterialTheme.colorScheme.onSurfaceVariant
                        })
                    Spacer(Modifier.width(12.dp))
                    Column {
                        Text(agent.name, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                        Text(agent.type, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                AssistChip(
                    onClick = if (isStopping) ({}).let { {} } else onStop,
                    label = { if (isStopping) CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp) else Text("Stop") },
                    leadingIcon = { if (!isStopping) Icon(Icons.Default.Stop, contentDescription = null, modifier = Modifier.size(16.dp)) }
                )
            }
            Spacer(Modifier.height(12.dp))
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                StatusChip("Status", agent.status)
                agent.roomId?.let { roomId -> StatusChip("Room", roomId.take(8) + "...") }
            }
            Spacer(Modifier.height(8.dp))
            Text("ID: ${agent.agentId.take(12)}...", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun StatusChip(label: String, value: String) {
    Surface(shape = MaterialTheme.shapes.small, color = MaterialTheme.colorScheme.surfaceVariant) {
        Row(modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("$label: ", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(value, style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold)
        }
    }
}

// ============================================================================
// HITL Approval Screen - NEW
// ============================================================================

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HitlApprovalScreen(
    onBack: () -> Unit,
    viewModel: HitlViewModel = koinViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val pendingApprovals by viewModel.pendingApprovals.collectAsStateWithLifecycle()
    var showApproveDialog by remember { mutableStateOf<HitlApproval?>(null) }
    var showRejectDialog by remember { mutableStateOf<HitlApproval?>(null) }
    var rejectionReason by remember { mutableStateOf("") }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Pending Approvals") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") }
                },
                actions = {
                    IconButton(onClick = { viewModel.loadPendingApprovals() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                }
            )
        }
    ) { paddingValues ->
        when (val state = uiState) {
            is HitlUiState.Loading -> {
                Box(modifier = Modifier.fillMaxSize().padding(paddingValues), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
            is HitlUiState.Error -> {
                Box(modifier = Modifier.fillMaxSize().padding(paddingValues), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Icon(Icons.Default.Error, contentDescription = null, modifier = Modifier.size(48.dp), tint = MaterialTheme.colorScheme.error)
                        Spacer(Modifier.height(16.dp))
                        Text(state.message, style = MaterialTheme.typography.bodyLarge)
                        Spacer(Modifier.height(16.dp))
                        Button(onClick = { viewModel.loadPendingApprovals() }) { Text("Retry") }
                    }
                }
            }
            else -> {
                if (pendingApprovals.isEmpty()) {
                    Box(modifier = Modifier.fillMaxSize().padding(paddingValues), contentAlignment = Alignment.Center) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Icon(Icons.Default.CheckCircle, contentDescription = null, modifier = Modifier.size(64.dp), tint = MaterialTheme.colorScheme.primary)
                            Spacer(Modifier.height(16.dp))
                            Text("All caught up!", style = MaterialTheme.typography.titleMedium)
                            Text("No pending approvals", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                } else {
                    LazyColumn(modifier = Modifier.fillMaxSize().padding(paddingValues)) {
                        items(pendingApprovals, key = { it.gateId }) { approval ->
                            HitlApprovalCard(
                                approval = approval,
                                isProcessing = uiState is HitlUiState.Processing && (uiState as HitlUiState.Processing).gateId == approval.gateId,
                                onApprove = { showApproveDialog = approval },
                                onReject = { showRejectDialog = approval }
                            )
                        }
                    }
                }
            }
        }

        showApproveDialog?.let { approval ->
            AlertDialog(
                onDismissRequest = { showApproveDialog = null },
                title = { Text("Approve Request") },
                text = { Column { Text("Approve this ${approval.requestType} request?"); Spacer(Modifier.height(8.dp)); Text(approval.description, style = MaterialTheme.typography.bodyMedium) } },
                confirmButton = { Button(onClick = { viewModel.approve(approval.gateId); showApproveDialog = null }) { Text("Approve") } },
                dismissButton = { TextButton(onClick = { showApproveDialog = null }) { Text("Cancel") } }
            )
        }

        showRejectDialog?.let { approval ->
            AlertDialog(
                onDismissRequest = { showRejectDialog = null; rejectionReason = "" },
                title = { Text("Reject Request") },
                text = {
                    Column {
                        Text("Reject this ${approval.requestType} request?")
                        Spacer(Modifier.height(16.dp))
                        OutlinedTextField(value = rejectionReason, onValueChange = { rejectionReason = it }, label = { Text("Reason (optional)") }, modifier = Modifier.fillMaxWidth())
                    }
                },
                confirmButton = {
                    Button(onClick = { viewModel.reject(approval.gateId, rejectionReason.ifBlank { null }); showRejectDialog = null; rejectionReason = "" },
                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error))
                    { Text("Reject") }
                },
                dismissButton = { TextButton(onClick = { showRejectDialog = null; rejectionReason = "" }) { Text("Cancel") } }
            )
        }
    }
}

@Composable
private fun HitlApprovalCard(
    approval: HitlApproval,
    isProcessing: Boolean,
    onApprove: () -> Unit,
    onReject: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(modifier = Modifier.fillMaxWidth().padding(16.dp)) {
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        PriorityIcon(approval.priority)
                        Spacer(Modifier.width(8.dp))
                        Text(approval.title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    }
                    Spacer(Modifier.height(4.dp))
                    Text(approval.requestType, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Surface(
                    shape = MaterialTheme.shapes.small,
                    color = when (approval.priority) {
                        "critical" -> MaterialTheme.colorScheme.errorContainer
                        "high" -> MaterialTheme.colorScheme.tertiaryContainer
                        else -> MaterialTheme.colorScheme.surfaceVariant
                    }
                ) {
                    Text(approval.priority.uppercase(), style = MaterialTheme.typography.labelSmall, modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp))
                }
            }
            Spacer(Modifier.height(12.dp))
            Text(approval.description, style = MaterialTheme.typography.bodyMedium, maxLines = 3, overflow = TextOverflow.Ellipsis)
            Spacer(Modifier.height(12.dp))
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = onApprove, enabled = !isProcessing, modifier = Modifier.weight(1f)) {
                    if (isProcessing) CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = MaterialTheme.colorScheme.onPrimary)
                    else { Icon(Icons.Default.Check, contentDescription = null); Spacer(Modifier.width(4.dp)); Text("Approve") }
                }
                OutlinedButton(onClick = onReject, enabled = !isProcessing, modifier = Modifier.weight(1f), colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error)) {
                    Icon(Icons.Default.Close, contentDescription = null); Spacer(Modifier.width(4.dp)); Text("Reject")
                }
            }
        }
    }
}

@Composable
private fun PriorityIcon(priority: String) {
    Icon(
        when (priority) {
            "critical" -> Icons.Default.Error
            "high" -> Icons.Default.Warning
            "medium" -> Icons.Default.Info
            else -> Icons.Default.Help
        },
        contentDescription = "Priority: $priority",
        tint = when (priority) {
            "critical" -> MaterialTheme.colorScheme.error
            "high" -> MaterialTheme.colorScheme.tertiary
            else -> MaterialTheme.colorScheme.onSurfaceVariant
        }
    )
}

// ============================================================================
// Common Settings Components (EXISTING)
// ============================================================================

/**
 * Common settings card
 */
@Composable
fun SettingsCard(
    title: String,
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )

            Divider()

            content()
        }
    }
}

/**
 * Common setting toggle
 */
@Composable
fun SettingToggle(
    title: String,
    description: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium
            )
            Text(
                text = description,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = if (enabled) 0.6f else 0.3f)
            )
        }
        
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange,
            enabled = enabled
        )
    }
}

/**
 * Common radio group
 */
@Composable
fun <T> RadioGroup(
    title: String,
    options: List<T>,
    selected: T,
    onSelected: (T) -> Unit,
    modifier: Modifier = Modifier,
    label: (T) -> String = { it.toString() }
) {
    Column(
        modifier = modifier,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium
        )
        
        options.forEach { option ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(8.dp))
                    .clickable { onSelected(option) }
                    .padding(8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                RadioButton(
                    selected = selected == option,
                    onClick = { onSelected(option) },
                    colors = RadioButtonDefaults.colors(
                        selectedColor = com.armorclaw.shared.ui.theme.AccentColor
                    )
                )
                
                Text(
                    text = label(option),
                    style = MaterialTheme.typography.bodyMedium
                )
            }
        }
    }
}

/**
 * Common setting slider
 */
@Composable
fun SettingSlider(
    title: String,
    value: Float,
    onValueChange: (Float) -> Unit,
    range: ClosedFloatingPointRange<Float>,
    steps: Int = 0,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    valueText: String = value.toInt().toString()
) {
    Column(
        modifier = modifier,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium
            )
            
            Text(
                text = valueText,
                style = MaterialTheme.typography.bodyMedium,
                color = com.armorclaw.shared.ui.theme.AccentColor,
                fontWeight = FontWeight.SemiBold
            )
        }
        
        Slider(
            value = value,
            onValueChange = onValueChange,
            valueRange = range,
            steps = steps,
            enabled = enabled,
            colors = SliderDefaults.colors(
                thumbColor = com.armorclaw.shared.ui.theme.AccentColor,
                activeTrackColor = com.armorclaw.shared.ui.theme.AccentColor,
                inactiveTrackColor = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.2f)
            )
        )
    }
}

/**
 * Common setting item with icon
 */
@Composable
fun SettingItem(
    icon: ImageVector,
    title: String,
    description: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(enabled = enabled, onClick = onClick)
            .padding(12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = if (enabled)
                com.armorclaw.shared.ui.theme.AccentColor
            else
                MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
            modifier = Modifier.size(24.dp)
        )
        
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = if (enabled)
                    MaterialTheme.colorScheme.onSurface
                else
                    MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
            )
            Text(
                text = description,
                style = MaterialTheme.typography.bodySmall,
                color = if (enabled)
                    MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                else
                    MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f)
            )
        }
        
        Icon(
            imageVector = Icons.Default.ChevronRight,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
            modifier = Modifier.size(20.dp)
        )
    }
}
