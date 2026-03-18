package com.armorclaw.app.screens.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.UserPresence
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandRed

/**
 * User profile screen for viewing other users' profiles
 *
 * Displays user information, status, and available actions.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UserProfileScreen(
    userId: String,
    displayName: String?,
    avatarUrl: String?,
    bio: String?,
    presence: UserPresence?,
    isVerified: Boolean?,
    sharedRoomsCount: Int,
    onNavigateBack: () -> Unit,
    onSendMessage: () -> Unit,
    onStartCall: (() -> Unit)? = null,
    onViewSharedRooms: (() -> Unit)? = null,
    onBlockUser: (() -> Unit)? = null,
    onReportUser: (() -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Profile") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(
                            imageVector = Icons.Filled.ArrowBack,
                            contentDescription = "Back"
                        )
                    }
                },
                actions = {
                    var showMenu by remember { mutableStateOf(false) }
                    
                    IconButton(onClick = { showMenu = true }) {
                        Icon(Icons.Default.MoreVert, "More options")
                    }
                    
                    DropdownMenu(
                        expanded = showMenu,
                        onDismissRequest = { showMenu = false }
                    ) {
                        onBlockUser?.let {
                            DropdownMenuItem(
                                text = { Text("Block User") },
                                onClick = {
                                    showMenu = false
                                    it()
                                },
                                leadingIcon = {
                                    Icon(Icons.Default.Block, null)
                                }
                            )
                        }
                        onReportUser?.let {
                            DropdownMenuItem(
                                text = { Text("Report User") },
                                onClick = {
                                    showMenu = false
                                    it()
                                },
                                leadingIcon = {
                                    Icon(Icons.Default.Report, null)
                                }
                            )
                        }
                    }
                }
            )
        },
        modifier = modifier
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(rememberScrollState())
        ) {
            // Profile header
            ProfileHeaderSection(
                userId = userId,
                displayName = displayName,
                avatarUrl = avatarUrl,
                presence = presence,
                isVerified = isVerified
            )
            
            // Bio section
            bio?.let { bioText ->
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)
                ) {
                    Column(
                        modifier = Modifier.padding(16.dp)
                    ) {
                        Text(
                            text = "Bio",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = bioText,
                            style = MaterialTheme.typography.bodyLarge
                        )
                    }
                }
                Spacer(modifier = Modifier.height(8.dp))
            }
            
            // Shared rooms
            if (sharedRoomsCount > 0) {
                Surface(
                    onClick = { onViewSharedRooms?.invoke() },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Row(
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Default.Group,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.primary
                            )
                            Column {
                                Text(
                                    text = "Shared Rooms",
                                    style = MaterialTheme.typography.bodyLarge
                                )
                                Text(
                                    text = "$sharedRoomsCount rooms in common",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                        Icon(
                            imageVector = Icons.Default.ChevronRight,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
                Spacer(modifier = Modifier.height(8.dp))
            }
            
            // Verification info
            isVerified?.let { verified ->
                if (verified) {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        color = BrandGreen.copy(alpha = 0.1f)
                    ) {
                        Row(
                            modifier = Modifier.padding(16.dp),
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Icon(
                                imageVector = Icons.Default.VerifiedUser,
                                contentDescription = null,
                                tint = BrandGreen
                            )
                            Column {
                                Text(
                                    text = "Verified",
                                    style = MaterialTheme.typography.bodyLarge,
                                    fontWeight = FontWeight.Bold,
                                    color = BrandGreen
                                )
                                Text(
                                    text = "You have verified this user",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = BrandGreen.copy(alpha = 0.8f)
                                )
                            }
                        }
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                }
            }
            
            Spacer(modifier = Modifier.weight(1f))
            
            // Action buttons
            ActionButtonsSection(
                onSendMessage = onSendMessage,
                onStartCall = onStartCall
            )
        }
    }
}

@Composable
private fun ProfileHeaderSection(
    userId: String,
    displayName: String?,
    avatarUrl: String?,
    presence: UserPresence?,
    isVerified: Boolean?
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // Avatar with presence indicator
        Box(contentAlignment = Alignment.BottomEnd) {
            Box(
                modifier = Modifier
                    .size(100.dp)
                    .clip(CircleShape)
                    .background(BrandPurple),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = (displayName ?: userId).firstOrNull()?.uppercase() ?: "?",
                    style = MaterialTheme.typography.displayMedium,
                    color = Color.White,
                    fontWeight = FontWeight.Bold
                )
            }
            
            // Presence indicator
            presence?.let {
                val (color, icon) = when (it) {
                    UserPresence.ONLINE -> BrandGreen to Icons.Default.CheckCircle
                    UserPresence.UNAVAILABLE -> BrandRed to Icons.Default.RemoveCircle
                    UserPresence.OFFLINE -> Color.Gray to Icons.Default.Circle
                    UserPresence.UNKNOWN -> Color.Gray to Icons.Default.Help
                }
                
                Surface(
                    modifier = Modifier.size(28.dp),
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.surface,
                    border = androidx.compose.foundation.BorderStroke(2.dp, color)
                ) {
                    Icon(
                        imageVector = icon,
                        contentDescription = null,
                        modifier = Modifier.padding(4.dp),
                        tint = color
                    )
                }
            }
        }
        
        Spacer(modifier = Modifier.height(16.dp))
        
        // Display name with verification badge
        Row(
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = displayName ?: "Unknown User",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold
            )
            if (isVerified == true) {
                Icon(
                    imageVector = Icons.Default.Verified,
                    contentDescription = "Verified",
                    tint = BrandGreen,
                    modifier = Modifier.size(24.dp)
                )
            }
        }
        
        Spacer(modifier = Modifier.height(4.dp))
        
        // User ID
        Text(
            text = userId,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        
        // Presence status
        presence?.let {
            Spacer(modifier = Modifier.height(8.dp))
            val (statusText, statusColor) = when (it) {
                UserPresence.ONLINE -> "Online" to BrandGreen
                UserPresence.UNAVAILABLE -> "Away" to BrandRed
                UserPresence.OFFLINE -> "Offline" to Color.Gray
                UserPresence.UNKNOWN -> "Unknown" to Color.Gray
            }
            Surface(
                shape = MaterialTheme.shapes.small,
                color = statusColor.copy(alpha = 0.1f)
            ) {
                Text(
                    text = statusText,
                    style = MaterialTheme.typography.labelMedium,
                    color = statusColor,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                )
            }
        }
    }
}

@Composable
private fun ActionButtonsSection(
    onSendMessage: () -> Unit,
    onStartCall: (() -> Unit)?
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        tonalElevation = 2.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            // Message button
            Button(
                onClick = onSendMessage,
                modifier = Modifier.weight(1f)
            ) {
                Icon(
                    imageVector = Icons.Default.Mail,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Message")
            }
            
            // Call button
            onStartCall?.let { onCall ->
                FilledTonalButton(
                    onClick = onCall,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(
                        imageVector = Icons.Default.Call,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp)
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Call")
                }
            }
        }
    }
}
