# OfflineQueue

> Offline operation queue
> Location: `shared/src/commonMain/kotlin/com/armorclaw/shared/data/OfflineQueue.kt`

## Overview

OfflineQueue manages operations that need to be performed when the device comes back online, ensuring message delivery and data consistency.

## Class Definition

```kotlin
class OfflineQueue(
    private val database: ArmorClawDatabase,
    private val networkMonitor: NetworkMonitor
)
```

---

## Functions

### enqueue
```kotlin
suspend fun enqueue(operation: QueuedOperation): String
```

**Description:** Adds an operation to the offline queue.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `operation` | `QueuedOperation` | Operation to queue |

**Returns:** Unique queue item ID.

---

### processQueue
```kotlin
suspend fun processQueue(): QueueProcessingResult
```

**Description:** Processes all pending operations in order.

**Returns:** Summary of processed operations.

---

### cancel
```kotlin
suspend fun cancel(queueItemId: String)
```

**Description:** Cancels a pending operation.

---

### getPendingCount
```kotlin
fun getPendingCount(): Int
```

**Description:** Returns count of pending operations.

---

## Data Models

### QueuedOperation
```kotlin
sealed class QueuedOperation {
    abstract val id: String
    abstract val timestamp: Instant
    abstract val priority: Priority

    data class SendMessage(
        override val id: String,
        override val timestamp: Instant,
        override val priority: Priority = Priority.NORMAL,
        val roomId: String,
        val content: MessageContent,
        val replyTo: String?
    ) : QueuedOperation()

    data class UpdateMessage(
        override val id: String,
        override val timestamp: Instant,
        override val priority: Priority = Priority.NORMAL,
        val messageId: String,
        val newContent: String
    ) : QueuedOperation()

    data class DeleteMessage(
        override val id: String,
        override val timestamp: Instant,
        override val priority: Priority = Priority.NORMAL,
        val messageId: String
    ) : QueuedOperation()

    data class MarkAsRead(
        override val id: String,
        override val timestamp: Instant,
        override val priority: Priority = Priority.LOW,
        val roomId: String,
        val messageId: String
    ) : QueuedOperation()

    data class UpdateStatus(
        override val id: String,
        override val timestamp: Instant,
        override val priority: Priority = Priority.HIGH,
        val status: UserStatus
    ) : QueuedOperation()
}

enum class Priority {
    LOW,
    NORMAL,
    HIGH,
    CRITICAL
}
```

### QueueItem
```kotlin
data class QueueItem(
    val id: String,
    val operation: QueuedOperation,
    val status: QueueStatus,
    val attempts: Int,
    val maxAttempts: Int,
    val lastError: String?,
    val createdAt: Instant,
    val updatedAt: Instant
)

enum class QueueStatus {
    PENDING,
    PROCESSING,
    COMPLETED,
    FAILED,
    CANCELLED
}
```

---

## Implementation

### Enqueue Operation
```kotlin
suspend fun enqueue(operation: QueuedOperation): String {
    val queueItem = QueueItem(
        id = UUID.randomUUID().toString(),
        operation = operation,
        status = QueueStatus.PENDING,
        attempts = 0,
        maxAttempts = getMaxAttempts(operation),
        lastError = null,
        createdAt = Clock.System.now(),
        updatedAt = Clock.System.now()
    )

    database.queueItems.insert(queueItem)

    // Try to process immediately if online
    if (networkMonitor.isOnline) {
        processNextItem()
    }

    return queueItem.id
}
```

### Process Queue
```kotlin
suspend fun processQueue(): QueueProcessingResult {
    if (!networkMonitor.isOnline) {
        return QueueProcessingResult(
            processed = 0,
            failed = 0,
            reason = "Offline"
        )
    }

    val pendingItems = database.queueItems
        .getByStatus(QueueStatus.PENDING)
        .sortedByDescending { it.operation.priority }

    var processed = 0
    var failed = 0

    for (item in pendingItems) {
        try {
            processItem(item)
            database.queueItems.updateStatus(item.id, QueueStatus.COMPLETED)
            processed++
        } catch (e: Exception) {
            handleItemFailure(item, e)
            if (item.status == QueueStatus.FAILED) {
                failed++
            }
        }
    }

    return QueueProcessingResult(
        processed = processed,
        failed = failed,
        reason = null
    )
}
```

### Process Single Item
```kotlin
private suspend fun processItem(item: QueueItem) {
    database.queueItems.updateStatus(item.id, QueueStatus.PROCESSING)

    when (val operation = item.operation) {
        is QueuedOperation.SendMessage -> {
            messageRepository.send(
                roomId = operation.roomId,
                content = operation.content,
                replyTo = operation.replyTo
            )
        }
        is QueuedOperation.UpdateMessage -> {
            messageRepository.update(
                messageId = operation.messageId,
                content = operation.newContent
            )
        }
        is QueuedOperation.DeleteMessage -> {
            messageRepository.delete(operation.messageId)
        }
        is QueuedOperation.MarkAsRead -> {
            roomRepository.markAsRead(
                roomId = operation.roomId,
                messageId = operation.messageId
            )
        }
        is QueuedOperation.UpdateStatus -> {
            userRepository.updateStatus(operation.status)
        }
    }
}
```

---

## Retry Logic

### Exponential Backoff
```kotlin
private suspend fun handleItemFailure(item: QueueItem, error: Exception) {
    val newAttempts = item.attempts + 1

    if (newAttempts >= item.maxAttempts) {
        database.queueItems.update(
            item.copy(
                status = QueueStatus.FAILED,
                attempts = newAttempts,
                lastError = error.message
            )
        )
    } else {
        database.queueItems.update(
            item.copy(
                status = QueueStatus.PENDING,
                attempts = newAttempts,
                lastError = error.message
            )
        )

        // Schedule retry with backoff
        delay(calculateBackoff(newAttempts))
    }
}

private fun calculateBackoff(attempt: Int): Long {
    return minOf(
        1000L * (2.0.pow(attempt).toLong()),
        60_000L // Max 1 minute
    )
}
```

### Max Attempts by Priority
| Priority | Max Attempts |
|----------|--------------|
| CRITICAL | 10 |
| HIGH | 5 |
| NORMAL | 3 |
| LOW | 1 |

---

## Usage Example

### Send Message (with offline support)
```kotlin
class MessageRepository(
    private val offlineQueue: OfflineQueue,
    private val api: MessageApi
) {
    suspend fun sendMessage(
        roomId: String,
        content: MessageContent,
        replyTo: String? = null
    ): Message {
        val message = Message(
            id = generateId(),
            roomId = roomId,
            content = content,
            status = MessageStatus.SENDING,
            ...
        )

        // Save locally first
        localDataSource.saveMessage(message)

        // Try to send or queue
        try {
            if (networkMonitor.isOnline) {
                api.sendMessage(message)
                localDataSource.updateStatus(message.id, MessageStatus.SENT)
            } else {
                offlineQueue.enqueue(QueuedOperation.SendMessage(
                    id = message.id,
                    timestamp = Clock.System.now(),
                    roomId = roomId,
                    content = content,
                    replyTo = replyTo
                ))
            }
        } catch (e: Exception) {
            offlineQueue.enqueue(QueuedOperation.SendMessage(...))
        }

        return message
    }
}
```

---

## State Flow

### Queue State
```kotlin
val queueState: StateFlow<QueueState> = database.queueItems
    .observeAll()
    .map { items ->
        QueueState(
            pendingCount = items.count { it.status == QueueStatus.PENDING },
            processingCount = items.count { it.status == QueueStatus.PROCESSING },
            failedCount = items.count { it.status == QueueStatus.FAILED },
            items = items
        )
    }
    .stateIn(viewModelScope, SharingStarted.Lazily, QueueState.EMPTY)
```

---

## Related Documentation

- [Offline Sync](../features/offline-sync.md) - Feature overview
- [BackgroundSyncWorker](BackgroundSyncWorker.md) - Background sync
- [SyncEngine](SyncEngine.md) - Sync engine
