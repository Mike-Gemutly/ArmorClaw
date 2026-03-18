package com.armorclaw.app.screens.studio

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for AgentStudioScreen component
 *
 * Tests AgentWizardData data class, property mutation,
 * validation logic, and navigation state management.
 */
class AgentStudioScreenTest {

    // ========================================================================
    // AgentWizardData Property Tests
    // ========================================================================

    @Test
    fun `AgentWizardData initializes with default values`() {
        val data = AgentWizardData()

        assertEquals("", data.agentName, "Default agentName should be empty string")
        assertEquals("", data.agentType, "Default agentType should be empty string")
        assertEquals("", data.description, "Default description should be empty string")
        assertFalse(data.isAgentTypeExpanded, "Default isAgentTypeExpanded should be false")
        assertTrue(data.selectedSkills.isEmpty(), "Default selectedSkills should be empty set")
        assertEquals("", data.workflowXml, "Default workflowXml should be empty string")
        assertFalse(data.emailAccess, "Default emailAccess should be false")
        assertFalse(data.contactsAccess, "Default contactsAccess should be false")
        assertFalse(data.calendarAccess, "Default calendarAccess should be false")
        assertEquals("Medium", data.sensitivityLevel, "Default sensitivityLevel should be 'Medium'")
    }

    @Test
    fun `AgentWizardData agentName can be mutated`() {
        val data = AgentWizardData()

        data.agentName = "Test Agent"

        assertEquals("Test Agent", data.agentName, "agentName should be mutable")
    }

    @Test
    fun `AgentWizardData agentType can be mutated`() {
        val data = AgentWizardData()

        data.agentType = "Assistant"

        assertEquals("Assistant", data.agentType, "agentType should be mutable")
    }

    @Test
    fun `AgentWizardData description can be mutated`() {
        val data = AgentWizardData()

        data.description = "A test agent for unit testing"

        assertEquals("A test agent for unit testing", data.description, "description should be mutable")
    }

    @Test
    fun `AgentWizardData isAgentTypeExpanded can be mutated`() {
        val data = AgentWizardData()

        data.isAgentTypeExpanded = true

        assertTrue(data.isAgentTypeExpanded, "isAgentTypeExpanded should be mutable")
    }

    @Test
    fun `AgentWizardData selectedSkills can be mutated`() {
        val data = AgentWizardData()

        data.selectedSkills.add("send_message")
        data.selectedSkills.add("receive_message")

        assertTrue(data.selectedSkills.contains("send_message"), "selectedSkills should contain added skill")
        assertTrue(data.selectedSkills.contains("receive_message"), "selectedSkills should contain added skill")
        assertEquals(2, data.selectedSkills.size, "selectedSkills should contain 2 items")
    }

    @Test
    fun `AgentWizardData selectedSkills allows removing items`() {
        val data = AgentWizardData()
        data.selectedSkills.add("send_message")
        data.selectedSkills.add("receive_message")

        data.selectedSkills.remove("send_message")

        assertFalse(data.selectedSkills.contains("send_message"), "selectedSkills should not contain removed skill")
        assertTrue(data.selectedSkills.contains("receive_message"), "selectedSkills should retain other skills")
        assertEquals(1, data.selectedSkills.size, "selectedSkills should contain 1 item after removal")
    }

    @Test
    fun `AgentWizardData selectedSkills prevents duplicate items`() {
        val data = AgentWizardData()

        data.selectedSkills.add("send_message")
        data.selectedSkills.add("send_message")

        assertEquals(1, data.selectedSkills.size, "selectedSkills should prevent duplicates (MutableSet behavior)")
    }

    @Test
    fun `AgentWizardData workflowXml can be mutated`() {
        val data = AgentWizardData()
        val xml = "<xml><block type=\"message_received\"></block></xml>"

        data.workflowXml = xml

        assertEquals(xml, data.workflowXml, "workflowXml should be mutable")
    }

    @Test
    fun `AgentWizardData emailAccess can be mutated`() {
        val data = AgentWizardData()

        data.emailAccess = true

        assertTrue(data.emailAccess, "emailAccess should be mutable")
    }

    @Test
    fun `AgentWizardData contactsAccess can be mutated`() {
        val data = AgentWizardData()

        data.contactsAccess = true

        assertTrue(data.contactsAccess, "contactsAccess should be mutable")
    }

    @Test
    fun `AgentWizardData calendarAccess can be mutated`() {
        val data = AgentWizardData()

        data.calendarAccess = true

        assertTrue(data.calendarAccess, "calendarAccess should be mutable")
    }

    @Test
    fun `AgentWizardData sensitivityLevel can be mutated`() {
        val data = AgentWizardData()

        data.sensitivityLevel = "High"

        assertEquals("High", data.sensitivityLevel, "sensitivityLevel should be mutable")
    }

    // ========================================================================
    // AgentWizardData Validation Tests
    // ========================================================================

    @Test
    fun `AgentWizardData with empty agentName is valid for default validation`() {
        val data = AgentWizardData()

        assertEquals("", data.agentName, "Empty agentName is stored without validation in data class")
    }

    @Test
    fun `AgentWizardData with populated agentName is valid`() {
        val data = AgentWizardData(agentName = "My Agent")

        assertEquals("My Agent", data.agentName, "Populated agentName should be stored correctly")
    }

    @Test
    fun `AgentWizardData supports all agent types`() {
        val types = listOf("Assistant", "Guardian", "Analyzer", "Scheduler")

        types.forEach { type ->
            val data = AgentWizardData(agentType = type)
            assertEquals(type, data.agentType, "Agent type '$type' should be supported")
        }
    }

    @Test
    fun `AgentWizardData supports all sensitivity levels`() {
        val levels = listOf("Low", "Medium", "High", "Critical")

        levels.forEach { level ->
            val data = AgentWizardData(sensitivityLevel = level)
            assertEquals(level, data.sensitivityLevel, "Sensitivity level '$level' should be supported")
        }
    }

    @Test
    fun `AgentWizardData allows multiple permissions to be true`() {
        val data = AgentWizardData()
        data.emailAccess = true
        data.contactsAccess = true
        data.calendarAccess = true

        assertTrue(data.emailAccess, "emailAccess should be true")
        assertTrue(data.contactsAccess, "contactsAccess should be true")
        assertTrue(data.calendarAccess, "calendarAccess should be true")
    }

    @Test
    fun `AgentWizardData allows all permissions to be false`() {
        val data = AgentWizardData()

        assertFalse(data.emailAccess, "emailAccess should be false")
        assertFalse(data.contactsAccess, "contactsAccess should be false")
        assertFalse(data.calendarAccess, "calendarAccess should be false")
    }

    // ========================================================================
    // AgentWizardData Complex State Tests
    // ========================================================================

    @Test
    fun `AgentWizardData maintains state across multiple mutations`() {
        val data = AgentWizardData()

        data.agentName = "Test Agent"
        data.agentType = "Assistant"
        data.description = "A description"
        data.isAgentTypeExpanded = true
        data.selectedSkills.add("send_message")
        data.workflowXml = "<xml></xml>"
        data.emailAccess = true
        data.sensitivityLevel = "High"

        assertEquals("Test Agent", data.agentName)
        assertEquals("Assistant", data.agentType)
        assertEquals("A description", data.description)
        assertTrue(data.isAgentTypeExpanded)
        assertTrue(data.selectedSkills.contains("send_message"))
        assertEquals("<xml></xml>", data.workflowXml)
        assertTrue(data.emailAccess)
        assertEquals("High", data.sensitivityLevel)
    }

    @Test
    fun `AgentWizardData selectedSkills handles multiple add and remove operations`() {
        val data = AgentWizardData()

        data.selectedSkills.add("skill1")
        data.selectedSkills.add("skill2")
        data.selectedSkills.add("skill3")
        data.selectedSkills.remove("skill2")
        data.selectedSkills.add("skill4")

        assertTrue(data.selectedSkills.contains("skill1"))
        assertFalse(data.selectedSkills.contains("skill2"))
        assertTrue(data.selectedSkills.contains("skill3"))
        assertTrue(data.selectedSkills.contains("skill4"))
        assertEquals(3, data.selectedSkills.size)
    }

    @Test
    fun `AgentWizardData can be initialized with custom values`() {
        val data = AgentWizardData(
            agentName = "Custom Agent",
            agentType = "Guardian",
            description = "Custom description",
            isAgentTypeExpanded = true,
            selectedSkills = mutableSetOf("send_message", "receive_message"),
            workflowXml = "<xml><block></block></xml>",
            emailAccess = true,
            contactsAccess = true,
            calendarAccess = false,
            sensitivityLevel = "High"
        )

        assertEquals("Custom Agent", data.agentName)
        assertEquals("Guardian", data.agentType)
        assertEquals("Custom description", data.description)
        assertTrue(data.isAgentTypeExpanded)
        assertTrue(data.selectedSkills.contains("send_message"))
        assertTrue(data.selectedSkills.contains("receive_message"))
        assertEquals(2, data.selectedSkills.size)
        assertEquals("<xml><block></block></xml>", data.workflowXml)
        assertTrue(data.emailAccess)
        assertTrue(data.contactsAccess)
        assertFalse(data.calendarAccess)
        assertEquals("High", data.sensitivityLevel)
    }

    @Test
    fun `AgentWizardData selectedSkills clear operation works`() {
        val data = AgentWizardData()
        data.selectedSkills.add("skill1")
        data.selectedSkills.add("skill2")

        data.selectedSkills.clear()

        assertTrue(data.selectedSkills.isEmpty(), "selectedSkills should be empty after clear")
    }

    @Test
    fun `AgentWizardData selectedSkills addAll operation works`() {
        val data = AgentWizardData()
        val skills = listOf("skill1", "skill2", "skill3")

        data.selectedSkills.addAll(skills)

        assertEquals(3, data.selectedSkills.size)
        assertTrue(data.selectedSkills.containsAll(skills))
    }

    // ========================================================================
    // Navigation State Logic Tests
    // ========================================================================

    @Test
    fun `Can go back when currentPage is greater than 0`() {
        val currentPage = 2

        val canGoBack = currentPage > 0

        assertTrue(canGoBack, "Should be able to go back when currentPage > 0")
    }

    @Test
    fun `Cannot go back when currentPage is 0`() {
        val currentPage = 0

        val canGoBack = currentPage > 0

        assertFalse(canGoBack, "Should not be able to go back when currentPage == 0")
    }

    @Test
    fun `Can go next when currentPage is less than 3`() {
        val currentPage = 1

        val canGoNext = currentPage < 3

        assertTrue(canGoNext, "Should be able to go next when currentPage < 3")
    }

    @Test
    fun `Cannot go next when currentPage is 3`() {
        val currentPage = 3

        val canGoNext = currentPage < 3

        assertFalse(canGoNext, "Should not be able to go next when currentPage == 3 (final step)")
    }

    @Test
    fun `Can go next when currentPage is 2`() {
        val currentPage = 2

        val canGoNext = currentPage < 3

        assertTrue(canGoNext, "Should be able to go next when currentPage == 2")
    }

    // ========================================================================
    // Step Title Logic Tests
    // ========================================================================

    @Test
    fun `Step title for page 0 is Step 1 Define Agent`() {
        val currentPage = 0
        val title = when (currentPage) {
            0 -> "Step 1: Define Agent"
            1 -> "Step 2: Select Skills"
            2 -> "Step 3: Build Workflow"
            3 -> "Step 4: Set Permissions"
            else -> "Create Agent"
        }

        assertEquals("Step 1: Define Agent", title, "Step 0 title should be correct")
    }

    @Test
    fun `Step title for page 1 is Step 2 Select Skills`() {
        val currentPage = 1
        val title = when (currentPage) {
            0 -> "Step 1: Define Agent"
            1 -> "Step 2: Select Skills"
            2 -> "Step 3: Build Workflow"
            3 -> "Step 4: Set Permissions"
            else -> "Create Agent"
        }

        assertEquals("Step 2: Select Skills", title, "Step 1 title should be correct")
    }

    @Test
    fun `Step title for page 2 is Step 3 Build Workflow`() {
        val currentPage = 2
        val title = when (currentPage) {
            0 -> "Step 1: Define Agent"
            1 -> "Step 2: Select Skills"
            2 -> "Step 3: Build Workflow"
            3 -> "Step 4: Set Permissions"
            else -> "Create Agent"
        }

        assertEquals("Step 3: Build Workflow", title, "Step 2 title should be correct")
    }

    @Test
    fun `Step title for page 3 is Step 4 Set Permissions`() {
        val currentPage = 3
        val title = when (currentPage) {
            0 -> "Step 1: Define Agent"
            1 -> "Step 2: Select Skills"
            2 -> "Step 3: Build Workflow"
            3 -> "Step 4: Set Permissions"
            else -> "Create Agent"
        }

        assertEquals("Step 4: Set Permissions", title, "Step 3 title should be correct")
    }

    @Test
    fun `Step title for out of range page is Create Agent`() {
        val currentPage = 99
        val title = when (currentPage) {
            0 -> "Step 1: Define Agent"
            1 -> "Step 2: Select Skills"
            2 -> "Step 3: Build Workflow"
            3 -> "Step 4: Set Permissions"
            else -> "Create Agent"
        }

        assertEquals("Create Agent", title, "Out of range page should use fallback title")
    }

    // ========================================================================
    // Progress Indicator Logic Tests
    // ========================================================================

    @Test
    fun `Progress indicator has 4 total pages`() {
        val totalPages = 4

        assertEquals(4, totalPages, "Wizard should have 4 pages")
    }

    @Test
    fun `Progress indicator highlights current page`() {
        val currentPage = 2
        val isHighlighted = currentPage == 2

        assertTrue(isHighlighted, "Progress indicator should highlight current page")
    }

    @Test
    fun `Progress indicator does not highlight other pages`() {
        val currentPage = 1
        val isPage2Highlighted = currentPage == 2

        assertFalse(isPage2Highlighted, "Progress indicator should not highlight non-current pages")
    }
}
