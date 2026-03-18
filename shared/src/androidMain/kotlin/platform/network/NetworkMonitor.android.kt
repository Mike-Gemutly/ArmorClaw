package com.armorclaw.shared.platform.network

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.flow.map

actual class NetworkMonitor {

    private var context: Context? = null
    private var connectivityManager: ConnectivityManager? = null

    companion object {
        @Volatile
        private var instance: NetworkMonitor? = null

        fun getInstance(): NetworkMonitor {
            return instance ?: synchronized(this) {
                instance ?: NetworkMonitor().also { instance = it }
            }
        }

        fun setContext(context: Context) {
            getInstance().context = context.applicationContext
            getInstance().initialize()
        }
    }

    private fun initialize() {
        val ctx = context ?: return
        connectivityManager = ctx.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    }

    actual fun isOnline(): Boolean {
        val cm = connectivityManager ?: return false
        val network = cm.activeNetwork ?: return false
        val capabilities = cm.getNetworkCapabilities(network) ?: return false

        return capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
            capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
    }

    actual fun observeNetworkState(): Flow<NetworkState> {
        val cm = connectivityManager ?: return callbackFlow {
            trySend(NetworkState.Offline)
            close()
        }

        return callbackFlow {
            val callback = object : ConnectivityManager.NetworkCallback() {
                override fun onAvailable(network: Network) {
                    trySend(NetworkState.Online)
                }

                override fun onLost(network: Network) {
                    trySend(NetworkState.Offline)
                }

                override fun onUnavailable() {
                    trySend(NetworkState.Offline)
                }
            }

            val request = NetworkRequest.Builder()
                .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                .addCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
                .build()

            cm.registerNetworkCallback(request, callback)

            // Send initial state
            trySend(if (isOnline()) NetworkState.Online else NetworkState.Offline)

            awaitClose {
                cm.unregisterNetworkCallback(callback)
            }
        }
    }

    actual fun getNetworkType(): NetworkType {
        val cm = connectivityManager ?: return NetworkType.NONE
        val network = cm.activeNetwork ?: return NetworkType.NONE
        val capabilities = cm.getNetworkCapabilities(network) ?: return NetworkType.NONE

        return when {
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> NetworkType.WIFI
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> NetworkType.CELLULAR
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> NetworkType.ETHERNET
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_VPN) -> NetworkType.VPN
            else -> NetworkType.OTHER
        }
    }

    actual fun observeNetworkType(): Flow<NetworkType> {
        return observeNetworkState().map {
            getNetworkType()
        }
    }
}
