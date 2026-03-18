package com.armorclaw.app.data.offline

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import androidx.work.Constraints
import androidx.work.NetworkType
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.domain.repository.SyncRepository
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject
import java.util.concurrent.TimeUnit

/**
 * Background sync worker for offline operations
 *
 * This worker executes pending offline operations when network is available
 * and the app is in the background.
 */
class BackgroundSyncWorker(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params), KoinComponent {

    override suspend fun doWork(): Result {
        val startTime = System.currentTimeMillis()
        logDebug("Background sync worker started", mapOf("workerId" to id.toString()))

        try {
            // Check if network is available
            if (!isNetworkAvailable()) {
                logWarning("Network not available, skipping sync", mapOf("workerId" to id.toString()))
                return Result.retry()
            }

            val syncRepository: SyncRepository by inject()

            if (!syncRepository.isOnline()) {
                logWarning("Repository reports offline status, skipping sync", mapOf("workerId" to id.toString()))
                return Result.retry()
            }

            logInfo("Starting background sync", mapOf("workerId" to id.toString()))
            val syncResult = syncRepository.syncWhenOnline()

            logInfo("Background sync completed", mapOf(
                "workerId" to id.toString(),
                "messagesSent" to syncResult.messagesSent,
                "messagesReceived" to syncResult.messagesReceived,
                "conflicts" to syncResult.conflicts
            ))

            val duration = System.currentTimeMillis() - startTime
            logPerformance("backgroundSync", duration, mapOf(
                "workerId" to id.toString(),
                "messagesSent" to syncResult.messagesSent,
                "messagesReceived" to syncResult.messagesReceived,
                "result" to "success"
            ))

            // Return success even if there were errors - sync completed without crashing
            return Result.success()

        } catch (e: Exception) {
            logError("Background sync worker failed", e, mapOf("workerId" to id.toString()))
            // Don't retry on every error - return failure to avoid infinite retry loops
            return Result.failure()
        }
    }

    /**
     * Check if network is available
     */
    @Suppress("DEPRECATION")
    private fun isNetworkAvailable(): Boolean {
        val connectivityManager = applicationContext.getSystemService(Context.CONNECTIVITY_SERVICE) as android.net.ConnectivityManager
        val activeNetwork = connectivityManager.activeNetworkInfo
        return activeNetwork?.isConnectedOrConnecting == true
    }

    companion object {
        const val WORK_NAME = "background_sync_work"

        /**
         * Create constraints for background sync
         */
        fun createConstraints(): Constraints {
            return Constraints.Builder()
                .setRequiredNetworkType(NetworkType.UNMETERED) // Only on WiFi
                .setRequiresBatteryNotLow(true)
                .setRequiresCharging(false)
                .build()
        }

        /**
         * Schedule periodic background sync
         */
        fun schedulePeriodicSync(context: Context, intervalMinutes: Long = 15) {
            val constraints = createConstraints()

            val workRequest = PeriodicWorkRequestBuilder<BackgroundSyncWorker>(
                intervalMinutes,
                TimeUnit.MINUTES
            )
                .setConstraints(constraints)
                .setInitialDelay(5, TimeUnit.MINUTES)
                .addTag(WORK_NAME)
                .build()

            WorkManager.getInstance(context).enqueueUniquePeriodicWork(
                WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                workRequest
            )

            AppLogger.info(LogTag.Worker.SyncWorker, "Scheduled periodic sync", mapOf("intervalMinutes" to intervalMinutes))
        }

        /**
         * Schedule one-time sync immediately
         */
        fun scheduleImmediateSync(context: Context) {
            val workRequest = OneTimeWorkRequestBuilder<BackgroundSyncWorker>()
                .setConstraints(createConstraints())
                .addTag(WORK_NAME)
                .build()

            WorkManager.getInstance(context).enqueueUniqueWork(
                "${WORK_NAME}_immediate",
                ExistingWorkPolicy.KEEP,
                workRequest
            )

            AppLogger.info(LogTag.Worker.SyncWorker, "Scheduled immediate sync")
        }

        /**
         * Cancel all background sync work
         */
        fun cancelSync(context: Context) {
            WorkManager.getInstance(context).cancelAllWorkByTag(WORK_NAME)
            AppLogger.info(LogTag.Worker.SyncWorker, "Cancelled background sync work")
        }
    }

    // Helper methods for logging using AppLogger directly
    private fun logDebug(message: String, data: Map<String, Any>? = null) {
        AppLogger.debug(LogTag.Worker.SyncWorker, message, data)
    }

    private fun logInfo(message: String, data: Map<String, Any>? = null) {
        AppLogger.info(LogTag.Worker.SyncWorker, message, data)
    }

    private fun logWarning(message: String, data: Map<String, Any>? = null) {
        AppLogger.warning(LogTag.Worker.SyncWorker, message, data)
    }

    private fun logError(message: String, throwable: Throwable? = null, data: Map<String, Any>? = null) {
        AppLogger.error(LogTag.Worker.SyncWorker, message, throwable, data)
    }

    private fun logPerformance(operation: String, durationMs: Long, data: Map<String, Any>? = null) {
        AppLogger.performance(LogTag.Worker.SyncWorker, operation, durationMs, data)
    }
}
