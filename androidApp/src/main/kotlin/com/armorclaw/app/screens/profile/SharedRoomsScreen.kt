package com.armorclaw.app.screens.profile

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
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.DesignTokens

/**
 * Screen displaying shared rooms with a specific user
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SharedRoomsScreen(
    userId: String,
    onNavigateBack: () -> Unit,
    onRoomClick: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    // Placeholder data - would normally come from repository
    val sharedRooms = remember {
        listOf(
            SharedRoom(
                id = "!room1:example.com",
                name = "Project Alpha",
                memberCount = 15,
                isEncrypted = true
            ),
            SharedRoom(
                id = "!room2:example.com",
                name = "Team General",
                memberCount = 8,
                isEncrypted = true
            ),
            SharedRoom(
                id = "!room3:example.com",
                name = "Announcements",
                memberCount = 50,
                isEncrypted = false
            )
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Shared Rooms") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        }
    ) { paddingValues ->
        Column(
            modifier = modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            // Info card
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(DesignTokens.Spacing.md),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Row(
                    modifier = Modifier.padding(DesignTokens.Spacing.md),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        Icons.Default.Group,
                        contentDescription = null,
                        tint = BrandPurple
                    )
                    Spacer(modifier = Modifier.width(DesignTokens.Spacing.md))
                    Text(
                        text = "Rooms you share with $userId",
                        style = MaterialTheme.typography.bodyMedium
                    )
                }
            }

            // Room list
            if (sharedRooms.isEmpty()) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Icon(
                            Icons.Default.MeetingRoom,
                            contentDescription = null,
                            modifier = Modifier.size(64.dp),
                            tint = MaterialTheme.colorScheme.outline
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        Text(
                            text = "No shared rooms",
                            style = MaterialTheme.typography.titleMedium
                        )
                        Text(
                            text = "You don't have any rooms in common with this user",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                        )
                    }
                }
            } else {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(DesignTokens.Spacing.md),
                    verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.sm)
                ) {
                    items(sharedRooms) { room ->
                        SharedRoomItem(
                            room = room,
                            onClick = { onRoomClick(room.id) }
                        )
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
private fun SharedRoomItem(
    room: SharedRoom,
    onClick: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        onClick = onClick
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(DesignTokens.Spacing.md),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Room avatar
            Surface(
                modifier = Modifier.size(48.dp),
                shape = MaterialTheme.shapes.medium,
                color = BrandPurple.copy(alpha = 0.1f)
            ) {
                Box(contentAlignment = Alignment.Center) {
                    Text(
                        text = room.name.firstOrNull()?.uppercase() ?: "?",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = BrandPurple
                    )
                }
            }

            Spacer(modifier = Modifier.width(DesignTokens.Spacing.md))

            // Room info
            Column(modifier = Modifier.weight(1f)) {
                Row(
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = room.name,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Medium
                    )
                    if (room.isEncrypted) {
                        Spacer(modifier = Modifier.width(4.dp))
                        Icon(
                            Icons.Default.Lock,
                            contentDescription = "Encrypted",
                            modifier = Modifier.size(14.dp),
                            tint = MaterialTheme.colorScheme.primary
                        )
                    }
                }
                Text(
                    text = "${room.memberCount} members",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                )
            }

            Icon(
                Icons.Default.ChevronRight,
                contentDescription = "Open room",
                tint = MaterialTheme.colorScheme.outline
            )
        }
    }
}

private data class SharedRoom(
    val id: String,
    val name: String,
    val memberCount: Int,
    val isEncrypted: Boolean
)
