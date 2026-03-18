package com.armorclaw.app.performance
import android.os.StrictMode
import com.armorclaw.app.BuildConfig

import android.os.Debug
import android.os.Trace
import android.util.Log
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.io.File

/**
 * Performance profiler for tracking app metrics
 * 
 * This class tracks method execution time, memory allocations,
 * and provides tracing for performance analysis.
 */
class PerformanceProfiler(
    private val enabled: Boolean = BuildConfig.DEBUG
) {
    
    private val _activeTraces = MutableStateFlow<List<ActiveTrace>>(emptyList())
    val activeTraces: StateFlow<List<ActiveTrace>> = _activeTraces.asStateFlow()
    
    init {
        Log.d(TAG, "PerformanceProfiler initialized (enabled: $enabled)")
    }
    
    /**
     * Start a trace section
     */
    fun beginTrace(name: String) {
        if (!enabled) return
        
        Trace.beginSection(name)
        
        // Track active trace
        val trace = ActiveTrace(name, System.currentTimeMillis())
        _activeTraces.value = _activeTraces.value + trace
        
        Log.v(TAG, "Trace started: $name")
    }
    
    /**
     * End a trace section
     */
    fun endTrace() {
        if (!enabled) return
        
        Trace.endSection()
        
        // Update active traces
        val activeTracesList = _activeTraces.value.toMutableList()
        if (activeTracesList.isNotEmpty()) {
            val trace = activeTracesList.removeLast()
            _activeTraces.value = activeTracesList
            
            val duration = System.currentTimeMillis() - trace.startTime
            Log.v(TAG, "Trace ended: ${trace.name} (${duration}ms)")
        }
    }
    
    /**
     * Execute a block with tracing
     */
    suspend fun <T> trace(name: String, block: suspend () -> T): T {
        if (enabled) {
            beginTrace(name)
            try {
                return block()
            } finally {
                endTrace()
            }
        } else {
            return block()
        }
    }
    
    /**
     * Execute a block with memory allocation tracking
     */
    suspend fun <T> trackAllocations(name: String, block: suspend () -> T): AllocationResult<T> {
        if (!enabled) {
            val result = block()
            return AllocationResult(result, 0L, 0L, 0L)
        }
        
        // Use Runtime for memory tracking (Debug.MemoryInfo fields are deprecated)
        val runtime = Runtime.getRuntime()
        val startAllocated = runtime.totalMemory() - runtime.freeMemory()
        
        try {
            val result = block()
            
            // Get final memory info
            val endAllocated = runtime.totalMemory() - runtime.freeMemory()
            
            val allocated = endAllocated - startAllocated
            val freed = 0L // Not easily measurable
            
            return AllocationResult(
                result = result,
                allocated = allocated,
                freed = freed,
                current = endAllocated
            )
        } catch (e: Exception) {
            Log.e(TAG, "Error in tracked block: $name", e)
            throw e
        }
    }
    
    /**
     * Dump heap to file
     */
    fun dumpHeap(outputFile: File): Boolean {
        if (!enabled) return false
        
        return try {
            Debug.dumpHprofData(outputFile.absolutePath)
            Log.d(TAG, "Heap dumped to: ${outputFile.absolutePath}")
            true
        } catch (e: Exception) {
            Log.e(TAG, "Failed to dump heap", e)
            false
        }
    }
    
    /**
     * Start strict mode (development only)
     */
    fun enableStrictMode() {
        if (!enabled) return
        
        StrictMode.setThreadPolicy(
            StrictMode.ThreadPolicy.Builder()
                .detectAll()
                .penaltyLog()
                .penaltyFlashScreen()
                .build()
        )
        
        StrictMode.setVmPolicy(
            StrictMode.VmPolicy.Builder()
                .detectAll()
                .penaltyLog()
                .build()
        )
        
        Log.d(TAG, "Strict mode enabled")
    }
    
    /**
     * Disable strict mode
     */
    fun disableStrictMode() {
        StrictMode.setThreadPolicy(StrictMode.ThreadPolicy.LAX)
        StrictMode.setVmPolicy(StrictMode.VmPolicy.LAX)
        
        Log.d(TAG, "Strict mode disabled")
    }
    
    /**
     * Get current method count
     * Note: Method counting APIs were removed in Android. This returns a placeholder.
     */
    fun getMethodCount(): Int {
        if (!enabled) return 0
        
        // Method counting APIs are deprecated/removed
        // Use Android Studio Profiler for method tracing
        return 0
    }
    
    /**
     * Reset method counting
     * Note: Method counting APIs were removed in Android.
     */
    fun resetMethodCounting() {
        if (!enabled) return
        
        // Method counting APIs are deprecated/removed
        // Use Android Studio Profiler for method tracing
        Log.d(TAG, "Method counting reset (not supported on this API level)")
    }
    
    companion object {
        private const val TAG = "PerformanceProfiler"
    }
}

/**
 * Active trace data class
 */
data class ActiveTrace(
    val name: String,
    val startTime: Long
)

/**
 * Allocation tracking result
 */
data class AllocationResult<T>(
    val result: T,
    val allocated: Long,
    val freed: Long,
    val current: Long
)
