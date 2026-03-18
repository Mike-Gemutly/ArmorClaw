# ArmorTerminal Play Console Submission Checklist

> **App Name:** ArmorTerminal
> **Package ID:** com.armorclaw.armorterminal
> **Version:** 2.5.0
> **Target SDK:** 35 (Android 15)
> **Minimum SDK:** 26 (Android 8.0)

> **Last Updated:** 2026-03-17

---

## Pre-Submission Checklist

### ✅ Build Verification
- [x] Target SDK 35 (Android 15)
- [x] Compile SDK 35
- [x] Build release AAB
- [x] All unit tests pass (17 test files)
- [x] Lint clean

- [x] ProGuard rules configured

### ✅ Documentation
- [x] Privacy Policy document created (`PRIVACY_POLICY.md`)
- [x] Data Safety documentation prepared (`DATA_SAFETY.md`)
- [x] Store Listing Checklist created (this file)

- [ ] Privacy policy URL hosted
- [ ] App icon (512x512)
- [ ] Feature graphic (1024x500)
- [ ] Phone screenshots (minimum 2)

### ⬜ Store Listing Assets Needed
| Asset | Size | Format | Status |
|-------|------|--------|--------|
| App Icon | 512x512 px | PNG with transparency | ⬜ Create |
| Feature Graphic | 1024x500 px | PNG/JPEG | ⬜ Create |
| Phone Screenshots | 1080x1920 px min | PNG | ⬜ Capture (min 2) |
| 7" Screenshots | 1080x1920 px min | PNG | ⬜ Optional |
| 10" Screenshots | 1080x1920 px min | PNG | ⬜ Optional |

---

## App Information

### Basic Details
- **App Name:** ArmorTerminal
- **Package Name:** com.armorclaw.armorterminal
- **Short Description:** (80 chars max)
  ```
  Control AI agents on your VPS from your phone. Secure terminal with multi-window UI, blind security, and human-in-the-loop approvals.
  ```

- **Full Description:** (4000 chars max)
  ```
  ArmorTerminal is a secure Android terminal for multi-agent AI orchestration. Run AI agents 24/7 on your VPS and control them from your phone with end-to-end encrypted Matrix communication.

  **Key Features:**
  - **Multi-Window Terminal** - Work with multiple AI agents simultaneously using GRID, PIPELINE, FOCUS, or SPLIT layouts
  - **BlindFill Security** - Agents never see your PII; secrets are injected directly into forms
  - **Human-in-the-Loop (HITL)** - Approve sensitive actions before execution
  - **Context Transfer** - Drag-and-drop content transfer between agents
  - **QR Device Pairing** - Quick setup by scanning QR code from ArmorClaw dashboard
  - **Workflow Management** - Visual workflow builder with checkpoint resume
  - **Account Recovery** - 12-word BIP39 recovery phrase backup

  **Security:**
  - End-to-end encryption via Matrix protocol
  - SQLCipher encrypted local storage
  - Biometric authentication (fingerprint/face)
  - Screenshot prevention (FLAG_SECURE)
  - Certificate pinning for network security

  **Requirements:**
  - ArmorClaw Bridge server (self-hosted or cloud)
  - Android 8.0 or higher
  - Camera access (optional, for QR scanning)

  **Permissions:**
  - INTERNET (required) - Server communication
  - ACCESS_NETWORK_STATE (required) - Check connectivity
  - CAMERA (optional) - QR code device pairing
  - USE_BIOMETRIC (optional) - Secure app access
  - VIBRATE (optional) - Haptic feedback
  - WAKE_LOCK (required) - Background sync
  - FOREGROUND_SERVICE (required) - Data synchronization
  ```

### Category
- **Productivity** (not Games, Tools, or Finance, etc.)

### Content Rating
**Expected Rating:** Everyone / PEGI 3 (or equivalent)

**Questionnaire Answers:**
1. **Violence:** None
2. **Language:** None (professional productivity app)
3. **Sexual Content:** None
4. **Controlled Substance:** None
5. **Gambling:** None

6. **User Interaction:** None required
7. **Location Sharing:** No
8. **Digital Purchases:** No in-app purchases

---

## Data Safety Section (Play Console)

### Data Collection

| Data Type | Collected | Purpose |
|----------|----------|---------|
| Personal Info | No | N/A |
| Financial Info | No | N/A |
| Health & Fitness | No | N/A |
| Messages | Yes | Communication between user and AI agents |
| Photos/Videos | No | N/A |
| Audio | No | N/A |
| Files | Yes | Encrypted file uploads to bridge server |
| Calendar | No | N/A |
| Contacts | No | N/A |
| Location | No | N/A |
| App Activity | Yes | Usage analytics and crash reporting (optional) |
| Web Browsing | Yes | Through ArmorClaw bridge for AI tasks |

### Data Handling

- **Encryption in Transit:** Yes (TLS 1.3)
- **Encryption at Rest:** Yes (AES-256-GCM for files, SQLCipher for local storage)
- **Data Deletion:** Yes (user can clear all data)
- **Data Sharing:** No (data stays on device and your server only)

### Security Practices

- Screenshot prevention (FLAG_SECURE)
- Biometric authentication available
- Certificate pinning for API communication
- Encrypted local storage (SQLCipher)
- Memory-only secrets (API keys never persisted to disk)

---

## Permissions & Justifications

| Permission | Type | Justification (for Play Console) |
|------------|------|--------------------------------|
| `INTERNET` | Required | Required for server communication |
| `ACCESS_NETWORK_STATE` | Required | Required to check network connectivity |
| `ACCESS_WIFI_STATE` | Required | Required for mDNS/Bonjour discovery |
| `CHANGE_WIFI_MULTICAST_STATE` | Required | Required for mDNS discovery |
| `CHANGE_WIFI_STATE` | Required | Required for mDNS discovery |
| `CAMERA` | Optional | Required for QR code device pairing |
| `POST_NOTIFICATIONS` | Optional | Required for push notifications (Android 13+) |
| `VIBRATE` | Optional | Required for haptic feedback |
| `RECEIVE_BOOT_COMPLETED` | Optional | Required for auto-start on boot |
| `FOREGROUND_SERVICE` | Required | Required for background data sync |
| `FOREGROUND_SERVICE_DATA_SYNC` | Required | Required for data sync foreground service |
| `FOREGROUND_SERVICE_SPECIAL_USE` | Required | Required for specialized foreground service |
| `WAKE_LOCK` | Required | Required to keep device awake during sync |
| `USE_BIOMETRIC` | Optional | Required for biometric authentication |
| `READ_CLIPBOARD` | Optional | Required for secure clipboard operations |

---

## Screenshots Guide

### Screenshot 1: Multi-Window Terminal
**Description:** Show 4 terminal windows in GRID layout with different AI agents
**Features to Highlight:**
- Dark theme (#0F172A background)
- Cyan cursor (#22D3EE)
- Green prompt (#10B981)
- Layout indicator (GRID)
- Connection status indicator

### Screenshot 2: HITL Approval Interface
**Description:** Show approval interface for sensitive action
**Features to Highlight:**
- Approval request details
- "Approve" and "Reject" buttons
- Security badge/icon
- Dark theme consistency

### Screenshot 3: QR Pairing (Optional)
**Description:** Show QR scanner interface
**Features to Highlight:**
- Camera view with QR overlay
- "Scanning..." indicator
- Connection status

### Screenshot 4: Workflow Builder (Optional)
**Description:** Show visual workflow builder interface
**Features to Highlight:**
- Workflow steps visualization
- YAML preview panel
- Dark theme consistency

---

## Privacy Policy Hosting

### Option 1: GitHub Pages (Free)
```bash
# Create gh-pages branch
git checkout -b gh-pages

# Copy privacy policy
cp PRIVACY_POLICY.md docs/PRIVACY_POLICY.md

# Push to GitHub
git add .
git commit -m "Add privacy policy for GitHub Pages"
git push origin gh-pages

# Enable GitHub Pages in repository settings
# URL will be: https://[username].github.io/[repo]/PRIVACY_POLICY.html
```

### Option 2: Netlify (Free)
```bash
# Install Netlify CLI
npm install -g netlify-cli

# Deploy
netlify deploy --prod

# Or drag-and-drop the PRIVACY_POLICY.md file in Netlify dashboard
# URL will be: https://[your-site].netlify.app/PRIVACY_POLICY.html
```

### Option 3: Custom Domain
```
# Host at: https://armorclaw.app/privacy/armorterminal
# Or: https://armorclaw.com/privacy/armorterminal
```

---

## Final Submission Steps

1. **Create App in Play Console**
   - Go to [Play Console](https://play.google.com/console)
   - Click "Create app"
   - Fill in basic app information

2. **Upload Assets**
   - Upload app icon (512x512)
   - Upload feature graphic (1024x500)
   - Upload at least 2 phone screenshots

3. **Complete Store Listing**
   - Fill in short and full descriptions
   - Add privacy policy URL
   - Select category

4. **Complete Content Rating**
   - Answer questionnaire (expected: Everyone/PEGI 3)

5. **Complete Data Safety Form**
   - Answer all questions based on DATA_SAFETY.md

6. **Set Up Pricing**
   - Select "Free" (no in-app purchases)

7. **Review & Release**
   - Review all sections
   - Click "Start rollout to production"

---

## Timeline Estimate
| Task | Time |
|------|------|
| Create app icon | 30 min |
| Create feature graphic | 30 min |
| Capture screenshots | 15 min |
| Complete Play Console forms | 30 min |
| Set up privacy policy hosting | 15 min |
| Review and submit | 15 min |
| **Total** | ~2-2.5 hours |

---

## Contact & Support
- **Email:** support@armorclaw.app
- **Website:** https://armorclaw.app
- **Documentation:** https://docs.armorclaw.app
