# Visual UI Improvement Plan: Premium Command Center

## TL;DR

> **Goal**: Transform ArmorChat from "Functional Chat Client" to **"Premium Command Center"**
>
> **Deliverables**: 5 phases of visual enhancements
> - Glassmorphic cards with animated status rings
> - Timeline view with status icons
> - Elevated dock-style command bar
> - Animated security overlays (Vault)
> - Navigation transitions and micro-interactions
>
> **Estimated Effort**: Large (8 days, 24 tasks)
> **Parallel Execution**: YES (6 waves)
> **Critical Path**: Foundation → Dashboard → Workspace → Vault → Polish

---

## Context

### Original Request
Transform ArmorChat's UI to a high-fidelity "Fintech/Dashboard" aesthetic with:
- Glassmorphic cards with teal glow borders
- Animated status rings (Active/Idle)
- Timeline view with status icons (✅ ⚠️ ⏳)
- Elevated dock-style command bar
- Animated lock/unlock for Vault UI
- Navigation transitions and micro-interactions

### Interview Summary

**Research Findings**:
- **Existing foundation**: Teal (#14F0C8)/Navy (#0A1428) theme, GlowModifiers.kt, DesignTokens.kt
- **Components**: 28 organism-level components already built
- **Animation infrastructure**: Compose Animation, no Lottie (needs to be added)
- **Test strategy**: TDD confirmed

**Technical Decisions Made**:
- **Complex animations**: Use Lottie (adds ~2MB to APK)
- **Background textures**: Compose Drawing (DrawScope, no external assets)
- **Timeline view**: Enhance existing ActivityTimeline (not rebuild)
- **Tests**: TDD approach

### Metis Review - Critical Gaps Addressed

**Identified Risks & Mitigations**:

1. **Android <12 Compatibility** (CRITICAL)
   - Glassmorphism blur only works on Android 12+ (API 31)
   - Mitigation: Implement alpha-only fallback for older devices
   - Pattern: `if (VERSION.SDK_INT >= VERSION_CODES.S) blur else alpha`

2. **Performance on Low-End Devices**
   - Heavy blur + animations = potential 30fps
   - Mitigation: Device capability detection, adaptive blur radius

3. **Lottie APK Bloat**
   - Adds ~2MB minimum
   - Mitigation: Use sparingly, prefer Compose Animation where possible

4. **Accessibility Compliance**
   - Glassmorphism creates contrast issues
   - Mitigation: WCAG 3:1 contrast ratio, high contrast mode fallback, respect REDUCED_MOTION

**Guardrails Applied**:
- Glassmorphism: Android 12+ hardware blur, <12 fallback (solid colors)
- Status rings: Max 15px radius, simple pulsing animation
- Timeline: Use status icons, avoid complex line patterns
- Command bar: Elevated dock with LazyRow chips (not draggable yet)
- Animations: Lottie for complex only, prefer Compose Animation
- Blur reveal: Only on long press (not constant blur)
- Typography: Monospace for timestamps only
- Navigation: Shared axis slide + fade only
- Accessibility: 3:1 contrast, reduced motion, high contrast mode

---

## Work Objectives

### Core Objective
Transform ArmorChat's UI from utilitarian to premium "Fintech/Dashboard" experience with glassmorphic elements, fluid animations, and clear visual hierarchy.

### Concrete Deliverables

| Phase | Deliverable | Files Affected |
|-------|-------------|----------------|
| 1 | Mission Control Dashboard | HomeScreen, GlowModifiers, DesignTokens |
| 2 | Agent Workspace | SplitViewLayout, ActivityTimeline, CommandBar |
| 3 | Security UI (Vault) | VaultScreen, BiometricGateOverlay |
| 4 | Micro-Interactions | Navigation transitions, pull-to-refresh |
| 5 | Typography & Polish | Theme, Type.kt, accessibility audit |

### Definition of Done
- [ ] All Phase 1-5 visual enhancements implemented
- [ ] All animations render at 60fps on Android 12+ devices
- [ ] WCAG 3:1 contrast ratio verified on all glass surfaces
- [ ] High contrast mode supported (solid fallbacks)
- [ ] Reduced motion setting respected
- [ ] Glassmorphism fallback tested on Android <12
- [ ] All TDD tests pass
- [ ] Performance profiling complete (no memory leaks)

### Must Have
- Glassmorphic cards with teal glow borders
- Animated status rings (Active/Idle)
- Timeline view with status icons
- Elevated command bar
- Lock animation on Vault
- Navigation transitions

### Must NOT Have (Guardrails)
- Light mode support (dark-only design constraint)
- Complex timeline line patterns (use status icons only)
- Draggable command bar (use LazyRow chips instead)
- Constant blur on PII fields (reveal on long-press only)
- Lottie for simple animations (prefer Compose)
- Screens outside scope (Settings, Profile, Onboarding unchanged)

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES (Compose Test)
- **Automated tests**: YES (TDD)
- **Framework**: kotlin-test with Compose TestRule

### QA Policy
Every task includes agent-executed QA scenarios using:
- **Playwright** for browser UI testing
- **tmux** for CLI/TUI verification
- **curl** for API verification

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation - 6 parallel tasks):
├── T1: Add Lottie dependency
├── T2: Create GlassModifiers system
├── T3: Create AnimationSpecs constants
├── T4: Extend DesignTokens with glass values
├── T5: Add background textures (Compose Drawing)
└── T6: Typography updates (monospace)

Wave 2 (Phase 1 - Mission Control - 3 parallel tasks):
├── T7: MissionControlHeader enhancements
├── T8: NeedsAttentionQueue with sticky banner
└── T9: Home screen background texture

Wave 3 (Phase 2 - Agent Workspace - 4 parallel tasks):
├── T10: SplitViewLayout with draggable divider
├── T11: ActivityTimeline enhancements
├── T12: CommandBar dock redesign
└── T13: ChatScreen transitions

Wave 4 (Phase 3 - Security UI - 3 parallel tasks):
├── T14: BiometricGateOverlay lock animation
├── T15: VaultScreen blur reveal
└── T16: VaultScreen gradient background

Wave 5 (Phase 4 - Micro-interactions - 4 parallel tasks):
├── T17: Pull-to-refresh morphing
├── T18: Message animations
├── T19: Approval card animations
└── T20: Navigation transitions

Wave 6 (Phase 5 - Polish - 4 parallel tasks):
├── T21: Typography updates
├── T22: Accessibility audit
├── T23: Performance profiling
└── T24: Lottie animations integration

Critical Path: T1 → T2 → T7 → T10 → T14 → T17 → T24
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|------------|--------|
| T1 | - | T24 |
| T2 | - | T7, T8, T14 |
| T3 | - | T10, T14, T17, T18, T19 |
| T4 | - | T2 |
| T5 | - | T9 |
| T6 | - | T21 |
| T7 | T2 | - |
| T8 | T2 | - |
| T9 | T5 | - |
| T10 | T3 | - |
| T11 | T3 | - |
| T12 | - | - |
| T13 | T3 | - |
| T14 | T2, T3 | - |
| T15 | T2 | - |
| T16 | - | - |
| T17 | T3 | - |
| T18 | T3 | - |
| T19 | T3 | - |
| T20 | T3 | - |
| T21 | T6 | - |
| T22 | All | - |
| T23 | All | - |
| T24 | T1 | - |

### Agent Dispatch Summary

- **Wave 1**: 6 tasks → quick (T1-T4, T6), visual-engineering (T5)
- **Wave 2**: 3 tasks → quick (T7-T9)
- **Wave 3**: 4 tasks → visual-engineering (T10-T12), quick (T13)
- **Wave 4**: 3 tasks → visual-engineering (T14-T16)
- **Wave 5**: 4 tasks → visual-engineering (T17-T20)
- **Wave 6**: 4 tasks → quick (T21), unspecified-high (T22-T23), visual-engineering (T24)

---

## TODOs

### Wave 1: Foundation (6 parallel tasks)

- [ ] 1. Add Lottie Dependency

  **What to do**:
  - Add `lottie-compose:6.1.0` to `libs.versions.toml`
  - Add dependency to `androidApp/build.gradle.kts`
  - Sync Gradle and verify import works

  **Must NOT do**:
  - Add Lottie to shared module (Android-only for now)
  - Add unnecessary Lottie animations (use sparingly)

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T24 (Lottie animations integration)
  - **Blocked By**: None

  **References**:
  - `gradle/libs.versions.toml` - Add version here
  - `androidApp/build.gradle.kts` - Add implementation

  **Acceptance Criteria**:
  - [ ] `libs.versions.toml` has `lottie-compose = "6.1.0"`
  - [ ] `androidApp/build.gradle.kts` has `implementation(libs.lottie.compose)`
  - [ ] `./gradlew assembleDebug` succeeds

  **QA Scenarios**:
  ```
  Scenario: Lottie dependency resolves
    Tool: Bash
    Steps:
      1. ./gradlew assembleDebug
    Expected Result: BUILD SUCCESSFUL
    Evidence: .sisyphus/evidence/task-01-gradle-build.txt
  ```

  **Commit**: YES
  - Message: `build(deps): add lottie-compose 6.1.0`
  - Files: `gradle/libs.versions.toml`, `androidApp/build.gradle.kts`

- [ ] 2. Create GlassModifiers System

  **What to do**:
  - Create `shared/src/commonMain/kotlin/ui/theme/GlassModifiers.kt`
  - Implement `Modifier.glassCard()` with blur + border
  - Implement `Modifier.glowRing()` for status indicators
  - Add Android version check for blur support

  **Must NOT do**:
  - Use blur on Android <12 without fallback
  - Animate blur radius directly (GPU expensive)
  - Apply glass to every surface (use sparingly)

  **Recommended Agent Profile**:
  - **Category**: `visual-engineering`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T7, T8, T14
  - **Blocked By**: None

  **References**:
  - `armorclaw-ui/.../GlowModifiers.kt` - Existing glow patterns
  - `shared/.../ui/theme/DesignTokens.kt` - Use existing tokens
  - WordPress-Android fallback pattern: `if (SDK_INT >= S) blur else alpha`

  **Acceptance Criteria**:
  - [ ] `GlassModifiers.kt` created
  - [ ] `Modifier.glassCard()` renders semi-transparent surface
  - [ ] `Modifier.glowRing()` creates pulsing ring animation
  - [ ] Android <12 fallback uses alpha instead of blur

  **QA Scenarios**:
  ```
  Scenario: Glass card renders on Android 12+
    Tool: Bash (adb)
    Preconditions: Android 12+ emulator running
    Steps:
      1. Launch app with glass card on home screen
      2. adb shell dumpsys gfxinfo com.armorclaw.app | grep "90th"
    Expected Result: 90th percentile < 16.6ms
    Evidence: .sisyphus/evidence/task-02-glass-fps.txt
  ```

  **Commit**: YES
  - Message: `feat(ui): add glassmorphism modifiers`
  - Files: `shared/.../ui/theme/GlassModifiers.kt`

- [ ] 3. Create AnimationSpecs Constants

  **What to do**:
  - Create `shared/src/commonMain/kotlin/ui/theme/AnimationSpecs.kt`
  - Define easing curves (FastOutSlowIn, EaseInOutCubic)
  - Define duration constants using DesignTokens.Duration
  - Create stagger helper for sequential animations

  **Must NOT do**:
  - Hardcode durations (use DesignTokens)
  - Create overly long animations (>400ms)
  - Use LinearEasing for UI transitions

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T10, T14, T17, T18, T19
  - **Blocked By**: None

  **References**:
  - `shared/.../ui/theme/DesignTokens.kt:Duration` - fast=150ms, normal=250ms, slow=400ms
  - Existing patterns in BiometricGateOverlay.kt

  **Acceptance Criteria**:
  - [ ] `AnimationSpecs.kt` created
  - [ ] Easing curves defined
  - [ ] Duration specs use DesignTokens values
  - [ ] Stagger helper function created

  **Commit**: YES
  - Message: `feat(ui): add centralized animation specs`
  - Files: `shared/.../ui/theme/AnimationSpecs.kt`

- [ ] 4. Extend DesignTokens with Glass Values

  **What to do**:
  - Add glass-specific tokens to `DesignTokens.kt`
  - Add blur radius constants (small=10dp, medium=20dp, large=30dp)
  - Add glass opacity values (surface=0.05f, border=0.1f)
  - Add glow colors (tealGlow, navyGlow)

  **Must NOT do**:
  - Change existing token values (breaks existing components)
  - Add tokens not used by this plan
  - Use hardcoded values in components

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1
  - **Blocks**: T2
  - **Blocked By**: None

  **References**:
  - `shared/.../ui/theme/DesignTokens.kt` - Extend existing object

  **Acceptance Criteria**:
  - [ ] Blur radius constants added
  - [ ] Glass opacity values added
  - [ ] Glow colors defined
  - [ ] Existing tokens unchanged

  **Commit**: YES
  - Message: `feat(ui): add glass design tokens`
  - Files: `shared/.../ui/theme/DesignTokens.kt`

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read plan end-to-end. Verify all "Must Have" items implemented. Check evidence files exist.

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `./gradlew detekt` + `./gradlew test`. Review for `as any`, empty catches, console.log.

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Execute all QA scenarios. Test on Android 12+ device. Verify 60fps animations.

- [ ] F4. **Scope Fidelity Check** — `deep`
  Compare diffs against plan. Verify no scope creep. Check "Must NOT Have" compliance.

---

## Commit Strategy

Group commits by phase:
- `feat(ui): add glassmorphism foundation (Wave 1)`
- `feat(ui): mission control dashboard enhancements (Wave 2)`
- `feat(ui): agent workspace improvements (Wave 3)`
- `feat(ui): security UI animations (Wave 4)`
- `feat(ui): micro-interactions and motion (Wave 5)`
- `feat(ui): typography and polish (Wave 6)`

---

## Success Criteria

### Verification Commands
```bash
./gradlew assembleDebug              # Build succeeds
./gradlew test                       # All tests pass
./gradlew detekt                     # No critical issues
adb shell dumpsys gfxinfo            # 90th percentile < 16.6ms
```

### Final Checklist
- [ ] All 24 tasks complete
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] 60fps on Android 12+ devices
- [ ] WCAG 3:1 contrast verified
- [ ] High contrast mode works
- [ ] Reduced motion respected
- [ ] No memory leaks
