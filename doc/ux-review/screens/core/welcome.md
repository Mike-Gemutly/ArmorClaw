# Welcome Screen

> **Route:** `welcome`
> **File:** `androidApp/src/main/kotlin/com/armorclaw/app/screens/onboarding/WelcomeScreen.kt`
> **Category:** Core / Onboarding

## Screenshot

![Welcome Screen](../../screenshots/core/welcome.png)

## Layout

```
┌─────────────────────────────────────┐
│ ▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒ │  ← TopAppBar (Primary color)
├─────────────────────────────────────┤
│                                     │
│              🛡️                     │  ← Shield Icon (120.dp)
│                                     │
│       Welcome to ArmorClaw          │  ← Headline
│                                     │
│    Secure AI agents in your pocket  │  ← Subtitle
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 🔒 Enterprise Security      │   │  ← Feature Card 1
│  │    Your API keys never      │   │
│  │    leave secure container   │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 💬 Chat Anywhere            │   │  ← Feature Card 2
│  │    Connect to agents from   │   │
│  │    anywhere with Matrix     │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 🔐 Zero-Trust               │   │  ← Feature Card 3
│  │    Protected even if agent  │   │
│  │    is compromised           │   │
│  └─────────────────────────────┘   │
│                                     │
│         (scrollable content)        │
│                                     │
├─────────────────────────────────────┤
│  ┌──────────┐  ┌────────────────┐  │
│  │   Skip   │  │  Get Started   │  │  ← Action buttons
│  └──────────┘  └────────────────┘  │
└─────────────────────────────────────┘
```

## UI States

### Default / Loaded

```
┌─────────────────────────────────────┐
│              🛡️                     │
│       Welcome to ArmorClaw          │
│    Secure AI agents in your pocket  │
│                                     │
│    [Feature cards displayed]        │
│                                     │
│   [Skip]        [Get Started]       │
└─────────────────────────────────────┘
```

**Description:**
- Shield logo centered at top
- Feature list scrollable
- Two action buttons at bottom

## State Flow

```
            ┌──────────────┐
            │    Idle      │
            └──────┬───────┘
                   │
        ┌──────────┼──────────┐
        ▼                     ▼
┌───────────────┐     ┌───────────────┐
│  Get Started  │     │     Skip      │
└───────┬───────┘     └───────┬───────┘
        │                     │
        ▼                     ▼
┌───────────────┐     ┌───────────────┐
│   Continue    │     │    Skip to    │
│   Onboarding  │     │    Login      │
└───────────────┘     └───────────────┘
```

## User Flow

1. **User arrives from:** Splash screen (first time user path)
2. **User can:**
   - Read feature descriptions
   - Tap "Get Started" to begin onboarding
   - Tap "Skip" to skip onboarding
3. **User navigates to:**
   - Onboarding flow (Get Started)
   - Login screen (Skip)

## Components Used

| Component | Source | Purpose |
|-----------|--------|---------|
| Scaffold | Material3 | Screen layout |
| TopAppBar | Material3 | Navigation bar |
| Icon | Material3 | Logo and feature icons |
| Text | Material3 | Titles and descriptions |
| OutlinedButton | Material3 | Skip action |
| ArmorClawButton | atoms/Button.kt | Primary action |
| FeatureCard | Local | Feature display card |
| FeatureList | Local | Feature collection |

## Accessibility

- **Content descriptions:**
  - Logo: "ArmorClaw Logo"
  - Feature icons: null (decorative)
- **Touch targets:**
  - Buttons: 48.dp minimum height
- **Focus order:** Top to bottom, buttons at end
- **Screen reader considerations:**
  - Feature cards read in sequence
  - Button actions clearly labeled

## Design Tokens

| Token | Value |
|-------|-------|
| TopAppBar color | Primary |
| Logo size | 120.dp |
| Logo tint | BrandPurple |
| Spacing | DesignTokens.Spacing.xl/md |
| Button weight | 1f each (equal split) |

## Feature Cards

| Icon | Title | Description |
|------|-------|-------------|
| Security | Enterprise Security | Your API keys never leave secure container |
| Chat | Chat Anywhere | Connect to agents from anywhere with Matrix |
| Lock | Zero-Trust | Protected even if agent is compromised |

## Notes

- Key value proposition screen
- Sets security-first tone for app
- Skip option respects user autonomy
- Scrolling accommodates smaller screens
- Feature cards use consistent layout pattern
