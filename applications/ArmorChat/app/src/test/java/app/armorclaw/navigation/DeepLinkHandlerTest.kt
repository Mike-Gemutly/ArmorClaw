package app.armorclaw.navigation

import android.net.Uri
import org.junit.Assert.*
import org.junit.Test

class DeepLinkHandlerTest {

    // ========================================================================
    // Room deep links
    // ========================================================================

    @Test
    fun `parses armorclaw room URI`() {
        val uri = Uri.parse("armorclaw://room/!abc123:matrix.example.com")
        val result = DeepLinkHandler.handle(uri)

        assertInstanceOf(Route.Room::class.java, result)
        assertEquals("!abc123:matrix.example.com", (result as Route.Room).roomId)
    }

    @Test
    fun `parses room URI with simple id`() {
        val uri = Uri.parse("armorclaw://room/my-room-id")
        val result = DeepLinkHandler.handle(uri)

        assertInstanceOf(Route.Room::class.java, result)
        assertEquals("my-room-id", (result as Route.Room).roomId)
    }

    @Test
    fun `returns null for room URI with empty segment`() {
        val uri = Uri.parse("armorclaw://room/")
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }

    // ========================================================================
    // Email approval deep links
    // ========================================================================

    @Test
    fun `parses armorclaw email approval URI`() {
        val uri = Uri.parse("armorclaw://email/approve/appr-xyz-789")
        val result = DeepLinkHandler.handle(uri)

        assertInstanceOf(Route.EmailApproval::class.java, result)
        assertEquals("appr-xyz-789", (result as Route.EmailApproval).approvalId)
    }

    @Test
    fun `returns null for email URI missing approve segment`() {
        val uri = Uri.parse("armorclaw://email/something/appr-123")
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }

    @Test
    fun `returns null for email URI missing approval id`() {
        val uri = Uri.parse("armorclaw://email/approve/")
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }

    @Test
    fun `returns null for email URI with only host`() {
        val uri = Uri.parse("armorclaw://email")
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }

    // ========================================================================
    // Config deep link — must NOT be handled
    // ========================================================================

    @Test
    fun `returns null for armorclaw config URI`() {
        val uri = Uri.parse("armorclaw://config?d=eyJ2ZXJzaW9uIjoxfQ==")
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }

    // ========================================================================
    // Unrecognised schemes and hosts
    // ========================================================================

    @Test
    fun `returns null for https scheme`() {
        val uri = Uri.parse("https://armorclaw.example.com/room/abc")
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }

    @Test
    fun `returns null for unknown host`() {
        val uri = Uri.parse("armorclaw://unknown/path")
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }

    @Test
    fun `returns null for empty URI`() {
        val uri = Uri.EMPTY
        val result = DeepLinkHandler.handle(uri)

        assertNull(result)
    }
}
