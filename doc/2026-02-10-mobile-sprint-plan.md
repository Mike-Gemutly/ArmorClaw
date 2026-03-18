# ArmorClaw Mobile App - Updated Sprint Plan

> **Document Purpose:** Complete implementation plan with all gap fixes integrated
> **Date Created:** 2026-02-10
> **Total Timeline:** 21-30 weeks (updated from 14-20 weeks)
> **Status:** Ready for Execution

---

## Sprint Overview

| Sprint | Duration | Focus | Deliverables |
|--------|----------|-------|--------------|
| **Sprint 0** | 2 weeks | Preparation | Wireframes, KMP schemas, contracts |
| **Sprint 1** | 3 weeks | Foundation Core | Auth, onboarding, offline/sync base |
| **Sprint 2** | 3 weeks | Chat Foundation | Matrix integration, message handling |
| **Sprint 3** | 3 weeks | UX Foundation | Input validation, empty states, loading |
| **Sprint 4** | 3 weeks | Security Core | Cert pinning, biometric, secure clipboard |
| **Sprint 5** | 3 weeks | Intelligence | Rich responses, typing, search |
| **Sprint 6** | 3 weeks | Tasks & Memory | Task automation, memory system |
| **Sprint 7** | 2 weeks | Collaboration | Sharing, handoff, analytics |
| **Sprint 8** | 2 weeks | Polish & Launch | Performance, testing, deployment |

---

## Sprint 0: Preparation (2 weeks)

**Goal:** Complete design and setup before implementation

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S0-1** | Create onboarding flow wireframes | 3 | None |
| **S0-2** | Design empty states and loading skeletons | 2 | None |
| **S0-3** | Define KMP shared schemas (events, commands) | 5 | None |
| **S0-4** | Write Matrix event contracts (JSON examples) | 3 | None |
| **S0-5** | Define agent task schemas with examples | 3 | None |
| **S0-6** | Setup CI/CD pipeline (GitHub Actions) | 5 | None |
| **S0-7** | Create code signing configuration | 3 | None |
| **S0-8** | Setup crash reporting (Sentry/Firebase) | 2 | None |
| **S0-9** | Security audit requirements document | 2 | None |
| **S0-10** | Performance benchmark definitions | 2 | None |

### Subtasks

**S0-1: Onboarding Wireframes**
- [ ] Design welcome screen layout
- [ ] Design security explanation screen
- [ ] Design server connection screen
- [ ] Design permissions request screen
- [ ] Design completion celebration screen
- [ ] Design tutorial overlay screens
- [ ] Document all transitions and animations

**S0-3: KMP Schemas**
- [ ] Define MessageContent shared class
- [ ] Define RoomState shared class
- [ ] Define UserPresence shared class
- [ ] Define CommandSchema shared class
- [ ] Define EventSchema shared class
- [ ] Create serialization contracts

**S0-6: CI/CD Pipeline**
- [ ] Create Android build workflow
- [ ] Create iOS build workflow
- [ ] Setup automated testing
- [ ] Configure code signing
- [ ] Setup beta distribution (TestFlight/Play Console)

---

## Sprint 1: Foundation Core (3 weeks)

**Goal:** Core authentication and onboarding

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S1-1** | Implement onboarding flow manager | 5 | S0-1 |
| **S1-2** | Build server connection UI | 5 | S0-3 |
| **S1-3** | Implement QR code scanner | 3 | S1-2 |
| **S1-4** | Build permissions request flow | 3 | S1-1 |
| **S1-5** | Implement demo server connection | 3 | S1-2 |
| **S1-6** | Create empty state components | 3 | S0-2 |
| **S1-7** | Build loading skeleton screens | 3 | S0-2 |
| **S1-8** | Implement Matrix SDK integration | 8 | S0-3 |
| **S1-9** | Create secure token storage | 5 | S1-8 |
| **S1-10** | Build connection state machine | 3 | S1-2 |

### Subtasks

**S1-1: Onboarding Manager**
- [ ] Implement OnboardingManager class
- [ ] Create onboarding state machine
- [ ] Build skip/resume logic
- [ ] Implement progress persistence
- [ ] Create completion tracking
- [ ] Write unit tests for state transitions

**S1-8: Matrix Integration**
- [ ] Integrate Matrix SDK for Android
- [ ] Integrate Matrix SDK for iOS
- [ ] Implement login flow
- [ ] Implement room sync
- [ ] Implement E2EE support
- [ ] Create Matrix client wrapper

**S1-9: Secure Storage**
- [ ] Implement Android Keystore integration
- [ ] Implement iOS Keychain integration
- [ ] Create token encryption/decryption
- [ ] Build secure preferences wrapper
- [ ] Write security tests

---

## Sprint 2: Chat Foundation (3 weeks)

**Goal:** Core messaging functionality

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S2-1** | Build message list component | 5 | S1-8 |
| **S2-2** | Implement message input UI | 3 | S1-8 |
| **S2-3** | Create message rendering (bubbles) | 5 | S2-1 |
| **S2-4** | Implement offline message queue | 8 | S1-9 |
| **S2-5** | Build sync state machine | 5 | S2-4 |
| **S2-6** | Implement conflict resolution | 5 | S2-5 |
| **S2-7** | Create sync status indicators | 3 | S2-5 |
| **S2-8** | Implement network monitoring | 3 | S2-5 |
| **S2-9** | Build periodic sync worker | 3 | S2-5 |
| **S2-10** | Create message database schema | 5 | S2-4 |

### Subtasks

**S2-4: Offline Queue**
- [ ] Implement message queueing
- [ ] Create pending message storage
- [ ] Build retry logic with exponential backoff
- [ ] Implement message expiry
- [ ] Create queue processor
- [ ] Write tests for offline scenarios

**S2-5: Sync Machine**
- [ ] Implement sync state machine
- [ ] Create sync triggers
- [ ] Build error recovery
- [ ] Implement batch processing
- [ ] Create sync metadata tracking
- [ ] Write integration tests

---

## Sprint 3: UX Foundation (3 weeks)

**Goal:** Input validation and user experience improvements

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S3-1** | Implement input validation | 3 | S2-2 |
| **S3-2** | Create PII warning indicators | 3 | S3-1 |
| **S3-3** | Build character count display | 2 | S2-2 |
| **S3-4** | Implement send button state logic | 2 | S3-1 |
| **S3-5** | Create pull-to-refresh | 3 | S2-5 |
| **S3-6** | Build error state displays | 3 | S2-5 |
| **S3-7** | Implement error recovery UI | 3 | S3-6 |
| **S3-8** | Create connection status banner | 2 | S2-8 |
| **S3-9** | Build retry UI components | 2 | S3-7 |
| **S3-10** | Implement push notification setup | 5 | S1-8 |

### Subtasks

**S3-1: Input Validation**
- [ ] Create input validator
- [ ] Implement empty validation
- [ ] Implement length validation
- [ ] Implement PII detection
- [ ] Create validation UI feedback
- [ ] Write validation tests

**S3-10: Push Notifications**
- [ ] Setup Firebase Cloud Messaging
- [ ] Setup APNs for iOS
- [ ] Implement Matrix push gateway
- [ ] Create notification categories
- [ ] Build notification handling
- [ ] Implement notification preferences

---

## Sprint 4: Security Core (3 weeks)

**Goal:** Certificate pinning, biometric auth, secure clipboard

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S4-1** | Implement certificate pinning | 8 | S1-8 |
| **S4-2** | Create pin extraction utility | 3 | S4-1 |
| **S4-3** | Implement remote pin updates | 5 | S4-1 |
| **S4-4** | Build biometric authentication | 8 | S1-9 |
| **S4-5** | Create biometric policy enforcer | 3 | S4-4 |
| **S4-6** | Implement secure clipboard | 5 | None |
| **S4-7** | Create auto-clear timer | 3 | S4-6 |
| **S4-8** | Build clipboard UI indicators | 2 | S4-7 |
| **S4-9** | Implement security logger | 3 | None |
| **S4-10** | Create security event monitoring | 3 | S4-9 |

### Subtasks

**S4-1: Certificate Pinning**
- [ ] Implement OkHttp interceptor
- [ ] Create SPKI hash extraction
- [ ] Build pin matching logic
- [ ] Implement pin expiry handling
- [ ] Create debug bypass option
- [ ] Write security tests

**S4-4: Biometric Auth**
- [ ] Implement Android BiometricPrompt
- [ ] Implement iOS LocalAuthentication
- [ ] Create token encryption/decryption
- [ ] Build authentication flow
- [ ] Implement session timeout
- [ ] Create fallback for no biometric

---

## Sprint 5: Intelligence (3 weeks)

**Goal:** Rich responses, typing indicators, search

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S5-1** | Implement typing indicators | 3 | S2-3 |
| **S5-2** | Create read receipts | 3 | S5-1 |
| **S5-3** | Build rich response renderer | 5 | S2-3 |
| **S5-4** | Implement message search | 5 | S2-10 |
| **S5-5** | Create search filters UI | 3 | S5-4 |
| **S5-6** | Build conversation threading | 3 | S2-3 |
| **S5-7** | Implement message reactions | 2 | S5-1 |
| **S5-8** | Create reply/quote UI | 3 | S5-6 |
| **S5-9** | Build message forwarding | 2 | S5-8 |
| **S5-10** | Implement screenshot detection | 3 | S4-9 |

### Subtasks

**S5-1: Typing Indicators**
- [ ] Implement Matrix typing notifications
- [ ] Create typing indicator UI
- [ ] Build typing state management
- [ ] Implement debouncing
- [ ] Create agent processing indicator
- [ ] Write tests

**S5-4: Search**
- [ ] Implement Matrix search API
- [ ] Create local search index
- [ ] Build search UI
- [ ] Implement search filters
- [ ] Create search results display
- [ ] Handle empty results

---

## Sprint 6: Tasks & Memory (3 weeks)

**Goal:** Task automation and memory system

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S6-1** | Implement task start approval | 5 | S2-3 |
| **S6-2** | Build task progress UI | 5 | S6-1 |
| **S6-3** | Create task tree visualization | 5 | S6-2 |
| **S6-4** | Implement task stop/pause | 3 | S6-2 |
| **S6-5** | Build TTL warning display | 2 | S6-2 |
| **S6-6** | Implement memory category view | 3 | S6-1 |
| **S6-7** | Create memory toggle UI | 2 | S6-6 |
| **S6-8** | Build feedback UI (👍/👎) | 2 | S2-3 |
| **S6-9** | Implement model selector UI | 3 | S2-3 |
| **S6-10** | Create cost display component | 2 | S6-9 |

### Subtasks

**S6-1: Task Approval**
- [ ] Create approval modal
- [ ] Implement task description display
- [ ] Build consent UI
- [ ] Implement task start RPC
- [ ] Create task monitoring
- [ ] Handle task errors

**S6-2: Task Progress**
- [ ] Implement real-time updates
- [ ] Create progress indicator
- [ ] Build step-by-step display
- [ ] Implement cost tracking
- [ ] Create notification system
- [ ] Handle task completion

---

## Sprint 7: Collaboration (2 weeks)

**Goal:** Sharing, handoff, GDPR features

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S7-1** | Implement conversation sharing | 5 | S5-6 |
| **S7-2** | Create share UI and URL generation | 3 | S7-1 |
| **S7-3** | Build shared conversation view | 3 | S7-2 |
| **S7-4** | Implement human handoff | 3 | S6-2 |
| **S7-5** | Create handoff notification | 2 | S7-4 |
| **S7-6** | Implement GDPR data export | 5 | S2-10 |
| **S7-7** | Build account deletion flow | 3 | S7-6 |
| **S7-8** | Create consent management UI | 3 | S3-10 |
| **S7-9** | Implement analytics integration | 3 | S4-9 |
| **S7-10** | Build OTA configuration updates | 3 | S1-8 |

### Subtasks

**S7-6: GDPR Export**
- [ ] Create data export manager
- [ ] Implement message export
- [ ] Build metadata export
- [ ] Create export job queue
- [ ] Build export status UI
- [ ] Implement format options (JSON, CSV)

**S7-7: Account Deletion**
- [ ] Create deletion request flow
- [ ] Implement cooldown period
- [ ] Build confirmation UI
- [ ] Create cancellation option
- [ ] Implement data wipe
- [ ] Send confirmation notification

---

## Sprint 8: Polish & Launch (2 weeks)

**Goal:** Performance optimization and deployment

### Stories

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| **S8-1** | Implement app size optimization | 3 | All |
| **S8-2** | Build performance profiling | 3 | All |
| **S8-3** | Create memory leak detection | 3 | All |
| [ ]S8-4** | Implement feature flags | 3 | S7-10 |
| **S8-5** | Build A/B testing framework | 3 | S8-4 |
| **S8-6** | Create rollback mechanism | 2 | S8-4 |
| **S8-7** | Implement security audit | 5 | All |
| **S8-8** | Build E2E test suite | 5 | All |
| **S8-9** | Create store submission assets | 2 | All |
| **S8-10** | Deploy to beta testing | 2 | All |

### Subtasks

**S8-1: App Size**
- [ ] Enable R8/ProGuard
- [ ] Configure resource shrinking
- [ ] Implement APK splits by ABI
- [ ] Remove unused resources
- [ ] Optimize image assets
- [ ] Measure final APK size

**S8-7: Security Audit**
- [ ] Run penetration testing
- [ ] Review certificate pinning
- [ ] Audit biometric implementation
- [ ] Test clipboard security
- [ ] Review data encryption
- [ ] Document findings

---

## Gap Fix Integration Matrix

### Critical Gaps Integrated

| Gap | Original Plan | Fixed In | Status |
|-----|---------------|----------|--------|
| Offline/Sync Strategy | ❌ Missing | Sprint 2 | ✅ S2-4, S2-5 |
| Matrix Disconnection Recovery | ❌ Missing | Sprint 2 | ✅ S2-5, S2-8 |
| Push Notifications | ❌ Missing | Sprint 3 | ✅ S3-10 |
| Onboarding Flow | ❌ Missing | Sprint 1 | ✅ S1-1 |
| Certificate Pinning | ❌ Missing | Sprint 4 | ✅ S4-1 |
| GDPR Data Export | ❌ Missing | Sprint 7 | ✅ S7-6 |
| Account Deletion | ❌ Missing | Sprint 7 | ✅ S7-7 |
| Input Validation UX | ❌ Missing | Sprint 3 | ✅ S3-1 |
| Typing Indicators | ❌ Missing | Sprint 5 | ✅ S5-1 |
| Read Receipts | ❌ Missing | Sprint 5 | ✅ S5-2 |
| Search Functionality | ❌ Missing | Sprint 5 | ✅ S5-4 |
| Feature Flags | ❌ Missing | Sprint 8 | ✅ S8-4 |
| Biometric Integration | ❌ Missing | Sprint 4 | ✅ S4-4 |
| Crash Reporting | ❌ Missing | Sprint 0 | ✅ S0-8 |
| CI/CD Pipeline | ❌ Missing | Sprint 0 | ✅ S0-6 |
| A/B Testing | ❌ Missing | Sprint 8 | ✅ S8-5 |

### All Gaps by Sprint

**Sprint 0:**
- S0-6: CI/CD Pipeline
- S0-8: Crash Reporting

**Sprint 1:**
- S1-1: Onboarding Flow
- S1-6: Empty States
- S1-7: Loading States

**Sprint 2:**
- S2-4: Offline Queue
- S2-5: Sync Machine
- S2-8: Network Monitoring

**Sprint 3:**
- S3-1: Input Validation
- S3-10: Push Notifications

**Sprint 4:**
- S4-1: Certificate Pinning
- S4-4: Biometric Auth
- S4-6: Secure Clipboard

**Sprint 5:**
- S5-1: Typing Indicators
- S5-2: Read Receipts
- S5-4: Search

**Sprint 6:**
- (No new gaps, task/memory were in original plan)

**Sprint 7:**
- S7-6: GDPR Export
- S7-7: Account Deletion

**Sprint 8:**
- S8-1: App Size Optimization
- S8-4: Feature Flags
- S8-5: A/B Testing

---

## Story Points Summary

| Sprint | Total Points | Duration | Velocity/Week |
|--------|--------------|----------|---------------|
| Sprint 0 | 31 | 2 weeks | 15.5 |
| Sprint 1 | 43 | 3 weeks | 14.3 |
| Sprint 2 | 43 | 3 weeks | 14.3 |
| Sprint 3 | 29 | 3 weeks | 9.7 |
| Sprint 4 | 43 | 3 weeks | 14.3 |
| Sprint 5 | 30 | 3 weeks | 10.0 |
| Sprint 6 | 28 | 3 weeks | 9.3 |
| Sprint 7 | 33 | 2 weeks | 16.5 |
| Sprint 8 | 31 | 2 weeks | 15.5 |
| **TOTAL** | **311** | **24 weeks** | **13.0 avg** |

---

## Definition of Done

Each story is complete when:

- [ ] Code is written and follows style guidelines
- [ ] Unit tests pass (>80% coverage)
- [ ] Integration tests pass
- [ ] Code is reviewed by at least one other developer
- [ ] Documentation is updated (if applicable)
- [ ] No critical security vulnerabilities
- [ ] Performance benchmarks met
- [ ] Accessibility requirements met
- [ ] User acceptance testing passed

---

## Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Matrix SDK changes break integration | Medium | High | Lock SDK version, create abstraction layer |
| Biometric auth not supported on some devices | High | Low | Provide fallback password option |
| Offline sync conflicts increase support load | Medium | Medium | Implement robust conflict resolution UI |
| Certificate pinning blocks valid connections | Low | High | Add debug bypass, provide update mechanism |
| GDPR requirements change | Medium | Medium | Flexible export system, regular audits |
| App size exceeds store limits | Low | High | Early optimization, APK splits |
| Push notification delivery unreliable | Medium | Medium | In-app notifications as fallback |

---

## Dependencies

### External Dependencies
- Matrix SDK stable release
- Firebase Console setup
- Apple Developer account
- Google Play Console access
- Sentry account (for crash reporting)

### Internal Dependencies
- ArmorClaw Bridge v1.0+
- Matrix Conduit server
- SSL certificates for homeserver
- Agent container images

---

## Release Criteria

**Phase 1 (Beta):**
- ✅ All Sprint 0-4 stories complete
- ✅ Security audit passed
- ✅ E2E tests passing
- ✅ Beta users recruited

**Phase 2 (Public Beta):**
- ✅ All Sprint 0-6 stories complete
- ✅ Performance benchmarks met
- ✅ Crash rate < 1%
- ✅ User feedback positive (>4.0 rating)

**Phase 3 (Production):**
- ✅ All stories complete
- ✅ Security audit clean
- ✅ Load testing passed
- ✅ Documentation complete
- ✅ Support team trained

---

## Communication Plan

**Daily:**
- Standup (15 minutes)
- Update task board

**Weekly:**
- Sprint review (demo completed stories)
- Sprint planning (next sprint)
- Risk review

**Milestone:**
- Stakeholder update
- Progress report
- Blocker summary

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-10
**Status:** Ready for Sprint Planning
