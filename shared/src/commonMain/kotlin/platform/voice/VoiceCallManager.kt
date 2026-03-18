package com.armorclaw.shared.platform.voice

import com.armorclaw.shared.domain.model.*
import kotlinx.coroutines.flow.Flow

/**
 * Platform interface for voice call management
 * Uses expect/actual pattern for Android/iOS implementations
 *
 * Supports MatrixRTC signaling (MSC3077) for call setup
 */
expect class VoiceCallManager() {

    /**
     * Initialize the voice call manager
     * Sets up WebRTC peer connection factory and audio devices
     * @return Result indicating success or error
     */
    suspend fun initialize(): Result<Unit>

    /**
     * Initialize with custom configuration
     * @param configuration Call configuration including ICE servers
     */
    suspend fun initialize(configuration: CallConfiguration): Result<Unit>

    /**
     * Initiate a new voice call
     * @param roomId The room to start the call in
     * @param participants List of user IDs to invite
     * @param callType Type of call (voice or video)
     * @return Result containing the CallSession or error
     */
    suspend fun initiateCall(
        roomId: String,
        participants: List<String>,
        callType: CallType = CallType.VOICE
    ): Result<CallSession>

    /**
     * Join an existing call
     * @param callId The call ID to join
     * @return Result containing the CallSession or error
     */
    suspend fun joinCall(callId: String): Result<CallSession>

    /**
     * Leave the current call
     * @param callId The call ID to leave
     * @param reason Optional reason for leaving
     * @return Result indicating success or error
     */
    suspend fun leaveCall(
        callId: String,
        reason: HangupReason = HangupReason.USER_HANGUP
    ): Result<Unit>

    /**
     * Answer an incoming call
     * @param callId The call ID to answer
     * @return Result containing the CallSession or error
     */
    suspend fun answerCall(callId: String): Result<CallSession>

    /**
     * Reject an incoming call
     * @param callId The call ID to reject
     * @return Result indicating success or error
     */
    suspend fun rejectCall(callId: String): Result<Unit>

    // ========== Audio Controls ==========

    /**
     * Set mute state for the local audio track
     * @param callId The call ID
     * @param muted Whether to mute the microphone
     * @return Result indicating success or error
     */
    suspend fun setMuted(callId: String, muted: Boolean): Result<Unit>

    /**
     * Toggle mute state
     * @param callId The call ID
     * @return Result containing the new mute state
     */
    suspend fun toggleMute(callId: String): Result<Boolean>

    /**
     * Set speaker phone state
     * @param callId The call ID
     * @param enabled Whether to enable speaker phone
     * @return Result indicating success or error
     */
    suspend fun setSpeakerPhone(callId: String, enabled: Boolean): Result<Unit>

    /**
     * Toggle speaker phone
     * @param callId The call ID
     * @return Result containing the new speaker state
     */
    suspend fun toggleSpeakerPhone(callId: String): Result<Boolean>

    /**
     * Set Bluetooth audio device state
     * @param callId The call ID
     * @param enabled Whether to use Bluetooth audio
     * @return Result indicating success or error
     */
    suspend fun setBluetoothAudio(callId: String, enabled: Boolean): Result<Unit>

    // ========== Video Controls ==========

    /**
     * Enable/disable local video track
     * @param callId The call ID
     * @param enabled Whether to enable video
     * @return Result indicating success or error
     */
    suspend fun setVideoEnabled(callId: String, enabled: Boolean): Result<Unit>

    /**
     * Switch between front and back camera
     * @param callId The call ID
     * @return Result indicating success or error
     */
    suspend fun switchCamera(callId: String): Result<Unit>

    // ========== Call Hold ==========

    /**
     * Put call on hold
     * @param callId The call ID
     * @param onHold Whether to hold or resume
     * @return Result indicating success or error
     */
    suspend fun setOnHold(callId: String, onHold: Boolean): Result<Unit>

    // ========== Call State Observation ==========

    /**
     * Observe all active calls
     * @return Flow emitting list of active call sessions
     */
    fun observeActiveCalls(): Flow<List<CallSession>>

    /**
     * Observe state changes for a specific call
     * @param callId The call ID to observe
     * @return Flow emitting call state updates
     */
    fun observeCallState(callId: String): Flow<CallState>

    /**
     * Observe participants in a call
     * @param callId The call ID
     * @return Flow emitting list of call participants
     */
    fun observeCallParticipants(callId: String): Flow<List<CallParticipant>>

    /**
     * Observe call statistics
     * @param callId The call ID
     * @return Flow emitting call statistics updates
     */
    fun observeCallStatistics(callId: String): Flow<CallStatistics>

    /**
     * Observe call events
     * @return Flow emitting call events
     */
    fun observeCallEvents(): Flow<CallEvent>

    // ========== Audio Device Management ==========

    /**
     * Get available audio devices
     * @return List of available audio devices
     */
    suspend fun getAudioDevices(): List<AudioDevice>

    /**
     * Get currently selected audio device
     * @return The selected audio device or null
     */
    suspend fun getSelectedAudioDevice(): AudioDevice?

    /**
     * Select an audio device
     * @param device The audio device to select
     * @return Result indicating success or error
     */
    suspend fun selectAudioDevice(device: AudioDevice): Result<Unit>

    // ========== Permissions ==========

    /**
     * Check call permissions
     * @param callType The type of call to check permissions for
     * @return Current permission states
     */
    suspend fun checkPermissions(callType: CallType): CallPermissions

    /**
     * Request call permissions
     * @param callType The type of call requesting permissions
     * @return Result containing updated permission states
     */
    suspend fun requestPermissions(callType: CallType): Result<CallPermissions>

    // ========== Cleanup ==========

    /**
     * Release all resources
     * Should be called when the manager is no longer needed
     */
    fun release()
}

/**
 * Available audio device types
 */
enum class AudioDevice {
    EARPIECE,
    SPEAKER_PHONE,
    WIRED_HEADSET,
    BLUETOOTH,
    UNKNOWN
}

/**
 * Voice call manager factory
 * Creates platform-specific implementation
 */
object VoiceCallManagerFactory {
    /**
     * Create a new VoiceCallManager instance
     */
    fun create(): VoiceCallManager = VoiceCallManager()
}
