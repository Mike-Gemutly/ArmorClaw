# Profile Screen

> **Route:** `profile`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/profile/ProfileScreen.kt`
> **Category:** Main

## Screenshot

![Profile Screen](../../screenshots/main/profile.png)

## Layout

```
┌─────────────────────────────────────┐
│ ←  Profile                     ✏️   │  ← TopAppBar
├─────────────────────────────────────┤
│                                     │
│              👤                     │  ← Avatar (large)
│           John Doe                  │  ← Name
│        @johndoe:matrix.org          │  ← Matrix ID
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 📝 Bio                      │   │
│  │    Security enthusiast.     │   │
│  │    Love encrypted chats!    │   │
│  └─────────────────────────────┘   │
│                                     │
│  ACCOUNT                            │  ← Section header
│  ┌─────────────────────────────┐   │
│  │ 📧 Email                    │   │
│  │    john@example.com      ▶ │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 📱 Phone                    │   │
│  │    +1 (555) 123-4567     ▶ │   │
│  └─────────────────────────────┘   │
│                                     │
│  SECURITY                           │
│  ┌─────────────────────────────┐   │
│  │ 🔒 Change Password        ▶ │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ 🔐 Two-Factor Auth        ▶ │   │
│  └─────────────────────────────┘   │
│                                     │
│  DANGER ZONE                        │
│  ┌─────────────────────────────┐   │
│  │ ⚠️ Delete Account           │   │  ← Destructive action
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Loading

```
┌─────────────────────────────────────┐
│ ←  Profile                          │
├─────────────────────────────────────┤
│                                     │
│           ◠ ◠ ◠                     │
│        Loading profile...           │
│                                     │
└─────────────────────────────────────┘
```

### Loaded (Default)

```
┌─────────────────────────────────────┐
│ ←  Profile                     ✏️   │
├─────────────────────────────────────┤
│  [Avatar]                           │
│  [Name, ID]                         │
│  [Bio]                              │
│                                     │
│  [Account settings]                 │
│  [Security settings]                │
│  [Danger zone]                      │
└─────────────────────────────────────┘
```

### Editing (Edit Mode)

```
┌─────────────────────────────────────┐
│ ←  Edit Profile          [Cancel][Save]│
├─────────────────────────────────────┤
│                                     │
│         [Change Avatar]             │
│              👤                     │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ Display Name                │   │
│  │ John Doe                    │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ Bio                         │   │
│  │ Security enthusiast...      │   │
│  └─────────────────────────────┘   │
│                                     │
└─────────────────────────────────────┘
```

## State Flow

```
            ┌─────────────┐
            │   Loaded    │
            └──────┬──────┘
                   │
    ┌──────────────┼──────────────┐
    ▼              ▼              ▼
┌─────────┐  ┌──────────┐  ┌──────────┐
│ Edit    │  │ Change   │  │ Navigate │
│ Profile │  │ Setting  │  │ to       │
│         │  │          │  │ Subpage  │
└────┬────┘  └──────────┘  └────┬─────┘
     │                         │
     ▼                         ▼
┌──────────┐           ┌──────────────┐
│ Save     │           │ Change PWD   │
│ Changes  │           │ Change Phone │
└──────────┘           │ Delete Acct  │
                       └──────────────┘
```

## User Flow

1. **User arrives from:**
   - Home screen (profile icon)
   - Settings screen (profile card)

2. **User can:**
   - View profile information
   - Edit display name and bio
   - Change avatar
   - Update email
   - Update phone number
   - Change password
   - Configure 2FA
   - Delete account

3. **User navigates to:**
   - Home screen (back)
   - Change password screen
   - Change phone screen
   - Edit bio screen
   - Delete account screen

## Profile Actions

| Action | Icon | Destination |
|--------|------|-------------|
| Edit bio | Edit | Inline edit |
| Change email | Email | Change email dialog |
| Change phone | Phone | Change phone screen |
| Change password | Lock | Change password screen |
| Two-factor auth | Shield | 2FA setup screen |
| Delete account | Warning | Delete account screen |

## Accessibility

- **Content descriptions:**
  - Avatar: "Profile picture, tap to change"
  - Edit button: "Edit profile"
  - Navigation items: "[setting name]"

- **Touch targets:**
  - All items: 48.dp minimum

- **Screen reader considerations:**
  - Profile info announced
  - Section headers announced

## Design Tokens

| Token | Value |
|-------|-------|
| Avatar size | 80.dp |
| Section title | labelLarge, primary |
| Danger zone | error color |

## Notes

- Central hub for account management
- Clear separation of sections
- Destructive actions isolated
- Inline editing for simple changes
