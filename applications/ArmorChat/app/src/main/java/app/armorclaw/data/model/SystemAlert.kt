package app.armorclaw.data.model

/**
 * System Alert Event Types
 *
 * Bridge-generated alerts that need distinct UI treatment.
 * These are sent as custom Matrix events with a specific type prefix.
 */

/**
 * System alert event type for Matrix
 *
 * This custom event type is used for bridge-generated alerts
 * that should be rendered differently from regular messages.
 */
object SystemAlertEventType {
    // Custom event type prefix for ArmorClaw system alerts
    const val PREFIX = "app.armorclaw.alert"

    // Full event type for system alerts
    const val SYSTEM_ALERT = "app.armorclaw.alert"

    // Alert type field in content
    const val FIELD_ALERT_TYPE = "alert_type"
    const val FIELD_SEVERITY = "severity"
    const val FIELD_TITLE = "title"
    const val FIELD_MESSAGE = "message"
    const val FIELD_ACTION = "action"
    const val FIELD_ACTION_URL = "action_url"
    const val FIELD_TIMESTAMP = "timestamp"
    const val FIELD_METADATA = "metadata"
}

/**
 * Alert severity levels
 */
enum class AlertSeverity(val displayName: String, val priority: Int) {
    INFO("Info", 0),
    WARNING("Warning", 1),
    ERROR("Error", 2),
    CRITICAL("Critical", 3);

    companion object {
        fun fromString(s: String): AlertSeverity {
            return values().find { it.name.equals(s, ignoreCase = true) } ?: INFO
        }
    }
}

/**
 * Alert types from the Bridge
 */
enum class AlertType(val displayName: String, val defaultSeverity: AlertSeverity) {
    // Budget alerts
    BUDGET_WARNING("Budget Warning", AlertSeverity.WARNING),
    BUDGET_EXCEEDED("Budget Exceeded", AlertSeverity.ERROR),

    // License alerts
    LICENSE_EXPIRING("License Expiring", AlertSeverity.WARNING),
    LICENSE_EXPIRED("License Expired", AlertSeverity.CRITICAL),
    LICENSE_INVALID("License Invalid", AlertSeverity.ERROR),

    // Security alerts
    SECURITY_EVENT("Security Event", AlertSeverity.WARNING),
    TRUST_DEGRADED("Trust Degraded", AlertSeverity.WARNING),
    VERIFICATION_REQUIRED("Verification Required", AlertSeverity.INFO),
    BRIDGE_SECURITY_DOWNGRADE("Bridge Security Downgrade", AlertSeverity.WARNING),

    // System alerts
    BRIDGE_ERROR("Bridge Error", AlertSeverity.ERROR),
    BRIDGE_RESTARTING("Bridge Restarting", AlertSeverity.WARNING),
    MAINTENANCE("Maintenance Scheduled", AlertSeverity.INFO),

    // Compliance alerts
    COMPLIANCE_VIOLATION("Compliance Violation", AlertSeverity.ERROR),
    AUDIT_EXPORT("Audit Export Ready", AlertSeverity.INFO);

    companion object {
        fun fromString(s: String): AlertType {
            return values().find { it.name.equals(s, ignoreCase = true) } ?: BRIDGE_ERROR
        }
    }
}

/**
 * System Alert content for Matrix events
 */
data class SystemAlertContent(
    val alertType: AlertType,
    val severity: AlertSeverity,
    val title: String,
    val message: String,
    val action: String? = null,
    val actionUrl: String? = null,
    val timestamp: Long = System.currentTimeMillis(),
    val metadata: Map<String, Any>? = null
) {
    /**
     * Convert to Matrix event content map
     */
    fun toContentMap(): Map<String, Any> {
        val content = mutableMapOf<String, Any>(
            SystemAlertEventType.FIELD_ALERT_TYPE to alertType.name,
            SystemAlertEventType.FIELD_SEVERITY to severity.name,
            SystemAlertEventType.FIELD_TITLE to title,
            SystemAlertEventType.FIELD_MESSAGE to message,
            SystemAlertEventType.FIELD_TIMESTAMP to timestamp
        )

        action?.let { content[SystemAlertEventType.FIELD_ACTION] = it }
        actionUrl?.let { content[SystemAlertEventType.FIELD_ACTION_URL] = it }
        metadata?.let { content[SystemAlertEventType.FIELD_METADATA] = it }

        return content
    }

    companion object {
        /**
         * Parse from Matrix event content
         */
        @Suppress("UNCHECKED_CAST")
        fun fromContentMap(content: Map<String, Any>): SystemAlertContent {
            return SystemAlertContent(
                alertType = AlertType.fromString(content[SystemAlertEventType.FIELD_ALERT_TYPE] as? String ?: ""),
                severity = AlertSeverity.fromString(content[SystemAlertEventType.FIELD_SEVERITY] as? String ?: "INFO"),
                title = content[SystemAlertEventType.FIELD_TITLE] as? String ?: "",
                message = content[SystemAlertEventType.FIELD_MESSAGE] as? String ?: "",
                action = content[SystemAlertEventType.FIELD_ACTION] as? String,
                actionUrl = content[SystemAlertEventType.FIELD_ACTION_URL] as? String,
                timestamp = (content[SystemAlertEventType.FIELD_TIMESTAMP] as? Number)?.toLong() ?: System.currentTimeMillis(),
                metadata = content[SystemAlertEventType.FIELD_METADATA] as? Map<String, Any>
            )
        }
    }
}

/**
 * Factory functions for common alerts
 */
object AlertFactory {

    fun budgetWarning(currentSpend: Double, limit: Double, percentage: Int): SystemAlertContent {
        return SystemAlertContent(
            alertType = AlertType.BUDGET_WARNING,
            severity = AlertSeverity.WARNING,
            title = "Budget Warning",
            message = "Token usage is at $percentage% (\$${String.format("%.2f", currentSpend)} of \$${String.format("%.2f", limit)} limit)",
            action = "View Usage",
            actionUrl = "armorclaw://dashboard/budget",
            metadata = mapOf(
                "current_spend" to currentSpend,
                "limit" to limit,
                "percentage" to percentage
            )
        )
    }

    fun budgetExceeded(currentSpend: Double, limit: Double): SystemAlertContent {
        return SystemAlertContent(
            alertType = AlertType.BUDGET_EXCEEDED,
            severity = AlertSeverity.ERROR,
            title = "Budget Exceeded",
            message = "Token budget has been exceeded. API calls are suspended until the budget resets.",
            action = "Upgrade Plan",
            actionUrl = "armorclaw://dashboard/billing",
            metadata = mapOf(
                "current_spend" to currentSpend,
                "limit" to limit,
                "overage" to (currentSpend - limit)
            )
        )
    }

    fun licenseExpiring(daysRemaining: Int, expiresAt: String): SystemAlertContent {
        return SystemAlertContent(
            alertType = AlertType.LICENSE_EXPIRING,
            severity = if (daysRemaining <= 7) AlertSeverity.ERROR else AlertSeverity.WARNING,
            title = "License Expiring",
            message = "Your license expires in $daysRemaining days ($expiresAt). Renew to avoid service interruption.",
            action = "Renew License",
            actionUrl = "armorclaw://dashboard/license",
            metadata = mapOf(
                "days_remaining" to daysRemaining,
                "expires_at" to expiresAt
            )
        )
    }

    fun licenseExpired(): SystemAlertContent {
        return SystemAlertContent(
            alertType = AlertType.LICENSE_EXPIRED,
            severity = AlertSeverity.CRITICAL,
            title = "License Expired",
            message = "Your license has expired. Bridge functionality is limited. Please renew to restore full access.",
            action = "Renew Now",
            actionUrl = "armorclaw://dashboard/license"
        )
    }

    fun bridgeError(component: String, errorMessage: String): SystemAlertContent {
        return SystemAlertContent(
            alertType = AlertType.BRIDGE_ERROR,
            severity = AlertSeverity.ERROR,
            title = "Bridge Error",
            message = "An error occurred in $component: $errorMessage",
            action = "View Logs",
            actionUrl = "armorclaw://dashboard/logs",
            metadata = mapOf(
                "component" to component,
                "error" to errorMessage
            )
        )
    }

    fun verificationRequired(deviceName: String): SystemAlertContent {
        return SystemAlertContent(
            alertType = AlertType.VERIFICATION_REQUIRED,
            severity = AlertSeverity.INFO,
            title = "Verification Required",
            message = "Your new device '$deviceName' needs to be verified for E2EE. Complete verification to access encrypted messages.",
            action = "Verify Device",
            actionUrl = "armorclaw://verification"
        )
    }

    fun bridgeSecurityDowngrade(roomName: String, platforms: List<String>): SystemAlertContent {
        val platformList = platforms.joinToString(", ")
        return SystemAlertContent(
            alertType = AlertType.BRIDGE_SECURITY_DOWNGRADE,
            severity = AlertSeverity.WARNING,
            title = "E2EE Bridge Warning",
            message = "Room '$roomName' is encrypted but bridged to $platformList. Messages will be decrypted before sending to these platforms.",
            action = "Learn More",
            actionUrl = "armorclaw://security/bridge-info",
            metadata = mapOf(
                "room_name" to roomName,
                "platforms" to platforms
            )
        )
    }
}
