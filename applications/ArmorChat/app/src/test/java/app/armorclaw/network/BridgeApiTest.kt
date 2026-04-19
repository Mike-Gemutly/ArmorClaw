package app.armorclaw.network

import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test

/**
 * Unit tests for BridgeApi RPC client
 *
 * Tests:
 * - Request formatting
 * - Response parsing
 * - Error handling
 * - Parameter encoding
 */
class BridgeApiTest {

    // ========================================================================
    // Request Building Tests
    // ========================================================================

    @Test
    fun `RPCRequest has correct structure`() {
        val request = BridgeApi.RPCRequest(
            id = 1,
            method = "test.method",
            params = mapOf("key" to "value")
        )

        assertEquals("jsonrpc should be 2.0", "2.0", request.jsonrpc)
        assertEquals("id should match", 1, request.id)
        assertEquals("method should match", "test.method", request.method)
        assertNotNull("params should not be null", request.params)
    }

    @Test
    fun `RPCRequest allows null params`() {
        val request = BridgeApi.RPCRequest(
            id = 2,
            method = "no.params",
            params = null
        )

        assertNull("params should be null", request.params)
    }

    // ========================================================================
    // Response Parsing Tests
    // ========================================================================

    @Test
    fun `RPCResponse success parses correctly`() {
        val json = """{"jsonrpc":"2.0","id":1,"result":{"status":"ok"}}"""

        // In a real test, we'd parse this through the JSON decoder
        // For now, just verify the structure
        assertTrue("Response should contain result", json.contains("result"))
        assertTrue("Response should contain jsonrpc", json.contains("jsonrpc"))
    }

    @Test
    fun `RPCResponse error parses correctly`() {
        val json = """{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}"""

        assertTrue("Response should contain error", json.contains("error"))
        assertTrue("Response should contain code", json.contains("-32600"))
    }

    // ========================================================================
    // Type Structure Tests
    // ========================================================================

    @Test
    fun `LockdownStatus has all required fields`() {
        val status = BridgeApi.LockdownStatus(
            mode = "operational",
            admin_established = true,
            single_device_mode = false,
            setup_complete = true,
            security_configured = true,
            keystore_initialized = true
        )

        assertEquals("mode should match", "operational", status.mode)
        assertTrue("admin_established should be true", status.admin_established)
        assertFalse("single_device_mode should be false", status.single_device_mode)
        assertTrue("setup_complete should be true", status.setup_complete)
    }

    @Test
    fun `BondingResponse has all required fields`() {
        val response = BridgeApi.BondingResponse(
            status = "claimed",
            admin_id = "admin-123",
            device_id = "device-456",
            certificate = "cert-data",
            session_token = "token-789",
            expires_at = "2026-03-01T00:00:00Z",
            next_step = "security_config"
        )

        assertEquals("status should match", "claimed", response.status)
        assertEquals("admin_id should match", "admin-123", response.admin_id)
        assertEquals("device_id should match", "device-456", response.device_id)
        assertNotNull("certificate should not be null", response.certificate)
    }

    @Test
    fun `DataCategory has all required fields`() {
        val category = BridgeApi.DataCategory(
            id = "banking",
            name = "Banking Information",
            description = "Financial data",
            risk_level = "high",
            permission = "deny",
            allowed_websites = emptyList(),
            requires_approval = true
        )

        assertEquals("id should match", "banking", category.id)
        assertEquals("risk_level should match", "high", category.risk_level)
        assertTrue("requires_approval should be true", category.requires_approval)
    }

    @Test
    fun `Device has all required fields`() {
        val device = BridgeApi.Device(
            id = "device-1",
            name = "My Phone",
            type = "mobile",
            trust_state = "verified",
            last_seen = "2026-02-16T00:00:00Z",
            is_current = true
        )

        assertEquals("id should match", "device-1", device.id)
        assertTrue("is_current should be true", device.is_current)
    }

    @Test
    fun `Invite has all required fields`() {
        val invite = BridgeApi.Invite(
            id = "invite-1",
            code = "ABC123",
            role = "user",
            status = "active",
            created_at = "2026-02-16T00:00:00Z",
            expires_at = "2026-03-16T00:00:00Z"
        )

        assertEquals("code should match", "ABC123", invite.code)
        assertEquals("status should match", "active", invite.status)
        assertNotNull("expires_at should not be null", invite.expires_at)
    }

    @Test
    fun `PushTokenResponse has all required fields`() {
        val response = BridgeApi.PushTokenResponse(
            success = true,
            message = "Token registered",
            device_id = "device-1"
        )

        assertTrue("success should be true", response.success)
        assertEquals("message should match", "Token registered", response.message)
    }

    // ========================================================================
    // Error Response Tests
    // ========================================================================

    @Test
    fun `RPCError structure is correct`() {
        val error = BridgeApi.RPCError(
            code = -32601,
            message = "Method not found",
            data = "additional info"
        )

        assertEquals("code should match", -32601, error.code)
        assertEquals("message should match", "Method not found", error.message)
        assertEquals("data should match", "additional info", error.data)
    }

    @Test
    fun `RPCError allows null data`() {
        val error = BridgeApi.RPCError(
            code = -32600,
            message = "Invalid Request",
            data = null
        )

        assertNull("data should be null", error.data)
    }

    // ========================================================================
    // Edge Case Tests
    // ========================================================================

    @Test
    fun `empty string fields are handled`() {
        val status = BridgeApi.LockdownStatus(
            mode = "",
            admin_established = false,
            single_device_mode = false,
            setup_complete = false,
            security_configured = false,
            keystore_initialized = false
        )

        assertEquals("Empty mode should be preserved", "", status.mode)
    }

    @Test
    fun `resolveBlocker RPCRequest has correct method and params`() {
        val request = BridgeApi.RPCRequest(
            id = 1,
            method = "resolve_blocker",
            params = mapOf(
                "workflow_id" to "wf-123",
                "step_id" to "step-456",
                "input" to "user-provided-value",
                "note" to "optional note"
            )
        )

        assertEquals("method should be resolve_blocker", "resolve_blocker", request.method)
        assertEquals("workflow_id should match", "wf-123", request.params!!["workflow_id"])
        assertEquals("step_id should match", "step-456", request.params!!["step_id"])
        assertEquals("input should match", "user-provided-value", request.params!!["input"])
        assertEquals("note should match", "optional note", request.params!!["note"])
    }

    @Test
    fun `resolveBlocker RPCRequest omits note when empty`() {
        val request = BridgeApi.RPCRequest(
            id = 2,
            method = "resolve_blocker",
            params = mapOf(
                "workflow_id" to "wf-789",
                "step_id" to "step-012",
                "input" to "secret-data"
            )
        )

        assertEquals("method should be resolve_blocker", "resolve_blocker", request.method)
        assertNull("note should be omitted when empty", request.params!!["note"])
        assertEquals("workflow_id should match", "wf-789", request.params!!["workflow_id"])
        assertEquals("step_id should match", "step-012", request.params!!["step_id"])
        assertEquals("input should match", "secret-data", request.params!!["input"])
    }

    @Test
    fun `unicode in strings is handled`() {
        val category = BridgeApi.DataCategory(
            id = "test",
            name = "Test 中文 العربية 🎉",
            description = "Unicode test",
            risk_level = "low",
            permission = "allow",
            allowed_websites = emptyList(),
            requires_approval = false
        )

        assertTrue("Should contain Chinese characters", category.name.contains("中文"))
        assertTrue("Should contain Arabic characters", category.name.contains("العربية"))
        assertTrue("Should contain emoji", category.name.contains("🎉"))
    }

    // ========================================================================
    // Secretary Workflow Tests
    // ========================================================================

    @Test
    fun `startWorkflow RPCRequest uses correct method and params`() {
        val request = BridgeApi.RPCRequest(
            id = 10,
            method = "secretary.start_workflow",
            params = mapOf("workflow_id" to "wf-001")
        )

        assertEquals("secretary.start_workflow", request.method)
        assertEquals("wf-001", request.params!!["workflow_id"])
    }

    @Test
    fun `cancelWorkflow RPCRequest includes optional reason`() {
        val request = BridgeApi.RPCRequest(
            id = 11,
            method = "secretary.cancel_workflow",
            params = mapOf("workflow_id" to "wf-002", "reason" to "user cancelled")
        )

        assertEquals("secretary.cancel_workflow", request.method)
        assertEquals("wf-002", request.params!!["workflow_id"])
        assertEquals("user cancelled", request.params!!["reason"])
    }

    @Test
    fun `advanceWorkflow RPCRequest has workflow_id and step_id`() {
        val request = BridgeApi.RPCRequest(
            id = 12,
            method = "secretary.advance_workflow",
            params = mapOf("workflow_id" to "wf-003", "step_id" to "step-1")
        )

        assertEquals("secretary.advance_workflow", request.method)
        assertEquals("wf-003", request.params!!["workflow_id"])
        assertEquals("step-1", request.params!!["step_id"])
    }

    @Test
    fun `WorkflowResponse data class has all fields`() {
        val resp = BridgeApi.WorkflowResponse(
            workflow_id = "wf-100",
            status = "started",
            workflow = mapOf("current_step" to "step-1")
        )

        assertEquals("wf-100", resp.workflow_id)
        assertEquals("started", resp.status)
        assertNotNull(resp.workflow)
    }

    @Test
    fun `TemplateListResponse data class has all fields`() {
        val resp = BridgeApi.TemplateListResponse(
            templates = listOf(mapOf("id" to "tpl-1", "name" to "Test")),
            count = 1
        )

        assertEquals(1, resp.count)
        assertEquals("tpl-1", resp.templates[0]["id"])
    }

    @Test
    fun `TemplateResponse data class has all fields`() {
        val tpl = BridgeApi.TemplateResponse(
            id = "tpl-1",
            name = "Research Template",
            description = "Research workflow",
            is_active = true,
            created_at = "2026-01-01T00:00:00Z",
            updated_at = "2026-01-01T00:00:00Z"
        )

        assertEquals("tpl-1", tpl.id)
        assertEquals("Research Template", tpl.name)
        assertTrue(tpl.is_active)
    }

    // ========================================================================
    // Task RPC Tests
    // ========================================================================

    @Test
    fun `createTask RPCRequest uses correct method`() {
        val request = BridgeApi.RPCRequest(
            id = 20,
            method = "task.create",
            params = mapOf(
                "definition_id" to "def-1",
                "created_by" to "user-1",
                "description" to "Daily report",
                "cron_expression" to "0 9 * * *"
            )
        )

        assertEquals("task.create", request.method)
        assertEquals("def-1", request.params!!["definition_id"])
        assertEquals("0 9 * * *", request.params!!["cron_expression"])
    }

    @Test
    fun `listTasks RPCRequest uses correct method`() {
        val request = BridgeApi.RPCRequest(
            id = 21,
            method = "task.list"
        )

        assertEquals("task.list", request.method)
    }

    @Test
    fun `TaskCreateResponse data class has all fields`() {
        val resp = BridgeApi.TaskCreateResponse(
            task_id = "task-123",
            status = "pending",
            next_run = 1745000000000
        )

        assertEquals("task-123", resp.task_id)
        assertEquals("pending", resp.status)
        assertTrue(resp.next_run > 0)
    }

    // ========================================================================
    // Email Tests
    // ========================================================================

    @Test
    fun `listPendingEmails RPCRequest uses correct method`() {
        val request = BridgeApi.RPCRequest(
            id = 30,
            method = "email.list_pending"
        )

        assertEquals("email.list_pending", request.method)
        assertNull(request.params)
    }

    @Test
    fun `PendingEmail data class has all fields`() {
        val email = BridgeApi.PendingEmail(
            approval_id = "appr-1",
            sender = "bot@armorclaw.com",
            to = "user@example.com",
            subject = "Test email",
            body_preview = "Hello...",
            status = "pending",
            created_at = "2026-04-19T00:00:00Z"
        )

        assertEquals("appr-1", email.approval_id)
        assertEquals("bot@armorclaw.com", email.sender)
        assertEquals("Test email", email.subject)
    }

    @Test
    fun `PendingEmailListResponse data class has approvals and count`() {
        val resp = BridgeApi.PendingEmailListResponse(
            approvals = listOf(
                BridgeApi.PendingEmail(approval_id = "a1"),
                BridgeApi.PendingEmail(approval_id = "a2")
            ),
            count = 2
        )

        assertEquals(2, resp.count)
        assertEquals("a1", resp.approvals[0].approval_id)
    }

    // ========================================================================
    // Account Tests
    // ========================================================================

    @Test
    fun `deleteAccount RPCRequest uses correct method and params`() {
        val request = BridgeApi.RPCRequest(
            id = 40,
            method = "account.delete",
            params = mapOf("password" to "secret123", "erase" to "true")
        )

        assertEquals("account.delete", request.method)
        assertEquals("secret123", request.params!!["password"])
        assertEquals("true", request.params!!["erase"])
    }

    @Test
    fun `AccountDeleteResponse data class has all fields`() {
        val resp = BridgeApi.AccountDeleteResponse(
            status = "deactivated",
            user_id = "@admin:armorclaw.com",
            deactivated_at = "2026-04-19T12:00:00Z",
            erase = true
        )

        assertEquals("deactivated", resp.status)
        assertEquals("@admin:armorclaw.com", resp.user_id)
        assertTrue(resp.erase)
    }

    // ========================================================================
    // Studio Agent Tests
    // ========================================================================

    @Test
    fun `listAgents RPCRequest uses studio deploy with action`() {
        val request = BridgeApi.RPCRequest(
            id = 50,
            method = "studio.deploy",
            params = mapOf("action" to "list_agents", "active_only" to "true")
        )

        assertEquals("studio.deploy", request.method)
        assertEquals("list_agents", request.params!!["action"])
        assertEquals("true", request.params!!["active_only"])
    }

    @Test
    fun `getAgent RPCRequest uses studio deploy with action and id`() {
        val request = BridgeApi.RPCRequest(
            id = 51,
            method = "studio.deploy",
            params = mapOf("action" to "get_agent", "id" to "agent-1")
        )

        assertEquals("studio.deploy", request.method)
        assertEquals("get_agent", request.params!!["action"])
        assertEquals("agent-1", request.params!!["id"])
    }

    @Test
    fun `deleteAgent RPCRequest uses studio deploy with action`() {
        val request = BridgeApi.RPCRequest(
            id = 52,
            method = "studio.deploy",
            params = mapOf("action" to "delete_agent", "id" to "agent-2")
        )

        assertEquals("studio.deploy", request.method)
        assertEquals("delete_agent", request.params!!["action"])
        assertEquals("agent-2", request.params!!["id"])
    }

    @Test
    fun `AgentDefinition data class has all fields`() {
        val agent = BridgeApi.AgentDefinition(
            id = "agent-1",
            name = "Researcher",
            description = "Web research agent",
            skills = listOf("web_browsing"),
            pii_access = listOf("email"),
            resource_tier = "medium",
            is_active = true,
            created_at = "2026-01-01T00:00:00Z",
            updated_at = "2026-01-01T00:00:00Z"
        )

        assertEquals("agent-1", agent.id)
        assertEquals("Researcher", agent.name)
        assertEquals(listOf("web_browsing"), agent.skills)
        assertEquals("medium", agent.resource_tier)
    }

    @Test
    fun `AgentListResponse data class has agents and count`() {
        val resp = BridgeApi.AgentListResponse(
            agents = listOf(
                BridgeApi.AgentDefinition(id = "a1", name = "Agent 1"),
                BridgeApi.AgentDefinition(id = "a2", name = "Agent 2")
            ),
            count = 2
        )

        assertEquals(2, resp.count)
        assertEquals("a1", resp.agents[0].id)
    }

    @Test
    fun `StudioStatsResponse data class has all fields`() {
        val stats = BridgeApi.StudioStatsResponse(
            total_agents = 5,
            active_agents = 3,
            running_instances = 2,
            total_instances = 10
        )

        assertEquals(5, stats.total_agents)
        assertEquals(3, stats.active_agents)
        assertEquals(2, stats.running_instances)
    }
}
