# Encryption Feature

> End-to-end encryption via Matrix Protocol (Olm/Megolm)
> Location: `shared/src/commonMain/kotlin/platform/matrix/`

## Overview

ArmorClaw uses Matrix's Olm/Megolm double ratchet algorithm for end-to-end encryption. Keys are held by the client, not the server.

---

## Critical Functions

### MatrixClient

```kotlin
// Encryption status
suspend fun isRoomEncrypted(roomId: String): Boolean

// Key management handled by Rust SDK
// - Automatic key exchange
// - Forward secrecy
// - Break-in recovery
```

### MatrixSessionStorage

```kotlin
// Encrypted session persistence
suspend fun saveSession(session: UserSession)
suspend fun loadSession(): UserSession?
suspend fun clearSession()
```

---

## Dependencies

### Platform
| Dependency | Purpose |
|------------|---------|
| `matrix-rust-sdk` | Olm/Megolm implementation |
| `EncryptedSharedPreferences` | Session token storage |
| `Android Keystore` | Master key protection |

### Crypto Libraries
| Library | Purpose |
|---------|---------|
| `libolm` | Double ratchet algorithm |
| `AES-256-GCM` | Symmetric encryption |
| `Curve25519` | Key exchange |
| `Ed25519` | Signing |

---

## Key Variables

### Session State
```kotlin
data class UserSession(
    val userId: String,
    val deviceId: String,
    val accessToken: String,
    val refreshToken: String?,
    val homeserver: String,
    val isVerified: Boolean
)
```

### Encryption Status
```kotlin
enum class EncryptionStatus {
    NONE,          // Not encrypted
    UNENCRYPTED,   // Room not encrypted
    UNVERIFIED,    // Encrypted, unverified device
    VERIFIED       // Encrypted and verified
}
```

---

## Encryption Standards

### Matrix Olm/Megolm

| Purpose | Algorithm | Details |
|----------|-----------|---------|
| Key Exchange | Curve25519 | X3DH protocol |
| Message Encryption | Megolm | Symmetric group cipher |
| Signing | Ed25519 | Message authentication |
| Hashing | SHA-256 | Key derivation |

### Key Types

| Key Type | Purpose | Lifetime |
|----------|---------|----------|
| Identity Key | Long-term identity | Permanent |
| Signed Pre-Key | Medium-term exchange | Weekly rotation |
| One-Time Pre-Keys | Single-use exchange | Used once |
| Megolm Session Key | Group message key | Per-room |

---

## Session Storage

### EncryptedSharedPreferences

```kotlin
// Location: MatrixSessionStorage.kt (androidMain)

class MatrixSessionStorage(context: Context) {
    private val masterKey = MasterKey.Builder(context)
        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
        .build()

    private val sharedPreferences = EncryptedSharedPreferences.create(
        context,
        "matrix_session",
        masterKey,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )
}
```

---

## Device Verification

### Verification Methods

| Method | Description | Security |
|--------|-------------|----------|
| Emoji SAS | Compare emoji sequence | Medium |
| QR Code | Scan QR code | High |

### Verification Flow

```
Device A                    Server                    Device B
   │                          │                          │
   │── Request Verification ─→│                          │
   │                          │── Forward Request ──────→│
   │                          │                          │
   │                          │←─ Accept ────────────────│
   │←─ Start SAS ─────────────│                          │
   │                          │                          │
   │════════ Compare Emojis ═════════════════════════════│
   │                          │                          │
   │── Confirm Match ─────────│── Confirm Match ────────→│
   │                          │                          │
   │←─ Verification Complete ─│                          │
```

---

## Trust Levels

| Level | Icon | Color | Description |
|-------|------|-------|-------------|
| UNVERIFIED | Warning | Yellow | New device, not verified |
| VERIFIED | Check | Green | Verified via SAS/QR |
| BLACKLISTED | X | Red | Explicitly blocked |

---

## Database Encryption

### SQLCipher

```kotlin
class EncryptedDatabase(context: Context, passphrase: ByteArray) {
    // PBKDF2 key derivation
    // 256-bit AES encryption
    // Hardware-backed key storage
}
```

---

## Network Security

### Certificate Pinning
- SHA-256 pins for homeserver
- Fallback pins for CDN
- Certificate transparency

### TLS Requirements
- TLS 1.3 minimum
- Strong cipher suites only
- HSTS enabled

---

## Key Backup Setup (NEW 2026-02-24)

ArmorChat now includes a mandatory key backup flow during onboarding to ensure encryption keys survive device loss.

### KeyBackupSetupScreen
**Location:** `onboarding/KeyBackupSetupScreen.kt`

6-step guided flow:
1. **Explain** — Why key backup matters
2. **Generate** — Create 12-word BIP39 recovery phrase (crypto-random)
3. **Display** — Show phrase; user writes it down
4. **Verify** — User confirms selected words
5. **Store** — Encrypted backup uploaded to homeserver (SSSS)
6. **Success** — Keys are recoverable

### Security Properties
- Recovery phrase generated client-side, never transmitted in plaintext
- Backup encrypted with key derived from recovery phrase
- Homeserver cannot decrypt backup (zero-knowledge)
- Phrase verification step prevents copy-paste shortcuts
- Mandatory during onboarding; optional re-entry from Security Settings

---

## Files

| File | Location |
|------|----------|
| MatrixClient | `platform/matrix/MatrixClient.kt` |
| MatrixSessionStorage | `platform/matrix/MatrixSessionStorage.kt` |
| EncryptionStatus | `ui/components/EncryptionStatus.kt` |
| KeyBackupSetupScreen | `screens/onboarding/KeyBackupSetupScreen.kt` |

---

## Related

- [Authentication](authentication.md)
- [Device Management](device-management.md)
- [Matrix Migration](../MATRIX_MIGRATION.md)
