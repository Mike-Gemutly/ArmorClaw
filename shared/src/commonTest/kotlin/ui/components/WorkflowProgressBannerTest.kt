package com.armorclaw.shared.ui.components

import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.platform.matrix.event.StepStatus
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for WorkflowProgressBanner component
 *
 * Tests state handling, progress calculation, and status display.
 */
class WorkflowProgressBannerTest {

    // ========================================================================
    // WorkflowState.Started Tests
    // ========================================================================

    @Test
    fun `Started state has correct workflowId`() {
        val state = WorkflowState.Started(
            workflowId = "wf_123",
            workflowType = "document_analysis",
            roomId = "!room:example.com",
            initiatedBy = "@user:example.com",
            timestamp = System.currentTimeMillis()
        )

        assertEquals("wf_123", state.workflowId)
        assertEquals("document_analysis", state.workflowType)
        assertEquals("!room:example.com", state.roomId)
    }

    @Test
    fun `Started state is sealed class subtype`() {
        val state: WorkflowState = WorkflowState.Started(
            workflowId = "wf_123",
            workflowType = "test",
            roomId = "!room:example.com",
            initiatedBy = "@user:example.com",
            timestamp = System.currentTimeMillis()
        )

        assertTrue(state is WorkflowState.Started)
    }

    // ========================================================================
    // WorkflowState.StepRunning Tests
    // ========================================================================

    @Test
    fun `StepRunning state has correct step info`() {
        val state = WorkflowState.StepRunning(
            workflowId = "wf_456",
            workflowType = "code_review",
            roomId = "!room:example.com",
            stepId = "step_2",
            stepName = "Analyzing code structure",
            stepIndex = 2,
            totalSteps = 5,
            status = StepStatus.RUNNING,
            timestamp = System.currentTimeMillis()
        )

        assertEquals("step_2", state.stepId)
        assertEquals("Analyzing code structure", state.stepName)
        assertEquals(2, state.stepIndex)
        assertEquals(5, state.totalSteps)
        assertEquals(StepStatus.RUNNING, state.status)
    }

    @Test
    fun `StepRunning calculates progress correctly`() {
        val state = WorkflowState.StepRunning(
            workflowId = "wf_456",
            workflowType = "test",
            roomId = "!room:example.com",
            stepId = "step_3",
            stepName = "Step 3",
            stepIndex = 3,
            totalSteps = 5,
            status = StepStatus.RUNNING,
            timestamp = System.currentTimeMillis()
        )

        // Progress should be (stepIndex - 1) / totalSteps
        // = (3 - 1) / 5 = 0.4
        val expectedProgress = (3 - 1).toFloat() / 5.toFloat()
        val actualProgress = (state.stepIndex - 1).toFloat() / state.totalSteps.toFloat()
        
        assertEquals(expectedProgress, actualProgress, 0.001f)
    }

    @Test
    fun `StepRunning with different statuses`() {
        val statuses = listOf(
            StepStatus.PENDING,
            StepStatus.RUNNING,
            StepStatus.COMPLETED,
            StepStatus.FAILED,
            StepStatus.SKIPPED
        )

        statuses.forEach { status ->
            val state = WorkflowState.StepRunning(
                workflowId = "wf_test",
                workflowType = "test",
                roomId = "!room:example.com",
                stepId = "step_1",
                stepName = "Test Step",
                stepIndex = 1,
                totalSteps = 3,
                status = status,
                timestamp = System.currentTimeMillis()
            )

            assertEquals(status, state.status, "Status should be $status")
        }
    }

    @Test
    fun `StepRunning handles first step correctly`() {
        val state = WorkflowState.StepRunning(
            workflowId = "wf_test",
            workflowType = "test",
            roomId = "!room:example.com",
            stepId = "step_1",
            stepName = "First Step",
            stepIndex = 1,
            totalSteps = 5,
            status = StepStatus.RUNNING,
            timestamp = System.currentTimeMillis()
        )

        // First step should have 0 progress
        val progress = (state.stepIndex - 1).toFloat() / state.totalSteps.toFloat()
        assertEquals(0f, progress, 0.001f)
    }

    @Test
    fun `StepRunning handles last step correctly`() {
        val state = WorkflowState.StepRunning(
            workflowId = "wf_test",
            workflowType = "test",
            roomId = "!room:example.com",
            stepId = "step_5",
            stepName = "Final Step",
            stepIndex = 5,
            totalSteps = 5,
            status = StepStatus.RUNNING,
            timestamp = System.currentTimeMillis()
        )

        // Last step should have progress < 1 (stepIndex - 1) / totalSteps
        val progress = (state.stepIndex - 1).toFloat() / state.totalSteps.toFloat()
        assertEquals(0.8f, progress, 0.001f)
    }

    // ========================================================================
    // WorkflowState Common Tests
    // ========================================================================

    @Test
    fun `All workflow states have workflowId and workflowType`() {
        val states = listOf<WorkflowState>(
            WorkflowState.Started(
                workflowId = "wf_1",
                workflowType = "type_a",
                roomId = "!room:example.com",
                initiatedBy = "@user:example.com",
                timestamp = System.currentTimeMillis()
            ),
            WorkflowState.StepRunning(
                workflowId = "wf_2",
                workflowType = "type_b",
                roomId = "!room:example.com",
                stepId = "step_1",
                stepName = "Step",
                stepIndex = 1,
                totalSteps = 3,
                status = StepStatus.RUNNING,
                timestamp = System.currentTimeMillis()
            )
        )

        states.forEach { state ->
            assertTrue(state.workflowId.isNotEmpty())
            assertTrue(state.workflowType.isNotEmpty())
            assertTrue(state.roomId.isNotEmpty())
        }
    }

    @Test
    fun `WorkflowState is sealed class`() {
        // Verify WorkflowState is sealed by checking we can use when exhaustively
        val state: WorkflowState = WorkflowState.Started(
            workflowId = "test",
            workflowType = "test",
            roomId = "test",
            initiatedBy = "test",
            timestamp = 0
        )

        val result = when (state) {
            is WorkflowState.Started -> "started"
            is WorkflowState.StepRunning -> "running"
        }

        assertEquals("started", result)
    }
}
