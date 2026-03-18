package com.armorclaw.app.screens.onboarding

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Phone
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Security
import androidx.compose.material.icons.filled.Shield
import androidx.compose.ui.graphics.vector.ImageVector
import org.junit.Test
import kotlin.test.assertEquals

class SecurityExplanationScreenTest {
    
    @Test
    fun `should have 4 security steps`() {
        val steps = SecurityExplanationData.steps
        assertEquals(4, steps.size)
    }
    
    @Test
    fun `should have correct step order`() {
        val steps = SecurityExplanationData.steps
        assertEquals("phone", steps[0].id)
        assertEquals("matrix", steps[1].id)
        assertEquals("bridge", steps[2].id)
        assertEquals("agent", steps[3].id)
    }
}

object SecurityExplanationData {
    val steps = listOf(
        SecurityStep(
            id = "phone",
            title = "Your Phone",
            description = "Where you chat with your AI agent",
            icon = Icons.Default.Phone
        ),
        SecurityStep(
            id = "matrix",
            title = "Matrix Protocol",
            description = "End-to-end encrypted messaging",
            icon = Icons.Default.Lock
        ),
        SecurityStep(
            id = "bridge",
            title = "ArmorClaw Bridge",
            description = "Policy enforcement and filtering",
            icon = Icons.Default.Security
        ),
        SecurityStep(
            id = "agent",
            title = "AI Agent Container",
            description = "Isolated environment for your agent",
            icon = Icons.Default.Shield
        )
    )
}

data class SecurityStep(
    val id: String,
    val title: String,
    val description: String,
    val icon: ImageVector
)
