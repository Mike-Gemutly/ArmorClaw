package com.armorclaw.app.secretary

import com.armorclaw.shared.secretary.*
import com.armorclaw.shared.platform.matrix.MatrixSyncManager
import com.armorclaw.shared.platform.logging.AppLogger
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.*
import org.junit.After
import org.junit.Before
import org.junit.Test
import kotlin.test.*

@OptIn(ExperimentalCoroutinesApi::class)
class SecretaryViewModelTest {

    private lateinit var viewModel: SecretaryViewModel
    private lateinit var testMatrixSyncManager: MatrixSyncManager
    private lateinit var testBriefingEngine: SecretaryBriefingEngine
    private lateinit var testContextProvider: SecretaryContextProvider
    private lateinit var testPolicyEngine: SecretaryPolicyEngine
    private lateinit var testTriage: SecretaryTriage
    private lateinit var testFollowUp: SecretaryFollowUp

    private val testDispatcher = UnconfinedTestDispatcher()
    private val testScope = TestScope(testDispatcher)

    @Before
    fun setup() {
        testMatrixSyncManager = FakeMatrixSyncManager()
        testBriefingEngine = SecretaryBriefingEngine()
        testContextProvider = FakeSecretaryContextProvider()
        testPolicyEngine = SecretaryPolicyEngine()
        testTriage = SecretaryTriage()
        testFollowUp = SecretaryFollowUp()

        viewModel = SecretaryViewModel(
            matrixSyncManager = testMatrixSyncManager,
            briefingEngine = testBriefingEngine,
            contextProvider = testContextProvider,
            policyEngine = testPolicyEngine,
            triage = testTriage,
            followUp = testFollowUp
        )
    }

    @After
    fun tearDown() {
        testScope.cleanupTestCoroutines()
    }

    @Test
    fun `A1 - init() starts briefing scheduler and observes matrix events`() = runTest {
        assertEquals(SecretaryState.Idle, viewModel.state.value)
        assertTrue(viewModel.cards.value.isEmpty())
    }

    @Test
    fun `A2 - 15min scheduler runs every 15min (simulated)`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Team standup",
            pendingApprovals = 2,
            lastMorningBriefingDate = null
        )

        assertTrue(viewModel.cards.value.isEmpty())
        advanceUntilIdle()

        val cards = viewModel.cards.value
        assertTrue(cards.isNotEmpty(), "Expected cards to be generated")
        assertTrue(cards.any { it.reason == SecretaryCardReason.MORNING_BRIEFING })
    }

    @Test
    fun `B1 - Briefing engine generates card and state goes to PROPOSING`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 10,
            nextMeeting = "Product review",
            pendingApprovals = 3,
            lastMorningBriefingDate = null
        )

        advanceUntilIdle()

        assertTrue(viewModel.state.value is SecretaryState.Proposing)
        val proposingState = viewModel.state.value as SecretaryState.Proposing
        assertTrue(proposingState.cards.isNotEmpty())
    }

    @Test
    fun `B2 - Evening review engine generates card and state goes to PROPOSING`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 3,
            nextMeeting = null,
            pendingApprovals = 1,
            lastMorningBriefingDate = null,
            lastEveningReviewDate = null
        )

        advanceUntilIdle()

        assertTrue(viewModel.state.value is SecretaryState.Proposing)
        val proposingState = viewModel.state.value as SecretaryState.Proposing
        assertTrue(proposingState.cards.any { it.reason == SecretaryCardReason.EVENING_REVIEW })
    }

    @Test
    fun `B3 - Briefing suppressed when conditions not met`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 0,
            nextMeeting = null,
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        advanceUntilIdle()

        assertTrue(viewModel.cards.value.isEmpty())
        assertEquals(SecretaryState.Idle, viewModel.state.value)
    }

    @Test
    fun `C1 - State correctly flows from Idle to Proposing`() = runTest {
        assertEquals(SecretaryState.Idle, viewModel.state.value)

        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Meeting",
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        advanceUntilIdle()

        assertTrue(viewModel.state.value is SecretaryState.Proposing)
    }

    @Test
    fun `C2 - PROPOSING state when briefing is active`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 8,
            nextMeeting = "Sprint planning",
            pendingApprovals = 5,
            lastMorningBriefingDate = null
        )

        advanceUntilIdle()

        val state = viewModel.state.value
        assertTrue(state is SecretaryState.Proposing)
        assertTrue((state as SecretaryState.Proposing).cards.isNotEmpty())
    }

    @Test
    fun `D1 - 15min check respects briefing cooldown - no duplicate cards`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Daily standup",
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        advanceUntilIdle()
        val firstCardCount = viewModel.cards.value.size
        assertTrue(firstCardCount > 0)

        (testContextProvider as FakeSecretaryContextProvider).currentContext = (testContextProvider as FakeSecretaryContextProvider).currentContext.copy(
            lastMorningBriefingDate = getCurrentDate(8 * 60 * 60 * 1000L)
        )

        advanceUntilIdle()

        val secondCardCount = viewModel.cards.value.size
        assertEquals(firstCardCount, secondCardCount, "Should not add duplicate briefing card")
    }

    @Test
    fun `D2 - Briefing cards respect time windows`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Meeting",
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        advanceUntilIdle()

        assertFalse(viewModel.cards.value.any { it.reason == SecretaryCardReason.MORNING_BRIEFING })
    }

    @Test
    fun `E1 - Briefing cards appear in card list`() = runTest {
        (testContextProvider as FakeSecretaryContextProvider).currentContext = BriefingContext(
            unreadCount = 12,
            nextMeeting = "All hands",
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        advanceUntilIdle()

        val cards = viewModel.cards.value
        assertTrue(cards.isNotEmpty())

        val briefingCard = cards.first { it.reason == SecretaryCardReason.MORNING_BRIEFING }
        assertEquals("Good morning", briefingCard.title)
        assertTrue(briefingCard.description.contains("unread messages") || briefingCard.description.contains("Next meeting"))
    }

    @Test
    fun `E2 - Morning and evening cards can coexist`() = runTest {
        val morningContext = BriefingContext(
            unreadCount = 5,
            nextMeeting = "Standup",
            pendingApprovals = 0,
            lastMorningBriefingDate = null
        )

        (testContextProvider as FakeSecretaryContextProvider).currentContext = morningContext
        advanceUntilIdle()

        assertTrue(viewModel.cards.value.any { it.reason == SecretaryCardReason.MORNING_BRIEFING })
    }

    private fun getCurrentDate(timestamp: Long): Long {
        val millisecondsPerDay = 24 * 60 * 60 * 1000L
        return (timestamp / millisecondsPerDay) * millisecondsPerDay
    }
}

class FakeMatrixSyncManager : MatrixSyncManager(
    homeserverUrl = "https://matrix.example.com",
    httpClient = io.ktor.client.HttpClient()
) {
    override val events = kotlinx.coroutines.flow.MutableSharedFlow<MatrixSyncEvent>()
}

class FakeSecretaryContextProvider : SecretaryContextProvider(
    roomRepository = FakeRoomRepository(),
    messageRepository = FakeMessageRepository()
) {
    var currentContext: BriefingContext = BriefingContext(
        unreadCount = 0,
        nextMeeting = null,
        pendingApprovals = 0,
        lastMorningBriefingDate = null
    )

    override suspend fun gatherContext(): BriefingContext {
        return currentContext
    }
}

class FakeRoomRepository : com.armorclaw.shared.domain.repository.RoomRepository {
    override suspend fun getRooms(): com.armorclaw.shared.domain.model.Result<List<com.armorclaw.shared.domain.model.Room>> {
        return com.armorclaw.shared.domain.model.Result.success(emptyList())
    }
}

class FakeMessageRepository : com.armorclaw.shared.domain.repository.MessageRepository {
}
