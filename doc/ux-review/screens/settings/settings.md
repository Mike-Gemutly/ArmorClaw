# Settings Screen

> **Route:** `settings`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/settings/SettingsScreen.kt`
> **Category:** Settings

## Screenshot

![Settings Screen](../../screenshots/settings/settings.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←  Settings                         │  ← TopAppBar
├─────────────────────────────────────┤
│  ┌─────────────────────────────┐   │
│  │  👤 John Doe            ▶   │   │  ← Profile section
│  │     john@example.com        │   │
│  └─────────────────────────────┘   │
│                                     │
│  APP SETTINGS                       │  ← Section header
│  ┌─────────────────────────────┐   │
│  │ 🔔 Notifications        [ON]│   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 🎨 Appearance            ▶  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 🔒 Security             [ON]│   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 📱 Devices               ▶  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 🌐 Server Connection     ▶  │   │
│  └─────────────────────────────┘   │
│                                     │
│  PRIVACY                            │
│  ┌─────────────────────────────┐   │
│  │ 🛡️ Privacy Policy        ▶  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ ✅ Data Safety           ▶  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 💾 Data & Storage        ▶  │   │
│  └─────────────────────────────┘   │
│                                     │
│  INVITE                             │
│  ┌─────────────────────────────┐   │
│  │ ➕ Invite to ArmorClaw   ▶  │   │
│  └─────────────────────────────┘   │
│                                     │
│  AI & AGENTS                        │
│  ┌─────────────────────────────┐   │
│  │ 🤖 Agent Management      ▶  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ ⏳ Pending Approvals     ▶  │   │
│  └─────────────────────────────┘   │
│                                     │
│  ABOUT                              │
│  ┌─────────────────────────────┐   │
│  │ ℹ️ About ArmorClaw       ▶  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 🐛 Report a Bug          ▶  │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ ❤️ Rate App                 │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │        LOG OUT              │   │  ← Logout button
│  └─────────────────────────────┘   │
│                                     │
│       ArmorClaw                     │
│       Version 1.0.0 (1)             │  ← Version info
│    Made with ❤️ for privacy         │
└─────────────────────────────────────┘
```

## UI States

### Default / Loaded

```
┌─────────────────────────────────────┐
│ ←  Settings                         │
├─────────────────────────────────────┤
│  [Profile card]                     │
│                                     │
│  [Settings sections with items]     │
│                                     │
│  [Logout button]                    │
│                                     │
│  [Version info]                     │
└─────────────────────────────────────┘
```

### Loading User Data

```
┌─────────────────────────────────────┐
│ ←  Settings                         │
├─────────────────────────────────────┤
│  ┌─────────────────────────────┐   │
│  │     Loading user data...    │   │
│  └─────────────────────────────┘   │
│                                     │
│  [Settings items...]                │
└─────────────────────────────────────┘
```

## State Flow

```
                    ┌──────────────┐
                    │    Loaded    │
                    └──────┬───────┘
                           │
    ┌──────────────────────┼──────────────────────┐
    ▼                      ▼                      ▼
┌─────────┐          ┌──────────┐          ┌──────────┐
│ Toggle  │          │ Navigate │          │ Logout   │
│ Setting │          │ to       │          │          │
│         │          │ Subpage  │          │          │
└─────────┘          └──────────┘          └────┬─────┘
                           │                    │
         ┌─────────────────┼─────────────────┐  ▼
         ▼                 ▼                 ▼  ┌──────────┐
    ┌──────────┐    ┌──────────┐    ┌──────────┐│ Confirm  │
    │ Profile  │    │ Security │    │ Devices  ││ Dialog   │
    │ Screen   │    │ Screen   │    │ Screen   │└──────────┘
    └──────────┘    └──────────┘    └──────────┘
```

## User Flow

1. **User arrives from:**
   - Home screen (settings icon)
   - Any screen via navigation drawer

2. **User can:**
   - View/edit profile (tap profile card)
   - Toggle notification settings
   - Toggle security settings
   - Navigate to all sub-settings
   - Invite friends
   - Manage AI agents
   - View privacy/legal info
   - Report bugs
   - Rate the app
   - Log out

3. **User navigates to:**
   - Profile screen
   - Security settings
   - Notification settings
   - Appearance settings
   - Privacy policy
   - Data safety
   - My data
   - Devices
   - Server connection
   - Agent management
   - Pending approvals
   - Invite screen
   - About screen
   - Report bug screen
   - Login screen (after logout)

## Components Used

| Component | Source | Purpose |
|-----------|--------|---------|
| Scaffold | Material3 | Screen layout |
| TopAppBar | Material3 | Navigation bar |
| Card | Material3 | Profile section |
| Column | Compose | Scrollable content |
| Switch | Material3 | Toggle settings |
| Button | Material3 | Logout action |
| ProfileSection | Local | User profile card |
| SettingsSection | Local | Grouped settings |
| SettingItemCard | Local | Individual setting |
| VersionInfo | Local | App version display |

## Settings Sections

### App Settings
| Item | Icon | Type | Navigation |
|------|------|------|------------|
| Notifications | Notifications | Toggle | - |
| Appearance | Palette | Link | Appearance screen |
| Security | Lock | Toggle | Security screen |
| Devices | Devices | Link | Devices screen |
| Server Connection | Dns | Link | Server screen |

### Privacy
| Item | Icon | Type | Navigation |
|------|------|------|------------|
| Privacy Policy | Security | Link | Privacy screen |
| Data Safety | VerifiedUser | Link | Data Safety screen |
| Data & Storage | Storage | Link | My Data screen |

### Invite
| Item | Icon | Type | Navigation |
|------|------|------|------------|
| Invite to ArmorClaw | PersonAdd | Link | Invite screen |

### AI & Agents
| Item | Icon | Type | Navigation |
|------|------|------|------------|
| Agent Management | SmartToy | Link | Agents screen |
| Pending Approvals | PendingActions | Link | Approvals screen |

### About
| Item | Icon | Type | Navigation |
|------|------|------|------------|
| About ArmorClaw | Info | Link | About screen |
| Report a Bug | BugReport | Link | Report screen |
| Rate App | Favorite | Link | Play Store |

## Accessibility

- **Content descriptions:**
  - Back: "Back"
  - Profile card: "View profile, [name], [email]"
  - Setting items: "[title], [description], [state]"
  - Logout: "Log out"
  - Toggle switches: "[setting name], [on/off]"

- **Touch targets:**
  - All items: 48.dp minimum height
  - Toggle switches: Standard Material size

- **Focus order:**
  1. Back button
  2. Profile section
  3. Settings items (top to bottom)
  4. Logout button

- **Screen reader considerations:**
  - Section headers announced
  - Toggle states announced
  - Navigation destinations announced

## Design Tokens

| Token | Value |
|-------|-------|
| TopAppBar color | SurfaceColor |
| Card background | surfaceVariant |
| Section title color | primary |
| Icon color | AccentColor |
| Chevron opacity | 0.5 (inactive), 0.3 (chevron) |
| Logout button | errorContainer |
| Item padding | 16.dp |
| Section spacing | 8.dp |

## Notes

- Central hub for all app configuration
- Profile section prominent at top
- Settings organized by category
- Toggle switches for quick settings
- Chevron indicates navigable items
- Logout button styled distinctly (red)
- Version info for support reference
- AI/Agent section for advanced features
- Invite section promotes viral growth
