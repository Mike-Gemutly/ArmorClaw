# ArmorClaw Mobile App - Offline/Sync Strategy Specification

> **Document Purpose:** Complete offline/sync architecture with conflict resolution
> **Date Created:** 2026-02-10
> **Phase:** 1 (Foundation)
> **Priority:** HIGH - Critical for user experience in poor network conditions

---

## 1. Architecture Overview

### 1.1 Design Philosophy

**Core Principles:**
1. **Offline-First:** App should feel responsive even without network
2. **Optimistic UI:** Show sent messages immediately, sync in background
3. **Graceful Degradation:** Feature downgrade rather than complete failure
4. **Conflict Resolution:** Last-write-wins with timestamp ordering
5. **Transparent Sync:** Users always know sync status

### 1.2 System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Mobile App                                  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    UI Layer                                  │   │
│  │  - Message list                                              │   │
│  │  - Send button                                               │   │
│  │  - Sync indicator                                            │   │
│  └────────────────────┬─────────────────────────────────────────┘   │
│                       │                                            │
│  ┌────────────────────▼─────────────────────────────────────────┐   │
│  │              Offline Sync Manager                             │   │
│  │  ┌──────────────────────────────────────────────────────┐    │   │
│  │  │  Message Queue (Pending sends)                        │    │   │
│  │  │  - Local message ID                                   │    │   │
│  │  │  - Matrix event ID (assigned later)                    │    │   │
│  │  │  - Room ID                                             │    │   │
│  │  │  - Content                                             │    │   │
│  │  │  - Timestamp (client-generated)                        │    │   │
│  │  │  - Retry count                                         │    │   │
│  │  └──────────────────────────────────────────────────────┘    │   │
│  │  ┌──────────────────────────────────────────────────────┐    │   │
│  │  │  Conflict Resolver                                    │    │   │
│  │  │  - Compare timestamps                                 │    │   │
│  │  │  - Apply resolution strategy                          │    │   │
│  │  │  - Notify user of conflicts                           │    │   │
│  │  └──────────────────────────────────────────────────────┘    │   │
│  │  ┌──────────────────────────────────────────────────────┐    │   │
│  │  │  Sync State Machine                                   │    │   │
│  │  │  - IDLE → SYNCING → COMPLETE                          │    │   │
│  │  │  - Error handling with backoff                        │    │   │
│  │  └──────────────────────────────────────────────────────┘    │   │
│  └────────────────────┬─────────────────────────────────────────┘   │
│                       │                                            │
│  ┌────────────────────▼─────────────────────────────────────────┐   │
│  │            Local Storage (SQLite/Room)                       │   │
│  │  ┌──────────────────────────────────────────────────────┐    │   │
│  │  │  Message Table                                        │    │   │
│  │  │  - local_id (PRIMARY KEY)                             │    │   │
│  │  │  - event_id (nullable, indexed)                       │    │   │
│  │  │  - room_id (indexed)                                  │    │   │
│  │  │  - sender_id                                          │    │   │
│  │  │  - content                                            │    │   │
│  │  │  - timestamp                                          │    │   │
│  │  │  - sync_status (PENDING, SYNCED, CONFLICT)            │    │   │
│  │  │  - is_outgoing                                        │    │   │
│  │  └──────────────────────────────────────────────────────┘    │   │
│  │  ┌──────────────────────────────────────────────────────┐    │   │
│  │  │  Sync Metadata Table                                  │    │   │
│  │  │  - room_id (PRIMARY KEY)                              │    │   │
│  │  │  - last_sync_token (batch token)                      │    │   │
│  │  │  - last_sync_time                                     │    │   │
│  │  │  - pending_count                                      │    │   │
│  │  └──────────────────────────────────────────────────────┘    │   │
│  └────────────────────┬─────────────────────────────────────────┘   │
│                                                                      │
└───────────────────────┼──────────────────────────────────────────────┘
                        │
                        │ Matrix Sync (when online)
                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Matrix Homeserver                                │
│  - /sync endpoint with since token                                  │
│  - /send endpoint for queued messages                               │
│  - E2EE message decryption                                          │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Data Models

### 2.1 Local Message Entity

```kotlin
@Entity(tableName = "messages")
data class LocalMessage(
    @PrimaryKey(autoGenerate = true)
    val localId: Long = 0,

    // Matrix event ID (null until synced)
    @ColumnInfo(name = "event_id", index = true)
    val eventId: String? = null,

    // Room identifier
    @ColumnInfo(name = "room_id", index = true)
    val roomId: String,

    // Sender's Matrix ID
    @ColumnInfo(name = "sender_id")
    val senderId: String,

    // Message content (encrypted before storage if E2EE)
    @ColumnInfo(name = "content")
    val content: MessageContent,

    // Client-generated timestamp (UTC)
    @ColumnInfo(name = "timestamp")
    val timestamp: Instant,

    // Sync status
    @ColumnInfo(name = "sync_status")
    val syncStatus: SyncStatus,

    // Direction (outgoing vs incoming)
    @ColumnInfo(name = "is_outgoing")
    val isOutgoing: Boolean,

    // For outgoing messages: retry count
    @ColumnInfo(name = "retry_count")
    val retryCount: Int = 0,

    // For conflict resolution: server timestamp (if available)
    @ColumnInfo(name = "server_timestamp")
    val serverTimestamp: Instant? = null,

    // Edit history (for conflict resolution)
    @ColumnInfo(name = "edit_count")
    val editCount: Int = 0,

    // Deletion flag
    @ColumnInfo(name = "is_deleted")
    val isDeleted: Boolean = false
)

sealed class SyncStatus {
    object Pending : SyncStatus()
    object Syncing : SyncStatus()
    object Synced : SyncStatus()
    data class Failed(val error: String, val retryAfter: Instant?) : SyncStatus()
    data class Conflict(val resolution: ConflictResolution) : SyncStatus()
}

data class MessageContent(
    val type: MessageType,
    val body: String,
    val formattedBody: String? = null,
    val attachments: List<Attachment> = emptyList(),
    val mentions: List<Mention> = emptyList(),
    val replyTo: String? = null  // Event ID being replied to
)

enum class MessageType {
    TEXT,
    IMAGE,
    FILE,
    AUDIO,
    VIDEO,
    NOTICE,
    EMOTE
}

data class Attachment(
    val url: String,
    val mimeType: String,
    val size: Long,
    val thumbnailUrl: String? = null,
    val fileName: String
)

data class Mention(
    val userId: String,
    val displayName: String
)
```

### 2.2 Sync Metadata Entity

```kotlin
@Entity(tableName = "sync_metadata")
data class SyncMetadata(
    @PrimaryKey
    val roomId: String,

    // Matrix sync token for this room
    @ColumnInfo(name = "last_sync_token")
    val lastSyncToken: String? = null,

    // Last successful sync time
    @ColumnInfo(name = "last_sync_time")
    val lastSyncTime: Instant? = null,

    // Count of pending messages
    @ColumnInfo(name = "pending_count")
    val pendingCount: Int = 0,

    // Sync state
    @ColumnInfo(name = "sync_state")
    val syncState: RoomSyncState = RoomSyncState.IDLE,

    // Known members cache
    @ColumnInfo(name = "members", typeAffinity = TEXT)
    val members: List<RoomMember> = emptyList(),

    // Room name (cached)
    @ColumnInfo(name = "room_name")
    val roomName: String? = null,

    // Room avatar (cached)
    @ColumnInfo(name = "room_avatar")
    val roomAvatar: String? = null
)

sealed class RoomSyncState {
    object IDLE : RoomSyncState()
    object SYNCING : RoomSyncState()
    data class ERROR(val message: String) : RoomSyncState()
}

data class RoomMember(
    val userId: String,
    val displayName: String,
    val avatarUrl: String? = null,
    val membership: Membership  // JOIN, INVITE, LEAVE, BAN
)

enum class Membership {
    JOIN, INVITE, LEAVE, BAN, KNOCK
}
```

### 2.3 Queued Message Entity

```kotlin
@Entity(tableName = "queued_messages")
data class QueuedMessage(
    @PrimaryKey
    val localId: Long,

    // Target room
    @ColumnInfo(name = "room_id")
    val roomId: String,

    // Message content
    @ColumnInfo(name = "content")
    val content: MessageContent,

    // Client timestamp
    @ColumnInfo(name = "timestamp")
    val timestamp: Instant,

    // Transaction ID (for deduplication)
    @ColumnInfo(name = "transaction_id")
    val transactionId: String,

    // Retry tracking
    @ColumnInfo(name = "retry_count")
    val retryCount: Int = 0,

    @ColumnInfo(name = "last_retry_at")
    val lastRetryAt: Instant? = null,

    @ColumnInfo(name = "next_retry_at")
    val nextRetryAt: Instant? = null,

    // Expiry
    @ColumnInfo(name = "expires_at")
    val expiresAt: Instant
)
```

---

## 3. Sync Manager Implementation

### 3.1 Core Sync Manager

```kotlin
class OfflineSyncManager(
    private val database: ArmorClawDatabase,
    private val matrixClient: MatrixClient,
    private val syncConfig: SyncConfig,
    private val context: Context
) {
    private val messageDao = database.messageDao()
    private val syncMetadataDao = database.syncMetadataDao()
    private val queuedMessageDao = database.queuedMessageDao()

    private val _syncState = MutableStateFlow<GlobalSyncState>(GlobalSyncState.IDLE)
    val syncState: StateFlow<GlobalSyncState> = _syncState.asStateFlow()

    private val syncScope = CoroutineScope(
        Dispatchers.IO +
        SupervisorJob() +
        CoroutineName("OfflineSyncManager")
    )

    /**
     * Initialize sync manager and start background sync
     */
    fun initialize() {
        // Register network callback
        registerNetworkCallback()

        // Start periodic sync worker
        schedulePeriodicSync()

        // Start processing queued messages
        startQueueProcessor()
    }

    /**
     * Queue a message for sending when online
     */
    suspend fun queueMessage(
        roomId: String,
        content: MessageContent
    ): Result<LocalMessage> {
        val transactionId = generateTransactionId()
        val timestamp = Clock.System.now()
        val expiresAt = timestamp + syncConfig.messageExpiry

        val localMessage = LocalMessage(
            localId = 0,  // Auto-generate
            eventId = null,
            roomId = roomId,
            senderId = matrixClient.userId(),
            content = content,
            timestamp = timestamp,
            syncStatus = SyncStatus.Pending,
            isOutgoing = true,
            serverTimestamp = null
        )

        val insertedId = messageDao.insert(localMessage)

        val queuedMessage = QueuedMessage(
            localId = insertedId,
            roomId = roomId,
            content = content,
            timestamp = timestamp,
            transactionId = transactionId,
            expiresAt = expiresAt
        )

        queuedMessageDao.insert(queuedMessage)

        // Update room metadata
        syncMetadataDao.incrementPendingCount(roomId)

        // Trigger immediate sync attempt if online
        if (isNetworkAvailable()) {
            syncScope.launch {
                processQueue()
            }
        }

        return Result.success(localMessage.copy(localId = insertedId))
    }

    /**
     * Sync all rooms when connection restored
     */
    suspend fun syncWhenOnline(): SyncResult {
        if (!isNetworkAvailable()) {
            return SyncResult.Offline
        }

        _syncState.value = GlobalSyncState.SYNCING

        return try {
            // Process outgoing queue first
            val queueResult = processQueue()

            // Then sync incoming messages
            val incomingResult = syncIncomingMessages()

            // Handle conflicts
            resolveConflicts()

            SyncResult.Success(
                messagesSent = queueResult.sent,
                messagesReceived = incomingResult.received,
                conflicts = incomingResult.conflicts
            )
        } catch (e: Exception) {
            SyncResult.Error(e.message ?: "Sync failed")
        } finally {
            _syncState.value = GlobalSyncState.IDLE
        }
    }

    /**
     * Process queued outgoing messages
     */
    private suspend fun processQueue(): QueueProcessResult {
        var sent = 0
        var failed = 0

        val queued = queuedMessageDao.getReadyToSend(Clock.System.now())

        for (message in queued) {
            try {
                val result = matrixClient.sendMessage(
                    roomId = message.roomId,
                    content = message.content,
                    transactionId = message.transactionId
                )

                if (result.isSuccess) {
                    val eventId = result.getOrNull()?.eventId
                        ?: throw Exception("No event ID in response")

                    // Update message with event ID
                    messageDao.updateEventId(
                        localId = message.localId,
                        eventId = eventId,
                        status = SyncStatus.Synced
                    )

                    // Remove from queue
                    queuedMessageDao.delete(message)

                    // Update metadata
                    syncMetadataDao.decrementPendingCount(message.roomId)

                    sent++
                } else {
                    handleQueueFailure(message, result.exceptionOrNull())
                    failed++
                }
            } catch (e: Exception) {
                handleQueueFailure(message, e)
                failed++
            }
        }

        return QueueProcessResult(sent = sent, failed = failed)
    }

    /**
     * Handle queued message failure with backoff
     */
    private suspend fun handleQueueFailure(
        message: QueuedMessage,
        error: Throwable?
    ) {
        val newRetryCount = message.retryCount + 1

        if (newRetryCount >= syncConfig.maxRetries) {
            // Give up, mark as failed
            messageDao.updateStatus(
                localId = message.localId,
                status = SyncStatus.Failed(
                    error = error?.message ?: "Max retries exceeded",
                    retryAfter = null
                )
            )
            queuedMessageDao.delete(message)
        } else {
            // Schedule retry with exponential backoff
            val backoffDelay = syncConfig.initialRetryDelay *
                (2.0.pow(newRetryCount.toDouble())).toLong()

            val nextRetryAt = Clock.System.now() + backoffDelay

            queuedMessageDao.updateRetry(
                localId = message.localId,
                retryCount = newRetryCount,
                nextRetryAt = nextRetryAt
            )
        }
    }

    /**
     * Sync incoming messages from Matrix
     */
    private suspend fun syncIncomingMessages(): IncomingSyncResult {
        var received = 0
        var conflicts = 0

        val rooms = syncMetadataDao.getAll()

        for (metadata in rooms) {
            try {
                val response = matrixClient.sync(
                    since = metadata.lastSyncToken,
                    timeout = syncConfig.syncTimeout
                )

                if (response.isSuccess) {
                    val syncData = response.getOrNull()!!

                    // Process new events
                    for (roomData in syncData.rooms.join) {
                        val timeline = roomData.timeline
                        val roomId = roomData.roomId

                        for (event in timeline.events) {
                            // Check for conflicts
                            val existing = messageDao.findByEventId(event.eventId)

                            if (existing != null) {
                                // Conflict detected - same event ID but different content
                                resolveMessageConflict(existing, event)
                                conflicts++
                            } else {
                                // New message, insert it
                                val localMessage = event.toLocalMessage(roomId)
                                messageDao.insert(localMessage)
                                received++
                            }
                        }

                        // Update sync token
                        syncMetadataDao.updateSyncToken(
                            roomId = roomId,
                            token = timeline.prevBatch,
                            time = Clock.System.now()
                        )
                    }
                }
            } catch (e: Exception) {
                // Log error but continue with other rooms
                Log.e("OfflineSyncManager", "Failed to sync room ${metadata.roomId}", e)
            }
        }

        return IncomingSyncResult(received = received, conflicts = conflicts)
    }

    /**
     * Resolve message conflicts
     */
    private suspend fun resolveMessageConflict(
        local: LocalMessage,
        remote: MatrixEvent
    ) {
        // Conflict resolution strategy: prefer server timestamp
        val resolution = when {
            local.serverTimestamp == null && remote.originServerTs != null -> {
                // Local has no server timestamp, use remote
                ConflictResolution.UseRemote
            }
            local.serverTimestamp != null && remote.originServerTs == null -> {
                // Remote has no server timestamp, keep local
                ConflictResolution.UseLocal
            }
            local.serverTimestamp != null && remote.originServerTs != null -> {
                // Both have timestamps, compare them
                if (remote.originServerTs > local.serverTimestamp) {
                    ConflictResolution.UseRemote
                } else {
                    ConflictResolution.UseLocal
                }
            }
            else -> {
                // Both null, compare client timestamps
                if (remote.originServerTs > local.timestamp.toEpochMilliseconds()) {
                    ConflictResolution.UseRemote
                } else {
                    ConflictResolution.UseLocal
                }
            }
        }

        when (resolution) {
            is ConflictResolution.UseRemote -> {
                val updated = local.copy(
                    content = remote.content.toMessageContent(),
                    serverTimestamp = remote.originServerTs?.let { Instant.fromEpochMilliseconds(it) },
                    syncStatus = SyncStatus.Synced,
                    editCount = local.editCount + 1
                )
                messageDao.update(updated)
            }
            is ConflictResolution.UseLocal -> {
                // Keep local, mark as sync conflict resolved
                val updated = local.copy(
                    syncStatus = SyncStatus.Conflict(resolution)
                )
                messageDao.update(updated)
            }
            is ConflictResolution.Manual -> {
                // Notify user for manual resolution
                notifyConflict(local, remote)
            }
        }
    }

    /**
     * Network monitoring
     */
    private fun registerNetworkCallback() {
        val connectivityManager = context.getSystemService(Context.CONNECTIVITY_SERVICE)
            as ConnectivityManager

        val networkCallback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                super.onAvailable(network)
                syncScope.launch {
                    syncWhenOnline()
                }
            }

            override fun onLost(network: Network) {
                super.onLost(network)
                _syncState.value = GlobalSyncState.Offline
            }
        }

        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .addCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
            .build()

        connectivityManager.registerNetworkCallback(request, networkCallback)
    }

    private fun isNetworkAvailable(): Boolean {
        val connectivityManager = context.getSystemService(Context.CONNECTIVITY_SERVICE)
            as ConnectivityManager

        val network = connectivityManager.activeNetwork ?: return false
        val capabilities = connectivityManager.getNetworkCapabilities(network) ?: return false

        return capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
            capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
    }

    private fun generateTransactionId(): String {
        return "m${Clock.System.now().toEpochMilliseconds()}_${randomUUID()}"
    }
}

sealed class GlobalSyncState {
    object IDLE : GlobalSyncState()
    object SYNCING : GlobalSyncState()
    object Offline : GlobalSyncState()
    data class Error(val error: String) : GlobalSyncState()
}

sealed class ConflictResolution {
    object UseLocal : ConflictResolution()
    object UseRemote : ConflictResolution()
    object Manual : ConflictResolution()
}

data class SyncConfig(
    val maxOfflineMessages: Int = 1000,
    val maxOfflineDays: Int = 7,
    val syncBatchSize: Int = 50,
    val syncTimeout: Duration = 30.seconds,
    val messageExpiry: Duration = 7.days,
    val initialRetryDelay: Duration = 5.seconds,
    val maxRetries: Int = 5,
    val periodicSyncInterval: Duration = 5.minutes
)
```

### 3.2 DAO Interfaces

```kotlin
@Dao
interface MessageDao {
    @Query("SELECT * FROM messages WHERE room_id = :roomId ORDER BY timestamp DESC LIMIT :limit OFFSET :offset")
    suspend fun getMessages(roomId: String, limit: Int = 50, offset: Int = 0): List<LocalMessage>

    @Query("SELECT * FROM messages WHERE local_id = :localId")
    suspend fun getByLocalId(localId: Long): LocalMessage?

    @Query("SELECT * FROM messages WHERE event_id = :eventId")
    suspend fun findByEventId(eventId: String): LocalMessage?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(message: LocalMessage): Long

    @Update
    suspend fun update(message: LocalMessage)

    @Query("UPDATE messages SET event_id = :eventId, sync_status = :status WHERE local_id = :localId")
    suspend fun updateEventId(localId: Long, eventId: String, status: SyncStatus)

    @Query("UPDATE messages SET sync_status = :status WHERE local_id = :localId")
    suspend fun updateStatus(localId: Long, status: SyncStatus)

    @Query("DELETE FROM messages WHERE local_id = :localId")
    suspend fun delete(localId: Long)

    @Query("DELETE FROM messages WHERE sync_status = 'PENDING' AND timestamp < :cutoff")
    suspend fun deleteExpiredPending(cutoff: Instant): Int

    @Query("SELECT COUNT(*) FROM messages WHERE sync_status = 'PENDING'")
    suspend fun getPendingCount(): Int
}

@Dao
interface SyncMetadataDao {
    @Query("SELECT * FROM sync_metadata")
    suspend fun getAll(): List<SyncMetadata>

    @Query("SELECT * FROM sync_metadata WHERE room_id = :roomId")
    suspend fun getRoom(roomId: String): SyncMetadata?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(metadata: SyncMetadata)

    @Update
    suspend fun update(metadata: SyncMetadata)

    @Query("UPDATE sync_metadata SET last_sync_token = :token, last_sync_time = :time WHERE room_id = :roomId")
    suspend fun updateSyncToken(roomId: String, token: String?, time: Instant)

    @Query("UPDATE sync_metadata SET pending_count = pending_count + 1 WHERE room_id = :roomId")
    suspend fun incrementPendingCount(roomId: String)

    @Query("UPDATE sync_metadata SET pending_count = pending_count - 1 WHERE room_id = :roomId")
    suspend fun decrementPendingCount(roomId: String)
}

@Dao
interface QueuedMessageDao {
    @Query("SELECT * FROM queued_messages WHERE next_retry_at IS NULL OR next_retry_at <= :now ORDER BY timestamp ASC")
    suspend fun getReadyToSend(now: Instant): List<QueuedMessage>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(message: QueuedMessage)

    @Query("UPDATE queued_messages SET retry_count = :retryCount, next_retry_at = :nextRetryAt WHERE local_id = :localId")
    suspend fun updateRetry(localId: Long, retryCount: Int, nextRetryAt: Instant)

    @Query("DELETE FROM queued_messages WHERE local_id = :localId")
    suspend fun delete(message: QueuedMessage)

    @Query("DELETE FROM queued_messages WHERE expires_at < :now")
    suspend fun deleteExpired(now: Int): Int
}
```

---

## 4. UI States & Indicators

### 4.1 Sync State Indicator

```
┌─────────────────────────────────────────────────────────────┐
│  ArmorClaw Agent                   [🔄 Syncing 12 msgs...] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  State Descriptions:                                        │
│                                                             │
│  🔄 Syncing - Show spinner with message count              │
│  ✓ All synced - Show green checkmark briefly               │
│  ⚠️ Pending - Show warning with pending count              │
│  ❌ Sync failed - Show error with retry button             │
│  📴 Offline - Show offline icon                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Message Status Indicators

```
┌─────────────────────────────────────────────────────────────┐
│  You                                        2:30 PM  ⏳     │
│  ┌─────────────────────────────────────────────┐           │
│  │ Can you analyze this data?                  │           │
│  └─────────────────────────────────────────────┘           │
│                                                             │
│  Status Indicators:                                         │
│  ⏳ Sending - Show clock icon                              │
│  ✓ Sent - Single checkmark                                 │
│  ✓✓ Delivered - Double checkmark                           │
│  ✗ Failed - Show red X with retry button                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Offline Banner

```
╔═══════════════════════════════════════════════════════════════╗
║ ⚠️ You're offline                                            ║
║ Messages will be sent when you reconnect.                     ║
║                                                      [Retry]  ║
╚══════──────────────────────────────────────────────────────────╝
```

---

## 5. Conflict Resolution UI

### 5.1 Conflict Notification

```
┌─────────────────────────────────────────────────────────────┐
│                    ⚠️ Sync Conflict                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  We found conflicting versions of a message:                │
│                                                             │
│  ┌─────────────────────────────────────────────────┐       │
│  │ Your version (2:30 PM):                          │       │
│  │ "Can you analyze this data?"                     │       │
│  │                                    [Keep Yours]  │       │
│  └─────────────────────────────────────────────────┘       │
│                                                             │
│  ┌─────────────────────────────────────────────────┐       │
│  │ Server version (2:31 PM):                       │       │
│  │ "Can you please analyze this data?"             │       │
│  │                                  [Keep Server]  │       │
│  └─────────────────────────────────────────────────┘       │
│                                                             │
│  By default, we'll keep the newer version.                  │
│                                                             │
│  [Auto-Resolve]                      [Dismiss]              │
└─────────────────────────────────────────────────────────────┘
```

---

## 6. Background Sync Worker

```kotlin
class PeriodicSyncWorker(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val syncManager = OfflineSyncManager(
            database = ArmorClawDatabase.getInstance(applicationContext),
            matrixClient = MatrixClient.getInstance(),
            syncConfig = SyncConfig(),
            context = applicationContext
        )

        return try {
            val result = syncManager.syncWhenOnline()
            when (result) {
                is SyncResult.Success -> Result.success()
                is SyncResult.Offline -> Result.retry()
                is SyncResult.Error -> Result.failure()
            }
        } catch (e: Exception) {
            Result.failure()
        }
    }
}

// Schedule worker
fun schedulePeriodicSync() {
    val constraints = Constraints.Builder()
        .setRequiredNetworkType(NetworkType.CONNECTED)
        .setRequiresBatteryNotLow(true)
        .build()

    val periodicRequest = PeriodicWorkRequestBuilder<PeriodicSyncWorker>(
        15,  // Minimum interval for periodic work
        TimeUnit.MINUTES
    )
        .setConstraints(constraints)
        .setBackoffCriteria(
            BackoffPolicy.EXPONENTIAL,
            30,
            TimeUnit.SECONDS
        )
        .build()

    WorkManager.getInstance(context)
        .enqueueUniquePeriodicWork(
            "periodic_sync",
            ExistingPeriodicWorkPolicy.KEEP,
            periodicRequest
        )
}
```

---

## 7. Testing Strategy

### 7.1 Test Scenarios

1. **Offline Message Queuing**
   - Enable airplane mode
   - Send 5 messages
   - Verify all show "⏳" status
   - Verify messages in local database with status PENDING

2. **Sync on Reconnect**
   - Disable airplane mode
   - Verify sync indicator appears
   - Verify all messages sent
   - Verify status changes to "✓✓"

3. **Conflict Resolution**
   - Simulate concurrent edits on same message
   - Verify conflict dialog appears
   - Test both resolution options
   - Verify final state consistent

4. **Retry Logic**
   - Send message with network error
   - Verify retry with exponential backoff
   - Verify max retry limit enforced

5. **Message Expiry**
   - Queue message with old timestamp
   - Verify expired messages cleaned up

### 7.2 Performance Tests

- Queue 1000 messages while offline
- Verify UI remains responsive
- Verify sync completes within timeout
- Test memory usage during sync

---

## 8. Security Considerations

1. **Local Storage Encryption**
   - Use SQLCipher for database encryption
   - Key derived from device lock credentials
   - Database locked when app backgrounded

2. **E2EE Message Handling**
   - Decrypt messages before storage
   - Re-encrypt with local storage key
   - Never store unencrypted E2EE keys

3. **Transaction ID Security**
   - Use cryptographically random IDs
   - Include timestamp for uniqueness
   - Prevent replay attacks

4. **Conflict Resolution Security**
   - Verify event signatures
   - Check sender identity
   - Prevent injection attacks

---

## 9. Offline Fallback Modes

### 9.1 Feature Degradation

| Feature | Online | Offline | Fallback |
|---------|--------|---------|----------|
| Send messages | ✅ | ⏳ Queue | Queue for later |
| Receive messages | ✅ | ❌ | Read from cache |
| Search | ✅ | ⚠️ | Search cached only |
| User presence | ✅ | ❌ | Show last known |
| Typing indicators | ✅ | ❌ | Disabled |
| File uploads | ✅ | ⏳ Queue | Queue for later |
| Voice input | ✅ | ⚠️ | Local processing only |
| Agent commands | ✅ | ❌ | Show "unavailable" |

### 9.2 Read-Only History Mode

When offline, users can:
- ✅ Read all cached messages
- ✅ Search cached content
- ✅ View room metadata
- ✅ Compose messages (queued)

When offline, users cannot:
- ❌ Send commands to agents
- ❌ Start/stop agents
- ❌ Modify settings
- ❌ Upload files
- ❌ Make voice calls

---

**Document Version:** 1.0.0
**Last Updated:** 2026-02-10
**Status:** Ready for Implementation
