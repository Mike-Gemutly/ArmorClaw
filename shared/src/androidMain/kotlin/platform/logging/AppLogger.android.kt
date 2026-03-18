package com.armorclaw.shared.platform.logging

import android.util.Log

/**
 * Android implementation of platform-specific logging
 */
actual object PlatformLogger {
    actual fun debug(tag: String, message: String, throwable: Throwable?) {
        if (throwable != null) {
            Log.d(tag, message, throwable)
        } else {
            Log.d(tag, message)
        }
    }

    actual fun info(tag: String, message: String) {
        Log.i(tag, message)
    }

    actual fun warning(tag: String, message: String, throwable: Throwable?) {
        if (throwable != null) {
            Log.w(tag, message, throwable)
        } else {
            Log.w(tag, message)
        }
    }

    actual fun error(tag: String, message: String, throwable: Throwable?) {
        if (throwable != null) {
            Log.e(tag, message, throwable)
        } else {
            Log.e(tag, message)
        }
    }
}
