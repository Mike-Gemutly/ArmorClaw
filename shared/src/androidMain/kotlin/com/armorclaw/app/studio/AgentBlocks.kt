package com.armorclaw.app.studio

import kotlinx.serialization.Serializable

/**
 * Agent Block Library
 *
 * Defines all available blocks for agent workflow automation in the Agent Studio.
 * Blocks are organized into categories: Triggers, Actions, Logic, Control Flow, Variables, API Calls.
 *
 * Each block follows the Blockly JSON schema format:
 * - type: Unique block identifier
 * - message0: Display text with placeholders
 * - args0: Input parameters (values/fields)
 * - output: Output type if applicable
 * - colour: Block category color (hex)
 * - tooltip: Help text
 *
 * ## Categories
 * - Triggers (Blue): Events that start workflows
 * - Actions (Green): Operations that perform work
 * - Logic (Yellow): Conditional and iterative logic
 * - Control Flow (Red): Loop control and agent lifecycle
 * - Variables (Purple): Data storage and retrieval
 * - API Calls (Cyan): External system interactions
 */

// ========================================
// BLOCK CATEGORIES
// ========================================

/**
 * Block category for organization and color coding
 */
@Serializable
enum class BlockCategory(val displayName: String, val colorHex: String) {
    TRIGGERS("Triggers", "#4A90E2"),           // Blue
    ACTIONS("Actions", "#50E3C2"),            // Green
    LOGIC("Logic", "#F5A623"),                // Yellow
    CONTROL_FLOW("Control Flow", "#E04F5F"),  // Red
    VARIABLES("Variables", "#BD10E0"),        // Purple
    API_CALLS("API Calls", "#00B5D8")         // Cyan
}

/**
 * Output types for type-safe block connections
 */
@Serializable
enum class BlockOutputType {
    BOOLEAN,
    NUMBER,
    STRING,
    ANY,
    NONE
}

// ========================================
// TRIGGER BLOCKS
// ========================================

/**
 * Trigger: Message Received
 * Fires when a message arrives in a monitored channel
 */
@Serializable
data class MessageReceivedBlock(
    val type: String = "message_received",
    val message0: String = "when message received from %1 in channel %2",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.TRIGGERS.colorHex,
    val tooltip: String = "Fires when a message is received from a specific user or channel",
    val category: BlockCategory = BlockCategory.TRIGGERS,
    val subcategory: String = "Events"
)

/**
 * Trigger: User Joins
 * Fires when a user joins a room or channel
 */
@Serializable
data class UserJoinsBlock(
    val type: String = "user_joins",
    val message0: String = "when user %1 joins channel %2",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.TRIGGERS.colorHex,
    val tooltip: String = "Fires when a user joins a channel",
    val category: BlockCategory = BlockCategory.TRIGGERS,
    val subcategory: String = "Events"
)

/**
 * Trigger: Timer Expired
 * Fires after a specified time interval
 */
@Serializable
data class TimerExpiredBlock(
    val type: String = "timer_expired",
    val message0: String = "after %1 milliseconds",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.TRIGGERS.colorHex,
    val tooltip: String = "Fires after the specified time interval",
    val category: BlockCategory = BlockCategory.TRIGGERS,
    val subcategory: String = "Timing"
)

/**
 * Trigger: Schedule Triggered
 * Fires at a specific time or schedule
 */
@Serializable
data class ScheduleTriggeredBlock(
    val type: String = "schedule_triggered",
    val message0: String = "at %1 (cron: %2)",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.TRIGGERS.colorHex,
    val tooltip: String = "Fires at the scheduled time (supports cron expressions)",
    val category: BlockCategory = BlockCategory.TRIGGERS,
    val subcategory: String = "Timing"
)

// ========================================
// ACTION BLOCKS
// ========================================

/**
 * Action: Send Message
 * Sends a message to a channel or user
 */
@Serializable
data class SendMessageBlock(
    val type: String = "send_message",
    val message0: String = "send message %1 to channel %2",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.ACTIONS.colorHex,
    val tooltip: String = "Sends a text message to the specified channel",
    val category: BlockCategory = BlockCategory.ACTIONS,
    val subcategory: String = "Communication"
)

/**
 * Action: Send Email
 * Sends an email notification
 */
@Serializable
data class SendEmailBlock(
    val type: String = "send_email",
    val message0: String = "send email to %1 subject %2 body %3",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.ACTIONS.colorHex,
    val tooltip: String = "Sends an email with the specified subject and body",
    val category: BlockCategory = BlockCategory.ACTIONS,
    val subcategory: String = "Communication"
)

/**
 * Action: API Call
 * Makes a generic API call
 */
@Serializable
data class ApiCallBlock(
    val type: String = "api_call",
    val message0: String = "call API %1 with method %2 body %3",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.ACTIONS.colorHex,
    val tooltip: String = "Makes an HTTP API call with the specified method and body",
    val category: BlockCategory = BlockCategory.ACTIONS,
    val subcategory: String = "Integration"
)

/**
 * Action: Run Command
 * Executes a shell command or script
 */
@Serializable
data class RunCommandBlock(
    val type: String = "run_command",
    val message0: String = "run command %1 with timeout %2",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.ACTIONS.colorHex,
    val tooltip: String = "Executes a shell command with the specified timeout",
    val category: BlockCategory = BlockCategory.ACTIONS,
    val subcategory: String = "System"
)

/**
 * Action: Conditional Response
 * Sends different responses based on conditions
 */
@Serializable
data class ConditionalResponseBlock(
    val type: String = "conditional_response",
    val message0: String = "if %1 then send %2 else send %3",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.ACTIONS.colorHex,
    val tooltip: String = "Sends different responses based on the condition",
    val category: BlockCategory = BlockCategory.ACTIONS,
    val subcategory: String = "Logic"
)

// ========================================
// LOGIC BLOCKS
// ========================================

/**
 * Logic: If Then Else
 * Conditional branching logic
 */
@Serializable
data class IfThenElseBlock(
    val type: String = "if_then_else",
    val message0: String = "if %1 then %2 else %3",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.LOGIC.colorHex,
    val tooltip: String = "Executes different blocks based on the condition",
    val category: BlockCategory = BlockCategory.LOGIC,
    val subcategory: String = "Conditional"
)

/**
 * Logic: Repeat
 * Repeats a block of code a specified number of times
 */
@Serializable
data class RepeatBlock(
    val type: String = "repeat",
    val message0: String = "repeat %1 times %2",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.LOGIC.colorHex,
    val tooltip: String = "Repeats the contained block the specified number of times",
    val category: BlockCategory = BlockCategory.LOGIC,
    val subcategory: String = "Loops"
)

/**
 * Logic: Wait
 * Pauses execution for a specified duration
 */
@Serializable
data class WaitBlock(
    val type: String = "wait",
    val message0: String = "wait %1 milliseconds",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.LOGIC.colorHex,
    val tooltip: String = "Pauses execution for the specified duration",
    val category: BlockCategory = BlockCategory.LOGIC,
    val subcategory: String = "Timing"
)

/**
 * Logic: Parallel Execute
 * Executes multiple blocks in parallel
 */
@Serializable
data class ParallelExecuteBlock(
    val type: String = "parallel_execute",
    val message0: String = "execute in parallel %1",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.LOGIC.colorHex,
    val tooltip: String = "Executes multiple blocks concurrently",
    val category: BlockCategory = BlockCategory.LOGIC,
    val subcategory: String = "Concurrency"
)

// ========================================
// CONTROL FLOW BLOCKS
// ========================================

/**
 * Control Flow: Loop Break
 * Exits the current loop
 */
@Serializable
data class LoopBreakBlock(
    val type: String = "loop_break",
    val message0: String = "break loop",
    val colour: String = BlockCategory.CONTROL_FLOW.colorHex,
    val tooltip: String = "Exits the current loop immediately",
    val category: BlockCategory = BlockCategory.CONTROL_FLOW,
    val subcategory: String = "Loops"
)

/**
 * Control Flow: Continue
 * Skips to the next iteration of the current loop
 */
@Serializable
data class ContinueBlock(
    val type: String = "continue",
    val message0: String = "continue to next iteration",
    val colour: String = BlockCategory.CONTROL_FLOW.colorHex,
    val tooltip: String = "Skips the rest of the current loop iteration",
    val category: BlockCategory = BlockCategory.CONTROL_FLOW,
    val subcategory: String = "Loops"
)

/**
 * Control Flow: Stop Agent
 * Stops the agent execution
 */
@Serializable
data class StopAgentBlock(
    val type: String = "stop_agent",
    val message0: String = "stop agent execution",
    val colour: String = BlockCategory.CONTROL_FLOW.colorHex,
    val tooltip: String = "Stops the agent and terminates the workflow",
    val category: BlockCategory = BlockCategory.CONTROL_FLOW,
    val subcategory: String = "Lifecycle"
)

/**
 * Control Flow: Pause Agent
 * Pauses the agent execution
 */
@Serializable
data class PauseAgentBlock(
    val type: String = "pause_agent",
    val message0: String = "pause agent for %1 milliseconds",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.CONTROL_FLOW.colorHex,
    val tooltip: String = "Pauses the agent for the specified duration",
    val category: BlockCategory = BlockCategory.CONTROL_FLOW,
    val subcategory: String = "Lifecycle"
)

// ========================================
// VARIABLE BLOCKS
// ========================================

/**
 * Variable: Set Variable
 * Stores a value in a variable
 */
@Serializable
data class SetVariableBlock(
    val type: String = "set_variable",
    val message0: String = "set variable %1 to %2",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.VARIABLES.colorHex,
    val tooltip: String = "Stores a value in the specified variable",
    val category: BlockCategory = BlockCategory.VARIABLES,
    val subcategory: String = "Storage"
)

/**
 * Variable: Get Variable
 * Retrieves a value from a variable
 */
@Serializable
data class GetVariableBlock(
    val type: String = "get_variable",
    val message0: String = "get variable %1",
    val args0: List<BlockArgument>,
    val output: BlockOutputType = BlockOutputType.ANY,
    val colour: String = BlockCategory.VARIABLES.colorHex,
    val tooltip: String = "Retrieves the value of the specified variable",
    val category: BlockCategory = BlockCategory.VARIABLES,
    val subcategory: String = "Storage"
)

/**
 * Variable: Increment Variable
 * Increments a numeric variable
 */
@Serializable
data class IncrementVariableBlock(
    val type: String = "increment_variable",
    val message0: String = "increment variable %1 by %2",
    val args0: List<BlockArgument>,
    val colour: String = BlockCategory.VARIABLES.colorHex,
    val tooltip: String = "Increments the numeric variable by the specified amount",
    val category: BlockCategory = BlockCategory.VARIABLES,
    val subcategory: String = "Math"
)

// ========================================
// API CALL BLOCKS
// ========================================

/**
 * API Call: HTTP Request
 * Makes a raw HTTP request
 */
@Serializable
data class HttpRequestBlock(
    val type: String = "http_request",
    val message0: String = "HTTP %1 %2 headers %3",
    val args0: List<BlockArgument>,
    val output: BlockOutputType = BlockOutputType.STRING,
    val colour: String = BlockCategory.API_CALLS.colorHex,
    val tooltip: String = "Makes a raw HTTP request with the specified method and headers",
    val category: BlockCategory = BlockCategory.API_CALLS,
    val subcategory: String = "HTTP"
)

/**
 * API Call: Database Query
 * Executes a database query
 */
@Serializable
data class DatabaseQueryBlock(
    val type: String = "database_query",
    val message0: String = "query database %1 with SQL %2",
    val args0: List<BlockArgument>,
    val output: BlockOutputType = BlockOutputType.ANY,
    val colour: String = BlockCategory.API_CALLS.colorHex,
    val tooltip: String = "Executes a SQL query on the specified database",
    val category: BlockCategory = BlockCategory.API_CALLS,
    val subcategory: String = "Database"
)

// ========================================
// SUPPORTING MODELS
// ========================================

/**
 * Block argument definition (for args0, args1, etc.)
 */
@Serializable
data class BlockArgument(
    val type: String,  // "field_value", "field_input", "input_value", "input_statement"
    val name: String,  // Argument name for reference
    val text: String? = null,  // Placeholder text
    val check: BlockOutputType? = null  // Type constraint for input_value
)

/**
 * Complete block definition for serialization
 */
@Serializable
data class BlockDefinition(
    val type: String,
    val message0: String? = null,
    val message1: String? = null,
    val args0: List<BlockArgument>? = null,
    val args1: List<BlockArgument>? = null,
    val output: BlockOutputType? = null,
    val colour: String,
    val tooltip: String,
    val category: BlockCategory,
    val subcategory: String,
    val previousStatement: Boolean = false,
    val nextStatement: Boolean = false
)

/**
 * Registry of all available blocks
 */
object AgentBlockRegistry {
    val allBlocks: List<BlockDefinition> = listOf(
        // Trigger blocks
        MessageReceivedBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "USER", text = "user"),
                BlockArgument(type = "field_input", name = "CHANNEL", text = "channel")
            )
        ).toDefinition(),

        UserJoinsBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "USER", text = "user"),
                BlockArgument(type = "field_input", name = "CHANNEL", text = "channel")
            )
        ).toDefinition(),

        TimerExpiredBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "MILLISECONDS", text = "1000")
            )
        ).toDefinition(),

        ScheduleTriggeredBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "TIME", text = "12:00"),
                BlockArgument(type = "field_input", name = "CRON", text = "0 12 * * *")
            )
        ).toDefinition(),

        // Action blocks
        SendMessageBlock(
            args0 = listOf(
                BlockArgument(type = "input_value", name = "MESSAGE", text = "message"),
                BlockArgument(type = "field_input", name = "CHANNEL", text = "channel")
            )
        ).toDefinition(),

        SendEmailBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "TO", text = "recipient@example.com"),
                BlockArgument(type = "field_input", name = "SUBJECT", text = "Subject"),
                BlockArgument(type = "input_value", name = "BODY", text = "body")
            )
        ).toDefinition(),

        ApiCallBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "URL", text = "https://api.example.com"),
                BlockArgument(type = "field_dropdown", name = "METHOD", text = "POST"),
                BlockArgument(type = "input_value", name = "BODY", text = "body")
            )
        ).toDefinition(),

        RunCommandBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "COMMAND", text = "echo 'hello'"),
                BlockArgument(type = "field_number", name = "TIMEOUT", text = "5000")
            )
        ).toDefinition(),

        ConditionalResponseBlock(
            args0 = listOf(
                BlockArgument(type = "input_value", name = "CONDITION", check = BlockOutputType.BOOLEAN, text = "condition"),
                BlockArgument(type = "input_value", name = "THEN", text = "then message"),
                BlockArgument(type = "input_value", name = "ELSE", text = "else message")
            )
        ).toDefinition(),

        // Logic blocks
        IfThenElseBlock(
            args0 = listOf(
                BlockArgument(type = "input_value", name = "CONDITION", check = BlockOutputType.BOOLEAN, text = "condition"),
                BlockArgument(type = "input_statement", name = "THEN", text = "then"),
                BlockArgument(type = "input_statement", name = "ELSE", text = "else")
            )
        ).toDefinition(),

        RepeatBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "TIMES", text = "5"),
                BlockArgument(type = "input_statement", name = "DO", text = "do")
            )
        ).toDefinition(),

        WaitBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "MILLISECONDS", text = "1000")
            )
        ).toDefinition(),

        ParallelExecuteBlock(
            args0 = listOf(
                BlockArgument(type = "input_statement", name = "BLOCKS", text = "blocks")
            )
        ).toDefinition(),

        // Control Flow blocks
        LoopBreakBlock().toDefinition(),

        ContinueBlock().toDefinition(),

        StopAgentBlock().toDefinition(),

        PauseAgentBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "MILLISECONDS", text = "5000")
            )
        ).toDefinition(),

        // Variable blocks
        SetVariableBlock(
            args0 = listOf(
                BlockArgument(type = "field_variable", name = "VAR", text = "variable"),
                BlockArgument(type = "input_value", name = "VALUE", text = "value")
            )
        ).toDefinition(),

        GetVariableBlock(
            args0 = listOf(
                BlockArgument(type = "field_variable", name = "VAR", text = "variable")
            )
        ).toDefinition(),

        IncrementVariableBlock(
            args0 = listOf(
                BlockArgument(type = "field_variable", name = "VAR", text = "counter"),
                BlockArgument(type = "field_number", name = "AMOUNT", text = "1")
            )
        ).toDefinition(),

        // API Call blocks
        HttpRequestBlock(
            args0 = listOf(
                BlockArgument(type = "field_dropdown", name = "METHOD", text = "GET"),
                BlockArgument(type = "field_input", name = "URL", text = "https://api.example.com"),
                BlockArgument(type = "input_value", name = "HEADERS", text = "headers")
            )
        ).toDefinition(),

        DatabaseQueryBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "DATABASE", text = "mydb"),
                BlockArgument(type = "input_value", name = "SQL", text = "SELECT * FROM users")
            )
        ).toDefinition()
    )

    /**
     * Get blocks by category
     */
    fun getBlocksByCategory(category: BlockCategory): List<BlockDefinition> {
        return allBlocks.filter { it.category == category }
    }

    /**
     * Get block by type
     */
    fun getBlockByType(type: String): BlockDefinition? {
        return allBlocks.find { it.type == type }
    }
}

// ========================================
// EXTENSION FUNCTIONS
// ========================================

/**
 * Extension function to convert typed block blocks to BlockDefinition
 */
private fun MessageReceivedBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = false,
    nextStatement = true
)

private fun UserJoinsBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = false,
    nextStatement = true
)

private fun TimerExpiredBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = false,
    nextStatement = true
)

private fun ScheduleTriggeredBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = false,
    nextStatement = true
)

private fun SendMessageBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun SendEmailBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun ApiCallBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun RunCommandBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun ConditionalResponseBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun IfThenElseBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun RepeatBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun WaitBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun ParallelExecuteBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun LoopBreakBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = false
)

private fun ContinueBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = false
)

private fun StopAgentBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = false
)

private fun PauseAgentBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun SetVariableBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun GetVariableBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    output = output,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = false,
    nextStatement = false
)

private fun IncrementVariableBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun HttpRequestBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    output = output,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)

private fun DatabaseQueryBlock.toDefinition() = BlockDefinition(
    type = type,
    message0 = message0,
    args0 = args0,
    output = output,
    colour = colour,
    tooltip = tooltip,
    category = category,
    subcategory = subcategory,
    previousStatement = true,
    nextStatement = true
)
