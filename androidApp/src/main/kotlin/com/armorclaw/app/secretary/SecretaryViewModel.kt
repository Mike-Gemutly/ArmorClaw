package com.armorclaw.app.secretary

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.platform.matrix.MatrixSyncManager
import com.armorclaw.shared.platform.matrix.MatrixSyncEvent
import com.armorclaw.shared.secretary.*
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.launch
import kotlin.math.max

/**
 * Secretary ViewModel - Phase 1 Foundation + Phase 2 Briefing + Phase 3 Engines
 *
 * Core orchestration layer for the Secretary system.
 * Observes MatrixSyncManager for real-time events and manages proactive cards.
 *
 * Phase 1 scope: Matrix event observation + proactive card display only.
 * Phase 2 scope: Briefing engine + context provider integration.
 * Phase 3 scope: Policy engine, triage, follow-up integration.
 *
 * Does NOT include:
 *   - Bridge RPC execution (added in Phase 5)
 *   - Privacy guard policies (added in Phase 4)
 *   - Calendar/sensor integration
 *   - Voice surfaces
 */
class SecretaryViewModel(
    private val matrixSyncManager: MatrixSyncManager,
    private val briefingEngine: SecretaryBriefingEngine,
    private val contextProvider: SecretaryContextProvider,
    private val policyEngine: SecretaryPolicyEngine,
    private val triage: SecretaryTriage,
    private val followUp: SecretaryFollowUp
) : ViewModel() {

    private val logger = AppLogger.create(LogTag.ViewModel.Secretary)

    // State
    private val _state = MutableStateFlow<SecretaryState>(SecretaryState.Idle)
    val state = _state.asStateFlow()

    // Proactive cards list
    private val _cards = MutableStateFlow<List<ProactiveCard>>(emptyList())
    val cards = _cards.asStateFlow()

    init {
        observeMatrixEvents()
        startBriefingScheduler()
        startFollowUpScheduler()
    }

    /**
     * Observe Matrix sync events and update Secretary state.
     * Phase 1: Monitor real-time events from MatrixSyncManager.
     */
    private fun observeMatrixEvents() {
        viewModelScope.launch {
            // Collect all Matrix sync events
            matrixSyncManager.events.collect { event ->
                handleMatrixEvent(event)
            }
        }
    }

    private fun startBriefingScheduler() {
        viewModelScope.launch {
            checkBriefings()

            while (true) {
                delay(15 * 60 * 1000L)
                checkBriefings()
            }
        }
    }

    private fun startFollowUpScheduler() {
        viewModelScope.launch {
            checkFollowUps()

            while (true) {
                delay(24 * 60 * 60 * 1000L)
                checkFollowUps()
            }
        }
    }

    private suspend fun checkBriefings() {
        val currentTime = System.currentTimeMillis()
        val context = contextProvider.gatherContext()

        val morningResult = briefingEngine.generateMorningBriefing(currentTime, context)
        if (morningResult != null) {
            addBriefingCard(morningResult, SecretaryCardReason.MORNING_BRIEFING)
            contextProvider.updateMorningBriefingDate(currentTime)
        }

        val eveningResult = briefingEngine.generateEveningReview(currentTime, context)
        if (eveningResult != null) {
            addBriefingCard(eveningResult, SecretaryCardReason.EVENING_REVIEW)
            contextProvider.updateEveningReviewDate(currentTime)
        }
    }

    private suspend fun checkFollowUps() {
        val currentTime = System.currentTimeMillis()

        val followUpContext = FollowUpContext(
            threads = emptyList(),
            currentTime = currentTime,
            followUpThresholdMs = 48 * 60 * 60 * 1000L
        )

        val followUpResult = followUp.detectStaleThreads(followUpContext)

        followUpResult.followUps.forEach { followUpItem ->
            val card = ProactiveCard(
                id = "followup-${followUpItem.threadId}-${System.currentTimeMillis()}",
                title = "Follow-up needed",
                description = followUpItem.recommendedAction,
                priority = SecretaryPriority.NORMAL,
                reason = SecretaryCardReason.VIP_SENDER,
                primaryAction = SecretaryAction.Local(LocalSecretaryAction.NAV_CHAT)
            )

            addCard(card)
        }
    }

    private fun addBriefingCard(result: BriefingResult, reason: SecretaryCardReason) {
        val cardId = when (reason) {
            SecretaryCardReason.MORNING_BRIEFING -> "morning-${System.currentTimeMillis()}"
            SecretaryCardReason.EVENING_REVIEW -> "evening-${System.currentTimeMillis()}"
            else -> "briefing-${System.currentTimeMillis()}"
        }

        val card = ProactiveCard(
            id = cardId,
            title = result.title,
            description = result.description,
            priority = SecretaryPriority.NORMAL,
            reason = reason,
            primaryAction = result.primaryAction
        )

        addCard(card)
    }

    /**
     * Handle incoming Matrix sync events.
     * This is the core reactive loop for Phase 1.
     *
     * Events to process:
     * - MessageReceived: Incoming message from user's contacts
     * - TypingNotification: Someone is typing
     * - PresenceUpdate: User online status change
     *
     * Phase 1: Only simple deterministic triage based on keywords.
     */
    private fun handleMatrixEvent(event: Any) {
        when (event) {
            is MatrixSyncEvent.MessageReceived -> {
                handleIncomingMessage(event)
            }

            is MatrixSyncEvent.TypingNotification -> {
                logger.logDebug("Typing notification from: ${event.userIds}")
            }

            is MatrixSyncEvent.PresenceUpdate -> {
                logger.logDebug("Presence update received")
            }

            else -> {
                logger.logDebug("Unhandled Matrix event: $event")
            }
        }
    }

    private fun handleIncomingMessage(event: MatrixSyncEvent.MessageReceived) {
        val messageContent = event.event.content?.toString() ?: ""
        val triageInput = TriageInput(
            mode = SecretaryMode.NORMAL,
            messageContent = messageContent,
            isVipSender = false,
            isCalendarLinked = false
        )

        val triageResult = triage.score(triageInput)

        if (triageResult.priority == SecretaryPriority.HIGH || triageResult.priority == SecretaryPriority.CRITICAL) {
            val card = ProactiveCard(
                id = "urgent-${event.event.eventId}-${System.currentTimeMillis()}",
                title = when (triageResult.priority) {
                    SecretaryPriority.CRITICAL -> "Critical Message"
                    SecretaryPriority.HIGH -> "Important Message"
                    else -> "Message"
                },
                description = messageContent.take(100),
                priority = triageResult.priority,
                reason = SecretaryCardReason.URGENT_KEYWORD,
                primaryAction = SecretaryAction.Local(LocalSecretaryAction.OPEN_MESSAGE)
            )

            val policyDecision = policyEngine.evaluateCard(card, PolicyContext(
                mode = SecretaryMode.NORMAL,
                whitelist = emptyList()
            ))

            if (!policyDecision.shouldSuppress) {
                addCard(card)
            } else {
                logger.logDebug("Card suppressed: ${policyDecision.suppressionReason}")
            }
        }
    }

    /**
     * Add an urgent keyword proactive card.
     * Phase 1: First implementation - simple deterministic rule.
     *
     * @param message The Matrix message that triggered this.
     */
    private fun addUrgentCard(message: Message) {
        val card = ProactiveCard(
            id = "urgent-${message.id}",
            title = "Urgent Message",
            description = message.content.body.substring(0, minOf(100, message.content.body.length)),
            priority = SecretaryPriority.HIGH,
            reason = SecretaryCardReason.URGENT_KEYWORD,
            primaryAction = SecretaryAction.Local(LocalSecretaryAction.NAV_CHAT)
        )

        addCard(card)
    }

    /**
     * Add a proactive card to the cards list.
     * Updates state to Proposing if this is the first card.
     */
    private fun addCard(card: ProactiveCard) {
        viewModelScope.launch {
            val currentCards = _cards.value.toMutableList()

            currentCards.removeAll { it.id == card.id }
            currentCards.add(card)
            _cards.value = currentCards.toList()

            when (_cards.value.isEmpty()) {
                true -> _state.value = SecretaryState.Idle
                false -> _state.value = SecretaryState.Proposing(_cards.value)
            }
        }
    }

    /**
     * Remove a proactive card by ID.
     * Updates state appropriately based on remaining cards.
     */
    fun dismissCard(cardId: String) {
        viewModelScope.launch {
            val currentCards = _cards.value.toMutableList()
            currentCards.removeAll { it.id == cardId }
            
            val remaining = currentCards.toList()
            
            when {
                remaining.isEmpty() -> {
                    // Return to Idle if no cards remain
                    _state.value = SecretaryState.Idle
                }
                else -> {
                    // Still have cards, stay in Proposing or keep current state
                    // State transition handled in addCard()
                }
            }
        }
    }

    fun onPrimaryAction(cardId: String, action: SecretaryAction) {
        when (action) {
            is SecretaryAction.Local -> {
                when (action.action) {
                    LocalSecretaryAction.NAV_CHAT -> {
                        logger.logInfo("Navigate to chat for card: $cardId")
                    }
                    LocalSecretaryAction.OPEN_MESSAGE -> {
                        logger.logInfo("Open message for card: $cardId")
                    }
                    LocalSecretaryAction.DISMISS_CARD -> {
                        dismissCard(cardId)
                    }
                    LocalSecretaryAction.SNOOZE_CARD -> {
                        logger.logInfo("Snooze card: $cardId (not yet implemented in Phase 1)")
                    }
                }
            }
        }
    }
}
