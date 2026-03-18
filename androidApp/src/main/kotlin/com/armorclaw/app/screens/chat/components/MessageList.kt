package com.armorclaw.app.screens.chat.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChatBubbleOutline
import androidx.compose.material.icons.filled.Error
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.ui.theme.BrandPurple
import com.armorclaw.shared.ui.theme.BrandRed
import com.armorclaw.shared.ui.theme.OnBackground
import com.armorclaw.shared.ui.theme.Primary
import com.google.accompanist.swiperefresh.SwipeRefresh
import com.google.accompanist.swiperefresh.rememberSwipeRefreshState
import kotlinx.coroutines.delay

data class MessageListState(
    val messages: List<Message> = emptyList(),
    val isLoading: Boolean = false,
    val isLoadingMore: Boolean = false,
    val isRefreshing: Boolean = false,
    val hasMore: Boolean = true,
    val error: String? = null
)

@Composable
fun MessageList(
    state: MessageListState,
    modifier: Modifier = Modifier,
    onLoadMore: () -> Unit = {},
    onRefresh: () -> Unit = {},
    onReplyClick: (Message) -> Unit = {},
    onReactionClick: (Message) -> Unit = {},
    onAttachmentClick: (String) -> Unit = {}
) {
    val listState = rememberLazyListState()
    val swipeRefreshState = rememberSwipeRefreshState(state.isRefreshing)

    // Auto-scroll to latest when new messages arrive
    LaunchedEffect(state.messages.size) {
        if (state.messages.isNotEmpty()) {
            delay(100)
            listState.animateScrollToItem(0)
        }
    }

    // Load more when scrolling to top (because we use reverse layout)
    val firstVisibleItemIndex by remember {
        derivedStateOf {
            listState.firstVisibleItemIndex
        }
    }

    LaunchedEffect(firstVisibleItemIndex) {
        if (firstVisibleItemIndex >= listState.layoutInfo.visibleItemsInfo.size - 2 && state.hasMore && !state.isLoadingMore) {
            onLoadMore()
        }
    }

    SwipeRefresh(
        state = swipeRefreshState,
        onRefresh = onRefresh,
        modifier = modifier
    ) {
        when {
            // Loading state
            state.isLoading && state.messages.isEmpty() -> {
                LoadingState()
            }

            // Error state
            state.error != null && state.messages.isEmpty() -> {
                ErrorState(
                    message = state.error,
                    onRetry = onRefresh
                )
            }

            // Empty state
            !state.isLoading && state.messages.isEmpty() -> {
                EmptyState()
            }

            // Messages list
            else -> {
                LazyColumn(
                    state = listState,
                    modifier = Modifier.fillMaxSize(),
                    reverseLayout = true,
                    horizontalAlignment = Alignment.Start
                ) {
                    // Loading more indicator
                    if (state.isLoadingMore) {
                        item {
                            LoadingMoreIndicator()
                        }
                    }

                    // Messages
                    items(
                        items = state.messages,
                        key = { it.id }
                    ) { message ->
                        MessageBubble(
                            message = message,
                            onReplyClick = onReplyClick,
                            onReactionClick = onReactionClick,
                            onAttachmentClick = onAttachmentClick
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun LoadingState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        CircularProgressIndicator(
            color = BrandPurple
        )
    }
}

@Composable
private fun LoadingMoreIndicator() {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(16.dp),
        contentAlignment = Alignment.Center
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(24.dp),
            color = BrandPurple
        )
    }
}

@Composable
private fun ErrorState(
    message: String,
    onRetry: () -> Unit
) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Icon(
                imageVector = Icons.Default.Error,
                contentDescription = null,
                tint = BrandRed,
                modifier = Modifier.size(64.dp)
            )

            Text(
                text = "Failed to load messages",
                style = MaterialTheme.typography.titleMedium
            )

            Text(
                text = message,
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.7f)
            )

            Button(onClick = onRetry) {
                Text("Retry")
            }
        }
    }
}

@Composable
private fun EmptyState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Icon(
                imageVector = Icons.Default.ChatBubbleOutline,
                contentDescription = null,
                tint = OnBackground.copy(alpha = 0.3f),
                modifier = Modifier.size(80.dp)
            )

            Text(
                text = "No messages yet",
                style = MaterialTheme.typography.titleMedium
            )

            Text(
                text = "Start a conversation by sending a message",
                style = MaterialTheme.typography.bodyMedium,
                color = OnBackground.copy(alpha = 0.7f)
            )
        }
    }
}

@Composable
private fun MessageBubble(
    message: Message,
    onReplyClick: (Message) -> Unit,
    onReactionClick: (Message) -> Unit,
    onAttachmentClick: (String) -> Unit
) {
    // TODO: Implement message bubble UI
    Text(
        text = message.content.body,
        style = MaterialTheme.typography.bodyMedium,
        modifier = Modifier.padding(8.dp)
    )
}
