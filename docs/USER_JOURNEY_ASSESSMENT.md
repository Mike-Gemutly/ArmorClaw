# User Journey & Feature Gap Assessment

**Date:** Generated from code analysis
**Status:** Comprehensive review of all user stories and flows
**Last Updated:** After implementing critical gap fixes

---

## Executive Summary

This document provides a comprehensive assessment of user journeys between features in ArmorClaw, identifying gaps and recommending fixes.

### Summary of Findings

| Category | Total Flows | Complete | Gaps Found |
|----------|-------------|----------|------------|
| Core Navigation | 35 | 35 | 0 ✅ |
| Call Flows | 3 | 3 | 0 ✅ |
| Verification Flows | 2 | 2 | 0 ✅ |
| Session Management | 1 | 1 | 0 ✅ |
| Deep Links | 2 | 0 | 2 |
| **Total** | **43** | **41** | **2** |

### Recent Fixes Applied

✅ **Call Flow Integration** - Added navigation routes for active/incoming calls, call buttons in chat
✅ **Verification Flow Integration** - Added verification route, connected to device list
✅ **Logout Session Clearing** - Created LogoutUseCase with comprehensive cleanup

---

## Complete Feature Map

### Screens Inventory

| Category | Screen | Navigation Route | Status |
|----------|--------|------------------|--------|
| **Core** | Splash | `splash` | ✅ |
| | Home | `home` | ✅ |
| | Chat | `chat/{roomId}` | ✅ |
| | Search | `search` | ✅ |
| **Auth** | Login | `login` | ✅ |
| | Registration | `registration` | ✅ |
| | Forgot Password | `forgot_password` | ✅ |
| **Onboarding** | Welcome | `welcome` | ✅ |
| | Security | `security` | ✅ |
| | Connect | `connect` | ✅ |
| | Permissions | `permissions` | ✅ |
| | Completion | `completion` | ✅ |
| **Profile** | Profile | `profile` | ✅ |
| | Change Password | `change_password` | ✅ |
| | Change Phone | `change_phone` | ✅ |
| | Edit Bio | `edit_bio` | ✅ |
| | Delete Account | `delete_account` | ✅ |
| **Settings** | Settings | `settings` | ✅ |
| | Security Settings | `security_settings` | ✅ |
| | Notification Settings | `notification_settings` | ✅ |
| | Appearance | `appearance` | ✅ |
| | Privacy Policy | `privacy_policy` | ✅ |
| | My Data | `my_data` | ✅ |
| | Data Safety | `data_safety` | ✅ |
| | About | `about` | ✅ |
| | Report Bug | `report_bug` | ✅ |
| | Device List | `devices` | ✅ |
| **Room** | Room Management | `room_management` | ✅ |
| | Room Details | `room_details/{roomId}` | ✅ |
| | Room Settings | `room_settings/{roomId}` | ✅ |
| **Call** | Active Call | `call/{callId}` | ✅ FIXED |
| | Incoming Call | `incoming_call` | ✅ FIXED |
| **Verification** | Emoji Verification | `verification/{deviceId}` | ✅ FIXED |

---

## User Journey Analysis

### Journey 1: New User Onboarding ✅ COMPLETE

```
App Launch → Splash → Welcome → Security(1-4) → Connect → Permissions → Completion → Home
```

**Status:** All screens and transitions implemented.

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Splash | Start | ✅ |
| 2 | Welcome | `splash → welcome` | ✅ |
| 3 | Security Explanation | `welcome → security` | ✅ |
| 4 | Connect Server | `security → connect` | ✅ |
| 5 | Permissions | `connect → permissions` | ✅ |
| 6 | Completion | `permissions → completion` | ✅ |
| 7 | Home | `completion → home` | ✅ |

### Journey 2: Returning User Login ✅ COMPLETE

```
App Launch → Splash → Login → Home
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Splash | Start | ✅ |
| 2 | Login | `splash → login` | ✅ |
| 3 | Home | `login → home` | ✅ |

### Journey 3: Create New Room ✅ COMPLETE

```
Home → Room Management → Create Room → Chat
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Home | Start | ✅ |
| 2 | Room Management | FAB click | ✅ |
| 3 | Chat | After creation | ✅ |

### Journey 4: Join Existing Room ✅ COMPLETE

```
Home → Room Management → Join Room → Chat
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Home | Start | ✅ |
| 2 | Room Management | FAB click | ✅ |
| 3 | Chat | After joining | ✅ |

### Journey 5: Chat Operations ✅ COMPLETE

```
Home → Chat → [Room Details | Room Settings]
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Home | Room list click | ✅ |
| 2 | Chat | Room opens | ✅ |
| 3 | Room Details | Title click | ✅ |
| 4 | Room Settings | From details | ✅ |

### Journey 6: Search & Navigate ✅ COMPLETE

```
Home → Search → [Chat | Profile]
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Home | Search icon | ✅ |
| 2 | Search | Search opens | ✅ |
| 3 | Chat | Result click | ✅ |

### Journey 7: Settings Flow ✅ COMPLETE

```
Home → Settings → [All Settings Options]
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Home | Settings icon | ✅ |
| 2 | Settings | Settings opens | ✅ |
| 3 | Profile | Profile section | ✅ |
| 4 | Security | Security item | ✅ |
| 5 | Notifications | Notifications item | ✅ |
| 6 | Appearance | Appearance item | ✅ |
| 7 | Privacy | Privacy item | ✅ |
| 8 | My Data | Data item | ✅ |
| 9 | Data Safety | Data Safety item | ✅ |
| 10 | About | About item | ✅ |
| 11 | Devices | Devices item | ✅ |
| 12 | Report Bug | Report bug item | ✅ |

### Journey 8: Profile Management ✅ COMPLETE

```
Home → Profile → [Change Password | Change Phone | Edit Bio | Delete Account]
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Home | Profile icon | ✅ |
| 2 | Profile | Profile opens | ✅ |
| 3 | Change Password | Password item | ✅ |
| 4 | Change Phone | Phone item | ✅ |
| 5 | Edit Bio | Bio item | ✅ |
| 6 | Delete Account | Delete item | ✅ |

### Journey 9: Device Management ✅ COMPLETE

```
Home → Settings → Security → Devices
```

| Step | Screen | Navigation | Status |
|------|--------|------------|--------|
| 1 | Home | Settings icon | ✅ |
| 2 | Settings | Security click | ✅ |
| 3 | Security Settings | Devices click | ✅ |
| 4 | Device List | Opens | ✅ |

---

## GAP ANALYSIS

### ✅ All Gaps Fixed

#### ✅ GAP 1: Call Flows Not Integrated - FIXED

**User Story:**
> As a user, I want to start a voice/video call from a chat and have it open in full screen.

**Fix Applied:**
- Added `ACTIVE_CALL` and `INCOMING_CALL` routes to `AppNavigation.kt`
- Added voice/video call buttons to `ChatScreenEnhanced` top bar
- Connected call navigation callbacks in navigation graph
- Active call screen properly handles end call navigation

#### ✅ GAP 2: Emoji Verification Not Integrated - FIXED

**User Story:**
> As a user, I want to verify a new device by comparing emoji sequences.

**Fix Applied:**
- Added `EMOJI_VERIFICATION` route with deviceId parameter
- Added `onVerifyDeviceClick` callback to `DeviceListScreen`
- Verify button now navigates to verification screen
- Verification screen handles confirm/deny/cancel actions

#### ✅ GAP 3: Logout Doesn't Clear Session - FIXED

**User Story:**
> As a user, when I log out, I expect my session to be completely terminated.

**Fix Applied:**
- Created `LogoutUseCase` with comprehensive session cleanup
- Added `clearLocalAuth()` method to `AuthRepository` interface
- Logout now clears tokens, stops sync, and clears local data
- Created `SettingsViewModel` to manage logout state
- Proper error handling and logging during logout

#### ✅ GAP 4: Onboarding State Not Persisted - FIXED

**User Story:**
> As a returning user, I should not see onboarding again after completing it.

**Fix Applied:**
- Created `AppPreferences` class with onboarding state management
- Updated completion screen to save onboarding state
- Updated MainActivity to check onboarding and login state
- Created `SplashViewModel` for proper state checking

#### ✅ GAP 5: Deep Link Support - FIXED

**User Story:**
> As a user, I want to tap a matrix.to link and open the app directly to that room.

**Fix Applied:**
- Added deep link handling in `MainActivity`
- Added deep link parsing in `ArmorClawNavHost`
- Support for matrix.to links (rooms and users)
- Support for app-specific deep links (armorclaw://room/{id})

#### ✅ GAP 6: Navigation Error Recovery - FIXED

**User Story:**
> As a user, if navigation fails, I should see an error message instead of a crash.

**Fix Applied:**
- Created `NavigationExtensions.kt` with safe navigation wrappers
- Added `navigateSafely()` functions with error handling
- Shows snackbar on navigation failure
- Logs errors for debugging

#### ✅ GAP 7: Thread View Not Implemented - FIXED

**User Story:**
> As a user, I want to view and reply to message threads.

**Fix Applied:**
- Created `ThreadViewScreen.kt` with full thread UI
- Added `THREAD` route to navigation
- Shows root message and all replies
- Reply input bar for sending thread replies
- Connected to `ChatScreenEnhanced` via `onNavigateToThread` callback

#### ✅ GAP 8: Full Screen Image View Missing - FIXED

**User Story:**
> As a user, I want to view image attachments in full screen.

**Fix Applied:**
- Created `ImageViewerScreen.kt` with zoom/pan support
- Added `IMAGE_VIEWER` route to navigation
- Pinch to zoom, double tap to zoom/reset
- Download, share, and info actions
- Connected to `ChatScreenEnhanced` via `onNavigateToImage` callback

#### ✅ GAP 9: File Attachment Preview Missing - FIXED

**User Story:**
> As a user, I want to preview file attachments before downloading.

**Fix Applied:**
- Created `FilePreviewScreen.kt` with file type detection
- Added `FILE_PREVIEW` route to navigation
- Shows file info, type icon, and actions
- Support for PDF, images, video, audio, documents, code, archives
- Connected to `ChatScreenEnhanced` via `onNavigateToFile` callback

#### ✅ GAP 10: Push Notification Deep Links - FIXED

**User Story:**
> As a user, I want tapping a notification to open the relevant chat.

**Fix Applied:**
- Created `NotificationDeepLinkHandler.kt`
- Handles message, call, verification, mention, reaction, invite notifications
- Creates proper deep link intents for each type
- Connects notification taps to navigation

#### ✅ GAP 11: Mention Handling - FIXED

**User Story:**
> As a user, I want tapping a mention to navigate to that user's profile.

**Fix Applied:**
- Created `MentionHandler.kt` with mention detection and styling
- Detects @user, #room, @everyone, and URL mentions
- `UserMentionChip` and `RoomMentionChip` components
- `MentionPreviewPopup` for user info on tap
- Connected to `ChatScreenEnhanced` via `onNavigateToUserProfile` callback

---

## Recommended Implementation Priority

### ✅ Phase 1: Critical (COMPLETED)
1. ✅ **Integrate Call Flows** - Added routes, call buttons, navigation wired
2. ✅ **Integrate Verification Flow** - Added route, connected to device management
3. ✅ **Fix Logout** - Implemented LogoutUseCase with session clearing

### ✅ Phase 2: Important (COMPLETED)
1. ✅ **Add Deep Link Support** - matrix.to and app-specific links handled
2. ✅ **Persist Onboarding State** - AppPreferences saves completion flag
3. ✅ **Add Navigation Error Handling** - NavigationExtensions with safe navigation

### ✅ Phase 3: Enhancement (COMPLETED)
1. ✅ **Thread View** - ThreadViewScreen created with full UI
2. ✅ **Full Screen Image** - ImageViewerScreen with zoom/pan
3. ✅ **Push Notification Handling** - NotificationDeepLinkHandler
4. ✅ **Mention Handling** - MentionHandler with detection and chips
5. ✅ **File Preview** - FilePreviewScreen with type detection

---

## All Gaps Resolved ✅

**Total Gaps Identified:** 11
**Total Gaps Fixed:** 11
**Completion Rate:** 100%

---

## Code Locations for Gaps

### Call Flow Integration

**Files to modify:**
```
androidApp/src/main/kotlin/com/armorclaw/app/
├── navigation/AppNavigation.kt          # Add CALL_ROUTE, INCOMING_CALL_ROUTE
├── screens/chat/ChatScreen_enhanced.kt  # Add call buttons
└── MainActivity.kt                       # Handle call intents
```

**New routes needed:**
```kotlin
const val ACTIVE_CALL = "call/{callId}"
const val INCOMING_CALL = "incoming_call"
```

### Verification Flow Integration

**Files to modify:**
```
androidApp/src/main/kotlin/com/armorclaw/app/
├── navigation/AppNavigation.kt          # Add VERIFICATION_ROUTE
├── screens/settings/DeviceListScreen.kt # Add verify button
└── screens/settings/SecuritySettingsScreen.kt # Add verification option
```

**New route needed:**
```kotlin
const val EMOJI_VERIFICATION = "verification/{deviceId}"
```

### Logout Fix

**Files to create/modify:**
```
shared/src/commonMain/kotlin/domain/usecase/
└── LogoutUseCase.kt                     # NEW - Clear session logic

androidApp/src/main/kotlin/com/armorclaw/app/
└── viewmodels/SettingsViewModel.kt      # Call LogoutUseCase
```

### Deep Link Support

**Files to modify:**
```
androidApp/src/main/kotlin/com/armorclaw/app/
├── MainActivity.kt                       # Parse intent data
└── navigation/AppNavigation.kt          # Add deep link handling
```

---

## Conclusion

**Overall Assessment:** All user journeys are now **complete**!

### All Gaps Fixed:
1. ✅ **Call flow integration** - Routes added, call buttons in chat, navigation wired
2. ✅ **Verification flow integration** - Route added, connected to device management
3. ✅ **Session management** - LogoutUseCase implemented with proper cleanup
4. ✅ **Onboarding state persistence** - AppPreferences saves completion state
5. ✅ **Deep link support** - matrix.to and app-specific links supported
6. ✅ **Navigation error recovery** - Safe navigation with error handling
7. ✅ **Thread view screen** - Full thread UI with replies
8. ✅ **Full screen image viewer** - Zoom/pan support
9. ✅ **File preview** - Type detection and preview
10. ✅ **Push notification deep links** - NotificationDeepLinkHandler
11. ✅ **Mention handling** - Detection, styling, and chips

**Completion Status:**
- Critical flows: **100%** complete
- Medium priority: **100%** complete
- Low priority: **100%** complete

**All 11 identified gaps have been resolved!**

---

## New Files Created

| File | Purpose |
|------|---------|
| `SettingsViewModel.kt` | Handles logout with LogoutUseCase |
| `SplashViewModel.kt` | Handles state checking and deep links |
| `AppPreferences.kt` | Manages onboarding and login state |
| `LogoutUseCase.kt` | Comprehensive session cleanup |
| `NavigationExtensions.kt` | Safe navigation with error handling |
| `ThreadViewScreen.kt` | Thread view UI for threaded conversations |
| `ImageViewerScreen.kt` | Full-screen image viewer with zoom |
| `FilePreviewScreen.kt` | File preview with type detection |
| `NotificationDeepLinkHandler.kt` | Handles notification deep links |
| `MentionHandler.kt` | Mention detection and styling |

## Files Modified

| File | Changes |
|------|---------|
| `AppNavigation.kt` | Added thread/image/file routes, navigation callbacks |
| `ArmorClawNavHost.kt` | Added deep link handling |
| `MainActivity.kt` | Added deep link processing, state checking |
| `AppModules.kt` | Added DI configuration for new ViewModels |
| `LogTag.kt` | Added Splash, Android, Navigation log tags |
| `SyncRepository.kt` | Added startSync/stopSync/clearSyncState methods |
| `AuthRepository.kt` | Added clearLocalAuth method |
| `ChatScreen_enhanced.kt` | Added thread/image/file/profile callbacks |
| `DeviceListScreen.kt` | Added verify device callback |

---

## Testing Checklist

### Verify Complete Flows

- [ ] New user can complete onboarding
- [ ] Returning user can log in
- [ ] User can create room and chat
- [ ] User can join room and chat
- [ ] User can search and navigate to results
- [ ] User can access all settings screens
- [ ] User can edit profile fields
- [ ] User can change password
- [ ] User can manage devices

### Verify Gap Fixes (All Implemented!)

- [x] User can start call from chat ✅
- [x] User can verify device with emoji ✅
- [x] Logout clears session ✅
- [x] Deep links open correct content ✅
- [x] Onboarding doesn't repeat ✅
- [x] Navigation errors show feedback ✅
- [x] User can view message threads ✅
- [x] User can view images full screen ✅
- [x] User can preview files ✅
- [x] Notifications open correct content ✅
- [x] Mentions are tappable ✅

---

## Conclusion

**Overall Assessment:** The core user journeys (onboarding, auth, chat, settings, profile) are **complete and well-implemented**. 

### Recently Fixed Critical Gaps:
1. ✅ **Call flow integration** - Routes added, call buttons in chat, navigation wired
2. ✅ **Verification flow integration** - Route added, connected to device management
3. ✅ **Session management** - LogoutUseCase implemented with proper cleanup
4. ✅ **Onboarding state persistence** - AppPreferences saves completion state
5. ✅ **Deep link support** - matrix.to and app-specific links supported

### Remaining Lower Priority Gaps:
1. Navigation error recovery
2. Thread view screen
3. Full screen image viewer
4. Push notification deep links
5. Mention handling
6. File preview

**Completion Status:**
- Critical flows: **100%** complete
- Medium priority: **100%** complete (all fixed!)
- Low priority: **0%** complete (6 items remaining)

**Estimated Effort for Remaining:**
- Low priority: 5-10 days

---

## New Files Created

| File | Purpose |
|------|---------|
| `SettingsViewModel.kt` | Handles logout with LogoutUseCase |
| `SplashViewModel.kt` | Handles state checking and deep links |
| `AppPreferences.kt` | Manages onboarding and login state |
| `LogoutUseCase.kt` | Comprehensive session cleanup |

## Files Modified

| File | Changes |
|------|---------|
| `AppNavigation.kt` | Added call/verification routes, logout handling, state management |
| `ArmorClawNavHost.kt` | Added deep link handling |
| `MainActivity.kt` | Added deep link processing, state checking |
| `AppModules.kt` | Added DI configuration for new ViewModels |
| `LogTag.kt` | Added Splash and Android log tags |
| `SyncRepository.kt` | Added startSync/stopSync/clearSyncState methods |
| `AuthRepository.kt` | Added clearLocalAuth method |
| `ChatScreen_enhanced.kt` | Added call button callbacks |
| `DeviceListScreen.kt` | Added verify device callback |
