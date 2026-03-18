package com.armorclaw.shared.ui.theme

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.ui.graphics.vector.ImageVector

/**
 * Cross-platform icon abstraction
 *
 * Maps extended Material icons to basic icons for Compose Multiplatform compatibility.
 * Only the most basic icons are available in commonMain - these are from the core material icons set.
 * 
 * When adding new icons, test that they work with Compose Multiplatform.
 * If an icon is not available, use a similar basic icon as fallback.
 */
object AppIcons {
    // Status icons - using core fallbacks
    val Schedule: ImageVector get() = Icons.Default.Info
    val DoneAll: ImageVector get() = Icons.Default.Done
    val Error: ImageVector get() = Icons.Default.Warning
    val Pending: ImageVector get() = Icons.Default.Info
    val Sync: ImageVector get() = Icons.Default.Refresh
    
    // Agent icons - mapping to closest core icons
    val AutoAwesome: ImageVector get() = Icons.Default.Star
    val Analytics: ImageVector get() = Icons.Default.BarChart
    val Code: ImageVector get() = Icons.Default.Code
    val Translate: ImageVector get() = Icons.Default.Translate
    val Event: ImageVector get() = Icons.Default.Event
    val AccountTree: ImageVector get() = Icons.Default.AccountCircle
    val Link: ImageVector get() = Icons.Default.Link
    val LinkOff: ImageVector get() = Icons.Default.Link
    
    // Communication icons
    val Reply: ImageVector get() = Icons.Default.Reply
    val PersonAdd: ImageVector get() = Icons.Default.PersonAdd
    val PersonRemove: ImageVector get() = Icons.Default.Person
    val Mail: ImageVector get() = Icons.Default.Email
    
    // Action icons
    val ContentCopy: ImageVector get() = Icons.Default.ContentCopy
    val QuestionAnswer: ImageVector get() = Icons.Default.QuestionAnswer
    val Download: ImageVector get() = Icons.Default.Download
    val OpenInNew: ImageVector get() = Icons.Default.OpenInNew
    
    // Verification icons
    val VerifiedUser: ImageVector get() = Icons.Default.VerifiedUser
    val Devices: ImageVector get() = Icons.Default.Devices
    
    // Description icons
    val Description: ImageVector get() = Icons.Default.Description
    val Language: ImageVector get() = Icons.Default.Language
    val Message: ImageVector get() = Icons.Default.Message
    val HourglassTop: ImageVector get() = Icons.Default.Info

    // Meeting icons
    val MeetingRoom: ImageVector get() = Icons.Default.MeetingRoom
    val EventNote: ImageVector get() = Icons.Default.EventNote

    // Security and status icons (Phase 2)
    val Payment: ImageVector get() = Icons.Default.Payment
    val Lock: ImageVector get() = Icons.Default.Lock
    val LockOpen: ImageVector get() = Icons.Default.Lock
    val Key: ImageVector get() = Icons.Default.VpnKey
    val Layers: ImageVector get() = Icons.Default.Layers
    val Fingerprint: ImageVector get() = Icons.Default.Fingerprint
    val Password: ImageVector get() = Icons.Default.Password
}
