# Screenshot Capture Guide

> ArmorChat - Play Store Screenshots
> Minimum: 2 screenshots | Recommended: 4-8 screenshots

---

## 1. Specifications

### Phone Screenshots (Required)

| Property | Requirement |
|----------|-------------|
| **Format** | PNG or JPEG |
| **Min dimensions** | 320 × 320 px |
| **Max dimensions** | 3840 × 3840 px |
| **Aspect ratio** | 16:9, 20:9, or device-native |
| **Max file size** | 8MB per image |
| **Min quantity** | 2 screenshots |
| **Recommended** | 4-8 screenshots |

### Recommended Resolutions

| Device Type | Resolution | Aspect Ratio |
|-------------|------------|--------------|
| **Modern phones** | 1080 × 2400 px | 20:9 |
| **Standard phones** | 1080 × 1920 px | 16:9 |
| **Pixel phones** | 1080 × 2340 px | 19.5:9 |
| **Samsung phones** | 1440 × 3200 px | 20:9 |

---

## 2. Screenshot List

### Required Screenshots (Minimum 4)

| # | Screen | Purpose | Caption |
|---|--------|---------|---------|
| 1 | **Login/Biometric** | Show security | "Secure biometric login" |
| 2 | **Home/Room List** | Show conversations | "Organized team chats" |
| 3 | **Chat Screen** | Show messaging | "End-to-end encrypted messages" |
| 4 | **Settings** | Show customization | "Comprehensive settings" |

### Recommended Additional (4 more)

| # | Screen | Purpose | Caption |
|---|--------|---------|---------|
| 5 | **Message Features** | Show reactions/replies | "Rich messaging features" |
| 6 | **Dark Mode** | Show theme option | "Dark mode available" |
| 7 | **Profile** | Show user settings | "Profile management" |
| 8 | **Room Creation** | Show team features | "Create and manage rooms" |

---

## 3. Capture Method

### Option A: Android Studio Emulator

1. **Launch emulator** with desired device/skin
2. **Run app:** `./gradlew installDebug`
3. **Navigate** to desired screen
4. **Capture:** Click camera icon in emulator toolbar
5. **Save:** Screenshots save to `~/Desktop/` or specified location

```bash
# Using adb for clean screenshots
adb shell screencap -p /sdcard/screen.png
adb pull /sdcard/screen.png screenshot-01.png
```

### Option B: Physical Device

1. **Connect device** via USB with debugging enabled
2. **Install app:** `./gradlew installDebug`
3. **Navigate** to desired screen
4. **Capture:** Power + Volume Down (varies by device)
5. **Transfer:** `adb pull /sdcard/Pictures/Screenshots/ ./screenshots/`

### Option C: Screenshot Tools

| Tool | Platform | Features |
|------|----------|----------|
| **Scrcpy** | Cross-platform | Mirror + capture |
| **Android Studio** | Desktop | Device frames |
| **Firebase Test Lab** | Cloud | Automated screenshots |

---

## 4. Screenshot Preparation

### Screen States to Prepare

#### 1. Login/Biometric Screen
```
Setup:
- Show biometric prompt dialog
- OR show login form with "Use biometric" button visible
- Ensure encryption/security indicator is visible

Do:
✓ Show security prominently
✓ Clean, professional look
✓ No personal data visible

Don't:
✗ Show error states
✗ Show loading states
```

#### 2. Home/Room List
```
Setup:
- Create 4-6 sample chat rooms
- Use generic room names: "Engineering", "Design", "General"
- Show mix of read/unread states
- Show timestamp variety

Sample Data:
┌──────────────────────────────┐
│ 🔒 Engineering        2m ago │
│ "Thanks for the update!"     │
├──────────────────────────────┤
│ 🔒 Design Team        1h ago │
│ "Let's review the mockups"   │
├──────────────────────────────┤
│ 🔒 Company Updates    3h ago │
│ "Welcome to the team!"       │
└──────────────────────────────┘
```

#### 3. Chat Screen
```
Setup:
- Show conversation with 5-8 messages
- Include variety: text, reaction, reply
- Show encryption indicator (🔒)
- Show typing indicator optional

Sample Messages:
┌─────────────────────────────────┐
│ 🔒 Engineering                  │
│                                 │
│         "Let's sync on the     │
│          security audit"  10:30│
│                                 │
│ "Sounds good! I've prepared   │
│  the documentation"      10:31│
│                                 │
│         "Perfect, see you     │
│          then" 👍  10:32       │
│                                 │
│ Type a message...          [📎]│
└─────────────────────────────────┘
```

#### 4. Settings Screen
```
Setup:
- Show security section expanded
- Highlight privacy options
- Show theme toggle (dark/light)

Key Settings Visible:
- Security & Privacy
- Notifications
- Appearance (Dark Mode)
- Account
```

#### 5. Dark Mode Versions
```
Repeat screenshots 1-4 in dark mode
- Toggle: Settings → Appearance → Dark
- Ensures consistency
- Shows theme capability
```

---

## 5. Post-Processing

### Cropping & Sizing

```bash
# Using ImageMagick to resize
convert screenshot-raw.png -resize 1080x2400 screenshot-phone.png

# Crop status bar (optional, cleaner look)
convert screenshot-raw.png -crop 1080x2340+0+60 screenshot-cropped.png
```

### Enhancement (Optional)

| Adjustment | Value | Purpose |
|------------|-------|---------|
| Brightness | +2-5% | Slightly brighter |
| Contrast | +5-10% | More vibrant |
| Saturation | +5% | Colors pop |

### Adding Captions (Optional)

```
┌─────────────────────────────────┐
│                                 │
│     [Screenshot Content]        │
│                                 │
│                                 │
│  ┌───────────────────────────┐  │
│  │   "Secure Biometric Login" │  │
│  └───────────────────────────┘  │
└─────────────────────────────────┘
```

**Caption style:**
- Font: Roboto Medium
- Size: 32-48px
- Color: White on semi-transparent dark bar
- Position: Bottom of screenshot

---

## 6. Device Frames (Optional)

### With Device Frame

```
┌─────────────────────────┐
│ ┌─────────────────────┐ │
│ │                     │ │
│ │  [Screenshot]       │ │
│ │                     │ │
│ │                     │ │
│ └─────────────────────┘ │
│         ▢ ▢ ▢           │
└─────────────────────────┘
```

**Pros:** Shows device context
**Cons:** Wastes space, dates quickly

### Without Frame (Recommended)

```
┌─────────────────────────┐
│                         │
│  [Screenshot]           │
│                         │
│                         │
│                         │
└─────────────────────────┘
```

**Pros:** Clean, maximizes content
**Cons:** Less context

**Recommendation:** No frame - let content speak

---

## 7. File Naming Convention

```
screenshot-[number]-[screen]-[theme].png

Examples:
screenshot-01-login-light.png
screenshot-02-home-light.png
screenshot-03-chat-light.png
screenshot-04-settings-light.png
screenshot-05-chat-dark.png
screenshot-06-home-dark.png
```

---

## 8. Quality Checklist

Before uploading:

- [ ] All screenshots same resolution/aspect ratio
- [ ] No personal information visible
- [ ] No debug/developer options visible
- [ ] Status bar clean (no notifications, low battery)
- [ ] Time set to reasonable value (10:30, 2:45, etc.)
- [ ] Network signal showing (not airplane mode)
- [ ] Encryption indicators visible (🔒)
- [ ] Sample data is professional (no "test", "asdf")
- [ ] No loading spinners or empty states
- [ ] Consistent theme per screenshot set
- [ ] File size under 8MB each

---

## 9. Play Store Upload

### Upload Location
```
Play Console → Main Store Listing → Phone screenshots
```

### Order
Upload in logical flow order:
1. Login → 2. Home → 3. Chat → 4. Settings → 5. Features → 6. Dark Mode

### Localization (Optional)
Upload different screenshots for different languages if UI text differs.

---

## 10. Screenshot Template (ASCII)

### Light Theme

```
╔══════════════════════════════════╗
║  10:30              🔋 85%  📶  ║
╠══════════════════════════════════╣
║                                  ║
║     ┌────────────────────┐       ║
║     │                    │       ║
║     │    🔐              │       ║
║     │                    │       ║
║     │  Verify Identity   │       ║
║     │                    │       ║
║     │  Touch the sensor  │       ║
║     │                    │       ║
║     │    [○ ○ ○ ○]       │       ║
║     │                    │       ║
║     │                    │       ║
║     └────────────────────┘       ║
║                                  ║
║         Use passcode instead     ║
║                                  ║
╚══════════════════════════════════╝
```

### Dark Theme

```
╔══════════════════════════════════╗
║  10:30              🔋 85%  📶  ║  ← Dark status bar
╠══════════════════════════════════╣
║  🔒 Engineering Team             ║  ← Dark background
║  ────────────────────────────    ║
║                                  ║
║         "Thanks for the         ║  ← Message bubbles
│          update!"      10:30    ║     adapted for
║                                  ║     dark mode
║  "Sure thing! I've prepared     ║
║   the docs"             10:31   ║
║                                  ║
║         "Great! See you        ║
│          tomorrow" 👍   10:32   ║
║                                  ║
║  ──────────────────────────────  ║
║  Type a message...        📎 📷 │
╚══════════════════════════════════╝
```

---

## 11. Quick Capture Script

```bash
#!/bin/bash
# capture-screenshots.sh

# Ensure device connected
adb devices

# List of screens to capture (navigate manually between)
SCREENS=("login" "home" "chat" "settings" "chat-dark" "home-dark")

for screen in "${SCREENS[@]}"; do
    echo "Navigate to $screen screen, then press Enter..."
    read

    timestamp=$(date +%s)
    adb shell screencap -p /sdcard/screenshot-$timestamp.png
    adb pull /sdcard/screenshot-$timestamp.png screenshots/screenshot-$screen.png
    adb shell rm /sdcard/screenshot-$timestamp.png

    echo "Captured: screenshot-$screen.png"
done

echo "All screenshots captured in screenshots/"
```

---

## 12. Output Files

**Save to:** `play-store/listing/screenshots/`

```
screenshots/
├── screenshot-01-login.png
├── screenshot-02-home.png
├── screenshot-03-chat.png
├── screenshot-04-settings.png
├── screenshot-05-chat-dark.png
└── screenshot-06-home-dark.png
```

---

*Screenshot guide for ArmorChat Play Store submission*
