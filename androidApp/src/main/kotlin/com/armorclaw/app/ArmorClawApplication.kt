package com.armorclaw.app

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Build
import com.armorclaw.app.di.appModules
import com.armorclaw.app.platform.Analytics
import com.armorclaw.app.platform.CrashReporter
import com.armorclaw.app.platform.logging.AnalyticsAdapterImpl
import com.armorclaw.app.platform.logging.CrashReportingAdapterImpl
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag
import org.koin.android.ext.android.get
import org.koin.android.ext.koin.androidContext
import org.koin.android.ext.koin.androidLogger
import org.koin.core.context.startKoin
import org.koin.core.logger.Level

class ArmorClawApplication : Application() {

    companion object {
        private lateinit var instance: ArmorClawApplication

        fun getInstance(): ArmorClawApplication = instance

        fun getContext(): Context = instance.applicationContext

        // Notification Channel IDs
        const val CHANNEL_MESSAGES = "messages"
        const val CHANNEL_CALLS = "calls"
        const val CHANNEL_ALERTS = "alerts"

        fun isNetworkAvailable(): Boolean = instance._isOnline
    }

    private lateinit var connectivityManager: ConnectivityManager
    private var _isOnline = true

    private val networkCallback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            _isOnline = true
            AppLogger.debug(LogTag.Platform.Network, "Network available")
        }

        override fun onLost(network: Network) {
            _isOnline = false
            AppLogger.debug(LogTag.Platform.Network, "Network lost")
        }

        override fun onCapabilitiesChanged(network: Network, networkCapabilities: NetworkCapabilities) {
            _isOnline = networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                          networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
            AppLogger.debug(LogTag.Platform.Network, "Network capabilities changed: $_isOnline")
        }
    }

    override fun onCreate() {
        super.onCreate()
        instance = this

        // Initialize logging first
        initializeLogging()

        // Log app startup
        AppLogger.info(LogTag.Lifecycle.App, "Application starting")

        try {
            // Initialize Koin DI
            startKoin {
                androidLogger(Level.ERROR)
                androidContext(this@ArmorClawApplication)
                modules(appModules)
            }
            AppLogger.info(LogTag.DI.Initialization, "Koin initialized successfully")

            initializeNetworkMonitor()

            // Create notification channels (Android 8.0+)
            createNotificationChannels()

            AppLogger.info(LogTag.Lifecycle.App, "Application initialized successfully")
        } catch (e: Exception) {
            AppLogger.fatal(
                LogTag.Lifecycle.App,
                "Failed to initialize application",
                e
            )
            throw e
        }
    }

    private fun initializeNetworkMonitor() {
        connectivityManager = getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager

        val networkRequest = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()

        connectivityManager.registerNetworkCallback(networkRequest, networkCallback)

        val activeNetwork = connectivityManager.activeNetwork
        if (activeNetwork != null) {
            val capabilities = connectivityManager.getNetworkCapabilities(activeNetwork)
            _isOnline = capabilities?.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) == true &&
                         capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
        }

        AppLogger.debug(LogTag.Platform.Network, "NetworkMonitor initialized, initial state: $_isOnline")
    }

    private fun initializeLogging() {
        val isDebugMode = BuildConfig.DEBUG

        // Get crash reporter and analytics from Koin (they'll be available after startKoin)
        // For now, initialize without them - they'll be connected later
        AppLogger.initialize(
            crashReporter = null, // Will be set after Koin init
            analytics = null,     // Will be set after Koin init
            isDebugMode = isDebugMode
        )

        // Connect crash reporting and analytics after Koin is initialized
        // This is done in a separate method called after startKoin
        connectLoggingAdapters()
    }

    private fun connectLoggingAdapters() {
        try {
            // Get instances from Koin
            val crashReporter: CrashReporter = get()
            val analytics: Analytics = get()

            // Create adapters
            val crashAdapter = CrashReportingAdapterImpl(crashReporter)
            val analyticsAdapter = AnalyticsAdapterImpl(analytics)

            // Re-initialize with adapters
            AppLogger.initialize(
                crashReporter = crashAdapter,
                analytics = analyticsAdapter,
                isDebugMode = BuildConfig.DEBUG
            )

            AppLogger.info(LogTag.CrashReporting.Initialization, "Logging adapters connected")
        } catch (e: Exception) {
            // Log to console if adapters fail
            AppLogger.warning(
                LogTag.CrashReporting.Initialization,
                "Failed to connect logging adapters: ${e.message}"
            )
        }
    }

    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val notificationManager = getSystemService(NotificationManager::class.java)

            // Messages channel
            val messagesChannel = NotificationChannel(
                CHANNEL_MESSAGES,
                getString(R.string.notification_channel_messages),
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = getString(R.string.notification_channel_messages_desc)
                enableLights(true)
                enableVibration(true)
                setShowBadge(true)
            }

            // Calls channel
            val callsChannel = NotificationChannel(
                CHANNEL_CALLS,
                getString(R.string.notification_channel_calls),
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = getString(R.string.notification_channel_calls_desc)
                enableLights(true)
                enableVibration(true)
                setShowBadge(true)
            }

            // Security alerts channel
            val alertsChannel = NotificationChannel(
                CHANNEL_ALERTS,
                "Security Alerts",
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Important security notifications"
                enableLights(true)
                enableVibration(true)
                setShowBadge(true)
            }

            notificationManager.createNotificationChannels(
                listOf(messagesChannel, callsChannel, alertsChannel)
            )

            AppLogger.debug(
                LogTag.Platform.Notification,
                "Notification channels created",
                mapOf("channelCount" to 3)
            )
        }
    }

    override fun onTerminate() {
        super.onTerminate()

        try {
            connectivityManager.unregisterNetworkCallback(networkCallback)
            AppLogger.debug(LogTag.Platform.Network, "NetworkMonitor cleaned up")
        } catch (e: Exception) {
            AppLogger.warning(
                LogTag.Platform.Network,
                "Failed to unregister network callback: ${e.message}"
            )
        }

        AppLogger.info(LogTag.Lifecycle.App, "Application terminating")
    }
}
