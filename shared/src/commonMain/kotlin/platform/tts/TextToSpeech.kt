package com.armorclaw.shared.platform.tts

/**
 * Text-to-Speech Platform Interface
 *
 * Expect/actual pattern for cross-platform TTS support.
 * Enables voice-first accessibility mode for eyes-free agent supervision.
 *
 * ## Architecture
 * ```
 * TextToSpeechProvider (singleton)
 *      └── getTextToSpeech(): TextToSpeech
 *
 * TextToSpeech (interface)
 *      ├── speak(text: String) - Synthesize speech
 *      ├── stop() - Stop current speech
 *      ├── setRate(rate: Float) - Adjust speech rate
 *      ├── setPitch(pitch: Float) - Adjust pitch
 *      └── isAvailable: StateFlow<Boolean> - TTS availability
 * ```
 *
 * ## Usage
 * ```kotlin
 * val tts: TextToSpeech = getTextToSpeech()
 *
 * tts.speak("Agent checkout completed successfully")
 * tts.speak("CAPTCHA required. Please check the screen.")
 *
 * // Adjust for faster feedback
 * tts.setRate(1.2f)
 * ```
 *
 * ## Android Implementation
 * Uses android.speech.tts.TextToSpeech with:
 * - Queue mode: QUEUE_ADD for sequential speech
 * - Utterance ID for completion callbacks
 * - Locale-aware voice selection
 *
 * ## Security Considerations
 * - Never speak PII data (masked in event descriptions)
 * - Speak sensitivity level warnings
 * - Confirm critical actions verbally
 */

/**
 * Platform-specific TTS interface
 */
interface TextToSpeech {
    /**
     * Speak the given text aloud
     *
     * @param text Text to synthesize
     * @param utteranceId Optional ID for tracking completion
     */
    fun speak(text: String, utteranceId: String? = null)

    /**
     * Stop any current speech synthesis
     */
    fun stop()

    /**
     * Set speech rate
     *
     * @param rate Speech rate multiplier (0.5 = half speed, 2.0 = double speed)
     */
    fun setRate(rate: Float)

    /**
     * Set speech pitch
     *
     * @param pitch Pitch multiplier (0.5 = low, 2.0 = high)
     */
    fun setPitch(pitch: Float)

    /**
     * Check if TTS is available and initialized
     */
    val isAvailable: kotlinx.coroutines.flow.StateFlow<Boolean>

    /**
     * Check if currently speaking
     */
    val isSpeaking: kotlinx.coroutines.flow.StateFlow<Boolean>

    /**
     * Release TTS resources
     */
    fun shutdown()
}

/**
 * Get platform-specific TTS instance
 * Must be called with Android Context on Android platform
 */
expect fun getTextToSpeech(): TextToSpeech

/**
 * TTS event types for voice mode announcements
 */
sealed class TtsEvent {
    abstract val priority: TtsPriority
    abstract val shouldInterrupt: Boolean

    /**
     * Agent status update
     */
    data class AgentStatus(
        val agentName: String,
        val status: String,
        override val priority: TtsPriority = TtsPriority.NORMAL,
        override val shouldInterrupt: Boolean = false
    ) : TtsEvent()

    /**
     * Intervention required - high priority
     */
    data class InterventionRequired(
        val type: String,
        val context: String,
        override val priority: TtsPriority = TtsPriority.HIGH,
        override val shouldInterrupt: Boolean = true
    ) : TtsEvent()

    /**
     * PII access request
     */
    data class PiiRequest(
        val fieldName: String,
        val sensitivity: String,
        override val priority: TtsPriority = TtsPriority.HIGH,
        override val shouldInterrupt: Boolean = true
    ) : TtsEvent()

    /**
     * Task completion
     */
    data class TaskComplete(
        val taskName: String,
        val result: String? = null,
        override val priority: TtsPriority = TtsPriority.NORMAL,
        override val shouldInterrupt: Boolean = false
    ) : TtsEvent()

    /**
     * Error occurred
     */
    data class Error(
        val message: String,
        override val priority: TtsPriority = TtsPriority.HIGH,
        override val shouldInterrupt: Boolean = true
    ) : TtsEvent()

    /**
     * System announcement
     */
    data class SystemAnnouncement(
        val message: String,
        override val priority: TtsPriority = TtsPriority.LOW,
        override val shouldInterrupt: Boolean = false
    ) : TtsEvent()
}

/**
 * TTS priority levels for queue management
 */
enum class TtsPriority {
    LOW,      // Background info, can be interrupted
    NORMAL,   // Standard announcements
    HIGH,     // Important alerts (interventions, PII)
    URGENT    // Critical alerts (always interrupt)
}

/**
 * Voice mode configuration
 */
data class VoiceModeConfig(
    val enabled: Boolean = false,
    val speechRate: Float = 1.0f,
    val pitch: Float = 1.0f,
    val announceAgentStatus: Boolean = true,
    val announceInterventions: Boolean = true,
    val announcePiiRequests: Boolean = true,
    val announceErrors: Boolean = true,
    val announceCompletions: Boolean = true,
    val filterSensitiveInfo: Boolean = true,
    val verboseMode: Boolean = false  // More detailed announcements
) {
    companion object {
        val DEFAULT = VoiceModeConfig()
        val ACCESSIBILITY = VoiceModeConfig(
            enabled = true,
            speechRate = 0.9f,
            verboseMode = true
        )
        val MINIMAL = VoiceModeConfig(
            enabled = true,
            announceAgentStatus = false,
            announceCompletions = false,
            verboseMode = false
        )
    }
}

/**
 * Convert TtsEvent to spoken text
 */
fun TtsEvent.toSpokenText(config: VoiceModeConfig): String {
    return when (this) {
        is TtsEvent.AgentStatus -> {
            if (config.verboseMode) {
                "Agent $agentName is now $status"
            } else {
                "$agentName $status"
            }
        }
        is TtsEvent.InterventionRequired -> {
            val typeText = when (type.lowercase()) {
                "captcha" -> "CAPTCHA"
                "2fa", "two_fa" -> "two-factor authentication"
                "error" -> "error"
                else -> type
            }
            if (config.verboseMode) {
                "$typeText required. $context. Please check your screen."
            } else {
                "$typeText needed. $context"
            }
        }
        is TtsEvent.PiiRequest -> {
            val sensitivityText = when (sensitivity.lowercase()) {
                "critical" -> "critical security field"
                "high" -> "sensitive field"
                "medium" -> "personal field"
                else -> "field"
            }
            if (config.filterSensitiveInfo) {
                "Agent requests access to $sensitivityText: $fieldName"
            } else {
                "Agent requests $fieldName, a $sensitivityText"
            }
        }
        is TtsEvent.TaskComplete -> {
            if (config.verboseMode && result != null) {
                "Task $taskName completed. $result"
            } else {
                "$taskName completed"
            }
        }
        is TtsEvent.Error -> {
            if (config.verboseMode) {
                "Error: $message"
            } else {
                message
            }
        }
        is TtsEvent.SystemAnnouncement -> {
            message
        }
    }
}
