package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable
import kotlinx.datetime.Instant

@Serializable
data class Message(
    val id: String,
    val roomId: String,
    val senderId: String,
    val content: MessageContent,
    val timestamp: Instant,
    val isOutgoing: Boolean,
    val status: MessageStatus,
    val serverTimestamp: Instant? = null,
    val editCount: Int = 0,
    val replyTo: String? = null,
    val isDeleted: Boolean = false,
    // Thread support (Epic A)
    val threadInfo: ThreadInfo? = null,
    // Reactions
    val reactions: List<Reaction> = emptyList()
)

/**
 * Thread information for threaded messages
 * Supports Matrix m.thread specification (MSC3440)
 */
@Serializable
data class ThreadInfo(
    /** ID of the root message that started this thread */
    val threadRootId: String? = null,
    /** Whether this message is a reply within a thread */
    val isThreadReply: Boolean = false,
    /** Depth in the thread tree (0 = root, 1+ = replies) */
    val threadDepth: Int = 0,
    /** List of user IDs participating in this thread */
    val threadParticipants: List<String> = emptyList(),
    /** Total number of replies in this thread */
    val replyCount: Int = 0,
    /** Timestamp of the most recent reply */
    val lastReplyAt: Instant? = null,
    /** ID of the last reply in this thread */
    val lastReplyId: String? = null,
    /** Whether this thread has unread messages for current user */
    val hasUnread: Boolean = false,
    /** Number of unread replies in this thread */
    val unreadCount: Int = 0
) {
    /**
     * Check if this is a thread root message
     */
    fun isThreadRoot(): Boolean = threadRootId != null && !isThreadReply

    /**
     * Check if this message belongs to a thread
     */
    fun isInThread(): Boolean = threadRootId != null

    /**
     * Get a summary for display
     */
    fun getSummary(): String {
        return when {
            replyCount == 0 -> "Start a thread"
            replyCount == 1 -> "1 reply"
            else -> "$replyCount replies"
        }
    }
}

/**
 * Emoji reaction on a message
 */
@Serializable
data class Reaction(
    val emoji: String,
    val count: Int,
    val includesMe: Boolean = false,
    val reactedBy: List<String> = emptyList() // User IDs
)

@Serializable
data class MessageContent(
    val type: MessageType,
    val body: String,
    val formattedBody: String? = null,
    val attachments: List<Attachment> = emptyList(),
    val mentions: List<Mention> = emptyList()
)

@Serializable
enum class MessageType {
    TEXT,
    IMAGE,
    FILE,
    AUDIO,
    VIDEO,
    NOTICE,
    EMOTE
}

@Serializable
enum class MessageStatus {
    PENDING,
    SENDING,
    SENT,
    DELIVERED,
    READ,
    FAILED,
    SYNCED
}

@Serializable
data class Attachment(
    val url: String,
    val mimeType: String,
    val size: Long,
    val thumbnailUrl: String? = null,
    val fileName: String
)

@Serializable
data class Mention(
    val userId: String,
    val displayName: String
)
