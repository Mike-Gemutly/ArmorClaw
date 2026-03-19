package com.armorclaw.app.secretary

import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.domain.repository.RoomRepository
import com.armorclaw.shared.secretary.BriefingContext

/**
 * SecretaryContextProvider - aggregates context data from app repositories.
 *
 * This provider collects data from existing repositories to build a BriefingContext
 * for the SecretaryBriefingEngine. It's designed to be simple and testable.
 *
 * Phase 2 Implementation:
 * - Aggregates unread count from rooms
 * - Placeholder for next meeting (Phase 3)
 * - Placeholder for pending approvals (Phase 3)
 * - Tracks briefing timestamps (persistent storage TBD)
 *
 * Note: This is in androidApp module (not shared) because it accesses
 * Android-specific repositories via Koin DI.
 */
class SecretaryContextProvider(
    private val roomRepository: RoomRepository,
    private val messageRepository: MessageRepository
) {
    // TODO: Add persistent storage for briefing timestamps in Phase 3
    private var lastMorningBriefingDate: Long? = null
    private var lastEveningReviewDate: Long? = null

    /**
     * Gather context from repositories for briefing generation.
     *
     * @return BriefingContext with aggregated data
     */
    suspend fun gatherContext(): BriefingContext {
        val roomsResult = roomRepository.getRooms()
        val totalUnread = if (roomsResult.isSuccess) {
            roomsResult.getOrNull()?.sumOf { it.unreadCount } ?: 0
        } else {
            0
        }

        val nextMeeting: String? = null
        val pendingApprovals: Int = 0

        return BriefingContext(
            unreadCount = totalUnread,
            nextMeeting = nextMeeting,
            pendingApprovals = pendingApprovals,
            lastMorningBriefingDate = lastMorningBriefingDate,
            eveningReviewEnabled = true,
            lastEveningReviewDate = lastEveningReviewDate
        )
    }

    /**
     * Update morning briefing timestamp to prevent duplicates.
     *
     * @param timestamp Current timestamp in milliseconds since epoch
     */
    fun updateMorningBriefingDate(timestamp: Long) {
        lastMorningBriefingDate = timestamp
    }

    /**
     * Update evening review timestamp to prevent duplicates.
     *
     * @param timestamp Current timestamp in milliseconds since epoch
     */
    fun updateEveningReviewDate(timestamp: Long) {
        lastEveningReviewDate = timestamp
    }
}
