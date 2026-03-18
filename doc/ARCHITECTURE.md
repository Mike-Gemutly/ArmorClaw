# ArmorClaw Architecture

> Architecture overview for ArmorClaw - Secure E2E Encrypted Chat Application

## 📐 Architecture Overview

ArmorClaw follows a **modular, multiplatform architecture** with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                          Presentation Layer                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   Screens    │  │ Components   │  │  Navigation  │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────────┐
│                           Business Logic Layer                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │  ViewModels  │  │  Use Cases   │  │  Repositories│        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────────┐
│                            Data Layer                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   Database   │  │     API      │  │   Offline    │        │
│  │    (Room)    │  │    (Ktor)    │  │     Queue    │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────────┐
│                         Platform Layer                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   Biometric  │  │   Clipboard   │  │ Notifications│        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   Network    │  │   Crash      │  │   Analytics  │        │
│  │   Monitor    │  │   Reporter   │  │              │        │
│  └──────────────┘  └──────────────┘  └──────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

## 📦 Module Structure

```
ArmorClaw/
├── shared/                    # Kotlin Multiplatform shared module
│   ├── domain/               # Domain layer (business logic)
│   │   ├── model/           # Domain models
│   │   ├── repository/      # Repository interfaces
│   │   └── usecase/         # Use case interfaces
│   ├── platform/             # Platform interfaces (expect/actual)
│   └── ui/                  # Shared UI components
│       ├── theme/            # Design system
│       ├── components/       # UI components
│       │   ├── atom/        # Atomic components
│       │   └── molecule/    # Molecular components
│       └── base/             # Base classes
└── androidApp/              # Android application
    ├── screens/              # Compose screens
    │   ├── onboarding/      # Onboarding: Welcome, Security, Server, Permissions,
    │   │                    #   MigrationScreen (NEW), KeyBackupSetupScreen (NEW)
    │   ├── home/            # Home screen
    │   ├── chat/            # Chat screens
    │   ├── profile/         # Profile screen
    │   ├── settings/        # Settings screen
    │   ├── auth/            # Authentication screens
    │   ├── room/            # Room management (incl. BridgeVerificationBanner)
    │   └── splash/          # Splash screen (incl. migration detection)
    ├── components/           # Android-specific components
    ├── viewmodels/           # ViewModels
    ├── navigation/           # Navigation configuration
    ├── data/                 # Data layer
    │   ├── persistence/      # DataStore persistence
    │   ├── database/        # Room database (SQLCipher)
    │   └── offline/         # Offline sync
    ├── platform/             # Platform implementations
    ├── performance/          # Performance monitoring
    ├── accessibility/        # Accessibility features
    └── release/             # Release configuration
```

## 🧱 Architecture Patterns

### Clean Architecture
- **Domain Layer**: Business logic, models, use cases
- **Data Layer**: Data sources, repositories, offline sync
- **Presentation Layer**: UI, ViewModels, navigation

### MVVM (Model-View-ViewModel)
- **Model**: Domain models, repositories
- **View**: Compose UI components
- **ViewModel**: State management, business logic

### Repository Pattern
- **Repository Interfaces**: Defined in shared module
- **Repository Implementations**: (To be implemented)

### Use Case Pattern
- **Use Case Interfaces**: Defined in shared module
- **Use Case Implementations**: (To be implemented)

## 🔄 Data Flow

### User Interaction Flow
```
User → UI → ViewModel → Use Case → Repository → Data Source
                  ↓
               StateFlow → UI
```

### Offline Sync Flow
```
User → Offline Queue → Sync Engine → Repository → API
                  ↓
               Database → StateFlow → UI
```

### Conflict Resolution Flow
```
Server → Conflict Resolver → Resolution Strategy → Repository → Database
                                      ↓
                                   StateFlow → UI
```

## 🔐 Security Architecture

### Encryption Layers
```
┌─────────────────────────────────────────────────────────────────┐
│                  Application Layer Encryption                    │
│  • Message Encryption (AES-256-GCM)                          │
│  • Key Exchange (ECDH)                                      │
└─────────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────────┐
│                    Data Layer Encryption                       │
│  • Database Encryption (SQLCipher, 256-bit passphrase)         │
│  • Clipboard Encryption (AES-256-GCM)                        │
└─────────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────────┐
│                  Platform Layer Encryption                      │
│  • Key Storage (AndroidKeyStore)                              │
│  • Biometric Authentication (AndroidX Biometric)                │
└─────────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────────┐
│                    Network Layer Encryption                     │
│  • TLS/SSL (HTTPS)                                           │
│  • Certificate Pinning (SHA-256)                              │
└─────────────────────────────────────────────────────────────────┘
```

## 🗄️ Database Architecture

### Room Database (SQLCipher)
```
AppDatabase
├── MessageEntity        # Message table
├── RoomEntity           # Room table
└── SyncQueueEntity      # Sync queue table
```

### Tables
- **MessageEntity**: Store messages with sync state, expiration
- **RoomEntity**: Store rooms with metadata, statistics
- **SyncQueueEntity**: Store pending operations for offline sync

### Indices
- MessageEntity: roomId, senderId, timestamp, status, isExpired
- RoomEntity: isJoined, isArchived, lastMessageTimestamp
- SyncQueueEntity: roomId, operationType, status, priority, retryCount

## 🔄 Offline Sync Architecture

### Components
```
┌─────────────────────────────────────────────────────────────────┐
│                      Offline Queue                             │
│  • Enqueue operations (send, update, delete, reaction)         │
│  • Priority-based execution (LOW, MEDIUM, HIGH)                │
│  • Retry logic (exponential backoff)                           │
└─────────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────────┐
│                       Sync Engine                             │
│  • State machine (Idle, Syncing, Success, Error)             │
│  • Execute operations                                        │
│  • Conflict detection                                        │
│  • Real-time sync status                                     │
└─────────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────────┐
│                     Conflict Resolver                          │
│  • Detect conflicts (content, reactions, read receipts)       │
│  • Resolution strategies (LOCAL_WINS, SERVER_WINS, MANUAL)    │
│  • Message merging                                           │
└─────────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────────┐
│                  Background Sync Worker                        │
│  • WorkManager for periodic sync                              │
│  • Network constraints (WiFi only)                            │
│  • Battery constraints (not low)                              │
│  • Sync status tracking                                       │
└─────────────────────────────────────────────────────────────────┘
```

## 🎨 UI Architecture

### Component Hierarchy
```
Screen
└── Organism Components
    └── Molecule Components
        └── Atom Components
```

### Atomic Components
- Button, InputField, Card, Badge, Icon, Text

### Molecular Components
- MessageBubble, TypingIndicator, EncryptionStatus, ReplyPreview

### Organism Components
- MessageList, ChatSearchBar, RoomItemCard, ProfileAvatar

### State Management
- **ViewModel**: StateFlow for UI state
- **UiEvent**: Channel for user events
- **UiState**: Data class for UI state

## 🔌 Platform Integration Architecture

### Expect/Actual Pattern
```
shared/platform/
├── BiometricAuth.kt        # Expect interface
├── SecureClipboard.kt      # Expect interface
├── NotificationManager.kt   # Expect interface
└── NetworkMonitor.kt       # Expect interface

androidApp/platform/
├── BiometricAuthImpl.kt    # Actual implementation
├── SecureClipboardImpl.kt  # Actual implementation
├── NotificationManagerImpl.kt # Actual implementation
└── NetworkMonitorImpl.kt   # Actual implementation
```

### Platform Services
- **BiometricAuth**: AndroidX Biometric API
- **SecureClipboard**: AndroidX ClipboardManager
- **NotificationManager**: FCM + NotificationManager
- **NetworkMonitor**: ConnectivityManager
- **CrashReporter**: Sentry SDK
- **Analytics**: Amplitude/Mixpanel (placeholder)

## 🧪 Testing Architecture

### Test Types
```
Unit Tests (shared/)
├── Domain Model Tests
├── Repository Interface Tests
└── Use Case Interface Tests

Unit Tests (androidApp/)
├── ViewModel Tests
├── Screen Tests (Compose UI)
└── Component Tests

Integration Tests
├── Database Tests
├── Repository Tests
└── Offline Sync Tests

E2E Tests
├── Onboarding Flow Tests
├── Authentication Flow Tests
├── Chat Flow Tests
└── Settings Flow Tests
```

### Test Framework
- **Unit Tests**: JUnit 5, MockK
- **UI Tests**: Compose Testing Framework
- **Integration Tests**: JUnit 5, Room
- **E2E Tests**: Compose Testing, Espresso

## 🚀 Build Configuration

### Build Types
- **Debug**: Development build with debugging
- **Release**: Production build with optimization

### Product Flavors
- **demo**: Demo build
- **alpha**: Alpha build
- **beta**: Beta build
- **stable**: Stable release

### Build Optimizations
- **R8**: Code shrinking and obfuscation
- **ProGuard**: Additional ProGuard rules
- **Resource Shrinking**: Remove unused resources
- **Native Library Optimization**: Strip unused symbols

## 📊 Performance Monitoring

### Components
```
PerformanceProfiler
├── Method Execution Tracing (Android Trace API)
├── Memory Allocation Tracking
├── Heap Dumping
└── Strict Mode Enforcement

MemoryMonitor
├── Memory Usage Monitoring (polling)
├── Memory Pressure Detection (4 levels)
├── Native Heap Tracking
├── Memory Leak Detection (heuristic)
└── Real-time Memory State (Flow)
```

## ♿ Accessibility Architecture

### Accessibility Features
```
AccessibilityConfig
├── Screen Reader Detection (TalkBack)
├── High Contrast Detection
├── Large Text Detection
├── Font Scale Detection
└── Reduced Motion Detection

AccessibilityExtensions
├── Content Description Modifier
├── Heading Modifier (1-6)
├── State Description Modifier
├── Traversal Order Modifier
└── Hidden Modifier (invisibleToUser)
```

---

*For detailed implementation, see source code.*
