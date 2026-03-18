# ArmorChat Visual Sync Status Specification

> **Spec:** ArmorChat Visual Sync Status Indicators
> **Created:** 2026-02-11
> **Status:** Planning
> **Version:** 1.0.0

---

## Overview

This specification describes the Visual Sync Status feature for ArmorChat, a Matrix client application. The feature provides users with clear, real-time visibility into their synchronization state, device trust, and connection health without overwhelming them with technical details.

### Goals

1. **Transparency** - Users should always know their sync status
2. **Simplicity** - Status should be understandable without technical background
3. **Actionability** - Problems should be resolvable with clear next steps
4. **Trust** - Users should feel confident about their message security

---

## User Stories

### Story 1: Basic Sync Status Display

**As** a new ArmorChat user, I want to see if my messages are synced and encrypted.

**Acceptance Criteria:**
- Status indicator is visible in chat interface
- Current sync state is clearly shown
- Last successful sync time is displayed

**Workflow:**
1. User opens ArmorChat
2. Glances at status bar (top of screen or in header)
3. Sees icon and current state:
   - 🟢 All synced - "Messages secure and up to date"
   - 🟡 Syncing - "Sending encrypted messages..."
   - 🔴 Offline - "Will sync when connection restored"
   - ⚠️ Sync conflict - "Messages modified elsewhere"
4. User continues using app normally

---

### Story 2: Device Trust Management

**As** an ArmorChat user, I want to know which devices have access to my encrypted messages.

**Acceptance Criteria:**
- Can see list of all trusted devices
- Each device shows verification status
- Can identify unknown/new devices

**Workflow:**
1. User opens Settings → Devices
2. User sees list of devices:
   - 📱 This Device - Verified (Desktop App, San Francisco, CA)
   - 💻 Phone - Verified (iPhone 14, San Francisco, CA)
   - 💻 Tablet - Verified (iPad, San Francisco, CA)
   - ❓ Unknown Device - Detected 2 hours ago (New York, NY)
3. User taps unknown device for details:
   - "When did you sign in?"
   - Option to "Verify this device" or "Remove access"
4. User continues using app with confidence

---

### Story 3: Connection Troubleshooting

**As** an ArmorChat user, my messages aren't sending.

**Acceptance Criteria:**
- Clear error message explains what's wrong
- Suggested actions are actionable
- One-tap resolution if possible

**Error States:**
- 🔴 Connection Failed - "Couldn't reach homeserver. Check your internet."
- 🔐 Certificate Error - "Server certificate changed. Tap to view new cert."
- 🔒 Sync Conflict - "Messages modified elsewhere. Pull to refresh."
- 🟡 Rate Limited - "Too many requests. Please wait a moment."

**User Actions:**
- "Retry Now" button (prominent)
- "Refresh" button
- "View Logs" button (advanced)
- Option to contact support

---

### Story 4: Offline Queue Visualization

**As** an ArmorChat user, I want to see what will happen when I'm offline.

**Acceptance Criteria:**
- Queued messages are clearly listed
- Pending count is shown
- Estimated send time is displayed

**Offline Panel:**
```
┌───────────────────────┐
│ 💤 Offline Queue      │
│                      │
│  3 messages queued    │
│                      │
│ Will send when online │
│                      │
└──────────────────────────┘
```

---

## Scope

### In Scope
- Visual status indicators in chat interface
- Device management and trust panel
- Connection error states and troubleshooting
- Offline queue visualization
- Pull-to-refresh for sync conflicts

### Out of Scope
- Backend implementation (this is a frontend spec)
- Matrix adapter enhancements
- Push notification system (separate project)

---

## UI Requirements

### Status Bar Component

**Location:** Header of main chat view or dedicated status bar

**States:**
| State | Icon | Color | Description |
|--------|------|-------|-------------|
| Synced | 🟢 | Green | All messages encrypted and synced |
| Syncing | 🟡 | Yellow | Sending encrypted messages... |
| Offline | 🔴 | Red | No connection, will queue messages |
| Conflict | ⚠️ | Orange | Messages modified elsewhere, tap to resolve |
| Error | 🔴 | Red | Connection or protocol error |

**Progress Indicator (when syncing):**
- Circular loading animation during active sync
- "Last synced: 2 minutes ago" timestamp

### Device List Component

**Location:** Settings → Devices

**Display Format:**
```
┌─────────────────────────────────────────┐
│ My Devices                            │
│                                        │
│ ┌─────────────────────────────────────────┐ │
│ │ ✅ This Device                      │ │
│ │ Your Desktop App                      │ │
│ │ Verified: Oct 15, 2024               │ │
│ │ San Francisco, CA                      │ │
│ │ IP: 192.168.1.100                 │ │
│ │ Last seen: 2 minutes ago              │ │
│ └─────────────────────────────────────────┘ │
│                                        │
│ ┌─────────────────────────────────────────┐ │
│ │ 💻 Phone                            │ │
│ │ Your iPhone 14                       │ │
│ │ Verified: Aug 22, 2024               │ │
│ │ San Francisco, CA                      │ │
│ │ IP: 10.0.0.45                       │ │
│ │ Last seen: 1 day ago                 │ │
│ └─────────────────────────────────────────┘ │
│                                        │
│ ┌─────────────────────────────────────────┐ │
│ │ 💻 Tablet                            │ │
│ │ Your iPad                           │ │
│ │ Verified: Feb 8, 2024                │ │
│ │ San Francisco, CA                      │ │
│ │ Last seen: 5 days ago                 │ │
│ └─────────────────────────────────────────┘ │
│                                        │
│ └───────────────────────────────────────────────┘ │
│                                        │
│ [ + Add New Device ]                    │
└───────────────────────────────────────────────────┘
```

**Unknown Device Handling:**
- Shows prominent warning icon
- One-tap "Verify" or "Remove"
- 3 attempts to verify before marking as lost

---

### Error Display Component

**Full Screen Error States:**

| Error Type | Title | Message | Action |
|------------|-------|---------|--------|
| Connection Failed | Can't Connect | "Couldn't reach homeserver. Check your internet or tap to retry." | Retry Button |
| Certificate Error | Security Alert | "Server certificate changed. This server may be impersonating another homeserver. Tap to view certificate." | View Cert / Remove Device |
| Sync Conflict | Messages Modified | "Messages were modified on another device. Pull down to refresh." | Pull to Refresh |
| Rate Limited | Too Many Requests | "You're making requests too quickly. Please wait a moment." | Auto-Retry (disabled for 30s) |
| Need Update | Update Available | "A new version of ArmorChat is available. Please update to continue syncing." | Update Button |

**Toast Notifications:**
- Non-blocking errors show as toast at bottom
- Connection errors persist as banner at top until resolved
- Success state changes show brief confirmation toast

---

## Technical Constraints

### Platform Considerations
- Mobile devices may have limited battery
- Status indicators should not poll excessively (respect battery)
- Animations should be subtle (avoid distracting users)
- Colors should meet accessibility standards (WCAG AA)

### Performance Requirements
- Status bar updates should not cause UI lag
- Device list should load quickly (< 500ms for 100 devices)
- Error states should clear automatically when resolved

### Data Flow

**ArmorChat ←→ Bridge ←→ Matrix Server**

```
┌──────────────┐                 ┌───────────────────────┐                 ┌─────────────────────────────────┐
│              │                 │              │                 │                │
│   ArmorChat  │  GET /devices  │  │                 │                │                │
│   (Mobile)   │  Query Devices   │  │                 │                │
│              │  ◄──────────────►│              │                 │                │
│              │                 │  ◄──────────────►│              │                 │                │
│              │                 │  Return device list   │─────────────────►│              │                 │                │
│              │                 │  ◄──────────────►│              │                 │                │
│              └──────────────┘                 └──────────────┘                 └───────────────┘                 └───────────────────┘
│                                                           │
│                                                           │
│                    ┌───────────────────────────────────────────────┐                 │
│                    │  GET /sync/status                │                 │                │
│                    │  ◄──────────────────────────────────────►│                 │                │
│                    │  Return sync status                 │─────────────────►│              │                 │                │
│                    │  ◄───────────────►│              │                 │                │
│                    └───────────────────────────────────────────────┘                 └───────────────────┘                 │
│                                                           │
│                                                           │
│              ┌───────────────────────────────────────────────────┐                 │
│              │  GET /bridge/status                 │                 │                │
│                    │  ◄──────────────────────────────────────►│                 │                │
│                    │  Return bridge & Matrix status        │─────────────────►│              │                 │                │
│                    │  ◄───────────────►│              │                 │                │
│                    └───────────────────────────────────────────────┘                 └───────────────┘                 └───────────────────┘                 │
│                                                           ▼
│                    ┌───────────────────────────────────────────────────┐                 │
│                    │  ArmorChat (Mobile)                       │                 │                │
│                    │  GET /sync/status                │                 │                │
│                    │  ◄──────────────────────────────────────►│                 │                │
│                    │  Return sync status                   │─────────────────►│              │                 │                │
│                    │  ◄───────────────►│              │                 │                │
│                    └───────────────────────────────────────────────┘                 └───────────────────┘                 └───────────────────┘                 │
│                                                           ▼
```

---

## Non-Functional Requirements

### Interaction Design
- Status must be scannable with QR code for quick device addition
- All actions should be undoable (no destructive actions without confirmation)
- Help text should be context-aware based on current state

### Privacy Requirements
- Device list should not expose full IP addresses (show city, state only)
- Last seen times should be relative ("2 hours ago", not exact timestamp)

---

## Open Questions

1. Should sync status be visible in all chat views or only specific screens?
2. Should device trust be per-device or global?
3. How should we handle Matrix connection failures on mobile?
4. What's the refresh strategy for sync conflicts - full message reload or differential sync?
5. Should we show full encryption status (E2EE verified) or just trust indicator?

---

## Success Metrics

- **Completion:** All stories implemented with testable outcomes
- **Adoption:** >90% of users complete device setup within 2 minutes
- **Error Reduction:** <5% of sync operations require manual intervention
- **Time to Resolution:** Average sync conflict resolution under 30 seconds

---

**Next Steps:**
1. Implement Phase 1: Bridge RPC Enhancements (4 tasks - ~M effort)
2. Design API contracts between ArmorChat and Bridge
3. Implement mobile UI components (3 tasks - ~XL effort)

---

**Note:** This spec is for ArmorChat, a SEPARATE project from ArmorClaw bridge. Implementation requires coordination between teams.
