# Draft: ArmorChat Visual UI Improvement Plan

## Requirements (confirmed)
- **Goal**: Transform from "Functional Chat Client" to "Premium Command Center"
- **Current State**: Dark mode, Teal/Navy branding, Material 3
- **Target State**: High-fidelity "Fintech/Dashboard" aesthetic with fluid motion

## Technical Decisions

### Theme System (Current State)
- **Primary Colors**: Teal (#14F0C8), TealGlow (#67F5D8), Navy (#0A1428)
- **Location**: `armorclaw-ui/src/commonMain/kotlin/com/armorclaw/ui/theme/`
- **Dark mode only**: Light mode not supported

### Existing Components (Already Built)
| Component | Location | Current Features |
|-----------|----------|------------------|
| MissionControlHeader | shared/ui/components/MissionControlHeader.kt | Animated vault status, attention badges with pulse |
| ActivityTimeline | shared/ui/components/ActivityTimeline.kt | Live indicator, filter chips, timeline events |
| BiometricGateOverlay | shared/ui/components/BiometricGateOverlay.kt | Gradient background, pulse, glow animations |
| CommandBar | shared/ui/components/CommandBar.kt | Chat input component |
| SplitViewLayout | shared/ui/components/SplitViewLayout.kt | Chat/workspace split view |

### Animation Infrastructure (Current)
- `androidx.compose.animation.*` - Already in use
- `rememberInfiniteTransition` - Pulse effects
- `animateFloat` - Alpha/scale animations
- `graphicsLayer` - Transformations
- `Brush.linearGradient` - Gradient backgrounds
- **No Lottie** - Would need to add dependency

### Key Screens
- **HomeScreen**: `androidApp/.../screens/home/HomeScreen.kt` - Mission Control Dashboard
- **ChatScreen**: `androidApp/.../screens/chat/ChatScreen_enhanced.kt` - Agent Workspace
- **VaultScreen**: `androidApp/.../screens/vault/VaultScreen.kt` - Secure Vault

## Scope Boundaries

### INCLUDE (Phase 1-5 from plan)
- [x] Glassmorphic card components with glow borders
- [x] Animated status rings (Active/Idle indicators)
- [x] Subtle grid/topographic background patterns
- [x] Draggable split divider with grip texture
- [x] Timeline view with connecting lines
- [x] Elevated dock-style command bar
- [x] Animated lock icon (3D rotation)
- [x] Blurred PII fields with reveal effect
- [x] Pull-to-refresh with morphing indicator
- [x] Navigation transitions (shared axis)
- [x] Typography updates (monospace timestamps, increased line height)

### EXCLUDE (Explicit Out of Scope)
- [ ] Light mode support (design constraint)
- [ ] Server-side changes
- [ ] New feature development (only visual improvements)
- [ ] Accessibility overhaul (audit only in Sprint 4)

## Research Findings

### Visual Inspiration References
- **Linear App**: Sidebar layout, clean typography
- **Discord**: Active/Idle status ring animations
- **Revolut/Monzo**: Vault secure visual language
- **Twitter/X**: Swipe actions, fluid list animations

### Existing Animation Patterns (Can Reuse)
1. **Pulse Animation** (BiometricGateOverlay):
   ```kotlin
   val alpha by infiniteTransition.animateFloat(
       initialValue = 0.7f,
       targetValue = 1f,
       animationSpec = infiniteRepeatable(...)
   )
   ```
2. **Gradient Background** (BiometricGateOverlay):
   ```kotlin
   .background(Brush.linearGradient(listOf(Navy, NavyLight)))
   ```
3. **Glow Effect** (BiometricGateOverlay):
   ```kotlin
   Box(modifier = Modifier.graphicsLayer { alpha = glowAlpha }
       .background(color.copy(alpha = 0.2f))
   )
   ```

## Research Findings (from explore agents)

### Existing Animation Infrastructure
- **Dependencies**: Compose Animation 1.5.0, Accompanist SwipeRefresh/Placeholder
- **Navigation**: Simple 300ms fade transitions (no slide/scale/shared element)
- **Loading States**: AgentThinkingIndicator, TypingIndicator, VaultPulseIndicator
- **Micro-interactions**: AnimatedVisibility, animateContentSize used extensively
- **MISSING**: Lottie, physics-based animations, swipe-to-dismiss, drag gestures

### Existing Visual Effects (Can Reuse/Extend)
- **GlowModifiers.kt**: `glowTeal()` and `tealBorder()` already implemented
- **Blur Effects**: `Modifier.blur()` used in ConsentOverlay, InterventionPreview
- **Gradient Overlays**: `Brush.verticalGradient()` with alpha transparency
- **Pulse Animations**: `rememberInfiniteTransition()` pattern established
- **DesignTokens.kt**: Spacing, elevation, icon sizes, durations (150/250/400ms)

### Component Inventory (28 organism-level components)
| Category | Components |
|----------|-----------|
| Dashboard | MissionControlHeader, QuickActionsBar, NeedsAttentionQueue, ActiveTasksSection |
| Chat | SplitViewLayout, ActivityLog, CommandBar, AgentThinkingIndicator |
| Cards | WorkflowCard, BlindFillCard, ActiveTaskCard, AttentionItemCard |
| Status | AgentStatusBanner, SensitivityBadge, VaultStatusIndicator |
| Security | BiometricGateOverlay, BiometricIndicator, VaultKeyPanel |

## Decisions Made (Confirmed with User)
- **Complex Animations**: Use **Lottie** for complex animations (Agent Thinking, success states)
- **Test Strategy**: **TDD** - Each task follows RED-GREEN-REFACTOR
- **Background Textures**: **Compose Drawing** - Use `DrawScope` and path drawing for grid/topographic patterns
- **Timeline View**: **Enhance ActivityTimeline** - Add connecting lines and status icons to existing component

## Ready for Plan Generation
All requirements clear. Proceeding to work plan creation.

## Dependencies to Add
- [ ] `io.github.airbnb.android:lottie-compose:6.1.0` (for complex animations)

## Estimated Effort
- **Sprint 1 (Foundations)**: 2 days
- **Sprint 2 (Key Screens)**: 3 days
- **Sprint 3 (Motion)**: 2 days
- **Sprint 4 (Polish)**: 1 day
- **Total**: ~8 days
