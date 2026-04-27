# Admin Panel ↔ Bridge RPC Endpoint Parity Matrix

> Generated: 2026-04-27  
> Scope: Cross-reference of every RPC method called by the admin panel against handlers registered in the Bridge RPC server.  
> **Documentation only — no implementation changes.**

## Summary

| Status | Count | Description |
|--------|-------|-------------|
| REGISTERED | 8 | Present in main 89-method handler map (`server.go:857-951`) |
| LOCKDOWN | 7 | Referenced in lockdown subsystem (`pkg/lockdown/bonding.go`) but NOT in main handler map |
| MISSING | 18 | No handler found anywhere in the bridge codebase |
| **Total** | **33** | All unique RPC methods called by admin panel |

### Handler Sources Audited

| Source | File | Method Count |
|--------|------|-------------|
| Main handler map | `bridge/pkg/rpc/server.go:857-951` | 89 methods |
| Public handlers | `bridge/pkg/rpc/public_handlers.go` | 5 methods (system.health, system.config, system.info, system.time, device.validate) |
| Enforcement handlers | `bridge/pkg/enforcement/rpc_handlers.go` | 6 functions (not wired into main map) |
| Lockdown subsystem | `bridge/pkg/lockdown/bonding.go` | Lists permitted methods per mode (not RPC handlers) |

---

## Parity Matrix

### Registered Methods (8)

Methods present in the main RPC handler map. These will resolve successfully.

| # | Method | Source File (Admin) | Handler Location | Status | Notes |
|---|--------|--------------------|-----------------|--------|-------|
| 1 | `device.list` | `bridgeApi.ts:239` | `server.go:940` | REGISTERED | `s.handleDeviceList` |
| 2 | `device.get` | `bridgeApi.ts:243` | `server.go:941` | REGISTERED | `s.handleDeviceGet` |
| 3 | `device.approve` | `bridgeApi.ts:247` | `server.go:942` | REGISTERED | `s.handleDeviceApprove` |
| 4 | `device.reject` | `bridgeApi.ts:250` | `server.go:943` | REGISTERED | `s.handleDeviceReject` |
| 5 | `invite.create` | `bridgeApi.ts:279` | `server.go:945` | REGISTERED | `s.handleInviteCreate` |
| 6 | `invite.list` | `bridgeApi.ts:283` | `server.go:944` | REGISTERED | `s.handleInviteList` |
| 7 | `invite.revoke` | `bridgeApi.ts:287` | `server.go:946` | REGISTERED | `s.handleInviteRevoke` |
| 8 | `invite.validate` | `bridgeApi.ts:291` | `server.go:947` | REGISTERED | `s.handleInviteValidate` |

### Lockdown Subsystem Methods (7)

Referenced in `pkg/lockdown/bonding.go` `getAvailableMethods()` as methods permitted during lockdown mode transitions. Logic exists in the lockdown package but these are **NOT registered in the main RPC handler map** — the RPC server's `Handle()` dispatch (`server.go:292`) will return "method not found" for these.

| # | Method | Source File (Admin) | Handler Location | Status | Notes |
|---|--------|--------------------|-----------------|--------|-------|
| 9 | `lockdown.status` | `bridgeApi.ts:191` | `lockdown/bonding.go:350,356,368,373` | LOCKDOWN | Available in all modes; NOT in RPC handler map |
| 10 | `lockdown.get_challenge` | `bridgeApi.ts:195` | `lockdown/bonding.go:351` | LOCKDOWN | Available in ModeLockdown only; NOT in RPC handler map |
| 11 | `lockdown.claim_ownership` | `bridgeApi.ts:213` | `lockdown/bonding.go:352` | LOCKDOWN | Available in ModeLockdown only; NOT in RPC handler map |
| 12 | `security.get_categories` | `bridgeApi.ts:222` | `lockdown/bonding.go:357` | LOCKDOWN | Available in ModeBonding/ModeConfiguring; NOT in RPC handler map |
| 13 | `security.set_category` | `bridgeApi.ts:226` | `lockdown/bonding.go:358` | LOCKDOWN | Available in ModeBonding/ModeConfiguring; NOT in RPC handler map |
| 14 | `security.get_tiers` | `bridgeApi.ts:229` | `lockdown/bonding.go:359` | LOCKDOWN | Available in ModeBonding/ModeConfiguring; NOT in RPC handler map |
| 15 | `security.set_tier` | `bridgeApi.ts:233` | `lockdown/bonding.go:360` | LOCKDOWN | Available in ModeBonding/ModeConfiguring; NOT in RPC handler map |

### Missing Methods (18)

No handler found in any bridge subsystem. These will return "method not found" at runtime.

| # | Method | Source File (Admin) | Handler Location | Status | Recommended Action |
|---|--------|--------------------|-----------------|--------|-------------------|
| 16 | `lockdown.transition` | `bridgeApi.ts:217` | — | MISSING | **Critical.** Admin panel calls this to advance lockdown modes. Not even in lockdown available-methods list. Implement in `pkg/lockdown/` and register in RPC handler map. |
| 17 | `adapter.list` | `bridgeApi.ts:256` | — | MISSING | Admin panel lists all adapters. Note: `bonding.go:361` references `"adapters.list"` (plural) — possible naming mismatch. Implement and register. |
| 18 | `adapter.enable` | `bridgeApi.ts:260` | — | MISSING | Admin enables an adapter. Implement and register. |
| 19 | `adapter.disable` | `bridgeApi.ts:264` | — | MISSING | Admin disables an adapter. Implement and register. |
| 20 | `adapter.configure` | `bridgeApi.ts:268` | — | MISSING | Admin configures adapter settings. Implement and register. Note: `bonding.go:362` references `"adapters.configure"` (plural). |
| 21 | `qr.generate_setup` | `bridgeApi.ts:296` | — | MISSING | Generates setup QR code. Implement and register. |
| 22 | `qr.generate_bonding` | `bridgeApi.ts:300` | — | MISSING | Generates bonding QR code. Implement and register. |
| 23 | `qr.generate_secret` | `bridgeApi.ts:304` | — | MISSING | Generates secret injection QR code. Implement and register. |
| 24 | `qr.generate_invite` | `bridgeApi.ts:308` | — | MISSING | Generates invite QR code. Implement and register. |
| 25 | `admin.initiate_claim` | `bridgeApi.ts:313` | — | MISSING | Initiates admin claim for Element X integration. Implement and register. |
| 26 | `admin.validate_token` | `bridgeApi.ts:317` | — | MISSING | Validates admin claim token. Implement and register. |
| 27 | `admin.respond_challenge` | `bridgeApi.ts:321` | — | MISSING | Responds to admin claim challenge. Implement and register. |
| 28 | `admin.approve_claim` | `bridgeApi.ts:325` | — | MISSING | Approves a pending claim. Implement and register. |
| 29 | `admin.reject_claim` | `bridgeApi.ts:329` | — | MISSING | Rejects a pending claim. Implement and register. |
| 30 | `audit.get_log` | `bridgeApi.ts:339` | — | MISSING | Retrieves audit log with filtering. Audit infrastructure exists (`pkg/audit/`) but no RPC handler. Implement and register. |
| 31 | `secrets.list` | `bridgeApi.ts:344` | — | MISSING | Lists API keys/secrets. Implement and register. |
| 32 | `secrets.revoke` | `bridgeApi.ts:348` | — | MISSING | Revokes an API key. Implement and register. |
| 33 | `secrets.generate_token` | `bridgeApi.ts:355` | — | MISSING | Generates a secret token for a provider. Implement and register. |

---

## Gap Analysis

### Priority 1 — Critical (Blocks Admin Panel Core Flows)

| Gap | Impact | Blocker For |
|-----|--------|-------------|
| `lockdown.transition` | Admin cannot advance through setup stages (lockdown → bonding → configuring → hardening → operational) | First-boot setup flow |
| All 7 LOCKDOWN methods | Referenced by lockdown subsystem but NOT in RPC handler map — server returns "method not found" | Entire lockdown setup UX |
| All 4 `adapter.*` methods | Adapter management page non-functional | Adapter configuration |
| All 4 `qr.*` methods | QR code generation for mobile pairing non-functional | Mobile onboarding |

### Priority 2 — High (Blocks Feature Pages)

| Gap | Impact | Blocker For |
|-----|--------|-------------|
| All 5 `admin.*` methods | Element X admin claim flow non-functional | Admin claim integration |
| `audit.get_log` | Audit log page non-functional | Security auditing |
| All 3 `secrets.*` methods | API key management page non-functional | Secret management |

### Naming Mismatch Alert

The lockdown subsystem (`bonding.go:361-362`) references `"adapters.list"` and `"adapters.configure"` (plural), while the admin panel calls `"adapter.list"` and `"adapter.configure"` (singular). If lockdown is meant to gate these methods, the naming must be reconciled.

---

## Bridge RPC Handler Inventory (Non-Admin Methods)

For reference, these 81 methods are registered in the RPC server but **not** called by the admin panel:

<details>
<summary>Click to expand (81 methods)</summary>

| Method | Handler |
|--------|---------|
| `ai.chat` | `s.handleAIChat` |
| `browser.navigate` | `s.handleBrowserNavigate` |
| `browser.fill` | `s.handleBrowserFill` |
| `browser.click` | `s.handleBrowserClick` |
| `browser.status` | `s.handleBrowserStatus` |
| `browser.wait_for_element` | `s.handleBrowserWaitForElement` |
| `browser.wait_for_captcha` | `s.handleBrowserWaitForCaptcha` |
| `browser.wait_for_2fa` | `s.handleBrowserWaitFor2FA` |
| `browser.complete` | `s.handleBrowserComplete` |
| `browser.fail` | `s.handleBrowserFail` |
| `browser.list` | `s.handleBrowserList` |
| `browser.cancel` | `s.handleBrowserCancel` |
| `bridge.start` | `s.handleBridgeStart` |
| `bridge.stop` | `s.handleBridgeStop` |
| `bridge.status` | `s.handleBridgeStatus` |
| `bridge.channel` | `s.handleBridgeChannel` |
| `bridge.unchannel` | `s.handleUnbridgeChannel` |
| `bridge.list` | `s.handleListBridgedChannels` |
| `bridge.ghost_list` | `s.handleGhostUserList` |
| `bridge.appservice_status` | `s.handleAppServiceStatus` |
| `pii.request` | `s.handlePIIRequest` |
| `pii.approve` | `s.handlePIIApprove` |
| `pii.deny` | `s.handlePIIDeny` |
| `pii.status` | `s.handlePIIStatus` |
| `pii.list_pending` | `s.handlePIIListPending` |
| `pii.stats` | `s.handlePIIStats` |
| `pii.cancel` | `s.handlePIICancel` |
| `pii.fulfill` | `s.handlePIIFulfill` |
| `pii.wait_for_approval` | `s.handlePIIWaitForApproval` |
| `skills.execute` | `s.handleSkillsExecute` |
| `skills.list` | `s.handleSkillsList` |
| `skills.get_schema` | `s.handleSkillsGetSchema` |
| `skills.allow` | `s.handleSkillsAllow` |
| `skills.block` | `s.handleSkillsBlock` |
| `skills.allowlist_add` | `s.handleSkillsAllowlistAdd` |
| `skills.allowlist_remove` | `s.handleSkillsAllowlistRemove` |
| `skills.allowlist_list` | `s.handleSkillsAllowlistList` |
| `skills.web_search` | `s.handleSkillsWebSearch` |
| `skills.web_extract` | `s.handleSkillsWebExtract` |
| `skills.email_send` | `s.handleSkillsEmailSend` |
| `skills.slack_message` | `s.handleSkillsSlackMessage` |
| `skills.file_read` | `s.handleSkillsFileRead` |
| `skills.data_analyze` | `s.handleSkillsDataAnalyze` |
| `matrix.status` | `s.handleMatrixStatus` |
| `matrix.login` | `s.handleMatrixLogin` |
| `matrix.send` | `s.handleMatrixSend` |
| `matrix.receive` | `s.handleMatrixReceive` |
| `matrix.join_room` | `s.handleMatrixJoinRoom` |
| `events.replay` | `s.handleEventsReplay` |
| `events.stream` | `s.handleEventsStream` |
| `studio.deploy` | `s.handleStudio` |
| `studio.stats` | `s.handleStudioStats` |
| `store_key` | `s.handleStoreKey` |
| `provisioning.start` | `s.handleProvisioningStart` |
| `provisioning.claim` | `s.handleProvisioningClaim` |
| `hardening.status` | `s.handleHardeningStatus` |
| `hardening.ack` | `s.handleHardeningAck` |
| `hardening.rotate_password` | `s.handleHardeningRotatePassword` |
| `health.check` | `s.handleHealthCheck` |
| `mobile.heartbeat` | `s.handleMobileHeartbeat` |
| `container.terminate` | `s.handleTerminateContainer` |
| `container.list` | `s.handleListContainers` |
| `resolve_blocker` | `s.handleResolveBlocker` |
| `approve_email` | `s.handleApproveEmail` |
| `deny_email` | `s.handleDenyEmail` |
| `email_approval_status` | `s.handleEmailApprovalStatus` |
| `email.list_pending` | `s.handleEmailListPending` |
| `account.delete` | `s.handleAccountDelete` |
| `secretary.start_workflow` | `s.handleSecretaryMethod` |
| `secretary.get_workflow` | `s.handleSecretaryMethod` |
| `secretary.cancel_workflow` | `s.handleSecretaryMethod` |
| `secretary.advance_workflow` | `s.handleSecretaryMethod` |
| `secretary.list_templates` | `s.handleSecretaryMethod` |
| `secretary.create_template` | `s.handleSecretaryMethod` |
| `secretary.get_template` | `s.handleSecretaryMethod` |
| `secretary.delete_template` | `s.handleSecretaryMethod` |
| `secretary.update_template` | `s.handleSecretaryMethod` |
| `task.create` | `s.handleSecretaryMethod` |
| `task.list` | `s.handleSecretaryMethod` |
| `task.cancel` | `s.handleSecretaryMethod` |
| `task.get` | `s.handleSecretaryMethod` |

</details>

---

## Recommended Next Steps

1. **Wire lockdown methods into RPC handler map** — The lockdown subsystem has logic for 7 methods but none are in `server.go:registerHandlers()`. Add handler registrations that delegate to `pkg/lockdown/`.

2. **Implement `lockdown.transition`** — Not even referenced in the lockdown subsystem. This is the critical mode-advancement method.

3. **Implement `adapter.*` handlers (4 methods)** — No backend exists. Reconcile naming with `bonding.go` which uses `"adapters.*"` (plural).

4. **Implement `qr.*` handlers (4 methods)** — No backend exists. Likely interacts with provisioning and keystore subsystems.

5. **Implement `admin.*` handlers (5 methods)** — No backend exists. Needed for Element X claim flow.

6. **Implement `audit.get_log`** — Audit infrastructure exists in `pkg/audit/` but no RPC handler. Straightforward to add.

7. **Implement `secrets.*` handlers (3 methods)** — No backend exists. Likely interacts with keystore.

8. **Register enforcement handlers** — `pkg/enforcement/rpc_handlers.go` has 6 handler functions but they use a `ServerInterface.RegisterMethod()` pattern that isn't wired into the main server. (Not called by admin panel, but should be registered for completeness.)
