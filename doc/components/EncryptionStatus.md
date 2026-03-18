# EncryptionStatus Component

> Visual encryption indicator for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/chat/components/EncryptionStatus.kt`

## Overview

EncryptionStatus provides visual feedback about the encryption state of messages and conversations, helping users understand their security level.

## Functions

### EncryptionStatus (Enum)
```kotlin
enum class EncryptionStatus {
    NONE,        // Encryption not available
    UNENCRYPTED, // Message not encrypted
    UNVERIFIED,  // Encrypted but device unverified
    VERIFIED     // Fully encrypted and verified
}
```

**Description:** Represents the encryption state of a conversation.

---

### EncryptionStatusIndicator
```kotlin
@Composable
fun EncryptionStatusIndicator(
    status: EncryptionStatus,
    modifier: Modifier = Modifier,
    showText: Boolean = false
)
```

**Description:** Compact indicator showing encryption state with icon.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | `EncryptionStatus` | Current encryption state |
| `modifier` | `Modifier` | Optional styling |
| `showText` | `Boolean` | Show status text |

**Visual Output:**
| Status | Icon | Color | Text |
|--------|------|-------|------|
| NONE | Info | Gray | "Encryption not available" |
| UNENCRYPTED | LockOpen | Red | "Not encrypted" |
| UNVERIFIED | Lock | Yellow | "Encrypted (unverified)" |
| VERIFIED | VerifiedUser | Green | "Verified encryption" |

---

### EncryptionStatusCard
```kotlin
@Composable
fun EncryptionStatusCard(
    status: EncryptionStatus,
    title: String,
    description: String,
    color: Color,
    modifier: Modifier = Modifier
)
```

**Description:** Expanded card with detailed encryption information.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | `EncryptionStatus` | Current encryption state |
| `title` | `String` | Status title |
| `description` | `String` | Detailed description |
| `color` | `Color` | Accent color |
| `modifier` | `Modifier` | Optional styling |

**Layout:**
```
┌────────────────────────────────────┐
│ 🔒 Verified Encryption             │
│                                    │
│ Messages are end-to-end encrypted  │
│ and verified with trusted devices. │
│                                    │
│ [Learn More]                       │
└────────────────────────────────────┘
```

---

### EncryptionStatusBanner
```kotlin
@Composable
fun EncryptionStatusBanner(
    status: EncryptionStatus,
    onActionClick: () -> Unit,
    modifier: Modifier = Modifier
)
```

**Description:** Warning banner for unencrypted/unverified states.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | `EncryptionStatus` | Current encryption state |
| `onActionClick` | `() -> Unit` | Action button callback |
| `modifier` | `Modifier` | Optional styling |

**Use Cases:**
- UNENCRYPTED: "Messages are not encrypted. Tap to learn more."
- UNVERIFIED: "Device not verified. Tap to verify."

---

### StatusIcon
```kotlin
@Composable
private fun StatusIcon(
    status: EncryptionStatus,
    tint: Color,
    modifier: Modifier = Modifier
)
```

**Description:** Renders the appropriate icon for the status.

**Icons by Status:**
| Status | ImageVector |
|--------|-------------|
| NONE | Icons.Outlined.Info |
| UNENCRYPTED | Icons.Default.LockOpen |
| UNVERIFIED | Icons.Default.Lock |
| VERIFIED | Icons.Default.VerifiedUser |

---

## Status Details

### Status Information
```kotlin
fun getStatusInfo(status: EncryptionStatus): Triple<String, String, Color> {
    return when (status) {
        EncryptionStatus.NONE -> Triple(
            "Encryption Not Available",
            "This conversation cannot be encrypted.",
            Color.Gray
        )
        EncryptionStatus.UNENCRYPTED -> Triple(
            "Not Encrypted",
            "Messages are sent in plain text. Avoid sharing sensitive information.",
            BrandRed
        )
        EncryptionStatus.UNVERIFIED -> Triple(
            "Encrypted (Unverified)",
            "Messages are encrypted but the recipient's device has not been verified.",
            StatusWarning
        )
        EncryptionStatus.VERIFIED -> Triple(
            "Verified Encryption",
            "Messages are end-to-end encrypted and verified with trusted devices.",
            BrandGreen
        )
    }
}
```

---

## Color Scheme

| Status | Primary Color | Background |
|--------|---------------|------------|
| NONE | Gray | Gray.copy(0.1) |
| UNENCRYPTED | BrandRed | BrandRed.copy(0.1) |
| UNVERIFIED | StatusWarning | StatusWarning.copy(0.1) |
| VERIFIED | BrandGreen | BrandGreen.copy(0.1) |

---

## Usage Examples

### In Top Bar
```kotlin
@Composable
fun ChatTopBar(encryptionStatus: EncryptionStatus) {
    TopAppBar(
        actions = {
            EncryptionStatusIndicator(
                status = encryptionStatus,
                showText = false
            )
        }
    )
}
```

### In Message Header
```kotlin
@Composable
fun MessageHeader(message: Message) {
    Row {
        Text(message.senderName)
        Spacer(Modifier.width(4.dp))
        EncryptionStatusIndicator(
            status = message.encryptionStatus,
            modifier = Modifier.size(16.dp)
        )
    }
}
```

### As Warning Banner
```kotlin
@Composable
fun ChatContent(encryptionStatus: EncryptionStatus) {
    Column {
        if (encryptionStatus != EncryptionStatus.VERIFIED) {
            EncryptionStatusBanner(
                status = encryptionStatus,
                onActionClick = { /* Navigate to verification */ }
            )
        }
        MessageList(...)
    }
}
```

---

## Accessibility

### Content Descriptions
- NONE: "Encryption not available"
- UNENCRYPTED: "Warning: Message not encrypted"
- UNVERIFIED: "Encrypted but device not verified"
- VERIFIED: "Verified encryption active"

---

## Related Documentation

- [Encryption Feature](../features/encryption.md) - Encryption implementation
- [Device Management](../features/device-management.md) - Device verification
- [ChatScreen](../screens/ChatScreen.md) - Chat screen
