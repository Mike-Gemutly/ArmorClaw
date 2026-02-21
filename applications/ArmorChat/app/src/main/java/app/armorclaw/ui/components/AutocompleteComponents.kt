package app.armorclaw.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.vectorResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import app.armorclaw.data.repository.AutocompleteUser
import app.armorclaw.data.repository.UserNamespace

/**
 * Autocomplete Dropdown for user mentions
 *
 * Resolves: G-04 (Identity Consistency)
 *
 * Shows visual indicators for bridged users from different platforms.
 */
@Composable
fun AutocompleteDropdown(
    query: String,
    results: List<AutocompleteUser>,
    onUserSelected: (AutocompleteUser) -> Unit,
    modifier: Modifier = Modifier
) {
    if (results.isEmpty()) return

    Card(
        modifier = modifier
            .fillMaxWidth()
            .heightIn(max = 300.dp),
        elevation = CardDefaults.cardElevation(defaultElevation = 4.dp)
    ) {
        LazyColumn(
            modifier = Modifier.fillMaxWidth()
        ) {
            items(results, key = { it.userId }) { user ->
                AutocompleteUserItem(
                    user = user,
                    query = query,
                    onClick = { onUserSelected(user) }
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AutocompleteUserItem(
    user: AutocompleteUser,
    query: String,
    onClick: () -> Unit
) {
    ListItem(
        modifier = Modifier.fillMaxWidth(),
        headlineContent = {
            Row(verticalAlignment = Alignment.CenterVertically) {
                // Highlight matching text
                Text(
                    text = user.displayName,
                    style = MaterialTheme.typography.bodyLarge,
                    fontWeight = FontWeight.Medium
                )

                // Platform indicator for bridged users
                if (user.isBridged) {
                    Spacer(modifier = Modifier.width(8.dp))
                    PlatformBadge(namespace = user.namespace)
                }
            }
        },
        supportingContent = {
            Text(
                text = user.userId,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline
            )
        },
        leadingContent = {
            // Avatar with platform indicator
            Box {
                Avatar(
                    displayName = user.displayName,
                    avatarUrl = user.avatarUrl,
                    size = 40.dp
                )

                // Platform icon overlay
                if (user.isBridged) {
                    Surface(
                        modifier = Modifier
                            .align(Alignment.BottomEnd)
                            .size(16.dp),
                        shape = CircleShape,
                        color = user.namespace.color
                    ) {
                        Icon(
                            imageVector = user.namespace.icon,
                            contentDescription = user.namespace.displayName,
                            modifier = Modifier
                                .padding(2.dp)
                                .size(12.dp),
                            tint = Color.White
                        )
                    }
                }
            }
        },
        trailingContent = {
            if (user.isBridged) {
                Icon(
                    imageVector = Icons.Default.Person,
                    contentDescription = "Bridged user",
                    tint = MaterialTheme.colorScheme.outline,
                    modifier = Modifier.size(20.dp)
                )
            }
        }
    )
}

@Composable
private fun PlatformBadge(namespace: UserNamespace) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = namespace.color.copy(alpha = 0.1f)
    ) {
        Text(
            text = namespace.displayName,
            style = MaterialTheme.typography.labelSmall,
            color = namespace.color,
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
        )
    }
}

@Composable
private fun Avatar(
    displayName: String,
    avatarUrl: String?,
    size: androidx.compose.ui.unit.Dp
) {
    Surface(
        modifier = Modifier.size(size),
        shape = CircleShape,
        color = MaterialTheme.colorScheme.primaryContainer
    ) {
        if (avatarUrl != null) {
            // In production, load actual avatar image
            // AsyncImage(model = avatarUrl, ...)
            Text(
                text = displayName.take(1).uppercase(),
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                modifier = Modifier.wrapContentSize(Alignment.Center)
            )
        } else {
            Text(
                text = displayName.take(1).uppercase(),
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                modifier = Modifier.wrapContentSize(Alignment.Center)
            )
        }
    }
}

/**
 * Namespace color and icon extensions
 */
val UserNamespace.color: Color
    @Composable
    get() = when (this) {
        UserNamespace.MATRIX -> MaterialTheme.colorScheme.primary
        UserNamespace.SLACK -> Color(0xFF4A154B) // Slack purple
        UserNamespace.DISCORD -> Color(0xFF5865F2) // Discord blurple
        UserNamespace.TEAMS -> Color(0xFF6264A7) // Teams purple
        UserNamespace.WHATSAPP -> Color(0xFF25D366) // WhatsApp green
        UserNamespace.BRIDGE -> MaterialTheme.colorScheme.secondary
    }

val UserNamespace.icon: ImageVector
    get() = Icons.Default.Person // In production, use actual platform icons

/**
 * Chat input with autocomplete support
 */
@Composable
fun ChatInputWithAutocomplete(
    onSendMessage: (String) -> Unit,
    onQueryUsers: (String) -> Unit,
    autocompleteResults: List<AutocompleteUser>,
    modifier: Modifier = Modifier
) {
    var text by remember { mutableStateOf("") }
    var showAutocomplete by remember { mutableStateOf(false) }

    Column(modifier = modifier) {
        // Autocomplete dropdown
        if (showAutocomplete && autocompleteResults.isNotEmpty()) {
            AutocompleteDropdown(
                query = text.substringAfterLast("@"),
                results = autocompleteResults,
                onUserSelected = { user ->
                    // Replace @query with user mention
                    val lastAtIndex = text.lastIndexOf("@")
                    if (lastAtIndex >= 0) {
                        text = text.substring(0, lastAtIndex) + "[${user.displayName}](${user.userId}) "
                    }
                    showAutocomplete = false
                },
                modifier = Modifier.padding(horizontal = 8.dp)
            )
            Spacer(modifier = Modifier.height(4.dp))
        }

        // Input row
        OutlinedTextField(
            value = text,
            onValueChange = { newValue ->
                text = newValue

                // Check for @ mentions
                val lastAtIndex = newValue.lastIndexOf("@")
                if (lastAtIndex >= 0 && lastAtIndex > newValue.lastIndexOf(" ")) {
                    val query = newValue.substring(lastAtIndex + 1)
                    if (query.isNotEmpty()) {
                        onQueryUsers(query)
                        showAutocomplete = true
                    } else {
                        showAutocomplete = false
                    }
                } else {
                    showAutocomplete = false
                }
            },
            modifier = Modifier.fillMaxWidth(),
            placeholder = { Text("Message...") },
            trailingIcon = {
                IconButton(
                    onClick = {
                        if (text.isNotBlank()) {
                            onSendMessage(text)
                            text = ""
                            showAutocomplete = false
                        }
                    }
                ) {
                    Icon(Icons.Default.Person, contentDescription = "Send")
                }
            }
        )
    }
}
