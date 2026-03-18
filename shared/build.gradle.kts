import org.jetbrains.compose.ExperimentalComposeLibrary

plugins {
    kotlin("multiplatform")
    id("com.android.library")
    id("org.jetbrains.compose")
    kotlin("plugin.serialization")
    id("app.cash.sqldelight")
    jacoco
}

kotlin {
    androidTarget {
        compilations.all {
            kotlinOptions {
                jvmTarget = "1.8"
            }
        }
    }

    // iOS targets commented out for now (requires Kotlin/Native download)
    // iosX64()
    // iosArm64()
    // iosSimulatorArm64()

    sourceSets {
        val commonMain by getting {
            dependencies {
                implementation(project(":armorclaw-ui"))
                implementation(compose.ui)
                implementation(compose.runtime)
                implementation(compose.foundation)
                implementation(compose.material)
                implementation(compose.material3)
                implementation(compose.animation)
                @OptIn(ExperimentalComposeLibrary::class)
                implementation(compose.components.resources)

                // Material Icons Extended - provides all Material icons
                @OptIn(ExperimentalComposeLibrary::class)
                implementation("org.jetbrains.compose.material:material-icons-extended:1.5.0")

                // Coroutines
                implementation(libs.kotlinx.coroutines.core)

                // Koin
                implementation(libs.koin.core)

                // Ktor
                implementation(libs.ktor.core)
                implementation(libs.ktor.cio)
                implementation(libs.ktor.content.negotiation)
                implementation(libs.ktor.serialization.json)
                implementation(libs.ktor.logging)
                implementation(libs.ktor.websockets)

                // Serialization
                implementation(libs.kotlinx.serialization.json)
                implementation(libs.kotlinx.datetime)
                implementation(libs.kotlinx.collections.immutable)

                // OkHttp (Android only via expect/actual)
            }
        }

        val androidMain by getting {
            dependencies {
                implementation(libs.activity.compose)
                implementation(compose.preview)
                implementation(libs.kotlinx.coroutines.android)
                implementation(libs.okhttp)
                implementation(libs.okhttp.logging)
                implementation(libs.ktor.okhttp)
                implementation(libs.sqldelight.android.driver)
                implementation(libs.sqldelight.coroutines.extensions)
                implementation(libs.biometric)

                // Matrix Rust SDK (E2EE support)
                // NOTE: Matrix Rust SDK is distributed via GitHub releases, not Maven Central
                // For now, we use a placeholder implementation in MatrixClientImpl
                // TODO: Add proper Matrix SDK integration when available
                // implementation(libs.matrix.sdk.android)
                // implementation(libs.matrix.crypto.android)

                // Secure storage for sessions
                implementation(libs.security.crypto)

                // SQLCipher for encrypted database
                implementation(libs.sqlcipher)
                implementation(libs.sqlite)
            }
        }

        val commonTest by getting {
            dependencies {
                implementation(libs.kotlin.test)
                implementation(libs.kotlin.test.junit)
                implementation(libs.kotlinx.coroutines.test)
                implementation(libs.turbine)
            }
        }

        val androidUnitTest by getting {
            dependencies {
                implementation(libs.robolectric)
            }
        }
    }
}

android {
    namespace = "com.armorclaw.shared"
    compileSdk = 34
    defaultConfig {
        minSdk = 24
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_1_8
        targetCompatibility = JavaVersion.VERSION_1_8
    }
    buildFeatures {
        compose = true
    }
    composeOptions {
        kotlinCompilerExtensionVersion = "1.5.4"
    }
}

sqldelight {
    databases {
        create("ArmorClawDatabase") {
            packageName.set("com.armorclaw.shared.database")
            // Contains both regular tables and encrypted vault tables
        }
    }
}

// JaCoCo Configuration for test coverage
jacoco {
    toolVersion = "0.8.11"
}

android {
    // Enable JaCoCo for Android unit tests
    buildTypes {
        getByName("debug") {
            enableUnitTestCoverage = true
        }
    }
}

tasks.register<JacocoReport>("jacocoTestReport") {
    dependsOn(tasks.named("testDebugUnitTest"))

    reports {
        xml.required.set(true)
        html.required.set(true)
        csv.required.set(false)
    }

    val coverageSourceDirs = files(
        "src/commonMain/kotlin",
        "src/androidMain/kotlin"
    )

    sourceDirectories.setFrom(coverageSourceDirs)

    classDirectories.setFrom(
        fileTree("${layout.buildDirectory.get().asFile}/intermediates/classes/debug") {
            exclude(
                "**/R.class",
                "**/R$*.class",
                "**/BuildConfig.*",
                "**/Manifest*.*",
                "**/*Test*.*",
                "**/di/**",
                "**/generated/**"
            )
        }
    )

    executionData.setFrom(
        fileTree(layout.buildDirectory.get().asFile) {
            include("**/*.exec", "**/*.ec")
        }
    )
}
