package com.armorclaw.app.secretary

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModel.viewModelScope
import com.armorclaw.shared.domain.model.Message
import com.armorclaw.shared.secretary.*
import com.armorclaw.shared.platform.matrix.MatrixSyncManager
import kotlinx.coroutines.flow.StateIn
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import org.koin.core.component.inject
import org.koin.core.component.KoinComponent
import kotlin.math.max

/**
 * Secretary ViewModel - Phase 1 Foundation
 *
 * Core orchestration layer for the Secretary system.
 * Observes MatrixSyncManager for real-time events and manages proactive cards.
 *
 * Phase 1 scope: Matrix event observation + proactive card display only.
 * Does NOT include:
 *   - Bridge RPC execution (added in Phase 5)
 *   - Privacy guard policies (added in Phase 4)
 *   - Briefing/Review engines (added in Phase 2)
 *   - Context providers (added in Phase 2)
 *   - Calendar/sensor integration
 *   - Voice surfaces
 */
class SecretaryViewModel(
    private val matrixSyncManager: MatrixSyncManager,
    private val logger: AppLogger
) : ViewModel() {

    // State
    private val _state = MutableStateFlow<SecretaryState>(SecretaryState.Idle)
    val state = _state.asStateFlow()

    // Proactive cards list
    private val _cards = MutableStateFlow<List<ProactiveCard>>(emptyList())
    val cards = _cards.asStateFlow()

    init {
        // Start observing Matrix events
        observeMatrixEvents()
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
                // Check if message should trigger proactive card
                val message = event.content
                
                // Simple urgent keyword detection
                val hasUrgentKeyword = message.body.contains("urgent", ignoreCase = true) ||
                                     message.body.contains("asap", ignoreCase = true)
                
                // TODO: Check for VIP sender in Phase 3 (Context & Triage)
                // Phase 1: Simple rule only
                
                if (hasUrgentKeyword) {
                    addUrgentCard(message)
                }
            }
            
            is MatrixSyncEvent.TypingNotification -> {
                // Update typing indicator if we have one (future feature)
                logger.debug("Typing notification from: ${event.senderId}")
            }
            
            is MatrixSyncEvent.PresenceUpdate -> {
                logger.debug("Presence update: ${event.userId} is now ${event.status}")
            }
            
            else -> {
                logger.debug("Unhandled Matrix event: $event")
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
            description = message.body.substring(0, minOf(100, message.body.length)),
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
            
            // Remove duplicate cards by ID
            currentCards.removeAll { existingCard ->
                existingCard.id == card.id
            }
            
            currentCards.add(card)
            _cards.value = currentCards.toList()
            
            // Update state
            val currentCards = _cards.value
            when {
                currentCards.isEmpty() -> {
                    // State remains Idle if no cards
                    _state.value = SecretaryState.Idle
                }
                else -> {
                    // Show cards to user
                    _state.value = SecretaryState.Proposing(currentCards)
                }
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

    /**
     * Handle primary action on a proactive card.
     * Phase 1: Only local navigation actions.
     */
    fun onPrimaryAction(cardId: String, action: SecretaryAction) {
        when (action) {
            LocalSecretaryAction.NAV_CHAT -> {
                // Navigate to chat (handled by Navigation component)
                logger.info("Navigate to chat for card: $cardId")
            }
            LocalSecretaryAction.DISMISS_CARD -> {
                dismissCard(cardId)
            }
            LocalSecretaryAction.SNOOZE_CARD -> {
                // Snooze implementation added in Phase 3 (Context & Triage)
                logger.info("Snooze card: $cardId (not yet implemented in Phase 1)")
            }
            else -> {
                logger.warning("Unknown local action: $action")
            }
        }
    }
}
