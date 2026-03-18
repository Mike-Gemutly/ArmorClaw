package com.armorclaw.app.data.database

import com.armorclaw.shared.domain.model.Message as DomainMessage
import com.armorclaw.shared.domain.model.MessageContent
import com.armorclaw.shared.domain.model.MessageStatus
import com.armorclaw.shared.domain.model.MessageType
import com.armorclaw.shared.domain.model.ThreadInfo
import com.armorclaw.shared.domain.model.Reaction
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * Local database entity for messages.
 * Note: This is used for local caching. The primary database is SQLDelight.
 */
@Serializable
data class MessageEntity(
    val id: String,
    val roomId: String,
    val senderId: String,
    val contentBody: String,
    val contentType: String,
    val timestamp: Long,
    val isOutgoing: Boolean,
    val status: String,
    val serverTimestamp: Long?,
    val editCount: Int,
    val replyToId: String?,
    val isDeleted: Boolean,
    val threadInfoJson: String?,
    val reactionsJson: String?
)

private val json = Json { ignoreUnknownKeys = true }

/**
 * Convert Domain Message to Entity
 */
fun DomainMessage.toEntity(): MessageEntity {
    return MessageEntity(
        id = id,
        roomId = roomId,
        senderId = senderId,
        contentBody = content.body,
        contentType = content.type.name,
        timestamp = timestamp.toEpochMilliseconds(),
        isOutgoing = isOutgoing,
        status = status.name,
        serverTimestamp = serverTimestamp?.toEpochMilliseconds(),
        editCount = editCount,
        replyToId = replyTo,
        isDeleted = isDeleted,
        threadInfoJson = threadInfo?.let { json.encodeToString(it) },
        reactionsJson = if (reactions.isNotEmpty()) json.encodeToString(reactions) else null
    )
}

/**
 * Convert Entity to Domain Message
 */
fun MessageEntity.toDomain(): DomainMessage {
    return DomainMessage(
        id = id,
        roomId = roomId,
        senderId = senderId,
        content = MessageContent(
            type = MessageType.valueOf(contentType),
            body = contentBody
        ),
        timestamp = kotlinx.datetime.Instant.fromEpochMilliseconds(timestamp),
        isOutgoing = isOutgoing,
        status = MessageStatus.valueOf(status),
        serverTimestamp = serverTimestamp?.let {
            kotlinx.datetime.Instant.fromEpochMilliseconds(it)
        },
        editCount = editCount,
        replyTo = replyToId,
        isDeleted = isDeleted,
        threadInfo = threadInfoJson?.let { json.decodeFromString<ThreadInfo>(it) },
        reactions = reactionsJson?.let { json.decodeFromString<List<Reaction>>(it) } ?: emptyList()
    )
}
