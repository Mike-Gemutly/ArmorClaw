# BackgroundSyncWorker

> WorkManager background sync service
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/data/BackgroundSyncWorker.kt`

## Overview

BackgroundSyncWorker handles periodic message synchronization using Android's WorkManager API to keep the app data fresh even when not actively in use.

## Class Definition

```kotlin
class BackgroundSyncWorker(
    context: Context,
    workerParams: WorkerParameters
) : CoroutineWorker(context, workerParams)
```

---

## Configuration

### Work Request Setup
```kotlin
fun scheduleBackgroundSync(context: Context) {
    val syncWorkRequest = PeriodicWorkRequestBuilder<BackgroundSyncWorker>(
        repeatInterval = 15,
        repeatIntervalTimeUnit = TimeUnit.MINUTES
    )
        .setConstraints(
            Constraints.Builder()
                .setRequiredNetworkType(NetworkType.CONNECTED)
                .setRequiresBatteryNotLow(true)
                .build()
        )
        .setBackoffCriteria(
            BackoffPolicy.LINEAR,
            WorkRequest.MIN_BACKOFF_MILLIS,
            TimeUnit.MILLISECONDS
        )
        .build()

    WorkManager.getInstance(context).enqueueUniquePeriodicWork(
        "background_sync",
        ExistingPeriodicWorkPolicy.KEEP,
        syncWorkRequest
    )
}
```

### Constraints
| Constraint | Value | Description |
|------------|-------|-------------|
| Network | CONNECTED | Requires internet |
| Battery | Not Low | Skip if low battery |
| Charging | Optional | Can run while charging |
| Idle | Optional | Can run in idle mode |

---

## Implementation

```kotlin
class BackgroundSyncWorker(
    context: Context,
    workerParams: WorkerParameters
) : CoroutineWorker(context, workerParams) {

    override suspend fun doWork(): Result {
        return try {
            // Check if user is authenticated
            if (!isUserAuthenticated()) {
                return Result.success()
            }

            // Get sync parameters
            val syncSince = inputData.getLong("last_sync_time", 0L)

            // Perform sync
            val syncResult = performSync(syncSince)

            // Update last sync time
            setLastSyncTime(System.currentTimeMillis())

            // Output result
            val outputData = workDataOf(
                "messages_synced" to syncResult.messagesSynced,
                "rooms_updated" to syncResult.roomsUpdated
            )

            Result.success(outputData)
        } catch (e: Exception) {
            Log.e("BackgroundSyncWorker", "Sync failed", e)

            if (shouldRetry(e)) {
                Result.retry()
            } else {
                Result.failure()
            }
        }
    }

    private suspend fun performSync(lastSyncTime: Long): SyncResult {
        // Sync messages
        val newMessages = messageRepository.fetchNewMessages(lastSyncTime)

        // Sync rooms
        val updatedRooms = roomRepository.fetchUpdatedRooms(lastSyncTime)

        // Sync user data
        userRepository.refreshUserData()

        // Show notification for new messages
        if (newMessages.isNotEmpty()) {
            notificationManager.showSyncNotification(newMessages)
        }

        return SyncResult(
            messagesSynced = newMessages.size,
            roomsUpdated = updatedRooms.size
        )
    }
}
```

---

## Sync Strategy

### Data Types
| Data | Sync Strategy |
|------|---------------|
| Messages | Pull new since last sync |
| Rooms | Pull updated metadata |
| Users | Pull profile changes |
| Read States | Push local, pull remote |

### Conflict Resolution
1. Server wins for message ordering
2. Latest timestamp wins for edits
3. Merge for read states

---

## Foreground Service (Android 12+)

### For Long-Running Sync
```kotlin
override suspend fun getForegroundInfo(): ForegroundInfo {
    val notification = NotificationCompat.Builder(applicationContext, CHANNEL_ID)
        .setContentTitle("Syncing messages")
        .setSmallIcon(R.drawable.ic_sync)
        .setPriority(NotificationCompat.PRIORITY_LOW)
        .build()

    return ForegroundInfo(
        NOTIFICATION_ID,
        notification,
        ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC
    )
}
```

---

## Scheduling

### Periodic Schedule
```kotlin
// Schedule periodic sync
fun schedulePeriodicSync(context: Context) {
    val request = PeriodicWorkRequestBuilder<BackgroundSyncWorker>(
        repeatInterval = 15,
        repeatIntervalTimeUnit = TimeUnit.MINUTES
    ).build()

    WorkManager.getInstance(context)
        .enqueueUniquePeriodicWork(
            SYNC_WORK_NAME,
            ExistingPeriodicWorkPolicy.KEEP,
            request
        )
}
```

### One-Time Sync
```kotlin
// Immediate sync
fun syncNow(context: Context) {
    val request = OneTimeWorkRequestBuilder<BackgroundSyncWorker>()
        .setExpedited(OutOfQuotaPolicy.RUN_AS_NON_EXPEDITED_WORK_REQUEST)
        .build()

    WorkManager.getInstance(context)
        .enqueueUniqueWork(
            SYNC_NOW_WORK_NAME,
            ExistingWorkPolicy.REPLACE,
            request
        )
}
```

---

## Status Monitoring

### LiveData Observation
```kotlin
WorkManager.getInstance(context)
    .getWorkInfosForUniqueWorkLiveData(SYNC_WORK_NAME)
    .observe(lifecycleOwner) { workInfos ->
        workInfos.firstOrNull()?.let { workInfo ->
            when (workInfo.state) {
                WorkInfo.State.RUNNING -> {
                    // Show sync indicator
                }
                WorkInfo.State.SUCCEEDED -> {
                    val messagesSynced = workInfo.outputData.getInt("messages_synced", 0)
                    // Update UI
                }
                WorkInfo.State.FAILED -> {
                    // Handle failure
                }
                else -> {}
            }
        }
    }
```

---

## Battery Optimization

### Best Practices
- Use `setRequiresBatteryNotLow(true)`
- Batch sync operations
- Use efficient network calls
- Respect doze mode

### Exemptions
```kotlin
// Request battery optimization exemption for critical sync
fun requestBatteryExemption(context: Context) {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
        val intent = Intent().apply {
            action = Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS
            data = Uri.parse("package:${context.packageName}")
        }
        context.startActivity(intent)
    }
}
```

---

## Testing

### Unit Test
```kotlin
@Test
fun `doWork returns success on successful sync`() = runTest {
    // Given
    val context = ApplicationProvider.getApplicationContext<Context>()
    val worker = TestListenableWorkerBuilder<BackgroundSyncWorker>(context)
        .build()

    // When
    val result = worker.doWork()

    // Then
    assertTrue(result is ListenableWorker.Result.Success)
}
```

### Integration Test
```kotlin
@Test
fun `sync work enqueues correctly`() {
    val context = ApplicationProvider.getApplicationContext<Context>()
    val workManager = WorkManager.getInstance(context)

    schedulePeriodicSync(context)

    val workInfo = workManager.getWorkInfosForUniqueWork(SYNC_WORK_NAME)
        .get()

    assertEquals(1, workInfo.size)
}
```

---

## Related Documentation

- [Offline Sync](../features/offline-sync.md) - Feature overview
- [SyncEngine](SyncEngine.md) - Sync engine
- [OfflineQueue](OfflineQueue.md) - Offline queue
