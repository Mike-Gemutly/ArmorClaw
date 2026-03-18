package com.armorclaw.shared.platform.notification

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.AppError
import com.armorclaw.shared.domain.model.OperationContext
import com.armorclaw.shared.platform.bridge.BridgeRpcClient
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Repository for managing push notification token registration with the Bridge
 *
 * This repository handles:
 * - Registering FCM/APNs tokens with the Bridge Server
 * - Token persistence across app restarts
 * - Automatic re-registration when needed
 *
 * ## Usage
 *
 * Call `registerToken()` when:
 * - FCM/APNs provides a new token
 * - User logs in (to associate token with user)
 * - Token is refreshed
 *
 * Call `unregisterToken()` when:
 * - User logs out
 * - User disables notifications
 */
interface PushNotificationRepository {
    /**
     * Register push token with the Bridge Server
     *
     * @param pushToken The FCM or APNs device token
     * @param pushPlatform The platform type ("fcm" or "apns")
     * @param deviceId The device identifier
     * @param context Operation context for tracing
     * @return Registration result
     */
    suspend fun registerToken(
        pushToken: String,
        pushPlatform: String,
        deviceId: String,
        context: OperationContext? = null
    ): AppResult<Unit>

    /**
     * Unregister push token from the Bridge Server
     *
     * @param context Operation context for tracing
     * @return Unregistration result
     */
    suspend fun unregisterToken(context: OperationContext? = null): AppResult<Unit>

    /**
     * Update push notification settings
     *
     * @param enabled Whether push notifications are enabled
     * @param quietHoursStart Quiet hours start time (e.g., "22:00")
     * @param quietHoursEnd Quiet hours end time (e.g., "08:00")
     * @param context Operation context for tracing
     * @return Update result
     */
    suspend fun updateSettings(
        enabled: Boolean,
        quietHoursStart: String? = null,
        quietHoursEnd: String? = null,
        context: OperationContext? = null
    ): AppResult<Unit>

    /**
     * Get the currently registered token
     */
    fun getRegisteredToken(): String?

    /**
     * Observe push notification registration state
     */
    fun observeRegistrationState(): StateFlow<PushRegistrationState>
}

/**
 * Push notification registration state
 */
sealed class PushRegistrationState {
    /** Not registered */
    object NotRegistered : PushRegistrationState()

    /** Registration in progress */
    object Registering : PushRegistrationState()

    /** Successfully registered */
    data class Registered(val pushToken: String) : PushRegistrationState()

    /** Registration failed */
    data class Error(val message: String) : PushRegistrationState()
}

/**
 * Implementation of PushNotificationRepository using both MatrixClient and BridgeRpcClient
 *
 * ## Dual Registration Strategy
 * Push tokens are registered with BOTH:
 * 1. **Matrix Homeserver** (via MatrixClient.setPusher()) — ensures the homeserver
 *    can send push notifications through the standard Matrix push gateway when new
 *    events arrive. This is critical for background/Doze mode wakeups.
 * 2. **Bridge Server** (via BridgeRpcClient.pushRegister()) — allows the Bridge to
 *    send custom push notifications for SDTW bridging events.
 *
 * If the Matrix pusher registration fails, we still attempt Bridge registration
 * (and vice versa), logging warnings for partial failures.
 */
class PushNotificationRepositoryImpl(
    private val rpcClient: BridgeRpcClient,
    private val matrixClient: com.armorclaw.shared.platform.matrix.MatrixClient,
    /**
     * RC-04: Provider for the push gateway URL from setup config or discovery.
     * Falls back to the hardcoded default if the provider returns null.
     * The URL comes from QR payload, /discover, well-known, or provisioning.start.
     */
    private val pushGatewayUrlProvider: () -> String? = { null }
) : PushNotificationRepository {

    companion object {
        /** Default push gateway used when no dynamic URL is available */
        private const val DEFAULT_PUSH_GATEWAY_URL =
            "https://push.armorclaw.app/_matrix/push/v1/notify"
    }

    private val _registrationState = MutableStateFlow<PushRegistrationState>(PushRegistrationState.NotRegistered)
    private var registeredToken: String? = null

    override suspend fun registerToken(
        pushToken: String,
        pushPlatform: String,
        deviceId: String,
        context: OperationContext?
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()

        AppLogger.info(
            LogTag.Network.Fcm,
            "Registering push token (dual: Matrix + Bridge)",
            mapOf(
                "platform" to pushPlatform,
                "device_id" to deviceId,
                "token_prefix" to pushToken.take(10) + "...",
                "correlation_id" to ctx.correlationId
            )
        )

        _registrationState.value = PushRegistrationState.Registering

        // Step 1: Register with Matrix Homeserver (primary push path)
        // RC-04: Use dynamic push gateway URL from discovery/QR config
        // instead of relying on the hardcoded default in MatrixClient.setPusher().
        val gatewayUrl = pushGatewayUrlProvider() ?: DEFAULT_PUSH_GATEWAY_URL

        val matrixResult = try {
            matrixClient.setPusher(
                pushKey = pushToken,
                appId = "com.armorclaw.app",
                appDisplayName = "ArmorClaw",
                deviceDisplayName = "Android ($deviceId)",
                pushGatewayUrl = gatewayUrl
            )
        } catch (e: Exception) {
            AppLogger.error(
                LogTag.Network.Fcm,
                "Matrix pusher registration threw exception",
                e
            )
            Result.failure(e)
        }

        if (matrixResult.isSuccess) {
            AppLogger.info(LogTag.Network.Fcm, "Matrix pusher registered successfully")
        } else {
            AppLogger.warning(
                LogTag.Network.Fcm,
                "Matrix pusher registration failed — background notifications may not work"
            )
        }

        // Step 2: Register with Bridge Server (for SDTW bridging notifications)
        return when (val result = rpcClient.pushRegister(pushToken, pushPlatform, deviceId, ctx)) {
            is com.armorclaw.shared.platform.bridge.RpcResult.Success -> {
                registeredToken = pushToken
                _registrationState.value = PushRegistrationState.Registered(pushToken)

                AppLogger.info(
                    LogTag.Network.Fcm,
                    "Push token registered successfully (Bridge + Matrix)",
                    mapOf("device_id" to (result.data.deviceId ?: "unknown"))
                )

                AppResult.success(Unit)
            }
            is com.armorclaw.shared.platform.bridge.RpcResult.Error -> {
                // If Matrix succeeded but Bridge failed, still mark as registered
                // since Matrix push is the critical path
                if (matrixResult.isSuccess) {
                    registeredToken = pushToken
                    _registrationState.value = PushRegistrationState.Registered(pushToken)

                    AppLogger.warning(
                        LogTag.Network.Fcm,
                        "Bridge push registration failed but Matrix pusher is active — push will work",
                        mapOf("bridge_error" to result.message)
                    )
                    AppResult.success(Unit)
                } else {
                    _registrationState.value = PushRegistrationState.Error(result.message)

                    AppLogger.error(
                        LogTag.Network.Fcm,
                        "Failed to register push token with both Matrix and Bridge: ${result.message}",
                        null,
                        mapOf("code" to result.code)
                    )

                    AppResult.error(
                        AppError(
                            code = result.code.toString(),
                            message = result.message,
                            source = "PushNotificationRepository"
                        )
                    )
                }
            }
        }
    }

    override suspend fun unregisterToken(context: OperationContext?): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()
        val token = registeredToken

        if (token == null) {
            AppLogger.info(LogTag.Network.Fcm, "No token to unregister")
            return AppResult.success(Unit)
        }

        AppLogger.info(
            LogTag.Network.Fcm,
            "Unregistering push token (dual: Matrix + Bridge)",
            mapOf("correlation_id" to ctx.correlationId)
        )

        // Step 1: Remove Matrix pusher
        try {
            matrixClient.removePusher(pushKey = token)
            AppLogger.info(LogTag.Network.Fcm, "Matrix pusher removed successfully")
        } catch (e: Exception) {
            AppLogger.warning(
                LogTag.Network.Fcm,
                "Failed to remove Matrix pusher (non-fatal)"
            )
        }

        // Step 2: Unregister from Bridge
        return when (val result = rpcClient.pushUnregister(token, ctx)) {
            is com.armorclaw.shared.platform.bridge.RpcResult.Success -> {
                registeredToken = null
                _registrationState.value = PushRegistrationState.NotRegistered

                AppLogger.info(LogTag.Network.Fcm, "Push token unregistered successfully")
                AppResult.success(Unit)
            }
            is com.armorclaw.shared.platform.bridge.RpcResult.Error -> {
                // Still clear local state even if Bridge unregister fails
                registeredToken = null
                _registrationState.value = PushRegistrationState.NotRegistered

                AppLogger.error(
                    LogTag.Network.Fcm,
                    "Failed to unregister push token from Bridge: ${result.message}",
                    null
                )
                AppResult.success(Unit) // Non-fatal — local state is cleaned up
            }
        }
    }

    override suspend fun updateSettings(
        enabled: Boolean,
        quietHoursStart: String?,
        quietHoursEnd: String?,
        context: OperationContext?
    ): AppResult<Unit> {
        val ctx = context ?: OperationContext.create()

        return when (val result = rpcClient.pushUpdateSettings(enabled, quietHoursStart, quietHoursEnd, ctx)) {
            is com.armorclaw.shared.platform.bridge.RpcResult.Success -> {
                AppLogger.info(
                    LogTag.Network.Fcm,
                    "Push settings updated",
                    mapOf("enabled" to enabled)
                )
                AppResult.success(Unit)
            }
            is com.armorclaw.shared.platform.bridge.RpcResult.Error -> {
                AppResult.error(
                    AppError(
                        code = result.code.toString(),
                        message = result.message,
                        source = "PushNotificationRepository"
                    )
                )
            }
        }
    }

    override fun getRegisteredToken(): String? = registeredToken

    override fun observeRegistrationState(): StateFlow<PushRegistrationState> =
        _registrationState.asStateFlow()
}
