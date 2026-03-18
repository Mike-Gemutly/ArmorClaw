package com.armorclaw.app.screens.chat.components

import androidx.compose.animation.core.animateIntAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.DesignTokens

val AVAILABLE_EMOJIS = listOf(
    "👍", "❤️", "😂", "😮", "😢", "😡",
    "🎉", "🔥", "❌", "✅", "❓", "👎"
)

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun ReactionPicker(
    onEmojiSelected: (String) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier
) {
    val selectedEmojiIndex by remember { mutableIntStateOf(-1) }

    Surface(
        modifier = modifier
            .width(300.dp)
            .clip(MaterialTheme.shapes.medium)
            .padding(DesignTokens.Spacing.sm),
        shadowElevation = 8.dp,
        shape = MaterialTheme.shapes.medium
    ) {
        Column(
            modifier = Modifier.padding(DesignTokens.Spacing.sm)
        ) {
            Text(
                text = "React to message",
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.onSurface,
                modifier = Modifier.padding(bottom = DesignTokens.Spacing.sm)
            )

            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 48.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                items(AVAILABLE_EMOJIS.size) { index ->
                    val emoji = AVAILABLE_EMOJIS[index]
                    val isSelected = selectedEmojiIndex == index

                    EmojiItem(
                        emoji = emoji,
                        isSelected = isSelected,
                        onClick = {
                            onEmojiSelected(emoji)
                            onDismiss()
                        }
                    )
                }
            }
        }
    }
}

@Composable
private fun EmojiItem(
    emoji: String,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    val size by animateIntAsState(
        targetValue = if (isSelected) 44 else 36,
        animationSpec = tween(durationMillis = 150),
        label = "emoji_size"
    )

    Surface(
        onClick = onClick,
        shape = androidx.compose.foundation.shape.CircleShape,
        color = if (isSelected) BrandPurple.copy(alpha = 0.2f) else Color.Transparent,
        modifier = Modifier
            .size(size.dp)
            .padding(2.dp)
    ) {
        Box(
            modifier = Modifier.fillMaxSize(),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = emoji,
                fontSize = 24.sp
            )
        }
    }
}
