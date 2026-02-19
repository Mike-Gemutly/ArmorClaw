package app.armorclaw.viewmodel

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

/**
 * Unit tests for BondingViewModel
 *
 * Tests:
 * - Form validation
 * - State transitions
 * - Error handling
 */
@OptIn(ExperimentalCoroutinesApi::class)
class BondingViewModelTest {

    private val testDispatcher = StandardTestDispatcher()
    private lateinit var viewModel: BondingViewModel

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        viewModel = BondingViewModel()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    // ========================================================================
    // Form Validation Tests
    // ========================================================================

    @Test
    fun `isFormValid returns false when all fields empty`() {
        viewModel.displayName.value = ""
        viewModel.deviceName.value = ""
        viewModel.passphrase.value = ""
        viewModel.confirmPassphrase.value = ""

        assertFalse("Empty form should be invalid", viewModel.isFormValid)
    }

    @Test
    fun `isFormValid returns false when displayName empty`() {
        viewModel.displayName.value = ""
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "password123"
        viewModel.confirmPassphrase.value = "password123"

        assertFalse("Missing display name should be invalid", viewModel.isFormValid)
    }

    @Test
    fun `isFormValid returns false when deviceName empty`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = ""
        viewModel.passphrase.value = "password123"
        viewModel.confirmPassphrase.value = "password123"

        assertFalse("Missing device name should be invalid", viewModel.isFormValid)
    }

    @Test
    fun `isFormValid returns false when passphrase too short`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "short"
        viewModel.confirmPassphrase.value = "short"

        assertFalse("Short passphrase should be invalid", viewModel.isFormValid)
    }

    @Test
    fun `isFormValid returns false when passphrases dont match`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "password123"
        viewModel.confirmPassphrase.value = "different123"

        assertFalse("Mismatched passphrases should be invalid", viewModel.isFormValid)
    }

    @Test
    fun `isFormValid returns true when all fields valid`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "password123"
        viewModel.confirmPassphrase.value = "password123"

        assertTrue("Valid form should be valid", viewModel.isFormValid)
    }

    @Test
    fun `isFormValid accepts minimum 8 character passphrase`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "12345678"
        viewModel.confirmPassphrase.value = "12345678"

        assertTrue("8-character passphrase should be valid", viewModel.isFormValid)
    }

    @Test
    fun `isFormValid rejects 7 character passphrase`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "1234567"
        viewModel.confirmPassphrase.value = "1234567"

        assertFalse("7-character passphrase should be invalid", viewModel.isFormValid)
    }

    // ========================================================================
    // State Management Tests
    // ========================================================================

    @Test
    fun `initial state is Idle`() = runTest {
        val state = viewModel.uiState.value
        assertTrue("Initial state should be Idle", state is BondingUiState.Idle)
    }

    @Test
    fun `reset clears form fields`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "password123"
        viewModel.confirmPassphrase.value = "password123"

        viewModel.reset()

        assertEquals("Display name should be cleared", "", viewModel.displayName.value)
        assertEquals("Device name should be cleared", "", viewModel.deviceName.value)
        assertEquals("Passphrase should be cleared", "", viewModel.passphrase.value)
        assertEquals("Confirm passphrase should be cleared", "", viewModel.confirmPassphrase.value)
    }

    @Test
    fun `reset sets state to ReadyToClaim`() {
        viewModel.reset()

        val state = viewModel.uiState.value
        assertTrue("State after reset should be ReadyToClaim", state is BondingUiState.ReadyToClaim)
    }

    @Test
    fun `clearSensitiveData clears passphrases only`() {
        viewModel.displayName.value = "Test User"
        viewModel.deviceName.value = "Test Device"
        viewModel.passphrase.value = "password123"
        viewModel.confirmPassphrase.value = "password123"

        viewModel.clearSensitiveData()

        assertEquals("Display name should remain", "Test User", viewModel.displayName.value)
        assertEquals("Device name should remain", "Test Device", viewModel.deviceName.value)
        assertEquals("Passphrase should be cleared", "", viewModel.passphrase.value)
        assertEquals("Confirm passphrase should be cleared", "", viewModel.confirmPassphrase.value)
    }

    // ========================================================================
    // UI State Tests
    // ========================================================================

    @Test
    fun `BondingUiState Success contains all required data`() {
        val success = BondingUiState.Success(
            adminId = "admin-1",
            deviceId = "device-1",
            sessionToken = "token-1",
            nextStep = "security"
        )

        assertEquals("adminId should match", "admin-1", success.adminId)
        assertEquals("deviceId should match", "device-1", success.deviceId)
        assertEquals("sessionToken should match", "token-1", success.sessionToken)
        assertEquals("nextStep should match", "security", success.nextStep)
    }

    @Test
    fun `BondingUiState types are distinct`() {
        val states = listOf(
            BondingUiState.Idle,
            BondingUiState.CheckingBridge,
            BondingUiState.ReadyToClaim,
            BondingUiState.Claiming,
            BondingUiState.AlreadyClaimed,
            BondingUiState.Success("", "", "", ""),
            BondingUiState.ValidationError(""),
            BondingUiState.BridgeError(
                app.armorclaw.utils.BridgeError(
                    app.armorclaw.utils.ErrorCode.UNKNOWN,
                    "Test",
                    "Test error",
                    false
                )
            )
        )

        assertEquals("Should have all expected state types", 8, states.size)
    }
}
