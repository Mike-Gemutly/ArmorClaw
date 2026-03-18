# ArmorClaw Documentation Index

> **Version:** 1.1.0 | **Last Updated:** 2026-02-18 | **Status:** Production Ready

## Quick Navigation

| Document | Description |
|----------|-------------|
| **[index.md](index.md)** | This document - Feature directory |
| **[CHANGELOG.md](CHANGELOG.md)** | Version history and changes |
| **[HISTORY.md](HISTORY.md)** | Consolidated development timeline |
| **[ARCHITECTURE.md](ARCHITECTURE.md)** | System architecture overview |
| **[SECURITY.md](SECURITY.md)** | Security implementation |
| **[MATRIX_MIGRATION.md](MATRIX_MIGRATION.md)** | Matrix protocol migration |
| **[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)** | Development setup guide |
| **[API.md](API.md)** | API reference |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         UI Layer                                 │
│  Screens → ViewModels → Components                              │
├─────────────────────────────────────────────────────────────────┤
│                       Domain Layer                               │
│  UseCases → Models → Repositories (interfaces)                  │
├─────────────────────────────────────────────────────────────────┤
│                        Data Layer                                │
│  RepositoryImpl → Store → DataSource                            │
├─────────────────────────────────────────────────────────────────┤
│                      Platform Layer                              │
│  MatrixClient → BiometricAuth → SessionStorage                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Features by Category

### 🔐 Authentication & Security

| Feature | Description | Key Functions |
|---------|-------------|---------------|
| [Authentication](features/authentication.md) | Login, registration, session | `login()`, `logout()`, `restoreSession()` |
| [Biometric Auth](features/biometric-auth.md) | Fingerprint/Face auth | `isAvailable()`, `authenticate()` |
| [Encryption](features/encryption.md) | Matrix E2EE (Olm/Megolm) | `isRoomEncrypted()`, key management |
| [Device Management](features/device-management.md) | Cross-device verification | `verifyDevice()`, `getDevices()` |
| [Secure Clipboard](features/secure-clipboard.md) | Encrypted clipboard | `encryptCopy()`, `decryptPaste()` |

### 💬 Messaging

| Feature | Description | Key Functions |
|---------|-------------|---------------|
| [Chat](features/chat.md) | Unified messaging | `sendMessage()`, `handleCommand()` |
| [Threads](features/threads.md) | Threaded conversations | `getThread()`, `sendReply()` |
| [Voice Calls](features/voice-calls.md) | VoIP via WebRTC | `startCall()`, `joinCall()` |

### 🏠 Rooms & Navigation

| Feature | Description | Key Functions |
|---------|-------------|---------------|
| [Home Screen](features/home-screen.md) | Room list, workflows | `getRooms()`, `activeWorkflows` |
| [Room Management](features/room-management.md) | Create, join, invite | `createRoom()`, `joinRoom()` |
| [Room Settings](features/room-settings.md) | Room configuration | `updateRoom()`, `leaveRoom()` |

### ⚙️ System

| Feature | Description | Key Functions |
|---------|-------------|---------------|
| [Onboarding](features/onboarding.md) | Welcome flow | `setServer()`, `checkPermissions()` |
| [Settings](features/settings.md) | App preferences | `getSettings()`, `updateSettings()` |
| [Profile](features/profile.md) | User profile | `getProfile()`, `updateProfile()` |
| [Offline Sync](features/offline-sync.md) | Background sync | `enqueue()`, `sync()`, `resolveConflict()` |
| [Notifications](features/notifications.md) | Push notifications | `registerPush()`, `handlePush()` |
| [Performance](features/performance.md) | Optimization | `startTrace()`, `monitorMemory()` |

---

## Screens Reference

| Screen | Route | ViewModel | Doc |
|--------|-------|-----------|-----|
| Splash | `splash` | - | [→](screens/SplashScreen.md) |
| Welcome | `welcome` | - | [→](screens/WelcomeScreen.md) |
| Login | `login` | - | [→](screens/LoginScreen.md) |
| Home | `home` | `HomeViewModel` | [→](screens/HomeScreen.md) |
| Chat | `chat/{roomId}` | `ChatViewModel` | [→](screens/ChatScreen.md) |
| Search | `search` | - | [→](screens/SearchScreen.md) |
| Profile | `profile` | - | [→](screens/ProfileScreen.md) |
| Settings | `settings` | `SettingsViewModel` | [→](screens/SettingsScreen.md) |
| Active Call | `call/{callId}` | - | [→](screens/ActiveCallScreen.md) |

---

## ViewModels Reference

| ViewModel | Purpose | Key State |
|-----------|---------|-----------|
| [ChatViewModel](viewmodels/ChatViewModel.md) | Messages, commands | `unifiedMessages`, `activeWorkflow` |
| [HomeViewModel](viewmodels/HomeViewModel.md) | Rooms, workflows | `rooms`, `activeWorkflows` |
| [DeviceListViewModel](viewmodels/DeviceListViewModel.md) | Device management | `devices`, `verificationState` |
| SettingsViewModel | App settings | `settings`, `logoutState` |

---

## Components Reference

| Component | Purpose | Key Props |
|-----------|---------|-----------|
| [MessageBubble](components/MessageBubble.md) | Message display | `message`, `onReaction` |
| [MessageList](components/MessageList.md) | Message list | `messages`, `onLoadMore` |
| [TypingIndicator](components/TypingIndicator.md) | Typing status | `isTyping`, `userName` |
| [EncryptionStatus](components/EncryptionStatus.md) | E2EE badge | `status`, `onVerify` |
| [ReplyPreview](components/ReplyPreview.md) | Reply context | `message`, `onCancel` |
| [CallControls](components/CallControls.md) | Call buttons | `onMute`, `onEnd` |
| [WorkflowProgressBanner](components/WorkflowProgressBanner.md) | Workflow progress | `workflow`, `onCancel` |
| [AgentThinkingIndicator](components/AgentThinkingIndicator.md) | Agent thinking | `agent`, `message` |

---

## Core Models

| Model | Location | Description |
|-------|----------|-------------|
| `UnifiedMessage` | `domain/model/` | All message types |
| `Message` | `domain/model/` | Legacy message |
| `Room` | `domain/model/` | Chat room |
| `UserSession` | `domain/model/` | Auth session |
| `WorkflowState` | `data/store/` | Workflow state |
| `AgentThinkingState` | `data/store/` | Agent state |

---

## Platform Layer

| Module | Purpose | Location |
|--------|---------|----------|
| `MatrixClient` | Matrix protocol | `platform/matrix/` |
| `BridgeAdminClient` | Admin operations | `platform/bridge/` |
| `BiometricAuth` | Biometric API | `platform/` |
| `MatrixSessionStorage` | Encrypted storage | `platform/matrix/` |

---

## Data Layer

| Module | Purpose | Location |
|--------|---------|----------|
| `ControlPlaneStore` | Event processing | `data/store/` |
| `WorkflowRepository` | Workflow state | `domain/repository/` |
| `AgentRepository` | Agent state | `domain/repository/` |
| `OfflineQueue` | Message queue | `data/` |

---

## Test Coverage

| Category | Tests | Coverage |
|----------|-------|----------|
| Matrix SDK | 25+ | Authentication, messaging, sync |
| Control Plane | 15+ | Workflow, agent events |
| UI Components | 45+ | Banners, cards, indicators |
| Unified Message | 25+ | All message types |
| ChatViewModel | 20+ | Integration tests |
| **Total** | **130+** | |

---

## File Quick Reference

### Critical Files

| File | Purpose |
|------|---------|
| `MatrixClient.kt` | Matrix SDK interface (40+ methods) |
| `UnifiedMessage.kt` | Unified message model |
| `ChatViewModel.kt` | Chat state management |
| `ControlPlaneStore.kt` | Event processor |
| `MatrixSessionStorage.kt` | Encrypted session |

### Configuration Files

| File | Purpose |
|------|---------|
| `libs.versions.toml` | Dependencies |
| `build.gradle.kts` | Build config |
| `AppModules.kt` | DI modules |

---

## Migration Status

| Phase | Description | Status |
|-------|-------------|--------|
| 1 | Matrix SDK Integration | ✅ |
| 2 | Transport Layer Split | ✅ |
| 3 | Control Plane Events | ✅ |
| 4 | UI Unification | ✅ |

**See:** [MATRIX_MIGRATION.md](MATRIX_MIGRATION.md)

---

## Related Documentation

- [User Guide](USER_GUIDE.md) - End-user documentation
- [Components Catalog](COMPONENTS.md) - UI component reference
- [Features List](FEATURES.md) - Complete feature list
