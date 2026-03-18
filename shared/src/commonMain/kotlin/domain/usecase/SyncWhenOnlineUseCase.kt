package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.SyncResult
import com.armorclaw.shared.domain.repository.SyncRepository

class SyncWhenOnlineUseCase(
    private val syncRepository: SyncRepository
) {
    suspend operator fun invoke(): SyncResult {
        return syncRepository.syncWhenOnline()
    }
}
