package com.armorclaw.shared.domain.model

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class ArmorClawErrorCodeTest {

    // ========== fromCode Tests ==========

    @Test
    fun `fromCode should return correct error code`() {
        assertEquals(ArmorClawErrorCode.VOICE_MIC_DENIED, ArmorClawErrorCode.fromCode("E001"))
        assertEquals(ArmorClawErrorCode.HOMESERVER_UNREACHABLE, ArmorClawErrorCode.fromCode("E011"))
        assertEquals(ArmorClawErrorCode.DEVICE_UNVERIFIED, ArmorClawErrorCode.fromCode("E021"))
        assertEquals(ArmorClawErrorCode.ENCRYPTION_KEY_ERROR, ArmorClawErrorCode.fromCode("E031"))
        assertEquals(ArmorClawErrorCode.SYNC_QUEUE_OVERFLOW, ArmorClawErrorCode.fromCode("E041"))
        assertEquals(ArmorClawErrorCode.THREAD_NOT_FOUND, ArmorClawErrorCode.fromCode("E051"))
        assertEquals(ArmorClawErrorCode.LOGIN_FAILED, ArmorClawErrorCode.fromCode("E081"))
    }

    @Test
    fun `fromCode should return UNKNOWN_ERROR for invalid code`() {
        assertEquals(ArmorClawErrorCode.UNKNOWN_ERROR, ArmorClawErrorCode.fromCode("INVALID"))
        assertEquals(ArmorClawErrorCode.UNKNOWN_ERROR, ArmorClawErrorCode.fromCode(""))
        assertEquals(ArmorClawErrorCode.UNKNOWN_ERROR, ArmorClawErrorCode.fromCode("E999"))
    }

    // ========== fromCategory Tests ==========

    @Test
    fun `fromCategory should return all errors in category`() {
        val voiceErrors = ArmorClawErrorCode.fromCategory(ErrorCategory.PERMISSION)
        assertTrue(voiceErrors.contains(ArmorClawErrorCode.VOICE_MIC_DENIED))
        assertTrue(voiceErrors.contains(ArmorClawErrorCode.VOICE_PERMISSION_REQUIRED))
        assertTrue(voiceErrors.contains(ArmorClawErrorCode.CAMERA_DENIED))

        val networkErrors = ArmorClawErrorCode.fromCategory(ErrorCategory.NETWORK)
        assertTrue(networkErrors.contains(ArmorClawErrorCode.HOMESERVER_UNREACHABLE))
        assertTrue(networkErrors.contains(ArmorClawErrorCode.CONNECTION_TIMEOUT))
        assertTrue(networkErrors.contains(ArmorClawErrorCode.NO_NETWORK))
    }

    // ========== Recoverable Tests ==========

    @Test
    fun `recoverable errors should be marked correctly`() {
        assertTrue(ArmorClawErrorCode.VOICE_MIC_DENIED.recoverable)
        assertTrue(ArmorClawErrorCode.HOMESERVER_UNREACHABLE.recoverable)
        assertTrue(ArmorClawErrorCode.DEVICE_UNVERIFIED.recoverable)

        assertFalse(ArmorClawErrorCode.CALL_NOT_FOUND.recoverable)
        assertFalse(ArmorClawErrorCode.ACCOUNT_LOCKED.recoverable)
        assertFalse(ArmorClawErrorCode.THREAD_ROOT_DELETED.recoverable)
    }

    // ========== User Message Tests ==========

    @Test
    fun `userMessage should be non-empty for all codes`() {
        ArmorClawErrorCode.entries.forEach { errorCode ->
            assertTrue(
                errorCode.userMessage.isNotBlank(),
                "Error code ${errorCode.code} should have a non-empty user message"
            )
        }
    }

    // ========== ArmorClawError Tests ==========

    @Test
    fun `ArmorClawError toDisplayString should include details when present`() {
        val error = ArmorClawError(
            code = ArmorClawErrorCode.HOMESERVER_UNREACHABLE,
            details = "Server returned 503"
        )

        val display = error.toDisplayString()
        assertTrue(display.contains("Server temporarily unavailable"))
        assertTrue(display.contains("Server returned 503"))
    }

    @Test
    fun `ArmorClawError toDisplayString should not include details when null`() {
        val error = ArmorClawError(
            code = ArmorClawErrorCode.HOMESERVER_UNREACHABLE,
            details = null
        )

        val display = error.toDisplayString()
        assertTrue(display.contains("Server temporarily unavailable"))
        assertFalse(display.contains("\n"))
    }

    @Test
    fun `ArmorClawError isRecoverable should return true only when code is recoverable and action exists`() {
        val recoverableError = ArmorClawError(
            code = ArmorClawErrorCode.HOMESERVER_UNREACHABLE,
            recoverableAction = RecoverableAction.Retry
        )
        assertTrue(recoverableError.isRecoverable())

        val nonRecoverableCode = ArmorClawError(
            code = ArmorClawErrorCode.ACCOUNT_LOCKED,
            recoverableAction = RecoverableAction.Retry
        )
        assertFalse(nonRecoverableCode.isRecoverable())

        val noAction = ArmorClawError(
            code = ArmorClawErrorCode.HOMESERVER_UNREACHABLE,
            recoverableAction = null
        )
        assertFalse(noAction.isRecoverable())
    }

    // ========== RecoverableAction Tests ==========

    @Test
    fun `RecoverableAction should serialize correctly`() {
        val actions = listOf(
            RecoverableAction.Retry,
            RecoverableAction.OpenSettings,
            RecoverableAction.ReLogin,
            RecoverableAction.VerifyDevice,
            RecoverableAction.CheckNetwork,
            RecoverableAction.Custom("custom_action", "Do something")
        )

        actions.forEach { action ->
            assertNotNull(action, "Action should not be null")
        }
    }

    // ========== Error Category Coverage Tests ==========

    @Test
    fun `all error categories should have at least one error code`() {
        ErrorCategory.entries.forEach { category ->
            val errors = ArmorClawErrorCode.fromCategory(category)
            assertTrue(
                errors.isNotEmpty(),
                "Category $category should have at least one error code"
            )
        }
    }

    @Test
    fun `error codes should be unique`() {
        val codes = ArmorClawErrorCode.entries.map { it.code }
        val uniqueCodes = codes.toSet()
        assertEquals(
            codes.size,
            uniqueCodes.size,
            "All error codes should be unique"
        )
    }
}
