package com.armorclaw.shared.domain.model

import kotlinx.datetime.Instant
import kotlinx.serialization.Serializable

/**
 * Unified Message Model
 *
 * Represents both regular Matrix messages and agent/terminal messages
 * in a single unified structure for the UI.
 *
 * ## Architecture
 * ```
 * UnifiedMessage
 *      ├── Regular (user message)
 *      ├── Agent (AI assistant message)
 *      ├── System (workflow events, notifications)
 *      └── Command (user command to agent)
 * ```
 *
 * ## Message Flow
 * 1. Matrix sync delivers events
 * 2. ControlPlaneStore identifies message type
 * 3. ChatViewModel creates UnifiedMessage instances
 * 4. MessageList renders based on type
 */
sealed class UnifiedMessage {
    abstract val id: String
    abstract val roomId: String
    abstract val timestamp: Instant
    abstract val sender: MessageSender

    /**
     * Regular user message (Matrix room message)
     */
    data class Regular(
        override val id: String,
        override val roomId: String,
        override val timestamp: Instant,
        override val sender: MessageSender,
        val content: MessageContent,
        val status: MessageStatus = MessageStatus.SENT,
        val replyTo: String? = null,
        val reactions: List<Reaction> = emptyList(),
        val isEncrypted: Boolean = false,
        val edits: List<EditInfo> = emptyList()
    ) : UnifiedMessage()

    /**
     * Agent/AI message (from ArmorClaw agent)
     */
    data class Agent(
        override val id: String,
        override val roomId: String,
        override val timestamp: Instant,
        override val sender: MessageSender.AgentSender,
        val content: MessageContent,
        val agentType: AgentType = AgentType.GENERAL,
        val confidence: Float? = null,
        val sources: List<SourceReference> = emptyList(),
        val actions: List<AgentAction> = emptyList(),
        val relatedTaskId: String? = null,
        val canRegenerate: Boolean = true
    ) : UnifiedMessage()

    /**
     * System message (workflow events, notifications)
     */
    data class System(
        override val id: String,
        override val roomId: String,
        override val timestamp: Instant,
        override val sender: MessageSender.SystemSender,
        val eventType: SystemEventType,
        val title: String,
        val description: String? = null,
        val data: Map<String, String> = emptyMap(),
        val actions: List<SystemAction> = emptyList()
    ) : UnifiedMessage()

    /**
     * Command message (user command to agent)
     */
    data class Command(
        override val id: String,
        override val roomId: String,
        override val timestamp: Instant,
        override val sender: MessageSender,
        val command: String,
        val args: List<String> = emptyList(),
        val status: CommandStatus = CommandStatus.PENDING,
        val result: String? = null,
        val executionTime: Long? = null
    ) : UnifiedMessage()
}

/**
 * Message sender information
 */
sealed class MessageSender {
    abstract val id: String
    abstract val displayName: String
    abstract val avatarUrl: String?

    data class UserSender(
        override val id: String,
        override val displayName: String,
        override val avatarUrl: String?,
        val isCurrentUser: Boolean = false,
        val isVerified: Boolean = false,
        /** Non-null when this user is a bridged ghost from an external platform */
        val bridgePlatform: BridgePlatform? = null
    ) : MessageSender()

    data class AgentSender(
        override val id: String,
        override val displayName: String,
        override val avatarUrl: String?,
        val agentType: AgentType,
        val capabilities: List<String> = emptyList(),
        val status: AgentStatus = AgentStatus.ONLINE
    ) : MessageSender()

    data class SystemSender(
        override val id: String = "system",
        override val displayName: String = "System",
        override val avatarUrl: String? = null
    ) : MessageSender()
}

/**
 * Platform origin for bridged "ghost" users
 *
 * When a user message arrives via the SDTW bridge (Slack, Discord, etc.),
 * the sender is a Matrix "ghost user" created by the bridge. This enum
 * identifies the originating platform so the UI can display an origin badge.
 */
enum class BridgePlatform {
    SLACK,
    DISCORD,
    TEAMS,
    WHATSAPP,
    TELEGRAM,
    SIGNAL,
    /** Native Matrix user (no bridge) */
    MATRIX_NATIVE
}

/**
 * Types of AI agents
 */
enum class AgentType {
    GENERAL,           // General purpose assistant
    ANALYSIS,          // Document/data analysis
    CODE_REVIEW,       // Code review and suggestions
    RESEARCH,          // Research and information gathering
    WRITING,           // Content writing
    TRANSLATION,       // Translation services
    SCHEDULING,        // Calendar and scheduling
    WORKFLOW,          // Workflow orchestration
    PLATFORM_BRIDGE    // External platform integration
}

/**
 * Agent online status
 */
enum class AgentStatus {
    ONLINE,
    BUSY,
    THINKING,
    OFFLINE,
    ERROR
}

/**
 * System event types
 */
enum class SystemEventType {
    // Workflow events
    WORKFLOW_STARTED,
    WORKFLOW_STEP,
    WORKFLOW_COMPLETED,
    WORKFLOW_FAILED,
    
    // Room events
    ROOM_CREATED,
    USER_JOINED,
    USER_LEFT,
    USER_INVITED,
    
    // Security events
    ENCRYPTION_ENABLED,
    VERIFICATION_REQUIRED,
    DEVICE_ADDED,
    
    // Platform events
    PLATFORM_CONNECTED,
    PLATFORM_DISCONNECTED,
    
    // Budget events
    BUDGET_WARNING,
    BUDGET_EXCEEDED,

    // Governance events (enterprise)
    /** Server license is expiring soon */
    LICENSE_WARNING,
    /** Server license has expired */
    LICENSE_EXPIRED,
    /** Message content was redacted/scrubbed by server-side policy (e.g., HIPAA, DLP) */
    CONTENT_POLICY_APPLIED,
    
    // General notifications
    INFO,
    WARNING,
    ERROR
}

/**
 * Command execution status
 */
enum class CommandStatus {
    PENDING,
    EXECUTING,
    COMPLETED,
    FAILED,
    CANCELLED
}

/**
 * Agent action (quick action button)
 */
data class AgentAction(
    val id: String,
    val label: String,
    val icon: String? = null,
    val actionType: AgentActionType,
    val data: Map<String, String> = emptyMap()
)

enum class AgentActionType {
    COPY,           // Copy to clipboard
    REGENERATE,     // Regenerate response
    FOLLOW_UP,      // Ask follow-up question
    APPLY,          // Apply suggestion
    SHARE,          // Share content
    DOWNLOAD,       // Download file
    VIEW_SOURCE,    // View source document
    EXECUTE         // Execute command
}

/**
 * System action (action button on system message)
 */
data class SystemAction(
    val id: String,
    val label: String,
    val actionType: SystemActionType,
    val data: Map<String, String> = emptyMap()
)

enum class SystemActionType {
    ACKNOWLEDGE,    // Dismiss notification
    RETRY,          // Retry failed operation
    CANCEL,         // Cancel workflow
    VIEW_DETAILS,   // View more details
    OPEN_SETTINGS,  // Open settings
    VERIFY,         // Start verification
    INVITE          // Invite user
}

/**
 * Source reference for agent messages
 */
data class SourceReference(
    val id: String,
    val type: SourceType,
    val title: String,
    val url: String? = null,
    val snippet: String? = null
)

enum class SourceType {
    DOCUMENT,
    WEB_PAGE,
    CODE_FILE,
    MESSAGE,
    EXTERNAL_PLATFORM
}

/**
 * Message edit information
 */
data class EditInfo(
    val timestamp: Instant,
    val previousContent: String
)

/**
 * Extension functions for UnifiedMessage
 */
fun UnifiedMessage.isFromCurrentUser(currentUserId: String): Boolean {
    return when (val s = sender) {
        is MessageSender.UserSender -> s.id == currentUserId
        is MessageSender.AgentSender -> false
        is MessageSender.SystemSender -> false
    }
}

/**
 * Bridge room capabilities — determines which features are available
 * in rooms where messages are relayed through the SDTW bridge.
 *
 * `null` capabilities default to `true` (unrestricted).
 */
@Serializable
data class BridgeRoomCapabilities(
    val supportsEdit: Boolean = false,
    val supportsReactions: Boolean = false,
    val supportsThreads: Boolean = false,
    val supportsReadReceipts: Boolean = true
)

fun UnifiedMessage.canReact(capabilities: BridgeRoomCapabilities? = null): Boolean {
    if (capabilities != null && !capabilities.supportsReactions) return false
    return this is UnifiedMessage.Regular || this is UnifiedMessage.Agent
}

fun UnifiedMessage.canReply(capabilities: BridgeRoomCapabilities? = null): Boolean {
    return this is UnifiedMessage.Regular || this is UnifiedMessage.Agent || this is UnifiedMessage.Command
}

fun UnifiedMessage.canEdit(capabilities: BridgeRoomCapabilities? = null): Boolean {
    if (capabilities != null && !capabilities.supportsEdit) return false
    return this is UnifiedMessage.Regular
}

fun UnifiedMessage.canDelete(): Boolean {
    return this is UnifiedMessage.Regular || this is UnifiedMessage.Command
}

fun UnifiedMessage.canForward(): Boolean {
    return this is UnifiedMessage.Regular || this is UnifiedMessage.Agent
}

fun UnifiedMessage.canThread(capabilities: BridgeRoomCapabilities? = null): Boolean {
    if (capabilities != null && !capabilities.supportsThreads) return false
    return this is UnifiedMessage.Regular
}
