package com.armorclaw.app.screens.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor

/**
 * Full home screen with room list
 *
 * This screen displays the user's chat rooms, organized into
 * active, favorites, and archived sections.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreenFull(
    onNavigateToChat: (roomId: String) -> Unit,
    onNavigateToSettings: () -> Unit,
    onNavigateToProfile: () -> Unit,
    onNavigateToSearch: () -> Unit,
    onCreateRoom: () -> Unit,
    onJoinRoom: () -> Unit,
    modifier: Modifier = Modifier
) {
    // View model state
    val activeRooms = remember { mutableStateListOf<RoomItem>() }
    val favoritedRooms = remember { mutableStateListOf<RoomItem>() }
    val archivedRooms = remember { mutableStateListOf<RoomItem>() }
    val unreadCount = remember { mutableIntStateOf(0) }
    val showFavorites = remember { mutableStateOf(true) }
    val showArchived = remember { mutableStateOf(false) }

    // Load actual data from repository (no mock data)
    LaunchedEffect(Unit) {
        // TODO: Load actual data from ViewModel
        // Data will be loaded from RoomRepository via HomeViewModel
    }
    
    // Top app bar
    val scrollBehavior = TopAppBarDefaults.pinnedScrollBehavior()
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Text(
                            text = "ArmorClaw",
                            style = MaterialTheme.typography.headlineSmall,
                            fontWeight = FontWeight.Bold
                        )
                        
                        // Unread badge
                        if (unreadCount.value > 0) {
                            UnreadBadge(count = unreadCount.value)
                        }
                    }
                },
                actions = {
                    // Search button
                    IconButton(onClick = onNavigateToSearch) {
                        Icon(
                            imageVector = Icons.Default.Search,
                            contentDescription = "Search"
                        )
                    }

                    // Profile button
                    IconButton(onClick = onNavigateToProfile) {
                        Icon(
                            imageVector = Icons.Default.AccountCircle,
                            contentDescription = "Profile"
                        )
                    }

                    // Settings button
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(
                            imageVector = Icons.Default.Settings,
                            contentDescription = "Settings"
                        )
                    }
                },
                scrollBehavior = scrollBehavior,
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = SurfaceColor
                )
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = onCreateRoom,
                containerColor = AccentColor
            ) {
                Icon(
                    imageVector = Icons.Default.Add,
                    contentDescription = "Create Room"
                )
            }
        },
        modifier = modifier
    ) { paddingValues ->
        // Room list
        RoomListContent(
            activeRooms = activeRooms,
            favoritedRooms = favoritedRooms,
            archivedRooms = archivedRooms,
            showFavorites = showFavorites.value,
            showArchived = showArchived.value,
            onToggleFavorites = { showFavorites.value = !showFavorites.value },
            onToggleArchived = { showArchived.value = !showArchived.value },
            onNavigateToChat = onNavigateToChat,
            onJoinRoom = onJoinRoom,
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        )
    }
}

@Composable
private fun RoomListContent(
    activeRooms: List<RoomItem>,
    favoritedRooms: List<RoomItem>,
    archivedRooms: List<RoomItem>,
    showFavorites: Boolean,
    showArchived: Boolean,
    onToggleFavorites: () -> Unit,
    onToggleArchived: () -> Unit,
    onNavigateToChat: (roomId: String) -> Unit,
    onJoinRoom: () -> Unit,
    modifier: Modifier = Modifier
) {
    LazyColumn(
        modifier = modifier,
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        // Join room button
        item {
            JoinRoomButton(onClick = onJoinRoom)
        }
        
        // Favorites section
        if (favoritedRooms.isNotEmpty()) {
            item {
                SectionHeader(
                    title = "Favorites (${favoritedRooms.size})",
                    isExpanded = showFavorites,
                    onToggle = onToggleFavorites
                )
            }
            
            if (showFavorites) {
                items(favoritedRooms) { room ->
                    RoomItemCard(
                        room = room,
                        onClick = { onNavigateToChat(room.id) }
                    )
                }
            }
        }
        
        // Active rooms section
        item {
            Spacer(modifier = Modifier.height(16.dp))
            SectionHeader(
                title = "Chats (${activeRooms.size})",
                isExpanded = true,
                onToggle = {}
            )
        }
        
        items(activeRooms) { room ->
            RoomItemCard(
                room = room,
                onClick = { onNavigateToChat(room.id) }
            )
        }
        
        // Archived rooms section
        if (archivedRooms.isNotEmpty()) {
            item {
                Spacer(modifier = Modifier.height(16.dp))
                SectionHeader(
                    title = "Archived (${archivedRooms.size})",
                    isExpanded = showArchived,
                    onToggle = onToggleArchived
                )
            }
            
            if (showArchived) {
                items(archivedRooms) { room ->
                    RoomItemCard(
                        room = room,
                        onClick = { onNavigateToChat(room.id) }
                    )
                }
            }
        }
    }
}

@Composable
private fun JoinRoomButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    OutlinedButton(
        onClick = onClick,
        modifier = modifier.fillMaxWidth(),
        colors = ButtonDefaults.outlinedButtonColors(
            contentColor = AccentColor
        )
    ) {
        Icon(
            imageVector = Icons.Default.Add,
            contentDescription = null,
            modifier = Modifier.size(20.dp)
        )
        Spacer(modifier = Modifier.width(8.dp))
        Text("Join a room")
    }
}

@Composable
private fun SectionHeader(
    title: String,
    isExpanded: Boolean,
    onToggle: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.small)
            .background(MaterialTheme.colorScheme.surfaceVariant)
            .clickable(onClick = onToggle)
            .padding(12.dp, 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold
        )
        
        Icon(
            imageVector = if (isExpanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore,
            contentDescription = if (isExpanded) "Collapse" else "Expand"
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RoomItemCard(
    room: RoomItem,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        onClick = onClick,
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Room avatar
            RoomAvatar(
                avatar = room.avatar,
                name = room.name,
                isEncrypted = room.isEncrypted
            )
            
            // Room info
            Column(
                modifier = Modifier.weight(1f),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    // Room name
                    Text(
                        text = room.name,
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f)
                    )
                    
                    // Timestamp
                    Text(
                        text = room.timestamp,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    // Last message
                    Text(
                        text = room.lastMessage,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f)
                    )
                    
                    // Unread count
                    if (room.unreadCount > 0) {
                        UnreadBadge(count = room.unreadCount)
                    }
                }
            }
        }
    }
}

@Composable
private fun RoomAvatar(
    avatar: String?,
    name: String,
    isEncrypted: Boolean,
    modifier: Modifier = Modifier
) {
    Box(
        modifier = modifier.size(48.dp)
    ) {
        // Avatar
        if (avatar != null) {
            // TODO: Load actual avatar image
            Text(
                text = "📷",
                fontSize = 32.sp,
                modifier = Modifier
                    .fillMaxSize()
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primaryContainer)
                    .padding(8.dp)
            )
        } else {
            // Initial avatar
            Text(
                text = name.firstOrNull()?.toString() ?: "?",
                fontSize = 24.sp,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                modifier = Modifier
                    .fillMaxSize()
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primaryContainer)
                    .padding(12.dp)
            )
        }
        
        // Encryption indicator
        if (isEncrypted) {
            Box(
                modifier = Modifier
                    .size(16.dp)
                    .background(AccentColor, CircleShape)
                    .align(Alignment.BottomEnd),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = "🔒",
                    fontSize = 10.sp
                )
            }
        }
    }
}

@Composable
private fun UnreadBadge(
    count: Int,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(AccentColor),
        color = AccentColor
    ) {
        Text(
            text = if (count > 99) "99+" else count.toString(),
            style = MaterialTheme.typography.labelSmall,
            color = Color.White,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            fontWeight = FontWeight.Bold
        )
    }
}

/**
 * Room item data class
 */
data class RoomItem(
    val id: String,
    val name: String,
    val avatar: String?,
    val lastMessage: String,
    val timestamp: String,
    val unreadCount: Int,
    val mentionCount: Int,
    val isEncrypted: Boolean,
    val isFavorited: Boolean
)

@Preview(showBackground = true)
@Composable
private fun HomeScreenFullPreview() {
    ArmorClawTheme {
        HomeScreenFull(
            onNavigateToChat = {},
            onNavigateToSettings = {},
            onNavigateToProfile = {},
            onNavigateToSearch = {},
            onCreateRoom = {},
            onJoinRoom = {}
        )
    }
}
