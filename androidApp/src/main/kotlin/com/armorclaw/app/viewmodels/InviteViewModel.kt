package com.armorclaw.app.viewmodels

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.armorclaw.shared.platform.bridge.*
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant

/**
 * ViewModel for managing invite links
 *
 * Features:
 * - Generate time-limited invite links
 * - Parse and validate incoming invite URLs
 * - Revoke existing invites
 * - Track invite usage
 */
class InviteViewModel(
    private val inviteService: InviteService,
    private val bridgeRepository: com.armorclaw.shared.platform.bridge.BridgeRepository
) : ViewModel() {

    // UI State
    private val _uiState = MutableStateFlow(InviteUiState())
    val uiState: StateFlow<InviteUiState> = _uiState.asStateFlow()

    // Generated link
    val generatedLink: StateFlow<String?> = inviteService.generatedLink

    // All invites
    val inviteLinks: StateFlow<List<InviteLink>> = inviteService.inviteLinks

    init {
        // Observe generated link
        viewModelScope.launch {
            inviteService.generatedLink.collect { link ->
                _uiState.update { it.copy(generatedLink = link) }
            }
        }

        // Observe invite links
        viewModelScope.launch {
            inviteService.inviteLinks.collect { links ->
                _uiState.update { it.copy(inviteLinks = links) }
            }
        }
    }

    /**
     * Generate a new invite link with current server configuration
     */
    fun generateInviteLink(
        expiration: InviteExpiration = InviteExpiration.SEVEN_DAYS,
        maxUses: Int? = null,
        serverName: String? = null,
        welcomeMessage: String? = null
    ) {
        viewModelScope.launch {
            _uiState.update { it.copy(isGenerating = true, error = null) }

            val session = bridgeRepository.getCurrentSession()
            val user = bridgeRepository.getCurrentUser()

            if (session == null) {
                _uiState.update { it.copy(
                    isGenerating = false,
                    error = "Not connected to a server"
                )}
                return@launch
            }

            // Create server config from current connection
            val serverConfig = ServerInviteConfig(
                homeserver = "https://matrix.armorclaw.app", // Would come from session
                bridgeUrl = "https://bridge.armorclaw.app",
                serverName = serverName ?: "ArmorClaw Server",
                serverRegion = "us-east",
                serverDescription = "Secure end-to-end encrypted chat",
                welcomeMessage = welcomeMessage,
                features = ServerFeatures(
                    e2ee = true,
                    voice = true,
                    video = true,
                    fileSharing = true,
                    reactions = true,
                    threads = true
                )
            )

            when (val result = inviteService.generateInviteLink(
                serverConfig = serverConfig,
                expiration = expiration,
                maxUses = maxUses,
                createdBy = user?.id ?: "unknown"
            )) {
                is InviteResult.Success -> {
                    _uiState.update { it.copy(
                        isGenerating = false,
                        lastGeneratedInvite = result.invite,
                        successMessage = "Invite link generated!"
                    )}
                }
                is InviteResult.Error -> {
                    _uiState.update { it.copy(
                        isGenerating = false,
                        error = result.message
                    )}
                }
            }
        }
    }

    /**
     * Parse an incoming invite URL
     */
    fun parseInviteUrl(url: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isParsing = true, parsedInvite = null, error = null) }

            when (val result = inviteService.parseInviteUrl(url)) {
                is InviteParseResult.Valid -> {
                    _uiState.update { it.copy(
                        isParsing = false,
                        parsedInvite = result.invite,
                        canAcceptInvite = true
                    )}
                }
                is InviteParseResult.Expired -> {
                    _uiState.update { it.copy(
                        isParsing = false,
                        parsedInvite = result.invite,
                        canAcceptInvite = false,
                        error = "This invite link has expired"
                    )}
                }
                is InviteParseResult.Exhausted -> {
                    _uiState.update { it.copy(
                        isParsing = false,
                        parsedInvite = result.invite,
                        canAcceptInvite = false,
                        error = "This invite link has reached its usage limit"
                    )}
                }
                is InviteParseResult.Revoked -> {
                    _uiState.update { it.copy(
                        isParsing = false,
                        parsedInvite = result.invite,
                        canAcceptInvite = false,
                        error = "This invite link has been revoked"
                    )}
                }
                is InviteParseResult.Error -> {
                    _uiState.update { it.copy(
                        isParsing = false,
                        error = result.message
                    )}
                }
            }
        }
    }

    /**
     * Accept an invite and configure server
     */
    fun acceptInvite() {
        val invite = _uiState.value.parsedInvite ?: return

        viewModelScope.launch {
            _uiState.update { it.copy(isAccepting = true) }

            // Record usage
            inviteService.recordInviteUsage(invite.id)

            // Store the server config for use in setup
            _uiState.update { it.copy(
                isAccepting = false,
                inviteServerConfig = invite.serverConfig,
                successMessage = "Server configured! You can now create your account."
            )}
        }
    }

    /**
     * Revoke an invite link
     */
    fun revokeInvite(inviteId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isRevoking = true, error = null) }

            when (val result = inviteService.revokeInviteLink(inviteId)) {
                is InviteResult.Success -> {
                    _uiState.update { it.copy(
                        isRevoking = false,
                        successMessage = "Invite link revoked"
                    )}
                }
                is InviteResult.Error -> {
                    _uiState.update { it.copy(
                        isRevoking = false,
                        error = result.message
                    )}
                }
            }
        }
    }

    /**
     * Copy invite link to clipboard
     */
    fun copyInviteLink(): String? {
        return inviteService.generatedLink.value
    }

    /**
     * Share invite link
     */
    fun getShareText(): String {
        val invite = _uiState.value.lastGeneratedInvite ?: return ""
        val link = inviteService.generatedLink.value ?: return ""

        return buildString {
            append("Join me on ${invite.serverConfig.serverName ?: "ArmorClaw"}!\n\n")
            invite.serverConfig.serverDescription?.let { append("$it\n\n") }
            append("Click to join: $link\n\n")
            append("This invite link expires: ${formatExpiration(invite.expiresAt)}")
        }
    }

    /**
     * Clear messages
     */
    fun clearMessages() {
        _uiState.update { it.copy(
            error = null,
            successMessage = null
        )}
    }

    /**
     * Clear generated link
     */
    fun clearGeneratedLink() {
        inviteService.clearGeneratedLink()
        _uiState.update { it.copy(
            generatedLink = null,
            lastGeneratedInvite = null
        )}
    }

    /**
     * Clear parsed invite
     */
    fun clearParsedInvite() {
        _uiState.update { it.copy(
            parsedInvite = null,
            canAcceptInvite = false,
            inviteServerConfig = null
        )}
    }

    private fun formatExpiration(expiresAt: Instant): String {
        val now = Clock.System.now()
        val duration = expiresAt - now

        return when {
            duration.inWholeHours < 1 -> "in ${duration.inWholeMinutes} minutes"
            duration.inWholeDays < 1 -> "in ${duration.inWholeHours} hours"
            else -> "in ${duration.inWholeDays} days"
        }
    }
}

/**
 * UI State for invite screens
 */
data class InviteUiState(
    val isGenerating: Boolean = false,
    val isParsing: Boolean = false,
    val isAccepting: Boolean = false,
    val isRevoking: Boolean = false,
    val generatedLink: String? = null,
    val lastGeneratedInvite: InviteLink? = null,
    val parsedInvite: InviteLink? = null,
    val canAcceptInvite: Boolean = false,
    val inviteServerConfig: ServerInviteConfig? = null,
    val inviteLinks: List<InviteLink> = emptyList(),
    val error: String? = null,
    val successMessage: String? = null
) {
    val isLoading: Boolean
        get() = isGenerating || isParsing || isAccepting || isRevoking

    val hasInviteLinks: Boolean
        get() = inviteLinks.isNotEmpty()

    val activeInvites: List<InviteLink>
        get() = inviteLinks.filter { it.isActive && !it.isExpired && !it.isExhausted }

    val expiredInvites: List<InviteLink>
        get() = inviteLinks.filter { it.isExpired }

    val exhaustedInvites: List<InviteLink>
        get() = inviteLinks.filter { it.isExhausted }

    val revokedInvites: List<InviteLink>
        get() = inviteLinks.filter { !it.isActive }
}
