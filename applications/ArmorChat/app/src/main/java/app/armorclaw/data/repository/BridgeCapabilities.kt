package app.armorclaw.data.repository

/**
 * Bridge Capabilities - Platform Feature Support
 *
 * Resolves: G-05 (Feature Suppression)
 *
 * Models bridge capabilities from m.room.bridge state event.
 * Used to conditionally show/hide features based on platform support.
 */

/**
 * Bridge capabilities for a specific room
 *
 * Parsed from m.room.bridge state event content
 */
data class BridgeCapabilities(
    val bridgeId: String,
    val protocol: BridgeProtocol,
    val networkId: String?,
    val channel: String?,
    val features: Set<Feature>,
    val limitations: Set<Limitation>
) {
    /**
     * Check if a feature is supported
     */
    fun supports(feature: Feature): Boolean = feature in features

    /**
     * Check if there's a limitation
     */
    fun hasLimitation(limitation: Limitation): Boolean = limitation in limitations

    companion object {
        /**
         * No bridge - native Matrix room
         */
        val NATIVE_MATRIX = BridgeCapabilities(
            bridgeId = "",
            protocol = BridgeProtocol.MATRIX,
            networkId = null,
            channel = null,
            features = Feature.entries.toSet(),
            limitations = emptySet()
        )
    }
}

/**
 * Supported bridge protocols
 */
enum class BridgeProtocol(val displayName: String, val iconRes: Int?) {
    MATRIX("Matrix", null),
    SLACK("Slack", android.R.drawable.ic_menu_send),
    DISCORD("Discord", android.R.drawable.ic_menu_send),
    TELEGRAM("Telegram", android.R.drawable.ic_menu_send),
    WHATSAPP("WhatsApp", android.R.drawable.ic_menu_send),
    SIGNAL("Signal", android.R.drawable.ic_menu_send),
    IRC("IRC", android.R.drawable.ic_menu_send),
    XMPP("XMPP", android.R.drawable.ic_menu_send),
    TEAMS("Microsoft Teams", android.R.drawable.ic_menu_send),
    UNKNOWN("Unknown", android.R.drawable.ic_menu_send);

    companion object {
        fun fromString(s: String): BridgeProtocol {
            return values().find { it.name.equals(s, ignoreCase = true) } ?: UNKNOWN
        }
    }
}

/**
 * Features that may or may not be supported by a bridge
 */
enum class Feature(val displayName: String) {
    // Messaging features
    TEXT_MESSAGES("Text Messages"),
    MARKDOWN("Markdown Formatting"),
    EMOJI("Emoji"),
    MENTIONS("User Mentions"),

    // Rich content
    IMAGES("Images"),
    VIDEOS("Videos"),
    AUDIO("Audio"),
    FILES("File Attachments"),
    LINKS("Link Previews"),

    // Interactive features
    REACTIONS("Reactions"),
    REPLIES("Threaded Replies"),
    EDITS("Message Editing"),
    DELETION("Message Deletion"),
    TYPING_INDICATORS("Typing Indicators"),
    READ_RECEIPTS("Read Receipts"),

    // Advanced features
    THREADS("Threads"),
    POLLS("Polls"),
    STICKERS("Stickers"),
    LOCATION("Location Sharing"),
    VOICE_MESSAGES("Voice Messages"),
    VIDEO_CALLS("Video Calls"),

    // Bridge-specific
    REDACTED_CONTENT("Redacted Content"),
    BRIDGE_ERRORS("Bridge Error Messages")
}

/**
 * Known bridge limitations
 */
enum class Limitation(val displayName: String, val description: String) {
    NO_UNICODE_EMOJI("No Unicode Emoji", "Bridge only supports ASCII emoticons"),
    NO_EDIT_HISTORY("No Edit History", "Edits replace original without history"),
    LIMITED_FILE_SIZE("Limited File Size", "Files over size limit may fail"),
    NO_REPLY_FALLBACK("No Reply Fallback", "Replies may not show context"),
    MESSAGE_DELAY("Message Delay", "Bridge has noticeable delay"),
    NO_DMS("No DMs", "Direct messages not supported"),
    GROUP_ONLY("Group Only", "Only group chats supported"),
    RATE_LIMITED("Rate Limited", "Bridge has strict rate limits"),
    NO_E2EE("No E2EE", "End-to-end encryption not supported")
}

/**
 * Bridge state event content (m.room.bridge)
 *
 * Reference: https://spec.matrix.org/v1.9/rooms/v10/#mroombridge
 */
data class BridgeStateEvent(
    val bridgebot: String,
    val protocol: BridgeProtocolInfo,
    val network: NetworkInfo?,
    val channel: ChannelInfo?,
    val creator: String? = null,
    val externalUrl: String? = null
)

data class BridgeProtocolInfo(
    val id: String,
    val displayName: String,
    val avatarUrl: String? = null,
    val iconUrl: String? = null
)

data class NetworkInfo(
    val id: String,
    val displayName: String,
    val avatarUrl: String? = null,
    val externalUrl: String? = null
)

data class ChannelInfo(
    val id: String,
    val displayName: String,
    val avatarUrl: String? = null,
    val externalUrl: String? = null
)

/**
 * Repository for bridge capabilities
 */
class BridgeCapabilitiesRepository {

    private val _capabilities = mutableMapOf<String, BridgeCapabilities>()
    val capabilities: Map<String, BridgeCapabilities> get() = _capabilities.toMap()

    /**
     * Get capabilities for a room
     * Returns NATIVE_MATRIX if no bridge exists
     */
    fun getCapabilities(roomId: String): BridgeCapabilities {
        return _capabilities[roomId] ?: BridgeCapabilities.NATIVE_MATRIX
    }

    /**
     * Parse capabilities from m.room.bridge state event
     */
    fun parseCapabilities(roomId: String, event: BridgeStateEvent): BridgeCapabilities {
        val protocol = BridgeProtocol.fromString(event.protocol.id)

        val (features, limitations) = inferCapabilitiesAndLimitations(protocol)

        val capabilities = BridgeCapabilities(
            bridgeId = event.bridgebot,
            protocol = protocol,
            networkId = event.network?.id,
            channel = event.channel?.id,
            features = features,
            limitations = limitations
        )

        _capabilities[roomId] = capabilities
        return capabilities
    }

    /**
     * Infer capabilities and limitations from protocol type
     *
     * These are based on common bridge implementations.
     * Real implementations should query the bridge's capabilities endpoint.
     */
    private fun inferCapabilitiesAndLimitations(
        protocol: BridgeProtocol
    ): Pair<Set<Feature>, Set<Limitation>> {
        return when (protocol) {
            BridgeProtocol.MATRIX -> {
                Pair(Feature.entries.toSet(), emptySet())
            }

            BridgeProtocol.SLACK -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.MARKDOWN,
                        Feature.EMOJI,
                        Feature.MENTIONS,
                        Feature.IMAGES,
                        Feature.VIDEOS,
                        Feature.AUDIO,
                        Feature.FILES,
                        Feature.LINKS,
                        Feature.REACTIONS,
                        Feature.EDITS,
                        Feature.DELETION,
                        Feature.THREADS
                    ),
                    limitations = setOf(
                        Limitation.NO_E2EE
                    )
                )
            }

            BridgeProtocol.DISCORD -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.MARKDOWN,
                        Feature.EMOJI,
                        Feature.MENTIONS,
                        Feature.IMAGES,
                        Feature.VIDEOS,
                        Feature.AUDIO,
                        Feature.FILES,
                        Feature.LINKS,
                        Feature.REACTIONS,
                        Feature.EDITS,
                        Feature.DELETION,
                        Feature.THREADS,
                        Feature.STICKERS,
                        Feature.VOICE_MESSAGES
                    ),
                    limitations = setOf(
                        Limitation.NO_E2EE
                    )
                )
            }

            BridgeProtocol.TELEGRAM -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.MARKDOWN,
                        Feature.EMOJI,
                        Feature.MENTIONS,
                        Feature.IMAGES,
                        Feature.VIDEOS,
                        Feature.AUDIO,
                        Feature.FILES,
                        Feature.LINKS,
                        Feature.REACTIONS,
                        Feature.EDITS,
                        Feature.DELETION,
                        Feature.VOICE_MESSAGES,
                        Feature.LOCATION
                    ),
                    limitations = setOf(
                        Limitation.NO_E2EE
                    )
                )
            }

            BridgeProtocol.WHATSAPP -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.EMOJI,
                        Feature.IMAGES,
                        Feature.VIDEOS,
                        Feature.AUDIO,
                        Feature.FILES,
                        Feature.LINKS,
                        Feature.REACTIONS,
                        Feature.DELETION,
                        Feature.VOICE_MESSAGES,
                        Feature.LOCATION
                    ),
                    limitations = setOf(
                        Limitation.NO_UNICODE_EMOJI,
                        Limitation.NO_EDIT_HISTORY,
                        Limitation.NO_E2EE
                    )
                )
            }

            BridgeProtocol.SIGNAL -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.EMOJI,
                        Feature.MENTIONS,
                        Feature.IMAGES,
                        Feature.VIDEOS,
                        Feature.AUDIO,
                        Feature.FILES,
                        Feature.LINKS,
                        Feature.REACTIONS,
                        Feature.DELETION
                    ),
                    limitations = emptySet() // Signal has E2EE natively
                )
            }

            BridgeProtocol.IRC -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.MENTIONS
                    ),
                    limitations = setOf(
                        Limitation.NO_UNICODE_EMOJI,
                        Limitation.NO_E2EE,
                        Limitation.RATE_LIMITED
                    )
                )
            }

            BridgeProtocol.XMPP -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.MARKDOWN,
                        Feature.EMOJI,
                        Feature.MENTIONS,
                        Feature.IMAGES,
                        Feature.FILES,
                        Feature.LINKS
                    ),
                    limitations = setOf(
                        Limitation.NO_E2EE
                    )
                )
            }

            BridgeProtocol.TEAMS -> {
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.MARKDOWN,
                        Feature.EMOJI,
                        Feature.MENTIONS,
                        Feature.IMAGES,
                        Feature.VIDEOS,
                        Feature.AUDIO,
                        Feature.FILES,
                        Feature.LINKS,
                        Feature.REACTIONS,
                        Feature.EDITS,
                        Feature.DELETION,
                        Feature.THREADS
                    ),
                    limitations = setOf(
                        Limitation.NO_E2EE
                    )
                )
            }

            BridgeProtocol.UNKNOWN -> {
                // Conservative defaults for unknown bridges
                Pair(
                    features = setOf(
                        Feature.TEXT_MESSAGES,
                        Feature.EMOJI,
                        Feature.IMAGES,
                        Feature.FILES,
                        Feature.LINKS
                    ),
                    limitations = setOf(
                        Limitation.NO_E2EE
                    )
                )
            }
        }
    }

    /**
     * Clear capabilities for a room
     */
    fun clearCapabilities(roomId: String) {
        _capabilities.remove(roomId)
    }

    /**
     * Clear all capabilities
     */
    fun clearAll() {
        _capabilities.clear()
    }
}
