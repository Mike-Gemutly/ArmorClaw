package app.armorclaw.validation

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import org.json.JSONObject
import java.io.File

/**
 * Exports the current validation state to a deterministic path for CI/ADB extraction.
 *
 * Trigger via: adb shell am broadcast -a com.armorclaw.validation.EXPORT_STATE
 * Extract via: MSYS_NO_PATHCONV=1 adb pull /sdcard/Android/data/app.armorclaw.app.debug/cache/validation_state.json
 */
class ValidationReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "ValidationReceiver"
        const val ACTION_EXPORT_STATE = "com.armorclaw.validation.EXPORT_STATE"

        private val EXPORT_DIR = File("/sdcard/Android/data/app.armorclaw.app.debug/cache")
        private const val EXPORT_FILE = "validation_state.json"
    }

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != ACTION_EXPORT_STATE) return

        try {
            val state = buildValidationState(context)
            exportToFile(state)

            Log.i(TAG, "Validation state exported to ${EXPORT_DIR}/${EXPORT_FILE}")

            val result = Intent(ACTION_EXPORT_STATE).apply {
                putExtra("success", true)
                putExtra("path", "${EXPORT_DIR}/${EXPORT_FILE}")
            }
            context.sendBroadcast(result)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to export validation state", e)
            val result = Intent(ACTION_EXPORT_STATE).apply {
                putExtra("success", false)
                putExtra("error", e.message)
            }
            context.sendBroadcast(result)
        }
    }

    private fun buildValidationState(context: Context): JSONObject {
        val state = JSONObject()
        state.put("timestamp", System.currentTimeMillis())
        state.put("app_version", getAppVersion(context))
        state.put("build_type", if (isDebugBuild(context)) "debug" else "release")
        state.put("device", android.os.Build.MODEL)
        state.put("sdk", android.os.Build.VERSION.SDK_INT)
        state.put("export_path", "${EXPORT_DIR}/${EXPORT_FILE}")
        return state
    }

    private fun exportToFile(state: JSONObject) {
        EXPORT_DIR.mkdirs()
        val file = File(EXPORT_DIR, EXPORT_FILE)
        file.writeText(state.toString(2))
    }

    private fun getAppVersion(context: Context): String {
        return try {
            val pInfo = context.packageManager.getPackageInfo(context.packageName, 0)
            pInfo.versionName ?: "unknown"
        } catch (e: Exception) {
            "unknown"
        }
    }

    private fun isDebugBuild(context: Context): Boolean {
        return (context.applicationInfo.flags and android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0
    }
}
