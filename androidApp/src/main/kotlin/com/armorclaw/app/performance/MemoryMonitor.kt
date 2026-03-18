package com.armorclaw.app.performance

import android.app.ActivityManager
import android.content.Context
import com.armorclaw.app.BuildConfig
import android.os.Debug
import android.os.Handler
import android.os.Looper
import android.util.Log
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Memory monitor for tracking app memory usage and leaks
 * 
 * This class monitors memory usage, detects potential leaks,
 * and provides memory pressure warnings.
 */
class MemoryMonitor(
    private val context: Context,
    private val enabled: Boolean = BuildConfig.DEBUG
) {
    
    private val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
    private val handler = Handler(Looper.getMainLooper())
    
    private val _memoryInfo = MutableStateFlow<MemoryInfo?>(null)
    val memoryInfo: StateFlow<MemoryInfo?> = _memoryInfo.asStateFlow()
    
    private val _memoryPressure = MutableStateFlow(MemoryPressure.NORMAL)
    val memoryPressure: StateFlow<MemoryPressure> = _memoryPressure.asStateFlow()
    
    private val _isLowMemory = MutableStateFlow(false)
    val isLowMemory: StateFlow<Boolean> = _isLowMemory.asStateFlow()
    
    private val memoryMonitorRunnable = object : Runnable {
        override fun run() {
            checkMemoryUsage()
            handler.postDelayed(this, POLL_INTERVAL_MS)
        }
    }
    
    init {
        Log.d(TAG, "MemoryMonitor initialized (enabled: $enabled)")
        
        if (enabled) {
            startMonitoring()
        }
    }
    
    /**
     * Start memory monitoring
     */
    fun startMonitoring() {
        if (!enabled) return
        
        handler.post(memoryMonitorRunnable)
        Log.d(TAG, "Memory monitoring started")
    }
    
    /**
     * Stop memory monitoring
     */
    fun stopMonitoring() {
        handler.removeCallbacks(memoryMonitorRunnable)
        Log.d(TAG, "Memory monitoring stopped")
    }
    
    /**
     * Check current memory usage
     */
    private fun checkMemoryUsage() {
        val androidMemoryInfo = android.app.ActivityManager.MemoryInfo()
        activityManager.getMemoryInfo(androidMemoryInfo)
        
        val debugMemoryInfo = Debug.MemoryInfo()
        Debug.getMemoryInfo(debugMemoryInfo)
        
        // Calculate memory info
        val totalMem = androidMemoryInfo.totalMem
        val availMem = androidMemoryInfo.availMem
        val usedMem = totalMem - availMem
        val usedPercentage = (usedMem.toFloat() / totalMem.toFloat() * 100).toInt()
        
        // Native memory - using Runtime instead of deprecated Debug.MemoryInfo fields
        val runtime = Runtime.getRuntime()
        val nativeHeapSize = runtime.totalMemory()
        val nativeHeapAllocated = runtime.totalMemory() - runtime.freeMemory()
        val nativeHeapFree = runtime.freeMemory()
        
        // Memory pressure
        val isLowMemory = androidMemoryInfo.lowMemory
        val memoryPressure = when {
            isLowMemory -> MemoryPressure.CRITICAL
            usedPercentage > MEMORY_HIGH_THRESHOLD -> MemoryPressure.HIGH
            usedPercentage > MEMORY_MEDIUM_THRESHOLD -> MemoryPressure.MEDIUM
            else -> MemoryPressure.NORMAL
        }
        
        // Update state
        _memoryInfo.value = MemoryInfo(
            totalMemory = totalMem,
            availableMemory = availMem,
            usedMemory = usedMem,
            usedPercentage = usedPercentage,
            isLowMemory = isLowMemory,
            nativeHeapSize = nativeHeapSize,
            nativeHeapAllocated = nativeHeapAllocated,
            nativeHeapFree = nativeHeapFree,
            timestamp = System.currentTimeMillis()
        )
        
        _isLowMemory.value = isLowMemory
        _memoryPressure.value = memoryPressure
        
        if (isLowMemory || memoryPressure != MemoryPressure.NORMAL) {
            Log.w(
                TAG,
                "Memory pressure: $memoryPressure, " +
                "Used: $usedPercentage%, " +
                "Avail: ${availMem / (1024 * 1024)}MB"
            )
        }
    }
    
    /**
     * Trigger garbage collection
     */
    fun forceGarbageCollection() {
        Log.d(TAG, "Forcing garbage collection")
        
        System.gc()
        
        // Wait for GC to complete
        try {
            Thread.sleep(100)
        } catch (e: InterruptedException) {
            // Ignore
        }
    }
    
    /**
     * Get current memory info
     */
    fun getCurrentMemoryInfo(): MemoryInfo? {
        return _memoryInfo.value
    }
    
    /**
     * Check for memory leak (detective method)
     */
    fun checkForMemoryLeak() {
        if (!enabled) return
        
        val memoryInfo = _memoryInfo.value ?: return
        
        // Simple heuristic: if memory keeps growing despite GC
        if (memoryInfo.usedPercentage > MEMORY_HIGH_THRESHOLD) {
            forceGarbageCollection()
            
            val newMemoryInfo = _memoryInfo.value
            if (newMemoryInfo != null && newMemoryInfo.usedPercentage > MEMORY_HIGH_THRESHOLD) {
                Log.w(
                    TAG,
                    "Potential memory leak detected: " +
                    "Used: ${newMemoryInfo.usedPercentage}%"
                )
            }
        }
    }
    
    /**
     * Dump memory summary to logs
     */
    fun dumpMemorySummary() {
        val memoryInfo = _memoryInfo.value ?: return
        
        Log.d(TAG, "=== Memory Summary ===")
        Log.d(TAG, "Total Memory: ${memoryInfo.totalMemory / (1024 * 1024)}MB")
        Log.d(TAG, "Used Memory: ${memoryInfo.usedMemory / (1024 * 1024)}MB (${memoryInfo.usedPercentage}%)")
        Log.d(TAG, "Available Memory: ${memoryInfo.availableMemory / (1024 * 1024)}MB")
        Log.d(TAG, "Native Heap: ${memoryInfo.nativeHeapAllocated / (1024 * 1024)}MB")
        Log.d(TAG, "Native Heap Free: ${memoryInfo.nativeHeapFree / (1024 * 1024)}MB")
        Log.d(TAG, "Memory Pressure: ${_memoryPressure.value}")
        Log.d(TAG, "Is Low Memory: ${_isLowMemory.value}")
        Log.d(TAG, "====================")
    }
    
    companion object {
        private const val TAG = "MemoryMonitor"
        private const val POLL_INTERVAL_MS = 5000L // Check every 5 seconds
        private const val MEMORY_LOW_THRESHOLD = 70 // 70%
        private const val MEMORY_MEDIUM_THRESHOLD = 50 // 50%
        private const val MEMORY_HIGH_THRESHOLD = 70 // 70%
    }
}

/**
 * Memory info data class
 */
data class MemoryInfo(
    val totalMemory: Long,
    val availableMemory: Long,
    val usedMemory: Long,
    val usedPercentage: Int,
    val isLowMemory: Boolean,
    val nativeHeapSize: Long,
    val nativeHeapAllocated: Long,
    val nativeHeapFree: Long,
    val timestamp: Long
) {
    /**
     * Format used memory for display
     */
    fun formatUsedMemory(): String {
        val mb = usedMemory / (1024 * 1024)
        return "${mb}MB"
    }
    
    /**
     * Format available memory for display
     */
    fun formatAvailableMemory(): String {
        val mb = availableMemory / (1024 * 1024)
        return "${mb}MB"
    }
}

/**
 * Memory pressure levels
 */
enum class MemoryPressure {
    NORMAL,    // < 50%
    MEDIUM,    // 50-70%
    HIGH,      // > 70%
    CRITICAL   // System low memory
}
