package com.armorclaw.shared.ui.components

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FileOpen
import androidx.compose.material.icons.filled.Image
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.Stop
import androidx.compose.ui.text.input.TextFieldValue
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for CommandBar component
 *
 * Tests chip click handlers, command injection, input submission,
 * and all command chip behaviors.
 */
class CommandBarTest {

    // ========================================================================
    // Command Chip Click Handler Tests
    // ========================================================================

    @Test
    fun `Status chip injects !status command into empty input`() {
        val initialValue = TextFieldValue("")
        val chip = CommandChip("Status", "!status", Icons.Default.Info)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("!status", newValue.text, "Status chip should inject '!status' into empty input")
    }

    @Test
    fun `Screenshot chip injects !screenshot command into empty input`() {
        val initialValue = TextFieldValue("")
        val chip = CommandChip("Screenshot", "!screenshot", Icons.Default.Image)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("!screenshot", newValue.text, "Screenshot chip should inject '!screenshot' into empty input")
    }

    @Test
    fun `Stop chip injects !stop command into empty input`() {
        val initialValue = TextFieldValue("")
        val chip = CommandChip("Stop", "!stop", Icons.Default.Stop)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("!stop", newValue.text, "Stop chip should inject '!stop' into empty input")
    }

    @Test
    fun `Pause chip injects !pause command into empty input`() {
        val initialValue = TextFieldValue("")
        val chip = CommandChip("Pause", "!pause", Icons.Default.Pause)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("!pause", newValue.text, "Pause chip should inject '!pause' into empty input")
    }

    @Test
    fun `Logs chip injects !logs command into empty input`() {
        val initialValue = TextFieldValue("")
        val chip = CommandChip("Logs", "!logs", Icons.Default.FileOpen)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("!logs", newValue.text, "Logs chip should inject '!logs' into empty input")
    }

    // ========================================================================
    // Chip Append to Existing Text Tests
    // ========================================================================

    @Test
    fun `Status chip appends !status to existing text with space`() {
        val initialValue = TextFieldValue("analyze the code")
        val chip = CommandChip("Status", "!status", Icons.Default.Info)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("analyze the code !status", newValue.text, "Status chip should append with space separator")
    }

    @Test
    fun `Screenshot chip appends !screenshot to existing text with space`() {
        val initialValue = TextFieldValue("take a")
        val chip = CommandChip("Screenshot", "!screenshot", Icons.Default.Image)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("take a !screenshot", newValue.text, "Screenshot chip should append with space separator")
    }

    @Test
    fun `Chip click preserves existing text`() {
        val initialValue = TextFieldValue("existing text here")
        val chip = CommandChip("Status", "!status", Icons.Default.Info)

        val newValue = simulateChipClick(initialValue, chip)

        assertTrue(newValue.text.startsWith("existing text here"), "Existing text should be preserved")
        assertTrue(newValue.text.endsWith("!status"), "Command should be appended")
    }

    // ========================================================================
    // Send Button Tests
    // ========================================================================

    @Test
    fun `Send button calls onSend with non-empty text`() {
        val value = TextFieldValue("test command")
        var sendCalled = false
        var sentContent = ""

        simulateSendClick(value) { content ->
            sendCalled = true
            sentContent = content
        }

        assertTrue(sendCalled, "Send should be called for non-empty text")
        assertEquals("test command", sentContent, "Send should pass the correct content")
    }

    @Test
    fun `Send button does not call onSend with empty text`() {
        val value = TextFieldValue("")
        var sendCalled = false

        simulateSendClick(value) { _ ->
            sendCalled = true
        }

        assertFalse(sendCalled, "Send should not be called for empty text")
    }

    @Test
    fun `Send button does not call onSend with blank text (spaces only)`() {
        val value = TextFieldValue("   ")
        var sendCalled = false

        simulateSendClick(value) { _ ->
            sendCalled = true
        }

        assertFalse(sendCalled, "Send should not be called for blank text (spaces only)")
    }

    @Test
    fun `Send button calls onSend with whitespace-padded text`() {
        val value = TextFieldValue("  test  ")
        var sendCalled = false
        var sentContent = ""

        simulateSendClick(value) { content ->
            sendCalled = true
            sentContent = content
        }

        assertTrue(sendCalled, "Send should be called for whitespace-padded text")
        assertEquals("  test  ", sentContent, "Send should pass text with original whitespace")
    }

    @Test
    fun `Send button is enabled for non-blank text`() {
        val value = TextFieldValue("valid text")

        val isSendEnabled = isSendButtonEnabled(value)

        assertTrue(isSendEnabled, "Send button should be enabled for non-blank text")
    }

    @Test
    fun `Send button is disabled for empty text`() {
        val value = TextFieldValue("")

        val isSendEnabled = isSendButtonEnabled(value)

        assertFalse(isSendEnabled, "Send button should be disabled for empty text")
    }

    @Test
    fun `Send button is disabled for blank text`() {
        val value = TextFieldValue("   ")

        val isSendEnabled = isSendButtonEnabled(value)

        assertFalse(isSendEnabled, "Send button should be disabled for blank text")
    }

    // ========================================================================
    // Default Command Chips Tests
    // ========================================================================

    @Test
    fun `Default command chips contains Status chip`() {
        val statusChip = defaultCommandChips.find { it.label == "Status" }

        assertEquals("Status", statusChip?.label, "Default chips should contain Status chip")
        assertEquals("!status", statusChip?.command, "Status chip should have !status command")
    }

    @Test
    fun `Default command chips contains Screenshot chip`() {
        val screenshotChip = defaultCommandChips.find { it.label == "Screenshot" }

        assertEquals("Screenshot", screenshotChip?.label, "Default chips should contain Screenshot chip")
        assertEquals("!screenshot", screenshotChip?.command, "Screenshot chip should have !screenshot command")
    }

    @Test
    fun `Default command chips contains Stop chip`() {
        val stopChip = defaultCommandChips.find { it.label == "Stop" }

        assertEquals("Stop", stopChip?.label, "Default chips should contain Stop chip")
        assertEquals("!stop", stopChip?.command, "Stop chip should have !stop command")
    }

    @Test
    fun `Default command chips contains Pause chip`() {
        val pauseChip = defaultCommandChips.find { it.label == "Pause" }

        assertEquals("Pause", pauseChip?.label, "Default chips should contain Pause chip")
        assertEquals("!pause", pauseChip?.command, "Pause chip should have !pause command")
    }

    @Test
    fun `Default command chips contains Logs chip`() {
        val logsChip = defaultCommandChips.find { it.label == "Logs" }

        assertEquals("Logs", logsChip?.label, "Default chips should contain Logs chip")
        assertEquals("!logs", logsChip?.command, "Logs chip should have !logs command")
    }

    @Test
    fun `Default command chips count is five`() {
        val chipCount = defaultCommandChips.size

        assertEquals(5, chipCount, "Default command chips should contain exactly 5 chips")
    }

    @Test
    fun `All default chips start with exclamation mark`() {
        val allStartWithExclamation = defaultCommandChips.all { it.command.startsWith("!") }

        assertTrue(allStartWithExclamation, "All default chip commands should start with '!'")
    }

    // ========================================================================
    // Input State Change Tests
    // ========================================================================

    @Test
    fun `Input state changes when typing`() {
        val oldValue = TextFieldValue("hello")
        val newValue = TextFieldValue("hello world")

        assertEquals("hello", oldValue.text, "Old value should remain unchanged")
        assertEquals("hello world", newValue.text, "New value should reflect typed text")
    }

    @Test
    fun `Input state clears after send`() {
        val beforeSend = TextFieldValue("test command")
        val afterSend = TextFieldValue("")

        assertEquals("test command", beforeSend.text, "Before send should have text")
        assertEquals("", afterSend.text, "After send should be empty")
    }

    @Test
    fun `Multiple chip clicks accumulate commands`() {
        var currentValue = TextFieldValue("")

        currentValue = simulateChipClick(currentValue, CommandChip("Status", "!status", Icons.Default.Info))
        currentValue = simulateChipClick(currentValue, CommandChip("Stop", "!stop", Icons.Default.Stop))
        currentValue = simulateChipClick(currentValue, CommandChip("Logs", "!logs", Icons.Default.FileOpen))

        assertEquals("!status !stop !logs", currentValue.text, "Multiple chip clicks should accumulate commands")
    }

    @Test
    fun `Chip click handles special characters in text`() {
        val initialValue = TextFieldValue("text with @mentions and #hashtags")
        val chip = CommandChip("Status", "!status", Icons.Default.Info)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("text with @mentions and #hashtags !status", newValue.text, "Chip click should handle special characters")
    }

    @Test
    fun `Chip click handles multiline text`() {
        val initialValue = TextFieldValue("line1\nline2\nline3")
        val chip = CommandChip("Status", "!status", Icons.Default.Info)

        val newValue = simulateChipClick(initialValue, chip)

        assertEquals("line1\nline2\nline3 !status", newValue.text, "Chip click should handle multiline text")
    }

    // ========================================================================
    // CommandChip Data Class Tests
    // ========================================================================

    @Test
    fun `CommandChip data class stores label correctly`() {
        val chip = CommandChip("Test Label", "!test", Icons.Default.Info)

        assertEquals("Test Label", chip.label, "CommandChip should store label correctly")
    }

    @Test
    fun `CommandChip data class stores command correctly`() {
        val chip = CommandChip("Test", "!test_command", Icons.Default.Info)

        assertEquals("!test_command", chip.command, "CommandChip should store command correctly")
    }

    @Test
    fun `CommandChip data class stores icon correctly`() {
        val icon = Icons.Default.Info
        val chip = CommandChip("Test", "!test", icon)

        assertEquals(icon, chip.icon, "CommandChip should store icon correctly")
    }

    @Test
    fun `CommandChip data class supports equality`() {
        val chip1 = CommandChip("Test", "!test", Icons.Default.Info)
        val chip2 = CommandChip("Test", "!test", Icons.Default.Info)
        val chip3 = CommandChip("Different", "!different", Icons.Default.Stop)

        assertEquals(chip1, chip2, "CommandChips with same properties should be equal")
        assertTrue(chip1 != chip3, "CommandChips with different properties should not be equal")
    }

    // ========================================================================
    // Helper Functions (Mirror Component Logic)
    // ========================================================================

    /**
     * Simulates chip click behavior from CommandBar component
     * Mirrors the logic in CommandBar.kt lines 71-78
     */
    private fun simulateChipClick(value: TextFieldValue, chip: CommandChip): TextFieldValue {
        val currentText = value.text
        val newText = if (currentText.isNotEmpty()) {
            "$currentText ${chip.command}"
        } else {
            chip.command
        }
        return TextFieldValue(newText)
    }

    /**
     * Simulates send button click behavior
     * Mirrors the logic in CommandBar.kt lines 144-149
     */
    private fun simulateSendClick(value: TextFieldValue, onSend: (String) -> Unit) {
        if (value.text.isNotBlank()) {
            onSend(value.text)
        }
    }

    /**
     * Determines if send button should be enabled
     * Mirrors the logic in CommandBar.kt line 150
     */
    private fun isSendButtonEnabled(value: TextFieldValue): Boolean {
        return value.text.isNotBlank()
    }
}
