package com.armorclaw.app.screens.home

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.app.components.sync.SyncStatusWrapper
import com.armorclaw.app.viewmodels.SyncStatusViewModel
import com.armorclaw.app.viewmodels.HomeViewModel
import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.domain.model.KeystoreStatus
import com.armorclaw.shared.domain.model.Room
import com.armorclaw.shared.domain.model.SyncState
import com.armorclaw.shared.ui.components.*
import com.armorclaw.shared.ui.theme.DesignTokens
import com.armorclaw.shared.ui.theme.OnBackground
import org.koin.androidx.compose.koinViewModel
import org.koin.core.parameter.parametersOf

/**
 * Home Screen - Mission Control Dashboard
 *
 * Displays the VPS Secretary Mode dashboard with agent supervision capabilities.
 *
 * ## Architecture (VPS Secretary Mode)
 * ```
 * HomeScreen
 *      ├── MissionControlHeader (status summary + greeting)
 *      ├── QuickActionsBar (emergency stop, pause, lock vault)
 *      ├── NeedsAttentionQueue (items requiring user intervention)
 *      ├── ActiveTasksSection (running agent tasks)
 *      ├── WorkflowSection (if active workflows)
 *      │       └── WorkflowCard[]
 *      └── RoomList
 *              └── RoomCard[]
 * ```
 *
 * ## State Management
 * - rooms: List of Matrix rooms from RoomRepository
 * - activeWorkflows: Active workflows from ControlPlaneStore
 * - needsAttentionItems: Items requiring user intervention
 * - activeAgentSummaries: Active agent tasks for dashboard
 * - vaultStatus: Current keystore/vault status
 * - syncState: Connection status from MatrixClient
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onRoomClick: (String) -> Unit,
    viewModel: HomeViewModel = koinViewModel()
) {
    val rooms by viewModel.rooms.collectAsState()
    val activeWorkflows by viewModel.activeWorkflows.collectAsState()
    val showWorkflowsSection by viewModel.showWorkflowsSection.collectAsState()
    val isRefreshing by viewModel.isRefreshing.collectAsState()

    // Mission Control State
    val needsAttentionItems by viewModel.needsAttentionItems.collectAsState()
    val activeAgentSummaries by viewModel.activeAgentSummaries.collectAsState()
    val vaultStatus by viewModel.vaultStatus.collectAsState()
    val isPaused by viewModel.isPaused.collectAsState()
    val highestPriority by viewModel.highestPriority.collectAsState()

    Scaffold(
        topBar = {
            Column {
                TopAppBar(
                    title = {
                        Text(
                            "Mission Control",
                            style = MaterialTheme.typography.titleLarge,
                            fontWeight = FontWeight.SemiBold
                        )
                    },
                    actions = {
                        IconButton(onClick = { /* Settings */ }) {
                            Icon(Icons.Default.Settings, contentDescription = "Settings")
                        }
                    }
                )
            }
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = { viewModel.onCreateRoom() }
            ) {
                Icon(Icons.Default.Add, contentDescription = "Create Room")
            }
        }
    ) { paddingValues ->
        if (rooms.isEmpty() && activeWorkflows.isEmpty() && needsAttentionItems.isEmpty()) {
            // Empty state
            EmptyConversationsContent(
                onCreateRoom = { viewModel.onCreateRoom() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues)
            )
        } else {
            // Content
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues),
                contentPadding = PaddingValues(vertical = 8.dp)
            ) {
                // ================================================================
                // Mission Control Dashboard (VPS Secretary Mode)
                // ================================================================

                // Mission Control Header
                item {
                    MissionControlHeader(
                        vaultStatus = vaultStatus,
                        activeAgentCount = activeAgentSummaries.size,
                        attentionCount = needsAttentionItems.size,
                        highestPriority = highestPriority,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                    )
                }

                // Quick Actions Bar
                item {
                    QuickActionsBar(
                        isPaused = isPaused,
                        isVaultLocked = vaultStatus.isSealed(),
                        hasActiveAgents = activeAgentSummaries.isNotEmpty(),
                        onEmergencyStop = { viewModel.emergencyStop() },
                        onPauseAll = { viewModel.pauseAllAgents() },
                        onResumeAll = { viewModel.resumeAllAgents() },
                        onLockVault = { viewModel.lockVault() },
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                    )
                }

                // Needs Attention Queue
                item {
                    NeedsAttentionQueue(
                        items = needsAttentionItems,
                        onItemClick = { viewModel.onAttentionItemClick(it) },
                        onApprove = { viewModel.onApproveAttentionItem(it) },
                        onDeny = { viewModel.onDenyAttentionItem(it) },
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                    )
                }

                // Active Tasks Section
                item {
                    ActiveTasksSection(
                        activeTasks = activeAgentSummaries,
                        onTaskClick = { /* Navigate to task details */ },
                        modifier = Modifier.padding(vertical = 8.dp)
                    )
                }

                // ================================================================
                // Existing Workflow Section
                // ================================================================

                // Active Workflows Section
                if (showWorkflowsSection && activeWorkflows.isNotEmpty()) {
                    item {
                        WorkflowSectionHeader(
                            count = activeWorkflows.size,
                            modifier = Modifier.padding(bottom = 8.dp),
                            onSeeAll = { /* Navigate to workflows list */ }
                        )
                    }

                    items(
                        count = activeWorkflows.take(3).size,
                        key = { index -> activeWorkflows[index].workflowId }
                    ) { index ->
                        val workflow = activeWorkflows[index]
                        WorkflowCard(
                            workflow = workflow,
                            onClick = { viewModel.onWorkflowClick(workflow.workflowId) },
                            onCancel = { viewModel.onCancelWorkflow(workflow.workflowId) },
                            showRoomName = true,
                            roomName = viewModel.getRoomNameForWorkflow(workflow.roomId),
                            modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)
                        )
                    }

                    item {
                        Spacer(modifier = Modifier.height(8.dp))
                        Divider(
                            modifier = Modifier.padding(horizontal = 16.dp),
                            color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f)
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                    }
                }

                // ================================================================
                // Conversations Section
                // ================================================================

                // Conversations Section
                item {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp, vertical = 8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = Icons.Default.ChatBubbleOutline,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(20.dp)
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "Conversations",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.SemiBold
                        )
                        if (rooms.isNotEmpty()) {
                            Spacer(modifier = Modifier.width(8.dp))
                            Surface(
                                shape = MaterialTheme.shapes.small,
                                color = MaterialTheme.colorScheme.surfaceVariant
                            ) {
                                Text(
                                    text = rooms.size.toString(),
                                    style = MaterialTheme.typography.labelSmall,
                                    modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                                )
                            }
                        }
                    }
                }

                items(
                    count = rooms.size,
                    key = { index -> rooms[index].id }
                ) { index ->
                    val room = rooms[index]
                    RoomListItem(
                        room = room,
                        onClick = { onRoomClick(room.id) },
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 2.dp)
                    )
                }
            }
        }
    }
}

/**
 * Check if keystore status is sealed
 */
private fun KeystoreStatus.isSealed(): Boolean = this is KeystoreStatus.Sealed

/**
 * Empty state content
 */
@Composable
private fun EmptyConversationsContent(
    onCreateRoom: () -> Unit,
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier,
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Icon(
                imageVector = Icons.Default.ChatBubbleOutline,
                contentDescription = "No conversations",
                modifier = Modifier.size(80.dp),
                tint = OnBackground.copy(alpha = 0.3f)
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

            Text(
                text = "No conversations yet",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Medium
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))

            Text(
                text = "Start chatting with your AI agent",
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.7f)
            )

            Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))

            FilledTonalButton(
                onClick = onCreateRoom
            ) {
                Icon(Icons.Default.Add, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("New Conversation")
            }
        }
    }
}

/**
 * Room list item
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RoomListItem(
    room: Room,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    ListItem(
        modifier = modifier.clickable(onClick = onClick),
        leadingContent = {
            AvatarBadge(
                name = room.name ?: room.id,
                size = 48.dp
            )
        },
        headlineContent = {
            Text(
                text = room.name ?: "Unnamed Room",
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium
            )
        },
        supportingContent = {
            room.lastMessage?.let { message ->
                Text(
                    text = message.content,
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 1
                )
            }
        },
        trailingContent = {
            Column(
                horizontalAlignment = Alignment.End
            ) {
                room.lastMessage?.let { message ->
                    Text(
                        text = formatTimestamp(message.timestamp.toEpochMilliseconds()),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                if (room.unreadCount > 0) {
                    Spacer(modifier = Modifier.height(4.dp))
                    Badge {
                        Text(
                            text = if (room.unreadCount > 99) "99+" else room.unreadCount.toString()
                        )
                    }
                }
            }
        }
    )
}

/**
 * Simple avatar badge with initials
 */
@Composable
private fun AvatarBadge(
    name: String,
    size: androidx.compose.ui.unit.Dp
) {
    val initials = name
        .split(" ")
        .take(2)
        .mapNotNull { it.firstOrNull()?.uppercase() }
        .joinToString("")

    Surface(
        modifier = Modifier.size(size),
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.primaryContainer
    ) {
        Box(
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = initials.ifEmpty { "?" },
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                fontWeight = FontWeight.Medium
            )
        }
    }
}

/**
 * Format timestamp to readable string
 */
private fun formatTimestamp(timestamp: Long): String {
    val now = System.currentTimeMillis()
    val diff = now - timestamp

    return when {
        diff < 60_000 -> "Now"
        diff < 3600_000 -> "${diff / 60_000}m"
        diff < 86400_000 -> "${diff / 3600_000}h"
        diff < 604800_000 -> "${diff / 86400_000}d"
        else -> {
            val date = java.util.Date(timestamp)
            java.text.SimpleDateFormat("MMM d").format(date)
        }
    }
}
