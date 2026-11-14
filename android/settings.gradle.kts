// Coddy for Android: the app ships the whole server.
//
// The Go binary is packaged in the APK as libcoddy.so (an APK asset cannot be
// executed on Android 10+, but a native library in the app's lib directory
// can), and the foreground service runs `libcoddy.so serve` on loopback. The
// WebView loads the same SPA the desktop build serves - from 127.0.0.1, not
// from someone else's server.
pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "coddy-android"
include(":app")
