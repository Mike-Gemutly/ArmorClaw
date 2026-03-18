# ArmorClaw API Documentation

> Public API documentation for ArmorClaw - Secure E2E Encrypted Chat Application

## 📚 Overview

ArmorClaw is primarily a UI-focused application, but it exposes several public APIs for platform integrations, data access, and extensions.

## 🔌 Platform APIs

### BiometricAuth

**Location:** `shared/platform/BiometricAuth.kt`

**Purpose:** Biometric authentication interface

**Methods:**

```kotlin
interface BiometricAuth {
    /**
     * Check if biometric authentication is available
     */
    suspend fun isAvailable(): Boolean

    /**
     * Authenticate user with biometrics
     * @param title Title for biometric prompt
     * @param subtitle Subtitle for biometric prompt
     * @param description Description for biometric prompt
     * @param negativeText Text for negative button
     * @return Result of authentication (success, failure, error)
     */
    suspend fun authenticate(
        title: String,
        subtitle: String? = null,
        description: String? = null,
        negativeText: String? = null
    ): BiometricResult

    /**
     * Cancel ongoing biometric authentication
     */
    fun cancel()
}
```

**Types:**

```kotlin
sealed class BiometricResult {
    object Success : BiometricResult()
    object Failure : BiometricResult()
    data class Error(val message: String) : BiometricResult()
    object Cancelled : BiometricResult()
}
```

**Usage:**

```kotlin
val biometricAuth = getBiometricAuth()
val result = biometricAuth.authenticate(
    title = "Unlock ArmorClaw",
    subtitle = "Authenticate to access your messages",
    description = "Use your fingerprint or face",
    negativeText = "Cancel"
)

when (result) {
    is BiometricResult.Success -> println("Authentication successful")
    is BiometricResult.Failure -> println("Authentication failed")
    is BiometricResult.Error -> println("Error: ${result.message}")
    is BiometricResult.Cancelled -> println("Authentication cancelled")
}
```

---

### SecureClipboard

**Location:** `shared/platform/SecureClipboard.kt`

**Purpose:** Secure clipboard interface

**Methods:**

```kotlin
interface SecureClipboard {
    /**
     * Copy text to clipboard securely
     * @param text Text to copy
     * @param timeoutMs Timeout in milliseconds (default: 30s)
     */
    suspend fun copy(text: String, timeoutMs: Long = 30000)

    /**
     * Paste text from clipboard securely
     * @return Pasted text or null if error
     */
    suspend fun paste(): String?

    /**
     * Clear clipboard immediately
     */
    suspend fun clear()
}
```

**Usage:**

```kotlin
val secureClipboard = getSecureClipboard()

// Copy securely
secureClipboard.copy("My secret message", timeoutMs = 60000)

// Paste securely
val text = secureClipboard.paste()

// Clear clipboard
secureClipboard.clear()
```

---

### NotificationManager

**Location:** `shared/platform/NotificationManager.kt`

**Purpose:** Notification management interface

**Methods:**

```kotlin
interface NotificationManager {
    /**
     * Show message notification
     * @param notification Notification to show
     */
    suspend fun showMessageNotification(notification: MessageNotification)

    /**
     * Show room notification
     * @param notification Notification to show
     */
    suspend fun showRoomNotification(notification: RoomNotification)

    /**
     * Cancel notification
     * @param id Notification ID
     */
    fun cancelNotification(id: String)

    /**
     * Cancel all notifications
     */
    fun cancelAllNotifications()

    /**
     * Create notification channels
     */
    fun createNotificationChannels()
}
```

**Types:**

```kotlin
data class MessageNotification(
    val id: String,
    val title: String,
    val body: String,
    val senderName: String,
    val senderAvatar: String?,
    val roomId: String,
    val isEncrypted: Boolean
)

data class RoomNotification(
    val id: String,
    val title: String,
    val body: String,
    val roomId: String,
    val roomName: String,
    val roomAvatar: String?,
    val isEncrypted: Boolean
)
```

**Usage:**

```kotlin
val notificationManager = getNotificationManager()

// Show message notification
notificationManager.showMessageNotification(
    MessageNotification(
        id = "msg_123",
        title = "New message from Alice",
        body = "Hey, how are you?",
        senderName = "Alice",
        senderAvatar = null,
        roomId = "!room1:matrix.org",
        isEncrypted = true
    )
)

// Cancel notification
notificationManager.cancelNotification("msg_123")
```

---

### NetworkMonitor

**Location:** `shared/platform/NetworkMonitor.kt`

**Purpose:** Network monitoring interface

**Methods:**

```kotlin
interface NetworkMonitor {
    /**
     * Get current network status
     * @return Network status
     */
    val networkStatus: Flow<NetworkStatus>

    /**
     * Check if network is available
     * @return True if network is available
     */
    fun isAvailable(): Boolean

    /**
     * Check if network is metered
     * @return True if network is metered (cellular)
     */
    fun isMetered(): Boolean
}
```

**Types:**

```kotlin
sealed class NetworkStatus {
    object Available : NetworkStatus()
    object Unavailable : NetworkStatus()
    object Losing : NetworkStatus()
    object Lost : NetworkStatus()
}
```

**Usage:**

```kotlin
val networkMonitor = getNetworkMonitor()

// Watch network status
networkMonitor.networkStatus.collect { status ->
    when (status) {
        is NetworkStatus.Available -> println("Network available")
        is NetworkStatus.Unavailable -> println("Network unavailable")
        is NetworkStatus.Losing -> println("Network losing")
        is NetworkStatus.Lost -> println("Network lost")
    }
}

// Check if network is available
if (networkMonitor.isAvailable()) {
    // Network is available
}

// Check if network is metered
if (networkMonitor.isMetered()) {
    // Network is metered (cellular)
}
```

---

## 🗄️ Database APIs

### MessageDao

**Location:** `androidApp/data/database/MessageDao.kt`

**Purpose:** Message data access object

**Methods:**

```kotlin
@Dao
interface MessageDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(message: MessageEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(messages: List<MessageEntity>)

    @Update
    suspend fun update(message: MessageEntity)

    @Delete
    suspend fun delete(message: MessageEntity)

    @Query("SELECT * FROM messages WHERE id = :id")
    suspend fun getMessageById(id: String): MessageEntity?

    @Query("SELECT * FROM messages WHERE roomId = :roomId ORDER BY timestamp DESC LIMIT :limit OFFSET :offset")
    suspend fun getMessagesByRoom(roomId: String, limit: Int, offset: Int): List<MessageEntity>

    @Query("SELECT * FROM messages WHERE roomId = :roomId ORDER BY timestamp DESC LIMIT :limit OFFSET :offset")
    fun getMessagesByRoomFlow(roomId: String, limit: Int, offset: Int): Flow<List<MessageEntity>>

    @Query("SELECT * FROM messages WHERE status = :status")
    fun getMessagesByStatus(status: String): Flow<List<MessageEntity>>

    @Query("SELECT * FROM messages WHERE localTransactionId = :transactionId")
    suspend fun getMessageByLocalTransactionId(transactionId: String): MessageEntity?

    @Query("SELECT * FROM messages WHERE serverTransactionId = :transactionId")
    suspend fun getMessageByServerTransactionId(transactionId: String): MessageEntity?

    @Query("SELECT * FROM messages WHERE roomId = :roomId AND content LIKE :query ORDER BY timestamp DESC LIMIT 50")
    suspend fun searchMessages(roomId: String, query: String): List<MessageEntity>

    @Query("UPDATE messages SET status = :status WHERE id = :id")
    suspend fun updateMessageStatus(id: String, status: String)

    @Query("UPDATE messages SET isExpired = 1 WHERE expirationTimestamp <= :currentTimestamp")
    suspend fun markExpiredMessages(currentTimestamp: Long)

    @Query("DELETE FROM messages WHERE isExpired = 1")
    suspend fun deleteExpiredMessages()

    @Query("DELETE FROM messages WHERE roomId = :roomId")
    suspend fun deleteMessagesByRoom(roomId: String)
}
```

**Usage:**

```kotlin
val messageDao = database.messageDao()

// Insert message
messageDao.insert(messageEntity)

// Get messages by room (Flow)
val messagesFlow = messageDao.getMessagesByRoomFlow(roomId, limit = 50, offset = 0)
messagesFlow.collect { messages ->
    // Handle messages
}

// Search messages
val results = messageDao.searchMessages(roomId, "Hello")

// Update message status
messageDao.updateMessageStatus(messageId, "read")
```

---

### RoomDao

**Location:** `androidApp/data/database/RoomDao.kt`

**Purpose:** Room data access object

**Methods:**

```kotlin
@Dao
interface RoomDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(room: RoomEntity)

    @Update
    suspend fun update(room: RoomEntity)

    @Delete
    suspend fun delete(room: RoomEntity)

    @Query("SELECT * FROM rooms WHERE id = :id")
    suspend fun getRoomById(id: String): RoomEntity?

    @Query("SELECT * FROM rooms WHERE isJoined = 1 ORDER BY lastMessageTimestamp DESC")
    fun getActiveRooms(): Flow<List<RoomEntity>>

    @Query("SELECT * FROM rooms WHERE isArchived = 1 ORDER BY lastMessageTimestamp DESC")
    fun getArchivedRooms(): Flow<List<RoomEntity>>

    @Query("SELECT * FROM rooms WHERE isFavorited = 1 ORDER BY lastMessageTimestamp DESC")
    fun getFavoritedRooms(): Flow<List<RoomEntity>>

    @Query("UPDATE rooms SET unreadCount = unreadCount + :delta WHERE id = :id")
    suspend fun updateUnreadCount(id: String, delta: Int)

    @Query("UPDATE rooms SET mentionCount = mentionCount + :delta WHERE id = :id")
    suspend fun updateMentionCount(id: String, delta: Int)

    @Query("UPDATE rooms SET lastMessageId = :messageId, lastMessageTimestamp = :timestamp WHERE id = :id")
    suspend fun updateLastMessage(id: String, messageId: String, timestamp: Long)
}
```

**Usage:**

```kotlin
val roomDao = database.roomDao()

// Insert room
roomDao.insert(roomEntity)

// Get active rooms (Flow)
val roomsFlow = roomDao.getActiveRooms()
roomsFlow.collect { rooms ->
    // Handle rooms
}

// Update unread count
roomDao.updateUnreadCount(roomId, delta = 1)
```

---

### SyncQueueDao

**Location:** `androidApp/data/database/SyncQueueDao.kt`

**Purpose:** Sync queue data access object

**Methods:**

```kotlin
@Dao
interface SyncQueueDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun enqueue(operation: SyncQueueEntity)

    @Query("SELECT * FROM sync_queue WHERE status = 'pending' ORDER BY priority DESC, createdAt ASC LIMIT :limit")
    fun getPendingOperations(limit: Int = 100): Flow<List<SyncQueueEntity>>

    @Query("SELECT * FROM sync_queue WHERE roomId = :roomId AND status = 'pending' ORDER BY priority DESC, createdAt ASC")
    fun getPendingOperationsByRoom(roomId: String): Flow<List<SyncQueueEntity>>

    @Query("SELECT * FROM sync_queue WHERE status = :status")
    fun getOperationsByStatus(status: String): Flow<List<SyncQueueEntity>>

    @Update
    suspend fun updateOperation(operation: SyncQueueEntity)

    @Query("UPDATE sync_queue SET status = 'processing' WHERE id = :id")
    suspend fun markAsProcessing(id: String)

    @Query("UPDATE sync_queue SET status = 'completed', completedAt = :timestamp WHERE id = :id")
    suspend fun markAsCompleted(id: String, timestamp: Long)

    @Query("UPDATE sync_queue SET status = 'failed', errorMessage = :message, errorCode = :code, retryCount = retryCount + 1, lastRetryAt = :lastRetryAt, nextRetryAt = :nextRetryAt WHERE id = :id")
    suspend fun markAsFailed(id: String, message: String, code: Int, lastRetryAt: Long, nextRetryAt: Long)

    @Query("SELECT COUNT(*) FROM sync_queue WHERE status = 'pending'")
    suspend fun getPendingCount(): Int

    @Query("SELECT COUNT(*) FROM sync_queue WHERE status = 'failed'")
    suspend fun getFailedCount(): Int

    @Query("DELETE FROM sync_queue WHERE roomId = :roomId")
    suspend fun deleteOperationsByRoom(roomId: String)

    @Query("DELETE FROM sync_queue WHERE status = 'completed' AND completedAt < :timestamp")
    suspend fun deleteCompletedOperations(timestamp: Long)
}
```

**Usage:**

```kotlin
val syncQueueDao = database.syncQueueDao()

// Enqueue operation
syncQueueDao.enqueue(operationEntity)

// Get pending operations (Flow)
val operationsFlow = syncQueueDao.getPendingOperations()
operationsFlow.collect { operations ->
    // Handle operations
}

// Mark as processing
syncQueueDao.markAsProcessing(operationId)

// Mark as completed
syncQueueDao.markAsCompleted(operationId, System.currentTimeMillis())
```

---

## 🔄 Offline Sync APIs

### OfflineQueue

**Location:** `androidApp/data/offline/OfflineQueue.kt`

**Purpose:** Offline queue for pending operations

**Methods:**

```kotlin
class OfflineQueue(
    private val syncQueueDao: SyncQueueDao
) {
    /**
     * Enqueue send message operation
     * @param roomId Room ID
     * @param message Message content
     * @param priority Operation priority
     */
    suspend fun enqueueSendMessage(
        roomId: String,
        message: String,
        priority: OperationPriority = OperationPriority.MEDIUM
    ): String

    /**
     * Enqueue update message operation
     * @param messageId Message ID
     * @param content New content
     * @param priority Operation priority
     */
    suspend fun enqueueUpdateMessage(
        messageId: String,
        content: String,
        priority: OperationPriority = OperationPriority.LOW
    ): String

    /**
     * Enqueue delete message operation
     * @param messageId Message ID
     * @param priority Operation priority
     */
    suspend fun enqueueDeleteMessage(
        messageId: String,
        priority: OperationPriority = OperationPriority.MEDIUM
    ): String

    /**
     * Enqueue reaction add operation
     * @param messageId Message ID
     * @param emoji Emoji reaction
     * @param priority Operation priority
     */
    suspend fun enqueueReactionAdd(
        messageId: String,
        emoji: String,
        priority: OperationPriority = OperationPriority.MEDIUM
    ): String

    /**
     * Enqueue reaction remove operation
     * @param messageId Message ID
     * @param emoji Emoji reaction
     * @param priority Operation priority
     */
    suspend fun enqueueReactionRemove(
        messageId: String,
        emoji: String,
        priority: OperationPriority = OperationPriority.MEDIUM
    ): String

    /**
     * Enqueue mark read operation
     * @param roomId Room ID
     * @param priority Operation priority
     */
    suspend fun enqueueMarkRead(
        roomId: String,
        priority: OperationPriority = OperationPriority.HIGH
    ): String

    /**
     * Get pending operations (Flow)
     * @return Flow of pending operations
     */
    fun getPendingOperations(): Flow<List<SyncQueueEntity>>

    /**
     * Get pending operations for room (Flow)
     * @param roomId Room ID
     * @return Flow of pending operations for room
     */
    fun getPendingOperationsForRoom(roomId: String): Flow<List<SyncQueueEntity>>

    /**
     * Mark operation as processing
     * @param operationId Operation ID
     */
    suspend fun markAsProcessing(operationId: String)

    /**
     * Mark operation as completed
     * @param operationId Operation ID
     */
    suspend fun markAsCompleted(operationId: String)

    /**
     * Mark operation as failed
     * @param operationId Operation ID
     * @param message Error message
     * @param code Error code
     */
    suspend fun markAsFailed(operationId: String, message: String, code: Int)

    /**
     * Get pending operation count
     * @return Pending operation count
     */
    suspend fun getPendingCount(): Int

    /**
     * Get failed operation count
     * @return Failed operation count
     */
    suspend fun getFailedCount(): Int

    /**
     * Delete completed operations
     * @param timestamp Timestamp threshold
     */
    suspend fun deleteCompletedOperations(timestamp: Long)

    /**
     * Clear room operations
     * @param roomId Room ID
     */
    suspend fun clearRoomOperations(roomId: String)
}
```

**Usage:**

```kotlin
val offlineQueue = OfflineQueue(database.syncQueueDao)

// Enqueue send message
val operationId = offlineQueue.enqueueSendMessage(roomId, "Hello")

// Get pending operations (Flow)
offlineQueue.getPendingOperations().collect { operations ->
    // Handle operations
}

// Mark as completed
offlineQueue.markAsCompleted(operationId)
```

---

## 📊 Performance APIs

### PerformanceProfiler

**Location:** `androidApp/performance/PerformanceProfiler.kt`

**Purpose:** Performance profiling

**Methods:**

```kotlin
class PerformanceProfiler(
    private val enabled: Boolean = BuildConfig.DEBUG
) {
    /**
     * Start a trace section
     * @param name Trace name
     */
    fun beginTrace(name: String)

    /**
     * End a trace section
     */
    fun endTrace()

    /**
     * Execute a block with tracing
     * @param name Trace name
     * @param block Block to execute
     * @return Block result
     */
    suspend fun <T> trace(name: String, block: suspend () -> T): T

    /**
     * Execute a block with memory allocation tracking
     * @param name Trace name
     * @param block Block to execute
     * @return Allocation result with block result
     */
    suspend fun <T> trackAllocations(name: String, block: suspend () -> T): AllocationResult<T>

    /**
     * Dump heap to file
     * @param outputFile Output file
     * @return True if successful
     */
    fun dumpHeap(outputFile: File): Boolean

    /**
     * Enable strict mode (development only)
     */
    fun enableStrictMode()

    /**
     * Disable strict mode
     */
    fun disableStrictMode()

    /**
     * Get current method count
     * @return Method count
     */
    fun getMethodCount(): Int

    /**
     * Reset method counting
     */
    fun resetMethodCounting()
}
```

**Usage:**

```kotlin
val profiler = PerformanceProfiler(enabled = true)

// Trace block
profiler.trace("sendMessage") {
    sendMessage(message)
}

// Track allocations
val result = profiler.trackAllocations("sendMessage") {
    sendMessage(message)
}

// Dump heap
profiler.dumpHeap(File("heap.hprof"))
```

---

*For detailed implementation, see source code.*
