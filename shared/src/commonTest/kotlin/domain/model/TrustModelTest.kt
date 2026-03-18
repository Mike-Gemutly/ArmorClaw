package com.armorclaw.shared.domain.model

import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class TrustModelTest {

    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
        classDiscriminator = "type"
    }

    // ========== TrustLevel Tests ==========

    @Test
    fun `TrustLevel isTrusted should return correct values`() {
        assertFalse(TrustLevel.UNVERIFIED.isTrusted())
        assertTrue(TrustLevel.CROSS_SIGNED.isTrusted())
        assertTrue(TrustLevel.VERIFIED_IN_PERSON.isTrusted())
        assertTrue(TrustLevel.KNOWN_UNCOMPROMISED.isTrusted())
        assertFalse(TrustLevel.COMPROMISED.isTrusted())
    }

    @Test
    fun `TrustLevel requiresVerification should return correct values`() {
        assertTrue(TrustLevel.UNVERIFIED.requiresVerification())
        assertFalse(TrustLevel.CROSS_SIGNED.requiresVerification())
        assertFalse(TrustLevel.VERIFIED_IN_PERSON.requiresVerification())
        assertFalse(TrustLevel.KNOWN_UNCOMPROMISED.requiresVerification())
        assertFalse(TrustLevel.COMPROMISED.requiresVerification())
    }

    @Test
    fun `TrustLevel should serialize correctly`() {
        val levels = TrustLevel.entries

        levels.forEach { level ->
            val serialized = json.encodeToString(level)
            val deserialized = json.decodeFromString<TrustLevel>(serialized)
            assertEquals(level, deserialized, "TrustLevel $level should round-trip correctly")
        }
    }

    // ========== VerificationState Tests ==========

    @Test
    fun `VerificationState isInProgress should return correct values`() {
        assertTrue(VerificationState.Requested.isInProgress())
        assertTrue(VerificationState.Ready.isInProgress())
        assertTrue(VerificationState.EmojiChallenge(emptyList(), "tx123").isInProgress())
        assertTrue(VerificationState.CodeChallenge("123456", "tx123").isInProgress())
        assertTrue(VerificationState.Verifying.isInProgress())

        assertFalse(VerificationState.Unverified.isInProgress())
        assertFalse(VerificationState.Verified.isInProgress())
        assertFalse(VerificationState.Cancelled("reason").isInProgress())
        assertFalse(VerificationState.Failed("error").isInProgress())
    }

    @Test
    fun `VerificationState isComplete should return correct values`() {
        assertTrue(VerificationState.Verified.isComplete())
        assertTrue(VerificationState.Cancelled("reason").isComplete())
        assertTrue(VerificationState.Failed("error").isComplete())

        assertFalse(VerificationState.Unverified.isComplete())
        assertFalse(VerificationState.Requested.isComplete())
        assertFalse(VerificationState.Ready.isComplete())
        assertFalse(VerificationState.EmojiChallenge(emptyList(), "tx123").isComplete())
    }

    @Test
    fun `EmojiChallenge should serialize correctly`() {
        val state: VerificationState = VerificationState.EmojiChallenge(
            emojis = listOf(
                EmojiInfo("🐶", "Dog", 0),
                EmojiInfo("🐱", "Cat", 1),
                EmojiInfo("🦁", "Lion", 2)
            ),
            transactionId = "tx_12345"
        )

        val serialized = json.encodeToString(state)
        val deserialized = json.decodeFromString<VerificationState>(serialized)

        assertTrue(deserialized is VerificationState.EmojiChallenge)
        assertEquals("tx_12345", (deserialized as VerificationState.EmojiChallenge).transactionId)
        assertEquals(3, deserialized.emojis.size)
    }

    @Test
    fun `Cancelled state should hold reason and byMe`() {
        val cancelledByMe = VerificationState.Cancelled("User cancelled", true)
        val cancelledByOther = VerificationState.Cancelled("Other user cancelled", false)

        assertEquals("User cancelled", cancelledByMe.reason)
        assertTrue(cancelledByMe.byMe)

        assertEquals("Other user cancelled", cancelledByOther.reason)
        assertFalse(cancelledByOther.byMe)
    }

    @Test
    fun `Failed state should hold reason and errorCode`() {
        val failed = VerificationState.Failed(
            reason = "Key mismatch",
            errorCode = "E023"
        )

        assertEquals("Key mismatch", failed.reason)
        assertEquals("E023", failed.errorCode)
    }

    // ========== EmojiInfo Tests ==========

    @Test
    fun `EmojiInfo STANDARD_EMOJIS should have 64 emojis`() {
        assertEquals(64, EmojiInfo.STANDARD_EMOJIS.size)
    }

    @Test
    fun `EmojiInfo should have correct indices`() {
        EmojiInfo.STANDARD_EMOJIS.forEachIndexed { index, emojiInfo ->
            assertEquals(index, emojiInfo.index, "Emoji at index $index should have matching index property")
        }
    }

    @Test
    fun `EmojiInfo should serialize correctly`() {
        val emoji = EmojiInfo("🚀", "Rocket", 36)

        val serialized = json.encodeToString(emoji)
        val deserialized = json.decodeFromString<EmojiInfo>(serialized)

        assertEquals("🚀", deserialized.emoji)
        assertEquals("Rocket", deserialized.description)
        assertEquals(36, deserialized.index)
    }

    // ========== DeviceInfo Tests ==========

    @Test
    fun `DeviceInfo should serialize correctly`() {
        val device = DeviceInfo(
            deviceId = "DEVICE123",
            displayName = "Android Phone",
            userId = "@user:example.com",
            isVerified = true,
            trustLevel = TrustLevel.CROSS_SIGNED,
            lastSeenIp = "192.168.1.1",
            lastSeenTimestamp = 1704067200000,
            isCurrentDevice = true
        )

        val serialized = json.encodeToString(device)
        val deserialized = json.decodeFromString<DeviceInfo>(serialized)

        assertEquals("DEVICE123", deserialized.deviceId)
        assertEquals("Android Phone", deserialized.displayName)
        assertEquals("@user:example.com", deserialized.userId)
        assertTrue(deserialized.isVerified)
        assertEquals(TrustLevel.CROSS_SIGNED, deserialized.trustLevel)
        assertTrue(deserialized.isCurrentDevice)
    }

    @Test
    fun `DeviceInfo defaults should be correct`() {
        val device = DeviceInfo(
            deviceId = "DEVICE123",
            displayName = null,
            userId = "@user:example.com"
        )

        assertFalse(device.isVerified)
        assertEquals(TrustLevel.UNVERIFIED, device.trustLevel)
        assertNull(device.lastSeenIp)
        assertNull(device.lastSeenTimestamp)
        assertFalse(device.isCurrentDevice)
        assertNull(device.keys)
    }

    // ========== CrossSigningKeys Tests ==========

    @Test
    fun `CrossSigningKeys should serialize correctly`() {
        val keys = CrossSigningKeys(
            masterKey = "master_key_value",
            selfSigningKey = "self_signing_key",
            userSigningKey = "user_signing_key",
            verifiedAt = 1704067200000
        )

        val serialized = json.encodeToString(keys)
        val deserialized = json.decodeFromString<CrossSigningKeys>(serialized)

        assertEquals("master_key_value", deserialized.masterKey)
        assertEquals("self_signing_key", deserialized.selfSigningKey)
        assertEquals("user_signing_key", deserialized.userSigningKey)
        assertEquals(1704067200000, deserialized.verifiedAt)
    }

    // ========== CrossSigningStatus Tests ==========

    @Test
    fun `CrossSigningStatus should have all expected values`() {
        val statuses = CrossSigningStatus.entries

        assertEquals(4, statuses.size)
        assertTrue(statuses.contains(CrossSigningStatus.NOT_SETUP))
        assertTrue(statuses.contains(CrossSigningStatus.SETUP_UNVERIFIED))
        assertTrue(statuses.contains(CrossSigningStatus.VERIFIED))
        assertTrue(statuses.contains(CrossSigningStatus.NEEDS_BACKUP))
    }

    // ========== VerificationTransaction Tests ==========

    @Test
    fun `VerificationTransaction should hold all data`() {
        val transaction = VerificationTransaction(
            transactionId = "tx_123",
            otherUserId = "@other:example.com",
            otherDeviceId = "DEVICE456",
            state = VerificationState.EmojiChallenge(emptyList(), "tx_123"),
            startedAt = 1704067200000,
            method = VerificationMethod.SAS
        )

        assertEquals("tx_123", transaction.transactionId)
        assertEquals("@other:example.com", transaction.otherUserId)
        assertEquals("DEVICE456", transaction.otherDeviceId)
        assertTrue(transaction.state is VerificationState.EmojiChallenge)
        assertEquals(VerificationMethod.SAS, transaction.method)
        assertNull(transaction.cancellationReason)
    }

    // ========== EncryptionStatus Tests ==========

    @Test
    fun `EncryptionStatus isEncrypted should return correct values`() {
        assertFalse(EncryptionStatus.NONE.isEncrypted())
        assertFalse(EncryptionStatus.UNENCRYPTED.isEncrypted())
        assertTrue(EncryptionStatus.UNVERIFIED.isEncrypted())
        assertTrue(EncryptionStatus.VERIFIED.isEncrypted())
        assertTrue(EncryptionStatus.CROSS_SIGNED.isEncrypted())
    }

    @Test
    fun `EncryptionStatus isTrusted should return correct values`() {
        assertFalse(EncryptionStatus.NONE.isTrusted())
        assertFalse(EncryptionStatus.UNENCRYPTED.isTrusted())
        assertFalse(EncryptionStatus.UNVERIFIED.isTrusted())
        assertTrue(EncryptionStatus.VERIFIED.isTrusted())
        assertTrue(EncryptionStatus.CROSS_SIGNED.isTrusted())
    }

    // ========== KeyBackupStatus Tests ==========

    @Test
    fun `KeyBackupStatus should have all expected values`() {
        val statuses = KeyBackupStatus.entries

        assertEquals(4, statuses.size)
        assertTrue(statuses.contains(KeyBackupStatus.NO_BACKUP))
        assertTrue(statuses.contains(KeyBackupStatus.UNVERIFIED))
        assertTrue(statuses.contains(KeyBackupStatus.ENABLED))
        assertTrue(statuses.contains(KeyBackupStatus.REQUIRED))
    }

    // ========== UserSession Tests ==========

    @Test
    fun `UserSession should serialize correctly`() {
        val session = UserSession(
            userId = "@user:example.com",
            deviceId = "DEVICE123",
            accessToken = "access_token_value",
            refreshToken = "refresh_token_value",
            homeserver = "https://matrix.example.com",
            expiresAt = kotlinx.datetime.Clock.System.now() + kotlin.time.Duration.parseIsoString("PT1H")
        )

        val serialized = json.encodeToString(session)
        val deserialized = json.decodeFromString<UserSession>(serialized)

        assertEquals("@user:example.com", deserialized.userId)
        assertEquals("DEVICE123", deserialized.deviceId)
        assertEquals("access_token_value", deserialized.accessToken)
        assertEquals("https://matrix.example.com", deserialized.homeserver)
    }
}
