# AudioVisualizer Component

> Audio level visualization
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/screens/call/AudioVisualizer.kt`

## Overview

AudioVisualizer provides a visual representation of audio levels during voice calls, showing speaking activity through animated bars.

## Functions

### AudioVisualizer
```kotlin
@Composable
fun AudioVisualizer(
    audioLevel: Float,
    isActive: Boolean,
    modifier: Modifier = Modifier,
    barCount: Int = 5
)
```

**Description:** Animated bar visualization of audio levels.

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `audioLevel` | `Float` | Current audio level (0.0-1.0) |
| `isActive` | `Boolean` | Whether call is active |
| `modifier` | `Modifier` | Optional styling |
| `barCount` | `Int` | Number of bars to display |

---

## Visual Layout

### Active Speaking
```
┌────────────────────────────────────┐
│                                    │
│      ┃  ┃     ┃  ┃                 │
│   ┃  ┃  ┃  ┃  ┃  ┃  ┃              │
│   ┃  ┃  ┃  ┃  ┃  ┃  ┃  ┃           │
│   ▢  ▢  ▢  ▢  ▢  ▢  ▢  ▢          │
│                                    │
│        Alice is speaking           │
│                                    │
└────────────────────────────────────┘
```

### Silence
```
┌────────────────────────────────────┐
│                                    │
│   ▢  ▢  ▢  ▢  ▢  ▢  ▢  ▢          │
│                                    │
└────────────────────────────────────┘
```

---

## Implementation

### Bar Animation
```kotlin
@Composable
fun AudioBar(
    index: Int,
    audioLevel: Float,
    modifier: Modifier = Modifier
) {
    val animatedHeight by animateFloatAsState(
        targetValue = calculateBarHeight(index, audioLevel),
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioMediumBouncy,
            stiffness = Spring.StiffnessLow
        ),
        label = "bar_$index"
    )

    Box(
        modifier = modifier
            .width(8.dp)
            .height(maxHeight * animatedHeight)
            .background(
                color = if (animatedHeight > 0.7f)
                    MaterialTheme.colorScheme.primary
                else
                    MaterialTheme.colorScheme.primary.copy(alpha = 0.6f),
                shape = RoundedCornerShape(4.dp)
            )
    )
}
```

### Height Calculation
```kotlin
fun calculateBarHeight(index: Int, audioLevel: Float): Float {
    val centerIndex = barCount / 2
    val distanceFromCenter = abs(index - centerIndex)
    val falloff = 1f - (distanceFromCenter * 0.15f)

    return (audioLevel * falloff).coerceIn(0.1f, 1f)
}
```

---

## Full Implementation

```kotlin
@Composable
fun AudioVisualizer(
    audioLevel: Float,
    isActive: Boolean,
    modifier: Modifier = Modifier,
    barCount: Int = 5
) {
    val animatedLevel by animateFloatAsState(
        targetValue = if (isActive) audioLevel else 0f,
        animationSpec = tween(100),
        label = "audioLevel"
    )

    Row(
        modifier = modifier
            .height(64.dp)
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.Bottom
    ) {
        repeat(barCount) { index ->
            AudioBar(
                index = index,
                audioLevel = animatedLevel,
                modifier = Modifier.weight(1f)
            )
        }
    }
}
```

---

## Color Scheme

### Bar Colors
| Level | Color | Alpha |
|-------|-------|-------|
| Low (0-30%) | primary | 0.4 |
| Medium (30-70%) | primary | 0.6 |
| High (70-100%) | primary | 1.0 |

### Gradient Option
```kotlin
val gradient = Brush.verticalGradient(
    colors = listOf(
        MaterialTheme.colorScheme.tertiary,
        MaterialTheme.colorScheme.primary
    )
)
```

---

## Animation Specs

### Response Time
| Property | Duration | Easing |
|----------|----------|--------|
| Level rise | 50ms | Linear |
| Level fall | 200ms | EaseOut |
| Bar bounce | Spring | MediumBouncy |

### Spring Configuration
```kotlin
spring(
    dampingRatio = Spring.DampingRatioMediumBouncy,
    stiffness = Spring.StiffnessLow
)
```

---

## State Management

### Audio Level Source
```kotlin
// In CallViewModel
val audioLevel = webRTCClient.audioLevel
    .collectAsState(initial = 0f)

// Usage
AudioVisualizer(
    audioLevel = audioLevel.value,
    isActive = callState.status == CallStatus.ACTIVE
)
```

### Level Smoothing
```kotlin
class AudioLevelSmoother {
    private val buffer = FloatArray(5)
    private var index = 0

    fun addLevel(level: Float): Float {
        buffer[index % buffer.size] = level
        index++
        return buffer.average().toFloat()
    }
}
```

---

## Variants

### Waveform Style
```kotlin
@Composable
fun WaveformVisualizer(
    audioLevels: List<Float>,
    modifier: Modifier = Modifier
) {
    Canvas(modifier = modifier.fillMaxWidth().height(64.dp)) {
        val barWidth = size.width / audioLevels.size

        audioLevels.forEachIndexed { index, level ->
            val barHeight = size.height * level
            drawRect(
                color = Color.Primary,
                topLeft = Offset(index * barWidth, (size.height - barHeight) / 2),
                size = Size(barWidth - 2.dp.toPx(), barHeight)
            )
        }
    }
}
```

### Circular Style
```kotlin
@Composable
fun CircularAudioVisualizer(
    audioLevel: Float,
    modifier: Modifier = Modifier
) {
    val animatedRadius by animateDpAsState(
        targetValue = (32 + audioLevel * 32).dp,
        animationSpec = spring(),
        label = "radius"
    )

    Box(
        modifier = modifier
            .size(animatedRadius * 2)
            .background(
                MaterialTheme.colorScheme.primary.copy(alpha = 0.3f),
                CircleShape
            )
    )
}
```

---

## Accessibility

### Content Descriptions
- Active: "Audio level: [high/medium/low]"
- Inactive: "No audio"

### Live Region
```kotlin
Modifier.semantics {
    liveRegion = LiveRegionMode.Polite
    stateDescription = "Audio level: ${(audioLevel * 100).toInt()}%"
}
```

---

## Related Documentation

- [CallControls](CallControls.md) - Call action buttons
- [ActiveCallScreen](../screens/ActiveCallScreen.md) - Call screen
- [Voice Calls](../features/voice-calls.md) - Call feature
