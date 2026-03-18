# Splash Screen

> **Route:** `splash`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/splash/SplashScreen.kt`
> **Category:** Core

## Screenshot

![Splash Screen](../../screenshots/core/splash.png)

## Layout

```
┌─────────────────────────────────────┐
│                                     │
│                                     │
│                                     │
│            ┌─────────┐              │
│            │  LOGO   │              │
│            │  🦀     │              │
│            └─────────┘              │
│                                     │
│           ArmorClaw                 │
│                                     │
│   Secure End-to-End Encrypted Chat  │
│                                     │
│                                     │
│            ◠ ◠ ◠ ◠                  │
│           Loading...                │
│                                     │
│                                     │
│                                     │
└─────────────────────────────────────┘
```

## UI States

### Loading (Default)

```
┌─────────────────────────────────────┐
│                                     │
│            [ANIMATED]               │
│             🦀                      │
│           ArmorClaw                 │
│    Secure End-to-End Encrypted Chat │
│                                     │
│         ◠ ◠ ◠ ◠ (spinner)           │
│                                     │
└─────────────────────────────────────┘
```

**Description:**
- Logo fades in and scales up with bounce animation
- CircularProgressIndicator shows loading state
- 1.5s delay before navigation decision

### Navigation Decision (Internal)

```
    ┌─────────────────────┐
    │ Check Auth Status   │
    └──────────┬──────────┘
               │
     ┌─────────┼─────────┐
     ▼         ▼         ▼
┌────────┐ ┌────────┐ ┌────────┐
│Has     │ │Has     │ │First   │
│Session │ │Onboard │ │Time    │
└───┬────┘ │But No  │ │User    │
    │      │Login   │ └───┬────┘
    ▼      └───┬────┘     │
  Home        ▼          ▼
           Login    Onboarding
                    (QR-first)
```

## State Flow

```
            ┌──────────────┐
            │   App Start  │
            └──────┬───────┘
                   │
                   ▼
            ┌──────────────┐
            │   Animating  │
            │  (800ms)     │
            └──────┬───────┘
                   │
                   ▼
            ┌──────────────┐
            │   Loading    │
            │  (1500ms)    │
            └──────┬───────┘
                   │
     ┌─────────────┼─────────────┐
     ▼             ▼             ▼
┌─────────┐  ┌─────────┐  ┌─────────────┐
│→ Home   │  │→ Login  │  │→ Connect    │
│(valid   │  │(needs   │  │(first time) │
│session) │  │auth)    │  │             │
└─────────┘  └─────────┘  └─────────────┘
```

## User Flow

1. **User arrives from:** App launch (cold start or warm start)
2. **User sees:** Animated logo and loading indicator
3. **User navigates to:**
   - Home (if valid session exists)
   - Login (if onboarding complete but not logged in)
   - Connect/Onboarding (if first time user)

## Components Used

| Component | Source | Purpose |
|-----------|--------|---------|
| Image | Compose | Logo display |
| Text | Material3 | App name and tagline |
| CircularProgressIndicator | Material3 | Loading indicator |
| Box, Column | Compose | Layout containers |

## Animation Details

| Animation | Duration | Easing |
|-----------|----------|--------|
| Alpha fade | 800ms | FastOutSlowInEasing |
| Scale bounce | Spring | DampingRatioMediumBouncy |

## Accessibility

- **Content descriptions:**
  - Logo: "ArmorClaw" (decorative, could be marked as such)
- **Touch targets:** N/A (no interactive elements)
- **Screen reader:** Announces app name during load

## Design Tokens

| Token | Value |
|-------|-------|
| Background | Navy |
| Logo size | 120.dp |
| Loading indicator | 32.dp, Teal color |
| Padding | 32.dp |
| Text color | Teal (headline), OnBackground.copy(0.7f) |

## Notes

- First impression screen - sets security-focused tone
- Animation provides perceived quality
- Navigation handled by SplashViewModel via StateFlow observation
- QR-first onboarding flow for new users
