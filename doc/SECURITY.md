# ArmorClaw Security Documentation

> **Last Updated:** 2026-02-18
> **Version:** 1.0.0

This document describes the security architecture, encryption model, and security features of ArmorClaw.

---

## Table of Contents

1. [Security Overview](#1-security-overview)
2. [Encryption Trust Model](#2-encryption-trust-model)
3. [Transport Security](#3-transport-security)
4. [Local Storage Security](#4-local-storage-security)
5. [Deep Link Security](#5-deep-link-security)
6. [Push Notification Security](#6-push-notification-security)
7. [Admin Role Assignment](#7-admin-role-assignment)
8. [Security Audit Checklist](#8-security-audit-checklist)

---

## 1. Security Overview

ArmorClaw is designed with security as a primary concern. The application implements multiple layers of security:

```
┌─────────────────────────────────────────────────────────────────────┐
│                     SECURITY LAYERS                                  │
├─────────────────────────────────────────────────────────────────────┤
│  Layer 1: Transport Security (TLS 1.3 + Certificate Pinning)        │
│  Layer 2: Message Encryption (AES-256-GCM)                          │
│  Layer 3: Database Encryption (SQLCipher 256-bit)                   │
│  Layer 4: Key Storage (AndroidKeyStore)                             │
│  Layer 5: Biometric Authentication                                  │
│  Layer 6: Deep Link Validation                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Encryption Trust Model

### 2.1 Overview

ArmorClaw supports two encryption trust models to accommodate different security requirements:

| Model | Trust Assumption | Key Management | Current Status |
|-------|-----------------|----------------|----------------|
| **Server-Side** | Trust Bridge Server | Server manages keys | ✅ Implemented |
| **Client-Side** | Trust only self | Client manages keys | 📋 Planned (vodozemac) |

### 2.2 Server-Side Encryption (Current)

```
┌─────────────────┐      ┌───────────────────────┐      ┌─────────────────┐
│    ArmorChat    │      │   ArmorClaw Go        │      │  Matrix Server  │
│    (Android)    │      │   Bridge Server       │      │                 │
├─────────────────┤      ├───────────────────────┤      ├─────────────────┤
│ No encryption   │─────▶│ libolm/libvodozemac   │─────▶│ Transport only  │
│ keys stored     │      │ Per-user container    │      │ TLS 1.3         │
│ Trusts Bridge   │      │ Full E2EE             │      │                 │
└─────────────────┘      └───────────────────────┘      └─────────────────┘
```

**How it works:**
1. ArmorChat sends plaintext message to Bridge Server
2. Bridge Server encrypts message using Matrix E2EE (libolm)
3. Encrypted message sent to Matrix homeserver
4. Recipients receive encrypted message
5. Recipient's Bridge Server decrypts and forwards to app

**Security considerations:**
- User must trust the Bridge Server operator
- Keys are managed securely in per-user containers
- No key backup required from user perspective
- Simpler user experience

### 2.3 Client-Side Encryption (Future)

```
┌─────────────────┐      ┌───────────────────────┐      ┌─────────────────┐
│    ArmorChat    │      │   ArmorClaw Go        │      │  Matrix Server  │
│    (Android)    │      │   Bridge Server       │      │                 │
├─────────────────┤      ├───────────────────────┤      ├─────────────────┤
│ vodozemac       │─────▶│ Transport only        │─────▶│ Transport only  │
│ Keys in Keystore│      │ No access to content  │      │ TLS 1.3         │
│ Zero-trust      │      │                       │      │                 │
└─────────────────┘      └───────────────────────┘      └─────────────────┘
```

**Implementation path:**
- Integrate vodozemac (Matrix Rust SDK crypto)
- Keys stored in AndroidKeyStore
- User responsible for key backup
- Maximum security, zero-trust architecture

### 2.4 EncryptionStatus Interface

```kotlin
enum class EncryptionMode {
    SERVER_SIDE,    // Bridge handles encryption
    CLIENT_SIDE,    // Client handles encryption
    NONE            // No encryption
}

sealed class RoomEncryptionStatus {
    object ServerEncrypted : RoomEncryptionStatus()
    object ClientEncrypted : RoomEncryptionStatus()
    object Unencrypted : RoomEncryptionStatus()
    object Unknown : RoomEncryptionStatus()
}
```

---

## 3. Transport Security

### 3.1 TLS Configuration

- **Minimum Version:** TLS 1.3
- **Cipher Suites:** Modern AEAD ciphers only
- **Certificate Validation:** Full chain validation

### 3.2 Certificate Pinning

ArmorClaw implements certificate pinning to prevent MITM attacks:

```kotlin
// Known pins for production servers
val PINS = setOf(
    "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", // Primary
    "sha256/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="  // Backup
)
```

### 3.3 Network Security Configuration

```xml
<!-- Android Network Security Config -->
<network-security-config>
    <domain-config>
        <domain includeSubdomains="true">bridge.armorclaw.app</domain>
        <pin-set>
            <pin digest="SHA-256">primary_pin</pin>
            <pin digest="SHA-256">backup_pin</pin>
        </pin-set>
        <trust-anchors>
            <certificates src="system"/>
        </trust-anchors>
    </domain-config>
</network-security-config>
```

---

## 4. Local Storage Security

### 4.1 Database Encryption

- **Technology:** SQLCipher (SQLite with AES-256 encryption)
- **Key Storage:** AndroidKeyStore
- **Key Derivation:** PBKDF2 with random salt

```kotlin
// Database configuration
val config = DatabaseConfiguration(
    name = "armorclaw.db",
    password = deriveKeyFromUserCredentials(),
    cipher = CipherAlgorithm.AES_256_GCM
)
```

### 4.2 Secure Storage

| Data Type | Storage Location | Encryption |
|-----------|-----------------|------------|
| Database | SQLCipher | AES-256 |
| Access Tokens | EncryptedSharedPreferences | AndroidKeyStore |
| Refresh Tokens | EncryptedSharedPreferences | AndroidKeyStore |
| Private Keys | AndroidKeyStore | Hardware-backed |
| Session Data | EncryptedSharedPreferences | AES-256 |

---

## 5. Deep Link Security

### 5.1 Threat Model

Deep links can be triggered by:
- Other apps on the device
- Websites via browser intents
- QR codes
- Email links

**Attacks mitigated:**
- Phishing (fake room invites)
- CSRF (unauthorized actions)
- XSS (via malicious parameters)

### 5.2 Validation Pipeline

```
Deep Link Received
       │
       ▼
┌─────────────────────┐
│ 1. Scheme Validation│ ─── Reject non-armorclaw/whitelisted HTTPS
└─────────────────────┘
       │
       ▼
┌─────────────────────┐
│ 2. Host Validation  │ ─── Reject unknown HTTPS hosts
└─────────────────────┘
       │
       ▼
┌─────────────────────┐
│ 3. Length Check     │ ─── Reject URIs > 2048 chars
└─────────────────────┘
       │
       ▼
┌─────────────────────┐
│ 4. Path Validation  │ ─── Reject .. and backslash
└─────────────────────┘
       │
       ▼
┌─────────────────────┐
│ 5. ID Format Check  │ ─── Validate Matrix ID format
└─────────────────────┘
       │
       ▼
┌─────────────────────┐
│ 6. Security State   │ ─── Determine confirmation needed
└─────────────────────┘
```

### 5.3 Security States

| State | Behavior | Example |
|-------|----------|---------|
| `Valid` | Navigate immediately | Settings link |
| `RequiresConfirmation` | Show dialog, require user OK | Room join |
| `Invalid` | Reject, log reason | Malformed URI |

### 5.4 Confirmation Dialog

```kotlin
DeepLinkConfirmationDialog(
    action = NavigateToRoom("!abc123:matrix.org"),
    securityCheck = ROOM_MEMBERSHIP,
    message = "Join room?",
    details = "Only join rooms from trusted sources.",
    onConfirm = { /* proceed */ },
    onDismiss = { /* cancel */ }
)
```

---

## 6. Push Notification Security

### 6.1 Architecture (Dual Registration)

ArmorChat registers FCM tokens with **both** the Matrix Homeserver and the Bridge Server:

```
┌──────────────┐                                                      
│  ArmorChat   │                                                      
│  (App)       │                                                      
└──────┬───────┘                                                      
       │                                                              
       ├── 1a. MatrixClient.setPusher() ──▶ Matrix Homeserver ──▶ Sygnal ──▶ FCM
       │       (Matrix-native events: mentions, DMs, invites)         
       │                                                              
       └── 1b. BridgeRpcClient.pushRegister() ──▶ Bridge Server ──▶ FCM
               (SDTW bridging events: Slack, Discord, etc.)           
```

**Why dual registration?**
- Matrix events originate from the homeserver — it needs its own pusher registration
- Bridge events (SDTW) originate from the bridge — it needs its own FCM token
- Partial failure is handled: if one registration fails, the other still delivers push

### 6.2 Token Registration

```kotlin
// Dual FCM token registration flow
suspend fun registerToken(pushToken: String, ...) {
    // Step 1: Register with Matrix Homeserver (primary push path)
    matrixClient.setPusher(
        pushKey = pushToken,
        appId = "com.armorclaw.app",
        pushGatewayUrl = "https://push.armorclaw.app/_matrix/push/v1/notify"
    )

    // Step 2: Register with Bridge Server (SDTW bridging notifications)
    rpcClient.pushRegister(pushToken, "fcm", deviceId)
}
```

### 6.3 Token Cleanup

```kotlin
// On logout — unregister from both channels
suspend fun unregisterToken() {
    matrixClient.removePusher(pushKey, appId)
    rpcClient.pushUnregister(pushToken)
}
```

### 6.4 Security Considerations

- **Token Scope:** Token is scoped to user and device
- **Token Rotation:** Tokens are rotated on app reinstall
- **Token Cleanup:** Tokens unregistered from both Matrix and Bridge on logout
- **Payload Encryption:** Push payload contains only metadata (room ID, event ID)
- **Content Protection:** Actual message content NOT included in push
- **Dual Channel Resilience:** If Matrix pusher fails, Bridge push still works (and vice versa)

---

## 7. Admin Role Assignment

### 7.1 Problem

Client-side admin detection based on `messageCount == 0` creates race conditions:
- Multiple users connecting simultaneously
- Network latency causing inconsistent state
- No authoritative source of truth

### 7.2 Solution: Server-Authoritative Roles

The Bridge Server is the authoritative source for user roles:

```kotlin
// Bridge status response includes user role
data class BridgeStatusResponse(
    val userRole: AdminLevel?,  // Server-assigned role
    val isNewServer: Boolean?   // First user becomes OWNER
)
```

### 7.3 Role Hierarchy

| Level | Assigned By | Permissions |
|-------|-------------|-------------|
| `OWNER` | First user on new server | All admin + server config |
| `ADMIN` | Existing OWNER/ADMIN | Invite, room management |
| `MODERATOR` | Existing ADMIN | Kick, mute users |
| `NONE` | Default | Basic messaging |

### 7.4 Implementation

```kotlin
// Server-side role check
suspend fun getUserPrivileges(): AdminLevel {
    val response = bridgeRpcClient.bridgeStatus()
    return response.userRole ?: AdminLevel.NONE
}
```

---

## 8. Key Backup Security (NEW 2026-02-24)

### 8.1 Recovery Phrase

ArmorChat now includes a mandatory key backup setup during onboarding:

- **12-word BIP39 recovery phrase** generated client-side
- Phrase displayed once and user must verify selected words
- Encrypted backup uploaded to Matrix homeserver (SSSS)
- Recovery phrase is the only way to restore encryption keys after device loss

### 8.2 Key Backup Flow

```
1. Generate → 12-word recovery phrase (crypto-random)
2. Display  → User writes down phrase
3. Verify   → User confirms selected words
4. Store    → Encrypted backup to homeserver
5. Success  → Keys are recoverable
```

### 8.3 Security Properties

- Recovery phrase never transmitted in plaintext
- Backup encrypted with key derived from recovery phrase
- Homeserver cannot decrypt backup (zero-knowledge)
- Phrase verification prevents copy-paste shortcuts

---

## 9. Security Audit Checklist

### 9.1 Pre-Release Checklist

#### Data Protection
- [x] Database encrypted with SQLCipher (256-bit)
- [x] Sensitive data in AndroidKeyStore
- [x] No plaintext credentials in logs
- [x] Secure deletion of sensitive data
- [x] Memory cleared after use

#### Network Security
- [x] TLS 1.3 for all connections
- [x] Certificate pinning enabled
- [x] No mixed HTTP/HTTPS content
- [x] Proper certificate validation

#### Authentication
- [x] Secure password handling (no logging)
- [x] Token secure storage
- [x] Session timeout implemented
- [x] Biometric fallback available

#### Encryption
- [x] Encryption trust model documented
- [x] Server-side E2EE via Bridge
- [x] Key rotation implemented (server-side)
- [x] Perfect forward secrecy

#### Deep Links
- [x] URI validation implemented
- [x] Confirmation dialogs for sensitive actions
- [x] ID format validation
- [x] Known host whitelist

#### Push Notifications
- [x] FCM token registration
- [x] Token cleanup on logout
- [x] No message content in push payload
- [x] Secure token storage

### 9.2 OWASP Mobile Top 10 Compliance

| Risk | Status | Mitigation |
|------|--------|------------|
| M1: Improper Platform Usage | ✅ | Proper intent handling, secure deep links |
| M2: Insecure Data Storage | ✅ | SQLCipher, AndroidKeyStore |
| M3: Insecure Communication | ✅ | TLS 1.3, Certificate Pinning |
| M4: Insecure Authentication | ✅ | Biometric, secure token storage |
| M5: Insufficient Cryptography | ✅ | AES-256-GCM, proper key management |
| M6: Insecure Authorization | ✅ | Server-authoritative roles |
| M7: Client Code Quality | ✅ | Clean architecture, code review |
| M8: Code Tampering | ✅ | ProGuard, signature verification |
| M9: Reverse Engineering | ✅ | Code obfuscation, native crypto |
| M10: Extraneous Functionality | ✅ | Debug logging disabled in release |

---

## Changelog

### 2026-02-24
- Updated push notification architecture to dual registration (Matrix + Bridge)
- Added Key Backup Security section
- Renumbered audit checklist sections

### 2026-02-18
- Added Encryption Trust Model documentation
- Added Deep Link Security documentation
- Added Push Notification Security documentation
- Added Admin Role Assignment documentation
- Updated security audit checklist

---

*For implementation details, see the source code in `shared/src/commonMain/kotlin/platform/` and `androidApp/src/main/kotlin/com/armorclaw/app/`.*
