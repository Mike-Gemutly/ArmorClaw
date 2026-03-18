# Compose
-keep class androidx.compose.** { *; }
-keep class androidx.compose.runtime.** { *; }
-keep class androidx.compose.foundation.** { *; }
-keep class androidx.compose.material.** { *; }
-keep class androidx.compose.material3.** { *; }
-keep class androidx.compose.ui.** { *; }

# Kotlin Coroutines
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}

# Kotlinx Serialization
-keepattributes RuntimeVisibleAnnotations,AnnotationDefault
-keepclassmembers class kotlinx.serialization.json.** {
    *** Companion;
}
-keepclasseswithmembers class kotlinx.serialization.json.** {
    kotlinx.serialization.KSerializer serializer(...);
}
-keep,includedescriptorclasses class com.armorclaw.**$$serializer { *; }
-keepclassmembers class com.armorclaw.** {
    *** Companion;
}
-keepclasseswithmembers class com.armorclaw.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# Koin
-keep class io.insert.koin.** { *; }
-dontwarn io.insert.koin.**

# SQLDelight
-keep class app.cash.sqldelight.** { *; }

# Ktor
-keep class io.ktor.** { *; }

# Sentry
-keep class io.sentry.** { *; }

# Matrix
-keep class org.matrix.** { *; }

# Gson
-keepattributes Signature
-keepattributes *Annotation*
-dontwarn sun.misc.**
-keep class com.google.gson.** { *; }
-keep class * implements com.google.gson.TypeAdapter
-keep class * implements com.google.gson.TypeAdapterFactory
-keep class * implements com.google.gson.JsonSerializer
-keep class * implements com.google.gson.JsonDeserializer

# OkHttp
-dontwarn okhttp3.**
-keep interface okhttp3.** { *; }
-keep class okhttp3.** { *; }

# Firebase
-keep class com.google.firebase.** { *; }
-dontwarn com.google.android.gms.**

# Biometric
-keep class android.hardware.biometrics.** { *; }

# Native methods
-keepclasseswithmembernames class * {
    native <methods>;
}

# Security - Remove debugging info
-assumenosideeffects class android.util.Log {
    public static int v(...);
    public static int d(...);
    public static int i(...);
}

# Keep encryption classes
-keep class com.armorclaw.shared.security.** { *; }
-keep class com.armorclaw.app.security.** { *; }

# R8/ProGuard: Suppress warnings for missing JDK classes not available on Android
-dontwarn java.lang.management.ManagementFactory
-dontwarn java.lang.management.RuntimeMXBean
-dontwarn org.slf4j.impl.StaticLoggerBinder

# Ktor debug detector - not needed in release
-dontwarn io.ktor.util.debug.**
