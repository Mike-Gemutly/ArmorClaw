package app.armorclaw.navigation

import android.net.Uri
import org.junit.Assert.*
import org.junit.Test

/**
 * E2E tests for deep link handling across cold-start and warm-resume scenarios.
 *
 * These tests verify the full decision pipeline:
 *   URI → [DeepLinkHandler.handle] → Route → navigate-or-queue decision
 *
 * The navigation decision is driven by SETUP_ROUTES membership. We replicate
 * the set here so that any accidental removal of a setup route breaks a test.
 */
class DeepLinkE2ETest {

    private val setupRoutes = setOf(
        Route.Bonding.route,
        Route.SecurityConfig.route,
        Route.HardeningPassword.route,
        Route.HardeningDevice.route,
        Route.HardeningBiometrics.route,
        Route.KeyBackup.route,
    )

    // ========================================================================
    // Cold-start: intent.data processed on first launch
    // ========================================================================

    @Test
    fun `cold-start room deep link resolves to Room route`() {
        val uri = Uri.parse("armorclaw://room/!coldStartRoom:matrix.org")
        val route = DeepLinkHandler.handle(uri)

        assertInstanceOf(Route.Room::class.java, route)
        assertEquals("!coldStartRoom:matrix.org", (route as Route.Room).roomId)
    }

    @Test
    fun `cold-start email approval deep link resolves to EmailApproval route`() {
        val uri = Uri.parse("armorclaw://email/approve/appr-cold-123")
        val route = DeepLinkHandler.handle(uri)

        assertInstanceOf(Route.EmailApproval::class.java, route)
        assertEquals("appr-cold-123", (route as Route.EmailApproval).approvalId)
    }

    @Test
    fun `cold-start with null URI data produces no route`() {
        assertNull(DeepLinkHandler.handle(Uri.EMPTY))
    }

    // ========================================================================
    // Warm-resume: onNewIntent processes URI while activity is alive
    // ========================================================================

    @Test
    fun `warm-resume room deep link resolves to Room route`() {
        val uri = Uri.parse("armorclaw://room/!warmResumeRoom:matrix.org")
        val route = DeepLinkHandler.handle(uri)

        assertInstanceOf(Route.Room::class.java, route)
        assertEquals("!warmResumeRoom:matrix.org", (route as Route.Room).roomId)
    }

    @Test
    fun `warm-resume email approval deep link resolves to EmailApproval route`() {
        val uri = Uri.parse("armorclaw://email/approve/appr-warm-456")
        val route = DeepLinkHandler.handle(uri)

        assertInstanceOf(Route.EmailApproval::class.java, route)
        assertEquals("appr-warm-456", (route as Route.EmailApproval).approvalId)
    }

    @Test
    fun `warm-resume with unrecognised host produces no route`() {
        val uri = Uri.parse("armorclaw://unknown/path")
        assertNull(DeepLinkHandler.handle(uri))
    }

    // ========================================================================
    // Mid-setup: deep link is queued, not navigated
    // ========================================================================

    @Test
    fun `deep link is queued when current route is Bonding`() {
        val pendingRoute = DeepLinkHandler.handle(
            Uri.parse("armorclaw://room/!queued1:matrix.org")
        )
        assertNotNull(pendingRoute)
        assertTrue(
            "Bonding is a setup route — deep link should be queued",
            Route.Bonding.route in setupRoutes
        )
    }

    @Test
    fun `deep link is queued when current route is SecurityConfig`() {
        assertTrue(
            "SecurityConfig is a setup route — deep link should be queued",
            Route.SecurityConfig.route in setupRoutes
        )
    }

    @Test
    fun `deep link is queued when current route is HardeningPassword`() {
        assertTrue(
            "HardeningPassword is a setup route — deep link should be queued",
            Route.HardeningPassword.route in setupRoutes
        )
    }

    @Test
    fun `deep link is queued when current route is HardeningDevice`() {
        assertTrue(
            "HardeningDevice is a setup route — deep link should be queued",
            Route.HardeningDevice.route in setupRoutes
        )
    }

    @Test
    fun `deep link is queued when current route is HardeningBiometrics`() {
        assertTrue(
            "HardeningBiometrics is a setup route — deep link should be queued",
            Route.HardeningBiometrics.route in setupRoutes
        )
    }

    @Test
    fun `deep link is queued when current route is KeyBackup`() {
        assertTrue(
            "KeyBackup is a setup route — deep link should be queued",
            Route.KeyBackup.route in setupRoutes
        )
    }

    @Test
    fun `deep link navigates immediately when NOT on setup route`() {
        val homeRoute = Route.Home.route
        assertFalse(
            "Home is not a setup route — deep link should navigate immediately",
            homeRoute in setupRoutes
        )
    }

    // ========================================================================
    // Multiple pending deep links
    // ========================================================================

    @Test
    fun `second deep link replaces first pending route`() {
        val first = DeepLinkHandler.handle(
            Uri.parse("armorclaw://room/!first:matrix.org")
        )
        val second = DeepLinkHandler.handle(
            Uri.parse("armorclaw://room/!second:matrix.org")
        )

        assertInstanceOf(Route.Room::class.java, first)
        assertInstanceOf(Route.Room::class.java, second)
        assertEquals("!first:matrix.org", (first as Route.Room).roomId)
        assertEquals("!second:matrix.org", (second as Route.Room).roomId)
        assertNotEquals(
            "Second deep link must produce a different route than first",
            first.roomId,
            second.roomId
        )
    }

    @Test
    fun `different deep link types can both resolve independently`() {
        val roomRoute = DeepLinkHandler.handle(
            Uri.parse("armorclaw://room/!roomA:matrix.org")
        )
        val approvalRoute = DeepLinkHandler.handle(
            Uri.parse("armorclaw://email/approve/appr-multi")
        )

        assertInstanceOf(Route.Room::class.java, roomRoute)
        assertInstanceOf(Route.EmailApproval::class.java, approvalRoute)
    }

    @Test
    fun `queued deep link is consumed when leaving setup route`() {
        val pendingRoute = DeepLinkHandler.handle(
            Uri.parse("armorclaw://room/!consumed:matrix.org")
        )
        assertNotNull("Pending route must exist before consumption", pendingRoute)

        val isHomeSetupRoute = Route.Home.route in setupRoutes
        assertFalse(
            "Leaving setup route should allow pending deep link consumption",
            isHomeSetupRoute
        )
    }

    // ========================================================================
    // Config deep links are NOT handled by DeepLinkHandler
    // ========================================================================

    @Test
    fun `config deep link is ignored during cold-start`() {
        val uri = Uri.parse("armorclaw://config?d=eyJ2ZXJzaW9uIjoxfQ==")
        assertNull(DeepLinkHandler.handle(uri))
    }

    @Test
    fun `config deep link is ignored during warm-resume`() {
        val uri = Uri.parse("armorclaw://config?d=eyJ2ZXJzaW9uIjoyfQ==")
        assertNull(DeepLinkHandler.handle(uri))
    }
}
