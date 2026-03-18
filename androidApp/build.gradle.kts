import java.util.Properties

plugins {
    id("com.android.application")
    id("org.jetbrains.compose")
    id("org.jetbrains.kotlin.android")
    kotlin("plugin.serialization")
    jacoco
}

// Load signing config from keystore.properties or environment variables
val keystorePropertiesFile = rootProject.file("keystore.properties")
val keystoreProperties = Properties()
if (keystorePropertiesFile.exists()) {
    keystoreProperties.load(keystorePropertiesFile.inputStream())
}

android {
    namespace = "com.armorclaw.app"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.armorclaw.app"
        minSdk = 24
        targetSdk = 35

        // Version strategy: MAJOR * 10000 + MINOR * 100 + PATCH
        // 1.0.0 = 10000, 1.0.1 = 10001, 1.1.0 = 10100, 2.0.0 = 20000
        // IMPORTANT: Increment versionCode for every release!
        versionCode = 10000
        versionName = "1.0.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"

        // Manifest placeholders
        manifestPlaceholders["crashlyticsCollectionEnabled"] = "false"
    }

    signingConfigs {
        create("release") {
            // Try keystore.properties first, fall back to environment variables
            storeFile = file(
                keystoreProperties.getProperty("storeFile")
                    ?: System.getenv("KEYSTORE_FILE")
                    ?: "armorclaw-release.keystore"
            )
            storePassword = keystoreProperties.getProperty("storePassword")
                ?: System.getenv("KEYSTORE_PASSWORD")
                ?: ""
            keyAlias = keystoreProperties.getProperty("keyAlias")
                ?: System.getenv("KEY_ALIAS")
                ?: "armorclaw"
            keyPassword = keystoreProperties.getProperty("keyPassword")
                ?: System.getenv("KEY_PASSWORD")
                ?: ""
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            // Only apply signing config if credentials are available
            if (keystoreProperties.containsKey("storePassword") ||
                System.getenv("KEYSTORE_PASSWORD") != null) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
        debug {
            isDebuggable = true
            applicationIdSuffix = ".debug"
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }

    kotlinOptions {
        jvmTarget = "11"
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    composeOptions {
        kotlinCompilerExtensionVersion = "1.5.15"
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }

    lint {
        disable += "FullBackupContent"
    }
}

dependencies {
    implementation(project(":shared"))
    implementation(project(":armorclaw-ui"))

    // Force consistent Kotlin version
    implementation(enforcedPlatform("org.jetbrains.kotlin:kotlin-bom:1.9.20"))

    implementation(libs.activity.compose)
    implementation(compose.material3)
    implementation(compose.preview)
    implementation("androidx.compose.material:material-icons-extended:1.5.4")
    implementation("androidx.appcompat:appcompat:1.6.1")

    // Lifecycle
    implementation(libs.lifecycle.viewmodel.compose)
    implementation(libs.lifecycle.runtime.compose)

    // Navigation
    implementation(libs.navigation.compose)

    // Koin
    implementation(libs.koin.android)
    implementation(libs.koin.compose)

    // Ktor (OkHttp engine for Android)
    implementation(libs.ktor.okhttp)
    implementation(libs.ktor.logging)

    // Firebase
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)
    implementation(libs.firebase.crashlytics)
    implementation(libs.firebase.analytics)

    // Biometric
    implementation(libs.biometric)

    // Security - AndroidX Security Crypto for Keystore-backed encryption
    implementation(libs.security.crypto)

    // SQLCipher for encrypted PII vault
    implementation(libs.sqlcipher)
    implementation(libs.sqlite)

    // Coil
    implementation(libs.coil.compose)

    // Kotlinx
    implementation(libs.kotlinx.datetime)
    implementation(libs.kotlinx.coroutines.core)
    implementation(libs.kotlinx.serialization.json)

    // Sentry
    implementation(libs.sentry.kotlin)
    implementation(libs.sentry.compose)

    // WorkManager
    implementation(libs.work.runtime)

    // Accompanist
    implementation(libs.accompanist.swiperefresh)
    implementation(libs.accompanist.placeholder)

    // CameraX for QR scanning
    val cameraxVersion = "1.3.1"
    implementation("androidx.camera:camera-core:$cameraxVersion")
    implementation("androidx.camera:camera-camera2:$cameraxVersion")
    implementation("androidx.camera:camera-lifecycle:$cameraxVersion")
    implementation("androidx.camera:camera-view:$cameraxVersion")

    // MLKit Barcode Scanning
    implementation("com.google.mlkit:barcode-scanning:17.2.0")

    // Testing
    testImplementation(libs.junit)
    testImplementation(libs.kotlin.test)
    testImplementation(libs.kotlin.test.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.mockk)
    testImplementation(libs.turbine)
    androidTestImplementation(libs.junit)
    androidTestImplementation(libs.kotlin.test)
}

compose {
    android {}
}

// JaCoCo Configuration for test coverage
jacoco {
    toolVersion = "0.8.11"
}

android {
    buildTypes {
        getByName("debug") {
            enableUnitTestCoverage = true
            enableAndroidTestCoverage = true
        }
    }
}

// Configure JacocoTestReport task
tasks.register<JacocoReport>("jacocoTestReport") {
    dependsOn("testDebugUnitTest")

    reports {
        xml.required.set(true)
        html.required.set(true)
        csv.required.set(false)
    }

    val fileFilter = listOf(
        "**/R.class",
        "**/R\$*.class",
        "**/BuildConfig.*",
        "**/Manifest*.*",
        "**/*Test*.*",
        "android/**/*.*",
        "**/databinding/*",
        "**/android/databinding/*",
        "**/androidx/databinding/*",
        "**/*_MembersInjector.class",
        "**/Dagger*Component*.*",
        "**/*Module_*Factory.class",
        "**/*_Factory.class",
        "**/*Module*.*",
        "**/*Dagger*.*",
        "**/di/*",
        "**/hilt/*",
        "**/Hilt_*.*"
    )

    val debugTreeKotlin = fileTree("${project.buildDir}/tmp/kotlin-classes/debug") {
        exclude(fileFilter)
    }

    val debugTreeJava = fileTree("${project.buildDir}/intermediates/javac/debug/classes") {
        exclude(fileFilter)
    }

    val mainSrc = "${project.projectDir}/src/main"

    sourceDirectories.setFrom(files(mainSrc))
    classDirectories.setFrom(files(debugTreeKotlin, debugTreeJava))
    executionData.setFrom(fileTree(project.buildDir) {
        include(listOf("jacoco/testDebugUnitTest.exec"))
    })
}

// Coverage verification task (for CI gate)
tasks.register<JacocoCoverageVerification>("jacocoCoverageVerification") {
    dependsOn("jacocoTestReport")

    violationRules {
        rule {
            limit {
                minimum = "0.50".toBigDecimal()  // 50% minimum coverage
            }
        }
    }

    val fileFilter = listOf(
        "**/R.class",
        "**/R\$*.class",
        "**/BuildConfig.*",
        "**/Manifest*.*",
        "**/*Test*.*",
        "android/**/*.*",
        "**/databinding/*",
        "**/android/databinding/*",
        "**/androidx/databinding/*",
        "**/*_MembersInjector.class",
        "**/Dagger*Component*.*",
        "**/*Module_*Factory.class",
        "**/*_Factory.class",
        "**/*Module*.*",
        "**/*Dagger*.*",
        "**/di/*",
        "**/hilt/*",
        "**/Hilt_*.*"
    )

    val debugTreeKotlin = fileTree("${project.buildDir}/tmp/kotlin-classes/debug") {
        exclude(fileFilter)
    }

    val debugTreeJava = fileTree("${project.buildDir}/intermediates/javac/debug/classes") {
        exclude(fileFilter)
    }

    val mainSrc = "${project.projectDir}/src/main"

    sourceDirectories.setFrom(files(mainSrc))
    classDirectories.setFrom(files(debugTreeKotlin, debugTreeJava))
    executionData.setFrom(fileTree(project.buildDir) {
        include(listOf("jacoco/testDebugUnitTest.exec"))
    })
}

tasks.named("check") {
    finalizedBy("jacocoTestReport")
}
