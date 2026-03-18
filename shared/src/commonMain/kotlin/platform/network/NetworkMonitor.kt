package com.armorclaw.shared.platform.network

import kotlinx.coroutines.flow.Flow

expect class NetworkMonitor() {
    fun isOnline(): Boolean
    fun observeNetworkState(): Flow<NetworkState>
    fun getNetworkType(): NetworkType
    fun observeNetworkType(): Flow<NetworkType>
}

enum class NetworkState {
    Online,
    Offline
}

enum class NetworkType {
    NONE,
    WIFI,
    CELLULAR,
    ETHERNET,
    VPN,
    OTHER
}
