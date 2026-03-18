# Code Review Fixes Plan

> **Status:** READY FOR EXECUTION
> **Created:** 2026-03-15
> **Priority:** HIGH - Critical for Production

---

## TL;DR

> **Quick Summary**: Fix 4 critical issues identified in code review (DI anti-pattern, empty method, PII masking, optional additional encryption)
> 
> **Deliverables**:
> - Fixed VaultScreen.kt DI pattern
> - Fixed BlocklyWebView.kt executeJavaScript()
> - Fixed VaultScreen.kt PII masking
> 
> **Estimated Effort**: Quick (30 minutes)
> **Parallel Execution**: YES - 3 independent fixes

---

## Context

### Original Request
Code Review identified 5 critical issues with Score 6/10 - NEEDS_CHANGES

### Issues Analysis

| Issue | Severity | Status | Action Needed |
|-------|----------|--------|---------------|
| Missing WindowWidthSizeClass | Critical | ✅ FALSE POSITIVE | Already defined in SplitViewLayout.kt:62-66 |
| DI Anti-Pattern | Critical | ❌ NEEDS FIX | VaultScreen.kt:60 |
| Empty executeJavaScript() | Critical | ❌ NEEDS FIX | BlocklyWebView.kt:421-422 |
| SQLCipher Not Configured | Critical | ⚠️ PARTIAL | SQLCipher IS used via provider, but encryptValue is just toByteArray() |
| PII Masking Incomplete | Critical | ❌ NEEDS FIX | VaultScreen.kt:254 |

---

## Work Objectives

### Core Objective
Fix all critical code review issues to achieve production-ready status.

### Concrete Deliverables
- VaultScreen.kt with proper DI for BiometricAuth
- BlocklyWebView.kt with functional executeJavaScript()
- VaultScreen.kt with proper PII masking for OMO_LOW sensitivity

### Definition of Done
- [ ] All 3 files compile successfully
- [ ] No new lint errors introduced
- [ ] `./gradlew :androidApp:compileDebugKotlin` passes

### Must Have
- BiometricAuth injected via Koin
- executeJavaScript() actually executes JS in WebView
- OMO_LOW sensitivity shows partial mask, not full value

### Must NOT Have
- Breaking changes to existing functionality
- New dependencies
- Changes to public API signatures

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: Tests-after (verify compilation)
- **Framework**: kotlin.test

### QA Policy
- Build verification with `./gradlew :androidApp:compileDebugKotlin`
- Run existing tests with `./gradlew :shared:testDebugUnitTest`

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - 3 independent fixes):
├── Task 1: Fix DI pattern in VaultScreen.kt [quick]
├── Task 2: Fix executeJavaScript() in BlocklyWebView.kt [quick]
└── Task 3: Fix PII masking in VaultScreen.kt [quick]

Wave 2 (After Wave 1):
└── Task 4: Build verification [quick]
```

### Dependency Matrix
- **1-3**: — — 4
- **4**: 1, 2, 3 —

---

## TODOs

- [ ] 1. Fix DI Pattern in VaultScreen.kt

  **What to do**:
  - Change `biometricAuth: BiometricAuth = BiometricAuth()` to `biometricAuth: BiometricAuth = koinInject()`
  - The koinInject() import already exists (line 26)

  **File**: `androidApp/src/main/kotlin/com/armorclaw/app/screens/vault/VaultScreen.kt`

  **Current Code** (line 60):
  ```kotlin
  biometricAuth: BiometricAuth = BiometricAuth()
  ```

  **Fixed Code**:
  ```kotlin
  biometricAuth: BiometricAuth = koinInject()
  ```

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)

  **Acceptance Criteria**:
  - [ ] Line 60 uses `koinInject()` instead of `BiometricAuth()`
  - [ ] File compiles without errors

  **Commit**: YES
  - Message: `fix(vault): use Koin DI for BiometricAuth injection`
  - Files: `androidApp/src/main/kotlin/com/armorclaw/app/screens/vault/VaultScreen.kt`

---

- [ ] 2. Fix executeJavaScript() in BlocklyWebView.kt

  **What to do**:
  - Implement the executeJavaScript() method to actually execute JavaScript in the WebView
  - Use `webViewRef?.evaluateJavascript(jsCode, null)` pattern

  **File**: `androidApp/src/main/kotlin/com/armorclaw/app/studio/BlocklyWebView.kt`

  **Current Code** (lines 421-422):
  ```kotlin
  private fun executeJavaScript(jsCode: String) {
  }
  ```

  **Fixed Code**:
  ```kotlin
  private fun executeJavaScript(jsCode: String) {
      webViewRef?.evaluateJavascript(jsCode, null)
  }
  ```

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)

  **Acceptance Criteria**:
  - [ ] Method body calls `webViewRef?.evaluateJavascript(jsCode, null)`
  - [ ] File compiles without errors

  **Commit**: YES
  - Message: `fix(studio): implement executeJavaScript to actually run JS`
  - Files: `androidApp/src/main/kotlin/com/armorclaw/app/studio/BlocklyWebView.kt`

---

- [ ] 3. Fix PII Masking for OMO_LOW in VaultScreen.kt

  **What to do**:
  - Change OMO_LOW masking from showing full fieldName to showing partial mask
  - Apply same pattern as LOW sensitivity

  **File**: `androidApp/src/main/kotlin/com/armorclaw/app/screens/vault/VaultScreen.kt`

  **Current Code** (line 254):
  ```kotlin
  VaultKeySensitivity.LOW, VaultKeySensitivity.OMO_LOW -> key.fieldName
  ```

  **Fixed Code**:
  ```kotlin
  VaultKeySensitivity.LOW, VaultKeySensitivity.OMO_LOW -> {
      val name = key.fieldName
      if (name.length > 4) "${name.take(2)}****${name.takeLast(2)}" else "****"
  }
  ```

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)

  **Acceptance Criteria**:
  - [ ] OMO_LOW shows partial mask (e.g., "cr****ls" for "credentials")
  - [ ] File compiles without errors

  **Commit**: YES (groups with Task 1)
  - Message: `fix(vault): properly mask OMO_LOW sensitivity values`
  - Files: `androidApp/src/main/kotlin/com/armorclaw/app/screens/vault/VaultScreen.kt`

---

- [ ] 4. Build Verification

  **What to do**:
  - Run `./gradlew :androidApp:compileDebugKotlin` to verify all fixes compile
  - Run `./gradlew :shared:testDebugUnitTest` to verify tests still pass

  **Commands**:
  ```bash
  ./gradlew :androidApp:compileDebugKotlin
  ./gradlew :shared:testDebugUnitTest
  ```

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: None needed

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Blocked By**: Tasks 1, 2, 3

  **Acceptance Criteria**:
  - [ ] `:androidApp:compileDebugKotlin` returns BUILD SUCCESSFUL
  - [ ] No compilation errors

  **Commit**: NO (verification only)

---

## Final Verification Wave

- [ ] F1. **Plan Compliance Audit**
  Verify all 3 fixes were applied correctly:
  - VaultScreen.kt line 60 uses koinInject()
  - BlocklyWebView.kt executeJavaScript() has implementation
  - VaultScreen.kt maskValue() properly masks OMO_LOW

- [ ] F2. **Build Verification**
  Run full build to ensure no regressions

---

## Success Criteria

### Verification Commands
```bash
./gradlew :androidApp:compileDebugKotlin  # Expected: BUILD SUCCESSFUL
./gradlew :shared:testDebugUnitTest        # Expected: BUILD SUCCESSFUL
```

### Final Checklist
- [ ] All 3 files fixed
- [ ] Build passes
- [ ] No new lint errors
- [ ] Code Review Score: 6/10 → 9/10 (after fixes)

---

## Notes

### SQLCipher Clarification
The code review claimed "SQLCipher Not Actually Configured" but this is partially incorrect:
- SQLCipher IS used via `SqlCipherProvider` (line 22)
- Database operations use `sqlCipherProvider.getDatabase()` (line 36)
- The `encryptValue()` method is intentionally simple because SQLCipher handles encryption at rest
- For additional security, we could add application-level encryption, but this is optional

### WindowWidthSizeClass Clarification
The code review claimed this enum was missing, but it's already defined in SplitViewLayout.kt lines 62-66. This was a false positive.
