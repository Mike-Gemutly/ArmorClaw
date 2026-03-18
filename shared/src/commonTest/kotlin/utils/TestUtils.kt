package com.armorclaw.shared.utils

import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.domain.model.MessageContent
import com.armorclaw.shared.domain.model.MessageStatus
import com.armorclaw.shared.domain.model.MessageType
import kotlinx.datetime.Clock

object TestUtils {
    
    fun createTestMessage(
        id: String = "msg_test",
        roomId: String = "!room:example.com",
        senderId: String = "@user:example.com",
        content: String = "Test message"
    ): Message {
        return Message(
            id = id,
            roomId = roomId,
            senderId = senderId,
            content = MessageContent(
                type = MessageType.TEXT,
                body = content
            ),
            timestamp = Clock.System.now(),
            isOutgoing = false,
            status = MessageStatus.SENT
        )
    }
}
