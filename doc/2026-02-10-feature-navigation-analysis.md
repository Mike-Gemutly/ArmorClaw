# ArmorClaw - Feature & Navigation Analysis

> Complete feature map with navigation flows and gap analysis

> **Analysis Date:** 2026-02-10
> **Status:** ✅ Complete

---

## 📱 All Screens (19)

### Onboarding Flow (5 screens)
1. **SplashScreen** - App launch
2. **WelcomeScreen** - Feature overview
3. **SecurityExplanationScreen** - Security explanation (4 steps)
4. **ConnectServerScreen** - Connect to server
5. **PermissionsScreen** - Grant permissions
6. **CompletionScreen** - Onboarding complete

### Authentication Flow (1 screen)
7. **LoginScreen** - Login, biometric, forgot password, register

### Main App Flow (12 screens)
8. **HomeScreenFull** - Room list, categories, actions
9. **ChatScreenEnhanced** - Enhanced chat with all features
10. **ProfileScreen** - Profile management, account options
11. **SettingsScreen** - App settings, privacy, about
12. **RoomManagementScreen** - Create/Join rooms
13. **(Placeholder) SearchScreen** - Global room/message search
14. **(Placeholder) SecuritySettingsScreen** - Security settings
15. **(Placeholder) NotificationSettingsScreen** - Notification settings
16. **(Placeholder) AppearanceSettingsScreen** - Appearance settings
17. **(Placeholder) PrivacyPolicyScreen** - Privacy policy
18. **(Placeholder) MyDataScreen** - Download data
19. **(Placeholder) AboutScreen** - About ArmorClaw

---

## 🗺️ Complete Navigation Flows

### User Journey 1: First-Time User (Onboarding)

**Flow:**
```
App Launch
    ↓
SplashScreen (1.5s)
    ↓ (if not onboarded)
WelcomeScreen
    ↓ (Get Started)
SecurityExplanationScreen (Step 1/4)
    ↓ (Next)
SecurityExplanationScreen (Step 2/4)
    ↓ (Next)
SecurityExplanationScreen (Step 3/4)
    ↓ (Next)
SecurityExplanationScreen (Step 4/4)
    ↓ (Next)
ConnectServerScreen
    ↓ (Connect OR Use Demo)
PermissionsScreen
    ↓ (Grant Permissions)
CompletionScreen
    ↓ (Start Chatting)
HomeScreenFull
```

**User Actions:**
- Tap "Get Started" on WelcomeScreen
- Tap "Next" on SecurityExplanationScreen (x4)
- Tap "Connect" OR "Use Demo" on ConnectServerScreen
- Tap "Grant" on PermissionsScreen
- Tap "Start Chatting" on CompletionScreen

**Alternatives:**
- Tap "Skip" on WelcomeScreen → Goes to HomeScreen (skips onboarding)
- Tap "Skip" on SecurityExplanationScreen → Goes to ConnectServerScreen
- Tap "Back" on any screen → Goes to previous screen

---

### User Journey 2: Returning User (Already Onboarded, Logged In)

**Flow:**
```
App Launch
    ↓
SplashScreen (1.5s)
    ↓ (if onboarded + logged in)
HomeScreenFull
```

**User Actions:**
- None (auto-navigates)

---

### User Journey 3: Returning User (Already Onboarded, Not Logged In)

**Flow:**
```
App Launch
    ↓
SplashScreen (1.5s)
    ↓ (if onboarded + not logged in)
Biometric Prompt (if enabled)
    ↓ (Success)
HomeScreenFull
    ↓ OR (Failure/Cancel)
LoginScreen
    ↓ (Log In)
HomeScreenFull
    ↓ OR (Log In with Biometrics)
HomeScreenFull
```

**User Actions:**
- Authenticate with biometrics (if enabled)
- OR enter username/email and password
- OR tap "Log In with Biometrics" button
- OR tap "Forgot Password?" link
- OR tap "Register" link

**Alternatives:**
- Tap "Forgot Password?" → Goes to ForgotPasswordScreen (placeholder)
- Tap "Register" → Goes to RegistrationScreen (placeholder)

---

### User Journey 4: Home → Chat

**Flow:**
```
HomeScreenFull
    ↓ (Tap Room)
ChatScreenEnhanced (with roomId)
```

**User Actions:**
- Tap any room card in room list

**Room Card Types:**
- Favorite rooms (expandable section)
- Active rooms (main list)
- Archived rooms (expandable section)

---

### User Journey 5: Chat → Send Message

**Flow:**
```
ChatScreenEnhanced
    ↓ (Type Message)
ChatScreenEnhanced (Input Field)
    ↓ (Tap Send Button)
ChatScreenEnhanced (Message Sent)
```

**User Actions:**
- Type message in input field
- Tap send button (paper plane icon)

**Message Status Flow:**
- Sending (clock icon) → Sent (check icon) → Delivered (double check) → Read (blue double check)
- OR Failed (red exclamation) → Tap to retry

---

### User Journey 6: Chat → Reply to Message

**Flow:**
```
ChatScreenEnhanced
    ↓ (Long Press on Message)
Reply Preview Bar (appears above input)
    ↓ (Type Reply)
ChatScreenEnhanced (Input Field)
    ↓ (Tap Send Button)
ChatScreenEnhanced (Reply Sent)
```

**User Actions:**
- Long press on message
- See reply preview bar above input
- Type reply in input field
- Tap send button

**Reply Preview Bar Features:**
- Original message quoted
- Sender name shown
- Cancel button to close

---

### User Journey 7: Chat → Add Reaction

**Flow:**
```
ChatScreenEnhanced
    ↓ (Long Press on Message)
Reaction Menu (appears)
    ↓ (Select Emoji)
ChatScreenEnhanced (Reaction Added)
```

**User Actions:**
- Long press on message
- See reaction menu
- Tap emoji to add reaction
- Tap reaction again to remove

**Reaction Menu Features:**
- Emoji picker
- Recent emojis
- Tap to add/remove

---

### User Journey 8: Chat → Attach File

**Flow:**
```
ChatScreenEnhanced
    ↓ (Tap Attachment Icon)
File Picker (System)
    ↓ (Select File)
ChatScreenEnhanced (File Attached)
    ↓ (Tap Send Button)
ChatScreenEnhanced (File Sent)
```

**User Actions:**
- Tap attachment icon (paper clip)
- Select file from system picker
- Add caption (optional)
- Tap send button

---

### User Journey 9: Chat → Send Image

**Flow:**
```
ChatScreenEnhanced
    ↓ (Tap Attachment Icon)
Image Picker (Camera/Gallery)
    ↓ (Select/Take Image)
ChatScreenEnhanced (Image Attached)
    ↓ (Add Caption - Optional)
    ↓ (Tap Send Button)
ChatScreenEnhanced (Image Sent)
```

**User Actions:**
- Tap attachment icon (paper clip)
- Select "Camera" or "Gallery"
- Take photo or select from gallery
- Crop image (optional)
- Add caption (optional)
- Tap send button

---

### User Journey 10: Chat → Record Voice Message

**Flow:**
```
ChatScreenEnhanced
    ↓ (Tap & Hold Microphone Icon)
Voice Recording (Start)
    ↓ (Record Voice)
    ↓ (Release Microphone Icon)
ChatScreenEnhanced (Voice Message Sent)
```

**User Actions:**
- Tap and hold microphone icon
- Speak into microphone
- Release microphone icon to send
- OR slide away to cancel

---

### User Journey 11: Chat → Search Messages

**Flow:**
```
ChatScreenEnhanced
    ↓ (Tap Search Icon in Top Bar)
ChatSearchBar (Expands)
    ↓ (Type Search Query)
ChatScreenEnhanced (Search Results Highlighted)
    ↓ (Tap Search Result)
ChatScreenEnhanced (Scrolls to Message)
```

**User Actions:**
- Tap search icon in top bar
- Type search query in search bar
- See search results highlighted
- Tap result to jump to message
- Tap cancel to close search

---

### User Journey 12: Chat → View Encryption Status

**Flow:**
```
ChatScreenEnhanced
    ↓ (View Lock Icon in Top Bar)
ChatScreenEnhanced (Encryption Status Displayed)
```

**User Actions:**
- View lock icon in top bar
- See encryption status (Encrypted, Verifying, Unverified)
- No action needed (display only)

---

### User Journey 13: Chat → View Typing Indicators

**Flow:**
```
ChatScreenEnhanced
    ↓ (Typing Indicators Appear)
ChatScreenEnhanced (Typing Users Displayed)
```

**User Actions:**
- View typing indicators
- See "X users are typing..."
- See animated dots
- No action needed (display only)

---

### User Journey 14: Chat → Back to Home

**Flow:**
```
ChatScreenEnhanced
    ↓ (Tap Back Button in Top Bar)
HomeScreenFull
```

**User Actions:**
- Tap back button in top bar
- Returns to home screen
- Message position saved

---

### User Journey 15: Home → Create Room

**Flow:**
```
HomeScreenFull
    ↓ (Tap FAB + Button)
RoomManagementScreen (Create Room Tab)
    ↓ (Enter Room Details)
    ↓ (Tap Create Room Button)
HomeScreenFull (Room Created)
```

**User Actions:**
- Tap FAB (+) button
- See RoomManagementScreen (Create Room tab)
- Enter room name (required)
- Enter room topic (optional)
- Set privacy (Private/Public)
- Add room avatar (optional)
- Tap "Create Room" button

---

### User Journey 16: Home → Join Room

**Flow:**
```
HomeScreenFull
    ↓ (Tap Join Room Button)
RoomManagementScreen (Join Room Tab)
    ↓ (Enter Room ID/Alias)
    ↓ (Tap Join Room Button)
HomeScreenFull (Room Joined)
```

**User Actions:**
- Tap "Join a room" button
- See RoomManagementScreen (Join Room tab)
- Enter room ID (required)
- Enter room alias (optional)
- Tap "Join Room" button

---

### User Journey 17: Home → Profile

**Flow:**
```
HomeScreenFull
    ↓ (Tap Profile Icon in Top Bar)
ProfileScreen
```

**User Actions:**
- Tap profile icon (account circle) in top bar

---

### User Journey 18: Profile → View Profile

**Flow:**
```
ProfileScreen
    ↓ (View Profile)
ProfileScreen (Profile Displayed)
```

**User Actions:**
- View profile avatar
- View status indicator
- View profile information (name, email, status)

---

### User Journey 19: Profile → Edit Profile

**Flow:**
```
ProfileScreen
    ↓ (Tap Edit Icon in Top Bar)
ProfileScreen (Edit Mode)
    ↓ (Edit Profile)
    ↓ (Tap Save Button)
ProfileScreen (Profile Updated)
```

**User Actions:**
- Tap edit icon (pencil) in top bar
- Edit profile fields (Name, Email, Status)
- Tap save button (top right)

---

### User Journey 20: Profile → Change Avatar

**Flow:**
```
ProfileScreen
    ↓ (Tap Edit Icon)
    ↓ (Tap Camera Icon on Avatar)
Image Picker (Camera/Gallery)
    ↓ (Select/Take Image)
ProfileScreen (Avatar Updated)
    ↓ (Tap Save Button)
ProfileScreen (Profile Updated)
```

**User Actions:**
- Tap edit icon (pencil) in top bar
- Tap camera icon on avatar
- Select "Camera" or "Gallery"
- Take photo or select from gallery
- Crop image (optional)
- Tap save button

---

### User Journey 21: Profile → Change Status

**Flow:**
```
ProfileScreen
    ↓ (Tap Status Dropdown)
Status Dropdown (Appears)
    ↓ (Select New Status)
ProfileScreen (Status Updated)
    ↓ (Tap Save Button)
ProfileScreen (Profile Updated)
```

**User Actions:**
- Tap status dropdown
- Select new status (Online, Away, Busy, Invisible)
- Tap save button

**Status Options:**
- Online (Green)
- Away (Orange)
- Busy (Red)
- Invisible (Grey)

---

### User Journey 22: Profile → Change Password

**Flow:**
```
ProfileScreen
    ↓ (Tap "Change Password")
ChangePasswordScreen (Placeholder)
    ↓ (Enter Current Password)
    ↓ (Enter New Password)
    ↓ (Confirm New Password)
    ↓ (Tap Update Button)
ProfileScreen (Password Updated)
```

**User Actions:**
- Tap "Change Password" in Account Options
- Enter current password
- Enter new password
- Confirm new password
- Tap "Update" button

**Note:** ChangePasswordScreen is placeholder

---

### User Journey 23: Profile → Change Phone Number

**Flow:**
```
ProfileScreen
    ↓ (Tap "Change Phone Number")
ChangePhoneNumberScreen (Placeholder)
    ↓ (Enter New Phone Number)
    ↓ (Verify Phone Number)
ProfileScreen (Phone Number Updated)
```

**User Actions:**
- Tap "Change Phone Number" in Account Options
- Enter new phone number
- Verify phone number (via SMS)
- Phone number updated

**Note:** ChangePhoneNumberScreen is placeholder

---

### User Journey 24: Profile → Edit Bio

**Flow:**
```
ProfileScreen
    ↓ (Tap "Edit Bio")
EditBioScreen (Placeholder)
    ↓ (Enter Bio)
ProfileScreen (Bio Updated)
```

**User Actions:**
- Tap "Edit Bio" in Account Options
- Enter bio (optional)
- Bio updated

**Note:** EditBioScreen is placeholder

---

### User Journey 25: Profile → Delete Account

**Flow:**
```
ProfileScreen
    ↓ (Tap "Delete Account")
DeleteAccountScreen (Placeholder)
    ↓ (Read Warning)
    ↓ (Confirm Deletion by Entering Password)
ProfileScreen (Account Deleted)
    ↓
LoginScreen
```

**User Actions:**
- Tap "Delete Account" in Account Options
- Read warning (deletion is permanent)
- Confirm deletion by entering password
- Account deleted with all data
- Navigates to LoginScreen

**Note:** DeleteAccountScreen is placeholder

---

### User Journey 26: Profile → Settings

**Flow:**
```
ProfileScreen
    ↓ (Tap Settings Icon in Top Bar)
SettingsScreen
```

**User Actions:**
- Tap settings icon (gear) in top bar

---

### User Journey 27: Home → Settings

**Flow:**
```
HomeScreenFull
    ↓ (Tap Settings Icon in Top Bar)
SettingsScreen
```

**User Actions:**
- Tap settings icon (gear) in top bar

---

### User Journey 28: Settings → Notifications

**Flow:**
```
SettingsScreen
    ↓ (Tap "Notifications")
NotificationSettingsScreen (Placeholder)
    ↓ (Toggle Notifications)
    ↓ (Toggle Sound)
    ↓ (Toggle Vibration)
SettingsScreen (Settings Updated)
```

**User Actions:**
- Tap "Notifications" in App Settings
- Toggle notifications (ON/OFF)
- Toggle sound (ON/OFF)
- Toggle vibration (ON/OFF)

**Note:** NotificationSettingsScreen is placeholder

---

### User Journey 29: Settings → Appearance

**Flow:**
```
SettingsScreen
    ↓ (Tap "Appearance")
AppearanceSettingsScreen (Placeholder)
    ↓ (Select Theme)
    ↓ (Adjust Font Size)
    ↓ (Adjust Display Settings)
SettingsScreen (Settings Updated)
```

**User Actions:**
- Tap "Appearance" in App Settings
- Select theme (Light, Dark, Auto)
- Adjust font size
- Adjust display settings

**Note:** AppearanceSettingsScreen is placeholder

---

### User Journey 30: Settings → Security

**Flow:**
```
SettingsScreen
    ↓ (Tap "Security")
SecuritySettingsScreen (Placeholder)
    ↓ (Toggle Biometric Auth)
    ↓ (View Encryption Settings)
    ↓ (Enable Two-Factor Auth)
SettingsScreen (Settings Updated)
```

**User Actions:**
- Tap "Security" in App Settings
- Toggle biometric auth (ON/OFF)
- View encryption settings
- Enable two-factor auth (placeholder)

**Note:** SecuritySettingsScreen is placeholder

---

### User Journey 31: Settings → Privacy Policy

**Flow:**
```
SettingsScreen
    ↓ (Tap "Privacy Policy" in Privacy Section)
PrivacyPolicyScreen (Placeholder)
    ↓ (Read Privacy Policy)
    ↓ (Tap Back)
SettingsScreen
```

**User Actions:**
- Tap "Privacy Policy" in Privacy section
- Read privacy policy
- Tap back to return

**Note:** PrivacyPolicyScreen is placeholder

---

### User Journey 32: Settings → My Data

**Flow:**
```
SettingsScreen
    ↓ (Tap "Data & Storage" in Privacy Section)
MyDataScreen (Placeholder)
    ↓ (Request Data Download)
    ↓ (Receive Data via Email)
SettingsScreen
```

**User Actions:**
- Tap "Data & Storage" in Privacy section
- Request data download
- Receive data via email (GDPR)

**Note:** MyDataScreen is placeholder

---

### User Journey 33: Settings → About

**Flow:**
```
SettingsScreen
    ↓ (Tap "About ArmorClaw")
AboutScreen (Placeholder)
    ↓ (View Version Info)
    ↓ (Tap Links)
SettingsScreen
```

**User Actions:**
- Tap "About ArmorClaw" in About section
- View version info
- Tap links (GitHub, Website, etc.)

**Note:** AboutScreen is placeholder

---

### User Journey 34: Settings → Report Bug

**Flow:**
```
SettingsScreen
    ↓ (Tap "Report a Bug")
ReportBugScreen (Placeholder)
    ↓ (Describe Bug)
    ↓ (Submit Bug Report)
SettingsScreen (Bug Report Submitted)
```

**User Actions:**
- Tap "Report a Bug" in About section
- Describe bug
- Submit bug report

**Note:** ReportBugScreen is placeholder

---

### User Journey 35: Settings → Rate App

**Flow:**
```
SettingsScreen
    ↓ (Tap "Rate App")
Play Store (External)
    ↓ (Rate App)
SettingsScreen (App Rated)
```

**User Actions:**
- Tap "Rate App" in About section
- Navigate to Play Store
- Rate app
- Return to app

---

### User Journey 36: Settings → Logout

**Flow:**
```
SettingsScreen
    ↓ (Tap "Log Out" Button)
Logout Confirmation (Dialog)
    ↓ (Confirm Logout)
LoginScreen
```

**User Actions:**
- Tap "Log Out" button (bottom)
- Confirm logout (dialog)
- Navigates to LoginScreen

---

### User Journey 37: Home → Search (Global)

**Flow:**
```
HomeScreenFull
    ↓ (Tap Search Icon in Top Bar)
SearchScreen (Placeholder)
    ↓ (Type Search Query)
    ↓ (View Search Results)
    ↓ (Tap Result)
ChatScreenEnhanced (OR HomeScreenFull)
```

**User Actions:**
- Tap search icon (magnifying glass) in top bar
- Type search query
- View search results (rooms, messages)
- Tap result to navigate
- Tap cancel to close search

**Note:** SearchScreen is placeholder

---

### User Journey 38: Login → Forgot Password

**Flow:**
```
LoginScreen
    ↓ (Tap "Forgot Password?")
ForgotPasswordScreen (Placeholder)
    ↓ (Enter Email)
    ↓ (Tap "Reset Password")
LoginScreen (Email Sent)
```

**User Actions:**
- Tap "Forgot Password?" link
- Enter email address
- Tap "Reset Password" button
- Receive reset email
- Return to LoginScreen

**Note:** ForgotPasswordScreen is placeholder

---

### User Journey 39: Login → Register

**Flow:**
```
LoginScreen
    ↓ (Tap "Register" Link)
RegistrationScreen (Placeholder)
    ↓ (Enter Username)
    ↓ (Enter Email)
    ↓ (Enter Password)
    ↓ (Confirm Password)
    ↓ (Tap "Register")
LoginScreen (Registration Successful)
    ↓ (Login)
HomeScreenFull
```

**User Actions:**
- Tap "Register" link
- Enter username
- Enter email
- Enter password
- Confirm password
- Tap "Register" button
- Login with new credentials

**Note:** RegistrationScreen is placeholder

---

### User Journey 40: Any Screen → Biometric Unlock

**Flow:**
```
(Any Screen)
    ↓ (App Backgrounded + Resumed)
Biometric Prompt (System)
    ↓ (Authenticate)
(Previous Screen - Unlocked)
    ↓ OR (Failure/Cancel)
LoginScreen
```

**User Actions:**
- Background app
- Resume app
- Authenticate with biometrics (if enabled)
- OR cancel and go to LoginScreen

---

## ❌ Identified Navigation Gaps (17 Gaps)

### High Priority Gaps (7)

1. **No ForgotPasswordScreen** - User cannot reset password
2. **No RegistrationScreen** - User cannot register new account
3. **No RoomDetailsScreen** - User cannot view room details, members, settings
4. **No ChangePasswordScreen** - User cannot change password
5. **No DeleteAccountScreen** - User cannot delete account
6. **No Logout Confirmation** - User might accidentally logout
7. **No Room Settings Screen** - User cannot configure room settings

### Medium Priority Gaps (6)

8. **No SearchScreen** - Global search not implemented
9. **No NotificationSettingsScreen** - Notification settings not implemented
10. **No AppearanceSettingsScreen** - Appearance settings not implemented
11. **No SecuritySettingsScreen** - Security settings not implemented
12. **No PrivacyPolicyScreen** - Privacy policy not implemented
13. **No MyDataScreen** - Data download not implemented

### Low Priority Gaps (4)

14. **No ChangePhoneNumberScreen** - Phone number change not implemented
15. **No EditBioScreen** - Bio edit not implemented
16. **No AboutScreen** - About screen not implemented
17. **No ReportBugScreen** - Bug report not implemented

---

## 📊 Navigation Gap Summary

| # | Gap | Priority | Status | Impact |
|---|------|-----------|---------|--------|
| 1 | No ForgotPasswordScreen | High | ❌ Missing | User cannot reset password |
| 2 | No RegistrationScreen | High | ❌ Missing | User cannot register |
| 3 | No RoomDetailsScreen | High | ❌ Missing | User cannot manage room |
| 4 | No ChangePasswordScreen | High | ❌ Missing | User cannot change password |
| 5 | No DeleteAccountScreen | High | ❌ Missing | User cannot delete account |
| 6 | No Logout Confirmation | High | ❌ Missing | User might logout accidentally |
| 7 | No Room Settings Screen | High | ❌ Missing | User cannot configure room |
| 8 | No SearchScreen | Medium | ❌ Missing | Global search not implemented |
| 9 | No NotificationSettingsScreen | Medium | ❌ Missing | Notification settings not implemented |
| 10 | No AppearanceSettingsScreen | Medium | ❌ Missing | Appearance settings not implemented |
| 11 | No SecuritySettingsScreen | Medium | ❌ Missing | Security settings not implemented |
| 12 | No PrivacyPolicyScreen | Medium | ❌ Missing | Privacy policy not implemented |
| 13 | No MyDataScreen | Medium | ❌ Missing | Data download not implemented |
| 14 | No ChangePhoneNumberScreen | Low | ❌ Missing | Phone number change not implemented |
| 15 | No EditBioScreen | Low | ❌ Missing | Bio edit not implemented |
| 16 | No AboutScreen | Low | ❌ Missing | About screen not implemented |
| 17 | No ReportBugScreen | Low | ❌ Missing | Bug report not implemented |

---

**Total Navigation Gaps:** 17
**High Priority:** 7
**Medium Priority:** 6
**Low Priority:** 4

---

*Next: Fix all navigation gaps*
