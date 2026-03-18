# Play Store Assets - Ready for Production

> **Status:** ✅ READY FOR SUBMISSION
> **Date:** 2026-03-17
> **App Version:** 1.0.0 (versionCode: 10000)

---

## ✅ Completed Assets

| Asset | Status | Location |
|-------|--------|----------|
| App Title | ✅ | "ArmorChat - Secure Team Chat" |
| Short Description | ✅ | play-store/store-listing.md |
| Full Description | ✅ | play-store/store-listing.md |
| Release Notes | ✅ | play-store/release-notes.md |
| Privacy Policy | ✅ | play-store/privacy-policy.md |
| Data Safety Form | ✅ | play-store/data-safety-form.md |
| Content Rating Answers | ✅ | play-store/metadata-compliance.md |
| Submission Checklist | ✅ | play-store/SUBMISSION-CHECKLIST.md |

---

## 📋 Pending Assets (Require Device/Design)

### 1. Feature Graphic (1024x500 px)

**Specifications:**
- Dimensions: 1024 x 500 pixels
- Format: PNG or JPEG (24-bit, no alpha)
- File size: < 1MB recommended

**Design Brief:**
```
Background: Dark gradient (Navy #0A1428 to #1A237E)
Left side: App icon (128px) with elevation/shadow
Center: "ArmorChat" in bold white text (Inter font)
Below title: "Secure Team Chat" in lighter weight
Right side: Feature icons (lock, shield, key) with labels
```

**File:** `play-store/assets/feature-graphic.png`

---

### 2. Phone Screenshots (4-8 required)

**Specifications:**
- Format: PNG or JPEG
- Aspect ratio: 16:9 or device-native (20:9)
- Recommended: 1080x1920 or 1440x2560

**Required Screenshots:**

| # | Screen | Caption |
|---|--------|---------|
| 1 | Login with biometric | "Secure biometric login" |
| 2 | Home/Room list | "Organized team conversations" |
| 3 | Chat screen | "End-to-end encrypted messaging" |
| 4 | Message features | "Rich messaging with reactions" |
| 5 | Settings | "Comprehensive security controls" |
| 6 | Dark mode | "Beautiful dark theme" |
| 7 | Profile | "Customizable profiles" |
| 8 | Room management | "Easy room creation" |

**Capture Command:**
```bash
# Using ADB
adb shell screencap -p /sdcard/screenshot.png
adb pull /sdcard/screenshot.png play-store/assets/screenshots/
```

**File Location:** `play-store/assets/screenshots/`

---

### 3. High-Res Icon (512x512 px)

**Specifications:**
- Dimensions: 512 x 512 pixels
- Format: PNG (32-bit with alpha)
- No transparency in safe zone (66% center)

**Source:** Export from `androidApp/src/main/res/mipmap-xxxhdpi/ic_launcher.png` (192x192 → scale up)

**Export Command (if ImageMagick installed):**
```bash
convert androidApp/src/main/res/mipmap-xxxhdpi/ic_launcher.png \
  -resize 512x512 \
  play-store/assets/icon-512x512.png
```

**File:** `play-store/assets/icon-512x512.png`

---

## 🚀 Quick Start for Asset Creation

### Step 1: Build and Install Debug APK
```bash
./gradlew installDebug
```

### Step 2: Capture Screenshots
```bash
# Create directory
mkdir -p play-store/assets/screenshots

# Navigate through app and capture each screen
# Run for each screen:
adb shell screencap -p /sdcard/screenshot-01.png
adb pull /sdcard/screenshot-01.png play-store/assets/screenshots/
```

### Step 3: Export Icon
```bash
# If using ImageMagick
convert androidApp/src/main/res/mipmap-xxxhdpi/ic_launcher.png \
  -resize 512x512 play-store/assets/icon-512x512.png

# Or use Android Studio's Image Asset Studio
# Right-click res folder > New > Image Asset > Export
```

### Step 4: Create Feature Graphic
- Use Figma, Canva, or Photoshop
- Template in `play-store/feature-graphic-brief.md`
- Export as PNG at 1024x500

---

## 📁 Final Asset Structure

```
play-store/
├── assets/
│   ├── icon-512x512.png           ← Create
│   ├── feature-graphic.png        ← Create
│   └── screenshots/
│       ├── 01-login.png           ← Capture
│       ├── 02-home.png            ← Capture
│       ├── 03-chat.png            ← Capture
│       ├── 04-reactions.png       ← Capture
│       ├── 05-settings.png        ← Capture
│       ├── 06-dark-mode.png       ← Capture
│       ├── 07-profile.png         ← Capture
│       └── 08-room-manage.png     ← Capture
├── SUBMISSION-CHECKLIST.md        ✅
├── store-listing.md               ✅
├── privacy-policy.md              ✅
├── release-notes.md               ✅
├── data-safety-form.md            ✅
├── metadata-compliance.md         ✅
└── screenshot-guide.md            ✅
```

---

## ✅ Submission Checklist

Once assets are created:

- [ ] Create `play-store/assets/` directory
- [ ] Export icon-512x512.png
- [ ] Create feature-graphic.png
- [ ] Capture 4-8 phone screenshots
- [ ] Host privacy policy at public URL
- [ ] Create Google Play Developer account ($25)
- [ ] Complete store listing in Play Console
- [ ] Upload AAB: `./gradlew bundleRelease`
- [ ] Submit for review

---

## 📊 Current Build Status

| Item | Status |
|------|--------|
| Debug APK | ✅ Builds |
| Release APK | ✅ Builds (17.7 MB) |
| Unit Tests | ✅ 127 passing |
| ProGuard/R8 | ✅ Configured |
| Signing | ⚠️ Needs keystore |

---

## 🔐 Signing Configuration

Before uploading to Play Store, create a release keystore:

```bash
# Generate keystore
keytool -genkey -v -keystore armorclaw-release.keystore \
  -alias armorclaw \
  -keyalg RSA \
  -keysize 2048 \
  -validity 10000

# Create keystore.properties (DO NOT COMMIT)
echo "storeFile=armorclaw-release.keystore" > keystore.properties
echo "storePassword=YOUR_PASSWORD" >> keystore.properties
echo "keyAlias=armorclaw" >> keystore.properties
echo "keyPassword=YOUR_KEY_PASSWORD" >> keystore.properties

# Build signed release
./gradlew bundleRelease
```

**IMPORTANT:** 
- Back up keystore securely (lose it = lose ability to update app)
- Never commit keystore.properties to git
- Use environment variables in CI/CD

---

## 📝 Store Listing Quick Fill

**App Name:** `ArmorChat - Secure Team Chat`

**Short Description:**
```
End-to-end encrypted messaging for teams. Military-grade security, zero-knowledge.
```

**Category:** Communication
**Tags:** encrypted messaging, secure chat, team communication, e2e encryption, privacy

**Privacy Policy URL:** [Host privacy-policy.md and enter URL]

---

**Ready for production! Complete the visual assets above and submit to Play Store.** 🚀
