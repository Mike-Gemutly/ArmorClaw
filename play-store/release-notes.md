# Release Notes Templates

## Initial Release (v1.0.0)

### Play Store Release Notes (500 characters max)

```
Welcome to ArmorChat - secure messaging for teams!

🛡️ End-to-end encryption (ECDH + AES-256-GCM)
🔐 Biometric authentication
💬 Rich messaging with reactions & replies
📱 Full offline support
🌙 Dark mode included

Your messages. Your privacy. Period.
```
*234 characters*

---

### Full Changelog (for GitHub/website)

## [1.0.0] - 2026-02-25

### Added
- End-to-end encrypted messaging with ECDH key exchange + AES-256-GCM
- Biometric authentication (fingerprint/face unlock)
- Chat room creation and management
- Rich message features:
  - Message reactions
  - Reply to messages
  - File attachments
  - Voice messages
- Message search functionality
- Full offline support with intelligent sync queue
- Dark mode with automatic theme switching
- Material Design 3 UI
- Accessibility support (TalkBack compatible)
- Secure clipboard with auto-clear
- Certificate pinning for network security
- SQLCipher encrypted database
- Crash reporting via Sentry

### Security
- Zero-knowledge architecture
- AndroidKeyStore for secure key storage
- TLS 1.3 with certificate pinning
- Session timeout and auto-lock

---

## Future Release Template

### Play Store Release Notes

```
## What's New in [VERSION]

### ✨ New Features
- [Feature 1]
- [Feature 2]

### 🐛 Bug Fixes
- [Fix 1]
- [Fix 2]

### 🔒 Security
- [Security improvement if any]
```

---

## Version Naming Convention

- **Major (X.0.0):** Breaking changes, major features
- **Minor (1.X.0):** New features, enhancements
- **Patch (1.0.X):** Bug fixes, minor improvements

---

## Release Checklist

### Before Release
- [ ] Update versionCode in build.gradle.kts
- [ ] Update versionName in build.gradle.kts
- [ ] Update CHANGELOG.md
- [ ] Write release notes
- [ ] Run full test suite: `./gradlew test`
- [ ] Build release AAB: `./gradlew bundleRelease`
- [ ] Test on physical device
- [ ] Verify signing with: `jarsigner -verify -verbose app.aab`

### Play Store Submission
- [ ] Upload AAB to Play Console
- [ ] Update store listing if needed
- [ ] Add release notes
- [ ] Select rollout percentage
- [ ] Submit for review

### After Release
- [ ] Create GitHub release with changelog
- [ ] Tag release: `git tag v1.0.0`
- [ ] Push tags: `git push --tags`
- [ ] Update website/docs if applicable
