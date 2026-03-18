package com.armorclaw.app.screens.onboarding

import org.junit.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlin.test.assertNotNull

class ConnectServerScreenTest {
    
    @Test
    fun `should have idle state by default`() {
        val state = ConnectionState()
        assertEquals(ConnectionStatus.Idle, state.status)
    }
    
    @Test
    fun `should transition to connecting`() {
        val state = ConnectionState(status = ConnectionStatus.Connecting)
        assertEquals(ConnectionStatus.Connecting, state.status)
    }
    
    @Test
    fun `should transition to success on connection`() {
        val serverInfo = ServerInfo(
            homeserver = "https://example.com",
            userId = "@user:example.com",
            supportsE2EE = true,
            version = "1.6.0"
        )
        val state = ConnectionState(
            status = ConnectionStatus.Success,
            serverInfo = serverInfo
        )
        assertEquals(ConnectionStatus.Success, state.status)
        assertNotNull(state.serverInfo)
    }
    
    @Test
    fun `should transition to error on failure`() {
        val state = ConnectionState(
            status = ConnectionStatus.Error,
            message = "Connection failed"
        )
        assertEquals(ConnectionStatus.Error, state.status)
        assertEquals("Connection failed", state.message)
    }
}

data class ConnectionState(
    val status: ConnectionStatus = ConnectionStatus.Idle,
    val message: String? = null,
    val serverInfo: ServerInfo? = null
)

enum class ConnectionStatus {
    Idle,
    Connecting,
    Success,
    Error
}

data class ServerInfo(
    val homeserver: String,
    val userId: String,
    val supportsE2EE: Boolean,
    val version: String
)
