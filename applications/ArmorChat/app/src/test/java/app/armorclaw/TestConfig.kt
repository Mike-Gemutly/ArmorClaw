package app.armorclaw

/**
 * Test Configuration and Utilities
 *
 * This file provides common test utilities and configuration.
 */

object TestConfig {
    // Test timeouts
    const val DEFAULT_TIMEOUT_MS = 5000L
    const val NETWORK_TIMEOUT_MS = 10000L
    const val CRYPTO_TIMEOUT_MS = 30000L

    // Test data generators
    object DataGenerator {
        fun randomString(length: Int): String {
            val chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
            return (1..length)
                .map { chars.random() }
                .joinToString("")
        }

        fun randomEmail(): String {
            return "${randomString(8)}@test.example.com"
        }

        fun randomDeviceId(): String {
            return "device-${randomString(16)}"
        }

        fun randomUserId(): String {
            return "@${randomString(8)}:test.example.com"
        }

        fun randomMessage(): String {
            return "Test message: ${randomString(32)}"
        }

        fun randomRoomId(): String {
            return "!${randomString(16)}:test.example.com"
        }
    }

    // Test fixtures
    object Fixtures {
        val testLockdownStatus = mapOf(
            "mode" to "operational",
            "admin_established" to true,
            "single_device_mode" to false,
            "setup_complete" to true,
            "security_configured" to true,
            "keystore_initialized" to true
        )

        val testDevice = mapOf(
            "id" to "device-test-001",
            "name" to "Test Device",
            "type" to "mobile",
            "trust_state" to "verified",
            "last_seen" to "2026-02-16T00:00:00Z",
            "is_current" to true
        )

        val testInvite = mapOf(
            "id" to "invite-test-001",
            "code" to "TEST123",
            "role" to "user",
            "status" to "active",
            "created_at" to "2026-02-16T00:00:00Z",
            "expires_at" to "2026-03-16T00:00:00Z"
        )

        val testDataCategories = listOf(
            mapOf(
                "id" to "banking",
                "name" to "Banking Information",
                "risk_level" to "high",
                "permission" to "deny"
            ),
            mapOf(
                "id" to "pii",
                "name" to "Personal Information",
                "risk_level" to "high",
                "permission" to "deny"
            ),
            mapOf(
                "id" to "location",
                "name" to "Location",
                "risk_level" to "low",
                "permission" to "allow"
            )
        )
    }

    // Test matrix for parameterized tests
    object TestMatrix {
        val passphraseLengths = listOf(7, 8, 12, 20, 64)
        val networkConditions = listOf("wifi", "cellular", "offline")
        val encryptionModes = listOf("olm", "megolm")
        val messageSizes = listOf(10, 100, 1000, 10000)
    }
}
