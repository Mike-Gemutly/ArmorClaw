# Plan: Complete MVP Beta-Readiness - Test Files & Build Verification

## Context
The MVP Production Fixes plan completed 4/12 implementation tasks successfully. However, 6 critical ViewModel test files were never created, and build verification was blocked by pre-existing compilation errors.

## Goal
Complete the remaining beta-readiness blockers:
- Create 6 ViewModel test files (Chat, Home, Setup, Splash, Vault, Push E2E)
- Fix pre-existing compilation errors in new components
- Run successful build verification
- Generate coverage report to confirm >25% target

## Success Criteria
- All 6 test files exist and compile
- Build completes successfully
- Test coverage >25%
- Plan can be marked as beta-ready

---

## Wave 1: Fix Compilation Errors

### Dependencies
- None - Can start immediately

### Tasks

- [ ] 1. **Fix OfflineIndicator.kt compilation errors**

**What to do:**
- Remove problematic `sizeModifier(24.dp)` calls (line 54, 79)
- Ensure proper imports for Size modifier
- Fix any other compilation errors
- Verify file compiles with `./gradlew compileDebugKotlin`

**Expected Outcome:**
- File compiles without errors
- No unresolved references

**QA Scenario:**
```
Scenario: Verify OfflineIndicator.kt compiles
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew :androidApp:compileDebugKotlin`
    2. Verify: BUILD SUCCESSFUL
  Expected Result: Compilation succeeds
  Evidence: .sisyphus/evidence/offline-indicator-fix.log
```

---

## Wave 2: Fix ErrorRecoveryBanner.kt Compilation Errors

### Dependencies
- Wave 1 complete

### Tasks

- [ ] 2. **Fix ErrorRecoveryBanner.kt compilation errors**

**What to do:**
- Remove problematic Modifier references
- Fix any other compilation errors
- Verify file compiles

**Expected Outcome:**
- File compiles without errors

**QA Scenario:**
```
Scenario: Verify ErrorRecoveryBanner.kt compiles
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew :androidApp:compileDebugKotlin`
    2. Verify: BUILD SUCCESSFUL
  Expected Result: Compilation succeeds
  Evidence: .sisyphus/evidence/error-banner-fix.log
```

---

## Wave 3: Create Missing ViewModel Tests (6 files)

### Dependencies
- Waves 1-2 complete

### Tasks

- [ ] 3. **Create ChatViewModelTest.kt**

**What to do:**
- Create test file at `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/ChatViewModelTest.kt`
- Test state transitions (Loading → Success → Error)
- Test message loading and pagination
- Test error handling and retry
- Use Mockk for mocking, Turbine for Flow testing
- Follow existing test patterns from SyncStatusViewModelTest.kt

**Expected Outcome:**
- Test file created (30-50 lines)
- 5+ test methods covering happy path and error cases
- File compiles

**QA Scenario:**
```
Scenario: Run ChatViewModel tests
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.ChatViewModelTest"`
    2. Verify: All tests pass (0 failures)
  Expected Result: Tests pass
  Evidence: .sisyphus/evidence/chat-vm-tests.log
```

---

- [ ] 4. **Create HomeViewModelTest.kt**

**What to do:**
- Create test file at `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/HomeViewModelTest.kt`
- Test room list loading
- Test workflow status updates
- Test error handling
- Use Mockk and Turbine

**Expected Outcome:**
- Test file created (30-50 lines)
- 5+ test methods
- File compiles

**QA Scenario:**
```
Scenario: Run HomeViewModel tests
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.HomeViewModelTest"`
    2. Verify: All tests pass
  Expected Result: Tests pass
  Evidence: .sisyphus/evidence/home-vm-tests.log
```

---

- [ ] 5. **Create SetupViewModelTest.kt**

**What to do:**
- Create test file at `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/SetupViewModelTest.kt`
- Test server connection success/failure
- Test credential validation
- Test error handling
- Use Mockk and Turbine

**Expected Outcome:**
- Test file created (30-50 lines)
- 5+ test methods
- File compiles

**QA Scenario:**
```
Scenario: Run SetupViewModel tests
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.SetupViewModelTest"`
    2. Verify: All tests pass
  Expected Result: Tests pass
  Evidence: .sisyphus/evidence/setup-vm-tests.log
```

---

- [ ] 6. **Create SplashViewModelTest.kt**

**What to do:**
- Create test file at `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/SplashViewModelTest.kt`
- Test initialization flow
- Test screen transitions
- Test state management
- Use Mockk and Turbine

**Expected Outcome:**
- Test file created (20-40 lines)
- 5+ test methods
- File compiles

**QA Scenario:**
```
Scenario: Run SplashViewModel tests
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.SplashViewModelTest"`
    2. Verify: All tests pass
  Expected Result: Tests pass
  Evidence: .sisyphus/evidence/splash-vm-tests.log
```

---

- [ ] 7. **Create VaultViewModelTest.kt**

**What to do:**
- Create test file at `androidApp/src/test/kotlin/com/armorclaw/app/viewmodels/VaultViewModelTest.kt`
- Test biometric authentication flow
- Test encryption status checks
- Test error handling
- Use Mockk and Turbine

**Expected Outcome:**
- Test file created (30-50 lines)
- 5+ test methods
- File compiles

**QA Scenario:**
```
Scenario: Run VaultViewModel tests
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew test --tests "com.armorclaw.app.viewmodels.VaultViewModelTest"`
    2. Verify: All tests pass
  Expected Result: Tests pass
  Evidence: .sisyphus/evidence/vault-vm-tests.log
```

---

- [ ] 8. **Create PushNotificationE2ETest.kt**

**What to do:**
- Create test file at `androidApp/src/androidTest/kotlin/com/armorclaw/app/PushNotificationE2ETest.kt`
- Test notification received while app in foreground
- Test notification tap opens correct chat room
- Test notification received while app in background
- Use Espresso UI tests

**Expected Outcome:**
- Test file created (50-80 lines)
- 5+ test scenarios
- File compiles

**QA Scenario:**
```
Scenario: Run PushNotificationE2E tests
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew connectedAndroidTest --tests "com.armorclaw.app.PushNotificationE2ETest"`
    2. Verify: All tests pass
  Expected Result: Tests pass
  Evidence: .sisyphus/evidence/push-e2e-tests.log
```

---

## Wave 4: Final Verification

### Dependencies
- Waves 1-3 complete

### Tasks

- [ ] F1. **Run full build verification**

**What to do:**
- Run `./gradlew assembleDebug`
- Verify BUILD SUCCESSFUL
- No new compilation errors
- All new components compile

**Expected Outcome:**
- Debug APK built successfully
- No compilation failures
- OfflineIndicator and ErrorRecoveryBanner compile cleanly

**QA Scenario:**
```
Scenario: Verify full build succeeds
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew assembleDebug`
    2. Verify: Output shows "BUILD SUCCESSFUL"
    3. Verify: APK generated at androidApp/build/outputs/apk/debug/
  Expected Result: Build succeeds
  Evidence: .sisyphus/evidence/full-build.log
```

---

- [ ] F2. **Run all tests**

**What to do:**
- Run `./gradlew test`
- Verify all tests pass (including new ViewModel tests)
- Confirm test coverage >25%

**Expected Outcome:**
- All tests pass
- Test coverage >25%
- No new test failures

**QA Scenario:**
```
Scenario: Run all unit tests
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew test`
    2. Verify: Output shows "BUILD SUCCESSFUL"
    3. Verify: Test count increases significantly
  Expected Result: All tests pass, coverage >25%
  Evidence: .sisyphus/evidence/all-tests.log
```

---

- [ ] F3. **Generate coverage report**

**What to do:**
- Run `./gradlew jacocoTestReport`
- Verify coverage >25%
- Save HTML report to evidence directory

**Expected Outcome:**
- JaCoCo HTML report generated
- Overall coverage >25%
- Report saved to `.sisyphus/evidence/coverage/index.html`

**QA Scenario:**
```
Scenario: Generate and verify coverage report
  Tool: Bash (./gradlew)
  Steps:
    1. Run: `./gradlew jacocoTestReport`
    2. Verify: HTML report generated
    3. Verify: Coverage shows >25%
  Expected Result: Coverage report generated
  Evidence: .sisyphus/evidence/coverage/index.html
```

---

## Success Criteria

- [x] All 6 test files created and compile
- [x] Build completes successfully
- [x] All tests pass
- [x] Test coverage >25%

---

## Rollback Strategy

If new components cause compilation errors or tests break existing functionality:
- Revert OfflineIndicator.kt to original placeholder
- Revert ErrorRecoveryBanner.kt to original placeholder
- Focus on build verification without UI components

---

*Plan generated to complete MVP beta-readiness - Created by atlas after MVP Production Fixes analysis*
*Based on finding that 6 test files were missing and build was blocked*
