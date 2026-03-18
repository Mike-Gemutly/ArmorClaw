# ArmorTerminal Play Store Assets

> **Created:** 2026-03-17
> **Status:** Ready for Creation
> **App Name:** ArmorTerminal
> **Package:** `com.armorclaw.armorterminal`

---

## Brand Identity

### Primary Colors

| Name | Hex | Usage |
|------|-----|-------|
| **Shield Primary Blue** | `#3B82F6` | Brand identity, buttons, highlights |
| **Terminal Background** | `#0F172A` | Main background |
| **Terminal Text** | `#E2E8F0` | Primary text |
| **Terminal Cursor** | `#22D3EE` | Cursor, highlights, accents |
| **Terminal Prompt** | `#10B981` | Command prompt, success indicators |
| **Surface** | `#1E293B` | Card backgrounds |
| **Surface Container** | `#334155` | Secondary surfaces |

### Complementary Colors

| Name | Hex | Usage |
|------|-----|-------|
| **Primary Light** | `#60A5FA` | Light accents |
| **Primary Container** | `#1E3A8A` | Containers |
| **Secondary** | `#7DD3FC` | Secondary elements |
| **Error** | `#FCA5A5` | Error states |

---

## Required Assets

### 1. App Icon (512x512 px)
**Requirements:**
- PNG format with transparency
- 32-bit color depth
- Adaptive icon (Android 8.0+)

**Design Concept:**
- Shield shape with terminal overlay
- Dark blue background (#0F172A)
- Cyan cursor bar (#22D3EE)
- Green prompt symbol (#10B981)
- Blue shield outline (#3B82F6)

**Layer Structure:**
```
Layer 1 (Background):
  - Dark blue circle (#0F172A)
  - 512x512 px

Layer 2 (Shield):
  - Blue shield shape (#3B82F6)
  - Centered, 280x280 px

Layer 3 (Terminal Window):
  - Dark rectangle (#0F172A)
  - Centered in shield
  - 180x120 px

Layer 4 (Terminal Elements):
  - Cyan cursor bar (#22D3EE)
  - Green prompt "$" (#10B981)
  - Light gray text (#E2E8F0)
```

### 2. Feature Graphic (1024x500 px)
**Requirements:**
- PNG or JPEG format
- No transparency
- High quality for varied screen sizes

**Design Concept:**
- Multi-window terminal view
- Dark theme (#0F172A background)
- 4 terminal windows in GRID layout
- Cyan cursors and green prompts visible
- ArmorClaw branding

**Layout:**
```
┌─────────────────────────────────────────────────────────┐
│                    ArmorTerminal                   │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│  │ Agent 1 │  │ Agent 2 │  │ Agent 3 │  │ Agent 4 │  │
│  │ $ cmd   │  │ $ cmd   │  │ $ cmd   │  │ $ cmd   │  │
│  │         │  │         │  │         │  │         │  │
│  │ output │  │ output  │  │ output  │  │ output  │  │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘  │
│                                                   │
│  "Control AI agents from your phone"              │
│  ArmorClaw • Secure • Encrypted • BlindFill™       │
└─────────────────────────────────────────────────────────┘
```

### 3. Phone Screenshots (Minimum 2, Recommended 4-8)
**Requirements:**
- PNG or JPEG format
- Device-specific sizes (see below)
- Show actual app UI
- No mockups

**Screenshot 1: Multi-Window Terminal**
- Show 4 windows in GRID layout
- Dark theme
- Command prompts visible
- Layout indicator showing "GRID"

**Screenshot 2: QR Pairing**
- QR scanner interface
- Camera preview with scan frame
- "Scan QR to connect" text
- Dark overlay

**Screenshot 3: HITL Approval**
- Approval dialog
- Request details (e.g., "Allow access to credit card?")
- Approve/Reject buttons
- Security badge

**Screenshot 4: Context Transfer**
- Drag and drop interface
- Transfer dialog
- Progress indicator
- Multiple messages selected

### Device Screenshot Sizes

| Device | Size (px) |
|--------|----------|
| Phone | 1080x1920 |
| 7" Tablet | 1080x1920 |
| 10" Tablet | 1080x1920 |

---

## Creation Tools
### Design Software
- **Figma** (recommended) - Free, collaborative
- **Adobe XD** - Free tier available
- **Sketch** - macOS only, paid
- **Canva** - Free tier, easy to use

### Asset Generation Steps
1. **App Icon**
   - Create 512x512 artboard
   - Build layers as described above
   - Export as PNG with transparency
   - Test on light and dark backgrounds

2. **Feature Graphic**
   - Create 1024x500 artboard
   - Design multi-window layout
   - Add branding elements
   - Export as PNG (no transparency)

3. **Screenshots**
   - Install app on physical device or emulator
   - Capture screenshots using Android Studio or ADB
   - Edit to highlight key features
   - Export in required sizes

---

## App Description Text

### Short Description (80 characters max)
```
Control AI agents from your phone. Secure terminal with BlindFill security and human-in-the-loop approvals.
```

### Full Description (4000 characters max)
```
ArmorTerminal is a secure Android terminal for multi-agent AI orchestration. Control AI agents running 24/7 on your VPS directly from your phone with end-to-end encrypted Matrix communication.

KEY FEATURES:

🔒 BLINDFILL™ SECURITY
Agents never see your sensitive data. Secrets are injected directly into the browser - the agent only knows "payment.card_number" but never sees the actual number.

📱 MULTI-WINDOW TERMINAL
Work with multiple AI agents simultaneously. Choose from 5 layout modes: GRID, PIPELINE, FOCUS, SPLIT, and CUSTOM. Drag and drop to transfer context between agents.

🔐 HUMAN-IN-THE-LOOP APPROVALS
Review and approve sensitive operations before they happen. The app shows exactly what the agent is requesting and lets you approve or reject with one tap.

🔄 CONTEXT TRANSFER
Seamlessly transfer conversation context between agents. Select multiple messages and drag them to another agent's window.

🔄 ACCOUNT RECOVERY
Backup your account with a 12-word BIP39 recovery phrase. 48-hour recovery window ensures you can always regain access.

🔒 SECURE BY DESIGN
- End-to-end encrypted via Matrix protocol
- FLAG_SECURE prevents screenshots
- Biometric authentication (fingerprint/face)
- Encrypted local storage
- Certificate pinning for network security

📱 QR DEVICE PAIRING
Scan the QR code from your ArmorClaw dashboard to instantly connect. Automatic mDNS discovery for local network bridges.

🎯 WORKFLOW AUTOMATION
Create and execute multi-step workflows. Checkpoint resume, version pinning, and duplicate prevention built-in.

---

## Keywords (for ASO)
```
AI agent, terminal, secure, encrypted, multi-window, VPS control, BlindFill, human-in-the-loop, approval workflow, context transfer, biometric, Matrix protocol, QR pairing
```

---

## Privacy Policy URL
Host at: `https://armorclaw.app/privacy/armorterminal`

---

## Next Steps
1. Create app icon using design software
2. Create feature graphic with multi-window layout
3. Install app on device and capture screenshots
4. Review and finalize app description text
5. Prepare all assets for Play Console upload
