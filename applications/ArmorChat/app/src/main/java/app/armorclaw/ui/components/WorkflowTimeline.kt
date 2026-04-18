package app.armorclaw.ui.components

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import app.armorclaw.ArmorClawTheme
import org.json.JSONObject

/**
 * Represents a single event in a workflow timeline.
 *
 * Events arrive via Matrix /sync and are surfaced through
 * a ViewModel/StateFlow pattern. This composable does NOT
 * manage WebSocket connections or network state.
 *
 * @param seq         Sequential step number (1-based)
 * @param type        Event type — matches Go bridge `stepIcon()` values
 * @param name        Human-readable step name
 * @param tsMs        Epoch millis when the event occurred
 * @param detail      Optional detail line (e.g. file path, command)
 * @param durationMs  Optional duration in milliseconds — sourced from
 *                    Bridge `StepEvent.duration_ms` via `TimelineEvent.duration_ms`
 */
data class WorkflowEvent(
    val seq: Int,
    val type: String,
    val name: String,
    val tsMs: Long,
    val detail: String = "",
    val durationMs: Long? = null
) {
    companion object {
        /**
         * Parse a [WorkflowEvent] from a Bridge [TimelineEvent] JSON object.
         *
         * The Bridge sends timeline events in the format produced by
         * [GetTimelineEvents] in `notifications.go`:
         * ```json
         * {
         *   "seq": 1,
         *   "type": "step",
         *   "name": "Initializing agent",
         *   "ts_ms": 1710000000000,
         *   "detail": {"lines": 100},
         *   "duration_ms": 1234
         * }
         * ```
         *
         * The `duration_ms` field is respected directly from the Bridge payload
         * (originating from `StepEvent.DurationMs` in `_events.jsonl`), not
         * estimated from timestamps.
         *
         * @param json TimelineEvent JSON object from Bridge
         * @return Parsed WorkflowEvent, or null if required fields are missing
         */
        fun fromTimelineEventJson(json: JSONObject): WorkflowEvent? {
            val seq = json.optInt("seq", -1)
            val type = json.optString("type", "")
            val name = json.optString("name", "")
            val tsMs = json.optLong("ts_ms", 0L)

            if (seq < 0 || type.isBlank() || name.isBlank()) return null

            val detailObj = json.optJSONObject("detail")
            val detail = detailObj?.let { d ->
                buildString {
                    d.optInt("lines").takeIf { it != 0 }?.let { append(it, " lines") }
                    d.optInt("changes").takeIf { it != 0 }?.let { append(it, " changes") }
                    d.optInt("exit_code").takeIf { it != 0 }?.let { append("exit ", it) }
                    d.optLong("size_bytes").takeIf { it != 0L }?.let { append(it, " bytes") }
                    d.optString("message", "").takeIf { it.isNotBlank() }?.let { append(it) }
                    d.optString("selector", "").takeIf { it.isNotBlank() }?.let { append(it) }
                }.trim()
            } ?: ""

            val durationMs = if (json.has("duration_ms")) {
                json.optLong("duration_ms")
            } else {
                null
            }

            return WorkflowEvent(
                seq = seq,
                type = type,
                name = name,
                tsMs = tsMs,
                detail = detail,
                durationMs = durationMs
            )
        }

        fun fromTimelineEventArray(jsonArray: org.json.JSONArray): List<WorkflowEvent> {
            val events = mutableListOf<WorkflowEvent>()
            for (i in 0 until jsonArray.length()) {
                val obj = jsonArray.optJSONObject(i) ?: continue
                fromTimelineEventJson(obj)?.let { events.add(it) }
            }
            return events
        }

        /**
         * Extract duration from a plain-text timeline line.
         *
         * The Bridge's [FormatTimelineMessage] appends `(1234ms)` to event lines
         * when duration is available. This parser extracts that value so that
         * plain-text timeline messages also display accurate durations.
         *
         * @param line A single timeline line, e.g. "🔹 Initializing agent (1234ms)"
         * @return Duration in ms, or null if not found
         */
        fun parseDurationFromText(line: String): Long? {
            val regex = Regex("""\((\d+)ms\)\s*$""")
            val match = regex.find(line) ?: return null
            return match.groupValues[1].toLongOrNull()
        }
    }
}

/**
 * Aggregate state consumed by [WorkflowTimeline].
 *
 * @param events       Ordered list of workflow events
 * @param progress     Overall progress 0.0 – 1.0
 * @param isRunning    Whether the workflow is actively executing
 * @param workflowName Display name for the workflow
 */
data class WorkflowTimelineState(
    val events: List<WorkflowEvent>,
    val progress: Float,
    val isRunning: Boolean,
    val workflowName: String = ""
)

/**
 * Maps a Go-bridge event type to an emoji icon.
 * Must stay in sync with `stepIcon()` in `notifications.go`.
 */
fun eventIcon(type: String): String = when (type) {
    "step" -> "\u2066🔹\u2069"
    "file_read" -> "\u2066📄\u2069"
    "file_write" -> "\u2066✏️\u2069"
    "file_delete" -> "\u2066🗑️\u2069"
    "command_run" -> "\u2066⌨️\u2069"
    "observation" -> "\u2066💭\u2069"
    "blocker" -> "\u2066🚧\u2069"
    "error" -> "\u2066❌\u2069"
    "artifact" -> "\u2066📦\u2069"
    "checkpoint" -> "\u2066🏁\u2069"
    else -> "•"
}

/**
 * Formats a duration in milliseconds to a human-readable string.
 * Examples: "0.3s", "2.3s", "1m 04s", "12m 34s"
 */
fun formatDuration(ms: Long): String {
    if (ms < 1000) return "${ms}ms"
    val totalSeconds = ms / 1000
    val minutes = totalSeconds / 60
    val seconds = totalSeconds % 60
    val fraction = ((ms % 1000) / 100)
    return if (minutes > 0) {
        "${minutes}m ${seconds.toString().padStart(2, '0')}s"
    } else {
        "${seconds}.${fraction}s"
    }
}

/**
 * WorkflowTimeline — scrollable vertical timeline for agent workflow progress.
 *
 * Layout:
 *  - Top: progress bar with percentage label
 *  - Status line: live indicator or completion badge
 *  - LazyColumn of timeline rows
 *  - Empty state when no events
 *
 * All colours use [MaterialTheme.colorScheme] for light/dark theme support.
 */
@Composable
fun WorkflowTimeline(
    state: WorkflowTimelineState,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        color = MaterialTheme.colorScheme.surfaceContainerLow
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp)
        ) {
            // Workflow title
            if (state.workflowName.isNotBlank()) {
                Text(
                    text = state.workflowName,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.onSurface
                )
                Spacer(modifier = Modifier.height(12.dp))
            }

            // Progress bar
            ProgressSection(state = state)

            Spacer(modifier = Modifier.height(12.dp))

            // Status line
            StatusLine(state = state)

            Spacer(modifier = Modifier.height(12.dp))

            // Timeline events or empty state
            if (state.events.isEmpty()) {
                EmptyState()
            } else {
                TimelineList(events = state.events)
            }
        }
    }
}

/**
 * Progress bar with percentage label.
 */
@Composable
private fun ProgressSection(state: WorkflowTimelineState) {
    val progressClamped = state.progress.coerceIn(0f, 1f)
    val animatedProgressColor by animateColorAsState(
        targetValue = if (progressClamped >= 1f) {
            MaterialTheme.colorScheme.primary
        } else {
            MaterialTheme.colorScheme.tertiary
        },
        animationSpec = tween(durationMillis = 400),
        label = "progressColor"
    )

    Column {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "Progress",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Text(
                text = "${(progressClamped * 100).toInt()}%",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        Spacer(modifier = Modifier.height(4.dp))
        LinearProgressIndicator(
            progress = { progressClamped },
            modifier = Modifier
                .fillMaxWidth()
                .height(6.dp),
            color = animatedProgressColor,
            trackColor = MaterialTheme.colorScheme.surfaceContainerHighest,
        )
    }
}

/**
 * Live indicator or completion badge.
 */
@Composable
private fun StatusLine(state: WorkflowTimelineState) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        if (state.isRunning) {
            // Pulsing live dot
            Surface(
                shape = CircleShape,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(8.dp)
            ) {}
            Text(
                text = "Live",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.error
            )
        } else if (state.progress >= 1f) {
            Text(
                text = "✅ Complete",
                style = MaterialTheme.typography.labelMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.primary
            )
        } else {
            Text(
                text = "Paused",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }

        if (state.events.isNotEmpty()) {
            Spacer(modifier = Modifier.weight(1f))
            Text(
                text = "${state.events.size} event${if (state.events.size != 1) "s" else ""}",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f)
            )
        }
    }
}

/**
 * Empty state centered in the available space.
 */
@Composable
private fun EmptyState() {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 32.dp),
        contentAlignment = Alignment.Center
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                text = "Waiting for agent activity...",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = "Events will appear here as the agent works",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
            )
        }
    }
}

/**
 * Scrollable list of timeline event rows.
 */
@Composable
private fun TimelineList(events: List<WorkflowEvent>) {
    LazyColumn(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(2.dp)
    ) {
        items(events, key = { it.seq }) { event ->
            TimelineRow(event = event)
        }
    }
}

/**
 * Single row in the timeline.
 *
 * Layout: [icon] [seq dot] [name + detail] [duration]
 */
@Composable
private fun TimelineRow(event: WorkflowEvent) {
    val iconColor = when (event.type) {
        "error" -> MaterialTheme.colorScheme.error
        "blocker" -> MaterialTheme.colorScheme.error
        "checkpoint" -> MaterialTheme.colorScheme.primary
        "artifact" -> MaterialTheme.colorScheme.tertiary
        else -> MaterialTheme.colorScheme.onSurfaceVariant
    }

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.Top
    ) {
        // Icon
        Text(
            text = eventIcon(event.type),
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.padding(end = 8.dp)
        )

        // Vertical track line indicator
        Box(
            modifier = Modifier
                .width(2.dp)
                .height(24.dp)
                .padding(top = 10.dp)
                .background(
                    MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f),
                    RoundedCornerShape(1.dp)
                )
        )

        Spacer(modifier = Modifier.width(8.dp))

        // Name + detail
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = event.name,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium,
                color = iconColor,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis
            )
            if (event.detail.isNotBlank()) {
                Text(
                    text = event.detail,
                    style = MaterialTheme.typography.bodySmall,
                    fontFamily = FontFamily.Monospace,
                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.7f),
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
            }
        }

        // Duration
        if (event.durationMs != null) {
            Spacer(modifier = Modifier.width(8.dp))
            Surface(
                shape = RoundedCornerShape(6.dp),
                color = MaterialTheme.colorScheme.surfaceContainerHigh
            ) {
                Text(
                    text = formatDuration(event.durationMs),
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                )
            }
        }
    }
}

// ──────────────────────────── Previews ────────────────────────────

private val sampleEvents = listOf(
    WorkflowEvent(
        seq = 1,
        type = "step",
        name = "Initializing agent",
        tsMs = 1710000000000,
        detail = "Setting up sandbox environment",
        durationMs = 1200
    ),
    WorkflowEvent(
        seq = 2,
        type = "file_read",
        name = "Reading configuration",
        tsMs = 1710000001000,
        detail = "/etc/armorclaw/agents/researcher.yaml",
        durationMs = 340
    ),
    WorkflowEvent(
        seq = 3,
        type = "command_run",
        name = "Running web search",
        tsMs = 1710000002000,
        detail = "query: best restaurants NYC birthday",
        durationMs = 4500
    ),
    WorkflowEvent(
        seq = 4,
        type = "observation",
        name = "Found 12 results",
        tsMs = 1710000007000,
        detail = "Filtering by rating > 4.0"
    ),
    WorkflowEvent(
        seq = 5,
        type = "file_write",
        name = "Writing summary",
        tsMs = 1710000008000,
        detail = "/tmp/research-output.md",
        durationMs = 800
    ),
    WorkflowEvent(
        seq = 6,
        type = "artifact",
        name = "Report generated",
        tsMs = 1710000009000,
        detail = "research-report-2024.pdf",
        durationMs = 200
    ),
    WorkflowEvent(
        seq = 7,
        type = "checkpoint",
        name = "Workflow complete",
        tsMs = 1710000010000,
        durationMs = 7340
    )
)

private val sampleRunningState = WorkflowTimelineState(
    events = sampleEvents.take(4),
    progress = 0.57f,
    isRunning = true,
    workflowName = "Restaurant Research"
)

private val sampleCompleteState = WorkflowTimelineState(
    events = sampleEvents,
    progress = 1.0f,
    isRunning = false,
    workflowName = "Restaurant Research"
)

private val sampleEmptyState = WorkflowTimelineState(
    events = emptyList(),
    progress = 0f,
    isRunning = true,
    workflowName = "New Task"
)

@Preview(name = "Timeline — Running", showBackground = true)
@Composable
private fun WorkflowTimelineRunningPreview() {
    ArmorClawTheme {
        WorkflowTimeline(state = sampleRunningState)
    }
}

@Preview(name = "Timeline — Complete", showBackground = true)
@Composable
private fun WorkflowTimelineCompletePreview() {
    ArmorClawTheme {
        WorkflowTimeline(state = sampleCompleteState)
    }
}

@Preview(name = "Timeline — Empty", showBackground = true)
@Composable
private fun WorkflowTimelineEmptyPreview() {
    ArmorClawTheme {
        WorkflowTimeline(state = sampleEmptyState)
    }
}
