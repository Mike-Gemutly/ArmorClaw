# ArmorChat Improvement Plan
> **Objective:** Address critical gaps in reliability, security compliance, and performance to achieve Production Readiness.
> **Current Status:** Feature Complete (v1.0.0)
> **Target Status:** Production Viable (v1.1.0)

---

## 🚨 Phase 1: Critical Blockers (Must-Have for Launch)
*These issues fundamentally break the core functionality (reliability) or security promises of the application.*

### 1.1 Push Notification System (FCM/UnifiedPush)
**Current State:** Placeholder FCM; reliance on background WebSocket.
**Risk:** Android OS aggressively kills background connections. Users will miss messages when the app is not foregrounded.
**Action Plan:**
- [ ] **Integrate FCM SDK:** Add Firebase Cloud Messaging dependencies.
- [ ] **Token Registration:** Implement logic to send the FCM token to the Bridge Server upon login.
- [ ] **Background Handling:** Ensure the app wakes up to process incoming push payloads.
- [ ] **Notification Channel:** Create a high-priority notification channel for message alerts.
- [ ] **Bridge Server Coordination:** Update Bridge API to trigger FCM calls on new events.

### 1.2 Cryptographic Standard Verification
**Current State:** Custom AES-256-GCM + ECDH implementation noted.
**Risk:** Incompatibility with Matrix ecosystem (Element, FluffyChat) and potential side-channel vulnerabilities if not using vetted libraries (Olm/Megolm).
**Action Plan:**
- [ ] **Audit Crypto Stack:** Verify if the implementation uses standard Matrix libraries (`libolm` or `vodozemac`) or a custom wrapper.
- [ ] **Interoperability Test:** Send an encrypted message from ArmorChat to Element to verify decryption success.
- [ ] **Key Storage:** Ensure the "Bridge" does not have access to decrypted message keys (True E2EE).
- [ ] **Remediation:** If custom crypto is used, replace with `vodozemac` bindings to align with Matrix v2 spec.

### 1.3 Architecture Trust Model Verification
**Current State:** Heavy reliance on "ArmorClaw Go Bridge."
**Risk:** If the Bridge handles decryption, the E2EE claim is invalid (Server-in-the-Middle).
**Action Plan:**
- [ ] **Data Flow Diagram:** Create a diagram showing where encryption/decryption happens.
- [ ] **Verification:** Confirm decryption happens strictly on the Android client (`DecryptionService`) and keys are never sent to the server.
- [ ] **Documentation:** Update `SECURITY.md` to explicitly state the Trust Model.

---

## 🛠 Phase 2: Stability & Hardening
*Improving the robustness of the application to handle real-world edge cases.*

### 2.1 Test Coverage Expansion
**Current State:** ~75 Tests for 19k LOC (~82% claimed coverage, likely line-coverage only).
**Target:** 300+ Unit Tests, 50+ Integration Tests.
**Action Plan:**
- [ ] **RPC Layer Tests:** Add unit tests for `BridgeRpcClient` covering malformed JSON, timeouts, and HTTP 500 errors.
- [ ] **Repository Layer:** Add integration tests verifying data mapping between RPC responses and Room Database.
- [ ] **Offline Queue:** Test the `OfflineQueue` behavior specifically when storage is full or corrupted.
- [ ] **Branch Coverage:** Enable Jacoco branch coverage reporting (target: 60% branch coverage).

### 2.2 Error Handling & Recovery
**Current State:** Basic retry logic in SetupService.
**Action Plan:**
- [ ] **WebSocket Reconnect Logic:** Implement exponential backoff with jitter for `BridgeWebSocketClient` disconnections.
- [ ] **Network Failover:** Test behavior when switching from Wi-Fi to Cellular mid-sync.
- [ ] **User Feedback:** Improve error toasts/snackbars to distinguish between "Server Down" vs. "No Internet."

---

## ⚡ Phase 3: Performance Optimization
*Improving user experience by reducing latency and resource usage.*

### 3.1 Startup Performance
**Current State:** Cold Start ~2.2s (Target: <1.5s).
**Action Plan:**
- [ ] **Dependency Injection:** Identify non-essential singletons and convert to lazy initialization.
- [ ] **Bridge Connection:** Move the initial WebSocket handshake off the main thread (strictly async).
- [ ] **Database Migration:** Pre-populate the database during the installation flow if possible.
- [ ] **Profiling:** Use Android Profiler to identify blocking I/O operations during `Application.onCreate()`.

### 3.2 Memory Management
**Current State:** Memory (Chat) ~125MB.
**Action Plan:**
- [ ] **Image Loading:** Ensure Coil/Glide is configured with a strict memory cache limit.
- [ ] **Message Paging:** Verify `LazyColumn` is using Paging 3.0 to avoid loading entire chat histories into memory.

---

## 🌍 Phase 4: Production Polish
*Preparing the app for a diverse user base and long-term maintenance.*

### 4.1 Internationalization (i18n)
**Current State:** Strings likely hardcoded in code/XML.
**Action Plan:**
- [ ] **String Extraction:** Run "Extract String Resource" across the codebase.
- [ ] **RTL Support:** Test layout mirroring for Right-to-Left languages (Arabic, Hebrew).
- [ ] **Pluralization:** Handle plural forms correctly in `strings.xml` (e.g., "1 message" vs "2 messages").

### 4.2 Accessibility (a11y)
**Current State:** Manual TalkBack support.
**Action Plan:**
- [ ] **Automated Scanning:** Integrate Android Accessibility Test Framework into CI pipeline.
- [ ] **Touch Target Sizes:** Ensure all interactive elements are min 48x48dp.
- [ ] **Contrast Ratios:** Verify color contrast meets WCAG AA standards.

---

## 📋 Summary Timeline

| Phase | Focus Area | Estimated Effort | Blocking? |
| :--- | :--- | :--- | :--- |
| **Phase 1** | Push & Crypto Spec | 2 Weeks | **YES** |
| **Phase 2** | Testing & Hardening | 2 Weeks | **YES** |
| **Phase 3** | Performance | 1 Week | NO |
| **Phase 4** | Polish & i18n | 1 Week | NO |

**Final Recommendation:** The "Production Ready" status should be revoked until Phase 1 and Phase 2 are completed. The current build serves as an excellent **Beta Candidate** but lacks the reliability (Push) and security verification (Crypto Spec) required for a production release.
