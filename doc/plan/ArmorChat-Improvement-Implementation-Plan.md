# ArmorChat Improvement Implementation Plan
> **Created:** 2026-02-16
> **Based On:** ArmorChat-Improvement.md review and codebase analysis
> **Status:** In Progress
> **Last Updated:** 2026-02-16

---

## Executive Summary

Based on a comprehensive codebase analysis, this plan prioritizes improvements based on current implementation status and production requirements.

### Current State Assessment

| Phase | Area | Status | Priority |
|-------|------|--------|----------|
| 1.1 | Push Notifications (FCM) | ✅ Complete | Done |
| 1.2 | Crypto Standard | ⚠️ Partial | Medium |
| 1.3 | Trust Model | ✅ Complete | Done |
| 2.1 | Test Coverage | ⚠️ In Progress | **HIGH** |
| 2.2 | Error Handling | ✅ Complete | Done |
| 2.3 | Distributed Tracing | ✅ Complete | Done |
| 2.4 | JSON Structured Logging | ✅ Complete | Done |
| 2.5 | Error Analytics | ✅ Complete | Done |
| 3.1 | Startup Performance | ❌ Not Started | Medium |
| 3.2 | Memory Management | ❌ Not Started | High |
| 4.1 | Internationalization | ❌ Not Started | Medium |
| 4.2 | Accessibility | ⚠️ Basic Only | Medium |

---

## 🚨 Priority 1: Critical Gaps (Week 1-2)

### 1.1 Test Coverage Expansion

**Current State:** 26+ test files (was 23, added 3 new test files)
**Progress:** Added BridgeRpcClientTest, BridgeWebSocketClientTest, ErrorAnalyticsTest

#### Implementation Tasks

**Week 1: Core Infrastructure Tests**
- [x] **1.1.1** Create `BridgeRpcClientTest.kt` (20+ tests) ✅
  - Test RpcResult sealed class behavior
  - Test BridgeConfig defaults
  - Test JSON-RPC request/response serialization
  - Test all response models
  - Test JsonRpcError codes
  - Test MockBridgeRpcClient behavior
  - Test OperationContext for RPC

- [x] **1.1.2** Create `BridgeWebSocketClientTest.kt` (20+ tests) ✅
  - Test WebSocketState transitions
  - Test WebSocketConfig defaults
  - Test all 12 BridgeEvent types serialization
  - Test BridgeEventContent
  - Test MockBridgeWebSocketClient behavior
  - Test exponential backoff calculation

- [x] **1.1.3** Create `ErrorAnalyticsTest.kt` (25+ tests) ✅
  - Test error rate calculation
  - Test threshold alerts (warning/critical)
  - Test statistics aggregation
  - Test category/source/code grouping
  - Test StateFlow exposure
  - Test clear operations
  - Test edge cases

**Week 2: Repository & Use Case Tests**
- [x] **1.1.4** Create `MessageRepositoryImplTest.kt` (15 tests) ✅
  - Test message CRUD operations
  - Test offline queue behavior
  - Test sync conflict resolution
  - Test encryption/decryption flow

- [x] **1.1.5** Create `RoomRepositoryImplTest.kt` (12 tests) ✅
  - Test room membership handling
  - Test room state caching
  - Test pagination

- [x] **1.1.6** Create Use Case Tests (20 tests across files) ✅
  - `SendMessageUseCaseTest.kt` (already existed)
  - `LoadMessagesUseCaseTest.kt` ✅
  - `LoginUseCaseTest.kt` ✅
  - `LogoutUseCaseTest.kt` ✅

**Test Infrastructure Setup:**
- [x] **1.1.7** Configure JaCoCo for branch coverage reporting ✅
- [ ] **1.1.8** Add test coverage CI gate (min 60% branch coverage)

### 1.2 Memory Management - Paging 3 Implementation

**Current State:** No Paging 3 implementation found
**Risk:** OutOfMemoryError on large chat histories

#### Implementation Tasks

- [ ] **1.2.1** Add Paging 3 dependencies to `shared/build.gradle.kts`
  ```kotlin
  implementation(libs.sqldelight.paging.extensions)
  implementation("androidx.paging:paging-compose:3.2.1")
  ```

- [ ] **1.2.2** Create `MessagePagingSource.kt`
  - Implement `PagingSource<Int, Message>`
  - Integrate with SQLDelight queries
  - Handle network/database sync

- [ ] **1.2.3** Update `ChatViewModel.kt`
  - Replace `List<Message>` with `PagingData<Message>`
  - Use `Pager` with configured page size (50)
  - Implement `Flow<PagingData<Message>>`

- [ ] **1.2.4** Update `ChatScreen.kt`
  - Use `LazyPagingItems` in LazyColumn
  - Add loading states for pages
  - Add retry mechanism for failed pages

- [ ] **1.2.5** Configure Coil memory cache limits
  ```kotlin
  ImageLoader.Builder(context)
      .memoryCachePolicy(CachePolicy.ENABLED)
      .memoryCache {
          MemoryCache.Builder(context)
              .maxSizePercent(0.1) // 10% of available memory
              .build()
      }
  ```

---

## 📋 Priority 2: Performance & Polish (Week 3)

### 2.1 Startup Performance Optimization

**Current State:** Cold Start ~2.2s
**Target:** <1.5s

#### Implementation Tasks

- [ ] **2.1.1** Create startup profile with Android Profiler
- [ ] **2.1.2** Convert non-essential singletons to lazy initialization
  - Audit `AppModules.kt` for lazy-eligible dependencies
  - Move Koin initialization to background thread where possible

- [ ] **2.1.3** Optimize Application.onCreate()
  ```kotlin
  // Before
  override fun onCreate() {
      super.onCreate()
      startKoin { ... }  // Blocking
      initializeLogger()  // Blocking
      initializeCrashReporting()  // Blocking
  }

  // After
  override fun onCreate() {
      super.onCreate()
      // Only essential sync initialization
      startKoin { ... }
      // Defer to background
      CoroutineScope(Dispatchers.IO).launch {
          initializeLogger()
          initializeCrashReporting()
      }
  }
  ```

- [ ] **2.1.4** Add startup metrics logging
- [ ] **2.1.5** Verify cold start <1.5s with profiling

### 2.2 Internationalization (i18n)

**Current State:** Hardcoded strings, no locale support

#### Implementation Tasks

- [ ] **2.2.1** Extract all string literals to `strings.xml`
  - Use Android Studio "Extract String Resource" refactoring
  - Priority order: UI labels > error messages > placeholders

- [ ] **2.2.2** Create string resource structure
  ```
  res/
  ├── values/
  │   └── strings.xml (English - base)
  ├── values-es/
  │   └── strings.xml (Spanish)
  ├── values-de/
  │   └── strings.xml (German)
  └── values-ar/
      └── strings.xml (Arabic - RTL test)
  ```

- [ ] **2.2.3** Implement pluralization rules
  ```xml
  <plurals name="message_count">
      <item quantity="one">%d message</item>
      <item quantity="other">%d messages</item>
  </plurals>
  ```

- [ ] **2.2.4** Add RTL layout support
  - Add `android:supportsRtl="true"` to manifest
  - Test with Arabic locale
  - Use `start/end` instead of `left/right` in layouts

- [ ] **2.2.5** Create `LocalizationManager.kt` for runtime locale switching

### 2.3 Accessibility (a11y) Completion

**Current State:** Basic configuration, missing implementations

#### Implementation Tasks

- [ ] **2.3.1** Add content descriptions to all interactive elements
  - Priority: ChatScreen > Navigation > Settings > Profile
  - Use `contentDescription` parameter in all Icons

- [ ] **2.3.2** Verify touch target sizes (min 48x48dp)
  - Create automated lint rule
  - Audit all Button, IconButton, and clickable components

- [ ] **2.3.3** Implement focus navigation
  - Add `focusable()` modifiers
  - Implement keyboard navigation
  - Test with TalkBack

- [ ] **2.3.4** Add semantic properties
  ```kotlin
  Modifier.semantics {
      contentDescription = "Send message button"
      stateDescription = if (enabled) "Enabled" else "Disabled"
      role = Role.Button
  }
  ```

- [ ] **2.3.5** Integrate Accessibility Test Framework
  ```kotlin
  androidTestImplementation("com.google.android.apps.common.testing.accessibility.framework:accessibility-test-framework:4.0.0")
  ```

---

## 🔒 Priority 3: Security Verification (Week 4)

### 3.1 Cryptographic Interoperability Testing

**Current State:** Custom crypto implementation (ECDH, AES-256-GCM, Ed25519)
**Risk:** May not interoperate with Matrix ecosystem (Element, FluffyChat)

#### Implementation Tasks

- [ ] **3.1.1** Create interoperability test matrix
  | Client | Encryption | Decryption | Status |
  |--------|------------|------------|--------|
  | ArmorChat → Element | ? | ? | Untested |
  | Element → ArmorChat | ? | ? | Untested |
  | FluffyChat ↔ ArmorChat | ? | ? | Untested |

- [ ] **3.1.2** Document current crypto implementation
  - Create `doc/Cryptography.md`
  - Document key generation, exchange, storage
  - Document encryption/decryption flow

- [ **3.1.3** Evaluate vodozemac integration
  - Research vodozemac Kotlin bindings
  - Assess migration complexity
  - Create migration plan if needed

- [ ] **3.1.4** Update `SECURITY.md`
  - Document trust model
  - Document key storage
  - Document cryptographic choices

### 3.2 Security Audit Preparation

- [ ] **3.2.1** Run automated security scanner (QARK, MobSF)
- [ ] **3.2.2** Review certificate pinning implementation
- [ ] **3.2.3** Audit network traffic for sensitive data exposure
- [ ] **3.2.4** Review SQLCipher key management

---

## 📊 Progress Tracking

### Metrics Dashboard

| Metric | Current | Target | Status |
|--------|---------|--------|--------|
| Unit Tests | 29+ (100+ test methods) | 300 | ⚠️ 33% |
| Integration Tests | 1 | 50 | 🚨 2% |
| Branch Coverage | ~0% | 60% | 🚨 (JaCoCo configured) |
| Cold Start | 2.2s | <1.5s | ⚠️ |
| Memory (Chat) | 125MB | <100MB | ⚠️ |
| i18n Languages | 1 | 4 | 🚨 |
| a11y Score | Basic | WCAG AA | ⚠️ |

### Weekly Milestones

| Week | Milestone | Deliverables | Status |
|------|-----------|--------------|--------|
| 1 | Test Foundation | 50+ new tests, JaCoCo configured | ✅ Done |
| 2 | Test + Memory | 100+ tests total, Paging 3 implemented | ✅ Tests Done |
| 3 | Performance | Cold start <1.5s, i18n started | ⏳ Pending |
| 4 | Polish + Security | i18n complete, a11y WCAG AA, crypto verified | ⏳ Pending |

---

## 🔄 Revised Timeline

| Phase | Focus | Effort | Dependencies |
|-------|-------|--------|--------------|
| **Phase 1** (Week 1-2) | Tests + Memory | 2 weeks | None |
| **Phase 2** (Week 3) | Performance + i18n + a11y | 1 week | Phase 1 |
| **Phase 3** (Week 4) | Security Verification | 1 week | Phase 2 |
| **Phase 4** (Week 5+) | Optional: vodozemac migration | 2 weeks | Phase 3 |

---

## 🎯 Acceptance Criteria

Before declaring "Production Ready v1.1.0":

- [ ] 300+ unit tests passing
- [ ] 50+ integration tests passing
- [ ] Branch coverage ≥60%
- [ ] Cold start <1.5s
- [ ] Memory usage <100MB in chat
- [ ] 4 languages supported
- [ ] WCAG AA compliance verified
- [ ] Matrix interoperability confirmed
- [ ] SECURITY.md documented
- [ ] No critical security findings

---

## 📝 Notes

### Already Complete (No Action Needed)
- Push Notifications (FCM) - Dependencies integrated
- Architecture Trust Model - Client-side decryption verified
- Error Handling & Recovery - WebSocket reconnection with backoff implemented
- Distributed Tracing - X-Request-ID/X-Trace-ID headers (added 2026-02-16)
- Error Analytics - Rate tracking and alerting (added 2026-02-16)
- JSON Structured Logging - For log aggregation systems (added 2026-02-16)
- Core Infrastructure Tests - BridgeRpcClient, BridgeWebSocket, ErrorAnalytics (added 2026-02-16)
- JaCoCo Configuration - Test coverage reporting enabled (added 2026-02-16)

### Deferred Items
- vodozemac migration - Significant effort, can be Phase 4
- iOS support - Out of scope for v1.1.0
