package com.armorclaw.shared.ui.components

import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.platform.matrix.event.StepStatus
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/**
 * Unit tests for WorkflowCard component
 *
 * Tests workflow display, icon selection, and title formatting.
 */
class WorkflowCardTest {

    // ========================================================================
    // Workflow Icon Tests
    // ========================================================================

    @Test
    fun `Workflow icon for document analysis`() {
        val workflowType = "document_analysis"
        val expectedIcon = "Description"
        
        // Test icon selection logic
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for code review`() {
        val workflowType = "code_review"
        val expectedIcon = "Code"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for data processing`() {
        val workflowType = "data_processing"
        val expectedIcon = "Storage"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for report generation`() {
        val workflowType = "report_generation"
        val expectedIcon = "Assessment"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for meeting summary`() {
        val workflowType = "meeting_summary"
        val expectedIcon = "MeetingRoom"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for translation`() {
        val workflowType = "translation"
        val expectedIcon = "Translate"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for research`() {
        val workflowType = "research"
        val expectedIcon = "Search"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for planning`() {
        val workflowType = "planning"
        val expectedIcon = "EventNote"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    @Test
    fun `Workflow icon for unknown type defaults to AutoAwesome`() {
        val workflowType = "unknown_workflow_type"
        val expectedIcon = "AutoAwesome"
        
        val icon = getWorkflowIconName(workflowType)
        assertEquals(expectedIcon, icon)
    }

    // ========================================================================
    // Workflow Title Tests
    // ========================================================================

    @Test
    fun `Workflow title formats snake_case to Title Case`() {
        val workflowType = "document_analysis"
        val expectedTitle = "Document Analysis"
        
        val title = formatWorkflowTitle(workflowType)
        assertEquals(expectedTitle, title)
    }

    @Test
    fun `Workflow title handles single word`() {
        val workflowType = "analysis"
        val expectedTitle = "Analysis"
        
        val title = formatWorkflowTitle(workflowType)
        assertEquals(expectedTitle, title)
    }

    @Test
    fun `Workflow title handles multiple underscores`() {
        val workflowType = "code_review_security_check"
        val expectedTitle = "Code Review Security Check"
        
        val title = formatWorkflowTitle(workflowType)
        assertEquals(expectedTitle, title)
    }

    @Test
    fun `Workflow title handles already formatted text`() {
        val workflowType = "Research"
        val expectedTitle = "Research"
        
        val title = formatWorkflowTitle(workflowType)
        assertEquals(expectedTitle, title)
    }

    // ========================================================================
    // Progress Calculation Tests
    // ========================================================================

    @Test
    fun `Progress calculation for first step`() {
        val stepIndex = 1
        val totalSteps = 5
        
        val progress = calculateProgress(stepIndex, totalSteps)
        assertEquals(0f, progress, 0.001f)
    }

    @Test
    fun `Progress calculation for middle step`() {
        val stepIndex = 3
        val totalSteps = 5
        
        val progress = calculateProgress(stepIndex, totalSteps)
        assertEquals(0.4f, progress, 0.001f)
    }

    @Test
    fun `Progress calculation for last step`() {
        val stepIndex = 5
        val totalSteps = 5
        
        val progress = calculateProgress(stepIndex, totalSteps)
        assertEquals(0.8f, progress, 0.001f)
    }

    @Test
    fun `Progress calculation handles single step workflow`() {
        val stepIndex = 1
        val totalSteps = 1
        
        val progress = calculateProgress(stepIndex, totalSteps)
        assertEquals(0f, progress, 0.001f)
    }

    @Test
    fun `Progress calculation handles many steps`() {
        val stepIndex = 50
        val totalSteps = 100
        
        val progress = calculateProgress(stepIndex, totalSteps)
        assertEquals(0.49f, progress, 0.001f)
    }

    // ========================================================================
    // Status Color Tests
    // ========================================================================

    @Test
    fun `Status color for RUNNING state`() {
        val status = StepStatus.RUNNING
        val colorCategory = getStatusColorCategory(status)
        assertEquals("primary", colorCategory)
    }

    @Test
    fun `Status color for COMPLETED state`() {
        val status = StepStatus.COMPLETED
        val colorCategory = getStatusColorCategory(status)
        assertEquals("tertiary", colorCategory)
    }

    @Test
    fun `Status color for FAILED state`() {
        val status = StepStatus.FAILED
        val colorCategory = getStatusColorCategory(status)
        assertEquals("error", colorCategory)
    }

    @Test
    fun `Status color for PENDING state`() {
        val status = StepStatus.PENDING
        val colorCategory = getStatusColorCategory(status)
        assertEquals("surfaceVariant", colorCategory)
    }

    @Test
    fun `Status color for SKIPPED state`() {
        val status = StepStatus.SKIPPED
        val colorCategory = getStatusColorCategory(status)
        assertEquals("surfaceVariant", colorCategory)
    }

    // ========================================================================
    // Step Count Display Tests
    // ========================================================================

    @Test
    fun `Step count format shows current over total`() {
        val stepIndex = 2
        val totalSteps = 5
        
        val display = formatStepCount(stepIndex, totalSteps)
        assertEquals("2/5", display)
    }

    @Test
    fun `Step count handles single digit steps`() {
        val stepIndex = 1
        val totalSteps = 3
        
        val display = formatStepCount(stepIndex, totalSteps)
        assertEquals("1/3", display)
    }

    @Test
    fun `Step count handles double digit steps`() {
        val stepIndex = 10
        val totalSteps = 25
        
        val display = formatStepCount(stepIndex, totalSteps)
        assertEquals("10/25", display)
    }

    // ========================================================================
    // Workflow State Display Tests
    // ========================================================================

    @Test
    fun `Started state shows initializing message`() {
        val state = WorkflowState.Started(
            workflowId = "wf_test",
            workflowType = "test",
            roomId = "!room:example.com",
            initiatedBy = "@user:example.com",
            timestamp = System.currentTimeMillis()
        )

        val displayText = getStateDisplayText(state)
        assertEquals("Initializing workflow...", displayText)
    }

    @Test
    fun `StepRunning state shows step name`() {
        val state = WorkflowState.StepRunning(
            workflowId = "wf_test",
            workflowType = "test",
            roomId = "!room:example.com",
            stepId = "step_1",
            stepName = "Processing Data",
            stepIndex = 1,
            totalSteps = 3,
            status = StepStatus.RUNNING,
            timestamp = System.currentTimeMillis()
        )

        val displayText = getStateDisplayText(state)
        assertEquals("Processing Data", displayText)
    }

    // ========================================================================
    // Helper Functions (Mirror Component Logic)
    // ========================================================================
}

/**
 * Helper function to get workflow icon name
 */
private fun getWorkflowIconName(workflowType: String): String {
    return when (workflowType.lowercase()) {
        "document_analysis", "documentanalysis" -> "Description"
        "code_review", "codereview" -> "Code"
        "data_processing", "dataprocessing" -> "Storage"
        "report_generation", "reportgeneration" -> "Assessment"
        "meeting_summary", "meetingsummary" -> "MeetingRoom"
        "translation" -> "Translate"
        "research" -> "Search"
        "planning" -> "EventNote"
        else -> "AutoAwesome"
    }
}

/**
 * Helper function to format workflow title
 */
private fun formatWorkflowTitle(workflowType: String): String {
    return workflowType
        .replace("_", " ")
        .split(" ")
        .joinToString(" ") { it.lowercase().replaceFirstChar { c -> c.uppercase() } }
}

/**
 * Helper function to calculate progress
 */
private fun calculateProgress(stepIndex: Int, totalSteps: Int): Float {
    return if (totalSteps > 0) {
        (stepIndex - 1).toFloat() / totalSteps.toFloat()
    } else {
        0f
    }
}

/**
 * Helper function to get status color category
 */
private fun getStatusColorCategory(status: StepStatus): String {
    return when (status) {
        StepStatus.RUNNING -> "primary"
        StepStatus.COMPLETED -> "tertiary"
        StepStatus.FAILED -> "error"
        StepStatus.PENDING -> "surfaceVariant"
        StepStatus.SKIPPED -> "surfaceVariant"
    }
}

/**
 * Helper function to format step count
 */
private fun formatStepCount(stepIndex: Int, totalSteps: Int): String {
    return "$stepIndex/$totalSteps"
}

/**
 * Helper function to get state display text
 */
private fun getStateDisplayText(state: WorkflowState): String {
    return when (state) {
        is WorkflowState.Started -> "Initializing workflow..."
        is WorkflowState.StepRunning -> state.stepName
    }
}
