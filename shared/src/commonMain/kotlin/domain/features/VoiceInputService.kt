package com.armorclaw.shared.domain.features

import com.armorclaw.shared.domain.model.AppResult
import com.armorclaw.shared.domain.model.OperationContext
import kotlinx.coroutines.flow.Flow

/**
 * Service interface for voice input functionality
 *
 * Provides voice-to-text conversion, audio recording management,
 * and voice command processing for enhanced user input.
 *
 * TODO: Implement voice recording state management
 * TODO: Add voice command recognition patterns
 * TODO: Integrate with platform-specific speech recognition APIs
 */
interface VoiceInputService {

    /**
     * Start voice recording
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun startRecording(context: OperationContext? = null): AppResult<Unit>

    /**
     * Stop voice recording and transcribe
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun stopRecording(context: OperationContext? = null): AppResult<String>

    /**
     * Cancel ongoing voice recording
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun cancelRecording(context: OperationContext? = null): AppResult<Unit>

    /**
     * Check if voice recording is currently active
     */
    fun isRecording(): Flow<Boolean>

    /**
     * Observe transcribed text (reactive)
     */
    fun observeTranscription(): Flow<String>

    /**
     * Get supported languages for voice input
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun getSupportedLanguages(context: OperationContext? = null): AppResult<List<VoiceLanguage>>

    /**
     * Set voice input language
     * @param languageCode Language code (e.g., "en-US", "es-ES")
     * @param context Optional operation context for correlation ID tracing
     */
    suspend fun setLanguage(languageCode: String, context: OperationContext? = null): AppResult<Unit>
}

/**
 * Voice language configuration
 *
 * TODO: Add language display names
 * TODO: Add offline availability flag
 */
@kotlinx.serialization.Serializable
data class VoiceLanguage(
    val code: String,
    val name: String,
    val isAvailableOffline: Boolean = false
)

/**
 * Voice input state
 *
 * TODO: Add error states
 * TODO: Add recording duration
 */
@kotlinx.serialization.Serializable
data class VoiceInputState(
    val isRecording: Boolean = false,
    val isTranscribing: Boolean = false,
    val hasPermission: Boolean = false
)
