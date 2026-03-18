# Performance Feature

> App profiling and performance monitoring
> Location: `androidApp/src/main/kotlin/com/armorclaw/app/platform/`

## Overview

The performance feature provides tools for monitoring, profiling, and optimizing app performance including memory usage, startup time, and render performance.

## Feature Components

### PerformanceProfiler
**Location:** `platform/PerformanceProfiler.kt`

Performance profiling and metrics collection.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `startProfiling()` | Begin profiling session | `sessionName` |
| `stopProfiling()` | End profiling session | - |
| `recordOperation()` | Record timed operation | `name`, `durationMs` |
| `getMetrics()` | Get collected metrics | - |
| `clearMetrics()` | Reset metrics | - |

#### Metrics Collected
| Metric | Type | Description |
|--------|------|-------------|
| Startup Time | Duration | Cold start to interactive |
| Screen Render | Duration | Frame render time |
| Network Latency | Duration | API response time |
| Memory Usage | Bytes | Current heap allocation |
| Database Query | Duration | Query execution time |

---

### MemoryMonitor
**Location:** `platform/MemoryMonitor.kt`

Memory usage tracking and leak detection.

#### Functions

| Function | Description | Parameters |
|----------|-------------|------------|
| `getMemoryInfo()` | Get current memory state | - |
| `logMemoryUsage()` | Record memory snapshot | `tag` |
| `isLowMemory()` | Check if memory constrained | - |
| `triggerGC()` | Request garbage collection | - |
| `getLeakCanaryRefWatcher()` | Leak detection reference | - |

#### Memory States
```kotlin
data class MemoryInfo(
    val totalMem: Long,
    val freeMem: Long,
    val usedMem: Long,
    val maxHeap: Long,
    val availableHeap: Long,
    val isLowMemory: Boolean,
    val trimLevel: Int
)
```

---

## Performance Categories

### App Startup
| Metric | Target | Description |
|--------|--------|-------------|
| Cold Start | < 2s | App not in memory |
| Warm Start | < 1s | App in background |
| Hot Start | < 500ms | App in recent |

### Frame Rendering
| Metric | Target | Description |
|--------|--------|-------------|
| Frame Time | < 16ms | 60 FPS target |
| Jank Count | 0 | Missed frames |
| Slow Frames | < 5% | Frames > 16ms |

### Network
| Metric | Target | Description |
|--------|--------|-------------|
| Message Send | < 500ms | Send latency |
| Message Receive | < 1s | Full roundtrip |
| Image Load | < 2s | Full resolution |

### Database
| Metric | Target | Description |
|--------|--------|-------------|
| Message Insert | < 50ms | Single message |
| Bulk Insert | < 500ms | 100 messages |
| Query (simple) | < 10ms | Indexed query |
| Query (complex) | < 100ms | Join queries |

---

## Performance Dashboard

### Debug Overlay
```
┌────────────────────────────────────┐
│ ⚡ Performance                      │
├────────────────────────────────────┤
│ CPU: 12% ████████░░░░░░░░░░░░      │
│ MEM: 156MB ████████████░░░░░░░     │
│ FPS: 58 ████████████████████░░     │
│                                    │
│ Recent Operations:                 │
│ ├─ loadMessages: 234ms             │
│ ├─ sendMessage: 89ms               │
│ └─ renderList: 12ms                │
│                                    │
│ [Clear] [Export] [Share]           │
└────────────────────────────────────┘
```

---

## Performance Annotations

### Composable Performance
```kotlin
@Composable
@OptIn(ExperimentalComposeRuntimeApi::class)
fun TrackedComposable(
    name: String,
    content: @Composable () -> Unit
) {
    if (BuildConfig.DEBUG) {
        key(name) {
            val startTime = remember { System.nanoTime() }
            content()
            DisposableEffect(Unit) {
                onDispose {
                    val duration = (System.nanoTime() - startTime) / 1_000_000
                    PerformanceProfiler.record("$name.compose", duration)
                }
            }
        }
    } else {
        content()
    }
}
```

### Coroutine Performance
```kotlin
suspend fun <T> trackedOperation(
    name: String,
    block: suspend () -> T
): T {
    val startTime = System.currentTimeMillis()
    return try {
        block()
    } finally {
        val duration = System.currentTimeMillis() - startTime
        PerformanceProfiler.record(name, duration)
    }
}
```

---

## Memory Management

### Memory Thresholds
| Level | Action |
|-------|--------|
| < 70% heap | Normal operation |
| 70-85% heap | Start caching cleanup |
| 85-95% heap | Aggressive cleanup |
| > 95% heap | Critical, clear caches |

### Image Memory Management
```kotlin
// Coil configuration for memory efficiency
val imageLoader = ImageLoader.Builder(context)
    .memoryCache {
        MemoryCache.Builder(context)
            .maxSizePercent(0.25) // 25% of available memory
            .build()
    }
    .diskCache {
        DiskCache.Builder()
            .directory(cacheDir.resolve("image_cache"))
            .maxSizeBytes(512 * 1024 * 1024) // 512MB
            .build()
    }
    .build()
```

---

## Performance Testing

### Benchmark Configuration
```kotlin
@RunWith(AndroidJUnit4::class)
class MessageListBenchmark {

    @get:Rule
    val benchmarkRule = BenchmarkRule()

    @Test
    fun benchmarkMessageRendering() = benchmarkRule.measureRepeated(
        packageName = "com.armorclaw.app",
        metrics = listOf(FrameTimingMetric()),
        iterations = 10
    ) {
        // Scroll through message list
        device.wait(Until.hasObject(By.res("message_list")), 5000)
        device.findObject(By.res("message_list")).scroll(Direction.DOWN, 100f)
    }
}
```

### Baseline Profile
```kotlin
// baseline-prof.txt
Lcom/armorclaw/app/screens/chat/ChatScreen;
Lcom/armorclaw/app/screens/home/HomeScreen;
Lcom/armorclaw/app/viewmodels/ChatViewModel;
```

---

## Crash Reporting Integration

### Sentry Configuration
```kotlin
class SentryInitializer {
    fun init(context: Context) {
        if (!BuildConfig.DEBUG) {
            SentryAndroid.init(context) { options ->
                options.dsn = BuildConfig.SENTRY_DSN
                options.tracesSampleRate = 0.2 // 20% of transactions
                options.profilesSampleRate = 0.1 // 10% profiled
            }
        }
    }
}
```

### Performance Spans
```kotlin
fun sendMessageWithTracing(message: Message) {
    val span = Sentry.startTransaction("message.send", "operation")
    try {
        span.setStatus(SpanStatus.OK)
        repository.sendMessage(message)
    } catch (e: Exception) {
        span.status = SpanStatus.INTERNAL_ERROR
        span.throwable = e
        throw e
    } finally {
        span.finish()
    }
}
```

---

## Optimization Tips

### Compose Optimization
- Use `remember` for expensive calculations
- Use `derivedStateOf` for derived state
- Use `key()` for list items
- Avoid recomposition with `stable` annotations

### Database Optimization
- Use indexes on frequently queried columns
- Batch database operations
- Use transactions for multiple writes
- Implement proper pagination

### Network Optimization
- Compress request/response payloads
- Use connection pooling
- Implement request caching
- Cancel in-flight requests on screen exit

---

## Related Documentation

- [Offline Sync](offline-sync.md) - Background sync
- [Architecture](../ARCHITECTURE.md) - System design
- [Developer Guide](../DEVELOPER_GUIDE.md) - Development setup
