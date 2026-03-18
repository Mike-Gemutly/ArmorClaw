# HomeScreen

> Main conversation list screen
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/home/`

## Overview

HomeScreen displays the user's list of conversations (rooms) and provides navigation to create new rooms or access existing conversations.

## Functions

### HomeScreen
```kotlin
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onRoomClick: (String) -> Unit
)
```

**Description:** Main home screen displaying conversation list with FAB for creating new rooms.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `onRoomClick` | `(String) -> Unit` | Callback when room is tapped |

---

### HomeScreenFull
```kotlin
@Composable
fun HomeScreenFull(
    onRoomClick: (String) -> Unit,
    onSettingsClick: () -> Unit,
    onProfileClick: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Extended home screen with settings and profile navigation.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `onRoomClick` | `(String) -> Unit` | Room tap callback |
| `onSettingsClick` | `() -> Unit` | Settings tap callback |
| `onProfileClick` | `() -> Unit` | Profile tap callback |
| `modifier` | `Modifier` | Optional styling |

---

## Screen Layout

```
┌────────────────────────────────────┐
│  Conversations              🔍 👤  │  ← Top bar
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │  ← Sync status bar
├────────────────────────────────────┤
│  FAVORITES                         │
│  ├─ ⭐ Team Updates      (2)       │
│  └─ ⭐ Important          (1)      │
├────────────────────────────────────┤
│  RECENT                            │
│  ├─ 👤 Alice                        │
│  │    Sounds good! Let me...       │
│  │    2:30 PM              (3)     │
│  ├─ 👤 Bob                          │
│  │    Did you see the report?      │
│  │    Yesterday                    │
│  ├─ # general                      │
│  │    Carol: Meeting at 3pm        │
│  │    10:00 AM             (12)    │
│  └─ # random                       │
│       Check out this link...       │
│       Monday                       │
│                                    │
│              ──                    │
│                                    │
│       No conversations yet         │
│   Start chatting with your AI      │
│                                    │
│                           ┌─────┐  │
│                           │  +  │  │  ← FAB
│                           └─────┘  │
└────────────────────────────────────┘
```

---

## Components

### TopAppBar
```kotlin
TopAppBar(
    title = {
        Text("Conversations", style = MaterialTheme.typography.titleLarge)
    }
)
```

### SyncStatusBar
```kotlin
SyncStatusBar(
    syncState = SyncState.Success(messagesSent = 0, messagesReceived = 0),
    lastSyncTime = System.currentTimeMillis() - 60000,
    onRefreshClick = { /* Trigger sync */ }
)
```

### FloatingActionButton
```kotlin
FloatingActionButton(
    onClick = { /* Create room */ }
) {
    Icon(Icons.Default.Add, contentDescription = "Create Room")
}
```

### EmptyState
```kotlin
Column(horizontalAlignment = Alignment.CenterHorizontally) {
    Icon(
        imageVector = Icons.Default.ChatBubbleOutline,
        modifier = Modifier.size(80.dp),
        tint = OnBackground.copy(alpha = 0.3f)
    )
    Spacer(modifier = Modifier.height(16.dp))
    Text("No conversations yet", style = MaterialTheme.typography.titleLarge)
    Text(
        "Start chatting with your AI agent",
        style = MaterialTheme.typography.bodyMedium,
        color = OnBackground.copy(alpha = 0.7f)
    )
}
```

---

## Room List Item

### RoomItem Layout
```
┌────────────────────────────────────┐
│  ┌────┐                            │
│  │ 👤 │  Alice                     │
│  │    │  Sounds good! Let me...    │
│  └────┘  2:30 PM          (3)     │
└────────────────────────────────────┘
```

### RoomItem Components
| Component | Description |
|-----------|-------------|
| Avatar | Room/user avatar image |
| Name | Room or user display name |
| Preview | Last message preview |
| Timestamp | Last activity time |
| Badge | Unread count badge |
| Mute Icon | Muted room indicator |

---

## Sync Status Integration

### SyncState Types
```kotlin
sealed class SyncState {
    object Idle : SyncState()
    object Syncing : SyncState()
    data class Success(val messagesSent: Int, val messagesReceived: Int) : SyncState()
    data class Error(val message: String) : SyncState()
}
```

### Status Bar Display
| State | Display |
|-------|---------|
| Idle | "Last sync: X ago" |
| Syncing | Spinner + "Syncing..." |
| Success | "Synced" |
| Error | "Sync failed - Tap to retry" |

---

## State Management

### HomeScreenState
```kotlin
data class HomeScreenState(
    val rooms: List<Room>,
    val favorites: List<Room>,
    val isLoading: Boolean,
    val isRefreshing: Boolean,
    val error: String?,
    val syncState: SyncState,
    val lastSyncTime: Long
)
```

### HomeScreenActions
| Action | Description |
|--------|-------------|
| `loadRooms()` | Fetch room list |
| `refreshRooms()` | Pull-to-refresh |
| `toggleFavorite(roomId)` | Add/remove favorite |
| `markAsRead(roomId)` | Clear unread count |
| `createRoom()` | Open room creation |

---

## User Interactions

### Room Actions
| Action | Trigger | Result |
|--------|---------|--------|
| Open room | Tap room item | Navigate to ChatScreen |
| Create room | Tap FAB | Open RoomManagementScreen |
| Settings | Tap settings icon | Navigate to SettingsScreen |
| Profile | Tap profile icon | Navigate to ProfileScreen |
| Search | Tap search icon | Open search overlay |
| Refresh | Pull down | Refresh room list |

### Long Press Actions
| Action | Description |
|--------|-------------|
| Favorite | Toggle favorite status |
| Mute | Mute/unmute room |
| Mark read | Mark all as read |
| Leave | Leave room |

---

## Filtering & Sorting

### Room Categories
| Category | Description |
|----------|-------------|
| Favorites | Starred rooms |
| Direct | 1-on-1 conversations |
| Groups | Group chats |
| Channels | Public channels |
| Archived | Archived rooms |

### Sort Options
| Sort | Order |
|------|-------|
| Activity | Most recent first |
| Alphabetical | A-Z |
| Unread | Unread count |

---

## Accessibility

### Content Descriptions
- Room item: "[Name], [preview], [time], [unread count] unread"
- FAB: "Create new conversation"
- Sync status: Current sync state

---

## Related Documentation

- [Home Screen Feature](../features/home-screen.md) - Feature overview
- [ChatScreen](ChatScreen.md) - Conversation screen
- [Room Management](../features/room-management.md) - Create/Join rooms
- [Offline Sync](../features/offline-sync.md) - Background sync
