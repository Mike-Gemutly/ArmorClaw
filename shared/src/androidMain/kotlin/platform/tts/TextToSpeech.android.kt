package com.armorclaw.shared.platform.tts

import android.content.Context
import android.speech.tts.TextToSpeech as AndroidTextToSpeech
import com.armorclaw.shared.platform.tts.TextToSpeech
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import java.util.Locale

/**
 * Android implementation of TextToSpeech
 *
 * Uses the native Android TTS engine for speech synthesis.
 * Supports multiple TTS engines and falls back gracefully.
 *
 * ## Features
 * - Queue-based speech synthesis
 * - Completion callbacks via UtteranceProgressListener
 * - Speech rate and pitch adjustment
 * - Locale-aware voice selection
 * - Graceful shutdown handling
 *
 * ## Usage
 * ```kotlin
 * val tts = AndroidTextToSpeechImpl(context)
 *
 * tts.speak("Agent task completed")
 * tts.setRate(1.2f) // 20% faster
 *
 * // Don't forget to shutdown when done
 * tts.shutdown()
 * ```
 */
class AndroidTextToSpeechImpl(
    private val context: Context
) : TextToSpeech {

    private var tts: AndroidTextToSpeech? = null
    private val _isAvailable = MutableStateFlow(false)
    private val _isSpeaking = MutableStateFlow(false)

    override val isAvailable: StateFlow<Boolean> = _isAvailable
    override val isSpeaking: StateFlow<Boolean> = _isSpeaking

    private var currentRate: Float = 1.0f
    private var currentPitch: Float = 1.0f

    init {
        initializeTts()
    }

    private fun initializeTts() {
        tts = AndroidTextToSpeech(context.applicationContext) { status ->
            if (status == AndroidTextToSpeech.SUCCESS) {
                _isAvailable.value = true

                // Set up utterance progress listener (deprecated but widely compatible)
                @Suppress("DEPRECATION")
                tts?.setOnUtteranceCompletedListener { _ ->
                    _isSpeaking.value = false
                }

                // Apply initial settings
                tts?.setSpeechRate(currentRate)
                tts?.setPitch(currentPitch)

                // Try to set English locale, fall back to default
                val result = tts?.setLanguage(Locale.US)
                if (result == AndroidTextToSpeech.LANG_MISSING_DATA ||
                    result == AndroidTextToSpeech.LANG_NOT_SUPPORTED) {
                    // Try default locale
                    tts?.setLanguage(Locale.getDefault())
                }
            } else {
                _isAvailable.value = false
            }
        }
    }

    override fun speak(text: String, utteranceId: String?) {
        if (!_isAvailable.value) return

        _isSpeaking.value = true

        @Suppress("DEPRECATION")
        tts?.speak(
            text,
            AndroidTextToSpeech.QUEUE_ADD,
            hashMapOf(AndroidTextToSpeech.Engine.KEY_PARAM_UTTERANCE_ID to (utteranceId ?: "tts_${System.currentTimeMillis()}"))
        )
    }

    /**
     * Speak with interrupt - clears queue and speaks immediately
     */
    fun speakNow(text: String, utteranceId: String? = null) {
        if (!_isAvailable.value) return

        stop()

        _isSpeaking.value = true

        @Suppress("DEPRECATION")
        tts?.speak(
            text,
            AndroidTextToSpeech.QUEUE_FLUSH,
            hashMapOf(AndroidTextToSpeech.Engine.KEY_PARAM_UTTERANCE_ID to (utteranceId ?: "tts_${System.currentTimeMillis()}"))
        )
    }

    override fun stop() {
        tts?.stop()
        _isSpeaking.value = false
    }

    override fun setRate(rate: Float) {
        currentRate = rate.coerceIn(0.5f, 2.0f)
        tts?.setSpeechRate(currentRate)
    }

    override fun setPitch(pitch: Float) {
        currentPitch = pitch.coerceIn(0.5f, 2.0f)
        tts?.setPitch(currentPitch)
    }

    override fun shutdown() {
        tts?.stop()
        tts?.shutdown()
        tts = null
        _isAvailable.value = false
        _isSpeaking.value = false
    }

    /**
     * Check if a specific language is available
     */
    fun isLanguageAvailable(locale: Locale): Boolean {
        val result = tts?.isLanguageAvailable(locale) ?: AndroidTextToSpeech.LANG_NOT_SUPPORTED
        return result >= AndroidTextToSpeech.LANG_AVAILABLE
    }

    /**
     * Set the speech language
     */
    fun setLanguage(locale: Locale): Boolean {
        val result = tts?.setLanguage(locale)
        return result != AndroidTextToSpeech.LANG_MISSING_DATA &&
               result != AndroidTextToSpeech.LANG_NOT_SUPPORTED
    }
}

/**
 * Singleton TTS instance holder
 */
private var _ttsInstance: AndroidTextToSpeechImpl? = null

/**
 * Get the TTS instance - requires Android context
 * Note: On Android, this should be called from the Application context
 */
actual fun getTextToSpeech(): TextToSpeech {
    // Return existing instance or throw if not initialized
    return _ttsInstance ?: throw IllegalStateException(
        "TextToSpeech not initialized. Call initializeTextToSpeech(context) first."
    )
}

/**
 * Initialize TTS with Android context
 * Should be called once from Application.onCreate()
 */
fun initializeTextToSpeech(context: Context): TextToSpeech {
    if (_ttsInstance == null) {
        _ttsInstance = AndroidTextToSpeechImpl(context.applicationContext)
    }
    return _ttsInstance!!
}

/**
 * Shutdown TTS instance
 */
fun shutdownTextToSpeech() {
    _ttsInstance?.shutdown()
    _ttsInstance = null
}
