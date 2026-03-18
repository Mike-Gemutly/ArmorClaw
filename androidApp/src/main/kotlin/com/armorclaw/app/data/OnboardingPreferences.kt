package com.armorclaw.app.data

import android.content.Context
import android.content.SharedPreferences

class OnboardingPreferences(context: Context) {
    private val prefs: SharedPreferences = context.getSharedPreferences(
        PREFS_NAME,
        Context.MODE_PRIVATE
    )
    
    companion object {
        private const val PREFS_NAME = "onboarding_prefs"
        private const val KEY_COMPLETED = "onboarding_completed"
        private const val KEY_CURRENT_STEP = "current_step"
        private const val KEY_SERVER_URL = "server_url"
        private const val KEY_USERNAME = "username"
        private const val KEY_PERMISSIONS_GRANTED = "permissions_granted"
    }
    
    var isCompleted: Boolean
        get() = prefs.getBoolean(KEY_COMPLETED, false)
        set(value) = prefs.edit().putBoolean(KEY_COMPLETED, value).apply()
    
    var currentStep: Int
        get() = prefs.getInt(KEY_CURRENT_STEP, 0)
        set(value) = prefs.edit().putInt(KEY_CURRENT_STEP, value).apply()
    
    var serverUrl: String
        get() = prefs.getString(KEY_SERVER_URL, "") ?: ""
        set(value) = prefs.edit().putString(KEY_SERVER_URL, value).apply()
    
    var username: String
        get() = prefs.getString(KEY_USERNAME, "") ?: ""
        set(value) = prefs.edit().putString(KEY_USERNAME, value).apply()
    
    var permissionsGranted: Map<String, Boolean>
        get() {
            val json = prefs.getString(KEY_PERMISSIONS_GRANTED, "{}") ?: "{}"
            return try {
                // Simple JSON parsing for demo
                val map = mutableMapOf<String, Boolean>()
                val pairs = json.removePrefix("{").removeSuffix("}").split(",")
                pairs.forEach { pair ->
                    val (key, value) = pair.split(":").map { it.trim() }
                    map[key.removePrefix("\"").removeSuffix("\"")] = value.toBoolean()
                }
                map
            } catch (e: Exception) {
                emptyMap()
            }
        }
        set(value) {
            // Simple JSON serialization for demo
            val json = value.entries.joinToString(",") { (key, v) ->
                "\"$key\":$v"
            }.let { "{$it}" }
            prefs.edit().putString(KEY_PERMISSIONS_GRANTED, json).apply()
        }
    
    fun reset() {
        prefs.edit().clear().apply()
    }
}
