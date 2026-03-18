# Home Screen

> **Route:** `home`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/home/HomeScreenFull.kt`
> **Category:** Main

## Screenshot

![Home Screen](../../screenshots/main/home.png)

## Layout

```
┌─────────────────────────────────────┐
│ ArmorClaw  (17)    🔍  👤  ⚙️       │  ← TopAppBar with unread badge
├─────────────────────────────────────┤
│  ┌─────────────────────────────┐   │
│  │  +   Join a room            │   │  ← Join room button
│  └─────────────────────────────┘   │
│                                     │
│  ▼ Favorites (1)                    │  ← Collapsible section
│  ┌─────────────────────────────┐   │
│  │ 👤 Family                   │   │
│  │    Don't forget dinner!  30m│   │  ← Room card
│  │                         (2) │   │
│  └─────────────────────────────┘   │
│                                     │
│  ▼ Chats (3)                        │  ← Active section
│  ┌─────────────────────────────┐   │
│  │ G General              🔒   │   │
│  │    Hey everyone!       2m   │   │
│  │                         (5) │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ T Team Alpha           🔒   │   │
│  │    Meeting in 10 min   15m  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ R Random               🔒   │   │
│  │    Check out this meme 1h   │   │
│  │                        (12) │   │
│  └─────────────────────────────┘   │
│                                     │
│  ▶ Archived (1)                     │  ← Collapsed section
│                                     │
│                                     │
│                            ┌───┐   │  ← FAB
│                            │ + │   │
│                            └───┘   │
└─────────────────────────────────────┘
```

## UI States

### Loading

```
┌─────────────────────────────────────┐
│ ArmorClaw         🔍  👤  ⚙️        │
├─────────────────────────────────────┤
│                                     │
│                                     │
│           ◠ ◠ ◠ ◠ ◠                 │
│           Loading rooms...          │
│                                     │
│                                     │
└─────────────────────────────────────┘
```

### Empty (No Rooms)

```
┌─────────────────────────────────────┐
│ ArmorClaw         🔍  👤  ⚙️        │
├─────────────────────────────────────┤
│                                     │
│                                     │
│              💬                     │
│    No conversations yet             │
│                                     │
│    Tap + to start a new chat        │
│                                     │
│                                     │
│                            ┌───┐   │
│                            │ + │   │
│                            └───┘   │
└─────────────────────────────────────┘
```

### Loaded (Default)

```
┌─────────────────────────────────────┐
│ ArmorClaw  (17)   🔍  👤  ⚙️        │
├─────────────────────────────────────┤
│  [+] Join a room                    │
│                                     │
│  ▼ Favorites (1)                    │
│  [Room cards...]                    │
│                                     │
│  ▼ Chats (3)                        │
│  [Room cards...]                    │
│                                     │
│  ▶ Archived (1)                     │
│                            ┌───┐   │
│                            │ + │   │
│                            └───┘   │
└─────────────────────────────────────┘
```

### Error

```
┌─────────────────────────────────────┐
│ ArmorClaw         🔍  👤  ⚙️        │
├─────────────────────────────────────┤
│                                     │
│              ⚠️                     │
│    Failed to load rooms             │
│                                     │
│         [Retry]                     │
│                                     │
└─────────────────────────────────────┘
```

## State Flow

```
                    ┌──────────────┐
                    │   Loading    │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
       ┌──────────┐ ┌──────────┐ ┌──────────┐
       │  Empty   │ │  Loaded  │ │  Error   │
       │  State   │ │  State   │ │  State   │
       └──────────┘ └────┬─────┘ └────┬─────┘
                         │            │
                         │            ▼
                         │     ┌──────────┐
                         │     │  Retry   │
                         │     │  → Load  │
                         │     └──────────┘
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
    ┌─────────┐    ┌─────────┐    ┌─────────┐
    │ Tap     │    │ Tap     │    │ Tap     │
    │ Room    │    │ Search  │    │ FAB     │
    │ Card    │    │ Icon    │    │         │
    └────┬────┘    └────┬────┘    └────┬────┘
         │              │              │
         ▼              ▼              ▼
    ┌─────────┐    ┌─────────┐    ┌─────────┐
    │→ Chat   │    │→ Search │    │→ Create │
    │ Screen  │    │ Screen  │    │ Room    │
    └─────────┘    └─────────┘    └─────────┘
```

## User Flow

1. **User arrives from:**
   - Splash screen (valid session)
   - Login screen (successful auth)
   - Any screen via bottom navigation

2. **User can:**
   - View room list organized by category
   - Expand/collapse favorites and archived sections
   - Tap room card to open chat
   - Search for rooms/messages
   - Join existing room
   - Create new room (FAB)
   - Access profile settings
   - Access app settings

3. **User navigates to:**
   - Chat screen (room tap)
   - Search screen
   - Profile screen
   - Settings screen
   - Room creation flow

## Components Used

| Component | Source | Purpose |
|-----------|--------|---------|
| Scaffold | Material3 | Screen layout |
| TopAppBar | Material3 | Title and actions |
| LazyColumn | Compose | Scrollable room list |
| Card | Material3 | Room item container |
| FloatingActionButton | Material3 | Create room action |
| IconButton | Material3 | Toolbar actions |
| UnreadBadge | Local | Unread count display |
| RoomAvatar | Local | Room avatar with encryption |
| SectionHeader | Local | Collapsible section |
| RoomItemCard | Local | Room list item |

## Accessibility

- **Content descriptions:**
  - Search: "Search"
  - Profile: "Profile"
  - Settings: "Settings"
  - FAB: "Create Room"
  - Expand/collapse: "Expand" / "Collapse"
  - Room cards: Read room name, last message, unread count

- **Touch targets:**
  - Icon buttons: 48.dp
  - Room cards: Full width, ~72.dp height
  - FAB: 56.dp

- **Focus order:**
  1. TopAppBar actions (left to right)
  2. Join room button
  3. Section headers (in order)
  4. Room cards (in order)
  5. FAB

- **Screen reader considerations:**
  - Unread counts announced
  - Encryption status indicated
  - Section state (expanded/collapsed) announced

## Design Tokens

| Token | Value |
|-------|-------|
| TopAppBar color | SurfaceColor |
| FAB color | AccentColor |
| Card background | surfaceVariant |
| Avatar size | 48.dp |
| Badge corner radius | 12.dp |
| Section header padding | 12.dp horizontal, 8.dp vertical |

## Room Card Structure

| Element | Description |
|---------|-------------|
| Avatar | 48.dp circle with initial or image |
| Encryption badge | 16.dp lock icon overlay |
| Room name | bodyLarge, semi-bold |
| Timestamp | bodySmall, 60% opacity |
| Last message | bodyMedium, 70% opacity |
| Unread badge | AccentColor pill |

## Section Behavior

| Section | Default State | Collapsible |
|---------|---------------|-------------|
| Favorites | Expanded | Yes |
| Chats | Always expanded | No |
| Archived | Collapsed | Yes |

## Notes

- Main hub of the application
- Rooms organized by user preference
- Unread badge in header shows total
- FAB for quick room creation
- Encryption indicator on all room avatars
- Sections remember expand/collapse state
- Pull-to-refresh capability (not shown in code)
