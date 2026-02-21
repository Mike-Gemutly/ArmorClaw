plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("com.google.devtools.ksp")
    id("com.google.dagger.hilt.android")
    id("com.google.gms.google-services")
    id("kotlin-parcelize")
    id("kotlinx-serialization")
}

android {
    namespace = "app.armorclaw"
    compileSdk = 34

    defaultConfig {
        applicationId = "app.armorclaw"
        minSdk = 26
        targetSdk = 34
        versionCode = 1
        versionName = "1.0.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"

        // BuildConfig fields for server configuration
        // These are defaults - can be overridden by mDNS discovery or QR code
        buildConfigField("String", "MATRIX_HOMESERVER", "\"${project.findProperty("MATRIX_HOMESERVER") ?: "https://matrix.armorclaw.com"}\"")
        buildConfigField("String", "BRIDGE_API_URL", "\"${project.findProperty("BRIDGE_API_URL") ?: "https://armorclaw.com/api"}\"")
        buildConfigField("String", "BRIDGE_WS_URL", "\"${project.findProperty("BRIDGE_WS_URL") ?: "wss://armorclaw.com/ws"}\"")
        buildConfigField("String", "PUSH_GATEWAY", "\"${project.findProperty("PUSH_GATEWAY") ?: "https://armorclaw.com/_matrix/push/v1/notify"}\"")
        buildConfigField("String", "SERVER_NAME", "\"${project.findProperty("SERVER_NAME") ?: "ArmorClaw"}\"")
        buildConfigField("String", "REGION", "\"${project.findProperty("REGION") ?: "us-east-1"}\"")
    }

    buildTypes {
        debug {
            isDebuggable = true
            applicationIdSuffix = ".debug"

            // Debug defaults point to local development server
            buildConfigField("String", "MATRIX_HOMESERVER", "\"http://10.0.2.2:8008\"")
            buildConfigField("String", "BRIDGE_API_URL", "\"http://10.0.2.2:8080/api\"")
            buildConfigField("String", "BRIDGE_WS_URL", "\"ws://10.0.2.2:8080/ws\"")
            buildConfigField("String", "PUSH_GATEWAY", "\"http://10.0.2.2:8080/_matrix/push/v1/notify\"")
            buildConfigField("String", "SERVER_NAME", "\"Development\"")
            buildConfigField("String", "REGION", "\"local\"")
        }

        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )

            // Production values are set in defaultConfig
            // Can be overridden via gradle.properties or command line:
            // ./gradlew assembleRelease -PMATRIX_HOMESERVER="https://matrix.example.com"
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    composeOptions {
        kotlinCompilerExtensionVersion = "1.5.8"
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

dependencies {
    // Core Android
    implementation("androidx.core:core-ktx:1.12.0")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.7.0")
    implementation("androidx.activity:activity-compose:1.8.2")

    // Compose
    implementation(platform("androidx.compose:compose-bom:2024.01.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-graphics")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.material:material-icons-extended")

    // Navigation
    implementation("androidx.navigation:navigation-compose:2.7.6")

    // Hilt DI
    implementation("com.google.dagger:hilt-android:2.50")
    ksp("com.google.dagger:hilt-compiler:2.50")
    implementation("androidx.hilt:hilt-navigation-compose:1.1.0")

    // Matrix SDK
    implementation("org.matrix.android:matrix-sdk-android:1.6.10")
    implementation("org.matrix.android:matrix-sdk-android-rx:1.6.10")

    // Networking
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")
    implementation("com.squareup.retrofit2:retrofit:2.9.0")
    implementation("com.squareup.retrofit2:converter-kotlinx-serialization:0.8.0")

    // Serialization
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.2")

    // Coroutines
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.7.3")

    // Security
    implementation("androidx.security:security-crypto:1.1.0-alpha06")
    implementation("androidx.biometric:biometric:1.2.0-alpha05")

    // DataStore
    implementation("androidx.datastore:datastore-preferences:1.0.0")

    // Firebase
    implementation(platform("com.google.firebase:firebase-bom:32.7.1"))
    implementation("com.google.firebase:firebase-messaging")

    // QR Code
    implementation("com.google.zxing:core:3.5.2")
    implementation("com.journeyapps:zxing-android-embedded:4.3.0")

    // Testing
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.7.3")
    testImplementation("io.mockk:mockk:1.13.9")
    androidTestImplementation("androidx.test.ext:junit:1.1.5")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.5.1")
    androidTestImplementation(platform("androidx.compose:compose-bom:2024.01.00"))
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")
    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}
