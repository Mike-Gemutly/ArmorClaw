package com.armorclaw.app.screens.room
import androidx.compose.foundation.layout.Arrangement

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor

/**
 * Room management screen for creating and joining rooms
 * 
 * This screen allows users to create new rooms, join
 * existing rooms, and manage room settings.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RoomManagementScreen(
    onNavigateBack: () -> Unit,
    onNavigateToHome: () -> Unit,
    onCreateRoom: (name: String, topic: String?, isPrivate: Boolean) -> Unit,
    onJoinRoom: (roomId: String, alias: String?) -> Unit,
    modifier: Modifier = Modifier
) {
    var selectedTab by remember { mutableIntStateOf(0) }
    val tabs = listOf("Create Room", "Join Room")
    
    // Create room form
    var roomName by remember { mutableStateOf("") }
    var roomTopic by remember { mutableStateOf("") }
    var isPrivate by remember { mutableStateOf(true) }
    var avatar by remember { mutableStateOf<String?>(null) }
    
    // Join room form
    var roomId by remember { mutableStateOf("") }
    var roomAlias by remember { mutableStateOf("") }
    
    // Form validation
    val isCreateValid = roomName.isNotBlank()
    val isJoinValid = roomId.isNotBlank()
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Room Management") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Close"
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = SurfaceColor
                )
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            // Tabs
            TabRow(
                selectedTabIndex = selectedTab,
                modifier = Modifier.fillMaxWidth()
            ) {
                tabs.forEachIndexed { index, title ->
                    Tab(
                        selected = selectedTab == index,
                        onClick = { selectedTab = index },
                        text = { Text(title, fontWeight = if (selectedTab == index) FontWeight.Bold else FontWeight.Normal) }
                    )
                }
            }
            
            // Content based on selected tab
            when (selectedTab) {
                0 -> CreateRoomContent(
                    roomName = roomName,
                    roomTopic = roomTopic,
                    isPrivate = isPrivate,
                    avatar = avatar,
                    isValid = isCreateValid,
                    onNameChange = { roomName = it },
                    onTopicChange = { roomTopic = it },
                    onPrivacyToggle = { isPrivate = it },
                    onAvatarChange = { /* TODO: Implement avatar picker */ },
                    onCreate = { onCreateRoom(roomName, roomTopic.ifBlank { null }, isPrivate) },
                    modifier = Modifier.weight(1f)
                )
                1 -> JoinRoomContent(
                    roomId = roomId,
                    roomAlias = roomAlias,
                    isValid = isJoinValid,
                    onRoomIdChange = { roomId = it },
                    onAliasChange = { roomAlias = it },
                    onJoin = { onJoinRoom(roomId, roomAlias.ifBlank { null }) },
                    modifier = Modifier.weight(1f)
                )
            }
        }
    }
}

@Composable
private fun CreateRoomContent(
    roomName: String,
    roomTopic: String,
    isPrivate: Boolean,
    avatar: String?,
    isValid: Boolean,
    onNameChange: (String) -> Unit,
    onTopicChange: (String) -> Unit,
    onPrivacyToggle: (Boolean) -> Unit,
    onAvatarChange: () -> Unit,
    onCreate: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(24.dp)
    ) {
        // Avatar section
        AvatarSection(
            avatar = avatar,
            roomName = roomName,
            onChangeAvatar = onAvatarChange
        )
        
        // Room name
        OutlinedTextField(
            value = roomName,
            onValueChange = onNameChange,
            label = { Text("Room Name *") },
            placeholder = { Text("e.g., General, Team Alpha") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            shape = RoundedCornerShape(12.dp),
            keyboardOptions = KeyboardOptions(
                capitalization = KeyboardCapitalization.Words,
                autoCorrect = true
            ),
            isError = roomName.isBlank() && roomName.isNotEmpty()
        )
        
        // Room topic
        OutlinedTextField(
            value = roomTopic,
            onValueChange = onTopicChange,
            label = { Text("Topic (Optional)") },
            placeholder = { Text("What's this room about?") },
            modifier = Modifier.fillMaxWidth(),
            minLines = 2,
            maxLines = 4,
            shape = RoundedCornerShape(12.dp)
        )
        
        // Privacy settings
        PrivacySection(
            isPrivate = isPrivate,
            onToggle = onPrivacyToggle
        )
        
        // Create button
        Button(
            onClick = onCreate,
            enabled = isValid,
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.buttonColors(
                containerColor = AccentColor,
                disabledContainerColor = AccentColor.copy(alpha = 0.3f)
            ),
            shape = RoundedCornerShape(12.dp)
        ) {
            Text(
                text = "Create Room",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(12.dp)
            )
        }
        
        // Info text
        InfoCard(
            title = "About Room Privacy",
            items = listOf(
                "Private rooms: Only invited users can join",
                "Public rooms: Anyone can join via room link",
                "All rooms are end-to-end encrypted"
            )
        )
    }
}

@Composable
private fun JoinRoomContent(
    roomId: String,
    roomAlias: String,
    isValid: Boolean,
    onRoomIdChange: (String) -> Unit,
    onAliasChange: (String) -> Unit,
    onJoin: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(24.dp)
    ) {
        // Header
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(80.dp)
                    .background(
                        color = AccentColor.copy(alpha = 0.1f),
                        shape = CircleShape
                    ),
                contentAlignment = Alignment.Center
            ) {
                Text(text = "🔗", fontSize = 40.sp)
            }
            
            Text(
                text = "Join a Room",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )
            
            Text(
                text = "Enter the room ID or alias to join",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
                textAlign = TextAlign.Center
            )
        }
        
        // Room ID
        OutlinedTextField(
            value = roomId,
            onValueChange = onRoomIdChange,
            label = { Text("Room ID *") },
            placeholder = { Text("!room:matrix.org") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            shape = RoundedCornerShape(12.dp),
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Uri
            ),
            isError = roomId.isBlank() && roomId.isNotEmpty()
        )
        
        // Room alias
        OutlinedTextField(
            value = roomAlias,
            onValueChange = onAliasChange,
            label = { Text("Room Alias (Optional)") },
            placeholder = { Text("#room:matrix.org") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            shape = RoundedCornerShape(12.dp),
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Uri
            )
        )
        
        // Join button
        Button(
            onClick = onJoin,
            enabled = isValid,
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.buttonColors(
                containerColor = AccentColor,
                disabledContainerColor = AccentColor.copy(alpha = 0.3f)
            ),
            shape = RoundedCornerShape(12.dp)
        ) {
            Text(
                text = "Join Room",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(12.dp)
            )
        }
        
        // Info text
        InfoCard(
            title = "About Room IDs",
            items = listOf(
                "Room ID: Unique identifier for the room",
                "Room Alias: Human-readable address",
                "Ask room owner for the correct ID",
                "All communications are encrypted"
            )
        )
    }
}

@Composable
private fun AvatarSection(
    avatar: String?,
    roomName: String,
    onChangeAvatar: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier,
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Avatar
        Box(
            modifier = Modifier.size(100.dp)
        ) {
            if (avatar != null) {
                // TODO: Load actual avatar image
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(RoundedCornerShape(20.dp))
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Text(text = "📷", fontSize = 48.sp)
                }
            } else {
                // Initial avatar
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(RoundedCornerShape(20.dp))
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = roomName.firstOrNull()?.toString() ?: "?",
                        fontSize = 48.sp,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                }
            }
            
            // Edit overlay
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(CircleShape)
                    .background(AccentColor)
                    .align(Alignment.BottomEnd)
                    .clickable(onClick = onChangeAvatar),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = Icons.Default.CameraAlt,
                    contentDescription = "Change avatar",
                    tint = Color.White,
                    modifier = Modifier.size(20.dp)
                )
            }
        }
        
        // Avatar hint
        Text(
            text = "Tap to change room avatar",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
        )
    }
}

@Composable
private fun PrivacySection(
    isPrivate: Boolean,
    onToggle: (Boolean) -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = "Room Privacy",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(
                    modifier = Modifier.weight(1f)
                ) {
                    Text(
                        text = "Private Room",
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = "Only invited users can join",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Switch(
                    checked = isPrivate,
                    onCheckedChange = onToggle
                )
            }
        }
    }
}

@Composable
private fun InfoCard(
    title: String,
    items: List<String>
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = AccentColor.copy(alpha = 0.05f)
        ),
        border = androidx.compose.foundation.BorderStroke(
            width = 1.dp,
            color = AccentColor.copy(alpha = 0.2f)
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                color = AccentColor
            )
            
            items.forEach { item ->
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Text(
                        text = "•",
                        color = AccentColor
                    )
                    Text(
                        text = item,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                    )
                }
            }
        }
    }
}

@Preview(showBackground = true)
@Composable
private fun RoomManagementScreenPreview() {
    ArmorClawTheme {
        RoomManagementScreen(
            onNavigateBack = {},
            onNavigateToHome = {},
            onCreateRoom = { _, _, _ -> },
            onJoinRoom = { _, _ -> }
        )
    }
}
