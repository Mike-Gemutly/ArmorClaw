# Offline Sync Feature

> Background synchronization for ArmorClaw
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/data/offline/`

## Overview

The offline sync feature ensures ArmorClaw works seamlessly without network connectivity, automatically synchronizing changes when connectivity is restored.

## Feature Components

### BackgroundSyncWorker
**Location:** `offline/BackgroundSyncWorker.kt`

WorkManager-based background synchronization.

#### Functions

| Function | Description |
|----------|-------------|
| `doWork()` | Main sync execution |
| `isNetworkAvailable()` | Network check |
| `createConstraints()` | Work constraints |
| `schedulePeriodicSync()` | Schedule recurring sync |
| `scheduleImmediateSync()` | Immediate sync trigger |
| `cancelSync()` | Cancel pending work |

#### Sync Constraints
- Network type: UNMETERED (WiFi preferred)
- Battery: NOT_LOW
- Charging: NOT_REQUIRED

#### Sync Configuration
```kotlin
companion object {
    const val WORK_NAME = "background_sync_work"

    fun schedulePeriodicSync(
        context: Context,
        intervalMinutes: Long = 15
    ) {
        // Periodic work scheduling
    }
}
```

---

### Sync Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Offline Sync System                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌─────────────┐ │
│  │    Local     │ ←→  │     Sync     │ ←→  │   Remote    │ │
│  │   Database   │     │    Engine    │     │    Server   │ │
│  └──────────────┘     └──────────────┘     └─────────────┘ │
│         ↑                     ↑                              │
│         │                     │                              │
│  ┌──────────────┐     ┌──────────────┐                      │
│  │   Offline    │     │   Conflict   │                      │
│  │    Queue     │     │   Resolver   │                      │
│  └──────────────┘     └──────────────┘                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Network Monitoring

### NetworkMonitor
**Location:** `platform/NetworkMonitorImpl.kt`

Monitors network connectivity changes.

#### Functions

| Function | Description |
|----------|-------------|
| `isOnline()` | Check current connectivity |
| `networkState()` | Flow of network state |
| `getConnectionType()` | WiFi/Cellular/None |

#### Network State
```kotlin
sealed class NetworkState {
    object Available : NetworkState()
    object Unavailable : NetworkState()
    object Lost : NetworkState()
}
```

---

## Pending Operations

### Operation Types

| Operation | Description |
|-----------|-------------|
| SEND_MESSAGE | Outgoing message |
| SEND_REACTION | Emoji reaction |
| MARK_READ | Read receipt |
| UPDATE_ROOM | Room settings |
| DELETE_MESSAGE | Message deletion |

### Operation Queue
Operations are queued locally when offline and processed in order when connectivity is restored.

---

## Conflict Resolution

### Conflict Types

| Type | Resolution Strategy |
|------|---------------------|
| Message Edit | Last-write-wins |
| Reaction | Merge sets |
| Read Status | Take latest |
| Room Settings | Server wins |

### Resolution Flow
1. Detect conflict during sync
2. Apply resolution strategy
3. Update local state
4. Notify user if needed

---

## Data Synchronization

### Sync Process
```
1. Check network availability
2. Get pending operations from queue
3. Send operations to server
4. Process server responses
5. Resolve any conflicts
6. Update local database
7. Mark operations as complete
8. Pull latest changes from server
9. Update UI state
```

### Sync Frequency
- **Foreground**: Immediate (when changes occur)
- **Background**: Every 15 minutes
- **On reconnect**: Immediate

---

## Error Handling

### Error Types

| Error | Action |
|-------|--------|
| Network Error | Retry with exponential backoff |
| Auth Error | Re-authenticate |
| Conflict Error | Apply resolution strategy |
| Server Error | Queue for retry |

### Retry Configuration
- Max retries: 3
- Backoff: Exponential
- Max delay: 1 hour

---

## Message Expiration

### Expiration Manager
Automatically removes expired messages.

#### Expiration Settings
| Type | Duration |
|------|----------|
| None | Never expire |
| 1 Hour | 60 minutes |
| 1 Day | 24 hours |
| 1 Week | 7 days |
| Custom | User defined |

### Cleanup Process
```kotlin
suspend fun cleanupExpiredMessages() {
    val now = Clock.System.now()
    val expired = messageDao.getExpiredMessages(now)
    expired.forEach { message ->
        messageDao.delete(message.id)
        // Also delete from server if needed
    }
}
```

---

## Performance Considerations

### Battery Optimization
- Use WorkManager for background tasks
- Batch operations when possible
- Defer non-critical syncs

### Network Optimization
- Compress data before transmission
- Use delta sync when possible
- Respect metered network settings

### Storage Optimization
- Regular cleanup of old data
- Compress cached attachments
- Limit local message history

---

## Testing Offline Scenarios

### Test Cases

| Scenario | Expected Behavior |
|----------|-------------------|
| Send message offline | Queued, sent on reconnect |
| Receive message offline | Synced on reconnect |
| Edit message offline | Conflict resolved |
| Long offline period | Full sync on reconnect |

---

## Related Documentation

- [Chat](chat.md) - Message functionality
- [Performance](performance.md) - Performance monitoring
- [Notifications](notifications.md) - Push notifications
