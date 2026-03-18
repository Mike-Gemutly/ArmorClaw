# ArmorTerminal Play Store Assets

> **Created:** 2026-03-17
> **Status:** Ready for Design Production

---

## 1. App Icon (512x512)

### Design Concept
A shield icon with a terminal cursor inside, shield, conveying security and command-line interface.

### Icon Specifications

| Property | Value |
|----------|-------|
| Size | 512x512 px |
| Format | PNG with transparency |
| Background | Transparent |
| Style | Flat design with subtle gradient |

### Color Palette

| Element | Color | Hex |
|--------|-------|-----|
| Shield Primary | Blue | `#3B82F6` |
| Shield Dark | Dark Blue | `#1E40AF` |
| Cursor | Cyan | `#22D3EE` |
| Prompt | Green | `#10B981` |
| Background | Dark Blue | `#0F172A` |

### Icon Design Brief
```
┌─────────────────────────────────┐
│                             │
│       ╭───────────────╮       │
│       │   ◀─▓ ────▀       │
│       │     ▂▂▂       │   <- Terminal cursor (cyan #22D3EE)
│       │   ◀─▀ ────▀       │
│       ╰───────────────╯       │
│                             │
│       SHIELD OUTLINE           │   <- Blue #3B82F6 with gradient to #1E40AF
│                             │
└─────────────────────────────────┘
```

### Production Notes
- Use 32px padding for safe area
- Export as WebP for Play Store (reduced file size)
- Provide PNG fallback for compatibility

---

## 2. Feature Graphic (1024x500)

### Design Concept
Split design showing:
- Left side: Terminal interface with multi-window layout
- Right side: Shield logo with tagline

### Specifications

| Property | Value |
|----------|-------|
| Size | 1024x500 px |
| Format | PNG or JPEG |
| Style | Modern, professional |

### Content Layout

```
┌────────────────────────────────────────────────────────────────┐
│                                          │                  │
│   ┌───────────────────────┐              │     [SHIELD]      │
│   │ ▶  Terminal          │              │                  │
│   │   ┌─────┐ ┌─────┐    │              │  Your AI Agents │
│   │   │Agent│ │Agent│    │              │   on Your VPS │
│   │   └─────┘ └─────┘    │              │                  │
│   │   Multi-Window View    │              │   [DOWNLOAD]      │
│   └───────────────────────┘              │                  │
│                                          │                  │
└────────────────────────────────────────────────────────────────┘
   ◄── 512px ───────────────────────────────► ◄── 512px ──┘
```

### Text Elements
- **Tagline:** "Your AI Agents on Your VPS"
- **CTA:** "DOWNLOAD" button style
- **Colors:** Use brand palette from above

---

## 3. Phone Screenshots (Min 2, Max 8)

### Required Screenshots

#### Screenshot 1: Login/Pairing Screen
- **Content:** QR code scanner with "Scan to Connect" prompt
- **Purpose:** Show easy onboarding
- **Size:** 1080x1920 (portrait) or 1920x1080 (landscape)

#### Screenshot 2: Multi-Window Terminal
- **Content:** 4-window grid layout with active agents
- **Purpose:** Demonstrate core functionality
- **Features to highlight:**
  - Multiple AI agent windows
  - Real-time output streaming
  - Command input at bottom

#### Screenshot 3: Settings/Server Config
- **Content:** Server configuration screen
- **Purpose:** Show customization options
- **Features:** Server URL, Matrix homeserver, quick setup buttons

#### Screenshot 4: HITL Approval Flow (Recommended)
- **Content:** Human-in-the-loop approval dialog
- **Purpose:** Demonstrate security features
- **Features:** Sensitive action approval, context display

### Screenshot Guidelines

| Guideline | Recommendation |
|-----------|---------------|
| Device Frame | Phone (not tablet) |
| Orientation | Portrait preferred |
| Background | Use actual app screenshots |
| Annotations | Add callouts for key features |
| Status Bar | Show in screenshots |

---

## 4. App Name & Descriptions

### App Name
**ArmorTerminal**

### Short Description (80 chars)
```
Control AI agents on your VPS from your phone with secure approval flows.
```

### Full Description (4000 chars)
```
ArmorTerminal is your mobile command center for AI agents running on your VPS.

🤖 **Multi-Agent Terminal**
Work with multiple AI agents simultaneously in a split-screen terminal interface. Each agent runs in its own window with real-time streaming output.

🔐 **Secure by Design**
- End-to-end encrypted communication
- Biometric authentication
- Human-in-the-loop approval for sensitive actions
- Screenshot prevention for security

⚡ **Quick Setup**
- Scan QR code from your ArmorClaw dashboard
- Automatic server discovery via mDNS
- Fallback servers with automatic failover

🎯 **Key Features:**
- Multi-window terminal with 5 layout modes
- Pipeline command chains with visual preview
- Context transfer between agents
- Workflow checkpoint and resume
- Voice calls with TURN relay
- Platform integration (Slack, Discord, Teams, WhatsApp)
- File upload with one-way security policy

🔒 **Security First:**
- Certificate pinning
- Encrypted local storage
- Session token management
- Automatic data purge on background

**Perfect for:**
- Developers managing AI agents remotely
- Teams with automated workflows
- Power users wanting mobile control of server-side AI

ArmorTerminal requires an ArmorClaw bridge server. Visit armorclaw.app to get started.
```

---

## 5. Categorization

### Primary Category
**Productivity**

### Secondary Categories
- Communication
- Developer Tools
- Business

### Content Rating
**Everyone** (PEGI 3 / IARC 3+)
- No violence, sexual content, or profanity
- No user-generated content
- No gambling or real-money transactions

---

## 6. Graphic Asset Checklist

| Asset | Size | Format | Status |
|-------|------|--------|--------|
| App Icon | 512x512 | PNG | ⬜ Needed |
| Feature Graphic | 1024x500 | PNG/JPEG | ⬜ Needed |
| Phone Screenshot 1 | 1080x1920 | PNG/JPEG | ⬜ Needed |
| Phone Screenshot 2 | 1080x1920 | PNG/JPEG | ⬜ Needed |
| Phone Screenshot 3 | 1080x1920 | PNG/JPEG | ⬜ Recommended |
| Phone Screenshot 4 | 1080x1920 | PNG/JPEG | ⬜ Recommended |
| 7" Tablet Screenshot | 1280x800 | PNG/JPEG | Optional |
| 10" Tablet Screenshot | 1280x800 | PNG/JPEG | Optional |

---

## 7. Design Tools Recommendations

### For Icon Creation
- **Figma** - Free, collaborative design
- **Adobe Illustrator** - Professional vector graphics
- **Inkscape** - Free vector graphics editor

### For Screenshots
- **Android Studio Emulator** - Take device screenshots
- **ADB Screenshot** - `adb shell screencap -p /sdcard/screen.png`
- **Physical Device** - Most authentic

### Color Export
All colors are available in CSS/Hex format for easy import into design tools.

---

## 8. Production Checklist

- [ ] Create 512x512 app icon with transparency
- [ ] Create 1024x500 feature graphic
- [ ] Take minimum 2 phone screenshots
- [ ] Export all assets as PNG (WebP optional)
- [ ] Verify all text is readable at target sizes
- [ ] Test icons on light and dark backgrounds
- [ ] Compress images for Play Store (under 1MB each recommended)
