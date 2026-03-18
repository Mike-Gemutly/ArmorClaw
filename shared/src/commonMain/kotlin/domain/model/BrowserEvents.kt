package com.armorclaw.shared.domain.model

import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName

// ============================================================================
// BROWSER COMMAND EVENTS (ArmorChat → Bridge)
// ============================================================================

/**
 * Event type constants for browser commands
 */
object BrowserEventTypes {
    const val NAVIGATE = "com.armorclaw.browser.navigate"
    const val FILL = "com.armorclaw.browser.fill"
    const val CLICK = "com.armorclaw.browser.click"
    const val WAIT = "com.armorclaw.browser.wait"
    const val EXTRACT = "com.armorclaw.browser.extract"
    const val SCREENSHOT = "com.armorclaw.browser.screenshot"
    const val RESPONSE = "com.armorclaw.browser.response"
    const val STATUS = "com.armorclaw.browser.status"
    const val AGENT_STATUS = "com.armorclaw.agent.status"
    const val PII_RESPONSE = "com.armorclaw.pii.response"
}

/**
 * Wait until condition for navigation
 */
@Serializable
enum class WaitUntil {
    @SerialName("load") LOAD,
    @SerialName("domcontentloaded") DOM_CONTENT_LOADED,
    @SerialName("networkidle") NETWORK_IDLE
}

/**
 * Navigate command - Load a URL
 */
@Serializable
data class NavigateCommand(
    val url: String,
    @SerialName("waitUntil")
    val waitUntil: WaitUntil = WaitUntil.NETWORK_IDLE,
    val timeout: Int = 30000
)

/**
 * Fill field for form filling
 */
@Serializable
data class FillField(
    val selector: String,
    val value: String? = null,        // Static value
    @SerialName("value_ref")
    val valueRef: String? = null,     // PII reference: "payment.card_number"
    val type: String = "text",
    @SerialName("clear_first")
    val clearFirst: Boolean = true,
    val humanize: Boolean = true
)

/**
 * Fill command - Fill form fields
 */
@Serializable
data class FillCommand(
    val fields: List<FillField>,
    @SerialName("auto_submit")
    val autoSubmit: Boolean = false,
    @SerialName("submit_selector")
    val submitSelector: String? = null,
    @SerialName("submit_delay")
    val submitDelay: Int = 500
)

/**
 * Click command - Click an element
 */
@Serializable
data class ClickCommand(
    val selector: String,
    @SerialName("waitFor")
    val waitFor: String = "none",     // "none" | "navigation" | "selector"
    @SerialName("waitSelector")
    val waitSelector: String? = null,
    val timeout: Int = 10000,
    val humanize: Boolean = true
)

/**
 * Wait command - Wait for condition
 */
@Serializable
data class WaitCommand(
    val condition: String,            // "selector" | "timeout" | "url" | "function"
    val value: String,
    val timeout: Int = 10000
)

/**
 * Extract field for data extraction
 */
@Serializable
data class ExtractField(
    val name: String,
    val selector: String,
    val attribute: String = "textContent"
)

/**
 * Extract command - Retrieve page data
 */
@Serializable
data class ExtractCommand(
    val fields: List<ExtractField>
)

/**
 * Screenshot command - Capture page
 */
@Serializable
data class ScreenshotCommand(
    @SerialName("fullPage")
    val fullPage: Boolean = false,
    val selector: String? = null,
    val format: String = "png"
)

// ============================================================================
// BROWSER RESPONSE EVENTS (Bridge → ArmorChat)
// ============================================================================

/**
 * Browser error codes
 */
@Serializable
enum class BrowserErrorCode {
    @SerialName("ELEMENT_NOT_FOUND") ELEMENT_NOT_FOUND,
    @SerialName("NAVIGATION_FAILED") NAVIGATION_FAILED,
    @SerialName("TIMEOUT") TIMEOUT,
    @SerialName("PII_REQUEST_DENIED") PII_REQUEST_DENIED,
    @SerialName("INVALID_SELECTOR") INVALID_SELECTOR,
    @SerialName("BROWSER_NOT_READY") BROWSER_NOT_READY,
    @SerialName("SESSION_EXPIRED") SESSION_EXPIRED,
    @SerialName("INTERVENTION_REQUIRED") INTERVENTION_REQUIRED
}

/**
 * Browser error details
 */
@Serializable
data class BrowserError(
    val code: BrowserErrorCode,
    val message: String,
    val screenshot: String? = null,
    val selector: String? = null
)

/**
 * Browser response status
 */
@Serializable
enum class BrowserResponseStatus {
    @SerialName("success") SUCCESS,
    @SerialName("error") ERROR
}

/**
 * Browser command response
 */
@Serializable
data class BrowserCommandResponse(
    val status: BrowserResponseStatus,
    val command: String,
    val data: Map<String, String>? = null,
    val error: BrowserError? = null
)

/**
 * Browser status event (real-time updates)
 */
@Serializable
data class BrowserStatusEvent(
    @SerialName("session_id")
    val sessionId: String,
    val url: String? = null,
    val title: String? = null,
    val loading: Boolean = false,
    val timestamp: Long = System.currentTimeMillis()
)

// ============================================================================
// AGENT STATUS EVENT (Bridge → ArmorChat)
// ============================================================================

/**
 * Agent status metadata
 */
@Serializable
data class AgentStatusMetadata(
    val url: String? = null,
    val step: String? = null,
    val progress: Int? = null,
    val error: String? = null,
    @SerialName("task_id")
    val taskId: String? = null,
    @SerialName("task_type")
    val taskType: String? = null,
    @SerialName("fields_requested")
    val fieldsRequested: List<String>? = null,
    val screenshot: String? = null,
    val hint: String? = null,
    @SerialName("input_selector")
    val inputSelector: String? = null,
    val result: Map<String, String>? = null
)

/**
 * Agent status event from Bridge
 *
 * Event type: com.armorclaw.agent.status
 */
@Serializable
data class BridgeAgentStatusEvent(
    @SerialName("agent_id")
    val agentId: String,
    val status: String,              // "idle" | "browsing" | "form_filling" | etc.
    val previous: String? = null,
    val timestamp: Long,
    val metadata: AgentStatusMetadata? = null
) {
    /**
     * Convert to domain AgentTaskStatus
     */
    fun toAgentTaskStatus(): AgentTaskStatus {
        return when (status) {
            "idle" -> AgentTaskStatus.IDLE
            "browsing" -> AgentTaskStatus.BROWSING
            "form_filling" -> AgentTaskStatus.FORM_FILLING
            "processing_payment" -> AgentTaskStatus.PROCESSING_PAYMENT
            "awaiting_captcha" -> AgentTaskStatus.AWAITING_CAPTCHA
            "awaiting_2fa" -> AgentTaskStatus.AWAITING_2FA
            "awaiting_approval" -> AgentTaskStatus.AWAITING_APPROVAL
            "error" -> AgentTaskStatus.ERROR
            "complete" -> AgentTaskStatus.COMPLETE
            else -> AgentTaskStatus.IDLE
        }
    }

    /**
     * Convert to domain AgentTaskStatusEvent
     */
    fun toAgentTaskStatusEvent(): AgentTaskStatusEvent {
        val metaMap = metadata?.let { meta ->
            mutableMapOf<String, String>().apply {
                meta.url?.let { put("url", it) }
                meta.step?.let { put("step", it) }
                meta.progress?.let { put("progress", it.toString()) }
                meta.error?.let { put("error", it) }
                meta.taskId?.let { put("taskId", it) }
                meta.taskType?.let { put("taskType", it) }
                meta.fieldsRequested?.let { put("fieldsRequested", it.joinToString(",")) }
                meta.screenshot?.let { put("screenshot", it) }
            }
        }
        return AgentTaskStatusEvent(
            agentId = agentId,
            status = toAgentTaskStatus(),
            timestamp = timestamp,
            metadata = metaMap
        )
    }
}

// ============================================================================
// PII FIELD REFERENCES
// ============================================================================

/**
 * PII field reference constants used by Bridge
 */
object PiiFieldRef {
    const val CARD_NUMBER = "payment.card_number"
    const val CARD_EXPIRY = "payment.card_expiry"
    const val CARD_CVV = "payment.cvv"
    const val CARD_NAME = "payment.card_name"
    const val PERSONAL_NAME = "personal.name"
    const val PERSONAL_ADDRESS = "personal.address"
    const val PERSONAL_EMAIL = "personal.email"
    const val PERSONAL_PHONE = "personal.phone"

    /**
     * All known field references
     */
    val ALL = listOf(
        CARD_NUMBER, CARD_EXPIRY, CARD_CVV, CARD_NAME,
        PERSONAL_NAME, PERSONAL_ADDRESS, PERSONAL_EMAIL, PERSONAL_PHONE
    )
}

/**
 * Map a Bridge PII field reference to a domain PiiField
 */
fun mapPiiFieldRef(
    ref: String,
    currentValue: String? = null
): PiiField {
    return when (ref) {
        PiiFieldRef.CARD_NUMBER -> PiiField(
            name = "Card Number",
            sensitivity = SensitivityLevel.HIGH,
            description = "Credit or debit card number",
            currentValue = currentValue
        )
        PiiFieldRef.CARD_EXPIRY -> PiiField(
            name = "Expiry Date",
            sensitivity = SensitivityLevel.MEDIUM,
            description = "Card expiry date (MM/YY)",
            currentValue = currentValue
        )
        PiiFieldRef.CARD_CVV -> PiiField(
            name = "CVV",
            sensitivity = SensitivityLevel.CRITICAL,
            description = "Card verification code",
            currentValue = null // Never show CVV
        )
        PiiFieldRef.CARD_NAME -> PiiField(
            name = "Cardholder Name",
            sensitivity = SensitivityLevel.LOW,
            description = "Name on card",
            currentValue = currentValue
        )
        PiiFieldRef.PERSONAL_NAME -> PiiField(
            name = "Full Name",
            sensitivity = SensitivityLevel.LOW,
            description = "Your full name",
            currentValue = currentValue
        )
        PiiFieldRef.PERSONAL_ADDRESS -> PiiField(
            name = "Address",
            sensitivity = SensitivityLevel.MEDIUM,
            description = "Street address",
            currentValue = currentValue
        )
        PiiFieldRef.PERSONAL_EMAIL -> PiiField(
            name = "Email",
            sensitivity = SensitivityLevel.LOW,
            description = "Email address",
            currentValue = currentValue
        )
        PiiFieldRef.PERSONAL_PHONE -> PiiField(
            name = "Phone",
            sensitivity = SensitivityLevel.LOW,
            description = "Phone number",
            currentValue = currentValue
        )
        else -> PiiField(
            name = ref.substringAfterLast(".").replaceFirstChar { it.uppercase() },
            sensitivity = SensitivityLevel.HIGH,
            description = "Requested field: $ref",
            currentValue = currentValue
        )
    }
}

/**
 * Convert PiiField name back to Bridge reference
 */
fun fieldNameToRef(fieldName: String): String {
    return when (fieldName) {
        "Card Number" -> PiiFieldRef.CARD_NUMBER
        "Expiry Date" -> PiiFieldRef.CARD_EXPIRY
        "CVV" -> PiiFieldRef.CARD_CVV
        "Cardholder Name" -> PiiFieldRef.CARD_NAME
        "Full Name" -> PiiFieldRef.PERSONAL_NAME
        "Address" -> PiiFieldRef.PERSONAL_ADDRESS
        "Email" -> PiiFieldRef.PERSONAL_EMAIL
        "Phone" -> PiiFieldRef.PERSONAL_PHONE
        else -> "custom.${fieldName.lowercase().replace(" ", "_")}"
    }
}

// ============================================================================
// INTERVENTION TYPES
// ============================================================================

/**
 * Intervention type detected by browser
 */
@Serializable
enum class InterventionType {
    @SerialName("captcha") CAPTCHA,
    @SerialName("twofa") TWO_FA,
    @SerialName("error") ERROR
}

/**
 * Intervention subtype
 */
@Serializable
enum class InterventionSubtype {
    @SerialName("recaptcha") RECAPTCHA,
    @SerialName("hcaptcha") HCAPTCHA,
    @SerialName("cloudflare") CLOUDFLARE,
    @SerialName("sms") SMS,
    @SerialName("email") EMAIL,
    @SerialName("totp") TOTP,
    @SerialName("form_error") FORM_ERROR
}

/**
 * Intervention detection result
 */
@Serializable
data class InterventionInfo(
    @SerialName("intervention_required")
    val interventionRequired: Boolean,
    val type: InterventionType? = null,
    val subtype: InterventionSubtype? = null,
    val selectors: List<String>? = null,
    val screenshot: String? = null,
    val hint: String? = null,
    val message: String? = null,
    @SerialName("input_selector")
    val inputSelector: String? = null
)

// ============================================================================
// BROWSER QUEUE MODELS (JSON-RPC Request/Response)
// ============================================================================

/**
 * Browser job status
 */
@Serializable
enum class BrowserJobStatus {
    @SerialName("pending") PENDING,
    @SerialName("running") RUNNING,
    @SerialName("paused") PAUSED,
    @SerialName("completed") COMPLETED,
    @SerialName("failed") FAILED,
    @SerialName("cancelled") CANCELLED
}

/**
 * Browser job priority
 */
@Serializable
enum class BrowserJobPriority {
    @SerialName("low") LOW,
    @SerialName("normal") NORMAL,
    @SerialName("high") HIGH,
    @SerialName("urgent") URGENT
}

/**
 * Browser job definition for queue
 */
@Serializable
data class BrowserJob(
    @SerialName("job_id")
    val jobId: String,
    @SerialName("agent_id")
    val agentId: String,
    @SerialName("room_id")
    val roomId: String,
    val url: String,
    val commands: List<BrowserCommand>,
    val status: BrowserJobStatus = BrowserJobStatus.PENDING,
    val priority: BrowserJobPriority = BrowserJobPriority.NORMAL,
    @SerialName("created_at")
    val createdAt: Long,
    @SerialName("updated_at")
    val updatedAt: Long,
    @SerialName("started_at")
    val startedAt: Long? = null,
    @SerialName("completed_at")
    val completedAt: Long? = null,
    val error: String? = null,
    val result: Map<String, String>? = null,
    @SerialName("requires_pii")
    val requiresPii: List<String>? = null,
    @SerialName("intervention_info")
    val interventionInfo: InterventionInfo? = null
)

/**
 * Generic browser command wrapper
 */
@Serializable
data class BrowserCommand(
    val type: String,
    val params: Map<String, String> = emptyMap()
)

/**
 * Response for browser.enqueue RPC call
 */
@Serializable
data class BrowserEnqueueResponse(
    @SerialName("job_id")
    val jobId: String,
    val status: BrowserJobStatus = BrowserJobStatus.PENDING,
    @SerialName("queue_position")
    val queuePosition: Int
)

/**
 * Response for browser.get_job RPC call
 */
@Serializable
data class BrowserJobResponse(
    val job: BrowserJob
)

/**
 * Response for browser.list RPC call
 */
@Serializable
data class BrowserJobListResponse(
    val jobs: List<BrowserJob>,
    val total: Int
)

/**
 * Response for browser.cancel RPC call
 */
@Serializable
data class BrowserCancelResponse(
    @SerialName("job_id")
    val jobId: String,
    val cancelled: Boolean,
    val status: BrowserJobStatus
)

/**
 * Response for browser.retry RPC call
 */
@Serializable
data class BrowserRetryResponse(
    @SerialName("job_id")
    val jobId: String,
    val retried: Boolean,
    @SerialName("new_job_id")
    val newJobId: String? = null
)

/**
 * Queue statistics response for browser.stats RPC call
 */
@Serializable
data class BrowserQueueStatsResponse(
    val pending: Int,
    val running: Int,
    val paused: Int,
    val completed: Int,
    val failed: Int,
    val cancelled: Int,
    val total: Int,
    @SerialName("active_workers")
    val activeWorkers: Int
)

/**
 * PII approval response for browser.pii_approve RPC call
 */
@Serializable
data class BrowserPiiApproveResponse(
    @SerialName("job_id")
    val jobId: String,
    val approved: Boolean,
    val message: String? = null
)
