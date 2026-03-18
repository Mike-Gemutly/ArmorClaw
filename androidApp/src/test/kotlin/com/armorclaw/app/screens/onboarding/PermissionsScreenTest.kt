package com.armorclaw.app.screens.onboarding

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.CameraAlt
import androidx.compose.ui.graphics.vector.ImageVector
import org.junit.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class PermissionsScreenTest {
    
    @Test
    fun `should have 3 permissions`() {
        val permissions = createTestPermissions()
        assertEquals(3, permissions.size)
    }
    
    @Test
    fun `should have 1 required permission`() {
        val permissions = createTestPermissions()
        val requiredCount = permissions.count { it.required }
        assertEquals(1, requiredCount)
    }
    
    @Test
    fun `should have 2 optional permissions`() {
        val permissions = createTestPermissions()
        val optionalCount = permissions.count { !it.required }
        assertEquals(2, optionalCount)
    }
    
    @Test
    fun `should grant permission`() {
        val permissions = createTestPermissions().toMutableList()
        assertFalse(permissions[0].granted)
        
        permissions[0] = permissions[0].copy(granted = true)
        assertTrue(permissions[0].granted)
    }
    
    @Test
    fun `should track progress correctly`() {
        val permissions = createTestPermissions().toMutableList()
        
        // Grant all required permissions
        permissions.forEachIndexed { index, _ ->
            if (permissions[index].required) {
                permissions[index] = permissions[index].copy(granted = true)
            }
        }
        
        val grantedCount = permissions.count { it.required && it.granted }
        val requiredCount = permissions.count { it.required }
        assertEquals(1, grantedCount)
        assertEquals(1, requiredCount)
    }
}

fun createTestPermissions(): List<Permission> {
    return listOf(
        Permission(
            id = "notifications",
            title = "Notifications",
            description = "Get notified when agent sends a message",
            icon = Icons.Default.Notifications,
            required = true
        ),
        Permission(
            id = "microphone",
            title = "Microphone",
            description = "Dictate messages instead of typing",
            icon = Icons.Default.Mic,
            required = false
        ),
        Permission(
            id = "camera",
            title = "Camera",
            description = "Send images to agents for analysis",
            icon = Icons.Default.CameraAlt,
            required = false
        )
    )
}

data class Permission(
    val id: String,
    val title: String,
    val description: String,
    val icon: ImageVector,
    val required: Boolean,
    val granted: Boolean = false
)
