# ArmorChat Screen Index

> **Total Routes:** 48
> **Generated:** 2026-02-24

## Navigation Flow Diagram

```
                                    ┌─────────────────────────────────────────────────────────────┐
                                    │                      APP START                               │
                                    └─────────────────────────────────────────────────────────────┘
                                                                │
                                                                ▼
                                                         ┌──────────┐
                                                         │  SPLASH  │
                                                         │  splash  │
                                                         └────┬─────┘
                                                              │
                                         ┌────────────────────┼────────────────────┐
                                         │                    │                    │
                                         ▼                    ▼                    ▼
                              ┌──────────────────┐   ┌──────────────┐    ┌──────────────┐
                              │ First Time User? │   │  Has Auth?   │    │ Needs Setup? │
                              └────────┬─────────┘   └──────┬───────┘    └──────┬───────┘
                                       │                    │                   │
                                       ▼                    │                   │
                                  ┌──────────┐              │                   │
                                  │ WELCOME  │◄─────────────┘                   │
                                  │ welcome  │                                  │
                                  └────┬─────┘                                  │
                                       │                                        │
                                       ▼                                        │
                            ┌───────────────────────┐                           │
                            │   ONBOARDING FLOW     │                           │
                            ├───────────────────────┤                           │
                            │ • migration           │                           │
                            │ • key_backup_setup    │                           │
                            │ • security/{step}     │                           │
                            │ • connect             │                           │
                            │ • permissions         │                           │
                            │ • completion          │                           │
                            │ • tutorial            │                           │
                            └───────────┬───────────┘                           │
                                        │                                       │
                                        ▼                                       │
                              ┌──────────────────┐                               │
                              │   AUTH FLOW      │                               │
                              ├──────────────────┤                               │
                              │ • login          │                               │
                              │ • registration   │                               │
                              │ • forgot_password│                               │
                              │ • key_recovery   │                               │
                              └────────┬─────────┘                               │
                                       │                                        │
                                       ▼                                        │
                              ┌──────────────────┐                               │
                              │   MAIN APP       │◄──────────────────────────────┘
                              ├──────────────────┤
                              │ • home           │
                              │ • profile        │
                              │ • settings       │
                              └────────┬─────────┘
                                       │
                 ┌─────────────────────┼─────────────────────┐
                 │                     │                     │
                 ▼                     ▼                     ▼
         ┌───────────────┐     ┌───────────────┐     ┌───────────────┐
         │  CHAT FLOW    │     │  SETTINGS     │     │   PROFILE     │
         ├───────────────┤     ├───────────────┤     ├───────────────┤
         │ • chat/{id}   │     │ • settings    │     │ • profile     │
         │ • thread/...  │     │ • security    │     │ • edit_bio    │
         │ • image/{id}  │     │ • appearance  │     │ • change_pwd  │
         │ • file/{id}   │     │ • devices     │     │ • change_phone│
         │ • call/{id}   │     │ • about       │     │ • delete_acct │
         └───────────────┘     └───────────────┘     └───────────────┘
```

---

## Complete Route Inventory

### Category 1: Core (2 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| SPLASH | `splash` | SplashScreen.kt | [splash.md](screens/core/splash.md) |
| WELCOME | `welcome` | WelcomeScreen.kt | [welcome.md](screens/core/welcome.md) |

---

### Category 2: Onboarding (10 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| MIGRATION | `migration` | MigrationScreen.kt | [migration.md](screens/onboarding/migration.md) |
| KEY_BACKUP_SETUP | `key_backup_setup` | KeyBackupSetupScreen.kt | [key-backup-setup.md](screens/onboarding/key-backup-setup.md) |
| SECURITY | `security/{step}` | SecurityExplanationScreen.kt | [security.md](screens/onboarding/security.md) |
| CONNECT | `connect` | ConnectServerScreen.kt | [connect.md](screens/onboarding/connect.md) |
| PERMISSIONS | `permissions` | PermissionsScreen.kt | [permissions.md](screens/onboarding/permissions.md) |
| COMPLETION | `completion` | CompletionScreen.kt | [completion.md](screens/onboarding/completion.md) |
| TUTORIAL | `tutorial` | TutorialScreen.kt | [tutorial.md](screens/onboarding/tutorial.md) |
| ONBOARDING_CONFIG | `onboarding/config` | Onboarding screens | [onboarding-config.md](screens/onboarding/onboarding-config.md) |
| ONBOARDING_SETUP | `onboarding/setup` | SetupModeSelectionScreen.kt | [onboarding-setup.md](screens/onboarding/onboarding-setup.md) |
| ONBOARDING_INVITE | `onboarding/invite` | Onboarding screens | [onboarding-invite.md](screens/onboarding/onboarding-invite.md) |

---

### Category 3: Authentication (4 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| LOGIN | `login` | LoginScreen.kt | [login.md](screens/auth/login.md) |
| REGISTRATION | `registration` | RegistrationScreen.kt | [registration.md](screens/auth/registration.md) |
| FORGOT_PASSWORD | `forgot_password` | ForgotPasswordScreen.kt | [forgot-password.md](screens/auth/forgot-password.md) |
| KEY_RECOVERY | `key_recovery` | KeyRecoveryScreen.kt | [key-recovery.md](screens/auth/key-recovery.md) |

---

### Category 4: Main App (2 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| HOME | `home` | HomeScreenFull.kt | [home.md](screens/main/home.md) |
| PROFILE | `profile` | ProfileScreen.kt | [profile.md](screens/main/profile.md) |

---

### Category 5: Chat & Messaging (3 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| CHAT | `chat/{roomId}` | ChatScreenEnhanced.kt | [chat.md](screens/main/chat.md) |
| THREAD | `thread/{roomId}/{rootMessageId}` | ThreadViewScreen.kt | [thread.md](screens/threads/thread.md) |
| SEARCH | `search` | SearchScreen.kt | [search.md](screens/search/search.md) |

---

### Category 6: Room Management (3 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| ROOM_MANAGEMENT | `room_management` | RoomManagementScreen.kt | [room-management.md](screens/rooms/room-management.md) |
| ROOM_DETAILS | `room_details/{roomId}` | RoomDetailsScreen.kt | [room-details.md](screens/rooms/room-details.md) |
| ROOM_SETTINGS | `room_settings/{roomId}` | RoomSettingsScreen.kt | [room-settings.md](screens/rooms/room-settings.md) |

---

### Category 7: Settings (17 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| SETTINGS | `settings` | SettingsScreen.kt | [settings.md](screens/settings/settings.md) |
| SECURITY_SETTINGS | `security_settings` | SecuritySettingsScreen.kt | [security-settings.md](screens/settings/security-settings.md) |
| NOTIFICATION_SETTINGS | `notification_settings` | NotificationSettingsScreen.kt | [notification-settings.md](screens/settings/notification-settings.md) |
| APPEARANCE | `appearance` | AppearanceSettingsScreen.kt | [appearance.md](screens/settings/appearance.md) |
| PRIVACY_POLICY | `privacy_policy` | PrivacyPolicyScreen.kt | [privacy-policy.md](screens/settings/privacy-policy.md) |
| MY_DATA | `my_data` | MyDataScreen.kt | [my-data.md](screens/settings/my-data.md) |
| DATA_SAFETY | `data_safety` | DataSafetyScreen.kt | [data-safety.md](screens/settings/data-safety.md) |
| ABOUT | `about` | AboutScreen.kt | [about.md](screens/settings/about.md) |
| REPORT_BUG | `report_bug` | ReportBugScreen.kt | [report-bug.md](screens/settings/report-bug.md) |
| INVITE | `invite` | InviteScreen.kt | [invite.md](screens/settings/invite.md) |
| SERVER_CONNECTION | `settings/server_connection` | ServerConnectionScreen.kt | [server-connection.md](screens/settings/server-connection.md) |
| AGENT_MANAGEMENT | `settings/agents` | AgentManagementScreen.kt | [agent-management.md](screens/settings/agent-management.md) |
| HITL_APPROVALS | `settings/approvals` | HitlApprovalScreen.kt | [hitl-approvals.md](screens/settings/hitl-approvals.md) |
| WORKFLOW_MANAGEMENT | `settings/workflows` | WorkflowManagementScreen.kt | [workflow-management.md](screens/settings/workflow-management.md) |
| BUDGET_STATUS | `settings/budget` | BudgetStatusScreen.kt | [budget-status.md](screens/settings/budget-status.md) |
| LICENSES | `licenses` | OpenSourceLicensesScreen.kt | [licenses.md](screens/settings/licenses.md) |
| TERMS_OF_SERVICE | `terms` | TermsOfServiceScreen.kt | [terms-of-service.md](screens/settings/terms-of-service.md) |

---

### Category 8: Profile Editing (4 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| CHANGE_PASSWORD | `change_password` | ChangePasswordScreen.kt | [change-password.md](screens/profile/change-password.md) |
| CHANGE_PHONE | `change_phone` | ChangePhoneNumberScreen.kt | [change-phone.md](screens/profile/change-phone.md) |
| EDIT_BIO | `edit_bio` | EditBioScreen.kt | [edit-bio.md](screens/profile/edit-bio.md) |
| DELETE_ACCOUNT | `delete_account` | DeleteAccountScreen.kt | [delete-account.md](screens/profile/delete-account.md) |

---

### Category 9: Device Management (4 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| DEVICES | `devices` | DeviceListScreen.kt | [devices.md](screens/devices/devices.md) |
| ADD_DEVICE | `add_device` | AddDeviceScreen.kt | [add-device.md](screens/devices/add-device.md) |
| EMOJI_VERIFICATION | `verification/{deviceId}` | EmojiVerificationScreen.kt | [emoji-verification.md](screens/verification/emoji-verification.md) |
| BRIDGE_VERIFICATION | `bridge_verification/{deviceId}` | BridgeVerificationScreen.kt | [bridge-verification.md](screens/verification/bridge-verification.md) |

---

### Category 10: Calls (2 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| ACTIVE_CALL | `call/{callId}` | ActiveCallScreen.kt | [active-call.md](screens/calls/active-call.md) |
| INCOMING_CALL | `incoming_call/{callId}/{callerId}/{callerName}/{callType}` | IncomingCallDialog.kt | [incoming-call.md](screens/calls/incoming-call.md) |

---

### Category 11: Media (2 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| IMAGE_VIEWER | `image/{imageId}` | ImageViewerScreen.kt | [image-viewer.md](screens/media/image-viewer.md) |
| FILE_PREVIEW | `file/{fileId}` | FilePreviewScreen.kt | [file-preview.md](screens/media/file-preview.md) |

---

### Category 12: User Profile (2 routes)

| Route | Pattern | Screen File | Doc |
|-------|---------|-------------|-----|
| USER_PROFILE | `user/{userId}` | UserProfileScreen.kt | [user-profile.md](screens/user-profile/user-profile.md) |
| SHARED_ROOMS | `shared_rooms/{userId}` | SharedRoomsScreen.kt | [shared-rooms.md](screens/user-profile/shared-rooms.md) |

---

## Deep Link Support

The following routes support deep linking:

| Deep Link Pattern | Route |
|-------------------|-------|
| `armorclaw://chat/{roomId}` | CHAT |
| `armorclaw://user/{userId}` | USER_PROFILE |
| `armorclaw://onboarding/config` | ONBOARDING_CONFIG |
| `armorclaw://onboarding/setup` | ONBOARDING_SETUP |
| `armorclaw://onboarding/invite` | ONBOARDING_INVITE |

---

## Route Arguments

Routes with dynamic parameters:

| Route | Arguments |
|-------|-----------|
| `chat/{roomId}` | `roomId: String` |
| `thread/{roomId}/{rootMessageId}` | `roomId: String`, `rootMessageId: String` |
| `room_details/{roomId}` | `roomId: String` |
| `room_settings/{roomId}` | `roomId: String` |
| `verification/{deviceId}` | `deviceId: String` |
| `bridge_verification/{deviceId}` | `deviceId: String` |
| `call/{callId}` | `callId: String` |
| `incoming_call/{callId}/{callerId}/{callerName}/{callType}` | `callId`, `callerId`, `callerName`, `callType` |
| `image/{imageId}` | `imageId: String` |
| `file/{fileId}` | `fileId: String` |
| `user/{userId}` | `userId: String` |
| `shared_rooms/{userId}` | `userId: String` |

---

## Statistics

| Metric | Count |
|--------|-------|
| Total Routes | 48 |
| Static Routes | 35 |
| Dynamic Routes | 13 |
| Deep Link Enabled | 5 |
| Auth Required | 42 |
| Public Access | 6 |

---

*Generated for ArmorChat UX Review*
