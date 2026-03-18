package com.armorclaw.shared.domain.repository

import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.platform.voice.AudioDevice
import kotlinx.coroutines.flow.Flow

/**
 * Repository for voice/video call operations
 * Manages call sessions, signaling, and state
 */
interface CallRepository {

    // ========== Call Initiation ==========

    /**
     * Initiate a new voice call in a room
     * @param roomId The room to start the call in
     * @param callType Type of call (voice or video)
     * @return Result containing the created call session
     */
    suspend fun initiateCall(
        roomId: String,
        callType: CallType = CallType.VOICE
    ): Result<CallSession>

    /**
     * Initiate a direct call with a specific user
     * @param userId The user to call
     * @param callType Type of call
     * @return Result containing the created call session
     */
    suspend fun initiateDirectCall(
        userId: String,
        callType: CallType = CallType.VOICE
    ): Result<CallSession>

    // ========== Call Participation ==========

    /**
     * Join an existing call
     * @param callId The call ID to join
     * @return Result containing the call session
     */
    suspend fun joinCall(callId: String): Result<CallSession>

    /**
     * Leave a call
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
     * @return Result containing the call session
     */
    suspend fun answerCall(callId: String): Result<CallSession>

    /**
     * Reject an incoming call
     * @param callId The call ID to reject
     * @return Result indicating success or error
     */
    suspend fun rejectCall(callId: String): Result<Unit>

    // ========== Call State ==========

    /**
     * Get current call session
     * @param callId The call ID
     * @return Result containing the call session or null
     */
    suspend fun getCallSession(callId: String): Result<CallSession?>

    /**
     * Get all active calls
     * @return Result containing list of active call sessions
     */
    suspend fun getActiveCalls(): Result<List<CallSession>>

    /**
     * Get the current active call (if any)
     * @return Result containing the active call or null
     */
    suspend fun getCurrentActiveCall(): Result<CallSession?>

    /**
     * Check if there's an active call in a room
     * @param roomId The room ID to check
     * @return Result containing boolean
     */
    suspend fun hasActiveCallInRoom(roomId: String): Result<Boolean>

    // ========== Call Observation ==========

    /**
     * Observe all active calls
     * @return Flow emitting list of active call sessions
     */
    fun observeActiveCalls(): Flow<List<CallSession>>

    /**
     * Observe a specific call session
     * @param callId The call ID to observe
     * @return Flow emitting call session updates
     */
    fun observeCallSession(callId: String): Flow<CallSession?>

    /**
     * Observe call state changes
     * @param callId The call ID
     * @return Flow emitting call state updates
     */
    fun observeCallState(callId: String): Flow<CallState>

    /**
     * Observe call participants
     * @param callId The call ID
     * @return Flow emitting list of participants
     */
    fun observeCallParticipants(callId: String): Flow<List<CallParticipant>>

    /**
     * Observe call statistics
     * @param callId The call ID
     * @return Flow emitting call statistics
     */
    fun observeCallStatistics(callId: String): Flow<CallStatistics>

    /**
     * Observe call events
     * @return Flow emitting call events
     */
    fun observeCallEvents(): Flow<CallEvent>

    /**
     * Observe incoming calls
     * @return Flow emitting incoming call sessions
     */
    fun observeIncomingCalls(): Flow<CallSession>

    // ========== Call Controls ==========

    /**
     * Set mute state
     * @param callId The call ID
     * @param muted Whether to mute
     * @return Result indicating success or error
     */
    suspend fun setMuted(callId: String, muted: Boolean): Result<Unit>

    /**
     * Set speaker phone state
     * @param callId The call ID
     * @param enabled Whether to enable speaker phone
     * @return Result indicating success or error
     */
    suspend fun setSpeakerPhone(callId: String, enabled: Boolean): Result<Unit>

    /**
     * Set video enabled state
     * @param callId The call ID
     * @param enabled Whether to enable video
     * @return Result indicating success or error
     */
    suspend fun setVideoEnabled(callId: String, enabled: Boolean): Result<Unit>

    /**
     * Switch camera (front/back)
     * @param callId The call ID
     * @return Result indicating success or error
     */
    suspend fun switchCamera(callId: String): Result<Unit>

    /**
     * Put call on hold
     * @param callId The call ID
     * @param onHold Whether to hold or resume
     * @return Result indicating success or error
     */
    suspend fun setOnHold(callId: String, onHold: Boolean): Result<Unit>

    // ========== Audio Devices ==========

    /**
     * Get available audio devices
     * @return Result containing list of audio devices
     */
    suspend fun getAudioDevices(): Result<List<AudioDevice>>

    /**
     * Get currently selected audio device
     * @return Result containing the selected device or null
     */
    suspend fun getSelectedAudioDevice(): Result<AudioDevice?>

    /**
     * Select an audio device
     * @param callId The call ID (for context)
     * @param device The audio device to select
     * @return Result indicating success or error
     */
    suspend fun selectAudioDevice(callId: String, device: AudioDevice): Result<Unit>

    // ========== Permissions ==========

    /**
     * Check call permissions
     * @param callType The type of call
     * @return Result containing permission states
     */
    suspend fun checkPermissions(callType: CallType): Result<CallPermissions>

    /**
     * Request call permissions
     * @param callType The type of call
     * @return Result containing updated permission states
     */
    suspend fun requestPermissions(callType: CallType): Result<CallPermissions>

    // ========== Call History ==========

    /**
     * Get call history for a room
     * @param roomId The room ID
     * @param limit Maximum number of calls to return
     * @return Result containing list of past call sessions
     */
    suspend fun getCallHistory(roomId: String, limit: Int = 50): Result<List<CallSession>>

    /**
     * Get recent calls across all rooms
     * @param limit Maximum number of calls to return
     * @return Result containing list of recent call sessions
     */
    suspend fun getRecentCalls(limit: Int = 50): Result<List<CallSession>>

    // ========== Signaling (Internal) ==========

    /**
     * Handle incoming call signaling message
     * @param signaling The signaling message
     * @return Result indicating success or error
     */
    suspend fun handleSignaling(signaling: CallSignaling): Result<Unit>

    /**
     * Send signaling message
     * @param roomId The room to send to
     * @param signaling The signaling message
     * @return Result indicating success or error
     */
    suspend fun sendSignaling(roomId: String, signaling: CallSignaling): Result<Unit>
}

/**
 * Call-related operation types for offline sync
 */
enum class CallOperationType {
    INITIATE_CALL,
    JOIN_CALL,
    LEAVE_CALL,
    ANSWER_CALL,
    REJECT_CALL,
    SET_MUTED,
    SET_SPEAKER_PHONE,
    SET_VIDEO_ENABLED,
    SET_ON_HOLD
}

/**
 * Exception for call-related errors
 */
class CallException(
    val errorCode: ArmorClawErrorCode,
    message: String,
    cause: Throwable? = null
) : Exception(message, cause)
