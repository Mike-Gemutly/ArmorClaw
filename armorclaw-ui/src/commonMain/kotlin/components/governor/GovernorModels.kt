package components.governor

import androidx.compose.runtime.Immutable

/**
 * Command Block - Action card that replaces message bubbles
 *
 * Represents a single action/command from an agent or user.
 */
@Immutable
data class CommandBlock(
    val id: String,
    val agentId: String,
    val agentName: String,
    val commandType: CommandType,
    val title: String,
    val description: String,
    val status: CommandStatus,
    val requiredCapabilities: List<Capability> = emptyList(),
    val requiredPiiKeys: List<String> = emptyList(),
    val createdAt: Long,
    val completedAt: Long? = null,
    val result: String? = null,
    val error: String? = null,
    val isApproved: Boolean = false,
    val approvedBy: String? = null,
    val approvedAt: Long? = null
)

/**
 * Command Type
 */
enum class CommandType {
    MESSAGE,           // Simple message (no action)
    ACTION,            // Action requiring approval
    QUERY,             // Information request
    SYSTEM,            // System notification
    APPROVAL_REQUIRED, // Needs user approval
    EXECUTING,         // Currently executing
    COMPLETED,         // Successfully completed
    FAILED             // Failed with error
}

/**
 * Command Status
 */
enum class CommandStatus {
    PENDING,           // Waiting for approval
    APPROVED,          // Approved, not yet started
    EXECUTING,         // Currently executing
    COMPLETED,         // Successfully completed
    FAILED,            // Failed with error
    CANCELLED          // Cancelled by user
}

/**
 * Capability
 */
@Immutable
data class Capability(
    val id: String,
    val name: String,
    val displayName: String,
    val description: String,
    val category: CapabilityCategory,
    val riskLevel: RiskLevel,
    val requiresApproval: Boolean,
    val icon: String = "default"
)

/**
 * Capability Category
 */
enum class CapabilityCategory {
    COMMUNICATION,     // Send messages, notifications
    DATA_ACCESS,       // Read user data
    DATA_MODIFY,       // Modify user data
    EXTERNAL,          // External API calls
    SYSTEM,            // System operations
    WORKFLOW           // Workflow management
}

/**
 * Risk Level
 */
enum class RiskLevel {
    LOW,               // Minimal risk
    MEDIUM,            // Some risk, may need approval
    HIGH,              // High risk, requires approval
    CRITICAL           // Critical, always requires approval
}

/**
 * Agent State for UI
 */
@Immutable
data class AgentStateUi(
    val id: String,
    val name: String,
    val status: AgentStatus,
    val capabilities: List<Capability> = emptyList(),
    val activeCommands: List<CommandBlock> = emptyList(),
    val lastActivity: Long? = null
)

/**
 * Agent Status
 */
enum class AgentStatus {
    ONLINE,
    BUSY,
    OFFLINE,
    ERROR
}
