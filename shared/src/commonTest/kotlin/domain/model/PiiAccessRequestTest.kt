package com.armorclaw.shared.domain.model

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for PiiAccessRequest, PiiField, and SensitivityLevel
 */
class PiiAccessRequestTest {

    @Test
    fun `PiiAccessRequest isExpired returns correct value`() {
        // Not expired request
        val validRequest = PiiAccessRequest(
            requestId = "req_001",
            agentId = "agent_001",
            fields = emptyList(),
            reason = "Test",
            expiresAt = System.currentTimeMillis() + 60000 // 1 minute from now
        )
        assertFalse(validRequest.isExpired())

        // Expired request
        val expiredRequest = PiiAccessRequest(
            requestId = "req_002",
            agentId = "agent_001",
            fields = emptyList(),
            reason = "Test",
            expiresAt = System.currentTimeMillis() - 1000 // 1 second ago
        )
        assertTrue(expiredRequest.isExpired())
    }

    @Test
    fun `PiiAccessRequest hasCriticalFields returns correct value`() {
        // No critical fields
        val noCritical = createRequestWithSensitivities(
            SensitivityLevel.LOW,
            SensitivityLevel.MEDIUM,
            SensitivityLevel.HIGH
        )
        assertFalse(noCritical.hasCriticalFields())

        // With critical field
        val withCritical = createRequestWithSensitivities(
            SensitivityLevel.LOW,
            SensitivityLevel.CRITICAL,
            SensitivityLevel.MEDIUM
        )
        assertTrue(withCritical.hasCriticalFields())
    }

    @Test
    fun `PiiAccessRequest hasHighSensitivityFields returns correct value`() {
        // Only low/medium fields
        val lowOnly = createRequestWithSensitivities(
            SensitivityLevel.LOW,
            SensitivityLevel.MEDIUM
        )
        assertFalse(lowOnly.hasHighSensitivityFields())

        // With high field
        val withHigh = createRequestWithSensitivities(
            SensitivityLevel.LOW,
            SensitivityLevel.HIGH
        )
        assertTrue(withHigh.hasHighSensitivityFields())

        // With critical field (also counts as high sensitivity)
        val withCritical = createRequestWithSensitivities(
            SensitivityLevel.LOW,
            SensitivityLevel.CRITICAL
        )
        assertTrue(withCritical.hasHighSensitivityFields())
    }

    @Test
    fun `PiiAccessRequest getFieldsBySensitivity groups correctly`() {
        val request = PiiAccessRequest(
            requestId = "req_001",
            agentId = "agent_001",
            fields = listOf(
                PiiField("Email", SensitivityLevel.LOW, "Contact"),
                PiiField("Name", SensitivityLevel.LOW, "Identity"),
                PiiField("Phone", SensitivityLevel.MEDIUM, "Contact"),
                PiiField("Credit Card", SensitivityLevel.HIGH, "Payment")
            ),
            reason = "Test",
            expiresAt = System.currentTimeMillis() + 60000
        )

        val grouped = request.getFieldsBySensitivity()

        assertEquals(2, grouped[SensitivityLevel.LOW]?.size)
        assertEquals(1, grouped[SensitivityLevel.MEDIUM]?.size)
        assertEquals(1, grouped[SensitivityLevel.HIGH]?.size)
        assertEquals(null, grouped[SensitivityLevel.CRITICAL])
    }

    @Test
    fun `SensitivityLevel requiresBiometric returns correct values`() {
        assertFalse(SensitivityLevel.LOW.requiresBiometric())
        assertFalse(SensitivityLevel.MEDIUM.requiresBiometric())
        assertFalse(SensitivityLevel.HIGH.requiresBiometric())
        assertTrue(SensitivityLevel.CRITICAL.requiresBiometric())
    }

    @Test
    fun `SensitivityLevel toDisplayString returns correct values`() {
        assertEquals("Low", SensitivityLevel.LOW.toDisplayString())
        assertEquals("Medium", SensitivityLevel.MEDIUM.toDisplayString())
        assertEquals("High", SensitivityLevel.HIGH.toDisplayString())
        assertEquals("Critical", SensitivityLevel.CRITICAL.toDisplayString())
    }

    @Test
    fun `PiiField creates with all fields`() {
        val field = PiiField(
            name = "Credit Card Number",
            sensitivity = SensitivityLevel.HIGH,
            description = "Required for payment",
            currentValue = "••••4242"
        )

        assertEquals("Credit Card Number", field.name)
        assertEquals(SensitivityLevel.HIGH, field.sensitivity)
        assertEquals("Required for payment", field.description)
        assertEquals("••••4242", field.currentValue)
    }

    @Test
    fun `PiiField creates with null currentValue`() {
        val field = PiiField(
            name = "CVV",
            sensitivity = SensitivityLevel.CRITICAL,
            description = "Card verification"
        )

        assertEquals("CVV", field.name)
        assertEquals(SensitivityLevel.CRITICAL, field.sensitivity)
        assertEquals(null, field.currentValue)
    }

    // Helper function
    private fun createRequestWithSensitivities(vararg levels: SensitivityLevel): PiiAccessRequest {
        return PiiAccessRequest(
            requestId = "req_test",
            agentId = "agent_test",
            fields = levels.mapIndexed { index, level ->
                PiiField("Field $index", level, "Description $index")
            },
            reason = "Test",
            expiresAt = System.currentTimeMillis() + 60000
        )
    }
}
