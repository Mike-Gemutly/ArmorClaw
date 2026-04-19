package app.armorclaw.ui.workflow

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccountTree
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import app.armorclaw.ui.components.BlockerResponseDialog
import app.armorclaw.ui.components.WorkflowTimeline
import app.armorclaw.ui.components.WorkflowTimelineState
import app.armorclaw.viewmodel.TemplateInfo
import app.armorclaw.viewmodel.WorkflowDetail
import app.armorclaw.viewmodel.WorkflowInfo
import app.armorclaw.viewmodel.WorkflowUiState
import app.armorclaw.viewmodel.WorkflowViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WorkflowScreen(
    viewModel: WorkflowViewModel = viewModel(),
    onBack: () -> Unit = {}
) {
    val uiState by viewModel.uiState.collectAsState()
    val selectedWorkflow by viewModel.selectedWorkflow.collectAsState()
    val timelineState by viewModel.timelineState.collectAsState()
    val activeBlocker by viewModel.activeBlocker.collectAsState()
    val blockerDialogState by viewModel.blockerDialogState.collectAsState()
    val blockerError by viewModel.blockerError.collectAsState()
    val operationLoading by viewModel.operationLoading.collectAsState()

    LaunchedEffect(Unit) { viewModel.loadWorkflows() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        if (selectedWorkflow != null) selectedWorkflow!!.name
                        else "Workflows"
                    )
                },
                navigationIcon = {
                    IconButton(onClick = {
                        if (selectedWorkflow != null) viewModel.clearSelection()
                        else onBack()
                    }) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    if (selectedWorkflow != null && selectedWorkflow!!.status == "running") {
                        IconButton(
                            onClick = { viewModel.cancelWorkflow(selectedWorkflow!!.workflowId) },
                            enabled = !operationLoading
                        ) {
                            Icon(Icons.Default.Cancel, contentDescription = "Cancel workflow")
                        }
                    }
                }
            )
        }
    ) { paddingValues ->
        Box(modifier = Modifier.padding(paddingValues)) {
            if (selectedWorkflow != null) {
                WorkflowDetailContent(
                    detail = selectedWorkflow!!,
                    timelineState = timelineState,
                    operationLoading = operationLoading
                )
            } else {
                WorkflowListContent(
                    viewModel = viewModel,
                    uiState = uiState,
                    operationLoading = operationLoading
                )
            }
        }
    }

    activeBlocker?.let { blocker ->
        BlockerResponseDialog(
            blocker = blocker,
            onDismiss = { viewModel.dismissBlocker() },
            onResolve = { wId, sId, input, note ->
                viewModel.resolveBlocker(wId, sId, input, note)
            },
            dialogState = blockerDialogState,
            errorMessage = blockerError
        )
    }
}

@Composable
private fun WorkflowListContent(
    viewModel: WorkflowViewModel,
    uiState: WorkflowUiState,
    operationLoading: Boolean
) {
    when (uiState) {
        is WorkflowUiState.Loading -> {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        }
        is WorkflowUiState.Error -> {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(
                        text = uiState.message,
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.error
                    )
                    Spacer(Modifier.height(16.dp))
                    Button(onClick = { viewModel.loadWorkflows() }) {
                        Text("Retry")
                    }
                }
            }
        }
        is WorkflowUiState.Loaded -> {
            if (uiState.workflows.isEmpty() && uiState.templates.isEmpty()) {
                EmptyWorkflowState()
            } else {
                WorkflowLazyList(
                    workflows = uiState.workflows,
                    templates = uiState.templates,
                    onSelectWorkflow = { viewModel.selectWorkflow(it) },
                    onStartTemplate = { viewModel.startWorkflow(it) },
                    operationLoading = operationLoading
                )
            }
        }
    }
}

@Composable
private fun EmptyWorkflowState() {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Icon(
                imageVector = Icons.Default.AccountTree,
                contentDescription = null,
                modifier = Modifier.size(64.dp),
                tint = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
            )
            Spacer(Modifier.height(16.dp))
            Text(
                text = "No workflows yet",
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = "Start a workflow from a template below",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
            )
        }
    }
}

@Composable
private fun WorkflowLazyList(
    workflows: List<WorkflowInfo>,
    templates: List<TemplateInfo>,
    onSelectWorkflow: (String) -> Unit,
    onStartTemplate: (String) -> Unit,
    operationLoading: Boolean
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        if (workflows.isNotEmpty()) {
            item {
                SectionHeader("Running & Recent")
            }
            items(workflows, key = { it.workflowId }) { workflow ->
                WorkflowCard(
                    workflow = workflow,
                    onClick = { onSelectWorkflow(workflow.workflowId) }
                )
            }
            item { Spacer(Modifier.height(16.dp)) }
        }

        if (templates.isNotEmpty()) {
            item {
                SectionHeader("Templates")
            }
            items(templates, key = { "tpl_${it.id}" }) { template ->
                TemplateCard(
                    template = template,
                    onStart = { onStartTemplate(template.id) },
                    enabled = !operationLoading
                )
            }
        }
    }
}

@Composable
private fun SectionHeader(title: String) {
    Column {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.primary
        )
        Spacer(Modifier.height(8.dp))
    }
}

@Composable
private fun WorkflowCard(
    workflow: WorkflowInfo,
    onClick: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerLow
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = workflow.name,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                if (workflow.startedAt.isNotBlank()) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = workflow.startedAt,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
                    )
                }
            }
            Spacer(Modifier.width(12.dp))
            StatusBadge(status = workflow.status)
        }
    }
}

@Composable
private fun StatusBadge(status: String) {
    val (color, label) = when (status) {
        "running" -> MaterialTheme.colorScheme.tertiary to "Running"
        "completed" -> MaterialTheme.colorScheme.primary to "Done"
        "failed" -> MaterialTheme.colorScheme.error to "Failed"
        "blocked" -> Color(0xFFFFA000) to "Blocked"
        else -> MaterialTheme.colorScheme.outline to status.replaceFirstChar { it.uppercase() }
    }

    Surface(
        shape = MaterialTheme.shapes.small,
        color = color.copy(alpha = 0.15f)
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
            color = color,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
        )
    }
}

@Composable
private fun TemplateCard(
    template: TemplateInfo,
    onStart: () -> Unit,
    enabled: Boolean
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerLow
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = template.name,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                if (template.description.isNotBlank()) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = template.description,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }
            Spacer(Modifier.width(12.dp))
            FilledTonalButton(
                onClick = onStart,
                enabled = enabled,
                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 4.dp)
            ) {
                Icon(
                    Icons.Default.PlayArrow,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp)
                )
                Spacer(Modifier.width(4.dp))
                Text("Start", style = MaterialTheme.typography.labelMedium)
            }
        }
    }
}

@Composable
private fun WorkflowDetailContent(
    detail: WorkflowDetail,
    timelineState: WorkflowTimelineState,
    operationLoading: Boolean
) {
    if (operationLoading) {
        LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        item {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                StatusBadge(status = detail.status)
                Text(
                    text = "ID: ${detail.workflowId.take(8)}",
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = FontFamily.Monospace,
                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
                )
            }
        }

        item {
            WorkflowTimeline(state = timelineState)
        }
    }
}
