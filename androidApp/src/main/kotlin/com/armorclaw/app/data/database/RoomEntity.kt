package com.armorclaw.app.data.database

import com.armorclaw.shared.domain.model.Room as DomainRoom
import com.armorclaw.shared.domain.model.RoomType
import com.armorclaw.shared.domain.model.Membership
import kotlinx.serialization.Serializable

/**
 * Local database entity for rooms.
 * Note: This is used for local caching. The primary database is SQLDelight.
 */
@Serializable
data class RoomEntity(
    val id: String,
    val name: String,
    val avatar: String?,
    val type: String,
    val membership: String,
    val topic: String?,
    val isDirect: Boolean,
    val isFavorite: Boolean,
    val isMuted: Boolean,
    val unreadCount: Int,
    val createdAt: Long
)

/**
 * Convert Domain Room to Entity
 */
fun DomainRoom.toEntity(): RoomEntity {
    return RoomEntity(
        id = id,
        name = name,
        avatar = avatar,
        type = type.name,
        membership = membership.name,
        topic = topic,
        isDirect = isDirect,
        isFavorite = isFavorite,
        isMuted = isMuted,
        unreadCount = unreadCount,
        createdAt = createdAt.toEpochMilliseconds()
    )
}

/**
 * Convert Entity to Domain Room
 */
fun RoomEntity.toDomain(): DomainRoom {
    return DomainRoom(
        id = id,
        name = name,
        avatar = avatar,
        type = RoomType.valueOf(type),
        membership = Membership.valueOf(membership),
        topic = topic,
        isDirect = isDirect,
        isFavorite = isFavorite,
        isMuted = isMuted,
        unreadCount = unreadCount,
        createdAt = kotlinx.datetime.Instant.fromEpochMilliseconds(createdAt)
    )
}
