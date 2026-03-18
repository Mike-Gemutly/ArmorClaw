# ArmorClaw

> 🛡️ Secure. Private. Encrypted. End-to-End Encrypted Chat Application for Android

[![Android](https://img.shields.io/badge/Android-21%2B-green.svg)](https://developer.android.com/studio)
[![Kotlin](https://img.shields.io/badge/Kotlin-1.9.20-blue.svg)](https://kotlinlang.org)
[![Compose](https://img.shields.io/badge/Compose-1.5.0-purple.svg)](https://developer.android.com/jetpack/compose)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

ArmorClaw is a modern, secure, end-to-end encrypted chat application built with Kotlin Multiplatform (KMP) and Jetpack Compose for Android. It features a clean, intuitive UI, comprehensive security features, and robust offline capabilities.

## 🚀 Features

- 🔒 **End-to-End Encryption** - All messages encrypted with ECDH+AES-GCM
- 🔐 **Biometric Authentication** - Fingerprint/FaceID for secure unlock
- 📱 **Offline Support** - Full offline capabilities with sync queue
- 💬 **Rich Chat Experience** - Reactions, replies, attachments, voice messages
- 🎨 **Material Design 3** - Modern, accessible UI with theming
- 🔍 **Message Search** - Search within conversations
- 🏠 **Room Management** - Create and join rooms easily
- 👤 **Profile Management** - Custom avatars, status, account settings
- ⚙️ **App Settings** - Comprehensive settings with feature flags
- 🌙 **Dark Mode** - Automatic dark mode support
- ♿ **Accessibility** - Full accessibility compliance (TalkBack, screen reader)
- 📊 **Performance Monitoring** - Built-in profiling and memory monitoring
- 🛡️ **Security Features** - Secure clipboard, certificate pinning, crash reporting
- 🚀 **Smooth Navigation** - Animated transitions, deep linking

## 📦 Installation

### Prerequisites

- Android Studio Hedgehog (2023.1.1) or later
- JDK 17
- Android SDK 34
- Gradle 8.2

### Clone Repository

```bash
git clone https://github.com/armorclaw/ArmorClaw.git
cd ArmorClaw
```

### Build Project

```bash
# Build debug APK
./gradlew assembleDebug

# Build release APK
./gradlew assembleRelease

# Run on device/emulator
./gradlew installDebug
```

## 🏗️ Architecture

ArmorClaw follows a modular, multiplatform architecture:

```
ArmorClaw/
├── shared/              # KMP shared module
│   ├── domain/          # Domain layer
│   ├── platform/        # Platform interfaces
│   └── ui/             # Shared UI components
└── androidApp/         # Android application
    ├── screens/         # Compose screens
    ├── data/            # Data layer
    ├── platform/        # Platform implementations
    └── release/         # Release configuration
```

### Key Components

- **Domain Layer**: Models, repositories, use cases
- **Data Layer**: Database (Room + SQLCipher), API, offline sync
- **Platform Layer**: Biometric auth, secure clipboard, notifications
- **UI Layer**: Compose screens, components, navigation

For detailed architecture, see [ARCHITECTURE.md](ARCHITECTURE.md).

## 🎨 Screens

### Onboarding Flow
1. **WelcomeScreen** - Feature overview, Get Started/Skip
2. **SecurityExplanationScreen** - Animated security diagram, 4 steps
3. **ConnectServerScreen** - Server connection, demo option
4. **PermissionsScreen** - Required/optional permissions
5. **CompletionScreen** - Celebration, confetti, what's next

### Main App Flow
1. **SplashScreen** - App launch with animations
2. **LoginScreen** - Authentication, biometric login
3. **HomeScreen** - Room list (Favorites, Chats, Archived)
4. **ChatScreen** - Enhanced chat with all features
5. **ProfileScreen** - Profile management, settings
6. **SettingsScreen** - App configuration, privacy
7. **RoomManagementScreen** - Create/Join rooms

For complete screen list, see [SCREENS.md](SCREENS.md).

## 🧩 Components

### Atomic Components
- Button, InputField, Card, Badge, Icon

### Molecular Components
- MessageBubble, TypingIndicator, EncryptionStatus, ReplyPreview

### Organism Components
- MessageList, ChatSearchBar, RoomItemCard, ProfileAvatar

For complete component catalog, see [COMPONENTS.md](COMPONENTS.md).

## 📱 Screenshots

### Onboarding
- Welcome screen with feature list
- Security explanation with animated diagram
- Server connection screen
- Permissions request screen
- Completion screen with celebration

### Main App
- Home screen with room list
- Chat screen with enhanced features
- Profile screen with settings
- Settings screen with configuration
- Room management screen

*(Screenshots to be added)*

## 🧪 Testing

### Run Tests

```bash
# Run unit tests
./gradlew test

# Run instrumented tests
./gradlew connectedAndroidTest

# Run specific test
./gradlew test --tests "com.armorclaw.app.*"
```

### Test Coverage

- Unit Tests: 50+
- Integration Tests: 15+
- E2E Tests: 11
- Total Tests: 75+

For testing details, see [TESTING.md](TESTING.md).

## 🔐 Security

### Encryption
- AES-256-GCM for message encryption
- ECDH for key exchange
- SQLCipher for database encryption (256-bit passphrase)

### Security Features
- Biometric authentication (Fingerprint/FaceID)
- Secure clipboard with auto-clear
- Certificate pinning (SHA-256)
- Secure key storage (AndroidKeyStore)
- Crash reporting (Sentry)

For security details, see [SECURITY.md](SECURITY.md).

## 📖 Documentation

- [Architecture](ARCHITECTURE.md) - Architecture overview
- [Features](FEATURES.md) - Complete feature list
- [Components](COMPONENTS.md) - UI component catalog
- [API](API.md) - Public API documentation
- [User Guide](USER_GUIDE.md) - User guide
- [Developer Guide](DEVELOPER_GUIDE.md) - Developer guide
- [Changelog](CHANGELOG.md) - Complete changelog

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

## 👥 Authors

- **ArmorClaw Team** - Initial work

## 🙏 Acknowledgments

- Kotlin team for KMP
- Google for Jetpack Compose
- Matrix team for Matrix protocol
- Sentry for crash reporting
- SQLCipher for encrypted database

## 📞 Support

- **Email**: support@armorclaw.app
- **Issues**: [GitHub Issues](https://github.com/armorclaw/ArmorClaw/issues)
- **Discussions**: [GitHub Discussions](https://github.com/armorclaw/ArmorClaw/discussions)

---

**Made with ❤️ for privacy**

*ArmorClaw - Secure. Private. Encrypted.*
