package com.armorclaw.shared.domain.model

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for KeystoreStatus and related models
 */
class KeystoreStatusTest {

    @Test
    fun `KeystoreStatus Sealed isAccessible returns false`() {
        val sealed = KeystoreStatus.Sealed()
        assertFalse(sealed.isAccessible())
    }

    @Test
    fun `KeystoreStatus Unsealed isAccessible returns correct value`() {
        // Not expired - accessible
        val validUnsealed = KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + 3600000, // 1 hour from now
            unsealedBy = UnsealMethod.PASSWORD
        )
        assertTrue(validUnsealed.isAccessible())

        // Expired - not accessible
        val expiredUnsealed = KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() - 1000, // 1 second ago
            unsealedBy = UnsealMethod.PASSWORD
        )
        assertFalse(expiredUnsealed.isAccessible())
    }

    @Test
    fun `KeystoreStatus Error isAccessible returns false`() {
        val error = KeystoreStatus.Error("Something went wrong")
        assertFalse(error.isAccessible())
    }

    @Test
    fun `KeystoreStatus Unsealed isExpired returns correct value`() {
        // Not expired
        val valid = KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + 3600000,
            unsealedBy = UnsealMethod.BIOMETRIC
        )
        assertFalse(valid.isExpired())

        // Expired
        val expired = KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() - 1000,
            unsealedBy = UnsealMethod.BIOMETRIC
        )
        assertTrue(expired.isExpired())
    }

    @Test
    fun `KeystoreStatus Unsealed remainingTimeMs returns correct value`() {
        val futureTime = System.currentTimeMillis() + 7200000 // 2 hours
        val unsealed = KeystoreStatus.Unsealed(
            expiresAt = futureTime,
            unsealedBy = UnsealMethod.PASSWORD
        )

        // Should be approximately 2 hours (allow 100ms tolerance for test execution)
        val remaining = unsealed.remainingTimeMs()
        assertTrue(remaining in 7199900..7200100)
    }

    @Test
    fun `KeystoreStatus Unsealed remainingTimeMs returns 0 when expired`() {
        val expired = KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() - 10000,
            unsealedBy = UnsealMethod.PASSWORD
        )

        assertEquals(0L, expired.remainingTimeMs())
    }

    @Test
    fun `KeystoreStatus Unsealed remainingTimeString returns formatted string`() {
        // Hours and minutes
        val twoHours = KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + (2 * 60 * 60 * 1000) + (30 * 60 * 1000),
            unsealedBy = UnsealMethod.BIOMETRIC
        )
        val twoHoursString = twoHours.remainingTimeString()
        assertTrue(twoHoursString.contains("2h"))
        assertTrue(twoHoursString.contains("30m"))

        // Only minutes
        val thirtyMinutes = KeystoreStatus.Unsealed(
            expiresAt = System.currentTimeMillis() + (30 * 60 * 1000),
            unsealedBy = UnsealMethod.PASSWORD
        )
        val thirtyMinutesString = thirtyMinutes.remainingTimeString()
        assertTrue(thirtyMinutesString.contains("30m"))
        assertFalse(thirtyMinutesString.contains("h"))
    }

    @Test
    fun `UnsealMethod toDisplayString returns correct values`() {
        assertEquals("Password", UnsealMethod.PASSWORD.toDisplayString())
        assertEquals("Biometric", UnsealMethod.BIOMETRIC.toDisplayString())
    }

    @Test
    fun `KeystoreStatus Sealed has correct lastUpdated`() {
        val before = System.currentTimeMillis()
        val sealed = KeystoreStatus.Sealed()
        val after = System.currentTimeMillis()

        assertTrue(sealed.lastUpdated in before..after)
    }

    @Test
    fun `KeystoreStatus Error stores message`() {
        val error = KeystoreStatus.Error("Decryption failed")

        assertEquals("Decryption failed", error.message)
        assertFalse(error.isAccessible())
    }

    @Test
    fun `KEYSTORE_SESSION_DURATION_MS is 4 hours`() {
        assertEquals(4 * 60 * 60 * 1000L, KEYSTORE_SESSION_DURATION_MS)
    }
}
