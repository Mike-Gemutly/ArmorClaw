# Profile Feature

> User profile management for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/profile/`

## Overview

The profile feature allows users to view and manage their profile information, including avatar, display name, status, and account settings.

## Feature Components

### ProfileScreen
**Location:** `profile/ProfileScreen.kt`

Main profile screen with viewing and editing modes.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `ProfileScreen()` | Main profile screen | `onNavigateBack`, `onNavigateToSettings`, `onSave`, `onChangeAvatar` |
| `ProfileHeader()` | Avatar and status display | `avatar`, `name`, `status` |
| `ProfileInfoSection()` | Profile fields | `name`, `email`, `status` |
| `EditToggleButton()` | Toggle edit mode | `isEditing`, `onToggle` |
| `ProfileActions()` | Action buttons | `onChangePassword`, `onChangePhone`, `onEditBio`, `onDeleteAccount` |

#### Screen Layout
```
┌────────────────────────────────────┐
│  ← Profile              [Edit]    │
├────────────────────────────────────┤
│                                    │
│          ┌─────────┐               │
│          │   👤    │               │
│          │  Avatar │               │
│          └─────────┘               │
│          John Doe                  │
│          ● Online                  │
│                                    │
├────────────────────────────────────┤
│  Display Name                      │
│  John Doe                          │
├────────────────────────────────────┤
│  Email                             │
│  john@example.com                  │
├────────────────────────────────────┤
│  Status                            │
│  Available                         │
├────────────────────────────────────┤
│  ACCOUNT                           │
│  ├─ Change Password               │
│  ├─ Change Phone Number           │
│  ├─ Edit Bio                      │
│  └─ Delete Account                │
└────────────────────────────────────┘
```

---

### Avatar Editing

#### Avatar with Camera Overlay
```kotlin
@Composable
fun AvatarWithEditOverlay(
    avatar: String?,
    isEditing: Boolean,
    onChangeAvatar: () -> Unit
)
```

**Features:**
- Tap to change avatar
- Camera overlay icon when editing
- Placeholder initial if no avatar

---

### Status Selection

#### Status Options
| Status | Icon | Color |
|--------|------|-------|
| Online | ● | Green |
| Away | ● | Yellow |
| Busy | ● | Red |
| Invisible | ○ | Gray |

#### Status Dropdown
```kotlin
@Composable
fun StatusDropdown(
    currentStatus: UserStatus,
    onStatusChange: (UserStatus) -> Unit
)
```

---

### Editable Fields

#### EditableField
```kotlin
@Composable
fun EditableField(
    label: String,
    value: String,
    isEditing: Boolean,
    onValueChange: (String) -> Unit,
    placeholder: String = ""
)
```

**Behavior:**
- View mode: Card with text
- Edit mode: OutlinedTextField

---

## Account Management Screens

### ChangePasswordScreen
**Location:** `profile/ChangePasswordScreen.kt`

#### Functions
| Function | Description |
|----------|-------------|
| `ChangePasswordScreen()` | Main screen |
| `CurrentPasswordField()` | Password verification |
| `NewPasswordField()` | New password input |
| `ConfirmPasswordField()` | Password confirmation |
| `PasswordStrengthIndicator()` | Strength meter |

#### Password Requirements
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one special character

---

### ChangePhoneNumberScreen
**Location:** `profile/ChangePhoneNumberScreen.kt`

#### Functions
| Function | Description |
|----------|-------------|
| `ChangePhoneNumberScreen()` | Main screen |
| `CurrentPhoneDisplay()` | Show current number |
| `NewPhoneInput()` | New number input |
| `VerificationCodeInput()` | SMS verification |

#### Flow
1. Enter new phone number
2. Request verification code
3. Enter received code
4. Confirm change

---

### EditBioScreen
**Location:** `profile/EditBioScreen.kt`

#### Functions
| Function | Description |
|----------|-------------|
| `EditBioScreen()` | Main screen |
| `BioTextField()` | Multi-line input |
| `CharacterCounter()` | Count with limit |
| `BioPreview()` | Formatted preview |

#### Bio Limits
- Maximum: 500 characters
- Supports: Plain text

---

### DeleteAccountScreen
**Location:** `profile/DeleteAccountScreen.kt`

#### Functions
| Function | Description |
|----------|-------------|
| `DeleteAccountScreen()` | Main screen |
| `WarningCard()` | Deletion warnings |
| `ConfirmationInput()` | Type "DELETE" |
| `DeleteButton()` | Final action |

#### Warnings Displayed
- All messages will be deleted
- All rooms will be left
- Data cannot be recovered
- 30-day grace period (optional)

---

## User Model

### UserProfile
```kotlin
data class UserProfile(
    val id: String,
    val displayName: String,
    val email: String,
    val phoneNumber: String?,
    val avatar: String?,
    val status: UserStatus,
    val statusMessage: String?,
    val bio: String?,
    val createdAt: Instant,
    val updatedAt: Instant
)
```

### UserStatus
```kotlin
enum class UserStatus {
    ONLINE,
    AWAY,
    BUSY,
    INVISIBLE
}
```

---

## State Management

### ProfileState
```kotlin
data class ProfileState(
    val profile: UserProfile?,
    val isEditing: Boolean,
    val isLoading: Boolean,
    val isSaving: Boolean,
    val error: String?
)
```

### ProfileActions
| Action | Description |
|--------|-------------|
| `loadProfile()` | Fetch user profile |
| `saveProfile()` | Save changes |
| `toggleEditMode()` | Toggle editing |
| `updateField()` | Update field value |
| `changeAvatar()` | Update avatar |
| `changeStatus()` | Update status |

---

## Related Documentation

- [Settings](settings.md) - App settings
- [Authentication](authentication.md) - Login/Registration
