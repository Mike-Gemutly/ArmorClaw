pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.PREFER_SETTINGS)
    repositories {
        google()
        mavenCentral()
        // Matrix Rust SDK releases
        maven { url = uri("https://jitpack.io") }
    }
}

rootProject.name = "ArmorClaw"

include(":shared")
include(":androidApp")
include(":armorclaw-ui")
