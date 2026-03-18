package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable

/**
 * Activity Event
 *
 * Represents an event in the Live Activity Timeline.
 * Displays real-time actions taken by AI agents for user monitoring.
 *
 * ## Event Types
 * - NAVIGATION: Agent navigated to a URL
 * - FORM_FILL: Agent filled a form field
 * - CLICK: Agent clicked an element
 * - EXTRACT: Agent extracted data from page
 * - SCREENSHOT: Agent captured a screenshot
 * - INTERVENTION: Agent encountered an intervention point (CAPTCHA, 2FA)
 * - APPROVAL: Agent waiting for user approval
 * - ERROR: Agent encountered an error
 * - SUCCESS: Agent completed a task successfully
 *
 * ## Usage
 * ```kotlin
 * val events by viewModel.activityTimeline.collectAsState()
 *
 * LazyColumn {
 *     items(events) { event ->
 *         TimelineEventItem(event = event, onClick = { viewModel.onEventClick(event) })
 *     }
 * }
 * ```
 */
@Serializable
sealed class ActivityEvent {
    abstract val id: String
    abstract val agentId: String
    abstract val agentName: String
    abstract val roomId: String
    abstract val timestamp: Long
    abstract val eventType: ActivityEventType
    abstract val title: String
    abstract val description: String
    abstract val icon: ActivityEventIcon
    abstract val severity: ActivityEventSeverity
    abstract val requiresAttention: Boolean

    /**
     * Navigation event - agent navigated to a URL
     */
    @Serializable
    data class Navigation(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val url: String,
        val pageTitle: String? = null
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.NAVIGATION
        override val title = "Navigated to ${pageTitle ?: url}"
        override val description = url
        override val icon = ActivityEventIcon.LANGUAGE
        override val severity = ActivityEventSeverity.INFO
        override val requiresAttention = false
    }

    /**
     * Form fill event - agent filled a form field
     */
    @Serializable
    data class FormFill(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val fieldName: String,
        val fieldSelector: String,
        val isPiiField: Boolean,
        val sensitivityLevel: SensitivityLevel? = null
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.FORM_FILL
        override val title = "Filled $fieldName"
        override val description = if (isPiiField) "PII field (${sensitivityLevel?.name ?: "unknown"})" else "Form field"
        override val icon = ActivityEventIcon.EDIT
        override val severity = when (sensitivityLevel) {
            SensitivityLevel.CRITICAL -> ActivityEventSeverity.WARNING
            SensitivityLevel.HIGH -> ActivityEventSeverity.WARNING
            else -> ActivityEventSeverity.INFO
        }
        override val requiresAttention = sensitivityLevel == SensitivityLevel.CRITICAL
    }

    /**
     * Click event - agent clicked an element
     */
    @Serializable
    data class Click(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val elementDescription: String,
        val elementSelector: String
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.CLICK
        override val title = "Clicked $elementDescription"
        override val description = elementSelector
        override val icon = ActivityEventIcon.TOUCH
        override val severity = ActivityEventSeverity.INFO
        override val requiresAttention = false
    }

    /**
     * Extract event - agent extracted data from page
     */
    @Serializable
    data class Extract(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val dataDescription: String,
        val selector: String
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.EXTRACT
        override val title = "Extracted data"
        override val description = dataDescription
        override val icon = ActivityEventIcon.DOWNLOAD
        override val severity = ActivityEventSeverity.INFO
        override val requiresAttention = false
    }

    /**
     * Screenshot event - agent captured a screenshot
     */
    @Serializable
    data class Screenshot(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val screenshotPath: String,
        val pageTitle: String? = null,
        val hasSensitiveContent: Boolean = false
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.SCREENSHOT
        override val title = "Captured screenshot"
        override val description = pageTitle ?: "Page screenshot"
        override val icon = ActivityEventIcon.CAMERA
        override val severity = if (hasSensitiveContent) ActivityEventSeverity.WARNING else ActivityEventSeverity.INFO
        override val requiresAttention = hasSensitiveContent
    }

    /**
     * Intervention event - agent needs help (CAPTCHA, 2FA, etc.)
     */
    @Serializable
    data class Intervention(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val interventionType: InterventionType,
        val interventionSubtype: String? = null,
        val context: String,
        val screenshotPath: String? = null
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.INTERVENTION
        override val title = when (interventionType) {
            InterventionType.CAPTCHA -> "CAPTCHA required"
            InterventionType.TWO_FA -> "2FA code needed"
            InterventionType.ERROR -> "Error encountered"
        }
        override val description = context
        override val icon = when (interventionType) {
            InterventionType.CAPTCHA -> ActivityEventIcon.SHIELD
            InterventionType.TWO_FA -> ActivityEventIcon.KEY
            InterventionType.ERROR -> ActivityEventIcon.ERROR
        }
        override val severity = ActivityEventSeverity.CRITICAL
        override val requiresAttention = true
    }

    /**
     * Approval event - agent waiting for user approval
     */
    @Serializable
    data class Approval(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val action: String,
        val context: String
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.APPROVAL
        override val title = "Approval needed"
        override val description = "$action - $context"
        override val icon = ActivityEventIcon.PENDING
        override val severity = ActivityEventSeverity.WARNING
        override val requiresAttention = true
    }

    /**
     * Error event - agent encountered an error
     */
    @Serializable
    data class Error(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val errorMessage: String,
        val errorType: String,
        val recoverable: Boolean
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.ERROR
        override val title = if (recoverable) "Recoverable error" else "Fatal error"
        override val description = errorMessage
        override val icon = ActivityEventIcon.ERROR
        override val severity = if (recoverable) ActivityEventSeverity.WARNING else ActivityEventSeverity.ERROR
        override val requiresAttention = true
    }

    /**
     * Success event - agent completed a task successfully
     */
    @Serializable
    data class Success(
        override val id: String,
        override val agentId: String,
        override val agentName: String,
        override val roomId: String,
        override val timestamp: Long,
        val taskDescription: String,
        val result: String? = null
    ) : ActivityEvent() {
        override val eventType = ActivityEventType.SUCCESS
        override val title = "Task completed"
        override val description = result ?: taskDescription
        override val icon = ActivityEventIcon.SUCCESS
        override val severity = ActivityEventSeverity.SUCCESS
        override val requiresAttention = false
    }
}

/**
 * Activity event types
 */
@Serializable
enum class ActivityEventType {
    NAVIGATION,
    FORM_FILL,
    CLICK,
    EXTRACT,
    SCREENSHOT,
    INTERVENTION,
    APPROVAL,
    ERROR,
    SUCCESS
}

/**
 * Activity event icons (mapping to Material Icons)
 */
@Serializable
enum class ActivityEventIcon {
    LANGUAGE,    // Web/browsing
    EDIT,        // Form editing
    TOUCH,       // Clicking
    DOWNLOAD,    // Data extraction
    CAMERA,      // Screenshot
    SHIELD,      // Security/CAPTCHA
    KEY,         // 2FA/Authentication
    PENDING,     // Waiting for approval
    ERROR,       // Error state
    SUCCESS      // Success/check
}

/**
 * Activity event severity levels
 */
@Serializable
enum class ActivityEventSeverity {
    INFO,
    SUCCESS,
    WARNING,
    ERROR,
    CRITICAL
}

/**
 * Activity timeline filter options
 */
@Serializable
data class ActivityFilter(
    val showNavigation: Boolean = true,
    val showFormFills: Boolean = true,
    val showClicks: Boolean = false,
    val showScreenshots: Boolean = true,
    val showInterventions: Boolean = true,
    val showErrors: Boolean = true,
    val showSuccess: Boolean = true,
    val agentId: String? = null,
    val roomId: String? = null
) {
    fun matches(event: ActivityEvent): Boolean {
        if (agentId != null && event.agentId != agentId) return false
        if (roomId != null && event.roomId != roomId) return false

        return when (event.eventType) {
            ActivityEventType.NAVIGATION -> showNavigation
            ActivityEventType.FORM_FILL -> showFormFills
            ActivityEventType.CLICK -> showClicks
            ActivityEventType.EXTRACT -> true // Always show extracts
            ActivityEventType.SCREENSHOT -> showScreenshots
            ActivityEventType.INTERVENTION -> showInterventions
            ActivityEventType.APPROVAL -> showInterventions
            ActivityEventType.ERROR -> showErrors
            ActivityEventType.SUCCESS -> showSuccess
        }
    }
}
