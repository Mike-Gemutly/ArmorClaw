package com.armorclaw.shared.domain.model

import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

class CallModelTest {

    // ========== CallSession Tests ==========

    @Test
    fun testCallSessionDefaults() {
        val session = CallSession(
            id = "call-123",
            roomId = "room-456",
            callerId = "user-789",
            state = CallState.Idle
        )

        assertEquals("call-123", session.id)
        assertEquals("room-456", session.roomId)
        assertEquals("user-789", session.callerId)
        assertEquals(CallState.Idle, session.state)
        assertEquals(CallType.VOICE, session.callType)
        assertFalse(session.isMuted)
        assertFalse(session.isSpeakerOn)
        assertFalse(session.isOutgoing)
        assertTrue(session.participants.isEmpty())
    }

    @Test
    fun testCallSessionCopy() {
        val original = CallSession(
            id = "call-123",
            roomId = "room-456",
            callerId = "user-789",
            state = CallState.Active,
            isMuted = true
        )

        val copy = original.copy(
            state = CallState.OnHold,
            isMuted = false
        )

        assertEquals("call-123", copy.id)
        assertEquals(CallState.OnHold, copy.state)
        assertFalse(copy.isMuted)
    }

    @Test
    fun testCallSessionIsActive() {
        val activeSession = CallSession(
            id = "call-123",
            roomId = "room-456",
            callerId = "user-789",
            state = CallState.Active
        )
        assertTrue(activeSession.isActive())

        val idleSession = CallSession(
            id = "call-123",
            roomId = "room-456",
            callerId = "user-789",
            state = CallState.Idle
        )
        assertFalse(idleSession.isActive())
    }

    @Test
    fun testCallSessionGetFormattedDuration() {
        val session1 = CallSession(
            id = "call-123",
            roomId = "room-456",
            callerId = "user-789",
            state = CallState.Idle,
            duration = 0
        )
        assertEquals("0:00", session1.getFormattedDuration())

        val session2 = CallSession(
            id = "call-123",
            roomId = "room-456",
            callerId = "user-789",
            state = CallState.Idle,
            duration = 65
        )
        assertEquals("1:05", session2.getFormattedDuration())

        val session3 = CallSession(
            id = "call-123",
            roomId = "room-456",
            callerId = "user-789",
            state = CallState.Idle,
            duration = 125
        )
        assertEquals("2:05", session3.getFormattedDuration())
    }

    // ========== CallState Tests ==========

    @Test
    fun testCallStateCanAnswer() {
        assertTrue(CallState.Ringing.canAnswer())
        assertFalse(CallState.Connecting.canAnswer())
        assertFalse(CallState.Idle.canAnswer())
        assertFalse(CallState.Active.canAnswer())
        assertFalse(CallState.OnHold.canAnswer())
        assertFalse(CallState.Ended.canAnswer())
    }

    @Test
    fun testCallStateIsActive() {
        assertTrue(CallState.Active.isActive())
        assertTrue(CallState.Connecting.isActive())
        assertTrue(CallState.Ringing.isActive())
        assertTrue(CallState.OnHold.isActive())
        assertFalse(CallState.Idle.isActive())
        assertFalse(CallState.Ended.isActive())
    }

    @Test
    fun testCallStateCanEnd() {
        assertTrue(CallState.Active.canEnd())
        assertTrue(CallState.Connecting.canEnd())
        assertTrue(CallState.Ringing.canEnd())
        assertTrue(CallState.OnHold.canEnd())
        assertFalse(CallState.Idle.canEnd())
        assertFalse(CallState.Ended.canEnd())
    }

    @Test
    fun testCallStateIsTerminated() {
        assertTrue(CallState.Ended.isTerminated())
        val errorState = CallState.Error(
            code = ArmorClawErrorCode.VOICE_MIC_DENIED,
            message = "Error"
        )
        assertTrue(errorState.isTerminated())
        assertFalse(CallState.Active.isTerminated())
        assertFalse(CallState.Idle.isTerminated())
    }

    @Test
    fun testCallStateErrorProperties() {
        val errorState = CallState.Error(
            code = ArmorClawErrorCode.VOICE_MIC_DENIED,
            message = "Microphone permission denied"
        )

        assertEquals(ArmorClawErrorCode.VOICE_MIC_DENIED, errorState.code)
        assertEquals("Microphone permission denied", errorState.message)
    }

    // ========== CallParticipant Tests ==========

    @Test
    fun testCallParticipantDefaults() {
        val participant = CallParticipant(
            userId = "user-123",
            deviceId = "device-456",
            isLocal = true
        )

        assertEquals("user-123", participant.userId)
        assertEquals("device-456", participant.deviceId)
        assertTrue(participant.isLocal)
        assertFalse(participant.isMuted)
        assertFalse(participant.isSpeaking)
        assertFalse(participant.isVideoEnabled)
        assertNull(participant.displayName)
        assertNull(participant.avatar)
        assertEquals(ConnectionQuality.UNKNOWN, participant.connectionState)
        assertEquals(0f, participant.audioLevel)
    }

    @Test
    fun testCallParticipantWithDisplayName() {
        val participant = CallParticipant(
            userId = "user-123",
            deviceId = "device-456",
            isLocal = false,
            displayName = "Alice",
            avatar = "https://example.com/avatar.png"
        )

        assertEquals("Alice", participant.displayName)
        assertEquals("https://example.com/avatar.png", participant.avatar)
    }

    // ========== ConnectionQuality Tests ==========

    @Test
    fun testConnectionQualityColors() {
        assertEquals("#10B981", ConnectionQuality.EXCELLENT.getColor())
        assertEquals("#F59E0B", ConnectionQuality.GOOD.getColor())
        assertEquals("#F97316", ConnectionQuality.POOR.getColor())
        assertEquals("#EF4444", ConnectionQuality.BAD.getColor())
        assertEquals("#6B7280", ConnectionQuality.UNKNOWN.getColor())
    }

    @Test
    fun testConnectionQualityValues() {
        assertEquals(5, ConnectionQuality.values().size)
    }

    // ========== CallType Tests ==========

    @Test
    fun testCallTypeValues() {
        assertEquals(2, CallType.values().size)
        assertEquals(CallType.VOICE, CallType.valueOf("VOICE"))
        assertEquals(CallType.VIDEO, CallType.valueOf("VIDEO"))
    }

    // ========== HangupReason Tests ==========

    @Test
    fun testHangupReasonValues() {
        assertEquals(6, HangupReason.values().size)
        assertEquals(HangupReason.USER_HANGUP, HangupReason.valueOf("USER_HANGUP"))
        assertEquals(HangupReason.USER_BUSY, HangupReason.valueOf("USER_BUSY"))
        assertEquals(HangupReason.ICE_FAILED, HangupReason.valueOf("ICE_FAILED"))
        assertEquals(HangupReason.INVITE_TIMEOUT, HangupReason.valueOf("INVITE_TIMEOUT"))
        assertEquals(HangupReason.USER_MEDIA_FAILED, HangupReason.valueOf("USER_MEDIA_FAILED"))
        assertEquals(HangupReason.UNKNOWN_ERROR, HangupReason.valueOf("UNKNOWN_ERROR"))
    }

    // ========== CallStatistics Tests ==========

    @Test
    fun testCallStatisticsDefaults() {
        val now = Clock.System.now()
        val stats = CallStatistics(
            callId = "call-123",
            timestamp = now
        )

        assertEquals("call-123", stats.callId)
        assertEquals(0L, stats.bytesSent)
        assertEquals(0L, stats.bytesReceived)
        assertEquals(0L, stats.packetsSent)
        assertEquals(0L, stats.packetsReceived)
        assertEquals(0L, stats.packetsLost)
        assertEquals(0.0, stats.jitter)
        assertEquals(0.0, stats.roundTripTime)
        assertEquals(0f, stats.audioLevel)
        assertEquals(0L, stats.bitrate)
        assertEquals(now, stats.timestamp)
    }

    @Test
    fun testCallStatisticsGetPacketLossPercentage() {
        val stats = CallStatistics(
            callId = "call-123",
            packetsSent = 100,
            packetsReceived = 100,
            packetsLost = 5,
            timestamp = Clock.System.now()
        )

        // 5 / (100 + 100) * 100 = 2.5
        assertEquals(2.5, stats.getPacketLossPercentage(), 0.01)
    }

    @Test
    fun testCallStatisticsGetPacketLossPercentageZeroTotal() {
        val stats = CallStatistics(
            callId = "call-123",
            timestamp = Clock.System.now()
        )

        assertEquals(0.0, stats.getPacketLossPercentage())
    }

    @Test
    fun testCallStatisticsGetQualityRating() {
        // Excellent: packet loss < 1%, RTT < 150ms
        val excellent = CallStatistics(
            callId = "call-123",
            packetsSent = 1000,
            packetsLost = 5, // 0.25%
            roundTripTime = 0.1, // 100ms
            timestamp = Clock.System.now()
        )
        assertEquals(ConnectionQuality.EXCELLENT, excellent.getQualityRating())

        // Good: packet loss < 3%, RTT < 300ms
        val good = CallStatistics(
            callId = "call-123",
            packetsSent = 1000,
            packetsLost = 20, // 1%
            roundTripTime = 0.2, // 200ms
            timestamp = Clock.System.now()
        )
        assertEquals(ConnectionQuality.GOOD, good.getQualityRating())

        // Poor: packet loss < 10%, RTT < 500ms
        val poor = CallStatistics(
            callId = "call-123",
            packetsSent = 1000,
            packetsLost = 50, // 2.5%
            roundTripTime = 0.4, // 400ms
            timestamp = Clock.System.now()
        )
        assertEquals(ConnectionQuality.POOR, poor.getQualityRating())

        // Bad: everything else
        val bad = CallStatistics(
            callId = "call-123",
            packetsSent = 1000,
            packetsLost = 150, // 7.5%
            roundTripTime = 0.6, // 600ms
            timestamp = Clock.System.now()
        )
        assertEquals(ConnectionQuality.BAD, bad.getQualityRating())
    }

    // ========== CallPermissions Tests ==========

    @Test
    fun testCallPermissionsDefaults() {
        val permissions = CallPermissions()

        assertEquals(PermissionState.NOT_DETERMINED, permissions.microphone)
        assertEquals(PermissionState.NOT_DETERMINED, permissions.camera)
        assertEquals(PermissionState.NOT_DETERMINED, permissions.bluetooth)
    }

    @Test
    fun testCallPermissionsHasRequiredForVoice() {
        val granted = CallPermissions(
            microphone = PermissionState.GRANTED
        )
        assertTrue(granted.hasRequiredPermissions(CallType.VOICE))

        val denied = CallPermissions(
            microphone = PermissionState.DENIED
        )
        assertFalse(denied.hasRequiredPermissions(CallType.VOICE))
    }

    @Test
    fun testCallPermissionsHasRequiredForVideo() {
        val granted = CallPermissions(
            microphone = PermissionState.GRANTED,
            camera = PermissionState.GRANTED
        )
        assertTrue(granted.hasRequiredPermissions(CallType.VIDEO))

        val missingCamera = CallPermissions(
            microphone = PermissionState.GRANTED,
            camera = PermissionState.DENIED
        )
        assertFalse(missingCamera.hasRequiredPermissions(CallType.VIDEO))
    }

    @Test
    fun testCallPermissionsGetMissingForVoice() {
        val partial = CallPermissions(
            microphone = PermissionState.DENIED
        )

        val missing = partial.getMissingPermissions(CallType.VOICE)
        assertEquals(1, missing.size)
        assertTrue(missing.contains(CallPermission.MICROPHONE))
    }

    @Test
    fun testCallPermissionsGetMissingForVideo() {
        val partial = CallPermissions(
            microphone = PermissionState.GRANTED,
            camera = PermissionState.DENIED
        )

        val missing = partial.getMissingPermissions(CallType.VIDEO)
        assertEquals(1, missing.size)
        assertTrue(missing.contains(CallPermission.CAMERA))
    }

    @Test
    fun testCallPermissionsGetMissingNone() {
        val allGranted = CallPermissions(
            microphone = PermissionState.GRANTED,
            camera = PermissionState.GRANTED
        )

        val missing = allGranted.getMissingPermissions(CallType.VIDEO)
        assertTrue(missing.isEmpty())
    }

    // ========== PermissionState Tests ==========

    @Test
    fun testPermissionStateValues() {
        assertEquals(4, PermissionState.values().size)
        assertEquals(PermissionState.GRANTED, PermissionState.valueOf("GRANTED"))
        assertEquals(PermissionState.DENIED, PermissionState.valueOf("DENIED"))
        assertEquals(PermissionState.NOT_DETERMINED, PermissionState.valueOf("NOT_DETERMINED"))
        assertEquals(PermissionState.PERMANENTLY_DENIED, PermissionState.valueOf("PERMANENTLY_DENIED"))
    }

    // ========== SessionDescription Tests ==========

    @Test
    fun testSessionDescription() {
        val sd = SessionDescription(
            type = SdpType.OFFER,
            sdp = "v=0..."
        )

        assertEquals(SdpType.OFFER, sd.type)
        assertEquals("v=0...", sd.sdp)
    }

    // ========== IceCandidateData Tests ==========

    @Test
    fun testIceCandidateData() {
        val candidate = IceCandidateData(
            candidate = "candidate:...",
            sdpMid = "audio",
            sdpMLineIndex = 0
        )

        assertEquals("candidate:...", candidate.candidate)
        assertEquals("audio", candidate.sdpMid)
        assertEquals(0, candidate.sdpMLineIndex)
    }

    // ========== CallConfiguration Tests ==========

    @Test
    fun testCallConfigurationDefaults() {
        val config = CallConfiguration()

        assertTrue(config.iceServers.isEmpty())
        assertTrue(config.audioEnabled)
        assertFalse(config.videoEnabled)
        assertNull(config.bandwidthLimit)
    }

    // ========== IceServer Tests ==========

    @Test
    fun testIceServer() {
        val server = IceServer(
            urls = listOf("stun:stun.example.com"),
            username = "user",
            credential = "pass"
        )

        assertEquals(1, server.urls.size)
        assertEquals("stun:stun.example.com", server.urls[0])
        assertEquals("user", server.username)
        assertEquals("pass", server.credential)
    }

    // ========== BandwidthLimit Tests ==========

    @Test
    fun testBandwidthLimit() {
        val limit = BandwidthLimit(
            audioBitrate = 128,
            videoBitrate = 1000,
            totalBitrate = 2000
        )

        assertEquals(128, limit.audioBitrate)
        assertEquals(1000, limit.videoBitrate)
        assertEquals(2000, limit.totalBitrate)
    }
}
