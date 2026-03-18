package com.armorclaw.app

import com.armorclaw.shared.data.store.ControlPlaneStore
import com.armorclaw.shared.data.store.WorkflowState
import com.armorclaw.shared.domain.model.*
import com.armorclaw.shared.domain.repository.MessageRepository
import com.armorclaw.shared.domain.repository.RoomRepository
import com.armorclaw.shared.platform.matrix.MatrixClient
import com.armorclaw.shared.platform.matrix.MessageBatch
import com.armorclaw.shared.platform.matrix.event.StepStatus
import io.mockk.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.flowOf
import kotlinx.datetime.Clock

/**
 * Shared test fixtures and factories for ArmorClaw tests.
 *
 * Provides reusable test data and mock setup helpers to reduce
 * boilerplate and ensure consistency across test suites.
 */

// ============================================================================
// ROOM FIXTURES
// ============================================================================

/**
 * Factory for creating test Room objects.
 */
object RoomFactory {

    /** Create a basic test room with minimal required fields. */
    fun create(
        id: String = "room_${Clock.System.now().toEpochMilliseconds()}",
        name: String = "Test Room",
        type: RoomType = RoomType.GROUP,
        membership: Membership = Membership.JOIN,
        unreadCount: Int = 0
    ) = Room(
        id = id,
        name = name,
        type = type,
        membership = membership,
        createdAt = Clock.System.now(),
        unreadCount = unreadCount
    )

    /** Create a list of test rooms. */
    fun createList(count: Int = 2): List<Room> = (1..count).map { index ->
        create(
            id = "room$index",
            name = "Room $index",
            unreadCount = index % 2  // Alternate between 0 and 1
        )
    }
}

// ============================================================================
// MESSAGE FIXTURES
// ============================================================================

/**
 * Factory for creating test Message objects.
 */
object MessageFactory {

    /** Create a basic test message. */
    fun create(
        id: String = "\$msg_${Clock.System.now().toEpochMilliseconds()}",
        roomId: String = "!room:example.com",
        senderId: String = "@user:example.com",
        body: String = "Test message",
        type: MessageType = MessageType.TEXT,
        isOutgoing: Boolean = false,
        status: MessageStatus = MessageStatus.SENT
    ) = Message(
        id = id,
        roomId = roomId,
        senderId = senderId,
        content = MessageContent(type = type, body = body),
        timestamp = Clock.System.now(),
        isOutgoing = isOutgoing,
        status = status
    )

    /** Create a list of test messages. */
    fun createList(count: Int = 3, roomId: String = "!room:example.com"): List<Message> =
        (1..count).map { index ->
            create(
                id = "\$msg$index",
                roomId = roomId,
                body = "Message $index",
                isOutgoing = index % 2 == 0
            )
        }

    /** Create a MessageBatch for Matrix client responses. */
    fun createBatch(
        messages: List<Message> = createList(),
        nextToken: String? = "next_token"
    ) = MessageBatch(
        messages = messages,
        nextToken = nextToken
    )
}

// ============================================================================
// USER FIXTURES
// ============================================================================

/**
 * Factory for creating test User objects.
 */
object UserFactory {

    /** Create a basic test user. */
    fun create(
        id: String = "@user:example.com",
        displayName: String = "Test User",
        avatar: String? = null,
        isVerified: Boolean = true
    ) = User(
        id = id,
        displayName = displayName,
        avatar = avatar,
        isVerified = isVerified
    )

    /** Create a test user representing the current user. */
    fun createCurrentUser(
        id: String = "@user:example.com",
        displayName: String = "Current User"
    ) = create(id = id, displayName = displayName, isVerified = true)

    /** Create a test user representing another user. */
    fun createOtherUser(
        id: String = "@other:example.com",
        displayName: String = "Other User"
    ) = create(id = id, displayName = displayName, isVerified = false)
}

// ============================================================================
// WORKFLOW FIXTURES
// ============================================================================

/**
 * Factory for creating test WorkflowState objects.
 */
object WorkflowFactory {

    /** Create a basic test workflow. */
    fun createStarted(
        workflowId: String = "wf_${Clock.System.now().toEpochMilliseconds()}",
        workflowType: String = "test",
        roomId: String = "!room:example.com",
        initiatedBy: String = "@user:example.com",
        timestamp: Long = 0L
    ) = WorkflowState.Started(
        workflowId = workflowId,
        workflowType = workflowType,
        roomId = roomId,
        initiatedBy = initiatedBy,
        timestamp = timestamp
    )

    /** Create a list of test workflows. */
    fun createList(count: Int = 2): List<WorkflowState> = (1..count).map { index ->
        createStarted(
            workflowId = "wf$index",
            workflowType = "workflow$index"
        )
    }
}

// ============================================================================
// MOCK HELPERS
// ============================================================================

/**
 * Helper functions for creating and configuring common mocks.
 */

/**
 * Create a relaxed RoomRepository mock.
 */
fun createMockRoomRepository(): RoomRepository = mockk(relaxed = true)

/**
 * Create a relaxed MessageRepository mock.
 */
fun createMockMessageRepository(): MessageRepository = mockk(relaxed = true)

/**
 * Create a relaxed MatrixClient mock with basic configuration.
 *
 * @param isEncrypted Whether rooms should be encrypted (default: true)
 * @param currentUser The current user to return from mock (default: test user)
 */
fun createMockMatrixClient(
    isEncrypted: Boolean = true,
    currentUser: User = UserFactory.createCurrentUser()
): MatrixClient {
    val mock = mockk<MatrixClient>(relaxed = true)

    coEvery { mock.isRoomEncrypted(any()) } returns isEncrypted
    every { mock.currentUser } returns MutableStateFlow(currentUser)

    return mock
}

/**
 * Create a relaxed ControlPlaneStore mock with default empty state.
 *
 * @param workflows Active workflows to include (default: empty)
 * @param keystoreStatus Keystore status to return (default: Sealed)
 */
fun createMockControlPlaneStore(
    workflows: List<WorkflowState> = emptyList(),
    keystoreStatus: KeystoreStatus = KeystoreStatus.Sealed()
): ControlPlaneStore {
    val mock = mockk<ControlPlaneStore>(relaxed = true)

    every { mock.activeWorkflows } returns MutableStateFlow(workflows)
    every { mock.thinkingAgents } returns MutableStateFlow(emptyMap())
    every { mock.needsAttentionQueue } returns MutableStateFlow(emptyList())
    every { mock.agentTasks } returns MutableStateFlow(emptyList())
    every { mock.agentStatuses } returns MutableStateFlow(emptyMap())
    every { mock.pendingPiiRequests } returns MutableStateFlow(emptyList())
    every { mock.keystoreStatus } returns MutableStateFlow(keystoreStatus)
    every { mock.isPaused } returns MutableStateFlow(false)

    return mock
}

/**
 * Configure a RoomRepository mock to return the given rooms.
 */
fun RoomRepository.withRooms(rooms: List<Room>): RoomRepository {
    every { observeRooms() } returns flowOf(rooms)
    return this
}

/**
 * Configure a MatrixClient mock to return the given messages.
 */
fun MatrixClient.withMessages(
    roomId: String,
    messages: List<Message>,
    nextToken: String? = "next_token"
): MatrixClient {
    coEvery { getMessages(roomId, any(), any()) } returns Result.success(
        MessageBatch(messages = messages, nextToken = nextToken)
    )
    return this
}

/**
 * Configure a MatrixClient mock to return typing indicators.
 */
fun MatrixClient.withTyping(typingUsers: List<String> = emptyList()): MatrixClient {
    coEvery { observeTyping(any()) } returns MutableStateFlow(typingUsers)
    return this
}

// ============================================================================
// CONSTANTS
// ============================================================================

/** Default test room ID used across tests. */
const val TEST_ROOM_ID = "!room:example.com"

/** Default test user ID used across tests. */
const val TEST_USER_ID = "@user:example.com"
