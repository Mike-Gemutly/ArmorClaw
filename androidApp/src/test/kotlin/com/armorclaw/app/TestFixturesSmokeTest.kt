package com.armorclaw.app

import com.armorclaw.shared.domain.repository.RoomRepository
import com.armorclaw.shared.platform.matrix.MatrixClient
import org.junit.Test
import kotlin.test.assertNotNull

class TestFixturesSmokeTest {

    @Test
    fun `test fixtures can be imported and used`() {
        val room = RoomFactory.create(name = "Test Room")
        val rooms = RoomFactory.createList(count = 3)
        assertNotNull(room)
        assertNotNull(rooms)
        assert(rooms.size == 3)

        val message = MessageFactory.create(body = "Hello")
        val messages = MessageFactory.createList(count = 2)
        val batch = MessageFactory.createBatch()
        assertNotNull(message)
        assertNotNull(messages)
        assertNotNull(batch)

        val user = UserFactory.create()
        val currentUser = UserFactory.createCurrentUser()
        val otherUser = UserFactory.createOtherUser()
        assertNotNull(user)
        assertNotNull(currentUser)
        assertNotNull(otherUser)

        val workflow = WorkflowFactory.createStarted()
        val workflows = WorkflowFactory.createList(count = 2)
        assertNotNull(workflow)
        assertNotNull(workflows)
        assert(workflows.size == 2)

        val roomRepo = createMockRoomRepository()
        val messageRepo = createMockMessageRepository()
        val matrixClient = createMockMatrixClient()
        val controlStore = createMockControlPlaneStore()
        assertNotNull(roomRepo)
        assertNotNull(messageRepo)
        assertNotNull(matrixClient)
        assertNotNull(controlStore)

        assert(TEST_ROOM_ID.isNotEmpty())
        assert(TEST_USER_ID.isNotEmpty())
    }
}
