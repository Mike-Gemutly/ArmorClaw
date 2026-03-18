# Complete Onboarding Flow

> **Flow:** Onboarding
> **Screens:** 10
> **Duration:** ~3-5 minutes

## Overview

The onboarding flow guides new users through setting up their secure messaging account, including server connection, key backup, and permission grants.

## Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           ONBOARDING FLOW                                        │
└─────────────────────────────────────────────────────────────────────────────────┘

┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  SPLASH  │────▶│ WELCOME  │────▶│  SETUP   │────▶│CONNECT   │────▶│SECURITY  │
│          │     │          │     │  MODE    │     │  SERVER  │     │  EXPLAIN │
└──────────┘     └──────────┘     └──────────┘     └──────────┘     └──────────┘
                      │                │                │                │
                      │ skip           │ express        │                │
                      ▼                ▼                │                ▼
               ┌──────────┐     ┌──────────┐            │         ┌──────────┐
               │  LOGIN   │     │ EXPRESS  │            │         │ Step 1-4 │
               │          │     │  SETUP   │            │         └──────────┘
               └──────────┘     └──────────┘            │               │
                                                        │               ▼
                                                        │         ┌──────────┐
                                                        │         │  KEY     │
                                                        │         │  BACKUP  │
                                                        │         └──────────┘
                                                        │               │
                                                        │               ▼
                                                        │         ┌──────────┐
                                                        │         │MIGRATION │
                                                        │         │(if data) │
                                                        │         └──────────┘
                                                        │               │
                                                        │               ▼
                                                        │         ┌──────────┐
                                                        │         │PERMISSION│
                                                        │         │  SCREEN  │
                                                        │         └──────────┘
                                                        │               │
                                                        │               ▼
                                                        │         ┌──────────┐
                                                        │         │TUTORIAL  │
                                                        │         │  SCREEN  │
                                                        │         └──────────┘
                                                        │               │
                                                        │               ▼
                                                        │         ┌──────────┐
                                                        │         │COMPLETION│
                                                        │         │  SCREEN  │
                                                        │         └──────────┘
                                                        │               │
                                                        └───────────────┘
                                                                        │
                                                                        ▼
                                                                 ┌──────────┐
                                                                 │   HOME   │
                                                                 │  SCREEN  │
                                                                 └──────────┘
```

## Step-by-Step Analysis

### Step 1: Splash → Welcome

```
┌─────────────┐                    ┌─────────────┐
│   SPLASH    │    1.5s delay      │   WELCOME   │
│             │ ─────────────────▶ │             │
│  [Loading]  │    auto-navigate   │ [Features]  │
└─────────────┘                    └─────────────┘
```

**Decision Point:** User can skip onboarding entirely
- **Get Started** → Continue to setup mode
- **Skip** → Jump to login (for returning users)

---

### Step 2: Setup Mode Selection

```
┌─────────────┐                    ┌─────────────┐
│   WELCOME   │                    │  SETUP MODE │
│             │ ─────────────────▶ │  SELECTION  │
│ [Started]   │                    │             │
└─────────────┘                    └─────────────┘
                                          │
                              ┌───────────┴───────────┐
                              ▼                       ▼
                    ┌─────────────┐          ┌─────────────┐
                    │   EXPRESS   │          │   MANUAL    │
                    │    SETUP    │          │   SETUP     │
                    │  (QR scan)  │          │  (detailed) │
                    └─────────────┘          └─────────────┘
```

**Options:**
- **Express Setup** → QR code scan for quick config
- **Manual Setup** → Step-by-step configuration

---

### Step 3: Server Connection

```
┌─────────────┐                    ┌─────────────┐
│   CONNECT   │                    │   SERVER    │
│   SERVER    │ ─────────────────▶ │  CONNECTED  │
│             │                    │             │
│ [URL input] │                    │  [Success]  │
└─────────────┘                    └─────────────┘
```

**Actions:**
- Enter server URL
- Validate connection
- Handle connection errors

---

### Step 4: Security Explanation (4 Steps)

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  SECURITY   │───▶│  SECURITY   │───▶│  SECURITY   │───▶│  SECURITY   │
│   Step 1    │    │   Step 2    │    │   Step 3    │    │   Step 4    │
│             │    │             │    │             │    │             │
│Encryption   │    │Key Storage  │    │  Verification│   │ Best Prac   │
│ Basics      │    │             │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

**Topics Covered:**
1. **Step 1:** What is E2E encryption
2. **Step 2:** How keys are stored (AndroidKeyStore)
3. **Step 3:** Device verification process
4. **Step 4:** Security best practices

---

### Step 5: Key Backup Setup

```
┌─────────────┐                    ┌─────────────┐
│    KEY      │                    │   BACKUP    │
│   BACKUP    │ ─────────────────▶ │   CREATED   │
│    SETUP    │                    │             │
│             │                    │  [Recovery  │
│ [Passphrase]│                    │   phrase]   │
└─────────────┘                    └─────────────┘
```

**Critical Step:**
- User sets up recovery passphrase
- Recovery phrase displayed (must be saved)
- Warning about losing keys

---

### Step 6: Migration (Conditional)

```
┌─────────────┐                    ┌─────────────┐
│  MIGRATION  │                    │  MIGRATED   │
│   CHECK     │ ─────────────────▶ │  SUCCESS    │
│             │                    │             │
│ (if old app │                    │  [Continue] │
│  data)      │                    │             │
└─────────────┘                    └─────────────┘
```

**Condition:** Only shown if migrating from previous app version

---

### Step 7: Permissions

```
┌─────────────┐                    ┌─────────────┐
│ PERMISSIONS │                    │  GRANTED    │
│   SCREEN    │ ─────────────────▶ │             │
│             │                    │  [Next]     │
│ [Allow... ] │                    │             │
└─────────────┘                    └─────────────┘
```

**Permissions Requested:**
- Notifications (required for messages)
- Camera (optional, for QR scanning)
- Microphone (optional, for calls)

---

### Step 8: Tutorial

```
┌─────────────┐                    ┌─────────────┐
│  TUTORIAL   │                    │  TUTORIAL   │
│   SCREEN    │ ─────────────────▶ │  COMPLETE   │
│             │                    │             │
│ [UI tips]   │                    │  [Finish]   │
└─────────────┘                    └─────────────┘
```

**Topics Covered:**
- Navigation basics
- Starting a chat
- Security features
- Settings overview

---

### Step 9: Completion

```
┌─────────────┐                    ┌─────────────┐
│ COMPLETION  │                    │    HOME     │
│   SCREEN    │ ─────────────────▶ │   SCREEN    │
│             │                    │             │
│ [Success!]  │                    │ [Chat list] │
└─────────────┘                    └─────────────┘
```

**Celebration:**
- Success animation
- Summary of what was set up
- Button to start messaging

---

## State Flow Summary

```
                    ┌──────────────────────┐
                    │    ONBOARDING        │
                    │      STATE           │
                    └──────────┬───────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
  ┌───────────┐          ┌───────────┐          ┌───────────┐
  │  NOT      │          │  IN       │          │  COMPLETE │
  │  STARTED  │          │ PROGRESS  │          │           │
  └───────────┘          └───────────┘          └───────────┘
                               │
                    ┌──────────┼──────────┐
                    ▼          ▼          ▼
              ┌─────────┐ ┌─────────┐ ┌─────────┐
              │ STEP 1  │ │ STEP 2  │ │ STEP N  │
              │         │ │         │ │         │
              │welcome  │ │connect  │ │complete │
              └─────────┘ └─────────┘ └─────────┘
```

## Error Handling

### Connection Errors
```
┌─────────────────────┐
│  ⚠️ Connection      │
│     Failed          │
│                     │
│  Could not connect  │
│  to server. Check   │
│  your internet and  │
│  try again.         │
│                     │
│  [Retry]  [Back]    │
└─────────────────────┘
```

### Permission Denied
```
┌─────────────────────┐
│  ⚠️ Permission      │
│     Required        │
│                     │
│  Notifications are  │
│  required for       │
│  message alerts.    │
│                     │
│  [Open Settings]    │
│  [Continue Anyway]  │
└─────────────────────┘
```

## Success Metrics

| Metric | Target |
|--------|--------|
| Completion rate | > 80% |
| Time to complete | < 5 minutes |
| Drop-off at key backup | < 15% |
| Skip rate | < 10% |

## Accessibility Considerations

- All screens support screen readers
- Sufficient color contrast throughout
- Touch targets meet 48dp minimum
- Text scalable to 200%
- Reduce motion option respected

## Notes

- Security-first messaging throughout
- Key backup is critical - users warned multiple times
- Express setup reduces friction for technical users
- Tutorial can be skipped and accessed later in settings
- Deep link support for invite-based onboarding
