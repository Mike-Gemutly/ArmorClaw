package com.armorclaw.app.screens.room
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.clickable
import androidx.compose.ui.draw.clip

import androidx.compose.material3.MaterialTheme

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor

/**
 * Room settings screen
 * 
 * This screen allows users to configure room settings including:
 * - Room name and topic
 * - Room avatar
 * - Room privacy (Private/Public)
 * - Encryption settings
 * - Member permissions
 * - Room notifications
 * - Archive/Leave room
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RoomSettingsScreen(
    roomId: String,
    roomName: String,
    onNavigateBack: () -> Unit,
    onSave: (name: String, topic: String) -> Unit,
    onChangeAvatar: () -> Unit,
    onArchiveRoom: () -> Unit,
    onLeaveRoom: () -> Unit,
    modifier: Modifier = Modifier
) {
    val scrollState = rememberScrollState()
    
    // Room settings state
    var name by remember { mutableStateOf(roomName) }
    var topic by remember { mutableStateOf("") }
    var isPrivate by remember { mutableStateOf(true) }
    var isEncrypted by remember { mutableStateOf(true) }
    var avatar by remember { mutableStateOf<String?>(null) }
    var notifications by remember { mutableStateOf(true) }
    var mentions by remember { mutableStateOf(true) }
    var isAdmin by remember { mutableStateOf(true) }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Room Settings") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Default.Close,
                            contentDescription = "Close"
                        )
                    }
                },
                actions = {
                    TextButton(
                        onClick = { onSave(name, topic) },
                        colors = ButtonDefaults.textButtonColors(
                            contentColor = AccentColor
                        )
                    ) {
                        Text(
                            text = "Save",
                            fontWeight = FontWeight.SemiBold
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
                .verticalScroll(scrollState),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Room info section
            RoomInfoSection(
                avatar = avatar,
                name = name,
                topic = topic,
                isEncrypted = isEncrypted,
                onChangeAvatar = onChangeAvatar,
                onNameChange = { name = it },
                onTopicChange = { topic = it }
            )
            
            // Privacy section
            PrivacySection(
                isPrivate = isPrivate,
                isEncrypted = isEncrypted,
                isAdmin = isAdmin,
                onTogglePrivate = { isPrivate = it },
                onToggleEncrypted = { isEncrypted = it }
            )
            
            // Notifications section
            NotificationsSection(
                notifications = notifications,
                mentions = mentions,
                onToggleNotifications = { notifications = it },
                onToggleMentions = { mentions = it }
            )
            
            // Advanced section
            AdvancedSection(isAdmin = isAdmin)
            
            // Actions section
            ActionsSection(
                onArchiveRoom = onArchiveRoom,
                onLeaveRoom = onLeaveRoom
            )
        }
    }
}

@Composable
private fun RoomInfoSection(
    avatar: String?,
    name: String,
    topic: String,
    isEncrypted: Boolean,
    onChangeAvatar: () -> Unit,
    onNameChange: (String) -> Unit,
    onTopicChange: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Avatar
            Box(
                modifier = Modifier
                    .size(80.dp)
                    .align(Alignment.CenterHorizontally)
            ) {
                if (avatar != null) {
                    Box(
                        modifier = Modifier
                            .fillMaxSize()
                            .background(MaterialTheme.colorScheme.primaryContainer, RoundedCornerShape(20.dp)),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(text = "📷", fontSize = 40.sp)
                    }
                } else {
                    Box(
                        modifier = Modifier
                            .fillMaxSize()
                            .background(MaterialTheme.colorScheme.primaryContainer, RoundedCornerShape(20.dp)),
                        contentAlignment = Alignment.Center
                    ) {
                        Text(
                            text = name.firstOrNull()?.toString() ?: "?",
                            fontSize = 40.sp,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onPrimaryContainer
                        )
                    }
                }
                
                // Edit overlay
                Box(
                    modifier = Modifier
                        .size(32.dp)
                        .background(AccentColor, RoundedCornerShape(50))
                        .align(Alignment.BottomEnd)
                        .clickable(onClick = onChangeAvatar),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = Icons.Default.CameraAlt,
                        contentDescription = "Change avatar",
                        tint = Color.White,
                        modifier = Modifier.size(18.dp)
                    )
                }
            }
            
            // Room name
            OutlinedTextField(
                value = name,
                onValueChange = onNameChange,
                label = { Text("Room Name") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                shape = RoundedCornerShape(12.dp)
            )
            
            // Room topic
            OutlinedTextField(
                value = topic,
                onValueChange = onTopicChange,
                label = { Text("Room Topic (Optional)") },
                placeholder = { Text("What's this room about?") },
                modifier = Modifier.fillMaxWidth(),
                minLines = 2,
                maxLines = 3,
                shape = RoundedCornerShape(12.dp)
            )
            
            // Encryption indicator
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Default.Lock,
                    contentDescription = null,
                    tint = AccentColor,
                    modifier = Modifier.size(24.dp)
                )
                Text(
                    text = "Room is end-to-end encrypted",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                )
            }
        }
    }
}

@Composable
private fun PrivacySection(
    isPrivate: Boolean,
    isEncrypted: Boolean,
    isAdmin: Boolean,
    onTogglePrivate: (Boolean) -> Unit,
    onToggleEncrypted: (Boolean) -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = "Privacy",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Divider()
            
            // Private room
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "Private Room",
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = if (isPrivate)
                            "Only invited users can join"
                        else
                            "Anyone can join via room link",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Switch(
                    checked = isPrivate,
                    onCheckedChange = onTogglePrivate,
                    enabled = isAdmin
                )
            }
            
            Divider()
            
            // Encryption
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "End-to-End Encryption",
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = "Messages are encrypted before sending",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Switch(
                    checked = isEncrypted,
                    onCheckedChange = onToggleEncrypted,
                    enabled = isAdmin
                )
            }
        }
    }
}

@Composable
private fun NotificationsSection(
    notifications: Boolean,
    mentions: Boolean,
    onToggleNotifications: (Boolean) -> Unit,
    onToggleMentions: (Boolean) -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = "Notifications",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Divider()
            
            // All messages
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "All Messages",
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = "Get notified for all messages",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Switch(
                    checked = notifications,
                    onCheckedChange = onToggleNotifications
                )
            }
            
            Divider()
            
            // Mentions only
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "Mentions Only",
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = "Get notified only for mentions",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Switch(
                    checked = mentions,
                    onCheckedChange = onToggleMentions
                )
            }
        }
    }
}

@Composable
private fun AdvancedSection(
    isAdmin: Boolean,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = "Advanced",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Divider()
            
            SettingsItem(
                icon = Icons.Default.People,
                title = "Manage Members",
                description = "Add or remove members",
                onClick = { /* Navigate to members */ },
                enabled = isAdmin
            )
            
            Divider()
            
            SettingsItem(
                icon = Icons.Default.AdminPanelSettings,
                title = "Member Permissions",
                description = "Configure member roles",
                onClick = { /* Navigate to permissions */ },
                enabled = isAdmin
            )
            
            Divider()
            
            SettingsItem(
                icon = Icons.Default.History,
                title = "Message History",
                description = "View or clear message history",
                onClick = { /* Navigate to history */ },
                enabled = isAdmin
            )
        }
    }
}

@Composable
private fun ActionsSection(
    onArchiveRoom: () -> Unit,
    onLeaveRoom: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = "Actions",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Divider()
            
            SettingsItem(
                icon = Icons.Default.Archive,
                title = "Archive Room",
                description = "Hide room from main list",
                onClick = onArchiveRoom
            )
        }
    }

    // Leave room (danger)
    Card(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.errorContainer
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp)
        ) {
            SettingsItem(
                icon = Icons.Default.ExitToApp,
                title = "Leave Room",
                description = "Leave this room",
                onClick = onLeaveRoom,
                isDanger = true
            )
        }
    }
}

@Composable
private fun SettingsItem(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    title: String,
    description: String,
    onClick: () -> Unit,
    enabled: Boolean = true,
    isDanger: Boolean = false
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(enabled = enabled, onClick = onClick)
            .padding(12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = if (isDanger)
                MaterialTheme.colorScheme.onErrorContainer
            else
                AccentColor,
            modifier = Modifier.size(24.dp)
        )
        
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                color = if (isDanger)
                    MaterialTheme.colorScheme.onErrorContainer
                else
                    MaterialTheme.colorScheme.onSurface
            )
            Text(
                text = description,
                style = MaterialTheme.typography.bodySmall,
                color = if (isDanger)
                    MaterialTheme.colorScheme.onErrorContainer.copy(alpha = 0.7f)
                else
                    MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
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

@Preview(showBackground = true)
@Composable
private fun RoomSettingsScreenPreview() {
    ArmorClawTheme {
        RoomSettingsScreen(
            roomId = "!room1:matrix.org",
            roomName = "General",
            onNavigateBack = {},
            onSave = { _, _ -> },
            onChangeAvatar = {},
            onArchiveRoom = {},
            onLeaveRoom = {}
        )
    }
}
