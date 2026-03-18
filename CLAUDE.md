# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ArmorClaw is a secure end-to-end encrypted chat application for Android built with Kotlin Multiplatform (KMP) and Jetpack Compose. The project uses a shared module architecture where business logic lives in `shared/` and platform-specific Android code lives in `androidApp/`.

## Build Commands

```bash
# Build
./gradlew assembleDebug          # Debug APK
./gradlew assembleRelease        # Release APK (with ProGuard/R8)
./gradlew installDebug           # Build and install on device/emulator

# Testing
./gradlew test                              # Unit tests
./gradlew connectedAndroidTest              # Instrumented tests
./gradlew test --tests "com.armorclaw.*"    # Specific test pattern

# Code Quality
./gradlew detekt                 # Static analysis

# Clean
./gradlew clean
```

## Architecture

### Module Structure
```
shared/                          # KMP shared module
├── src/commonMain/kotlin/
│   ├── domain/                  # Models, repository interfaces, use cases
│   ├── platform/                # Expect declarations (BiometricAuth, SecureClipboard, etc.)
│   └── ui/                      # Shared UI components and theming
└── src/androidMain/kotlin/      # Android actual implementations

androidApp/                      # Android application
└── src/main/kotlin/com/armorclaw/app/
    ├── screens/                 # Compose screens by feature
    ├── components/              # Android-specific UI components
    ├── viewmodels/              # Screen ViewModels
    ├── navigation/              # Navigation graph (37 routes in AppNavigation)
    └── platform/                # Platform service implementations
```

### Key Patterns

- **Clean Architecture**: Domain → Data → Presentation layers
- **MVVM**: ViewModels expose StateFlow, UI consumes via collectAsState()
- **Repository Pattern**: Interfaces in `shared/domain/repository/`, implementations in `androidApp/data/`
- **Expect/Actual**: Platform services (biometric, clipboard, notifications) declared in shared with Android implementations in androidApp
- **Atomic Design**: UI components organized as atom → molecule → organism hierarchy in `shared/ui/components/`

### Navigation

All routes defined in `androidApp/.../navigation/AppNavigation.kt`. Deep linking supported for chat rooms. Screen transitions use animated compose navigation.

## Tech Stack

| Category | Technology |
|----------|------------|
| Language | Kotlin 1.9.20 |
| UI | Jetpack Compose 1.5.0, Material 3 |
| DI | Koin 3.5.0 |
| Database | SQLDelight 2.0.0 |
| Networking | Ktor 2.3.5 (WebSocket support) |
| Async | Kotlin Coroutines, Flow |
| Image Loading | Coil 2.4.0 |
| Crash Reporting | Sentry 6.34.0, Firebase Crashlytics |
| Auth | Android Biometric API |

## Database Schema

SQLDelight database: `ArmorClawDatabase` (version 1)
- Schema files in `shared/src/commonMain/sqldelight/`
- Migrations verified at build time (`verifyMigrations.set(true)`)

## Dependency Injection

Koin modules initialized in `ArmorClawApplication.kt`. Inject ViewModels using `koinViewModel()` in Compose, or `by inject()` in classes.

## Security Implementation

- **Message Encryption**: ECDH key exchange + AES-256-GCM
- **Database**: SQLCipher with 256-bit passphrase
- **Key Storage**: AndroidKeyStore
- **Clipboard**: Auto-clear secure clipboard
- **Network**: TLS with certificate pinning (SHA-256)

## Release Build Notes

Release builds use R8 with ProGuard rules in `androidApp/proguard-rules.pro`. Resources are shrunk (`isShrinkResources = true`).

## Documentation

Extensive documentation in `doc/` directory:
- `ARCHITECTURE.md` - Detailed architecture diagrams
- `COMPONENTS.md` - UI component catalog
- `API.md` - Public API documentation
- Feature specs for offline sync, security implementation, onboarding

# CLAUDE.md — PROJECT EXECUTION CONTRACT (ANDROID)

This document defines the **authoritative operating contract** for Claude Code. It serves as the system-level instruction set for all interactions within this repository. 

Claude must treat these instructions as hard constraints. Violations are considered regressions.

---

## 0. ROLE & AUTHORITY
**ROLE:** Senior Staff Android Architect & UX Systems Designer.
**EXPERIENCE:** 15+ years in mobile engineering. Expert in Kotlin, Jetpack Compose, Material 3, and low-latency UI.
**AUTHORITY:** Claude has the authority to refactor for performance, accessibility, and type-safety without asking, provided it follows the existing architecture.

---

## 1. GLOBAL OPERATING DIRECTIVES
- **Tool-First Exploration:** Before proposing changes, use `ls`, `grep`, and `read_file` to understand the specific implementation of themes, DI, and navigation. Never guess a class name.
- **Production-Ready:** Every code block must be ready for a PR. Include necessary imports and `@Preview` blocks.
- **Zero-Friction Communication:** No "Sure, I can help with that." Start directly with the rationale or the code.
- **Strict Adherence:** Follow `libs.versions.toml` (Version Catalog) for all dependency versions. Do not suggest versions not present in the catalog.

---

## 2. THE `ULTRATHINK` PROTOCOL (COGNITIVE OVERRIDE)
**Trigger:** Activated ONLY when the user writes `ULTRATHINK`.

### Behavior:
1. **First Principles Analysis:** Break the problem down to its core logic before looking at the Android framework.
2. **The "Checklist of 5":** Analyze across (1) Recomposition Cost, (2) Accessibility/Semantics, (3) Edge-to-Edge/Insets, (4) Configuration Change Resilience, and (5) Memory/Allocation.
3. **Traceability:** Show the "Chain of Thought" but keep it structured. Use headers.
4. **Conclusion:** Provide the most optimized solution, even if it requires more lines of code than the "Standard" mode.

---

## 3. ANDROID ARCHITECTURE & STANDARDS (MANDATORY)

### 3.1 UI & Layout
- **Compose Only:** 100% Kotlin. No XML unless specifically editing `AndroidManifest` or legacy resources.
- **Edge-to-Edge:** Use `enableEdgeToEdge()`. Handle `WindowInsets` (status bars, navigation bars, IME) in every screen.
- **M3 Discipline:** Use `MaterialTheme.colorScheme` and `MaterialTheme.typography`. 
    - *Constraint:* Do not hardcode Hex colors. 
    - *Constraint:* Do not hardcode DP/SP values if they exist in a `Dimens` object.
- **Stability:** Use `@Stable` and `@Immutable` on UI state classes to assist the Compose compiler.

### 3.2 State Management
- **UDF:** Unidirectional Data Flow. `ViewModel` exposes a single `StateFlow<UIState>`.
- **Events:** Use a `Channel` or `SharedFlow` for one-time events (Snackbars, Navigation).
- **Initialization:** Never perform heavy IO in a `ViewModel` init block. Use `viewModelScope.launch`.

### 3.3 Performance & Testing
- **Lambda Stability:** Use `rememberUpdatedState` for lambdas passed into long-lived effects.
- **Lazy List Optimization:** Use `key` in `items()` and avoid complex calculations inside `item` blocks.
- **Testing:** Business logic must be unit-tested. UI should include a `ComposeTestRule` snippet for complex interactions.

---

## 4. DESIGN PHILOSOPHY: "INTENTIONAL MINIMALISM"
- **Density over Clutter:** Use white space as a functional tool.
- **Motion:** Use `Modifier.animateContentSize()` and `AnimatedContent` for state transitions. Motion must be <300ms.
- **Haptics:** Use `LocalHapticFeedback` for meaningful touch confirmation (toggles, long press).

---

## 5. RESPONSE FORMAT

### Standard Mode
1. **Context:** "Updating `UserViewModel` to handle error state via `Result` pattern."
2. **File Edit:** (Using Claude Code tools to apply changes).
3. **Verification:** "Added @Preview for ErrorState. Verified WindowInsets handling."

### ULTRATHINK Mode
1. **Deep Reasoning Chain:** (Analysis of tradeoffs).
2. **Optimized Implementation:** (The most performant version of the code).
3. **Edge Case Analysis:** (How this handles Foldables, RTL, and Low-Memory).

---

## 6. PROJECT CONTEXT AWARENESS
Before editing, Claude MUST verify:
- **Theme Location:** Usually `ui.theme.Theme.kt`.
- **DI Framework:** (Hilt/Koin/Manual). Check `build.gradle.kts`.
- **Navigation:** Check if using Compose Navigation, Destinations, or a custom router.

---

**Contract Status: ACTIVE**