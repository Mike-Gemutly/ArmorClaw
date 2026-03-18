package com.armorclaw.app.screens.chat.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Popup
import androidx.compose.ui.window.PopupProperties
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.Primary

/**
 * Enhanced reaction display component that shows reaction summary and allows interaction
 * 
 * @param reactions List of message reactions
 * @param onReactionClick Callback when reaction is clicked
 * @param onAddReaction Callback when user wants to add a new reaction
 */
@Composable
fun ReactionDisplay(
    reactions: List<MessageReaction>,
    onReactionClick: () -> Unit,
    onAddReaction: () -> Unit
) {
    if (reactions.isEmpty()) return
    
    var showReactionPicker by remember { mutableStateOf(false) }
    
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onReactionClick)
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        reactions.forEach { reaction ->
            ReactionSummaryBadge(
                emoji = reaction.emoji,
                count = reaction.count,
                hasReacted = reaction.hasReacted,
                onClick = { onReactionClick() }
            )
        }
        
        Spacer(modifier = Modifier.weight(1f))
        
        Text(
            text = "+",
            color = Primary,
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Bold,
            modifier = Modifier
                .clickable(onClick = onAddReaction)
                .padding(horizontal = 8.dp)
        )
    }
    
    if (showReactionPicker) {
        ReactionPickerOverlay(
            onReactionSelected = { emoji -> 
                // Handle reaction selection
                showReactionPicker = false
            },
            onDismiss = { showReactionPicker = false }
        )
    }
}

@Composable
private fun ReactionSummaryBadge(
    emoji: String,
    count: Int,
    hasReacted: Boolean,
    onClick: () -> Unit
) {
    Surface(
        onClick = onClick,
        shape = RoundedCornerShape(12.dp),
        color = if (hasReacted)
            BrandPurple.copy(alpha = 0.3f)
        else
            Color.Transparent,
        border = if (hasReacted)
            androidx.compose.foundation.BorderStroke(1.dp, BrandPurple)
        else
            androidx.compose.foundation.BorderStroke(1.dp, Color.White.copy(alpha = 0.3f))
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = emoji,
                style = MaterialTheme.typography.bodySmall
            )
            if (count > 1) {
                Text(
                    text = count.toString(),
                    style = MaterialTheme.typography.bodySmall,
                    color = if (hasReacted) BrandPurple else Color.Black
                )
            }
        }
    }
}