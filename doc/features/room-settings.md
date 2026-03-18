# Room Settings Feature

> Room configuration and management
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/room/`

## Overview

Room settings allow administrators and moderators to configure room behavior, manage members, and control room security options.

## Feature Components

### RoomSettingsScreen
**Location:** `room/RoomSettingsScreen.kt`

Main settings screen for room configuration.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `RoomSettingsScreen()` | Main settings screen | `roomId`, `onNavigateBack` |
| `RoomInfoSection()` | Room name and avatar | `room`, `onEdit` |
| `NotificationSettings()` | Notification preferences | `settings`, `onChange` |
| `MemberManagementSection()` | Member list and roles | `members`, `onManage` |
| `SecuritySettings()` | Encryption and access | `room`, `onChange` |
| `DangerZoneSection()` | Delete/leave room | `isOwner`, `onDelete`, `onLeave` |

#### Settings Layout
```
┌────────────────────────────────────┐
│  ← Room Settings                  │
├────────────────────────────────────┤
│  ROOM INFO                         │
│  ├─ Room Name              >       │
│  ├─ Description            >       │
│  └─ Avatar                 >       │
├────────────────────────────────────┤
│  NOTIFICATIONS                     │
│  ├─ Message Notifications  [ON]    │
│  ├─ Mention Only           [OFF]   │
│  └─ Mute Room              [OFF]   │
├────────────────────────────────────┤
│  SECURITY                          │
│  ├─ Encryption Status      ✅      │
│  ├─ Message Retention      30 days │
│  └─ Join Approval          [ON]    │
├────────────────────────────────────┤
│  MEMBERS (12)                      │
│  ├─ Manage Members         >       │
│  └─ Invite Link            >       │
├────────────────────────────────────┤
│  DANGER ZONE                       │
│  ├─ Leave Room             >       │
│  └─ Delete Room (Owner)    >       │
└────────────────────────────────────┘
```

---

## Settings Categories

### Room Information
| Setting | Description | Editable By |
|---------|-------------|-------------|
| Room Name | Display name | Admin, Moderator |
| Description | Room description | Admin, Moderator |
| Avatar | Room image | Admin, Moderator |

### Notification Settings
| Setting | Options | Default |
|---------|---------|---------|
| Message Notifications | All, Mentions, None | All |
| Sound | Custom sounds | Default |
| Vibration | On/Off | On |
| Mute Duration | 1h, 8h, 24h, Forever | - |

### Security Settings
| Setting | Description | Default |
|---------|-------------|---------|
| Encryption | E2E encryption status | Enabled |
| Message Retention | Days to keep messages | Forever |
| Join Approval | Require admin approval | Disabled |
| Invite Link | Enable/disable public link | Enabled |

---

## Member Management

### Member Roles
| Role | Permissions |
|------|-------------|
| Member | Send messages, react, leave |
| Moderator | All member + remove users, edit messages |
| Admin | All moderator + manage settings, delete room |

### Member Actions
| Action | Admin | Moderator | Member |
|--------|-------|-----------|--------|
| View members | ✓ | ✓ | ✓ |
| Invite members | ✓ | ✓ | ✗ |
| Remove members | ✓ | ✓ | ✗ |
| Change roles | ✓ | ✗ | ✗ |
| Edit settings | ✓ | ✗ | ✗ |
| Delete room | ✓ | ✗ | ✗ |

---

## Data Models

### RoomSettings
```kotlin
data class RoomSettings(
    val roomId: String,
    val notificationLevel: NotificationLevel,
    val muteExpiresAt: Instant?,
    val messageRetentionDays: Int?,
    val requireApproval: Boolean,
    val inviteLinkEnabled: Boolean,
    val customInviteCode: String?
)

enum class NotificationLevel {
    ALL,
    MENTIONS_ONLY,
    NONE
}
```

### RoomMemberWithRole
```kotlin
data class RoomMemberWithRole(
    val userId: String,
    val displayName: String,
    val avatar: String?,
    val role: MemberRole,
    val joinedAt: Instant,
    val lastActiveAt: Instant?
)
```

---

## State Management

### RoomSettingsState
```kotlin
data class RoomSettingsState(
    val room: Room?,
    val settings: RoomSettings?,
    val members: List<RoomMemberWithRole>,
    val isLoading: Boolean,
    val isSaving: Boolean,
    val error: String?,
    val currentUserRole: MemberRole
)
```

### RoomSettingsActions
| Action | Description |
|--------|-------------|
| `loadSettings(roomId)` | Load room settings |
| `updateName(name)` | Update room name |
| `updateDescription(desc)` | Update description |
| `updateAvatar(uri)` | Update room avatar |
| `setNotificationLevel(level)` | Change notification preference |
| `toggleMute(duration)` | Mute/unmute room |
| `setJoinApproval(enabled)` | Toggle approval requirement |
| `regenerateInviteCode()` | Generate new invite code |
| `removeMember(userId)` | Remove member from room |
| `changeRole(userId, role)` | Change member role |
| `leaveRoom()` | Leave room |
| `deleteRoom()` | Delete room (owner only) |

---

## User Interactions

### Settings Flow
1. Open room
2. Tap menu → Settings
3. Navigate settings categories
4. Make changes
5. Auto-save on change

### Member Management Flow
1. Tap Manage Members
2. View all members with roles
3. Tap member for actions
4. Promote/Demote/Remove

### Danger Zone Actions
1. Confirm before destructive action
2. Leave: Simple confirmation
3. Delete: Type room name to confirm

---

## Validation

### Room Name
- Required, 1-100 characters
- Cannot be empty or whitespace only

### Description
- Optional, max 500 characters

### Message Retention
- Options: 1, 7, 30, 90 days, or Forever
- Only configurable by admin

---

## Related Documentation

- [Room Management](room-management.md) - Create/Join rooms
- [Home Screen](home-screen.md) - Room list
- [Encryption](encryption.md) - Security features
