package com.armorclaw.app.util

import android.content.Context
import android.content.Intent
import android.net.Uri
import com.armorclaw.shared.platform.logging.AppLogger
import com.armorclaw.shared.platform.logging.LogTag

/**
 * Handler for external links
 *
 * Provides safe ways to open external URLs in the default browser.
 */
class ExternalLinkHandler(
    private val context: Context
) {
    /**
     * Open URL in Chrome Custom Tab (stays in app)
     * Falls back to browser if custom tabs not available.
     */
    fun openInCustomTab(url: String) {
        // Just use browser for now
        openInBrowser(url)
    }

    /**
     * Open URL in default browser
     */
    fun openInBrowser(url: String) {
        try {
            val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url))
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            context.startActivity(intent)
            
            AppLogger.info(
                LogTag.Platform.Navigation,
                "Opened URL in browser",
                mapOf("url" to url)
            )
        } catch (e: Exception) {
            AppLogger.error(
                LogTag.Platform.Navigation,
                "Failed to open URL: ${e.message}",
                e,
                mapOf("url" to url)
            )
        }
    }

    /**
     * Open email client with pre-filled email
     */
    fun openEmail(to: String, subject: String? = null, body: String? = null) {
        try {
            val uri = Uri.Builder()
                .scheme("mailto")
                .authority(to)
                .apply {
                    subject?.let { appendQueryParameter("subject", it) }
                    body?.let { appendQueryParameter("body", it) }
                }
                .build()
            
            val intent = Intent(Intent.ACTION_SENDTO, uri)
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            context.startActivity(intent)
            
            AppLogger.info(
                LogTag.Platform.Navigation,
                "Opened email client",
                mapOf("to" to to)
            )
        } catch (e: Exception) {
            AppLogger.error(
                LogTag.Platform.Navigation,
                "Failed to open email: ${e.message}",
                e,
                mapOf("to" to to)
            )
        }
    }

    /**
     * Open app in Play Store
     */
    fun openPlayStore(appId: String = context.packageName) {
        try {
            val intent = Intent(Intent.ACTION_VIEW, Uri.parse("market://details?id=$appId"))
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            context.startActivity(intent)
            
            AppLogger.info(
                LogTag.Platform.Navigation,
                "Opened Play Store",
                mapOf("appId" to appId)
            )
        } catch (e: Exception) {
            // Fallback to browser if Play Store not installed
            openInBrowser("https://play.google.com/store/apps/details?id=$appId")
        }
    }

    /**
     * Share content via system share sheet
     */
    fun share(title: String, text: String, url: String? = null) {
        try {
            val shareText = if (url != null) {
                "$text\n$url"
            } else {
                text
            }
            
            val intent = Intent(Intent.ACTION_SEND).apply {
                type = "text/plain"
                putExtra(Intent.EXTRA_SUBJECT, title)
                putExtra(Intent.EXTRA_TEXT, shareText)
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            
            val chooserIntent = Intent.createChooser(intent, title)
            chooserIntent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            context.startActivity(chooserIntent)
            
            AppLogger.info(
                LogTag.Platform.Navigation,
                "Opened share sheet",
                mapOf("title" to title)
            )
        } catch (e: Exception) {
            AppLogger.error(
                LogTag.Platform.Navigation,
                "Failed to share: ${e.message}",
                e
            )
        }
    }

    companion object {
        // App URLs
        const val WEBSITE_URL = "https://armorclaw.app"
        const val GITHUB_URL = "https://github.com/armorclaw/armorclaw"
        const val TWITTER_URL = "https://twitter.com/armorclaw"
        const val MATRIX_ROOM = "https://matrix.to/#/#armorclaw:matrix.org"

        // Legal URLs
        const val PRIVACY_POLICY_URL = "https://armorclaw.app/privacy"
        const val TERMS_OF_SERVICE_URL = "https://armorclaw.app/terms"

        // Support URLs
        const val SUPPORT_EMAIL = "support@armorclaw.app"
        const val BUG_REPORT_URL = "https://github.com/armorclaw/armorclaw/issues"
    }
}
