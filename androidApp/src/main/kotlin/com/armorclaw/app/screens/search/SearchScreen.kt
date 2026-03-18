package com.armorclaw.app.screens.search
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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.armorclaw.shared.ui.theme.ArmorClawTheme
import com.armorclaw.shared.ui.theme.AccentColor
import com.armorclaw.shared.ui.theme.SurfaceColor

/**
 * Global search screen
 * 
 * This screen allows users to search for:
 * - Rooms
 * - Messages within rooms
 * - People
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SearchScreen(
    onNavigateBack: () -> Unit,
    onNavigateToRoom: (roomId: String) -> Unit,
    onNavigateToMessage: (roomId: String, messageId: String) -> Unit,
    modifier: Modifier = Modifier
) {
    // Search state
    var query by remember { mutableStateOf("") }
    var searchType by remember { mutableStateOf(SearchType.ALL) }
    var isSearching by remember { mutableStateOf(false) }
    
    // Mock search results
    var roomResults by remember { mutableStateOf(emptyList<RoomSearchResult>()) }
    var messageResults by remember { mutableStateOf(emptyList<MessageSearchResult>()) }
    var peopleResults by remember { mutableStateOf(emptyList<PersonSearchResult>()) }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { },
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
                .background(SurfaceColor),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Search bar
            SearchBar(
                query = query,
                onQueryChange = { query = it },
                onSearch = {
                    isSearching = true
                    // TODO: Perform real search - replace mock data
                    // roomResults = searchRooms(query)
                    // messageResults = searchMessages(query)
                    // peopleResults = searchPeople(query)
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp)
            )
            
            // Search type tabs
            SearchTypeTabs(
                selectedType = searchType,
                onSelectType = { searchType = it },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp)
            )
            
            // Search results
            LazyColumn(
                modifier = Modifier.weight(1f),
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                if (isSearching) {
                    // Room results
                    if (searchType == SearchType.ALL || searchType == SearchType.ROOMS) {
                        if (roomResults.isNotEmpty()) {
                            item {
                                Text(
                                    text = "Rooms (${roomResults.size})",
                                    style = MaterialTheme.typography.labelLarge,
                                    fontWeight = FontWeight.SemiBold,
                                    color = AccentColor
                                )
                            }
                            items(roomResults) { result ->
                                RoomSearchResultItem(
                                    result = result,
                                    onClick = { onNavigateToRoom(result.id) }
                                )
                            }
                        }
                    }
                    
                    // Message results
                    if (searchType == SearchType.ALL || searchType == SearchType.MESSAGES) {
                        if (messageResults.isNotEmpty()) {
                            item {
                                Text(
                                    text = "Messages (${messageResults.size})",
                                    style = MaterialTheme.typography.labelLarge,
                                    fontWeight = FontWeight.SemiBold,
                                    color = AccentColor
                                )
                            }
                            items(messageResults) { result ->
                                MessageSearchResultItem(
                                    result = result,
                                    onClick = { onNavigateToMessage(result.roomId, result.id) }
                                )
                            }
                        }
                    }
                    
                    // People results
                    if (searchType == SearchType.ALL || searchType == SearchType.PEOPLE) {
                        if (peopleResults.isNotEmpty()) {
                            item {
                                Text(
                                    text = "People (${peopleResults.size})",
                                    style = MaterialTheme.typography.labelLarge,
                                    fontWeight = FontWeight.SemiBold,
                                    color = AccentColor
                                )
                            }
                            items(peopleResults) { result ->
                                PersonSearchResultItem(
                                    result = result,
                                    onClick = { /* Navigate to person */ }
                                )
                            }
                        }
                    }
                    
                    // No results
                    if (roomResults.isEmpty() && messageResults.isEmpty() && peopleResults.isEmpty()) {
                        item {
                            NoResultsItem(query = query)
                        }
                    }
                } else {
                    // Search suggestions
                    item {
                        SearchSuggestionsItem()
                    }
                }
            }
        }
    }
}

@Composable
private fun SearchBar(
    query: String,
    onQueryChange: (String) -> Unit,
    onSearch: () -> Unit,
    modifier: Modifier = Modifier
) {
    OutlinedTextField(
        value = query,
        onValueChange = onQueryChange,
        placeholder = { Text("Search rooms, messages, people...") },
        modifier = modifier,
        singleLine = true,
        leadingIcon = {
            Icon(
                imageVector = Icons.Default.Search,
                contentDescription = null
            )
        },
        trailingIcon = {
            if (query.isNotEmpty()) {
                IconButton(onClick = { onQueryChange("") }) {
                    Icon(
                        imageVector = Icons.Default.Clear,
                        contentDescription = "Clear"
                    )
                }
            }
        },
        shape = RoundedCornerShape(12.dp)
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SearchTypeTabs(
    selectedType: SearchType,
    onSelectType: (SearchType) -> Unit,
    modifier: Modifier = Modifier
) {
    Row(
        modifier = modifier
            .clip(RoundedCornerShape(8.dp))
            .background(MaterialTheme.colorScheme.surfaceVariant)
            .padding(4.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        SearchType.values().forEach { type ->
            FilterChip(
                selected = selectedType == type,
                onClick = { onSelectType(type) },
                label = { Text(type.name) },
                modifier = Modifier.weight(1f)
            )
        }
    }
}

@Composable
private fun RoomSearchResultItem(
    result: RoomSearchResult,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = onClick),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Room avatar
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primaryContainer),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = result.name.firstOrNull()?.toString() ?: "?",
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onPrimaryContainer
                )
            }
            
            // Room info
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = result.name,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium
                )
                Text(
                    text = result.topic,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                )
            }
            
            // Encryption indicator
            if (result.isEncrypted) {
                Text(text = "🔒", fontSize = 16.sp)
            }
        }
    }
}

@Composable
private fun MessageSearchResultItem(
    result: MessageSearchResult,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = onClick),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            // Header
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                // Sender avatar
                Box(
                    modifier = Modifier
                        .size(32.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = result.senderName.firstOrNull()?.toString() ?: "?",
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                }
                
                // Sender name and room
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = result.senderName,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium
                    )
                    Text(
                        text = result.roomName,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
                    )
                }
                
                // Timestamp
                Text(
                    text = result.timestamp,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
                )
            }
            
            // Message preview
            Text(
                text = result.preview,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.8f)
            )
        }
    }
}

@Composable
private fun PersonSearchResultItem(
    result: PersonSearchResult,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = onClick),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Row(
            modifier = Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Person avatar
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primaryContainer),
                contentAlignment = Alignment.Center
            ) {
                if (result.avatar != null) {
                    Text(text = "📷", fontSize = 24.sp)
                } else {
                    Text(
                        text = result.name.firstOrNull()?.toString() ?: "?",
                        fontSize = 20.sp,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                }
            }
            
            // Person info
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = result.name,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium
                )
                if (result.status != null) {
                    Text(
                        text = result.status.name,
                        style = MaterialTheme.typography.bodySmall,
                        color = AccentColor
                    )
                }
            }
            
            // Action button
            IconButton(onClick = { /* Start chat */ }) {
                Icon(
                    imageVector = Icons.Default.Chat,
                    contentDescription = "Start chat"
                )
            }
        }
    }
}

@Composable
private fun NoResultsItem(
    query: String,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        Icon(
            imageVector = Icons.Default.SearchOff,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f),
            modifier = Modifier.size(64.dp)
        )
        
        Text(
            text = "No results found for \"$query\"",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
            textAlign = TextAlign.Center
        )
        
        Text(
            text = "Try different keywords or search for rooms, messages, or people",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
            textAlign = TextAlign.Center
        )
    }
}

@Composable
private fun SearchSuggestionsItem(
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        Text(
            text = "Recent Searches",
            style = MaterialTheme.typography.labelLarge,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
        )
        
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            listOf(
                "General",
                "Alice",
                "Project meeting"
            ).forEach { query ->
                SuggestionChip(
                    onClick = { /* Search query */ },
                    label = { Text(query) }
                )
            }
        }
        
        Divider()
        
        Text(
            text = "Suggested",
            style = MaterialTheme.typography.labelLarge,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f)
        )
        
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            listOf(
                "Team Alpha",
                "Project Updates",
                "John Doe"
            ).forEach { query ->
                SuggestionChip(
                    onClick = { /* Search query */ },
                    label = { Text(query) }
                )
            }
        }
    }
}

/**
 * Search types
 */
enum class SearchType {
    ALL,
    ROOMS,
    MESSAGES,
    PEOPLE
}

/**
 * Room search result
 */
data class RoomSearchResult(
    val id: String,
    val name: String,
    val topic: String,
    val isEncrypted: Boolean
)

/**
 * Message search result
 */
data class MessageSearchResult(
    val id: String,
    val roomId: String,
    val roomName: String,
    val senderName: String,
    val preview: String,
    val timestamp: String
)

/**
 * Person search result
 */
data class PersonSearchResult(
    val id: String,
    val name: String,
    val avatar: String?,
    val status: UserStatus?
)

/**
 * User status
 */
enum class UserStatus {
    ONLINE,
    AWAY,
    OFFLINE
}

@Preview(showBackground = true)
@Composable
private fun SearchScreenPreview() {
    ArmorClawTheme {
        SearchScreen(
            onNavigateBack = {},
            onNavigateToRoom = {},
            onNavigateToMessage = { _, _ -> }
        )
    }
}
