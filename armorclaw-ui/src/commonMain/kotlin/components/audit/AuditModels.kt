package components.audit

import androidx.compose.runtime.Immutable
import components.governor.Capability
import components.governor.RiskLevel

/**
 * Task Receipt
 *
 * Immutable audit record for every agent action.
 * Provides complete transparency and accountability.
 *
 * Phase 3 Implementation - Governor Strategy
 */
@Immutable
data class TaskReceipt(
    val id: String,
    val taskId: String,
    val agentId: String,
    val agentName: String,
    val action: String,
    val actionType: ActionType,
    val status: TaskStatus,
    val timestamp: Long,
    val duration: Long? = null,
    val inputSummary: String? = null,
    val outputSummary: String? = null,
    val capabilities: List<CapabilityUsage>,
    val piiAccessed: List<PiiAccess>,
    val approvedBy: String? = null,
    val approvedAt: Long? = null,
    val errorMessage: String? = null,
    val revocable: Boolean = true,
    val revoked: Boolean = false,
    val revokedAt: Long? = null,
    val revokedBy: String? = null,
    val riskLevel: RiskLevel
)

/**
 * Action Type
 */
enum class ActionType {
    READ,           // Read data
    WRITE,          // Write/modify data
    EXECUTE,        // Execute action
    COMMUNICATE,    // Send message/notification
    QUERY,          // Query/search
    EXTERNAL        // External API call
}

/**
 * Task Status
 */
enum class TaskStatus {
    PENDING,        // Waiting for approval
    APPROVED,       // Approved, not started
    EXECUTING,      // Currently executing
    COMPLETED,      // Successfully completed
    FAILED,         // Failed with error
    CANCELLED,      // Cancelled by user
    REVOKED         // Revoked after completion
}

/**
 * Capability Usage
 *
 * Records how a specific capability was used
 */
@Immutable
data class CapabilityUsage(
    val capabilityId: String,
    val capabilityName: String,
    val usedAt: Long,
    val duration: Long? = null,
    val dataVolume: String? = null,  // e.g., "5 records", "1.2KB"
    val success: Boolean = true
)

/**
 * PII Access Record
 *
 * Records access to protected PII fields
 */
@Immutable
data class PiiAccess(
    val keyId: String,
    val fieldName: String,
    val displayName: String,
    val accessType: PiiAccessType,
    val accessedAt: Long,
    val purpose: String,
    val hashedValue: String? = null  // SHA-256 of accessed value for verification
)

/**
 * PII Access Type
 */
enum class PiiAccessType {
    READ,           // Read the value
    HASH,           // Only hashed for comparison
    MASKED,         // Partially masked access
    FULL            // Full access (requires approval)
}

/**
 * Revocation Record
 *
 * Records when a capability or action was revoked
 */
@Immutable
data class RevocationRecord(
    val id: String,
    val receiptId: String,
    val revokedAt: Long,
    val revokedBy: String,
    val reason: String,
    val affectedResources: List<String>,
    val rollbackInitiated: Boolean,
    val rollbackCompleted: Boolean? = null
)

/**
 * Audit Session
 *
 * Groups receipts for a single agent session
 */
@Immutable
data class AuditSession(
    val id: String,
    val agentId: String,
    val agentName: String,
    val startedAt: Long,
    val endedAt: Long? = null,
    val receiptCount: Int = 0,
    val totalCapabilitiesUsed: Int = 0,
    val totalPiiAccessed: Int = 0,
    val riskSummary: RiskSummary
)

/**
 * Risk Summary
 *
 * Aggregated risk metrics for a session
 */
@Immutable
data class RiskSummary(
    val lowRiskCount: Int = 0,
    val mediumRiskCount: Int = 0,
    val highRiskCount: Int = 0,
    val criticalRiskCount: Int = 0,
    val totalActions: Int = 0,
    val approvalRate: Float = 0f,  // Percentage of actions that required approval
    val revocationCount: Int = 0
) {
    val overallRiskLevel: RiskLevel
        get() = when {
            criticalRiskCount > 0 -> RiskLevel.CRITICAL
            highRiskCount > 0 -> RiskLevel.HIGH
            mediumRiskCount > 0 -> RiskLevel.MEDIUM
            else -> RiskLevel.LOW
        }
}
