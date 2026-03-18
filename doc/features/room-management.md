# Room Management Feature

> Create and join chat rooms
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/room/`

## Overview

The room management feature allows users to create new chat rooms, join existing rooms via invite codes or links, and manage room discovery.

## Feature Components

### RoomManagementScreen
**Location:** `room/RoomManagementScreen.kt`

Main screen for creating and joining rooms.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `RoomManagementScreen()` | Main management screen | `onNavigateBack`, `onRoomCreated`, `onRoomJoined` |
| `CreateRoomTab()` | Create new room form | `onCreate` |
| `JoinRoomTab()` | Join existing room form | `onJoin` |
| `RoomTypeSelector()` | Select room type | `selectedType`, `onSelect` |
| `InviteCodeInput()` | Enter invite code | `onSubmit` |

#### Screen Layout
```
┌────────────────────────────────────┐
│  ← Rooms                      ──  │
├────────────────────────────────────┤
│  [ Create ]  [ Join ]              │  ← Tabs
├────────────────────────────────────┤
│                                    │
│  ┌──────────────────────────────┐  │
│  │ Room Name                    │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ Enter room name...       │ │  │
│  │ └──────────────────────────┘ │  │
│  │                              │  │
│  │ Room Type                    │  │
│  │ ○ Direct Message             │  │
│  │ ○ Group Chat                 │  │
│  │ ○ Public Channel             │  │
│  │                              │  │
│  │ Description (optional)       │  │
│  │ ┌──────────────────────────┐ │  │
│  │ │ Add description...       │ │  │
│  │ └──────────────────────────┘ │  │
│  └──────────────────────────────┘  │
│                                    │
│  ┌──────────────────────────────┐  │
│  │       Create Room            │  │  ← Action button
│  └──────────────────────────────┘  │
└────────────────────────────────────┘
```

---

### RoomDetailsScreen
**Location:** `room/RoomDetailsScreen.kt`

Displays room information and member list.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `RoomDetailsScreen()` | Room details view | `roomId`, `onNavigateBack`, `onEditSettings` |
| `RoomInfoCard()` | Room name, description, avatar | `room` |
| `MemberList()` | List of room members | `members`, `onMemberClick` |
| `InviteSection()` | Invite link/code display | `inviteCode`, `onShare` |
| `LeaveRoomButton()` | Leave room action | `onLeave` |

#### Details Layout
```
┌────────────────────────────────────┐
│  ← Room Details              ⚙️   │
├────────────────────────────────────┤
│                                    │
│          ┌─────────┐               │
│          │   🏠    │               │
│          │ Avatar  │               │
│          └─────────┘               │
│          Project Team              │
│          12 members                │
│                                    │
├────────────────────────────────────┤
│  Description                       │
│  Team collaboration and updates    │
├────────────────────────────────────┤
│  Invite Link                       │
│  armorclaw://join/abc123    [Copy] │
├────────────────────────────────────┤
│  MEMBERS (12)                      │
│  ├─ 👤 Alice (Admin)              │
│  ├─ 👤 Bob                         │
│  ├─ 👤 Carol                       │
│  └─ ... View all                   │
├────────────────────────────────────┤
│  [     Leave Room     ]            │
└────────────────────────────────────┘
```

---

### RoomSettingsScreen
**Location:** `room/RoomSettingsScreen.kt`

Room configuration and management settings.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `RoomSettingsScreen()` | Room settings screen | `roomId`, `onNavigateBack` |
| `RoomNameSetting()` | Edit room name | `currentName`, `onSave` |
| `RoomAvatarSetting()` | Change room avatar | `currentAvatar`, `onChange` |
| `NotificationSetting()` | Room notification preferences | `settings`, `onChange` |
| `MemberManagement()` | Manage members (admin) | `members`, `onAction` |
| `DangerZone()` | Delete/leave room | `onDelete`, `onLeave` |

---

## Room Types

### RoomType Enum
```kotlin
enum class RoomType {
    DIRECT,      // 1-on-1 conversation
    GROUP,       // Private group chat
    CHANNEL      // Public channel
}
```

### Room Type Features

| Feature | Direct | Group | Channel |
|---------|--------|-------|---------|
| Max members | 2 | 256 | Unlimited |
| Invite required | No | Yes | No |
| Discoverable | No | No | Yes |
| Message history | Full | Full | configurable |
| Encryption | E2E | E2E | Optional |

---

## Data Models

### Room
```kotlin
data class Room(
    val id: String,
    val name: String,
    val description: String?,
    val avatar: String?,
    val type: RoomType,
    val createdAt: Instant,
    val updatedAt: Instant,
    val createdBy: String,
    val memberCount: Int,
    val unreadCount: Int,
    val lastMessage: Message?,
    val isEncrypted: Boolean,
    val inviteCode: String?
)
```

### RoomMember
```kotlin
data class RoomMember(
    val userId: String,
    val roomId: String,
    val role: MemberRole,
    val joinedAt: Instant,
    val lastReadAt: Instant?,
    val mutedUntil: Instant?
)

enum class MemberRole {
    MEMBER,
    MODERATOR,
    ADMIN
}
```

---

## State Management

### RoomManagementState
```kotlin
data class RoomManagementState(
    val selectedTab: Tab,
    val isLoading: Boolean,
    val error: String?,
    val roomName: String,
    val roomDescription: String,
    val roomType: RoomType,
    val inviteCode: String
)
```

### RoomManagementActions
| Action | Description |
|--------|-------------|
| `createRoom()` | Create new room |
| `joinRoom(code)` | Join via invite code |
| `validateCode(code)` | Check if code is valid |
| `resetForm()` | Clear form fields |

---

## User Interactions

### Create Room Flow
1. Tap FAB on home screen
2. Enter room name
3. Select room type
4. Add description (optional)
5. Tap Create
6. Navigate to new room

### Join Room Flow
1. Tap Join tab
2. Enter/paste invite code
3. Preview room info
4. Tap Join
5. Navigate to room

### Invite Flow
1. Open room details
2. Copy invite link
3. Share with others

---

## Validation

### Room Name Rules
- Required field
- Min 1 character, max 100 characters
- No special characters except spaces, hyphens, underscores

### Invite Code Format
- 6-12 alphanumeric characters
- Case-insensitive
- Auto-generated on room creation

---

## Related Documentation

- [Home Screen](home-screen.md) - Room list
- [Room Settings](room-settings.md) - Room configuration
- [Chat Feature](chat.md) - Messaging in rooms
