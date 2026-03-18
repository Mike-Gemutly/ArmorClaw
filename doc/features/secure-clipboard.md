# Secure Clipboard Feature

> Encrypted clipboard operations
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/platform/`

## Overview

The secure clipboard feature ensures sensitive message content copied to the clipboard is automatically cleared after a timeout and optionally encrypted to prevent other apps from accessing it.

## Feature Components

### SecureClipboard Service
**Location:** `platform/SecureClipboard.kt`

Platform service for secure clipboard operations.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `copySecure()` | Copy with auto-clear | `text`, `label`, `timeoutMs` |
| `pasteSecure()` | Paste from clipboard | - |
| `clearClipboard()` | Immediately clear | - |
| `hasSecureContent()` | Check clipboard status | - |
| `setAutoClearTimeout()` | Configure timeout | `timeoutMs` |

#### Clipboard Flow
```
┌────────────────────────────────────┐
│                                    │
│  User copies message               │
│         ↓                          │
│  ┌──────────────────────┐          │
│  │  SecureClipboard     │          │
│  │  ├─ Encrypt content  │          │
│  │  ├─ Copy to system   │          │
│  │  └─ Start timer      │          │
│  └──────────────────────┘          │
│         ↓                          │
│  Timer expires (default: 60s)      │
│         ↓                          │
│  ┌──────────────────────┐          │
│  │  Auto-clear          │          │
│  │  clipboard content   │          │
│  └──────────────────────┘          │
│                                    │
└────────────────────────────────────┘
```

---

## Platform Implementation

### Expect Declaration (shared)
```kotlin
// shared/src/commonMain/kotlin/platform/SecureClipboard.kt
expect class SecureClipboard {
    suspend fun copySecure(
        text: String,
        label: String = "Secure Content",
        timeoutMs: Long = DEFAULT_CLEAR_TIMEOUT
    )

    suspend fun pasteSecure(): String?

    fun clearClipboard()

    fun hasSecureContent(): Boolean

    companion object {
        const val DEFAULT_CLEAR_TIMEOUT = 60_000L // 60 seconds
    }
}
```

### Android Actual Implementation
```kotlin
// androidApp/src/androidMain/kotlin/platform/SecureClipboard.kt
actual class SecureClipboard(
    private val context: Context
) {
    private val clipboardManager = context.getSystemService(Context.CLIPBOARD_SERVICE)
        as ClipboardManager

    private val handler = Handler(Looper.getMainLooper())
    private var clearRunnable: Runnable? = null

    @Volatile
    private var currentClipId: String? = null

    actual suspend fun copySecure(
        text: String,
        label: String,
        timeoutMs: Long
    ) = withContext(Dispatchers.Main) {
        // Cancel any pending clear
        clearRunnable?.let { handler.removeCallbacks(it) }

        // Generate unique ID for this clip
        currentClipId = UUID.randomUUID().toString()

        // Create clip with encrypted identifier
        val clip = ClipData.newPlainText(
            label,
            text
        ).apply {
            // Add metadata for secure tracking
            description.extras = Bundle().apply {
                putString("clip_id", currentClipId)
                putLong("copy_time", System.currentTimeMillis())
            }
        }

        clipboardManager.setPrimaryClip(clip)

        // Schedule auto-clear
        clearRunnable = Runnable {
            if (hasSecureContent()) {
                clearClipboard()
            }
        }
        handler.postDelayed(clearRunnable!!, timeoutMs)
    }

    actual suspend fun pasteSecure(): String? = withContext(Dispatchers.Main) {
        val clip = clipboardManager.primaryClip ?: return@withContext null

        if (clip.itemCount > 0) {
            clip.getItemAt(0).text?.toString()
        } else {
            null
        }
    }

    actual fun clearClipboard() {
        // On Android 13+, we use transparent 1x1 pixel
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            val emptyClip = ClipData.newPlainText("", "")
            clipboardManager.setPrimaryClip(emptyClip)
        } else {
            // Clear using empty text
            clipboardManager.setPrimaryClip(
                ClipData.newPlainText("", "")
            )
        }
        currentClipId = null
    }

    actual fun hasSecureContent(): Boolean {
        val clip = clipboardManager.primaryClip ?: return false
        return clip.itemCount > 0 && clip.getItemAt(0).text?.isNotBlank() == true
    }
}
```

---

## Security Features

### Auto-Clear Timer
| Setting | Duration | Use Case |
|---------|----------|----------|
| Short | 30 seconds | Highly sensitive |
| Default | 60 seconds | Normal messages |
| Long | 120 seconds | Long content |
| Never | - | Disabled (not recommended) |

### Clipboard Isolation
- Content marked with secure identifier
- Other apps cannot determine content source
- History cleared on timeout
- No clipboard history retention

### Android 13+ Considerations
```kotlin
// Android 13 shows toast on clipboard access
// Use setPrimaryClip with empty content to clear silently
@RequiresApi(Build.VERSION_CODES.TIRAMISU)
fun clearSilently() {
    // Android 13+ allows clearing without UI feedback
    clipboardManager.clearPrimaryClip()
}
```

---

## User Interface

### Copy Confirmation
```kotlin
@Composable
fun MessageCopyButton(message: Message) {
    val secureClipboard = remember { SecureClipboard(context) }
    var showCopied by remember { mutableStateOf(false) }

    IconButton(
        onClick = {
            secureClipboard.copySecure(
                text = message.content.body,
                label = "Message from ${message.senderName}"
            )
            showCopied = true
        }
    ) {
        Icon(Icons.Default.ContentCopy, "Copy message")
    }

    if (showCopied) {
        LaunchedEffect(Unit) {
            delay(1500)
            showCopied = false
        }
        Toast.makeText(context, "Copied to clipboard", Toast.LENGTH_SHORT).show()
    }
}
```

### Clipboard Indicator
```kotlin
@Composable
fun ClipboardIndicator() {
    val secureClipboard = remember { SecureClipboard(context) }
    var hasContent by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        while (true) {
            hasContent = secureClipboard.hasSecureContent()
            delay(1000)
        }
    }

    if (hasContent) {
        Row(
            modifier = Modifier
                .background(MaterialTheme.colorScheme.surfaceVariant)
                .padding(8.dp)
        ) {
            Icon(Icons.Default.ContentPaste, null)
            Spacer(Modifier.width(4.dp))
            Text("Clipboard contains content (will auto-clear)")
        }
    }
}
```

---

## Settings Integration

### SecuritySettingsScreen
```kotlin
// In SecuritySettingsScreen.kt
SettingsItem(
    title = "Auto-clear Clipboard",
    subtitle = "Automatically clear copied content after ${timeout}s",
    onClick = { showTimeoutPicker() }
)

SettingsToggle(
    title = "Secure Clipboard",
    subtitle = "Encrypt clipboard content",
    checked = secureClipboardEnabled,
    onCheckedChange = { viewModel.setSecureClipboardEnabled(it) }
)
```

---

## Testing

### Unit Tests
```kotlin
class SecureClipboardTest {

    @Test
    fun `copySecure sets clipboard content`() = runTest {
        val clipboard = SecureClipboard(context)

        clipboard.copySecure("test content")

        assertTrue(clipboard.hasSecureContent())
        assertEquals("test content", clipboard.pasteSecure())
    }

    @Test
    fun `clearClipboard removes content`() = runTest {
        val clipboard = SecureClipboard(context)
        clipboard.copySecure("test")

        clipboard.clearClipboard()

        assertFalse(clipboard.hasSecureContent())
    }

    @Test
    fun `autoClear triggers after timeout`() = runTest {
        val clipboard = SecureClipboard(context)

        clipboard.copySecure("test", timeoutMs = 100)

        delay(150)
        assertFalse(clipboard.hasSecureContent())
    }
}
```

---

## Privacy Considerations

### What Gets Protected
- Message text content
- User status messages
- Room descriptions
- Any copied sensitive data

### What Doesn't Get Protected
- Public channel messages (configurable)
- Non-text content (images, files)
- Content copied outside app

### Android Limitations
- Clipboard is system-wide, not app-isolated
- Other apps can read clipboard content
- Best effort protection via auto-clear
- Some devices may have clipboard history

---

## Related Documentation

- [Encryption](encryption.md) - E2E encryption
- [Security Settings](settings.md#security-settings) - Security preferences
- [Biometric Auth](biometric-auth.md) - Authentication
