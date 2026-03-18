package com.armorclaw.shared.domain.usecase

import com.armorclaw.shared.domain.model.AppError
import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.ArmorClawErrorCode
import com.armorclaw.shared.platform.bridge.BridgeRpcClient
import com.armorclaw.shared.platform.bridge.FeatureInfo
import com.armorclaw.shared.platform.bridge.LicenseFeaturesResponse
import com.armorclaw.shared.platform.bridge.RpcResult
import com.armorclaw.shared.platform.logging.LogTag
import com.armorclaw.shared.platform.logging.repositoryLogger
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/**
 * Check Feature Use Case
 *
 * Checks if specific features are available for the current license tier.
 * Since license.checkFeature doesn't exist on Bridge, we use license.features
 * and cache the results locally.
 *
 * ## Usage
 * ```kotlin
 * val checkFeature = CheckFeatureUseCase(bridgeClient)
 *
 * // Check if Slack bridging is available
 * val result = checkFeature("platform_slack")
 * if (result.getOrThrow()) {
 *     // Enable Slack integration
 * }
 *
 * // Check with limit info
 * val platformResult = checkFeature("platform_bridging")
 * println("Limit: ${platformResult.limit}, Current: ${platformResult.current}")
 * ```
 *
 * ## Caching
 * Features are cached after first fetch to avoid repeated RPC calls.
 * Call refreshFeatures() to force a refresh from the server.
 *
 * ## Available Feature IDs
 * - platform_slack: Slack bridging
 * - platform_discord: Discord bridging
 * - platform_teams: Microsoft Teams bridging
 * - platform_whatsapp: WhatsApp bridging (Enterprise only)
 * - e2ee: End-to-end encryption
 * - voice_calls: Voice calling
 * - video_calls: Video calling
 * - file_sharing: File/media sharing
 * - message_history: Extended message history
 * - ai_assistant: AI assistant features
 * - compliance_hipaa: HIPAA compliance mode
 * - compliance_audit: Audit logging
 * - multi_device: Multi-device sync
 */
class CheckFeatureUseCase(
    private val bridgeClient: BridgeRpcClient
) {
    private val logger = repositoryLogger("CheckFeatureUseCase", LogTag.Domain.License)
    private val mutex = Mutex()

    // Feature cache
    private var cachedFeatures: Map<String, FeatureInfo>? = null
    private var cachedTier: String? = null
    private var lastRefreshTime: Long = 0
    private val cacheValidDurationMs: Long = 5 * 60 * 1000 // 5 minutes

    /**
     * Check if a feature is available
     *
     * @param featureId The feature identifier to check
     * @param forceRefresh Force a refresh from the server
     * @return AppResult with availability boolean
     */
    suspend operator fun invoke(
        featureId: String,
        forceRefresh: Boolean = false
    ): AppResult<FeatureAvailability> {
        return checkFeature(featureId, forceRefresh)
    }

    /**
     * Check if a feature is available (simple boolean result)
     *
     * @param featureId The feature identifier to check
     * @param forceRefresh Force a refresh from the server
     * @return AppResult<Boolean>
     */
    suspend fun isAvailable(
        featureId: String,
        forceRefresh: Boolean = false
    ): AppResult<Boolean> {
        return checkFeature(featureId, forceRefresh).map { it.available }
    }

    /**
     * Check if a feature is available and within limits
     *
     * For features with limits (e.g., max channels), this checks both
     * availability and whether the current usage is within the limit.
     *
     * @param featureId The feature identifier
     * @param currentUsage Current usage count
     * @return AppResult<Boolean>
     */
    suspend fun isWithinLimit(
        featureId: String,
        currentUsage: Int,
        forceRefresh: Boolean = false
    ): AppResult<Boolean> {
        return checkFeature(featureId, forceRefresh).map { availability ->
            if (!availability.available) {
                false
            } else if (availability.limit != null) {
                currentUsage < availability.limit
            } else {
                true // No limit defined, allow
            }
        }
    }

    /**
     * Get all available features
     *
     * @param forceRefresh Force a refresh from the server
     * @return Map of feature ID to FeatureInfo
     */
    suspend fun getAllFeatures(forceRefresh: Boolean = false): AppResult<Map<String, FeatureInfo>> {
        return mutex.withLock {
            if (!forceRefresh && isCacheValid() && cachedFeatures != null) {
                return@withLock AppResult.success(cachedFeatures!!)
            }

            when (val result = bridgeClient.licenseFeatures()) {
                is RpcResult.Success -> {
                    cachedFeatures = result.data.features
                    cachedTier = result.data.tier
                    lastRefreshTime = System.currentTimeMillis()

                    logger.logOperationSuccess(
                        "getAllFeatures",
                        "cached ${result.data.features.size} features for tier ${result.data.tier}"
                    )

                    AppResult.success(result.data.features)
                }
                is RpcResult.Error -> {
                    logger.logOperationError("getAllFeatures", Exception(result.message))

                    // Return cached data if available, even if stale
                    if (cachedFeatures != null) {
                        logger.logWarning("Returning stale cache due to error")
                        AppResult.success(cachedFeatures!!)
                    } else {
                        createErrorResult(result, "getAllFeatures")
                    }
                }
            }
        }
    }

    /**
     * Get the current license tier
     */
    suspend fun getLicenseTier(): AppResult<String> {
        return mutex.withLock {
            if (isCacheValid() && cachedTier != null) {
                return@withLock AppResult.success(cachedTier!!)
            }

            when (val result = bridgeClient.licenseStatus()) {
                is RpcResult.Success -> {
                    cachedTier = result.data.tier
                    AppResult.success(result.data.tier)
                }
                is RpcResult.Error -> {
                    if (cachedTier != null) {
                        AppResult.success(cachedTier!!)
                    } else {
                        createErrorResult(result, "getLicenseTier")
                    }
                }
            }
        }
    }

    /**
     * Force refresh features from server
     */
    suspend fun refreshFeatures(): AppResult<Unit> {
        return getAllFeatures(forceRefresh = true).map { }
    }

    /**
     * Clear the feature cache
     */
    fun clearCache() {
        cachedFeatures = null
        cachedTier = null
        lastRefreshTime = 0
    }

    // Private implementation

    private suspend fun checkFeature(
        featureId: String,
        forceRefresh: Boolean
    ): AppResult<FeatureAvailability> {
        return mutex.withLock {
            // Use cached result if available and valid
            if (!forceRefresh && isCacheValid() && cachedFeatures != null) {
                val feature = cachedFeatures!![featureId]
                return@withLock if (feature != null) {
                    AppResult.success(FeatureAvailability(
                        featureId = featureId,
                        available = feature.available,
                        limit = feature.limit,
                        description = feature.description,
                        tier = cachedTier ?: "unknown"
                    ))
                } else {
                    AppResult.success(FeatureAvailability(
                        featureId = featureId,
                        available = false,
                        reason = "Feature not found in license"
                    ))
                }
            }

            // Fetch from server
            when (val result = bridgeClient.licenseFeatures()) {
                is RpcResult.Success -> {
                    cachedFeatures = result.data.features
                    cachedTier = result.data.tier
                    lastRefreshTime = System.currentTimeMillis()

                    val feature = result.data.features[featureId]
                    if (feature != null) {
                        AppResult.success(FeatureAvailability(
                            featureId = featureId,
                            available = feature.available,
                            limit = feature.limit,
                            description = feature.description,
                            tier = result.data.tier
                        ))
                    } else {
                        AppResult.success(FeatureAvailability(
                            featureId = featureId,
                            available = false,
                            reason = "Feature not found in license",
                            tier = result.data.tier
                        ))
                    }
                }
                is RpcResult.Error -> {
                    logger.logOperationError("checkFeature", Exception(result.message))

                    // Try to use stale cache
                    if (cachedFeatures != null) {
                        val feature = cachedFeatures!![featureId]
                        if (feature != null) {
                            logger.logWarning("Using stale cache for $featureId")
                            return@withLock AppResult.success(FeatureAvailability(
                                featureId = featureId,
                                available = feature.available,
                                limit = feature.limit,
                                description = feature.description,
                                tier = cachedTier ?: "unknown",
                                fromCache = true
                            ))
                        }
                    }

                    createErrorResult(result, "checkFeature")
                }
            }
        }
    }

    private fun isCacheValid(): Boolean {
        if (lastRefreshTime == 0L) return false
        return (System.currentTimeMillis() - lastRefreshTime) < cacheValidDurationMs
    }

    private fun createErrorResult(rpcError: RpcResult.Error, operation: String): AppResult<Nothing> {
        val errorCode = when (rpcError.code) {
            -32001, -32002 -> ArmorClawErrorCode.SESSION_EXPIRED
            -32006 -> ArmorClawErrorCode.NETWORK_CHANGED
            else -> ArmorClawErrorCode.UNKNOWN_ERROR
        }

        return AppResult.error(
            AppError(
                code = errorCode.code,
                message = rpcError.message,
                source = "CheckFeatureUseCase:$operation"
            )
        )
    }
}

/**
 * Feature availability information
 */
data class FeatureAvailability(
    val featureId: String,
    val available: Boolean,
    val limit: Int? = null,
    val description: String? = null,
    val tier: String? = null,
    val reason: String? = null,
    val fromCache: Boolean = false
) {
    /**
     * Check if usage is within limit
     */
    fun isWithinLimit(currentUsage: Int): Boolean {
        if (!available) return false
        if (limit == null) return true
        return currentUsage < limit
    }

    /**
     * Get remaining capacity
     */
    fun remainingCapacity(currentUsage: Int): Int? {
        if (limit == null) return null
        return maxOf(0, limit - currentUsage)
    }
}

/**
 * Feature IDs constants
 */
object FeatureIds {
    // Platform bridging
    const val PLATFORM_SLACK = "platform_slack"
    const val PLATFORM_DISCORD = "platform_discord"
    const val PLATFORM_TEAMS = "platform_teams"
    const val PLATFORM_WHATSAPP = "platform_whatsapp"
    const val PLATFORM_BRIDGING = "platform_bridging"

    // Communication features
    const val E2EE = "e2ee"
    const val VOICE_CALLS = "voice_calls"
    const val VIDEO_CALLS = "video_calls"
    const val FILE_SHARING = "file_sharing"
    const val MESSAGE_HISTORY = "message_history"

    // AI features
    const val AI_ASSISTANT = "ai_assistant"
    const val AI_SUMMARIZATION = "ai_summarization"
    const val AI_TRANSLATION = "ai_translation"

    // Compliance features
    const val COMPLIANCE_HIPAA = "compliance_hipaa"
    const val COMPLIANCE_AUDIT = "compliance_audit"
    const val COMPLIANCE_RETENTION = "compliance_retention"

    // Account features
    const val MULTI_DEVICE = "multi_device"
    const val CUSTOM_DOMAIN = "custom_domain"
    const val SSO = "sso"

    // Storage features
    const val FILE_STORAGE = "file_storage"
    const val MESSAGE_STORAGE = "message_storage"

    // Admin features
    const val ADMIN_DASHBOARD = "admin_dashboard"
    const val USER_MANAGEMENT = "user_management"
    const val ANALYTICS = "analytics"
}
