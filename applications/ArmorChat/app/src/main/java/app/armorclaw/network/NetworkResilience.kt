package app.armorclaw.network

import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.util.Log
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import java.util.concurrent.ThreadLocalRandom
import java.util.concurrent.atomic.AtomicInteger

/**
 * Network resilience manager with exponential backoff and jitter
 *
 * Provides:
 * - Exponential backoff with jitter for reconnection
 * - Network change detection (WiFi <-> Cellular)
 * - Connection state monitoring
 * - Retry coordination
 */
class NetworkResilience(
    private val connectivityManager: ConnectivityManager
) {

    companion object {
        private const val TAG = "NetworkResilience"

        // Backoff configuration
        const val INITIAL_BACKOFF_MS = 1000L      // 1 second
        const val MAX_BACKOFF_MS = 30000L         // 30 seconds
        const val BACKOFF_MULTIPLIER = 2.0        // Double each attempt
        const val JITTER_FACTOR = 0.3             // 30% jitter

        // Retry configuration
        const val MAX_RETRY_ATTEMPTS = 10
        const val CONNECTION_TIMEOUT_MS = 10000L  // 10 seconds
    }

    // Connection state
    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Disconnected)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    // Network availability
    private val _isNetworkAvailable = MutableStateFlow(false)
    val isNetworkAvailable: StateFlow<Boolean> = _isNetworkAvailable.asStateFlow()

    // Retry tracking
    private val retryCount = AtomicInteger(0)
    private var retryJob: Job? = null

    // Network callback
    private val networkCallback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            Log.d(TAG, "Network available: $network")
            _isNetworkAvailable.value = true
            _connectionState.value = ConnectionState.Available(network)
        }

        override fun onLost(network: Network) {
            Log.d(TAG, "Network lost: $network")
            _isNetworkAvailable.value = checkNetworkAvailability()
            if (!_isNetworkAvailable.value) {
                _connectionState.value = ConnectionState.Lost(network)
            }
        }

        override fun onCapabilitiesChanged(
            network: Network,
            networkCapabilities: NetworkCapabilities
        ) {
            val type = when {
                networkCapabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> "WiFi"
                networkCapabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "Cellular"
                networkCapabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> "Ethernet"
                else -> "Unknown"
            }
            Log.d(TAG, "Network capabilities changed: $type")
            _connectionState.value = ConnectionState.Changed(network, type)
        }
    }

    init {
        // Register network callback
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
        connectivityManager.registerNetworkCallback(request, networkCallback)

        // Check initial state
        _isNetworkAvailable.value = checkNetworkAvailability()
    }

    /**
     * Check if network is currently available
     */
    private fun checkNetworkAvailability(): Boolean {
        val network = connectivityManager.activeNetwork ?: return false
        val capabilities = connectivityManager.getNetworkCapabilities(network) ?: return false
        return capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
    }

    /**
     * Calculate backoff delay with exponential increase and jitter
     *
     * Formula: min(maxDelay, baseDelay * 2^attempt) * (1 + jitter * random(-1, 1))
     */
    fun calculateBackoff(attempt: Int): Long {
        val exponentialDelay = INITIAL_BACKOFF_MS * Math.pow(BACKOFF_MULTIPLIER, attempt.toDouble())
        val cappedDelay = minOf(exponentialDelay.toLong(), MAX_BACKOFF_MS)

        // Add jitter to prevent thundering herd
        val jitter = ThreadLocalRandom.current().nextDouble(-JITTER_FACTOR, JITTER_FACTOR)
        val delayWithJitter = cappedDelay * (1 + jitter)

        return delayWithJitter.toLong().coerceAtLeast(100) // Minimum 100ms
    }

    /**
     * Execute an operation with retry logic
     */
    suspend fun <T> withRetry(
        operation: suspend () -> Result<T>,
        onSuccess: (T) -> Unit = {},
        onFailure: (Throwable) -> Unit = {},
        shouldRetry: (Throwable) -> Boolean = { true }
    ): Result<T> = withContext(Dispatchers.IO) {
        var lastError: Throwable? = null

        for (attempt in 0 until MAX_RETRY_ATTEMPTS) {
            // Wait for network if not available
            if (!_isNetworkAvailable.value) {
                Log.d(TAG, "Waiting for network availability...")
                waitForNetwork()
            }

            try {
                val result = operation()

                if (result.isSuccess) {
                    retryCount.set(0)
                    _connectionState.value = ConnectionState.Connected
                    onSuccess(result.getOrThrow())
                    return@withContext result
                }

                // Check if error is retryable
                val error = result.exceptionOrNull()!!
                if (!shouldRetry(error)) {
                    onFailure(error)
                    return@withContext result
                }

                lastError = error
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                lastError = e
                if (!shouldRetry(e)) {
                    onFailure(e)
                    return@withContext Result.failure(e)
                }
            }

            // Calculate and apply backoff
            val backoff = calculateBackoff(attempt)
            Log.d(TAG, "Retry attempt ${attempt + 1}/$MAX_RETRY_ATTEMPTS, waiting ${backoff}ms")
            _connectionState.value = ConnectionState.Retrying(attempt + 1, backoff)

            delay(backoff)
        }

        // All retries exhausted
        val error = lastError ?: Exception("Max retries exceeded")
        _connectionState.value = ConnectionState.Failed(error)
        onFailure(error)
        Result.failure(error)
    }

    /**
     * Wait for network to become available
     */
    private suspend fun waitForNetwork(timeoutMs: Long = 30000) {
        withTimeout(timeoutMs) {
            _isNetworkAvailable
                .filter { it }
                .first()
        }
    }

    /**
     * Start automatic reconnection
     */
    fun startAutoReconnect(connect: suspend () -> Result<Unit>) {
        retryJob?.cancel()
        retryJob = CoroutineScope(Dispatchers.IO).launch {
            connectionState
                .filter { state ->
                    state is ConnectionState.Lost || state is ConnectionState.Failed
                }
                .collect {
                    Log.d(TAG, "Connection lost, starting auto-reconnect")
                    withRetry(
                        operation = connect,
                        shouldRetry = { _ -> true }
                    )
                }
        }
    }

    /**
     * Stop automatic reconnection
     */
    fun stopAutoReconnect() {
        retryJob?.cancel()
        retryJob = null
    }

    /**
     * Reset retry counter
     */
    fun resetRetryCount() {
        retryCount.set(0)
    }

    /**
     * Get current retry count
     */
    fun getRetryCount(): Int = retryCount.get()

    /**
     * Clean up resources
     */
    fun cleanup() {
        stopAutoReconnect()
        try {
            connectivityManager.unregisterNetworkCallback(networkCallback)
        } catch (e: Exception) {
            Log.w(TAG, "Failed to unregister network callback", e)
        }
    }
}

/**
 * Connection state sealed class
 */
sealed class ConnectionState {
    object Disconnected : ConnectionState()
    data class Available(val network: Network) : ConnectionState()
    data class Lost(val network: Network) : ConnectionState()
    data class Changed(val network: Network, val type: String) : ConnectionState()
    object Connected : ConnectionState()
    data class Retrying(val attempt: Int, val nextDelayMs: Long) : ConnectionState()
    data class Failed(val error: Throwable) : ConnectionState()
}

/**
 * Extension function to check if error is retryable
 */
fun Throwable.isRetryable(): Boolean {
    return when (this) {
        is java.net.SocketTimeoutException -> true
        is java.net.ConnectException -> true
        is java.net.UnknownHostException -> true
        is java.io.IOException -> true
        else -> false
    }
}
