package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable
import kotlinx.datetime.Instant

// ========== Call Session ==========

/**
 * Active call session information
 */
@Serializable
data class CallSession(
    val id: String,
    val roomId: String,
    val callerId: String,
    val callerName: String? = null,
    val callerAvatar: String? = null,
    val participants: List<CallParticipant> = emptyList(),
    val state: CallState,
    val callType: CallType = CallType.VOICE,
    val isMuted: Boolean = false,
    val isSpeakerOn: Boolean = false,
    val isLocalVideoEnabled: Boolean = false,
    val startedAt: Instant? = null,
    val endedAt: Instant? = null,
    val duration: Long = 0, // Duration in seconds
    val isOutgoing: Boolean = false
) {
    /**
     * Check if call is active (connecting, ringing, or in progress)
     */
    fun isActive(): Boolean = state.isActive()

    /**
     * Get call duration as formatted string
     */
    fun getFormattedDuration(): String {
        if (duration <= 0) return "0:00"
        val minutes = duration / 60
        val seconds = duration % 60
        return "${minutes}:${seconds.toString().padStart(2, '0')}"
    }
}

/**
 * Call state machine
 */
@Serializable
sealed class CallState {
    @Serializable
    object Idle : CallState()

    @Serializable
    object Connecting : CallState()

    @Serializable
    object Ringing : CallState()

    @Serializable
    object Active : CallState()

    @Serializable
    object OnHold : CallState()

    @Serializable
    data class Error(
        val code: ArmorClawErrorCode,
        val message: String
    ) : CallState()

    @Serializable
    object Ended : CallState()

    fun isActive(): Boolean = this is Connecting || this is Ringing || this is Active || this is OnHold

    fun canAnswer(): Boolean = this is Ringing

    fun canEnd(): Boolean = this is Connecting || this is Ringing || this is Active || this is OnHold

    fun isTerminated(): Boolean = this is Ended || this is Error
}

/**
 * Call participant information
 */
@Serializable
data class CallParticipant(
    val userId: String,
    val deviceId: String,
    val displayName: String? = null,
    val avatar: String? = null,
    val isMuted: Boolean = false,
    val isVideoEnabled: Boolean = false,
    val isSpeaking: Boolean = false,
    val audioLevel: Float = 0f, // 0.0 to 1.0
    val connectionState: ConnectionQuality = ConnectionQuality.UNKNOWN,
    val isLocal: Boolean = false
)

/**
 * Connection quality indicator
 */
@Serializable
enum class ConnectionQuality {
    EXCELLENT,  // Green - Full quality
    GOOD,       // Yellow - Minor issues
    POOR,       // Orange - Significant issues
    BAD,        // Red - Severe issues
    UNKNOWN;    // Gray - Not connected

    fun getColor(): String = when (this) {
        EXCELLENT -> "#10B981" // Green
        GOOD -> "#F59E0B"      // Yellow
        POOR -> "#F97316"      // Orange
        BAD -> "#EF4444"       // Red
        UNKNOWN -> "#6B7280"   // Gray
    }
}

/**
 * Type of call
 */
@Serializable
enum class CallType {
    VOICE,
    VIDEO
}

// ========== Call Signaling ==========

/**
 * Call signaling message types (MatrixRTC MSC3077)
 */
@Serializable
sealed class CallSignaling {
    @Serializable
    data class Invite(
        val callId: String,
        val roomId: String,
        val callType: CallType,
        val offer: SessionDescription
    ) : CallSignaling()

    @Serializable
    data class Answer(
        val callId: String,
        val answer: SessionDescription
    ) : CallSignaling()

    @Serializable
    data class IceCandidate(
        val callId: String,
        val candidate: IceCandidateData
    ) : CallSignaling()

    @Serializable
    data class Hangup(
        val callId: String,
        val reason: HangupReason = HangupReason.USER_HANGUP
    ) : CallSignaling()

    @Serializable
    data class Negotiate(
        val callId: String,
        val description: SessionDescription
    ) : CallSignaling()
}

/**
 * SDP session description
 */
@Serializable
data class SessionDescription(
    val type: SdpType,
    val sdp: String
)

@Serializable
enum class SdpType {
    OFFER,
    ANSWER,
    PR_ANSWER,
    ROLLBACK
}

/**
 * ICE candidate data
 */
@Serializable
data class IceCandidateData(
    val candidate: String,
    val sdpMid: String?,
    val sdpMLineIndex: Int?
)

/**
 * Reason for call hangup
 */
@Serializable
enum class HangupReason {
    USER_HANGUP,
    USER_MEDIA_FAILED,
    ICE_FAILED,
    INVITE_TIMEOUT,
    USER_BUSY,
    UNKNOWN_ERROR
}

// ========== Call Statistics ==========

/**
 * Call quality statistics
 */
@Serializable
data class CallStatistics(
    val callId: String,
    val bytesSent: Long = 0,
    val bytesReceived: Long = 0,
    val packetsSent: Long = 0,
    val packetsReceived: Long = 0,
    val packetsLost: Long = 0,
    val jitter: Double = 0.0,        // In seconds
    val roundTripTime: Double = 0.0, // In seconds
    val audioLevel: Float = 0f,
    val bitrate: Long = 0,           // In bits per second
    val timestamp: Instant
) {
    /**
     * Calculate packet loss percentage
     */
    fun getPacketLossPercentage(): Double {
        val total = packetsSent + packetsReceived
        if (total == 0L) return 0.0
        return (packetsLost.toDouble() / total) * 100
    }

    /**
     * Get quality rating based on statistics
     */
    fun getQualityRating(): ConnectionQuality {
        val packetLoss = getPacketLossPercentage()
        val rtt = roundTripTime * 1000 // Convert to ms

        return when {
            packetLoss < 1 && rtt < 150 -> ConnectionQuality.EXCELLENT
            packetLoss < 3 && rtt < 300 -> ConnectionQuality.GOOD
            packetLoss < 10 && rtt < 500 -> ConnectionQuality.POOR
            else -> ConnectionQuality.BAD
        }
    }
}

// ========== Call Configuration ==========

/**
 * Call configuration settings
 */
@Serializable
data class CallConfiguration(
    val iceServers: List<IceServer> = emptyList(),
    val audioEnabled: Boolean = true,
    val videoEnabled: Boolean = false,
    val bandwidthLimit: BandwidthLimit? = null
)

/**
 * ICE server configuration
 */
@Serializable
data class IceServer(
    val urls: List<String>,
    val username: String? = null,
    val credential: String? = null,
    val credentialType: CredentialType = CredentialType.PASSWORD
)

@Serializable
enum class CredentialType {
    PASSWORD,
    OAUTH
}

/**
 * Bandwidth limit configuration
 */
@Serializable
data class BandwidthLimit(
    val audioBitrate: Int? = null,  // kbps
    val videoBitrate: Int? = null,  // kbps
    val totalBitrate: Int? = null   // kbps
)

// ========== Call Events ==========

/**
 * Call-related events for UI updates
 */
@Serializable
sealed class CallEvent {
    @Serializable
    data class IncomingCall(val session: CallSession) : CallEvent()

    @Serializable
    data class CallAnswered(val session: CallSession) : CallEvent()

    @Serializable
    data class CallEnded(val session: CallSession, val reason: HangupReason) : CallEvent()

    @Serializable
    data class ParticipantJoined(val participant: CallParticipant) : CallEvent()

    @Serializable
    data class ParticipantLeft(val userId: String) : CallEvent()

    @Serializable
    data class ParticipantMuted(val userId: String, val muted: Boolean) : CallEvent()

    @Serializable
    data class ParticipantSpeaking(val userId: String, val speaking: Boolean) : CallEvent()

    @Serializable
    data class CallError(val error: ArmorClawErrorCode, val message: String) : CallEvent()

    @Serializable
    data class QualityChanged(val quality: ConnectionQuality) : CallEvent()
}

// ========== Call Permissions ==========

/**
 * Call permission states
 */
@Serializable
enum class CallPermission {
    MICROPHONE,
    CAMERA,
    BLUETOOTH,
    SPEAKER_PHONE
}

@Serializable
data class CallPermissions(
    val microphone: PermissionState = PermissionState.NOT_DETERMINED,
    val camera: PermissionState = PermissionState.NOT_DETERMINED,
    val bluetooth: PermissionState = PermissionState.NOT_DETERMINED
) {
    fun hasRequiredPermissions(callType: CallType): Boolean {
        return when (callType) {
            CallType.VOICE -> microphone == PermissionState.GRANTED
            CallType.VIDEO -> microphone == PermissionState.GRANTED &&
                              camera == PermissionState.GRANTED
        }
    }

    fun getMissingPermissions(callType: CallType): List<CallPermission> {
        val missing = mutableListOf<CallPermission>()
        if (microphone != PermissionState.GRANTED) missing.add(CallPermission.MICROPHONE)
        if (callType == CallType.VIDEO && camera != PermissionState.GRANTED) {
            missing.add(CallPermission.CAMERA)
        }
        return missing
    }
}

@Serializable
enum class PermissionState {
    NOT_DETERMINED,
    GRANTED,
    DENIED,
    PERMANENTLY_DENIED
}
