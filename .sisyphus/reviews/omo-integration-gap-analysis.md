# Self-Review: OMO Integration Plan Gap Analysis

**Review Date**: 2025-03-14
**Plan File**: `.sisyphus/plans/omo-integration.md`

---

## Critical Gaps (MUST resolve before execution)

### Gap C1: Trixnity POC Dependency
**Issue**: Task 0.1 (Trixnity POC) is a prerequisite for Phase 1, but plan doesn't clarify what happens if POC shows current MatrixClientImpl is sufficient.

**Risk**: If POC shows no benefit, entire Phase 1 (3-4 tasks) would be wasted effort.

**Resolution Required**:
- [ ] Add decision criteria to Task 0.1 acceptance criteria: "If POC shows current implementation is better, ABORT Phase 1 and document rationale"
- [ ] Add fallback task: "0.1b: Keep current MatrixClientImpl" (if POC recommends against migration)

**Impact**: HIGH - Blocks Phase 1 execution until resolved.

---

### Gap C2: HomeScreen Scope Ambiguity
**Issue**: Plan says "Replace HomeScreen with Mission Control Dashboard" but doesn't clarify whether this is:
- **Complete removal** of HomeScreenFull.kt
- **Extension** keeping existing room list
- **New screen** separate from HomeScreen

**Risk**: We might accidentally delete useful features (room filtering, search, favorites) or create duplicate screens.

**Resolution Required**:
- [ ] Update Task 0.2 to explicitly document existing HomeScreenFull features
- [ ] Update Task 2.1 to clarify which features are preserved/removed
- [ ] Add feature comparison matrix: Existing vs Required vs Preserved

**Impact**: HIGH - Affects Phase 2 execution.

---

### Gap C3: Vault Access Discovery
**Issue**: Plan doesn't specify how users discover/access the Vault UI (Task 6.4).

**Missing from Plan**:
- Menu item to access Vault?
- Settings screen entry point?
- Quick access from Dashboard?

**Risk**: Vault UI exists but users can't find it → feature unusable.

**Resolution Required**:
- [ ] Add navigation entry point in Task 6.5 (likely: Settings → Vault)
- [ ] Add quick access button in Dashboard (Task 2.1)
- [ ] Document vault access flow in Task 6.4

**Impact**: HIGH - Vault feature would be inaccessible without fix.

---

## Minor Gaps (Should address in first sprint)

### Gap M1: Blockly Offline Mode
**Issue**: Plan assumes online Blockly (CDN or network). What if users want to edit workflows offline (no internet)?

**Current Plan**: Loads Blockly from local assets or CDN but doesn't address offline workspace editing.

**Recommended Fix**:
- [ ] Add offline workspace caching in Task 5.1
- [ ] Store workspace JSON in SQLCipher vault
- [ ] Load local workspace if network unavailable

**Impact**: MEDIUM - Friction for users in poor connectivity.

---

### Gap M2: Agent Deletion from Studio
**Issue**: Plan covers agent creation (Wizard) but not deletion from Studio UI.

**Current Plan**: AgentManagementViewModel exists but no explicit deletion workflow in Studio.

**Recommended Fix**:
- [ ] Add "Delete Agent" option in Task 5.3 (Step 0 or separate confirmation)
- [ ] Add biometric confirmation for deletion
- [ ] Archive deleted agent workflows

**Impact**: MEDIUM - Users can't clean up unused agents.

---

### Gap M3: Workflow Import/Export
**Issue**: Plan doesn't address workflow sharing between users.

**Use Case**: User creates workflow for "email automation" and wants to share with team member.

**Current Plan**: No import/export functionality.

**Recommended Fix** (mark as FUTURE CONSIDERATION):
- [ ] Add "Export Workflow" button in Task 5.3 (Step 2)
- [ ] Add "Import Workflow" button
- [ ] Define workflow JSON schema for sharing
- [ ] Add to OUT OF SCOPE if not critical for MVP

**Impact**: LOW - Nice-to-have, not blocking.

---

### Gap M4: Voice Command Placeholder Clarity
**Issue**: CommandBar (Task 4.1) has mic icon with "placeholder" note but no implementation details.

**Risk**: Users might expect voice input to work, but it's just a visual placeholder.

**Resolution Required**:
- [ ] Update Task 4.1 to clarify: "Mic icon is for FUTURE voice integration (not MVP)"
- [ ] OR add basic voice input if in scope
- [ ] Add tooltip: "Voice coming soon" to manage expectations

**Impact**: MEDIUM - Could confuse users.

---

## Ambiguous Gaps (Need clarification before execution)

### Gap A1: Split View Breakpoints
**Issue**: Task 3.1 says "responsive adaptation" but doesn't define exact breakpoints.

**Questions**:
- What screen width triggers split pane? (e.g., > 600dp)
- What screen size category uses BottomSheet? (e.g., Compact, Medium)
- Are there landscape-specific rules?

**Current Plan**: Generic "tablet/landscape" vs "phone/portrait" without numbers.

**Resolution Required**:
- [ ] Define exact breakpoint values in Task 3.1 acceptance criteria
- [ ] Use WindowSizeClass enums: COMPACT, MEDIUM, EXPANDED
- [ ] Document behavior per breakpoint in design spec

**Impact**: MEDIUM - Implementers must guess breakpoints.

---

### Gap A2: "Mission Control" Scope Definition
**Issue**: "Mission Control Dashboard" (Task 2.1) concept isn't defined with concrete examples.

**Questions**:
- What does an "Agent Card" look like specifically?
- What metrics should Dashboard show? (Agent count, success rate, uptime?)
- What are "quick actions" beyond Stop/Pause/View Log?

**Current Plan**: "Agent fleet card grid" and "metrics display" are vague.

**Resolution Required**:
- [ ] Add mockups/wireframes for Dashboard
- [ ] Define specific metrics to display
- [ ] Define exact quick action buttons

**Impact**: MEDIUM - Implementers have to interpret "Mission Control" themselves.

---

### Gap A3: Vault Access Control
**Issue**: Task 6.4 (VaultScreen) doesn't specify which users can see which PII.

**Questions**:
- Can users view ALL PII or only their own?
- Is there role-based access (admin vs user)?
- What happens if user loses biometric auth?

**Current Plan**: Generic "view stored PII" without access controls.

**Resolution Required**:
- [ ] Define access control model in Task 6.4
- [ ] Add account recovery flow for lost biometrics
- [ ] Clarify multi-user support (or explicitly mark as single-user)

**Impact**: MEDIUM - Security unclear.

---

## Gap Summary

| Severity | Count | Gap IDs |
|----------|--------|-----------|
| **Critical** | 3 | C1, C2, C3 |
| **Minor** | 4 | M1, M2, M3, M4 |
| **Ambiguous** | 3 | A1, A2, A3 |
| **Total** | 10 | |

---

## Recommended Actions Before Plan Approval

### Immediate (Must Do)
1. **Resolve Critical Gaps**:
   - Add Trixnity POC decision criteria (C1)
   - Clarify HomeScreen scope (C2)
   - Define Vault access points (C3)

### Sprint 1 (Should Do)
2. **Resolve Minor Gaps**:
   - Add Blockly offline mode (M1)
   - Add agent deletion (M2)
   - Clarify voice placeholder (M4)
   - Consider workflow import/export (M3)

### Before Execution (Should Do)
3. **Resolve Ambiguous Gaps**:
   - Define split view breakpoints (A1)
   - Provide Dashboard mockups (A2)
   - Define Vault access control (A3)

---

## Self-Assessment

**Plan Quality**: GOOD (with caveats)
- ✅ Comprehensive scope (7 phases, 40+ tasks)
- ✅ Clear acceptance criteria per task
- ✅ Verification commands defined
- ✅ Guardrails and boundaries set
- ⚠️ 3 critical gaps must be resolved before execution
- ⚠️ 7 minor/ambiguous gaps should be addressed

**Risk Level**: MEDIUM (due to critical gaps)

**Recommendation**: Address critical gaps (C1, C2, C3) and update plan before starting execution. Minor gaps can be resolved in Sprint 1.

---

**Review Status**: ✅ COMPLETE
