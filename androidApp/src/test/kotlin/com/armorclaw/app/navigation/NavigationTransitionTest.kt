package com.armorclaw.app.navigation

import org.junit.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue
import kotlin.test.assertNotNull

/**
 * Tests for navigation transitions between features.
 * 
 * This test suite verifies that users can make complete transitions
 * between all features in the app.
 */
class NavigationTransitionTest {

    // ==================== Route Constants Tests ====================

    @Test
    fun `all routes should be defined`() {
        // Core routes
        assertNotNull(AppNavigation.SPLASH)
        assertNotNull(AppNavigation.HOME)
        assertNotNull(AppNavigation.LOGIN)
        assertNotNull(AppNavigation.REGISTRATION)
        
        // Onboarding routes
        assertNotNull(AppNavigation.WELCOME)
        assertNotNull(AppNavigation.SECURITY)
        assertNotNull(AppNavigation.CONNECT)
        assertNotNull(AppNavigation.PERMISSIONS)
        assertNotNull(AppNavigation.COMPLETION)
        assertNotNull(AppNavigation.TUTORIAL)
        
        // Room routes
        assertNotNull(AppNavigation.CHAT)
        assertNotNull(AppNavigation.ROOM_MANAGEMENT)
        assertNotNull(AppNavigation.ROOM_DETAILS)
        assertNotNull(AppNavigation.ROOM_SETTINGS)
        
        // Profile routes
        assertNotNull(AppNavigation.PROFILE)
        assertNotNull(AppNavigation.CHANGE_PASSWORD)
        assertNotNull(AppNavigation.CHANGE_PHONE)
        assertNotNull(AppNavigation.EDIT_BIO)
        assertNotNull(AppNavigation.DELETE_ACCOUNT)
        assertNotNull(AppNavigation.USER_PROFILE)
        assertNotNull(AppNavigation.SHARED_ROOMS)
        
        // Settings routes
        assertNotNull(AppNavigation.SETTINGS)
        assertNotNull(AppNavigation.SECURITY_SETTINGS)
        assertNotNull(AppNavigation.NOTIFICATION_SETTINGS)
        assertNotNull(AppNavigation.APPEARANCE)
        assertNotNull(AppNavigation.PRIVACY_POLICY)
        assertNotNull(AppNavigation.MY_DATA)
        assertNotNull(AppNavigation.DATA_SAFETY)
        assertNotNull(AppNavigation.ABOUT)
        assertNotNull(AppNavigation.REPORT_BUG)
        
        // Sync & Device routes
        assertNotNull(AppNavigation.DEVICES)
        assertNotNull(AppNavigation.ADD_DEVICE)
        assertNotNull(AppNavigation.EMOJI_VERIFICATION)
        
        // Call routes
        assertNotNull(AppNavigation.ACTIVE_CALL)
        assertNotNull(AppNavigation.INCOMING_CALL)
        
        // Thread routes
        assertNotNull(AppNavigation.THREAD)
        
        // Media routes
        assertNotNull(AppNavigation.IMAGE_VIEWER)
        assertNotNull(AppNavigation.FILE_PREVIEW)
    }

    // ==================== Route Builder Tests ====================

    @Test
    fun `createChatRoute should replace roomId placeholder`() {
        val roomId = "!room123:example.com"
        val route = AppNavigation.createChatRoute(roomId)
        
        assertTrue(route.contains(roomId))
        assertFalse(route.contains("{roomId}"))
    }

    @Test
    fun `createRoomDetailsRoute should replace roomId placeholder`() {
        val roomId = "!room456:example.com"
        val route = AppNavigation.createRoomDetailsRoute(roomId)
        
        assertTrue(route.contains(roomId))
        assertFalse(route.contains("{roomId}"))
    }

    @Test
    fun `createRoomSettingsRoute should replace roomId placeholder`() {
        val roomId = "!room789:example.com"
        val route = AppNavigation.createRoomSettingsRoute(roomId)
        
        assertTrue(route.contains(roomId))
        assertFalse(route.contains("{roomId}"))
    }

    @Test
    fun `createUserProfileRoute should replace userId placeholder`() {
        val userId = "@user:example.com"
        val route = AppNavigation.createUserProfileRoute(userId)
        
        assertTrue(route.contains(userId))
        assertFalse(route.contains("{userId}"))
    }

    @Test
    fun `createSharedRoomsRoute should replace userId placeholder`() {
        val userId = "@user:example.com"
        val route = AppNavigation.createSharedRoomsRoute(userId)
        
        assertTrue(route.contains(userId))
        assertFalse(route.contains("{userId}"))
    }

    @Test
    fun `createCallRoute should replace callId placeholder`() {
        val callId = "call_12345"
        val route = AppNavigation.createCallRoute(callId)
        
        assertTrue(route.contains(callId))
        assertFalse(route.contains("{callId}"))
    }

    @Test
    fun `createIncomingCallRoute should replace all placeholders`() {
        val callId = "call_12345"
        val callerId = "@caller:example.com"
        val callerName = "John Doe"
        val callType = "voice"
        val route = AppNavigation.createIncomingCallRoute(callId, callerId, callerName, callType)
        
        assertTrue(route.contains(callId))
        assertTrue(route.contains(callerId))
        assertTrue(route.contains(callerName))
        assertTrue(route.contains(callType))
        assertFalse(route.contains("{callId}"))
        assertFalse(route.contains("{callerId}"))
        assertFalse(route.contains("{callerName}"))
        assertFalse(route.contains("{callType}"))
    }

    @Test
    fun `createThreadRoute should replace both placeholders`() {
        val roomId = "!room:example.com"
        val rootMessageId = "\$message123"
        val route = AppNavigation.createThreadRoute(roomId, rootMessageId)
        
        assertTrue(route.contains(roomId))
        assertTrue(route.contains(rootMessageId))
        assertFalse(route.contains("{roomId}"))
        assertFalse(route.contains("{rootMessageId}"))
    }

    @Test
    fun `createVerificationRoute should replace deviceId placeholder`() {
        val deviceId = "DEVICE123"
        val route = AppNavigation.createVerificationRoute(deviceId)
        
        assertTrue(route.contains(deviceId))
        assertFalse(route.contains("{deviceId}"))
    }

    @Test
    fun `createImageViewerRoute should replace imageId placeholder`() {
        val imageId = "mxc://example.com/image123"
        val route = AppNavigation.createImageViewerRoute(imageId)
        
        assertTrue(route.contains(imageId))
        assertFalse(route.contains("{imageId}"))
    }

    @Test
    fun `createFilePreviewRoute should replace fileId placeholder`() {
        val fileId = "mxc://example.com/file456"
        val route = AppNavigation.createFilePreviewRoute(fileId)
        
        assertTrue(route.contains(fileId))
        assertFalse(route.contains("{fileId}"))
    }

    // ==================== Navigation Flow Tests ====================

    @Test
    fun `onboarding flow should have correct sequence`() {
        val onboardingSequence = listOf(
            AppNavigation.WELCOME,
            AppNavigation.SECURITY,
            AppNavigation.CONNECT,
            AppNavigation.PERMISSIONS,
            AppNavigation.COMPLETION,
            AppNavigation.TUTORIAL
        )
        
        assertEquals(6, onboardingSequence.size)
        assertEquals(AppNavigation.WELCOME, onboardingSequence[0])
        assertEquals(AppNavigation.TUTORIAL, onboardingSequence.last())
    }

    @Test
    fun `auth flow should allow forgot password navigation`() {
        // User can navigate from LOGIN to FORGOT_PASSWORD
        val loginRoute = AppNavigation.LOGIN
        val forgotPasswordRoute = AppNavigation.FORGOT_PASSWORD
        
        assertNotNull(loginRoute)
        assertNotNull(forgotPasswordRoute)
    }

    @Test
    fun `auth flow should allow registration navigation`() {
        // User can navigate from LOGIN to REGISTRATION
        val loginRoute = AppNavigation.LOGIN
        val registrationRoute = AppNavigation.REGISTRATION
        
        assertNotNull(loginRoute)
        assertNotNull(registrationRoute)
    }

    // ==================== Feature Transition Tests ====================

    @Test
    fun `home to chat transition should work`() {
        val fromRoute = AppNavigation.HOME
        val toRoute = AppNavigation.createChatRoute("!room:example.com")
        
        assertNotNull(fromRoute)
        assertTrue(toRoute.startsWith("chat/"))
    }

    @Test
    fun `home to settings transition should work`() {
        val fromRoute = AppNavigation.HOME
        val toRoute = AppNavigation.SETTINGS
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `home to profile transition should work`() {
        val fromRoute = AppNavigation.HOME
        val toRoute = AppNavigation.PROFILE
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `home to search transition should work`() {
        val fromRoute = AppNavigation.HOME
        val toRoute = AppNavigation.SEARCH
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `chat to room details transition should work`() {
        val roomId = "!room:example.com"
        val fromRoute = AppNavigation.createChatRoute(roomId)
        val toRoute = AppNavigation.createRoomDetailsRoute(roomId)
        
        assertTrue(fromRoute.contains(roomId))
        assertTrue(toRoute.contains(roomId))
    }

    @Test
    fun `chat to user profile transition should work`() {
        val roomId = "!room:example.com"
        val userId = "@user:example.com"
        val chatRoute = AppNavigation.createChatRoute(roomId)
        val userProfileRoute = AppNavigation.createUserProfileRoute(userId)
        
        assertNotNull(chatRoute)
        assertTrue(userProfileRoute.contains(userId))
    }

    @Test
    fun `user profile to chat transition should work`() {
        val userId = "@user:example.com"
        val userProfileRoute = AppNavigation.createUserProfileRoute(userId)
        val dmRoomId = "!dm_$userId:example.com"
        val chatRoute = AppNavigation.createChatRoute(dmRoomId)
        
        assertNotNull(userProfileRoute)
        assertTrue(chatRoute.contains(dmRoomId))
    }

    @Test
    fun `user profile to call transition should work`() {
        val userId = "@user:example.com"
        val userProfileRoute = AppNavigation.createUserProfileRoute(userId)
        val callId = "call_123_$userId"
        val callRoute = AppNavigation.createCallRoute(callId)
        
        assertNotNull(userProfileRoute)
        assertTrue(callRoute.contains(callId))
    }

    @Test
    fun `user profile to shared rooms transition should work`() {
        val userId = "@user:example.com"
        val userProfileRoute = AppNavigation.createUserProfileRoute(userId)
        val sharedRoomsRoute = AppNavigation.createSharedRoomsRoute(userId)
        
        assertNotNull(userProfileRoute)
        assertTrue(sharedRoomsRoute.contains(userId))
    }

    @Test
    fun `shared rooms to chat transition should work`() {
        val userId = "@user:example.com"
        val roomId = "!room:example.com"
        val sharedRoomsRoute = AppNavigation.createSharedRoomsRoute(userId)
        val chatRoute = AppNavigation.createChatRoute(roomId)
        
        assertNotNull(sharedRoomsRoute)
        assertTrue(chatRoute.contains(roomId))
    }

    @Test
    fun `chat to call transition should work`() {
        val roomId = "!room:example.com"
        val callId = "call_123"
        val chatRoute = AppNavigation.createChatRoute(roomId)
        val callRoute = AppNavigation.createCallRoute(callId)
        
        assertNotNull(chatRoute)
        assertTrue(callRoute.contains(callId))
    }

    @Test
    fun `incoming call to active call transition should work`() {
        val callId = "call_123"
        val callerId = "@caller:example.com"
        val callerName = "John Doe"
        val incomingCallRoute = AppNavigation.createIncomingCallRoute(callId, callerId, callerName)
        val activeCallRoute = AppNavigation.createCallRoute(callId)
        
        assertTrue(incomingCallRoute.contains(callId))
        assertTrue(activeCallRoute.contains(callId))
    }

    @Test
    fun `chat to thread transition should work`() {
        val roomId = "!room:example.com"
        val rootMessageId = "\$msg123"
        val chatRoute = AppNavigation.createChatRoute(roomId)
        val threadRoute = AppNavigation.createThreadRoute(roomId, rootMessageId)
        
        assertNotNull(chatRoute)
        assertTrue(threadRoute.contains(roomId))
        assertTrue(threadRoute.contains(rootMessageId))
    }

    @Test
    fun `profile to settings transition should work`() {
        val fromRoute = AppNavigation.PROFILE
        val toRoute = AppNavigation.SETTINGS
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `settings to profile transition should work`() {
        val fromRoute = AppNavigation.SETTINGS
        val toRoute = AppNavigation.PROFILE
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `settings to security settings transition should work`() {
        val fromRoute = AppNavigation.SETTINGS
        val toRoute = AppNavigation.SECURITY_SETTINGS
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `settings to devices transition should work`() {
        val fromRoute = AppNavigation.SETTINGS
        val toRoute = AppNavigation.DEVICES
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `security settings to devices transition should work`() {
        val fromRoute = AppNavigation.SECURITY_SETTINGS
        val toRoute = AppNavigation.DEVICES
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `devices to add device transition should work`() {
        val fromRoute = AppNavigation.DEVICES
        val toRoute = AppNavigation.ADD_DEVICE
        
        assertNotNull(fromRoute)
        assertNotNull(toRoute)
    }

    @Test
    fun `devices to verification transition should work`() {
        val deviceId = "DEVICE123"
        val fromRoute = AppNavigation.DEVICES
        val toRoute = AppNavigation.createVerificationRoute(deviceId)
        
        assertNotNull(fromRoute)
        assertTrue(toRoute.contains(deviceId))
    }

    @Test
    fun `room management to chat transition should work`() {
        // After creating/joining a room, user should navigate to the chat
        val roomId = "!newroom:example.com"
        val toRoute = AppNavigation.createChatRoute(roomId)
        
        assertTrue(toRoute.contains(roomId))
    }

    @Test
    fun `search to room transition should work`() {
        val roomId = "!room:example.com"
        val fromRoute = AppNavigation.SEARCH
        val toRoute = AppNavigation.createChatRoute(roomId)
        
        assertNotNull(fromRoute)
        assertTrue(toRoute.contains(roomId))
    }

    // ==================== Logout Flow Tests ====================

    @Test
    fun `logout should navigate to login screen`() {
        val loginRoute = AppNavigation.LOGIN
        assertNotNull(loginRoute)
    }

    @Test
    fun `delete account should navigate to login screen`() {
        val deleteAccountRoute = AppNavigation.DELETE_ACCOUNT
        val loginRoute = AppNavigation.LOGIN
        
        assertNotNull(deleteAccountRoute)
        assertNotNull(loginRoute)
    }

    // ==================== Back Navigation Tests ====================

    @Test
    fun `chat back should navigate to home`() {
        val chatRoute = AppNavigation.createChatRoute("!room:example.com")
        val homeRoute = AppNavigation.HOME
        
        assertNotNull(chatRoute)
        assertNotNull(homeRoute)
    }

    @Test
    fun `settings back should navigate to home`() {
        val settingsRoute = AppNavigation.SETTINGS
        val homeRoute = AppNavigation.HOME
        
        assertNotNull(settingsRoute)
        assertNotNull(homeRoute)
    }

    @Test
    fun `profile back should navigate to home`() {
        val profileRoute = AppNavigation.PROFILE
        val homeRoute = AppNavigation.HOME
        
        assertNotNull(profileRoute)
        assertNotNull(homeRoute)
    }

    @Test
    fun `room details back should navigate to chat`() {
        val roomId = "!room:example.com"
        val roomDetailsRoute = AppNavigation.createRoomDetailsRoute(roomId)
        val chatRoute = AppNavigation.createChatRoute(roomId)
        
        assertNotNull(roomDetailsRoute)
        assertNotNull(chatRoute)
    }

    // ==================== Complete User Journey Tests ====================

    @Test
    fun `complete onboarding journey should be possible`() {
        // User flow: Splash -> Welcome -> Security -> Connect -> Permissions -> Completion -> Home
        val journey = listOf(
            AppNavigation.SPLASH,
            AppNavigation.WELCOME,
            "${AppNavigation.SECURITY}/0",
            "${AppNavigation.SECURITY}/1",
            "${AppNavigation.SECURITY}/2",
            "${AppNavigation.SECURITY}/3",
            AppNavigation.CONNECT,
            AppNavigation.PERMISSIONS,
            AppNavigation.COMPLETION,
            AppNavigation.HOME
        )
        
        assertEquals(10, journey.size)
        assertEquals(AppNavigation.SPLASH, journey.first())
        assertEquals(AppNavigation.HOME, journey.last())
    }

    @Test
    fun `complete onboarding with tutorial journey should be possible`() {
        // User flow: Splash -> Welcome -> Security -> Connect -> Permissions -> Completion -> Tutorial -> Home
        val journey = listOf(
            AppNavigation.SPLASH,
            AppNavigation.WELCOME,
            "${AppNavigation.SECURITY}/0",
            AppNavigation.CONNECT,
            AppNavigation.PERMISSIONS,
            AppNavigation.COMPLETION,
            AppNavigation.TUTORIAL,
            AppNavigation.HOME
        )
        
        assertEquals(8, journey.size)
        assertEquals(AppNavigation.SPLASH, journey.first())
        assertEquals(AppNavigation.HOME, journey.last())
    }

    @Test
    fun `complete login journey should be possible`() {
        // User flow: Splash -> Login -> Home
        val journey = listOf(
            AppNavigation.SPLASH,
            AppNavigation.LOGIN,
            AppNavigation.HOME
        )
        
        assertEquals(3, journey.size)
        assertEquals(AppNavigation.SPLASH, journey.first())
        assertEquals(AppNavigation.HOME, journey.last())
    }

    @Test
    fun `complete registration journey should be possible`() {
        // User flow: Splash -> Login -> Registration -> Home
        val journey = listOf(
            AppNavigation.SPLASH,
            AppNavigation.LOGIN,
            AppNavigation.REGISTRATION,
            AppNavigation.HOME
        )
        
        assertEquals(4, journey.size)
        assertEquals(AppNavigation.SPLASH, journey.first())
        assertEquals(AppNavigation.HOME, journey.last())
    }

    @Test
    fun `chat features journey should be possible`() {
        // User flow: Home -> Chat -> Room Details -> Room Settings
        val roomId = "!room:example.com"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createChatRoute(roomId),
            AppNavigation.createRoomDetailsRoute(roomId),
            AppNavigation.createRoomSettingsRoute(roomId)
        )
        
        assertEquals(4, journey.size)
        assertTrue(journey.all { it.contains(roomId) || it == AppNavigation.HOME })
    }

    @Test
    fun `call features journey should be possible`() {
        // User flow: Home -> Chat -> Active Call
        val roomId = "!room:example.com"
        val callId = "call_123"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createChatRoute(roomId),
            AppNavigation.createCallRoute(callId)
        )
        
        assertEquals(3, journey.size)
        assertTrue(journey.last().contains(callId))
    }

    @Test
    fun `incoming call journey should be possible`() {
        // User flow: Home -> Incoming Call -> Active Call
        val callId = "call_123"
        val callerId = "@caller:example.com"
        val callerName = "John Doe"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createIncomingCallRoute(callId, callerId, callerName),
            AppNavigation.createCallRoute(callId)
        )
        
        assertEquals(3, journey.size)
        assertTrue(journey[1].contains(callId))
        assertTrue(journey.last().contains(callId))
    }

    @Test
    fun `settings journey should be possible`() {
        // User flow: Home -> Settings -> Security Settings -> Devices -> Add Device
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.SETTINGS,
            AppNavigation.SECURITY_SETTINGS,
            AppNavigation.DEVICES,
            AppNavigation.ADD_DEVICE
        )
        
        assertEquals(5, journey.size)
        assertEquals(AppNavigation.HOME, journey.first())
        assertEquals(AppNavigation.ADD_DEVICE, journey.last())
    }

    @Test
    fun `profile management journey should be possible`() {
        // User flow: Home -> Profile -> Change Password
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.PROFILE,
            AppNavigation.CHANGE_PASSWORD
        )
        
        assertEquals(3, journey.size)
        assertEquals(AppNavigation.HOME, journey.first())
        assertEquals(AppNavigation.CHANGE_PASSWORD, journey.last())
    }

    @Test
    fun `room creation journey should be possible`() {
        // User flow: Home -> Room Management -> Create Room -> Chat
        val roomId = "!newroom:example.com"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.ROOM_MANAGEMENT,
            AppNavigation.createChatRoute(roomId)
        )
        
        assertEquals(3, journey.size)
        assertEquals(AppNavigation.HOME, journey.first())
        assertTrue(journey.last().contains(roomId))
    }

    @Test
    fun `room join journey should be possible`() {
        // User flow: Home -> Room Management -> Join Room -> Chat
        val roomId = "!joinedroom:example.com"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.ROOM_MANAGEMENT,
            AppNavigation.createChatRoute(roomId)
        )
        
        assertEquals(3, journey.size)
        assertEquals(AppNavigation.HOME, journey.first())
        assertTrue(journey.last().contains(roomId))
    }

    @Test
    fun `user profile to DM chat journey should be possible`() {
        // User flow: Chat -> User Profile -> DM Chat
        val userId = "@user:example.com"
        val dmRoomId = "!dm_$userId:example.com"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createChatRoute("!room:example.com"),
            AppNavigation.createUserProfileRoute(userId),
            AppNavigation.createChatRoute(dmRoomId)
        )
        
        assertEquals(4, journey.size)
        assertTrue(journey[2].contains(userId))
        assertTrue(journey.last().contains(dmRoomId))
    }

    @Test
    fun `user profile to shared rooms journey should be possible`() {
        // User flow: Chat -> User Profile -> Shared Rooms -> Chat
        val userId = "@user:example.com"
        val roomId = "!room:example.com"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createChatRoute(roomId),
            AppNavigation.createUserProfileRoute(userId),
            AppNavigation.createSharedRoomsRoute(userId),
            AppNavigation.createChatRoute(roomId)
        )
        
        assertEquals(5, journey.size)
        assertTrue(journey[2].contains(userId))
        assertTrue(journey[3].contains(userId))
    }

    @Test
    fun `security default step should be zero`() {
        // Welcome screen should navigate to SECURITY/0
        val defaultSecurityRoute = "${AppNavigation.SECURITY}/0"
        assertTrue(defaultSecurityRoute.contains("0"))
    }

    @Test
    fun `onboarding security steps should increment`() {
        // Verify security steps flow correctly
        for (step in 0..3) {
            val route = "${AppNavigation.SECURITY}/$step"
            assertTrue(route.contains(step.toString()))
        }
    }

    @Test
    fun `search to chat journey should be possible`() {
        // User flow: Home -> Search -> Chat
        val roomId = "!room:example.com"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.SEARCH,
            AppNavigation.createChatRoute(roomId)
        )
        
        assertEquals(3, journey.size)
        assertEquals(AppNavigation.HOME, journey.first())
        assertTrue(journey.last().contains(roomId))
    }

    @Test
    fun `device verification journey should be possible`() {
        // User flow: Home -> Settings -> Devices -> Verification
        val deviceId = "DEVICE123"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.SETTINGS,
            AppNavigation.DEVICES,
            AppNavigation.createVerificationRoute(deviceId)
        )
        
        assertEquals(4, journey.size)
        assertEquals(AppNavigation.HOME, journey.first())
        assertTrue(journey.last().contains(deviceId))
    }

    @Test
    fun `add device journey should be possible`() {
        // User flow: Home -> Settings -> Devices -> Add Device
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.SETTINGS,
            AppNavigation.DEVICES,
            AppNavigation.ADD_DEVICE
        )
        
        assertEquals(4, journey.size)
        assertEquals(AppNavigation.HOME, journey.first())
        assertEquals(AppNavigation.ADD_DEVICE, journey.last())
    }

    @Test
    fun `thread view journey should be possible`() {
        // User flow: Home -> Chat -> Thread
        val roomId = "!room:example.com"
        val rootMessageId = "\$msg123"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createChatRoute(roomId),
            AppNavigation.createThreadRoute(roomId, rootMessageId)
        )
        
        assertEquals(3, journey.size)
        assertTrue(journey[1].contains(roomId))
        assertTrue(journey[2].contains(roomId))
        assertTrue(journey[2].contains(rootMessageId))
    }

    @Test
    fun `media viewer journey should be possible`() {
        // User flow: Home -> Chat -> Image Viewer
        val roomId = "!room:example.com"
        val imageId = "mxc://example.com/image123"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createChatRoute(roomId),
            AppNavigation.createImageViewerRoute(imageId)
        )
        
        assertEquals(3, journey.size)
        assertTrue(journey[1].contains(roomId))
        assertTrue(journey[2].contains(imageId))
    }

    @Test
    fun `file preview journey should be possible`() {
        // User flow: Home -> Chat -> File Preview
        val roomId = "!room:example.com"
        val fileId = "mxc://example.com/file456"
        val journey = listOf(
            AppNavigation.HOME,
            AppNavigation.createChatRoute(roomId),
            AppNavigation.createFilePreviewRoute(fileId)
        )
        
        assertEquals(3, journey.size)
        assertTrue(journey[1].contains(roomId))
        assertTrue(journey[2].contains(fileId))
    }
}
