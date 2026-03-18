# ArmorClaw Components

> UI component catalog for ArmorClaw - Secure E2E Encrypted Chat Application

## 🧩 Component Hierarchy

```
Screen
└── Organism Components
    └── Molecule Components
        └── Atom Components
```

## ⚛️ Atomic Components

### Button
**Location:** `shared/ui/components/atom/Button.kt`

**Variants:**
- Primary
- Secondary
- Tertiary
- Outlined
- Text

**Features:**
- Ripple effect
- Disabled state
- Loading state
- Custom colors
- Icon support

**Example:**
```kotlin
Button(
    text = "Get Started",
    onClick = { /* Handle click */ },
    variant = ButtonVariant.Primary
)
```

---

### InputField
**Location:** `shared/ui/components/atom/InputField.kt`

**Variants:**
- Outlined
- Filled

**Features:**
- Label
- Placeholder
- Helper text
- Error state
- Leading icon
- Trailing icon
- Password visibility toggle

**Example:**
```kotlin
InputField(
    value = username,
    onValueChange = { username = it },
    label = "Username",
    placeholder = "Enter your username",
    isError = false
)
```

---

### Card
**Location:** `shared/ui/components/atom/Card.kt`

**Variants:**
- Elevated
- Filled
- Outlined

**Features:**
- Elevation
- Clickable
- Custom padding
- Custom shape

**Example:**
```kotlin
Card(
    variant = CardVariant.Elevated,
    onClick = { /* Handle click */ }
) {
    // Card content
}
```

---

### Badge
**Location:** `shared/ui/components/atom/Badge.kt`

**Variants:**
- Default
- Primary
- Secondary
- Error

**Features:**
- Text content
- Number content
- Custom colors
- Rounded shape

**Example:**
```kotlin
Badge(
    count = 5,
    variant = BadgeVariant.Default
)
```

---

### Icon
**Location:** `shared/ui/components/atom/Icon.kt`

**Variants:**
- Default
- Tinted

**Features:**
- ImageVector support
- Painter support
- Custom size
- Custom tint

**Example:**
```kotlin
Icon(
    imageVector = Icons.Default.Home,
    contentDescription = "Home",
    size = 24.dp,
    tint = AccentColor
)
```

---

## 🔗 Molecular Components

### MessageBubble
**Location:** `androidApp/screens/chat/components/MessageBubble.kt`

**Variants:**
- Sent
- Received
- System

**Features:**
- Message content
- Sender avatar
- Timestamp
- Message status (Sending, Sent, Delivered, Read, Failed)
- Reactions
- Attachments
- Reply preview
- Encryption indicator

**Example:**
```kotlin
MessageBubble(
    message = message,
    isSent = true,
    onLongClick = { /* Handle long click */ },
    onReactionClick = { /* Handle reaction */ }
)
```

---

### TypingIndicator
**Location:** `androidApp/screens/chat/components/TypingIndicator.kt`

**Variants:**
- Dots
- Text

**Features:**
- Animated dots
- Typing text ("X users are typing...")
- Custom colors

**Example:**
```kotlin
TypingIndicator(
    typingUsers = listOf("Alice", "Bob"),
    variant = TypingIndicatorVariant.Text
)
```

---

### EncryptionStatus
**Location:** `androidApp/screens/chat/components/EncryptionStatus.kt`

**Variants:**
- Encrypted
- Verifying
- Unverified

**Features:**
- Lock icon
- Status text
- Custom colors

**Example:**
```kotlin
EncryptionStatus(
    status = EncryptionStatusType.Encrypted,
    variant = EncryptionStatusVariant.Default
)
```

---

### ReplyPreview
**Location:** `androidApp/screens/chat/components/ReplyPreview.kt**

**Variants:**
- Single Message
- Forward Multiple

**Features:**
- Original message
- Sender name
- Cancel button
- Custom colors

**Example:**
```kotlin
ReplyPreview(
    originalMessage = message,
    variant = ReplyPreviewVariant.SingleMessage,
    onCancel = { /* Handle cancel */ }
)
```

---

### ChatSearchBar
**Location:** `androidApp/screens/chat/components/ChatSearchBar.kt`

**Variants:**
- Collapsed
- Expanded

**Features:**
- Search input
- Clear button
- Cancel button
- Real-time search
- Search results

**Example:**
```kotlin
ChatSearchBar(
    query = searchQuery,
    onQueryChange = { searchQuery = it },
    onClear = { searchQuery = "" },
    onCancel = { /* Handle cancel */ }
)
```

---

## 🧬 Organism Components

### MessageList
**Location:** `androidApp/screens/chat/components/MessageList.kt`

**States:**
- Loading
- Loaded
- Empty
- Error

**Features:**
- Vertical scrolling
- Pull-to-refresh
- Pagination
- Message grouping
- Scroll to bottom

**Example:**
```kotlin
MessageList(
    messages = messageList,
    state = MessageListState.Loaded,
    onRefresh = { /* Handle refresh */ },
    onScrollToBottom = { /* Handle scroll */ }
)
```

---

### RoomItemCard
**Location:** `androidApp/screens/home/RoomItemCard.kt`

**Variants:**
- Active
- Favorite
- Archived

**Features:**
- Room avatar
- Room name
- Last message
- Timestamp
- Unread badge
- Encryption indicator
- Clickable

**Example:**
```kotlin
RoomItemCard(
    room = roomItem,
    variant = RoomItemCardVariant.Active,
    onClick = { /* Handle click */ }
)
```

---

### ProfileAvatar
**Location:** `androidApp/screens/profile/ProfileScreen.kt`

**Features:**
- Large avatar
- Edit overlay
- Camera icon
- Clickable

**Example:**
```kotlin
ProfileAvatar(
    avatar = userAvatar,
    name = userName,
    isEditing = true,
    onClick = { /* Handle click */ }
)
```

---

## 🖼️ Screen Components

### SplashScreen
**Location:** `androidApp/screens/splash/SplashScreen.kt`

**Features:**
- App logo
- Fade-in animation
- Scale-up animation
- Loading indicator
- Auto-navigation

---

### WelcomeScreen
**Location:** `androidApp/screens/onboarding/WelcomeScreen.kt`

**Features:**
- Feature list
- Get Started button
- Skip button
- Animations

---

### SecurityExplanationScreen
**Location:** `androidApp/screens/onboarding/SecurityExplanationScreen.kt`

**Features:**
- Animated diagram
- 4-step security explanation
- Next/Back navigation
- Skip button

---

### ConnectServerScreen
**Location:** `androidApp/screens/onboarding/ConnectServerScreen.kt`

**Features:**
- Server URL input
- Connect button
- Demo option
- Validation

---

### PermissionsScreen
**Location:** `androidApp/screens/onboarding/PermissionsScreen.kt`

**Features:**
- Required permissions
- Optional permissions
- Progress tracking
- Grant button

---

### CompletionScreen
**Location:** `androidApp/screens/onboarding/CompletionScreen.kt`

**Features:**
- Celebration animation
- Confetti
- What's next list
- Start button

---

### LoginScreen
**Location:** `androidApp/screens/auth/LoginScreen.kt`

**Features:**
- Login form (username/email, password)
- Show/hide password
- Forgot password link
- Biometric login button
- Register link
- Terms/Privacy links

---

### HomeScreenFull
**Location:** `androidApp/screens/home/HomeScreenFull.kt`

**Features:**
- Room list (Favorites, Chats, Archived)
- Unread badges
- Room avatars
- Encryption indicators
- Expandable sections
- Search button
- Profile button
- Settings button
- Create room FAB
- Join room button

---

### ChatScreenEnhanced
**Location:** `androidApp/screens/chat/ChatScreenEnhanced.kt`

**Features:**
- Message list
- Input field
- Send button
- Voice input
- Attachment button
- Reply preview
- Search bar
- Encryption status
- Typing indicator

---

### ProfileScreen
**Location:** `androidApp/screens/profile/ProfileScreen.kt`

**Features:**
- Profile avatar
- Status indicator
- Profile information (name, email, status)
- Edit/Save mode
- Account options
- Privacy settings
- Logout button

---

### SettingsScreen
**Location:** `androidApp/screens/settings/SettingsScreen.kt`

**Features:**
- User profile section
- App settings (notifications, appearance, security)
- Privacy section
- About section
- Logout button
- Toggle switches

---

### RoomManagementScreen
**Location:** `androidApp/screens/room/RoomManagementScreen.kt`

**Features:**
- Tab navigation (Create/Join)
- Create room form
- Join room form
- Privacy settings
- Form validation

---

## 🔧 Platform Components

### BiometricAuth
**Location:** `androidApp/platform/BiometricAuthImpl.kt`

**Features:**
- Fingerprint authentication
- FaceID authentication
- Biometric prompt
- Secure data unlock

---

### SecureClipboard
**Location:** `androidApp/platform/SecureClipboardImpl.kt`

**Features:**
- Clipboard encryption
- Hash verification
- Auto-clear
- Secure paste

---

### NotificationManager
**Location:** `androidApp/platform/NotificationManagerImpl.kt`

**Features:**
- FCM integration
- Notification channels
- Grouped notifications
- Notification actions

---

### CertificatePinner
**Location:** `androidApp/platform/CertificatePinnerImpl.kt`

**Features:**
- OkHttp certificate pinning
- SHA-256 hash pinning
- Certificate validation

---

### CrashReporter
**Location:** `androidApp/platform/CrashReporterImpl.kt`

**Features:**
- Sentry integration
- Breadcrumbs
- Performance monitoring
- Error logging

---

### Analytics
**Location:** `androidApp/platform/AnalyticsImpl.kt`

**Features:**
- Event tracking
- Screen tracking
- User tracking

---

## 📊 Performance Components

### PerformanceProfiler
**Location:** `androidApp/performance/PerformanceProfiler.kt`

**Features:**
- Method execution tracing
- Memory allocation tracking
- Heap dumping
- Strict mode enforcement

---

### MemoryMonitor
**Location:** `androidApp/performance/MemoryMonitor.kt`

**Features:**
- Memory usage monitoring
- Memory pressure detection
- Native heap tracking
- Memory leak detection

---

## ♿ Accessibility Components

### AccessibilityConfig
**Location:** `androidApp/accessibility/AccessibilityConfig.kt`

**Features:**
- Screen reader detection
- High contrast detection
- Large text detection
- Font scale detection
- Reduced motion detection

---

### AccessibilityExtensions
**Location:** `androidApp/accessibility/AccessibilityExtensions.kt`

**Modifiers:**
- accessibilityContentDescription
- accessibilityHeading
- accessibilityStateDescription
- accessibilityValue
- accessibilityTestTag
- accessibilityTraversalIndex
- accessibilityHidden
- accessibilityClickable
- accessibilityToggleable
- accessibilitySelectable

---

## 🚀 Navigation Components

### AppNavigation
**Location:** `androidApp/navigation/AppNavigation.kt`

**Routes:**
- splash
- welcome
- security
- connect
- permissions
- completion
- login
- home
- chat/{roomId}
- profile
- settings
- room_management

**Features:**
- Animated transitions
- Route parameters
- Pop-up-to handling
- Deep linking

---

## 📦 Database Components

### AppDatabase
**Location:** `androidApp/data/database/AppDatabase.kt`

**Entities:**
- MessageEntity
- RoomEntity
- SyncQueueEntity

**DAOs:**
- MessageDao
- RoomDao
- SyncQueueDao

**Features:**
- SQLCipher encryption
- Type converters
- Migrations

---

## 🔄 Offline Sync Components

### OfflineQueue
**Location:** `androidApp/data/offline/OfflineQueue.kt`

**Features:**
- Operation enqueue
- Priority-based execution
- Retry logic
- Status tracking

---

### SyncEngine
**Location:** `androidApp/data/offline/SyncEngine.kt`

**Features:**
- State machine
- Operation execution
- Conflict detection
- Real-time sync status

---

### ConflictResolver
**Location:** `androidApp/data/offline/ConflictResolver.kt`

**Features:**
- Conflict detection
- Resolution strategies
- Message merging

---

### BackgroundSyncWorker
**Location:** `androidApp/data/offline/BackgroundSyncWorker.kt`

**Features:**
- WorkManager integration
- Network constraints
- Periodic sync
- Sync status tracking

---

### MessageExpirationManager
**Location:** `androidApp/data/offline/MessageExpirationManager.kt`

**Features:**
- Message expiration
- Auto-expiration checker
- Expiration status tracking

---

## ⚙️ Release Components

### ReleaseConfig
**Location:** `androidApp/release/ReleaseConfig.kt`

**Features:**
- Build type detection
- Release channel detection
- Feature flag management
- Configuration logging

---

*For detailed implementation, see source code.*
