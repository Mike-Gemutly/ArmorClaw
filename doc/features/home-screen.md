# Home Screen Feature

> Main navigation hub for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/home/`

## Overview

The home screen is the central hub of ArmorClaw, displaying all chat rooms, providing navigation to other features, and managing the overall app experience.

## Feature Components

### HomeScreenFull
**Location:** `home/HomeScreenFull.kt`

Main home screen with room list and navigation.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `HomeScreenFull()` | Main home screen | `onNavigateToChat`, `onNavigateToSettings`, `onNavigateToProfile`, `onCreateRoom`, `onJoinRoom` |
| `RoomList()` | List of chat rooms | `rooms`, `onRoomClick` |
| `FavoritesSection()` | Favorite rooms | `rooms`, `isExpanded` |
| `ActiveRoomsSection()` | Active rooms | `rooms` |
| `ArchivedSection()` | Archived rooms | `rooms`, `isExpanded` |
| `EmptyState()` | No rooms placeholder | `onCreateRoom` |

#### Screen Layout
```
┌────────────────────────────────────┐
│  ArmorClaw        [🔍] [👤] [⚙️]  │
├────────────────────────────────────┤
│  ⭐ Favorites (expandable)          │
│  ├─ Room 1                          │
│  └─ Room 2                          │
├────────────────────────────────────┤
│  💬 Active Chats                    │
│  ├─ Room 3                          │
│  ├─ Room 4                          │
│  └─ Room 5                          │
├────────────────────────────────────┤
│  📦 Archived (collapsed)            │
└────────────────────────────────────┘
         [FAB: Create Room]
```

---

### Room List Item
**Location:** Integrated in `HomeScreenFull.kt`

Individual room display in the list.

#### RoomItem Features

| Element | Description |
|---------|-------------|
| Avatar | Room image or initial letter |
| Room Name | Bold text, truncated |
| Last Message | Preview, gray text |
| Timestamp | Relative time (2m, 1h, 1d) |
| Unread Badge | Count of unread messages |
| Mention Badge | Count of mentions |
| Encryption Icon | Lock for encrypted rooms |

#### Room Item States

| State | Visual |
|-------|--------|
| Normal | Standard display |
| Unread | Bold room name, badge |
| Mention | Red badge with count |
| Archived | Faded appearance |

---

### Section Management

#### ExpandableSection
```kotlin
@Composable
fun ExpandableSection(
    title: String,
    icon: ImageVector,
    isExpanded: Boolean,
    onToggle: () -> Unit,
    content: @Composable () -> Unit
)
```

#### Section States
- **Expanded** - Full content visible
- **Collapsed** - Only header visible
- **Loading** - Skeleton loading state

---

### Pull-to-Refresh
```kotlin
@Composable
fun RoomListWithRefresh(
    rooms: List<RoomItem>,
    isRefreshing: Boolean,
    onRefresh: () -> Unit,
    onRoomClick: (String) -> Unit
)
```

---

## Room Model

### RoomItem
```kotlin
data class RoomItem(
    val id: String,
    val name: String,
    val avatar: String?,
    val isEncrypted: Boolean,
    val lastMessage: String?,
    val lastMessageTime: Instant?,
    val unreadCount: Int,
    val mentionCount: Int,
    val isFavorite: Boolean,
    val isArchived: Boolean
)
```

### RoomType
| Type | Description |
|------|-------------|
| DIRECT | 1-on-1 chat |
| GROUP | Group chat |
| CHANNEL | Broadcast channel |

---

## Navigation Actions

### Top Bar Actions
| Icon | Action | Destination |
|------|--------|-------------|
| Search | Global search | SearchScreen |
| Profile | View profile | ProfileScreen |
| Settings | App settings | SettingsScreen |

### FAB Actions
| Action | Destination |
|--------|-------------|
| Create Room | RoomManagementScreen (Create tab) |
| Join Room | RoomManagementScreen (Join tab) |

---

## State Management

### HomeState
```kotlin
data class HomeState(
    val isLoading: Boolean,
    val activeRooms: List<RoomItem>,
    val favoriteRooms: List<RoomItem>,
    val archivedRooms: List<RoomItem>,
    val isRefreshing: Boolean,
    val error: String?
)
```

### HomeActions
| Action | Description |
|--------|-------------|
| `loadRooms()` | Initial room load |
| `refreshRooms()` | Pull-to-refresh |
| `toggleFavorite(roomId)` | Toggle favorite status |
| `toggleArchive(roomId)` | Toggle archive status |
| `markAsRead(roomId)` | Clear unread count |

---

## Search Integration

### Global Search
**Location:** `search/SearchScreen.kt`

Accessible from home screen search icon.

#### Search Types
- Rooms
- Messages
- People

#### Search Flow
1. Tap search icon
2. Enter query
3. Select search type
4. View results
5. Tap result to navigate

---

## Related Documentation

- [Chat](chat.md) - Chat screen functionality
- [Room Management](room-management.md) - Create/Join rooms
- [Profile](profile.md) - User profile
- [Settings](settings.md) - App settings
