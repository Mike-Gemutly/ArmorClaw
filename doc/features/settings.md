# Settings Feature

> App configuration and preferences for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/settings/`

## Overview

The settings feature provides users with control over app behavior, appearance, security, and privacy options.

## Feature Components

### SettingsScreen
**Location:** `settings/SettingsScreen.kt`

Main settings screen with categorized options.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `SettingsScreen()` | Main settings screen | `onNavigateBack`, `onNavigateToProfile`, `onNavigateToSecurity`, `onNavigateToNotifications`, `onNavigateToAppearance`, `onLogout` |
| `SettingsHeader()` | User profile summary | `avatar`, `name`, `email` |
| `SettingsSection()` | Grouped settings items | `title`, `items` |
| `SettingsItem()` | Individual setting row | `icon`, `title`, `subtitle`, `onClick` |

#### Screen Layout
```
┌────────────────────────────────────┐
│  ← Settings                        │
├────────────────────────────────────┤
│  ┌──────────────────────────────┐  │
│  │  👤 John Doe                 │  │
│  │     john@example.com         │  │
│  └──────────────────────────────┘  │
├────────────────────────────────────┤
│  APP SETTINGS                      │
│  ├─ 🔔 Notifications              │
│  ├─ 🔊 Sounds                     │
│  ├─ 📳 Vibration                  │
│  └─ 🎨 Appearance                 │
├────────────────────────────────────┤
│  PRIVACY & SECURITY                │
│  ├─ 🔐 Security                   │
│  ├─ 📱 Devices                    │
│  ├─ 📋 Privacy Policy             │
│  └─ 💾 My Data                    │
├────────────────────────────────────┤
│  ABOUT                             │
│  ├─ ℹ️ About ArmorClaw            │
│  ├─ 🐛 Report a Bug               │
│  └─ ⭐ Rate App                   │
├────────────────────────────────────┤
│  [  LOGOUT  ]                      │
└────────────────────────────────────┘
```

---

### NotificationSettingsScreen
**Location:** `settings/NotificationSettingsScreen.kt`

Notification preferences configuration.

#### Settings

| Setting | Type | Default |
|---------|------|---------|
| Enable Notifications | Toggle | true |
| Message Notifications | Toggle | true |
| Mention Notifications | Toggle | true |
| Call Notifications | Toggle | true |
| Sound | Toggle | true |
| Vibration | Toggle | true |
| Preview Messages | Toggle | true |
| Quiet Hours | Time Range | Off |

#### Functions

| Function | Description |
|----------|-------------|
| `NotificationSettingsScreen()` | Main screen |
| `NotificationToggle()` | Toggle switch |
| `QuietHoursPicker()` | Time range selector |

---

### SecuritySettingsScreen
**Location:** `settings/SecuritySettingsScreen.kt`

Security and privacy options.

#### Settings

| Setting | Type | Description |
|---------|------|-------------|
| Biometric Login | Toggle | Use fingerprint/face |
| Auto-Lock | Dropdown | Lock after X minutes |
| Screen Security | Toggle | Prevent screenshots |
| Incognito Keyboard | Toggle | Disable keyboard suggestions |
| Message Expiration | Dropdown | Auto-delete messages |

#### Functions

| Function | Description |
|----------|-------------|
| `SecuritySettingsScreen()` | Main screen |
| `BiometricSetting()` | Biometric toggle |
| `AutoLockSetting()` | Timeout selector |
| `SecurityInfoCard()` | Security explanation |

---

### AppearanceSettingsScreen
**Location:** `settings/AppearanceSettingsScreen.kt`

Visual appearance configuration.

#### Settings

| Setting | Type | Options |
|---------|------|---------|
| Theme | Dropdown | Light, Dark, System |
| Accent Color | Color Picker | Brand colors |
| Font Size | Slider | Small, Normal, Large |
| Chat Bubbles | Toggle | Rounded/Squared |
| Timestamp Format | Dropdown | 12h/24h |

#### Functions

| Function | Description |
|----------|-------------|
| `AppearanceSettingsScreen()` | Main screen |
| `ThemeSelector()` | Theme picker |
| `ColorPicker()` | Accent color selection |
| `PreviewCard()` | Settings preview |

---

### DeviceListScreen
**Location:** `settings/DeviceListScreen.kt`

Manage trusted devices.

#### Functions

| Function | Description |
|----------|-------------|
| `DeviceListScreen()` | Main screen |
| `DeviceListItem()` | Device card |
| `DeviceSectionHeader()` | Section divider |
| `DeviceDetailsDialog()` | Device info dialog |
| `DeviceListEmptyState()` | No devices placeholder |

#### Device Information Displayed
- Device name
- Last seen time
- Trust level
- Last IP (masked)
- Device ID (truncated)

#### Device Actions
- Verify unverified device
- Remove device
- View details

---

### AboutScreen
**Location:** `settings/AboutScreen.kt`

App information and credits.

#### Information Displayed

| Item | Content |
|------|---------|
| App Name | ArmorClaw |
| Version | 1.0.0 |
| Build | Debug/Release |
| License | Open Source |
| Website | armorclaw.app |
| Support | support@armorclaw.app |

#### Functions

| Function | Description |
|----------|-------------|
| `AboutScreen()` | Main screen |
| `VersionInfo()` | Version display |
| `LinkButton()` | External link |
| `LicenseInfo()` | License details |

---

### ReportBugScreen
**Location:** `settings/ReportBugScreen.kt`

Bug reporting interface.

#### Functions

| Function | Description |
|----------|-------------|
| `ReportBugScreen()` | Main screen |
| `BugTypeSelector()` | Bug category |
| `DescriptionInput()` | Text input |
| `ScreenshotAttachment()` | Image picker |
| `SubmitButton()` | Send report |

#### Bug Categories
- App Crash
- UI Issue
- Feature Request
- Security Issue
- Other

---

### PrivacyPolicyScreen
**Location:** `settings/PrivacyPolicyScreen.kt`

Privacy policy display.

#### Sections
1. Information We Collect
2. How We Use Information
3. Data Security
4. Third-Party Services
5. Your Rights
6. Contact Us

---

### MyDataScreen
**Location:** `settings/MyDataScreen.kt`

Data management options.

#### Options

| Action | Description |
|--------|-------------|
| Export Data | Download all user data |
| Clear Cache | Remove cached files |
| Delete Messages | Remove all messages |
| Delete Account | Permanently delete account |

---

## Settings Components

### SettingsComponents
**Location:** `settings/SettingsComponents.kt`

Shared settings UI components.

#### Components

| Component | Description |
|-----------|-------------|
| `SettingsItem()` | Standard setting row |
| `SettingsToggle()` | Toggle switch row |
| `SettingsDropdown()` | Dropdown selector |
| `SettingsHeader()` | Section header |
| `SettingsDivider()` | Visual separator |
| `SettingsInfoCard()` | Information card |

---

## State Persistence

### Settings State
- Stored in DataStore
- Encrypted where sensitive
- Synced across devices

### Preference Keys
```kotlin
object PreferenceKeys {
    val NOTIFICATIONS_ENABLED = booleanPreferencesKey("notifications_enabled")
    val THEME_MODE = stringPreferencesKey("theme_mode")
    val BIOMETRIC_ENABLED = booleanPreferencesKey("biometric_enabled")
    // ... etc
}
```

---

## Related Documentation

- [Profile](profile.md) - User profile management
- [Device Management](device-management.md) - Device verification
- [Encryption](encryption.md) - Security implementation
