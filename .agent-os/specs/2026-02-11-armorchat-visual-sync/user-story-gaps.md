# Missing User Stories - Gap Analysis

> **Date:** 2026-02-11
> **Version:** 1.0.0
> **Status:** Identified and Fixed

---

## Critical Gaps (Must Fix)

### Gap 1: QR Scanning Flow

**Problem:** QR Onboarding spec has generation but no scanning UX

**New Story 5: QR Code Scanning**

**As** a new ArmorChat user, I want to scan a QR code to quickly configure my app.

**Acceptance Criteria:**
- Camera permission requested proactively
- QR scanner opens automatically on "Quick Setup" selection
- Visual feedback shows when QR is detected and validated
- Configuration auto-populates on successful scan
- User can fallback to manual setup if scanning fails

**Workflow:**
1. User opens ArmorChat for first time
2. Taps "Quick Setup" (primary button)
3. App requests camera permission (explain why)
4. Scanner view opens with framing guide
5. User holds QR code in frame
6. Visual feedback (border flash) when QR detected
7. Upon validation:
   - Success: "Configuration loaded! Connecting..." → auto-connect
   - Failure: "Invalid QR code. Try again or Manual Setup"
8. On connect success: Show "Welcome! You're all set."

**Edge Cases:**
- QR code expired → "This QR code has expired. Please generate a new one."
- Invalid signature → "This QR code isn't from a trusted source."
- Network unavailable → "No internet. Please connect and try again."

---

### Gap 2: First-Time Device Verification

**Problem:** Device Trust Management assumes devices are already verified, but how do users verify initially?

**New Story 6: Initial Device Verification**

**As** a new ArmorChat user, I want to verify my device to access encrypted messages.

**Acceptance Criteria:**
- First device auto-verifies on account creation
- Clear explanation of what verification means
- Verification code displayed for cross-device verification
- Step-by-step guidance for multi-device setup

**Workflow:**
1. User creates ArmorChat account
2. First device automatically marked "Verified" (this is their trusted anchor)
3. Screen appears: "This device is now your trusted anchor"
4. Shows 6-character verification code: `ABC123`
5. Instructions: "Use this code to verify other devices"
6. Option to "Add another device now" or "Skip for later"

**Second Device Verification:**
1. User opens ArmorChat on second device
2. Prompts: "Enter verification code from your primary device"
3. User enters `ABC123`
4. Bridge validates code and marks second device as verified
5. Success: "Your device is now verified!"

---

### Gap 3: Security Tier Upgrade UX

**Problem:** Progressive Security has tier definitions but no user-facing upgrade flow

**New Story 7: Security Tier Upgrade Notification**

**As** an ArmorChat user, I want to be notified when I can upgrade my security level.

**Acceptance Criteria:**
- In-app notification when tier requirements are met
- Clear explanation of what new tier provides
- One-tap upgrade with confirmation
- Option to defer and upgrade later

**Workflow:**
1. User completes 3rd successful login (triggers Tier 2 eligibility)
2. Non-blocking notification appears: "Enhanced security available!"
3. User taps notification
4. Modal appears explaining Tier 2 benefits:
   - ✅ Biometric unlock (Face ID/Touch ID)
   - ✅ Verified device badges
   - ✅ Cross-device trust
5. Primary button: "Enable Enhanced Security"
6. Secondary button: "Remind Me Later"
7. On enable:
   - System prompts for biometric enrollment
   - Success confirmation: "Enhanced security enabled!"
8. User can manage security level anytime in Settings → Security

**Voice Call Tier Unlock:**
1. User completes first voice call (alternative Tier 2 trigger)
2. Same notification appears: "Voice call detected! Enhanced security now available."

---

### Gap 4: QR Expiration Handling

**Problem:** QR codes can expire but no user story for handling expired codes

**New Story 8: QR Code Expiration**

**As** an ArmorChat user, I want clear guidance when my QR code expires.

**Acceptance Criteria:**
- Expiration message explains why QR expired
- Option to regenerate QR code
- Fallback to manual setup always available
- Time-limited QR codes show countdown/expiration time

**Workflow:**
1. User opens ArmorChat, tries to scan QR
2. Scanner reads QR but validation fails with "expired" flag
3. App displays:
   - "This QR code expired on [date/time]"
   - "For security, QR codes are valid for 24 hours"
   - "Ask the bridge owner to generate a new QR code"
4. Primary button: "Manual Setup" (fallback)
5. Secondary button: "Scan Again" (reopens scanner)

**For QR Generator:**
1. Bridge owner generates QR
2. QR payload includes expiration timestamp
3. Display shows: "This QR code expires in 24 hours"
4. Option to "Regenerate" with new expiration

---

### Gap 5: Multiple Concurrent Devices

**Problem:** No handling for race conditions when multiple devices sync simultaneously

**New Story 9: Concurrent Sync Handling**

**As** an ArmorChat user, I want to know what happens when I sync from multiple devices at once.

**Acceptance Criteria:**
- User is informed of concurrent activity
- Last-write-wins strategy with user notification
- Clear indication of which device "won"
- Option to force full sync if needed

**Workflow:**
1. User has phone and desktop both open
2. User sends message from phone
3. Desktop is syncing simultaneously
4. Desktop detects conflict:
   - Toast notification: "Message modified on another device"
   - Status bar shows ⚠️ (yellow warning)
5. User taps notification
6. Modal appears: "Your phone sent a message 2 seconds ago"
7. Options:
   - "Refresh Now" (pull latest from server)
   - "Dismiss" (keep current state, will resolve on next sync)
8. On refresh:
   - All conflicts resolved
   - Status returns to 🟢 (synced)

---

### Gap 6: Device Lockout Recovery

**Problem:** What happens when user loses all devices? No recovery story.

**New Story 10: Account Recovery After Device Loss**

**As** an ArmorChat user, I want to recover my account if I lose all my devices.

**Acceptance Criteria:**
- Recovery option available when no verified devices exist
- Email/SMS verification for recovery
- Security questions or backup phrase option
- Clear warning that recovery takes 24-48 hours

**Workflow:**
1. User loses phone and tablet
2. User opens ArmorChat on new device
3. No verified devices detected
4. Recovery screen appears:
   - "You don't have any verified devices"
   - "Let's verify your identity to recover your account"
5. Recovery options:
   - Email verification (if email on file)
   - SMS verification (if phone number on file)
   - Backup phrase (12-word recovery phrase shown during setup)
6. User selects verification method
7. Bridge initiates verification:
   - Sends code via email/SMS
   - Or prompts for backup phrase
8. User enters verification code
9. Bridge verifies:
   - Success: "Your account is recovered! This device is now your trusted anchor."
   - Warning: "For security, full access will be restored in 24 hours."
10. During waiting period:
    - Read-only access to messages
    - Cannot send messages until fully verified

---

### Gap 7: Device Removal Workflow

**Problem:** Device list shows devices but no story for removing them

**New Story 11: Device Removal**

**As** an ArmorChat user, I want to remove a device I no longer use.

**Acceptance Criteria:**
- Device list shows "Remove" option for each device
- Confirmation dialog explains consequences
- Cannot remove last verified device (safety check)
- Removal happens immediately with confirmation

**Workflow:**
1. User opens Settings → Devices
2. User sees old phone they no longer use
3. User taps device → Details screen
4. "Remove This Device" button (secondary, red text)
5. Confirmation dialog:
   - "Remove iPhone 11?"
   - "This device will lose access to your encrypted messages."
   - "This action cannot be undone."
6. User confirms
7. Device removed immediately from list
8. Toast notification: "iPhone 11 removed"

**Safety Check:**
- Cannot remove the only verified device (anchor)
- Attempt shows: "This is your only verified device. You cannot remove it."
- Suggest: "Add another device first, then remove this one."

---

### Gap 8: Sync State Transitions

**Problem:** Status shows states but no story for visual transitions between states

**New Story 12: Sync State Transition Animations**

**As** an ArmorChat user, I want smooth, understandable transitions between sync states.

**Acceptance Criteria:**
- State transitions are animated (not abrupt)
- Loading indicators show progress
- Success states have positive feedback
- Error states are clearly distinguished from normal states

**Transition Matrix:**

| From State | To State | Animation | Duration |
|-----------|----------|-----------|----------|
| Synced → Syncing | Yellow spinner fades in | 200ms fade-in | 200ms |
| Syncing → Synced | Green checkmark animation | 300ms bounce | 300ms |
| Syncing → Offline | Red icon with shake | 400ms shake | 400ms |
| Offline → Synced | Green pulse from red | 500ms pulse | 500ms |
| Any → Conflict | Orange warning badge pop | 200ms scale-up | 200ms |

**Implementation:**
- Use Lottie animations for smooth transitions
- Status bar icon animates using `AnimatedIcon` component
- Background color transitions use `AnimatedContainer`
- Progress indicators use `CircularProgressIndicator`

---

## Medium Priority Gaps

### Gap 9: Background Sync Behavior

**Problem:** No story for what happens when app is in background

**New Story 13: Background Sync**

**As** an ArmorChat user, I want my messages to sync even when the app is closed.

**Acceptance Criteria:**
- Background sync runs periodically (every 15-30 min)
- Respect OS battery optimization settings
- Show badge count when new messages arrive
- User can disable background sync in settings

---

### Gap 10: Sync Health Dashboard

**Problem:** No story for viewing overall sync health over time

**New Story 14: Sync Health Dashboard**

**As** a security-conscious ArmorChat user, I want to see my sync history and health.

**Acceptance Criteria:**
- Settings → Sync Health screen
- Shows: last 7 days of sync activity
- Graphs: successful syncs, failures, conflicts
- Device trust timeline
- Export option for audit trail

---

## Summary

**Gaps Identified:** 12
**Critical:** 8 (Must implement for v1)
**Medium:** 4 (Can defer to v1.1)

**Stories Added:**
1. Story 5: QR Code Scanning
2. Story 6: Initial Device Verification
3. Story 7: Security Tier Upgrade Notification
4. Story 8: QR Code Expiration
5. Story 9: Concurrent Sync Handling
6. Story 10: Account Recovery After Device Loss
7. Story 11: Device Removal
8. Story 12: Sync State Transition Animations
9. Story 13: Background Sync
10. Story 14: Sync Health Dashboard

**Updated Total User Stories:** 24 (was 4)

---

## Integration Plan

| Priority | Stories | Effort | Target Release |
|----------|---------|--------|----------------|
| P0 (Critical) | 5, 6, 10, 11 | L | v1.0 |
| P1 (High) | 7, 8, 12 | M | v1.0 |
| P2 (Medium) | 9, 13, 14 | M | v1.1 |

**Total Additional Effort:** ~3-4 weeks for all 12 stories
