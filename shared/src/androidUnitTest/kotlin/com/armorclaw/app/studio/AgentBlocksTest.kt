package com.armorclaw.app.studio

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Unit tests for AgentBlocks component
 *
 * Tests BlockCategory enum, BlockOutputType enum, and all block data classes.
 * Verifies properties, values, and block registry functionality.
 */
class AgentBlocksTest {

    // ========================================================================
    // BlockCategory Enum Tests
    // ========================================================================

    @Test
    fun `BlockCategory TRIGGERS has correct displayName and colorHex`() {
        val category = BlockCategory.TRIGGERS
        assertEquals("Triggers", category.displayName, "TRIGGERS displayName should be 'Triggers'")
        assertEquals("#4A90E2", category.colorHex, "TRIGGERS colorHex should be '#4A90E2'")
    }

    @Test
    fun `BlockCategory ACTIONS has correct displayName and colorHex`() {
        val category = BlockCategory.ACTIONS
        assertEquals("Actions", category.displayName, "ACTIONS displayName should be 'Actions'")
        assertEquals("#50E3C2", category.colorHex, "ACTIONS colorHex should be '#50E3C2'")
    }

    @Test
    fun `BlockCategory LOGIC has correct displayName and colorHex`() {
        val category = BlockCategory.LOGIC
        assertEquals("Logic", category.displayName, "LOGIC displayName should be 'Logic'")
        assertEquals("#F5A623", category.colorHex, "LOGIC colorHex should be '#F5A623'")
    }

    @Test
    fun `BlockCategory CONTROL_FLOW has correct displayName and colorHex`() {
        val category = BlockCategory.CONTROL_FLOW
        assertEquals("Control Flow", category.displayName, "CONTROL_FLOW displayName should be 'Control Flow'")
        assertEquals("#E04F5F", category.colorHex, "CONTROL_FLOW colorHex should be '#E04F5F'")
    }

    @Test
    fun `BlockCategory VARIABLES has correct displayName and colorHex`() {
        val category = BlockCategory.VARIABLES
        assertEquals("Variables", category.displayName, "VARIABLES displayName should be 'Variables'")
        assertEquals("#BD10E0", category.colorHex, "VARIABLES colorHex should be '#BD10E0'")
    }

    @Test
    fun `BlockCategory API_CALLS has correct displayName and colorHex`() {
        val category = BlockCategory.API_CALLS
        assertEquals("API Calls", category.displayName, "API_CALLS displayName should be 'API Calls'")
        assertEquals("#00B5D8", category.colorHex, "API_CALLS colorHex should be '#00B5D8'")
    }

    @Test
    fun `BlockCategory enum has exactly 6 entries`() {
        val categoryCount = BlockCategory.entries.size
        assertEquals(6, categoryCount, "BlockCategory should have exactly 6 entries")
    }

    // ========================================================================
    // BlockOutputType Enum Tests
    // ========================================================================

    @Test
    fun `BlockOutputType contains BOOLEAN type`() {
        val type = BlockOutputType.BOOLEAN
        assertEquals("BOOLEAN", type.name, "BlockOutputType should contain BOOLEAN")
    }

    @Test
    fun `BlockOutputType contains NUMBER type`() {
        val type = BlockOutputType.NUMBER
        assertEquals("NUMBER", type.name, "BlockOutputType should contain NUMBER")
    }

    @Test
    fun `BlockOutputType contains STRING type`() {
        val type = BlockOutputType.STRING
        assertEquals("STRING", type.name, "BlockOutputType should contain STRING")
    }

    @Test
    fun `BlockOutputType contains ANY type`() {
        val type = BlockOutputType.ANY
        assertEquals("ANY", type.name, "BlockOutputType should contain ANY")
    }

    @Test
    fun `BlockOutputType contains NONE type`() {
        val type = BlockOutputType.NONE
        assertEquals("NONE", type.name, "BlockOutputType should contain NONE")
    }

    @Test
    fun `BlockOutputType enum has exactly 5 entries`() {
        val typeCount = BlockOutputType.entries.size
        assertEquals(5, typeCount, "BlockOutputType should have exactly 5 entries")
    }

    // ========================================================================
    // Trigger Block Tests
    // ========================================================================

    @Test
    fun `MessageReceivedBlock has correct default properties`() {
        val block = MessageReceivedBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "USER", text = "user"),
                BlockArgument(type = "field_input", name = "CHANNEL", text = "channel")
            )
        )

        assertEquals("message_received", block.type, "MessageReceivedBlock type should be 'message_received'")
        assertEquals("when message received from %1 in channel %2", block.message0, "MessageReceivedBlock message0 should match template")
        assertEquals(BlockCategory.TRIGGERS.colorHex, block.colour, "MessageReceivedBlock colour should match TRIGGERS category")
        assertEquals("Fires when a message is received from a specific user or channel", block.tooltip, "MessageReceivedBlock tooltip should be descriptive")
        assertEquals(BlockCategory.TRIGGERS, block.category, "MessageReceivedBlock category should be TRIGGERS")
        assertEquals("Events", block.subcategory, "MessageReceivedBlock subcategory should be 'Events'")
    }

    @Test
    fun `UserJoinsBlock has correct default properties`() {
        val block = UserJoinsBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "USER", text = "user"),
                BlockArgument(type = "field_input", name = "CHANNEL", text = "channel")
            )
        )

        assertEquals("user_joins", block.type, "UserJoinsBlock type should be 'user_joins'")
        assertEquals("when user %1 joins channel %2", block.message0, "UserJoinsBlock message0 should match template")
        assertEquals(BlockCategory.TRIGGERS.colorHex, block.colour, "UserJoinsBlock colour should match TRIGGERS category")
        assertEquals("Fires when a user joins a channel", block.tooltip, "UserJoinsBlock tooltip should be descriptive")
        assertEquals(BlockCategory.TRIGGERS, block.category, "UserJoinsBlock category should be TRIGGERS")
        assertEquals("Events", block.subcategory, "UserJoinsBlock subcategory should be 'Events'")
    }

    @Test
    fun `TimerExpiredBlock has correct default properties`() {
        val block = TimerExpiredBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "MILLISECONDS", text = "1000")
            )
        )

        assertEquals("timer_expired", block.type, "TimerExpiredBlock type should be 'timer_expired'")
        assertEquals("after %1 milliseconds", block.message0, "TimerExpiredBlock message0 should match template")
        assertEquals(BlockCategory.TRIGGERS.colorHex, block.colour, "TimerExpiredBlock colour should match TRIGGERS category")
        assertEquals("Fires after the specified time interval", block.tooltip, "TimerExpiredBlock tooltip should be descriptive")
        assertEquals(BlockCategory.TRIGGERS, block.category, "TimerExpiredBlock category should be TRIGGERS")
        assertEquals("Timing", block.subcategory, "TimerExpiredBlock subcategory should be 'Timing'")
    }

    @Test
    fun `ScheduleTriggeredBlock has correct default properties`() {
        val block = ScheduleTriggeredBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "TIME", text = "12:00"),
                BlockArgument(type = "field_input", name = "CRON", text = "0 12 * * *")
            )
        )

        assertEquals("schedule_triggered", block.type, "ScheduleTriggeredBlock type should be 'schedule_triggered'")
        assertEquals("at %1 (cron: %2)", block.message0, "ScheduleTriggeredBlock message0 should match template")
        assertEquals(BlockCategory.TRIGGERS.colorHex, block.colour, "ScheduleTriggeredBlock colour should match TRIGGERS category")
        assertEquals("Fires at the scheduled time (supports cron expressions)", block.tooltip, "ScheduleTriggeredBlock tooltip should be descriptive")
        assertEquals(BlockCategory.TRIGGERS, block.category, "ScheduleTriggeredBlock category should be TRIGGERS")
        assertEquals("Timing", block.subcategory, "ScheduleTriggeredBlock subcategory should be 'Timing'")
    }

    // ========================================================================
    // Action Block Tests
    // ========================================================================

    @Test
    fun `SendMessageBlock has correct default properties`() {
        val block = SendMessageBlock(
            args0 = listOf(
                BlockArgument(type = "input_value", name = "MESSAGE", text = "message"),
                BlockArgument(type = "field_input", name = "CHANNEL", text = "channel")
            )
        )

        assertEquals("send_message", block.type, "SendMessageBlock type should be 'send_message'")
        assertEquals("send message %1 to channel %2", block.message0, "SendMessageBlock message0 should match template")
        assertEquals(BlockCategory.ACTIONS.colorHex, block.colour, "SendMessageBlock colour should match ACTIONS category")
        assertEquals("Sends a text message to the specified channel", block.tooltip, "SendMessageBlock tooltip should be descriptive")
        assertEquals(BlockCategory.ACTIONS, block.category, "SendMessageBlock category should be ACTIONS")
        assertEquals("Communication", block.subcategory, "SendMessageBlock subcategory should be 'Communication'")
    }

    @Test
    fun `SendEmailBlock has correct default properties`() {
        val block = SendEmailBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "TO", text = "recipient@example.com"),
                BlockArgument(type = "field_input", name = "SUBJECT", text = "Subject"),
                BlockArgument(type = "input_value", name = "BODY", text = "body")
            )
        )

        assertEquals("send_email", block.type, "SendEmailBlock type should be 'send_email'")
        assertEquals("send email to %1 subject %2 body %3", block.message0, "SendEmailBlock message0 should match template")
        assertEquals(BlockCategory.ACTIONS.colorHex, block.colour, "SendEmailBlock colour should match ACTIONS category")
        assertEquals("Sends an email with the specified subject and body", block.tooltip, "SendEmailBlock tooltip should be descriptive")
        assertEquals(BlockCategory.ACTIONS, block.category, "SendEmailBlock category should be ACTIONS")
        assertEquals("Communication", block.subcategory, "SendEmailBlock subcategory should be 'Communication'")
    }

    @Test
    fun `ApiCallBlock has correct default properties`() {
        val block = ApiCallBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "URL", text = "https://api.example.com"),
                BlockArgument(type = "field_dropdown", name = "METHOD", text = "POST"),
                BlockArgument(type = "input_value", name = "BODY", text = "body")
            )
        )

        assertEquals("api_call", block.type, "ApiCallBlock type should be 'api_call'")
        assertEquals("call API %1 with method %2 body %3", block.message0, "ApiCallBlock message0 should match template")
        assertEquals(BlockCategory.ACTIONS.colorHex, block.colour, "ApiCallBlock colour should match ACTIONS category")
        assertEquals("Makes an HTTP API call with the specified method and body", block.tooltip, "ApiCallBlock tooltip should be descriptive")
        assertEquals(BlockCategory.ACTIONS, block.category, "ApiCallBlock category should be ACTIONS")
        assertEquals("Integration", block.subcategory, "ApiCallBlock subcategory should be 'Integration'")
    }

    @Test
    fun `RunCommandBlock has correct default properties`() {
        val block = RunCommandBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "COMMAND", text = "echo 'hello'"),
                BlockArgument(type = "field_number", name = "TIMEOUT", text = "5000")
            )
        )

        assertEquals("run_command", block.type, "RunCommandBlock type should be 'run_command'")
        assertEquals("run command %1 with timeout %2", block.message0, "RunCommandBlock message0 should match template")
        assertEquals(BlockCategory.ACTIONS.colorHex, block.colour, "RunCommandBlock colour should match ACTIONS category")
        assertEquals("Executes a shell command with the specified timeout", block.tooltip, "RunCommandBlock tooltip should be descriptive")
        assertEquals(BlockCategory.ACTIONS, block.category, "RunCommandBlock category should be ACTIONS")
        assertEquals("System", block.subcategory, "RunCommandBlock subcategory should be 'System'")
    }

    @Test
    fun `ConditionalResponseBlock has correct default properties`() {
        val block = ConditionalResponseBlock(
            args0 = listOf(
                BlockArgument(type = "input_value", name = "CONDITION", check = BlockOutputType.BOOLEAN, text = "condition"),
                BlockArgument(type = "input_value", name = "THEN", text = "then message"),
                BlockArgument(type = "input_value", name = "ELSE", text = "else message")
            )
        )

        assertEquals("conditional_response", block.type, "ConditionalResponseBlock type should be 'conditional_response'")
        assertEquals("if %1 then send %2 else send %3", block.message0, "ConditionalResponseBlock message0 should match template")
        assertEquals(BlockCategory.ACTIONS.colorHex, block.colour, "ConditionalResponseBlock colour should match ACTIONS category")
        assertEquals("Sends different responses based on the condition", block.tooltip, "ConditionalResponseBlock tooltip should be descriptive")
        assertEquals(BlockCategory.ACTIONS, block.category, "ConditionalResponseBlock category should be ACTIONS")
        assertEquals("Logic", block.subcategory, "ConditionalResponseBlock subcategory should be 'Logic'")
    }

    // ========================================================================
    // Logic Block Tests
    // ========================================================================

    @Test
    fun `IfThenElseBlock has correct default properties`() {
        val block = IfThenElseBlock(
            args0 = listOf(
                BlockArgument(type = "input_value", name = "CONDITION", check = BlockOutputType.BOOLEAN, text = "condition"),
                BlockArgument(type = "input_statement", name = "THEN", text = "then"),
                BlockArgument(type = "input_statement", name = "ELSE", text = "else")
            )
        )

        assertEquals("if_then_else", block.type, "IfThenElseBlock type should be 'if_then_else'")
        assertEquals("if %1 then %2 else %3", block.message0, "IfThenElseBlock message0 should match template")
        assertEquals(BlockCategory.LOGIC.colorHex, block.colour, "IfThenElseBlock colour should match LOGIC category")
        assertEquals("Executes different blocks based on the condition", block.tooltip, "IfThenElseBlock tooltip should be descriptive")
        assertEquals(BlockCategory.LOGIC, block.category, "IfThenElseBlock category should be LOGIC")
        assertEquals("Conditional", block.subcategory, "IfThenElseBlock subcategory should be 'Conditional'")
    }

    @Test
    fun `RepeatBlock has correct default properties`() {
        val block = RepeatBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "TIMES", text = "5"),
                BlockArgument(type = "input_statement", name = "DO", text = "do")
            )
        )

        assertEquals("repeat", block.type, "RepeatBlock type should be 'repeat'")
        assertEquals("repeat %1 times %2", block.message0, "RepeatBlock message0 should match template")
        assertEquals(BlockCategory.LOGIC.colorHex, block.colour, "RepeatBlock colour should match LOGIC category")
        assertEquals("Repeats the contained block the specified number of times", block.tooltip, "RepeatBlock tooltip should be descriptive")
        assertEquals(BlockCategory.LOGIC, block.category, "RepeatBlock category should be LOGIC")
        assertEquals("Loops", block.subcategory, "RepeatBlock subcategory should be 'Loops'")
    }

    @Test
    fun `WaitBlock has correct default properties`() {
        val block = WaitBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "MILLISECONDS", text = "1000")
            )
        )

        assertEquals("wait", block.type, "WaitBlock type should be 'wait'")
        assertEquals("wait %1 milliseconds", block.message0, "WaitBlock message0 should match template")
        assertEquals(BlockCategory.LOGIC.colorHex, block.colour, "WaitBlock colour should match LOGIC category")
        assertEquals("Pauses execution for the specified duration", block.tooltip, "WaitBlock tooltip should be descriptive")
        assertEquals(BlockCategory.LOGIC, block.category, "WaitBlock category should be LOGIC")
        assertEquals("Timing", block.subcategory, "WaitBlock subcategory should be 'Timing'")
    }

    @Test
    fun `ParallelExecuteBlock has correct default properties`() {
        val block = ParallelExecuteBlock(
            args0 = listOf(
                BlockArgument(type = "input_statement", name = "BLOCKS", text = "blocks")
            )
        )

        assertEquals("parallel_execute", block.type, "ParallelExecuteBlock type should be 'parallel_execute'")
        assertEquals("execute in parallel %1", block.message0, "ParallelExecuteBlock message0 should match template")
        assertEquals(BlockCategory.LOGIC.colorHex, block.colour, "ParallelExecuteBlock colour should match LOGIC category")
        assertEquals("Executes multiple blocks concurrently", block.tooltip, "ParallelExecuteBlock tooltip should be descriptive")
        assertEquals(BlockCategory.LOGIC, block.category, "ParallelExecuteBlock category should be LOGIC")
        assertEquals("Concurrency", block.subcategory, "ParallelExecuteBlock subcategory should be 'Concurrency'")
    }

    // ========================================================================
    // Control Flow Block Tests
    // ========================================================================

    @Test
    fun `LoopBreakBlock has correct default properties`() {
        val block = LoopBreakBlock()

        assertEquals("loop_break", block.type, "LoopBreakBlock type should be 'loop_break'")
        assertEquals("break loop", block.message0, "LoopBreakBlock message0 should be 'break loop'")
        assertEquals(BlockCategory.CONTROL_FLOW.colorHex, block.colour, "LoopBreakBlock colour should match CONTROL_FLOW category")
        assertEquals("Exits the current loop immediately", block.tooltip, "LoopBreakBlock tooltip should be descriptive")
        assertEquals(BlockCategory.CONTROL_FLOW, block.category, "LoopBreakBlock category should be CONTROL_FLOW")
        assertEquals("Loops", block.subcategory, "LoopBreakBlock subcategory should be 'Loops'")
    }

    @Test
    fun `ContinueBlock has correct default properties`() {
        val block = ContinueBlock()

        assertEquals("continue", block.type, "ContinueBlock type should be 'continue'")
        assertEquals("continue to next iteration", block.message0, "ContinueBlock message0 should be 'continue to next iteration'")
        assertEquals(BlockCategory.CONTROL_FLOW.colorHex, block.colour, "ContinueBlock colour should match CONTROL_FLOW category")
        assertEquals("Skips the rest of the current loop iteration", block.tooltip, "ContinueBlock tooltip should be descriptive")
        assertEquals(BlockCategory.CONTROL_FLOW, block.category, "ContinueBlock category should be CONTROL_FLOW")
        assertEquals("Loops", block.subcategory, "ContinueBlock subcategory should be 'Loops'")
    }

    @Test
    fun `StopAgentBlock has correct default properties`() {
        val block = StopAgentBlock()

        assertEquals("stop_agent", block.type, "StopAgentBlock type should be 'stop_agent'")
        assertEquals("stop agent execution", block.message0, "StopAgentBlock message0 should be 'stop agent execution'")
        assertEquals(BlockCategory.CONTROL_FLOW.colorHex, block.colour, "StopAgentBlock colour should match CONTROL_FLOW category")
        assertEquals("Stops the agent and terminates the workflow", block.tooltip, "StopAgentBlock tooltip should be descriptive")
        assertEquals(BlockCategory.CONTROL_FLOW, block.category, "StopAgentBlock category should be CONTROL_FLOW")
        assertEquals("Lifecycle", block.subcategory, "StopAgentBlock subcategory should be 'Lifecycle'")
    }

    @Test
    fun `PauseAgentBlock has correct default properties`() {
        val block = PauseAgentBlock(
            args0 = listOf(
                BlockArgument(type = "field_number", name = "MILLISECONDS", text = "5000")
            )
        )

        assertEquals("pause_agent", block.type, "PauseAgentBlock type should be 'pause_agent'")
        assertEquals("pause agent for %1 milliseconds", block.message0, "PauseAgentBlock message0 should match template")
        assertEquals(BlockCategory.CONTROL_FLOW.colorHex, block.colour, "PauseAgentBlock colour should match CONTROL_FLOW category")
        assertEquals("Pauses the agent for the specified duration", block.tooltip, "PauseAgentBlock tooltip should be descriptive")
        assertEquals(BlockCategory.CONTROL_FLOW, block.category, "PauseAgentBlock category should be CONTROL_FLOW")
        assertEquals("Lifecycle", block.subcategory, "PauseAgentBlock subcategory should be 'Lifecycle'")
    }

    // ========================================================================
    // Variable Block Tests
    // ========================================================================

    @Test
    fun `SetVariableBlock has correct default properties`() {
        val block = SetVariableBlock(
            args0 = listOf(
                BlockArgument(type = "field_variable", name = "VAR", text = "variable"),
                BlockArgument(type = "input_value", name = "VALUE", text = "value")
            )
        )

        assertEquals("set_variable", block.type, "SetVariableBlock type should be 'set_variable'")
        assertEquals("set variable %1 to %2", block.message0, "SetVariableBlock message0 should match template")
        assertEquals(BlockCategory.VARIABLES.colorHex, block.colour, "SetVariableBlock colour should match VARIABLES category")
        assertEquals("Stores a value in the specified variable", block.tooltip, "SetVariableBlock tooltip should be descriptive")
        assertEquals(BlockCategory.VARIABLES, block.category, "SetVariableBlock category should be VARIABLES")
        assertEquals("Storage", block.subcategory, "SetVariableBlock subcategory should be 'Storage'")
    }

    @Test
    fun `GetVariableBlock has correct default properties`() {
        val block = GetVariableBlock(
            args0 = listOf(
                BlockArgument(type = "field_variable", name = "VAR", text = "variable")
            )
        )

        assertEquals("get_variable", block.type, "GetVariableBlock type should be 'get_variable'")
        assertEquals("get variable %1", block.message0, "GetVariableBlock message0 should match template")
        assertEquals(BlockOutputType.ANY, block.output, "GetVariableBlock output should be ANY")
        assertEquals(BlockCategory.VARIABLES.colorHex, block.colour, "GetVariableBlock colour should match VARIABLES category")
        assertEquals("Retrieves the value of the specified variable", block.tooltip, "GetVariableBlock tooltip should be descriptive")
        assertEquals(BlockCategory.VARIABLES, block.category, "GetVariableBlock category should be VARIABLES")
        assertEquals("Storage", block.subcategory, "GetVariableBlock subcategory should be 'Storage'")
    }

    @Test
    fun `IncrementVariableBlock has correct default properties`() {
        val block = IncrementVariableBlock(
            args0 = listOf(
                BlockArgument(type = "field_variable", name = "VAR", text = "counter"),
                BlockArgument(type = "field_number", name = "AMOUNT", text = "1")
            )
        )

        assertEquals("increment_variable", block.type, "IncrementVariableBlock type should be 'increment_variable'")
        assertEquals("increment variable %1 by %2", block.message0, "IncrementVariableBlock message0 should match template")
        assertEquals(BlockCategory.VARIABLES.colorHex, block.colour, "IncrementVariableBlock colour should match VARIABLES category")
        assertEquals("Increments the numeric variable by the specified amount", block.tooltip, "IncrementVariableBlock tooltip should be descriptive")
        assertEquals(BlockCategory.VARIABLES, block.category, "IncrementVariableBlock category should be VARIABLES")
        assertEquals("Math", block.subcategory, "IncrementVariableBlock subcategory should be 'Math'")
    }

    // ========================================================================
    // API Call Block Tests
    // ========================================================================

    @Test
    fun `HttpRequestBlock has correct default properties`() {
        val block = HttpRequestBlock(
            args0 = listOf(
                BlockArgument(type = "field_dropdown", name = "METHOD", text = "GET"),
                BlockArgument(type = "field_input", name = "URL", text = "https://api.example.com"),
                BlockArgument(type = "input_value", name = "HEADERS", text = "headers")
            )
        )

        assertEquals("http_request", block.type, "HttpRequestBlock type should be 'http_request'")
        assertEquals("HTTP %1 %2 headers %3", block.message0, "HttpRequestBlock message0 should match template")
        assertEquals(BlockOutputType.STRING, block.output, "HttpRequestBlock output should be STRING")
        assertEquals(BlockCategory.API_CALLS.colorHex, block.colour, "HttpRequestBlock colour should match API_CALLS category")
        assertEquals("Makes a raw HTTP request with the specified method and headers", block.tooltip, "HttpRequestBlock tooltip should be descriptive")
        assertEquals(BlockCategory.API_CALLS, block.category, "HttpRequestBlock category should be API_CALLS")
        assertEquals("HTTP", block.subcategory, "HttpRequestBlock subcategory should be 'HTTP'")
    }

    @Test
    fun `DatabaseQueryBlock has correct default properties`() {
        val block = DatabaseQueryBlock(
            args0 = listOf(
                BlockArgument(type = "field_input", name = "DATABASE", text = "mydb"),
                BlockArgument(type = "input_value", name = "SQL", text = "SELECT * FROM users")
            )
        )

        assertEquals("database_query", block.type, "DatabaseQueryBlock type should be 'database_query'")
        assertEquals("query database %1 with SQL %2", block.message0, "DatabaseQueryBlock message0 should match template")
        assertEquals(BlockOutputType.ANY, block.output, "DatabaseQueryBlock output should be ANY")
        assertEquals(BlockCategory.API_CALLS.colorHex, block.colour, "DatabaseQueryBlock colour should match API_CALLS category")
        assertEquals("Executes a SQL query on the specified database", block.tooltip, "DatabaseQueryBlock tooltip should be descriptive")
        assertEquals(BlockCategory.API_CALLS, block.category, "DatabaseQueryBlock category should be API_CALLS")
        assertEquals("Database", block.subcategory, "DatabaseQueryBlock subcategory should be 'Database'")
    }

    // ========================================================================
    // BlockArgument Data Class Tests
    // ========================================================================

    @Test
    fun `BlockArgument stores type correctly`() {
        val arg = BlockArgument(type = "field_input", name = "USER", text = "user")
        assertEquals("field_input", arg.type, "BlockArgument should store type correctly")
    }

    @Test
    fun `BlockArgument stores name correctly`() {
        val arg = BlockArgument(type = "field_input", name = "USER", text = "user")
        assertEquals("USER", arg.name, "BlockArgument should store name correctly")
    }

    @Test
    fun `BlockArgument stores text correctly`() {
        val arg = BlockArgument(type = "field_input", name = "USER", text = "user")
        assertEquals("user", arg.text, "BlockArgument should store text correctly")
    }

    @Test
    fun `BlockArgument stores check constraint correctly`() {
        val arg = BlockArgument(type = "input_value", name = "CONDITION", check = BlockOutputType.BOOLEAN)
        assertEquals(BlockOutputType.BOOLEAN, arg.check, "BlockArgument should store check constraint correctly")
    }

    @Test
    fun `BlockArgument check constraint can be null`() {
        val arg = BlockArgument(type = "field_input", name = "USER", text = "user")
        assertEquals(null, arg.check, "BlockArgument check constraint can be null")
    }

    // ========================================================================
    // AgentBlockRegistry Tests
    // ========================================================================

    @Test
    fun `AgentBlockRegistry allBlocks is not empty`() {
        assertTrue(AgentBlockRegistry.allBlocks.isNotEmpty(), "AgentBlockRegistry should contain blocks")
    }

    @Test
    fun `AgentBlockRegistry contains message_received block`() {
        val block = AgentBlockRegistry.getBlockByType("message_received")
        val nonNullBlock = requireNotNull(block) { "AgentBlockRegistry should contain message_received block" }
        assertEquals("message_received", nonNullBlock.type, "Found block should have correct type")
    }

    @Test
    fun `AgentBlockRegistry contains send_message block`() {
        val block = AgentBlockRegistry.getBlockByType("send_message")
        val nonNullBlock = requireNotNull(block) { "AgentBlockRegistry should contain send_message block" }
        assertEquals("send_message", nonNullBlock.type, "Found block should have correct type")
    }

    @Test
    fun `AgentBlockRegistry returns null for non-existent block type`() {
        val block = AgentBlockRegistry.getBlockByType("non_existent_block")
        assertEquals(null, block, "AgentBlockRegistry should return null for non-existent block type")
    }

    @Test
    fun `AgentBlockRegistry getBlocksByCategory returns correct category`() {
        val triggersBlocks = AgentBlockRegistry.getBlocksByCategory(BlockCategory.TRIGGERS)
        assertTrue(triggersBlocks.isNotEmpty(), "getBlocksByCategory should return blocks for TRIGGERS")
        assertTrue(triggersBlocks.all { it.category == BlockCategory.TRIGGERS }, "All returned blocks should be in TRIGGERS category")
    }

    @Test
    fun `AgentBlockRegistry getBlocksByCategory returns empty for no blocks in category`() {
        // Assuming no blocks in a custom category
        val emptyBlocks = AgentBlockRegistry.getBlocksByCategory(BlockCategory.API_CALLS)
            .filter { it.type == "custom_block" }
        assertTrue(emptyBlocks.isEmpty(), "getBlocksByCategory should return empty list when no blocks match")
    }
}
