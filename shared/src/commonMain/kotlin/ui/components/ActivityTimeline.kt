package com.armorclaw.shared.ui.components

import androidx.compose.animation.*
import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.armorclaw.shared.domain.model.ActivityEvent
import com.armorclaw.shared.domain.model.ActivityEventSeverity
import com.armorclaw.shared.domain.model.ActivityEventType
import com.armorclaw.shared.domain.model.ActivityFilter
import com.armorclaw.shared.ui.theme.AppIcons
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Activity Timeline
 *
 * Real-time event stream displaying agent actions with intervention support.
 * Shows a scrollable list of events with timestamps, severity indicators, and actions.
 *
 * ## Architecture
 * ```
 * ActivityTimeline
 *      ├── FilterBar (optional)
 *      ├── TimelineHeader
 *      └── LazyColumn of TimelineEventItem[]
 *          ├── Timestamp
 *          ├── Event icon with severity color
 *          ├── Event content
 *          └── Action buttons (if applicable)
 * ```
 *
 * ## Usage
 * ```kotlin
 * val events by viewModel.activityTimeline.collectAsState()
 *
 * ActivityTimeline(
 *     events = events,
 *     onEventClick = { event -> viewModel.onEventClick(event) },
 *     onInterventionClick = { event -> viewModel.handleIntervention(event) },
 *     modifier = Modifier.fillMaxSize()
 * )
 * ```
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ActivityTimeline(
    events: List<ActivityEvent>,
    modifier: Modifier = Modifier,
    onEventClick: (ActivityEvent) -> Unit = {},
    onInterventionClick: (ActivityEvent) -> Unit = {},
    showFilters: Boolean = false,
    filter: ActivityFilter = ActivityFilter(),
    onFilterChange: (ActivityFilter) -> Unit = {},
    isLive: Boolean = true
) {
    val filteredEvents = remember(events, filter) {
        events.filter { filter.matches(it) }
            .sortedByDescending { it.timestamp }
    }

    Column(modifier = modifier) {
        // Header with live indicator
        TimelineHeader(
            eventCount = filteredEvents.size,
            isLive = isLive,
            showFilters = showFilters,
            filter = filter,
            onFilterChange = onFilterChange
        )

        // Event list
        if (filteredEvents.isEmpty()) {
            EmptyTimelineContent()
        } else {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                items(
                    items = filteredEvents,
                    key = { it.id }
                ) { event ->
                    TimelineEventItem(
                        event = event,
                        onClick = { onEventClick(event) },
                        onInterventionClick = { onInterventionClick(event) }
                    )
                }
            }
        }
    }
}

/**
 * Timeline header with live indicator and filters
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TimelineHeader(
    eventCount: Int,
    isLive: Boolean,
    showFilters: Boolean,
    filter: ActivityFilter,
    onFilterChange: (ActivityFilter) -> Unit,
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Live indicator
            if (isLive) {
                LiveIndicator()
                Spacer(modifier = Modifier.width(8.dp))
            }

            Text(
                text = "Activity Timeline",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold
            )

            Spacer(modifier = Modifier.weight(1f))

            Text(
                text = "$eventCount events",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }

        // Filter chips
        if (showFilters) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .horizontalScroll(rememberScrollState())
                    .padding(horizontal = 16.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                FilterChip(
                    selected = filter.showInterventions,
                    onClick = { onFilterChange(filter.copy(showInterventions = !filter.showInterventions)) },
                    label = { Text("Needs Attention") },
                    leadingIcon = {
                        Icon(
                            imageVector = Icons.Default.PriorityHigh,
                            contentDescription = null,
                            modifier = Modifier.size(16.dp)
                        )
                    }
                )
                FilterChip(
                    selected = filter.showErrors,
                    onClick = { onFilterChange(filter.copy(showErrors = !filter.showErrors)) },
                    label = { Text("Errors") },
                    leadingIcon = {
                        Icon(
                            imageVector = Icons.Default.Error,
                            contentDescription = null,
                            modifier = Modifier.size(16.dp)
                        )
                    }
                )
                FilterChip(
                    selected = filter.showFormFills,
                    onClick = { onFilterChange(filter.copy(showFormFills = !filter.showFormFills)) },
                    label = { Text("Form Fills") }
                )
                FilterChip(
                    selected = filter.showScreenshots,
                    onClick = { onFilterChange(filter.copy(showScreenshots = !filter.showScreenshots)) },
                    label = { Text("Screenshots") }
                )
            }
            Spacer(modifier = Modifier.height(8.dp))
        }
    }
}

/**
 * Animated live indicator
 */
@Composable
private fun LiveIndicator(modifier: Modifier = Modifier) {
    val infiniteTransition = rememberInfiniteTransition(label = "live")
    val alpha by infiniteTransition.animateFloat(
        initialValue = 0.5f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(500, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "alpha"
    )

    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Box(
            modifier = Modifier
                .size(8.dp)
                .clip(RoundedCornerShape(percent = 50))
                .background(MaterialTheme.colorScheme.error.copy(alpha = alpha))
        )
        Text(
            text = "LIVE",
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.error
        )
    }
}

/**
 * Individual timeline event item
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TimelineEventItem(
    event: ActivityEvent,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    onInterventionClick: () -> Unit = {}
) {
    val (icon, color) = getEventIconAndColor(event.severity, event.icon)
    val timeString = formatEventTime(event.timestamp)

    // Check if this is an attention-requiring event
    val needsAction = event.requiresAttention

    Card(
        onClick = onClick,
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (needsAction) {
                color.copy(alpha = 0.1f)
            } else {
                MaterialTheme.colorScheme.surface
            }
        ),
        elevation = CardDefaults.cardElevation(
            defaultElevation = if (needsAction) 2.dp else 0.dp
        )
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.Top
        ) {
            // Timestamp
            Text(
                text = timeString,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.outline,
                modifier = Modifier.width(48.dp)
            )

            // Event icon
            Box(
                modifier = Modifier
                    .size(32.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .background(color.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = color,
                    modifier = Modifier.size(18.dp)
                )
            }

            Spacer(modifier = Modifier.width(12.dp))

            // Content
            Column(modifier = Modifier.weight(1f)) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(6.dp)
                ) {
                    Text(
                        text = event.title,
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f, fill = false)
                    )

                    // Agent badge
                    Surface(
                        shape = RoundedCornerShape(4.dp),
                        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f)
                    ) {
                        Text(
                            text = event.agentName.take(10),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }

                if (event.description.isNotEmpty()) {
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = event.description,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                }

                // Intervention action button
                if (needsAction) {
                    Spacer(modifier = Modifier.height(8.dp))
                    Button(
                        onClick = onInterventionClick,
                        shape = RoundedCornerShape(8.dp),
                        colors = ButtonDefaults.buttonColors(
                            containerColor = color,
                            contentColor = Color.White
                        ),
                        contentPadding = PaddingValues(horizontal = 12.dp, vertical = 6.dp)
                    ) {
                        Text(
                            text = when (event.eventType) {
                                ActivityEventType.INTERVENTION -> "Resolve"
                                ActivityEventType.APPROVAL -> "Review"
                                ActivityEventType.ERROR -> "Help"
                                else -> "Take Action"
                            },
                            style = MaterialTheme.typography.labelMedium
                        )
                    }
                }
            }
        }
    }
}

/**
 * Get icon and color for event based on severity
 */
@Composable
private fun getEventIconAndColor(
    severity: ActivityEventSeverity,
    iconType: com.armorclaw.shared.domain.model.ActivityEventIcon
): Pair<ImageVector, Color> {
    val color = when (severity) {
        ActivityEventSeverity.INFO -> MaterialTheme.colorScheme.primary
        ActivityEventSeverity.SUCCESS -> MaterialTheme.colorScheme.tertiary
        ActivityEventSeverity.WARNING -> MaterialTheme.colorScheme.secondary
        ActivityEventSeverity.ERROR -> MaterialTheme.colorScheme.error
        ActivityEventSeverity.CRITICAL -> MaterialTheme.colorScheme.error
    }

    val icon = when (iconType) {
        com.armorclaw.shared.domain.model.ActivityEventIcon.LANGUAGE -> Icons.Default.Language
        com.armorclaw.shared.domain.model.ActivityEventIcon.EDIT -> Icons.Default.Edit
        com.armorclaw.shared.domain.model.ActivityEventIcon.TOUCH -> Icons.Default.TouchApp
        com.armorclaw.shared.domain.model.ActivityEventIcon.DOWNLOAD -> Icons.Default.Download
        com.armorclaw.shared.domain.model.ActivityEventIcon.CAMERA -> Icons.Default.PhotoCamera
        com.armorclaw.shared.domain.model.ActivityEventIcon.SHIELD -> Icons.Default.Security
        com.armorclaw.shared.domain.model.ActivityEventIcon.KEY -> Icons.Default.Key
        com.armorclaw.shared.domain.model.ActivityEventIcon.PENDING -> Icons.Default.Pending
        com.armorclaw.shared.domain.model.ActivityEventIcon.ERROR -> AppIcons.Error
        com.armorclaw.shared.domain.model.ActivityEventIcon.SUCCESS -> Icons.Default.CheckCircle
    }

    return Pair(icon, color)
}

/**
 * Format event timestamp
 */
private fun formatEventTime(timestamp: Long): String {
    val instant = Instant.fromEpochMilliseconds(timestamp)
    val localDateTime = instant.toLocalDateTime(TimeZone.currentSystemDefault())
    return "${localDateTime.hour.toString().padStart(2, '0')}:${localDateTime.minute.toString().padStart(2, '0')}"
}

/**
 * Empty timeline content
 */
@Composable
private fun EmptyTimelineContent(modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .fillMaxSize()
            .padding(32.dp),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Icon(
                imageVector = Icons.Default.History,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.outline,
                modifier = Modifier.size(64.dp)
            )
            Text(
                text = "No Activity Yet",
                style = MaterialTheme.typography.titleMedium,
                color = MaterialTheme.colorScheme.outline
            )
            Text(
                text = "Agent activity will appear here in real-time",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline.copy(alpha = 0.7f)
            )
        }
    }
}

/**
 * Preview helper
 */
@Composable
fun ActivityTimelinePreview() {
    val sampleEvents = listOf<ActivityEvent>(
        ActivityEvent.Navigation(
            id = "1",
            agentId = "agent_1",
            agentName = "Checkout Bot",
            roomId = "room_1",
            timestamp = System.currentTimeMillis() - 60000,
            url = "https://example.com/checkout",
            pageTitle = "Checkout"
        ),
        ActivityEvent.FormFill(
            id = "2",
            agentId = "agent_1",
            agentName = "Checkout Bot",
            roomId = "room_1",
            timestamp = System.currentTimeMillis() - 30000,
            fieldName = "Credit Card",
            fieldSelector = "#cc-number",
            isPiiField = true,
            sensitivityLevel = com.armorclaw.shared.domain.model.SensitivityLevel.HIGH
        ),
        ActivityEvent.Intervention(
            id = "3",
            agentId = "agent_1",
            agentName = "Checkout Bot",
            roomId = "room_1",
            timestamp = System.currentTimeMillis(),
            interventionType = com.armorclaw.shared.domain.model.InterventionType.CAPTCHA,
            context = "reCAPTCHA v2 detected"
        )
    )

    ActivityTimeline(
        events = sampleEvents,
        isLive = true
    )
}
