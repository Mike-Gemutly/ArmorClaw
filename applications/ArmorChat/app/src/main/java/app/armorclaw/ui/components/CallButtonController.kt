package app.armorclaw.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import app.armorclaw.data.repository.BridgeCapabilities
import app.armorclaw.data.repository.BridgeCapabilitiesRepository
import app.armorclaw.data.repository.Feature
import androidx.compose.runtime.collectAsState

/**
 * Call Button Controller - Voice/Video Call Suppression for Bridged Rooms
 *
 * Resolves: Hardening Area #5 (Broken Voice UX Gap)
 *
 * Checks BridgeCapabilities for the current room and either:
 * - Shows native call button (Matrix-to-Matrix rooms)
 * - Shows disabled button with tooltip (bridged rooms without call support)
 * - Hides button entirely (rooms with no messaging support)
 *
 * Usage in room header:
 *   CallButton(
 *       roomId = currentRoomId,
 *       capabilitiesRepository = bridgeCapabilitiesRepo,
 *       onVoiceCall = { startVoiceCall(it) },
 *       onVideoCall = { startVideoCall(it) }
 *   )
 */

/**
 * Call button state based on room capabilities
 */
sealed class CallButtonState {
    /** Native Matrix room - full call support */
    data object Enabled : CallButtonState()

    /** Bridged room - calls not supported */
    data class Disabled(val protocol: String, val reason: String) : CallButtonState()

    /** Room type doesn't support any interactive features */
    data object Hidden : CallButtonState()
}

/**
 * Determines call button state for a given room
 */
fun resolveCallButtonState(
    capabilities: BridgeCapabilities
): CallButtonState {
    // Check if video calls are supported
    if (capabilities.supports(Feature.VIDEO_CALLS)) {
        return CallButtonState.Enabled
    }

    // Bridged room without call support
    val protocolName = capabilities.protocol.displayName
    return CallButtonState.Disabled(
        protocol = protocolName,
        reason = "Voice/video calls are not available in $protocolName bridged rooms"
    )
}

/**
 * Call button composable that respects bridge capabilities
 */
@Composable
fun CallButton(
    roomId: String,
    capabilitiesRepository: BridgeCapabilitiesRepository,
    onVoiceCall: (String) -> Unit,
    onVideoCall: ((String) -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    // Observe repository version so we recompose when capabilities change
    val repoVersion by capabilitiesRepository.version.collectAsState()
    val capabilities = remember(roomId, repoVersion) {
        capabilitiesRepository.getCapabilities(roomId)
    }
    val state = remember(capabilities) {
        resolveCallButtonState(capabilities)
    }

    when (state) {
        is CallButtonState.Enabled -> {
            Row(
                modifier = modifier,
                horizontalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                // Voice call button
                IconButton(onClick = { onVoiceCall(roomId) }) {
                    Icon(
                        imageVector = Icons.Default.Call,
                        contentDescription = "Voice Call",
                        tint = MaterialTheme.colorScheme.primary
                    )
                }

                // Video call button (optional)
                if (onVideoCall != null) {
                    IconButton(onClick = { onVideoCall(roomId) }) {
                        Icon(
                            imageVector = Icons.Default.Videocam,
                            contentDescription = "Video Call",
                            tint = MaterialTheme.colorScheme.primary
                        )
                    }
                }
            }
        }

        is CallButtonState.Disabled -> {
            var showTooltip by remember { mutableStateOf(false) }

            Box(modifier = modifier) {
                // Disabled call icon with bridge indicator
                IconButton(
                    onClick = { showTooltip = !showTooltip },
                    enabled = false
                ) {
                    Icon(
                        imageVector = Icons.Default.CallEnd,
                        contentDescription = state.reason,
                        tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.38f)
                    )
                }

                // Tooltip showing why calls are unavailable
                if (showTooltip) {
                    Surface(
                        modifier = Modifier
                            .align(Alignment.BottomCenter)
                            .padding(top = 48.dp),
                        shape = MaterialTheme.shapes.small,
                        color = MaterialTheme.colorScheme.inverseSurface,
                        tonalElevation = 2.dp
                    ) {
                        Text(
                            text = state.reason,
                            modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.inverseOnSurface
                        )
                    }
                }
            }
        }

        is CallButtonState.Hidden -> {
            // Render nothing
        }
    }
}
