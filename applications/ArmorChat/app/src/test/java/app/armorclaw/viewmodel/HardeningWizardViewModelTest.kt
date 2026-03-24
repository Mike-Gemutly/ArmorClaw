package app.armorclaw.viewmodel

import app.armorclaw.network.BridgeApi
import io.mockk.coEvery
import io.mockk.mockk
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

@OptIn(ExperimentalCoroutinesApi::class)
class HardeningWizardViewModelTest {

    private val testDispatcher = StandardTestDispatcher()
    private val mockApi = mockk<BridgeApi>()
    private lateinit var viewModel: HardeningWizardViewModel

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        viewModel = HardeningWizardViewModel(api = mockApi)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    // ========================================================================
    // loadState Tests
    // ========================================================================

    @Test
    fun `loadState updates UI state to Loading then Loaded`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = false,
            bootstrap_wiped = false,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue("State should be Loaded", state is HardeningUiState.Loaded)
        assertEquals("Status should match", mockStatus, (state as HardeningUiState.Loaded).status)
    }

    @Test
    fun `loadState sets UI state to Error on API failure`() = runTest {
        coEvery { mockApi.getHardeningStatus() } returns Result.failure(Exception("API error"))

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue("State should be Error", state is HardeningUiState.Error)
        assertEquals("Error message should match", "API error", (state as HardeningUiState.Error).message)
    }

    @Test
    fun `initial state is NotStarted`() {
        val state = viewModel.uiState.value
        assertTrue("Initial state should be NotStarted", state is HardeningUiState.NotStarted)
    }

    // ========================================================================
    // rotatePassword Tests
    // ========================================================================

    @Test
    fun `rotatePassword calls API and reloads state`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = false,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.rotateBootstrapPassword(any()) } returns Result.success(mapOf("success" to true))
        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.rotatePassword("newPassword123")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue("State should be Loaded after rotation", state is HardeningUiState.Loaded)
        coEvery { mockApi.rotateBootstrapPassword("newPassword123") }
    }

    @Test
    fun `rotatePassword sets Error state on API failure`() = runTest {
        coEvery { mockApi.rotateBootstrapPassword(any()) } returns Result.failure(Exception("Rotation failed"))

        viewModel.rotatePassword("newPassword123")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue("State should be Error on failure", state is HardeningUiState.Error)
    }

    // ========================================================================
    // acknowledgeStep Tests
    // ========================================================================

    @Test
    fun `acknowledgeStep calls API and reloads state`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.acknowledgeHardeningStep("WIPE_BOOTSTRAP") } returns Result.success(mockStatus)
        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.acknowledgeStep("WIPE_BOOTSTRAP")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue("State should be Loaded after acknowledgement", state is HardeningUiState.Loaded)
    }

    @Test
    fun `acknowledgeStep sets Error state on API failure`() = runTest {
        coEvery { mockApi.acknowledgeHardeningStep(any()) } returns Result.failure(Exception("Ack failed"))

        viewModel.acknowledgeStep("ROTATE_PASSWORD")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue("State should be Error on failure", state is HardeningUiState.Error)
    }

    // ========================================================================
    // getCurrentStep Tests
    // ========================================================================

    @Test
    fun `getCurrentStep returns ROTATE_PASSWORD when password not rotated`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = false,
            bootstrap_wiped = false,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertEquals("Should return ROTATE_PASSWORD", HardeningStep.ROTATE_PASSWORD, viewModel.getCurrentStep())
    }

    @Test
    fun `getCurrentStep returns WIPE_BOOTSTRAP when password rotated but bootstrap not wiped`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = false,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertEquals("Should return WIPE_BOOTSTRAP", HardeningStep.WIPE_BOOTSTRAP, viewModel.getCurrentStep())
    }

    @Test
    fun `getCurrentStep returns VERIFY_DEVICE when previous steps complete`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertEquals("Should return VERIFY_DEVICE", HardeningStep.VERIFY_DEVICE, viewModel.getCurrentStep())
    }

    @Test
    fun `getCurrentStep returns BACKUP_RECOVERY when device verified`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = true,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertEquals("Should return BACKUP_RECOVERY", HardeningStep.BACKUP_RECOVERY, viewModel.getCurrentStep())
    }

    @Test
    fun `getCurrentStep returns ENABLE_BIOMETRICS when mandatory steps complete`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = true,
            recovery_backed_up = true,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertEquals("Should return ENABLE_BIOMETRICS", HardeningStep.ENABLE_BIOMETRICS, viewModel.getCurrentStep())
    }

    @Test
    fun `getCurrentStep returns COMPLETE when all steps done`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = true,
            recovery_backed_up = true,
            biometrics_enabled = true,
            delegation_ready = true,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertEquals("Should return COMPLETE", HardeningStep.COMPLETE, viewModel.getCurrentStep())
    }

    // ========================================================================
    // isDelegationReady Tests
    // ========================================================================

    @Test
    fun `isDelegationReady returns false when no steps complete`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = false,
            bootstrap_wiped = false,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertFalse("Should return false when no steps complete", viewModel.isDelegationReady())
    }

    @Test
    fun `isDelegationReady returns false when only password rotated`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = false,
            device_verified = false,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertFalse("Should return false when only password rotated", viewModel.isDelegationReady())
    }

    @Test
    fun `isDelegationReady returns false when 3 of 4 mandatory steps complete`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = true,
            recovery_backed_up = false,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertFalse("Should return false when 3 of 4 mandatory steps complete", viewModel.isDelegationReady())
    }

    @Test
    fun `isDelegationReady returns true when all 4 mandatory steps complete`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = true,
            recovery_backed_up = true,
            biometrics_enabled = false,
            delegation_ready = false,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertTrue("Should return true when all 4 mandatory steps complete", viewModel.isDelegationReady())
    }

    @Test
    fun `isDelegationReady returns true when all mandatory steps complete, biometrics optional`() = runTest {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = true,
            recovery_backed_up = true,
            biometrics_enabled = true,
            delegation_ready = true,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        coEvery { mockApi.getHardeningStatus() } returns Result.success(mockStatus)

        viewModel.loadState()
        testScheduler.advanceUntilIdle()

        assertTrue("Should return true with biometrics enabled", viewModel.isDelegationReady())
    }

    @Test
    fun `isDelegationReady returns false when status not loaded`() {
        assertFalse("Should return false when status not loaded", viewModel.isDelegationReady())
    }

    // ========================================================================
    // UI State Tests
    // ========================================================================

    @Test
    fun `HardeningUiState Loaded contains status data`() {
        val mockStatus = BridgeApi.HardeningStatus(
            user_id = "user-1",
            password_rotated = true,
            bootstrap_wiped = true,
            device_verified = true,
            recovery_backed_up = true,
            biometrics_enabled = true,
            delegation_ready = true,
            created_at = "2025-01-01T00:00:00Z",
            updated_at = "2025-01-01T00:00:00Z"
        )

        val state = HardeningUiState.Loaded(mockStatus)

        assertEquals("Status should match", mockStatus, state.status)
    }

    @Test
    fun `HardeningUiState Error contains message`() {
        val state = HardeningUiState.Error("Test error")

        assertEquals("Error message should match", "Test error", state.message)
    }

    @Test
    fun `HardeningStep enum has all expected values`() {
        val steps = HardeningStep.values()
        assertEquals("Should have 6 steps", 6, steps.size)
        assertTrue("Should contain ROTATE_PASSWORD", steps.contains(HardeningStep.ROTATE_PASSWORD))
        assertTrue("Should contain WIPE_BOOTSTRAP", steps.contains(HardeningStep.WIPE_BOOTSTRAP))
        assertTrue("Should contain VERIFY_DEVICE", steps.contains(HardeningStep.VERIFY_DEVICE))
        assertTrue("Should contain BACKUP_RECOVERY", steps.contains(HardeningStep.BACKUP_RECOVERY))
        assertTrue("Should contain ENABLE_BIOMETRICS", steps.contains(HardeningStep.ENABLE_BIOMETRICS))
        assertTrue("Should contain COMPLETE", steps.contains(HardeningStep.COMPLETE))
    }
}
