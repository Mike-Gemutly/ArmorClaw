# SearchScreen

> Global search screen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/search/`

## Overview

SearchScreen provides global search functionality across messages, rooms, and users.

## Functions

### SearchScreen
```kotlin
@Composable
fun SearchScreen(
    onNavigateBack: () -> Unit,
    onResultClick: (String) -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Main search screen with query input and results display.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `onNavigateBack` | `() -> Unit` | Back navigation |
| `onResultClick` | `(String) -> Unit` | Result tap callback |
| `modifier` | `Modifier` | Optional styling |

---

## Screen Layout

```
┌────────────────────────────────────┐
│  ← Search                    ✕    │
│  ┌──────────────────────────────┐  │
│  │ 🔍 Search messages, rooms... │  │
│  └──────────────────────────────┘  │
├────────────────────────────────────┤
│  Recent Searches                   │
│  ├─ project updates         ✕     │
│  ├─ meeting notes           ✕     │
│  └─ alice                   ✕     │
├────────────────────────────────────┤
│  FILTERS                           │
│  [Messages] [Rooms] [Users] [All] │
├────────────────────────────────────┤
│  RESULTS (5)                       │
│  ┌──────────────────────────────┐  │
│  │ 💬 #general                  │  │
│  │    ...project updates for    │  │
│  │    the Q1 release...         │  │
│  │    Yesterday                 │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ 👤 Alice                     │  │
│  │    ...project updates are    │  │
│  │    ready for review...       │  │
│  │    2 hours ago               │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ 🏠 Project Team              │  │
│  │    Room                      │  │
│  └──────────────────────────────┘  │
└────────────────────────────────────┘
```

---

## Components

### SearchBar
```kotlin
@Composable
fun SearchBar(
    query: String,
    onQueryChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier
) {
    OutlinedTextField(
        value = query,
        onValueChange = onQueryChange,
        placeholder = { Text(placeholder) },
        leadingIcon = { Icon(Icons.Default.Search, null) },
        trailingIcon = {
            if (query.isNotEmpty()) {
                IconButton(onClick = { onQueryChange("") }) {
                    Icon(Icons.Default.Clear, "Clear")
                }
            }
        },
        singleLine = true
    )
}
```

### FilterChips
```kotlin
Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
    FilterChip(
        selected = selectedFilter == Filter.ALL,
        onClick = { selectedFilter = Filter.ALL },
        label = { Text("All") }
    )
    FilterChip(
        selected = selectedFilter == Filter.MESSAGES,
        onClick = { selectedFilter = Filter.MESSAGES },
        label = { Text("Messages") }
    )
    FilterChip(
        selected = selectedFilter == Filter.ROOMS,
        onClick = { selectedFilter = Filter.ROOMS },
        label = { Text("Rooms") }
    )
    FilterChip(
        selected = selectedFilter == Filter.USERS,
        onClick = { selectedFilter = Filter.USERS },
        label = { Text("Users") }
    )
}
```

### SearchResultItem
```kotlin
@Composable
fun SearchResultItem(
    result: SearchResult,
    onClick: () -> Unit
) {
    ListItem(
        headlineContent = { Text(result.title) },
        supportingContent = { Text(result.preview) },
        trailingContent = { Text(result.timestamp) },
        leadingContent = {
            when (result.type) {
                ResultType.MESSAGE -> Icon(Icons.Default.Chat, null)
                ResultType.ROOM -> Icon(Icons.Default.Group, null)
                ResultType.USER -> Avatar(result.avatar)
            }
        },
        onClick = onClick
    )
}
```

---

## Search Filters

### Filter Types
```kotlin
enum class SearchFilter {
    ALL,
    MESSAGES,
    ROOMS,
    USERS
}
```

### Filter Descriptions
| Filter | Scope |
|--------|-------|
| All | Search everything |
| Messages | Message content only |
| Rooms | Room names and descriptions |
| Users | User names and profiles |

---

## Data Models

### SearchResult
```kotlin
data class SearchResult(
    val id: String,
    val type: ResultType,
    val title: String,
    val preview: String,
    val roomId: String?,
    val messageId: String?,
    val timestamp: String,
    val avatar: String?
)

enum class ResultType {
    MESSAGE,
    ROOM,
    USER
}
```

### RecentSearch
```kotlin
data class RecentSearch(
    val query: String,
    val timestamp: Instant
)
```

---

## State Management

### SearchState
```kotlin
data class SearchState(
    val query: String,
    val results: List<SearchResult>,
    val recentSearches: List<RecentSearch>,
    val selectedFilter: SearchFilter,
    val isSearching: Boolean,
    val error: String?
)
```

### SearchActions
| Action | Description |
|--------|-------------|
| `search(query)` | Execute search |
| `setFilter(filter)` | Change filter |
| `clearRecentSearches()` | Clear history |
| `removeRecentSearch(query)` | Remove single item |

---

## Search Behavior

### Debouncing
```kotlin
LaunchedEffect(query) {
    delay(300) // 300ms debounce
    if (query.length >= 2) {
        viewModel.search(query)
    }
}
```

### Minimum Query Length
- Minimum: 2 characters
- Below minimum: Show recent searches
- Above minimum: Execute search

---

## Highlighting

### Query Highlighting
```kotlin
@Composable
fun HighlightedText(
    text: String,
    query: String
) {
    val annotatedString = buildAnnotatedString {
        val startIndex = text.indexOf(query, ignoreCase = true)
        if (startIndex >= 0) {
            append(text.substring(0, startIndex))
            withStyle(SpanStyle(background = Yellow.copy(0.3f))) {
                append(text.substring(startIndex, startIndex + query.length))
            }
            append(text.substring(startIndex + query.length))
        } else {
            append(text)
        }
    }
    Text(annotatedString)
}
```

---

## Recent Searches

### Storage
- Persisted in DataStore
- Maximum 10 recent searches
- Sorted by timestamp (newest first)

### Clear Options
- Clear individual search
- Clear all recent searches

---

## Accessibility

### Content Descriptions
- Search field: "Search messages, rooms, and users"
- Filter chips: "Filter by [type]"
- Result item: "[Type]: [title], [preview]"

---

## Related Documentation

- [ChatScreen](ChatScreen.md) - Message search in chat
- [Home Screen](../features/home-screen.md) - Home navigation
