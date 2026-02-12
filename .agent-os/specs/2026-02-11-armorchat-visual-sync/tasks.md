# ArmorChat Visual Sync - Implementation Tasks

> **Spec:** [spec.md](spec.md)
> **Created:** 2026-02-11
> **Status:** Ready for Implementation

---

## Task Breakdown

### Phase 1: Bridge RPC Enhancements (Effort: M)

- [ ] **1.1** Add `get_sync_status` RPC method to ArmorClaw bridge
  - Returns: current sync state, last sync time, queue depth
  - File: `bridge/pkg/rpc/server.go`
  - Estimate: 2-3 hours

- [ ] **1.2** Add `get_devices` RPC method to ArmorClaw bridge
  - Returns: list of trusted devices with verification status
  - File: `bridge/pkg/rpc/server.go`
  - Estimate: 1-2 hours

- [ ] **1.3** Add `verify_device` RPC method to ArmorClaw bridge
  - Marks device as verified or removes access
  - File: `bridge/pkg/rpc/server.go`
  - Estimate: 1-2 hours

- [ ] **1.4** Add `resolve_sync_conflict` RPC method to ArmorClaw bridge
  - Forces refresh from homeserver for current device
  - File: `bridge/pkg/rpc/server.go`
  - Estimate: 2-3 hours

### Phase 2: Matrix Adapter Enhancements (Effort: L)

- [ ] **2.1** Implement sync status tracking in Matrix adapter
  - Track last successful sync time per device
  - Detect sync conflicts when messages modified elsewhere
  - File: `bridge/internal/adapter/matrix.go`
  - Estimate: 1-2 days

- [ ] **2.2** Add device trust list storage to Matrix adapter
  - Store trusted devices with metadata
  - Support device verification flow
  - File: `bridge/internal/adapter/matrix.go`
  - Estimate: 1-2 days

### Phase 3: Bridge Configuration (Effort: M)

- [ ] **3.1** Add sync configuration to bridge config system
  - Add `sync.poll_interval` setting (default: 30s)
  - Add `sync.conflict_resolution` setting (auto_refresh, user_prompt)
  - File: `bridge/pkg/config/config.go`
  - Estimate: 2-3 hours

### Phase 4: Push Notification System (Effort: L)

- [ ] **4.1** Implement push notification for sync status changes
  - Bridge sends notifications when sync state changes
  - File: `bridge/pkg/notification/sync.go`
  - Estimate: 1-2 days

### Phase 5: Mobile/Web UI Implementation (Effort: XL)

**Note:** This phase is OUT OF SCOPE for ArmorClaw bridge. ArmorChat is a separate project.

- [ ] **5.1** Implement status bar component in ArmorChat
  - Visual indicator with color coding
  - Last synced timestamp display
  - Pull-to-refresh button for sync conflicts
  - File: `armorchat/components/StatusBar.tsx`
  - Estimate: 3-5 days

- [ ] **5.2** Implement device list component in ArmorChat
  - Scrollable device list with verification badges
  - Device details modal on tap
  - File: `armorchat/components/DeviceList.tsx`
  - Estimate: 2-3 days

- [ ] **5.3** Implement connection error handling in ArmorChat
  - Error banner display for connection failures
  - Toast notifications for transient errors
  - Auto-retry logic for connection failures
  - File: `armorchat/utils/ConnectionErrorHandler.ts`
  - Estimate: 2-3 days

### Phase 6: Testing & Documentation (Effort: M)

- [ ] **6.1** Write comprehensive tests for sync status
  - Unit tests for sync state machine
  - Integration tests for device trust flow
  - File: `tests/armorchat/sync-status_test.go`
  - Estimate: 2-3 hours

- [ ] **6.2** Document ArmorChat sync API
  - API documentation for mobile client integration
  - Sequence diagrams for sync flow
  - File: `docs/api/armorchat-sync-api.md`
  - Estimate: 2-3 hours

---

## Dependencies

| Task | Depends On | Blocks |
|-------|------------|---------|
| 1.1 | None | - |
| 1.2 | None | - |
| 1.3 | None | - |
| 1.4 | None | - |
| 2.1 | 1.2 complete | Phase 2 tasks |
| 2.2 | 1.2 complete | Phase 2 tasks |
| 3.1 | None | - |
| 3.1 | 2.2 complete | Phase 2 tasks |
| 4.1 | 3.1 complete | - |
| 5.1 | None | - |
| 5.2 | None | - |
| 5.3 | None | - |
| 6.1 | 3.1, 4.1, 5.2, 5.3 complete | - |

---

## Implementation Order

**Sequential (can work in parallel):**
1. Phase 1 (all tasks) - Bridge RPC methods
2. Phase 2 (tasks 2.1-2.2) - Matrix adapter tracking
3. Phase 3 (task 3.1) - Bridge configuration
4. Phase 6 (tasks 6.1-6.2) - Testing documentation

**Out of Scope (separate project):**
5. Phase 5 (tasks 5.1-5.3) - ArmorChat mobile UI

---

## Risk Assessment

| Risk | Impact | Mitigation |
|-------|--------|------------|
| Matrix API changes | Medium | Follow Matrix spec, version carefully |
| Mobile sync timing | Low | Implement client-side polling with backoff |
| Cross-platform state | Low | Test on both iOS and Android early |
| Notification spam | Low | Implement rate limiting on push notifications |

---

**Total Estimated Effort:** ~15-18 days for bridge work, ~5-8 days for ArmorChat UI

**Next Step:** Implement Phase 1, Task 1.1 - Add `get_sync_status` RPC method
