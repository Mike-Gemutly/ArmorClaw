# ArmorClaw Mobile App - Onboarding Flow Wireframes & Specifications

> **Document Purpose:** Complete onboarding experience for first-time users
> **Date Created:** 2026-02-10
> **Phase:** 1 (Foundation)
> **Priority:** HIGH - Critical for user adoption

---

## 1. Onboarding Flow Overview

### Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          APP LAUNCH                                  │
│                              │                                       │
│                              ▼                                       │
│                    ┌─────────────────┐                               │
│                    │  Has Account?   │                               │
│                    └────────┬────────┘                               │
│                             │                                         │
│              ┌──────────────┴──────────────┐                        │
│              │                             │                        │
│             YES                           NO                       │
│              │                             │                        │
│              ▼                             ▼                        │
│        ┌───────────┐              ┌─────────────────┐              │
│        │ Biometric │              │ Welcome Screen  │              │
│        │  Unlock   │              │   (Optional)    │              │
│        └─────┬─────┘              └────────┬────────┘              │
│              │                             │                        │
│              ▼                             ▼                        │
│        ┌─────────────┐            ┌──────────────┐                 │
│        │   Home      │            │   Security   │                 │
│        │   Screen    │            │ Explanation  │                 │
│        └─────────────┘            └──────┬───────┘                 │
│                                            │                         │
│                                            ▼                         │
│                                    ┌───────────────┐                │
│                                    │   Connect     │                │
│                                    │   Server      │                │
│                                    └───────┬───────┘                │
│                                            │                         │
│                                            ▼                         │
│                                    ┌───────────────┐                │
│                                    │  Permissions  │                │
│                                    └───────┬───────┘                │
│                                            │                         │
│                                            ▼                         │
│                                    ┌───────────────┐                │
│                                    │   Complete    │                │
│                                    └───────┬───────┘                │
│                                            │                         │
│                                            ▼                         │
│                                    ┌───────────────┐                │
│                                    │   Home        │                │
│                                    │   Screen      │                │
│                                    └───────────────┘                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Screen-by-Screen Wireframes

### 2.1 Welcome Screen (Optional - Skip for Power Users)

```
╔═══════════════════════════════════════════════════════════════════╗
║                                                                   ║
║                    [ArmorClaw Logo]                               ║
║                        🦞🔒                                       ║
║                                                                   ║
║              Welcome to ArmorClaw                                 ║
║                                                                   ║
║         Secure AI agents in your pocket                           ║
║                                                                   ║
║     Chat with powerful AI while keeping your                      ║
║     API keys and data completely safe.                            ║
║                                                                   ║
├───────────────────────────────────────────────────────────────────┤
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  🔒 Enterprise-Grade Security                           │     ║
║  │     Your API keys never leave the secure container      │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  💬 Chat Anywhere                                        │     ║
║  │     Connect to your agents from anywhere with Matrix     │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  🔐 Zero-Trust Architecture                              │     ║
║  │     Even if the agent is compromised, you're protected   │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
║                                                                   ║
║                      [Skip]              [Get Started →]          ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

**Specifications:**
- **Purpose:** Introduce value proposition to new users
- **Duration:** Auto-dismiss after 5 seconds if no interaction
- **Skip Option:** Available for power users (remembered in prefs)
- **Animations:** Subtle fade-in, logo bounce on load

**State Machine:**
```kotlin
sealed class WelcomeState {
    object Loading : WelcomeState()
    object ShowWelcome : WelcomeState()
    object SkipWelcome : WelcomeState()
}
```

---

### 2.2 Security Explanation Screen

```
╔═══════════════════════════════════════════════════════════════════╗
║                                                                   ║
║                        ← Back                                     ║
║                                                                   ║
║              🔒 Your Security, Explained                          ║
║                                                                   ║
├───────────────────────────────────────────────────────────────────┤
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │                                                           │     ║
║  │    [Your Phone]                                         │     ║
║  │         └──────────────┐                                │     ║
║  │                        │                                │     ║
║  │                  ┌─────▼─────┐                          │     ║
║  │                  │  Matrix   │                          │     ║
║  │                  │  E2EE 🔒  │                          │     ║
║  │                  └─────┬─────┘                          │     ║
║  │                        │                                │     ║
║  │                  ┌─────▼─────┐                          │     ║
║  │                  │  ArmorClaw│                          │     ║
║  │                  │  Bridge    │                          │     ║
║  │                  └─────┬─────┘                          │     ║
║  │                        │                                │     ║
║  │                  ┌─────▼─────┐                          │     ║
║  │                  │ Container │                          │     ║
║  │                  │  🦞 Agent │                          │     ║
║  │                  └───────────┘                          │     ║
║  │                                                           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  How ArmorClaw protects you:                                     ║
║                                                                   ║
║  ✓ All messages encrypted end-to-end                             ║
║  ✓ API keys injected into memory only                            ║
║  ✓ Agent runs in isolated container                              ║
║  ✓ No data stored on your device                                 ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
║                                                                   ║
║                      [← Back]              [Continue →]           ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

**Specifications:**
- **Purpose:** Build trust through transparency
- **Duration:** 45 seconds suggested reading time
- **Interactive:** Tap each component for more details
- **Learn More Link:** Links to security documentation

**Interactive Details:**
```kotlin
data class SecurityComponent(
    val id: String,
    val title: String,
    val description: String,
    val icon: Icon
)

val components = listOf(
    SecurityComponent("phone", "Your Phone", "Where you chat", Icons.Phone),
    SecurityComponent("matrix", "Matrix Protocol", "E2EE transport", Icons.Lock),
    SecurityComponent("bridge", "ArmorClaw Bridge", "Policy enforcement", Icons.Shield),
    SecurityComponent("container", "Container", "Isolated agent", Icons.Box)
)
```

---

### 2.3 Connect Server Screen

```
╔═══════════════════════════════════════════════════════════════════╗
║                        ← Back                                     ║
║                                                                   ║
║              Connect to Your ArmorClaw Server                     ║
║                                                                   ║
├───────────────────────────────────────────────────────────────────┤
║                                                                   ║
║  Enter your server details:                                       ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ Homeserver URL                                          │     ║
║  │ ┌─────────────────────────────────────────────────────┐ │     ║
║  │ │ https://matrix.example.com                          │ │     ║
║  │ └─────────────────────────────────────────────────────┘ │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ Username                                                │     ║
║  │ ┌─────────────────────────────────────────────────────┐ │     ║
║  │ │ @username:example.com                               │ │     ║
║  │ └─────────────────────────────────────────────────────┘ │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ Password                                                │     ║
║  │ ┌─────────────────────────────────────────────────────┐ │     ║
║  │ │ ••••••••••••••••                                    │ │     ║
║  │ └─────────────────────────────────────────────────────┘ │     ║
║  │                        [👁 Show]                        │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │                                                           │     ║
║  │  ⚡ Quick Options:                                       │     ║
║  │                                                           │     ║
║  │  📱 Scan QR Code                                         │     ║
║  │     Scan QR code from your server admin                  │     ║
║  │                                                           │     ║
║  │  🧪 Use Demo Server                                      │     ║
║  │     Try ArmorClaw with our demo server                   │     ║
║  │                                                           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
║                                                                   ║
║  Status: Testing connection... ⏳                               ║
║                                                                   ║
║                      [← Back]        [Connect →]                 ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

**States:**

**Initial State:**
```
Status: Idle
[Connect →] enabled
```

**Validating State:**
```
Status: Testing connection... ⏳
[Connect →] disabled (show spinner)
```

**Success State:**
```
Status: ✓ Connected! Ready to go
[Connect →] → [Continue →]
```

**Error State:**
```
Status: ✗ Could not connect
        - Homeserver URL not reachable
        - Check your internet connection
        - Verify the URL with your server admin
[Connect →] → [Retry]
```

**Specifications:**
```kotlin
sealed class ConnectionState {
    object Idle : ConnectionState()
    object Connecting : ConnectionState()
    data class Success(val serverInfo: ServerInfo) : ConnectionState()
    data class Error(val message: String, val details: List<String>) : ConnectionState()
}

data class ServerInfo(
    val homeserver: String,
    val userId: String,
    val supportsE2EE: Boolean,
    val version: String
)
```

**QR Code Flow:**
```
┌──────────────────────────────────────────────┐
│         Scan QR Code                         │
│                                              │
│     ┌──────────────┐                        │
│     │              │                        │
│     │   [Camera    │                        │
│     │    View]     │                        │
│     │              │                        │
│     └──────────────┘                        │
│                                              │
│  Point at a QR code from:                   │
│  • Server admin panel                        │
│  • Setup wizard output                       │
│  • Element X desktop                         │
│                                              │
│  [Enter manually instead]                    │
└──────────────────────────────────────────────┘
```

---

### 2.4 Permissions Screen

```
╔═══════════════════════════════════════════════════════════════════╗
║                        ← Back                                     ║
║                                                                   ║
║              Permissions for Best Experience                      ║
║                                                                   ║
├───────────────────────────────────────────────────────────────────┤
║                                                                   ║
║  We need a few permissions to give you the best experience:       ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  🔔 Notifications                                       │     ║
║  │                                                           │     ║
║  │  Get notified when:                                      │     ║
║  │  • Agent sends a message                                 │     ║
║  │  • Budget limit is approached                             │     ║
║  │  • Security event occurs                                  │     ║
║  │                                                           │     ║
║  │  Status: Not granted → [Allow]                           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  🎤 Microphone (Optional)                                │     ║
║  │                                                           │     ║
║  │  For voice input to agents:                               │     ║
║  │  • Dictate messages instead of typing                    │     ║
║  │  • Voice commands for quick actions                       │     ║
║  │                                                           │     ║
║  │  Status: Not granted → [Allow]  [Skip]                   │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  📷 Camera (Optional)                                    │     ║
║  │                                                           │     ║
║  │  For sending images to agents:                            │     ║
║  │  • Scan documents                                        │     ║
║  │  • Share photos for analysis                              │     ║
║  │                                                           │     ║
║  │  Status: Not granted → [Allow]  [Skip]                   │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ℹ️ You can change these later in Settings                       ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
║                                                                   ║
║              Status: 1 of 3 required granted                     ║
║              [Continue Anyway]  [Continue →]                     ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

**Specifications:**
```kotlin
data class Permission(
    val type: PermissionType,
    val title: String,
    val description: String,
    val useCases: List<String>,
    val required: Boolean,
    val granted: Boolean = false
)

enum class PermissionType {
    NOTIFICATIONS,
    MICROPHONE,
    CAMERA,
    BIOMETRIC,
    STORAGE
}

data class PermissionsState(
    val permissions: List<Permission>,
    val requiredGranted: Int,
    val requiredTotal: Int
) {
    val canContinue: Boolean
        get() = requiredGranted >= requiredTotal
}
```

**Permission Request Flow:**
```kotlin
class PermissionManager(
    private val context: Context
) {
    suspend fun requestPermission(
        permission: PermissionType,
        showRationale: Boolean = true
    ): PermissionResult

    fun shouldShowRationale(permission: PermissionType): Boolean
}

sealed class PermissionResult {
    object Granted : PermissionResult()
    object Denied : PermissionResult()
    data class PermanentlyDenied(val openSettings: () -> Unit) : PermissionResult()
}
```

---

### 2.5 Completion Screen

```
╔═══════════════════════════════════════════════════════════════════╗
║                                                                   ║
║                        ✅ Setup Complete!                          ║
║                                                                   ║
║                     [Celebration Animation]                        ║
║                                                                   ║
├───────────────────────────────────────────────────────────────────┤
║                                                                   ║
║  You're all set to chat with your AI agents securely!             ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  What's Next:                                            │     ║
║  │                                                           │     ║
║  │  1. 💬 Join or create a room                              │     ║
║  │  2. 🤖 Start an agent                                     │     ║
║  │  3. ✨ Ask anything!                                       │     ║
║  │                                                           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  Quick Tip:                                               │     ║
║  │                                                           │     ║
║  │  Type /help to see all available commands                │     ║
║  │                                                           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
║                                                                   ║
║                    [Take Tutorial]  [Start Chatting →]           ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

**Specifications:**
```kotlin
sealed class OnboardingCompletion {
    object ShowCompletion : OnboardingCompletion()
    object ShowTutorial : OnboardingCompletion()
    object GoToHome : OnboardingCompletion()
}
```

---

### 2.6 Tutorial Overlay (Optional)

```
╔═══════════════════════════════════════════════════════════════════╗
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │  💬 Messages                                             │     ║
║  │  ────────────────────────────────────────────────       │     ║
║  │                                                           │     ║
║  │  This is where you chat with your AI agents.              │     ║
║  │  Type a message or use voice input to get started.       │     ║
║  │                                                           │     ║
║  │                                    [Skip Tutorial]       │     ║
║  │                                    [Next →]              │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║                    [Dark overlay behind]                         ║
║                    [Spotlight on message input]                  ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

**Tutorial Steps:**
1. Messages - How to send messages
2. Quick Actions - Using quick action buttons
3. Settings - Accessing configuration
4. Security - Understanding security indicators

---

## 3. Empty States

### 3.1 No Conversations

```
╔═══════════════════════════════════════════════════════════════════╗
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │                                                           │     ║
║  │                    [Chat Icon]                            │     ║
║  │                        💬                                 │     ║
║  │                                                           │     ║
║  │              No conversations yet                         │     ║
║  │                                                           │     ║
║  │  Start chatting with your AI agent by:                    │     ║
║  │                                                           │     ║
║  │  • Joining an existing room                               │     ║
║  │  • Creating a new room                                    │     ║
║  │  • Scanning a room QR code                                │     ║
║  │                                                           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║                    [+ Create Room]   [Join Room]                 ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

### 3.2 No Messages

```
╔═══════════════════════════════════════════════════════════════════╗
║  ← Rooms                                    ArmorClaw Agent 🔒   ║
├───────────────────────────────────────────────────────────────────┤
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │                                                           │     ║
║  │                    [Message Icon]                         │     ║
║  │                         ✉️                                 │     ║
║  │                                                           │     ║
║  │                    No messages yet                         │     ║
║  │                                                           │     ║
║  │  Send a message to start the conversation.                │     ║
║  │                                                           │     ║
║  │  Try:                                                       │     ║
║  │  • "Hello!"                                                │     ║
║  │  • "What can you do?"                                      │     ║
║  │  • "Help me with..."                                       │     ║
║  │                                                           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ Type a message...                        [📎] [🎤]      │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

### 3.3 Offline State

```
╔═══════════════════════════════════════════════════════════════════╗
║  ← Rooms                                    ArmorClaw Agent 🔒   ║
├───────────────────────────────────────────────────────────────────┤
║  ═══════════════════════════════════════════════════════════════  ║
║  ⚠️ You're offline                                              ║
║                                                                   ║
║  Check your internet connection.                                 ║
║  Messages will be sent when you're back online.                  ║
║                                                                   ║
║  ═══════════════════════════════════════════════════════════════  ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ Agent                                    Yesterday       │     ║
║  │ Hello! How can I help you today?          [✓✓]           │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ You                                                    │     ║
║  │ Can you analyze this data?                [⏳ Queued]   │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

---

## 4. Loading States

### 4.1 Skeleton Screens

```
╔═══════════════════════════════════════════════════════════════════╗
║  ← Rooms                                              [Settings] ║
├───────────────────────────────────────────────────────────────────┤
║                                                                   ║
║  ═══════════════════════════════════════════════════════════════  ║
║  Loading conversations...                                         ║
║  ═══════════════════════════════════════════════════════════════  ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ ┌────┐                                                   │     ║
║  │ │░░░ │  ░░░░░░░░░░░░░░░░░░░░░░              Just now   │     ║
║  │ └────┘  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░           │     ║
║  │         ░░░░░░░░░░░                                     │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ ┌────┐                                                   │     ║
║  │ │░░░ │  ░░░░░░░░░░░░░░░░░░░░░░              Yesterday │     ║
║  │ └────┘  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░           │     ║
║  │         ░░░░░░░░░░░                                     │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
║  ┌─────────────────────────────────────────────────────────┐     ║
║  │ ┌────┐                                                   │     ║
║  │ │░░░ │  ░░░░░░░░░░░░░░░░░░░░░░             2 days ago │     ║
║  │ └────┘  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░           │     ║
║  │         ░░░░░░░░░░                                     │     ║
║  └─────────────────────────────────────────────────────────┘     ║
║                                                                   ║
╚═══════════════════════════════════════════════════════════════════╝
```

### 4.2 Message Loading

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  Agent                                            2:30 PM  │
│  ┌─────────────────────────────────────────────────┐       │
│  │ Here's the analysis you requested:               │       │
│  │                                                   │       │
│  │ 1. Revenue increased by 23%                       │       │
│  │ 2. Customer satisfaction improved                │       │
│  │ 3. ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░              │       │
│  │    ░░░░░░░░░░░░░░░░░░░░░░                        │       │
│  │    ░░░░░░░░░░░░░░░░░░░                           │       │
│  │                                                   │       │
│  └─────────────────────────────────────────────────┘       │
│                                                             │
│  ┌─────────────────────────────────────────────────┐       │
│  │  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░        │       │
│  │  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░        │       │
│  │  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░        │       │
│  └─────────────────────────────────────────────────┘       │
│                                                             │
│                                    [✓✓]    Still typing...  │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Implementation Specifications

### 5.1 Onboarding Manager

```kotlin
class OnboardingManager(
    private val preferences: PreferencesManager,
    private val permissions: PermissionManager,
    private val matrixClient: MatrixClient
) {
    // Check if onboarding completed
    fun hasCompletedOnboarding(): Boolean {
        return preferences.getBoolean("onboarding_completed", false)
    }

    // Get current step
    fun getCurrentStep(): OnboardingStep {
        if (!hasCompletedOnboarding()) {
            return OnboardingStep.Welcome
        }
        return OnboardingStep.Complete
    }

    // Complete onboarding
    suspend fun completeOnboarding() {
        preferences.putBoolean("onboarding_completed", true)
    }

    // Skip onboarding
    suspend fun skipOnboarding() {
        preferences.putBoolean("onboarding_skipped", true)
        completeOnboarding()
    }

    // Reset onboarding (for testing)
    suspend fun resetOnboarding() {
        preferences.remove("onboarding_completed")
        preferences.remove("onboarding_skipped")
    }
}

sealed class OnboardingStep {
    object Welcome : OnboardingStep()
    object Security : OnboardingStep()
    object ConnectServer : OnboardingStep()
    object Permissions : OnboardingStep()
    object Complete : OnboardingStep()
}
```

### 5.2 Connection Manager

```kotlin
class ServerConnectionManager(
    private val matrixClient: MatrixClient
) {
    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Idle)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    suspend fun testConnection(config: ServerConfig): Result<ServerInfo> {
        _connectionState.value = ConnectionState.Connecting

        return try {
            val info = matrixClient.testConnection(config)
            _connectionState.value = ConnectionState.Success(info)
            Result.success(info)
        } catch (e: Exception) {
            _connectionState.value = ConnectionState.Error(
                message = "Could not connect",
                details = listOf(
                    "Homeserver URL not reachable",
                    "Check your internet connection",
                    "Verify the URL with your server admin"
                )
            )
            Result.failure(e)
        }
    }

    suspend fun connect(config: ServerConfig): Result<Unit> {
        return try {
            matrixClient.login(config)
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

sealed class ConnectionState {
    object Idle : ConnectionState()
    object Connecting : ConnectionState()
    data class Success(val serverInfo: ServerInfo) : ConnectionState()
    data class Error(val message: String, val details: List<String>) : ConnectionState()
}
```

### 5.3 Permission Manager

```kotlin
class PermissionManager(
    private val context: Context
) {
    private val permissions = listOf(
        Permission(
            type = PermissionType.NOTIFICATIONS,
            title = "Notifications",
            description = "Get notified when agent sends a message",
            useCases = listOf("Agent messages", "Budget alerts", "Security events"),
            required = true
        ),
        Permission(
            type = PermissionType.MICROPHONE,
            title = "Microphone",
            description = "Dictate messages instead of typing",
            useCases = listOf("Voice input", "Voice commands"),
            required = false
        ),
        Permission(
            type = PermissionType.CAMERA,
            title = "Camera",
            description = "Send images to agents for analysis",
            useCases = listOf("Scan documents", "Share photos"),
            required = false
        )
    )

    val permissionsState: StateFlow<PermissionsState> =
        combine(
            permissions.map { it.grantFlow }
        ) { grantedFlags ->
            permissions.mapIndexed { index, permission ->
                permission.copy(granted = grantedFlags[index])
            }
        }.map { perms ->
            PermissionsState(
                permissions = perms,
                requiredGranted = perms.count { it.required && it.granted },
                requiredTotal = perms.count { it.required }
            )
        }.stateIn(
            scope = CoroutineScope(Dispatchers.Default),
            started = SharingStarted.Eagerly,
            initialValue = PermissionsState(
                permissions = permissions,
                requiredGranted = 0,
                requiredTotal = permissions.count { it.required }
            )
        )

    suspend fun requestPermission(
        permission: PermissionType,
        showRationale: Boolean = true
    ): PermissionResult {
        val perm = permissions.find { it.type == permission }
            ?: return PermissionResult.Denied

        if (showRationale && shouldShowRationale(permission)) {
            // Show rationale dialog first
            showRationaleDialog(perm)
        }

        return when (permission) {
            PermissionType.NOTIFICATIONS -> requestNotificationPermission()
            PermissionType.MICROPHONE -> requestMicrophonePermission()
            PermissionType.CAMERA -> requestCameraPermission()
            else -> PermissionResult.Denied
        }
    }

    fun shouldShowRationale(permission: PermissionType): Boolean {
        return when (permission) {
            PermissionType.NOTIFICATIONS -> !isNotificationPermissionGranted()
            PermissionType.MICROPHONE -> context.shouldShowRequestPermissionRationale(
                Manifest.permission.RECORD_AUDIO
            )
            PermissionType.CAMERA -> context.shouldShowRequestPermissionRationale(
                Manifest.permission.CAMERA
            )
            else -> false
        }
    }
}
```

---

## 6. Testing Checklist

- [ ] Welcome screen displays correctly
- [ ] Skip button works and is remembered
- [ ] Security screen animations play
- [ ] Connection test handles all states (idle, connecting, success, error)
- [ ] QR code scanner launches and parses correctly
- [ ] Demo server option connects successfully
- [ ] Permissions are requested with rationale
- [ ] Optional permissions can be skipped
- [ ] Completion celebration animation plays
- [ ] Tutorial overlay highlights correctly
- [ ] Empty states display when appropriate
- [ ] Skeleton screens display during loading
- [ ] Can skip onboarding and go directly to login
- [ ] Onboarding state persists across app restarts

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-10
**Status:** Ready for Implementation
