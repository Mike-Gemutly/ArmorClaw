# Room Management Screen

> **Route:** `room_management`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/room/RoomManagementScreen.kt`
> **Category:** Rooms

## Screenshot

![Room Management Screen](../../screenshots/rooms/room-management.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←  Room Management                  │  ← TopAppBar
├─────────────────────────────────────┤
│                                     │
│  ┌─────────────────────────────┐   │
│  │ + Create New Room           │   │  ← Create room button
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 🔗 Join Room by Link/ID     │   │  ← Join room button
│  └─────────────────────────────┘   │
│                                     │
│  YOUR ROOMS                         │  ← Section header
│  ┌─────────────────────────────┐   │
│  │ 👤 General              🔒  │   │
│  │    5 members                 │   │
│  │    Created 2 weeks ago       │   │
│  └─────────────────────────────┘   │
│                                     │
│  SHARED WITH YOU                    │
│  ┌─────────────────────────────┐   │
│  │ 👤 Team Alpha           🔒  │   │
│  │    12 members                │   │
│  │    Invited by John           │   │
│  └─────────────────────────────┘   │
│                                     │
│  ARCHIVED                           │
│  ┌─────────────────────────────┐   │
│  │ 👤 Old Project          🔒  │   │
│  │    3 members                 │   │
│  │    Archived 1 month ago      │   │
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Loading

```
┌─────────────────────────────────────┐
│                                     │
│           ◠ ◠ ◠                     │
│        Loading rooms...             │
│                                     │
└─────────────────────────────────────┘
```

### Empty

```
┌─────────────────────────────────────┐
│                                     │
│              💬                     │
│        No rooms yet                 │
│                                     │
│  Create a new room or join one      │
│  using an invite link.              │
│                                     │
│  [Create Room] [Join Room]          │
└─────────────────────────────────────┘
```

### Loaded

```
┌─────────────────────────────────────┐
│  [Create] [Join]                    │
│                                     │
│  YOUR ROOMS                         │
│  [Room cards...]                    │
│                                     │
│  SHARED WITH YOU                    │
│  [Room cards...]                    │
│                                     │
│  ARCHIVED                           │
│  [Room cards...]                    │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────────┐
            │   Loading   │
            └──────┬──────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐
   │  Empty  │ │ Loaded  │ │  Error  │
   └─────────┘ └────┬────┘ └─────────┘
                   │
    ┌──────────────┼──────────────┐
    ▼              ▼              ▼
┌─────────┐  ┌──────────┐  ┌──────────┐
│ Create  │  │ Join     │  │ Open     │
│ Room    │  │ Room     │  │ Room     │
└────┬────┘  └────┬─────┘  └────┬─────┘
     │            │             │
     ▼            ▼             ▼
┌─────────┐  ┌──────────┐  ┌──────────┐
│ Create  │  │ Enter    │  │ Chat     │
│ Dialog  │  │ Link/ID  │  │ Screen   │
└─────────┘  └──────────┘  └──────────┘
```

## User Flow

1. **User arrives from:** Home screen or settings
2. **User can:**
   - Create new room
   - Join room by link/ID
   - View all rooms
   - Open room details
3. **User navigates to:**
   - Chat screen
   - Room details screen
   - Create room dialog
   - Join room dialog

## Notes

- Central room management hub
- Clear room ownership indication
- Encryption status visible
- Quick actions for create/join
