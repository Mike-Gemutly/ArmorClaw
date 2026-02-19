package com.armorclaw.armorterminal.network

import kotlinx.coroutines.*
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.channels.SendChannel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import okhttp3.*
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/**
 * Resilient WebSocket Client
 *
 * Features:
 * - Automatic reconnection with exponential backoff
 * - Message queueing during disconnection
 * - Heartbeat/ping-pong
 * - Connection state management
 */
class ResilientWebSocket(
    private val url: String,
    private val deviceId: String,
    private val sessionToken: String,
    private val client: OkHttpClient = defaultClient()
) {

    companion object {
        private const val INITIAL_BACKOFF_MS = 1000L
        private const val MAX_BACKOFF_MS = 30000L
        private const val BACKOFF_MULTIPLIER = 2.0
        private const val JITTER_FACTOR = 0.3
        private const val HEARTBEAT_INTERVAL_MS = 30000L
        private const val MAX_QUEUE_SIZE = 100

        private fun defaultClient() = OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(0, TimeUnit.SECONDS) // WebSocket needs infinite read timeout
            .writeTimeout(30, TimeUnit.SECONDS)
            .pingInterval(25, TimeUnit.SECONDS)
            .build()
    }

    private val _connectionState = MutableStateFlow<ConnectionState>(ConnectionState.Disconnected)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private var webSocket: WebSocket? = null
    private var reconnectJob: Job? = null
    private var heartbeatJob: Job? = null
    private val reconnectAttempts = AtomicInteger(0)
    private val messageQueue = Channel<String>(MAX_QUEUE_SIZE)

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    /**
     * Connect to the WebSocket server
     */
    fun connect(listener: WebSocketListener? = null) {
        if (_connectionState.value == ConnectionState.Connecting ||
            _connectionState.value == ConnectionState.Connected) {
            return
        }

        _connectionState.value = ConnectionState.Connecting

        val request = Request.Builder()
            .url(url)
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                reconnectAttempts.set(0)
                _connectionState.value = ConnectionState.Connected

                // Send registration message
                val registerMsg = """{"type":"register","payload":{"device_id":"$deviceId"}}"""
                webSocket.send(registerMsg)

                // Start heartbeat
                startHeartbeat()

                // Flush queued messages
                flushQueue()

                listener?.onOpen(webSocket, response)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                listener?.onMessage(webSocket, text)
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                listener?.onClosing(webSocket, code, reason)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                _connectionState.value = ConnectionState.Disconnected
                stopHeartbeat()
                listener?.onClosed(webSocket, code, reason)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                _connectionState.value = ConnectionState.Disconnected(t.message ?: "Connection failed")
                stopHeartbeat()
                scheduleReconnect(listener)
                listener?.onFailure(webSocket, t, response)
            }
        })
    }

    /**
     * Disconnect from the server
     */
    fun disconnect(code: Int = 1000, reason: String = "Client disconnecting") {
        reconnectJob?.cancel()
        stopHeartbeat()
        webSocket?.close(code, reason)
        webSocket = null
        _connectionState.value = ConnectionState.Disconnected
    }

    /**
     * Send a message
     *
     * If disconnected, the message will be queued for later delivery.
     */
    fun send(message: String): Boolean {
        return when (_connectionState.value) {
            is ConnectionState.Connected -> {
                webSocket?.send(message) ?: false
            }
            else -> {
                // Queue the message
                try {
                    messageQueue.trySend(message)
                    true
                } catch (e: Exception) {
                    false
                }
            }
        }
    }

    /**
     * Send an RPC message over WebSocket
     */
    fun sendRPC(method: String, params: Map<String, Any> = emptyMap(), id: Int? = null): Boolean {
        val message = buildString {
            append("{\"type\":\"rpc\",\"payload\":{")
            append("\"method\":\"$method\"")
            if (id != null) append(",\"id\":$id")
            if (params.isNotEmpty()) {
                append(",\"params\":{")
                params.entries.forEachIndexed { index, (key, value) ->
                    if (index > 0) append(",")
                    append("\"$key\":")
                    append(when (value) {
                        is String -> "\"$value\""
                        is Number -> value.toString()
                        is Boolean -> value.toString()
                        else -> "\"$value\""
                    })
                }
                append("}")
            }
            append("}}")
        }
        return send(message)
    }

    private fun startHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = scope.launch {
            while (isActive) {
                delay(HEARTBEAT_INTERVAL_MS)
                if (_connectionState.value is ConnectionState.Connected) {
                    webSocket?.send("""{"type":"ping"}""")
                }
            }
        }
    }

    private fun stopHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = null
    }

    private fun flushQueue() {
        scope.launch {
            while (true) {
                val message = messageQueue.tryReceive().getOrNull() ?: break
                webSocket?.send(message)
            }
        }
    }

    private fun scheduleReconnect(listener: WebSocketListener?) {
        reconnectJob?.cancel()

        val attempts = reconnectAttempts.incrementAndGet()
        val backoff = calculateBackoff(attempts)

        reconnectJob = scope.launch {
            delay(backoff)
            if (_connectionState.value !is ConnectionState.Connected) {
                connect(listener)
            }
        }
    }

    private fun calculateBackoff(attempt: Int): Long {
        val exponential = INITIAL_BACKOFF_MS * Math.pow(BACKOFF_MULTIPLIER, attempt - 1.0)
        val capped = minOf(exponential.toLong(), MAX_BACKOFF_MS)
        val jitter = capped * JITTER_FACTOR * (Math.random() * 2 - 1)
        return (capped + jitter).toLong().coerceAtLeast(100)
    }
}

/**
 * Connection state
 */
sealed class ConnectionState {
    object Disconnected : ConnectionState()
    object Connecting : ConnectionState()
    object Connected : ConnectionState()
    data class Disconnecting(val reason: String) : ConnectionState()
    data class Error(val message: String) : ConnectionState()

    companion object {
        fun Disconnected(reason: String? = null): ConnectionState {
            return if (reason != null) Error(reason) else Disconnected
        }
    }
}

/**
 * WebSocket message types
 */
sealed class WSMessage {
    data class DeviceApproved(val deviceId: String) : WSMessage()
    data class DeviceRejected(val deviceId: String, val reason: String) : WSMessage()
    data class NewMessage(val roomId: String, val messageId: String, val sender: String, val content: String) : WSMessage()
    data class WorkflowUpdate(val workflowId: String, val stepId: String, val status: String, val progress: Int) : WSMessage()
    data class Pong(val timestamp: String) : WSMessage()
    data class Unknown(val raw: String) : WSMessage()

    companion object {
        fun parse(json: String): WSMessage {
            return try {
                when {
                    json.contains("\"device.approved\"") -> {
                        val deviceId = json.extractValue("device_id") ?: ""
                        DeviceApproved(deviceId)
                    }
                    json.contains("\"device.rejected\"") -> {
                        val deviceId = json.extractValue("device_id") ?: ""
                        val reason = json.extractValue("reason") ?: ""
                        DeviceRejected(deviceId, reason)
                    }
                    json.contains("\"message.new\"") -> {
                        NewMessage(
                            roomId = json.extractValue("room_id") ?: "",
                            messageId = json.extractValue("message_id") ?: "",
                            sender = json.extractValue("sender") ?: "",
                            content = json.extractValue("content") ?: ""
                        )
                    }
                    json.contains("\"workflow.update\"") -> {
                        WorkflowUpdate(
                            workflowId = json.extractValue("workflow_id") ?: "",
                            stepId = json.extractValue("step_id") ?: "",
                            status = json.extractValue("status") ?: "",
                            progress = json.extractValue("progress")?.toIntOrNull() ?: 0
                        )
                    }
                    json.contains("\"pong\"") -> {
                        Pong(json.extractValue("timestamp") ?: "")
                    }
                    else -> Unknown(json)
                }
            } catch (e: Exception) {
                Unknown(json)
            }
        }

        private fun String.extractValue(key: String): String? {
            val regex = """"$key"\s*:\s*"([^"]+)"""".toRegex()
            return regex.find(this)?.groupValues?.get(1)
        }
    }
}
