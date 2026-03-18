package com.armorclaw.shared.domain.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * System Alert Domain Model
 *
 * Represents system alerts from the ArmorClaw Bridge.
 * These are displayed with distinct UI (colored cards, badges, action buttons).
 *
 * Event Type: app.armorclaw.alert
 *
 * @see com.armorclaw.shared.platform.bridge.BridgeEvent.SystemAlertEvent
 */
@Serializable
data class SystemAlert(
    val id: String,
    val type: AlertType,
    val severity: AlertSeverity,
    val title: String,
    val message: String,
    val action: AlertAction? = null,
    val metadata: Map<String, String> = emptyMap(),
    val roomId: String? = null,
    val eventId: String? = null,
    val timestamp: Long = System.currentTimeMillis(),
    val read: Boolean = false
) {
    /**
     * Get the deep link URL for this alert
     */
    fun getDeepLink(): String? = action?.url

    /**
     * Check if this alert requires immediate attention
     */
    fun requiresImmediateAttention(): Boolean {
        return severity == AlertSeverity.CRITICAL || severity == AlertSeverity.ERROR
    }

    /**
     * Get a unique key for deduplication
     */
    fun getDedupeKey(): String = "${type.name}:$roomId:$eventId"
}

/**
 * Alert Type Categories
 */
@Serializable
enum class AlertType {
    // Budget alerts
    @SerialName("BUDGET_WARNING")
    BUDGET_WARNING,

    @SerialName("BUDGET_EXCEEDED")
    BUDGET_EXCEEDED,

    // License alerts
    @SerialName("LICENSE_EXPIRING")
    LICENSE_EXPIRING,

    @SerialName("LICENSE_EXPIRED")
    LICENSE_EXPIRED,

    @SerialName("LICENSE_INVALID")
    LICENSE_INVALID,

    // Security alerts
    @SerialName("SECURITY_EVENT")
    SECURITY_EVENT,

    @SerialName("TRUST_DEGRADED")
    TRUST_DEGRADED,

    @SerialName("VERIFICATION_REQUIRED")
    VERIFICATION_REQUIRED,

    // System alerts
    @SerialName("BRIDGE_ERROR")
    BRIDGE_ERROR,

    @SerialName("BRIDGE_RESTARTING")
    BRIDGE_RESTARTING,

    @SerialName("MAINTENANCE")
    MAINTENANCE,

    // Compliance alerts
    @SerialName("COMPLIANCE_VIOLATION")
    COMPLIANCE_VIOLATION,

    @SerialName("AUDIT_EXPORT")
    AUDIT_EXPORT;

    /**
     * Get the category for this alert type
     */
    fun getCategory(): AlertCategory = when (this) {
        BUDGET_WARNING, BUDGET_EXCEEDED -> AlertCategory.BUDGET
        LICENSE_EXPIRING, LICENSE_EXPIRED, LICENSE_INVALID -> AlertCategory.LICENSE
        SECURITY_EVENT, TRUST_DEGRADED, VERIFICATION_REQUIRED -> AlertCategory.SECURITY
        BRIDGE_ERROR, BRIDGE_RESTARTING, MAINTENANCE -> AlertCategory.SYSTEM
        COMPLIANCE_VIOLATION, AUDIT_EXPORT -> AlertCategory.COMPLIANCE
    }

    /**
     * Get default severity for this alert type
     */
    fun getDefaultSeverity(): AlertSeverity = when (this) {
        BUDGET_WARNING -> AlertSeverity.WARNING
        BUDGET_EXCEEDED -> AlertSeverity.ERROR
        LICENSE_EXPIRING -> AlertSeverity.WARNING
        LICENSE_EXPIRED -> AlertSeverity.CRITICAL
        LICENSE_INVALID -> AlertSeverity.CRITICAL
        SECURITY_EVENT -> AlertSeverity.INFO
        TRUST_DEGRADED -> AlertSeverity.WARNING
        VERIFICATION_REQUIRED -> AlertSeverity.INFO
        BRIDGE_ERROR -> AlertSeverity.ERROR
        BRIDGE_RESTARTING -> AlertSeverity.INFO
        MAINTENANCE -> AlertSeverity.INFO
        COMPLIANCE_VIOLATION -> AlertSeverity.ERROR
        AUDIT_EXPORT -> AlertSeverity.INFO
    }

    companion object {
        fun fromString(value: String): AlertType? {
            return entries.find { it.name == value }
        }
    }
}

/**
 * Alert Severity Levels
 */
@Serializable
enum class AlertSeverity {
    @SerialName("INFO")
    INFO,

    @SerialName("WARNING")
    WARNING,

    @SerialName("ERROR")
    ERROR,

    @SerialName("CRITICAL")
    CRITICAL;

    companion object {
        fun fromString(value: String): AlertSeverity? {
            return entries.find { it.name == value }
        }
    }
}

/**
 * Alert Category
 */
@Serializable
enum class AlertCategory {
    BUDGET,
    LICENSE,
    SECURITY,
    SYSTEM,
    COMPLIANCE
}

/**
 * Alert Action
 */
@Serializable
data class AlertAction(
    val label: String,
    val url: String,
    val type: AlertActionType = AlertActionType.DEEP_LINK
)

/**
 * Alert Action Type
 */
@Serializable
enum class AlertActionType {
    @SerialName("deep_link")
    DEEP_LINK,

    @SerialName("external_url")
    EXTERNAL_URL,

    @SerialName("dismiss")
    DISMISS
}

/**
 * Deep Link Routes for Alert Actions
 */
object AlertDeepLinks {
    const val BUDGET = "armorclaw://dashboard/budget"
    const val BILLING = "armorclaw://dashboard/billing"
    const val LICENSE = "armorclaw://dashboard/license"
    const val LOGS = "armorclaw://dashboard/logs"
    const val VERIFICATION = "armorclaw://verification"
    const val DEVICES = "armorclaw://settings/devices"
    const val SECURITY = "armorclaw://settings/security"

    /**
     * Get the appropriate deep link for an alert type
     */
    fun forAlertType(type: AlertType): String? = when (type) {
        AlertType.BUDGET_WARNING, AlertType.BUDGET_EXCEEDED -> BUDGET
        AlertType.LICENSE_EXPIRING, AlertType.LICENSE_EXPIRED, AlertType.LICENSE_INVALID -> LICENSE
        AlertType.SECURITY_EVENT, AlertType.TRUST_DEGRADED -> SECURITY
        AlertType.VERIFICATION_REQUIRED -> VERIFICATION
        AlertType.BRIDGE_ERROR, AlertType.BRIDGE_RESTARTING -> LOGS
        AlertType.MAINTENANCE -> null
        AlertType.COMPLIANCE_VIOLATION, AlertType.AUDIT_EXPORT -> LOGS
    }
}

/**
 * Factory for creating SystemAlerts from bridge events
 */
object SystemAlertFactory {
    /**
     * Create a SystemAlert from a bridge event
     */
    fun fromBridgeEvent(
        eventId: String,
        alertType: String,
        severity: String,
        title: String,
        message: String,
        actionLabel: String?,
        actionUrl: String?,
        metadata: Map<String, String>?,
        roomId: String?
    ): SystemAlert? {
        val type = AlertType.fromString(alertType) ?: return null
        val sev = AlertSeverity.fromString(severity) ?: type.getDefaultSeverity()

        val alertAction = if (!actionLabel.isNullOrBlank() && !actionUrl.isNullOrBlank()) {
            AlertAction(
                label = actionLabel,
                url = actionUrl,
                type = if (actionUrl.startsWith("armorclaw://")) {
                    AlertActionType.DEEP_LINK
                } else {
                    AlertActionType.EXTERNAL_URL
                }
            )
        } else {
            AlertAction(
                label = getDefaultActionLabel(type),
                url = AlertDeepLinks.forAlertType(type) ?: "",
                type = AlertActionType.DEEP_LINK
            )
        }

        return SystemAlert(
            id = eventId,
            type = type,
            severity = sev,
            title = title,
            message = message,
            action = alertAction,
            metadata = metadata ?: emptyMap(),
            roomId = roomId,
            eventId = eventId
        )
    }

    private fun getDefaultActionLabel(type: AlertType): String = when (type) {
        AlertType.BUDGET_WARNING, AlertType.BUDGET_EXCEEDED -> "View Usage"
        AlertType.LICENSE_EXPIRING, AlertType.LICENSE_EXPIRED, AlertType.LICENSE_INVALID -> "Renew License"
        AlertType.SECURITY_EVENT, AlertType.TRUST_DEGRADED -> "View Security"
        AlertType.VERIFICATION_REQUIRED -> "Verify Device"
        AlertType.BRIDGE_ERROR, AlertType.BRIDGE_RESTARTING -> "View Logs"
        AlertType.MAINTENANCE -> "Dismiss"
        AlertType.COMPLIANCE_VIOLATION, AlertType.AUDIT_EXPORT -> "View Details"
    }
}
