# SearchBar Component

> Message and global search bar
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/SearchBar.kt`

## Overview

SearchBar provides a reusable search input component for searching messages within a chat or globally across the app.

## Functions

### SearchBar
```kotlin
@Composable
fun SearchBar(
    query: String,
    onQueryChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier,
    onSearch: (String) -> Unit = {},
    showClearButton: Boolean = true,
    enabled: Boolean = true
)
```

**Description:** Material 3 styled search bar with query input and clear button.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `query` | `String` | Current search text |
| `onQueryChange` | `(String) -> Unit` | Query change callback |
| `placeholder` | `String` | Placeholder text |
| `modifier` | `Modifier` | Optional styling |
| `onSearch` | `(String) -> Unit` | Search submit callback |
| `showClearButton` | `Boolean` | Show clear button |
| `enabled` | `Boolean` | Enable/disable input |

---

## Visual Layout

### Default State
```
┌────────────────────────────────────┐
│ 🔍 Search messages...              │
└────────────────────────────────────┘
```

### With Text
```
┌────────────────────────────────────┐
│ 🔍 meeting notes              ✕   │
└────────────────────────────────────┘
```

### Focused State
```
┌────────────────────────────────────┐
│ 🔍 |meeting notes             ✕   │  ← Cursor visible
└────────────────────────────────────┘
```

---

## Implementation

```kotlin
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SearchBar(
    query: String,
    onQueryChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier,
    onSearch: (String) -> Unit = {},
    showClearButton: Boolean = true,
    enabled: Boolean = true
) {
    val focusRequester = remember { FocusRequester() }
    var isFocused by remember { mutableStateOf(false) }

    Surface(
        modifier = modifier
            .fillMaxWidth()
            .height(56.dp),
        shape = MaterialTheme.shapes.extraLarge,
        color = if (isFocused)
            MaterialTheme.colorScheme.surfaceVariant
        else
            MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.7f),
        tonalElevation = if (isFocused) 2.dp else 0.dp
    ) {
        Row(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // Search icon
            Icon(
                imageVector = Icons.Default.Search,
                contentDescription = "Search",
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(24.dp)
            )

            // Input field
            BasicTextField(
                value = query,
                onValueChange = onQueryChange,
                modifier = Modifier
                    .weight(1f)
                    .focusRequester(focusRequester)
                    .onFocusChanged { isFocused = it.isFocused },
                enabled = enabled,
                singleLine = true,
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Text,
                    imeAction = ImeAction.Search
                ),
                keyboardActions = KeyboardActions(
                    onSearch = { onSearch(query) }
                ),
                textStyle = MaterialTheme.typography.bodyLarge.copy(
                    color = MaterialTheme.colorScheme.onSurface
                ),
                decorationBox = { innerTextField ->
                    if (query.isEmpty()) {
                        Text(
                            text = placeholder,
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                    innerTextField()
                }
            )

            // Clear button
            if (showClearButton && query.isNotEmpty()) {
                IconButton(
                    onClick = { onQueryChange("") },
                    modifier = Modifier.size(24.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = "Clear search",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(18.dp)
                    )
                }
            }
        }
    }
}
```

---

## Variants

### Elevated SearchBar
```kotlin
@Composable
fun ElevatedSearchBar(
    query: String,
    onQueryChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier
) {
    SearchBar(
        query = query,
        onQueryChange = onQueryChange,
        placeholder = placeholder,
        modifier = modifier.elevation(8.dp)
    )
}
```

### Compact SearchBar
```kotlin
@Composable
fun CompactSearchBar(
    query: String,
    onQueryChange: (String) -> Unit,
    placeholder: String,
    modifier: Modifier = Modifier
) {
    SearchBar(
        query = query,
        onQueryChange = onQueryChange,
        placeholder = placeholder,
        modifier = modifier.height(40.dp),
        showClearButton = false
    )
}
```

---

## Search Overlay

### ChatSearchOverlay
```kotlin
@Composable
fun ChatSearchOverlay(
    query: String,
    results: List<Message>,
    onQueryChange: (String) -> Unit,
    onResultClick: (Message) -> Unit,
    onClose: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
    ) {
        // Search bar
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            IconButton(onClick = onClose) {
                Icon(Icons.Default.ArrowBack, "Back")
            }

            SearchBar(
                query = query,
                onQueryChange = onQueryChange,
                placeholder = "Search in chat...",
                modifier = Modifier.weight(1f)
            )
        }

        // Results
        if (results.isEmpty() && query.isNotEmpty()) {
            EmptySearchResult(query = query)
        } else {
            LazyColumn {
                items(results, key = { it.id }) { message ->
                    SearchResultItem(
                        message = message,
                        query = query,
                        onClick = { onResultClick(message) }
                    )
                }
            }
        }
    }
}
```

---

## Highlighting

### Query Highlight
```kotlin
@Composable
fun HighlightedText(
    text: String,
    query: String,
    style: TextStyle = MaterialTheme.typography.bodyMedium
) {
    val annotatedString = buildAnnotatedString {
        var startIndex = 0
        var foundIndex = text.indexOf(query, ignoreCase = true)

        while (foundIndex >= 0) {
            append(text.substring(startIndex, foundIndex))
            withStyle(
                SpanStyle(
                    background = MaterialTheme.colorScheme.primary.copy(alpha = 0.3f),
                    fontWeight = FontWeight.Bold
                )
            ) {
                append(text.substring(foundIndex, foundIndex + query.length))
            }
            startIndex = foundIndex + query.length
            foundIndex = text.indexOf(query, startIndex, ignoreCase = true)
        }
        append(text.substring(startIndex))
    }

    Text(text = annotatedString, style = style)
}
```

---

## Accessibility

### Content Descriptions
- Search icon: "Search"
- Clear button: "Clear search"
- Input field: "Search input"

---

## Related Documentation

- [SearchScreen](../screens/SearchScreen.md) - Search screen
- [ChatScreen](../screens/ChatScreen.md) - Chat screen
