package com.armorclaw.app.screens.onboarding

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CheckboxDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.Camera
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.DesignTokens

data class Permission(
    val id: String,
    val title: String,
    val description: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector,
    val required: Boolean,
    val granted: Boolean = false
)

/**
 * Map permission ID to the actual Android manifest permission string
 */
private fun permissionIdToManifest(id: String): String? = when (id) {
    "notifications" -> if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
        Manifest.permission.POST_NOTIFICATIONS
    } else null // Pre-Android 13: notifications don't need runtime permission
    "microphone" -> Manifest.permission.RECORD_AUDIO
    "camera" -> Manifest.permission.CAMERA
    else -> null
}

@OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)
@Composable
fun PermissionsScreen(
    onComplete: () -> Unit,
    onBack: () -> Unit
) {
    val context = LocalContext.current

    var permissions by remember {
        mutableStateOf(
            listOf(
                Permission(
                    id = "notifications",
                    title = "Notifications",
                    description = "Get notified when agent sends a message",
                    icon = Icons.Default.Notifications,
                    required = true
                ),
                Permission(
                    id = "microphone",
                    title = "Microphone",
                    description = "Dictate messages instead of typing",
                    icon = Icons.Default.Mic,
                    required = false
                ),
                Permission(
                    id = "camera",
                    title = "Camera",
                    description = "Send images to agents for analysis",
                    icon = Icons.Default.Camera,
                    required = false
                )
            )
        )
    }

    // Check which permissions are already granted on first composition
    LaunchedEffect(Unit) {
        permissions = permissions.map { perm ->
            val manifestPerm = permissionIdToManifest(perm.id)
            if (manifestPerm == null) {
                // No runtime permission needed (e.g., notifications on pre-Android 13)
                perm.copy(granted = true)
            } else {
                val alreadyGranted = ContextCompat.checkSelfPermission(
                    context, manifestPerm
                ) == PackageManager.PERMISSION_GRANTED
                perm.copy(granted = alreadyGranted)
            }
        }
    }

    // Track which permission index we're requesting
    var pendingPermissionIndex by remember { mutableStateOf(-1) }

    // Single permission launcher that updates the correct permission entry
    val permissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { granted ->
        if (pendingPermissionIndex in permissions.indices) {
            permissions = permissions.toMutableList().apply {
                this[pendingPermissionIndex] = this[pendingPermissionIndex].copy(granted = granted)
            }
            pendingPermissionIndex = -1
        }
    }

    val requiredGranted = permissions.count { it.required && it.granted }
    val requiredTotal = permissions.count { it.required }
    val canProceed = requiredGranted == requiredTotal
    
    ArmorClawTheme {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Permissions") },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                        }
                    }
                )
            }
        ) { paddingValues ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(paddingValues)
                    .padding(DesignTokens.Spacing.lg)
                    .verticalScroll(rememberScrollState()),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
                
                // Header
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Icon(
                        imageVector = Icons.Default.Notifications,
                        contentDescription = null,
                        tint = BrandPurple,
                        modifier = Modifier.size(64.dp)
                    )
                    
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.md))
                    
                    Text(
                        text = "We need a few permissions",
                        style = MaterialTheme.typography.headlineSmall
                    )
                    
                    Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))
                    
                    Text(
                        text = "$requiredGranted of $requiredTotal required permissions granted",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.xl))
                
                // Permissions list
                Column(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                ) {
                    permissions.forEachIndexed { index, permission ->
                        PermissionCard(
                            permission = permission,
                            onGrant = {
                                val manifestPerm = permissionIdToManifest(permission.id)
                                if (manifestPerm != null) {
                                    pendingPermissionIndex = index
                                    permissionLauncher.launch(manifestPerm)
                                } else {
                                    // No runtime permission needed
                                    permissions = permissions.toMutableList().apply {
                                        this[index] = permission.copy(granted = true)
                                    }
                                }
                            },
                            onSkip = {
                                permissions = permissions.toMutableList().apply {
                                    this[index] = permission.copy(granted = false)
                                }
                            }
                        )
                    }
                }
                
                Spacer(modifier = Modifier.weight(1f))
                
                // Info
                PermissionInfoCard()
                
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.lg))
                
                // Actions
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md)
                ) {
                    TextButton(
                        onClick = onBack,
                        modifier = Modifier.weight(1f)
                    ) {
                        Text("Back")
                    }
                    
                    Button(
                        onClick = onComplete,
                        modifier = Modifier.weight(1f),
                        enabled = canProceed
                    ) {
                        Text("Continue")
                    }
                }
            }
        }
    }
}

@Composable
private fun PermissionCard(
    permission: Permission,
    onGrant: () -> Unit,
    onSkip: () -> Unit
) {
    val cardColor by animateColorAsState(
        targetValue = if (permission.granted)
            BrandGreen.copy(alpha = 0.1f)
        else
            MaterialTheme.colorScheme.surface,
        animationSpec = tween(300),
        label = "cardColor"
    )
    
    val borderColor by animateColorAsState(
        targetValue = if (permission.granted)
            BrandGreen
        else
            BrandPurple,
        animationSpec = tween(300),
        label = "borderColor"
    )
    
    val iconColor by animateColorAsState(
        targetValue = if (permission.granted)
            BrandGreen
        else
            if (permission.required) BrandPurple else Color.Gray,
        animationSpec = tween(300),
        label = "iconColor"
    )
    
    val scale by animateFloatAsState(
        targetValue = if (permission.granted) 1.05f else 1f,
        animationSpec = tween(300),
        label = "scale"
    )
    
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .scale(scale)
            .border(
                width = 2.dp,
                color = borderColor,
                shape = RoundedCornerShape(12.dp)
            ),
        colors = CardDefaults.cardColors(containerColor = cardColor),
        elevation = CardDefaults.cardElevation(defaultElevation = if (permission.granted) 4.dp else 2.dp)
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.lg)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Icon and text
                Row(
                    horizontalArrangement = Arrangement.spacedBy(DesignTokens.Spacing.md),
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(
                        imageVector = permission.icon,
                        contentDescription = null,
                        tint = iconColor,
                        modifier = Modifier.size(32.dp)
                    )
                    
                    Column {
                        Text(
                            text = permission.title,
                            style = MaterialTheme.typography.titleMedium.copy(
                                fontWeight = FontWeight.Bold
                            ),
                            color = iconColor
                        )
                        
                        Text(
                            text = permission.description,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f)
                        )
                    }
                }
                
                // Check icon or grant button
                if (permission.granted) {
                    Icon(
                        imageVector = Icons.Default.CheckCircle,
                        contentDescription = "Granted",
                        tint = BrandGreen,
                        modifier = Modifier.size(32.dp)
                    )
                } else {
                    Button(
                        onClick = onGrant,
                        modifier = Modifier.height(36.dp)
                    ) {
                        Text("Grant", style = MaterialTheme.typography.labelLarge)
                    }
                }
            }
            
            // Optional skip button
            if (!permission.required && !permission.granted) {
                Spacer(modifier = Modifier.height(DesignTokens.Spacing.sm))
                
                TextButton(
                    onClick = onSkip,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = DesignTokens.Spacing.xs)
                ) {
                    Text(
                        "Skip for now",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
                    )
                }
            }
        }
    }
}

@Composable
private fun PermissionInfoCard() {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = BrandPurple.copy(alpha = 0.1f)),
        border = androidx.compose.foundation.BorderStroke(
            1.dp,
            BrandPurple.copy(alpha = 0.3f)
        )
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.md)
        ) {
            Text(
                text = "Why do we need these?",
                style = MaterialTheme.typography.titleSmall.copy(
                    fontWeight = FontWeight.Bold
                ),
                color = BrandPurple
            )
            
            Spacer(modifier = Modifier.height(DesignTokens.Spacing.xs))
            
            Text(
                text = "• Notifications: Never miss an important message\n" +
                        "• Microphone: Send voice messages easily\n" +
                        "• Camera: Share images for AI analysis",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
            )
        }
    }
}
