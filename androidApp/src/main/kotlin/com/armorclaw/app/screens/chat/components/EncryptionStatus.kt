package com.armorclaw.app.screens.chat.components
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width

import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.LockOpen
import androidx.compose.material.icons.filled.VerifiedUser
import androidx.compose.material.icons.outlined.Info
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.ui.theme.ArmorClawTypography
import com.armorclaw.shared.ui.theme.BrandGreen
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandRed

enum class EncryptionStatus {
    NONE,
    UNENCRYPTED,
    UNVERIFIED,
    VERIFIED
}

@Composable
fun EncryptionStatusIndicator(
    status: EncryptionStatus,
    modifier: Modifier = Modifier,
    showText: Boolean = false
) {
    val (icon, color, text) = when (status) {
        EncryptionStatus.NONE -> Triple(
            Icons.Outlined.Info,
            Color.Gray,
            "Encryption not available"
        )
        EncryptionStatus.UNENCRYPTED -> Triple(
            Icons.Default.LockOpen,
            BrandRed,
            "Not encrypted"
        )
        EncryptionStatus.UNVERIFIED -> Triple(
            Icons.Default.Lock,
            BrandPurple,
            "Unverified encryption"
        )
        EncryptionStatus.VERIFIED -> Triple(
            Icons.Default.VerifiedUser,
            BrandGreen,
            "Verified encryption"
        )
    }
    
    Row(
        modifier = modifier
            .padding(4.dp)
            .semantics { contentDescription = text },
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = icon,
            contentDescription = text,
            tint = color,
            modifier = Modifier.size(16.dp)
        )
        
        if (showText) {
            Text(
                text = text,
                style = MaterialTheme.typography.bodySmall.copy(
                    fontWeight = FontWeight.Medium,
                    color = color
                )
            )
        }
    }
}

@Composable
fun EncryptionInfoCard(
    status: EncryptionStatus,
    members: List<String>,
    modifier: Modifier = Modifier
) {
    val (title, description, color) = when (status) {
        EncryptionStatus.NONE -> Triple(
            "Encryption Not Available",
            "This room does not support end-to-end encryption.",
            Color.Gray
        )
        EncryptionStatus.UNENCRYPTED -> Triple(
            "Unencrypted",
            "Messages are not encrypted. This is not recommended.",
            BrandRed
        )
        EncryptionStatus.UNVERIFIED -> Triple(
            "Unverified Encryption",
            "Messages are encrypted, but the encryption keys could not be verified.",
            BrandPurple
        )
        EncryptionStatus.VERIFIED -> Triple(
            "Verified Encryption",
            "Messages are end-to-end encrypted and verified with trusted devices.",
            BrandGreen
        )
    }

    Card(
        modifier = modifier,
        colors = CardDefaults.cardColors(
            containerColor = color.copy(alpha = 0.1f)
        ),
        border = androidx.compose.foundation.BorderStroke(1.dp, color)
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(8.dp)
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = when (status) {
                        EncryptionStatus.VERIFIED -> Icons.Default.VerifiedUser
                        EncryptionStatus.UNENCRYPTED -> Icons.Default.LockOpen
                        else -> Icons.Default.Lock
                    },
                    contentDescription = title,
                    tint = color,
                    modifier = Modifier.size(20.dp)
                )
                
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium.copy(
                        fontWeight = FontWeight.Bold,
                        color = color
                    )
                )
            }
            
            Text(
                text = description,
                style = MaterialTheme.typography.bodyMedium
            )
            
            if (status != EncryptionStatus.NONE && members.isNotEmpty()) {
                Spacer(modifier = Modifier.height(4.dp))
                
                Text(
                    text = "Members with verified devices:",
                    style = MaterialTheme.typography.bodySmall.copy(
                        fontWeight = FontWeight.Bold
                    )
                )
                
                members.forEach { member ->
                    Text(
                        text = "• $member",
                        style = MaterialTheme.typography.bodySmall
                    )
                }
            }
        }
    }
}
