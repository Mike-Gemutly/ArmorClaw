package com.armorclaw.app.screens.room
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.text.style.TextAlign

import androidx.compose.material3.MaterialTheme

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
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor

/**
 * Room details screen
 * 
 * This screen displays room details including:
 * - Room information (name, topic, avatar)
 * - Room settings (privacy, encryption)
 * - Room members
 * - Room actions (leave, archive)
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RoomDetailsScreen(
    roomId: String,
    onNavigateBack: () -> Unit,
    onNavigateToSettings: () -> Unit,
    onLeaveRoom: () -> Unit,
    onArchiveRoom: () -> Unit,
    onVerifyBridge: ((deviceId: String) -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    // TODO: Load actual room data from repository
    var room by remember { mutableStateOf(
        RoomDetails(
            id = roomId,
            name = "Loading...",
            topic = "",
            avatar = null,
            isEncrypted = true,
            isPrivate = true,
            memberCount = 0,
            isAdmin = false
        )
    ) }

    // TODO: Load actual members from repository
    val members by remember { mutableStateOf(emptyList<RoomMember>()) }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Room Details") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Default.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(
                            imageVector = Icons.Default.Settings,
                            contentDescription = "Settings"
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
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Bridge verification banner (shown when Bridge is unverified)
            item {
                BridgeVerificationBanner(
                    roomId = roomId,
                    onVerifyBridge = onVerifyBridge
                )
            }

            // Room information section
            item {
                RoomInfoCard(room = room)
            }
            
            // Room settings section
            item {
                RoomSettingsCard(room = room, onTogglePrivacy = { isPrivate ->
                    room = room.copy(isPrivate = isPrivate)
                })
            }
            
            // Room members section
            item {
                RoomMembersCard(
                    members = members,
                    memberCount = room.memberCount
                )
            }
            
            // Room actions section
            item {
                RoomActionsCard(
                    isAdmin = room.isAdmin,
                    onLeaveRoom = onLeaveRoom,
                    onArchiveRoom = onArchiveRoom
                )
            }
            
            // Dangerous actions section
            item {
                DangerousActionsCard(
                    roomId = room.id,
                    onLeaveRoom = onLeaveRoom
                )
            }
        }
    }
}

/**
 * Banner that warns when a Bridge device is connected but unverified.
 * Without verification, the Bridge cannot decrypt E2EE messages for
 * SDTW (Slack/Discord/Teams/WhatsApp) bridging.
 *
 * This checks for a bridge device in the room and shows a prominent
 * warning if it hasn't been verified via cross-signing.
 */
@Composable
private fun BridgeVerificationBanner(
    roomId: String,
    onVerifyBridge: ((deviceId: String) -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    // TODO: Replace with actual bridge device detection from MatrixClient
    // This would query the room members for a device with a bridge user agent,
    // then check its verification status via VerificationRepository.
    //
    // val bridgeDevice = matrixClient.getRoomBridgeDevice(roomId)
    // val isVerified = bridgeDevice?.let { verificationRepo.getTrustLevel(it.userId, it.deviceId) }
    //
    // For now, simulate: show banner if bridge is configured but not verified
    val hasBridgeDevice = remember { mutableStateOf(true) }  // TODO: detect from room state
    val isBridgeVerified = remember { mutableStateOf(false) } // TODO: check trust level
    val bridgeDeviceId = remember { "bridge_device_001" }     // TODO: get actual device ID

    if (hasBridgeDevice.value && !isBridgeVerified.value && onVerifyBridge != null) {
        Card(
            modifier = modifier
                .fillMaxWidth()
                .padding(horizontal = 0.dp),
            colors = CardDefaults.cardColors(
                containerColor = Color(0xFFFFF3E0) // Warning amber
            ),
            shape = RoundedCornerShape(12.dp)
        ) {
            Column(
                modifier = Modifier.padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Warning,
                        contentDescription = null,
                        tint = Color(0xFFE65100),
                        modifier = Modifier.size(24.dp)
                    )
                    Text(
                        text = "Bridge Device Not Verified",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.Bold,
                        color = Color(0xFFE65100)
                    )
                }

                Text(
                    text = "The SDTW Bridge cannot decrypt messages in this room. " +
                           "Verify the Bridge device to enable Slack, Discord, and Teams bridging.",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFF795548)
                )

                Button(
                    onClick = { onVerifyBridge(bridgeDeviceId) },
                    colors = ButtonDefaults.buttonColors(
                        containerColor = Color(0xFFE65100)
                    ),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Icon(
                        imageVector = Icons.Default.VerifiedUser,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp)
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Verify Bridge Device")
                }
            }
        }
    }
}

@Composable
private fun RoomInfoCard(
    room: RoomDetails,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(20.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Room avatar
            RoomAvatar(
                avatar = room.avatar,
                name = room.name,
                isEncrypted = room.isEncrypted,
                size = 100.dp
            )
            
            // Room name
            Text(
                text = room.name,
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )
            
            // Room topic
            if (room.topic.isNotBlank()) {
                Text(
                    text = room.topic,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
                    textAlign = TextAlign.Center
                )
            }
            
            // Room ID
            Text(
                text = "Room ID: ${room.id}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                textAlign = TextAlign.Center
            )
        }
    }
}

@Composable
private fun RoomSettingsCard(
    room: RoomDetails,
    onTogglePrivacy: (Boolean) -> Unit,
    modifier: Modifier = Modifier
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
                text = "Room Settings",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Divider()
            
            // Privacy setting
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
                        text = if (room.isPrivate)
                            "Only invited users can join"
                        else
                            "Anyone can join via room link",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Switch(
                    checked = room.isPrivate,
                    onCheckedChange = onTogglePrivacy,
                    enabled = room.isAdmin
                )
            }
            
            Divider()
            
            // Encryption info
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
                
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "End-to-End Encrypted",
                        style = MaterialTheme.typography.bodyLarge,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = "All messages are encrypted",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
            }
        }
    }
}

@Composable
private fun RoomMembersCard(
    members: List<RoomMember>,
    memberCount: Int,
    modifier: Modifier = Modifier
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
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Members ($memberCount)",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold
                )
                
                TextButton(
                    onClick = { /* View all members */ }
                ) {
                    Text("View All")
                }
            }
            
            Divider()
            
            // Member list
            members.take(5).forEach { member ->
                MemberItem(
                    member = member,
                    onClick = { /* View member details */ }
                )
            }
            
            if (memberCount > 5) {
                TextButton(
                    onClick = { /* View all members */ },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("View All $memberCount Members")
                }
            }
        }
    }
}

@Composable
private fun MemberItem(
    member: RoomMember,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .clickable(onClick = onClick)
            .padding(12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        // Member avatar
        Box(
            modifier = Modifier
                .size(48.dp)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.primaryContainer),
            contentAlignment = Alignment.Center
        ) {
            if (member.avatar != null) {
                // TODO: Load actual avatar
                Text(text = "📷", fontSize = 24.sp)
            } else {
                Text(
                    text = member.name.firstOrNull()?.toString() ?: "?",
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onPrimaryContainer
                )
            }
        }
        
        // Member info
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = member.name,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium
            )
            if (member.isAdmin) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        imageVector = Icons.Default.Star,
                        contentDescription = null,
                        tint = AccentColor,
                        modifier = Modifier.size(14.dp)
                    )
                    Text(
                        text = "Admin",
                        style = MaterialTheme.typography.bodySmall,
                        color = AccentColor
                    )
                }
            }
        }
        
        // Status indicator
        Box(
            modifier = Modifier
                .size(12.dp)
                .background(
                    color = when (member.status) {
                        MemberStatus.ONLINE -> Color(0xFF4CAF50)
                        MemberStatus.AWAY -> Color(0xFFFF9800)
                        MemberStatus.BUSY -> Color(0xFFF44336)
                        MemberStatus.OFFLINE -> Color(0xFF9E9E9E)
                    },
                    shape = CircleShape
                )
        )
    }
}

@Composable
private fun RoomActionsCard(
    isAdmin: Boolean,
    onLeaveRoom: () -> Unit,
    onArchiveRoom: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Text(
                text = "Room Actions",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )
            
            Divider()
            
            // Archive room
            OutlinedButton(
                onClick = onArchiveRoom,
                modifier = Modifier.fillMaxWidth(),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = MaterialTheme.colorScheme.onSurface
                )
            ) {
                Icon(
                    imageVector = Icons.Default.Archive,
                    contentDescription = null
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(if (isAdmin) "Archive Room" else "Leave Room")
            }
        }
    }
}

@Composable
private fun DangerousActionsCard(
    roomId: String,
    onLeaveRoom: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.errorContainer
        )
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Text(
                text = "Dangerous Actions",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onErrorContainer
            )
            
            Divider(color = MaterialTheme.colorScheme.onErrorContainer.copy(alpha = 0.3f))
            
            // Leave room
            Button(
                onClick = onLeaveRoom,
                modifier = Modifier.fillMaxWidth(),
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.error,
                    contentColor = MaterialTheme.colorScheme.onError
                )
            ) {
                Icon(
                    imageVector = Icons.Default.ExitToApp,
                    contentDescription = null
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Leave Room")
            }
        }
    }
}

@Composable
private fun RoomAvatar(
    avatar: String?,
    name: String,
    isEncrypted: Boolean,
    size: androidx.compose.ui.unit.Dp,
    modifier: Modifier = Modifier
) {
    Box(modifier = modifier.size(size)) {
        // Avatar
        if (avatar != null) {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primaryContainer),
                contentAlignment = Alignment.Center
            ) {
                Text(text = "📷", fontSize = (size.value / 2).sp)
            }
        } else {
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primaryContainer),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = name.firstOrNull()?.toString() ?: "?",
                    fontSize = (size.value * 0.5f).sp,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onPrimaryContainer
                )
            }
        }
        
        // Encryption indicator
        if (isEncrypted) {
            Box(
                modifier = Modifier
                    .size(24.dp)
                    .background(AccentColor, CircleShape)
                    .align(Alignment.BottomEnd),
                contentAlignment = Alignment.Center
            ) {
                Text(text = "🔒", fontSize = 12.sp)
            }
        }
    }
}

/**
 * Room details data class
 */
data class RoomDetails(
    val id: String,
    val name: String,
    val topic: String,
    val avatar: String?,
    val isEncrypted: Boolean,
    val isPrivate: Boolean,
    val memberCount: Int,
    val isAdmin: Boolean
)

/**
 * Room member data class
 */
data class RoomMember(
    val id: String,
    val name: String,
    val avatar: String?,
    val isAdmin: Boolean,
    val status: MemberStatus
)

/**
 * Member status enum
 */
enum class MemberStatus {
    ONLINE,
    AWAY,
    BUSY,
    OFFLINE
}

@Preview(showBackground = true)
@Composable
private fun RoomDetailsScreenPreview() {
    ArmorClawTheme {
        RoomDetailsScreen(
            roomId = "!room1:matrix.org",
            onNavigateBack = {},
            onNavigateToSettings = {},
            onLeaveRoom = {},
            onArchiveRoom = {}
        )
    }
}
