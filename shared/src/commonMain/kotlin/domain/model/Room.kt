package com.armorclaw.shared.domain.model

import com.armorclaw.shared.domain.model.BridgeRoomCapabilities
import kotlinx.serialization.Serializable
import kotlinx.datetime.Instant

@Serializable
data class Room(
    val id: String,
    val name: String,
    val avatar: String? = null,
    val type: RoomType,
    val membership: Membership,
    val topic: String? = null,
    val isDirect: Boolean = false,
    val isFavorite: Boolean = false,
    val isMuted: Boolean = false,
    val lastMessage: MessageSummary? = null,
    val unreadCount: Int = 0,
    val members: List<RoomMember> = emptyList(),
    val createdAt: Instant,
    /** Non-null when this room is bridged to an external platform via SDTW */
    val bridgeCapabilities: BridgeRoomCapabilities? = null
)

@Serializable
data class RoomMember(
    val userId: String,
    val displayName: String,
    val avatar: String? = null,
    val membership: Membership,
    val powerLevel: Int = 0,
    val presence: UserPresence = UserPresence.OFFLINE
)

@Serializable
enum class RoomType {
    DIRECT,
    GROUP,
    SPACE
}

@Serializable
enum class Membership {
    JOIN,
    INVITE,
    LEAVE,
    BAN,
    KNOCK
}

@Serializable
data class MessageSummary(
    val id: String,
    val content: String,
    val senderId: String,
    val timestamp: Instant,
    val isOutgoing: Boolean
)
