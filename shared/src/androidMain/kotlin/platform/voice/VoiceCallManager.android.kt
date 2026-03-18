package com.armorclaw.shared.platform.voice

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.media.AudioManager
import androidx.core.content.ContextCompat
import com.armorclaw.shared.domain.model.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.withContext
import kotlinx.datetime.Clock
import java.util.concurrent.ConcurrentHashMap

/**
 * Android implementation of VoiceCallManager
 *
 * Note: Full WebRTC implementation requires adding the WebRTC dependency.
 * This implementation provides the structure and basic audio management.
 * WebRTC peer connection setup should be added when the dependency is available.
 *
 * TODO: Add WebRTC dependency and implement full peer connection management
 */
actual class VoiceCallManager {

    private var context: Context? = null
    private var audioManager: AudioManager? = null
    private var isInitialized = false
    private var configuration: CallConfiguration? = null

    private val activeCalls = ConcurrentHashMap<String, CallSession>()
    private val callStateFlow = MutableStateFlow<CallState>(CallState.Idle)
    private val activeCallsFlow = MutableStateFlow<List<CallSession>>(emptyList())
    private val callEventsFlow = MutableSharedFlow<CallEvent>()

    private val callParticipantFlows = ConcurrentHashMap<String, MutableStateFlow<List<CallParticipant>>>()
    private val callStatisticsFlows = ConcurrentHashMap<String, MutableStateFlow<CallStatistics>>()
    private val callStateFlows = ConcurrentHashMap<String, MutableStateFlow<CallState>>()

    /**
     * Initialize with application context
     * Should be called from Android application setup
     */
    fun setContext(appContext: Context) {
        context = appContext
        audioManager = appContext.getSystemService(Context.AUDIO_SERVICE) as? AudioManager
    }

    actual suspend fun initialize(): Result<Unit> = withContext(Dispatchers.Default) {
        initialize(CallConfiguration())
    }

    actual suspend fun initialize(configuration: CallConfiguration): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            val ctx = context
                ?: return@withContext Result.failure(IllegalStateException("Context not set. Call setContext() first."))

            this@VoiceCallManager.configuration = configuration

            // Initialize audio manager for call audio
            audioManager = ctx.getSystemService(Context.AUDIO_SERVICE) as? AudioManager

            // TODO: Initialize WebRTC PeerConnectionFactory when dependency is added
            // PeerConnectionFactory.initialize(
            //     PeerConnectionFactory.InitializationOptions.builder(ctx)
            //         .setEnableInternalTracerCollect(true)
            //         .createInitializationOptions()
            // )

            isInitialized = true
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun initiateCall(
        roomId: String,
        participants: List<String>,
        callType: CallType
    ): Result<CallSession> = withContext(Dispatchers.Default) {
        try {
            if (!isInitialized) {
                return@withContext Result.failure(
                    IllegalStateException("VoiceCallManager not initialized")
                )
            }

            // Check permissions
            val permissions = checkPermissions(callType)
            if (!permissions.hasRequiredPermissions(callType)) {
                return@withContext Result.failure(
                    SecurityException("Missing permissions: ${permissions.getMissingPermissions(callType)}")
                )
            }

            val callId = "call_${System.currentTimeMillis()}"
            val now = Clock.System.now()

            val session = CallSession(
                id = callId,
                roomId = roomId,
                callerId = "local_user", // Should be replaced with actual user ID
                participants = participants.map { userId ->
                    CallParticipant(
                        userId = userId,
                        deviceId = "",
                        isLocal = false
                    )
                },
                state = CallState.Connecting,
                callType = callType,
                isOutgoing = true,
                startedAt = now
            )

            activeCalls[callId] = session
            updateActiveCallsFlow()
            getOrCreateCallStateFlow(callId).value = CallState.Connecting

            // TODO: Implement WebRTC offer creation and signaling
            // val peerConnection = createPeerConnection(configuration?.iceServers)
            // val offer = peerConnection.createOffer()
            // sendSignalingMessage(CallSignaling.Invite(callId, roomId, callType, offer))

            Result.success(session)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun joinCall(callId: String): Result<CallSession> = withContext(Dispatchers.Default) {
        try {
            if (!isInitialized) {
                return@withContext Result.failure(
                    IllegalStateException("VoiceCallManager not initialized")
                )
            }

            val existingCall = activeCalls[callId]
            if (existingCall != null) {
                return@withContext Result.success(existingCall)
            }

            // Create new session for joining
            val session = CallSession(
                id = callId,
                roomId = "",
                callerId = "",
                state = CallState.Connecting,
                isOutgoing = false,
                startedAt = Clock.System.now()
            )

            activeCalls[callId] = session
            updateActiveCallsFlow()
            getOrCreateCallStateFlow(callId).value = CallState.Connecting

            // TODO: Implement WebRTC answer creation

            Result.success(session)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun leaveCall(callId: String, reason: HangupReason): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            val session = activeCalls.remove(callId)
            if (session != null) {
                // Update state to ended
                getOrCreateCallStateFlow(callId).value = CallState.Ended

                // Calculate duration
                val duration = session.startedAt?.let { start ->
                    (Clock.System.now() - start).inWholeSeconds
                } ?: 0L

                // Emit call ended event
                callEventsFlow.emit(
                    CallEvent.CallEnded(
                        session.copy(duration = duration, endedAt = Clock.System.now()),
                        reason
                    )
                )

                // Cleanup flows
                callParticipantFlows.remove(callId)
                callStatisticsFlows.remove(callId)
                callStateFlows.remove(callId)
            }

            updateActiveCallsFlow()
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun answerCall(callId: String): Result<CallSession> = withContext(Dispatchers.Default) {
        try {
            val session = activeCalls[callId]
                ?: return@withContext Result.failure(IllegalArgumentException("Call not found: $callId"))

            if (!session.state.canAnswer()) {
                return@withContext Result.failure(IllegalStateException("Call cannot be answered in state: ${session.state}"))
            }

            val updatedSession = session.copy(state = CallState.Active)
            activeCalls[callId] = updatedSession
            getOrCreateCallStateFlow(callId).value = CallState.Active

            callEventsFlow.emit(CallEvent.CallAnswered(updatedSession))
            updateActiveCallsFlow()

            Result.success(updatedSession)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun rejectCall(callId: String): Result<Unit> = withContext(Dispatchers.Default) {
        leaveCall(callId, HangupReason.USER_BUSY)
    }

    // ========== Audio Controls ==========

    actual suspend fun setMuted(callId: String, muted: Boolean): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            val session = activeCalls[callId]
                ?: return@withContext Result.failure(IllegalArgumentException("Call not found: $callId"))

            val updatedSession = session.copy(isMuted = muted)
            activeCalls[callId] = updatedSession
            updateActiveCallsFlow()

            // TODO: Mute actual audio track
            // localAudioTrack?.setEnabled(!muted)

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun toggleMute(callId: String): Result<Boolean> {
        val session = activeCalls[callId]
            ?: return Result.failure(IllegalArgumentException("Call not found: $callId"))

        val newMuted = !session.isMuted
        return setMuted(callId, newMuted).map { newMuted }
    }

    actual suspend fun setSpeakerPhone(callId: String, enabled: Boolean): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            audioManager?.isSpeakerphoneOn = enabled

            val session = activeCalls[callId]
            if (session != null) {
                activeCalls[callId] = session.copy(isSpeakerOn = enabled)
                updateActiveCallsFlow()
            }

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun toggleSpeakerPhone(callId: String): Result<Boolean> {
        val currentEnabled = audioManager?.isSpeakerphoneOn ?: false
        val newEnabled = !currentEnabled
        return setSpeakerPhone(callId, newEnabled).map { newEnabled }
    }

    actual suspend fun setBluetoothAudio(callId: String, enabled: Boolean): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            audioManager?.let { am ->
                if (enabled) {
                    am.startBluetoothSco()
                    am.isBluetoothScoOn = true
                } else {
                    am.stopBluetoothSco()
                    am.isBluetoothScoOn = false
                }
            }
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ========== Video Controls ==========

    actual suspend fun setVideoEnabled(callId: String, enabled: Boolean): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            val session = activeCalls[callId]
                ?: return@withContext Result.failure(IllegalArgumentException("Call not found: $callId"))

            val updatedSession = session.copy(isLocalVideoEnabled = enabled)
            activeCalls[callId] = updatedSession
            updateActiveCallsFlow()

            // TODO: Enable/disable video track

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    actual suspend fun switchCamera(callId: String): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            // TODO: Implement camera switching with WebRTC
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ========== Call Hold ==========

    actual suspend fun setOnHold(callId: String, onHold: Boolean): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            val session = activeCalls[callId]
                ?: return@withContext Result.failure(IllegalArgumentException("Call not found: $callId"))

            val newState = if (onHold) CallState.OnHold else CallState.Active
            activeCalls[callId] = session.copy(state = newState)
            getOrCreateCallStateFlow(callId).value = newState
            updateActiveCallsFlow()

            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ========== Call State Observation ==========

    actual fun observeActiveCalls(): Flow<List<CallSession>> = activeCallsFlow.asStateFlow()

    actual fun observeCallState(callId: String): Flow<CallState> =
        getOrCreateCallStateFlow(callId).asStateFlow()

    actual fun observeCallParticipants(callId: String): Flow<List<CallParticipant>> =
        getOrCreateParticipantFlow(callId).asStateFlow()

    actual fun observeCallStatistics(callId: String): Flow<CallStatistics> =
        getOrCreateStatisticsFlow(callId).asStateFlow()

    actual fun observeCallEvents(): Flow<CallEvent> = callEventsFlow.asSharedFlow()

    // ========== Audio Device Management ==========

    actual suspend fun getAudioDevices(): List<AudioDevice> = withContext(Dispatchers.Default) {
        val devices = mutableListOf<AudioDevice>()

        audioManager?.let { am ->
            devices.add(AudioDevice.EARPIECE)
            devices.add(AudioDevice.SPEAKER_PHONE)

            if (am.isWiredHeadsetOn) {
                devices.add(AudioDevice.WIRED_HEADSET)
            }

            if (am.isBluetoothScoAvailableOffCall) {
                devices.add(AudioDevice.BLUETOOTH)
            }
        }

        devices
    }

    actual suspend fun getSelectedAudioDevice(): AudioDevice? = withContext(Dispatchers.Default) {
        audioManager?.let { am ->
            when {
                am.isBluetoothScoOn -> AudioDevice.BLUETOOTH
                am.isSpeakerphoneOn -> AudioDevice.SPEAKER_PHONE
                am.isWiredHeadsetOn -> AudioDevice.WIRED_HEADSET
                else -> AudioDevice.EARPIECE
            }
        }
    }

    actual suspend fun selectAudioDevice(device: AudioDevice): Result<Unit> = withContext(Dispatchers.Default) {
        try {
            when (device) {
                AudioDevice.SPEAKER_PHONE -> {
                    audioManager?.isSpeakerphoneOn = true
                    audioManager?.stopBluetoothSco()
                }
                AudioDevice.EARPIECE -> {
                    audioManager?.isSpeakerphoneOn = false
                    audioManager?.stopBluetoothSco()
                }
                AudioDevice.BLUETOOTH -> {
                    audioManager?.startBluetoothSco()
                    audioManager?.isSpeakerphoneOn = false
                }
                AudioDevice.WIRED_HEADSET -> {
                    audioManager?.isSpeakerphoneOn = false
                    audioManager?.stopBluetoothSco()
                }
                AudioDevice.UNKNOWN -> {
                    // No action
                }
            }
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ========== Permissions ==========

    actual suspend fun checkPermissions(callType: CallType): CallPermissions = withContext(Dispatchers.Default) {
        val ctx = context ?: return@withContext CallPermissions()

        val micPermission = ContextCompat.checkSelfPermission(ctx, Manifest.permission.RECORD_AUDIO)
        val cameraPermission = ContextCompat.checkSelfPermission(ctx, Manifest.permission.CAMERA)
        val bluetoothPermission = ContextCompat.checkSelfPermission(
            ctx,
            Manifest.permission.BLUETOOTH_CONNECT
        )

        CallPermissions(
            microphone = if (micPermission == PackageManager.PERMISSION_GRANTED)
                PermissionState.GRANTED else PermissionState.DENIED,
            camera = if (cameraPermission == PackageManager.PERMISSION_GRANTED)
                PermissionState.GRANTED else PermissionState.DENIED,
            bluetooth = if (bluetoothPermission == PackageManager.PERMISSION_GRANTED)
                PermissionState.GRANTED else PermissionState.NOT_DETERMINED
        )
    }

    actual suspend fun requestPermissions(callType: CallType): Result<CallPermissions> {
        // Note: Actual permission request needs to be done from an Activity
        // This returns the current state; Activity should handle the request
        return Result.success(checkPermissions(callType))
    }

    // ========== Cleanup ==========

    actual fun release() {
        activeCalls.clear()
        callParticipantFlows.clear()
        callStatisticsFlows.clear()
        callStateFlows.clear()
        isInitialized = false
    }

    // ========== Helper Methods ==========

    private fun updateActiveCallsFlow() {
        activeCallsFlow.value = activeCalls.values.toList()
    }

    private fun getOrCreateCallStateFlow(callId: String): MutableStateFlow<CallState> =
        callStateFlows.getOrPut(callId) { MutableStateFlow(CallState.Idle) }

    private fun getOrCreateParticipantFlow(callId: String): MutableStateFlow<List<CallParticipant>> =
        callParticipantFlows.getOrPut(callId) { MutableStateFlow(emptyList()) }

    private fun getOrCreateStatisticsFlow(callId: String): MutableStateFlow<CallStatistics> =
        callStatisticsFlows.getOrPut(callId) {
            MutableStateFlow(
                CallStatistics(
                    callId = callId,
                    timestamp = Clock.System.now()
                )
            )
        }
}
