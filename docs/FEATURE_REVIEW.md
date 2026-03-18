# ArmorClaw Feature Review & User Flow Analysis

## Executive Summary
This document outlines all features in the ArmorClaw app, user stories for transitions between features, and identifies gaps in the user journey.

**Status:** ✅ All gaps have been fixed.

---

## Recent Fixes Applied (Updated)

### ✅ GAP 1: Search Navigation Fixed
- Added `onNavigateToSearch` callback to HomeScreenFull
- Wired navigation from Home → Search in AppNavigation
- Search icon in Home app bar now navigates to Search screen

### ✅ GAP 2: Chat Screen Room Details Fixed
- Added `onNavigateToRoomDetails` callback to ChatScreenEnhanced
- Chat top bar title is now clickable
- Clicking on chat title navigates to Room Details screen

### ✅ GAP 3: Settings Navigation Fixed
- Added `onNavigateToMyData`, `onNavigateToReportBug`, `onNavigateToDevices` callbacks to SettingsScreen
- Added "Devices" menu item in Settings
- All settings items now properly navigate to their respective screens

### ✅ GAP 4: Security Settings Device Management Fixed
- Added `onNavigateToDevices` callback to SecuritySettingsScreen
- Added "Manage Devices" menu item at the top of Security settings
- Added proper navigation to Device List screen

### ✅ GAP 5: Profile Account Options Fixed
- Added `onNavigateToChangePassword`, `onNavigateToChangePhone`, `onNavigateToEditBio`, `onNavigateToDeleteAccount` callbacks to ProfileScreen
- Account options now navigate to their respective screens

### ✅ GAP 6: Call Flow Integration Fixed
- Added `ACTIVE_CALL` and `INCOMING_CALL` routes to AppNavigation
- Added call buttons (voice & video) to ChatScreenEnhanced top bar
- Created call route navigation with callId parameter
- Incoming call dialog navigates to active call on accept
- Active call screen properly handles end call navigation

### ✅ GAP 7: Emoji Verification Integration Fixed
- Added `EMOJI_VERIFICATION` route to AppNavigation
- Added `onVerifyDeviceClick` callback to DeviceListScreen
- Verify button in device details now navigates to verification screen
- Verification screen handles confirm/deny/cancel actions

### ✅ GAP 8: Logout Session Clearing Fixed
- Created `LogoutUseCase` with comprehensive session cleanup
- Added `clearLocalAuth()` method to AuthRepository interface
- Logout now clears tokens, stops sync, and clears local data
- Proper error handling and logging during logout

### ✅ GAP 9: About Screen External Links Fixed
- Created `ExternalLinkHandler` utility class
- Added website, GitHub, Twitter, Terms, Privacy navigation
- Licenses screen with open source library list
- Terms of Service screen with legal content

### ✅ GAP 10: User Profile Screen Added
- Created `UserProfileScreen` for viewing other users
- Added `USER_PROFILE` route to navigation
- Shows user info, presence, verification status
- Actions: Message, Call, Block, Report

### ✅ GAP 11: Thread View Added
- Created `ThreadViewScreen` for threaded conversations
- Added `THREAD` route to navigation
- Shows root message and replies
- Reply input for thread responses

### ✅ GAP 12: Media Viewers Added
- Created `ImageViewerScreen` with zoom/pan
- Created `FilePreviewScreen` with type detection
- Added navigation routes for both

### ✅ GAP 13: Deep Link Support Added
- Deep link handling in MainActivity
- Matrix.to link parsing
- App-specific deep link support (armorclaw://room/{id})

### ✅ GAP 14: Notification Deep Links Added
- Created `NotificationDeepLinkHandler`
- Handles message, call, verification notifications
- Creates proper navigation actions

### ✅ GAP 15: Mention Handling Added
- Created `MentionHandler` with detection and styling
- User and room mention chips
- Mention preview popup

---

## Feature Categories

### 1. Onboarding Flow
**Screens:** Welcome → Security Explanation (4 steps) → Connect Server → Permissions → Completion

**User Stories:**
- As a new user, I want to learn about the app's security features before signing up
- As a new user, I want to connect to my Matrix server
- As a new user, I want to grant necessary permissions
- As a new user, I want to start chatting after setup completes

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Splash | Welcome | First time user detected |
| Welcome | Security/0 | Tap "Get Started" |
| Security/N | Security/N+1 or Connect | Tap "Next" |
| Connect | Permissions | Server connected |
| Permissions | Completion | Permissions granted |
| Completion | Home | Tap "Start Chatting" |

### 2. Authentication Flow
**Screens:** Login, Registration, Forgot Password

**User Stories:**
- As a returning user, I want to log in with my credentials
- As a returning user, I want to log in with biometrics
- As a new user, I want to register a new account
- As a user, I want to reset my password if I forgot it

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Splash | Login | User has account |
| Login | Home | Successful login |
| Login | Registration | Tap "Create Account" |
| Login | Forgot Password | Tap "Forgot Password" |
| Registration | Home | Successful registration |
| Forgot Password | Login | Password reset sent |

### 3. Home/Room List
**Screens:** Home Screen

**User Stories:**
- As a user, I want to see all my chat rooms
- As a user, I want to see unread message counts
- As a user, I want to quickly access my favorites
- As a user, I want to create or join a new room

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Home | Chat | Tap on room |
| Home | Settings | Tap settings icon |
| Home | Profile | Tap profile icon |
| Home | Room Management | Tap FAB |
| Home | Search | Tap search icon |

### 4. Chat Flow
**Screens:** Chat Screen, Thread View, Message Actions

**User Stories:**
- As a user, I want to send and receive messages
- As a user, I want to reply to specific messages
- As a user, I want to start or participate in threads
- As a user, I want to see encryption status
- As a user, I want to send attachments

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Chat | Home | Back navigation |
| Chat | Room Details | Tap room name |
| Chat | Thread View | Tap thread indicator |
| Chat | Full Screen Image | Tap image attachment |

### 5. Room Management
**Screens:** Room Management (Create/Join), Room Details, Room Settings

**User Stories:**
- As a user, I want to create a new room
- As a user, I want to join an existing room
- As a user, I want to view room details and members
- As a user, I want to configure room settings
- As a user, I want to leave or archive a room

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Home | Room Management | Tap FAB |
| Room Management | Home | Room created/joined |
| Chat | Room Details | Tap room name |
| Room Details | Room Settings | Tap settings |
| Room Settings | Chat | Save settings |
| Room Settings | Home | Leave room |

### 6. Profile Management
**Screens:** Profile, Change Password, Change Phone, Edit Bio, Delete Account

**User Stories:**
- As a user, I want to view and edit my profile
- As a user, I want to change my avatar
- As a user, I want to change my password
- As a user, I want to update my phone number
- As a user, I want to delete my account

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Home | Profile | Tap profile icon |
| Settings | Profile | Tap profile section |
| Profile | Settings | Tap settings button |
| Profile | Change Password | Tap change password |
| Profile | Change Phone | Tap change phone |
| Profile | Edit Bio | Tap edit bio |
| Profile | Delete Account | Tap delete account |
| Delete Account | Login | Account deleted |

### 7. Settings Flow
**Screens:** Settings, Security Settings, Notification Settings, Appearance, Privacy Policy, My Data, About, Report Bug, Device List

**User Stories:**
- As a user, I want to configure notification preferences
- As a user, I want to enable/disable security features
- As a user, I want to change the app appearance
- As a user, I want to read the privacy policy
- As a user, I want to download my data
- As a user, I want to view connected devices
- As a user, I want to report a bug

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Home | Settings | Tap settings |
| Settings | Profile | Tap profile section |
| Settings | Security Settings | Tap security |
| Settings | Notifications | Tap notifications |
| Settings | Appearance | Tap appearance |
| Settings | Privacy Policy | Tap privacy |
| Settings | My Data | Tap data & storage |
| Settings | About | Tap about |
| Settings | Login | Tap logout |
| Security Settings | Device List | Tap manage devices |
| About | Privacy Policy | Tap privacy |

### 8. Search Flow
**Screens:** Search

**User Stories:**
- As a user, I want to search for rooms
- As a user, I want to search for messages
- As a user, I want to search for people

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Home | Search | Tap search icon |
| Search | Chat | Tap on result |
| Search | Home | Back navigation |

### 9. Call Flow (Future)
**Screens:** Incoming Call Dialog, Active Call Screen, Call Controls

**User Stories:**
- As a user, I want to receive voice calls
- As a user, I want to receive video calls
- As a user, I want to mute/end calls

**Transitions:**
| From | To | Trigger |
|------|-----|---------|
| Any | Incoming Call | Incoming call notification |
| Incoming Call | Active Call | Accept call |
| Incoming Call | Previous | Reject call |
| Active Call | Previous | End call |

---

## All Gaps Resolved ✅

### ✅ GAP 6: Deep Link Support (Fixed 2026-02-15)
**Solution:** Created `DeepLinkHandler.kt` utility class
- Supports `armorclaw://room/{id}`, `armorclaw://user/{id}`, `armorclaw://call/{id}`
- Supports `https://matrix.to/#/{roomId}` links
- Supports `https://chat.armorclaw.app/room/{id}` links
- Updated `MainActivity.kt` with `onNewIntent()` handling and `DeepLinkState`
- Updated `ArmorClawNavHost.kt` to use `DeepLinkAction`

### ✅ GAP 7: Logout Session Clearing (Fixed 2026-02-15)
**Solution:** Connected ProfileScreen to LogoutUseCase via ProfileViewModel
- `ProfileViewModel` calls `LogoutUseCase` on logout
- Proper session cleanup: stops sync, clears tokens, clears local data
- Navigation to login screen after successful logout

### ✅ GAP 8: Profile State Persistence (Fixed 2026-02-15)
**Solution:** Created `ProfileViewModel` for state management
- Profile state now survives configuration changes (rotation, etc.)
- Added to DI module with UserRepository mock
- Proper loading/saving flow with error handling

### ✅ GAP 9: Onboarding State Persistence (Already Implemented)
**Solution:** `OnboardingPreferences.kt` uses SharedPreferences
- Persists onboarding completion status
- Stores current step, server URL, username
- Persists permissions granted status

### ✅ GAP 10: Navigation Error Handling (Already Implemented)
**Solution:** `NavigationExtensions.kt` provides safe navigation
- `navigateSafely()` with try-catch and error logging
- `popBackStackSafely()` with error handling
- `navigateUpSafely()` with error handling
- Shows snackbar on navigation failures

---

## Transition Matrix (Updated)

| From/To | Home | Chat | Profile | Settings | Search | Rooms | Auth |
|---------|------|------|---------|----------|--------|-------|------|
| Home    | -    | ✅   | ✅      | ✅       | ✅     | ✅    | ✅   |
| Chat    | ✅   | -    | ✅      | ✅       | ✅     | ✅    | -    |
| Profile | ✅   | -    | -       | ✅       | -      | -     | ✅   |
| Settings| ✅   | -    | ✅      | -        | -      | -     | ✅   |
| Search  | ✅   | ✅   | -       | -        | -      | -     | -    |
| Rooms   | ✅   | ✅   | -       | -        | -      | -     | -    |
| Auth    | ✅   | -    | -       | -        | -      | -     | ✅   |

Legend:
- ✅ = Transition implemented
- \- = Not applicable

### Chat Screen Navigation
| From Chat | To | Trigger |
|-----------|-----|---------|
| Chat | Home | Back button |
| Chat | Room Details | Tap chat title |
| Chat | Search | Tap search icon |

### Profile Screen Navigation
| From Profile | To | Trigger |
|--------------|-----|---------|
| Profile | Home | Back button |
| Profile | Settings | Tap settings |
| Profile | Change Password | Tap change password |
| Profile | Change Phone | Tap change phone |
| Profile | Edit Bio | Tap edit bio |
| Profile | Delete Account | Tap delete account |

### Settings Screen Navigation
| From Settings | To | Trigger |
|---------------|-----|---------|
| Settings | Home | Back button |
| Settings | Profile | Tap profile section |
| Settings | Security | Tap security |
| Settings | Notifications | Tap notifications |
| Settings | Appearance | Tap appearance |
| Settings | Devices | Tap devices |
| Settings | Privacy | Tap privacy |
| Settings | My Data | Tap data & storage |
| Settings | About | Tap about |
| Settings | Report Bug | Tap report bug |
| Settings | Login | Tap logout |

---

## Priority Fixes Status

### ✅ High Priority (All Complete)
1. ✅ GAP 1: Add search navigation
2. ✅ GAP 3: Add missing settings navigation
3. ✅ GAP 5: Add profile options navigation

### ✅ Medium Priority (All Complete)
4. ✅ GAP 2: Add room details from chat
5. ✅ GAP 4: Add device management from security

### ✅ Low Priority (All Complete)
6. ✅ GAP 6: Deep link support (Fixed 2026-02-15)
7. ✅ GAP 7: Fix logout session clearing (Fixed 2026-02-15)
8. ✅ GAP 8: Profile state persistence (Fixed 2026-02-15)
9. ✅ GAP 9: Onboarding state (Already implemented)
10. ✅ GAP 10: Navigation error handling (Already implemented)

---

## User Journey Examples

### Journey 1: New User Onboarding
1. Open app → Splash screen
2. First time user → Welcome screen
3. Tap "Get Started" → Security explanation (4 steps)
4. Complete security → Connect to server
5. Server connected → Permissions request
6. Permissions granted → Completion screen
7. Tap "Start Chatting" → Home screen

### Journey 2: Existing User Login
1. Open app → Splash screen
2. Returning user → Login screen
3. Enter credentials or use biometric → Home screen

### Journey 3: Create Room
1. Home screen → Tap FAB → Room Management
2. Enter room name and settings → Tap Create
3. Room created → Navigate to new room's Chat

### Journey 4: Search and Navigate
1. Home screen → Tap search icon → Search screen
2. Enter search query → View results
3. Tap on room result → Navigate to Chat
4. Chat → Tap title → Room Details
5. Room Details → Tap settings → Room Settings

### Journey 5: Account Management
1. Home screen → Tap profile icon → Profile screen
2. Edit profile fields → Tap Save
3. Tap "Change Password" → Change Password screen
4. Enter passwords → Save → Back to Profile
5. Profile → Tap settings → Settings screen
6. Settings → Tap Security → Security Settings
7. Security Settings → Tap Manage Devices → Device List

### Journey 6: Settings to All Features
1. Home → Settings icon → Settings screen
2. Any of the following:
   - Profile section → Profile
   - Notifications → Notification Settings
   - Appearance → Appearance Settings
   - Security → Security Settings → Devices
   - Privacy → Privacy Policy
   - Data & Storage → My Data
   - About → About → Privacy Policy
   - Report a Bug → Report Bug
3. All screens have back navigation to return to Settings

---

## Call Flow (Future Feature)

### Journey 7: Incoming Call
1. Any screen → Incoming call notification → Incoming Call Dialog
2. Accept → Active Call Screen
3. End call → Return to previous screen

### Journey 8: Outgoing Call (To Implement)
1. Chat screen → Tap call button → Active Call Screen
2. End call → Return to Chat
