package app.armorclaw.data.local.entity

import androidx.room.Entity
import androidx.room.PrimaryKey

/**
 * User Entity - Represents a Matrix user or bridged ghost user
 *
 * Supports namespace tagging for identity consistency (G-04)
 */
@Entity(tableName = "users")
data class UserEntity(
    @PrimaryKey
    val userId: String,
    val displayName: String? = null,
    val avatarUrl: String? = null,
    val isBridgeBot: Boolean = false,
    val lastActive: Long? = null,
    val presence: String = "offline"
)
