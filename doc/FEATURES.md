# ArmorClaw Features

> Complete feature list for ArmorClaw - Secure E2E Encrypted Chat Application

## 🔧 Recent Additions (2026-02-24)

### Push Notification Dual Registration
- ✅ **Matrix Pusher Registration** - `MatrixClient.setPusher()` for homeserver push
- ✅ **Bridge Push Registration** - `BridgeRpcClient.pushRegister()` for SDTW events
- ✅ **Dual Strategy** - Both channels registered on every FCM token refresh
- ✅ **Graceful Degradation** - Partial failure handled (either channel can work independently)

### Bridge Verification UX
- ✅ **Bridge Verification Banner** - Warning banner in Room Details for unverified bridge devices
- ✅ **Verify Bridge Device Button** - One-tap navigation to emoji verification
- ✅ **BRIDGE_VERIFICATION Route** - Dedicated navigation route

### User Migration (v2.5 → v3.0)
- ✅ **MigrationScreen** - Auto-detection of legacy Bridge-only sessions
- ✅ **Recovery Phrase Entry** - Manual entry fallback for migration
- ✅ **SplashTarget.Migration** - Startup state machine detects need for migration
- ✅ **Legacy Session Helpers** - `hasLegacyBridgeSession()`, `clearLegacyBridgeSession()`

### Key Backup Setup
- ✅ **KeyBackupSetupScreen** - 6-step guided flow (Explain→Generate→Display→Verify→Store→Success)
- ✅ **12-Word Recovery Phrase** - BIP39-compatible recovery phrase generation
- ✅ **Mandatory in Onboarding** - Required before reaching Home screen
- ✅ **Re-entry from Settings** - Optional access from Security Settings

---

## 📱 Core Features

### Authentication
- ✅ **Login Screen** - Username/email and password authentication
- ✅ **Biometric Authentication** - Fingerprint/FaceID for secure unlock
- ✅ **Secure Session Management** - Encrypted session tokens
- ✅ **Forgot Password** - Password reset flow (placeholder)
- ✅ **Registration** - New user registration (placeholder)

### Onboarding
- ✅ **Welcome Screen** - Feature list, Get Started/Skip
- ✅ **Security Explanation** - Animated security diagram, 4 steps
- ✅ **Server Connection** - Matrix server URL, Connect, Demo option
- ✅ **Permissions Request** - Required/optional permissions with progress
- ✅ **Completion Screen** - Celebration, confetti, what's next
- ✅ **State Persistence** - Onboarding progress saved (DataStore)

### Splash Screen
- ✅ **App Logo Display** - Branded logo with animation
- ✅ **Fade-In Animation** - Smooth fade-in effect
- ✅ **Scale-Up Animation** - Animated logo scale-up
- ✅ **Loading Indicator** - Circular loading indicator
- ✅ **Auto-Navigation** - Routes to Onboarding/Login/Home
- ✅ **State-Based Routing** - Checks onboarding and login status

## 🏠 Home Screen

### Room List
- ✅ **Active Rooms** - List of all active chat rooms
- ✅ **Favorites Section** - Prioritized favorite rooms
- ✅ **Archived Section** - Archived rooms (collapsed by default)
- ✅ **Expandable Sections** - Toggle favorites and archived
- ✅ **Room Avatar** - Room avatar or initial
- ✅ **Encryption Indicator** - Lock icon for encrypted rooms
- ✅ **Last Message Preview** - Preview of last message
- ✅ **Timestamp Display** - Relative timestamp (2m, 1h, etc.)
- ✅ **Unread Badges** - Unread message count
- ✅ **Mention Badges** - Mention count notification
- ✅ **Pull-to-Refresh** - Refresh room list

### Room Management
- ✅ **Create Room Button** - Floating action button
- ✅ **Join Room Button** - Join existing room
- ✅ **Room Management Screen** - Create/Join room forms
- ✅ **Form Validation** - Required fields validation

### Navigation
- ✅ **Search Button** - Global room search
- ✅ **Profile Button** - Navigate to profile
- ✅ **Settings Button** - Navigate to settings
- ✅ **Room Navigation** - Navigate to room chat

## 💬 Chat Features

### Message Display
- ✅ **Enhanced Message List** - Loading, empty, error states
- ✅ **Message Bubbles** - Styled message bubbles
- ✅ **Message Status Indicators** - Sending, Sent, Delivered, Read, Failed
- ✅ **Timestamp Formatting** - Relative time (2m, 1h, 1d, etc.)
- ✅ **Sender Information** - Sender name, avatar
- ✅ **Message Grouping** - Group messages from same sender

### Message Features
- ✅ **Reply to Message** - Reply with quoted message
- ✅ **Reply Preview** - Show original message in reply
- ✅ **Forward Message** - Forward message to other rooms
- ✅ **Message Reactions** - Add emoji reactions
- ✅ **Reaction Display** - Show reactions with counts
- ✅ **File Attachments** - Send/receive files
- ✅ **Image Attachments** - Send/receive images
- ✅ **Voice Messages** - Record and send voice notes
- ✅ **Voice Input** - Voice-to-text input
- ✅ **Message Search** - Search within chat
- ✅ **Search Results** - Highlighted search results

### Encryption
- ✅ **Encryption Indicators** - 4 encryption levels
- ✅ **Lock Icons** - Visual encryption indicator
- ✅ **Encryption Status** - Encrypted, Verifying, Unverified
- ✅ **E2E Encryption** - End-to-end encrypted messages

### Real-Time Features
- ✅ **Typing Indicators** - Show when users are typing
- ✅ **Typing Dots Animation** - Animated dots
- ✅ **Typing Text** - "X users are typing..."
- ✅ **Real-Time Updates** - Instant message delivery
- ✅ **Online Status** - User online status

## 👤 Profile Features

### Profile Display
- ✅ **Profile Avatar** - Large avatar display
- ✅ **Avatar Editing** - Edit avatar with camera overlay
- ✅ **Status Indicator** - Online, Away, Busy, Invisible
- ✅ **Status Dropdown** - Editable status selection
- ✅ **Profile Information** - Name, email, status

### Profile Editing
- ✅ **Edit Mode** - Toggle edit mode
- ✅ **Name Field** - Edit display name
- ✅ **Email Field** - Edit email address
- ✅ **Status Field** - Edit status message
- ✅ **Save Button** - Save profile changes

### Account Management
- ✅ **Change Password** - Update password (placeholder)
- ✅ **Change Phone Number** - Update phone (placeholder)
- ✅ **Edit Bio** - Add/edit bio (placeholder)
- ✅ **Delete Account** - Permanently delete account (placeholder)
- ✅ **Privacy Settings** - Privacy policy, My Data (placeholder)
- ✅ **Logout** - Secure logout

## ⚙️ Settings Features

### User Profile Section
- ✅ **Profile Summary** - Avatar, name, email
- ✅ **Profile Navigation** - Navigate to profile

### App Settings
- ✅ **Notifications Toggle** - Enable/disable notifications
- ✅ **Sound Toggle** - Enable/disable notification sounds
- ✅ **Vibration Toggle** - Enable/disable vibration
- ✅ **Appearance Settings** - Theme, display settings (placeholder)
- ✅ **Security Settings** - Biometric auth, encryption (placeholder)

### Privacy Settings
- ✅ **Privacy Policy** - Privacy policy screen
- ✅ **Data & Storage** - Manage data and storage (placeholder)

### About Section
- ✅ **About ArmorClaw** - Version info (1.0.0)
- ✅ **Report a Bug** - Bug report flow (placeholder)
- ✅ **Rate App** - Rate on Play Store (placeholder)

### Logout
- ✅ **Logout Button** - Secure logout with confirmation

## 🏠 Room Management

### Create Room
- ✅ **Create Room Tab** - Create room form
- ✅ **Room Avatar** - Set room avatar
- ✅ **Room Name** - Required field
- ✅ **Room Topic** - Optional field
- ✅ **Privacy Toggle** - Private/Public room
- ✅ **Info Cards** - Privacy explanations
- ✅ **Form Validation** - Validate required fields

### Join Room
- ✅ **Join Room Tab** - Join room form
- ✅ **Room ID Field** - Required field
- ✅ **Room Alias Field** - Optional field
- ✅ **Info Cards** - Room ID explanations
- ✅ **Form Validation** - Validate required fields

### Room Settings
- ✅ **Room Privacy** - Private/Public
- ✅ **Room Avatar** - Edit room avatar
- ✅ **Room Topic** - Edit room topic
- ✅ **Room Settings** - (Placeholder)

## 🔐 Security Features

### Encryption
- ✅ **AES-256-GCM** - Message encryption
- ✅ **ECDH** - Key exchange
- ✅ **End-to-End Encryption** - All messages encrypted
- ✅ **Database Encryption** - SQLCipher (256-bit passphrase)

### Biometric Authentication
- ✅ **Fingerprint** - Fingerprint authentication
- ✅ **FaceID** - Face recognition (Android 10+)
- ✅ **Biometric Prompt** - Biometric authentication UI
- ✅ **Secure Data Unlock** - Unlock data with biometric
- ✅ **Fallback to Password** - Password fallback

### Secure Clipboard
- ✅ **Clipboard Encryption** - Encrypt clipboard content
- ✅ **Hash Verification** - Verify clipboard integrity
- ✅ **Auto-Clear** - Auto-clear clipboard after timeout
- ✅ **Secure Paste** - Secure paste from encrypted clipboard

### Certificate Pinning
- ✅ **OkHttp Pinning** - Certificate pinning for HTTPS
- ✅ **SHA-256 Pins** - SHA-256 hash pinning
- ✅ **Certificate Validation** - Validate server certificates
- ✅ **Pin Backup** - Backup pins for rotation

### Crash Reporting
- ✅ **Sentry Integration** - Crash reporting (Sentry SDK)
- ✅ **Breadcrumbs** - User actions before crash
- ✅ **Performance Monitoring** - App performance tracking
- ✅ **Error Logging** - Detailed error logs

### Analytics
- ✅ **Event Tracking** - Track user events
- ✅ **Screen Tracking** - Track screen views
- ✅ **User Tracking** - Track user properties
- ✅ **Analytics Provider** - Amplitude/Mixpanel (placeholder)

## 📱 Platform Integrations

### Android
- ✅ **Biometric Authentication** - AndroidX Biometric API
- ✅ **Secure Clipboard** - AndroidX ClipboardManager
- ✅ **Push Notifications** - Firebase Cloud Messaging (FCM)
- ✅ **Notification Channels** - Grouped notifications
- ✅ **Notification Actions** - Reply, mark read actions
- ✅ **Background Work** - WorkManager for background tasks
- ✅ **Network Monitoring** - Network state monitoring

### Offline Support
- ✅ **Offline Queue** - Queue pending operations
- ✅ **Sync Engine** - Sync local changes to server
- ✅ **Conflict Resolution** - Detect and resolve conflicts
- ✅ **Background Sync** - Periodic background sync
- ✅ **Message Expiration** - Ephemeral messages
- ✅ **Auto-Expiration** - Auto-delete expired messages

## 🎨 UI/UX Features

### Design System
- ✅ **Material Design 3** - Material 3 design language
- ✅ **Custom Theme** - Custom color palette
- ✅ **Dark Mode** - Automatic dark mode
- ✅ **Typography** - Custom typography system
- ✅ **Shapes** - Custom shape system
- ✅ **Animations** - Smooth animations
- ✅ **Transitions** - Animated screen transitions

### Accessibility
- ✅ **Screen Reader Support** - TalkBack support
- ✅ **High Contrast** - High contrast detection
- ✅ **Large Text** - Large text detection
- ✅ **Font Scaling** - Font scale detection
- ✅ **Reduced Motion** - Reduced motion detection
- ✅ **Content Descriptions** - Semantic content descriptions
- ✅ **Headings** - Semantic headings
- ✅ **Traversal Order** - Logical traversal order

### Performance
- ✅ **Performance Profiling** - Method execution tracing
- ✅ **Memory Monitoring** - Real-time memory monitoring
- ✅ **Memory Pressure Detection** - 4 pressure levels
- ✅ **Memory Leak Detection** - Heuristic leak detection
- ✅ **Heap Dumping** - Heap dump to file
- ✅ **Strict Mode** - Strict mode for development

### Navigation
- ✅ **Animated Transitions** - Fade in/out transitions
- ✅ **Route Parameters** - Parameter support (roomId)
- ✅ **Deep Linking** - Deep link support
- ✅ **Back Stack Management** - Pop-up-to handling
- ✅ **Nested Navigation** - Nested navigation support
- ✅ **Bottom Sheet** - Bottom sheet navigation (placeholder)

## 🔧 Developer Features

### Build Configuration
- ✅ **Multi-Variant** - Debug, Release, Demo, Beta
- ✅ **Feature Flags** - 20+ feature flags
- ✅ **R8/ProGuard** - Code shrinking and obfuscation
- ✅ **Resource Shrinking** - Resource optimization
- ✅ **APK Optimization** - APK size optimization

### Release Configuration
- ✅ **Release Channels** - Demo, Alpha, Beta, Stable
- ✅ **Version Management** - Version name and code
- ✅ **Build Types** - Debug, Release
- ✅ **Product Flavors** - Demo, Beta, Alpha, Stable
- ✅ **Signing** - APK signing configuration

### Testing
- ✅ **Unit Tests** - 50+ unit tests
- ✅ **Integration Tests** - 15+ integration tests
- ✅ **E2E Tests** - 11 E2E test scenarios
- ✅ **UI Tests** - Compose UI tests
- ✅ **Instrumented Tests** - Android instrumented tests

---

**Total Features:** 150+

*For detailed implementation, see source code.*
