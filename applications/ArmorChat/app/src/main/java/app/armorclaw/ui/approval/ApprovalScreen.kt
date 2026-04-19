package app.armorclaw.ui.approval

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import app.armorclaw.data.model.EmailApprovalEvent
import app.armorclaw.data.model.SystemAlertContent
import app.armorclaw.network.BridgeApi
import androidx.compose.ui.graphics.vector.ImageVector
import app.armorclaw.ui.components.EmailApprovalCard
import app.armorclaw.ui.components.PiiApprovalCard
import app.armorclaw.viewmodel.HitlViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ApprovalScreen(
    viewModel: HitlViewModel = viewModel(factory = HitlViewModel.Factory()),
    modifier: Modifier = Modifier
) {
    val selectedTab by viewModel.selectedTab.collectAsState()
    val pendingPii by viewModel.pendingPii.collectAsState()
    val pendingMcp by viewModel.pendingMcp.collectAsState()
    val pendingEmails by viewModel.pendingEmails.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()
    val error by viewModel.error.collectAsState()

    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(error) {
        error?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearError()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            TopAppBar(
                title = { Text("Approvals") },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface
                )
            )
        },
        bottomBar = {
            NavigationBar {
                HitlViewModel.ApprovalTab.entries.forEach { tab ->
                    val badgeCount = when (tab) {
                        HitlViewModel.ApprovalTab.PII -> pendingPii.size
                        HitlViewModel.ApprovalTab.MCP -> pendingMcp.size
                        HitlViewModel.ApprovalTab.EMAIL -> pendingEmails.size
                    }
                    NavigationBarItem(
                        icon = {
                            BadgedBox(
                                badge = {
                                    if (badgeCount > 0) {
                                        Badge { Text("$badgeCount") }
                                    }
                                }
                            ) {
                                Icon(
                                    imageVector = when (tab) {
                                        HitlViewModel.ApprovalTab.PII -> Icons.Default.PrivacyTip
                                        HitlViewModel.ApprovalTab.MCP -> Icons.Default.SmartToy
                                        HitlViewModel.ApprovalTab.EMAIL -> Icons.Default.Email
                                    },
                                    contentDescription = tab.label
                                )
                            }
                        },
                        label = { Text(tab.label) },
                        selected = selectedTab == tab,
                        onClick = { viewModel.selectTab(tab) }
                    )
                }
            }
        },
        modifier = modifier
    ) { innerPadding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            when (selectedTab) {
                HitlViewModel.ApprovalTab.PII -> PiiTabContent(
                    pendingPii = pendingPii,
                    onApprove = { requestId, fields -> viewModel.approvePii(requestId, fields) },
                    onDeny = { requestId, reason -> viewModel.denyPii(requestId, reason) }
                )

                HitlViewModel.ApprovalTab.MCP -> McpTabContent(
                    pendingMcp = pendingMcp,
                    onApprove = { agentId -> viewModel.approveMcp(agentId) },
                    onReject = { agentId -> viewModel.rejectMcp(agentId) }
                )

                HitlViewModel.ApprovalTab.EMAIL -> EmailTabContent(
                    pendingEmails = pendingEmails,
                    onApprove = { approvalId -> viewModel.approveEmail(approvalId) },
                    onDeny = { approvalId, reason -> viewModel.denyEmail(approvalId, reason = reason) }
                )
            }

            if (isLoading) {
                CircularProgressIndicator(
                    modifier = Modifier.align(Alignment.Center)
                )
            }
        }
    }
}

@Composable
private fun PiiTabContent(
    pendingPii: List<SystemAlertContent>,
    onApprove: (String, List<String>) -> Unit,
    onDeny: (String, String) -> Unit
) {
    if (pendingPii.isEmpty()) {
        EmptyState(
            icon = Icons.Default.CheckCircle,
            message = "No pending PII approvals"
        )
        return
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(pendingPii, key = { it.metadata?.get("request_id") as? String ?: it.timestamp }) { alert ->
            PiiApprovalCard(
                alert = alert,
                onApprove = onApprove,
                onDeny = onDeny
            )
        }
    }
}

@Composable
private fun McpTabContent(
    pendingMcp: List<BridgeApi.AgentDefinition>,
    onApprove: (String) -> Unit,
    onReject: (String) -> Unit
) {
    if (pendingMcp.isEmpty()) {
        EmptyState(
            icon = Icons.Default.CheckCircle,
            message = "No pending MCP approvals"
        )
        return
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(pendingMcp, key = { it.id }) { agent ->
            McpAgentCard(
                agent = agent,
                onApprove = { onApprove(agent.id) },
                onReject = { onReject(agent.id) }
            )
        }
    }
}

@Composable
private fun McpAgentCard(
    agent: BridgeApi.AgentDefinition,
    onApprove: () -> Unit,
    onReject: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = Color(0xFFF3F8FF))
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.SmartToy,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(28.dp)
                )
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = agent.name,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    if (agent.description.isNotEmpty()) {
                        Text(
                            text = agent.description,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
            }

            if (agent.skills.isNotEmpty()) {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = "Skills: ${agent.skills.joinToString()}",
                    style = MaterialTheme.typography.bodySmall
                )
            }

            if (agent.pii_access.isNotEmpty()) {
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = "PII Access: ${agent.pii_access.joinToString()}",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFFE65100)
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                OutlinedButton(
                    onClick = onReject,
                    modifier = Modifier.weight(1f),
                    colors = ButtonDefaults.outlinedButtonColors(contentColor = Color(0xFFD32F2F))
                ) {
                    Icon(Icons.Default.Close, contentDescription = null, modifier = Modifier.size(18.dp))
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Reject")
                }
                Button(
                    onClick = onApprove,
                    modifier = Modifier.weight(1f),
                    colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF388E3C))
                ) {
                    Icon(Icons.Default.Check, contentDescription = null, modifier = Modifier.size(18.dp))
                    Spacer(modifier = Modifier.width(4.dp))
                    Text("Approve")
                }
            }
        }
    }
}

@Composable
private fun EmailTabContent(
    pendingEmails: List<EmailApprovalEvent>,
    onApprove: (String) -> Unit,
    onDeny: (String, String) -> Unit
) {
    if (pendingEmails.isEmpty()) {
        EmptyState(
            icon = Icons.Default.CheckCircle,
            message = "No pending email approvals"
        )
        return
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(pendingEmails, key = { it.approvalId }) { event ->
            EmailApprovalCard(
                approvalId = event.approvalId,
                emailId = event.emailId,
                to = event.to,
                piiFieldCount = event.piiFields,
                timeoutSeconds = event.timeoutS,
                onApprove = onApprove,
                onDeny = onDeny
            )
        }
    }
}

@Composable
private fun EmptyState(
    icon: ImageVector,
    message: String
) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                modifier = Modifier.size(48.dp),
                tint = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.height(12.dp))
            Text(
                text = message,
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}
